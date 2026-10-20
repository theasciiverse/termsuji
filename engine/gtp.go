// Package engine plays Go against a local engine, GNU Go, through the Go
// Text Protocol (GTP): it starts the engine as a child process and sends
// it commands.
package engine

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Colours of stones, as in api.BoardState: 0 empty, 1 black, 2 white.
const (
	Empty = 0
	Black = 1
	White = 2
)

// Pass and Resign are the coordinates of moves that place no stone.
const (
	Pass   = -1
	Resign = -2
)

// gtpLetters are the column letters GTP uses: I is skipped.
const gtpLetters = "ABCDEFGHJKLMNOPQRSTUVWXYZ"

// Engine is a running GTP engine.
type Engine struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	out   *bufio.Reader
	mu    sync.Mutex
	size  int
	dir   string
	komi  float64
	// the game so far, kept here for undo and for SaveSGF (GNU Go's own
	// printsgf writes only the position, without its moves)
	handicap []Move
	moves    []Move
}

// Move is one move: Colour plays at X, Y, or passes (X and Y Pass).
type Move struct {
	Colour, X, Y int
}

// FindGnuGo returns the GNU Go shipped beside this program, else the one on
// the PATH, else "".
func FindGnuGo() string {
	name := "gnugo"
	if runtime.GOOS == "windows" {
		name = "gnugo.exe"
	}
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		beside := filepath.Join(filepath.Dir(exe), name)
		if fi, err := os.Stat(beside); err == nil && !fi.IsDir() {
			return beside
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

// Start runs GNU Go from path at the given strength (1 to 10) in GTP mode,
// in the folder dir, where SaveSGF and LoadSGF keep their files (GTP
// arguments cannot hold spaces, and a data folder's path can).
func Start(path string, level int, dir string) (*Engine, error) {
	if path == "" {
		return nil, errors.New("GNU Go was not found: it should be beside termsuji, or on the PATH")
	}
	cmd := exec.Command(path, "--mode", "gtp", "--level", fmt.Sprint(level))
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Engine{cmd: cmd, stdin: stdin, out: bufio.NewReader(stdout), dir: dir}, nil
}

// Command sends one GTP command and returns the engine's answer without its
// leading "= ", or an error with the engine's message for "? ".
func (e *Engine) Command(format string, args ...interface{}) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, err := fmt.Fprintf(e.stdin, format+"\n", args...); err != nil {
		return "", err
	}
	// an answer is "= text" or "? text", ended by an empty line
	var lines []string
	for {
		line, err := e.out.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if len(lines) == 0 {
				continue
			}
			break
		}
		lines = append(lines, line)
	}
	answer := strings.Join(lines, "\n")
	if strings.HasPrefix(answer, "?") {
		return "", errors.New(strings.TrimSpace(answer[1:]))
	}
	return strings.TrimSpace(strings.TrimPrefix(answer, "=")), nil
}

// Close ends the engine.
func (e *Engine) Close() {
	e.Command("quit")
	e.stdin.Close()
	e.cmd.Wait()
}

// NewGame clears the board for a game of the given size and komi, with a
// handicap of fixed stones for Black (0 for none).
func (e *Engine) NewGame(size int, komi float64, handicap int) error {
	if _, err := e.Command("boardsize %d", size); err != nil {
		return err
	}
	e.size = size
	e.komi = komi
	e.handicap, e.moves = nil, nil
	if _, err := e.Command("clear_board"); err != nil {
		return err
	}
	if _, err := e.Command("komi %g", komi); err != nil {
		return err
	}
	if handicap >= 2 {
		stones, err := e.Command("fixed_handicap %d", handicap)
		if err != nil {
			return err
		}
		for _, v := range strings.Fields(stones) {
			if x, y, err := e.parseVertex(v); err == nil {
				e.handicap = append(e.handicap, Move{Black, x, y})
			}
		}
	}
	return nil
}

// Moves is the game so far, handicap stones not included.
func (e *Engine) Moves() []Move {
	return e.moves
}

// Handicap is the number of handicap stones.
func (e *Engine) Handicap() int {
	return len(e.handicap)
}

// Size is the board's size.
func (e *Engine) Size() int {
	return e.size
}

// Vertex turns board coordinates (x from the left, y from the top, as the
// board is drawn) into a GTP vertex such as "D4".
func (e *Engine) Vertex(x, y int) string {
	if x == Pass {
		return "pass"
	}
	return fmt.Sprintf("%c%d", gtpLetters[x], e.size-y)
}

// parseVertex turns a GTP vertex into board coordinates.
func (e *Engine) parseVertex(v string) (x, y int, err error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "PASS":
		return Pass, Pass, nil
	case "RESIGN":
		return Resign, Resign, nil
	}
	if len(v) < 2 {
		return 0, 0, fmt.Errorf("bad vertex %q", v)
	}
	x = strings.IndexByte(gtpLetters, v[0])
	var row int
	if _, err := fmt.Sscan(v[1:], &row); err != nil || x < 0 {
		return 0, 0, fmt.Errorf("bad vertex %q", v)
	}
	return x, e.size - row, nil
}

func colourName(c int) string {
	if c == White {
		return "white"
	}
	return "black"
}

// Play plays a move for colour c at x, y (Pass to pass). An illegal move is
// an error and leaves the game as it was.
func (e *Engine) Play(c, x, y int) error {
	if _, err := e.Command("play %s %s", colourName(c), e.Vertex(x, y)); err != nil {
		return err
	}
	e.moves = append(e.moves, Move{c, x, y})
	return nil
}

// GenMove asks the engine for its move as colour c, plays it and returns
// it: Pass or Resign in x and y for those.
func (e *Engine) GenMove(c int) (x, y int, err error) {
	v, err := e.Command("genmove %s", colourName(c))
	if err != nil {
		return 0, 0, err
	}
	x, y, err = e.parseVertex(v)
	if err == nil && x != Resign {
		e.moves = append(e.moves, Move{c, x, y})
	}
	return x, y, err
}

// Board returns the stones on the board, [y][x] from the top left.
func (e *Engine) Board() ([][]int, error) {
	board := make([][]int, e.size)
	for y := range board {
		board[y] = make([]int, e.size)
	}
	for _, c := range []int{Black, White} {
		list, err := e.Command("list_stones %s", colourName(c))
		if err != nil {
			return nil, err
		}
		for _, v := range strings.Fields(list) {
			x, y, err := e.parseVertex(v)
			if err != nil || x < 0 || y < 0 || x >= e.size || y >= e.size {
				continue
			}
			board[y][x] = c
		}
	}
	return board, nil
}

// Captures is the number of stones colour c has captured.
func (e *Engine) Captures(c int) int {
	n := 0
	if s, err := e.Command("captures %s", colourName(c)); err == nil {
		fmt.Sscan(s, &n)
	}
	return n
}

// Undo takes back the last move.
func (e *Engine) Undo() error {
	if len(e.moves) == 0 {
		return errors.New("nothing to undo")
	}
	if _, err := e.Command("undo"); err != nil {
		return err
	}
	e.moves = e.moves[:len(e.moves)-1]
	return nil
}

// FinalScore is the engine's count of the game, such as "B+3.5" or "W+12.5".
func (e *Engine) FinalScore() (string, error) {
	return e.Command("final_score")
}

// SaveSGF writes the game so far, its moves and handicap stones, to an SGF
// file in the engine's folder (written through a temporary file, so a
// crash never leaves half a game).
func (e *Engine) SaveSGF(name string) error {
	point := func(m Move) string {
		if m.X < 0 {
			return ""
		}
		return string([]byte{byte('a' + m.X), byte('a' + m.Y)})
	}
	var b strings.Builder
	fmt.Fprintf(&b, "(;GM[1]FF[4]CA[UTF-8]AP[termsuji]SZ[%d]KM[%g]RU[Japanese]", e.size, e.komi)
	if len(e.handicap) > 0 {
		fmt.Fprintf(&b, "HA[%d]AB", len(e.handicap))
		for _, m := range e.handicap {
			fmt.Fprintf(&b, "[%s]", point(m))
		}
	}
	b.WriteString("\n")
	for i, m := range e.moves {
		c := "B"
		if m.Colour == White {
			c = "W"
		}
		fmt.Fprintf(&b, ";%s[%s]", c, point(m))
		if i%10 == 9 {
			b.WriteString("\n")
		}
	}
	b.WriteString(")\n")
	path := filepath.Join(e.dir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadSGF reads a game from an SGF file in the engine's folder, as SaveSGF
// writes it, and returns whose turn it is. The moves are replayed one by
// one (GNU Go's own loadsgf leaves nothing to undo): the board size (SZ),
// komi (KM), handicap stones (AB) and the moves (B, W; empty for a pass)
// of the main line.
func (e *Engine) LoadSGF(name string) (int, error) {
	data, err := os.ReadFile(filepath.Join(e.dir, name))
	if err != nil {
		return 0, err
	}
	props := sgfProperties(string(data))
	size, komi := 19, 0.0
	for _, p := range props {
		switch p.name {
		case "SZ":
			fmt.Sscan(p.value, &size)
		case "KM":
			fmt.Sscan(p.value, &komi)
		}
	}
	if err := e.NewGame(size, komi, 0); err != nil {
		return 0, err
	}
	var handicap []string
	turn := Black
	for _, p := range props {
		switch p.name {
		case "AB":
			x, y, ok := sgfPoint(p.value)
			if ok {
				handicap = append(handicap, e.Vertex(x, y))
			}
		case "B", "W":
			if len(handicap) > 0 {
				if err := e.setHandicap(handicap); err != nil {
					return 0, err
				}
				handicap = nil
			}
			c := Black
			if p.name == "W" {
				c = White
			}
			x, y, ok := sgfPoint(p.value)
			if !ok {
				x, y = Pass, Pass
			}
			if err := e.Play(c, x, y); err != nil {
				return 0, fmt.Errorf("%s[%s]: %v", p.name, p.value, err)
			}
			turn = Black + White - c
		}
	}
	if len(handicap) > 0 {
		if err := e.setHandicap(handicap); err != nil {
			return 0, err
		}
		turn = White
	}
	return turn, nil
}

// setHandicap places handicap stones for Black, given as GTP vertices.
func (e *Engine) setHandicap(vertices []string) error {
	if _, err := e.Command("set_free_handicap %s", strings.Join(vertices, " ")); err != nil {
		return err
	}
	for _, v := range vertices {
		if x, y, err := e.parseVertex(v); err == nil {
			e.handicap = append(e.handicap, Move{Black, x, y})
		}
	}
	return nil
}

type sgfProperty struct {
	name, value string
}

// sgfProperties lists the properties of an SGF file's main line in order,
// a property with several values once for each.
func sgfProperties(sgf string) []sgfProperty {
	var props []sgfProperty
	name := ""
	depth := 0
	for i := 0; i < len(sgf); i++ {
		c := sgf[i]
		switch {
		case c == '(':
			depth++
			if depth > 1 {
				// a variation: only the main line is read
				return props
			}
		case c >= 'A' && c <= 'Z':
			if i > 0 && sgf[i-1] >= 'A' && sgf[i-1] <= 'Z' {
				name += string(c)
			} else {
				name = string(c)
			}
		case c == '[':
			j := i + 1
			var value strings.Builder
			for ; j < len(sgf) && sgf[j] != ']'; j++ {
				if sgf[j] == '\\' && j+1 < len(sgf) {
					j++
				}
				value.WriteByte(sgf[j])
			}
			props = append(props, sgfProperty{name, value.String()})
			i = j
		}
	}
	return props
}

// sgfPoint turns an SGF point such as "dd" (column, row from the top left)
// into board coordinates; an empty value (or "tt" on a small board) is a
// pass.
func sgfPoint(v string) (x, y int, ok bool) {
	if len(v) != 2 || v == "tt" {
		return 0, 0, false
	}
	return int(v[0] - 'a'), int(v[1] - 'a'), true
}
