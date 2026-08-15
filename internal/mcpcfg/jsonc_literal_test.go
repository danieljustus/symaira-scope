package mcpcfg

import (
	"strings"
	"testing"

	"github.com/tailscale/hujson"
)

// findMemberNames parses doc as JSONC and returns the member-name Values of
// the object at the given key.
func findMemberNames(t *testing.T, doc, key string) []hujson.Value {
	t.Helper()
	ast, err := hujson.Parse([]byte(doc))
	if err != nil {
		t.Fatalf("parse jsonc %q: %v", doc, err)
	}
	v := ast.Find("/" + key)
	if v == nil {
		t.Fatalf("key %q not found in %q", key, doc)
	}
	obj, ok := v.Value.(*hujson.Object)
	if !ok {
		t.Fatalf("value at %q is not an object in %q", key, doc)
	}
	names := make([]hujson.Value, 0, len(obj.Members))
	for _, m := range obj.Members {
		names = append(names, m.Name)
	}
	return names
}

// anyLiteralEquals reports whether any member name matches quoted.
func anyLiteralEquals(names []hujson.Value, quoted string) bool {
	for _, n := range names {
		if memberLiteralEquals(n, quoted) {
			return true
		}
	}
	return false
}

func TestMemberLiteralEquals(t *testing.T) {
	tests := []struct {
		name   string
		doc    string
		quoted string
		want   bool
	}{
		{
			name:   "plain match",
			doc:    `{"servers":{"alpha":{}}}`,
			quoted: "alpha",
			want:   true,
		},
		{
			name:   "whitespace around member name",
			doc:    "{\n  \"servers\": {\n    \"alpha\": {}\n  }\n}",
			quoted: "alpha",
			want:   true,
		},
		{
			name:   "comment and newline before member name",
			doc:    "{\n  \"servers\": {\n    // which server?\n    /* block */ \"alpha\": {}\n  }\n}",
			quoted: "alpha",
			want:   true,
		},
		{
			name:   "non-matching quoted name",
			doc:    `{"servers":{"alpha":{}}}`,
			quoted: "beta",
			want:   false,
		},
		{
			name:   "empty quoted name",
			doc:    `{"servers":{"":{}}}`,
			quoted: "",
			want:   true,
		},
		{
			name:   "trailing comma after member",
			doc:    "{\"servers\":{\"alpha\":{},\n}}",
			quoted: "alpha",
			want:   true,
		},
		{
			name:   "trailing comma before member",
			doc:    "{\n  \"servers\": {\n    \"alpha\": {},\n    \"beta\": {},\n  }\n}",
			quoted: "beta",
			want:   true,
		},
		{
			name:   "nested object value",
			doc:    `{"servers":{"outer":{"inner":{"deep":1}}}}`,
			quoted: "outer",
			want:   true,
		},
		{
			name:   "name with escaped quote",
			doc:    `{"servers":{"a\"b":{}}}`,
			quoted: `a\"b`,
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := findMemberNames(t, tt.doc, "servers")
			if got := anyLiteralEquals(names, tt.quoted); got != tt.want {
				t.Errorf("memberLiteralEquals(%q): want %v, got %v (doc %s)", tt.quoted, tt.want, got, tt.doc)
			}
		})
	}

	// A non-literal member name (not a hujson.Literal) never matches.
	t.Run("non-literal name", func(t *testing.T) {
		v := hujson.Value{Value: &hujson.Object{Members: []hujson.ObjectMember{}}}
		if memberLiteralEquals(v, "alpha") {
			t.Error("memberLiteralEquals on non-literal name: want false, got true")
		}
	})
}

func TestIsJSONCMemberNotFound(t *testing.T) {
	// A real not-found error from jsoncRemoveMember must satisfy the sentinel.
	t.Run("member missing", func(t *testing.T) {
		_, err := jsoncRemoveMember([]byte(`{"servers":{"alpha":{}}}`), "servers", "missing")
		if err == nil {
			t.Fatal("expected error for missing member")
		}
		if !IsJSONCMemberNotFound(err) {
			t.Errorf("IsJSONCMemberNotFound(%v): want true, got false", err)
		}
	})

	// A parse error must not satisfy the sentinel.
	t.Run("parse error", func(t *testing.T) {
		_, err := jsoncRemoveMember([]byte(`{`), "servers", "alpha")
		if err == nil {
			t.Fatal("expected parse error")
		}
		if IsJSONCMemberNotFound(err) {
			t.Errorf("parse error %v must not satisfy IsJSONCMemberNotFound", err)
		}
	})

	// A missing top-level key is a different error kind.
	t.Run("key not found", func(t *testing.T) {
		_, err := jsoncRemoveMember([]byte(`{"other":{}}`), "servers", "alpha")
		if err == nil {
			t.Fatal("expected key-not-found error")
		}
		if IsJSONCMemberNotFound(err) {
			t.Errorf("key-not-found error %v must not satisfy IsJSONCMemberNotFound", err)
		}
	})

	// A non-object value at the key is a different error kind.
	t.Run("not an object", func(t *testing.T) {
		_, err := jsoncRemoveMember([]byte(`{"servers":[]}`), "servers", "alpha")
		if err == nil {
			t.Fatal("expected not-an-object error")
		}
		if IsJSONCMemberNotFound(err) {
			t.Errorf("not-an-object error %v must not satisfy IsJSONCMemberNotFound", err)
		}
	})

	// nil is never the sentinel.
	if IsJSONCMemberNotFound(nil) {
		t.Error("IsJSONCMemberNotFound(nil): want false, got true")
	}

	// A successful removal returns no error at all.
	t.Run("successful remove", func(t *testing.T) {
		out, err := jsoncRemoveMember([]byte(`{"servers":{"alpha":{}}}`), "servers", "alpha")
		if err != nil {
			t.Fatalf("remove existing member: %v", err)
		}
		if strings.Contains(string(out), `"alpha"`) {
			t.Errorf("member alpha should be removed, got:\n%s", out)
		}
		if IsJSONCMemberNotFound(err) {
			t.Error("IsJSONCMemberNotFound(nil error): want false, got true")
		}
	})
}

func TestJSONCAddRemoveMemberEdgeCases(t *testing.T) {
	// Document with trailing commas, comments, and empty/nested values:
	// add, replace, and remove must all behave correctly.
	doc := `{
  // servers block
  "servers": {
    "empty": {},
    "nested": {"inner": {"deep": true}},
    "keep": {"command": "a"},
  },
}`
	out, err := jsoncAddMember([]byte(doc), "servers", "added", []byte(`{"command":"x"}`))
	if err != nil {
		t.Fatalf("jsoncAddMember on jsonc doc: %v", err)
	}
	names := findMemberNames(t, string(out), "servers")
	if !anyLiteralEquals(names, "added") {
		t.Errorf("added member missing after add:\n%s", out)
	}
	if !anyLiteralEquals(names, "empty") || !anyLiteralEquals(names, "nested") || !anyLiteralEquals(names, "keep") {
		t.Errorf("existing members lost after add:\n%s", out)
	}

	// Replacing an existing member whose name has surrounding comments.
	out, err = jsoncAddMember(out, "servers", "keep", []byte(`{"command":"b"}`))
	if err != nil {
		t.Fatalf("jsoncAddMember replace: %v", err)
	}
	if !strings.Contains(string(out), `"command":"b"`) {
		t.Errorf("replaced value not found:\n%s", out)
	}

	// Remove the added member again.
	out, err = jsoncRemoveMember(out, "servers", "added")
	if err != nil {
		t.Fatalf("jsoncRemoveMember: %v", err)
	}
	if strings.Contains(string(out), `"added"`) {
		t.Errorf("added member should be removed:\n%s", out)
	}

	// Empty member name round-trip.
	out, err = jsoncAddMember([]byte(`{"servers":{}}`), "servers", "", []byte(`{"command":"z"}`))
	if err != nil {
		t.Fatalf("jsoncAddMember empty name: %v", err)
	}
	if !anyLiteralEquals(findMemberNames(t, string(out), "servers"), "") {
		t.Errorf("empty-named member missing:\n%s", out)
	}
	out, err = jsoncRemoveMember(out, "servers", "")
	if err != nil {
		t.Fatalf("jsoncRemoveMember empty name: %v", err)
	}
	if strings.Contains(string(out), "z") {
		t.Errorf("empty-named member should be removed:\n%s", out)
	}
}
