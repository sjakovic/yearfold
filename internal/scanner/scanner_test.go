package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/testutil"
)

func setup(t *testing.T) (string, *store.Store) {
	t.Helper()
	root := t.TempDir()
	st, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return root, st
}

func scan(t *testing.T, root string, st *store.Store) *Changes {
	t.Helper()
	ch, err := Diff(root, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(st, ch, nil); err != nil {
		t.Fatal(err)
	}
	return ch
}

func hashAll(t *testing.T, root string, st *store.Store) {
	t.Helper()
	files, err := st.PendingHash()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		h, err := HashFile(filepath.Join(root, filepath.FromSlash(f.RelPath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := st.SetHash(f.ID, h, f.Size, f.Mtime); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFirstScanFindsEveryFileButSkipsOwnData(t *testing.T) {
	root, st := setup(t)
	testutil.WriteFile(t, filepath.Join(root, "a/one.jpg"), "1")
	testutil.WriteFile(t, filepath.Join(root, "notes.txt"), "2")
	testutil.WriteFile(t, filepath.Join(root, ".DS_Store"), "x")
	testutil.WriteFile(t, filepath.Join(root, MetaDir, "trash/old.jpg"), "x")

	var seen int
	ch, err := Diff(root, st, func(n int) { seen = n })
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.New) != 2 || seen != 2 {
		t.Fatalf("new = %+v, progress = %d", ch.New, seen)
	}
	if err := Apply(st, ch, nil); err != nil {
		t.Fatal(err)
	}
	if again := scan(t, root, st); !again.Empty() {
		t.Errorf("second scan = %+v", again)
	}
}

func TestModifiedMissingAndRevived(t *testing.T) {
	root, st := setup(t)
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	testutil.WriteFile(t, a, "one")
	testutil.WriteFile(t, b, "two")
	scan(t, root, st)

	testutil.WriteFile(t, a, "one, longer")
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	var invalidated []int64
	ch, err := Diff(root, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.Modified) != 1 || ch.Modified[0].RelPath != "a.txt" || len(ch.Missing) != 1 || len(ch.New) != 0 {
		t.Fatalf("changes = %+v", ch)
	}
	if err := Apply(st, ch, func(id int64) { invalidated = append(invalidated, id) }); err != nil {
		t.Fatal(err)
	}
	if len(invalidated) != 1 || invalidated[0] != ch.Modified[0].ID {
		t.Errorf("invalidated = %v", invalidated)
	}
	if stats, _ := st.Stats(); stats.Files != 1 || stats.Missing != 1 {
		t.Errorf("stats = %+v", stats)
	}

	testutil.WriteFile(t, b, "two")
	ch = scan(t, root, st)
	if len(ch.Revived) != 1 || len(ch.New) != 0 {
		t.Errorf("revived = %+v, new = %+v", ch.Revived, ch.New)
	}
	if stats, _ := st.Stats(); stats.Files != 2 || stats.Missing != 0 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestExternalMoveIsRecognisedByContent(t *testing.T) {
	root, st := setup(t)
	testutil.WriteFile(t, filepath.Join(root, "old/a.txt"), "same content")
	testutil.WriteFile(t, filepath.Join(root, "old/b.txt"), "other content")
	scan(t, root, st)
	hashAll(t, root, st)

	if err := os.MkdirAll(filepath.Join(root, "new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "old/a.txt"), filepath.Join(root, "new/renamed.txt")); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(root, "new/fresh.txt"), "SAME CONTENT")

	ch, err := Diff(root, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.Moved) != 1 || ch.Moved[0].From != "old/a.txt" || ch.Moved[0].To.RelPath != "new/renamed.txt" {
		t.Fatalf("moved = %+v", ch.Moved)
	}
	if len(ch.New) != 1 || ch.New[0].RelPath != "new/fresh.txt" || len(ch.Missing) != 0 {
		t.Errorf("new = %+v, missing = %+v", ch.New, ch.Missing)
	}
	if err := Apply(st, ch, nil); err != nil {
		t.Fatal(err)
	}
	f, err := st.GetFile(ch.Moved[0].ID)
	if err != nil || f.RelPath != "new/renamed.txt" || f.Status != store.StatusPresent || f.Hash == "" {
		t.Errorf("moved file = %+v, %v", f, err)
	}
}

func TestWithoutHashAMoveLooksLikeNewAndMissing(t *testing.T) {
	root, st := setup(t)
	testutil.WriteFile(t, filepath.Join(root, "a.txt"), "content")
	scan(t, root, st)

	if err := os.Rename(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")); err != nil {
		t.Fatal(err)
	}
	ch, _ := Diff(root, st, nil)
	if len(ch.Moved) != 0 || len(ch.New) != 1 || len(ch.Missing) != 1 {
		t.Errorf("changes = %+v", ch)
	}
}
