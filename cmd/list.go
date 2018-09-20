package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the set of secrets current available",
	Long:  `List the set of secrets current available`,
	Run: func(cmd *cobra.Command, args []string) {
		login, err := dirState.Whoami()
		if err != nil {
			errorAndExit(fmt.Errorf("unable to get login name: %+v", err), 1)
		}

		if err := listSecrets(login); err != nil {
			errorAndExit(err, 1)
		}

		for _, team := range dirState.GetActiveMemberTeams() {
			if err := listSecrets(team); err != nil {
				errorAndExit(err, 1)
			}
		}
	},
}

func listSecrets(entity string) error {
	secret, err := storageClient.List(entity)
	if err != nil {
		return err
	}

	if secret != nil && len(secret) > 0 {
		fmt.Println(entity)
		fmt.Println("=======")
		for _, s := range secret {
			fmt.Printf("  %v\n", s)
		}
		fmt.Println()
	}

	return nil
}
