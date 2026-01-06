package main

// FullLayout implements a monocle/stacked layout
// All windows are maximized, only the focused one is visible
type FullLayout struct{}

func NewFullLayout() *FullLayout {
	return &FullLayout{}
}

func (l *FullLayout) Name() string {
	return "full"
}

func (l *FullLayout) Arrange(clients []*Client, area Rect) []Rect {
	rects := make([]Rect, len(clients))
	for i := range clients {
		rects[i] = area
	}
	return rects
}

func (l *FullLayout) HandleMessage(msg LayoutMessage) {
	// Full layout doesn't respond to resize messages
}

func (l *FullLayout) IsMonocle() bool {
	return true
}
