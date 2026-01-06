package main

// CenteredMasterLayout implements a centered master layout
// Master windows are centered, stack windows fill the sides
type CenteredMasterLayout struct {
	MasterCount int
	MasterRatio float64 // Width ratio of master area
}

// NewCenteredMasterLayout creates a new centered master layout
func NewCenteredMasterLayout() *CenteredMasterLayout {
	return &CenteredMasterLayout{
		MasterCount: 1,
		MasterRatio: 0.6, // Master takes 60% width
	}
}

func (l *CenteredMasterLayout) Name() string {
	return "centered"
}

func (l *CenteredMasterLayout) Arrange(clients []*Client, area Rect) []Rect {
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

	// Only master windows - center them
	if stackCount == 0 {
		masterWidth := uint16(float64(area.Width) * l.MasterRatio)
		offsetX := (area.Width - masterWidth) / 2
		masterHeight := area.Height / uint16(masterCount)

		for i := 0; i < masterCount; i++ {
			rects[i] = Rect{
				X:      area.X + int16(offsetX),
				Y:      area.Y + int16(uint16(i)*masterHeight),
				Width:  masterWidth,
				Height: masterHeight,
			}
		}
		return rects
	}

	// Calculate widths
	masterWidth := uint16(float64(area.Width) * l.MasterRatio)
	sideWidth := (area.Width - masterWidth) / 2

	// Master area (centered)
	masterHeight := area.Height / uint16(masterCount)
	for i := 0; i < masterCount; i++ {
		rects[i] = Rect{
			X:      area.X + int16(sideWidth),
			Y:      area.Y + int16(uint16(i)*masterHeight),
			Width:  masterWidth,
			Height: masterHeight,
		}
	}

	// Stack windows alternate between left and right sides
	leftStack := make([]int, 0)
	rightStack := make([]int, 0)

	for i := masterCount; i < n; i++ {
		if (i-masterCount)%2 == 0 {
			leftStack = append(leftStack, i)
		} else {
			rightStack = append(rightStack, i)
		}
	}

	// Left side stack
	if len(leftStack) > 0 {
		leftHeight := area.Height / uint16(len(leftStack))
		for i, idx := range leftStack {
			rects[idx] = Rect{
				X:      area.X,
				Y:      area.Y + int16(uint16(i)*leftHeight),
				Width:  sideWidth,
				Height: leftHeight,
			}
		}
	}

	// Right side stack
	if len(rightStack) > 0 {
		rightHeight := area.Height / uint16(len(rightStack))
		for i, idx := range rightStack {
			rects[idx] = Rect{
				X:      area.X + int16(sideWidth) + int16(masterWidth),
				Y:      area.Y + int16(uint16(i)*rightHeight),
				Width:  sideWidth,
				Height: rightHeight,
			}
		}
	}

	return rects
}

func (l *CenteredMasterLayout) HandleMessage(msg LayoutMessage) {
	switch msg {
	case LayoutMsgShrink:
		if l.MasterRatio > 0.3 {
			l.MasterRatio -= 0.03
		}
	case LayoutMsgExpand:
		if l.MasterRatio < 0.85 {
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
