package mdloader_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
)

// BenchmarkValidateDocs loads the repo's docs/ vault (~850 notes) with every
// note listing the given demo validators, to price validation per sync.
//
//	go test ./internal/mdloader -run '^$' -bench ValidateDocs -benchmem
func BenchmarkValidateDocs(b *testing.B) {
	cases := []struct {
		name       string
		validators []string
	}{
		{"none", nil},
		{"required_frontmatter", []string{"required_frontmatter"}},
		{"session_has_image", []string{"session_has_image"}},
		{"all_7", []string{
			"required_frontmatter", "seo", "only_h3", "sections_order",
			"session_has_image", "media_hygiene", "external_links",
		}},
	}

	for _, c := range cases {
		srcs := docsSources(b, c.validators)
		for _, cached := range []bool{false, true} {
			name := c.name
			var cache *mdloader.ValidateCache
			if cached {
				// Warmed up below: each iteration is a sync where nothing changed.
				name += "/cached"
				cache = mdloader.NewValidateCache()
			}
			b.Run(name, func(b *testing.B) {
				log := logger.DummyLogger{}
				if cache != nil {
					_, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log, Validate: true, ValidateCache: cache})
					if err != nil {
						b.Fatal(err)
					}
				}
				for b.Loop() {
					_, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log, Validate: true, ValidateCache: cache})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func docsSources(b *testing.B, validators []string) []mdloader.SourceFile {
	b.Helper()

	var srcs []mdloader.SourceFile
	add := func(root, prefix string, withValidate bool) {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return err
			}
			content, readErr := os.ReadFile(p)
			if readErr != nil {
				return readErr
			}
			if withValidate && len(validators) > 0 {
				content = withValidateField(content, validators)
			}
			rel, _ := filepath.Rel(root, p)
			srcs = append(srcs, mdloader.SourceFile{Path: prefix + filepath.ToSlash(rel), Content: content})
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	add("../../docs", "", true)
	add("testdata/validate/validators", "validators/", false)

	return srcs
}

func withValidateField(content []byte, validators []string) []byte {
	field := "validate:\n"
	for _, v := range validators {
		field += "  - \"[[validators/" + v + "]]\"\n"
	}

	s := string(content)
	if rest, ok := strings.CutPrefix(s, "---\n"); ok && !strings.Contains(firstLines(rest), "validate:") {
		return []byte("---\n" + field + rest)
	}
	return []byte("---\n" + field + "---\n" + s)
}

func firstLines(s string) string {
	head, _, _ := strings.Cut(s, "\n---")
	return head
}
