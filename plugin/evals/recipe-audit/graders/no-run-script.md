---
type: tool_used
tool: Bash
input_match: '"command"\s*:\s*"(?:(?:[^"\\]|\\.)*?(?:[;&|(]|\\n)\s*)?(?:(?:env|command|exec|timeout\s+\d+|[A-Za-z_]\w*=[^\s"\\]*)\s+)*(?:(?:(?:ba|z)?sh|node)\s+(?:-c\s+)?)?(?:\\"|'')?(?:[^\s"\\;&|()'']*/)?(?:forge(?![\w.''|)-])|cli\.js\b)'
min: 0
max: 0
arm: both
---
