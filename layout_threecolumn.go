package main

// ThreeColumnLayout implements a three-column layout
// Master in the center, stacks on left and right
type ThreeColumnLayout struct {
	MasterCount int
	MasterRatio float64 // Width ratio of center column
}

// NewThreeColumnLayout creates a new three-column layout
func NewThreeColumnLayout() *ThreeColumnLayout {
	return &ThreeColumnLayout{
		MasterCount: 1,
		MasterRatio: 0.5, // Center takes 50%, sides take 25% each
	}
}

func (l *ThreeColumnLayout) Name() string {
	return "threecol"
}

func (l *ThreeColumnLayout) Arrange(clients []*Client, area Rect) []Rect {
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

	// Two windows: master center, one on right
	if n == 2 {
		masterWidth := uint16(float64(area.Width) * l.MasterRatio)
		sideWidth := (area.Width - masterWidth) / 2

		// Master in center
		rects[0] = Rect{
			X:      area.X + int16(sideWidth),
			Y:      area.Y,
			Width:  masterWidth,
			Height: area.Height,
		}
		// Second window on right
		rects[1] = Rect{
			X:      area.X + int16(sideWidth) + int16(masterWidth),
			Y:      area.Y,
			Width:  sideWidth,
			Height: area.Height,
		}
		return rects
	}

	masterCount := l.MasterCount
	if masterCount > n {
		masterCount = n
	}
	if masterCount < 1 {
		masterCount = 1
	}

	// Calculate column widths
	masterWidth := uint16(float64(area.Width) * l.MasterRatio)
	sideWidth := (area.Width - masterWidth) / 2

	// Distribute non-master windows between left and right columns
	stackCount := n - masterCount
	leftCount := stackCount / 2
	rightCount := stackCount - leftCount

	idx := 0

	// Left column
	if leftCount > 0 {
		leftHeight := area.Height / uint16(leftCount)
		for i := 0; i < leftCount; i++ {
			rects[masterCount+i] = Rect{
				X:      area.X,
				Y:      area.Y + int16(uint16(i)*leftHeight),
				Width:  sideWidth,
				Height: leftHeight,
			}
		}
		idx = leftCount
	}

	// Center column (master)
	masterHeight := area.Height / uint16(masterCount)
	for i := 0; i < masterCount; i++ {
		rects[i] = Rect{
			X:      area.X + int16(sideWidth),
			Y:      area.Y + int16(uint16(i)*masterHeight),
			Width:  masterWidth,
			Height: masterHeight,
		}
	}

	// Right column
	if rightCount > 0 {
		rightHeight := area.Height / uint16(rightCount)
		for i := 0; i < rightCount; i++ {
			rects[masterCount+idx+i] = Rect{
				X:      area.X + int16(sideWidth) + int16(masterWidth),
				Y:      area.Y + int16(uint16(i)*rightHeight),
				Width:  sideWidth,
				Height: rightHeight,
			}
		}
	}

	return rects
}

func (l *ThreeColumnLayout) HandleMessage(msg LayoutMessage) {
	switch msg {
	case LayoutMsgShrink:
		if l.MasterRatio > 0.2 {
			l.MasterRatio -= 0.03
		}
	case LayoutMsgExpand:
		if l.MasterRatio < 0.8 {
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
