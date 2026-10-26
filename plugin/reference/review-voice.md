# Review voice

How each review comment is worded. Draft a body only after its finding is
verified, and follow every rule below. The `review_voice` option replaces this
file with the user's own. Each example after the first shows the text that
follows the label.

## Label, then the claim in one line

Open with a [Conventional Comments](https://conventionalcomments.org/) label,
an `issue` label carrying the finding's kind as a decoration, then say what is
wrong in one line:

| Shown as | Label |
| --- | --- |
| blocker | `issue (blocking, <kind>):` |
| defect | `issue (non-blocking, <kind>):` |
| question | `question:` |

The kind is one of `bug`, `convention`, `security`, `spec` or `test`.

- Write: "issue (blocking, bug): `Close` runs twice when `Flush` fails, and the
  second call panics."
- Not: "**Bug:** I noticed a few things around the close path worth a look."

## First person singular

- Write: "I'd return the error here."
- Not: "We should return the error here." / "It is recommended to return the
  error."

## Propose, never instruct

- Write: "I'd guard the nil here."
- Not: "Guard the nil."

## The assumption inside the question

Put the assumption in the question, so yes is enough, and say what the other
answer would change. Where the fix can go ahead on the assumption, state it,
naming the assumption and how to change it.

- Write: "Is `retries` ever zero here, assuming the config default applies? If
  it can be, the loop never runs."
- Not: "What is `retries` here?"
- Write: "I'd cap this at three retries; say if the limit should differ."
- Not: "What should the retry limit be?"

## Evidence, never "might break"

Give the input and the wrong output it produces.

- Write: "An empty `items` reaches `items[0]` and panics."
- Not: "This might break on some inputs."

## What the claim was checked against

- Write: "I traced `handler.Create` through `store.Save`, and neither checks the
  ID."
- Not: "The ID looks unchecked."

## No narration of the review

- Write: "`Parse` drops the last field when the line has no trailing newline."
- Not: "On re-reading, I missed this earlier: `Parse` seems to drop the last
  field."

## What could not be checked, plainly

State what the review could not check, then why, without apology.

- Write: "I read this and didn't run it, so the timeout under load is
  unconfirmed."
- Not: "Sorry, I wasn't able to test this properly."

## Short declarative sentences

Put code in backticks, and use em-dashes sparingly.

- Write: "`ttl` is in seconds here. `cache.Set` takes milliseconds."
- Not: "It seems the ttl value — presumably in seconds — goes to cache.Set,
  which — if I read it right — expects milliseconds."
