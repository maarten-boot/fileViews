# Plan — fileViews

_Status: draft v0.19 · Last updated: 2026-10-29_

## Approach
Build the non-GUI core first, with unit tests (NFR-003), then add the Fyne GUI on top of it, one area at a time.

## Proposed package layout
```
cmd/fileviews/main.go     entry point
internal/listing/         read directory, entries, ls -lF formatting, sorting, stats
internal/history/         back/forward history, recent directories
internal/bookmarks/       side pane items, depth, colours, hover text
internal/config/          load/save config.json (atomic, defaults, broken-file handling)
internal/contrast/        WCAG contrast ratio, fallback text colour, allowed gray
internal/ui/              Fyne windows and widgets
internal/arch/            architecture test: core packages have no GUI dependency (NFR-003)
```

## Milestones
| # | Milestone | Spec refs | Status |
|---|-----------|-----------|--------|
| 0 | Gather scenario, draft Spec | — | done |
| 1 | Project scaffolding: `go mod init`, layout, Fyne dependency, empty window, Makefile / `go test ./...` | NFR-004 | done (Fyne v2.8.2) |
| 2 | Core: directory listing, `ls -lF` formatting, type markers, sorting, hidden filter, stats, free space | FR-020–026, FR-029, FR-050 | done |
| 3 | Core: navigation history, recent directories, path resolution and checks | FR-011–016 | done |
| 4 | Core: config load/save, normalisation | FR-060–064 | done |
| 5 | Core: bookmarks (defaults, order, depth, colours with contrast check, hover text, missing style) | FR-040–047, FR-075–076 | done |
| 6 | GUI: main window layout — light theme passing the contrast minimum, menu bar, shortcuts, path bar, split panes, status bar, startup directory | FR-001, FR-002, FR-010–017, FR-070–071, FR-077, §4 | done — panes are plain lists until M7/M8; Settings dialogs disabled until M9; settings not saved until M10 |
| 7 | GUI: right pane table — columns, sorting, double-click, hidden toggle | FR-020–028 | done — multi-select (Q-4) and light/toggleable grid lines (FR-031) added afterwards |
| 8 | GUI: left pane — indent, colours, tooltips, context menu, colour dialog with live ratio | FR-040–047, FR-072, FR-075 | done |
| 9 | GUI: Settings dialogs — column order/visibility, indent, contrast minimum (size format is already a menu toggle; colour dialog done in M8) | FR-024, FR-044, FR-074, FR-076 | done |
| 10 | Persistence wiring (save on change/quit), window state, error handling, performance check | FR-061, FR-065, NFR-001, NFR-002 | done (before M9, at the user's request) |
| 11 | Final crosscheck Spec ↔ Plan ↔ code, manual test pass | all | crosscheck done; manual pass by the user pending (`ManualTests.md`, `make testdir`) |
| 12 | Feedback from the manual pass: indent field 1–25, bookmark order (move up/down, drag within pane), drag directories from the right pane | FR-040, FR-044, FR-048, FR-049 | done |

**Crosscheck points:** after milestones 5 (core done), 8 (main GUI done) and 11; after that, on request (#4 after milestone 12).

## Risks / notes
- Fyne's standard widgets do not support per-item background colours; a custom list item widget is needed (milestone 8).
- **Confirmed at crosscheck 1:** Fyne v2.8.2 has no tooltip support at all. **Done in M8:** bookmark items show a pop-up after 0.6 s of hovering (`desktop.Hoverable`); an atomic counter cancels pending pop-ups when the mouse leaves.
- No display in the assistant's shell: GUI code is verified with Fyne's headless `test` driver; visual checks are done manually by the user.
- Owner/group lookup and link count use `syscall.Stat_t` (Unix only); keep that in a file with a build tag.
- Free disk space uses `statfs` (Unix only); same approach. Both live in `internal/listing/stat_unix.go`, with a stub in `stat_other.go`.
- Fyne follows the desktop dark mode by default; FR-070 needs a custom theme that forces the light variant. **Done in M6** (`internal/ui/theme.go`); Fyne's own light theme failed FR-071/FR-077 (gray `#565656` text, `#E3E3E3` disabled text, `#888` placeholder on `#F3F3F3`, white on light green/orange/red).
- Fyne has no drag and drop between widgets: bookmark rows and table cells implement `fyne.Draggable` and report the mouse; the drop slot is computed from the bookmark rows Fyne has actually laid out (`sidePane.rendered`), skipping the list's measuring templates (they never get a row).
- **Found in the manual test pass (scrolling):** Fyne turns a vertical wheel movement into a sideways scroll when only the columns overflow, and its Table does not re-lay-out its scroller when the row count changes (a big directory opened after a small one could not scroll). Fixes: a wheel-catching layer filters wheel events (FR-032), the Name column fills the width, and the scroller is refreshed after row changes. Fyne v2.8 has no public access to the Table's scroller or column widths, so `internal/ui/fyneinternals.go` reads them via reflection/unsafe; `TestFyneInternals` fails if a Fyne upgrade changes them. Worth re-checking (or reporting upstream) on each Fyne upgrade.
- Fyne gives a mouse press to the innermost widget that handles clicks. Found at crosscheck 2: clickable header cells therefore kept the table from starting a column resize; header cells now forward mouse presses to the table.
- Under Fyne's test driver `fyne.Do` runs on the calling goroutine, so background work races with the test. Background work goes through `background()` (a goroutine in the app, inline in tests via `TestMain`).
- Fyne sends a click to one object only: table cells handle both tap (select row) and double tap (open), so the table's own cell selection is not used; whole-row highlighting is drawn by the cells.
- Fyne triggers main-menu shortcuts itself on Linux; shortcuts are only set on the menu items. F5 has no modifier, so it uses the canvas key handler (only when no text field has focus).

## Crosscheck log
Record each Spec ↔ Plan ↔ code crosscheck here.

| Date | Result | Follow-ups |
|------|--------|------------|
| 2026-10-29 | **#4, after M12 (user request).** 53 requirement IDs, 148 test cases, 133 tests (none skipped, all pass, also with `-race`): IDs, test names, test comments and `ManualTests.md` all consistent. Read-through of the M12 changes (manual bookmark order, drag & drop, indent field) found: FR-042 still listed the old bookmark menu (without Move up/down); `clearSelection` was unused outside tests (removed); M12 was listed above M11 in this table. No code defects found. | Manual pass by the user (53 checks) still pending |
| 2026-10-29 | **#3, final (M11).** 50 requirement IDs, 133 test cases, 119 tests: all cross-references consistent, every requirement has a test case, all automated tests pass (also with `-race`). Read-through found NFR-002 still broken in two places — every navigation checked its target (`stat` + open) on the UI thread, and a double-click did a `stat` there — so a hung network mount could still freeze the window; fixed (navigator split into Target/Commit, checks in the background, newer navigation wins). Also: the recent drop-down (FR-015) had no test (added); FR-001 named the size toggle differently from the menu; 2 test case IDs used in tests had no row. Manual checklist generated (`ManualTests.md`, 42 checks) with `make testdir` to set up a test folder. | Manual pass by the user; known, documented limits: startup (config load, start directory) and the final save on quit are synchronous (NFR-002) |
| 2026-10-29 | **#2, after M8 (main GUI done).** Checked 49 requirement IDs, 116 test cases and 104 tests; ID and test-name cross-references all consistent. Found and fixed: (1) dragging a header border could not resize columns (FR-020) — header cells swallowed the mouse press; now forwarded; (2) bookmark missing-checks ran `stat` on the UI thread on every redraw — a hung NFS mount would freeze the window (NFR-002); now cached and checked in the background; (3) 7 test comments with wrong/missing case IDs and one nonexistent ID (TC-023f); (4) spec layout sketch showed an old status-bar format; FR-002 now says F5 needs the path field unfocused; README gained a Usage section. | Settings, bookmarks, sort and toggles are not saved yet, so the startup "last directory" (FR-017) only works after M10; column resizing and all visual checks need a manual run |
| 2026-10-29 | **#1, after M5 (core done).** Checked all 46 requirement IDs, 94 test cases and 69 tests. Code: every core requirement is implemented and tested; no ID in the code is missing from the spec; core packages have no GUI dependency. Fixed: 8 test comments with outdated case IDs and 19 with none; 3 tests not listed in TestCases; 4 requirements with no test case (FR-001, FR-010, FR-072, NFR-003); NFR-003 now enforced by a test; spec text out of date for FR-001 (missing settings), FR-022 (size/time), FR-029 (`?` fields), FR-050 (what is counted); accepted assumptions moved from open questions into FR-002, FR-022, FR-040. | Q-1 startup directory, Q-2 recent-list size (Spec §7); tooltip widget needed (M8) |
