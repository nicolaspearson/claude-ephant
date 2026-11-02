package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		name     string
		redacted string
		text     string
	}{
		{name: "anthropic key", redacted: "key [redacted:anthropic] end", text: "key sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123 end"},
		{name: "assignment escaped quote inside a quoted value", redacted: `{"password":[redacted:assignment]}`, text: `{"password":"a\"b"}`},
		{name: "assignment escaped quotes", redacted: `API_KEY=[redacted:assignment] next`, text: `API_KEY=\"s3cr3tvalue123\" next`},
		{name: "assignment escaped single quotes", redacted: `PASSWORD=[redacted:assignment] next`, text: `PASSWORD=\'s3cr3tvalue\' next`},
		{name: "assignment go short declaration", redacted: `password := [redacted:assignment]`, text: `password := "hunter2"`},
		{name: "assignment hyphenated header", redacted: "X-Api-Key: [redacted:assignment]", text: "X-Api-Key: s3cr3t"},
		{name: "assignment keeps the key name", redacted: "export GITLAB_TOKEN=[redacted:assignment]", text: "export GITLAB_TOKEN=abc123def"},
		{name: "assignment quoted across lines", redacted: "PASSWORD=[redacted:assignment]\necho done", text: "PASSWORD=\"abc\ndef\"\necho done"},
		{name: "assignment quoted past the limit keeps its line", redacted: "PASSWORD=[redacted:assignment]\n" + strings.Repeat("x", 4100) + `"`, text: "PASSWORD=\"abc\n" + strings.Repeat("x", 4100) + `"`},
		{name: "assignment quoted value", redacted: `password: [redacted:assignment]`, text: `password: "hunter2 two"`},
		{name: "assignment unclosed quote ends at its line", redacted: "PASSWORD=[redacted:assignment]\nnext line", text: "PASSWORD=\"abc\nnext line"},
		{name: "assignment unquoted value with a backslash", redacted: "PASSWORD=[redacted:assignment] next", text: `PASSWORD=abc\$def next`},
		{name: "assignment unquoted value with a json escape", redacted: `"TOKEN=[redacted:assignment]"`, text: `"TOKEN=abc\` + `u0026def"`},
		{name: "assignment with an empty value untouched", redacted: `password: ""`, text: `password: ""`},
		{name: "aws access key", redacted: "id [redacted:aws]", text: "id AKIAABCDEFGHIJKLMNOP"},
		{name: "escaped json in a command", redacted: `curl -d "{\"password\":[redacted:assignment]}"`, text: `curl -d "{\"password\":\"hunter2\"}"`},
		{name: "github token", redacted: "[redacted:github]", text: "ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"},
		{name: "gitlab ci build token", redacted: "[redacted:gitlab]", text: "glcbt-abcdefghijklmnopqrst"},
		{name: "gitlab deploy token", redacted: "[redacted:gitlab]", text: "gldt-abcdefghijklmnopqrst"},
		{name: "gitlab feed token", redacted: "[redacted:gitlab]", text: "glft-abcdefghijklmnopqrst"},
		{name: "gitlab oauth secret", redacted: "[redacted:gitlab]", text: "gloas-abcdefghijklmnopqrst"},
		{name: "gitlab pipeline trigger token", redacted: "[redacted:gitlab]", text: "glptt-abcdefghijklmnopqrst"},
		{name: "gitlab runner token", redacted: "[redacted:gitlab]", text: "glrt-abcdefghijklmnopqrst"},
		{name: "gitlab token", redacted: "[redacted:gitlab]", text: "glpat-abcdefghijklmnopqrst"},
		{name: "json keys", redacted: `{"password": [redacted:assignment], "api_key":[redacted:assignment]}`, text: `{"password": "hunter2hunter2", "api_key":"abcdefghijkl"}`},
		{name: "jwt", redacted: "Bearer [redacted:jwt]", text: "Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NSJ9.abcdefghijklmnop"},
		{name: "known token keeps its label when quoted", redacted: `{"token": "[redacted:github]"}`, text: `{"token": "ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"}`},
		{name: "openai key", redacted: "[redacted:openai]", text: "sk-proj-abcdefghijklmnopqrstuvwxyz0123456789"},
		{name: "pem block", redacted: "a [redacted:pem] b", text: "a -----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY----- b"},
		{name: "pem key assigned to a variable", redacted: "PRIVATE_KEY=[redacted:pem]", text: "PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----"},
		{name: "prefix inside a word untouched", redacted: "task-abcdefghijklmnopqrstuvwxyz0123456789", text: "task-abcdefghijklmnopqrstuvwxyz0123456789"},
		{name: "prose without an assignment untouched", redacted: "the token expires; pass the secret through vault", text: "the token expires; pass the secret through vault"},
		{name: "redaction is stable", redacted: "API_KEY=[redacted:assignment]", text: "API_KEY=[redacted:assignment]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.redacted, Redact(test.text))
		})
	}
}
