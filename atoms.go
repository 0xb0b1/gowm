package main

import (
	"log"

	"github.com/jezek/xgb/xproto"
)

// Atoms holds all interned X11 atoms
type Atoms struct {
	// ICCCM
	WM_PROTOCOLS     xproto.Atom
	WM_DELETE_WINDOW xproto.Atom
	WM_STATE         xproto.Atom
	WM_TAKE_FOCUS    xproto.Atom
	WM_TRANSIENT_FOR xproto.Atom
	WM_CLASS         xproto.Atom
	WM_NAME          xproto.Atom

	// EWMH
	NET_SUPPORTED             xproto.Atom
	NET_CLIENT_LIST           xproto.Atom
	NET_CLIENT_LIST_STACKING  xproto.Atom
	NET_NUMBER_OF_DESKTOPS    xproto.Atom
	NET_DESKTOP_GEOMETRY      xproto.Atom
	NET_DESKTOP_VIEWPORT      xproto.Atom
	NET_CURRENT_DESKTOP       xproto.Atom
	NET_DESKTOP_NAMES         xproto.Atom
	NET_ACTIVE_WINDOW         xproto.Atom
	NET_WORKAREA              xproto.Atom
	NET_SUPPORTING_WM_CHECK   xproto.Atom
	NET_WM_NAME               xproto.Atom
	NET_WM_VISIBLE_NAME       xproto.Atom
	NET_WM_DESKTOP            xproto.Atom
	NET_WM_WINDOW_TYPE        xproto.Atom
	NET_WM_WINDOW_TYPE_DESKTOP xproto.Atom
	NET_WM_WINDOW_TYPE_DOCK    xproto.Atom
	NET_WM_WINDOW_TYPE_TOOLBAR xproto.Atom
	NET_WM_WINDOW_TYPE_MENU    xproto.Atom
	NET_WM_WINDOW_TYPE_UTILITY xproto.Atom
	NET_WM_WINDOW_TYPE_SPLASH  xproto.Atom
	NET_WM_WINDOW_TYPE_DIALOG  xproto.Atom
	NET_WM_WINDOW_TYPE_NORMAL  xproto.Atom
	NET_WM_STATE              xproto.Atom
	NET_WM_STATE_MODAL        xproto.Atom
	NET_WM_STATE_STICKY       xproto.Atom
	NET_WM_STATE_MAXIMIZED_VERT   xproto.Atom
	NET_WM_STATE_MAXIMIZED_HORZ   xproto.Atom
	NET_WM_STATE_SHADED       xproto.Atom
	NET_WM_STATE_SKIP_TASKBAR xproto.Atom
	NET_WM_STATE_SKIP_PAGER   xproto.Atom
	NET_WM_STATE_HIDDEN       xproto.Atom
	NET_WM_STATE_FULLSCREEN   xproto.Atom
	NET_WM_STATE_ABOVE        xproto.Atom
	NET_WM_STATE_BELOW        xproto.Atom
	NET_WM_STATE_DEMANDS_ATTENTION xproto.Atom
	NET_WM_STRUT              xproto.Atom
	NET_WM_STRUT_PARTIAL      xproto.Atom
	NET_CLOSE_WINDOW          xproto.Atom

	// UTF8
	UTF8_STRING xproto.Atom
}

// initAtoms interns all required atoms
func (wm *WindowManager) initAtoms() {
	atomNames := map[string]*xproto.Atom{
		// ICCCM
		"WM_PROTOCOLS":     &wm.atoms.WM_PROTOCOLS,
		"WM_DELETE_WINDOW": &wm.atoms.WM_DELETE_WINDOW,
		"WM_STATE":         &wm.atoms.WM_STATE,
		"WM_TAKE_FOCUS":    &wm.atoms.WM_TAKE_FOCUS,
		"WM_TRANSIENT_FOR": &wm.atoms.WM_TRANSIENT_FOR,
		"WM_CLASS":         &wm.atoms.WM_CLASS,
		"WM_NAME":          &wm.atoms.WM_NAME,

		// EWMH
		"_NET_SUPPORTED":             &wm.atoms.NET_SUPPORTED,
		"_NET_CLIENT_LIST":           &wm.atoms.NET_CLIENT_LIST,
		"_NET_CLIENT_LIST_STACKING":  &wm.atoms.NET_CLIENT_LIST_STACKING,
		"_NET_NUMBER_OF_DESKTOPS":    &wm.atoms.NET_NUMBER_OF_DESKTOPS,
		"_NET_DESKTOP_GEOMETRY":      &wm.atoms.NET_DESKTOP_GEOMETRY,
		"_NET_DESKTOP_VIEWPORT":      &wm.atoms.NET_DESKTOP_VIEWPORT,
		"_NET_CURRENT_DESKTOP":       &wm.atoms.NET_CURRENT_DESKTOP,
		"_NET_DESKTOP_NAMES":         &wm.atoms.NET_DESKTOP_NAMES,
		"_NET_ACTIVE_WINDOW":         &wm.atoms.NET_ACTIVE_WINDOW,
		"_NET_WORKAREA":              &wm.atoms.NET_WORKAREA,
		"_NET_SUPPORTING_WM_CHECK":   &wm.atoms.NET_SUPPORTING_WM_CHECK,
		"_NET_WM_NAME":               &wm.atoms.NET_WM_NAME,
		"_NET_WM_VISIBLE_NAME":       &wm.atoms.NET_WM_VISIBLE_NAME,
		"_NET_WM_DESKTOP":            &wm.atoms.NET_WM_DESKTOP,
		"_NET_WM_WINDOW_TYPE":        &wm.atoms.NET_WM_WINDOW_TYPE,
		"_NET_WM_WINDOW_TYPE_DESKTOP": &wm.atoms.NET_WM_WINDOW_TYPE_DESKTOP,
		"_NET_WM_WINDOW_TYPE_DOCK":    &wm.atoms.NET_WM_WINDOW_TYPE_DOCK,
		"_NET_WM_WINDOW_TYPE_TOOLBAR": &wm.atoms.NET_WM_WINDOW_TYPE_TOOLBAR,
		"_NET_WM_WINDOW_TYPE_MENU":    &wm.atoms.NET_WM_WINDOW_TYPE_MENU,
		"_NET_WM_WINDOW_TYPE_UTILITY": &wm.atoms.NET_WM_WINDOW_TYPE_UTILITY,
		"_NET_WM_WINDOW_TYPE_SPLASH":  &wm.atoms.NET_WM_WINDOW_TYPE_SPLASH,
		"_NET_WM_WINDOW_TYPE_DIALOG":  &wm.atoms.NET_WM_WINDOW_TYPE_DIALOG,
		"_NET_WM_WINDOW_TYPE_NORMAL":  &wm.atoms.NET_WM_WINDOW_TYPE_NORMAL,
		"_NET_WM_STATE":              &wm.atoms.NET_WM_STATE,
		"_NET_WM_STATE_MODAL":        &wm.atoms.NET_WM_STATE_MODAL,
		"_NET_WM_STATE_STICKY":       &wm.atoms.NET_WM_STATE_STICKY,
		"_NET_WM_STATE_MAXIMIZED_VERT":   &wm.atoms.NET_WM_STATE_MAXIMIZED_VERT,
		"_NET_WM_STATE_MAXIMIZED_HORZ":   &wm.atoms.NET_WM_STATE_MAXIMIZED_HORZ,
		"_NET_WM_STATE_SHADED":       &wm.atoms.NET_WM_STATE_SHADED,
		"_NET_WM_STATE_SKIP_TASKBAR": &wm.atoms.NET_WM_STATE_SKIP_TASKBAR,
		"_NET_WM_STATE_SKIP_PAGER":   &wm.atoms.NET_WM_STATE_SKIP_PAGER,
		"_NET_WM_STATE_HIDDEN":       &wm.atoms.NET_WM_STATE_HIDDEN,
		"_NET_WM_STATE_FULLSCREEN":   &wm.atoms.NET_WM_STATE_FULLSCREEN,
		"_NET_WM_STATE_ABOVE":        &wm.atoms.NET_WM_STATE_ABOVE,
		"_NET_WM_STATE_BELOW":        &wm.atoms.NET_WM_STATE_BELOW,
		"_NET_WM_STATE_DEMANDS_ATTENTION": &wm.atoms.NET_WM_STATE_DEMANDS_ATTENTION,
		"_NET_WM_STRUT":              &wm.atoms.NET_WM_STRUT,
		"_NET_WM_STRUT_PARTIAL":      &wm.atoms.NET_WM_STRUT_PARTIAL,
		"_NET_CLOSE_WINDOW":          &wm.atoms.NET_CLOSE_WINDOW,

		// UTF8
		"UTF8_STRING": &wm.atoms.UTF8_STRING,
	}

	for name, atom := range atomNames {
		reply, err := xproto.InternAtom(wm.conn, false, uint16(len(name)), name).Reply()
		if err != nil {
			log.Printf("Failed to intern atom %s: %v", name, err)
			continue
		}
		*atom = reply.Atom
	}
}
