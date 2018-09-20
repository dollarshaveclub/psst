package cmd

import (
	"fmt"

	"github.com/dollarshaveclub/psst/pkg/directory"
	"github.com/spf13/cobra"
)

var (
	filename string
	members  []string
	name     string
	teams    []string
	ttl      string
)

func init() {
	rootCmd.AddCommand(shareCmd)

	shareCmd.Flags().StringVarP(&filename, "filename", "f", "", "file containing the secret")
	shareCmd.Flags().StringArrayVarP(&members, "member", "m", []string{}, "members to provide secret to (use multiple times for multiple members)")
	shareCmd.Flags().StringVarP(&name, "name", "n", "", "name of the secret")
	shareCmd.Flags().StringArrayVarP(&teams, "team", "t", []string{}, "team to provide secrets to (use multiple times for multiple teams)")

	shareCmd.MarkFlagRequired("filename")
	shareCmd.MarkFlagRequired("name")
}

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share a secret in a user or set of user's drop(s)",
	Long:  `Share a secret in a user or set of user's drop(s)`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(members) == 0 && len(teams) == 0 {
			errorAndExit(fmt.Errorf("you must provide either members and/or teams"), 1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Use a map as an easy way to have a list without duplicates
		targets, err := targets(dirState, members, teams)
		if err != nil {
			errorAndExit(err, 1)
		}

		if err := storageClient.Write(filename, name, targets); err != nil {
			errorAndExit(err, 1)
		}
	},
}

func targets(dirState directory.Backend, members, teams []string) (map[string]struct{}, error) {
	targets := make(map[string]struct{})

	for _, m := range members {
		name, ok := dirState.IsMember(m)
		if !ok {
			return nil, fmt.Errorf("member '%s' does not exist in directory", m)
		}
		targets[name] = struct{}{}
	}

	for _, t := range teams {
		name, ok := dirState.IsTeam(t)
		if !ok {
			return nil, fmt.Errorf("team '%s' does not exist in directory", t)
		}
		targets[name] = struct{}{}
	}
	return targets, nil
}
