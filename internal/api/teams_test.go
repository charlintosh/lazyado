package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/config"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func clientReplying(t *testing.T, status int, body string, seen *http.Request) *Client {
	t.Helper()
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", Team: "web team", PAT: "secret"})
	c.httpClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		*seen = *r
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})
	return c
}

func TestGetCurrentUserReadsTheIdentityBehindThePAT(t *testing.T) {
	var req http.Request
	body := `{"authenticatedUser":{"id":"6f1c2e7a-0000-4b1d-9c55-1f0e2d3c4b5a","providerDisplayName":"Ana Ruiz",` +
		`"properties":{"Account":{"$type":"System.String","$value":"ana@contoso.com"}}},` +
		`"authorizedUser":{"id":"6f1c2e7a-0000-4b1d-9c55-1f0e2d3c4b5a"}}`
	me, err := clientReplying(t, http.StatusOK, body, &req).GetCurrentUser()
	if err != nil {
		t.Fatal(err)
	}
	if me.ID != "6f1c2e7a-0000-4b1d-9c55-1f0e2d3c4b5a" || me.UniqueName != "ana@contoso.com" || me.DisplayName != "Ana Ruiz" {
		t.Fatalf("got %+v", me)
	}
	if req.URL.String() != "https://dev.azure.com/contoso/_apis/connectionData" || !strings.HasPrefix(req.Header.Get("Authorization"), "Basic ") {
		t.Fatalf("asked %s with auth %q, want the PAT sent to connectionData", req.URL, req.Header.Get("Authorization"))
	}
}

func TestGetCurrentUserRejectsAnAnswerWithoutAUser(t *testing.T) {
	var req http.Request
	if _, err := clientReplying(t, http.StatusOK, `{"authenticatedUser":{}}`, &req).GetCurrentUser(); err == nil {
		t.Fatal("an empty identity must be an error, not an anonymous user")
	}
	if _, err := clientReplying(t, http.StatusUnauthorized, `{"message":"bad PAT"}`, &req).GetCurrentUser(); err == nil {
		t.Fatal("a rejected PAT must be an error")
	}
}
