package api

import (
	"strings"
	"testing"
)

func TestCommentsKeepLikesAndUseTheReactionsAPI(t *testing.T) {
	var seen []recorded
	body := `{"count":2,"comments":[` +
		`{"id":7,"text":"Ship it","createdBy":{"id":"u1","displayName":"Marco Lima"},"reactions":[{"type":"like","count":3,"isCurrentUserEngaged":true},{"type":"heart","count":1}]},` +
		`{"id":8,"text":"Agreed","createdBy":{"id":"u2","displayName":"Priya Nair"}}]}`
	c := clientRouting(t, map[string]string{"/comments": body}, &seen)
	comments, err := c.GetComments(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 || comments[0].Likes != 3 || !comments[0].LikedByMe || comments[1].Likes != 0 || comments[1].LikedByMe {
		t.Fatalf("got %+v, want only the like reaction counted", comments)
	}
	if err := c.LikeComment(42, 7, true); err != nil {
		t.Fatal(err)
	}
	if err := c.LikeComment(42, 7, false); err != nil {
		t.Fatal(err)
	}
	for i, method := range []string{"PUT", "DELETE"} {
		r := seen[len(seen)-2+i]
		if r.method != method || !strings.Contains(r.url, "/wit/workItems/42/comments/7/reactions/like?api-version=7.1-preview.1") {
			t.Errorf("request %d: got %s %s, want %s on the like reaction", i, r.method, r.url, method)
		}
	}
}
