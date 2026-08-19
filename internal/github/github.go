// Package github runs everything that talks to GitHub, and only ever on
// the local machine — never shelled out to from the box itself. `gh`
// must already be authenticated (`gh auth status`); hcode doesn't
// manage that credential.
package github

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dikee/hetzner-code/internal/run"
)

var sshURLRe = regexp.MustCompile(
	`^(?:git@|ssh://git@)github\.com[:/](?P<owner>[^/]+)/(?P<repo>[^/.]+?)(?:\.git)?/?$`,
)

// RepoRef names an owner/repo pair.
type RepoRef struct {
	Owner string
	Name  string
}

// Slug returns "owner/name".
func (r RepoRef) Slug() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// ParseRepoURL extracts owner/repo from a GitHub SSH URL.
func ParseRepoURL(url string) (RepoRef, error) {
	match := sshURLRe.FindStringSubmatch(strings.TrimSpace(url))
	if match == nil {
		return RepoRef{}, run.Errorf(
			"'%s' doesn't look like a GitHub SSH URL (expected git@github.com:owner/repo.git)", url,
		)
	}
	names := sshURLRe.SubexpNames()
	var owner, name string
	for i, n := range names {
		switch n {
		case "owner":
			owner = match[i]
		case "repo":
			name = match[i]
		}
	}
	return RepoRef{Owner: owner, Name: name}, nil
}

type deployKey struct {
	// gh emits a bare JSON number here. Decoding it as `any` would hit
	// Go's default float64 conversion and mangle large ids into
	// scientific notation (e.g. 160723208 -> "1.60723208e+08") —
	// caught by a live end-to-end run, where the resulting delete call
	// 404'd. int64 keeps it exact.
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// AddDeployKey registers publicKeyPath as a deploy key on repo, returns
// its id.
//
// `gh repo deploy-key add` doesn't print the id, so this looks it back
// up by the (unique, caller-supplied) title immediately after adding.
func AddDeployKey(repo RepoRef, publicKeyPath, title string, write bool) (string, error) {
	cmd := []string{
		"gh", "repo", "deploy-key", "add", publicKeyPath,
		"-R", repo.Slug(),
		"-t", title,
	}
	if write {
		cmd = append(cmd, "-w")
	}
	if _, err := run.Run(cmd, "", true); err != nil {
		return "", err
	}

	var keys []deployKey
	if err := run.RunJSON(
		[]string{"gh", "repo", "deploy-key", "list", "-R", repo.Slug(), "--json", "id,title"},
		&keys,
	); err != nil {
		return "", err
	}

	for _, k := range keys {
		if k.Title == title {
			return strconv.FormatInt(k.ID, 10), nil
		}
	}
	return "", run.Errorf(
		"added deploy key '%s' to %s but couldn't find it back by title "+
			"— check `gh repo deploy-key list` by hand", title, repo.Slug(),
	)
}

// DeleteDeployKey removes a deploy key by id.
func DeleteDeployKey(repo RepoRef, keyID string) error {
	_, err := run.Run([]string{"gh", "repo", "deploy-key", "delete", keyID, "-R", repo.Slug()}, "", true)
	return err
}

// RepoDefaultBranch returns repo's default branch, or "main" if GitHub
// doesn't report one.
func RepoDefaultBranch(repo RepoRef) (string, error) {
	var data struct {
		DefaultBranchRef *struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}
	if err := run.RunJSON(
		[]string{"gh", "repo", "view", repo.Slug(), "--json", "defaultBranchRef"}, &data,
	); err != nil {
		return "", err
	}
	if data.DefaultBranchRef == nil || data.DefaultBranchRef.Name == "" {
		return "main", nil
	}
	return data.DefaultBranchRef.Name, nil
}
