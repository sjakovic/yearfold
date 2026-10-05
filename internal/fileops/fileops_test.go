package fileops

import (
	"path/filepath"
	"testing"

	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/testutil"
)

type fixture struct {
	t    *testing.T
	root string
	st   *store.Store
	ops  *Ops
}

func setup(t *testing.T, files ...string) *fixture {
	t.Helper()
	root := t.TempDir()
	st, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	for _, f := range files {
		testutil.WriteFile(t, filepath.Join(root, f), f)
	}
	ch, err := scanner.Diff(root, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := scanner.Apply(st, ch, nil); err != nil {
		t.Fatal(err)
	}
	ops := &Ops{Root: root, TrashDir: filepath.Join(root, scanner.MetaDir, "trash"), St: st}
	return &fixture{t: t, root: root, st: st, ops: ops}
}

func (f *fixture) id(rel string) int64 {
	f.t.Helper()
	dir, name, _ := store.SplitPath(rel)
	id, found, err := f.st.PresentID(dir, name)
	if err != nil || !found {
		f.t.Fatalf("%s is not in the index (%v)", rel, err)
	}
	return id
}

func (f *fixture) onDisk(rel string) bool {
	return testutil.Exists(filepath.Join(f.root, rel))
}

func TestCleanDir(t *testing.T) {
	valid := map[string]string{
		"":              "",
		"2005":          "2005",
		" 2005/Trip/ ":  "2005/Trip",
		`2005\Trip`:     "2005/Trip",
		"/2005//Trip":   "2005/Trip",
		"a/./b":         "a/b",
		".yearfoldish":  ".yearfoldish",
		"x/.yearfold/y": "x/.yearfold/y",
	}
	for in, want := range valid {
		if got, err := CleanDir(in); err != nil || got != want {
			t.Errorf("CleanDir(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"..", "../x", "a/../../b", "a/..", ".yearfold", ".yearfold/trash"} {
		if got, err := CleanDir(in); err == nil {
			t.Errorf("CleanDir(%q) = %q, want an error", in, got)
		}
	}
}

func TestMoveCreatesFoldersAndRemovesEmptyOnes(t *testing.T) {
	f := setup(t, "in/deep/a.jpg", "in/b.jpg")
	n, err := f.ops.Move([]Target{{ID: f.id("in/deep/a.jpg"), DestDir: "2005/Trip"}})
	if err != nil || n != 1 {
		t.Fatalf("move = %d, %v", n, err)
	}
	if !f.onDisk("2005/Trip/a.jpg") || f.onDisk("in/deep") {
		t.Error("file not moved or empty folder left behind")
	}
	if !f.onDisk("in/b.jpg") {
		t.Error("folder with other files was removed")
	}
	if n, err := f.ops.Move([]Target{{ID: f.id("2005/Trip/a.jpg"), DestDir: "2005/Trip"}}); err != nil || n != 0 {
		t.Errorf("no-op move = %d, %v", n, err)
	}
	if _, err := f.ops.Move([]Target{{ID: f.id("in/b.jpg"), DestDir: "../out"}}); err == nil {
		t.Error("move outside the library was allowed")
	}
}

func TestMoveNeverOverwrites(t *testing.T) {
	f := setup(t, "a/pic.jpg", "b/pic.jpg", "c/pic.jpg")
	testutil.WriteFile(t, filepath.Join(f.root, "all/pic_1.jpg"), "stray")

	targets := []Target{
		{ID: f.id("a/pic.jpg"), DestDir: "all"},
		{ID: f.id("b/pic.jpg"), DestDir: "all"},
		{ID: f.id("c/pic.jpg"), DestDir: "all"},
	}
	if n, err := f.ops.Move(targets); err != nil || n != 3 {
		t.Fatalf("move = %d, %v", n, err)
	}
	for _, want := range []string{"all/pic.jpg", "all/pic_1.jpg", "all/pic_2.jpg", "all/pic_3.jpg"} {
		if !f.onDisk(want) {
			t.Errorf("%s missing", want)
		}
	}
}

func TestSidecarsTravelWithTheirPhoto(t *testing.T) {
	f := setup(t, "in/a.jpg", "in/a.jpg.json")
	photo, sidecar := f.id("in/a.jpg"), f.id("in/a.jpg.json")
	if err := f.st.LinkSidecars([]store.SidecarLink{{SidecarID: sidecar, MediaID: photo}}); err != nil {
		t.Fatal(err)
	}

	if n, err := f.ops.Move([]Target{{ID: photo, DestDir: "2005"}}); err != nil || n != 2 {
		t.Fatalf("move = %d, %v", n, err)
	}
	if !f.onDisk("2005/a.jpg") || !f.onDisk("2005/a.jpg.json") {
		t.Fatal("sidecar did not follow the photo")
	}

	if n, err := f.ops.Trash([]int64{photo}); err != nil || n != 2 {
		t.Fatalf("trash = %d, %v", n, err)
	}
	if f.onDisk("2005/a.jpg") || f.onDisk("2005/a.jpg.json") {
		t.Fatal("files still in place after trash")
	}
	if n, err := f.ops.Restore([]int64{photo}); err != nil || n != 2 {
		t.Fatalf("restore = %d, %v", n, err)
	}
	if !f.onDisk("2005/a.jpg") || !f.onDisk("2005/a.jpg.json") {
		t.Error("restore did not bring both files back")
	}
}

func TestUndoGoesBackOneBatchAtATime(t *testing.T) {
	f := setup(t, "in/a.jpg", "in/b.jpg")
	a, b := f.id("in/a.jpg"), f.id("in/b.jpg")

	if n, err := f.ops.Undo(); err != nil || n != 0 {
		t.Fatalf("undo with empty journal = %d, %v", n, err)
	}
	if _, err := f.ops.Move([]Target{{ID: a, DestDir: "2005"}, {ID: b, DestDir: "2005"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ops.Trash([]int64{a}); err != nil {
		t.Fatal(err)
	}

	if n, err := f.ops.Undo(); err != nil || n != 1 {
		t.Fatalf("undo trash = %d, %v", n, err)
	}
	if !f.onDisk("2005/a.jpg") {
		t.Fatal("trash not undone")
	}
	if n, err := f.ops.Undo(); err != nil || n != 2 {
		t.Fatalf("undo move = %d, %v", n, err)
	}
	if !f.onDisk("in/a.jpg") || !f.onDisk("in/b.jpg") || f.onDisk("2005") {
		t.Error("move not undone")
	}
	if n, _ := f.ops.Undo(); n != 0 {
		t.Errorf("third undo = %d", n)
	}
}

func TestUndoSkipsFilesChangedSince(t *testing.T) {
	f := setup(t, "in/a.jpg")
	a := f.id("in/a.jpg")
	if _, err := f.ops.Move([]Target{{ID: a, DestDir: "2005"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ops.Trash([]int64{a}); err != nil {
		t.Fatal(err)
	}
	var deleted []int64
	f.ops.OnDelete = func(id int64) { deleted = append(deleted, id) }
	if n, err := f.ops.EmptyTrash(); err != nil || n != 1 {
		t.Fatalf("empty trash = %d, %v", n, err)
	}
	if len(deleted) != 1 || deleted[0] != a {
		t.Errorf("OnDelete calls = %v", deleted)
	}
	for i := 0; i < 2; i++ {
		if n, err := f.ops.Undo(); err != nil || n != 0 {
			t.Errorf("undo %d = %d, %v", i, n, err)
		}
	}
	if stats, _ := f.st.Stats(); stats.Files != 0 || stats.Trashed != 0 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestRestoreWhenThePlaceIsTaken(t *testing.T) {
	f := setup(t, "a.jpg")
	a := f.id("a.jpg")
	if _, err := f.ops.Trash([]int64{a}); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(f.root, "a.jpg"), "newcomer")
	if n, err := f.ops.Restore([]int64{a}); err != nil || n != 1 {
		t.Fatalf("restore = %d, %v", n, err)
	}
	if !f.onDisk("a.jpg") || !f.onDisk("a_1.jpg") {
		t.Error("restore overwrote the newcomer or lost the file")
	}
}
