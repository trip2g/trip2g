package mdloader_test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
)

// BenchmarkReloadOneChanged prices the everyday sync: a vault of n notes is
// loaded once, then every iteration reloads it with one note changed, reusing
// unchanged notes through NoteCache the way noteloader does.
//
//	go test ./internal/mdloader -run '^$' -bench ReloadOneChanged -benchmem
func BenchmarkReloadOneChanged(b *testing.B) {
	for _, n := range []int{1000} {
		for _, hosts := range []int{0, 1, 3} {
			b.Run(fmt.Sprintf("notes=%d/hosts=%d", n, hosts), func(b *testing.B) {
				srcs := reloadBenchSources(n, hosts)
				log := logger.DummyLogger{}

				prev, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log, PublicURL: "https://main.test"})
				if err != nil {
					b.Fatal(err)
				}

				rev := 0
				for b.Loop() {
					rev++
					srcs[0].Content = reloadBenchNote(0, n, rev)
					cache := prev
					opts := mdloader.Options{
						Sources:   srcs,
						Log:       &log,
						PublicURL: "https://main.test",
						NoteCache: func(src mdloader.SourceFile) *model.NoteView {
							old, ok := cache.PathMap[src.Path]
							if ok && bytes.Equal(old.Content, src.Content) {
								return old
							}
							return nil
						},
					}
					prev, err = mdloader.Load(opts)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func reloadBenchSources(n, hosts int) []mdloader.SourceFile {
	srcs := make([]mdloader.SourceFile, 0, n+hosts)
	for i := range n {
		srcs = append(srcs, mdloader.SourceFile{
			Path:    "gen/note_" + strconv.Itoa(i) + ".md",
			PathID:  int64(i + 1),
			Content: reloadBenchNote(i, n, 0),
		})
	}
	for h := range hosts {
		content := "---\nfree: true\nroute: host" + strconv.Itoa(h) + ".test/\n---\n# Home\n\n[[note_1]]\n"
		srcs = append(srcs, mdloader.SourceFile{
			Path:    "domains/home_" + strconv.Itoa(h) + ".md",
			PathID:  int64(n + h + 1),
			Content: []byte(content),
		})
	}
	return srcs
}

func reloadBenchNote(i, n, rev int) []byte {
	var sb strings.Builder
	sb.WriteString("---\nfree: true\ntitle: Note " + strconv.Itoa(i) + "\n---\n")
	sb.WriteString("# Note " + strconv.Itoa(i) + "\n\nRevision " + strconv.Itoa(rev) + ".\n\n")
	for k := range 8 {
		sb.WriteString("## Section " + strconv.Itoa(k) + "\n\n")
		sb.WriteString(strings.Repeat("Paragraph text with Кириллица and words. ", 12))
		sb.WriteString("See [[note_" + strconv.Itoa((i*7+k*13)%n) + "]].\n\n")
		sb.WriteString("| a | b |\n|---|---|\n| 1 | [[note_" + strconv.Itoa((i+k+1)%n) + "]] |\n\n")
		sb.WriteString("```go\nfunc f(x int) int { return x * 2 }\n```\n\n")
	}
	return []byte(sb.String())
}
