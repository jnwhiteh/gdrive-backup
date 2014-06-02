package main

import (
	"io/ioutil"
	"log"
	"strings"
)

// valueOrFileContents returns the value if it is non-empty, otherwise it
// returns the contents of a file.
func valueOrFileContents(value string, filename string) string {
	if value != "" {
		return value
	}
	slurp, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	return strings.TrimSpace(string(slurp))
}
