// Package forwards parses -L/--forward specs — shared by `create` and
// `ssh`, which both open an interactive session and can both usefully
// tunnel ports.
package forwards

import (
	"strings"

	"github.com/dikee/hetzner-code/internal/run"
)

// NormalizeForward accepts ssh's own `-L` syntax, plus shorthand:
//
//   - "8000"               -> "8000:localhost:8000"
//   - "8000:9000"           -> "8000:localhost:9000"  (different remote port)
//   - "8000:localhost:9000" -> unchanged (full form, any remote host)
func NormalizeForward(spec string) (string, error) {
	parts := strings.Split(spec, ":")
	switch len(parts) {
	case 1:
		return parts[0] + ":localhost:" + parts[0], nil
	case 2:
		return parts[0] + ":localhost:" + parts[1], nil
	case 3:
		return spec, nil
	default:
		return "", run.Errorf("--forward %q doesn't look like PORT, PORT:PORT, or PORT:HOST:PORT", spec)
	}
}
