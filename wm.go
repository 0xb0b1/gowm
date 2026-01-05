package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// WindowManager is the main window manager struct
type WindowManager struct {
	conn   *xgb.Conn
	root   xproto.Window
	screen *xproto.ScreenInfo

	workspaces []*Workspace
	current    int

	clients    map[xproto.Window]*Client
	focused    *Client
	config     *Config
	atoms      Atoms
	layouts    []Layout
	running    bool
	wmCheckWin xproto.Window

	// Keyboard mapping
	minKeycode     xproto.Keycode
	maxKeycode     xproto.Keycode
	keysymsPerCode int
	keysyms        []xproto.Keysym

	// Struts (reserved space for panels/bars)
	// [left, right, top, bottom]
	struts [4]uint32

	// Scratchpad
	scratchpad *Scratchpad

	// Mouse drag state
	drag DragState

	// Window rules
	rules []WindowRule

	// IPC server
	ipc *IPCServer
}

// NewWindowManager creates a new window manager
func NewWindowManager(conn *xgb.Conn) (*WindowManager, error) {
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)

	wm := &WindowManager{
		conn:       conn,
		root:       screen.Root,
		screen:     screen,
		clients:    make(map[xproto.Window]*Client),
		config:     DefaultConfig(),
		running:    true,
		minKeycode: setup.MinKeycode,
		maxKeycode: setup.MaxKeycode,
	}

	// Initialize layouts
	wm.layouts = []Layout{
		NewTallLayout(),
		NewFullLayout(),
		NewGridLayout(),
	}

	// Create 9 workspaces
	for i := 1; i <= 9; i++ {
		wm.workspaces = append(wm.workspaces, NewWorkspace(i-1, fmt.Sprintf("%d", i)))
	}

	// Initialize scratchpad
	wm.scratchpad = DefaultScratchpad()

	// Initialize window rules
	wm.rules = DefaultRules()

	return wm, nil
}

// becomeWM requests window management control from X
func (wm *WindowManager) becomeWM() error {
	mask := uint32(
		xproto.EventMaskSubstructureRedirect |
			xproto.EventMaskSubstructureNotify |
			xproto.EventMaskEnterWindow |
			xproto.EventMaskLeaveWindow |
			xproto.EventMaskPropertyChange |
			xproto.EventMaskButtonPress,
	)

	err := xproto.ChangeWindowAttributesChecked(wm.conn, wm.root,
		xproto.CwEventMask, []uint32{mask}).Check()

	if err != nil {
		return fmt.Errorf("another window manager is running: %v", err)
	}

	return nil
}

// init initializes the window manager
func (wm *WindowManager) init() error {
	// Initialize keyboard mapping
	wm.initKeyboardMapping()

	// Initialize atoms
	wm.initAtoms()

	// Setup EWMH
	wm.setupEWMH()

	// Setup keybindings
	wm.SetupKeybindings()

	// Grab keys
	wm.grabKeys()

	// Scan for existing windows
	wm.scan()

	// Update struts from any existing panels/bars
	wm.updateStruts()

	return nil
}

// initKeyboardMapping loads the keyboard mapping from X
func (wm *WindowManager) initKeyboardMapping() {
	mapping, err := xproto.GetKeyboardMapping(wm.conn, wm.minKeycode,
		byte(wm.maxKeycode-wm.minKeycode+1)).Reply()
	if err != nil {
		log.Printf("Failed to get keyboard mapping: %v", err)
		return
	}

	wm.keysymsPerCode = int(mapping.KeysymsPerKeycode)
	wm.keysyms = mapping.Keysyms
}

// keysymToKeycode converts a keysym to a keycode
func (wm *WindowManager) keysymToKeycode(keysym xproto.Keysym) xproto.Keycode {
	for i := int(wm.minKeycode); i <= int(wm.maxKeycode); i++ {
		for j := 0; j < wm.keysymsPerCode; j++ {
			idx := (i-int(wm.minKeycode))*wm.keysymsPerCode + j
			if idx < len(wm.keysyms) && wm.keysyms[idx] == keysym {
				return xproto.Keycode(i)
			}
		}
	}
	return 0
}

// grabKeys grabs all configured keybindings
func (wm *WindowManager) grabKeys() {
	// Ungrab all first
	xproto.UngrabKey(wm.conn, xproto.GrabAny, wm.root, xproto.ModMaskAny)

	// Modifiers to try (for num lock, caps lock combinations)
	modifiers := []uint16{0, xproto.ModMask2, xproto.ModMaskLock, xproto.ModMask2 | xproto.ModMaskLock}

	for combo := range wm.config.Keybindings {
		if combo.Keycode == 0 {
			continue
		}
		for _, mod := range modifiers {
			xproto.GrabKey(wm.conn,
				true,
				wm.root,
				combo.Mod|mod,
				combo.Keycode,
				xproto.GrabModeAsync,
				xproto.GrabModeAsync,
			)
		}
	}
}

// scan looks for existing windows to manage
func (wm *WindowManager) scan() {
	tree, err := xproto.QueryTree(wm.conn, wm.root).Reply()
	if err != nil {
		log.Printf("Failed to query tree: %v", err)
		return
	}

	for _, win := range tree.Children {
		attrs, err := xproto.GetWindowAttributes(wm.conn, win).Reply()
		if err != nil {
			continue
		}

		// Skip override-redirect windows and unmapped windows
		if attrs.OverrideRedirect || attrs.MapState != xproto.MapStateViewable {
			continue
		}

		wm.manageWindow(win)
	}
}

// currentWorkspace returns the current workspace
func (wm *WindowManager) currentWorkspace() *Workspace {
	return wm.workspaces[wm.current]
}

// manageWindow adds a window to management
func (wm *WindowManager) manageWindow(win xproto.Window) {
	if _, exists := wm.clients[win]; exists {
		return
	}

	// Get geometry
	geom, err := xproto.GetGeometry(wm.conn, xproto.Drawable(win)).Reply()
	if err != nil {
		log.Printf("Failed to get geometry for window %d: %v", win, err)
		return
	}

	log.Printf("Managing window %d: %dx%d+%d+%d", win, geom.Width, geom.Height, geom.X, geom.Y)

	// Apply window rules to determine floating and workspace
	shouldFloat, ruleWorkspace := wm.applyRules(win)
	targetWorkspace := wm.current
	if ruleWorkspace != nil {
		targetWorkspace = *ruleWorkspace
	}

	// Also check EWMH window type for floating
	if !shouldFloat {
		shouldFloat = wm.shouldFloat(win)
	}

	client := &Client{
		Window:    win,
		X:         geom.X,
		Y:         geom.Y,
		Width:     geom.Width,
		Height:    geom.Height,
		Mapped:    true,
		Floating:  shouldFloat,
		Workspace: targetWorkspace,
	}

	wm.clients[win] = client

	// Subscribe to events on this window
	xproto.ChangeWindowAttributes(wm.conn, win,
		xproto.CwEventMask, []uint32{
			xproto.EventMaskEnterWindow |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskPropertyChange,
		})

	// Set border
	xproto.ChangeWindowAttributes(wm.conn, win,
		xproto.CwBorderPixel, []uint32{wm.config.UnfocusedBorderColor})
	xproto.ConfigureWindow(wm.conn, win,
		xproto.ConfigWindowBorderWidth, []uint32{uint32(wm.config.BorderWidth)})

	// Setup mouse button grabs for move/resize
	wm.grabMouseButtons(win)

	// Add to target workspace (may differ from current if rule-assigned)
	wm.workspaces[targetWorkspace].Add(client)

	// If window goes to a different workspace, unmap it
	if targetWorkspace != wm.current {
		xproto.UnmapWindow(wm.conn, win)
		client.Mapped = false
	}

	// Set EWMH desktop
	wm.setClientDesktop(client)

	// Tile
	wm.tile()

	// Focus the new window
	wm.focus(client)

	// Update EWMH
	wm.updateClientList()
}

// unmanageWindow removes a window from management
func (wm *WindowManager) unmanageWindow(win xproto.Window) {
	client, exists := wm.clients[win]
	if !exists {
		return
	}

	// Remove from workspace
	ws := wm.workspaces[client.Workspace]
	ws.Remove(client)

	// Remove from clients
	delete(wm.clients, win)

	// Update focus if needed
	if wm.focused == client {
		wm.focused = nil
		if ws.Focused != nil {
			wm.focus(ws.Focused)
		}
	}

	// Retile
	wm.tile()

	// Update EWMH
	wm.updateClientList()
}

// focus sets input focus to a client
func (wm *WindowManager) focus(c *Client) {
	if c == nil {
		return
	}

	// Unfocus previous
	if wm.focused != nil && wm.focused != c {
		// Use urgent color if urgent, otherwise unfocused color
		borderColor := wm.config.UnfocusedBorderColor
		if wm.focused.Urgent {
			borderColor = wm.config.UrgentBorderColor
		}
		xproto.ChangeWindowAttributes(wm.conn, wm.focused.Window,
			xproto.CwBorderPixel, []uint32{borderColor})
	}

	// Clear urgent status on focus
	wm.clearUrgent(c)

	// Focus new - use RevertToParent which is safer
	xproto.SetInputFocus(wm.conn,
		xproto.InputFocusParent,
		c.Window,
		xproto.TimeCurrentTime,
	)

	xproto.ChangeWindowAttributes(wm.conn, c.Window,
		xproto.CwBorderPixel, []uint32{wm.config.FocusedBorderColor})

	// Raise window
	xproto.ConfigureWindow(wm.conn, c.Window,
		xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove})

	wm.focused = c
	wm.currentWorkspace().Focused = c

	// Update EWMH
	wm.updateActiveWindow()
}

// tile arranges windows according to the current layout
func (wm *WindowManager) tile() {
	ws := wm.currentWorkspace()
	clients := ws.TiledClients()

	if len(clients) == 0 {
		return
	}

	// Calculate usable area accounting for struts (panels/bars)
	gap := wm.config.GapWidth
	left := uint16(wm.struts[0])
	right := uint16(wm.struts[1])
	top := uint16(wm.struts[2])
	bottom := uint16(wm.struts[3])

	area := Rect{
		X:      int16(gap + left),
		Y:      int16(gap + top),
		Width:  wm.screen.WidthInPixels - 2*gap - left - right,
		Height: wm.screen.HeightInPixels - 2*gap - top - bottom,
	}

	// Get positions from layout
	rects := ws.Layout.Arrange(clients, area)

	// Apply positions
	bw := wm.config.BorderWidth
	for i, client := range clients {
		r := rects[i]

		// Account for gaps between windows
		r = r.Shrink(gap)

		// Account for border width
		w := r.Width
		h := r.Height
		if w > 2*bw {
			w -= 2 * bw
		}
		if h > 2*bw {
			h -= 2 * bw
		}

		client.X = r.X
		client.Y = r.Y
		client.Width = w
		client.Height = h

		xproto.ConfigureWindow(wm.conn, client.Window,
			xproto.ConfigWindowX|
				xproto.ConfigWindowY|
				xproto.ConfigWindowWidth|
				xproto.ConfigWindowHeight,
			[]uint32{uint32(r.X), uint32(r.Y), uint32(w), uint32(h)},
		)
	}
}

// switchToWorkspace switches to a different workspace
func (wm *WindowManager) switchToWorkspace(index int) {
	if index < 0 || index >= len(wm.workspaces) {
		return
	}
	if index == wm.current {
		return
	}

	// Hide windows on current workspace
	for _, c := range wm.currentWorkspace().Clients {
		xproto.UnmapWindow(wm.conn, c.Window)
	}

	// Switch
	wm.current = index

	// Show windows on new workspace
	for _, c := range wm.currentWorkspace().Clients {
		xproto.MapWindow(wm.conn, c.Window)
	}

	// Tile and focus
	wm.tile()
	if ws := wm.currentWorkspace(); ws.Focused != nil {
		wm.focus(ws.Focused)
	} else if len(ws.Clients) > 0 {
		wm.focus(ws.Clients[0])
	} else {
		wm.focused = nil
		wm.updateActiveWindow()
	}

	// Update EWMH
	wm.updateCurrentDesktop()

	log.Printf("Switched to workspace %d", index+1)
}

// moveToWorkspace moves a client to another workspace
func (wm *WindowManager) moveToWorkspace(c *Client, index int) {
	if index < 0 || index >= len(wm.workspaces) {
		return
	}
	if c.Workspace == index {
		return
	}

	// Remove from current workspace
	currentWs := wm.workspaces[c.Workspace]
	currentWs.Remove(c)

	// Add to target workspace
	targetWs := wm.workspaces[index]
	targetWs.Add(c)

	// Update EWMH desktop
	wm.setClientDesktop(c)

	// Hide if moving to different workspace
	if index != wm.current {
		xproto.UnmapWindow(wm.conn, c.Window)
	}

	// Focus next window in current workspace if we moved the focused one
	if wm.focused == c {
		wm.focused = nil
		if currentWs.Focused != nil {
			wm.focus(currentWs.Focused)
		} else if len(currentWs.Clients) > 0 {
			wm.focus(currentWs.Clients[0])
		}
	}

	// Retile
	wm.tile()

	log.Printf("Moved window to workspace %d", index+1)
}

// supportsProtocol checks if a window supports a WM protocol
func (wm *WindowManager) supportsProtocol(win xproto.Window, protocol xproto.Atom) bool {
	prop, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.WM_PROTOCOLS, xproto.AtomAtom,
		0, 64).Reply()

	if err != nil || prop == nil || prop.ValueLen == 0 {
		return false
	}

	for i := uint32(0); i < prop.ValueLen; i++ {
		atom := xproto.Atom(binary.LittleEndian.Uint32(prop.Value[i*4:]))
		if atom == protocol {
			return true
		}
	}
	return false
}

// sendDeleteWindow sends WM_DELETE_WINDOW to a client
func (wm *WindowManager) sendDeleteWindow(c *Client) {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint32(data[0:], uint32(wm.atoms.WM_DELETE_WINDOW))
	binary.LittleEndian.PutUint32(data[4:], uint32(xproto.TimeCurrentTime))

	event := xproto.ClientMessageEvent{
		Format: 32,
		Window: c.Window,
		Type:   wm.atoms.WM_PROTOCOLS,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			uint32(wm.atoms.WM_DELETE_WINDOW),
			uint32(xproto.TimeCurrentTime),
			0, 0, 0,
		}),
	}

	xproto.SendEvent(wm.conn, false, c.Window, xproto.EventMaskNoEvent, string(event.Bytes()))
}

// destroyClient forcefully destroys a client
func (wm *WindowManager) destroyClient(c *Client) {
	xproto.KillClient(wm.conn, uint32(c.Window))
}

// updateStruts recalculates the reserved screen space from all dock windows
func (wm *WindowManager) updateStruts() {
	// Reset struts
	wm.struts = [4]uint32{0, 0, 0, 0}

	// Query all children of root for strut properties
	tree, err := xproto.QueryTree(wm.conn, wm.root).Reply()
	if err != nil {
		return
	}

	for _, win := range tree.Children {
		wm.checkWindowStruts(win)
	}

	log.Printf("Struts updated: left=%d right=%d top=%d bottom=%d",
		wm.struts[0], wm.struts[1], wm.struts[2], wm.struts[3])
}

// checkWindowStruts checks a window for strut properties
func (wm *WindowManager) checkWindowStruts(win xproto.Window) {
	// Try _NET_WM_STRUT_PARTIAL first (more precise)
	prop, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.NET_WM_STRUT_PARTIAL, xproto.AtomCardinal,
		0, 12).Reply()

	if err != nil || prop == nil || prop.ValueLen < 4 {
		// Fall back to _NET_WM_STRUT
		prop, err = xproto.GetProperty(wm.conn, false, win,
			wm.atoms.NET_WM_STRUT, xproto.AtomCardinal,
			0, 4).Reply()

		if err != nil || prop == nil || prop.ValueLen < 4 {
			return
		}
	}

	// Struts are: left, right, top, bottom
	for i := 0; i < 4 && i*4 < len(prop.Value); i++ {
		val := binary.LittleEndian.Uint32(prop.Value[i*4:])
		if val > wm.struts[i] {
			wm.struts[i] = val
		}
	}
}
