package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandIsWork(t *testing.T) {
	tests := []struct {
		command string
		name    string
		work    bool
	}{
		{command: "", name: "an empty command", work: true},
		{command: "A=1 B=2", name: "assignments alone"},
		{command: "GIT_EXTERNAL_DIFF=./x.sh git diff", name: "an assignment before a command", work: true},
		{command: "S=/private/tmp/s; grep -rn x . > \"$S/hits.txt\" 2>&1", name: "a redirect to an assigned scratch directory"},
		{command: "S=/private/tmp/s; grep x . > \"$S2/hits.txt\"", name: "a redirect to a longer, unassigned variable", work: true},
		{command: "S=/private/tmp/s; grep x . > \"${S}/hits.txt\"", name: "a redirect to a braced assigned variable"},
		{command: "X=../Users; echo hi > /tmp/$X/f", name: "a variable inside a /tmp target", work: true},
		{command: "cat <(rm cat)", name: "a write inside a process substitution", work: true},
		{command: "cat <<'EOF' > notes.md\nx\nEOF", name: "a heredoc into a file", work: true},
		{command: "cat <<EOF\necho it's\nEOF\nrm -rf y #'", name: "a heredoc whose quote runs past its delimiter", work: true},
		{command: "cat <<EOF\nls\nEOF", name: "a heredoc of read-like lines", work: true},
		{command: "cat <> new.txt", name: "a read-write redirect", work: true},
		{command: "cat a.go | grep -n x | head -5", name: "a pipeline of reads"},
		{command: "cd /w && git log --oneline -3", name: "cd then a git read"},
		{command: "cd /w; ele plan show", name: "cd then an ele read"},
		{command: "diff <(ls a) <(ls b)", name: "a process substitution", work: true},
		{command: "echo \"$(cat <<\"EOF\"\nx\nEOF\n)\"", name: "a double-quoted-delimiter heredoc in a substitution"},
		{command: "echo \"$(cat <<'EOF'\nit's done\nEOF\n)\"", name: "a quoted-delimiter heredoc in a substitution"},
		{command: "echo \"$(cat <<'EOF'\nx\n\tEOF\n)\"", name: "a tab-indented delimiter without <<-", work: true},
		{command: "echo \"$(cat <<'EOF'\nx\nEOF\nrm y\n)\"", name: "a command after a heredoc in a substitution", work: true},
		{command: "echo \"$(cat <<'EOF'\nx\nEOF\nrm y\nEOF\n)\"", name: "a command between two delimiter lines", work: true},
		{command: "echo \"$(cat <<-'EOF'\n\tx\n\tEOF\n)\"", name: "a tab-stripped heredoc in a substitution"},
		{command: "echo \"$(cat <<EOF\n$(rm x)\nEOF\n)\"", name: "an unquoted-delimiter heredoc in a substitution", work: true},
		{command: "echo \"$(echo \")\"; rm x)\"", name: "a write after a quoted parenthesis in a substitution", work: true},
		{command: "echo \"$(rm -rf x)\"", name: "a write inside a quoted substitution", work: true},
		{command: "echo $(echo \"$(echo \")\")\")", name: "a quoted substitution inside a substitution"},
		{command: "echo $(echo \")\")", name: "a quoted parenthesis in a substitution"},
		{command: "echo $(echo \"<<x\")", name: "a quoted << in a substitution"},
		{command: "echo $(echo \"a\\\")\")", name: "an escaped quote in a quoted string in a substitution"},
		{command: "echo $(echo $(rm x))", name: "a write inside a nested substitution", work: true},
		{command: "echo $(git status --porcelain) `pwd`", name: "reads inside substitutions"},
		{command: "echo 'unclosed", name: "an unclosed quote", work: true},
		{command: "echo `rm x`", name: "a write inside backticks", work: true},
		{command: "echo a#b; rm x", name: "a hash inside a word", work: true},
		{command: "echo hi &> /tmp/out.txt", name: "both streams to /tmp"},
		{command: "echo hi >&1x", name: "a duplication to a file", work: true},
		{command: "echo hi >| out.txt", name: "a clobbering redirect", work: true},
		{command: "echo x >&2", name: "a duplication to stderr at the end"},
		{command: "ele checkpoint log /w", name: "an ele write", work: true},
		{command: "ele plan start p.md", name: "an ele plan write", work: true},
		{command: "ele recall colour 2>&1 | head -40", name: "an ele read with stderr joined"},
		{command: "ele reminder close 20261023-ba83b6", name: "ele bookkeeping"},
		{command: "find . -name '*.go'", name: "find listing"},
		{command: "find . -name '*.go' -exec rm {} +", name: "find running a command", work: true},
		{command: "find . -name '*.tmp' -delete", name: "find deleting", work: true},
		{command: "for f in a b; do\n  grep -n x \"$f\"\ndone", name: "a loop of reads"},
		{command: "git --no-pager log -3", name: "a git read without a pager"},
		{command: "git -C /w status --porcelain", name: "a git read with -C"},
		{command: "git -c diff.external=./x.sh diff", name: "git with a config override", work: true},
		{command: "git branch --show-current", name: "git branch showing the current branch"},
		{command: "git branch -D old", name: "git branch deleting", work: true},
		{command: "git commit -m x", name: "a git write", work: true},
		{command: "git diff --output=x.patch", name: "git diff writing a file", work: true},
		{command: "git show --output=x HEAD", name: "git show writing a file", work: true},
		{command: "go test ./...", name: "a command not listed", work: true},
		{command: "grep -E 'a|rm -rf b' f.go", name: "a pipe inside quotes"},
		{command: "grep x . > \"$HOME/hits.txt\"", name: "a redirect to an unassigned variable", work: true},
		{command: "grep x . > /tmp/../Users/hits.txt", name: "a redirect escaping /tmp", work: true},
		{command: "grep x f.go\nrm -rf y", name: "a write on its own line", work: true},
		{command: "ls 2>> errors.txt", name: "an appending stderr redirect", work: true},
		{command: "ls > out.txt", name: "a redirect into the repository", work: true},
		{command: "rg --hostname-bin=./x.sh x", name: "rg running a hostname command", work: true},
		{command: "rg --pre ./decode.sh x", name: "rg running a preprocessor", work: true},
		{command: "rg -n x", name: "rg searching"},
		{command: "sed 's/a/b/e' f.go", name: "sed running the substitution", work: true},
		{command: "sed 's/a/b/g' f.go", name: "sed substituting to stdout"},
		{command: "sed 's/a/b/w out' f.go", name: "sed writing substitutions", work: true},
		{command: "sed -f script.sed f.go", name: "sed from a script file", work: true},
		{command: "sed -i 's/a/b/' f.go", name: "sed in place", work: true},
		{command: "sed -ie 'p' f.go", name: "sed in place with an expression flag", work: true},
		{command: "sed -n '/x/,/y/p' f.go", name: "sed printing a pattern range"},
		{command: "sed -n 'p;q' f.go", name: "sed printing then quitting"},
		{command: "sed -n 'w out' f.go", name: "sed writing a file", work: true},
		{command: "sed -n -e '1p' -e '$p' f.go", name: "sed printing through expressions"},
		{command: "sed -n -e 'w x' f.go", name: "sed writing through an expression", work: true},
		{command: "sed -n 1,20p f.go # head of file", name: "sed printing, with a comment"},
		{command: "sort --compress-program=./x.sh in.txt", name: "sort running a compressor", work: true},
		{command: "sort -o out.txt in.txt", name: "sort writing a file", work: true},
		{command: "sort -uo out.txt in.txt", name: "sort writing a file with grouped flags", work: true},
		{command: "uniq in.txt", name: "uniq to stdout"},
		{command: "uniq in.txt out.txt", name: "uniq writing a file", work: true},
		{command: "wc -l < f.go", name: "an input redirect"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.work, commandIsWork(test.command))
		})
	}
}
