// Package keys handles per-repo SSH keypairs. One keypair per
// (instance, repo) pair, never reused across repos, so removing one
// repo's access can never affect another's — see the design note in
// README.md.
package keys

import (
	"os"
	"path/filepath"

	"github.com/dikee/hetzner-code/internal/run"
)

// Generate creates a passwordless ed25519 keypair at
// keyDir/id_ed25519{,.pub}.
func Generate(keyDir, comment string) (privatePath, publicPath string, err error) {
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		return "", "", err
	}
	privatePath = filepath.Join(keyDir, "id_ed25519")
	publicPath = filepath.Join(keyDir, "id_ed25519.pub")
	_, err = run.Run(
		[]string{"ssh-keygen", "-t", "ed25519", "-N", "", "-C", comment, "-f", privatePath},
		"", true,
	)
	if err != nil {
		return "", "", err
	}
	return privatePath, publicPath, nil
}

// ForgetPrivateKey deletes the local private key copy once it's safely
// on the box. The public key file is left in place — it's public, and
// handy for "which key did I register" debugging without another gh
// round trip.
func ForgetPrivateKey(privateKeyPath string) error {
	err := os.Remove(privateKeyPath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
