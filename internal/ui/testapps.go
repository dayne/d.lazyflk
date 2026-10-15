package ui

import (
	"strings"

	"github.com/dayne/d.lazyflk/internal/actions"
)

// --test-apps PREFIX: actions run only on apps named PREFIX* (flkt-), so
// a live check can't touch a real app by a wrong selection. Reads, logs
// and copies still work everywhere. Enforced where things run (testGuard)
// and shown in the menus (testOnly).

func (m *Model) isTestApp(app string) bool {
	return m.opt.TestApps == "" || (app != "" && strings.HasPrefix(app, m.opt.TestApps))
}

func (m *Model) notTestApp(app string) string {
	if app == "" {
		return "--test-apps " + m.opt.TestApps + ": only commands on one test app run"
	}
	return "--test-apps " + m.opt.TestApps + ": " + app + " isn't a test app"
}

// testGuard is why a command may not run under --test-apps, or "".
func (m *Model) testGuard(c actions.Command) string {
	if m.opt.TestApps == "" {
		return ""
	}
	if app := actions.TargetApp(c); !m.isTestApp(app) {
		return m.notTestApp(app)
	}
	return ""
}

// testOnly greys out what would act on something other than a test app;
// reads (logs, record, health, files), copies and lazyflk's own files stay.
func (m *Model) testOnly(acts []actions.Action, app string) []actions.Action {
	if m.isTestApp(app) {
		return acts
	}
	out := make([]actions.Action, len(acts))
	for i, a := range acts {
		switch {
		case a.View == actions.ViewLogs, a.View == actions.ViewRecord, a.View == actions.ViewHealth,
			a.View == actions.ViewCopy, a.View == actions.ViewFiles:
		case a.View == actions.ViewEdit && app == "": // lazyflk's own files (pins), no app's
		default:
			if a.Disabled == "" {
				a.Disabled = m.notTestApp(app)
			}
		}
		out[i] = a
	}
	return out
}
