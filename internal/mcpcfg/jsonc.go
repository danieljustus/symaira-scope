// Package mcpcfg discovers MCP servers configured across local AI clients.
package mcpcfg

import (
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
)

// jsoncMemberError is sentinel for "member not found".
type jsoncMemberError struct{ msg string }

func (e *jsoncMemberError) Error() string { return e.msg }

// IsJSONCMemberNotFound reports whether err indicates that the searched-for
// member was not found in the target JSONC object.
func IsJSONCMemberNotFound(err error) bool {
	_, ok := err.(*jsoncMemberError)
	return ok
}

// jsoncAddMember adds a member with the given name and JSON value to the
// object at the given JSON Pointer key in a JSONC document.  If the member
// already exists its value is replaced.  Comments, key order, and all
// untouched bytes are preserved.
func jsoncAddMember(data []byte, key, name string, entryJSON []byte) ([]byte, error) {
	ast, err := hujson.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse jsonc: %w", err)
	}

	v := ast.Find("/" + key)
	if v == nil {
		return nil, fmt.Errorf("key %q not found", key)
	}

	obj, ok := v.Value.(*hujson.Object)
	if !ok {
		return nil, fmt.Errorf("value at %q is not an object", key)
	}

	// Check if member already exists — replace value in-place.
	for i, m := range obj.Members {
		if memberLiteralEquals(m.Name, name) {
			entryAST, err := hujson.Parse(entryJSON)
			if err != nil {
				return nil, fmt.Errorf("parse entry json: %w", err)
			}
			entryAST.BeforeExtra = hujson.Extra(" ")
			entryAST.AfterExtra = hujson.Extra("")
			obj.Members[i].Value = entryAST
			ast.UpdateOffsets()
			return ast.Pack(), nil
		}
	}

	// Detect indentation from existing members.
	beforeExtra := detectMemberIndent(obj)

	// Parse the new entry value.
	entryAST, err := hujson.Parse(entryJSON)
	if err != nil {
		return nil, fmt.Errorf("parse entry json: %w", err)
	}
	entryAST.BeforeExtra = hujson.Extra(" ")
	entryAST.AfterExtra = hujson.Extra("")

	newMember := hujson.ObjectMember{
		Name: hujson.Value{
			BeforeExtra: hujson.Extra(beforeExtra),
			Value:       hujson.Literal(`"` + name + `"`),
		},
		Value: entryAST,
	}

	obj.Members = append(obj.Members, newMember)
	ast.UpdateOffsets()
	return ast.Pack(), nil
}

// jsoncRemoveMember removes the named member from the object at the given key
// in a JSONC document.  The member must exist; otherwise an error wrapping
// jsoncMemberError is returned.
func jsoncRemoveMember(data []byte, key, name string) ([]byte, error) {
	ast, err := hujson.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse jsonc: %w", err)
	}

	v := ast.Find("/" + key)
	if v == nil {
		return nil, fmt.Errorf("key %q not found", key)
	}

	obj, ok := v.Value.(*hujson.Object)
	if !ok {
		return nil, fmt.Errorf("value at %q is not an object", key)
	}

	found := false
	for i, m := range obj.Members {
		if memberLiteralEquals(m.Name, name) {
			obj.Members = append(obj.Members[:i], obj.Members[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return nil, &jsoncMemberError{msg: fmt.Sprintf("member %q not found", name)}
	}

	ast.UpdateOffsets()
	return ast.Pack(), nil
}

// jsoncAddTopLevelKey adds a new top-level key with the given JSON value to
// a JSON object document.  Used when the target key (e.g. "mcpServers") does
// not yet exist in the config file.
func jsoncAddTopLevelKey(data []byte, key string, valueJSON []byte) ([]byte, error) {
	ast, err := hujson.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse jsonc: %w", err)
	}

	rootObj, ok := ast.Value.(*hujson.Object)
	if !ok {
		return nil, fmt.Errorf("root value is not an object")
	}

	// Detect indentation from existing top-level members.
	beforeExtra := detectTopLevelIndent(rootObj, data)

	// Parse the value JSON.
	valueAST, err := hujson.Parse(valueJSON)
	if err != nil {
		return nil, fmt.Errorf("parse value json: %w", err)
	}
	valueAST.BeforeExtra = hujson.Extra(" ")
	valueAST.AfterExtra = hujson.Extra("")

	newMember := hujson.ObjectMember{
		Name: hujson.Value{
			BeforeExtra: hujson.Extra(beforeExtra),
			Value:       hujson.Literal(`"` + key + `"`),
		},
		Value: valueAST,
	}

	rootObj.Members = append(rootObj.Members, newMember)
	ast.UpdateOffsets()
	return ast.Pack(), nil
}

// detectMemberIndent extracts a suitable member-indentation string from an
// existing object's members (e.g. "\n    " for 4-space indented members).
// The returned string includes a leading newline and the indent whitespace
// (but no comma — hujson handles commas between members automatically).
func detectMemberIndent(obj *hujson.Object) string {
	if len(obj.Members) == 0 {
		return "\n  "
	}
	// Use the first member's Name.BeforeExtra.
	be := string(obj.Members[0].Name.BeforeExtra)
	if lastNL := strings.LastIndex(be, "\n"); lastNL >= 0 {
		return "\n" + be[lastNL+1:]
	}
	// For compact JSON without newline separation, use the BeforeExtra as-is.
	if be != "" {
		return be
	}
	return "\n  "
}

// detectTopLevelIndent detects indentation for adding a new top-level key.
func detectTopLevelIndent(obj *hujson.Object, _ []byte) string {
	return detectMemberIndent(obj)
}

// memberLiteralEquals reports whether the member name's literal equals
// quoted (e.g. "\"my-server\"").  It compares against the ValueTrimmed
// Literal directly so that whitespace/comments in BeforeExtra do not
// interfere.
func memberLiteralEquals(name hujson.Value, quoted string) bool {
	lit, ok := name.Value.(hujson.Literal)
	if !ok {
		return false
	}
	return string(lit) == `"`+quoted+`"`
}
