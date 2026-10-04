package library

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sjakovic/yearfold/internal/fileops"
	"github.com/sjakovic/yearfold/internal/organize"
	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
)

func writeJPEG(t *testing.T, path string, shade uint8) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for i := range img.Pix {
		img.Pix[i] = shade
	}
	img.Set(1, 1, color.RGBA{shade, 0, 0, 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newLibrary builds a small Takeout-like tree and runs the full pipeline.
func newLibrary(t *testing.T) *Library {
	t.Helper()
	root := t.TempDir()
	writeJPEG(t, filepath.Join(root, "Takeout/Photos from 2019/IMG_1.jpg"), 10)
	// 2019-06-15 12:00:00 UTC
	write(t, filepath.Join(root, "Takeout/Photos from 2019/IMG_1.jpg.supplemental-metadata.json"),
		`{"title":"IMG_1.jpg","photoTakenTime":{"timestamp":"1560600000"},"geoData":{"latitude":44.8,"longitude":20.4}}`)
	writeJPEG(t, filepath.Join(root, "Takeout/Photos from 2019/IMG_2.jpg"), 20)
	writeJPEG(t, filepath.Join(root, "Takeout/Album/IMG_1.jpg"), 10) // duplicate content
	write(t, filepath.Join(root, "Takeout/notes.txt"), "not an image")

	lib, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lib.Close() })
	if _, err := lib.Scan(nil); err != nil {
		t.Fatal(err)
	}
	process(t, lib)
	return lib
}

func process(t *testing.T, lib *Library) {
	t.Helper()
	if err := lib.LinkSidecars(); err != nil {
		t.Fatal(err)
	}
	if err := lib.ExtractMeta(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if err := lib.HashPending(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func byPath(t *testing.T, lib *Library, rel string) store.File {
	t.Helper()
	var id int64
	if err := lib.St.DB.QueryRow(`SELECT id FROM files WHERE rel_path = ? AND status = 'present'`, rel).Scan(&id); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	f, err := lib.St.GetFile(id)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func TestScanIndexesEverything(t *testing.T) {
	lib := newLibrary(t)
	stats, err := lib.St.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Files != 5 || stats.Media != 3 || stats.Other != 2 {
		t.Fatalf("stats = %+v", stats)
	}

	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg")
	if img.TakenSrc != store.SrcTakeout || img.TakenAt != 1560600000 {
		t.Errorf("taken = %d (%s), want takeout date", img.TakenAt, img.TakenSrc)
	}
	if !img.HasGPS || img.Width != 64 || img.Height != 48 || img.Hash == "" {
		t.Errorf("metadata not filled: %+v", img)
	}
	sc := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg.supplemental-metadata.json")
	if sc.SidecarOf != img.ID || sc.Kind != store.KindSidecar {
		t.Errorf("sidecar not linked: %+v", sc)
	}
	if other := byPath(t, lib, "Takeout/Photos from 2019/IMG_2.jpg"); other.TakenSrc != store.SrcMtime {
		t.Errorf("undated file has src %q", other.TakenSrc)
	}

	// A second scan of an unchanged tree finds nothing.
	ch, err := scanner.Diff(lib.Root, lib.St, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ch.Empty() {
		t.Errorf("rescan found changes: %+v", ch)
	}
}

func TestMoveKeepsAlbumAndTagsAndUndo(t *testing.T) {
	lib := newLibrary(t)
	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg")

	album, err := lib.St.CreateAlbum("Leto")
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.St.AddToAlbum(album, []int64{img.ID}); err != nil {
		t.Fatal(err)
	}
	if err := lib.St.AddTag([]int64{img.ID}, "more"); err != nil {
		t.Fatal(err)
	}
	// Adding to an album must not touch the file.
	if got := byPath(t, lib, img.RelPath); got.ID != img.ID {
		t.Fatal("album changed the file path")
	}

	n, err := lib.Ops.Move([]fileops.Target{{ID: img.ID, DestDir: "2019"}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("moved %d files, want media + sidecar", n)
	}
	if !exists(filepath.Join(lib.Root, "2019/IMG_1.jpg")) ||
		!exists(filepath.Join(lib.Root, "2019/IMG_1.jpg.supplemental-metadata.json")) {
		t.Fatal("files not at destination")
	}
	if exists(filepath.Join(lib.Root, img.RelPath)) {
		t.Fatal("file still at old location")
	}

	page, err := lib.St.List(store.Filter{AlbumID: album})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].RelPath != "2019/IMG_1.jpg" {
		t.Errorf("album after move = %+v", page)
	}
	tags, _ := lib.St.FileTags(img.ID)
	if len(tags) != 1 || tags[0].Name != "more" {
		t.Errorf("tags after move = %+v", tags)
	}

	if n, err := lib.Ops.Undo(); err != nil || n != 2 {
		t.Fatalf("undo = %d, %v", n, err)
	}
	if !exists(filepath.Join(lib.Root, img.RelPath)) || exists(filepath.Join(lib.Root, "2019")) {
		t.Error("undo did not restore the original layout")
	}
	if got := byPath(t, lib, img.RelPath); got.ID != img.ID {
		t.Error("index not restored by undo")
	}
}

func TestMoveRejectsOutsideRootAndAvoidsCollisions(t *testing.T) {
	lib := newLibrary(t)
	a := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg")
	b := byPath(t, lib, "Takeout/Album/IMG_1.jpg")

	for _, dest := range []string{"../elsewhere", "a/../../b", ".yearfold/trash"} {
		if _, err := lib.Ops.Move([]fileops.Target{{ID: a.ID, DestDir: dest}}); err == nil {
			t.Errorf("move to %q was allowed", dest)
		}
	}
	if _, err := lib.Ops.Move([]fileops.Target{{ID: a.ID, DestDir: "all"}, {ID: b.ID, DestDir: "all"}}); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(lib.Root, "all/IMG_1.jpg")) || !exists(filepath.Join(lib.Root, "all/IMG_1_1.jpg")) {
		t.Error("name collision not resolved with a suffix")
	}
}

func TestTrashRestoreAndEmpty(t *testing.T) {
	lib := newLibrary(t)
	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg")
	abs := filepath.Join(lib.Root, img.RelPath)

	if n, err := lib.Ops.Trash([]int64{img.ID}); err != nil || n != 2 {
		t.Fatalf("trash = %d, %v", n, err)
	}
	if exists(abs) {
		t.Fatal("trashed file still in place")
	}
	page, _ := lib.St.List(store.Filter{Status: store.StatusTrashed})
	if page.Total != 2 {
		t.Errorf("trash holds %d files", page.Total)
	}
	// The trash folder is not picked up as new files.
	if ch, _ := scanner.Diff(lib.Root, lib.St, nil); !ch.Empty() {
		t.Errorf("scan sees trash: %+v", ch)
	}

	if n, err := lib.Ops.Restore([]int64{img.ID}); err != nil || n != 2 {
		t.Fatalf("restore = %d, %v", n, err)
	}
	if !exists(abs) || !exists(abs+".supplemental-metadata.json") {
		t.Fatal("restore did not bring files back")
	}

	if _, err := lib.Ops.Trash([]int64{img.ID}); err != nil {
		t.Fatal(err)
	}
	if n, err := lib.Ops.EmptyTrash(); err != nil || n != 2 {
		t.Fatalf("empty trash = %d, %v", n, err)
	}
	stats, _ := lib.St.Stats()
	if stats.Trashed != 0 || stats.Files != 3 {
		t.Errorf("stats after empty = %+v", stats)
	}
}

func TestCheckChangesDetectsNewMissingAndExternalMove(t *testing.T) {
	lib := newLibrary(t)
	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_2.jpg")
	if err := lib.St.AddTag([]int64{img.ID}, "keep"); err != nil {
		t.Fatal(err)
	}

	// Outside the app: move one file, delete one, add a new folder.
	if err := os.MkdirAll(filepath.Join(lib.Root, "sorted"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(lib.Root, img.RelPath), filepath.Join(lib.Root, "sorted/renamed.jpg")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(lib.Root, "Takeout/notes.txt")); err != nil {
		t.Fatal(err)
	}
	writeJPEG(t, filepath.Join(lib.Root, "new folder/fresh.jpg"), 99)

	ch, err := scanner.Diff(lib.Root, lib.St, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.New) != 1 || ch.New[0].RelPath != "new folder/fresh.jpg" {
		t.Errorf("new = %+v", ch.New)
	}
	if len(ch.Missing) != 1 || ch.Missing[0].RelPath != "Takeout/notes.txt" {
		t.Errorf("missing = %+v", ch.Missing)
	}
	if len(ch.Moved) != 1 || ch.Moved[0].ID != img.ID || ch.Moved[0].To.RelPath != "sorted/renamed.jpg" {
		t.Errorf("moved = %+v", ch.Moved)
	}

	// Nothing is written until the changes are applied.
	if got, _ := lib.St.GetFile(img.ID); got.RelPath != img.RelPath {
		t.Error("diff modified the index")
	}
	if err := lib.Apply(ch); err != nil {
		t.Fatal(err)
	}
	moved, _ := lib.St.GetFile(img.ID)
	if moved.RelPath != "sorted/renamed.jpg" || moved.Name != "renamed.jpg" || moved.Status != store.StatusPresent {
		t.Errorf("moved file = %+v", moved)
	}
	if tags, _ := lib.St.FileTags(img.ID); len(tags) != 1 {
		t.Error("externally moved file lost its tags")
	}
	stats, _ := lib.St.Stats()
	if stats.Missing != 1 || stats.Files != 5 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestOrganizeByYearAndDuplicates(t *testing.T) {
	lib := newLibrary(t)

	groups, err := lib.St.Duplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Fatalf("duplicates = %+v", groups)
	}

	plan, err := organize.ByDate(lib.St, false)
	if err != nil {
		t.Fatal(err)
	}
	// Only the file with a Takeout date is planned; the two without a date are skipped.
	if len(plan.Moves) != 1 || plan.Moves[0].To != "2019" || plan.NoDate != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	if exists(filepath.Join(lib.Root, "2019")) {
		t.Fatal("planning touched the disk")
	}
	byMonth, _ := organize.ByDate(lib.St, true)
	if len(byMonth.Moves) != 1 || byMonth.Moves[0].To != "2019/06" {
		t.Errorf("month plan = %+v", byMonth)
	}
}

func TestSetDateWritesIntoTheFile(t *testing.T) {
	lib := newLibrary(t)
	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_2.jpg")
	if img.TakenSrc != store.SrcMtime {
		t.Fatalf("precondition: file already has a date (%s)", img.TakenSrc)
	}
	album, _ := lib.St.CreateAlbum("A")
	if err := lib.St.AddToAlbum(album, []int64{img.ID}); err != nil {
		t.Fatal(err)
	}

	want := time.Date(2004, 6, 15, 12, 30, 0, 0, time.UTC)
	if res, err := lib.SetDate([]int64{img.ID}, want); err != nil || res.Written != 1 || res.Indexed != 0 {
		t.Fatalf("SetDate = %+v, %v", res, err)
	}
	process(t, lib)

	got := byPath(t, lib, img.RelPath)
	if got.ID != img.ID || got.TakenSrc != store.SrcExif || got.TakenAt != want.Unix() {
		t.Errorf("after SetDate: id %d, taken %d (%s); want exif %d", got.ID, got.TakenAt, got.TakenSrc, want.Unix())
	}
	if got.Width != 64 || got.Height != 48 || got.Hash == "" || got.Hash == img.Hash {
		t.Errorf("file not re-read correctly: %+v", got)
	}
	// The rewritten file is still a valid image and the index matches the disk.
	if _, err := lib.Thumbs.Get(got.ID, lib.Abs(got), got.Ext, got.Kind, got.Orientation); err != nil {
		t.Errorf("image no longer decodes: %v", err)
	}
	if ch, _ := scanner.Diff(lib.Root, lib.St, nil); !ch.Empty() {
		t.Errorf("scan reports changes after SetDate: %+v", ch)
	}
	if page, _ := lib.St.List(store.Filter{AlbumID: album}); page.Total != 1 {
		t.Error("file dropped out of its album")
	}
	plan, _ := organize.ByDate(lib.St, false)
	found := false
	for _, m := range plan.Moves {
		found = found || (m.ID == img.ID && m.To == "2004")
	}
	if !found {
		t.Errorf("dated file not planned into 2004: %+v", plan.Moves)
	}

	// A second change replaces the date instead of adding another one.
	later := time.Date(2010, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := lib.SetDate([]int64{img.ID}, later); err != nil {
		t.Fatal(err)
	}
	process(t, lib)
	if got := byPath(t, lib, img.RelPath); got.TakenAt != later.Unix() {
		t.Errorf("second SetDate: taken %d, want %d", got.TakenAt, later.Unix())
	}

	txt := byPath(t, lib, "Takeout/notes.txt")
	if _, err := lib.SetDate([]int64{txt.ID}, want); err == nil {
		t.Error("SetDate accepted a non-image")
	}
}

func TestSetDateFallsBackToTheIndex(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no exiftool
	lib := newLibrary(t)
	write(t, filepath.Join(lib.Root, "clips/video.mp4"), "not a real video")
	if _, err := lib.Scan(nil); err != nil {
		t.Fatal(err)
	}
	process(t, lib)
	video := byPath(t, lib, "clips/video.mp4")

	want := time.Date(2006, 3, 4, 5, 6, 7, 0, time.UTC)
	res, err := lib.SetDate([]int64{video.ID}, want)
	if err != nil || res.Indexed != 1 || res.Written != 0 {
		t.Fatalf("SetDate = %+v, %v", res, err)
	}
	if data, _ := os.ReadFile(lib.Abs(video)); string(data) != "not a real video" {
		t.Error("file was modified although its format cannot hold a date")
	}
	got := byPath(t, lib, video.RelPath)
	if got.TakenSrc != store.SrcManual || got.TakenAt != want.Unix() {
		t.Errorf("taken = %d (%s), want manual %d", got.TakenAt, got.TakenSrc, want.Unix())
	}

	// The date survives metadata being read again and counts as a real date.
	if _, err := lib.St.DB.Exec(`UPDATE files SET meta_done = 0 WHERE id = ?`, video.ID); err != nil {
		t.Fatal(err)
	}
	process(t, lib)
	if got := byPath(t, lib, video.RelPath); got.TakenSrc != store.SrcManual || got.TakenAt != want.Unix() {
		t.Errorf("after re-read: taken = %d (%s)", got.TakenAt, got.TakenSrc)
	}
	plan, _ := organize.ByDate(lib.St, false)
	found := false
	for _, m := range plan.Moves {
		found = found || (m.ID == video.ID && m.To == "2006")
	}
	if !found {
		t.Errorf("manually dated file not planned into 2006: %+v", plan.Moves)
	}
}

func TestMergeTakeoutMakesJSONUnnecessary(t *testing.T) {
	lib := newLibrary(t)
	ctx := context.Background()
	rel := "Takeout/Photos from 2019/IMG_1.jpg"
	img := byPath(t, lib, rel)

	plan, err := lib.PlanTakeoutMerge(ctx)
	if err != nil || plan.Sidecars != 1 || plan.Dates != 1 {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if !exists(filepath.Join(lib.Root, rel+".supplemental-metadata.json")) {
		t.Fatal("planning removed the JSON")
	}

	res, err := lib.MergeTakeout(ctx)
	if err != nil || res.Trashed != 1 || res.Dates != 1 || res.Skipped != 0 {
		t.Fatalf("merge = %+v, %v", res, err)
	}
	if exists(filepath.Join(lib.Root, rel+".supplemental-metadata.json")) {
		t.Error("JSON still next to the photo")
	}
	process(t, lib)

	// The date now comes from the photo itself; location stays in the index.
	got := byPath(t, lib, rel)
	if got.ID != img.ID || got.TakenSrc != store.SrcExif || got.TakenAt != 1560600000 {
		t.Errorf("after merge: taken %d (%s)", got.TakenAt, got.TakenSrc)
	}
	if !got.HasGPS || got.Lat != 44.8 {
		t.Errorf("location lost: %+v", got)
	}
	if ch, _ := scanner.Diff(lib.Root, lib.St, nil); !ch.Empty() {
		t.Errorf("scan reports changes after merge: %+v", ch)
	}
	if again, _ := lib.PlanTakeoutMerge(ctx); again.Sidecars != 0 {
		t.Errorf("second plan = %+v", again)
	}

	// Undo brings the JSON back from the trash.
	if n, err := lib.Ops.Undo(); err != nil || n != 1 {
		t.Fatalf("undo = %d, %v", n, err)
	}
	if !exists(filepath.Join(lib.Root, rel+".supplemental-metadata.json")) {
		t.Error("undo did not restore the JSON")
	}
}

func TestTakeoutDataSurvivesWithoutJSONForOtherFormats(t *testing.T) {
	lib := newLibrary(t)
	// A video cannot hold the date, so everything must come from the index.
	write(t, filepath.Join(lib.Root, "clips/VID_1.mp4"), "not a real video")
	write(t, filepath.Join(lib.Root, "clips/VID_1.mp4.supplemental-metadata.json"),
		`{"title":"VID_1.mp4","description":"more","photoTakenTime":{"timestamp":"1262304000"}}`)
	if _, err := lib.Scan(nil); err != nil {
		t.Fatal(err)
	}
	process(t, lib)

	if _, err := lib.MergeTakeout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.St.DB.Exec(`UPDATE files SET meta_done = 0`); err != nil {
		t.Fatal(err)
	}
	process(t, lib)

	video := byPath(t, lib, "clips/VID_1.mp4")
	if video.TakenSrc != store.SrcTakeout || video.TakenAt != 1262304000 {
		t.Errorf("video date lost: %d (%s)", video.TakenAt, video.TakenSrc)
	}
	if !strings.Contains(video.MetaJSON, "more") {
		t.Errorf("description lost: %s", video.MetaJSON)
	}
	if data, _ := os.ReadFile(lib.Abs(video)); string(data) != "not a real video" {
		t.Error("video file was modified")
	}
}

func TestOpenAdoptsLibraryFromOldName(t *testing.T) {
	lib := newLibrary(t)
	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg")
	if err := lib.St.AddTag([]int64{img.ID}, "keep"); err != nil {
		t.Fatal(err)
	}
	root := lib.Root
	lib.Close()
	if err := os.Rename(filepath.Join(root, ".yearfold"), filepath.Join(root, ".imagemanager")); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if exists(filepath.Join(root, ".imagemanager")) || !exists(filepath.Join(root, ".yearfold", "library.db")) {
		t.Error("old data folder was not renamed")
	}
	if tags, _ := reopened.St.FileTags(img.ID); len(tags) != 1 {
		t.Error("tags lost while adopting the old data folder")
	}
}

func TestThumbnail(t *testing.T) {
	lib := newLibrary(t)
	img := byPath(t, lib, "Takeout/Photos from 2019/IMG_1.jpg")
	p, err := lib.Thumbs.Get(img.ID, lib.Abs(img), img.Ext, img.Kind, img.Orientation)
	if err != nil {
		t.Fatal(err)
	}
	if !exists(p) {
		t.Fatal("thumbnail not written")
	}
	txt := byPath(t, lib, "Takeout/notes.txt")
	if _, err := lib.Thumbs.Get(txt.ID, lib.Abs(txt), txt.Ext, txt.Kind, 1); err == nil {
		t.Error("thumbnail for a text file should fail")
	}
}
