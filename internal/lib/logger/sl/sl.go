// Package sl holds log/slog helpers.
package sl

import "log/slog"

// Err returns an slog attribute for an error.
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "err",
		Value: slog.StringValue(err.Error()),
	}
}
