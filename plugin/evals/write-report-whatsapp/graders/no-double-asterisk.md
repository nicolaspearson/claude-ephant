---
type: regex
pattern: '(?<!`)(`{3,})text\n(?:(?!\n\1(?!`))[\s\S])*?\*\*'
match: not_contains
---
