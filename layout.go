package main

// Layout defines how windows are arranged in a workspace
type Layout interface {
	Name() string
	Arrange(clients []*Client, area Rect) []Rect
	// HandleMessage processes layout-specific messages (resize, etc.)
	HandleMessage(msg LayoutMessage)
	// IsMonocle returns true if only the focused window should be visible
	IsMonocle() bool
}

// LayoutMessage is a message sent to layouts for modifications
type LayoutMessage int

const (
	LayoutMsgShrink LayoutMessage = iota
	LayoutMsgExpand
	LayoutMsgIncMaster
	LayoutMsgDecMaster
	LayoutMsgMirrorShrink
	LayoutMsgMirrorExpand
)
