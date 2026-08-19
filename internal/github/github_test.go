package github

import "testing"

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
