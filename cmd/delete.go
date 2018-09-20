package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVarP(&team, "team", "t", "", "the team currently owning the secret")
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

		entity := login
		if team != "" {
			var ok bool
			entity, ok = dirState.IsTeam(team)
			if !ok {
				errorAndExit(fmt.Errorf("unable to find team '%s'", team), 1)
			}
		}

		path := storageClient.SecretPath(entity, name)
		if err := storageClient.Delete(path); err != nil {
			errorAndExit(err, 1)
		}
	},
}
