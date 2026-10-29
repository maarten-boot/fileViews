# Manual Test Checklist — fileViews

_Generated from `TestCases.md` (regenerated 2026-10-29 after bookmark ordering and drag & drop). Run on a desktop; tick each item and note anything odd next to it._

**Preparation:** `make testdir` creates `/tmp/fileviews-test` with files, a subfolder, a hidden file, an executable, symlinks to a file and a folder, a broken symlink and a FIFO, plus `remove-me/` to bookmark and then delete (missing bookmarks), and starts the app there. For the scrolling checks use a big folder too, e.g. `/usr/bin`.


## Menu bar

- [ ] **TC-001** (FR-001) Menus present — *Open each menu* → File, View, Go, Settings, Help with the listed items
- [ ] **TC-001b** (FR-001) Settings menu usable — *Choose each Settings item* → Each opens its dialog / toggles
- [ ] **TC-002** (FR-002) Keyboard shortcuts — *Press each shortcut* → Matching action runs

## Right pane / listing (`internal/listing`)

- [ ] **TC-020b** (FR-020) Whole-row selection — *Click a cell of `a.txt`; sort by name descending; show hidden; refresh; go to `sub`* → Whole row highlighted; selection kept through sort, filter and refresh; cleared on navigation
- [ ] **TC-020c** (FR-020) Multi-select — *Click b; Ctrl+click d; Ctrl+click b; Shift+click a; Shift+click e; click c; anchor gone* → `b`; `b,d`; `d`; `a,b`; `b..e`; `c`; Shift acts as plain click
- [ ] **TC-020d** (FR-020) Column resize reaches the table — *Mouse press on a header cell* → Forwarded to the table in table coordinates (drag itself: manual)
- [ ] **TC-020g** (FR-032) Few rows do not move — *2 entries; wheel; pane too narrow; wheel; Shift+wheel* → Offset stays 0,0; Shift+wheel moves sideways only
- [ ] **TC-020h** (FR-032) Many rows scroll; refresh keeps position — *200 entries; wheel down; F5* → Moves down only; same position after F5
- [ ] **TC-020i** (FR-032) New directory starts at the top — *Scroll down in a big directory, open another* → Starts at top-left
- [ ] **TC-020j** (FR-020) Name fills the width — *Window 1000, 1200, 500 wide; drag Name; resize; change columns* → Fills (wider at 1200); 150 px and sideways scroll at 500; dragged width kept; refilled after column change
- [ ] **TC-020k** (FR-020) Measured column widths — *Default widths; sort by Size* → Each column at its measured width; Size widened for the arrow; still fits
- [ ] **TC-022c** (FR-022) Table cells — *`a.txt` mode 0640, 100 bytes* → `-rw-r-----`, `1`, owner, group, `100`, `YYYY-MM-DD HH:MM`, `a.txt`; Size and Links right-aligned
- [ ] **TC-024g** (FR-024) Columns dialog — *Move Name up; hide Links; hide+show Group; try to hide Name; apply; reopen* → Table and config show the new order without Links; Name locked; Links shown unticked when reopened
- [ ] **TC-025e** (FR-025) Sort by clicking headers — *Click Size, Size again, Name* → Size ascending (dirs first); descending (whole order reversed, dirs first); back to name; arrow icon on sorted column
- [ ] **TC-027** (FR-027) Double-click directory — *Double-click `sub/`; double-click `link -> sub`* → Right pane shows `sub`; shows `link`
- [ ] **TC-028** (FR-028) Double-click file — *Double-click `a.txt`; a FIFO; a file removed since listing; opener fails* → `xdg-open a.txt`; others: error, nothing opened
- [ ] **TC-031** (FR-031) Grid lines — *Default; View → Show grid lines; new window with saved setting* → Light gray `#E8E8E8` only in the table; hidden after toggle; setting applied
- [ ] **TC-030** (FR-030) Read-only — *Browse around* → No file is modified

## Path bar / history (`internal/history`)

- [ ] **TC-010** (FR-010) Path bar shows current path — *Navigate to `/tmp`* → Path field shows `/tmp`
- [ ] **TC-011a** (FR-011) Type valid path — *Enter `/tmp`* → Navigates to `/tmp`
- [ ] **TC-011b** (FR-011) Type invalid path — *Missing path, a file, empty input* → Error; directory and recent list unchanged
- [ ] **TC-017d** (FR-017) Bad command-line directory — *`fileviews /gone`* → Error shown, next choice used
- [ ] **TC-015d** (FR-015) Recent drop-down — *Visit root, sub; open ▾; choose root* → Menu lists sub, root; goes to root
- [ ] **TC-017e** (FR-017) Start in last directory after restart — *Saved config with last directory* → Window starts there
- [ ] **TC-016** (FR-016) Recent kept across sessions — *Visit A, restart* → A in dropdown

## Left pane / bookmarks (`internal/bookmarks`)

- [ ] **TC-043** (FR-043) Click opens directory — *Click `mboot`* → Right pane shows `$HOME`
- [ ] **TC-041d** (FR-041) Default items in menu — *Right-click `/`* → Remove bookmark and Reset colours disabled
- [ ] **TC-042c** (FR-042) Bookmark menu actions — *Add current directory (twice), reset colours, open, remove* → Added once (duplicate: error); reset; opened; removed
- [ ] **TC-042d** (FR-042) Bookmark from right pane — *Right-click `sub/`; right-click a file* → `sub` selected, "Add to bookmarks" adds it; no menu for files
- [ ] **TC-044c** (FR-044) Indent drawn — *Home at depth d, indent 12 and 20* → Spacer width 12·d, 20·d; `/` none
- [ ] **TC-045d** (FR-045) Colours drawn — *White on navy for home* → Home drawn white on navy; `/` stays black on white
- [ ] **TC-046d** (FR-046) Hover text pop-up — *Hover home; leave; custom text; leave before delay* → Full path shown; hidden; "work" shown; nothing shown
- [ ] **TC-047c** (FR-047) Missing bookmark drawn and clicked — *Bookmark removed dir* → `gone (missing)`, italic, `#767676`; click: error, directory unchanged
- [ ] **TC-044d** (FR-044) Indent setting — *Field 12 / slider max 25; type 10; drag to 7; type 0, 26, abc, empty; type " 3 " and apply; apply 500* → Slider and field follow each other; invalid input disables Apply with a hint; 3 applied and saved; 500 limited to 25
- [ ] **TC-048b** (FR-048) Move up / Move down menu — *Move `/a2` up, `/` down; menu of last item* → Order changed; Move down disabled on the last item
- [ ] **TC-048c** (FR-048) Drag a bookmark within the pane — *Drag last item above row 1; drag first item below the last* → Drop line shown; moved to the slot; drag state cleared
- [ ] **TC-049a** (FR-049) Drag a directory to the pane — *Drag `other/` above `/a1`* → Label "Add bookmark: other", drop line; inserted at that slot
- [ ] **TC-049b** (FR-049) Already bookmarked: refused while dragging — *Drag `sub/` which is a bookmark* → Label "sub: already a bookmark", no drop line; drop changes nothing, no error
- [ ] **TC-049c** (FR-049) Drag a selection — *Select `other/`, `sub/` (bookmarked), `a.txt`; drag* → Label "Add 1 bookmark (1 already bookmarked)"; only `other` added
- [ ] **TC-049d** (FR-049) Ignored drags — *Drag a file; drop a directory outside the pane* → Nothing happens, no drop line
- [ ] **TC-047b** (FR-047) Missing style — *Default white; black bg; light yellow bg with navy text* → `gone (missing)`, italic; text `#767676`; `#757575`; navy — all pass 4.5:1

## Status bar

- [ ] **TC-051** (FR-051) Updates on toggle hidden — *Ctrl+H* → Counts/size update

## Persistence (`internal/config`)

- [ ] **TC-061c** (FR-065) Save on quit — *Resize to 800×500, split 0.4, quit* → Window size and split saved
- [ ] **TC-061d** (FR-061) Restore — *Config with every setting* → New window shows toggles, sort, bookmarks, recent, split, size; starts in last directory

## Non-functional

- [ ] **TC-N02** (NFR-002) Responsive UI — *Open a large directory* → Window stays responsive

## Build / smoke

- [ ] **TC-N04c** (NFR-004) Window opens on desktop — *`make run`* → Window "fileViews" appears, 1000×650

## Appearance

- [ ] **TC-070a** (FR-070) Light theme by default — *Start app; bookmark without colours* → White background, black text everywhere
- [ ] **TC-070b** (FR-070) Ignores desktop dark mode — *Desktop in dark mode, start app* → Still white background, black text
- [ ] **TC-071c** (FR-071) No gray-on-gray in the GUI — *Inspect missing bookmark, disabled buttons, status bar* → No gray text on gray background
- [ ] **TC-072** (FR-072) Colours only affect their item — *Colour one bookmark* → Other items and panes stay white/black
- [ ] **TC-074c** (FR-074, FR-076) Contrast minimum setting — *Slider 3–7 at 4.5; apply 7; apply 9* → Theme gray `#595959`; `#767676` bookmark drawn black, saved colour kept; saved 7; 9 limited to 7
- [ ] **TC-075b** (FR-075) Dialog refuses failing colours — *Pick yellow text on white; black text on navy* → Refused with ratio; black resp. white suggested; nothing stored
- [ ] **TC-075c** (FR-075) Colour dialog live check — *Defaults; yellow text; black on navy → "Use white text"; invalid `#12345`; save* → 21.00:1 OK; below minimum, Save disabled, "Use black text"; white suggested and applied; invalid message; colours stored

## Result

| Date | Tester | Passed | Failed / notes |
|------|--------|--------|----------------|
