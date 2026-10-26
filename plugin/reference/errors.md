# Errors

When a call fails, quote its error verbatim, find its row below, apply that
row's one fix and run once more, unless the row says not to rerun or retry —
never the same call unchanged. An error in no row, or a second failure: stop and
report it, quoting the error, where your instructions put errors.

| Error contains | Class | One fix |
| --- | --- | --- |
| `not found` after a word starting with `=`, such as `=== not found` | zsh took the word, often an `echo ====` separator, for a command and dropped the rest of the line: nothing after it ran | Quote it, `echo '===='`, or use `echo ----`; rerun what came after it |
| `Exit code 1` with no error text at the end | Usually a `grep` that matched nothing, or a command with stderr sent to `/dev/null`; the output cannot tell which. Output ending in `Traceback`, an `…Error:` line, `no such file`, `FAIL`, `panic`, `fatal:` or `command not found` is a real failure, not this row | Read the output; if what is missing matters, rerun that part alone with stderr kept |
| `no matches found: <glob>` | zsh expanded an unquoted glob that matched nothing; that command did not run | Quote the glob |
| `Blocked: cd chained with an edit` | A guard hook, where one is installed | Absolute paths, no `cd` |
| `Blocked: git writes are the user's to run` | A guard hook, where one is installed: only listed git reads pass, with or without `-C`, and one unlisted git command blocks the whole call; `branch -a`, `branch -r`, `branch --contains`, `tag --contains`, `tag --sort` and `reflog` are not listed | Run the listed reads apart from the rest; for the unlisted, `git for-each-ref --contains <sha> refs/heads refs/remotes refs/tags`, `git for-each-ref refs/heads refs/remotes`, or `git log -g`; a real write goes to the user |
| `Blocked: slow command piped to tail/head` | A guard hook, where one is installed | Capture to a scratchpad file, then grep it |
| `File content (N tokens) exceeds maximum allowed tokens` | A file, often a frozen diff, too big for one Read | Read it in parts with offset and limit, or grep it |
| Edit or Write: `File has been modified since read`, `String to replace not found` or `File has not been read yet` | Stale or missing read | Read the file again, then one Edit |
