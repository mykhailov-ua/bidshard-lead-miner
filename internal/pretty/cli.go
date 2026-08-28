package pretty

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// ColorEnabled reports whether ANSI styling should be applied for w.
// Respects NO_COLOR and FORCE_COLOR; falls back to terminal detection.
func ColorEnabled(w io.Writer) bool {
	if v := strings.TrimSpace(os.Getenv("NO_COLOR")); v != "" {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("FORCE_COLOR")); v != "" && v != "0" {
		return true
	}
	if term := strings.TrimSpace(os.Getenv("TERM")); term == "" || term == "dumb" {
		return false
	}
	return IsTerminal(w)
}

// Section prints a titled block header.
func Section(w io.Writer, color bool, title string) {
	if color {
		_, _ = fmt.Fprintf(w, "\n%s%s%s\n", Bold, title, Reset)
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s\n", title)
}

// StatusOK prints a green success line (ok  message).
func StatusOK(w io.Writer, color bool, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if color && ColorEnabled(w) {
		_, _ = fmt.Fprintf(w, "%sok%s  %s\n", Green+Bold, Reset, msg)
		return
	}
	_, _ = fmt.Fprintf(w, "ok  %s\n", msg)
}

// StatusWarn prints a yellow warning line.
func StatusWarn(w io.Writer, color bool, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if color && ColorEnabled(w) {
		_, _ = fmt.Fprintf(w, "%swarn%s %s\n", Yellow+Bold, Reset, msg)
		return
	}
	_, _ = fmt.Fprintf(w, "warn %s\n", msg)
}

// StatusErr prints a red error line (for check output, not process exit).
func StatusErr(w io.Writer, color bool, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if color && ColorEnabled(w) {
		_, _ = fmt.Fprintf(w, "%serror%s %s\n", Red+Bold, Reset, msg)
		return
	}
	_, _ = fmt.Fprintf(w, "error %s\n", msg)
}

// StatusNote prints a dim informational line.
func StatusNote(w io.Writer, color bool, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if color && ColorEnabled(w) {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Dim, msg, Reset)
		return
	}
	_, _ = fmt.Fprintln(w, msg)
}

// StatusInfo prints a cyan info line.
func StatusInfo(w io.Writer, color bool, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if color && ColorEnabled(w) {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Cyan, msg, Reset)
		return
	}
	_, _ = fmt.Fprintln(w, msg)
}

// FatalErr prints a red error to stderr (process exit messaging).
func FatalErr(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if ColorEnabled(os.Stderr) {
		_, _ = fmt.Fprintf(os.Stderr, "%serror:%s %s\n", Red+Bold, Reset, msg)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", msg)
}

// PrintTable writes aligned columns with an optional colored header row.
func PrintTable(w io.Writer, color bool, header []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if len(header) > 0 {
		if color && ColorEnabled(w) {
			styled := make([]string, len(header))
			for i, h := range header {
				styled[i] = Bold + strings.ToUpper(h) + Reset
			}
			_, _ = fmt.Fprintln(tw, strings.Join(styled, "\t"))
		} else {
			_, _ = fmt.Fprintln(tw, strings.Join(header, "\t"))
		}
	}
	for _, row := range rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
}

// PrintLabeled writes key=value lines with colored keys.
func PrintLabeled(w io.Writer, color bool, pairs [][2]string) {
	for _, p := range pairs {
		KV(w, color, p[0], p[1])
	}
}
