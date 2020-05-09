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

		repositoryRequest := neo.NewRepositoryRequest{
			Name:   filepath.Base(repositoryPath),
			Origin: remote.Config().URLs[0],
			Tags:   processor.Tags.Repository,
		}

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

		contributorRequests := make(map[string]*neo.NewContributorRequest, 0)

		err = cIter.ForEach(func(c *object.Commit) error {
			if c.Author.When.After(threeMonthsAgo) {
				repositoryRequest.CommitCount++
				r, exists := contributorRequests[contributorKey(c)]
				if exists {
					r.CommitCount++
				} else {
					contributorRequests[c.Author.Email] = &neo.NewContributorRequest{
						Origin:      filepath.Base(repositoryPath),
						Name:        c.Author.Name,
						Email:       c.Author.Email,
						CommitCount: 1,
						Tags:        processor.Tags.Contributor,
						When:        c.Author.When,
						Message:     c.Message,
					}
				}
			}
			return nil
		})

		client.NewRepository(repositoryRequest)

		for _, request := range contributorRequests {
			client.NewContributor(*request)
		}

		return nil
	})
}

func contributorKey(c *object.Commit) string {
	return c.Author.Email
}
