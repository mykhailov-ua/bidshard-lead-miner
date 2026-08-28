package pretty

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	Reset   = "\033[0m"
	Dim     = "\033[2m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
)

func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func KV(w io.Writer, color bool, key, value string) {
	indent := "    "
	if color {
		_, _ = fmt.Fprintf(w, "%s%s%s%s %s%s%s\n", indent, Cyan, key, Reset, Dim, value, Reset)
		return
	}
	_, _ = fmt.Fprintf(w, "%s%s %s\n", indent, key, value)
}

func Header(w io.Writer, color bool, title string) {
	if color {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Bold, title, Reset)
		return
	}
	_, _ = fmt.Fprintln(w, title)
}

func Divider(w io.Writer, color bool) {
	line := strings.Repeat("-", 40)
	if color {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Dim, line, Reset)
		return
	}
	_, _ = fmt.Fprintln(w, line)
}

func PriorityStyle(priority string) (label, style string) {
	switch strings.ToLower(priority) {
	case "high":
		return "HIGH", Red + Bold
	case "medium":
		return "MEDIUM", Yellow + Bold
	default:
		return strings.ToUpper(priority), Cyan
	}
}

func QuoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \t\n\"") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
