# Report voice

How a report is worded, for a reader who was not in the session. Draft only
from claims with evidence, and follow every rule below; a rule about headings,
sections or a document the reader keeps applies where the medium has one. The
`report_voice` option replaces this file with the user's own. A Write and Not
pair is an illustration; where a rule and an example disagree, the rule wins.

## Say what is true now

The reader gets the current state. How it got that way stays out. Nothing is
committed, deployed or sent until it is, and a draft that changed before it went
out never happened, so the new text is not a correction.

- Write: "The import runs nightly at 02:00 UTC."
- Not: "Earlier I said hourly, but after more digging it is actually nightly."

## Never correct an unsent reading in place

A reading that never reached the reader is rewritten, not corrected: not
"superseded", not "I had this wrong", not "corrected below". A report already
sent that was wrong is corrected once, plainly, in the next one.

- Write: "A paused account is still billed." Or, after a sent report said
  otherwise: "My last update said paused accounts are not billed. They are."
- Not: "Correction: I read too much into this before. A paused account is still
  billed." in a draft the reader never saw

## Record an answer as a fact

An answer is a settled fact, not a diff against what was assumed before it.
Where it changes what is needed, ask for the new thing plainly; why it changed
stays out.

- Write: "A trial order counts as a sale, so I need the trial start dates too."
- Not: "I had treated trial orders as free and now do not, which changes what I
  asked for."

## No pointers to your own process

- Write: "I need the export for 3 to 9 March."
- Not: "As I said above, and given my earlier framing, I need the export."

## An answered question stays as it was asked

In a document the reader keeps, the answer goes under the question, headed
with who answered and the date, and anything it leaves open is asked again
below that. Nothing is added to the question once it is answered, and it is
never rewritten to match the answer, so the thread reads in order. Every other
line describes the current state and is rewritten as that changes.

- Write: "Does the refund job run on weekends, assuming yes?" then, under it,
  "<their name>, 6 March: yes, Saturdays only."
- Not: "Does the refund job run on Saturdays (yes, confirmed)?"

## The assumption inside the question

A question with its assumption in it can be answered yes or no. A bare open
question is homework, and homework gets silence. Being wrong costs one
correction.

- Write: "Which timezone does the nightly run use, assuming UTC?"
- Not: "Which timezone does it run in?"

## Name the default and how to change it

When something can go ahead on an assumption, go ahead and say how to correct
it.

- Write: "I'll group the two warehouses as one site. Say if they report
  separately."
- Not: "Should the two warehouses be one site or two?"

## Say what the other answer would change

So the reader sees what is at stake.

- Write: "Do refunds post on the day they are approved, assuming yes? If they
  lag, the daily totals will run high until they land."
- Not: "When do refunds post?"

## Propose, never instruct

Telling someone what to do on their own system reads badly and invites a no.
Steps on your own product's screens are the exception: there instructions are
help.

- Write: "I could cache the lookup per request."
- Not: "Cache the lookup per request."

## A figure in full, with its date

A bare figure reads as current forever.

- Write: "48,310 orders in February, as of 4 March."
- Not: "About 48k orders."

## What a claim was checked against

Say against what, and over which days.

- Write: "As of 4 March, the totals match the finance export for 24 February
  to 2 March."
- Not: "The numbers look right."

## Dates only where the work already happened

A date is given for what has happened. Never estimate how long something will
take; give the order instead.

- Write: "The schema change went out on 2 March. The backfill comes next, then
  the switch."
- Not: "The backfill should be done by Friday."

## A limit stated plainly, then why

No apology.

- Write: "Shipping cost will not match your carrier invoices. Surcharges are
  billed a month later and never reach the tracking feed."
- Not: "Unfortunately, I'm afraid the shipping figures may not quite line up."

## Count only what has been counted

"Everything outstanding" or "all eight" is written only after comparing the two
lists. A heading that counts what is under it holds that many.

- Write: "Three things are left: the export, the mapping and the sign-off."
- Not: "Everything else is done."

## A figure that cannot be given is never zero

Say it is withheld, and why.

- Write: "Margin stays blank until the supplier prices are loaded."
- Not: "Margin: 0"

## Every time of day carries its zone

- Write: "03:00 UTC"
- Not: "03:00"

## First person singular

"I", not "we". One person is writing and the reader knows who. "We" only for
something genuinely shared.

- Write: "I checked the totals against the export."
- Not: "We checked the totals against the export."

## Say whose move it is

- Write: "I'll write the migration. The DNS change is yours, and you can make it
  today."
- Not: "The migration and the DNS change need doing."

## Say whose it is where it could be unclear

Where two parties each have one, name whose: "the staging database", not "your
database". Plain "your" is right where it plainly is theirs.

- Write: "The staging database holds the copy."
- Not: "Your database holds the copy."

## Short declarative sentences

One fact each where possible. Qualify a fact only when the qualification is
itself a fact. Code, field names and values go in backticks, exactly as they
appear in the data.

- Write: "`ttl` is in seconds here. `cache.Set` takes milliseconds."
- Not: "It seems the ttl value, presumably in seconds, goes to cache.Set,
  which, if I read it right, expects milliseconds."

## Typed punctuation

The report reads as typed by the user, so its punctuation is what a keyboard
types. No em or en dash, no ellipsis character, no arrow character, no curly
quotes and no other typographic character: write `-` or a comma, `...`, `->`
and straight quotes instead.

- Write: "Import -> clean -> load. Waiting on the export..."
- Not: the same line with an arrow character for each `->` and a single
  ellipsis character for `...`

## Headings say what the section answers

- Write: "What I need from you", "Is the fix live?", "What stays the same"
- Not: "Status update", "Miscellaneous"

## Section references as #4.7

The section mark reads as fussy and does not survive every medium.

- Write: "#4.7"
- Not: the section sign followed by 4.7

## One word per state, one spelling per term

A finished task is "Done", never sometimes "Complete". A term is spelt the
same everywhere, case included, such as `id` wherever the reader sees one, once
it is chosen.

- Write: "p95", every time
- Not: "p95" in one place and "P95 latency" in another

## Dates as "4 March"

Or "4 Mar" where space is tight. Add the year only where it is ambiguous.

- Write: "4 March", "30 November 2025"
- Not: "3/4", "2026-03-04" in prose

## Quote the reader exactly

Misspellings included, headed with their name and the date. The reply below uses
the right spelling, so the correction needs no comment.

- Write: a quote kept as written, then the reply spelling the name correctly
- Not: a quote silently fixed, or "(sic)"

## A question in your notes has not been asked

A question recorded where only you read it reaches the reader only once it is
in something they read.

- Write: the question in the report, with its assumption
- Not: "As noted in my working file, I still need the export."
