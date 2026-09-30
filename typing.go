package banner

import (
	"errors"
	"time"
)

// TypingOptions controls typed input. Zero values select two characters per
// step, a 30ms hold, Plain text and Caret cursor style. Negative counts or holds
// are invalid. The cursor is a nonblank single-column cell so SVG can render it.
type TypingOptions struct {
	CharactersPerStep int
	StepHold          time.Duration
	TextStyle         Style
	CursorStyle       Style
}

// TypeLine animates one line in a copy of the supplied frame. It returns held
// intermediate frames and the completed frame without a hold. The caller owns
// the reading hold and scene transition. Existing content outside row y stays
// unchanged; the whole typed row is cleared before each step. Text must fit,
// including one cursor column, and use the single-column characters Frame supports.
func TypeLine(f Frame, x, y int, text string, opts TypingOptions) ([]Frame, Frame, error) {
	chars := []rune(text)
	if x < 0 || y < 0 || y >= Rows || x+len(chars) >= Cols {
		return nil, Frame{}, errors.New("banner: typed line and cursor must fit the terminal")
	}
	step := opts.CharactersPerStep
	if step == 0 {
		step = 2
	}
	hold := opts.StepHold
	if hold == 0 {
		hold = 30 * time.Millisecond
	}
	if step < 0 || hold < 0 {
		return nil, Frame{}, errors.New("banner: invalid typing step or hold")
	}
	cursor := opts.CursorStyle
	if cursor == Plain {
		cursor = Caret
	}
	var frames []Frame
	for i := 0; i <= len(chars); i += step {
		f.Clear(y, y+1)
		f.Text(x, y, string(chars[:i]), opts.TextStyle)
		f.Cells[y][x+i] = Cell{Rune: '▏', Style: cursor}
		f.Hold = hold
		frames = append(frames, f)
	}
	f.Clear(y, y+1)
	f.Text(x, y, text, opts.TextStyle)
	f.Hold = 0
	return frames, f, nil
}
