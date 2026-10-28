> **Play it with Asciiverse.** With [av](https://asciiverse.sh), the Asciiverse
> client:
>
> ```
> av install termsuji
> av play termsuji
> ```
>
> The original README follows.

# Termsuji

termsuji plays Go in your terminal: against GNU Go on your own computer,
or (in a build with an online-go.com client id) your games on
online-go.com.

This is the Asciiverse edition of [lvank's termsuji](https://github.com/lvank/termsuji),
an online-go.com client, with play against GNU Go added: GNU Go 3.9.1
comes beside termsuji in each download. What changed is in `PATCHES.md`.

## Playing

    termsuji

**New game** sets up a game against GNU Go: the board (9x9, 13x13 or
19x19), your colour (Black moves first), GNU Go's strength from 1 to 10,
and handicap stones for Black. Komi is 6.5, or 0.5 with a handicap.
**Continue** picks up your game where you left it: it is saved after
every move.

- Arrow keys move the cursor (the first press puts it on the last stone
  played), Return plays a stone there.
- **p** passes; when both players pass in a row the game ends and GNU Go
  counts it. GNU Go may also resign.
- **u** takes back your last move and GNU Go's answer.
- **t** chooses a theme, **q** goes back to the menu (q first clears the
  cursor if one is shown).
- **n** starts a new game once a game is over.

The game is saved in `$XDG_DATA_HOME/termsuji` (`~/.local/share/termsuji`,
`%LOCALAPPDATA%\termsuji` on Windows) as `game.sgf`, which any Go program
can open, beside `game.json` with its settings.

termsuji finds GNU Go beside itself, else on the PATH.

## Playing on online-go.com

The online part is lvank's, unchanged: it can play and finish games in
progress on online-go.com, and is limited in features. It needs an OAuth
application registered at https://online-go.com/oauth2/applications/
(client type "Public", grant type "resource owner password based"); put
its client ID in `api/client_id.txt` before building, and **Play online**
appears in the menu.

## Building

    go build .                 # termsuji; api/client_id.txt may be empty
    asciiverse/gnugo.sh        # GNU Go into stage/ (see the script for Windows)

## Configuration

There's a themes option in-application with some preset themes, but you can get more detailed configuration by editing the configuration file.
The application stores a configuration file in $XDG_CONFIG_HOME/termsuji/config.json (or C:/Users/YourUsername/AppData/Roaming/termsuji/config.json on Windows) with the following configurable values.

To find out "Unicode code points", you can visit https://unicode-table.com/, look up a symbol you want to use and copy the number from "HTML code" (or for the technically inclined, convert the Unicode number from hex to int).

To find out colour numbers, refer to the bottom left numbers on https://upload.wikimedia.org/wikipedia/commons/1/15/Xterm_256color_chart.svg
To reset your default settings, just delete the configuration file and it'll be regenerated with the defaults.

(Note: for symbols, you can't use 1-31 and 127-159.)

```
{"theme": {
  "grid": If true (the classic theme), draws the board as lines and points on wood, scaled to the window: colours "board" for the wood, "board_alt" for the lines, "black"/"white" for the stones, "cursor_bg" and "last_played_bg" for the marks; symbols "black"/"white" for the stones when the board is too dense for discs. If false, draws squares as below.
  "draw_stone_bg": If true, will draw a stone with the black/black_alt colours. If false, draws the black/white symbols instead. Default true.
  "draw_cursor_bg": If true, will draw the currently selected square with the cursor_bg colour. If false, draws the cursor symbol instead. Default false.
  "draw_last_played_bg": If true, will draw the last played stone with the last_played_bg colour. If false, draws the last_played symbol instead. Default false.
  "fullwidth_letters": If true, will draw letter coordinates as fullwidth Japanese characters, occupying two spaces. Default false.
  "colors": {
    "board":             Go board background colour
    "board_alt":         Go board background colour (for alternating squares)
    "black":             Black player stone colour
    "black_alt":         Black player stone colour (for alternating squares)
    "white":             White player stone colour
    "white_alt":         White player stone colour (for alternating squares)
    "cursor_fg":         Cursor foreground colour (if no stone is selected)
    "cursor_bg":         Cursor background colour (if draw_stone_bg is true)
    "last_played_bg":    Last played stone background colour (if draw_last_played_bg is true)
  },
  "symbols": {
    "black":       Unicode code point for black stones. Default 32 (a blank space)
    "white":       Unicode code point for white stones. Default 32 (a blank space)
    "board":       Unicode code point for the board itself. Default 32 (a blank space)
    "cursor":      Unicode code point for the cursor. Default 88, or 'X'
    "last_played": Unicode code point to mark the last played stone. Default 47, or '/'
  }
}}
```
