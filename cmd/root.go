package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/dollarshaveclub/psst/pkg/directory"
	"github.com/dollarshaveclub/psst/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	directoryBackend string
	storageBackend   string
	// CompiledDirectory points and the directory compiled into the binary
	CompiledDirectory = ""
	// CompiledStorage points to the storage backend compiled into the binary
	CompiledStorage = ""
	// Org is the default organization to use
	Org = ""

	dirState      directory.Backend
	storageClient storage.Backend
)

func init() {
	rootCmd.PersistentFlags().StringVar(&Org, "org", Org, "organization for the directory")
	rootCmd.PersistentFlags().StringVar(&directoryBackend, "directory-backend", CompiledDirectory, "directory to use to find members and teams (e.g. GitHub)")
	rootCmd.PersistentFlags().StringVar(&storageBackend, "storage-backend", CompiledStorage, "storage backend to use for secrets (e.g. Vault)")
}

var rootCmd = &cobra.Command{
	Use:   "psst",
	Short: "Psst is a tool for securely sharing secrets inside of your organization",
	Long:  `Psst uses GitHub and Hashicorp to share secrets with one or more users`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error

		switch directoryBackend {
		case "github":
			dirState, err = directory.NewGitHub(Org)
			if err != nil {
				errorAndExit(fmt.Errorf("unable to get directory client: %+v", err), 1)
			}
		default:
			errorAndExit(errors.New("you must provide a valid directory backend"), 1)
		}

		switch storageBackend {
		case "vault":
			storageClient, err = storage.NewVault()
			if err != nil {
				errorAndExit(fmt.Errorf("unable to get storage client: %+v", err), 1)
			}
		default:
			errorAndExit(errors.New("you must provide a valid storage backend"), 1)
		}
	},
}

// Execute is the entrypoint for running the different commands of psst
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func errorAndExit(err error, code int) {
	fmt.Fprintf(os.Stderr, "%+v\n", err)
	os.Exit(code)
}
