package main

// Rect represents a rectangular area on screen
type Rect struct {
	X, Y          int16
	Width, Height uint16
}

// Shrink returns a new Rect reduced by the given amount on all sides
func (r Rect) Shrink(amount uint16) Rect {
	return Rect{
		X:      r.X + int16(amount),
		Y:      r.Y + int16(amount),
		Width:  r.Width - 2*amount,
		Height: r.Height - 2*amount,
	}
}
