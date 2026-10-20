package ui

import (
	"github.com/gdamore/tcell/v2"
)

// gridSpacings are the distances between points, in columns and rows, from
// the largest; a terminal cell is about twice as tall as it is wide.
var gridSpacings = [][2]int{{8, 4}, {6, 3}, {4, 2}, {2, 1}}

// starPoints are the marked points (hoshi) for each board size, as
// distances from the edge.
func starPoints(n int) map[[2]int]bool {
	var lines []int
	switch {
	case n >= 19:
		lines = []int{3, n / 2, n - 4}
	case n >= 13:
		lines = []int{3, n - 4}
	case n >= 9:
		lines = []int{2, n - 3}
	}
	stars := map[[2]int]bool{}
	for _, a := range lines {
		for _, b := range lines {
			stars[[2]int{a, b}] = true
		}
	}
	if n%2 == 1 && n >= 9 {
		stars[[2]int{n / 2, n / 2}] = true
	}
	return stars
}

// intersection is the line drawing for the point at column c, row r of an
// n by n board.
func intersection(c, r, n int, star bool) rune {
	top, bottom, left, right := r == 0, r == n-1, c == 0, c == n-1
	switch {
	case top && left:
		return '┌'
	case top && right:
		return '┐'
	case bottom && left:
		return '└'
	case bottom && right:
		return '┘'
	case top:
		return '┬'
	case bottom:
		return '┴'
	case left:
		return '├'
	case right:
		return '┤'
	case star:
		return '╋'
	}
	return '┼'
}

// drawGrid draws the board as lines and points, as big as fits in the box
// at x, y of the given size, centred, with coordinates.
func (g *GoBoardUI) drawGrid(screen tcell.Screen, x, y, width, height int) {
	n := g.BoardState.Width()
	if n == 0 {
		return
	}
	const left = 4 // room for the row numbers
	sx, sy := 1, 1
	for _, s := range gridSpacings {
		if left+(n-1)*s[0]+3 <= width && (n-1)*s[1]+3 <= height {
			sx, sy = s[0], s[1]
			break
		}
	}
	boardW, boardH := (n-1)*sx+1, (n-1)*sy+1
	// the board and its coordinates, centred; one cell of wood around
	ox := x + (width-(left+boardW+2))/2 + left + 1
	oy := y + (height-(boardH+3))/2 + 1
	colors := g.cfg.Theme.Colors
	wood := tcell.StyleDefault.Background(tcell.PaletteColor(colors.BoardColor)).
		Foreground(tcell.PaletteColor(colors.BoardColorAlt))
	for row := oy - 1; row <= oy+boardH; row++ {
		for col := ox - 2; col <= ox+boardW+1; col++ {
			screen.SetContent(col, row, ' ', nil, wood)
		}
	}
	stars := starPoints(n)
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			px, py := ox+c*sx, oy+r*sy
			// the lines to the next point right and down
			if c < n-1 {
				for i := 1; i < sx; i++ {
					screen.SetContent(px+i, py, '─', nil, wood)
				}
			}
			if r < n-1 {
				for i := 1; i < sy; i++ {
					screen.SetContent(px, py+i, '│', nil, wood)
				}
			}
			// the cursor or the last move, marked on the point
			var mark tcell.Color = tcell.ColorDefault
			if c == g.selX && r == g.selY {
				mark = tcell.PaletteColor(colors.CursorColorBG)
			} else if c == g.BoardState.LastMove.X && r == g.BoardState.LastMove.Y {
				mark = tcell.PaletteColor(colors.LastPlayedColorBG)
			}
			stone := g.BoardState.Board[r][c]
			if stone == 0 {
				style := wood
				if mark != tcell.ColorDefault {
					style = style.Background(mark)
				}
				screen.SetContent(px, py, intersection(c, r, n, stars[[2]int{c, r}]), nil, style)
				continue
			}
			color, symbol := colors.BlackColor, g.cfg.Theme.Symbols.BlackStone
			if stone == 2 {
				color, symbol = colors.WhiteColor, g.cfg.Theme.Symbols.WhiteStone
			}
			drawStone(screen, wood, tcell.PaletteColor(color), symbol, mark, px, py, sx, sy)
		}
	}
	// coordinates: letters under the board (no I), numbers to its left
	plain := tcell.StyleDefault
	highlight := plain.Background(tcell.PaletteColor(colors.CursorColorBG))
	for c := 0; c < n; c++ {
		letter := 'A' + rune(c)
		if c >= 8 {
			letter++
		}
		style := plain
		if c == g.selX {
			style = highlight
		}
		screen.SetContent(ox+c*sx, oy+boardH+1, letter, nil, style)
	}
	for r := 0; r < n; r++ {
		num := n - r
		style := plain
		if r == g.selY {
			style = highlight
		}
		if num >= 10 {
			screen.SetContent(ox-left, oy+r*sy, rune('0'+num/10), nil, style)
		}
		screen.SetContent(ox-left+1, oy+r*sy, rune('0'+num%10), nil, style)
	}
}

// drawStone draws a stone on the point at px, py: a disc of block
// characters when the points are far enough apart (5 by 3 cells from a
// spacing of 6 by 3, 3 by 1 from 4 by 2), else the theme's symbol. A mark
// (the cursor, the last move; tcell.ColorDefault for none) is a square of
// that colour in the middle of the stone.
func drawStone(screen tcell.Screen, wood tcell.Style, color tcell.Color, symbol rune, mark tcell.Color, px, py, sx, sy int) {
	stone := wood.Foreground(color)
	switch {
	case sx >= 6 && sy >= 3:
		for i := -1; i <= 1; i++ {
			screen.SetContent(px+i, py-1, '▄', nil, stone)
			screen.SetContent(px+i, py+1, '▀', nil, stone)
		}
		screen.SetContent(px-2, py, '▐', nil, stone)
		screen.SetContent(px+2, py, '▌', nil, stone)
		screen.SetContent(px-1, py, '█', nil, stone)
		screen.SetContent(px+1, py, '█', nil, stone)
	case sx >= 4 && sy >= 2:
		screen.SetContent(px-1, py, '▐', nil, stone)
		screen.SetContent(px+1, py, '▌', nil, stone)
	default:
		style := stone
		if mark != tcell.ColorDefault {
			style = style.Background(mark)
		}
		screen.SetContent(px, py, symbol, nil, style)
		return
	}
	// the middle of a disc: solid, or the mark
	if mark != tcell.ColorDefault {
		screen.SetContent(px, py, ' ', nil, wood.Background(mark))
	} else {
		screen.SetContent(px, py, '█', nil, stone)
	}
}
