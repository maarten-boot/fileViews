# Test Cases — fileViews

_Last updated: 2026-10-29_

Each test case links to the requirement it verifies. **Auto** = planned as a Go unit test; **Manual** = GUI check.

## Right pane / listing
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-020 | FR-020 | List directory | Temp dir with `a.txt`, `sub/` | Both entries listed | Auto |
| TC-021a | FR-021 | Hidden filtered by default | Dir with `.x`, `y` | Only `y` listed | Auto |
| TC-021b | FR-021 | Show hidden | Same dir, show-hidden on | `.x` and `y` listed | Auto |
| TC-022 | FR-022 | Permission string | File with mode 0754 | `-rwxr-xr--` | Auto |
| TC-023a | FR-023 | Directory marker | `sub/` | Name `sub/` | Auto |
| TC-023b | FR-023 | Executable marker | File mode 0755 | Name ends in `*` | Auto |
| TC-023c | FR-023 | Symlink marker + target | `l -> a.txt` | `l@ -> a.txt` | Auto |
| TC-023d | FR-023 | FIFO marker | `mkfifo p` | `p\|` | Auto |
| TC-023e | FR-023 | Socket marker | Unix socket `s` | `s=` | Auto |
| TC-024a | FR-024 | Column order applied | Order `[Name, Size, Perms]` | Rows formatted in that order | Auto |
| TC-024b | FR-024 | Name cannot be hidden | Config hides Name | Name still shown | Auto |
| TC-025a | FR-025 | Default sort | `b`, `A/`, `a` | `A/` first, then `a`, `b` | Auto |
| TC-025b | FR-025 | Sort by size, reverse | Files of 1, 3, 2 bytes | Order 3, 2, 1 | Auto |
| TC-026a | FR-026 | Human-readable size | 4300 bytes | `4.2K` | Auto |
| TC-026b | FR-026 | Exact size | 4300 bytes, exact mode | `4300` | Auto |
| TC-027 | FR-027 | Double-click directory | Double-click `sub/` | Right pane shows `sub` | Manual |
| TC-028 | FR-028 | Double-click file | Double-click `a.txt` | Opens in default app | Manual |
| TC-029a | FR-029 | Broken symlink | `l -> missing` | Listed; rest of listing unaffected | Auto |
| TC-029b | FR-029 | Unreadable subdir | `chmod 000 sub` | `sub/` listed | Auto |
| TC-030 | FR-030 | Read-only | Browse around | No file is modified | Manual |

## Path bar / history
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-011a | FR-011 | Type valid path | Enter `/tmp` | Navigates to `/tmp` | Manual |
| TC-011b | FR-011 | Type invalid path | Enter `/nonexistent` | Error; directory unchanged | Auto + Manual |
| TC-012a | FR-012 | Back/forward | Visit A, B, C; Back; Back; Forward | Current = B | Auto |
| TC-012b | FR-012 | New visit clears forward | Visit A, B; Back; visit C | Forward is empty | Auto |
| TC-013 | FR-013 | Up at root | At `/` | Up disabled / no-op | Auto |
| TC-015a | FR-015 | Recent: no duplicates | Visit A, B, A | Recent = `[A, B]` | Auto |
| TC-015b | FR-015 | Recent: max size | Visit 25 dirs | 20 most recent kept | Auto |
| TC-016 | FR-016 | Recent kept across sessions | Visit A, restart | A in dropdown | Auto (config) + Manual |

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
| TC-047 | FR-047 | Missing target | Bookmark to removed dir | Marked missing | Auto + Manual |

## Status bar
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-050a | FR-050 | Counts | 2 dirs, 3 files (1 hidden) | dirs 2, files 3, hidden 1 | Auto |
| TC-050b | FR-050 | Total size | Files of 100 + 200 bytes | 300 | Auto |
| TC-050c | FR-050 | Selection size | Select a 100-byte file | selected 100 | Auto |
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
| TC-N01 | NFR-001 | Large directory | 10,000 files | Listing < 1 s (benchmark) | Auto (bench) |
| TC-N02 | NFR-002 | Responsive UI | Open a large directory | Window stays responsive | Manual |

## Build / smoke
| ID | Req | Description | Steps / input | Expected result | Type |
|----|-----|-------------|---------------|-----------------|------|
| TC-N04a | NFR-004 | Build | `make build` | `fileviews` binary, no errors | Auto |
| TC-N04b | NFR-004 | Main window created (headless) | `TestNewMainWindow` | Title `fileViews`, content set | Auto |
| TC-N04c | NFR-004 | Window opens on desktop | `make run` | Window "fileViews" appears, 1000×650 | Manual |
