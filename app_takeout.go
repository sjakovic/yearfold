package main

import (
	"github.com/sjakovic/yearfold/internal/library"
)

// TakeoutSummary previews merging the Google Takeout JSON files.
func (a *App) TakeoutSummary() (library.TakeoutSummary, error) {
	lib, err := a.library()
	if err != nil {
		return library.TakeoutSummary{}, err
	}
	return lib.PlanTakeoutMerge(a.libraryContext())
}

// MergeTakeout keeps the Google Takeout data in the library, writes missing
// dates into photos and moves the JSON files to the trash.
func (a *App) MergeTakeout() (library.TakeoutResult, error) {
	lib, err := a.library()
	if err != nil {
		return library.TakeoutResult{}, err
	}
	res, err := lib.MergeTakeout(a.libraryContext())
	if res.Dates > 0 {
		go a.process()
	}
	return res, err
}
