package main

import (
	"fmt"
	"github.com/grahambrooks/attribute/neo"
	"github.com/grahambrooks/attribute/scan/tag"
	"github.com/spf13/cobra"
	"os"
)

var (
	username string
	password string
	neoHost  string
	tags     []string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&username, "user", "u", "attribute", "Neo4j username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "attribute", "Neo4j password")
	rootCmd.PersistentFlags().StringVarP(&neoHost, "neo", "n", "http://localhost:7474", "Neo4j base URL")
	rootCmd.PersistentFlags().StringArrayVarP(&tags, "tag", "t", []string{}, "Node tags in the format contributor:tag repository:tag")
}

var rootCmd = &cobra.Command{
	Use:   "attribute",
	Short: "Git repository scanner to generate pretty contribution graphs in Neo4j",
	Long:  `Git repository scanner to generate pretty contribution graphs in Neo4j`,
	Example: `
attribute /dev

Scans /dev for git repositories. When found reads the commit history for the last 3 months and creates relationships between the repository and the contributors
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := neo.NewNeoClient(neo.ClientOptions{
			Host:     neoHost,
			Username: username,
			Password: password,
		})
		tags, err := tag.Parse(tags)
		if err != nil {
			fmt.Printf("Error: %v", err)
		}
		processor := NeoProcessor{neoClient: &client, Tags: tags}
		scanner := Scanner{Processor: processor.process}
		for _, p := range args {
			scanner.Scan(p)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
