package library

import (
	"context"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sjakovic/yearfold/internal/fileops"
	"github.com/sjakovic/yearfold/internal/organize"
	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/testutil"
)

func newLibrary(t *testing.T) *Library {
	t.Helper()
	root := t.TempDir()
	testutil.WriteJPEG(t, filepath.Join(root, "Takeout/Photos from 2019/IMG_1.jpg"), 10)
	testutil.WriteFile(t, filepath.Join(root, "Takeout/Photos from 2019/IMG_1.jpg.supplemental-metadata.json"),
		`{"title":"IMG_1.jpg","photoTakenTime":{"timestamp":"1560600000"},"geoData":{"latitude":44.8,"longitude":20.4}}`)
	testutil.WriteJPEG(t, filepath.Join(root, "Takeout/Photos from 2019/IMG_2.jpg"), 20)
	testutil.WriteJPEG(t, filepath.Join(root, "Takeout/Album/IMG_1.jpg"), 10) // duplicate content
	testutil.WriteFile(t, filepath.Join(root, "Takeout/notes.txt"), "not an image")

	lib, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
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
	dir, name, _ := store.SplitPath(rel)
	id, found, err := lib.St.PresentID(dir, name)
	if err != nil || !found {
		t.Fatalf("%s: not in the index (%v)", rel, err)
	}
	f, err := lib.St.GetFile(id)
	if err != nil {
		t.Fatal(err)
	}
	return f
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
	if !testutil.Exists(filepath.Join(lib.Root, "2019/IMG_1.jpg")) ||
		!testutil.Exists(filepath.Join(lib.Root, "2019/IMG_1.jpg.supplemental-metadata.json")) {
		t.Fatal("files not at destination")
	}
	if testutil.Exists(filepath.Join(lib.Root, img.RelPath)) {
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
	if !testutil.Exists(filepath.Join(lib.Root, img.RelPath)) || testutil.Exists(filepath.Join(lib.Root, "2019")) {
		t.Error("undo did not restore the original layout")
	}
	if got := byPath(t, lib, img.RelPath); got.ID != img.ID {
		t.Error("index not restored by undo")
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
	if _, err := lib.Thumbs.Get(got.ID, lib.Abs(got), got.Ext, got.Kind, got.Orientation); err != nil {
		t.Errorf("image no longer decodes: %v", err)
	}
	if ch, _ := scanner.Diff(lib.Root, lib.St, nil); !ch.Empty() {
		t.Errorf("scan reports changes after SetDate: %+v", ch)
	}
	if page, _ := lib.St.List(store.Filter{AlbumID: album}); page.Total != 1 {
		t.Error("file dropped out of its album")
	}
	plan, _ := organize.ByDate(lib.St, lib.Root, organize.LayoutYear)
	found := false
	for _, m := range plan.Moves {
		found = found || (m.ID == img.ID && m.To == "2004")
	}
	if !found {
		t.Errorf("dated file not planned into 2004: %+v", plan.Moves)
	}

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
	testutil.WriteFile(t, filepath.Join(lib.Root, "clips/video.mp4"), "not a real video")
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

	if err := lib.St.RequeueMeta(video.ID); err != nil {
		t.Fatal(err)
	}
	process(t, lib)
	if got := byPath(t, lib, video.RelPath); got.TakenSrc != store.SrcManual || got.TakenAt != want.Unix() {
		t.Errorf("after re-read: taken = %d (%s)", got.TakenAt, got.TakenSrc)
	}
	plan, _ := organize.ByDate(lib.St, lib.Root, organize.LayoutYear)
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
	if !testutil.Exists(filepath.Join(lib.Root, rel+".supplemental-metadata.json")) {
		t.Fatal("planning removed the JSON")
	}

	res, err := lib.MergeTakeout(ctx)
	if err != nil || res.Trashed != 1 || res.Dates != 1 || res.Skipped != 0 {
		t.Fatalf("merge = %+v, %v", res, err)
	}
	if testutil.Exists(filepath.Join(lib.Root, rel+".supplemental-metadata.json")) {
		t.Error("JSON still next to the photo")
	}
	process(t, lib)

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

	if n, err := lib.Ops.Undo(); err != nil || n != 1 {
		t.Fatalf("undo = %d, %v", n, err)
	}
	if !testutil.Exists(filepath.Join(lib.Root, rel+".supplemental-metadata.json")) {
		t.Error("undo did not restore the JSON")
	}
}

func TestTakeoutDataSurvivesWithoutJSONForOtherFormats(t *testing.T) {
	lib := newLibrary(t)
	testutil.WriteFile(t, filepath.Join(lib.Root, "clips/VID_1.mp4"), "not a real video")
	testutil.WriteFile(t, filepath.Join(lib.Root, "clips/VID_1.mp4.supplemental-metadata.json"),
		`{"title":"VID_1.mp4","description":"more","photoTakenTime":{"timestamp":"1262304000"}}`)
	if _, err := lib.Scan(nil); err != nil {
		t.Fatal(err)
	}
	process(t, lib)

	if _, err := lib.MergeTakeout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := lib.St.RequeueMeta(byPath(t, lib, "clips/VID_1.mp4").ID); err != nil {
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
	_ = lib.Close()
	if err := os.Rename(filepath.Join(root, ".yearfold"), filepath.Join(root, ".imagemanager")); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	if testutil.Exists(filepath.Join(root, ".imagemanager")) || !testutil.Exists(filepath.Join(root, ".yearfold", "library.db")) {
		t.Error("old data folder was not renamed")
	}
	if tags, _ := reopened.St.FileTags(img.ID); len(tags) != 1 {
		t.Error("tags lost while adopting the old data folder")
	}
}

func TestVideoThumbnailOnMacOS(t *testing.T) {
	sample := os.Getenv("YEARFOLD_TEST_VIDEO")
	if runtime.GOOS != "darwin" || sample == "" {
		t.Skip("set YEARFOLD_TEST_VIDEO to a video file to run this on macOS")
	}
	data, err := os.ReadFile(sample)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "clip.mov"), string(data))
	lib, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	if _, err := lib.Scan(nil); err != nil {
		t.Fatal(err)
	}
	clip := byPath(t, lib, "clip.mov")
	p, err := lib.Thumbs.Get(clip.ID, lib.Abs(clip), clip.Ext, clip.Kind, 1)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil || cfg.Width == 0 || cfg.Width > 360 || cfg.Height > 360 {
		t.Errorf("thumbnail = %dx%d, %v", cfg.Width, cfg.Height, err)
	}
}
