package github

import (
	"encoding/json"
	"testing"
)

// TestDeployKeyIDDecodesExactly guards against a live-test-caught bug:
// decoding gh's bare JSON number id as `any` hits Go's default
// float64 conversion and mangles large ids into scientific notation
// (160723208 -> "1.60723208e+08"), which then 404s on delete.
func TestDeployKeyIDDecodesExactly(t *testing.T) {
	var keys []deployKey
	payload := `[{"id":160723208,"title":"hcode-inzu-goport-test-inzu-8779f6"}]`
	if err := json.Unmarshal([]byte(payload), &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("got %d keys, want 1", len(keys))
	}
	if got, want := keys[0].ID, int64(160723208); got != want {
		t.Errorf("ID = %d, want %d", got, want)
	}
}

func TestParseRepoURL(t *testing.T) {
	cases := []struct {
		url       string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{url: "git@github.com:owner/repo.git", wantOwner: "owner", wantName: "repo"},
		{url: "git@github.com:owner/repo", wantOwner: "owner", wantName: "repo"},
		{url: "ssh://git@github.com/owner/repo.git", wantOwner: "owner", wantName: "repo"},
		{url: "git@github.com:owner/repo/", wantOwner: "owner", wantName: "repo"},
		{url: "https://github.com/owner/repo.git", wantErr: true},
		{url: "owner/repo", wantErr: true},
	}
	for _, c := range cases {
		got, err := ParseRepoURL(c.url)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseRepoURL(%q) = %+v, want error", c.url, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRepoURL(%q) unexpected error: %v", c.url, err)
			continue
		}
		if got.Owner != c.wantOwner || got.Name != c.wantName {
			t.Errorf("ParseRepoURL(%q) = %+v, want owner=%q name=%q", c.url, got, c.wantOwner, c.wantName)
		}
	}
}
