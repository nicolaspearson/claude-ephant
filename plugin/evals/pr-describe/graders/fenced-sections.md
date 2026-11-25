---
type: regex
pattern: '(?<!~)(~{3,})[^\n~]*\n(?:(?!\n\1(?!~))[\s\S])*?^## Summary$(?:(?!\n\1(?!~))[\s\S])*?^## Evidence$(?:(?!\n\1(?!~))[\s\S])*?^## Merge Danger$(?:(?!\n\1(?!~))[\s\S])*\n\1(?!~)'
flags: m
---
