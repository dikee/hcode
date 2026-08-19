package commands

import "testing"

func TestParseGitStatus(t *testing.T) {
	cases := []struct {
		name            string
		output          string
		wantUncommitted int
		wantAhead       int
	}{
		{name: "clean", output: "## main...origin/main\n", wantUncommitted: 0, wantAhead: 0},
		{
			name:            "ahead and dirty",
			output:          "## main...origin/main [ahead 2]\n M foo.go\n?? bar.go\n",
			wantUncommitted: 2,
			wantAhead:       2,
		},
		{name: "empty output", output: "", wantUncommitted: 0, wantAhead: 0},
	}
	for _, c := range cases {
		gotUncommitted, gotAhead := parseGitStatus(c.output)
		if gotUncommitted != c.wantUncommitted || gotAhead != c.wantAhead {
			t.Errorf(
				"%s: parseGitStatus(%q) = (%d, %d), want (%d, %d)",
				c.name, c.output, gotUncommitted, gotAhead, c.wantUncommitted, c.wantAhead,
			)
		}
	}
}
