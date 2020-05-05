package main

import (
	"fmt"
	"github.com/grahambrooks/attribute/neo"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	username string
	password string
	neoHost  string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&username, "user", "u", "attribute", "Neo4j username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "attribute", "Neo4j password")
	rootCmd.PersistentFlags().StringVarP(&neoHost, "neo", "n", "http://localhost:7474", "Neo4j base URL")
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
		for _, p := range args {
			scanPath(p, &client)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func scanPath(p string, client *neo.NeoClient) {
	_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		fileInfo, err := os.Stat(filepath.Join(path, ".git"))

		if err == nil {
			if fileInfo.IsDir() {
				_ = processRepository(path, client)
			}
			return filepath.SkipDir
		}
		return nil
	})
}

func processRepository(repositoryPath string, neoClient *neo.NeoClient) error {
	today := time.Now()

	threeMonthsAgo := today.AddDate(0, -6, 0)

	r, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return err
	}

	remote, err := r.Remote("origin")
	if err != nil {
		return err
	}

	return neoClient.Transaction(func(client *neo.TransactionalClient) error {
		log.Printf("Scanning %s", repositoryPath)

		client.NewRepository(neo.NewRepositoryRequest{
			Name:   filepath.Base(repositoryPath),
			Origin: remote.Config().URLs[0],
		})

		ref, err := r.Head()
		if err != nil {
			log.Printf("Error: %v\n", err)
			return err
		}

		cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			log.Printf("Error: %v\n", err)
			return err
		}

		seen := make(map[string]bool)
		err = cIter.ForEach(func(c *object.Commit) error {
			_, exists := seen[c.Author.Email]
			if !exists {
				if c.Author.When.After(threeMonthsAgo) {
					request := neo.NewContributorRequest{
						Origin:  filepath.Base(repositoryPath),
						Name:    c.Author.Name,
						Email:   c.Author.Email,
						When:    c.Author.When,
						Message: c.Message,
					}
					client.NewContributor(request)
					seen[c.Author.Email] = true
				}
			}
			return nil
		})
		return nil
	})
}
