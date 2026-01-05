package main

import (
	"github.com/jezek/xgb/xproto"
)

// Client represents a managed window
type Client struct {
	Window    xproto.Window
	X, Y      int16
	Width     uint16
	Height    uint16
	Mapped    bool
	Floating  bool
	Workspace int
	Urgent    bool // Window requests attention
}

// Geometry returns the client's current geometry as a Rect
func (c *Client) Geometry() Rect {
	return Rect{
		X:      c.X,
		Y:      c.Y,
		Width:  c.Width,
		Height: c.Height,
	}
}
