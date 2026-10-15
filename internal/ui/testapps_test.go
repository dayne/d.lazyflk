package ui

import (
	"strings"
	"testing"

	"github.com/dayne/d.lazyflk/internal/actions"
	"github.com/dayne/d.lazyflk/internal/flk"
)

// --test-apps flkt-: a real app's actions are greyed out and refused
// wherever they'd run; reads still work; a test app's actions run.
func TestTestApps(t *testing.T) {
	m := demoModel(t, 120, 40)
	m.opt.TestApps = "flkt-"
	goToKey(t, m, "app/minecraft-flk")
	if !strings.Contains(screen(m), "test apps: flkt-*") {
		t.Error("the header should say actions are limited")
	}

	press(m, "enter")
	acts, _ := m.menuActions()
	for _, a := range acts {
		switch a.Label {
		case "deploy", "stop", "edit flk.toml", "destroy app", "who":
			if !strings.Contains(a.Disabled, "minecraft-flk isn't a test app") {
				t.Errorf("%s: %q", a.Label, a.Disabled)
			}
		case "logs":
			if a.Disabled != "" {
				t.Errorf("logs are a read: %q", a.Disabled)
			}
		}
	}
	press(m, "esc")

	for _, k := range []string{"s", "d", "E"} { // the shortcuts end at the same checks
		m.note = ""
		press(m, k)
		if m.confirm != nil || m.form != nil || m.attach != nil {
			press(m, "enter", "y", "enter")
		}
		if len(m.jobs) != 0 || m.attach != nil {
			t.Fatalf("%s ran something on a real app: %v", k, actionCalls(m))
		}
		press(m, "esc")
	}

	// the places things run refuse on their own: host-wide, other tools, the palette's -a
	for _, c := range []actions.Command{
		{Args: []string{"stop-all"}},
		{Args: []string{"-a", "mqtt-flk", "stop"}},
		{Args: []string{"apps", "destroy", "minecraft-flk", "--yes"}},
		{Program: "podman", Args: []string{"rm", "x"}},
	} {
		m.note = ""
		if m.runJob("t", c) != nil || !strings.HasPrefix(m.note, "--test-apps flkt-: ") {
			t.Errorf("%s ran (note %q)", c.Line(), m.note)
		}
	}
	if len(m.jobs) != 0 {
		t.Fatalf("jobs: %v", actionCalls(m))
	}

	// a test app's actions go ahead
	withState(t, m, func(hc *flk.HostContext) { hc.Apps[0].Name = "flkt-check" })
	goToKey(t, m, "app/flkt-check")
	press(m, "s")
	if m.confirm == nil || !strings.Contains(screen(m), "$ flk -a flkt-check stop") {
		t.Errorf("s on a test app:\n%s", screen(m))
	}
	press(m, "y")
	if len(m.jobs) != 1 || m.jobs[0].line != "flk -a flkt-check stop" {
		t.Errorf("jobs: %v", actionCalls(m))
	}
}
