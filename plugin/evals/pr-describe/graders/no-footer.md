---
type: regex
pattern: '(?<!~)(~{3,})[^\n~]*\n(?:(?!\n\1(?!~))[\s\S])*?Generated with \[?Claude Code(?:(?!\n\1(?!~))[\s\S])*\n\1(?!~)'
flags: i
match: not_contains
---
