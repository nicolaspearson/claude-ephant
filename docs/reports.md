# Reports

`/ele:write-report` drafts one report of what a session researched, changed and
found, for one reader outside it: a client, a stakeholder or a teammate. It asks
where the report is going, shapes it to what that medium can carry, and has
every claim checked against the session's evidence before you see it. It sends
nothing: Slack and WhatsApp text comes back ready to paste, and a file is
written where you say.

```text
/ele:write-report [medium] [for whom] [what they must do] [since when]
```

It runs only when you type it. Say as much as you like when you do; it asks for
the rest in one question.

## A run

1. **Checks the code it will describe.** In each repository the session
   touched, uncommitted changes the last checkpoint does not cover are offered
   to `/ele:checkpoint` first. Decline, and the report calls them "changed, not
   yet reviewed", never done
2. **Asks** what the invocation did not say: the medium, what the reader must
   do (decide, approve, act or know), and whether they are inside your team or
   outside it
3. **Saves the evidence** into the session scratchpad: each commit a claim
   points to, the diff with new files included, each command output and note
   it rests on, and the text of each web page it cites, downloaded again. A
   page it cannot download as text, such as one behind a sign-in or drawn by
   script, gets the fetch tool's quotes instead, marked as a model's, and the
   closing reply names it for you to open before you send
4. **Writes the core** once, in your report voice: the bottom line, the ask
   with its owner, the findings each with its evidence and date, what changed,
   the options with a recommendation, and what is open. It covers everything
   the session found, whatever the reader must do: their action picks the
   opening and the ask, and the rest gets a line each. A claim with no evidence
   is left out
5. **Renders** it for the medium:

   | Medium   | What it carries                                                                     |
   | -------- | ----------------------------------------------------------------------------------- |
   | Slack    | Every item, a line each; bold lines for headings, lists for tables, bare links      |
   | WhatsApp | At most 4,096 characters, else a Markdown file; bold headings, lists, bare links    |
   | Markdown | The whole core: headings, tables, links                                             |
   | HTML     | The Markdown, in one self-contained file: styles inline, nothing fetched when open |
   | PDF      | The HTML, printed by Chrome or `weasyprint`                                         |

   No charts. Punctuation is what a keyboard types: `-`, `...`, `->` and
   straight quotes
6. **Reviews** the draft before you see it. A reviewer with Read, Grep and Glob
   only, so a web page written to steer it cannot run anything, checks each
   claim against the saved evidence, each render against its medium, and
   for an outside reader that no internal host, path, ticket or other client's
   name gets through. A secret is cut whoever the reader. Findings are fixed and
   the draft reviewed once more; an unbacked claim is dropped, never softened,
   and one still open is shown to you beside the draft. The review may run in
   the background: a turn that ends while it runs says so, and the block
   follows on its own when the review returns. One report runs at a time: a
   report you start while another is still being drafted or reviewed does
   nothing but ask you to run it again once that one is delivered, and the
   first carries on. A report that stopped for a checkpoint goes ahead when you
   run it again once that checkpoint is done; run while any checkpoint over the
   same changes is still going, it says it waits. Tell it to run anyway and it
   does
7. **Delivers** it. Slack and WhatsApp text is the closing reply, in a block to
   copy. A file goes to the path you confirm, by default
   `~/Downloads/<slug>-<date>.<ext>`, and a PDF is opened for you

## Reports in your voice

The shipped voice writes as one person to a reader who was not there: the
current state with no session history, the assumption inside each question so
yes is enough, every figure dated, and limits stated plainly. Point
`report_voice` at your own, per owner if your clients and your teammates should
hear different voices; see [Options](data-and-privacy.md#options).

## What it leaves behind

The core, the saved evidence and the renders stay in the session scratchpad
until it goes. Nothing is written to ele's data, and no note is kept: only you
know whether the report was sent. A report raises no notes ask of its own, but
work the session did before it still does.

To build HTML it needs `pandoc`; without it the HTML is written by hand. A PDF
needs Chrome or `weasyprint`; with neither, you get the HTML and are told why.
