package main

import (
	"log"

	"github.com/jezek/xgb/xproto"
)

// DragState tracks mouse drag operations
type DragState struct {
	Active   bool
	Window   xproto.Window
	StartX   int16 // Mouse start position
	StartY   int16
	WinX     int16 // Window start position
	WinY     int16
	WinW     uint16 // Window start size
	WinH     uint16
	IsResize bool // true for resize, false for move
}

// grabMouseButtons sets up mouse button grabs for window operations
func (wm *WindowManager) grabMouseButtons(win xproto.Window) {
	// Grab Super+Button1 for move
	xproto.GrabButton(wm.conn, true, win,
		xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskPointerMotion,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
		wm.root, 0,
		xproto.ButtonIndex1, xproto.ModMask4)

	// Grab Super+Button3 for resize
	xproto.GrabButton(wm.conn, true, win,
		xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskPointerMotion,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
		wm.root, 0,
		xproto.ButtonIndex3, xproto.ModMask4)
}

// handleButtonPress handles mouse button press events
func (wm *WindowManager) handleButtonPress(e xproto.ButtonPressEvent) {
	client, exists := wm.clients[e.Event]
	if !exists {
		return
	}

	// Only allow move/resize on floating windows
	if !client.Floating {
		// Make window floating first if trying to move/resize tiled window
		client.Floating = true
		wm.tile()
	}

	// Focus the window
	wm.focus(client)

	// Check for Super modifier (Mod4)
	if e.State&xproto.ModMask4 == 0 {
		return
	}

	// Get current window geometry
	geom, err := xproto.GetGeometry(wm.conn, xproto.Drawable(e.Event)).Reply()
	if err != nil {
		return
	}

	wm.drag = DragState{
		Active:   true,
		Window:   e.Event,
		StartX:   e.RootX,
		StartY:   e.RootY,
		WinX:     geom.X,
		WinY:     geom.Y,
		WinW:     geom.Width,
		WinH:     geom.Height,
		IsResize: e.Detail == xproto.ButtonIndex3, // Button3 = resize
	}

	// Raise window to top
	xproto.ConfigureWindow(wm.conn, e.Event,
		xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove})

	if wm.drag.IsResize {
		log.Printf("Starting resize on window %d", e.Event)
	} else {
		log.Printf("Starting move on window %d", e.Event)
	}
}

// handleButtonRelease handles mouse button release events
func (wm *WindowManager) handleButtonRelease(e xproto.ButtonReleaseEvent) {
	if !wm.drag.Active {
		return
	}

	// Update client geometry
	if client, exists := wm.clients[wm.drag.Window]; exists {
		geom, err := xproto.GetGeometry(wm.conn, xproto.Drawable(wm.drag.Window)).Reply()
		if err == nil {
			client.X = geom.X
			client.Y = geom.Y
			client.Width = geom.Width
			client.Height = geom.Height
		}
	}

	wm.drag = DragState{}
	log.Println("Drag operation ended")
}

// handleMotionNotify handles mouse motion events
func (wm *WindowManager) handleMotionNotify(e xproto.MotionNotifyEvent) {
	if !wm.drag.Active {
		return
	}

	// Calculate delta from drag start
	dx := e.RootX - wm.drag.StartX
	dy := e.RootY - wm.drag.StartY

	if wm.drag.IsResize {
		// Resize: adjust width and height
		newW := int32(wm.drag.WinW) + int32(dx)
		newH := int32(wm.drag.WinH) + int32(dy)

		// Minimum size
		if newW < 100 {
			newW = 100
		}
		if newH < 100 {
			newH = 100
		}

		xproto.ConfigureWindow(wm.conn, wm.drag.Window,
			xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
			[]uint32{uint32(newW), uint32(newH)})
	} else {
		// Move: adjust position
		newX := int32(wm.drag.WinX) + int32(dx)
		newY := int32(wm.drag.WinY) + int32(dy)

		xproto.ConfigureWindow(wm.conn, wm.drag.Window,
			xproto.ConfigWindowX|xproto.ConfigWindowY,
			[]uint32{uint32(newX), uint32(newY)})
	}
}
