package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// TableData holds the data for rendering in different formats.
type TableData struct {
	Headers []string
	Rows    [][]string
	Raw     any
}

// Print formats and writes the data to the provided writer based on the format and theme.
func Print(out io.Writer, format, theme string, data TableData) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")

		return encoder.Encode(data.Raw)
	case "csv":
		writer := csv.NewWriter(out)
		if err := writer.Write(data.Headers); err != nil {
			return err
		}

		if err := writer.WriteAll(data.Rows); err != nil {
			return err
		}

		writer.Flush()

		return writer.Error()
	case "table":
		return printTable(out, theme, data)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

func printTable(out io.Writer, theme string, data TableData) error {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers(data.Headers...).
		Rows(data.Rows...)

	var (
		headerColor, evenRowColor, oddRowColor, borderColor lipgloss.Color
		useTheme                                            = true
	)

	switch theme {
	case "dracula":
		headerColor = lipgloss.Color("#bd93f9")  // purple
		evenRowColor = lipgloss.Color("#282a36") // background
		oddRowColor = lipgloss.Color("#44475a")  // current line
		borderColor = lipgloss.Color("#6272a4")  // comment
	case "nord":
		headerColor = lipgloss.Color("#88c0d0")  // frost
		evenRowColor = lipgloss.Color("#2e3440") // polar night
		oddRowColor = lipgloss.Color("#3b4252")  // polar night lighter
		borderColor = lipgloss.Color("#4c566a")  // polar night darkest
	case "none", "":
		useTheme = false
	default:
		// fallback to catppuccin-frappe (default)
		headerColor = lipgloss.Color("#8caaee")
		evenRowColor = lipgloss.Color("#303446")
		oddRowColor = lipgloss.Color("#292c3c")
		borderColor = lipgloss.Color("#414559")
	}

	if useTheme {
		t.BorderStyle(lipgloss.NewStyle().Foreground(borderColor))
		t.StyleFunc(func(row, _ int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return lipgloss.NewStyle().
					Foreground(headerColor).
					Bold(true).
					Align(lipgloss.Center)
			case row%2 == 0:
				return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(evenRowColor).Padding(0, 1)
			default:
				return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(oddRowColor).Padding(0, 1)
			}
		})
	}

	fmt.Fprintln(out, t.Render())

	return nil
}
