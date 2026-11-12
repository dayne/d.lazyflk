package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/dayne/d.lazyflk/fixtures"
	"github.com/dayne/d.lazyflk/internal/flk"
	"github.com/dayne/d.lazyflk/internal/providers"
)

func TestEditor(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if got := editor(); len(got) != 1 || got[0] != "vi" {
		t.Errorf("no editor set: %v", got)
	}
	t.Setenv("EDITOR", "code -w")
	if got := strings.Join(editor(), " "); got != "code -w" {
		t.Errorf("EDITOR: %v", got)
	}
	t.Setenv("VISUAL", "nano")
	if got := strings.Join(editor(), " "); got != "nano" {
		t.Errorf("VISUAL first: %v", got)
	}
}

// E (or the app menu) offers the editor on flk.toml, showing the command
// first; demo mode only says what it would run.
func TestEditOffer(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nano")
	m := demoModel(t, 100, 30)
	press(m, "E")
	s := screen(m)
	if m.attach == nil || !strings.Contains(s, "$ nano /home/dayne/minecraft.flk/flk.toml") ||
		!strings.Contains(s, "Config shows how flk reads it") || !strings.Contains(s, "enter edit") {
		t.Fatalf("edit box:\n%s", s)
	}
	press(m, "enter")
	if m.attach != nil || !strings.Contains(m.note, "demo mode: would run nano /home/dayne/minecraft.flk/flk.toml") || len(m.jobs) != 0 {
		t.Errorf("demo: note %q, %d jobs", m.note, len(m.jobs))
	}
	press(m, "enter", "A") // the app's own actions, from its menu
	if m.attach == nil || !strings.Contains(screen(m), "$ nano /home/dayne/minecraft.flk/.lazyflk.toml") {
		t.Errorf("app actions:\n%s", screen(m))
	}

	ro := New(Options{Client: flk.New(flk.NewFake(fixtures.FS)), Demo: true, ReadOnly: true})
	send(ro, tea.WindowSizeMsg{Width: 100, Height: 30})
	send(ro, runCmd(ro.Init())...)
	press(ro, "E")
	if ro.attach != nil || !strings.Contains(ro.note, "read-only") {
		t.Errorf("read-only: %q", ro.note)
	}
}

// Back from the editor: a changed flk.toml goes to Config with a nudge to
// deploy; an unchanged one says so; either way it's in the command log.
func TestEditDone(t *testing.T) {
	m := demoModel(t, 100, 30)
	file := filepath.Join(t.TempDir(), "flk.toml")
	os.WriteFile(file, []byte("app = \"minecraft-flk\"\n"), 0o644)
	before := fileSum(file)

	j := &job{id: 1, title: "edit ─ minecraft-flk", line: "vi " + file}
	m.addJob(j)
	send(m, editDoneMsg{j: j, app: "minecraft-flk", file: file, before: before})
	if m.note != "flk.toml unchanged" || j.running() {
		t.Errorf("unchanged: %q", m.note)
	}

	goToKey(t, m, "app/vintagestory-flk")
	os.WriteFile(file, []byte("app = \"minecraft-flk\"\nkill_timeout = 30\n"), 0o644)
	j2 := &job{id: 2, title: "edit ─ minecraft-flk", line: "vi " + file}
	m.addJob(j2)
	send(m, editDoneMsg{j: j2, app: "minecraft-flk", file: file, before: before})
	n, _ := m.selected()
	if n.key(m.apps) != "app/minecraft-flk" || m.tabName(n) != "Config" || !strings.Contains(m.note, "d deploys it") {
		t.Errorf("changed: on %s tab %s, note %q", n.key(m.apps), m.tabName(n), m.note)
	}

	j3 := &job{id: 3, title: "edit ─ minecraft-flk", line: "vi " + file}
	m.addJob(j3)
	send(m, editDoneMsg{j: j3, err: errors.New("exit status 1"), app: "minecraft-flk", file: file, before: before})
	if !strings.HasPrefix(m.note, "editor: ") || j3.err == nil {
		t.Errorf("failed editor: %q", m.note)
	}
}

// A volume whose root takes the image's owner at first use says so (G14).
func TestVolumeOwnerFromImage(t *testing.T) {
	m := demoModel(t, 120, 40)
	withState(t, m, func(hc *flk.HostContext) { hc.Apps[appIndex(hc.Apps)].Volumes[0].OwnerFromImage = true })
	goTo(m, kindVolume)
	if s := screen(m); !strings.Contains(s, "owner     not set yet: its root takes the image's owner") {
		t.Errorf("volume overview:\n%s", s)
	}
}

// A deploy whose build failed: flk's cause, its Error: and Next: lines
// (G13), and the status says the machines weren't touched.
func TestFailedDeployUnchanged(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {160, 48}} {
		m := demoModel(t, sz[0], sz[1])
		start := time.Date(2026, 10, 14, 21, 0, 0, 0, time.UTC)
		m.now = func() time.Time { return start.Add(12 * time.Second) }
		j := &job{id: 1, title: "deploy ─ minecraft-flk", line: "flk -a minecraft-flk deploy", started: start, ended: start.Add(12 * time.Second),
			err: errors.New("exit status 1")}
		for _, l := range []flk.Line{
			{Text: "==> Building image flk.local/minecraft-flk:v10"},
			{Text: "STEP 3/7: RUN groupadd -g 1000 mc", Stderr: true},
			{Text: "groupadd: GID '1000' already exists", Stderr: true},
			{Text: "Error: build failed; machines unchanged (still on v9)", Stderr: true},
			{Text: "Next: flk deploy (after fixing the build)   or: flk status", Stderr: true},
		} {
			j.lines = append(j.lines, classifyJobLine(l))
		}
		m.addJob(j)
		m.shown = j
		if got := ansi.Strip(j.status(m.now())); got != "✘ failed after 12s · machines unchanged (still on v9)" {
			t.Errorf("status: %q", got)
		}
		if k := j.lines[4].kind; k != logHeader {
			t.Errorf("Next: on stderr is %v, want a header", k)
		}
		snap(t, m, "job-deploy-unchanged")
	}
	if u := unchanged(&job{lines: []logLine{{text: "Error: v10 failed its health checks", kind: logError}}}); u != "" {
		t.Errorf("a roll that touched machines isn't unchanged: %q", u)
	}
}

// s on an app: start what's stopped, else stop them all (flk stop).
func TestAppStartStopKey(t *testing.T) {
	m := demoModel(t, 100, 30)
	goToKey(t, m, "app/minecraft-flk")
	press(m, "s")
	if m.confirm == nil || !strings.Contains(screen(m), "$ flk -a minecraft-flk stop") {
		t.Errorf("s on a running app:\n%s", screen(m))
	}
}

// Internal names (flk M4, G16): flk's list on the machine, APP.internal on
// the app, the setting on Config; off says how to turn it on.
func TestInternalNames(t *testing.T) {
	m := demoModel(t, 120, 40)
	goTo(m, kindMachine)
	if s := screen(m); !strings.Contains(s, "names     none: [flk] internal_hosts = true") {
		t.Errorf("off:\n%s", s)
	}
	withState(t, m, func(hc *flk.HostContext) {
		mc := &hc.Apps[appIndex(hc.Apps)].Machines[0]
		mc.InternalNames = []string{"minecraft-flk.internal", mc.ID + ".vm.minecraft-flk.internal", mc.Name + ".vm.minecraft-flk.internal"}
	})
	goTo(m, kindMachine)
	s := screen(m)
	if !strings.Contains(s, "names     minecraft-flk.internal") || !strings.Contains(s, "mellow-aurora-7882.vm.minecraft-flk.internal") {
		t.Errorf("on:\n%s", s)
	}
	goToKey(t, m, "app/minecraft-flk")
	if s := screen(m); !strings.Contains(s, "internal minecraft-flk.internal (other machines reach it so; answered by 1") {
		t.Errorf("app:\n%s", s)
	}
	m.setTabNamed("Config")
	send(m, runCmd(m.sync())...)
	if s := screen(m); !strings.Contains(s, "internal  off: [flk] internal_hosts = true") {
		t.Errorf("config:\n%s", s)
	}
}

// An event whose detail carries rsync's lines stays one entry (they used
// to break the Events tab's layout).
func TestEventDetailOneLine(t *testing.T) {
	m := demoModel(t, 100, 30)
	e := flk.Event{At: "2026-10-15T05:13:38.580Z", App: "flkt-x", Type: "backup.copy_failed",
		Detail: "data: copy to nohost:/srv/flk failed: ssh: No route to host\nrsync: connection unexpectedly closed (0 bytes received so far) [Receiver]\r\nrsync error: unexplained error (code 255) at io.c(232) [Receiver=3.4.1]\n"}
	s := ansi.Strip(m.eventLine(e, true))
	if strings.ContainsAny(s, "\n\r") || !strings.Contains(s, "No route to host · rsync: connection unexpectedly closed") ||
		!strings.HasSuffix(s, "[Receiver] · rsync error: unexplained error (code 255) at io.c(232) [Receiver=3.4.1]") {
		t.Errorf("%q", s)
	}
}

// P on a service edits lazyflk's pins; a missing file is created first,
// with the services shown now and a note on the format.
func TestEditPins(t *testing.T) {
	m := demoModel(t, 120, 40)
	file := filepath.Join(t.TempDir(), "lazyflk", "pins")
	m.opt.Providers[2] = providers.Services{PinsFile: file}
	m.opt.TestApps = "flkt-" // pins are no app's: still editable
	delete(m.cache, m.opt.Providers[2].Command())
	send(m, runCmd(m.sync())...)
	goToKey(t, m, "prov/service/ssh.service")
	if s := screen(m); !strings.Contains(s, "pinned by default: there's no") || !strings.Contains(s, "P (on a service) creates it") {
		t.Errorf("service hint:\n%s", s)
	}
	press(m, "enter", "P")
	s := screen(m)
	if m.attach == nil || !strings.Contains(s, "lazyflk's pins") || !strings.Contains(s, "exist yet") {
		t.Fatalf("edit pins box:\n%s", s)
	}
	at := m.attach
	m.opt.Demo = false
	if cmd := m.runEdit(at); cmd == nil { // the editor would run now; don't
		t.Fatalf("not run: %q", m.note)
	}
	b, err := os.ReadFile(file)
	if err != nil || !strings.HasPrefix(string(b), "# lazyflk pins") || !strings.Contains(string(b), "\nssh.service\n") {
		t.Errorf("seeded %q, %v", b, err)
	}
	j := m.jobs[len(m.jobs)-1]
	send(m, editDoneMsg{j: j, file: file, before: fileSum(file), created: true})
	if !strings.HasPrefix(m.note, "created "+file) {
		t.Errorf("note %q", m.note)
	}
	goToKey(t, m, "section/Services")
	if s := screen(m); !strings.Contains(s, "pinned in ") {
		t.Errorf("section hint:\n%s", s)
	}
}
