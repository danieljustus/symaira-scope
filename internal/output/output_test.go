package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewDefault(t *testing.T) {
	r := New("")
	if _, ok := r.(*JSON); !ok {
		t.Fatalf("expected *JSON, got %T", r)
	}
}

func TestNewJSON(t *testing.T) {
	r := New("json")
	if _, ok := r.(*JSON); !ok {
		t.Fatalf("expected *JSON, got %T", r)
	}
}

func TestNewNDJSON(t *testing.T) {
	r := New("ndjson")
	if _, ok := r.(*NDJSON); !ok {
		t.Fatalf("expected *NDJSON, got %T", r)
	}
}

func TestNewUnknownFormat(t *testing.T) {
	r := New("xml")
	if _, ok := r.(*JSON); !ok {
		t.Fatalf("expected *JSON fallback for unknown format, got %T", r)
	}
}

func TestJSONPrint(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = nil }()

	j := NewJSON()
	v := map[string]string{"hello": "world"}
	if err := j.Print(v); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, `"hello"`) {
		t.Fatalf("expected hello key in output, got: %s", out)
	}
	if !strings.Contains(out, `"world"`) {
		t.Fatalf("expected world value in output, got: %s", out)
	}
	// Should be indented JSON (leading spaces on non-first lines)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected multi-line indented JSON, got: %s", out)
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("expected indented second line, got: %q", lines[1])
	}
}

func TestJSONPrintNoHTMLEscaping(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = nil }()

	j := NewJSON()
	v := map[string]string{"msg": "<hello>"}
	if err := j.Print(v); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, `\u003c`) {
		t.Fatalf("HTML escaping should be disabled; got escaped chars in: %s", out)
	}
	if !strings.Contains(out, `"<hello>"`) {
		t.Fatalf("expected unescaped <hello>, got: %s", out)
	}
}

func TestNDJSONPrint(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = nil }()

	n := NewNDJSON()
	v := map[string]string{"event": "start"}
	if err := n.Print(v); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	// NDJSON: one line, no leading whitespace
	if strings.Contains(out, "\n ") {
		t.Fatalf("NDJSON should not be indented, got: %s", out)
	}
	if !strings.Contains(out, `"event"`) {
		t.Fatalf("expected event key in output, got: %s", out)
	}
	// Should end with newline
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("expected trailing newline, got: %s", out)
	}
	// Should be exactly one line (plus newline)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || lines[0] == "" {
		t.Fatalf("expected single NDJSON line, got %d lines", len(lines)-1)
	}
}

func TestNDJSONPrintNoHTMLEscaping(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = nil }()

	n := NewNDJSON()
	v := map[string]string{"msg": "<hello>"}
	if err := n.Print(v); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, `\u003c`) {
		t.Fatalf("HTML escaping should be disabled; got escaped chars in: %s", out)
	}
	if !strings.Contains(out, `"<hello>"`) {
		t.Fatalf("expected unescaped <hello>, got: %s", out)
	}
}

func TestMultipleNDJSONPrints(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = nil }()

	n := NewNDJSON()
	for _, v := range []int{1, 2, 3} {
		if err := n.Print(v); err != nil {
			t.Fatal(err)
		}
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 NDJSON lines, got %d", len(lines))
	}
}
