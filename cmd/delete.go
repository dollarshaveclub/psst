package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a secret from the current user's drop location",
	Long:  `Delete a secret from the current user's drop location`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		login, err := dirState.Whoami()
		if err != nil {
			errorAndExit(fmt.Errorf("unable to get login name: %+v", err), 1)
		}

		// cobra.ExactArgs(1) makes sure we have a single argument
		name := args[0]

		path := storageClient.SecretPath(login, name)
		if err := storageClient.Delete(path); err != nil {
			errorAndExit(err, 1)
		}
	},
}
