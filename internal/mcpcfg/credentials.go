package mcpcfg

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// credentialPatterns are known provider-specific secret shapes. Each value is
// matched against a single env value after exclusions (vault:// references,
// empty values, obvious placeholders) are applied.
var credentialPatterns = []struct {
	label string
	re    *regexp.Regexp
}{
	{"OpenAI API key", regexp.MustCompile(`^sk-[A-Za-z0-9_-]{20,}$`)},
	{"GitHub personal access token", regexp.MustCompile(`^ghp_[A-Za-z0-9]{36}$`)},
	{"GitHub fine-grained PAT", regexp.MustCompile(`^github_pat_[A-Za-z0-9_]{20,}$`)},
	{"GitLab personal access token", regexp.MustCompile(`^glpat-[A-Za-z0-9_-]{20,}$`)},
	{"Slack token", regexp.MustCompile(`^xox[baprs]-[A-Za-z0-9-]+$`)},
	{"AWS access key ID", regexp.MustCompile(`^(AKIA|ASIA)[A-Z0-9]{16}$`)},
	{"Stripe live key", regexp.MustCompile(`^sk_live_[A-Za-z0-9]+$`)},
	{"JSON Web Token", regexp.MustCompile(`^eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*$`)},
}

// placeholderSubstrings are hints that a value is not a real secret and
// should not be flagged. Matching is case-insensitive.
var placeholderSubstrings = []string{
	"your",
	"replace",
	"placeholder",
	"example",
	"todo",
	"dummy",
	"fake",
	"test",
	"changeme",
	"none",
	"null",
}

// CheckCredentials returns one warning per env value that looks like an
// exposed credential. vault:// references, empty values, and obvious
// placeholders are ignored. The warnings are sorted for stable output.
func CheckCredentials(server model.MCPServer) []string {
	var warnings []string
	for key, value := range server.Env {
		if reason := credentialLeakReason(value); reason != "" {
			warnings = append(warnings, fmt.Sprintf("env.%s appears to be an exposed %s", key, reason))
		}
	}
	sort.Strings(warnings)
	return warnings
}

func credentialLeakReason(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "vault://") {
		return ""
	}
	lv := strings.ToLower(value)
	for _, p := range placeholderSubstrings {
		if strings.Contains(lv, p) {
			return ""
		}
	}
	// Common non-secret literals.
	switch lv {
	case "true", "false", "yes", "no", "1", "0":
		return ""
	}
	// URLs, paths, and hostnames are normally not secrets.
	if strings.HasPrefix(lv, "http://") || strings.HasPrefix(lv, "https://") ||
		strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") {
		return ""
	}
	for _, p := range credentialPatterns {
		if p.re.MatchString(value) {
			return p.label
		}
	}
	if isHighEntropySecret(value) {
		return "high-entropy secret"
	}
	return ""
}

// isHighEntropySecret flags long, random-looking strings that do not match a
// known provider pattern. The heuristic is intentionally conservative to keep
// false positives low.
func isHighEntropySecret(value string) bool {
	if len(value) < 32 {
		return false
	}
	// Must look like a token (no spaces, mostly ASCII alphanumerics and a few
	// common separators) and have a high ratio of unique characters.
	var unique int
	seen := make(map[rune]bool)
	for _, r := range value {
		if unicode.IsSpace(r) {
			return false
		}
		if !isTokenRune(r) {
			return false
		}
		if !seen[r] {
			seen[r] = true
			unique++
		}
	}
	return float64(unique)/float64(len(value)) > 0.65
}

func isTokenRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '=' || r == '/'
}
