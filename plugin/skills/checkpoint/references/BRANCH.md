# Branch scope

Every commit since the base, plus the uncommitted work. Throughout `SKILL.md`,
HEAD, where it names what the change is compared against, reads the base: a
path returned to the base leaves the change list, every freeze passes
`--base <base>`, and `HEAD-EXPORT.md`'s export archives `<base>`, not `HEAD`.
Where its sections differ further:

- §1: the base is the one the caller names, such as `/ele:plan` pickup's
  commit that ticked the previous item; with none named, it comes from

  ```sh
  ele checkpoint base <root>
  ```

  It prints the merge-base with the upstream, else with `origin/HEAD`, and
  refuses with neither, or with no shared history: ask the user for the base. A
  base equal to HEAD means nothing is unpushed. The change list takes
  `git -C <root> diff <base> --name-status -M` in place of `git diff HEAD`
- §2: `git log <base>..HEAD`'s subjects join what was asked
- §3: the brief names the base and the commit range, and that the prose
  reviewer runs `ele checkpoint density --ref <base> <root>`
- §9: only the uncommitted part gets a commit; what is committed is already
  there
