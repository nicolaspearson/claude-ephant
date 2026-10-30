# Tracker commands

Run each in the repository the tracker belongs to. Count `OPEN` (gh) and
`opened` (glab) as open.

## GitHub

The state of a `?` blocker:

```sh
gh issue view <n> -R <owner/repo> --json state --jq .state
```

Each ticket's native blockers, closed ones included, with their state:

```sh
gh api --paginate repos/{owner}/{repo}/issues/<n>/dependencies/blocked_by --jq '.[] | "\(.number) \(.state)"'
```

## GitLab

The state of a `?` blocker:

```sh
glab issue view <n> -R <host/group/project> -F json | jq -r .state
```

Each ticket's links with their state and type; the blockers are the
`is_blocked_by` ones:

```sh
glab api 'projects/:id/issues/<n>/links' | jq -r '.[] | "\(.iid) \(.state) \(.link_type)"'
```
