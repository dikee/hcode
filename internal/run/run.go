// Package run is a thin subprocess wrapper. Every external call (hcloud,
// gh, ssh, scp) goes through here so failures come back as one clear
// error type instead of a bare exec error with no context about which
// step broke.
package run

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// HcodeError is the one error type every hcode failure surfaces as.
type HcodeError struct {
	msg string
}

func (e *HcodeError) Error() string { return e.msg }

// Errorf builds an *HcodeError from a format string.
func Errorf(format string, args ...any) *HcodeError {
	return &HcodeError{msg: fmt.Sprintf(format, args...)}
}

// Result mirrors the pieces of Python's subprocess.CompletedProcess this
// tool actually uses.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run runs cmd, capturing stdout/stderr. Returns an *HcodeError with the
// command and stderr on non-zero exit unless check is false.
func Run(cmd []string, input string, check bool) (Result, error) {
	c := exec.Command(cmd[0], cmd[1:]...)
	if input != "" {
		c.Stdin = strings.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	exitCode := 0
	if err := c.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return Result{}, err
		}
		exitCode = exitErr.ExitCode()
	}

	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}
	if check && exitCode != 0 {
		return result, Errorf(
			"command failed (%d): %s\n%s",
			exitCode, strings.Join(cmd, " "), strings.TrimSpace(result.Stderr),
		)
	}
	return result, nil
}

// RunJSON runs cmd and unmarshals its stdout into v. hcloud/gh both
// support -o json / --json on the calls this tool needs.
func RunJSON(cmd []string, v any) error {
	result, err := Run(cmd, "", true)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(result.Stdout), v); err != nil {
		out := result.Stdout
		if len(out) > 500 {
			out = out[:500]
		}
		return Errorf("expected JSON from: %s\ngot: %s", strings.Join(cmd, " "), out)
	}
	return nil
}
