# Test Cases — fileViews

_Last updated: 2026-10-29_

Each test case links to the requirement it verifies. **Auto** = Go unit test; **Manual** = GUI check.
A ✔ with a test name means the automated test exists and passes.

## Right pane / listing (`internal/listing`)
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-020 | FR-020 | List directory | Temp dir with `a.txt`, `sub/` | Both entries listed | Auto ✔ `TestReadListsFilesAndDirs` |
| TC-021a | FR-021 | Hidden filtered by default | Dir with `.x`, `y` | Only `y` listed | Auto ✔ `TestFilterHidden` |
| TC-021b | FR-021 | Show hidden | Same dir, show-hidden on | `.x` and `y` listed | Auto ✔ `TestFilterHidden` |
| TC-022 | FR-022 | Permission string | File with mode 0754; also dir, link, pipe, socket, devices, setuid/setgid/sticky | `-rwxr-xr--` etc. | Auto ✔ `TestPermString`, `TestReadPermissions` |
| TC-022b | FR-022 | Unknown uid/gid | uid 4000000000 | Shown numerically | Auto ✔ `TestUnknownIDsAreNumeric` |
| TC-023a | FR-023 | Directory marker | `sub/` | Name `sub/` | Auto ✔ `TestDisplayNameMarkers` |
| TC-023b | FR-023 | Executable marker | File mode 0755 | Name ends in `*` | Auto ✔ `TestDisplayNameMarkers` |
| TC-023c | FR-023 | Symlink marker + target | `l -> a.txt` | `l@ -> a.txt` | Auto ✔ `TestDisplayNameMarkers` |
| TC-023d | FR-023 | FIFO marker | `mkfifo p` | `p\|` | Auto ✔ `TestFIFO` |
| TC-023e | FR-023 | Socket marker | Unix socket `s` | `s=` | Auto ✔ `TestSocket` |
| TC-024a | FR-024 | Column order applied | Order `[Name, Size, Perms]` | Rows formatted in that order | Auto ✔ `TestColumnOrder` |
| TC-024b | FR-024 | Name cannot be hidden | Config hides Name | Name still shown | Auto ✔ `TestColumnLayoutNormalises` |
| TC-024c | FR-024 | Stale/bad column config | Duplicates, unknown ids, missing columns | Dropped / appended | Auto ✔ `TestColumnLayoutNormalises` |
| TC-025a | FR-025 | Default sort | `b`, `A/`, `a`, `B` | `A/`, `a`, `B`, `b` | Auto ✔ `TestDefaultSort` |
| TC-025b | FR-025 | Sort by size, reverse | Files of 1, 3, 2 bytes + a dir | dir, then 3, 2, 1 | Auto ✔ `TestSortBySizeDescending` |
| TC-025c | FR-025 | Ties broken by name | Equal mtimes | Ordered by name | Auto ✔ `TestSortTiesByName` |
| TC-025d | FR-025 | Symlink to dir sorts as dir | `ls -> sub` | Treated as directory | Auto ✔ `TestSymlinkToDir` |
| TC-026a | FR-026 | Human-readable size | 4300 bytes | `4.2K` | Auto ✔ `TestSizeFormat` |
| TC-026b | FR-026 | Exact size | 4300 bytes, exact mode | `4300` | Auto ✔ `TestSizeFormat` |
| TC-026c | — | Modified time format | 2026-10-29 11:04:59 | `2026-10-29 11:04` | Auto ✔ `TestModifiedFormat` |
| TC-027 | FR-027 | Double-click directory | Double-click `sub/` | Right pane shows `sub` | Manual |
| TC-028 | FR-028 | Double-click file | Double-click `a.txt` | Opens in default app | Manual |
| TC-029a | FR-029 | Broken symlink | `l -> missing` | Listed; rest of listing unaffected | Auto ✔ `TestBrokenSymlink` |
| TC-029b | FR-029 | Unreadable subdir | `chmod 000 sub` | `sub/` listed | Auto ✔ `TestUnreadableSubdir` |
| TC-029c | FR-029 | Unreadable metadata | Entry with stat error | Fields `?`, name shown | Auto ✔ `TestUnreadableFieldsShowQuestionMark` |
| TC-030 | FR-030 | Read-only | Browse around | No file is modified | Manual |

## Path bar / history (`internal/history`)
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-011a | FR-011 | Type valid path | Enter `/tmp` | Navigates to `/tmp` | Manual |
| TC-011b | FR-011 | Type invalid path | Missing path, a file, empty input | Error; directory and recent list unchanged | Auto ✔ `TestGoInvalidKeepsCurrent`, `TestReadMissingDir` + Manual |
| TC-011c | FR-011 | Unreadable directory | `chmod 000 a`, go to `a` | Permission error; directory unchanged | Auto ✔ `TestGoUnreadableDir` |
| TC-011d | FR-011 | Path resolution | `a`, `b/..`, `~`, `~/a/b`, spaces around path | Resolved, cleaned absolute paths | Auto ✔ `TestGoResolvesPaths` |
| TC-011e | FR-011 | Relative path with no current dir | `a` at startup | Error | Auto ✔ `TestRelativeWithoutCurrent` |
| TC-012a | FR-012 | Back/forward | Visit A, B, C; Back; Back; Forward | Current = B | Auto ✔ `TestBackForward`, `TestNavigatorBackForward` |
| TC-012b | FR-012 | New visit clears forward | Visit A, B; Back; visit C | Forward is empty | Auto ✔ `TestVisitClearsForward` |
| TC-012c | FR-012 | Revisit current adds nothing | Visit A, A | No Back entry | Auto ✔ `TestHistoryEdges` |
| TC-012d | FR-012 | Back to removed dir | Visit B, root; remove B; Back | Error; still at root, Back still possible | Auto ✔ `TestBackToRemovedDir` |
| TC-013 | FR-013 | Up, and Up at root | `/x/a/b` → Up; at `/` | `/x/a`; at `/` Up disabled, error, no move | Auto ✔ `TestParent`, `TestUpAndHome` |
| TC-014 | FR-014 | Home | Home | `$HOME` | Auto ✔ `TestUpAndHome` |
| TC-015a | FR-015 | Recent: no duplicates | Visit A, B, A | Recent = `[A, B]` | Auto ✔ `TestRecentNoDuplicates` |
| TC-015b | FR-015 | Recent: max size | Visit 25 dirs | 20 most recent kept | Auto ✔ `TestRecentMax` |
| TC-015c | FR-015 | Recent updated by Back | A, B, C; Back | Recent starts with B | Auto ✔ `TestNavigatorBackForward` |
| TC-016 | FR-016 | Recent kept across sessions | Visit A, restart | A in dropdown | Auto (load cleanup ✔ `TestRecentFromConfig`; save in M4) + Manual |

## Left pane / bookmarks
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-041a | FR-041 | Default items | Empty config, `HOME=/home/mboot` | Items `/` and `mboot` | Auto |
| TC-041b | FR-041 | Defaults not removable | Remove `/` | Refused | Auto |
| TC-042 | FR-042 | Add/remove bookmark | Add `/tmp`, then remove | Present, then gone | Auto |
| TC-044a | FR-044 | Depth | `/`, `/home/mboot`, `/a/b/c/` | 0, 2, 3 | Auto |
| TC-044b | FR-044 | Indent | Depth 2, indent 12 | 24 px | Auto |
| TC-045 | FR-045 | Colours kept | Set bg/fg, save, reload | Same colours | Auto |
| TC-046a | FR-046 | Default hover text | No custom text | Full path | Auto |
| TC-046b | FR-046 | Custom hover text kept | Set "work", save, reload | "work" | Auto |
| TC-046c | FR-046 | Empty text resets | Set "" | Full path | Auto |
| TC-047 | FR-047 | Missing target | Bookmark to removed dir | Italic label with ` (missing)`, black text | Auto + Manual |

## Status bar
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-050a | FR-050 | Counts | 2 dirs, 3 files (2 hidden), with and without show-hidden | Hidden count always 2; dirs/files follow the filter | Auto ✔ `TestComputeStats` |
| TC-050b | FR-050 | Total size | Files of 100 + 200 bytes | 300 | Auto ✔ `TestComputeStats` |
| TC-050c | FR-050 | Selection size | Select a 100-byte file and a dir | selected 100 | Auto ✔ `TestSelectionSize` |
| TC-050d | FR-050 | Free space | Temp dir; missing path | Value; error | Auto ✔ `TestFreeSpace` |
| TC-051 | FR-051 | Updates on toggle hidden | Ctrl+H | Counts/size update | Manual |

## Persistence
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-060 | FR-060 | Config path | `XDG_CONFIG_HOME=/x` | `/x/fileViews/config.json` | Auto |
| TC-062a | FR-062 | Missing config | No file | Defaults | Auto |
| TC-062b | FR-062 | Broken config | File with `{garbage` | Renamed to `.bak`, defaults, warning | Auto |
| TC-063 | FR-063 | Atomic write | Save | No partial file; temp file removed | Auto |

## Non-functional
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-N01 | NFR-001 | Large directory | 10,000 files: read + sort + format | < 1 s (measured ≈ 17 ms) | Bench ✔ `BenchmarkListLargeDir` |
| TC-N02 | NFR-002 | Responsive UI | Open a large directory | Window stays responsive | Manual |

## Build / smoke
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-N04a | NFR-004 | Build | `make build` | `fileviews` binary, no errors | Auto |
| TC-N04b | NFR-004 | Main window created (headless) | `TestNewMainWindow` | Title `fileViews`, content set | Auto ✔ `TestNewMainWindow` |
| TC-N04c | NFR-004 | Window opens on desktop | `make run` | Window "fileViews" appears, 1000×650 | Manual ✔ (user, 2026-10-29) |

## Appearance
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-070a | FR-070 | Light theme by default | Start app | White background, black text everywhere | Auto + Manual |
| TC-070b | FR-070 | Ignores desktop dark mode | Desktop in dark mode, start app | Still white background, black text | Manual |
| TC-071a | FR-071 | Gray on pure white / black | Min 4.5 | `#767676` on white, `#757575` on black; each passes the minimum | Auto ✔ `TestGray` |
| TC-071b | FR-071 | No gray on other backgrounds | `#FEFEFE`, `#808080`, `#FFFFE0` | Gray refused | Auto ✔ `TestGray` |
| TC-071c | FR-071 | No gray-on-gray in the GUI | Inspect missing bookmark, disabled buttons, status bar | No gray text on gray background | Manual |
| TC-073a | FR-073 | Contrast ratio | black/white, yellow/white, light green/white, `#767676`/white | 21, 1.07, 1.42, 4.54 | Auto ✔ `TestRatio` |
| TC-074a | FR-074 | Default minimum 4.5 | Yellow, light green, `#777777` on white | Rejected; `#767676` and black accepted | Auto ✔ `TestMeetsDefault` |
| TC-074b | FR-074 | Minimum limits | Set 1, 30 | Clamped to 3, 21 | Auto ✔ `TestClampMin` |
| TC-075a | FR-075 | Fallback text colour | Background white, yellow, black, dark blue | black, black, white, white | Auto ✔ `TestBestText` |
| TC-075b | FR-075 | Dialog refuses failing colours | Pick yellow text on white | Ratio shown, Save disabled, black text offered | Manual |
| TC-076 | FR-076 | Saved colours re-checked | Yellow on white; gray at min 7 | Drawn in black; saved colours unchanged | Auto ✔ `TestEffective` (+ config test in M5) |
| TC-077 | FR-077 | Theme colours pass | Every theme text/background pair | All ≥ minimum | Auto (M6) |
