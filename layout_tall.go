package main

// TallLayout implements a master/stack layout (like xmonad's Tall)
// Master windows on the left, stack on the right
type TallLayout struct {
	MasterCount int
	MasterRatio float64
}

// NewTallLayout creates a new tall layout with default settings
func NewTallLayout() *TallLayout {
	return &TallLayout{
		MasterCount: 1,
		MasterRatio: 0.5,
	}
}

func (l *TallLayout) Name() string {
	return "tall"
}

func (l *TallLayout) Arrange(clients []*Client, area Rect) []Rect {
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

	masterCount := l.MasterCount
	if masterCount > n {
		masterCount = n
	}
	if masterCount < 1 {
		masterCount = 1
	}
	stackCount := n - masterCount

	// Calculate widths
	masterWidth := uint16(float64(area.Width) * l.MasterRatio)
	stackWidth := area.Width - masterWidth

	// If no stack windows, master takes full width
	if stackCount == 0 {
		masterWidth = area.Width
	}

	// Arrange master windows (left side)
	if masterCount > 0 {
		masterHeight := area.Height / uint16(masterCount)
		for i := 0; i < masterCount; i++ {
			rects[i] = Rect{
				X:      area.X,
				Y:      area.Y + int16(uint16(i)*masterHeight),
				Width:  masterWidth,
				Height: masterHeight,
			}
		}
	}

	// Arrange stack windows (right side)
	if stackCount > 0 {
		stackHeight := area.Height / uint16(stackCount)
		for i := 0; i < stackCount; i++ {
			rects[masterCount+i] = Rect{
				X:      area.X + int16(masterWidth),
				Y:      area.Y + int16(uint16(i)*stackHeight),
				Width:  stackWidth,
				Height: stackHeight,
			}
		}
	}

	return rects
}

func (l *TallLayout) HandleMessage(msg LayoutMessage) {
	switch msg {
	case LayoutMsgShrink:
		if l.MasterRatio > 0.1 {
			l.MasterRatio -= 0.03
		}
	case LayoutMsgExpand:
		if l.MasterRatio < 0.9 {
			l.MasterRatio += 0.03
		}
	case LayoutMsgIncMaster:
		l.MasterCount++
	case LayoutMsgDecMaster:
		if l.MasterCount > 1 {
			l.MasterCount--
		}
	}
}
