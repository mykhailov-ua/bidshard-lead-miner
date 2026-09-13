package telegrambot

import "log/slog"

// LogAPIError writes a single structured warn line (no raw HTTP dumps).
func LogAPIError(msg string, err error, extra ...any) {
	attrs := extra
	if ae, ok := AsAPIError(err); ok {
		attrs = append(attrs, ae.LogAttrs()...)
	} else if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	slog.Warn(msg, attrs...)
}
