package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func sampleTags() []models.Tag {
	return []models.Tag{
		{Name: "perf-baseline", ObjectID: "45532e0", CommitID: "45532e0"},
		{Name: "v2026.10.2", ObjectID: "t102", CommitID: "078d719"},
		{Name: "v2026.11.0", ObjectID: "t110", CommitID: "7c1e0b4"},
	}
}

func openTagsHub(t testing.TB, width, height int) App {
	t.Helper()
	a := openFilesHub(t, width, height)
	a = feed(t, a, press("T"))
	if !a.inTags() || !a.tags.loading {
		t.Fatal("T must open the Tags hub and load the tags")
	}
	return feed(t, a, tagsLoadedMsg{seq: a.tags.seq, tags: sampleTags()})
}

func TestTagsHubListsTagsAndPreviewsAnnotatedOnes(t *testing.T) {
	a := openTagsHub(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Repos / Tags / storefront-api", "New tag", "TAGS", "Tags 3", "perf-baseline", "Lightweight", "v2026.11.0", "Annotated", "7c1e0b4",
		"A lightweight tag has no message"} {
		if !strings.Contains(out, want) {
			t.Errorf("Tags hub is missing %q:\n%s", want, out)
		}
	}
	a = feed(t, a, press("G"))
	if a.tags.detailFor != "t110" {
		t.Fatalf("an annotated tag's details must load for the preview, got %q", a.tags.detailFor)
	}
	a = feed(t, a, tagDetailMsg{objectID: "t110", detail: models.TagDetail{Name: "v2026.11.0", Message: "November release: regional tax", Tagger: "Tomás Ruiz", Date: time.Now()}})
	out = ansi.Strip(a.render())
	for _, want := range []string{"November release: regional tax", "Tagger", "Tomás Ruiz"} {
		if !strings.Contains(out, want) {
			t.Errorf("tag preview is missing %q:\n%s", want, out)
		}
	}
	a = feed(t, a, press("k"), tagDetailMsg{objectID: "t102", err: errors.New("TF401019: no access")})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "TF401019") || a.tagsLoading() {
		t.Errorf("a failed detail must show its error and stop loading:\n%s", out)
	}
}

func TestNewTagFormRequiresADescriptionAndCreatesAnAnnotatedTag(t *testing.T) {
	a := openTagsHub(t, 160, 45)
	a = feed(t, a, press("n"))
	form := topModal[modals.RefForm](t, a)
	if out := ansi.Strip(form.View()); !strings.Contains(out, "New tag") || !strings.Contains(out, "Description") || !strings.Contains(out, "annotated tag") {
		t.Fatalf("got:\n%s", out)
	}
	a = feed(t, a, tea.PasteMsg{Content: "v2026.11.0"}, ctrl('s'))
	if out := ansi.Strip(topModal[modals.RefForm](t, a).View()); !strings.Contains(out, "a tag named v2026.11.0 already exists") {
		t.Fatalf("an existing tag must be refused:\n%s", out)
	}
	a = feed(t, a, press("1"), ctrl('s'))
	if out := ansi.Strip(topModal[modals.RefForm](t, a).View()); !strings.Contains(out, "enter a description") {
		t.Fatalf("the description is required:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyTab), special(tea.KeyTab), special(tea.KeyTab), tea.PasteMsg{Content: "Cart rounding hotfix"})
	_, msgs := step(t, a, ctrl('s'))
	req, ok := find[modals.CreateTagRequestMsg](msgs)
	if !ok || req.Name != "v2026.11.01" || req.Message != "Cart rounding hotfix" || req.Base != models.BranchVersion("main") {
		t.Fatalf("got %+v from %v", req, msgs)
	}
	a = feed(t, a, req)
	if !a.loading {
		t.Fatal("creating must show progress")
	}
	a = feed(t, a, refChangedMsg{notice: "Tag v2026.11.01 created", name: "v2026.11.01", tag: true})
	if a.modals.open() || !a.tags.loading {
		t.Fatal("success must close the form and reload the tags")
	}
	a = feed(t, a, tagsLoadedMsg{seq: a.tags.seq, tags: append(sampleTags(), models.Tag{Name: "v2026.11.01", ObjectID: "t111", CommitID: "a3f9c21"})})
	if tag := a.tags.panel.Selected(); tag == nil || tag.Name != "v2026.11.01" {
		t.Fatalf("the new tag must be selected, got %+v", tag)
	}
}

func TestTagShortcutsBrowseFilesAndCommitsAtTheTag(t *testing.T) {
	a := openTagsHub(t, 160, 45)
	a = feed(t, a, press("G"))
	files := feed(t, a, special(tea.KeyEnter))
	if !files.inFiles() || files.code.version != (models.GitVersion{Kind: models.VersionTag, Name: "v2026.11.0"}) {
		t.Fatalf("enter must browse the tag's files, got %+v", files.code.version)
	}
	commits := feed(t, a, press("H"))
	if !commits.inCommits() || commits.code.version.Name != "v2026.11.0" {
		t.Fatalf("H must show the commits up to the tag, got %+v", commits.code.version)
	}
	if got := a.tagWebURL("v2026.11.0"); !strings.Contains(got, "version=GTv2026.11.0") {
		t.Errorf("got %s", got)
	}
}

func TestTagsHubFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {90, 26}, {70, 22}, {50, 14}} {
		a := openTagsHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		n := feed(t, a, press("n"))
		assertFits(t, n.render(), sz[0], sz[1])
	}
}
