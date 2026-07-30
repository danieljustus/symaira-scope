// Package output provides a Reporter interface and implementations for
// structured CLI output (JSON and NDJSON).
package output

import (
	"encoding/json"
	"io"
	"os"
)

// Reporter formats and writes structured data to stdout.
type Reporter interface {
	// Print encodes v and writes it to the output stream.
	Print(v any) error
}

// New creates a Reporter for the given format.
// Supported formats: "json", "ndjson".
// Defaults to "json" when format is empty or unknown.
func New(format string) Reporter {
	switch format {
	case "json":
		return NewJSON()
	case "ndjson":
		return NewNDJSON()
	default:
		return NewJSON()
	}
}

// writeEncoded is a helper that encodes v with a preconfigured *json.Encoder
// writing to w.
func writeEncoded(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// stdout is os.Stdout; exposed as a variable so tests can override it.
var stdout io.Writer = os.Stdout
