package organize

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sjakovic/yearfold/internal/store"
)

type library struct {
	t    *testing.T
	root string
	st   *store.Store
}

func setup(t *testing.T) *library {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &library{t: t, root: t.TempDir(), st: st}
}

func (l *library) add(rel string, taken time.Time) int64 {
	l.t.Helper()
	if err := l.st.ApplyScan(store.ScanUpdate{New: []store.FileStat{{RelPath: rel, Size: 1, Mtime: 1}}}); err != nil {
		l.t.Fatal(err)
	}
	dir, name, _ := store.SplitPath(rel)
	id, _, err := l.st.PresentID(dir, name)
	if err != nil {
		l.t.Fatal(err)
	}
	u := store.MetaUpdate{ID: id, TakenAt: 1, TakenSrc: store.SrcMtime}
	if !taken.IsZero() {
		u.TakenAt, u.TakenSrc = taken.Unix(), store.SrcExif
	}
	if err := l.st.SaveMeta([]store.MetaUpdate{u}); err != nil {
		l.t.Fatal(err)
	}
	return id
}

func (l *library) plan(layout Layout) (map[string]string, Plan) {
	l.t.Helper()
	plan, err := ByDate(l.st, l.root, layout)
	if err != nil {
		l.t.Fatal(err)
	}
	targets := map[string]string{}
	for _, m := range plan.Moves {
		targets[m.From] = m.To
	}
	return targets, plan
}

func (l *library) apply(layout Layout) {
	l.t.Helper()
	plan, err := ByDate(l.st, l.root, layout)
	if err != nil {
		l.t.Fatal(err)
	}
	batch, _ := l.st.NextBatch()
	for _, m := range plan.Moves {
		_, name, _ := store.SplitPath(m.From)
		if err := l.st.RecordMove(batch, m.ID, m.From, m.To+"/"+name); err != nil {
			l.t.Fatal(err)
		}
	}
}

func day(year int, month time.Month) time.Time {
	return time.Date(year, month, 15, 12, 0, 0, 0, time.UTC)
}

func expect(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("plan = %v, want %v", got, want)
		return
	}
	for from, to := range want {
		if got[from] != to {
			t.Errorf("%s -> %q, want %q", from, got[from], to)
		}
	}
}

func TestYearAndMonthLayouts(t *testing.T) {
	l := setup(t)
	l.add("Takeout/a.jpg", day(2019, time.March))
	l.add("Takeout/clip.mp4", day(2020, time.December))
	l.add("2019/already.jpg", day(2019, time.May))
	l.add("Takeout/undated.jpg", time.Time{})
	l.add("Takeout/notes.txt", day(2019, time.March))

	byYear, plan := l.plan(LayoutYear)
	expect(t, byYear, map[string]string{"Takeout/a.jpg": "2019", "Takeout/clip.mp4": "2020"})
	if plan.NoDate != 1 {
		t.Errorf("no date = %d", plan.NoDate)
	}

	byMonth, _ := l.plan(LayoutMonth)
	expect(t, byMonth, map[string]string{
		"Takeout/a.jpg":    "2019/03",
		"Takeout/clip.mp4": "2020/12",
		"2019/already.jpg": "2019/05",
	})

	if _, err := ByDate(l.st, l.root, "bogus"); err == nil {
		t.Error("unknown layout accepted")
	}
}

func TestFolderLayoutKeepsFolderNames(t *testing.T) {
	l := setup(t)
	l.add("Bogdan/Party/p1.jpg", day(2019, time.June))
	l.add("Bogdan/Party/p2.jpg", day(2020, time.January))
	l.add("Other/Party/p3.jpg", day(2019, time.June))
	l.add("Old/2019/p4.jpg", day(2019, time.June))
	l.add("loose.jpg", day(2019, time.June))

	got, _ := l.plan(LayoutFolder)
	expect(t, got, map[string]string{
		"Bogdan/Party/p1.jpg": "2019/Party",
		"Bogdan/Party/p2.jpg": "2020/Party",
		"Other/Party/p3.jpg":  "2019/Party 2",
		"Old/2019/p4.jpg":     "2019",
		"loose.jpg":           "2019",
	})

	l.apply(LayoutFolder)
	if again, _ := l.plan(LayoutFolder); len(again) != 0 {
		t.Errorf("second plan = %v", again)
	}
}

func TestFolderLayoutIsStableOverTime(t *testing.T) {
	l := setup(t)
	l.add("Bogdan/Party/p1.jpg", day(2019, time.June))
	late := l.add("Bogdan/Party/late.jpg", time.Time{})
	l.apply(LayoutFolder)

	if err := l.st.SaveMeta([]store.MetaUpdate{{ID: late, TakenAt: day(2019, time.July).Unix(), TakenSrc: store.SrcExif}}); err != nil {
		t.Fatal(err)
	}
	l.add("Third/Party/p5.jpg", day(2019, time.June))

	got, _ := l.plan(LayoutFolder)
	expect(t, got, map[string]string{
		"Bogdan/Party/late.jpg": "2019/Party",
		"Third/Party/p5.jpg":    "2019/Party 2",
	})
}

func TestFolderLayoutLeavesYearFoldersAlone(t *testing.T) {
	l := setup(t)
	l.add("2018/New Year/after-midnight.jpg", day(2019, time.January))
	l.add("2018/deep/er/x.jpg", day(2019, time.January))

	if got, _ := l.plan(LayoutFolder); len(got) != 0 {
		t.Errorf("plan = %v", got)
	}
	got, _ := l.plan(LayoutYear)
	expect(t, got, map[string]string{
		"2018/New Year/after-midnight.jpg": "2019",
		"2018/deep/er/x.jpg":               "2019",
	})
}

func TestFolderLayoutAvoidsFoldersOnDisk(t *testing.T) {
	l := setup(t)
	l.add("in/Trip/a.jpg", day(2019, time.June))
	if err := mkdir(l.root, "2019/Trip"); err != nil {
		t.Fatal(err)
	}
	got, _ := l.plan(LayoutFolder)
	expect(t, got, map[string]string{"in/Trip/a.jpg": "2019/Trip 2"})
}

func mkdir(root, rel string) error {
	return os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0o755)
}
