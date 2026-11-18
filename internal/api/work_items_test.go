package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/charlintosh/lazyado/internal/config"
)

type fakeBoard struct {
	mu      sync.Mutex
	parents map[int]int
	wiql    []int
	asked   []string
}

func (b *fakeBoard) client(t *testing.T, maxItems int) *Client {
	t.Helper()
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", PAT: "test-pat-not-real", Boards: config.Boards{MaxWorkItems: maxItems}})
	c.httpClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		b.mu.Lock()
		b.asked = append(b.asked, r.URL.String())
		b.mu.Unlock()
		var body any
		switch {
		case strings.HasSuffix(r.URL.Path, "/wit/wiql"):
			refs := []map[string]int{}
			for _, id := range b.wiql {
				refs = append(refs, map[string]int{"id": id})
			}
			body = map[string]any{"workItems": refs}
		case strings.HasSuffix(r.URL.Path, "/wit/workitems"):
			var value []map[string]any
			for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
				id, _ := strconv.Atoi(raw)
				value = append(value, map[string]any{"id": id, "fields": map[string]any{
					"System.Id": id, "System.Title": fmt.Sprintf("Item %d", id), "System.Parent": b.parents[id],
				}})
			}
			body = map[string]any{"value": value}
		default:
			t.Errorf("unexpected request %s", r.URL)
		}
		raw, _ := json.Marshal(body)
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})
	return c
}

func (b *fakeBoard) requests(path string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []string
	for _, u := range b.asked {
		if strings.Contains(u, path) {
			out = append(out, u)
		}
	}
	return out
}

func seq(from, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = from + i
	}
	return out
}

func TestQueryWorkItemsAsksForAtMostTheLimit(t *testing.T) {
	b := &fakeBoard{wiql: seq(1, 3)}
	page, err := b.client(t, 3).QueryWorkItems("all", "all", "me", "all")
	if err != nil {
		t.Fatal(err)
	}
	wiql := b.requests("/wit/wiql")
	if len(wiql) != 1 || !strings.Contains(wiql[0], "%24top=3") && !strings.Contains(wiql[0], "$top=3") {
		t.Fatalf("WIQL asked %v, want $top=3", wiql)
	}
	if !page.Truncated || len(page.Items) != 3 {
		t.Fatalf("got %d items, truncated=%v; a full page must be reported as truncated", len(page.Items), page.Truncated)
	}
}

func TestQueryWorkItemsUnderTheLimitIsComplete(t *testing.T) {
	b := &fakeBoard{wiql: seq(1, 2)}
	page, err := b.client(t, 1000).QueryWorkItems("all", "all", "all", "all")
	if err != nil {
		t.Fatal(err)
	}
	if page.Truncated || len(page.Items) != 2 {
		t.Fatalf("got %d items, truncated=%v", len(page.Items), page.Truncated)
	}
}

func TestQueryWorkItemsKeepsTheQueryOrderAcrossBatches(t *testing.T) {
	ids := seq(1, 450)
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	b := &fakeBoard{wiql: ids}
	page, err := b.client(t, 1000).QueryWorkItems("all", "all", "all", "all")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(b.requests("/wit/workitems?")); n != 3 {
		t.Fatalf("fetched in %d batches, want 3 of at most 200", n)
	}
	for i, it := range page.Items {
		if it.ID != ids[i] {
			t.Fatalf("item %d is #%d, want #%d: the most recently changed must stay first", i, it.ID, ids[i])
		}
	}
}

func TestParentTitlesSkipLoadedParentsAndSplitLargeSets(t *testing.T) {
	parents := map[int]int{2: 1}
	ids := []int{1, 2}
	for i := range 250 {
		child := 1000 + i
		parents[child] = 5000 + i
		ids = append(ids, child)
	}
	b := &fakeBoard{wiql: ids, parents: parents}
	page, err := b.client(t, 1000).QueryWorkItems("all", "all", "all", "all")
	if err != nil {
		t.Fatal(err)
	}
	titleRequests := 0
	for _, u := range b.requests("/wit/workitems?") {
		if strings.Contains(u, "System.Title") && !strings.Contains(u, "System.State") {
			titleRequests++
			if strings.Contains(u, "ids=1,") || strings.Contains(u, "ids=1&") {
				t.Fatalf("parent #1 is already loaded and must not be fetched again: %s", u)
			}
		}
	}
	if titleRequests != 2 {
		t.Fatalf("parent titles took %d requests, want 2 (250 parents in chunks of 200)", titleRequests)
	}
	byID := map[int]string{}
	for _, it := range page.Items {
		byID[it.ID] = it.ParentTitle
	}
	if byID[2] != "Item 1" || byID[1249] != "Item 5249" {
		t.Fatalf("parent titles: #2=%q #1249=%q", byID[2], byID[1249])
	}
}
