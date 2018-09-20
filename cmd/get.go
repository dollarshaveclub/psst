package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	team string
)

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringVarP(&team, "team", "t", "", "the team currently owning the secret")
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a secret from the current user's drop",
	Long:  `Get a secret from the current user's drop`,
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
				errorAndExit(fmt.Errorf("could not find team '%s'", team), 1)
			}
		}
		path := storageClient.SecretPath(entity, name)

		data, err := storageClient.Get(path)
		if err != nil {
			errorAndExit(err, 1)
		}

		fmt.Println(data)
	},
}
