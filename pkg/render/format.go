package render

import (
	"fmt"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func Bold(s string) string   { return colorBold + s + colorReset }
func Green(s string) string  { return colorGreen + s + colorReset }
func Red(s string) string    { return colorRed + s + colorReset }
func Yellow(s string) string { return colorYellow + s + colorReset }
func Blue(s string) string   { return colorBlue + s + colorReset }
func Cyan(s string) string   { return colorCyan + s + colorReset }
func Gray(s string) string   { return colorGray + s + colorReset }

func StatusColor(status string) string {
	switch status {
	case "active", "completed", "approved":
		return Green(status)
	case "draft", "paused", "pending", "running", "waiting_approval":
		return Yellow(status)
	case "failed", "rejected", "expired", "cancelled":
		return Red(status)
	case "archived":
		return Gray(status)
	default:
		return status
	}
}

func Table(headers []string, rows [][]string) string {
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && visibleLen(cell) > colWidths[i] {
				colWidths[i] = visibleLen(cell)
			}
		}
	}

	var sb strings.Builder

	// Header
	sb.WriteString("  ")
	for i, h := range headers {
		sb.WriteString(Bold(padRight(h, colWidths[i])))
		if i < len(headers)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("  ")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w))
		if i < len(colWidths)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		sb.WriteString("  ")
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			sb.WriteString(padRight(cell, colWidths[i]))
			if i < len(headers)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// TimeAgo returns a human-readable relative time string.
func TimeAgo(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return ts
		}
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// visibleLen returns the length of a string excluding ANSI escape codes.
func visibleLen(s string) int {
	inEsc := false
	n := 0
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

func padRight(s string, width int) string {
	vis := visibleLen(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}

// Success prints a green success message.
func Success(msg string) { fmt.Println(Green("✓ " + msg)) }

// Warn prints a yellow warning message.
func Warn(msg string) { fmt.Println(Yellow("⚠ " + msg)) }

// FormatTime returns a short date string from an RFC3339 timestamp.
func FormatTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("1/2/2006")
}

// PrintHeader prints a table header row.
func PrintHeader(cols ...string) {
	for i, c := range cols {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-16s", Gray(c))
	}
	fmt.Println()
}

// PrintRow prints a table data row.
func PrintRow(cols ...string) {
	for i, c := range cols {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-16s", c)
	}
	fmt.Println()
}
