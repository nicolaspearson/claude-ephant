---
type: regex
pattern: '(?<!~)(~{3,})[^\n~]*\n(?:(?!\n\1(?!~))[\s\S])*?\*\*Door:\*\*\s*(?:one|two)-way(?:(?!\n\1(?!~))[\s\S])*\n\1(?!~)'
flags: i
---
