---
type: regex
pattern: '^(?:(?!(?:^|\n)[ \t]*(?:#{1,6}[ \t]*(?:\d+[.)][ \t]*)?|(?:[-*+][ \t]+)?\*\*[ \t]*)(?:what[ \t]+|claims?[ \t]+(?:that[ \t]+|i[ \t]+)?|these[ \t]+|everything[ \t]+else[ \t]+|(?:commands?|flags?)(?:[ \t]+and[ \t]+(?:commands?|flags?))?[ \t]+that[ \t]+(?:are[ \t]+)?)?(?:(?:still|also)[ \t]+)?(?:holds?|works?|fine|valid|verified|ok|checks?[ \t]+out|not[ \t]+checked|unchecked|skipped|couldn.t[ \t]+(?:be[ \t]+)?check(?:ed)?|could[ \t]+not[ \t]+be[ \t]+checked)\b(?:[ \t]*(?:[:—–(]|-[ \t])(?:(?!recall)[^\n*])*)?[ \t]*(?:\*\*|(?=\n|$)))[\s\S])*?recall\s+--project'
flags: i
match: not_contains
---
