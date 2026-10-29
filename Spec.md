# Specification — fileViews

_Status: draft v0.4 · Last updated: 2026-10-29_

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
| 3 dirs · 12 files (2 hidden) · 14.3M · selected: 1.2M · free: 120G |
+---------------------------------------------------------------+
```
The split between left and right pane is user-resizable.

## 5. Functional requirements

### 5.1 Menu bar
- **FR-001** The menu bar has these menus:
  - File: Quit
  - View: Show hidden files (toggle), Refresh
  - Go: Back, Forward, Up, Home
  - Settings: Columns…, Side pane indent…
  - Help: About

### 5.2 Path bar and history
- **FR-010** The path bar shows the absolute path of the current directory.
- **FR-011** The path field is editable; Enter navigates to the typed path. A relative path is resolved against the current directory, a leading `~` means `$HOME`, surrounding spaces are ignored, and the path is cleaned (`a/b/..` → `a`). Symlinks in the path are kept as typed (not resolved). A path that does not exist, is not a directory or cannot be read shows an error and leaves the current directory unchanged.
- **FR-012** Back and Forward buttons move through the navigation history, like a browser: navigating somewhere new after going Back clears the Forward entries. Opening the current directory again adds no entry. If the Back/Forward target no longer exists or cannot be read, an error is shown and the position stays unchanged. The Back/Forward history lasts for the session only.
- **FR-013** An Up button goes to the parent directory; it is disabled at `/`.
- **FR-014** A Home button goes to `$HOME`.
- **FR-015** A dropdown on the path bar lists recently visited directories (most recent first, no duplicates, max 20 by default). Every successful navigation (typed path, Back, Forward, Up, Home, bookmark, double-click) moves that directory to the top. Selecting one navigates there.
- **FR-016** The recent-directories list is kept across sessions.

### 5.3 Right pane (directory listing)
- **FR-020** The right pane lists the entries of the current directory: files and directories.
- **FR-021** Hidden entries (names starting with `.`) are hidden by default. View → Show hidden files (Ctrl+H) toggles them; the setting is kept across sessions.
- **FR-022** Columns, comparable to `ls -lF`:
  | Column | Content |
  |--------|---------|
  | Permissions | `drwxr-xr-x` style mode string |
  | Links | hard link count |
  | Owner | user name (numeric uid if unknown) |
  | Group | group name (numeric gid if unknown) |
  | Size | size in bytes |
  | Modified | modification time |
  | Name | name plus type marker |
- **FR-023** Type markers in the Name column: `/` directory, `*` executable regular file, `@` symlink, `|` named pipe (FIFO), `=` socket. For symlinks the target is shown as `name@ -> target`.
- **FR-024** Settings → Columns lets the user change the column order and show/hide columns. The Name column cannot be hidden. The configuration is kept across sessions.
- **FR-025** Clicking a column header sorts by that column; clicking again reverses the order. Directories (and symlinks to directories) always come before other entries, whatever the sort column or direction. Names compare case-insensitively; ties are broken by name. Default sort: by name, ascending.
- **FR-026** Sizes are shown human-readable by default (e.g. `4.2K`); a setting switches to exact bytes.
- **FR-027** Double-clicking a directory (or a symlink to a directory) navigates into it.
- **FR-028** Double-clicking a file opens it with the system default application (`xdg-open` on Linux).
- **FR-029** Entries that cannot be read (permission denied, broken symlink) are still listed with the information that is available; they must not stop the listing.
- **FR-030** The viewer is read-only: it does not change files or directories.

### 5.4 Left pane (side pane / bookmarks)
- **FR-040** The left pane shows a list of bookmarked directories (not an expandable tree).
- **FR-041** Two default items are always present and cannot be removed:
  - `/` — label `/`
  - `$HOME` — label is the basename of `$HOME` (e.g. `mboot`), target is the full path
- **FR-042** The user can add the current directory (or a selected directory in the right pane) as a bookmark, and remove user-added bookmarks, via a right-click menu.
- **FR-043** Single-clicking an item opens its directory in the right pane.
- **FR-044** Each item is indented by `depth × indent`, where depth is the number of path components from `/` (`/` = 0, `/home/mboot` = 2). The indent is configurable in Settings; default 12 px.
- **FR-045** Each item can have a custom background and foreground colour, set via right-click → "Set colours…", with a "Reset" option. Colours are kept across sessions. The combination must meet the contrast minimum (FR-073–FR-075).
- **FR-046** Each item has a hover text (tooltip). The default is the full path. The user can change it via right-click → "Edit hover text…"; an empty text goes back to the default. Hover texts are kept across sessions.
- **FR-047** A bookmark whose target no longer exists is still shown, in italics with the suffix ` (missing)`; clicking it shows an error. On a pure white or pure black background its text is also gray (FR-071). On a user-chosen background it keeps its own text colour (no gray). The text marker is always shown, so the state never depends on colour alone.

### 5.5 Status bar (bottom line)
- **FR-050** The status bar shows, for the current directory:
  - number of directories
  - number of files
  - number of hidden entries (counted even when hidden entries are not shown)
  - total size of the listed files
  - total size of the selected entries (when there is a selection)
  - free space on the filesystem of the current directory
- **FR-051** The status bar updates on navigation, refresh, selection change and toggling hidden files.

### 5.6 Persistence
- **FR-060** All settings and state are saved in `$XDG_CONFIG_HOME/fileViews/config.json` (default `~/.config/fileViews/config.json`).
- **FR-061** Saved: bookmarks (with colours and hover texts), column order and visibility, sort column and direction, indent size, show-hidden setting, size format, recent directories, last directory, window size and pane split.
- **FR-062** A missing config file means defaults are used. A broken (unreadable) config file is renamed to `config.json.bak`, defaults are used, and the user gets a warning.
- **FR-063** The config is written atomically (write a temporary file, then rename it).

### 5.7 Appearance
- **FR-070** Default colours: white background, black text, in all panes and bars. The app uses a light theme regardless of the desktop's dark-mode setting.
- **FR-071** Gray text ("greyed out") is only used on a pure white (`#FFFFFF`) or pure black (`#000000`) background, and must meet the contrast minimum (on white: no lighter than about `#767676`; on black: no darker than about `#757575`). Never gray on gray. On other backgrounds, state is shown with text markers, italics, bold or icons instead.
- **FR-072** User-chosen bookmark colours (FR-045) override the defaults only for that item.
- **FR-073** Contrast is measured as the WCAG 2 contrast ratio, from 1:1 (same colour) to 21:1 (black on white). Every text/background combination in the app must be at or above the minimum ratio.
- **FR-074** The minimum ratio defaults to **4.5:1** and can be changed in Settings; it cannot be set below 3:1. Examples on white: yellow `#FFFF00` ≈ 1.07:1 and light green `#90EE90` ≈ 1.4:1 are rejected; `#767676` ≈ 4.54:1 is allowed.
- **FR-075** The bookmark colour dialog shows the contrast ratio live and refuses to save a combination below the minimum. It then offers black or white text, whichever gives the higher contrast with the chosen background.
- **FR-076** Saved bookmark colours are checked at startup and when the minimum changes. A failing combination is drawn with black or white text (whichever contrasts more) until the user changes it; the saved colours are kept.
- **FR-077** The app's own theme colours (including disabled buttons, placeholder text and the status bar) also meet the minimum; the custom light theme overrides Fyne defaults that do not.

## 6. Non-functional requirements
- **NFR-001** A directory with 10,000 entries lists in under 1 second on a typical desktop.
- **NFR-002** The UI stays responsive while reading a directory: directories are read off the UI thread.
- **NFR-003** Core logic (listing, formatting, history, config, bookmarks, colour contrast) is in packages without GUI dependencies, so it can be unit-tested headless.
- **NFR-004** Builds with `go build` (`make build`); Go 1.26+, Fyne v2.8+.

## 7. Open questions
- Keyboard shortcuts beyond Ctrl+H? *Assumed: Alt+Left / Alt+Right (history), Alt+Up, Alt+Home, F5 (refresh).*
- Modified time format? *Assumed: `2006-01-02 15:04`.*
- Should bookmarks be reorderable (drag & drop)? *Assumed: not in v1.*
