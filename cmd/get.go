package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getCmd)
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

		path := storageClient.SecretPath(login, name)

		data, err := storageClient.Get(path)
		if err != nil {
			errorAndExit(err, 1)
		}

		fmt.Println(data)
	},
}
