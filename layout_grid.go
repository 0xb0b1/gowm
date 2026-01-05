package main

import "math"

// GridLayout arranges windows in an equal grid
type GridLayout struct{}

func NewGridLayout() *GridLayout {
	return &GridLayout{}
}

func (l *GridLayout) Name() string {
	return "grid"
}

func (l *GridLayout) Arrange(clients []*Client, area Rect) []Rect {
	n := len(clients)
	if n == 0 {
		return nil
	}

	// Calculate grid dimensions
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	rows := int(math.Ceil(float64(n) / float64(cols)))

	cellWidth := area.Width / uint16(cols)
	cellHeight := area.Height / uint16(rows)

	rects := make([]Rect, n)
	for i := 0; i < n; i++ {
		row := i / cols
		col := i % cols
		rects[i] = Rect{
			X:      area.X + int16(uint16(col)*cellWidth),
			Y:      area.Y + int16(uint16(row)*cellHeight),
			Width:  cellWidth,
			Height: cellHeight,
		}
	}
	return rects
}

func (l *GridLayout) HandleMessage(msg LayoutMessage) {
	// Grid layout doesn't respond to resize messages
}
