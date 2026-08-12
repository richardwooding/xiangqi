# xiangqi

Xiangqi (Chinese chess) rules engine — pure logic, no protocol. The board is
9 files × 10 ranks = 90 points as a flat `[90]int8` (0 empty, positive red,
negative black; magnitude = piece type), so positions copy cheaply.

Full movement rules: general/advisor palace confinement, elephant river and
eye-block, horse leg-block, cannon screens, soldier promotion at the river,
and the flying-general rule; move validation rejects anything that leaves
your own general in check. Includes legal-move enumeration, checkmate/
stalemate detection (`Winner`), and coordinate/glyph helpers for notation.

```go
b := xiangqi.Start()
if err := xiangqi.Validate(b, xiangqi.Red, from, to); err == nil {
	b = xiangqi.Apply(b, from, to)
}
winner, over := xiangqi.Winner(b, toMove)
```

Deterministic, dependency-free, compiles to WASM. Extracted from
[kibitz](https://github.com/richardwooding/kibitz).

MIT licensed.
