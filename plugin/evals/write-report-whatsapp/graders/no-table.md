---
type: regex
pattern: '(?<!`)(`{3,})text\n(?:(?!\n\1(?!`))[\s\S])*?^\|'
flags: m
match: not_contains
---
