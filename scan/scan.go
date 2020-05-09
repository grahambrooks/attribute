package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Scanner struct {
	Processor func(repositoryPath string) error
}

func (scanner Scanner) Scan(scanPath string) {
	_ = filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		fileInfo, err := os.Stat(filepath.Join(path, ".git"))

		if err == nil {
			if fileInfo.IsDir() {
				err = scanner.Processor(path)
				if err != nil {
					log.Printf("Error processing directory '%s': %v", path, err)
				}
			}
			return filepath.SkipDir
		}
		return nil
	})
}
