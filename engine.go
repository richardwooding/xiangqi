// Xiangqi (Chinese chess) rules — pure logic, no protocol. The board is 9
// files (0..8) × 10 ranks (0..9) = 90 points, idx = rank*9 + file. Red (P1)
// starts on ranks 0..4 (bottom), Black (P2) on ranks 5..9 (top). Board cells:
// 0 empty, POSITIVE = red, NEGATIVE = black; magnitude = piece type.
package xiangqi

import (
	"errors"
	"fmt"
	"slices"
)

// Board is the 90-point Xiangqi board (idx = rank*9 + file).
type Board [90]int8

// Piece types (magnitude of a board cell).
const (
	General  int8 = 1
	Advisor  int8 = 2
	Elephant int8 = 3
	Horse    int8 = 4
	Chariot  int8 = 5
	Cannon   int8 = 6
	Soldier  int8 = 7
)

// Side signs used by the engine: red = +1, black = -1.
const (
	Red   int8 = 1
	Black int8 = -1
)

var (
	// ErrOffBoard is a from/to index outside 0..89.
	ErrOffBoard = errors.New("xiangqi: off the board")
	// ErrNotYourPiece is a from square that is empty or holds an enemy piece.
	ErrNotYourPiece = errors.New("xiangqi: not your piece")
	// ErrIllegalMove is a destination the piece cannot legally reach.
	ErrIllegalMove = errors.New("xiangqi: illegal move")
	// ErrLeavesInCheck is a move that would leave the mover's general in check.
	ErrLeavesInCheck = errors.New("xiangqi: leaves general in check")
)

var (
	orthoDirs = [4][2]int8{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	diagDirs  = [4][2]int8{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	// horseMoves: {legRank, legFile, destRank, destFile} offsets. The leg must
	// be empty for the destination to be reachable.
	horseMoves = [8][4]int8{
		{1, 0, 2, 1}, {1, 0, 2, -1},
		{-1, 0, -2, 1}, {-1, 0, -2, -1},
		{0, 1, 1, 2}, {0, 1, -1, 2},
		{0, -1, 1, -2}, {0, -1, -1, -2},
	}
	redGlyphs   = [8]string{"", "帥", "仕", "相", "馬", "車", "炮", "兵"}
	blackGlyphs = [8]string{"", "將", "士", "象", "馬", "車", "砲", "卒"}
)

func RankOf(idx int8) int8   { return idx / 9 }
func FileOf(idx int8) int8   { return idx % 9 }
func IdxOf(r, f int8) int8   { return r*9 + f }
func onBoard(r, f int8) bool { return r >= 0 && r < 10 && f >= 0 && f < 9 }
func Sign(p int8) int8 {
	switch {
	case p > 0:
		return 1
	case p < 0:
		return -1
	}
	return 0
}

func Abs8(p int8) int8 {
	if p < 0 {
		return -p
	}
	return p
}

// friendly reports whether (r,f) is on the board and holds a piece of side.
func friendly(b Board, r, f, side int8) bool {
	if !onBoard(r, f) {
		return false
	}
	v := b[IdxOf(r, f)]
	return v != 0 && Sign(v) == side
}

// inPalace reports whether (r,f) is inside side's palace (files 3..5; red ranks
// 0..2, black ranks 7..9).
func inPalace(r, f, side int8) bool {
	if f < 3 || f > 5 {
		return false
	}
	if side == Red {
		return r >= 0 && r <= 2
	}
	return r >= 7 && r <= 9
}

// ownHalf reports whether rank r is on side's half of the river (red 0..4,
// black 5..9) — used to keep elephants from crossing.
func ownHalf(r, side int8) bool {
	if side == Red {
		return r <= 4
	}
	return r >= 5
}

// Start returns the standard opening position.
func Start() Board {
	var b Board
	back := [9]int8{Chariot, Horse, Elephant, Advisor, General, Advisor, Elephant, Horse, Chariot}
	for f := range int8(9) {
		b[IdxOf(0, f)] = back[f]
		b[IdxOf(9, f)] = -back[f]
	}
	b[IdxOf(2, 1)] = Cannon
	b[IdxOf(2, 7)] = Cannon
	b[IdxOf(7, 1)] = -Cannon
	b[IdxOf(7, 7)] = -Cannon
	for f := int8(0); f < 9; f += 2 {
		b[IdxOf(3, f)] = Soldier
		b[IdxOf(6, f)] = -Soldier
	}
	return b
}

// Apply returns the board after moving the piece at from to to (no validation).
func Apply(b Board, from, to int8) Board {
	b[to] = b[from]
	b[from] = 0
	return b
}

// pieceMoves returns the pseudo-legal destination squares for the piece at
// from (ignores whether the move leaves the mover's general in check).
func pieceMoves(b Board, from int8) []int8 {
	p := b[from]
	side := Sign(p)
	r, f := RankOf(from), FileOf(from)
	switch Abs8(p) {
	case General:
		return generalMoves(b, r, f, side)
	case Advisor:
		return advisorMoves(b, r, f, side)
	case Elephant:
		return elephantMoves(b, r, f, side)
	case Horse:
		return horseMovesFrom(b, r, f, side)
	case Chariot:
		return chariotMoves(b, r, f, side)
	case Cannon:
		return cannonMoves(b, r, f, side)
	case Soldier:
		return soldierMoves(b, r, f, side)
	}
	return nil
}

func generalMoves(b Board, r, f, side int8) []int8 {
	var out []int8
	for _, d := range orthoDirs {
		nr, nf := r+d[0], f+d[1]
		if inPalace(nr, nf, side) && !friendly(b, nr, nf, side) {
			out = append(out, IdxOf(nr, nf))
		}
	}
	return out
}

func advisorMoves(b Board, r, f, side int8) []int8 {
	var out []int8
	for _, d := range diagDirs {
		nr, nf := r+d[0], f+d[1]
		if inPalace(nr, nf, side) && !friendly(b, nr, nf, side) {
			out = append(out, IdxOf(nr, nf))
		}
	}
	return out
}

func elephantMoves(b Board, r, f, side int8) []int8 {
	var out []int8
	for _, d := range diagDirs {
		er, ef := r+d[0], f+d[1]     // the "eye" (midpoint)
		nr, nf := r+2*d[0], f+2*d[1] // destination
		if !onBoard(nr, nf) || !ownHalf(nr, side) {
			continue
		}
		if b[IdxOf(er, ef)] != 0 { // eye blocked
			continue
		}
		if !friendly(b, nr, nf, side) {
			out = append(out, IdxOf(nr, nf))
		}
	}
	return out
}

func horseMovesFrom(b Board, r, f, side int8) []int8 {
	var out []int8
	for _, m := range horseMoves {
		lr, lf := r+m[0], f+m[1] // the leg
		if !onBoard(lr, lf) || b[IdxOf(lr, lf)] != 0 {
			continue // leg off-board or blocked
		}
		nr, nf := r+m[2], f+m[3]
		if onBoard(nr, nf) && !friendly(b, nr, nf, side) {
			out = append(out, IdxOf(nr, nf))
		}
	}
	return out
}

func chariotMoves(b Board, r, f, side int8) []int8 {
	var out []int8
	for _, d := range orthoDirs {
		nr, nf := r+d[0], f+d[1]
		for onBoard(nr, nf) {
			v := b[IdxOf(nr, nf)]
			if v == 0 {
				out = append(out, IdxOf(nr, nf))
			} else {
				if Sign(v) != side {
					out = append(out, IdxOf(nr, nf))
				}
				break
			}
			nr, nf = nr+d[0], nf+d[1]
		}
	}
	return out
}

func cannonMoves(b Board, r, f, side int8) []int8 {
	var out []int8
	for _, d := range orthoDirs {
		out = cannonRay(b, r, f, side, d, out)
	}
	return out
}

// cannonRay walks one direction: empty squares are quiet moves; the first piece
// is the screen; the first piece beyond the screen is capturable if it is an
// enemy.
func cannonRay(b Board, r, f, side int8, d [2]int8, out []int8) []int8 {
	nr, nf := r+d[0], f+d[1]
	for onBoard(nr, nf) && b[IdxOf(nr, nf)] == 0 {
		out = append(out, IdxOf(nr, nf))
		nr, nf = nr+d[0], nf+d[1]
	}
	if !onBoard(nr, nf) {
		return out // no screen
	}
	nr, nf = nr+d[0], nf+d[1] // skip the screen
	for onBoard(nr, nf) {
		v := b[IdxOf(nr, nf)]
		if v != 0 {
			if Sign(v) != side {
				out = append(out, IdxOf(nr, nf))
			}
			break
		}
		nr, nf = nr+d[0], nf+d[1]
	}
	return out
}

func soldierMoves(b Board, r, f, side int8) []int8 {
	var out []int8
	// Forward one: red advances up (+rank), black down (-rank).
	nr := r + side
	if onBoard(nr, f) && !friendly(b, nr, f, side) {
		out = append(out, IdxOf(nr, f))
	}
	// After crossing the river a soldier may also step sideways.
	crossed := (side == Red && r >= 5) || (side == Black && r <= 4)
	if crossed {
		for _, df := range [2]int8{-1, 1} {
			if onBoard(r, f+df) && !friendly(b, r, f+df, side) {
				out = append(out, IdxOf(r, f+df))
			}
		}
	}
	return out
}

// findGeneral returns the board index of side's general, or -1 if absent.
func findGeneral(b Board, side int8) int8 {
	want := General * side
	for i := range int8(90) {
		if b[i] == want {
			return i
		}
	}
	return -1
}

// generalsFacing reports whether the two generals share a file with no pieces
// between them (the illegal "flying general" position).
func generalsFacing(b Board) bool {
	rg, bg := findGeneral(b, Red), findGeneral(b, Black)
	if rg < 0 || bg < 0 || FileOf(rg) != FileOf(bg) {
		return false
	}
	f := FileOf(rg)
	lo, hi := RankOf(rg), RankOf(bg)
	if lo > hi {
		lo, hi = hi, lo
	}
	for r := lo + 1; r < hi; r++ {
		if b[IdxOf(r, f)] != 0 {
			return false
		}
	}
	return true
}

// attackedBy reports whether any piece of side `by` pseudo-attacks target.
func attackedBy(b Board, by, target int8) bool {
	for i := range int8(90) {
		if b[i] == 0 || Sign(b[i]) != by {
			continue
		}
		if slices.Contains(pieceMoves(b, i), target) {
			return true
		}
	}
	return false
}

// InCheck reports whether side's general is attacked, or the two generals face
// each other on an open file (both are "in check" for the flying-general rule).
func InCheck(b Board, side int8) bool {
	if generalsFacing(b) {
		return true
	}
	g := findGeneral(b, side)
	if g < 0 {
		return true
	}
	return attackedBy(b, -side, g)
}

// LegalMoves returns every legal {from,to} for side (+1 red, -1 black): a
// pseudo-legal move that does not leave the mover's own general in check.
func LegalMoves(b Board, side int8) [][2]int8 {
	var out [][2]int8
	for i := range int8(90) {
		if b[i] == 0 || Sign(b[i]) != side {
			continue
		}
		for _, t := range pieceMoves(b, i) {
			if !InCheck(Apply(b, i, t), side) {
				out = append(out, [2]int8{i, t})
			}
		}
	}
	return out
}

// Validate checks a single move for side, returning nil if legal.
func Validate(b Board, side, from, to int8) error {
	if from < 0 || from >= 90 || to < 0 || to >= 90 {
		return ErrOffBoard
	}
	if b[from] == 0 || Sign(b[from]) != side {
		return ErrNotYourPiece
	}
	found := slices.Contains(pieceMoves(b, from), to)
	if !found {
		return ErrIllegalMove
	}
	if InCheck(Apply(b, from, to), side) {
		return ErrLeavesInCheck
	}
	return nil
}

// Winner reports the outcome for the position with toMove to play: the side to
// move with no legal moves loses (checkmate or stalemate both lose in Xiangqi).
// Returns (winning side +1/-1, true) when over, else (0, false).
func Winner(b Board, toMove int8) (int8, bool) {
	if len(LegalMoves(b, toMove)) == 0 {
		return -toMove, true
	}
	return 0, false
}

// Coord renders a board index as file letter (a..i) + 1-based rank ("e1"..).
func Coord(idx int8) string {
	return fmt.Sprintf("%c%d", 'a'+FileOf(idx), RankOf(idx)+1)
}

// Glyph returns the Chinese glyph for a signed piece value ("" for empty).
func Glyph(p int8) string {
	t := Abs8(p)
	if t < 1 || t > 7 {
		return ""
	}
	if p > 0 {
		return redGlyphs[t]
	}
	return blackGlyphs[t]
}
