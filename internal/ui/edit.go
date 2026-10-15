package ui

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/dayne/d.lazyflk/internal/flk"
)

// Editing an app's flk.toml (SPEC 4: in $EDITOR, not in the TUI), then
// seeing how flk reads it and deploying it: edit, check, deploy.

type editDoneMsg struct {
	j      *job
	err    error
	app    string
	file   string
	before [32]byte
}

// editor is the admin's editor as a command: $VISUAL, then $EDITOR (either
// may carry flags, "code -w"), then vi.
func editor() []string {
	for _, v := range []string{"VISUAL", "EDITOR"} {
		if f := strings.Fields(os.Getenv(v)); len(f) > 0 {
			return f
		}
	}
	return []string{"vi"}
}

// fileSum is what's in a file now; a missing one sums like an empty one.
func fileSum(path string) [32]byte {
	b, _ := os.ReadFile(path)
	return sha256.Sum256(b)
}

// openEdit offers to open an app's file in the editor; it hands over the
// terminal like the console, after saying so.
func (m *Model) openEdit(app, file string) tea.Cmd {
	if m.opt.ReadOnly {
		m.note = "read-only: editing is off"
		return nil
	}
	if file == "" {
		m.note = "flk doesn't know where " + app + "'s flk.toml is: run flk deploy once in its directory"
		return nil
	}
	m.attach = &attachView{title: "edit ─ " + app, app: app, file: file, editor: append(editor(), file)}
	return nil
}

// editKey is E on the main view: edit the selected app's flk.toml.
func (m *Model) editKey() tea.Cmd {
	n, ok := m.selected()
	if !ok || n.kind < kindApp || n.kind > kindPorts {
		return nil
	}
	a := m.apps[n.app]
	return m.openEdit(a.Name, a.ConfigPath)
}

// editAbout says what the editor is for, and what comes after.
func editAbout(at *attachView) []string {
	if filepath.Base(at.file) == ".lazyflk.toml" {
		return []string{"This opens the app's own actions in your editor ($VISUAL, $EDITOR, else vi); saving creates the file if it's new. Each one is a command line with placeholders: see the README's \"App actions\".",
			"", "When you close the editor, they're in the app's menu (enter)."}
	}
	return []string{"This opens the app's flk.toml in your editor ($VISUAL, $EDITOR, else vi). Nothing runs until you deploy.",
		"", "When you close the editor, Config shows how flk reads it (warnings first), and d deploys it."}
}

// runEdit hands the terminal to the editor; the edit goes in the command
// log like any action.
func (m *Model) runEdit(at *attachView) tea.Cmd {
	line := flk.CommandLineFor(at.editor[0], at.editor[1:]...)
	if m.opt.Demo {
		m.note = "demo mode: would run " + line
		return nil
	}
	m.jobSeq++
	j := &job{id: m.jobSeq, title: at.title, line: line, started: m.now()}
	m.addJob(j)
	cmd := exec.Command(at.editor[0], at.editor[1:]...)
	cmd.Dir = filepath.Dir(at.file)
	before := fileSum(at.file)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editDoneMsg{j: j, err: err, app: at.app, file: at.file, before: before}
	})
}

// editDone is back from the editor: if the file changed, show what flk
// makes of it.
func (m *Model) editDone(msg editDoneMsg) tea.Cmd {
	j, name := msg.j, filepath.Base(msg.file)
	j.lines = append(j.lines, logLine{text: "(it had the terminal)", kind: logText})
	switch {
	case msg.err != nil:
		j.err = msg.err
		m.note = "editor: " + exitText(msg.err)
	case fileSum(msg.file) == msg.before:
		j.lines = append(j.lines, logLine{text: name + " unchanged", kind: logText})
		m.note = name + " unchanged"
	case name == ".lazyflk.toml":
		j.lines = append(j.lines, logLine{text: name + " saved", kind: logText})
		m.note = name + " saved: its actions are in the app's menu (enter)"
	default:
		j.lines = append(j.lines, logLine{text: name + " saved", kind: logText})
		m.rebuild("app/" + msg.app)
		m.setTabNamed("Config")
		m.note = name + " saved: Config shows how flk reads it; d deploys it"
	}
	return m.jobEnded(j)
}
