package output

// NDJSON is a Reporter that writes newline-delimited JSON (one object per line)
// to stdout, with no HTML escaping.
type NDJSON struct{}

// NewNDJSON creates an NDJSON reporter.
func NewNDJSON() *NDJSON { return &NDJSON{} }

// Print writes v as a single NDJSON line (no indent, no HTML escaping).
func (n *NDJSON) Print(v any) error {
	return writeEncoded(stdout, v)
}
