package store

import (
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func add(t *testing.T, s *Store, paths ...string) {
	t.Helper()
	var u ScanUpdate
	for i, p := range paths {
		u.New = append(u.New, FileStat{RelPath: p, Size: int64(100 + i), Mtime: 1000})
	}
	if err := s.ApplyScan(u); err != nil {
		t.Fatal(err)
	}
}

func id(t *testing.T, s *Store, rel string) int64 {
	t.Helper()
	dir, name, _ := SplitPath(rel)
	fileID, found, err := s.PresentID(dir, name)
	if err != nil || !found {
		t.Fatalf("%s not found: %v", rel, err)
	}
	return fileID
}

func names(p Page) []string {
	out := make([]string, len(p.Items))
	for i, it := range p.Items {
		out[i] = it.RelPath
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSplitPathAndKind(t *testing.T) {
	tests := []struct {
		rel, dir, name, ext, kind string
	}{
		{"a/b/IMG.JPG", "a/b", "IMG.JPG", "jpg", KindImage},
		{"clip.mov", "", "clip.mov", "mov", KindVideo},
		{"x/notes", "x", "notes", "", KindOther},
		{".hidden", "", ".hidden", "", KindOther},
		{"a/photo.jpg.json", "a", "photo.jpg.json", "json", KindOther},
	}
	for _, tt := range tests {
		dir, name, ext := SplitPath(tt.rel)
		if dir != tt.dir || name != tt.name || ext != tt.ext || KindOf(ext) != tt.kind {
			t.Errorf("SplitPath(%q) = %q, %q, %q (%s)", tt.rel, dir, name, ext, KindOf(ext))
		}
	}
}

func TestReopenKeepsData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	add(t, s, "a.jpg")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if st, _ := s.Stats(); st.Files != 1 {
		t.Errorf("files after reopen = %d", st.Files)
	}
}

func TestListFilters(t *testing.T) {
	s := open(t)
	add(t, s, "a/one.jpg", "a/sub/two.jpg", "a/clip.mp4", "b/notes.txt", "a_b/three.jpg")
	y2005 := time.Date(2005, 6, 1, 0, 0, 0, 0, time.UTC).Unix()
	y2006 := time.Date(2006, 6, 1, 0, 0, 0, 0, time.UTC).Unix()
	err := s.SaveMeta([]MetaUpdate{
		{ID: id(t, s, "a/one.jpg"), TakenAt: y2005, TakenSrc: SrcExif},
		{ID: id(t, s, "a/sub/two.jpg"), TakenAt: y2006, TakenSrc: SrcTakeout},
		{ID: id(t, s, "a/clip.mp4"), TakenAt: 1000, TakenSrc: SrcMtime},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		filter Filter
		want   []string
	}{
		{"media", Filter{Kind: "media", Sort: "name"}, []string{"a/clip.mp4", "a/one.jpg", "a/sub/two.jpg", "a_b/three.jpg"}},
		{"videos", Filter{Kind: KindVideo}, []string{"a/clip.mp4"}},
		{"other", Filter{Kind: KindOther}, []string{"b/notes.txt"}},
		{"folder", Filter{InDir: true, Dir: "a", Sort: "name"}, []string{"a/clip.mp4", "a/one.jpg"}},
		{"folder recursive", Filter{InDir: true, Dir: "a", Recursive: true, Sort: "name"},
			[]string{"a/clip.mp4", "a/one.jpg", "a/sub/two.jpg"}},
		{"year", Filter{Year: 2005}, []string{"a/one.jpg"}},
		{"no date", Filter{NoDate: true, Sort: "name"}, []string{"a/clip.mp4", "a_b/three.jpg"}},
		{"search", Filter{Search: "TWO"}, []string{"a/sub/two.jpg"}},
		{"by date", Filter{Kind: KindImage, InDir: true, Dir: "a", Recursive: true}, []string{"a/sub/two.jpg", "a/one.jpg"}},
	}
	for _, tt := range tests {
		page, err := s.List(tt.filter)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got := names(page); !equal(got, tt.want) || page.Total != len(tt.want) {
			t.Errorf("%s: got %v (total %d), want %v", tt.name, got, page.Total, tt.want)
		}
	}

	page, _ := s.List(Filter{Sort: "name", Limit: 2, Offset: 1})
	if page.Total != 5 || !equal(names(page), []string{"a/one.jpg", "a/sub/two.jpg"}) {
		t.Errorf("paging: %v (total %d)", names(page), page.Total)
	}

	years, _ := s.Years()
	if len(years) != 2 || years[0].Year != 2006 || years[1].Year != 2005 {
		t.Errorf("years = %+v", years)
	}
	stats, _ := s.Stats()
	if stats.Files != 5 || stats.Media != 4 || stats.Other != 1 || stats.NoDate != 2 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestTags(t *testing.T) {
	s := open(t)
	add(t, s, "a.jpg", "b.jpg")
	a, b := id(t, s, "a.jpg"), id(t, s, "b.jpg")

	if err := s.AddTag([]int64{a, b}, "Sea"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTag([]int64{a}, " sea "); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTag([]int64{a}, "  "); err == nil {
		t.Error("empty tag name accepted")
	}
	tags, _ := s.Tags()
	if len(tags) != 1 || tags[0].Name != "Sea" || tags[0].Count != 2 {
		t.Fatalf("tags = %+v", tags)
	}
	if page, _ := s.List(Filter{TagID: tags[0].ID}); page.Total != 2 {
		t.Errorf("files with tag = %d", page.Total)
	}

	if err := s.RemoveTag([]int64{a}, tags[0].ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.FileTags(a); len(got) != 0 {
		t.Errorf("tag still on file: %+v", got)
	}
	if err := s.DeleteTag(tags[0].ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.FileTags(b); len(got) != 0 {
		t.Errorf("deleted tag still on file: %+v", got)
	}
}

func TestAlbumsFollowFilesNotPaths(t *testing.T) {
	s := open(t)
	add(t, s, "x/a.jpg", "x/b.jpg", "x/c.jpg")
	a, b, c := id(t, s, "x/a.jpg"), id(t, s, "x/b.jpg"), id(t, s, "x/c.jpg")

	album, err := s.CreateAlbum("Trip")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddToAlbum(album, []int64{c, a}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddToAlbum(album, []int64{a, b}); err != nil {
		t.Fatal(err)
	}
	page, _ := s.List(Filter{AlbumID: album})
	if !equal(names(page), []string{"x/c.jpg", "x/a.jpg", "x/b.jpg"}) {
		t.Errorf("album order = %v", names(page))
	}

	batch, _ := s.NextBatch()
	if err := s.RecordMove(batch, a, "x/a.jpg", "2005/a.jpg"); err != nil {
		t.Fatal(err)
	}
	page, _ = s.List(Filter{AlbumID: album})
	if !equal(names(page), []string{"x/c.jpg", "2005/a.jpg", "x/b.jpg"}) {
		t.Errorf("album after move = %v", names(page))
	}

	if err := s.RemoveFromAlbum(album, []int64{c}); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameAlbum(album, "Holiday"); err != nil {
		t.Fatal(err)
	}
	albums, _ := s.Albums()
	if len(albums) != 1 || albums[0].Name != "Holiday" || albums[0].Count != 2 {
		t.Errorf("albums = %+v", albums)
	}
	if err := s.DeleteAlbum(album); err != nil {
		t.Fatal(err)
	}
	if f, err := s.GetFile(a); err != nil || f.Status != StatusPresent {
		t.Errorf("deleting an album touched the file: %+v, %v", f, err)
	}
}

func TestJournal(t *testing.T) {
	s := open(t)
	add(t, s, "a.jpg", "b.jpg")
	a, b := id(t, s, "a.jpg"), id(t, s, "b.jpg")

	if ops, err := s.LastBatch(); err != nil || ops != nil {
		t.Fatalf("empty journal = %v, %v", ops, err)
	}

	first, _ := s.NextBatch()
	if err := s.RecordMove(first, a, "a.jpg", "x/a.jpg"); err != nil {
		t.Fatal(err)
	}
	second, _ := s.NextBatch()
	if second == first {
		t.Fatal("batch numbers repeat")
	}
	if err := s.RecordMove(second, b, "b.jpg", "x/b.jpg"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordTrash(second, a, "x/a.jpg", "1_a.jpg"); err != nil {
		t.Fatal(err)
	}

	ops, _ := s.LastBatch()
	if len(ops) != 2 || ops[0].Type != OpTrash || ops[1].Type != OpMove || ops[1].FileID != b {
		t.Fatalf("last batch = %+v", ops)
	}
	trashed, _ := s.Trashed()
	if len(trashed) != 1 || trashed[0].TrashPath != "1_a.jpg" {
		t.Errorf("trashed = %+v", trashed)
	}

	for _, op := range ops {
		if err := s.MarkUndone(op.ID); err != nil {
			t.Fatal(err)
		}
	}
	ops, _ = s.LastBatch()
	if len(ops) != 1 || ops[0].FileID != a || ops[0].To != "x/a.jpg" {
		t.Errorf("batch after undo = %+v", ops)
	}
	history, _ := s.MoveHistory()
	if len(history) != 1 || history[0].From != "a.jpg" {
		t.Errorf("history = %+v", history)
	}

	if err := s.RecordRestore(a, "x/a.jpg"); err != nil {
		t.Fatal(err)
	}
	if f, _ := s.GetFile(a); f.Status != StatusPresent || f.Dir != "x" || f.TrashPath != "" {
		t.Errorf("restored file = %+v", f)
	}
}

func TestApplyScanReusesFreedPaths(t *testing.T) {
	s := open(t)
	add(t, s, "a.jpg", "b.jpg")
	a, b := id(t, s, "a.jpg"), id(t, s, "b.jpg")

	err := s.ApplyScan(ScanUpdate{
		Missing: []int64{a},
		Moved:   []KnownStat{{ID: b, FileStat: FileStat{RelPath: "a.jpg", Size: 5, Mtime: 6}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := id(t, s, "a.jpg"); got != b {
		t.Errorf("a.jpg is file %d, want %d", got, b)
	}
	if f, _ := s.GetFile(a); f.Status != StatusMissing {
		t.Errorf("old file status = %s", f.Status)
	}
}

func TestMetaLifecycle(t *testing.T) {
	s := open(t)
	add(t, s, "a.jpg", "a.jpg.json", "clip.mp4")
	a, sidecar := id(t, s, "a.jpg"), id(t, s, "a.jpg.json")

	if pending, _ := s.PendingMeta(); len(pending) != 2 {
		t.Fatalf("pending meta = %d", len(pending))
	}
	if err := s.SaveMeta([]MetaUpdate{{ID: a, TakenAt: 5, TakenSrc: SrcTakeout, TakeoutJSON: `{"x":1}`}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMeta([]MetaUpdate{{ID: a, TakenAt: 5, TakenSrc: SrcTakeout}}); err != nil {
		t.Fatal(err)
	}
	if raw, _ := s.TakeoutJSON(a); raw != `{"x":1}` {
		t.Errorf("takeout json = %q", raw)
	}

	if err := s.LinkSidecars([]SidecarLink{{SidecarID: sidecar, MediaID: a}}); err != nil {
		t.Fatal(err)
	}
	sidecars, _ := s.Sidecars(a)
	if len(sidecars) != 1 || sidecars[0].Kind != KindSidecar {
		t.Errorf("sidecars = %+v", sidecars)
	}
	if links, _ := s.MergeableSidecars(); len(links) != 0 {
		t.Error("sidecar mergeable before metadata was read again")
	}
	if err := s.SaveMeta([]MetaUpdate{{ID: a, TakenAt: 5, TakenSrc: SrcTakeout}}); err != nil {
		t.Fatal(err)
	}
	if links, _ := s.MergeableSidecars(); len(links) != 1 || links[0].MediaID != a {
		t.Errorf("mergeable = %+v", links)
	}

	if err := s.SetDateOverride(a, 99); err != nil {
		t.Fatal(err)
	}
	if f, _ := s.GetFile(a); f.DateOverride != 99 || f.TakenSrc != SrcManual {
		t.Errorf("override = %+v", f)
	}
	if err := s.MarkRewritten(a, 500, 600); err != nil {
		t.Fatal(err)
	}
	if f, _ := s.GetFile(a); f.DateOverride != 0 || f.Size != 500 || f.Hash != "" {
		t.Errorf("rewritten = %+v", f)
	}
}

func TestHashAndDuplicates(t *testing.T) {
	s := open(t)
	add(t, s, "a.jpg", "b.jpg", "c.jpg")
	files, _ := s.PendingHash()
	if len(files) != 3 {
		t.Fatalf("pending hash = %d", len(files))
	}
	for i, f := range files {
		hash := "same"
		if i == 2 {
			hash = "different"
		}
		if err := s.SetHash(f.ID, hash, f.Size, f.Mtime); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetHash(files[0].ID, "stale", files[0].Size+1, files[0].Mtime); err != nil {
		t.Fatal(err)
	}
	groups, _ := s.Duplicates()
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Errorf("duplicates = %+v", groups)
	}
}

func TestExport(t *testing.T) {
	s := open(t)
	add(t, s, "a.jpg", "b.jpg")
	a := id(t, s, "a.jpg")
	_ = s.AddTag([]int64{a}, "sea")
	album, _ := s.CreateAlbum("Trip")
	_ = s.AddToAlbum(album, []int64{a})
	_ = s.SaveMeta([]MetaUpdate{{ID: a, TakenAt: 77, TakenSrc: SrcExif}})

	files, err := s.Export()
	if err != nil || len(files) != 2 {
		t.Fatalf("export = %+v, %v", files, err)
	}
	if files[0].Path != "a.jpg" || files[0].TakenAt != 77 || !equal(files[0].Tags, []string{"sea"}) ||
		!equal(files[0].Albums, []string{"Trip"}) {
		t.Errorf("first = %+v", files[0])
	}
	if files[1].TakenAt != 0 || files[1].Tags != nil {
		t.Errorf("second = %+v", files[1])
	}
}
