package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

const (
	screenshotWidth  = 120
	screenshotHeight = 34
)

const screenshotReadme = "# Storefront API\n\nCart, pricing and tax for the storefront.\n\n## Running locally\n\n```sh\ngo run ./cmd/api\n```\n\n- `GET /cart/{id}` returns a cart with its totals\n- `POST /checkout` places an order\n"

func screenshotComments() []models.Comment {
	now := time.Now()
	return []models.Comment{
		{ID: 1, Text: "Guest orders still need an email for the receipt, so the field stays required.", Format: "markdown", CreatedBy: "Dana Reyes", CreatedDate: now.Add(-26 * time.Hour)},
		{ID: 2, Text: "Agreed. The session token lands in the cart API first, then the sign-in step.", Format: "markdown", CreatedBy: "Marco Lima", CreatedDate: now.Add(-3 * time.Hour), Likes: 1},
	}
}

type screenshot struct {
	name string
	open func(t *testing.T) App
}

func screenshots() []screenshot {
	w, h := screenshotWidth, screenshotHeight
	return []screenshot{
		{"backlog", func(t *testing.T) App { return feed(t, newTestApp(t, w, h), press("j"), press("l"), press("j")) }},
		{"board", func(t *testing.T) App { return feed(t, newTestApp(t, w, h), press("B")) }},
		{"sprint", func(t *testing.T) App { return taskboardApp(t, w, h) }},
		{"work-item", func(t *testing.T) App {
			a := newTestApp(t, w, h)
			item, _ := a.loadedItem(1043)
			a = feed(t, a, panels.ViewWorkItemMsg{Item: item})
			return feed(t, a, commentsLoadedMsg{workItemID: 1043, comments: screenshotComments()})
		}},
		{"files", func(t *testing.T) App {
			a := openFilesHub(t, w, h)
			return feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/README.md", text: screenshotReadme})
		}},
		{"commits", func(t *testing.T) App { return openCommitsHub(t, w, h) }},
		{"branches", func(t *testing.T) App { return openBranchesHub(t, w, h) }},
		{"tags", func(t *testing.T) App { return openTagsHub(t, w, h) }},
		{"pull-requests", func(t *testing.T) App { return openPullRequestHub(t, w, h) }},
		{"pull-request-review", func(t *testing.T) App { return openReview(t, w, h) }},
		{"pull-request-files", func(t *testing.T) App { return openFiles(t, w, h) }},
		{"pipelines", func(t *testing.T) App { return openPipelinesHub(t, w, h) }},
		{"pipeline-run", func(t *testing.T) App { return openRunView(t, w, h) }},
		{"releases", func(t *testing.T) App { return openReleasesHub(t, w, h) }},
		{"release", func(t *testing.T) App { return openReleaseView(t, w, h) }},
		{"help", func(t *testing.T) App { return feed(t, newTestApp(t, w, h), press("?")) }},
	}
}

func TestRenderScreenshots(t *testing.T) {
	dir := os.Getenv("LAZYADO_SCREENSHOTS_DIR")
	if dir == "" {
		t.Skip("set LAZYADO_SCREENSHOTS_DIR to write a frame of every view")
	}
	for _, s := range screenshots() {
		t.Run(s.name, func(t *testing.T) {
			a := s.open(t)
			if err := os.WriteFile(filepath.Join(dir, s.name+".ans"), []byte(a.render()), 0o644); err != nil {
				t.Fatal(err)
			}
		})
	}
}
