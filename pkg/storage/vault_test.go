package storage

import (
	"io/ioutil"
	"testing"

	"github.com/dollarshaveclub/psst/pkg/storage/testhelper"
	"github.com/hashicorp/vault/vault"
)

// TestVault is a larger test because it needs to start up Vault cluster
func TestVault(t *testing.T) {
	// Launch Vault
	testCluster, err := testhelper.BuildGoodCluster(t)
	if err != nil {
		t.Fatalf("unable to create test cluster: %v", err)
	}
	testCluster.Start()
	defer testCluster.Cleanup()

	core := testCluster.Cores[0].Core
	vClient := testCluster.Cores[0].Client

	vault.TestWaitActive(t, core)
	if err := testCluster.UnsealWithStoredKeys(t); err != nil {
		t.Fatalf("unsealing error: %+v", err)
	}

	// Setup useful variables used across tests
	v := &VaultStore{vClient}
	login := "test-user"
	name := "test-secret"
	secretText := "this is a secret"
	path := v.SecretPath(login, name)

	// Setup a file with the secret
	tmpFile, err := ioutil.TempFile("", "vault-test-")
	if err != nil {
		t.Fatalf("unable to create temporary file")
	}
	filename := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("unable to close temporary file")
	}
	if err := ioutil.WriteFile(filename, []byte(secretText), 755); err != nil {
		t.Fatalf("unable to populate data into temporary file")
	}

	targets := make(map[string]struct{})
	targets[login] = struct{}{}

	// Test Write
	err = v.Write(filename, name, targets)
	if err != nil {
		t.Fatalf("write error: %+v", err)
	}

	// Test List
	secrets, err := v.List(login)
	if err != nil {
		t.Fatalf("unable to list: %+v", err)
	}
	found := false
	for _, secret := range secrets {
		if name == secret {
			found = true
		}
	}
	if !found {
		t.Fatalf("secret not found when listing for user")
	}

	// Test Get
	sec, err := v.Get(path)
	if err != nil {
		t.Fatalf("get error: %+v", err)
	}
	if secretText != sec {
		t.Fatalf("got: %v, expected: %v", secretText, sec)
	}

	// Test Delete
	if err := v.Delete(path); err != nil {
		t.Fatalf("unable to delete secret: %+v", err)
	}

	secrets2, err := v.List(login)
	if err != nil {
		t.Fatalf("unable to list: %+v", err)
	}
	for _, secret := range secrets2 {
		if name == secret {
			t.Fatalf("found deleted secret")
		}
	}

}
