package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/models"
)

func largeFinder(tb testing.TB) App {
	items := make([]models.GitItem, 0, models.MaxIndexedItems)
	for i := range models.MaxIndexedItems {
		items = append(items, models.GitItem{Path: fmt.Sprintf("/services/svc%03d/internal/pkg%02d/handler_file_%05d.go", i%400, i%37, i)})
	}
	a := openFilesHub(tb, 200, 50)
	a, msgs := step(tb, a, ctrl('p'))
	a = feed(tb, a, msgs...)
	a = feed(tb, a, fileIndexLoadedMsg{key: a.fileIndexKey(), gen: a.code.indexes.gen, index: models.NewFileIndex(items, "abc", false)})
	for _, r := range "handler" {
		a, _ = step(tb, a, press(string(r)))
	}
	return feed(tb, a, modals.FileFinderPreviewMsg{Path: "/services/svc000/internal/pkg00/handler_file_00000.go", Text: strings.Repeat("func Handle(w http.ResponseWriter, r *http.Request) {}\n", 300)})
}

func BenchmarkFileFinderCursorMove(b *testing.B) {
	a := largeFinder(b)
	down, up := special(tea.KeyDown), special(tea.KeyUp)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := down
		if i%40 >= 20 {
			k = up
		}
		m, _ := a.Update(k)
		a = m.(App)
		_ = a.View()
	}
}
