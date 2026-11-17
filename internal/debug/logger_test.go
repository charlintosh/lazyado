package debug

import "testing"

func TestRedactURLDropsIdentities(t *testing.T) {
	got := RedactURL("https://user:pw@vsrm.example.com/contoso/web/_apis/release/approvals?assignedToFilter=dana%40example.com&api-version=7.1#x")
	want := "https://vsrm.example.com/contoso/web/_apis/release/approvals"
	if got != want {
		t.Fatalf("RedactURL = %q, want %q", got, want)
	}
}
