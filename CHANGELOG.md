# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [0.4.0] - 2026-10-04

### Changed

- The application is now called **Yearfold** (previously ImageManager). The
  library data folder is `.yearfold`; existing `.imagemanager` folders and the
  old settings folder are renamed automatically on first use.

## [0.3.0] - 2026-10-04

### Added

- "Merge Google Takeout data…" makes the Google JSON files unnecessary: their
  data is kept in the library, missing dates are written into JPEG and PNG
  photos, and the JSON files are moved to the trash.
- Type filter (all, photos, videos) in the toolbar.

### Changed

- Takeout data is copied into the index when a file is first read, so the
  library no longer loses dates if a JSON file disappears.

## [0.2.0] - 2026-10-04

### Added

- "Set date…" gives one or many selected photos and videos a capture date.
  For JPEG and PNG it is written into the file itself (EXIF) without touching
  any other tag; for other formats and videos it is kept in the library.

### Fixed

- EXIF dates were read in the computer's time zone and shown shifted by its
  UTC offset. They are now kept as the camera's wall-clock time; existing
  libraries are re-read automatically.

## [0.1.0] - 2026-10-04

### Added

- Open any folder as a library; every file in it, including non-images, is
  indexed into `<root>/.imagemanager/library.db`.
- "Check for changes" report of new, modified, missing and externally moved
  files, applied only after confirmation.
- Folder tree, virtualized thumbnail grid and a detail view with all embedded
  metadata (EXIF, IPTC, XMP) and Google Takeout sidecar data.
- Tags and albums stored in the index; they never move or modify files.
- Moving files inside the library, trash with restore, and undo of the last
  move or trash operation.
- Organize by year (optionally by month) with a preview before anything moves.
- Duplicate detection by content hash, year and "no date" filters, search,
  album from folder, JSON export.
- English (default) and Serbian Cyrillic interface.
