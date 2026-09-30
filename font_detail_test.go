package banner

import "testing"

func TestCompactMHasThreeSeparatedStems(t *testing.T) {
	layout, err := LayoutWordmark("mcpserver", WordmarkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := LayoutWordmark("mcpserver", WordmarkOptions{Font: CompactFont()})
	if err != nil || layout.Width != explicit.Width || layout.Gap < 1 {
		t.Fatal("compact layout did not fit", err)
	}
	var rows [9][8]bool
	for _, c := range layout.Cells {
		if c.Letter == 0 {
			x, y := c.X-layout.Left, c.Y-markTop
			if x < 0 || x >= 8 || y < 0 || y >= 9 {
				t.Fatal("m wider than eight columns")
			}
			rows[y][x] = true
		}
	}
	for y := 3; y < 7; y++ {
		for x := range 8 {
			want := x != 2 && x != 5
			if rows[y][x] != want {
				t.Fatalf("stem row %d column %d: lit=%v", y, x, rows[y][x])
			}
		}
	}
	for x := range 8 {
		if rows[2][x] != (x != 3 && x != 7) {
			t.Fatal("shoulders are not distinct")
		}
	}
	old := CompactFont()
	old['m'] = Glyph{"...", "...", "###", "###", "#.#", "#.#", "#.#", "...", "..."}
	previous, err := LayoutWordmark("mcpserver", WordmarkOptions{Font: old})
	if err != nil || layout.Width != previous.Width+2 {
		t.Fatal("m did not grow by exactly two columns", err)
	}
}

func TestHalfPixelColumns(t *testing.T) {
	layout, err := LayoutWordmark("x", WordmarkOptions{Font: map[rune]Glyph{'x': {"<>#"}}})
	if err != nil {
		t.Fatal(err)
	}
	expected := []int{0, 3, 4, 5}
	if layout.Width != 6 || len(layout.Cells) != len(expected) {
		t.Fatal("half-pixel layout dimensions")
	}
	for i, c := range layout.Cells {
		if c.X-layout.Left != expected[i] {
			t.Fatal("lit wrong half column")
		}
	}
}
