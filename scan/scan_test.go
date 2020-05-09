package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func Test(t *testing.T) {
	t.Run("Scanning", func(t *testing.T) {
		var paths = []string{}
		scanner := Scanner{Processor: func(repositoryPath string) error {
			paths = append(paths, repositoryPath)
			return nil
		}}

		scanner.Scan("testdata")
		assert.Equal(t, 1, len(paths))
	})

	t.Run("Scanning logs processor error", func(t *testing.T) {
		scannerCalled := false
		scanner := Scanner{Processor: func(repositoryPath string) error {
			scannerCalled = true
			return fmt.Errorf("yep we failed")
		}}

		var buf bytes.Buffer
		log.SetOutput(&buf)

		scanner.Scan("testdata")

		assert.True(t, scannerCalled)
		assert.Contains(t,buf.String(), "Error processing directory 'testdata/repo': yep we failed")
	})
}
