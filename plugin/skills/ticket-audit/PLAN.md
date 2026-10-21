# A plan as the corpus

An ele plan, a Markdown file under `.documentation/plans/`, audits as a map: each
item under `## Actions` or `## Inbox`, its bullet line and indented body, stands
for a ticket, a ticked one for a closed ticket and an unticked Inbox one for an
untriaged ticket, state `inbox`. Where the sections of `SKILL.md` differ for a
plan:

- §0: recall the plan's filename and `audit` rather than a map number
- §1: `ele ticket corpus --plan <path> --out "$AUDIT"`
  writes each item under Actions and Inbox as `<nn>.md`, in file order. Add its
  `spec` from the front matter, and read its Context and Decisions as the
  plan's notes. Items carry no ids: name one by its line number and title,
  re-reading the line number before citing it, since every edit moves it. Its
  ledger key is the plan's path plus its bullet line, so rewording the bullet or
  moving the plan loses its rulings
- Frontier: the open Actions items with no `blocked:` line. A `blocked:` reason
  is prose, not an edge
- §4 and §5 are what a plan audit is for and are never left out
- §6: no graph. Check each `blocked:` reason still holds, and that file order
  builds: an item reading what a later item produces is out of order
- §7: a decision lives once, in Decisions or the spec; an item restating it is a
  second copy. An inbox item that an action already covers, or contradicts, is a
  finding
- §9: a fix is an edit to a repository file, not a tracker write: propose each,
  apply it on approval, no script. Audit the edits as §9 says. A ruling that
  commits an item to an edit not made in this audit is also written as a
  `next: <date>, …` line in the item's body
- Elsewhere: `<n>.md` is `<nn>.md`; a ticket's comments are its item's body;
  the map's notes are the plan's notes; `<name>` is the basename of the
  repository holding the plan
