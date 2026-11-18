package api

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/charlintosh/lazyado/internal/config"
)

func routedClient(t *testing.T, routes map[string]string) (*Client, func() []string) {
	t.Helper()
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", PAT: "test-pat-not-real"})
	var (
		mu    sync.Mutex
		asked []string
	)
	c.httpClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		asked = append(asked, r.URL.Path)
		mu.Unlock()
		body, ok := routes[r.URL.Path]
		if !ok {
			return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"message":"no route"}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	return c, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), asked...)
	}
}

func TestTypeStatesComeFromTheTypeListInOneCall(t *testing.T) {
	c, asked := routedClient(t, map[string]string{
		"/contoso/web/_apis/wit/workitemtypes": `{"count":2,"value":[
			{"name":"Bug","states":[{"name":"New","color":"b2b2b2","category":"Proposed"},{"name":"Done","color":"339933","category":"Completed"}]},
			{"name":"Task","states":[{"name":"To Do","color":"b2b2b2","category":"Proposed"}]}]}`,
	})
	got, err := c.GetAllWorkItemTypeStates()
	if err != nil {
		t.Fatal(err)
	}
	if n := len(asked()); n != 1 {
		t.Fatalf("made %d requests %v, want 1", n, asked())
	}
	if len(got["Bug"]) != 2 || got["Bug"][1].Category != "Completed" || got["Task"][0].Name != "To Do" {
		t.Fatalf("states %+v", got)
	}
}

func TestTypeStatesFallBackPerTypeWhenTheListLacksThem(t *testing.T) {
	c, asked := routedClient(t, map[string]string{
		"/contoso/web/_apis/wit/workitemtypes": `{"count":2,"value":[
			{"name":"Bug","states":[{"name":"New","color":"b2b2b2","category":"Proposed"}]},
			{"name":"Epic","states":[{"name":"New","color":"b2b2b2"}]}]}`,
		"/contoso/web/_apis/wit/workitemtypes/Epic/states": `{"count":1,"value":[{"name":"New","color":"b2b2b2","stateCategory":"Proposed"}]}`,
	})
	got, err := c.GetAllWorkItemTypeStates()
	if err != nil {
		t.Fatal(err)
	}
	if n := len(asked()); n != 2 {
		t.Fatalf("made %d requests %v, want the list plus one fallback for Epic", n, asked())
	}
	if got["Epic"][0].Category != "Proposed" || got["Bug"][0].Category != "Proposed" {
		t.Fatalf("states %+v", got)
	}
}

func TestUpdatingAWorkItemReturnsTheSavedItem(t *testing.T) {
	c, _ := routedClient(t, map[string]string{
		"/contoso/web/_apis/wit/workitems/1052": `{"id":1052,"rev":7,"fields":{"System.Id":1052,"System.Title":"Email receipt","System.State":"Done","System.Parent":1043}}`,
	})
	item, err := c.UpdateWorkItemState(1052, "Done")
	if err != nil {
		t.Fatal(err)
	}
	if item == nil || item.ID != 1052 || item.State != "Done" || item.ParentID != 1043 {
		t.Fatalf("got %+v, want the item as the server saved it", item)
	}
}

func TestASaveWithAnUnreadableReplyStillSucceeds(t *testing.T) {
	c, _ := routedClient(t, map[string]string{"/contoso/web/_apis/wit/workitems/1052": `<html>`})
	item, err := c.AssignWorkItem(1052, "lena@example.com")
	if err != nil || item != nil {
		t.Fatalf("got %+v, %v; the save succeeded, so no error and no item (the caller reloads)", item, err)
	}
}
