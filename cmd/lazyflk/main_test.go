package main

import (
	"runtime/debug"
	"testing"
)

func TestVersion(t *testing.T) {
	for _, c := range []struct {
		info *debug.BuildInfo
		want string
	}{
		{nil, "lazyflk (unknown version)"},
		{&debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "lazyflk devel"},
		{&debug.BuildInfo{Main: debug.Module{Version: "v1.2.0"}, Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "739fd88e0123456789"}}}, "lazyflk v1.2.0 (commit 739fd88)"},
		{&debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "739fd88e0123456789"}, {Key: "vcs.modified", Value: "true"}}}, "lazyflk devel (commit 739fd88, modified)"},
	} {
		if got := version(c.info); got != c.want {
			t.Errorf("version(%+v) = %q, want %q", c.info, got, c.want)
		}
	}
}
