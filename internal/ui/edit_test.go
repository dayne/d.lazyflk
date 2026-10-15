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
	withState(t, m, func(hc *flk.HostContext) { hc.Apps[0].Volumes[0].OwnerFromImage = true })
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
