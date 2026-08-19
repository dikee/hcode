package commands

import (
	"strings"
	"testing"

	"github.com/dikee/hcode/internal/config"
	"github.com/dikee/hcode/internal/state"
)

func testInstance() state.Instance {
	return state.Instance{
		Name: "box1",
		Repos: []state.Repo{
			{Name: "repoA", Worktrees: []string{"cc2", "cc3"}},
			{Name: "repoB", Worktrees: []string{"cc2"}},
		},
	}
}

func TestResolveCwd_WorktreeUniqueMatch(t *testing.T) {
	inst := testInstance()
	got, err := resolveCwd(inst, "", "cc3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := config.RemoteCodeDir + "/repoA-cc3"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCwd_WorktreeAmbiguousWithoutRepo(t *testing.T) {
	inst := testInstance()
	_, err := resolveCwd(inst, "", "cc2")
	if err == nil {
		t.Fatal("expected an error for a worktree present on more than one repo")
	}
	if !strings.Contains(err.Error(), "disambiguate with --repo") {
		t.Errorf("error %q doesn't mention disambiguation", err.Error())
	}
}

func TestResolveCwd_WorktreeDisambiguatedByRepo(t *testing.T) {
	inst := testInstance()
	got, err := resolveCwd(inst, "repoB", "cc2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := config.RemoteCodeDir + "/repoB-cc2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCwd_WorktreeNotFound(t *testing.T) {
	inst := testInstance()
	_, err := resolveCwd(inst, "", "cc9")
	if err == nil {
		t.Fatal("expected an error for a nonexistent worktree")
	}
}

func TestResolveCwd_RepoOnly(t *testing.T) {
	inst := testInstance()
	got, err := resolveCwd(inst, "repoB", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := config.RemoteCodeDir + "/repoB"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCwd_UnknownRepo(t *testing.T) {
	inst := testInstance()
	_, err := resolveCwd(inst, "repoZ", "")
	if err == nil {
		t.Fatal("expected an error for a repo not on the instance")
	}
}

func TestResolveCwd_NoArgsSingleRepo(t *testing.T) {
	inst := testInstance()
	inst.Repos = inst.Repos[:1]
	got, err := resolveCwd(inst, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := config.RemoteCodeDir + "/repoA"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCwd_NoArgsMultipleRepos(t *testing.T) {
	inst := testInstance()
	got, err := resolveCwd(inst, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string (ambiguous, no default cd)", got)
	}
}
