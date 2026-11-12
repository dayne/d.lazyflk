package host

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/dayne/d.lazyflk/fixtures"
)

func TestParseDFFixture(t *testing.T) {
	out, err := fs.ReadFile(fixtures.FS, "df.txt")
	if err != nil {
		t.Fatal(err)
	}
	// df was run over flk's mount_dirs of that day: the same list, from the same capture
	var hc struct {
		MountDirs []string `json:"mount_dirs"`
	}
	raw, _ := fs.ReadFile(fixtures.FS, "context-all.json")
	if err := json.Unmarshal(raw, &hc); err != nil || len(hc.MountDirs) == 0 {
		t.Fatalf("mount dirs: %v %v", hc.MountDirs, err)
	}
	disks, err := ParseDF(string(out), hc.MountDirs)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, d := range disks {
		if d.Target == "" || d.SizeKB == 0 || d.Percent < 0 || d.Percent > 100 || len(d.Paths) == 0 {
			t.Errorf("disk: %+v", d)
		}
		for _, p := range d.Paths { // each dir is on the filesystem df named for it
			if d.Target != "/" && !strings.HasPrefix(p, d.Target) {
				t.Errorf("%s isn't under %s", p, d.Target)
			}
			seen++
		}
	}
	if disks[0].Target != "/" || !reflect.DeepEqual(disks[0].Paths[:1], hc.MountDirs[:1]) || seen != len(hc.MountDirs) {
		t.Errorf("%+v", disks)
	}
}

func TestParseDFGroupsAndSpaces(t *testing.T) {
	out := "Mounted on 1K-blocks Used Avail Use%\n/ 100 90 10 90%\n/mnt/my disk 2048 1024 1024 50%\n/ 100 90 10 90%\n"
	disks, err := ParseDF(out, []string{"/a", "/mnt/my disk/x", "/b"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Disk{
		{Target: "/", SizeKB: 100, UsedKB: 90, AvailKB: 10, Percent: 90, Paths: []string{"/a", "/b"}},
		{Target: "/mnt/my disk", SizeKB: 2048, UsedKB: 1024, AvailKB: 1024, Percent: 50, Paths: []string{"/mnt/my disk/x"}},
	}
	if !reflect.DeepEqual(disks, want) {
		t.Errorf("got %+v", disks)
	}
	if _, err := ParseDF("Mounted on\n/ x y z 1%\n", nil); err == nil {
		t.Error("bad numbers should fail")
	}
}

func TestHuman(t *testing.T) {
	for kb, want := range map[int64]string{512: "512K", 2048: "2.0M", 102626232: "98G", 33216356: "32G", 5 << 30: "5.0T"} {
		if got := Human(kb); got != want {
			t.Errorf("Human(%d) = %s, want %s", kb, got, want)
		}
	}
}
