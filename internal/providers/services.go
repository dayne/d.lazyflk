package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dayne/d.lazyflk/internal/actions"
)

// Services is flk's own systemd units, then the ones the admin pins
// (SPEC 10.3). State comes from `systemctl show`, logs from journalctl;
// both work as the user. Start, stop and restart need root, so they're
// copied, not run.
type Services struct {
	PinsFile string // one unit per line; "" or missing: DefaultPins
}

var flkUnits = []string{"flk-net.service", "flk-machines.service"}

// DefaultPins is what the reference host runs besides flk.
var DefaultPins = []string{"ssh.service", "snap.mosquitto.mosquitto.service", "chrony.service"}

const unitProps = "Id,Description,LoadState,ActiveState,SubState,UnitFileState,ActiveEnterTimestamp,StateChangeTimestamp,MainPID,Result,NRestarts"

// PinsPath is where pins live: ~/.config/lazyflk/pins.
func PinsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "lazyflk", "pins")
}

func (s Services) pins() ([]string, bool) {
	if s.PinsFile == "" {
		return DefaultPins, false
	}
	b, err := os.ReadFile(s.PinsFile)
	if err != nil {
		return DefaultPins, false
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if !hasUnitSuffix(l) { // snap.mosquitto.mosquitto has dots, but no type
			l += ".service"
		}
		out = append(out, l)
	}
	return out, true
}

// hasUnitSuffix reports a unit name that already says its type.
func hasUnitSuffix(u string) bool {
	for _, t := range []string{".service", ".socket", ".timer", ".mount", ".automount", ".path", ".target", ".slice", ".scope", ".swap", ".device"} {
		if strings.HasSuffix(u, t) {
			return true
		}
	}
	return false
}

func (Services) Name() string { return "Services" }
func (s Services) Command() string {
	return "systemctl show --timestamp=utc -p ... UNIT... · journalctl -u UNIT"
}

func (s Services) Load(ctx context.Context, h Host, env Env) ([]Item, error) {
	pins, fromFile := s.pins()
	units := append(append([]string(nil), flkUnits...), pins...)
	args := append([]string{"show", "--timestamp=utc", "-p", unitProps}, units...)
	out, err := run(ctx, h, "systemctl", args...)
	if err != nil {
		return nil, err
	}
	journal := ""
	if g, err := run(ctx, h, "id", "-Gn"); err == nil && !canReadJournal(string(g)) {
		journal = "journalctl only shows your own messages: add yourself to the systemd-journal (or adm) group: sudo usermod -aG systemd-journal $USER"
	}
	pinNote := s.Hint()
	var items, installed []Item
	var seen []string
	for i, props := range parseShow(string(out)) {
		unit := props["Id"]
		if unit == "" && i < len(units) {
			unit = units[i]
		}
		isFlk := i < len(flkUnits)
		if !isFlk && !fromFile && props["LoadState"] == "not-found" {
			continue // a default this host doesn't have: not news
		}
		it := unitItem(unit, props, isFlk, pinNote, journal)
		if !isFlk {
			seen = append(seen, unit)
		}
		installed = append(installed, it)
	}
	listening(ctx, h, installed)
	for _, it := range installed {
		if s.PinsFile != "" {
			it.Actions = append(it.Actions, actions.Action{Key: "P", Label: "edit pins",
				Help: "choose the services shown here: " + s.PinsFile + ", one unit per line", View: actions.ViewEdit,
				File: s.PinsFile, Seed: PinsSeed(seen)})
		}
		items = append(items, it)
	}
	return items, nil
}

// socketProps is what listening asks about a service's socket unit.
const socketProps = "Id,LoadState,ActiveState"

// listening marks the stopped services whose socket unit (NAME.socket) is
// active: systemd starts them on a connection (ssh on Ubuntu), so idle
// isn't off. One more systemctl call, only when something is stopped.
func listening(ctx context.Context, h Host, items []Item) {
	bySocket := map[string]int{}
	var sockets []string
	for i, it := range items {
		unit := strings.TrimPrefix(it.Key, "service/")
		if name, ok := strings.CutSuffix(unit, ".service"); ok && it.State == Off && !strings.HasSuffix(it.Label, "(not here)") {
			bySocket[name+".socket"] = i
			sockets = append(sockets, name+".socket")
		}
	}
	if len(sockets) == 0 {
		return
	}
	out, err := run(ctx, h, "systemctl", append([]string{"show", "-p", socketProps}, sockets...)...)
	if err != nil {
		return
	}
	for _, p := range parseShow(string(out)) {
		i, ok := bySocket[p["Id"]]
		if !ok || p["LoadState"] != "loaded" || p["ActiveState"] != "active" {
			continue
		}
		it := &items[i]
		it.Label, it.State = "◌ "+strings.TrimPrefix(it.Label, "○ ")+" (listening)", OK
		for j, l := range it.Lines {
			if strings.HasPrefix(l, field("state", "")) {
				it.Lines[j] = field("state", "listening: "+p["Id"]+" is active and starts it on a connection")
			}
		}
	}
}

// Hint says where the pinned services come from, and how to change them.
func (s Services) Hint() string {
	_, fromFile := s.pins()
	switch {
	case s.PinsFile == "":
		return "pinned by default"
	case fromFile:
		return "pinned in " + s.PinsFile + "; P (on a service) edits the list"
	}
	return "pinned by default: there's no " + s.PinsFile + " yet. P (on a service) creates it with these and opens it in your editor"
}

// PinsSeed is a new pins file: what it's for, then the units shown now.
func PinsSeed(units []string) string {
	var b strings.Builder
	b.WriteString("# lazyflk pins: the services shown under Services, besides flk's own.\n")
	b.WriteString("# One systemd unit per line (.service may be left off); # starts a comment.\n")
	b.WriteString("# Logs (l) need the systemd-journal or adm group; start, stop and restart\n")
	b.WriteString("# need root, so lazyflk copies them as sudo commands instead of running them.\n")
	for _, u := range units {
		b.WriteString(u + "\n")
	}
	return b.String()
}

func canReadJournal(groups string) bool {
	for _, g := range strings.Fields(groups) {
		if g == "systemd-journal" || g == "adm" || g == "wheel" || g == "root" {
			return true
		}
	}
	return false
}

// parseShow splits `systemctl show` output into one map per unit.
func parseShow(out string) []map[string]string {
	var res []map[string]string
	for _, block := range strings.Split(strings.TrimSpace(out), "\n\n") {
		m := map[string]string{}
		for _, l := range strings.Split(block, "\n") {
			if k, v, ok := strings.Cut(l, "="); ok {
				m[k] = v
			}
		}
		res = append(res, m)
	}
	return res
}

func unitItem(unit string, p map[string]string, isFlk bool, pinNote, journal string) Item {
	name := strings.TrimSuffix(unit, ".service")
	it := Item{Key: "service/" + unit, Title: "service " + unit}
	active, sub := p["ActiveState"], p["SubState"]
	switch {
	case p["LoadState"] == "not-found":
		it.Label, it.State = "○ "+name+" (not here)", Off
		it.Lines = []string{field("state", "not installed on this host")}
		if !isFlk {
			it.Lines = append(it.Lines, "", pinNote)
		} else {
			it.Lines = append(it.Lines, "", "flk's boot units: sudo setup-host.sh --install-service")
		}
		return it
	case active == "failed":
		it.Label, it.State = "✘ "+name, Bad
		if !isFlk { // flk's own failed units are doctor warnings already
			it.Problems = append(it.Problems, fmt.Sprintf("service %s failed: journalctl -u %s -b, then sudo systemctl restart %s", unit, unit, unit))
		}
	case active == "active":
		it.Label = "● " + name
	default:
		it.Label, it.State = "○ "+name, Off
	}
	state := active
	if sub != "" {
		state += " (" + sub + ")"
	}
	it.Lines = []string{
		field("unit", unit),
		field("what", p["Description"]),
		field("state", state),
		field("enabled", or(p["UnitFileState"], "-")),
	}
	if t := p["StateChangeTimestamp"]; t != "" {
		it.Lines = append(it.Lines, field("since", t))
	}
	if pid := p["MainPID"]; pid != "" && pid != "0" {
		it.Lines = append(it.Lines, field("pid", pid))
	}
	if n := p["NRestarts"]; n != "" && n != "0" {
		it.Lines = append(it.Lines, field("restarts", n))
	}
	if r := p["Result"]; r != "" && r != "success" {
		it.Lines = append(it.Lines, field("result", r))
	}
	if isFlk {
		it.Lines = append(it.Lines, "", "one of flk's units (oneshot: healthy is active (exited))")
	} else {
		it.Lines = append(it.Lines, "", pinNote)
	}
	if journal != "" {
		it.Lines = append(it.Lines, "", journal)
	}
	logs := actions.Command{Program: "journalctl", Args: []string{"-u", unit, "-f", "-n", "200", "--no-pager", "-o", "short-iso"}}
	it.Logs = &logs
	it.Actions = []actions.Action{
		actions.FollowLogs("l", "follow its journal (journalctl -u "+unit+" -f)", logs),
		actions.CopyOnly("r", "copy: sudo systemctl restart "+unit, "needs root, so lazyflk copies it", "sudo systemctl restart "+unit),
		actions.CopyOnly("S", "copy: sudo systemctl stop "+unit, "needs root, so lazyflk copies it", "sudo systemctl stop "+unit),
		actions.CopyOnly("s", "copy: sudo systemctl start "+unit, "needs root, so lazyflk copies it", "sudo systemctl start "+unit),
	}
	return it
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Summary counts the units running, the ones waiting on their socket, and
// the others (stopped, not here).
func (Services) Summary(items []Item) string {
	var running, waiting, other int
	for _, it := range items {
		switch {
		case strings.HasSuffix(it.Label, "(listening)"):
			waiting++
		case it.State == OK:
			running++
		default:
			other++
		}
	}
	s := fmt.Sprintf("%d running", running)
	if waiting > 0 {
		s += fmt.Sprintf(", %d listening", waiting)
	}
	if other > 0 {
		s += fmt.Sprintf(", %d off", other)
	}
	return s
}
