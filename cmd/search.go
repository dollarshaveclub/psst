package cmd

import (
	"fmt"
	"strings"

	"github.com/dollarshaveclub/psst/pkg/directory"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for a member or team in your GitHub organization",
	Long:  `Search for a member or team in your GitHub organization. You may provide a search team or leave it blank to see all available members and teams`,
	Run: func(cmd *cobra.Command, args []string) {
		matches := search(dirState, args)

		if len(matches.Members) > 0 {
			fmt.Println("Members:")
			for _, u := range matches.Members {
				fmt.Printf("\t%s (%s)\n", u.Login, u.Name)
			}
		}
		if len(matches.Members) > 0 && len(matches.Teams) > 0 {
			fmt.Println()
		}
		if len(matches.Teams) > 0 {
			fmt.Println("Teams:")
			for _, t := range matches.Teams {
				fmt.Printf("\t%s\n", t.Name)
			}
		}
	},
}

func search(dirState directory.Backend, args []string) directory.Matches {
	lookup := "*"
	if len(args) > 0 {
		lookup = strings.Join(args, " ")
	}

	return dirState.GetMatches(lookup)
}
