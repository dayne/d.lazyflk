# Working on lazyflk (for AI agents)

lazyflk is a terminal UI for hosts running flk. flk is a separate project
(Ruby, github.com/dayne/d.flk; on the reference host its checkout is
`~/flk`, called `$FLK_SRC` below). `SPEC.md` is the source of truth for
scope, principles and screens; v1 is done (SPEC section 8), and the open
work is in `TODO.md`.

Ground rules:

- **flk's CLI is the only interface** (SPEC section 3.1). Run `flk ... --json`;
  never read `~/.local/share/flk` or call firecracker, tmux or nft directly.
  If flk lacks something, it's a gap: check SPEC section 9 and propose the
  flk change instead of working around it. Another agent develops flk:
  register gaps with it (the "For lazyflk" section of flk's `TODO.md`, and
  a message to the flk session if one is running), never build them here
  or in flk's repo yourself.
- **Learn flk from flk:** `flk help --all` (full reference),
  `flk help --all --json` (the command table, ideal for the palette),
  `flk context --all`.
- **Compatibility:** flk's JSON carries `"schema"` (`internal/flk/types.go`
  has the number lazyflk understands). Unknown fields are ignored and
  missing ones mean "this flk doesn't support it", so lazyflk keeps working
  with an older flk.
- **Real data:** `fixtures/` has actual output from the reference host, at
  the flk commit named in `fixtures/README.md`. Build and test against it
  first (`--demo`), and add new fixtures by running the same commands
  (`fixtures/capture.sh`), never by hand-editing them.
- **Don't touch real apps.** Every app not named `flkt-*` is real: on the
  reference host, game servers with players' data (`minecraft-flk`,
  `vintagestory-flk`), `opencode-sandbox` and `mqtt-flk`; new ones appear
  without notice, and an app with a problem sorts first in the tree. In a
  live run, always start lazyflk with `--test-apps flkt-` (it refuses
  actions on any other app), and still read the confirm box: press y only
  when its `$ flk -a NAME` names a `flkt-*` app. Read-only commands against
  them are fine; for anything that starts, stops, deploys or destroys, use
  test apps named `flkt-*` (create them from
  `$FLK_SRC/tests/apps/hello` or `mcshadow`) and clean them up with
  `flk apps destroy NAME --yes`.
- **Never run sudo.** Root steps are shown to the admin, not run.

Build and test: `make build`, `make test`, `make check` (what CI runs)
and `make snapshots` (rewrites the screen snapshots; review the diff); or
the go commands they show (`make` lists them all).
