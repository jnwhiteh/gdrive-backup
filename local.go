package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type localFile struct {
	os.FileInfo
	filename string
	md5      string
}

// getLocalFiles returns a list of files that are in the given directory, but
// does not currently recurse through subdirectories.
func getLocalFiles(root string) []localFile {
	var files []localFile
	err := filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			log.Printf("Walked to %s", path)
			if info.IsDir() && path != root {
				return filepath.SkipDir
			}

			if !info.IsDir() {
				md5, err := fileMD5(path)
				if err != nil {
					return err
				}
				files = append(files, localFile{info, path, md5})
				return nil
			}

			return nil
		})

	if err != nil {
		log.Fatalf("Failed when walking %s: %v", root, err)
	}
	return files
}

func fileMD5(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	md5er := md5.New()
	io.Copy(md5er, file)
	return fmt.Sprintf("%x", md5er.Sum(nil)), nil
}
