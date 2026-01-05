package main

import (
	"log"
	"os/exec"
	"strings"
	"syscall"

	"github.com/jezek/xgb/xproto"
)

// Scratchpad represents a toggle-able floating window
type Scratchpad struct {
	Command string // Command to spawn (e.g., "kitty --class scratchpad")
	Class   string // WM_CLASS to identify the window
	Width   uint16 // Width as percentage of screen (0-100)
	Height  uint16 // Height as percentage of screen (0-100)

	window  xproto.Window // The actual window (0 if not spawned)
	visible bool          // Whether currently visible
}

// DefaultScratchpad returns the default scratchpad configuration
func DefaultScratchpad() *Scratchpad {
	return &Scratchpad{
		Command: "kitty --class scratchpad",
		Class:   "scratchpad",
		Width:   70,
		Height:  60,
	}
}

// Toggle shows or hides the scratchpad
func (wm *WindowManager) ToggleScratchpad() {
	sp := wm.scratchpad

	// If we have a window, toggle visibility
	if sp.window != 0 {
		// Check if window still exists
		_, err := xproto.GetWindowAttributes(wm.conn, sp.window).Reply()
		if err != nil {
			// Window was destroyed, reset
			sp.window = 0
			sp.visible = false
		}
	}

	if sp.window != 0 {
		if sp.visible {
			// Hide it
			xproto.UnmapWindow(wm.conn, sp.window)
			sp.visible = false
			log.Println("Scratchpad hidden")
		} else {
			// Show it
			wm.showScratchpad()
		}
	} else {
		// Spawn new scratchpad
		wm.spawnScratchpad()
	}
}

// spawnScratchpad launches the scratchpad application
func (wm *WindowManager) spawnScratchpad() {
	sp := wm.scratchpad
	log.Printf("Spawning scratchpad: %s", sp.Command)

	cmd := exec.Command("sh", "-c", sp.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to spawn scratchpad: %v", err)
	}
}

// showScratchpad makes the scratchpad visible and positions it
func (wm *WindowManager) showScratchpad() {
	sp := wm.scratchpad

	// Calculate centered position
	w := uint16(float64(wm.screen.WidthInPixels) * float64(sp.Width) / 100)
	h := uint16(float64(wm.screen.HeightInPixels) * float64(sp.Height) / 100)
	x := int16((wm.screen.WidthInPixels - w) / 2)
	y := int16((wm.screen.HeightInPixels-h)/2) + int16(wm.struts[2]) // Account for top bar

	// Configure and map
	xproto.ConfigureWindow(wm.conn, sp.window,
		xproto.ConfigWindowX|xproto.ConfigWindowY|
			xproto.ConfigWindowWidth|xproto.ConfigWindowHeight|
			xproto.ConfigWindowStackMode,
		[]uint32{uint32(x), uint32(y), uint32(w), uint32(h), xproto.StackModeAbove})

	xproto.MapWindow(wm.conn, sp.window)
	wm.focus(wm.clients[sp.window])
	sp.visible = true
	log.Println("Scratchpad shown")
}

// isScratchpadWindow checks if a window is the scratchpad
func (wm *WindowManager) isScratchpadWindow(win xproto.Window) bool {
	class := wm.getWMClass(win)
	return strings.EqualFold(class, wm.scratchpad.Class)
}

// handleScratchpadMap handles mapping of a scratchpad window
func (wm *WindowManager) handleScratchpadMap(win xproto.Window) {
	sp := wm.scratchpad
	sp.window = win
	sp.visible = true

	// Create client but mark as floating
	geom, err := xproto.GetGeometry(wm.conn, xproto.Drawable(win)).Reply()
	if err != nil {
		return
	}

	client := &Client{
		Window:    win,
		X:         geom.X,
		Y:         geom.Y,
		Width:     geom.Width,
		Height:    geom.Height,
		Mapped:    true,
		Floating:  true, // Scratchpad is always floating
		Workspace: wm.current,
	}

	wm.clients[win] = client

	// Subscribe to events
	xproto.ChangeWindowAttributes(wm.conn, win,
		xproto.CwEventMask, []uint32{
			xproto.EventMaskEnterWindow |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskPropertyChange,
		})

	// Set border
	xproto.ChangeWindowAttributes(wm.conn, win,
		xproto.CwBorderPixel, []uint32{wm.config.FocusedBorderColor})
	xproto.ConfigureWindow(wm.conn, win,
		xproto.ConfigWindowBorderWidth, []uint32{uint32(wm.config.BorderWidth)})

	// Position and show
	wm.showScratchpad()

	log.Printf("Scratchpad window registered: %d", win)
}

// handleScratchpadDestroy handles destruction of scratchpad window
func (wm *WindowManager) handleScratchpadDestroy(win xproto.Window) {
	if wm.scratchpad.window == win {
		wm.scratchpad.window = 0
		wm.scratchpad.visible = false
		delete(wm.clients, win)
		log.Println("Scratchpad window destroyed")
	}
}
