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

		secret, err := storageClient.List(login)
		if err != nil {
			errorAndExit(err, 1)
		}

		if len(secret) > 0 {
			fmt.Println("Secrets:")
			for _, s := range secret {
				fmt.Printf("  %v\n", s)
			}
		} else {
			fmt.Println("No secrets found")
		}
	},
}
