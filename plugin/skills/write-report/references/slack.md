# Slack

Text pasted into Slack's message composer, delivered as the closing reply's
copyable block. Read in a channel or a thread, often on a phone, between other
messages, so the bottom line goes first and the rest can be skipped.

## Markup

As of 2026-11-13, from Slack's docs, the composer takes `*bold*`, `_italic_`,
`~strike~`, inline code, code blocks, `>` quotes, and `-` and `1.` lists. How
it converts pasted Markdown links, headings and tables is not checked, so the
render uses none of them:

- A heading becomes a `*bold*` line of its own
- A table becomes a list, one row a line: `- *name*: value, value`
- A link becomes the bare URL on its own, never `[text](url)`
- `**bold**` is Markdown's, not Slack's: one asterisk each side

## Length

No cap this profile checks. The bottom line and the ask first, then every other
item the core holds in a line each.

## Render

1. Write the render to `<scratch>/render-slack.txt` from the core, in its order,
   shortening or reshaping and never adding a claim, keeping every item the
   core holds, each at least a line
2. Count its characters for the brief: `wc -m < <scratch>/render-slack.txt`
