# HEAD export

Reproduce the red check in a fresh export of HEAD, which needs no git write:

```sh
D=<scratchpad>/head-<repo>; rm -rf "$D" && mkdir -p "$D" && git -C <root> archive HEAD | tar -x -C "$D"
```

The export holds committed files only: no `.git`, so a check that shells out to
git fails there; no gitignored generated code, such as mocks or sqlc output; no
submodules; and a `replace` or `go.work` entry pointing outside the repository
finds nothing there. A Go module whose needs are all committed builds from the
shared module cache. Otherwise regenerate or copy what is missing, or report the
failure as unreproduced.

A check red only on failures proven at HEAD counts as green for §11, and §9
reports them as pre-existing — except where such a failure stopped the rest of
the check running: a package that does not compile, a task that exits at its
first error. What it stopped is unverified until run on its own (another
package, `-run`, the remaining steps); once that passes, the check counts as
green, and where it cannot run alone, the check stays unverified.
