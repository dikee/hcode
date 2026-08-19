// Package sshutil does SSH/SCP to a box, as root, using the login key
// that was injected into Hetzner at create time (never a repo's deploy
// key — those live only on the box, and are only ever used by git
// there).
//
// Every call here takes identityPath explicitly and passes
// `-i <path> -o IdentitiesOnly=yes` — the box only trusts the one
// public key registered with Hetzner for it, and that key almost never
// lives at one of ssh's default-tried filenames (id_rsa, id_ed25519,
// ...), so letting ssh guess would hang or silently try the wrong key.
//
// Hetzner reuses IPv4 addresses across different servers over time, so
// a freshly created box's IP may already sit in ~/.ssh/known_hosts from
// some unrelated past host. hcode boxes are throwaway by design, so
// every call here skips host-key pinning rather than risk a spurious
// "REMOTE HOST IDENTIFICATION HAS CHANGED" failure — and never touches
// the user's real known_hosts file to do it.
package sshutil

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/dikee/hetzner-code/internal/run"
)

func sshOpts(identityPath string) []string {
	return []string{
		"-i", identityPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
	}
}

// WaitForSSH polls until the box accepts an SSH connection, then until
// cloud-init's boot script has actually finished — the connection
// coming up doesn't mean provisioning has.
func WaitForSSH(ip, identityPath string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	opts := sshOpts(identityPath)
	deadline := time.Now().Add(timeout)
	lastError := "connection never succeeded"
	connected := false
	for time.Now().Before(deadline) {
		args := append([]string{}, opts...)
		args = append(args, "-o", "ConnectTimeout=5", "root@"+ip, "true")
		c := exec.Command("ssh", args...)
		var stderr bytes.Buffer
		c.Stderr = &stderr
		if err := c.Run(); err == nil {
			connected = true
			break
		}
		lastError = strings.TrimSpace(stderr.String())
		time.Sleep(3 * time.Second)
	}
	if !connected {
		return run.Errorf("timed out waiting for SSH on %s: %s", ip, lastError)
	}

	args := append([]string{}, opts...)
	args = append(args, "root@"+ip, "cloud-init", "status", "--wait")
	c := exec.Command("ssh", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return run.Errorf(
			"cloud-init failed on %s — SSH in and check `cloud-init status --long` "+
				"or /var/log/cloud-init-output.log\n%s", ip, strings.TrimSpace(stderr.String()),
		)
	}
	return nil
}

// RunRemote runs command over SSH, capturing stdout/stderr.
func RunRemote(ip, command, identityPath string) (run.Result, error) {
	args := append([]string{}, sshOpts(identityPath)...)
	args = append(args, "root@"+ip, command)
	c := exec.Command("ssh", args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	result := run.Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return run.Result{}, err
		}
	}
	if result.ExitCode != 0 {
		return result, run.Errorf(
			"remote command failed on %s: %s\n%s", ip, command, strings.TrimSpace(result.Stderr),
		)
	}
	return result, nil
}

// RunRemoteStreaming is like RunRemote, but inherits this process's
// stdout/stderr instead of capturing it — for a repo's own --post-clone
// script, which can run apt-get/make for a while and shouldn't look
// like create has hung.
func RunRemoteStreaming(ip, command, identityPath string) error {
	args := append([]string{}, sshOpts(identityPath)...)
	args = append(args, "root@"+ip, command)
	c := exec.Command("ssh", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	err := c.Run()
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return err
	}
	return run.Errorf("remote command failed on %s (exit %d): %s", ip, exitErr.ExitCode(), command)
}

// CopyTo scp's a single local file up to remotePath.
func CopyTo(ip, localPath, remotePath, identityPath string) error {
	if _, err := RunRemote(ip, "mkdir -p "+shellQuote(dirname(remotePath)), identityPath); err != nil {
		return err
	}
	args := append([]string{}, sshOpts(identityPath)...)
	args = append(args, localPath, "root@"+ip+":"+remotePath)
	c := exec.Command("scp", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return run.Errorf("scp to %s:%s failed\n%s", ip, remotePath, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// CopyDirTo is like CopyTo but recursive — for a whole directory (the
// ops folder), not a single file.
func CopyDirTo(ip, localDir, remoteDir, identityPath string) error {
	if _, err := RunRemote(ip, "mkdir -p "+shellQuote(dirname(remoteDir)), identityPath); err != nil {
		return err
	}
	args := []string{"-r"}
	args = append(args, sshOpts(identityPath)...)
	args = append(args, localDir, "root@"+ip+":"+remoteDir)
	c := exec.Command("scp", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return run.Errorf("scp -r to %s:%s failed\n%s", ip, remoteDir, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// CopyFrom pulls a file or directory back down. -r is harmless on a
// plain file, so one code path covers both without an extra round trip
// to ask the box which kind remotePath is.
func CopyFrom(ip, remotePath, localPath, identityPath string) error {
	if err := os.MkdirAll(path.Dir(localPath), 0o755); err != nil {
		return err
	}
	args := []string{"-r"}
	args = append(args, sshOpts(identityPath)...)
	args = append(args, "root@"+ip+":"+remotePath, localPath)
	c := exec.Command("scp", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return run.Errorf("scp from %s:%s failed\n%s", ip, remotePath, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// SyncDirFrom pulls remoteDir's *contents* into localDir, which is
// expected to already exist (it's --ops-dir's origin). The trailing
// `/.` on the remote side is the standard scp idiom for "copy contents
// into an existing directory" — without it, scp would nest remoteDir as
// a new subdirectory inside localDir instead of merging into it.
func SyncDirFrom(ip, remoteDir, localDir, identityPath string) error {
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return err
	}
	args := []string{"-r"}
	args = append(args, sshOpts(identityPath)...)
	args = append(args, "root@"+ip+":"+remoteDir+"/.", localDir)
	c := exec.Command("scp", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return run.Errorf("scp from %s:%s/. failed\n%s", ip, remoteDir, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Interactive opens an interactive session — inherits this process's
// tty. Returns the ssh process's exit code.
func Interactive(ip, identityPath string, cwd string, forwards []string) int {
	args := append([]string{}, sshOpts(identityPath)...)
	for _, spec := range forwards {
		args = append(args, "-L", spec)
	}
	args = append(args, "-t", "root@"+ip)
	if cwd != "" {
		args = append(args, "cd "+shellQuote(cwd)+" && exec bash -l")
	}
	c := exec.Command("ssh", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func dirname(remotePath string) string {
	if i := strings.LastIndex(remotePath, "/"); i >= 0 {
		return remotePath[:i]
	}
	return "."
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
