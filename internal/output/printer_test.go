package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON(t *testing.T) {
	data := TableData{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}},
		Raw:     map[string]string{"foo": "bar"},
	}

	var buf bytes.Buffer

	err := Print(&buf, "json", "none", data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `"foo": "bar"`)
}

func TestPrintCSV(t *testing.T) {
	data := TableData{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}, {"3", "4"}},
	}

	var buf bytes.Buffer

	err := Print(&buf, "csv", "none", data)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "A,B\n")
	assert.Contains(t, out, "1,2\n")
	assert.Contains(t, out, "3,4\n")
}

func TestPrintTableThemes(t *testing.T) {
	data := TableData{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}, {"3", "4"}},
	}

	themes := []string{"catppuccin-frappe", "dracula", "nord", "none", "unknown"}

	for _, theme := range themes {
		t.Run(theme, func(t *testing.T) {
			var buf bytes.Buffer

			err := Print(&buf, "table", theme, data)
			require.NoError(t, err)

			out := buf.String()
			// Basic checks for table structure
			assert.Contains(t, out, "A")
			assert.Contains(t, out, "B")
			assert.Contains(t, out, "1")
			assert.Contains(t, out, "4")

			// Check for ANSI color escape sequences if a theme is applied
			if theme == "none" {
				assert.NotContains(t, out, "\x1b[")
			}
			// Themes apply colors, so lipgloss should generate escape sequences
			// This might be false in a CI environment where lipgloss strips colors,
			// but forcing colors or checking for structure is generally safe.
			// We just verify it doesn't crash or error out.
		})
	}
}

func TestPrintUnknownFormat(t *testing.T) {
	data := TableData{}

	var buf bytes.Buffer

	err := Print(&buf, "unknown", "none", data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}

type errorWriter struct{}

func (*errorWriter) Write(_ []byte) (int, error) {
	return 0, assert.AnError
}

func TestPrintCSVError(t *testing.T) {
	data := TableData{
		Headers: []string{"A", "B"},
	}

	err := Print(&errorWriter{}, "csv", "none", data)
	require.Error(t, err)
}
