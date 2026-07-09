package mcpcfg

import (
	"strings"
	"testing"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestCheckCredentialsDetectsOpenAIKey(t *testing.T) {
	// Assembled at runtime so no single source literal matches a real secret format.
	key := "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"OPENAI_API_KEY": key,
		},
	}
	got := CheckCredentials(server)
	if len(got) != 1 {
		t.Fatalf("want 1 warning, got %d: %v", len(got), got)
	}
	if got[0] != "env.OPENAI_API_KEY appears to be an exposed OpenAI API key" {
		t.Errorf("unexpected warning: %q", got[0])
	}
}

func TestCheckCredentialsIgnoresVaultReference(t *testing.T) {
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"OPENAI_API_KEY": "vault://secret/openai",
		},
	}
	got := CheckCredentials(server)
	if len(got) != 0 {
		t.Fatalf("want 0 warnings for vault reference, got %v", got)
	}
}

func TestCheckCredentialsIgnoresEmptyAndPlaceholder(t *testing.T) {
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"EMPTY":       "",
			"PLACEHOLDER": "YOUR_API_KEY_HERE",
			"TODO":        "replace-me-later",
			"BOOL":        "true",
		},
	}
	got := CheckCredentials(server)
	if len(got) != 0 {
		t.Fatalf("want 0 warnings for placeholders, got %v", got)
	}
}

func TestCheckCredentialsDetectsGitHubToken(t *testing.T) {
	// Assembled at runtime so no single source literal matches a real secret format.
	token := "ghp_" + "123456789012345678901234567890123456"
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"GITHUB_TOKEN": token,
		},
	}
	got := CheckCredentials(server)
	if len(got) != 1 {
		t.Fatalf("want 1 warning, got %d: %v", len(got), got)
	}
}

func TestCheckCredentialsDetectsJWT(t *testing.T) {
	// Assembled at runtime so no single source literal matches a real JWT format.
	jwt := strings.Join([]string{
		"eyJ" + "hbGciOiJIUzI1NiJ9",
		"eyJ" + "zdWIiOiIxMjMifQ",
		"SflKxw" + "RJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
	}, ".")
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"TOKEN": jwt,
		},
	}
	got := CheckCredentials(server)
	if len(got) != 1 {
		t.Fatalf("want 1 warning, got %d: %v", len(got), got)
	}
}

func TestCheckCredentialsDetectsHighEntropySecret(t *testing.T) {
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"SECRET": "aB3dEfGhJkLmNpQrStUvWxYz0123456789AbCdEfGh",
		},
	}
	got := CheckCredentials(server)
	if len(got) != 1 {
		t.Fatalf("want 1 warning, got %d: %v", len(got), got)
	}
	if got[0] != "env.SECRET appears to be an exposed high-entropy secret" {
		t.Errorf("unexpected warning: %q", got[0])
	}
}

func TestCheckCredentialsIgnoresCommonNonSecrets(t *testing.T) {
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"HOST":    "http://localhost:3000",
			"PATH":    "/usr/local/bin",
			"VERSION": "1.2.3",
		},
	}
	got := CheckCredentials(server)
	if len(got) != 0 {
		t.Fatalf("want 0 warnings for non-secrets, got %v", got)
	}
}

func TestCheckCredentialsMultipleWarnings(t *testing.T) {
	server := model.MCPServer{
		Name: "test",
		Env: map[string]string{
			"GITHUB_TOKEN":   "ghp_" + "123456789012345678901234567890123456",
			"OPENAI_API_KEY": "sk-" + "abcdefghijklmnopqrstuvwxyz123456",
		},
	}
	got := CheckCredentials(server)
	if len(got) != 2 {
		t.Fatalf("want 2 warnings, got %d: %v", len(got), got)
	}
	if got[0] >= got[1] {
		t.Errorf("warnings should be sorted, got %v", got)
	}
}

func TestCheckCredentialsNoEnv(t *testing.T) {
	server := model.MCPServer{Name: "test"}
	got := CheckCredentials(server)
	if len(got) != 0 {
		t.Fatalf("want 0 warnings, got %v", got)
	}
}
