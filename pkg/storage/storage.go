package storage

import "github.com/dollarshaveclub/psst/pkg/directory"

const (
	filePrefix = "psst"
)

// Backend gives us basic methods for storing secrets
type Backend interface {
	Delete(string) error
	Get(string) (string, error)
	List(string) ([]string, error)
	GeneratePoliciesAndRoles(string, string, string, string, []directory.Member) error
	SecretPath(string, string) string
	Write(string, string, map[string]struct{}) error
}
