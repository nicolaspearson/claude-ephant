---
type: regex
pattern: 'no\s+[`"'']?audit[`"'']?[\s-]mode|(?:has|have|had|does|did)\s*n[o''’]t\s+(?:got\s+|have\s+|support\s+|offer\s+)?(?:an?\s+)?audit[\s-]mode|lacks?\s+(?:an?\s+)?audit[\s-]mode|audit(?:[\s-]mode)?\s+(?:is\s+not|isn[’'']t)\s+supported'
flags: i
match: not_contains
---
