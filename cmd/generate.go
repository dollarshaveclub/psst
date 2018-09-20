package cmd

import (
	"fmt"
	"path"

	"github.com/spf13/cobra"
)

var (
	policyDir   string
	roleDir     string
	allTeam     string
	defaultTeam = "all"
)

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVar(&policyDir, "policy-dir", "", "directory for the generated policy files")
	generateCmd.Flags().StringVar(&roleDir, "role-dir", "", "directory for the generated roles")
	generateCmd.Flags().StringVar(&allTeam, "default-team", defaultTeam, "team containing every member of your organization")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate polices missing for new users in GitHub",
	Long:  `Generate polices missing for new users in GitHub`,
	Run: func(cmd *cobra.Command, args []string) {
		entities := []string{}
		for _, m := range dirState.GetMembers() {
			entities = append(entities, m.Login)
		}
		if err := storageClient.GeneratePoliciesAndRoles(directoryBackend, path.Join(roleDir, "users"), policyDir, allTeam, entities); err != nil {
			errorAndExit(fmt.Errorf("unable to generate policies and roles: %v", err), 1)
		}

		entities = []string{}
		for _, t := range dirState.GetTeams() {
			entities = append(entities, t.Name)
		}
		if err := storageClient.GeneratePoliciesAndRoles(directoryBackend, path.Join(roleDir, "teams"), policyDir, allTeam, entities); err != nil {
			errorAndExit(fmt.Errorf("unable to generate policies and roles: %v", err), 1)
		}
	},
}
