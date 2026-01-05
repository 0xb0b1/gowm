package main

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/jezek/xgb/xproto"
)

const (
	// WM_HINTS flags
	UrgencyHint = 256 // (1L << 8) - XUrgencyHint
)

// checkUrgentHint checks if a window has the urgent hint set
func (wm *WindowManager) checkUrgentHint(win xproto.Window) bool {
	// Get WM_HINTS property
	reply, err := xproto.GetProperty(wm.conn, false, win,
		xproto.AtomWmHints, xproto.AtomWmHints, 0, 9).Reply()
	if err != nil || reply == nil || reply.ValueLen < 1 {
		return false
	}

	// WM_HINTS structure: flags is the first 32-bit value
	if len(reply.Value) < 4 {
		return false
	}
	flags := binary.LittleEndian.Uint32(reply.Value)

	return flags&UrgencyHint != 0
}

// handleUrgentHint handles urgent hint on a window
func (wm *WindowManager) handleUrgentHint(win xproto.Window) {
	client, exists := wm.clients[win]
	if !exists {
		return
	}

	// Don't mark focused window as urgent
	if wm.focused != nil && wm.focused.Window == win {
		return
	}

	urgent := wm.checkUrgentHint(win)
	if urgent && !client.Urgent {
		client.Urgent = true
		wm.setUrgentBorder(client)
		log.Printf("Window %d marked urgent", win)
	} else if !urgent && client.Urgent {
		client.Urgent = false
		wm.setNormalBorder(client)
		log.Printf("Window %d urgency cleared", win)
	}
}

// setUrgentBorder sets the urgent border color on a window
func (wm *WindowManager) setUrgentBorder(c *Client) {
	xproto.ChangeWindowAttributes(wm.conn, c.Window,
		xproto.CwBorderPixel, []uint32{wm.config.UrgentBorderColor})
}

// setNormalBorder sets the normal (unfocused) border color on a window
func (wm *WindowManager) setNormalBorder(c *Client) {
	if c == wm.focused {
		xproto.ChangeWindowAttributes(wm.conn, c.Window,
			xproto.CwBorderPixel, []uint32{wm.config.FocusedBorderColor})
	} else {
		xproto.ChangeWindowAttributes(wm.conn, c.Window,
			xproto.CwBorderPixel, []uint32{wm.config.UnfocusedBorderColor})
	}
}

// clearUrgent clears urgent status when window is focused
func (wm *WindowManager) clearUrgent(c *Client) {
	if c.Urgent {
		c.Urgent = false
		// Clear the WM_HINTS urgency flag
		wm.clearUrgentHint(c.Window)
		log.Printf("Window %d urgency cleared on focus", c.Window)
	}
}

// clearUrgentHint clears the urgent hint in WM_HINTS
func (wm *WindowManager) clearUrgentHint(win xproto.Window) {
	// Get current WM_HINTS
	reply, err := xproto.GetProperty(wm.conn, false, win,
		xproto.AtomWmHints, xproto.AtomWmHints, 0, 9).Reply()
	if err != nil || reply == nil || reply.ValueLen < 1 {
		return
	}

	if len(reply.Value) < 4 {
		return
	}

	// Clear urgency flag
	flags := binary.LittleEndian.Uint32(reply.Value)
	flags &^= UrgencyHint

	// Write back
	newValue := make([]byte, len(reply.Value))
	copy(newValue, reply.Value)
	binary.LittleEndian.PutUint32(newValue, flags)

	xproto.ChangeProperty(wm.conn, xproto.PropModeReplace, win,
		xproto.AtomWmHints, xproto.AtomWmHints, 32,
		uint32(len(newValue)/4), newValue)
}

// flashUrgent flashes the urgent border (optional visual effect)
func (wm *WindowManager) flashUrgent(c *Client) {
	if !c.Urgent {
		return
	}

	// Flash effect: alternate colors
	go func() {
		for i := 0; i < 3; i++ {
			xproto.ChangeWindowAttributes(wm.conn, c.Window,
				xproto.CwBorderPixel, []uint32{wm.config.FocusedBorderColor})
			time.Sleep(100 * time.Millisecond)
			xproto.ChangeWindowAttributes(wm.conn, c.Window,
				xproto.CwBorderPixel, []uint32{wm.config.UrgentBorderColor})
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// checkNetWMStateDemandsAttention checks _NET_WM_STATE for demands attention
func (wm *WindowManager) checkNetWMStateDemandsAttention(win xproto.Window) bool {
	reply, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.NET_WM_STATE, xproto.AtomAtom, 0, 32).Reply()
	if err != nil || reply == nil {
		return false
	}

	for i := uint32(0); i < reply.ValueLen; i++ {
		atom := xproto.Atom(binary.LittleEndian.Uint32(reply.Value[i*4:]))
		if atom == wm.atoms.NET_WM_STATE_DEMANDS_ATTENTION {
			return true
		}
	}
	return false
}
