package ui

import (
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/cache"
	"github.com/charlintosh/lazyado/internal/models"
)

func TestEvictedFileTextsAreFetchedAgain(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	f := &a.code.files
	f.texts = cache.New[string, fileText](100, func(t fileText) int { return len(t.text) })

	a = feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/README.md", text: strings.Repeat("r", 60)})
	f = &a.code.files
	if !f.haveFile("/README.md") || f.requested[fileKey+"/README.md"] {
		t.Fatal("a loaded file is cached and no longer in flight")
	}
	if cmd := a.requestFile("/README.md"); cmd != nil {
		t.Fatal("a cached file must not be fetched again")
	}

	a = feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/go.mod", text: strings.Repeat("g", 60)})
	f = &a.code.files
	if _, cached := f.texts.Get("/README.md"); cached {
		t.Fatal("the older file must be evicted once the budget is exceeded")
	}
	if !f.requested[fileKey+"/README.md"] {
		t.Fatal("the evicted README is still on screen, so it must be fetched again right away")
	}
	if cmd := a.requestFile("/README.md"); cmd != nil {
		t.Fatal("a file already in flight must not be requested twice")
	}
}

func TestFileTextBudgetAlwaysFitsTheLargestAllowedFile(t *testing.T) {
	f := newFilesState(newTestApp(t, 80, 24).code.files.tree, newTestApp(t, 80, 24).code.files.content, 64<<20)
	f.texts.Put("/big.bin", fileText{text: strings.Repeat("x", 64<<20)})
	if _, ok := f.texts.Get("/big.bin"); !ok {
		t.Fatal("a file as large as diff.max_file_kb allows must fit in the cache, or it would be fetched in a loop")
	}
}

func TestStageCacheDropsRunsOutOfView(t *testing.T) {
	s := pipelineState{stageCache: map[int]stageEntry{}, previewRun: 7}
	for id := range stagesKept + 50 {
		s.stageCache[id] = stageEntry{}
	}
	s.stageCache[9999] = stageEntry{loading: true}
	s.pruneStages([]models.PipelineRun{{ID: 3}})
	if len(s.stageCache) != 3 {
		t.Fatalf("kept %d stage entries, want the visible run, the previewed run and the one still loading", len(s.stageCache))
	}
	for _, id := range []int{3, 7, 9999} {
		if _, ok := s.stageCache[id]; !ok {
			t.Errorf("run %d must be kept", id)
		}
	}
}

func TestStageCacheUnderTheCapIsLeftAlone(t *testing.T) {
	s := pipelineState{stageCache: map[int]stageEntry{1: {}, 2: {}}}
	s.pruneStages(nil)
	if len(s.stageCache) != 2 {
		t.Fatal("a small cache must not be pruned")
	}
}
