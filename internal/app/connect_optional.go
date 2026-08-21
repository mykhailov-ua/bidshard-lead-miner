package app

import "log/slog"

func connectOptional[T any](name string, connect func() (*T, error), onOK func(*T)) *T {
	v, err := connect()
	if err != nil {
		slog.Warn(name+" failed", "error", err)
		return nil
	}
	if onOK != nil {
		onOK(v)
	}
	return v
}
