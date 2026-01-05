package main

import (
	"encoding/binary"

	"github.com/jezek/xgb/xproto"
)

// WindowType represents the EWMH window type
type WindowType int

const (
	WindowTypeNormal WindowType = iota
	WindowTypeDesktop
	WindowTypeDock
	WindowTypeToolbar
	WindowTypeMenu
	WindowTypeUtility
	WindowTypeSplash
	WindowTypeDialog
)

// setupEWMH sets up EWMH support
func (wm *WindowManager) setupEWMH() {
	// Create a supporting window for EWMH compliance
	wm.wmCheckWin, _ = xproto.NewWindowId(wm.conn)
	xproto.CreateWindow(wm.conn, 0, wm.wmCheckWin, wm.root,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOnly,
		wm.screen.RootVisual,
		0, nil)

	// Set WM name on check window
	wmName := "gowm"
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.wmCheckWin,
		wm.atoms.NET_WM_NAME, wm.atoms.UTF8_STRING, 8,
		uint32(len(wmName)), []byte(wmName))

	// Set _NET_SUPPORTING_WM_CHECK on both root and check window
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(wm.wmCheckWin))
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_SUPPORTING_WM_CHECK, xproto.AtomWindow, 32,
		1, data)
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.wmCheckWin,
		wm.atoms.NET_SUPPORTING_WM_CHECK, xproto.AtomWindow, 32,
		1, data)

	// Set _NET_SUPPORTED
	supported := []xproto.Atom{
		wm.atoms.NET_SUPPORTED,
		wm.atoms.NET_CLIENT_LIST,
		wm.atoms.NET_NUMBER_OF_DESKTOPS,
		wm.atoms.NET_CURRENT_DESKTOP,
		wm.atoms.NET_DESKTOP_NAMES,
		wm.atoms.NET_ACTIVE_WINDOW,
		wm.atoms.NET_SUPPORTING_WM_CHECK,
		wm.atoms.NET_WM_NAME,
		wm.atoms.NET_WM_DESKTOP,
		wm.atoms.NET_WM_WINDOW_TYPE,
		wm.atoms.NET_WM_STATE,
		wm.atoms.NET_WM_STATE_FULLSCREEN,
		wm.atoms.NET_WM_STRUT_PARTIAL,
		wm.atoms.NET_CLOSE_WINDOW,
	}
	data = make([]byte, len(supported)*4)
	for i, atom := range supported {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(atom))
	}
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_SUPPORTED, xproto.AtomAtom, 32,
		uint32(len(supported)), data)

	// Set _NET_NUMBER_OF_DESKTOPS
	wm.updateDesktopCount()

	// Set _NET_CURRENT_DESKTOP
	wm.updateCurrentDesktop()

	// Set _NET_DESKTOP_NAMES
	wm.updateDesktopNames()
}

// updateClientList updates _NET_CLIENT_LIST
func (wm *WindowManager) updateClientList() {
	windows := make([]xproto.Window, 0, len(wm.clients))
	for win := range wm.clients {
		windows = append(windows, win)
	}

	data := make([]byte, len(windows)*4)
	for i, win := range windows {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(win))
	}
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_CLIENT_LIST, xproto.AtomWindow, 32,
		uint32(len(windows)), data)
}

// updateDesktopCount updates _NET_NUMBER_OF_DESKTOPS
func (wm *WindowManager) updateDesktopCount() {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(len(wm.workspaces)))
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_NUMBER_OF_DESKTOPS, xproto.AtomCardinal, 32,
		1, data)
}

// updateCurrentDesktop updates _NET_CURRENT_DESKTOP
func (wm *WindowManager) updateCurrentDesktop() {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(wm.current))
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_CURRENT_DESKTOP, xproto.AtomCardinal, 32,
		1, data)
}

// updateDesktopNames updates _NET_DESKTOP_NAMES
func (wm *WindowManager) updateDesktopNames() {
	// Names are null-separated
	var names []byte
	for _, ws := range wm.workspaces {
		names = append(names, []byte(ws.Name)...)
		names = append(names, 0)
	}
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_DESKTOP_NAMES, wm.atoms.UTF8_STRING, 8,
		uint32(len(names)), names)
}

// updateActiveWindow updates _NET_ACTIVE_WINDOW
func (wm *WindowManager) updateActiveWindow() {
	var win xproto.Window
	if wm.focused != nil {
		win = wm.focused.Window
	}
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(win))
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, wm.root,
		wm.atoms.NET_ACTIVE_WINDOW, xproto.AtomWindow, 32,
		1, data)
}

// setClientDesktop sets _NET_WM_DESKTOP for a client
func (wm *WindowManager) setClientDesktop(c *Client) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(c.Workspace))
	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, c.Window,
		wm.atoms.NET_WM_DESKTOP, xproto.AtomCardinal, 32,
		1, data)
}

// getClientDesktop reads _NET_WM_DESKTOP from a window (for WM restart)
func (wm *WindowManager) getClientDesktop(win xproto.Window) int {
	prop, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.NET_WM_DESKTOP, xproto.AtomCardinal,
		0, 1).Reply()

	if err != nil || prop == nil || prop.ValueLen == 0 {
		return -1
	}

	return int(binary.LittleEndian.Uint32(prop.Value))
}

// setFullscreenState sets or removes _NET_WM_STATE_FULLSCREEN on a window
func (wm *WindowManager) setFullscreenState(win xproto.Window, fullscreen bool) {
	if fullscreen {
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(wm.atoms.NET_WM_STATE_FULLSCREEN))
		xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, win,
			wm.atoms.NET_WM_STATE, xproto.AtomAtom, 32,
			1, data)
	} else {
		xproto.DeleteProperty(wm.conn, win, wm.atoms.NET_WM_STATE)
	}
}

// getWindowType returns the EWMH window type
func (wm *WindowManager) getWindowType(win xproto.Window) WindowType {
	prop, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.NET_WM_WINDOW_TYPE, xproto.AtomAtom,
		0, 32).Reply()

	if err != nil || prop == nil || prop.ValueLen == 0 {
		return WindowTypeNormal
	}

	atom := xproto.Atom(binary.LittleEndian.Uint32(prop.Value))

	switch atom {
	case wm.atoms.NET_WM_WINDOW_TYPE_DESKTOP:
		return WindowTypeDesktop
	case wm.atoms.NET_WM_WINDOW_TYPE_DOCK:
		return WindowTypeDock
	case wm.atoms.NET_WM_WINDOW_TYPE_TOOLBAR:
		return WindowTypeToolbar
	case wm.atoms.NET_WM_WINDOW_TYPE_MENU:
		return WindowTypeMenu
	case wm.atoms.NET_WM_WINDOW_TYPE_UTILITY:
		return WindowTypeUtility
	case wm.atoms.NET_WM_WINDOW_TYPE_SPLASH:
		return WindowTypeSplash
	case wm.atoms.NET_WM_WINDOW_TYPE_DIALOG:
		return WindowTypeDialog
	default:
		return WindowTypeNormal
	}
}

// isTransient checks if the window is transient (a dialog)
func (wm *WindowManager) isTransient(win xproto.Window) bool {
	prop, err := xproto.GetProperty(wm.conn, false, win,
		xproto.AtomWmTransientFor, xproto.AtomWindow,
		0, 1).Reply()

	return err == nil && prop != nil && prop.ValueLen > 0
}

// hasFullscreenState checks if a window has _NET_WM_STATE_FULLSCREEN set
func (wm *WindowManager) hasFullscreenState(win xproto.Window) bool {
	prop, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.NET_WM_STATE, xproto.AtomAtom,
		0, 32).Reply()

	if err != nil || prop == nil || prop.ValueLen == 0 {
		return false
	}

	for i := uint32(0); i < prop.ValueLen; i++ {
		atom := xproto.Atom(binary.LittleEndian.Uint32(prop.Value[i*4:]))
		if atom == wm.atoms.NET_WM_STATE_FULLSCREEN {
			return true
		}
	}
	return false
}

// shouldFloat determines if a window should be floating
func (wm *WindowManager) shouldFloat(win xproto.Window) bool {
	windowType := wm.getWindowType(win)

	switch windowType {
	case WindowTypeDialog, WindowTypeSplash, WindowTypeUtility, WindowTypeMenu:
		return true
	}

	if wm.isTransient(win) {
		return true
	}

	// Check window rules
	shouldFloat, _ := wm.applyRules(win)
	return shouldFloat
}

// getWMClass returns the WM_CLASS instance name
func (wm *WindowManager) getWMClass(win xproto.Window) string {
	prop, err := xproto.GetProperty(wm.conn, false, win,
		xproto.AtomWmClass, xproto.AtomString,
		0, 256).Reply()

	if err != nil || prop == nil || prop.ValueLen == 0 {
		return ""
	}

	// WM_CLASS is null-separated: instance\0class\0
	// Return the class (second part)
	for i, b := range prop.Value {
		if b == 0 && i+1 < len(prop.Value) {
			end := i + 1
			for end < len(prop.Value) && prop.Value[end] != 0 {
				end++
			}
			return string(prop.Value[i+1 : end])
		}
	}
	return string(prop.Value)
}
