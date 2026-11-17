# WhatsApp

Text pasted into a WhatsApp chat, delivered as the closing reply's copyable
block. Read on a phone, usually by someone outside the team, so it stands alone:
no term the reader has not been given, and the bottom line first.

## Markup

As of 2026-11-13, from WhatsApp's docs, a message takes `*bold*`, `_italic_`,
`~strike~`, inline code, `-` and `1.` lists, and `>` quotes. It has no
headings, no tables and no `[text](url)` links:

- A heading becomes a `*bold*` line of its own
- A table becomes a list, one row a line: `- *name*: value, value`
- A link becomes the bare URL on its own line
- `**bold**` is Markdown's, not WhatsApp's: one asterisk each side

## Length

At most 4,096 characters, the cap WhatsApp's API puts on a message; the app's
own cap is not checked. Over it, cut to the bottom line, the ask and the
findings, and offer the rest as a file; never split one report into several
messages without the user asking.

## Render

1. Write the render to `<scratch>/render-whatsapp.txt` from the core, in the
   core's order, dropping or reshaping and never adding a claim
2. Count its characters for the brief: `wc -m < <scratch>/render-whatsapp.txt`
