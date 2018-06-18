package testhelper

import (
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	bplugin "github.com/hashicorp/vault/builtin/plugin"
	"github.com/hashicorp/vault/helper/logging"
	vaulthttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/physical/inmem"
	"github.com/hashicorp/vault/vault"
)

func BuildGoodCluster(t *testing.T) (*vault.TestCluster, error) {
	l := logging.NewVaultLogger(hclog.NoLevel)
	inm, err := inmem.NewInmemHA(nil, l)
	if err != nil {
		return nil, err
	}
	testCluster := vault.NewTestCluster(t, &vault.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"plugin": bplugin.Factory,
		},
		Physical: inm,
	},
		&vault.TestClusterOptions{
			HandlerFunc: vaulthttp.Handler,
			NumCores:    1,
		},
	)
	return testCluster, nil
}
