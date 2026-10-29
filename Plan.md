# Plan — fileViews

_Status: draft v0.3 · Last updated: 2026-10-29_

## Approach
Build the non-GUI core first, with unit tests (NFR-003), then add the Fyne GUI on top of it, one area at a time.

## Proposed package layout
```
cmd/fileviews/main.go     entry point
internal/listing/         read directory, entries, ls -lF formatting, sorting, stats
internal/history/         back/forward history, recent directories
internal/bookmarks/       side pane items, depth, colours, hover text
internal/config/          load/save config.json (atomic, defaults, broken-file handling)
internal/ui/              Fyne windows and widgets
```

## Milestones
| # | Milestone | Spec refs | Status |
|---|-----------|-----------|--------|
| 0 | Gather scenario, draft Spec | — | done |
| 1 | Project scaffolding: `go mod init`, layout, Fyne dependency, empty window, Makefile / `go test ./...` | NFR-004 | done (Fyne v2.8.2) |
| 2 | Core: directory listing, `ls -lF` formatting, type markers, sorting, hidden filter, stats, free space | FR-020–026, FR-029, FR-050 | done |
| 3 | Core: navigation history and recent directories | FR-012–016 | todo |
| 4 | Core: config load/save | FR-060–063 | todo |
| 5 | Core: bookmarks (defaults, depth, colours, hover text) | FR-040–047 | todo |
| 6 | GUI: main window layout — light theme, menu bar, path bar, split panes, status bar | FR-001, FR-010–015, FR-070–071, §4 | todo |
| 7 | GUI: right pane table — columns, sorting, double-click, hidden toggle | FR-020–028 | todo |
| 8 | GUI: left pane — indent, colours, tooltips, context menu | FR-040–047 | todo |
| 9 | GUI: Settings dialogs — column order/visibility, indent, size format | FR-024, FR-026, FR-044 | todo |
| 10 | Persistence wiring, window state, error handling, performance check | FR-061, NFR-001, NFR-002 | todo |
| 11 | Final crosscheck Spec ↔ Plan ↔ code, manual test pass | all | todo |

**Crosscheck points:** after milestones 5 (core done), 8 (main GUI done) and 11.

## Risks / notes
- Fyne's standard widgets may not support per-item background colours or tooltips directly; custom widgets may be needed (milestone 8). Tooltip support in recent Fyne versions still to be verified.
- No display in the assistant's shell: GUI code is verified with Fyne's headless `test` driver; visual checks are done manually by the user.
- Owner/group lookup and link count use `syscall.Stat_t` (Unix only); keep that in a file with a build tag.
- Free disk space uses `statfs` (Unix only); same approach. Both live in `internal/listing/stat_unix.go`, with a stub in `stat_other.go`.
- Fyne follows the desktop dark mode by default; FR-070 needs a custom theme that forces the light variant (milestone 6).

## Crosscheck log
Record each Spec ↔ Plan ↔ code crosscheck here.

| Date | Result | Follow-ups |
|------|--------|------------|
