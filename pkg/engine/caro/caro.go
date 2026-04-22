// Package caro implements the Caro game engine.
// Caro is a two-player turn-based strategy game where players take turns placing their marks on a 10x10 grid.
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
	if pos.X < 0 || pos.Y < 0 || pos.X >= b.Size || pos.Y >= b.Size {
		return ErrOutOfBounds
	}
	if b.Cells[pos.Y][pos.X] != '.' {
		return ErrOccupied
	}
	b.Cells[pos.Y][pos.X] = mark
	return nil
}

func (b *Board) Snapshot() string {
	lines := make([]string, 0, b.Size)
	for _, row := range b.Cells {
		lines = append(lines, string(row))
	}
	return strings.Join(lines, "\n")
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

var playerMarks = map[int]rune{0: 'X', 1: 'O'}

func NewGame(players []string) *Game {
	if len(players) != 2 {
		return nil
	}
	return &Game{
		board:              NewBoard(10),
		players:            players,
		currentPlayerIndex: 0,
		turnCount:          0,
		gameOver:           false,
		gameResult:         "",
	}
}

func (g *Game) ApplyMove(pos Position) error {
	prevPlayerIdx := g.currentPlayerIndex
	if err := g.board.Apply(pos, playerMarks[g.currentPlayerIndex]); err != nil {
		return err
	}
	g.turnCount++
	if g.turnCount >= 100 {
		g.gameOver = true
		g.gameResult = "draw"
	}
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)
	if g.isWinningMove(pos) {
		g.gameOver = true
		g.gameResult = g.players[prevPlayerIdx]
	}
	return nil
}

func (g *Game) GetBoard() *Board {
	return g.board
}

func (g *Game) isWinningMove(pos Position) bool {
	board := g.board
	if board == nil {
		return false
	}

	// The player who made this move is the previous player.
	// Because in ApplyMove, currentPlayerIndex is updated after a move.
	prevPlayerIdx := (g.currentPlayerIndex + len(g.players) - 1) % len(g.players)
	mark := playerMarks[prevPlayerIdx]

	directions := [][2]int{
		{0, 1},  // vertical
		{1, 0},  // horizontal
		{1, 1},  // diagonal down-right
		{1, -1}, // diagonal up-right
	}

	for _, dir := range directions {
		count := 1
		// Positive direction
		x, y := pos.X, pos.Y
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
