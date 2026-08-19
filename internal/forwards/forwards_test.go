package forwards

import "testing"

func TestNormalizeForward(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "8000", want: "8000:localhost:8000"},
		{in: "8000:9000", want: "8000:localhost:9000"},
		{in: "8000:localhost:9000", want: "8000:localhost:9000"},
		{in: "1:2:3:4", wantErr: true},
		{in: "", want: ":localhost:"}, // matches the Python original's behavior on an empty spec
	}
	for _, c := range cases {
		got, err := NormalizeForward(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizeForward(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeForward(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeForward(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
