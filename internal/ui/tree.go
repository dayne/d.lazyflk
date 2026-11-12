package ui

import (
	"fmt"
	"github.com/dayne/d.lazyflk/internal/actions"
	"sort"
	"strings"

	"github.com/dayne/d.lazyflk/internal/flk"
)

type nodeKind int

const (
	kindHeading nodeKind = iota // "Apps", "Host": not selectable
	kindInfo                    // a hint line: not selectable
	kindApp
	kindMachine
	kindVolume
	kindPorts
	kindHost
	kindProvider // an item of a provider's section (Storage, Podman, Services)
	kindSection  // a provider's heading: folds, and says how it is when folded
)

// node is one line of the left tree. app, machine and volume index into
// the model's sorted apps.
type node struct {
	kind    nodeKind
	text    string
	app     int
	machine int
	volume  int
	prov    int // kindProvider: the provider, and its item
	item    int
	pkey    string // kindProvider: the item's stable key
	folded  bool   // kindApp, kindSection: its lines are hidden
}

func (n node) selectable() bool { return n.kind >= kindApp }

// key identifies a node across refreshes so the selection stays put.
func (n node) key(apps []flk.App) string {
	switch n.kind {
	case kindApp:
		return "app/" + apps[n.app].Name
	case kindMachine:
		return "machine/" + apps[n.app].Name + "/" + apps[n.app].Machines[n.machine].ID
	case kindVolume:
		return "volume/" + apps[n.app].Name + "/" + apps[n.app].Volumes[n.volume].ID
	case kindPorts:
		return "ports/" + apps[n.app].Name
	case kindHost:
		return "host"
	case kindProvider:
		return "prov/" + n.pkey
	case kindSection:
		return "section/" + n.text
	}
	return ""
}

// appProblems lists what's wrong with an app, from its own state and the
// next_steps flk reports for it ("APP: ...").
func appProblems(a flk.App, steps []string) []string {
	var out []string
	for _, m := range a.Machines {
		if m.ShouldRun() {
			out = append(out, fmt.Sprintf("machine %s should be running but is %s", m.ID, m.State))
		}
	}
	for _, p := range a.Ports.Blocked {
		out = append(out, fmt.Sprintf("port %s blocked: %s", p.Shown(), p.Blocked))
	}
	for _, s := range steps {
		if rest, ok := strings.CutPrefix(s, a.Name+": "); ok && !parkedStep(a, rest) {
			out = append(out, rest)
		}
	}
	return out
}

// parkedStep is whether one of an app's next steps (without its "APP: ")
// only says that a machine the admin stopped is stopped ("machine ID
// [NAME] is stopped; start it: ..."). flk lists those; here they aren't
// problems. Any other wording stays one.
func parkedStep(a flk.App, rest string) bool {
	if !strings.Contains(rest, " is stopped;") {
		return false
	}
	for _, m := range a.Machines {
		if m.Parked() && strings.HasPrefix(rest, "machine "+m.ID+" ") {
			return true
		}
	}
	return false
}

// isParkedStep is parkedStep for a step as flk lists it ("APP: ...").
func isParkedStep(apps []flk.App, step string) bool {
	for _, a := range apps {
		if rest, ok := strings.CutPrefix(step, a.Name+": "); ok && parkedStep(a, rest) {
			return true
		}
	}
	return false
}

// appParked is an app stopped on purpose: it has machines, and the admin
// stopped every one.
func appParked(a flk.App) bool {
	for _, m := range a.Machines {
		if !m.Parked() {
			return false
		}
	}
	return len(a.Machines) > 0
}

// stoppedSummary is one line for everything stopped on purpose: "3 apps
// stopped: a, b, c", and machines stopped in apps that still run. Empty
// when nothing is.
func stoppedSummary(apps []flk.App) string {
	var names []string
	machines := 0
	for _, a := range apps {
		if appParked(a) {
			names = append(names, a.Name)
			continue
		}
		for _, m := range a.Machines {
			if m.Parked() {
				machines++
			}
		}
	}
	var parts []string
	switch len(names) {
	case 0:
	case 1:
		parts = append(parts, "1 app stopped: "+names[0])
	default:
		parts = append(parts, fmt.Sprintf("%d apps stopped: %s", len(names), strings.Join(names, ", ")))
	}
	switch {
	case machines == 1:
		parts = append(parts, "1 machine stopped in a running app")
	case machines > 1:
		parts = append(parts, fmt.Sprintf("%d machines stopped in running apps", machines))
	}
	return strings.Join(parts, " · ")
}

// sortApps orders apps with problems first, then the others by name, then
// the ones stopped on purpose (by name too).
func sortApps(apps []flk.App, steps []string) []flk.App {
	out := append([]flk.App(nil), apps...)
	rank := func(a flk.App) int {
		switch {
		case len(appProblems(a, steps)) > 0:
			return 0
		case appParked(a):
			return 2
		}
		return 1
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := rank(out[i]), rank(out[j]); ri != rj {
			return ri < rj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// portKeys is an app's ports for its tree line, grouped by the address
// they're bound to: "tcp/25565 udp/19132", or "10.47.11.3: tcp/25565
// udp/19132". Waiting ones end in ?, blocked ones in !.
func portKeys(t flk.PortTable, binds bool) []string {
	var order []string
	byBind := map[string][]string{}
	add := func(p flk.Port, mark string) {
		b := ""
		if p.Bind != nil {
			b = *p.Bind
		}
		if _, ok := byBind[b]; !ok {
			order = append(order, b)
		}
		byBind[b] = append(byBind[b], p.Key+mark)
	}
	for _, p := range t.Active {
		add(p, "")
	}
	for _, p := range t.Waiting {
		add(p, "?")
	}
	for _, p := range t.Blocked {
		add(p, "!")
	}
	var out []string
	for _, b := range order {
		keys := byBind[b]
		if b != "" && binds {
			keys[0] = b + ": " + keys[0]
		}
		out = append(out, keys...)
	}
	return out
}

func startedCount(a flk.App) int {
	n := 0
	for _, m := range a.Machines {
		if m.Started() {
			n++
		}
	}
	return n
}

// buildTree lays out apps (each expanded unless folded says otherwise) and
// the host.
func buildTree(hc *flk.HostContext, apps []flk.App, folded func(a flk.App) bool, extra ...node) []node {
	nodes := []node{{kind: kindHeading, text: "Apps"}}
	if len(apps) == 0 {
		nodes = append(nodes, node{kind: kindInfo, text: "no apps: flk help first-app"})
	}
	for i, a := range apps {
		f := folded(a)
		nodes = append(nodes, node{kind: kindApp, app: i, folded: f})
		if f {
			continue
		}
		for j := range a.Machines {
			nodes = append(nodes, node{kind: kindMachine, app: i, machine: j})
		}
		for j := range a.Volumes {
			nodes = append(nodes, node{kind: kindVolume, app: i, volume: j})
		}
		if len(portKeys(a.Ports, true)) > 0 {
			nodes = append(nodes, node{kind: kindPorts, app: i})
		}
	}
	nodes = append(nodes, node{kind: kindInfo}, node{kind: kindHeading, text: "Host"}, node{kind: kindHost})
	return append(nodes, extra...)
}

// treeLine renders one node, without the selection highlight.
func (m *Model) treeLine(n node) string {
	switch n.kind {
	case kindHeading:
		return styleTitle.Render(n.text)
	case kindInfo:
		return styleDim.Render("  " + n.text)
	case kindProvider:
		return m.providerLine(n)
	case kindSection:
		return m.sectionLine(n)
	case kindHost:
		if m.hc.Host.OK {
			return "  health " + styleOK.Render("✔")
		}
		return "  health " + styleBad.Render(fmt.Sprintf("✘ %d", len(m.hc.Host.Problems)))
	}
	a := m.apps[n.app]
	switch n.kind {
	case kindApp:
		arrow := foldArrow(n.folded)
		started := startedCount(a)
		icon := styleOK.Render("●")
		if started < len(a.Machines) || len(a.Machines) == 0 {
			icon = styleDim.Render("○")
		}
		s := fmt.Sprintf("%s %s  %d/%d %s", arrow, a.Name, started, len(a.Machines), icon)
		switch {
		case len(appProblems(a, m.hc.NextSteps)) > 0:
			s += " " + styleBad.Render("!")
		case appParked(a):
			s = styleDim.Render(fmt.Sprintf("%s %s  stopped", arrow, a.Name))
		}
		return s
	case kindMachine:
		mc := a.Machines[n.machine]
		return "  " + machineIcon(mc) + " " + or(mc.Name, mc.ID)
	case kindVolume:
		v := a.Volumes[n.volume]
		s := fmt.Sprintf("  ▢ %s %dGB", v.Name, v.SizeGB)
		if actions.External(v, m.hc.Flk.StateDir) {
			s += styleDim.Render(" (external)")
		}
		return s
	case kindPorts:
		return "  ⇄ " + strings.Join(portKeys(a.Ports, true), " ")
	}
	return ""
}

func foldArrow(folded bool) string {
	if folded {
		return "▸"
	}
	return "▾"
}

func machineIcon(mc flk.Machine) string {
	switch {
	case mc.ShouldRun():
		return styleBad.Render("!")
	case mc.Started():
		return styleOK.Render("●")
	}
	return styleDim.Render("○")
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
