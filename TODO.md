# lazyflk TODO

lazyflk's own backlog. What lazyflk needs from flk is in flk's `TODO.md`
("For lazyflk", github.com/dayne/d.flk) and SPEC section 9; G1–G11 are done.

## Future

- [x] **Mouse support (optional, the keys stay primary).** Done: on by
  default, `--no-mouse` to leave it to the terminal. The admin
  reached for the mouse; clicking should do the obvious thing:
  - click a tree row to select it, click an app's ▾/▸ to fold it
  - click a tab to open it; wheel scrolls the tree, the tab, logs, job
    output, guides and the palette list
  - click a menu item, a palette entry or a Needs attention entry to pick
    it (same as moving there and pressing enter: confirms still apply)
  - click the footer's key hints to press that key
  *Feasible:* Bubble Tea v2 reports clicks and the wheel with cell
  positions (`tea.View.MouseMode = tea.MouseModeCellMotion`, then
  `tea.MouseClickMsg` / `tea.MouseWheelMsg`). lazyflk draws plain
  strings, so it needs hit-testing: record each clickable region (row,
  tab, menu line, hint) with its screen rectangle while rendering, and
  look clicks up there. Test with the same fixtures by sending mouse
  messages at known cells.
  *Trade-off:* with mouse reporting on, the terminal's own text
  selection needs Shift (most terminals) or Option (macOS), and copying
  commands by selecting them is common. So: on by default with a
  `--no-mouse` flag, or off by default with `--mouse`; decide when
  building it.

## From flk 1d428f4 (deploy batch) and 553b1a3

Done; checked end to end on a test app (deploy --no-rollback failed, the
attention fix rolled it back).
- [x] **Rollback:** app menu `r` (kept, healthy releases; `--no-rollback`
  as "failing checks: leave it"), also in deploy's form.
- [x] **Releases tab:** failed (and why), current, kept, image pruned.
- [x] **Health checks** and **config warnings** lead the Config tab
  (`config show --json`); warnings are in Needs attention via G11.
- [x] A machine left on a failed release: Needs attention, fix = roll back.
- [x] **Huge pages** on the machine Overview's size line.
- [x] After a failed `--no-rollback` deploy, roll back goes to the current
  (last good) release (flk 3fd7177); the form offers it first and the
  attention fix runs `releases rollback vN`. Checked end to end.

## From flk M1 (backups you can trust)

- [x] M1.1 (4ff9e9b): volume menu `v` verify (one backup or all) and `a`
  restore as new (nothing stops); verify results and `restored_from`
  shown. Checked live on a test app.
- [x] M1.2 (a04fcc6): volume menu `c` copy (`--dest`, else
  backup_copy_to); copies (ok, or why not) under each backup; the
  doctor's copy check shows on Health and in Needs attention.
- [x] M1.3 (4b26b65): Disk tab lists backups per volume (size, count);
  the doctor's space estimate is on Health (a warn when tight goes to
  Needs attention); back up's confirm says what happens when short.
- [x] 1bdb910: destroy app asks keep/delete backups (`--with-backups`);
  host menu `B` prune backups of destroyed apps (typed confirm).
- [x] M1.4 (3f618a4): roll back's "volumes: back to then"
  (`--with-volumes`), checked against the before-deploy backups like flk
  does; before-deploy backups labelled in Backups, and on the release
  that took them. Checked live: the volume's data went back with v1.
- [x] M1.5 (63db57f, fbedbe7): volume menu `m` move (`--dir` or
  `--path`, machine stopped). Checked live, out and back into the state
  dir.

## From flk M2 (files in volumes)

- [x] ef6f865: volume menu `f` browse files: directories, text files (up
  to 1M; binary and bigger ones are copied out), the live volume or any
  backup (`b`), copy out (`c`, to the dir lazyflk started in). Read-only
  on purpose: writes need the machine stopped and stay on the command
  line. Checked live on a test app.
- [x] Fixtures recaptured at flk c49d762 (help-all.json has files,
  events, exec).

## From flk M3

- [x] G7 events (c0a98f4): one `flk events --all --json --follow` per
  session; a live event reloads the main view at once (and drops that
  app's cached tabs), so polling is a 30s backstop; reconnects with
  `--since`, dropping repeats; Events tabs on the host, apps, machines.
  Checked live: a machine stopped from the CLI showed within 1s.
- [x] exec (25b23ad): machine menu `e` run a command (`flk exec -i ID
  -- sh -c LINE`, output and exit status in the job view) and `h` shell
  (`flk ssh console`, the terminal handed over like the console; `!` on
  the main view). The palette puts exec's command after `--` and treats
  ssh console without -C/--send and exec --tty as interactive. Checked
  live. "run it: as is" skips sh -c, for images without a shell (static
  init, 4f47d96).

## From flk 21ffec9 and 7cb3ddb (host settings from the CLI, 2026-10-15)

Note left by the flk side for whoever picks this up. All additive; nothing
existing changed shape. Recapture fixtures (help-all.json) first so the
palette and forms see the new command and options.
- [x] **`flk config set KEY=VALUE...`** (done 2026-11-11: host menu `c`
  settings, a field per key flk's help names, prefilled, only what
  changed is set; a key with several values, like two binds, is left to
  the file because config set writes one; tested on fixtures only, not
  run on the reference host) writes `~/.config/flk/host.toml`.
  Keys: backup_dir, backup_keep, address, bind, backup_copy_to,
  backup_copy_keep, backup_copy_warn_days (also listed in `flk help config
  set`). `*_keep` / `*_days` must be whole numbers; bind/address are
  validated before the file is replaced, so a bad value exits 1 with
  `Error: ...` and the file is unchanged. Success prints `key = value
  (path)` per key, then `Next: flk doctor`.
  Idea: a host menu "settings" form (one field per key, prefilled from
  `config show --host`), run as a job, then refresh doctor.
- [x] **`flk config show --host`** (prefills the form; not yet on the
  Health view) prints `{"file", "settings"}` (settings
  as parsed; empty `{}` when the file doesn't exist). Use it to prefill the
  form and to show host settings on the Health/host view.
- [x] **Doctor's backup_dir hints changed** (done: a step naming `flk
  config set KEY=` opens the settings form with KEY first): "no backup_dir set" now hints
  `flk config set backup_dir=/path/to/backups` (a placeholder path), and
  "backup_dir ... missing or not writable" hints `mount it, or: flk config
  set backup_dir=/new/path`. Today `actions.Fix` makes these copy-only
  (not in `runnable`'s allow-list), which is safe. Don't add them to the
  allow-list as is: the path is a placeholder. Better: Needs attention's
  fix opens the settings form with backup_dir focused.
- [ ] **`flk setup --fix`** (new option) applies the no-root fixes: the
  `~/.local/bin/flk` symlink and podman short-name registries. Without
  `--fix`, `flk setup` prompts only when stdin and stdout are both
  terminals, so a lazyflk job never hangs on it; pass `--fix` if lazyflk
  offers setup as an action.

## From flk 97f57a1 and after (R1 landed; M6 planned; updated 2026-11-08)

Note left by the flk side: what flk's PLAN.md R1 and M6 add that lazyflk
could use. R1 is on flk main since 7f0317f (2026-11-05), so its items
below are ready to integrate (recapture fixtures first). M6 isn't built
yet (flk's TODO.md has the specs). All additive.

R1 (on flk main, 7f0317f):
- [ ] **Build logs** (half done 2026-11-11: the path shows on the
  Releases tab, for the current and failed releases, and on a
  deploy.failed event; opening it waits for a flk command that prints
  it, asked in flk's TODO, cf4b96f). Every build's output goes in `apps/APP/builds/vN.log`
  (kept for the kept releases plus the last 3 failures). `deploy.failed`
  events get a `"log"` field with the path, and their `detail` carries the
  build's first error line. `flk releases --json` gets `build_log`. Idea:
  `l` on a failed deploy or release opens its log in the log viewer.
- [x] **Volume owner from the image's USER** (done 2026-11-11: volume
  menu `o` owner runs `flk files chown UID[:GID] VOL:/`, or
  `--recursive`, machine stopped; always offered, not only after a
  "can't write" failure. Checked live on a test app): new volumes take it, and a
  deploy says so. No new command: `flk files chown UID:GID VOL:/` (machine
  stopped) already repairs one. Idea: a volume menu entry for it, shown
  when the machine fails with "can't write /data".
- [x] **`flk deploy --no-backup`** (done 2026-11-11: deploy's form has
  "backup first: as configured / skip it"; not yet offered by itself
  after a failed verify): skips this deploy's before_deploy
  backup. A failed verify names it in its `Next:` lines; the deploy action
  could offer it on that error.
- [x] **`flk images prune [--dry-run]`** (done 2026-11-11: host menu `F`
  free space, "look first" is the default and runs --dry-run), and flk's images labelled
  `flk.app`/`flk.release`. Idea: a host-level "free space" action showing
  the dry run first. Note for the palette: in `help --all --json`,
  `images` now has subcommands (`list`, the default, and `prune`) like
  `releases`; `flk images` and `flk images --json` are unchanged.
- [ ] **flk-tmux.service restarts resume the VMs.** No new CLI. Events
  show the stop/resume pair; a restart now shows as a short blip.
- [x] **Pruned copies** (flk 56721dc; done, backups-pruned.json): in
  `volumes backups --json`, a `copies` entry whose file was pruned at
  its destination has `pruned_at`; don't show it as a copy that's there.
- [x] **New help topics** `dockerfile`, `host`, `bugs` (`flk help TOPIC`;
  they're in `help --all --json` "topics"). Recaptured `help-all.json`
  and the topics at flk 39e07f4.

M6 (not started; for flkforge, the new app wizard at ~/d.flkforge):
- [ ] **`flk build [--json]`** (built on flk branch m6-making-apps,
  2ca82f0/aba8002; not on main yet): build and inspect without deploying.
  JSON: ok, app, source, tag (flk.local/APP:build), build_log, seconds,
  image_bytes, rootfs_bytes, rootfs_used_bytes, init (shell|static),
  agent, cmd, cmd_from (flk.toml|image|default), workdir, env (names),
  user {name, uid, gid, root, switch: setpriv|agent|chroot|none|null},
  mounts [{source, destination, in_image, type, uid, gid, entries}],
  warnings [{code, message, fix}]. A failure with --json: {ok: false,
  error, said, step, build_log}, exit 1. Idea: a "check build" action in
  the app menu, its warnings in the Config tab.
- [ ] **Image warnings** (8947c87, same branch): deploy prints them and
  keeps them in each release (`releases --json` "warnings": [{code,
  message, fix}]); `config show` has "image_warnings" (the current
  release's; null before a deploy) and its "warnings" strings now include
  [[mounts]] sources that aren't volume names. `flk images --json` rows
  have "trial" (a flk build's image), which `images prune` removes.
- [ ] **New check types**: `exec`, `udp`, `a2s`, and a `[deploy]
  preflight` hook. Check results in deploy output and events will name
  them; show the type in the release/check views.
- [ ] **Health while running** (planned, from enshrouded.flk #3): checks
  on an interval, the last result per machine in `flk status --json`
  (`checks: {NAME: {ok, at, since}}`), `check.failed`/`check.recovered`
  events. Would give the tree a live health mark per machine.
- [ ] **`flk logs --boot`** (planned): the log since the latest boot,
  across rotation. The log viewer could open there.
- [ ] **flk.toml keys in `help --all --json`**: could drive config hints
  in a config view.
- [ ] **flkforge**: a separate tool (CLI and wizard) that writes new apps.
  Possible later: a "new app…" action that starts it. Talk to the user
  before building anything.

## Polish (2026-10-13)

- [x] `{input}` takes any one line (apostrophes too): passed to sh as $1.
- [x] 80 columns: the detail pane wraps long lines (under a field's
  value; paths cut, not split at hyphens); short header; footers drop
  whole hints and keep `? help` and `esc`; tree ports drop the host's
  own address when narrow; port lines name the address once.

## Sprint 1 after the split (2026-10-14)

- [x] `--version`; GitHub Actions (gofmt, vet, build, `go test -race`:
  the race detector can't run on the reference host, no C compiler);
  AGENTS.md: every non-`flkt-*` app is real (`opencode-sandbox` too),
  and flk gaps go to the flk agent.
- [x] Fixtures at flk 39febf8 (a third app, with no ports).
- [x] **Healthy folds** (SPEC 5.1): Storage, Podman and Services fold to
  a summary line while fine and open by themselves on a problem; healthy
  apps fold from the bottom when the tree doesn't fit; a fold by hand
  holds until the health changes.
- [x] **Edit, check, deploy:** `E` opens flk.toml in `$EDITOR`, then the
  Config tab (warnings first) and a nudge to `d`; app menu `A` edits
  `.lazyflk.toml`. Checked live on a test app.
- [x] Machine menu shell asks which user (`ssh console --user`).
- [x] **G13** (flk a6c6975): a failed deploy's status says "machines
  unchanged (still on vN)"; flk's Next: on stderr is styled; deploy.failed
  is a failure in Events. Checked live with a failing Dockerfile.
- [x] **G14** (flk a6c6975): new volume has an owner field when flk lists
  `volumes create --owner` (`flk.Help.HasOption`, so an older flk gets
  the old form); `owner_from_image` on the volume Overview.
- [x] App-level `flk start|stop|restart` (a6c6975): app menu s, S, R and
  `s` on an app; stop all moved to p. Checked live on a test app.
- [x] `--test-apps flkt-`: actions only on test apps (after a live check
  stopped mqtt-flk by a wrong selection; it was started again in a
  minute). Checked live: refused on minecraft-flk.
- [x] Internal names (flk M4 ad960ec, G16 0227eac): flk's `internal_names`
  on the machine Overview, APP.internal on the app's, `internal_hosts` on
  Config. Checked live on a two-machine test app.
- [x] Pins with hints: a default pin this host doesn't have is left out;
  `P` on a service edits `~/.config/lazyflk/pins` in the editor (created
  first with what's shown and a note on the format); each service and the
  section say where the pins come from. `snap.mosquitto.mosquitto`-style
  names get `.service` (they have dots but no type). Checked live with a
  scratch config dir.
- [x] Services: a socket-activated unit (ssh here) showed as off while
  idle; now `◌ ssh (listening)` when NAME.socket is active, and the
  section says "4 running, 1 listening" (2026-11-11).
- [ ] Maybe: `/` filters the tree, if a host outgrows folding.

## Stopped on purpose (2026-11-11)

- [x] An app whose machines the admin stopped (`desired_state` stopped)
  isn't a problem: it sorts last, folds to `▸ NAME  stopped`, and Needs
  attention has one line for all of them (`○ 3 apps stopped: ...`)
  instead of flk's "is stopped; start it" step per machine. Opened by
  hand it stays open; started again it's an app like the others. A
  machine whose restart policy gave up, or that should be running, is
  still a problem. Checked live (read-only) on the reference host.
  lazyflk tells flk's step apart by its wording plus the machine's
  state; asked flk for a `kind` per step (flk's TODO, "For lazyflk").
- [x] The attach snapshot no longer depends on the test running inside
  tmux (`make test` failed there).

## Fixtures on any day (2026-11-11)

- [x] The tests no longer need a quiet three-app host with minecraft-flk
  first: they find the test app by name (`testApp`, `goTo`, `mc`), check
  what must agree instead of the day's counts, and build the states the
  capture doesn't have (ssh idle on its socket). All fixtures recaptured
  at flk cf4b96f: seven apps, three stopped, one doctor warning, so
  `--demo` shows stopped apps folded at the bottom.
- [ ] `logs-minecraft-flk.txt` is the real server log: player names and
  client addresses. Fine in a private repo; before publishing, capture it
  from a test app or have capture.sh mask them.

## From flk's lazyflk-gaps branch (c4afa65, 40ae7f4, 1cb50a2; 2026-11-11)

What lazyflk asked flk for the same day. Each is used only where flk has
it, so lazyflk works the same with flk main until the branch is merged.
- [x] `flk releases log`: app menu `B` build log opens the newest build's
  output (failed or not) in the log viewer; a read, so it works on any app
  under --test-apps. Checked live on a test app with the branch's flk.
  Not yet: `l` straight from a failed deploy's event or job, or picking
  an older release's log.
- [x] `next_steps_detail`: a step of kind `stopped` is what stays out of
  Needs attention; the " is stopped;" wording is only the fallback for a
  flk without the field. The other kinds (setup, info, ok) aren't used yet.
- [x] `config set bind=A,B` and `config unset`: the settings form offers
  a bind with several addresses, and a field emptied is unset (set and
  unset are two commands: one at a time). Tested on fixtures only.

## Known gaps

- [x] **Reboots and log rotation** pin a banner in the followed log
  (`x` dismisses); checked live with a restart.
- [x] **Detaching from the console inside tmux** (`Ctrl-b Ctrl-b d`):
  checked through a real outer tmux client (2026-10-13): back in
  lazyflk, the outer tmux stays attached, the machine keeps running.
