package caro

import "testing"

func TestNewBoard(t *testing.T) {
	t.Parallel()

	b := NewBoard(3)
	if b.Size != 3 {
		t.Fatalf("expected board size 3, got %d", b.Size)
	}
	if len(b.Cells) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(b.Cells))
	}
	for y := range b.Cells {
		if len(b.Cells[y]) != 3 {
			t.Fatalf("expected row %d to have 3 columns, got %d", y, len(b.Cells[y]))
		}
		for x := range b.Cells[y] {
			if b.Cells[y][x] != '.' {
				t.Fatalf("expected empty cell '.' at (%d,%d), got %q", x, y, b.Cells[y][x])
			}
		}
	}
}

func TestApplySuccess(t *testing.T) {
	t.Parallel()

	b := NewBoard(5)
	err := b.Apply(Position{X: 2, Y: 3}, 'X')
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := b.Cells[2][1]; got != 'X' {
		t.Fatalf("expected cell (2,3) to be 'X', got %q", got)
	}
}

func TestApplyOutOfBounds(t *testing.T) {
	t.Parallel()

	testCases := []Position{
		{X: 0, Y: 1}, // x < 1
		{X: 1, Y: 0}, // y < 1
		{X: 4, Y: 1}, // x > size
		{X: 1, Y: 4}, // y > size
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.String(), func(t *testing.T) {
			t.Parallel()

			b := NewBoard(3)
			err := b.Apply(tc, 'O')
			if err != ErrOutOfBounds {
				t.Fatalf("expected ErrOutOfBounds, got %v", err)
			}
		})
	}
}

func TestApplyOccupied(t *testing.T) {
	t.Parallel()

	b := NewBoard(3)
	if err := b.Apply(Position{X: 1, Y: 1}, 'X'); err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}
	err := b.Apply(Position{X: 1, Y: 1}, 'O')
	if err != ErrOccupied {
		t.Fatalf("expected ErrOccupied, got %v", err)
	}
}

func TestSnapshot(t *testing.T) {
	t.Parallel()

	b := NewBoard(3)
	if err := b.Apply(Position{X: 1, Y: 1}, 'X'); err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}
	if err := b.Apply(Position{X: 3, Y: 2}, 'O'); err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	got := b.Snapshot()
	want := "X....O..."
	if got != want {
		t.Fatalf("snapshot mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestNewGame(t *testing.T) {
	t.Parallel()

	if g := NewGame([]string{"alice"}); g != nil {
		t.Fatalf("expected nil game for invalid players count")
	}

	g := NewGame([]string{"alice", "bob"})
	if g == nil {
		t.Fatalf("expected non-nil game for 2 players")
	}
	if g.board == nil {
		t.Fatalf("expected board to be initialized")
	}
	if g.board.Size != 15 {
		t.Fatalf("expected board size 15, got %d", g.board.Size)
	}
	if g.currentPlayerIndex != 0 {
		t.Fatalf("expected first player turn, got index %d", g.currentPlayerIndex)
	}
	if g.gameOver {
		t.Fatalf("expected gameOver false at start")
	}
	if g.gameResult != "" {
		t.Fatalf("expected empty gameResult at start, got %q", g.gameResult)
	}
}

func TestGameWinConditions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		moves []Position
	}{
		{
			name: "horizontal win",
			moves: []Position{
				{X: 1, Y: 1}, {X: 15, Y: 15},
				{X: 2, Y: 1}, {X: 15, Y: 14},
				{X: 3, Y: 1}, {X: 15, Y: 13},
				{X: 4, Y: 1}, {X: 15, Y: 12},
				{X: 5, Y: 1},
			},
			/*
				Final board state:
				+-----------------------------+
				| X X X X X . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				+-----------------------------+
			*/
		},
		{
			name: "vertical win",
			moves: []Position{
				{X: 1, Y: 1}, {X: 15, Y: 15},
				{X: 1, Y: 2}, {X: 15, Y: 14},
				{X: 1, Y: 3}, {X: 15, Y: 13},
				{X: 1, Y: 4}, {X: 15, Y: 12},
				{X: 1, Y: 5},
			},
			/*
				Final board state:
				+-----------------------------+
				| X . . . . . . . . . . . . . . |
				| X . . . . . . . . . . . . . . |
				| X . . . . . . . . . . . . . . |
				| X . . . . . . . . . . . . . . |
				| X . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				+-----------------------------+
			*/
		},
		{
			name: "diagonal down-right win",
			moves: []Position{
				{X: 1, Y: 1}, {X: 15, Y: 15},
				{X: 2, Y: 2}, {X: 15, Y: 14},
				{X: 3, Y: 3}, {X: 15, Y: 13},
				{X: 4, Y: 4}, {X: 15, Y: 12},
				{X: 5, Y: 5},
			},
			/*
				Final board state:
				+-----------------------------+
				| X . . . . . . . . . . . . . . |
				| . X . . . . . . . . . . . . . |
				| . . X . . . . . . . . . . . . |
				| . . . X . . . . . . . . . . . |
				| . . . . X . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				+-----------------------------+
			*/
		},
		{
			name: "diagonal up-right win",
			moves: []Position{
				{X: 1, Y: 5}, {X: 15, Y: 15},
				{X: 2, Y: 4}, {X: 15, Y: 14},
				{X: 3, Y: 3}, {X: 15, Y: 13},
				{X: 4, Y: 2}, {X: 15, Y: 12},
				{X: 5, Y: 1},
			},
			/*
				Final board state:
				+-----------------------------+
				| . . . . X . . . . . . . . . . |
				| . . . X . . . . . . . . . . . |
				| . . X . . . . . . . . . . . . |
				| . X . . . . . . . . . . . . . |
				| X . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . . |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				| . . . . . . . . . . . . . . O |
				+-----------------------------+
			*/
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewGame([]string{"alice", "bob"})
			for _, move := range tc.moves {
				if err := g.ApplyMove(move); err != nil {
					t.Fatalf("unexpected move error at %s: %v", move.String(), err)
				}
			}

			if !g.gameOver {
				t.Fatalf("expected gameOver true")
			}
			if g.gameResult != "alice" {
				t.Fatalf("expected winner alice, got %q", g.gameResult)
			}
		})
	}
}

func TestGameDoesNotWinAtFourInRow(t *testing.T) {
	t.Parallel()

	g := NewGame([]string{"alice", "bob"})
	moves := []Position{
		{X: 1, Y: 1}, {X: 15, Y: 15},
		{X: 2, Y: 1}, {X: 15, Y: 14},
		{X: 3, Y: 1}, {X: 15, Y: 13},
		{X: 4, Y: 1},
	}
	for _, move := range moves {
		if err := g.ApplyMove(move); err != nil {
			t.Fatalf("unexpected move error at %s: %v", move.String(), err)
		}
	}

	if g.gameOver {
		t.Fatalf("expected gameOver false with only 4 in a row")
	}
	if g.gameResult != "" {
		t.Fatalf("expected empty gameResult, got %q", g.gameResult)
	}
}

func TestGameWinsWhenLastMoveIsInMiddleOfStreak(t *testing.T) {
	t.Parallel()

	g := NewGame([]string{"alice", "bob"})
	// Alice builds four stones with one gap at x=2, then wins by filling that gap.
	moves := []Position{
		{X: 1, Y: 6}, {X: 10, Y: 10},
		{X: 2, Y: 6}, {X: 10, Y: 9},
		{X: 4, Y: 6}, {X: 10, Y: 8},
		{X: 5, Y: 6}, {X: 10, Y: 7},
		{X: 3, Y: 6},
	}
	/*
		Final board state:
		+-----------------------------+
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| X X X X X . . . . . . . . . . |
		| . . . . . . . . . O . . . . . |
		| . . . . . . . . . O . . . . . |
		| . . . . . . . . . O . . . . . |
		| . . . . . . . . . O . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		| . . . . . . . . . . . . . . . |
		+-----------------------------+
	*/
	for _, move := range moves {
		if err := g.ApplyMove(move); err != nil {
			t.Fatalf("unexpected move error at %s: %v", move.String(), err)
		}
	}

	if !g.gameOver {
		t.Fatalf("expected gameOver true when center gap is filled")
	}
	if g.gameResult != "alice" {
		t.Fatalf("expected winner alice, got %q", g.gameResult)
	}
}
