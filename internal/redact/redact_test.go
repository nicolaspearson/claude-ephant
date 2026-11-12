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
		{name: "assignment credential key", redacted: "credential=[redacted:assignment] next", text: "credential=abc123 next"},
		{name: "assignment escaped quote inside a quoted value", redacted: `{"password":[redacted:assignment]}`, text: `{"password":"a\"b"}`},
		{name: "assignment escaped quotes", redacted: `API_KEY=[redacted:assignment] next`, text: `API_KEY=\"s3cr3tvalue123\" next`},
		{name: "assignment escaped single quotes", redacted: `PASSWORD=[redacted:assignment] next`, text: `PASSWORD=\'s3cr3tvalue\' next`},
		{name: "assignment go short declaration", redacted: `password := [redacted:assignment]`, text: `password := "hunter2"`},
		{name: "assignment hyphenated header", redacted: "X-Api-Key: [redacted:assignment]", text: "X-Api-Key: s3cr3t"},
		{name: "assignment keeps the key name", redacted: "export GITLAB_TOKEN=[redacted:assignment]", text: "export GITLAB_TOKEN=abc123def"},
		{name: "assignment passphrase key", redacted: `{"passphrase": [redacted:assignment]}`, text: `{"passphrase": "correct horse"}`},
		{name: "assignment quoted across lines", redacted: "PASSWORD=[redacted:assignment]\necho done", text: "PASSWORD=\"abc\ndef\"\necho done"},
		{name: "assignment quoted past the limit keeps its line", redacted: "PASSWORD=[redacted:assignment]\n" + strings.Repeat("x", 4100) + `"`, text: "PASSWORD=\"abc\n" + strings.Repeat("x", 4100) + `"`},
		{name: "assignment quoted value", redacted: `password: [redacted:assignment]`, text: `password: "hunter2 two"`},
		{name: "assignment quoted value starting with a bracket", redacted: `password: [redacted:assignment] next`, text: `password: "[abc" next`},
		{name: "assignment unclosed quote ends at its line", redacted: "PASSWORD=[redacted:assignment]\nnext line", text: "PASSWORD=\"abc\nnext line"},
		{name: "assignment unquoted value with a backslash", redacted: "PASSWORD=[redacted:assignment] next", text: `PASSWORD=abc\$def next`},
		{name: "assignment unquoted value with a json escape", redacted: `"TOKEN=[redacted:assignment]"`, text: `"TOKEN=abc\` + `u0026def"`},
		{name: "assignment value after a redaction marker", redacted: "password=[redacted:assignment] next", text: "password=[redacted:assignment]leak next"},
		{name: "assignment value starting with a bracket", redacted: "password=[redacted:assignment] next", text: "password=[notredacted]secret next"},
		{name: "assignment value with a spaced bracket", redacted: "password=[redacted:assignment] next", text: "password=[a b] next"},
		{name: "assignment with an empty value untouched", redacted: `password: ""`, text: `password: ""`},
		{name: "aws access key", redacted: "id [redacted:aws]", text: "id AKIAABCDEFGHIJKLMNOP"},
		{name: "bearer token in a curl header", redacted: `curl -H "Authorization: Bearer [redacted:bearer]"`, text: `curl -H "Authorization: Bearer abc123opaque456"`},
		{name: "bearer token in a header", redacted: "Authorization: Bearer [redacted:bearer]", text: "Authorization: Bearer abc123opaque456"},
		{name: "bearer word before a word untouched", redacted: "uses Bearer authentication for the API", text: "uses Bearer authentication for the API"},
		{name: "bearer word in prose untouched", redacted: "the bearer of bad news", text: "the bearer of bad news"},
		{name: "count of tokens as a flag kept", redacted: "claude --max-tokens 4096", text: "claude --max-tokens 4096"},
		{name: "count of tokens in prose kept", redacted: "total tokens: 500", text: "total tokens: 500"},
		{name: "count of tokens in yaml kept", redacted: "input_tokens: 123", text: "input_tokens: 123"},
		{name: "count of tokens kept", redacted: `{"max_tokens": 4096}`, text: `{"max_tokens": 4096}`},
		{name: "count of tokens quoted kept", redacted: `{"max_tokens": "4096"}`, text: `{"max_tokens": "4096"}`},
		{name: "escaped json in a command", redacted: `curl -d "{\"password\":[redacted:assignment]}"`, text: `curl -d "{\"password\":\"hunter2\"}"`},
		{name: "flag after a json-escaped newline", redacted: `"mysql\n--password [redacted:flag]"`, text: `"mysql\n--password hunter2"`},
		{name: "flag after a valueless secret flag", redacted: "tool --token --password [redacted:flag]", text: "tool --token --password hunter2"},
		{name: "flag after an opening parenthesis runs to the value's end", redacted: "x(--password [redacted:flag]", text: "x(--password hunter2)"},
		{name: "flag escaped quoted value", redacted: `ssh host "mysql --password [redacted:flag]"`, text: `ssh host "mysql --password \"hunter2 x\""`},
		{name: "flag file path untouched", redacted: "tool --password-file /run/secrets/db", text: "tool --password-file /run/secrets/db"},
		{name: "flag followed by another flag untouched", redacted: "tool --token --verbose", text: "tool --token --verbose"},
		{name: "flag inside a word untouched", redacted: "build--password value", text: "build--password value"},
		{name: "flag naming a secret untouched", redacted: "aws secretsmanager get-secret-value --secret-id prod/db", text: "aws secretsmanager get-secret-value --secret-id prod/db"},
		{name: "flag negated untouched", redacted: "tool --no-password foo.txt", text: "tool --no-password foo.txt"},
		{name: "flag quoted value", redacted: "tool --api-key [redacted:flag] next", text: `tool --api-key "s3cr3t value" next`},
		{name: "flag unclosed quote ends at its line", redacted: "tool --password [redacted:flag]\nnext", text: "tool --password \"abc\nnext"},
		{name: "flag value", redacted: "tool --password [redacted:flag] next", text: "tool --password hunter2 next"},
		{name: "flag value after an api key flag", redacted: "tool --api-key [redacted:flag]", text: "tool --api-key s3cr3t"},
		{name: "github token", redacted: "[redacted:github]", text: "ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"},
		{name: "gitlab ci build token", redacted: "[redacted:gitlab]", text: "glcbt-abcdefghijklmnopqrst"},
		{name: "gitlab deploy token", redacted: "[redacted:gitlab]", text: "gldt-abcdefghijklmnopqrst"},
		{name: "gitlab feed token", redacted: "[redacted:gitlab]", text: "glft-abcdefghijklmnopqrst"},
		{name: "gitlab oauth secret", redacted: "[redacted:gitlab]", text: "gloas-abcdefghijklmnopqrst"},
		{name: "gitlab pipeline trigger token", redacted: "[redacted:gitlab]", text: "glptt-abcdefghijklmnopqrst"},
		{name: "gitlab runner token", redacted: "[redacted:gitlab]", text: "glrt-abcdefghijklmnopqrst"},
		{name: "gitlab token", redacted: "[redacted:gitlab]", text: "glpat-abcdefghijklmnopqrst"},
		{name: "google api key", redacted: "key [redacted:google] end", text: "key AIzaSyAbcdefghijklmnopqrstuvwxyz0123456 end"},
		{name: "json keys", redacted: `{"password": [redacted:assignment], "api_key":[redacted:assignment]}`, text: `{"password": "hunter2hunter2", "api_key":"abcdefghijkl"}`},
		{name: "jwt", redacted: "Bearer [redacted:jwt]", text: "Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NSJ9.abcdefghijklmnop"},
		{name: "known token keeps its label when escaped quoted", redacted: `{\"token\": \"[redacted:github]\"}`, text: `{\"token\": \"ghp_abcdefghijklmnopqrstuvwxyz0123456789AB\"}`},
		{name: "known token keeps its label when quoted", redacted: `{"token": "[redacted:github]"}`, text: `{"token": "ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"}`},
		{name: "npm token", redacted: "[redacted:npm]", text: "npm_abcdefghijklmnopqrstuvwxyz0123456789"},
		{name: "number under a singular token key redacted", redacted: "token: [redacted:assignment]", text: "token: 123456"},
		{name: "openai key", redacted: "[redacted:openai]", text: "sk-proj-abcdefghijklmnopqrstuvwxyz0123456789"},
		{name: "pem block", redacted: "a [redacted:pem] b", text: "a -----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY----- b"},
		{name: "pem block cut before its end", redacted: "a [redacted:pem]\n(truncated)", text: "a -----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASC\nAbCdEfGhIjKlMnOp+/==\n(truncated)"},
		{name: "pem block cut before its end keeps the next line", redacted: "[redacted:pem]\nDone with it", text: "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASC\nDone with it"},
		{name: "pem block cut in escaped json", redacted: `"[redacted:pem]"`, text: `"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASC"`},
		{name: "pem block cut with its lines joined", redacted: "[redacted:pem] then", text: "-----BEGIN PRIVATE KEY----- MIIEvQIBADANBgkqhkiG9w0BAQEFAASC then"},
		{name: "pem key assigned to a variable", redacted: "PRIVATE_KEY=[redacted:pem]", text: "PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----"},
		{name: "prefix inside a word untouched", redacted: "task-abcdefghijklmnopqrstuvwxyz0123456789", text: "task-abcdefghijklmnopqrstuvwxyz0123456789"},
		{name: "prose without an assignment untouched", redacted: "the token expires; pass the secret through vault", text: "the token expires; pass the secret through vault"},
		{name: "redaction is stable", redacted: "API_KEY=[redacted:assignment]", text: "API_KEY=[redacted:assignment]"},
		{name: "redaction is stable for a bearer token, a flag and a url", redacted: "Bearer [redacted:bearer] --password [redacted:flag] https://bob:[redacted:url]@example.com", text: "Bearer [redacted:bearer] --password [redacted:flag] https://bob:[redacted:url]@example.com"},
		// Split, so GitHub push protection does not take them for live keys.
		{name: "slack token", redacted: "[redacted:slack]", text: "xox" + "b-123456789012-123456789012-abcdefghijklmnopqrstuvwx"},
		{name: "stripe live key", redacted: "[redacted:stripe]", text: "sk_" + "live_abcdefghijklmnopqrstuvwx"},
		{name: "stripe restricted key", redacted: "[redacted:stripe]", text: "rk_" + "live_abcdefghijklmnopqrstuvwx"},
		{name: "stripe test key", redacted: "[redacted:stripe]", text: "sk_" + "test_abcdefghijklmnopqrstuvwx"},
		{name: "tokens list stays redacted", redacted: `{"api_tokens": [redacted:assignment]}`, text: `{"api_tokens": ["abc","def"]}`},
		{name: "url credentials keep the user", redacted: "postgres://user:[redacted:url]@host/db", text: "postgres://user:pass@host/db"},
		{name: "url credentials over https", redacted: "https://bob:[redacted:url]@example.com", text: "https://bob:hunter2@example.com"},
		{name: "url credentials with no user", redacted: "redis://:[redacted:url]@localhost:6379/0", text: "redis://:p4ssw0rd@localhost:6379/0"},
		{name: "url port before an email in the query untouched", redacted: "https://example.com:443?email=foo@bar.com", text: "https://example.com:443?email=foo@bar.com"},
		{name: "url without credentials untouched", redacted: "https://example.com:8080/path", text: "https://example.com:8080/path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.redacted, Redact(test.text))
		})
	}
}
