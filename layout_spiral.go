package main

// SpiralLayout implements a Fibonacci spiral layout
// Each window takes half of the remaining space, alternating direction
type SpiralLayout struct {
	Ratio float64 // Split ratio (default 0.5)
}

// NewSpiralLayout creates a new spiral layout
func NewSpiralLayout() *SpiralLayout {
	return &SpiralLayout{
		Ratio: 0.5,
	}
}

func (l *SpiralLayout) Name() string {
	return "spiral"
}

func (l *SpiralLayout) Arrange(clients []*Client, area Rect) []Rect {
	n := len(clients)
	if n == 0 {
		return nil
	}

	rects := make([]Rect, n)

	// Single window takes full area
	if n == 1 {
		rects[0] = area
		return rects
	}

	// Spiral arrangement: alternate between horizontal and vertical splits
	// Direction: right, down, left, up (clockwise spiral)
	remaining := area
	for i := 0; i < n; i++ {
		if i == n-1 {
			// Last window takes remaining space
			rects[i] = remaining
			break
		}

		direction := i % 4 // 0=right, 1=down, 2=left, 3=up

		switch direction {
		case 0: // Split horizontally, take left portion
			splitW := uint16(float64(remaining.Width) * l.Ratio)
			rects[i] = Rect{
				X:      remaining.X,
				Y:      remaining.Y,
				Width:  splitW,
				Height: remaining.Height,
			}
			remaining.X += int16(splitW)
			remaining.Width -= splitW

		case 1: // Split vertically, take top portion
			splitH := uint16(float64(remaining.Height) * l.Ratio)
			rects[i] = Rect{
				X:      remaining.X,
				Y:      remaining.Y,
				Width:  remaining.Width,
				Height: splitH,
			}
			remaining.Y += int16(splitH)
			remaining.Height -= splitH

		case 2: // Split horizontally, take right portion
			splitW := uint16(float64(remaining.Width) * l.Ratio)
			rects[i] = Rect{
				X:      remaining.X + int16(remaining.Width-splitW),
				Y:      remaining.Y,
				Width:  splitW,
				Height: remaining.Height,
			}
			remaining.Width -= splitW

		case 3: // Split vertically, take bottom portion
			splitH := uint16(float64(remaining.Height) * l.Ratio)
			rects[i] = Rect{
				X:      remaining.X,
				Y:      remaining.Y + int16(remaining.Height-splitH),
				Width:  remaining.Width,
				Height: splitH,
			}
			remaining.Height -= splitH
		}
	}

	return rects
}

func (l *SpiralLayout) HandleMessage(msg LayoutMessage) {
	switch msg {
	case LayoutMsgShrink:
		if l.Ratio > 0.2 {
			l.Ratio -= 0.05
		}
	case LayoutMsgExpand:
		if l.Ratio < 0.8 {
			l.Ratio += 0.05
		}
	}
}

func (l *SpiralLayout) IsMonocle() bool {
	return false
}
