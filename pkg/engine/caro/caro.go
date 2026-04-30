// Package caro implements the Caro game engine.
// Caro is a two-player turn-based strategy game where players take turns placing their marks on an 8x8 grid.
// The first player to get 5 of their marks in a row (horizontally, vertically, or diagonally) wins.

package caro

import (
	"errors"
	"fmt"
	"strings"
)

var ErrOutOfBounds = errors.New("move out of bounds")
var ErrOccupied = errors.New("cell already occupied")

type Position struct {
	X int
	Y int
}

type Board struct {
	Size  int
	Cells [][]rune
}

func NewBoard(size int) *Board {
	cells := make([][]rune, size)
	for y := range cells {
		cells[y] = make([]rune, size)
		for x := range cells[y] {
			cells[y][x] = '.'
		}
	}

	return &Board{
		Size:  size,
		Cells: cells,
	}
}

func (b *Board) Apply(pos Position, mark rune) error {
	// Contract uses one-based coordinates: x,y ∈ 1..Size.
	if pos.X < 1 || pos.Y < 1 || pos.X > b.Size || pos.Y > b.Size {
		return ErrOutOfBounds
	}
	x := pos.X - 1
	y := pos.Y - 1

	if b.Cells[y][x] != '.' {
		return ErrOccupied
	}
	b.Cells[y][x] = mark
	return nil
}

func (b *Board) Snapshot() string {
	// Snapshot must be a single line for the bot protocol (runner -> bot).
	// Row-major order (y first, then x) with '.' for empty cells.
	parts := make([]string, 0, b.Size)
	for _, row := range b.Cells {
		parts = append(parts, string(row))
	}
	return strings.Join(parts, "")
}

func (p Position) String() string {
	return fmt.Sprintf("%d,%d", p.X, p.Y)
}

type Game struct {
	board              *Board
	players            []string
	currentPlayerIndex int
	turnCount          int
	gameOver           bool
	gameResult         string
}

func NewGame(players []string) *Game {
	if len(players) != 2 {
		return nil
	}
	return &Game{
		board:              NewBoard(8),
		players:            players,
		currentPlayerIndex: 0,
		turnCount:          0,
		gameOver:           false,
		gameResult:         "",
	}
}

var playerMarks = map[int]rune{0: 'X', 1: 'O'}

func (g *Game) ApplyMove(pos Position) error {
	prevPlayerIdx := g.currentPlayerIndex
	mark := playerMarks[prevPlayerIdx]
	if err := g.board.Apply(pos, mark); err != nil {
		return err
	}
	g.turnCount++
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)

	if g.isWinningMove(pos, mark) {
		g.gameOver = true
		g.gameResult = g.players[prevPlayerIdx]
		return nil
	}

	if isBoardFull(g.board.Cells) {
		g.gameOver = true
		g.gameResult = "draw"
	}
	return nil
}

func (g *Game) GetBoard() *Board {
	return g.board
}

func (g *Game) Snapshot() string {
	if g == nil || g.board == nil {
		return ""
	}
	return g.board.Snapshot()
}

func (g *Game) CurrentPlayer() string {
	if g == nil || len(g.players) == 0 {
		return ""
	}
	return g.players[g.currentPlayerIndex]
}

func (g *Game) IsOver() bool {
	return g != nil && g.gameOver
}

func (g *Game) Result() string {
	if g == nil {
		return ""
	}
	return g.gameResult
}

func (g *Game) MoveCount() int {
	if g == nil {
		return 0
	}
	return g.turnCount
}

func (g *Game) isWinningMove(pos Position, mark rune) bool {
	board := g.board
	if board == nil {
		return false
	}

	directions := [][2]int{
		{0, 1},  // vertical
		{1, 0},  // horizontal
		{1, 1},  // diagonal down-right
		{1, -1}, // diagonal up-right
	}

	// Convert one-based coordinates to internal 0-based indices.
	x0 := pos.X - 1
	y0 := pos.Y - 1

	for _, dir := range directions {
		count := 1
		// Positive direction
		x, y := x0, y0
		for i := 1; i < 5; i++ {
			nx, ny := x+dir[0]*i, y+dir[1]*i
			if nx < 0 || nx >= board.Size || ny < 0 || ny >= board.Size {
				break
			}
			if board.Cells[ny][nx] != mark {
				break
			}
			count++
		}
		// Negative direction
		for i := 1; i < 5; i++ {
			nx, ny := x-dir[0]*i, y-dir[1]*i
			if nx < 0 || nx >= board.Size || ny < 0 || ny >= board.Size {
				break
			}
			if board.Cells[ny][nx] != mark {
				break
			}
			count++
		}
		if count >= 5 {
			return true
		}
	}
	return false
}

func isBoardFull(cells [][]rune) bool {
	for _, row := range cells {
		for _, cell := range row {
			if cell == '.' {
				return false
			}
		}
	}
	return true
}
