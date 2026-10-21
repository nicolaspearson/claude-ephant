package redact

import (
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
		{name: "assignment keeps the key name", redacted: "export GITLAB_TOKEN=[redacted:assignment]", text: "export GITLAB_TOKEN=abc123def"},
		{name: "assignment quoted value", redacted: `password: [redacted:assignment]`, text: `password: "hunter2 two"`},
		{name: "aws access key", redacted: "id [redacted:aws]", text: "id AKIAABCDEFGHIJKLMNOP"},
		{name: "github token", redacted: "[redacted:github]", text: "ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"},
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
