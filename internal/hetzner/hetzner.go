// Package hetzner runs everything that talks to the Hetzner API, via
// the `hcloud` CLI (assumed already authenticated —
// `hcloud context list`).
package hetzner

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dikee/hcode/internal/config"
	"github.com/dikee/hcode/internal/run"
)

type sshKeyInfo struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// PickLoginKey is the SSH key hcode injects for the human's own
// access — the first key in the account if the caller didn't name one
// with --login-key. This is separate from the per-repo deploy keys:
// it's for *you* logging in, not for git operations.
func PickLoginKey() (string, error) {
	var keys []sshKeyInfo
	if err := run.RunJSON([]string{"hcloud", "ssh-key", "list", "-o", "json"}, &keys); err != nil {
		return "", err
	}
	if len(keys) == 0 {
		return "", run.Errorf(
			"no SSH key registered with Hetzner yet. Run:\n" +
				"  hcloud ssh-key create --name laptop --public-key-from-file ~/.ssh/id_ed25519.pub\n" +
				"then retry, or pass --login-key explicitly.",
		)
	}
	return keys[0].Name, nil
}

// VerifyLoginKey fails fast, before spending money on a server, if
// loginKeyPath doesn't actually match what's registered as loginKey on
// Hetzner — otherwise the mismatch only surfaces as a mysterious SSH
// hang after the box is already up and billing.
func VerifyLoginKey(loginKey, loginKeyPath string) error {
	publicKeyPath := loginKeyPath + ".pub"
	localBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return run.Errorf(
			"--login-key-path %s has no matching %s next to it — pass the private key path, not the public one",
			loginKeyPath, publicKeyPath,
		)
	}
	localMaterial := keyMaterial(string(localBytes))

	var info sshKeyInfo
	if err := run.RunJSON([]string{"hcloud", "ssh-key", "describe", loginKey, "-o", "json"}, &info); err != nil {
		return err
	}
	remoteMaterial := keyMaterial(info.PublicKey)

	if localMaterial != remoteMaterial {
		return run.Errorf(
			"%s doesn't match the key registered as '%s' on Hetzner — SSH would hang after "+
				"the box comes up. Pass the right --login-key-path, or register this one: "+
				"hcloud ssh-key create --name <n> --public-key-from-file %s",
			loginKeyPath, loginKey, publicKeyPath,
		)
	}
	return nil
}

// keyMaterial keeps type + base64 blob only — drops the trailing
// comment, which differs between the local file and what Hetzner
// echoes back.
func keyMaterial(pubKeyText string) string {
	parts := strings.Fields(strings.TrimSpace(pubKeyText))
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, " ")
}

// ServerInfo is the subset of `hcloud server describe`'s JSON this tool
// actually reads.
type ServerInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	PublicNet struct {
		IPv4 struct {
			IP string `json:"ip"`
		} `json:"ipv4"`
	} `json:"public_net"`
}

// CreateServer creates the server, returning (serverID, ipv4).
func CreateServer(name, serverType, location, loginKey, userDataFile string, labels map[string]string) (string, string, error) {
	merged := map[string]string{}
	for k, v := range config.OwnerLabel {
		merged[k] = v
	}
	for k, v := range labels {
		merged[k] = v
	}

	cmd := []string{
		"hcloud", "server", "create",
		"--name", name,
		"--type", serverType,
		"--image", config.DefaultImage,
		"--location", location,
		"--ssh-key", loginKey,
		"--user-data-from-file", userDataFile,
	}
	for k, v := range merged {
		cmd = append(cmd, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	if _, err := run.Run(cmd, "", true); err != nil {
		return "", "", err
	}

	var info ServerInfo
	if err := run.RunJSON([]string{"hcloud", "server", "describe", name, "-o", "json"}, &info); err != nil {
		return "", "", err
	}
	return strconv.FormatInt(info.ID, 10), info.PublicNet.IPv4.IP, nil
}

// DeleteServer deletes a server by id.
func DeleteServer(serverID string) error {
	_, err := run.Run([]string{"hcloud", "server", "delete", serverID}, "", true)
	return err
}

// DescribeServer returns live server info, or nil if it no longer
// exists (deleted out of band — the case `status --reconcile` needs to
// catch).
func DescribeServer(serverID string) (*ServerInfo, error) {
	result, err := run.Run([]string{"hcloud", "server", "describe", serverID, "-o", "json"}, "", false)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, nil
	}
	var info ServerInfo
	if err := json.Unmarshal([]byte(result.Stdout), &info); err != nil {
		return nil, run.Errorf("expected JSON from: hcloud server describe %s -o json\ngot: %s", serverID, result.Stdout)
	}
	return &info, nil
}

// ListManagedServers returns every server on the account carrying
// hcode's owner label — the universe `status --reconcile` compares
// local state against.
func ListManagedServers() ([]ServerInfo, error) {
	labelSelector := ""
	first := true
	for k, v := range config.OwnerLabel {
		if !first {
			labelSelector += ","
		}
		labelSelector += fmt.Sprintf("%s=%s", k, v)
		first = false
	}
	var servers []ServerInfo
	if err := run.RunJSON([]string{"hcloud", "server", "list", "-l", labelSelector, "-o", "json"}, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}
