package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/dollarshaveclub/psst/pkg/directory"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

const (
	keyPrefix       = "/secret/psst"
	vaultSecretName = "secret"

	filePerms = 0750
)

// PolicyTemplates are used with Vault to give users permissions
type PolicyTemplates struct {
	GeneralPolicyTemplate string
	PolicyTemplate        string
}

type ghUserPolicy struct {
	Value string `json:"value"`
}
type space struct {
	Path string
}

var (
	policies = map[string]PolicyTemplates{
		"github": PolicyTemplates{
			GeneralPolicyTemplate: `# Allows all users to write secrets to other users
path "{{.Path}}/*" {
	capabilities = ["create", "update"]
}
`,
			PolicyTemplate: `# Allows a user to read secrets from personal drop keyspace
path "{{.Path}}/*" {
	capabilities = ["read", "list", "delete"]
}
`,
		},
	}
)

// VaultStore stores a Vault client
type VaultStore struct {
	*api.Client
}

// NewVault will connect to a Vault server using VAULT_ADDR and VAUL_TOKEN variables
func NewVault() (*VaultStore, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return &VaultStore{}, fmt.Errorf("unable to get Vault client: %+v", err)
	}
	return &VaultStore{client}, nil
}

// Get will return the stored secret at a given path
func (v *VaultStore) Get(path string) (string, error) {
	secret, err := v.Client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("unable to read secret from vault: %+v", err)
	}
	if secret == nil {
		return "", errors.New("no secret found")
	}
	if data, ok := secret.Data[vaultSecretName]; ok {
		return data.(string), nil
	}
	return "", errors.New("improperly formatted secret")
}

// Write will write the provided secret to the given user
func (v *VaultStore) Write(filename, name string, targets map[string]struct{}) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %+v", filename, err)
	}

	data := make(map[string]interface{})
	data[vaultSecretName] = string(buf)

	for t := range targets {
		_, err := v.Logical().Write(path.Join(keyPrefix, t, name), data)
		if err != nil {
			return fmt.Errorf("unable to add secret for target %s: %+v", t, err)
		}
	}
	return nil
}

// List will list a set of secrets available
func (v *VaultStore) List(login string) ([]string, error) {
	path := getSecretPathPrefix(login)
	secret, err := v.Client.Logical().List(path)
	if err != nil {
		return []string{}, fmt.Errorf("unable to list secrets at %s: %v", path, err)
	}

	if secret == nil {
		return []string{}, nil
	}

	names := []string{}
	for _, s := range secret.Data {
		for _, v := range s.([]interface{}) {
			names = append(names, v.(string))
		}
	}
	return names, nil
}

// Delete will delete a secret from Vault
func (v *VaultStore) Delete(path string) error {
	_, err := v.Client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("unable to delete secret %s: %+v", path, err)
	}
	return nil
}

// GeneratePoliciesAndRoles will generate a set of policies for a given directory of members
func (v *VaultStore) GeneratePoliciesAndRoles(directoryBackend, roleDir, policyDir, defaultTeam string, members []directory.Member) error {
	policies, ok := policies[directoryBackend]
	if !ok {
		return fmt.Errorf("unknown directory backend %s", directoryBackend)
	}

	buf := bytes.NewBuffer([]byte{})

	gt := template.Must(template.New("generalPolicy").Parse(policies.GeneralPolicyTemplate))
	s := space{Path: keyPrefix}
	if err := gt.Execute(buf, s); err != nil {
		return fmt.Errorf("unable to execute template for general policy: %+v", err)
	}
	p := path.Join(policyDir, fmt.Sprintf("%s.hcl", filePrefix))
	if err := ioutil.WriteFile(p, buf.Bytes(), filePerms); err != nil {
		return fmt.Errorf("unable to write general psst policy file: %+v", err)
	}

	// Adds default role for the "all" team in GH
	teamRoles := roleDir
	if path.Base(roleDir) == "users" {
		teamRoles = path.Join(path.Dir(roleDir), "teams")
	}
	if err := checkRole(defaultTeam, filePrefix, teamRoles); err != nil {
		return fmt.Errorf("unable to write all team role: %v", err)
	}

	t := template.Must(template.New("policy").Parse(policies.PolicyTemplate))

	for _, m := range members {
		buf.Reset()
		s = space{Path: getSecretPathPrefix(m.Login)}
		if err := t.Execute(buf, s); err != nil {
			return fmt.Errorf("unable to execute template for user %s: %+v", m.Login, err)
		}
		roleName := fmt.Sprintf("%s-%s", filePrefix, m.Login)
		p = path.Join(policyDir, fmt.Sprintf("%s.hcl", roleName))
		if err := ioutil.WriteFile(p, buf.Bytes(), filePerms); err != nil {
			return fmt.Errorf("unable to write policy file for user %s: %+v", m.Login, err)
		}

		if err := checkRole(m.Login, roleName, roleDir); err != nil {
			return fmt.Errorf("Unable to setup role for %s: %v", m.Login, err)
		}
	}
	return nil
}

// checkRole will append the user's psst policy if it is missing. It will also add a file for new users added to the GitHub
// organization since the last update.
func checkRole(login, roleName, roleDir string) error {
	filename := path.Join(roleDir, fmt.Sprintf("%s.json", login))
	policy := ghUserPolicy{Value: ""}

	if _, err := os.Stat(filename); err == nil {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to read %s", filename))
		}
		if err := json.Unmarshal(b, &policy); err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to unmarshal %s", filename))
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, fmt.Sprintf("unexpected error using stat on %s", filename))
	}

	roles := strings.Split(policy.Value, ",")
	exists := false
	for _, role := range roles {
		if role == roleName {
			exists = true
			break
		}
	}

	if !exists {
		if len(roles) >= 1 && policy.Value != "" {
			policy.Value = strings.Join([]string{policy.Value, roleName}, ",")
		} else {
			policy.Value = roleName
		}
		b, err := json.Marshal(policy)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to marshal JSON when adding %s to %s", roleName, filename))
		}
		if err := ioutil.WriteFile(filename, b, 0644); err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to write %s", filename))
		}
	}
	return nil
}

func getSecretPathPrefix(login string) string {
	return path.Join(keyPrefix, login)
}

// SecretPath will return the path for a given secret
func (v *VaultStore) SecretPath(login, name string) string {
	return path.Join(getSecretPathPrefix(login), name)
}
