// lazyflk is a terminal UI for hosts running flk. See SPEC.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime/debug"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/dayne/d.lazyflk/fixtures"
	"github.com/dayne/d.lazyflk/internal/flk"
	"github.com/dayne/d.lazyflk/internal/host"
	"github.com/dayne/d.lazyflk/internal/providers"
	"github.com/dayne/d.lazyflk/internal/ui"
)

func main() {
	fs := flag.NewFlagSet("lazyflk", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: lazyflk [--demo] [--read-only] [--no-mouse] [--refresh 5s] [--flk PATH] [--version]")
		fs.PrintDefaults()
	}
	demo := fs.Bool("demo", false, "show captured output from fixtures/ instead of running flk")
	readOnly := fs.Bool("read-only", false, "hide all actions")
	noMouse := fs.Bool("no-mouse", false, "don't use the mouse (leaves click-and-drag selection to the terminal)")
	refresh := fs.Duration("refresh", 5*time.Second, "how often to reload flk context --all --json (0: load once)")
	flkPath := fs.String("flk", "", "flk binary (default: $LAZYFLK_FLK, then PATH)")
	showVersion := fs.Bool("version", false, "print lazyflk's version and exit")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unexpected argument %q\n", fs.Arg(0))
		os.Exit(2)
	}
	if *showVersion {
		info, _ := debug.ReadBuildInfo()
		fmt.Println(version(info))
		return
	}

	var runner flk.Runner
	disk := host.DF
	hostRunner := realHost
	provs := []providers.Provider{providers.Storage{}, providers.Podman{}, providers.Services{PinsFile: providers.PinsPath()}}
	var appActions func(app, configPath string) ([]providers.AppAction, error) // nil: the real .lazyflk.toml
	if *demo {
		runner = flk.NewFake(fixtures.FS)
		disk = demoDisk
		hostRunner = (&providers.Fake{FS: fixtures.FS, Files: providers.DemoFiles}).For
		provs[2] = providers.Services{} // the default pins, as captured
		appActions = providers.DemoAppActions(fixtures.FS)
	} else {
		path, err := flk.FindFlk(*flkPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		runner = flk.Exec{Path: path}
	}

	m := ui.New(ui.Options{Client: flk.New(runner), Disk: disk, Demo: *demo, Host: hostRunner, Providers: provs, ReadOnly: *readOnly, NoMouse: *noMouse, Refresh: *refresh, AppActions: appActions})
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// version says which lazyflk this is: the module version when installed
// with go install NAME@VERSION, and the git commit it was built from.
func version(info *debug.BuildInfo) string {
	if info == nil {
		return "lazyflk (unknown version)"
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		v = "devel"
	}
	var rev, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				modified = ", modified"
			}
		}
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if rev == "" {
		return "lazyflk " + v
	}
	return "lazyflk " + v + " (commit " + rev + modified + ")"
}

// realHost runs host tools for the providers; a missing one fails when
// used, with the reason, instead of at start.
func realHost(prog string) flk.Runner {
	path, err := exec.LookPath(prog)
	if err != nil {
		return missing{prog}
	}
	return flk.Exec{Path: path}
}

type missing struct{ prog string }

func (m missing) err(args []string) error {
	return &flk.Error{Args: args, Stderr: "Error: " + m.prog + " isn't installed (or isn't on PATH)"}
}

func (m missing) Run(_ context.Context, args ...string) ([]byte, error) { return nil, m.err(args) }
func (m missing) Stream(_ context.Context, _ string, args ...string) (*flk.Stream, error) {
	return nil, m.err(args)
}

// demoDisk answers df from the captured fixtures/df.txt.
func demoDisk(_ context.Context, paths []string) ([]host.Disk, error) {
	out, err := fs.ReadFile(fixtures.FS, "df.txt")
	if err != nil {
		return nil, err
	}
	return host.ParseDF(string(out), paths)
}
