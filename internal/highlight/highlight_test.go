package highlight

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var markers = strings.NewReplacer(
	"[b]", "\033[1m", "[/b]", "\033[22m",
	"[c]", "\033[36m", "[/c]", "\033[39m",
	"[d]", "\033[2m", "[/d]", "\033[22m",
	"[g]", "\033[32m", "[/g]", "\033[39m",
	"[m]", "\033[35m", "[/m]", "\033[39m",
	"[r]", "\033[31m", "[/r]", "\033[39m",
	"[y]", "\033[33m", "[/y]", "\033[39m",
)

func TestMessage(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	tests := []struct {
		highlighted string
		name        string
		text        string
	}{
		{name: "blocked label", highlighted: "[b]Blocked:[/b]\n- Phase 2\n  [r]blocked:[/r] waits on the hook ports", text: "Blocked:\n- Phase 2\n  blocked: waits on the hook ports"},
		{name: "blocked label needs an indent", highlighted: "line\n\nblocked: x", text: "line\n\nblocked: x"},
		{name: "code span wins over the path and command inside it", highlighted: "run [c]`ele plan done a/b.md`[/c] now", text: "run `ele plan done a/b.md` now"},
		{name: "command keeps its path and drops the full stop", highlighted: "[c]ele plan show .documentation/plans/x.md[/c].", text: "ele plan show .documentation/plans/x.md."},
		{name: "command without an argument stops before prose", highlighted: "[c]ele plan show[/c] prints one", text: "ele plan show prints one"},
		{name: "control characters dropped and a tab kept", highlighted: "[b]Next:[/b]\t[31mred title", text: "Next:\t\033[31mred\u009b title\a"},
		{name: "counts on a several-plans line", highlighted: "- [c]a/b.md[/c]: [y]2 open[/y], [r]1 blocked[/r], [d]0 done[/d], [c]4 in the inbox[/c]. [b]Next:[/b] Item", text: "- a/b.md: 2 open, 1 blocked, 0 done, 4 in the inbox. Next: Item"},
		{name: "counts with zeros dimmed", highlighted: "[b]Plan[/b] [c].documentation/plans/x.md[/c]: [y]10 open[/y], [d]0 blocked[/d], [g]1 done[/g], [d]0 in the inbox[/d].", text: "Plan .documentation/plans/x.md: 10 open, 0 blocked, 1 done, 0 in the inbox."},
		{name: "ele's own warning", highlighted: "[y]ele could not find the checkout for 1 active plan: gone[/y]; more", text: "ele could not find the checkout for 1 active plan: gone; more"},
		{name: "handoff header with its note dimmed", highlighted: "[b]Handoff from ele[/b], [d]note 20261019-301b5a of 2026-10-19 09:01Z, session 2be5afc9[/d], shown once:", text: "Handoff from ele, note 20261019-301b5a of 2026-10-19 09:01Z, session 2be5afc9, shown once:"},
		{name: "headers", highlighted: "[b]Active plan from ele:[/b]\n[b]2 active plans from ele[/b], newest first:\n[b]3 handoffs from ele[/b], newest first", text: "Active plan from ele:\n2 active plans from ele, newest first:\n3 handoffs from ele, newest first"},
		{name: "missing lone plan", highlighted: "[y]ele: plan a/b.md is not in this checkout[/y]; switch to its branch, or [c]ele plan done a/b.md[/c] closes it", text: "ele: plan a/b.md is not in this checkout; switch to its branch, or ele plan done a/b.md closes it"},
		{name: "missing plan among several", highlighted: "- [y]plan b.md is not in this checkout[/y]; switch", text: "- plan b.md is not in this checkout; switch"},
		{name: "note id alone", highlighted: "see [d]20261019-1ede12[/d] first", text: "see 20261019-1ede12 first"},
		{name: "sub-day nudge's due time yellow and interval magenta", highlighted: "- [m]↻[/m] due [y]2026-10-22T15:05[/y], reminder [d]20261022-3abb91[/d], [m]every 5 minutes[/m]: Check.\n- [y]due[/y]  due [y]2026-10-22T09:30[/y], reminder [d]20261022-dddddd[/d] (project, [m]every 1 hour[/m]): Later.", text: "- ↻ due 2026-10-22T15:05, reminder 20261022-3abb91, every 5 minutes: Check.\n- due  due 2026-10-22T09:30, reminder 20261022-dddddd (project, every 1 hour): Later."},
		{name: "nudge marker and interval magenta, due date yellow", highlighted: "- [m]↻[/m] due [y]2026-10-22[/y], reminder [d]20261022-3abb91[/d], [m]every 7 days[/m]: Run [c]/hook-stats[/c].", text: "- ↻ due 2026-10-22, reminder 20261022-3abb91, every 7 days: Run /hook-stats."},
		{name: "path inside a URL stays plain", highlighted: "see http://x/y.md", text: "see http://x/y.md"},
		{name: "path is not a slash command", highlighted: "under /private/tmp/x and http://host/a", text: "under /private/tmp/x and http://host/a"},
		{name: "plain reminder's due date yellow and body's words plain", highlighted: "[b]Reminders from ele[/b], due:\n- due [y]2026-10-23[/y], reminder [d]20261023-ba83b6[/d]: Due 2026-11-01, every 7 days.", text: "Reminders from ele, due:\n- due 2026-10-23, reminder 20261023-ba83b6: Due 2026-11-01, every 7 days."},
		{name: "previous commit and date dimmed", highlighted: "[b]Previous:[/b] Add the rule [d](45404d0, 2026-10-17)[/d]\n[b]Next, line 22:[/b]\n- [ ] Item", text: "Previous: Add the rule (45404d0, 2026-10-17)\nNext, line 22:\n- [ ] Item"},
		{name: "reminder list's closed and superseded states dimmed", highlighted: "- [d]closed 2026-10-22T09:00:00Z[/d]  due [y]2026-10-20[/y], reminder [d]20261020-aaaaaa[/d] (global): Done.\n- [d]superseded by 20261022-bbbbbb[/d]  due [y]2026-10-21[/y], reminder [d]20261021-cccccc[/d] (project): Moved.", text: "- closed 2026-10-22T09:00:00Z  due 2026-10-20, reminder 20261020-aaaaaa (global): Done.\n- superseded by 20261022-bbbbbb  due 2026-10-21, reminder 20261021-cccccc (project): Moved."},
		{name: "reminder list's header bold, a due nudge's state yellow and its commands cyan", highlighted: "[b]Reminders for corp · gateway[/b], soonest due first:\n- [y]due[/y]  [m]↻[/m] due [y]2026-10-23[/y], reminder [d]20261022-3abb91[/d] (global, [m]every 7 days[/m]): Run (every 7 days): now.\n- open  due [y]2099-01-01[/y], reminder [d]20261022-dddddd[/d] (project): Later.\n[c]ele reminder close <id>[/c]... or [c]ele reminder reopen <id>[/c]... to change one; [c]ele forget <id>[/c] drops it.", text: "Reminders for corp · gateway, soonest due first:\n- due  ↻ due 2026-10-23, reminder 20261022-3abb91 (global, every 7 days): Run (every 7 days): now.\n- open  due 2099-01-01, reminder 20261022-dddddd (project): Later.\nele reminder close <id>... or ele reminder reopen <id>... to change one; ele forget <id> drops it."},
		{name: "slash command before a curly apostrophe", highlighted: "run [c]/clear[/c]’s twin", text: "run /clear’s twin"},
		{name: "slash command glued to a letter or digit stays plain", highlighted: "run /clearÉ and /clear2", text: "run /clearÉ and /clear2"},
		{name: "stop-check hint", highlighted: "[b]Context 162k.[/b] Write a handoff ([c]/ele:handoff[/c]), then [c]/clear[/c]", text: "Context 162k. Write a handoff (/ele:handoff), then /clear"},
		{name: "uncommitted previous", highlighted: "[b]Previous:[/b] Item [d](uncommitted)[/d]", text: "Previous: Item (uncommitted)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, markers.Replace(test.highlighted), Message(test.text, true))
		})
	}
}

func TestMessageUncolouredOnlyDropsControlCharacters(t *testing.T) {
	tests := []struct {
		coloured bool
		name     string
		noColor  string
	}{
		{coloured: false, name: "colour off"},
		{coloured: true, name: "NO_COLOR over colour on", noColor: "1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", test.noColor)
			require.Equal(t, "Plan a/b.md: 1 open, 0 blocked, 0 done, 0 in the inbox. [31mred", Message("Plan a/b.md: 1 open, 0 blocked, 0 done, 0 in the inbox. \033[31mred", test.coloured))
		})
	}
}
