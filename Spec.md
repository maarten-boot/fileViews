# Specification — fileViews

_Status: draft v0.20 · Last updated: 2026-10-29_

## 1. Purpose
A simple hierarchical file and directory viewer with a graphical interface, laid out similar to Thunar.

## 2. Scope
- **In scope:** browsing directories, viewing file metadata (`ls -lF` style), a bookmark side pane with per-item colours and hover texts, navigation history, persistent settings.
- **Out of scope (for now):** file operations (copy, move, delete, rename, create), search, thumbnails/previews, multiple tabs.

## 3. Technology
| Item | Choice |
|------|--------|
| Language | Go |
| GUI toolkit | Fyne (v2) |
| Target platform | Linux first; avoid needless platform lock-in |
| Go module | `github.com/maarten-boot/fileViews` |

## 4. Window layout
```
+---------------------------------------------------------------+
| File  View  Go  Settings  Help                    (menu bar)  |
+---------------------------------------------------------------+
| [<] [>] [^] [⌂]  [ /home/mboot/projects           v ]  (path)  |
+-------------------+-------------------------------------------+
| /                 | Perms      Lnk Owner Group  Size Modified  Name |
|   mboot           | drwxr-xr-x  2 mboot mboot  4.0K 2026-10-29 src/  |
|     projects      | -rwxr-xr-x  1 mboot mboot  1.2M 2026-10-28 run*  |
|  (left pane)      |  (right pane)                              |
+-------------------+-------------------------------------------+
| 3 directories · 12 files · 2 hidden · 14.3M total · 1 selected (1.2M) · 120G free |
+---------------------------------------------------------------+
```
The split between left and right pane is user-resizable.

## 5. Functional requirements

### 5.1 Menu bar
- **FR-001** The menu bar has these menus:
  - File: Quit
  - View: Show hidden files (toggle), Show grid lines (toggle), Refresh
  - Go: Back, Forward, Up, Home
  - Settings: Columns…, Side pane indent…, Exact sizes (bytes) (toggle; off = human-readable), Contrast minimum…
  - Help: About
- **FR-002** Keyboard shortcuts: Ctrl+H show hidden files, Alt+Left back, Alt+Right forward, Alt+Up up, Alt+Home home, F5 refresh, Ctrl+Q quit. F5 works when the path field does not have keyboard focus (it is not a modifier shortcut, so the focused text field receives it).

### 5.2 Path bar and history
- **FR-010** The path bar shows the absolute path of the current directory.
- **FR-011** The path field is editable; Enter navigates to the typed path. A relative path is resolved against the current directory, a leading `~` means `$HOME`, surrounding spaces are ignored, and the path is cleaned (`a/b/..` → `a`). Symlinks in the path are kept as typed (not resolved). A path that does not exist, is not a directory or cannot be read shows an error and leaves the current directory unchanged.
- **FR-012** Back and Forward buttons move through the navigation history, like a browser: navigating somewhere new after going Back clears the Forward entries. Opening the current directory again adds no entry. If the Back/Forward target no longer exists or cannot be read, an error is shown and the position stays unchanged. The Back/Forward history lasts for the session only.
- **FR-013** An Up button goes to the parent directory; it is disabled at `/`.
- **FR-014** A Home button goes to `$HOME`.
- **FR-015** A dropdown on the path bar lists recently visited directories (most recent first, no duplicates, max 20; not configurable). Every successful navigation (typed path, Back, Forward, Up, Home, bookmark, double-click) moves that directory to the top. Selecting one navigates there.
- **FR-016** The recent-directories list is kept across sessions.
- **FR-017** Startup directory: a directory given on the command line (`fileviews /some/dir`; a relative path is taken from the working directory, `~` means `$HOME`; more than one argument prints a usage message) if it can be read; otherwise the last directory (FR-061) if it can still be read; otherwise `$HOME`; otherwise `/`. If the command-line directory cannot be read, the user gets an error and the next choice is used.

### 5.3 Right pane (directory listing)
- **FR-020** The right pane lists the entries of the current directory: files and directories, as a table with one row per entry. Clicking a row selects the whole row; Ctrl+click adds or removes a row, Shift+click selects the range from the last clicked row. The selection stays on the same entries when sorting, filtering or refreshing (entries no longer shown drop out of it), and is cleared when going to another directory. Columns start at a width measured from their header and a typical value (room for the sort arrow only on the sorted column; sorting by another column widens it if needed) and can be resized by dragging the header borders; Size and Links are right-aligned. The Name column stretches to fill the remaining width (at least 150 px), unless the user has dragged it to another width; so normally there is no sideways scrolling.
- **FR-032** Scrolling the right pane: a newly opened directory always starts at the top-left (a refresh keeps the position). The mouse wheel scrolls up and down only when there are more rows than fit; otherwise it does nothing. Sideways scrolling happens only with Shift+wheel or a sideways touchpad movement, and only when the columns are wider than the pane.
- **FR-021** Hidden entries (names starting with `.`) are hidden by default. View → Show hidden files (Ctrl+H) toggles them; the setting is kept across sessions.
- **FR-022** Columns, comparable to `ls -lF`:
  | Column | Content |
  |--------|---------|
  | Permissions | `drwxr-xr-x` style mode string |
  | Links | hard link count |
  | Owner | user name (numeric uid if unknown) |
  | Group | group name (numeric gid if unknown) |
  | Size | size, human-readable or exact bytes (FR-026) |
  | Modified | modification time in local time, `YYYY-MM-DD HH:MM` |
  | Name | name plus type marker |
- **FR-023** Type markers in the Name column: `/` directory, `*` executable regular file, `@` symlink, `|` named pipe (FIFO), `=` socket. For symlinks the target is shown as `name@ -> target`.
- **FR-024** Settings → Columns lets the user change the column order (up/down buttons per column) and show/hide columns (a checkbox per column, including hidden ones). The Name column cannot be hidden. Changes apply on "Apply"; the configuration is kept across sessions.
- **FR-025** Clicking a column header sorts by that column; clicking again reverses the order. Directories (and symlinks to directories) always come before other entries, whatever the sort column or direction. Names compare case-insensitively; ties are broken by name. Descending reverses the whole order within each group, including that tie-break. Default sort: by name, ascending.
- **FR-026** Sizes are shown human-readable by default (e.g. `4.2K`); a setting switches to exact bytes.
- **FR-027** Double-clicking a directory (or a symlink to a directory) navigates into it.
- **FR-028** Double-clicking a regular file (or a symlink to one) opens it with the system default application (`xdg-open` on Linux). Other entries (pipes, sockets, devices, broken links, entries removed since the listing) show an error instead.
- **FR-029** Entries that cannot be read (permission denied, broken symlink) are still listed with the information that is available; they must not stop the listing. Fields that cannot be read are shown as `?`. A broken symlink is listed with its target, and is not treated as a directory.
- **FR-030** The viewer is read-only: it does not change files or directories.
- **FR-031** The lines between table cells are light gray (`#E8E8E8`), lighter than separators elsewhere. View → Show grid lines turns them off and on (default on); the setting is kept across sessions.

### 5.4 Left pane (side pane / bookmarks)
- **FR-040** The left pane shows a list of bookmarked directories (not an expandable tree), in the user's order (FR-048). The indent (FR-044) shows each item's depth. User bookmarks are labelled with their base name. A config saved by an earlier version keeps its (path-sorted) order as the starting point.
- **FR-041** Two default items are always present and cannot be removed:
  - `/` — label `/`
  - `$HOME` — label is the basename of `$HOME` (e.g. `mboot`), target is the full path. If `$HOME` is `/`, there is only one default item.
- **FR-042** The user can add the current directory (or a selected directory in the right pane) as a bookmark, and remove user-added bookmarks, via a right-click menu. Bookmark right-click menu: Open, Move up, Move down (FR-048), Set colours…, Reset colours (only when colours are set), Edit hover text…, Add current directory, Remove bookmark (not for default items). Right-clicking a directory row in the right pane selects it and offers "Add to bookmarks". New bookmarks from the menus are added at the end. Adding a path that is already bookmarked is refused. Saved bookmarks with an empty or relative path, or a duplicate path, are ignored when loading (the first one wins); invalid saved colours are cleared.
- **FR-043** Single-clicking an item opens its directory in the right pane.
- **FR-044** Each item is indented by `depth × indent`, where depth is the number of path components from `/` (`/` = 0, `/home/mboot` = 2). The indent is configurable in Settings → Side pane indent: a slider and a number field that follow each other (whole numbers 1–25 px; anything else disables Apply); default 12 px.
- **FR-045** Each item can have a custom background and foreground colour, set via right-click → "Set colours…" (background and text fields as `#RRGGBB`, empty = default, each with a colour picker; live preview), with a "Reset colours" option. Colours are kept across sessions. The combination must meet the contrast minimum (FR-073–FR-075).
- **FR-046** Each item has a hover text (tooltip), shown in a small pop-up after the mouse rests on the item for about 0.6 s and hidden when it leaves. The default is the full path. The user can change it via right-click → "Edit hover text…"; an empty text goes back to the default. Hover texts are kept across sessions.
- **FR-047** A bookmark whose target no longer exists is still shown, in italics with the suffix ` (missing)`; clicking it shows an error. On a pure white or pure black background (default or chosen) its text is also gray (FR-071). On any other background it keeps its own text colour (no gray). The text marker is always shown, so the state never depends on colour alone. Whether a bookmark is missing is checked in the background (NFR-002) when bookmarks change and after every navigation or refresh, so the marker can lag a moment behind the file system.
- **FR-048** Bookmarks can be reordered, all of them including the default items: right-click → Move up / Move down (disabled at the ends), or drag a bookmark to another place in the pane; a line shows where it will land. The order is kept across sessions.
- **FR-049** Directories (and links to directories) can be dragged from the right pane onto the bookmark pane; a line shows where they will be added. Dragging a selected row drags all selected directories (files in the selection are ignored); dragging a file does nothing. A directory that is already a bookmark is never placed a second time: while dragging, the floating label says so ("already a bookmark", or how many are already bookmarked), no drop line is shown when nothing new would be added, and on drop only the new ones are added. Dropping outside the bookmark pane does nothing. Only bookmarks are created; the directory itself is never moved (FR-030).

### 5.5 Status bar (bottom line)
- **FR-050** The status bar shows, for the current directory:
  - number of directories shown (including symlinks to directories)
  - number of other entries shown ("files")
  - number of hidden entries (counted even when hidden entries are not shown)
  - total size of the shown non-directory entries
  - total size of the selected entries (when there is a selection)
  - free space on the filesystem of the current directory
- **FR-051** The status bar updates on navigation, refresh, selection change and toggling hidden files.

### 5.6 Persistence
- **FR-060** All settings and state are saved in `$XDG_CONFIG_HOME/fileViews/config.json` (default `~/.config/fileViews/config.json`).
- **FR-061** Saved: bookmarks (with colours and hover texts), column order and visibility, sort column and direction, indent size, show-hidden setting, grid lines setting, size format, contrast minimum, recent directories, last directory, window size and pane split. The file has a format `version` field (currently 1).
- **FR-062** Loading the config file:
  - missing file: defaults, no warning
  - invalid JSON: the file is renamed to `config.json.bak` (replacing an older `.bak`), defaults are used, and the user gets a warning
  - file exists but cannot be read (e.g. permissions): defaults are used, the user gets a warning, and the app does **not** save during that session, so the file is never overwritten with defaults
- **FR-063** The config is written atomically (write a temporary file in the same directory, then rename it). A failed save leaves the old file intact and no temporary file behind. The file is created with mode 0644; the directory is created if needed.
- **FR-065** When the config is saved:
  - right after every change to something in FR-061 (toggles, sorting, bookmark edits, and every navigation for the recent list and last directory), written in the background so a slow disk cannot freeze the window (NFR-002); a snapshot is never overwritten by an older one
  - on quit (File → Quit, Ctrl+Q or closing the window), together with the window size and pane split
  - a save error is shown once per session; after an unreadable config (FR-062) nothing is saved and no further error is shown
- **FR-064** Missing or out-of-range values are replaced by defaults, unknown fields and column names are ignored: indent 1–25 px (default 12), contrast minimum 3–7 (default 4.5), window 1000×650 if unset, pane split between 0 and 1 (default 0.25), recent list cleaned as in FR-015.

### 5.7 Appearance
- **FR-070** Default colours: white background, black text, in all panes and bars. The app uses a light theme regardless of the desktop's dark-mode setting.
- **FR-071** Gray text ("greyed out") is only used on a pure white (`#FFFFFF`) or pure black (`#000000`) background, and must meet the contrast minimum (on white: no lighter than about `#767676`; on black: no darker than about `#757575`). Never gray on gray. On other backgrounds, state is shown with text markers, italics, bold or icons instead.
- **FR-072** User-chosen bookmark colours (FR-045) override the defaults only for that item.
- **FR-073** Contrast is measured as the WCAG 2 contrast ratio, from 1:1 (same colour) to 21:1 (black on white). Every text/background combination in the app must be at or above the minimum ratio.
- **FR-074** The minimum ratio defaults to **4.5:1** and can be changed in Settings between 3:1 and 7:1 (7:1 is WCAG AAA; above about 7.8:1 coloured backgrounds cannot pass with black or white text). Examples on white: yellow `#FFFF00` ≈ 1.07:1 and light green `#90EE90` ≈ 1.4:1 are rejected; `#767676` ≈ 4.54:1 is allowed.
- **FR-075** The bookmark colour dialog shows the contrast ratio live and refuses to save a combination below the minimum. It then offers black or white text, whichever gives the higher contrast with the chosen background.
- **FR-076** Saved bookmark colours are checked at startup and when the minimum changes (Settings → Contrast minimum: slider 3–7 in steps of 0.5; applying it also changes the theme gray, FR-071). A failing combination is drawn with black or white text (whichever contrasts more) until the user changes it; the saved colours are kept.
- **FR-077** The app's own theme colours (including disabled buttons, placeholder text and the status bar) also meet the minimum; the custom light theme overrides Fyne defaults that do not:
  - normal text black; window, input fields, menus, pop-ups and disabled buttons pure white (so gray text can only appear on white)
  - disabled and placeholder text: the gray from FR-071 for the current minimum
  - buttons and table headers light gray (`#F0F0F0`) with black text only
  - coloured backgrounds (primary blue, error red, success green, warning orange) with white text at 7:1 or better

## 6. Non-functional requirements
- **NFR-001** A directory with 10,000 entries lists in under 1 second on a typical desktop.
- **NFR-002** The UI stays responsive while using the file system: directory listings, navigation targets, double-clicked entries, bookmark directories and config saves after a change are all checked or written off the UI thread (a hung network mount must not freeze the window). Exceptions: loading the config and choosing the startup directory happen before the window appears, and the final save on quit is synchronous so no settings are lost.
- **NFR-003** Core logic (listing, formatting, history, config, bookmarks, colour contrast) is in packages without GUI dependencies, so it can be unit-tested headless.
- **NFR-004** Builds with `go build` (`make build`); Go 1.26+, Fyne v2.8+.

## 7. Open questions
_None._ (Q-1 → FR-017, Q-2 → FR-015, Q-3 → FR-074, Q-4 → FR-020; all resolved 2026-10-29.)
