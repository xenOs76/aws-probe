package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpCmdRegistered(t *testing.T) {
	t.Parallel()

	root := newRootCmd()
	cmd, _, err := root.Find([]string{"mcp"})
	require.NoError(t, err)
	assert.Equal(t, "mcp", cmd.Name())
}
