package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type cmdOutput struct {
	stdout string
	stderr string
}

func captureCmdOutput(t *testing.T, fn func() error) (cmdOutput, error) {
	t.Helper()

	oldStdout, oldStderr := os.Stdout, os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	fnErr := fn()

	stdoutW.Close()
	stderrW.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer

	_, err = io.Copy(&stdoutBuf, stdoutR)
	require.NoError(t, err)

	_, err = io.Copy(&stderrBuf, stderrR)
	require.NoError(t, err)

	return cmdOutput{stdout: stdoutBuf.String(), stderr: stderrBuf.String()}, fnErr
}
