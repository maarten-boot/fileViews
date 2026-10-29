# Test Cases — fileViews

_Last updated: 2026-10-29_

Each test case links to the requirement it verifies. **Auto** = Go unit test; **Manual** = GUI check.
A ✔ with a test name means the automated test exists and passes.

## Menu bar
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-001 | FR-001 | Menus present | Open each menu | File, View, Go, Settings, Help with the listed items | Auto ✔ `TestMenus` + Manual |
| TC-002 | FR-002 | Keyboard shortcuts | Press each shortcut | Matching action runs | Auto (menu shortcuts ✔ `TestMenus`, F5 ✔ `TestF5Refresh`) + Manual |

## Right pane / listing (`internal/listing`)
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-020 | FR-020 | List directory | Temp dir with `a.txt`, `sub/` | Both entries listed | Auto ✔ `TestReadListsFilesAndDirs`, `TestNavigateShowsDirectory` |
| TC-020b | FR-020 | Whole-row selection | Click a cell of `a.txt`; sort by name descending; show hidden; refresh; go to `sub` | Whole row highlighted; selection kept through sort, filter and refresh; cleared on navigation | Auto ✔ `TestTableSelection` + Manual |
| TC-020c | FR-020 | Multi-select | Click b; Ctrl+click d; Ctrl+click b; Shift+click a; Shift+click e; click c; anchor gone | `b`; `b,d`; `d`; `a,b`; `b..e`; `c`; Shift acts as plain click | Auto ✔ `TestSelectionClicks`, `TestSelectionShiftWithoutAnchor`, `TestSelectionPrune`, `TestTableMultiSelect` + Manual |
| TC-021a | FR-021 | Hidden filtered by default | Dir with `.x`, `y` | Only `y` listed | Auto ✔ `TestFilterHidden` |
| TC-021b | FR-021 | Show hidden | Same dir, show-hidden on | `.x` and `y` listed | Auto ✔ `TestFilterHidden`, `TestToggleHidden` |
| TC-022 | FR-022 | Permission string | File with mode 0754; also dir, link, pipe, socket, devices, setuid/setgid/sticky | `-rwxr-xr--` etc. | Auto ✔ `TestPermString`, `TestReadPermissions` |
| TC-022c | FR-022 | Table cells | `a.txt` mode 0640, 100 bytes | `-rw-r-----`, `1`, owner, group, `100`, `YYYY-MM-DD HH:MM`, `a.txt`; Size and Links right-aligned | Auto ✔ `TestTableCells` + Manual |
| TC-022b | FR-022 | Unknown uid/gid | uid 4000000000 | Shown numerically | Auto ✔ `TestUnknownIDsAreNumeric` |
| TC-023a | FR-023 | Directory marker | `sub/` | Name `sub/` | Auto ✔ `TestDisplayNameMarkers` |
| TC-023b | FR-023 | Executable marker | File mode 0755 | Name ends in `*` | Auto ✔ `TestDisplayNameMarkers` |
| TC-023c | FR-023 | Symlink marker + target | `l -> a.txt` | `l@ -> a.txt` | Auto ✔ `TestDisplayNameMarkers` |
| TC-023d | FR-023 | FIFO marker | `mkfifo p` | `p\|` | Auto ✔ `TestFIFO`, `TestDisplayNameSpecial` |
| TC-023e | FR-023 | Socket marker | Unix socket `s` | `s=` | Auto ✔ `TestSocket`, `TestDisplayNameSpecial` |
| TC-024a | FR-024 | Column order applied | Order `[Name, Size, Perms]` | Rows formatted in that order | Auto ✔ `TestColumnOrder` |
| TC-024b | FR-024 | Name cannot be hidden | Config hides Name | Name still shown | Auto ✔ `TestColumnLayoutNormalises` |
| TC-024e | FR-024 | Table follows layout | Columns `[Name, Size]` | 2 columns, header `Size` second | Auto ✔ `TestTableColumns` |
| TC-024c | FR-024 | Stale/bad column config | Duplicates, unknown ids, missing columns | Dropped / appended | Auto ✔ `TestColumnLayoutNormalises` |
| TC-024d | FR-024 | Column names round trip | Every column name → column → name; `Bogus` | Same column; `Bogus` rejected | Auto ✔ `TestParseColumn` |
| TC-025a | FR-025 | Default sort | `b`, `A/`, `a`, `B` | `A/`, `a`, `B`, `b` | Auto ✔ `TestDefaultSort` |
| TC-025b | FR-025 | Sort by size, reverse | Files of 1, 3, 2 bytes + a dir | dir, then 3, 2, 1 | Auto ✔ `TestSortBySizeDescending` |
| TC-025e | FR-025 | Sort by clicking headers | Click Size, Size again, Name | Size ascending (dirs first); descending (whole order reversed, dirs first); back to name; arrow icon on sorted column | Auto ✔ `TestTableSortByHeader` + Manual |
| TC-025c | FR-025 | Ties broken by name | Equal mtimes | Ordered by name | Auto ✔ `TestSortTiesByName` |
| TC-025d | FR-025 | Symlink to dir sorts as dir | `ls -> sub` | Treated as directory | Auto ✔ `TestSymlinkToDir` |
| TC-026a | FR-026 | Human-readable size | 4300 bytes | `4.2K` | Auto ✔ `TestSizeFormat` |
| TC-026b | FR-026 | Exact size | 4300 bytes, exact mode | `4300` | Auto ✔ `TestSizeFormat`, `TestStatusSelectionAndExactSizes` |
| TC-026c | — | Modified time format | 2026-10-29 11:04:59 | `2026-10-29 11:04` | Auto ✔ `TestModifiedFormat` |
| TC-027 | FR-027 | Double-click directory | Double-click `sub/`; double-click `link -> sub` | Right pane shows `sub`; shows `link` | Auto ✔ `TestTableOpenDirectory` + Manual |
| TC-028 | FR-028 | Double-click file | Double-click `a.txt`; a FIFO; a file removed since listing; opener fails | `xdg-open a.txt`; others: error, nothing opened | Auto ✔ `TestTableOpenFile` + Manual |
| TC-029a | FR-029 | Broken symlink | `l -> missing` | Listed; rest of listing unaffected | Auto ✔ `TestBrokenSymlink` |
| TC-029b | FR-029 | Unreadable subdir | `chmod 000 sub` | `sub/` listed | Auto ✔ `TestUnreadableSubdir` |
| TC-029c | FR-029 | Unreadable metadata | Entry with stat error | Fields `?`, name shown | Auto ✔ `TestUnreadableFieldsShowQuestionMark` |
| TC-031 | FR-031 | Grid lines | Default; View → Show grid lines; new window with saved setting | Light gray `#E8E8E8` only in the table; hidden after toggle; setting applied | Auto ✔ `TestTableGridLines` + Manual |
| TC-030 | FR-030 | Read-only | Browse around | No file is modified | Manual |

## Path bar / history (`internal/history`)
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-010 | FR-010 | Path bar shows current path | Navigate to `/tmp` | Path field shows `/tmp` | Auto ✔ `TestNavigateShowsDirectory` + Manual |
| TC-011a | FR-011 | Type valid path | Enter `/tmp` | Navigates to `/tmp` | Auto ✔ `TestNavigateShowsDirectory` + Manual |
| TC-011b | FR-011 | Type invalid path | Missing path, a file, empty input | Error; directory and recent list unchanged | Auto ✔ `TestGoInvalidKeepsCurrent`, `TestReadMissingDir`, `TestInvalidPathKeepsCurrent` + Manual |
| TC-011c | FR-011 | Unreadable directory | `chmod 000 a`, go to `a` | Permission error; directory unchanged | Auto ✔ `TestGoUnreadableDir` |
| TC-011d | FR-011 | Path resolution | `a`, `b/..`, `~`, `~/a/b`, spaces around path | Resolved, cleaned absolute paths | Auto ✔ `TestGoResolvesPaths` |
| TC-011e | FR-011 | Relative path with no current dir | `a` at startup | Error | Auto ✔ `TestRelativeWithoutCurrent` |
| TC-012a | FR-012 | Back/forward | Visit A, B, C; Back; Back; Forward | Current = B | Auto ✔ `TestBackForward`, `TestNavigatorBackForward`, `TestButtonsFollowHistory` |
| TC-012b | FR-012 | New visit clears forward | Visit A, B; Back; visit C | Forward is empty | Auto ✔ `TestVisitClearsForward` |
| TC-012c | FR-012 | Revisit current adds nothing | Visit A, A | No Back entry | Auto ✔ `TestHistoryEdges` |
| TC-012d | FR-012 | Back to removed dir | Visit B, root; remove B; Back | Error; still at root, Back still possible | Auto ✔ `TestBackToRemovedDir` |
| TC-013 | FR-013 | Up, and Up at root | `/x/a/b` → Up; at `/` | `/x/a`; at `/` Up disabled, error, no move | Auto ✔ `TestParent`, `TestUpAndHome`, `TestButtonsFollowHistory` |
| TC-014 | FR-014 | Home | Home | `$HOME` | Auto ✔ `TestUpAndHome`, `TestButtonsFollowHistory` |
| TC-015a | FR-015 | Recent: no duplicates | Visit A, B, A | Recent = `[A, B]` | Auto ✔ `TestRecentNoDuplicates`, `TestRecentFollowsNavigation` |
| TC-015b | FR-015 | Recent: max size | Visit 25 dirs | 20 most recent kept | Auto ✔ `TestRecentMax` |
| TC-015c | FR-015 | Recent updated by Back | A, B, C; Back | Recent starts with B | Auto ✔ `TestNavigatorBackForward` |
| TC-017a | FR-017 | Command-line directory | `fileviews /x/sub`; relative `sub` from `/x`; `~/sub` | Starts in `/x/sub` | Auto ✔ `TestStartupDir` |
| TC-017b | FR-017 | Last directory | No argument, last = `/x/sub` | Starts in `/x/sub` | Auto ✔ `TestStartupDir` |
| TC-017c | FR-017 | Fallbacks | Last gone → home; home gone → `/` | `$HOME`; `/` | Auto ✔ `TestStartupDir` |
| TC-017d | FR-017 | Bad command-line directory | `fileviews /gone` | Error shown, next choice used | Auto ✔ `TestStartupDir` + Manual |
| TC-016 | FR-016 | Recent kept across sessions | Visit A, restart | A in dropdown | Auto ✔ `TestRecentFromConfig`, `TestRoundTrip` + Manual |

## Left pane / bookmarks (`internal/bookmarks`)
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-041a | FR-041 | Default items | Empty config, `HOME=/home/mboot` | Items `/` and `mboot`, both default | Auto ✔ `TestDefaultItems` |
| TC-041b | FR-041 | Defaults not removable | Remove `/`, remove home | Refused | Auto ✔ `TestAddRemove` |
| TC-041c | FR-041 | Home is `/` | `HOME=/` | One item `/` | Auto ✔ `TestHomeIsRoot` |
| TC-042a | FR-042 | Add/remove bookmark | Add `/tmp`, add `/tmp/` again, add `tmp`, remove twice | Added; duplicate refused; relative refused; removed; then not found | Auto ✔ `TestAddRemove` |
| TC-042b | FR-042 | Saved data cleaned up | Relative, empty, duplicate paths, bad colour | Ignored / first wins / colour cleared | Auto ✔ `TestNewFromSavedCleansUp` |
| TC-040 | FR-040 | Tree-like order | `/home/mboot-old`, `/home/mboot/projects`, `/etc`, `/home` | `/`, `/etc`, `/home`, `/home/mboot`, `/home/mboot/projects`, `/home/mboot-old` | Auto ✔ `TestOrder` |
| TC-043 | FR-043 | Click opens directory | Click `mboot` | Right pane shows `$HOME` | Auto ✔ `TestBookmarkClickNavigates` + Manual |
| TC-041d | FR-041 | Default items in menu | Right-click `/` | Remove bookmark and Reset colours disabled | Auto ✔ `TestBookmarkMenu` + Manual |
| TC-042c | FR-042 | Bookmark menu actions | Add current directory (twice), reset colours, open, remove | Added once (duplicate: error); reset; opened; removed | Auto ✔ `TestBookmarkMenu` + Manual |
| TC-042d | FR-042 | Bookmark from right pane | Right-click `sub/`; right-click a file | `sub` selected, "Add to bookmarks" adds it; no menu for files | Auto ✔ `TestRowMenuAddBookmark` + Manual |
| TC-044c | FR-044 | Indent drawn | Home at depth d, indent 12 and 20 | Spacer width 12·d, 20·d; `/` none | Auto ✔ `TestBookmarkIndent` + Manual |
| TC-045d | FR-045 | Colours drawn | White on navy for home | Home drawn white on navy; `/` stays black on white | Auto ✔ `TestBookmarkStyle` + Manual |
| TC-046d | FR-046 | Hover text pop-up | Hover home; leave; custom text; leave before delay | Full path shown; hidden; "work" shown; nothing shown | Auto ✔ `TestBookmarkTooltip`, `TestBookmarkTooltipCancelled` + Manual |
| TC-047c | FR-047 | Missing bookmark drawn and clicked | Bookmark removed dir | `gone (missing)`, italic, `#767676`; click: error, directory unchanged | Auto ✔ `TestBookmarkStyle` + Manual |
| TC-044a | FR-044 | Depth | `/`, `/home`, `/home/mboot`, `/a/b/c/` | 0, 1, 2, 3 | Auto ✔ `TestDepthAndIndent` |
| TC-044b | FR-044 | Indent | Depth 2, indent 12 | 24 px | Auto ✔ `TestDepthAndIndent` |
| TC-045a | FR-045 | Set and reset colours | Navy on white; reset | Stored as `#RRGGBB`; reset to defaults | Auto ✔ `TestSetColours` |
| TC-045b | FR-045 | Colours kept | Set colours, save config, reload | Same items and colours | Auto ✔ `TestSavedRoundTrip` |
| TC-045c | FR-045 | Colour notation | `#000000`, `abcdef`, `#FFF`, `#GGGGGG` | Parsed and written as `#RRGGBB`; bad values rejected | Auto ✔ `TestHexRoundTrip` |
| TC-046a | FR-046 | Default hover text | No custom text | Full path | Auto ✔ `TestHoverText` |
| TC-046b | FR-046 | Custom hover text kept | Set "work", save, reload | "work" | Auto ✔ `TestHoverText`, `TestSavedRoundTrip` |
| TC-046c | FR-046 | Empty text resets | Set "   " | Full path | Auto ✔ `TestHoverText` |
| TC-047a | FR-047 | Missing detection | Existing dir; removed dir; a file | false; true; true | Auto ✔ `TestMissing` |
| TC-047b | FR-047 | Missing style | Default white; black bg; light yellow bg with navy text | `gone (missing)`, italic; text `#767676`; `#757575`; navy — all pass 4.5:1 | Auto ✔ `TestStyleMissing` + Manual |

## Status bar
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-050a | FR-050 | Counts | 2 dirs, 3 files (2 hidden), with and without show-hidden | Hidden count always 2; dirs/files follow the filter | Auto ✔ `TestComputeStats` |
| TC-050b | FR-050 | Total size | Files of 100 + 200 bytes | 300 | Auto ✔ `TestComputeStats` |
| TC-050c | FR-050 | Selection size | Select a 100-byte file and a dir | selected 100 | Auto ✔ `TestSelectionSize`, `TestStatusSelectionAndExactSizes` |
| TC-050e | FR-050 | Status bar text | Counts with singular/plural, sizes, selection, free space, exact mode | `1 directory · 1 file · 2 hidden · 4.2K total · 5.0G free` etc. | Auto ✔ `TestStatusText` |
| TC-050f | FR-050 | Status for multi-select | Select `a.txt` (100) and `b.txt` (200), then a range with `sub/` | `2 selected (300)`, `3 selected (300)` | Auto ✔ `TestTableMultiSelect` |
| TC-050d | FR-050 | Free space | Temp dir; missing path | Value; error | Auto ✔ `TestFreeSpace` |
| TC-051 | FR-051 | Updates on toggle hidden | Ctrl+H | Counts/size update | Auto ✔ `TestToggleHidden` + Manual |

## Persistence (`internal/config`)
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-060 | FR-060 | Config path | `XDG_CONFIG_HOME=/x`; unset with `HOME=/home/u` | `/x/fileViews/config.json`; `/home/u/.config/fileViews/config.json` | Auto ✔ `TestDefaultPath` |
| TC-061 | FR-061 | All settings round trip | Save every field, reload | Identical config | Auto ✔ `TestRoundTrip` |
| TC-062a | FR-062 | Missing config | No file | Defaults, no warning | Auto ✔ `TestLoadMissing`, `TestDefaults` |
| TC-062b | FR-062 | Broken config | File with `{garbage` | Renamed to `.bak`, defaults, warning; saving works afterwards | Auto ✔ `TestLoadBroken` |
| TC-062c | FR-062 | Unreadable config | File with mode 000 | Defaults, warning; Save refused; file unchanged | Auto ✔ `TestLoadUnreadableBlocksSave` |
| TC-063a | FR-063 | Atomic write | Save 3 times into a new directory | Only `config.json` (0644) remains, last values | Auto ✔ `TestSaveAtomic` |
| TC-063b | FR-063 | Failed save | Read-only config directory | Error; old file unchanged; no temp file | Auto ✔ `TestSaveFailureKeepsOld` |
| TC-064 | FR-064 | Bad values normalised | Negative indent, contrast 1, split 1.5, unknown sort/columns, duplicate recent, unknown field | Defaults/limits applied, unknowns ignored | Auto ✔ `TestNormalize` |

## Non-functional
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-N01 | NFR-001 | Large directory | 10,000 files: read + sort + format | < 1 s (measured ≈ 17 ms) | Bench ✔ `BenchmarkListLargeDir` |
| TC-N03 | NFR-003 | Core has no GUI dependency | `go list -deps` of listing, history, bookmarks, config, contrast | No `fyne.io/…` or `internal/ui` | Auto ✔ `TestCoreHasNoGUIDeps` |
| TC-N02 | NFR-002 | Responsive UI | Open a large directory | Window stays responsive | Auto ✔ `TestStaleLoadDropped` + Manual |

## Build / smoke
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-N04a | NFR-004 | Build | `make build` | `fileviews` binary, no errors | Auto |
| TC-N04b | NFR-004 | Main window created (headless) | Build window with test app | Title `fileViews`, content set | Auto ✔ `TestNewMainWindow` |
| TC-N04c | NFR-004 | Window opens on desktop | `make run` | Window "fileViews" appears, 1000×650 | Manual ✔ (user, 2026-10-29) |

## Appearance
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-070a | FR-070 | Light theme by default | Start app; bookmark without colours | White background, black text everywhere | Auto ✔ `TestStyleColours`, `TestThemeContrast`, `TestThemeIgnoresDarkMode` + Manual |
| TC-070b | FR-070 | Ignores desktop dark mode | Desktop in dark mode, start app | Still white background, black text | Auto ✔ `TestThemeIgnoresDarkMode` + Manual |
| TC-071a | FR-071 | Gray on pure white / black | Min 4.5 | `#767676` on white, `#757575` on black; each passes the minimum | Auto ✔ `TestGray` |
| TC-071b | FR-071 | No gray on other backgrounds | `#FEFEFE`, `#808080`, `#FFFFE0` | Gray refused | Auto ✔ `TestGray` |
| TC-071c | FR-071 | No gray-on-gray in the GUI | Inspect missing bookmark, disabled buttons, status bar | No gray text on gray background | Auto ✔ `TestThemeContrast` (theme) + Manual |
| TC-072 | FR-072 | Colours only affect their item | Colour one bookmark | Other items and panes stay white/black | Auto ✔ `TestStyleColours`, `TestBookmarkStyle` + Manual |
| TC-073a | FR-073 | Contrast ratio | black/white, yellow/white, light green/white, `#767676`/white | 21, 1.07, 1.42, 4.54 | Auto ✔ `TestRatio` |
| TC-074a | FR-074 | Default minimum 4.5 | Yellow, light green, `#777777` on white | Rejected; `#767676` and black accepted | Auto ✔ `TestMeetsDefault` |
| TC-074b | FR-074 | Minimum limits | Set 1, 7.5, 30 | Clamped to 3, 7, 7 | Auto ✔ `TestClampMin` |
| TC-075a | FR-075 | Fallback text colour | Background white, yellow, black, dark blue | black, black, white, white | Auto ✔ `TestBestText` |
| TC-075b | FR-075 | Dialog refuses failing colours | Pick yellow text on white; black text on navy | Refused with ratio; black resp. white suggested; nothing stored | Auto ✔ `TestSetColours` + Manual |
| TC-075c | FR-075 | Colour dialog live check | Defaults; yellow text; black on navy → "Use white text"; invalid `#12345`; save | 21.00:1 OK; below minimum, Save disabled, "Use black text"; white suggested and applied; invalid message; colours stored | Auto ✔ `TestColourDialog` + Manual |
| TC-076 | FR-076 | Saved colours re-checked | Yellow on white; `#767676` at min 4.5 and 7; invalid colour | Drawn in black where failing; saved colours unchanged | Auto ✔ `TestEffective`, `TestStyleColours` |
| TC-077 | FR-077 | Theme colours pass | Every theme text/background pair | All ≥ minimum | Auto ✔ `TestThemeContrast`, `TestThemeGrayFollowsMinimum` |
