package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"trip2g/internal/router/gencmd/views"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=./views

func main() {
	cases := []views.CaseItem{}
	cases = append(cases, scanDir("")...)
	cases = append(cases, scanDir("admin")...)

	f, err := os.Create("./endpoints_gen.go")
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			log.Fatal(closeErr)
		}
	}()

	views.WriteCode(f, cases)
}

func scanDir(localDir string) []views.CaseItem {
	var cases []views.CaseItem

	dirPath := "../case"
	importPath := "trip2g/internal/case"

	if localDir != "" {
		dirPath = filepath.Join("../case", localDir)
		importPath = filepath.Join("trip2g/internal/case", localDir)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			casePath := filepath.Join(dirPath, entry.Name(), "endpoint.go")

			_, statErr := os.Stat(casePath)
			if statErr == nil {
				cs := views.CaseItem{
					PackageName:  entry.Name(),
					PackageAlias: strings.ReplaceAll(entry.Name(), "/", "") + entry.Name(),
					ImportPath:   filepath.Join(importPath, entry.Name()),
				}

				cases = append(cases, cs)
			}
		}
	}

	return cases
}
