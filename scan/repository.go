package main

import (
	"github.com/grahambrooks/attribute/neo"
	"github.com/grahambrooks/attribute/scan/tag"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"log"
	"path/filepath"
	"time"
)

type NeoProcessor struct {
	neoClient *neo.NeoClient
	Tags      *tag.Tags
}

func (processor *NeoProcessor) process(repositoryPath string) error {
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

	return processor.neoClient.Transaction(func(client *neo.TransactionalClient) error {
		log.Printf("Scanning %s", repositoryPath)

		client.NewRepository(neo.NewRepositoryRequest{
			Name:   filepath.Base(repositoryPath),
			Origin: remote.Config().URLs[0],
			Tags:   processor.Tags.Repository,
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
						Tags:    processor.Tags.Contributor,
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
