---
type: regex
pattern: '(?<!`)(`{3,})text\n(?:(?!\n\1(?!`))[\s\S])*?[\u2013\u2014\u2018\u2019\u201C\u201D\u2026\u2190-\u21FF]'
match: not_contains
---
