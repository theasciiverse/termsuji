package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lvank/termsuji/api"
	"github.com/lvank/termsuji/engine"
)

// SaveName is the file a local game is kept in while it is played, in the
// engine's folder (see LocalSettings).
const SaveName = "game.sgf"

// LocalSettings is how a local game was set up; it is saved beside the
// game, so a game continued later is played the same way.
type LocalSettings struct {
	Size     int     `json:"size"`
	Human    int     `json:"human"` // engine.Black or engine.White
	Level    int     `json:"level"` // GNU Go's strength, 1 to 10
	Handicap int     `json:"handicap"`
	Komi     float64 `json:"komi"`
}

// localGame is a game against GNU Go on this computer.
type localGame struct {
	eng      *engine.Engine
	settings LocalSettings
	dir      string // where the game is saved
	thinking bool   // GNU Go is choosing its move
	over     bool
	outcome  string
	message  string // the last move's news: an illegal move, a pass
}

func (l *localGame) engineColour() int {
	return engine.Black + engine.White - l.settings.Human
}

func colourWord(c int) string {
	if c == engine.White {
		return "White"
	}
	return "Black"
}

// toMove is whose turn it is.
func (l *localGame) toMove() int {
	moves := l.eng.Moves()
	if len(moves) == 0 {
		if l.eng.Handicap() > 0 {
			return engine.White
		}
		return engine.Black
	}
	return engine.Black + engine.White - moves[len(moves)-1].Colour
}

// StartLocal plays a local game with eng, already set up (a new game, or one
// loaded with LoadSGF), saved in dir. If it is GNU Go's turn, it moves.
func (g *GoBoardUI) StartLocal(eng *engine.Engine, s LocalSettings, dir string) {
	g.Close()
	g.local = &localGame{eng: eng, settings: s, dir: dir}
	g.ResetSelection()
	g.refreshLocal()
	if g.local.toMove() == g.local.engineColour() {
		g.engineMove()
	}
}

// LocalGame reports whether a local game is shown.
func (g *GoBoardUI) LocalGame() bool {
	return g.local != nil
}

// playLocal plays the player's move at x, y (engine.Pass to pass), then
// GNU Go's.
func (g *GoBoardUI) playLocal(x, y int) {
	l := g.local
	if l.thinking || l.over || l.toMove() != l.settings.Human {
		return
	}
	if err := l.eng.Play(l.settings.Human, x, y); err != nil {
		l.message = "You can't play there: " + err.Error() + "."
		g.refreshLocal()
		return
	}
	l.message = ""
	if x == engine.Pass {
		l.message = "You passed."
	}
	g.afterMove()
	if !l.over {
		g.engineMove()
	}
}

// engineMove lets GNU Go choose its move in the background.
func (g *GoBoardUI) engineMove() {
	l := g.local
	l.thinking = true
	g.refreshLocal()
	go func() {
		x, _, err := l.eng.GenMove(l.engineColour())
		g.app.QueueUpdateDraw(func() {
			if g.local != l {
				return // the game was left while GNU Go thought
			}
			l.thinking = false
			switch {
			case err != nil:
				l.message = "GNU Go failed: " + err.Error()
			case x == engine.Resign:
				l.over = true
				l.outcome = fmt.Sprintf("GNU Go resigns: %s wins.", colourWord(l.settings.Human))
				l.forget()
			case x == engine.Pass:
				l.message = "GNU Go passed."
			}
			g.afterMove()
		})
	}()
}

// afterMove saves the game and ends it after two passes in a row.
func (g *GoBoardUI) afterMove() {
	l := g.local
	moves := l.eng.Moves()
	n := len(moves)
	if !l.over && n >= 2 && moves[n-1].X == engine.Pass && moves[n-2].X == engine.Pass {
		l.over = true
		score, err := l.eng.FinalScore()
		if err != nil {
			l.outcome = "Both passed.\nGNU Go could not count the game: " + err.Error()
		} else {
			l.outcome = "Both passed.\n" + scoreWords(score)
		}
		l.forget()
	}
	if !l.over {
		l.eng.SaveSGF(SaveName)
	}
	g.refreshLocal()
}

// scoreWords turns a GTP score such as "W+12.5" into words.
func scoreWords(score string) string {
	switch {
	case score == "0":
		return "The game is a draw (jigo)."
	case strings.HasPrefix(score, "B+"):
		return fmt.Sprintf("Black wins by %s points.", score[2:])
	case strings.HasPrefix(score, "W+"):
		return fmt.Sprintf("White wins by %s points.", score[2:])
	}
	return "Result: " + score
}

// forget removes the saved game once it is over.
func (l *localGame) forget() {
	os.Remove(filepath.Join(l.dir, SaveName))
}

// UndoLocal takes back the player's last move, and GNU Go's after it.
func (g *GoBoardUI) UndoLocal() {
	l := g.local
	if l == nil || l.thinking || l.over {
		return
	}
	moves := l.eng.Moves()
	if len(moves) > 0 && moves[len(moves)-1].Colour == l.engineColour() {
		l.eng.Undo()
		moves = l.eng.Moves()
	}
	if len(moves) > 0 && moves[len(moves)-1].Colour == l.settings.Human {
		l.eng.Undo()
	}
	l.message = "Move taken back."
	l.eng.SaveSGF(SaveName)
	g.refreshLocal()
}

// closeLocal leaves the local game (it stays saved, unless it is over).
func (g *GoBoardUI) closeLocal() {
	if g.local == nil {
		return
	}
	l := g.local
	g.local = nil
	go l.eng.Close()
}

// refreshLocal rebuilds the board and the hint from the engine.
func (g *GoBoardUI) refreshLocal() {
	l := g.local
	board, err := l.eng.Board()
	if err != nil {
		l.message = "GNU Go failed: " + err.Error()
		board = g.BoardState.Board
	}
	state := &api.BoardState{Board: board, Phase: "play"}
	state.LastMove.X, state.LastMove.Y = -1, -1
	moves := l.eng.Moves()
	if n := len(moves); n > 0 && moves[n-1].X >= 0 {
		state.LastMove.X, state.LastMove.Y = moves[n-1].X, moves[n-1].Y
	}
	if l.over {
		state.Phase = "finished"
		state.Outcome = l.outcome
		g.ResetSelection()
	}
	g.BoardState = state

	you, them := l.settings.Human, l.engineColour()
	var status string
	switch {
	case l.over:
		status = "The game is over.\n" + l.outcome + "\n\nn: new game\nq: menu"
	case l.thinking:
		status = "GNU Go is thinking..."
	default:
		status = "It is your turn."
	}
	if l.message != "" && !l.over {
		status = l.message + "\n" + status
	}
	keys := ""
	if !l.over {
		keys = "\n\narrow keys: move cursor\nReturn: play move\np: pass turn\nu: take back a move\nq: menu (the game is saved)"
	}
	g.hint.SetText(fmt.Sprintf("You: %s    GNU Go (level %d): %s\nCaptured: you %d, GNU Go %d\nMove %d\n\n%s%s",
		colourWord(you), l.settings.Level, colourWord(them),
		l.eng.Captures(you), l.eng.Captures(them), len(moves)+1, status, keys))
}
