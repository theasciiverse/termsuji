package engine

import (
	"os"
	"strings"
	"testing"
)

// The tests talk to a real GNU Go: TERMSUJI_GNUGO, or one beside the test
// binary or on the PATH; without one they are skipped.
func start(t *testing.T) *Engine {
	path := os.Getenv("TERMSUJI_GNUGO")
	if path == "" {
		path = FindGnuGo()
	}
	if path == "" {
		t.Skip("no GNU Go")
	}
	e, err := Start(path, 1, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.Close)
	return e
}

func TestVertices(t *testing.T) {
	e := &Engine{size: 19}
	cases := map[[2]int]string{{0, 18}: "A1", {7, 0}: "H19", {8, 0}: "J19", {18, 18}: "T1", {Pass, Pass}: "pass"}
	for xy, v := range cases {
		if got := e.Vertex(xy[0], xy[1]); got != v {
			t.Errorf("Vertex(%d, %d) = %s, want %s", xy[0], xy[1], got, v)
		}
		if xy[0] == Pass {
			continue
		}
		x, y, err := e.parseVertex(v)
		if err != nil || x != xy[0] || y != xy[1] {
			t.Errorf("parseVertex(%s) = %d, %d, %v", v, x, y, err)
		}
	}
}

func TestGame(t *testing.T) {
	e := start(t)
	if err := e.NewGame(9, 6.5, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.Play(Black, 4, 4); err != nil {
		t.Fatal(err)
	}
	if err := e.Play(White, 4, 4); err == nil {
		t.Fatal("a stone on a stone was allowed")
	}
	x, y, err := e.GenMove(White)
	if err != nil {
		t.Fatal(err)
	}
	board, err := e.Board()
	if err != nil {
		t.Fatal(err)
	}
	if board[4][4] != Black {
		t.Errorf("E5 is %d, want black", board[4][4])
	}
	if x >= 0 && board[y][x] != White {
		t.Errorf("White's move %d, %d is not on the board", x, y)
	}
	if err := e.SaveSGF("game.sgf"); err != nil {
		t.Fatal(err)
	}
	if err := e.NewGame(13, 6.5, 0); err != nil {
		t.Fatal(err)
	}
	turn, err := e.LoadSGF("game.sgf")
	if err != nil {
		t.Fatal(err)
	}
	if e.Size() != 9 || turn != Black {
		t.Errorf("loaded size %d, turn %d; want 9, black", e.Size(), turn)
	}
	if len(e.Moves()) != 2 {
		t.Fatalf("loaded %d moves, want 2", len(e.Moves()))
	}
	if err := e.Undo(); err != nil {
		t.Fatal(err)
	}
	if b, _ := e.Board(); x >= 0 && b[y][x] != Empty {
		t.Errorf("White's move %d, %d still on the board after undo", x, y)
	}
	// a handicap game saved and loaded
	if err := e.NewGame(9, 0.5, 3); err != nil {
		t.Fatal(err)
	}
	if e.Handicap() != 3 {
		t.Fatalf("handicap %d, want 3", e.Handicap())
	}
	if _, _, err := e.GenMove(White); err != nil {
		t.Fatal(err)
	}
	if err := e.SaveSGF("h.sgf"); err != nil {
		t.Fatal(err)
	}
	turn, err = e.LoadSGF("h.sgf")
	if err != nil || turn != Black || e.Handicap() != 3 || len(e.Moves()) != 1 {
		t.Errorf("handicap game loaded: turn %d, handicap %d, %d moves, %v", turn, e.Handicap(), len(e.Moves()), err)
	}
	score, err := e.FinalScore()
	if err != nil || !(strings.HasPrefix(score, "B+") || strings.HasPrefix(score, "W+") || score == "0") {
		t.Errorf("final score %q, %v", score, err)
	}
}
