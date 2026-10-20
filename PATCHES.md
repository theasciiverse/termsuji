# Changes from upstream

This is the Asciiverse edition of [termsuji](https://github.com/lvank/termsuji)
by lvank (2022), an online-go.com client for the terminal in Go, on tview
and tcell. Upstream is the `upstream` remote, last changed 2022-08-10
(`01b3ecb`); the `asciiverse` branch carries the changes below, which add
play against GNU Go on the player's own computer. Upstream's online play
needs an online-go.com account and an OAuth application of the builder's
own, so without this the game could not be played out of the box.

## Licence

MIT (LICENSE, "Copyright (c) 2022 lvank"); our changes are under the same
licence. GNU Go 3.9.1, shipped beside termsuji (below), is GPL-3.0-or-later;
it is a separate program that termsuji starts and talks to over GTP, so
the two keep their own licences. The Go modules built into termsuji are
upstream's (tview, tcell, adrg/xdg, go-runewidth, golang-socketio,
gorilla/websocket and their dependencies), unchanged.

## GNU Go

GNU Go's last release tarball is 3.8 (2009); its maintainers tagged 3.9.1
in their git (`rel-3-9-1`, git.savannah.gnu.org/git/gnugo.git, commit
`e2b23a8`, 2010-12-21) without making a tarball. Debian ships 3.9.1 since
June 2026 (unstable and forky) from a `git archive` of that tag, which its
changelog records as byte-identical to the one its watch file (uscan) makes; that tarball,
`gnugo_3.9.1.orig.tar.gz` from deb.debian.org, sha256
`268e7e46653c00f27ae4f46523a2a76f56d26b3612fe41a68cb852cc10de80fb`, is
what `asciiverse/gnugo.sh` builds, unchanged, and every release carries it
as `gnugo-3.9.1-source.tar.gz`.

The build needs, for today's compilers, `-fcommon` (variables defined in
headers, an error since GCC 10) and four GCC 14 errors turned back into
warnings (old declarations in its Windows code). Linux is static (Alpine),
macOS native. For Windows it is cross-built with MinGW; its build runs
small generators that write the pattern databases as C, which a cross
build cannot run, so a native build comes first and its generators are put
where the cross build calls them (`./mkpat` and the others, without
`.exe`); the databases are plain tables, the same on every platform.
Checked: static Linux plays over GTP; the Windows build answers GTP under
Wine (`genmove white` after E5: D4).

## Code

- **engine: play against GNU Go over GTP** (new package). Starts GNU Go,
  found beside termsuji, else on the PATH, in the player's data folder (GTP
  arguments cannot hold spaces, and a data folder's path can); plays,
  asks for moves, lists the stones, counts captures, undoes, scores, saves
  and loads. It keeps the game's moves itself: GNU Go's `printsgf` writes
  only the position, as setup stones, and after its `loadsgf` nothing can
  be undone, so a game is saved as real SGF with its moves and handicap
  stones, written through a temporary file, and loaded by replaying them.
  Tests against a real GNU Go: moves, an illegal move refused, undo after
  loading, a 3-stone handicap game saved and loaded.
- **Board coordinates: skip the letter I; row numbers beside the board.**
  The columns were lettered straight through (A to T on 19x19 should skip
  I, as online-go.com and GTP write them), and the row numbers were drawn
  at the screen's columns 1 and 2 wherever the board was.
- **Play against GNU Go on this computer.** A start menu: New game (board
  9x9, 13x13 or 19x19; Black or White; GNU Go's strength 1 to 10, default
  5; handicap stones; komi 6.5, or 0.5 with a handicap), Continue,
  Themes, Quit, and Play online only when the build has an online-go.com
  client id (it no longer logs in before the menu). GNU Go answers in the
  background; p passes, u takes back your move and GNU Go's answer, two
  passes in a row end the game with GNU Go's count, and a resignation
  ends it too; the end screen gives the result. The game is saved after every move in
  `$XDG_DATA_HOME/termsuji` (`game.sgf`, `game.json` with its settings)
  and removed once it is over.
- **A classic board.** The new default theme draws a Go board: lines and
  intersections in box-drawing characters on wood, star points, and
  stones on the points as discs of block characters (5x3 cells) where
  there is room, else as symbols. The distance between points is the
  largest that fits (8x4 down to 2x1 cells), so 9x9 fills the window and
  19x19 fits in 100x30, centred. Upstream's squares remain the other
  themes (`grid` in the config switches).

## Packaging

- `asciiverse/gnugo.sh` builds GNU Go (above); `asciiverse/package.sh`
  packs `termsuji-<target>` with `termsuji`, `gnugo`, LICENSE, README.md,
  this file and GNU Go's licence as `GNUGO-COPYING`.
- `.github/workflows/release.yml`: static Linux (Alpine, x86-64 and
  ARM64), macOS (Intel and Apple silicon runners, each native) and Windows
  x86-64 (Go cross-compiled, GNU Go with MinGW), plus SHA256SUMS and the
  GNU Go source, on our tags only (`*-av.*`). An online-go.com client id in
  the `OGS_CLIENT_ID` secret turns on online play in the release; without
  one it plays GNU Go only.
- `asciiverse/record.py` records the catalog preview,
  `asciiverse/termsuji.cast`: a 9x9 game as Black against GNU Go at
  strength 5, Black's moves from a second GNU Go at strength 1, entered
  with the cursor keys, the game read back from termsuji's save file.
- README.md is rewritten for players (lvank's configuration notes kept,
  with the new `grid` setting).

## Checked

Linux, static (Alpine): new games on 9x9 and 19x19, moves and
GNU Go's answers, leaving to the menu and Continue (the save is SGF with
its moves), undo; GNU Go exits with termsuji when its window is closed.
Windows: termsuji cross-compiled, GNU Go cross-built and answering GTP
under Wine; termsuji itself is not run there. macOS: not built here.
