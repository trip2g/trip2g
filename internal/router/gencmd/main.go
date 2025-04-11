package main

import (
	"log"
	"os"
	"path/filepath"
	"trip2g/internal/router/gencmd/views"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=./views

func main() {
	var cases []string

	const casePath = "../case"

	entries, err := os.ReadDir(casePath)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			casePath := filepath.Join(casePath, entry.Name(), "endpoint.go")

			_, statErr := os.Stat(casePath)
			if statErr == nil {
				cases = append(cases, entry.Name())
			}
		}
	}

	f, err := os.Create("./endpoints_gen.go")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	views.WriteCode(f, cases)
}
