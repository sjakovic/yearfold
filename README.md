<h1 align="center">Yearfold</h1>

<p align="center">
  <strong>Bring your photos home and put them in order.</strong><br>
  A desktop app that turns a messy photo export into a tidy library on your own disk.
</p>

<p align="center">
  <a href="https://github.com/sjakovic/yearfold/actions/workflows/ci.yml"><img src="https://github.com/sjakovic/yearfold/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT license"></a>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/built%20with-Wails-DF0000" alt="Built with Wails">
</p>

![Yearfold showing a library organized into year folders, with albums and tags in the sidebar](docs/screenshot.jpg)

## Why

Exporting everything from Google Photos gives you one big pile: thousands of
files in folders like `Photos from 2019`, the same picture repeated in every
album folder, and a `.json` file next to each photo holding the date that the
photo itself is missing.

Yearfold is built to clean that up. Point it at the folder and it indexes
every file, shows you what you have, and helps you fold it into a simple
structure — `2004/`, `2005/`, `2006/` … — without a cloud service and without
taking ownership of your files. Your photos stay ordinary files in ordinary
folders that any other program can read.

## Features

- **Indexes everything** – every file under the folder you pick, including the
  ones that are not photos, so nothing is hidden from you.
- **Organize by year** – preview exactly what will move where, then move dated
  photos and videos into `2004/`, `2005/` … (optionally `2004/05/`).
- **Understands Google Takeout** – reads dates, locations, descriptions and
  people from the JSON sidecars, including Takeout's truncated and numbered
  file names. One action merges that data into the library, writes missing
  dates into the photos and clears the JSON files away.
- **Albums and tags that never move a file** – they live in the library index.
  A photo stays in its album when you move it to another folder.
- **Real moves when you want them** – move photos to any folder inside the
  library; sidecar files travel with them.
- **Fix missing dates** – set a capture date on one photo or many. JPEG and
  PNG get it written into the file (EXIF) with every other tag left intact;
  other formats and videos keep it in the library.
- **Duplicates by content** – found by SHA-256 of the file, not by name.
- **Check for changes** – after you add or rearrange files outside the app,
  review what is new, changed, missing or moved before the index is updated.
  Files moved in Finder or Explorer keep their tags and albums.
- **Hard to lose anything** – delete goes to a library trash you can restore
  from, and the last move or delete can be undone.
- **All the metadata** – EXIF, IPTC and XMP for every photo, in one panel.
- **Fast with big libraries** – virtualized grid, thumbnails generated on
  demand, metadata and hashes computed in the background.
- **English and Serbian (Cyrillic)** interface.

## How it treats your files

Yearfold does not change your photos on its own. Only actions you start touch
the disk:

| Action | What happens on disk |
|---|---|
| Add to album, tag | Nothing. Stored in the index. |
| Move, Organize by year | Files are moved inside the library folder. Undoable. |
| Move to trash | Files go to the library trash. Restorable until you empty it. |
| Set date | JPEG/PNG: the date is written into the file. Other formats: nothing. |
| Merge Google Takeout data | Missing dates are written into JPEG/PNG; JSON files go to the trash. |

Everything Yearfold knows lives next to your photos:

```
Photos/
├── 2004/
├── 2005/
├── …
└── .yearfold/          hidden
    ├── library.db      index: files, tags, albums, dates, undo history
    ├── thumbs/         thumbnail cache, safe to delete
    └── trash/          deleted files until you empty the trash
```

Paths in the index are relative to the library folder, so a library on an
external disk — or copied to another computer — opens with its tags and albums
intact. App settings (recent folders, language) are kept separately in the
operating system's config directory.

## Getting started

There are no prebuilt downloads yet; build it from source.

**Requirements:** [Go](https://go.dev) 1.25+, [Node.js](https://nodejs.org) 20+
and the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation).

```sh
git clone https://github.com/sjakovic/yearfold.git
cd yearfold
wails build
```

The app is created in `build/bin`. Yearfold is built with Wails, which targets
macOS, Windows and Linux; so far it has been developed and tested on macOS
(Apple silicon).

Optional: with `ffmpeg` on the `PATH`, videos get thumbnails instead of an icon.

### A typical Takeout clean-up

1. Unpack the Takeout archives into one folder and open it in Yearfold.
2. Wait for the status bar to finish reading metadata and computing hashes.
3. Open **Duplicates**, keep one copy of each photo and trash the rest.
4. Run **Merge Google Takeout data…** from the `⋯` menu.
5. Open **No date**, select photos you can place in time and **Set date…**.
6. Click **Organize by year**, check the preview and confirm.

Try it on a copy of a small part of your library first.

## Development

```sh
wails dev        # run with live reload
go test ./...    # backend tests
wails build      # production build
```

```
main.go, app*.go     entry point and the API exposed to the UI
internal/store       SQLite index: files, tags, albums, operations journal
internal/scanner     walks the library and diffs it against the index
internal/meta        reads metadata and Takeout sidecars, writes dates
internal/thumbs      on-demand thumbnail cache
internal/fileops     move, trash, restore, undo
internal/organize    rule-based move plans (by year)
internal/library     ties the above together for one library
internal/assets      serves thumbnails and originals to the webview
frontend/            React + TypeScript UI
```

The backend is plain Go with a pure-Go SQLite driver, so the logic in
`internal/` builds and tests without cgo. The UI talks to it only through the
methods in `app*.go`.

The project follows [Semantic Versioning](https://semver.org/). The version
lives in `wails.json` (`info.productVersion`), is embedded into the binary and
shown in the app menu. Changes are recorded in [CHANGELOG.md](CHANGELOG.md).

## License

[MIT](LICENSE) © Simo Jaković
