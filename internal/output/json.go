package output

import "encoding/json"

// JSON is a Reporter that writes indented JSON to stdout.
type JSON struct{}

// NewJSON creates a JSON reporter.
func NewJSON() *JSON { return &JSON{} }

// Print writes v as indented JSON (two-space indent, no HTML escaping).
func (j *JSON) Print(v any) error {
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
