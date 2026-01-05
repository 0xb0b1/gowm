package main

import (
	"log"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Println("Starting gowm...")

	// Connect to X server
	conn, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Failed to connect to X server: %v", err)
	}
	defer conn.Close()

	// Create window manager
	wm, err := NewWindowManager(conn)
	if err != nil {
		log.Fatalf("Failed to create window manager: %v", err)
	}

	// Become the window manager
	if err := wm.becomeWM(); err != nil {
		log.Fatalf("Failed to become window manager: %v", err)
	}

	// Initialize
	if err := wm.init(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	log.Printf("gowm started on screen %dx%d",
		wm.screen.WidthInPixels, wm.screen.HeightInPixels)

	// Start IPC server
	ipc, err := NewIPCServer(wm)
	if err != nil {
		log.Printf("Warning: Failed to start IPC server: %v", err)
	} else {
		wm.ipc = ipc
		ipc.Start()
		defer ipc.Stop()
	}

	// Run startup commands
	wm.runStartup()

	// Main event loop
	wm.eventLoop()

	log.Println("gowm exiting")
	os.Exit(0)
}

// eventLoop handles X events
func (wm *WindowManager) eventLoop() {
	for wm.running {
		event, err := wm.conn.WaitForEvent()
		if err != nil {
			log.Printf("X error: %v", err)
			continue
		}
		if event == nil {
			log.Println("X connection closed")
			return
		}

		switch e := event.(type) {
		case xproto.MapRequestEvent:
			wm.handleMapRequest(e)

		case xproto.UnmapNotifyEvent:
			wm.handleUnmapNotify(e)

		case xproto.DestroyNotifyEvent:
			wm.handleDestroyNotify(e)

		case xproto.ConfigureRequestEvent:
			wm.handleConfigureRequest(e)

		case xproto.ConfigureNotifyEvent:
			wm.handleConfigureNotify(e)

		case xproto.KeyPressEvent:
			wm.handleKeyPress(e)

		case xproto.EnterNotifyEvent:
			wm.handleEnterNotify(e)

		case xproto.PropertyNotifyEvent:
			wm.handlePropertyNotify(e)

		case xproto.ClientMessageEvent:
			wm.handleClientMessage(e)

		case xproto.ButtonPressEvent:
			wm.handleButtonPress(e)

		case xproto.ButtonReleaseEvent:
			wm.handleButtonRelease(e)

		case xproto.MotionNotifyEvent:
			wm.handleMotionNotify(e)
		}
	}
}

// handleMapRequest handles a window wanting to be mapped
func (wm *WindowManager) handleMapRequest(e xproto.MapRequestEvent) {
	log.Printf("MapRequest: window=%d", e.Window)

	// Check for override redirect
	attrs, err := xproto.GetWindowAttributes(wm.conn, e.Window).Reply()
	if err != nil {
		return
	}
	if attrs.OverrideRedirect {
		return
	}

	// Map the window first
	xproto.MapWindow(wm.conn, e.Window)

	// Check if it's a dock/panel - don't manage but check for struts
	windowType := wm.getWindowType(e.Window)
	if windowType == WindowTypeDock {
		log.Printf("Dock window detected: %d", e.Window)
		wm.updateStruts()
		wm.tile() // Retile to account for new struts
		return
	}

	// Check if it's the scratchpad window
	if wm.isScratchpadWindow(e.Window) {
		wm.handleScratchpadMap(e.Window)
		return
	}

	// Then manage it (this will tile and focus)
	wm.manageWindow(e.Window)
}

// handleUnmapNotify handles a window being unmapped
func (wm *WindowManager) handleUnmapNotify(e xproto.UnmapNotifyEvent) {
	// Ignore synthetic events
	if e.Event != wm.root {
		return
	}

	client, exists := wm.clients[e.Window]
	if !exists {
		return
	}

	log.Printf("UnmapNotify: window=%d", e.Window)
	client.Mapped = false
}

// handleDestroyNotify handles a window being destroyed
func (wm *WindowManager) handleDestroyNotify(e xproto.DestroyNotifyEvent) {
	log.Printf("DestroyNotify: window=%d", e.Window)

	// Check if scratchpad was destroyed
	if wm.scratchpad.window == e.Window {
		wm.handleScratchpadDestroy(e.Window)
		return
	}

	wm.unmanageWindow(e.Window)
}

// handleConfigureRequest handles a window configure request
func (wm *WindowManager) handleConfigureRequest(e xproto.ConfigureRequestEvent) {
	client, managed := wm.clients[e.Window]

	if !managed || client.Floating {
		// Allow unmanaged or floating windows to configure themselves
		values := []uint32{}
		mask := uint16(0)

		if e.ValueMask&xproto.ConfigWindowX != 0 {
			mask |= xproto.ConfigWindowX
			values = append(values, uint32(e.X))
		}
		if e.ValueMask&xproto.ConfigWindowY != 0 {
			mask |= xproto.ConfigWindowY
			values = append(values, uint32(e.Y))
		}
		if e.ValueMask&xproto.ConfigWindowWidth != 0 {
			mask |= xproto.ConfigWindowWidth
			values = append(values, uint32(e.Width))
		}
		if e.ValueMask&xproto.ConfigWindowHeight != 0 {
			mask |= xproto.ConfigWindowHeight
			values = append(values, uint32(e.Height))
		}
		if e.ValueMask&xproto.ConfigWindowBorderWidth != 0 {
			mask |= xproto.ConfigWindowBorderWidth
			values = append(values, uint32(e.BorderWidth))
		}
		if e.ValueMask&xproto.ConfigWindowSibling != 0 {
			mask |= xproto.ConfigWindowSibling
			values = append(values, uint32(e.Sibling))
		}
		if e.ValueMask&xproto.ConfigWindowStackMode != 0 {
			mask |= xproto.ConfigWindowStackMode
			values = append(values, uint32(e.StackMode))
		}

		if mask != 0 {
			xproto.ConfigureWindow(wm.conn, e.Window, mask, values)
		}
		return
	}

	// For tiled windows, send a synthetic ConfigureNotify with current geometry
	wm.sendConfigureNotify(client)
}

// sendConfigureNotify sends a synthetic configure notify to a client
func (wm *WindowManager) sendConfigureNotify(c *Client) {
	event := xproto.ConfigureNotifyEvent{
		Event:            c.Window,
		Window:           c.Window,
		AboveSibling:     0,
		X:                c.X,
		Y:                c.Y,
		Width:            c.Width,
		Height:           c.Height,
		BorderWidth:      wm.config.BorderWidth,
		OverrideRedirect: false,
	}
	xproto.SendEvent(wm.conn, false, c.Window,
		xproto.EventMaskStructureNotify, string(event.Bytes()))
}

// handleConfigureNotify handles configure notify events
func (wm *WindowManager) handleConfigureNotify(e xproto.ConfigureNotifyEvent) {
	// Update screen dimensions if root changed
	if e.Window == wm.root {
		wm.screen.WidthInPixels = e.Width
		wm.screen.HeightInPixels = e.Height
		wm.tile()
	}
}

// handleKeyPress handles key press events
func (wm *WindowManager) handleKeyPress(e xproto.KeyPressEvent) {
	// Clean modifier state (ignore num lock, caps lock)
	cleanMod := e.State & (xproto.ModMask1 | xproto.ModMask4 |
		xproto.ModMaskShift | xproto.ModMaskControl)

	combo := KeyCombo{Mod: cleanMod, Keycode: e.Detail}

	if action, ok := wm.config.Keybindings[combo]; ok {
		action(wm)
	}
}

// handleEnterNotify handles pointer entering a window
func (wm *WindowManager) handleEnterNotify(e xproto.EnterNotifyEvent) {
	if !wm.config.FocusFollowsMouse {
		return
	}

	// Ignore events from grab
	if e.Mode != xproto.NotifyModeNormal {
		return
	}

	client, exists := wm.clients[e.Event]
	if !exists {
		return
	}

	// Only focus if on current workspace
	if client.Workspace == wm.current {
		wm.focus(client)
	}
}

// handlePropertyNotify handles property changes
func (wm *WindowManager) handlePropertyNotify(e xproto.PropertyNotifyEvent) {
	// Check for WM_HINTS changes (urgent hint)
	if e.Atom == xproto.AtomWmHints {
		wm.handleUrgentHint(e.Window)
	}

	// Check for _NET_WM_STATE changes (demands attention)
	if e.Atom == wm.atoms.NET_WM_STATE {
		if wm.checkNetWMStateDemandsAttention(e.Window) {
			wm.handleUrgentHint(e.Window)
		}
	}
}

// handleClientMessage handles client messages (EWMH requests)
func (wm *WindowManager) handleClientMessage(e xproto.ClientMessageEvent) {
	data := e.Data.Data32

	switch e.Type {
	case wm.atoms.NET_CURRENT_DESKTOP:
		// Switch to requested desktop
		if len(data) > 0 {
			wm.switchToWorkspace(int(data[0]))
		}

	case wm.atoms.NET_ACTIVE_WINDOW:
		// Focus requested window
		if client, exists := wm.clients[e.Window]; exists {
			if client.Workspace != wm.current {
				wm.switchToWorkspace(client.Workspace)
			}
			wm.focus(client)
		}

	case wm.atoms.NET_CLOSE_WINDOW:
		// Close window
		if client, exists := wm.clients[e.Window]; exists {
			if wm.supportsProtocol(client.Window, wm.atoms.WM_DELETE_WINDOW) {
				wm.sendDeleteWindow(client)
			} else {
				wm.destroyClient(client)
			}
		}

	case wm.atoms.NET_WM_STATE:
		// Handle state changes (fullscreen, etc.)
		wm.handleNetWMState(e)
	}
}

// handleNetWMState handles _NET_WM_STATE client messages
func (wm *WindowManager) handleNetWMState(e xproto.ClientMessageEvent) {
	client, exists := wm.clients[e.Window]
	if !exists {
		return
	}

	data := e.Data.Data32
	action := data[0]
	prop1 := xproto.Atom(data[1])
	prop2 := xproto.Atom(data[2])

	const (
		_NET_WM_STATE_REMOVE = 0
		_NET_WM_STATE_ADD    = 1
		_NET_WM_STATE_TOGGLE = 2
	)

	handleState := func(prop xproto.Atom) {
		if prop == wm.atoms.NET_WM_STATE_FULLSCREEN {
			switch action {
			case _NET_WM_STATE_REMOVE:
				client.Floating = false
			case _NET_WM_STATE_ADD:
				client.Floating = true
				// Make fullscreen
				client.X = 0
				client.Y = 0
				client.Width = wm.screen.WidthInPixels
				client.Height = wm.screen.HeightInPixels
				xproto.ConfigureWindow(wm.conn, client.Window,
					xproto.ConfigWindowX|xproto.ConfigWindowY|
						xproto.ConfigWindowWidth|xproto.ConfigWindowHeight|
						xproto.ConfigWindowBorderWidth,
					[]uint32{0, 0, uint32(client.Width), uint32(client.Height), 0})
			case _NET_WM_STATE_TOGGLE:
				client.Floating = !client.Floating
			}
			wm.tile()
		}
	}

	if prop1 != 0 {
		handleState(prop1)
	}
	if prop2 != 0 {
		handleState(prop2)
	}
}
