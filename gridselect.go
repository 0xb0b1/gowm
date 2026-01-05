package main

import (
	"log"
	"strings"

	"github.com/jezek/xgb/xproto"
)

// GridSelect displays a grid of all windows for selection (like xmonad's GridSelect)
type GridSelect struct {
	wm       *WindowManager
	window   xproto.Window
	gc       xproto.Gcontext
	visible  bool
	items    []*GridItem
	filtered []*GridItem
	selected int
	cols     int
	rows     int
	search   string

	// Config matching xmonad
	cellW   int
	cellH   int
	padding int
}

// GridItem represents a window in the grid
type GridItem struct {
	Client    *Client
	Workspace int
	Label     string
	X, Y      int
	W, H      int
	BGColor   uint32
	FGColor   uint32
}

// Catppuccin Frappe colors for grid
var gridColors = struct {
	Base      uint32
	Surface0  uint32
	Surface1  uint32
	Mauve     uint32
	Blue      uint32
	Lavender  uint32
	Sapphire  uint32
	Sky       uint32
	Teal      uint32
	Green     uint32
	Yellow    uint32
	Peach     uint32
	Maroon    uint32
	Red       uint32
	Pink      uint32
	Text      uint32
	Crust     uint32
}{
	Base:     0x303446,
	Surface0: 0x414559,
	Surface1: 0x51576d,
	Mauve:    0xca9ee6,
	Blue:     0x8caaee,
	Lavender: 0xbabbf1,
	Sapphire: 0x85c1dc,
	Sky:      0x99d1db,
	Teal:     0x81c8be,
	Green:    0xa6d189,
	Yellow:   0xe5c890,
	Peach:    0xef9f76,
	Maroon:   0xea999c,
	Red:      0xe78284,
	Pink:     0xf4b8e4,
	Text:     0xc6d0f5,
	Crust:    0x232634,
}

// Color palette for windows (cycles through these)
var windowPalette = []uint32{
	0x8caaee, // Blue
	0xca9ee6, // Mauve
	0x81c8be, // Teal
	0xa6d189, // Green
	0xe5c890, // Yellow
	0xef9f76, // Peach
	0xf4b8e4, // Pink
	0x85c1dc, // Sapphire
	0xbabbf1, // Lavender
	0x99d1db, // Sky
}

// NewGridSelect creates a new grid select instance
func NewGridSelect(wm *WindowManager) *GridSelect {
	return &GridSelect{
		wm:      wm,
		cellW:   280, // Match xmonad: gs_cellwidth = 280
		cellH:   80,  // Match xmonad: gs_cellheight = 80
		padding: 8,   // Match xmonad: gs_cellpadding = 8
		cols:    4,
	}
}

// Show displays the grid select window
func (gs *GridSelect) Show() {
	if gs.visible {
		return
	}

	// Collect all windows from all workspaces
	gs.items = nil
	gs.search = ""
	colorIdx := 0

	for wsIdx, ws := range gs.wm.workspaces {
		for _, client := range ws.Clients {
			title := gs.wm.getWindowTitle(client.Window)
			if title == "" {
				title = gs.wm.getWMClass(client.Window)
			}
			if title == "" {
				title = "Unknown"
			}

			// Add workspace indicator
			label := ws.Name + ": " + title
			if len(label) > 40 {
				label = label[:37] + "..."
			}

			// Assign color from palette (cycles)
			bgColor := windowPalette[colorIdx%len(windowPalette)]
			colorIdx++

			gs.items = append(gs.items, &GridItem{
				Client:    client,
				Workspace: wsIdx,
				Label:     label,
				BGColor:   bgColor,
				FGColor:   gridColors.Crust,
			})
		}
	}

	if len(gs.items) == 0 {
		log.Println("GridSelect: No windows to show")
		return
	}

	gs.filtered = gs.items
	gs.selected = 0
	gs.visible = true

	gs.createWindow()
	gs.Draw()
}

// createWindow creates and maps the grid window
func (gs *GridSelect) createWindow() {
	// Calculate grid dimensions based on filtered items
	n := len(gs.filtered)
	if n == 0 {
		return
	}

	// Calculate optimal columns (max 4, or less if fewer items)
	gs.cols = 4
	if n < gs.cols {
		gs.cols = n
	}
	gs.rows = (n + gs.cols - 1) / gs.cols

	gridW := gs.cols*(gs.cellW+gs.padding) + gs.padding
	gridH := gs.rows*(gs.cellH+gs.padding) + gs.padding + 30 // +30 for search bar

	// Center on screen (gs_originFractX/Y = 0.5)
	screenW := int(gs.wm.screen.WidthInPixels)
	screenH := int(gs.wm.screen.HeightInPixels)
	x := (screenW - gridW) / 2
	y := (screenH - gridH) / 2

	// Calculate item positions
	for i, item := range gs.filtered {
		col := i % gs.cols
		row := i / gs.cols
		item.X = gs.padding + col*(gs.cellW+gs.padding)
		item.Y = gs.padding + row*(gs.cellH+gs.padding) + 30 // Below search bar
		item.W = gs.cellW
		item.H = gs.cellH
	}

	// Create window
	gs.window, _ = xproto.NewWindowId(gs.wm.conn)

	xproto.CreateWindow(
		gs.wm.conn,
		gs.wm.screen.RootDepth,
		gs.window,
		gs.wm.root,
		int16(x), int16(y),
		uint16(gridW), uint16(gridH),
		2,
		xproto.WindowClassInputOutput,
		gs.wm.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwBorderPixel|xproto.CwOverrideRedirect|xproto.CwEventMask,
		[]uint32{
			gridColors.Base,
			gridColors.Surface1, // Border color
			1,                   // override redirect - bypass WM
			xproto.EventMaskExposure | xproto.EventMaskKeyPress,
		},
	)

	// Create graphics context
	gs.gc, _ = xproto.NewGcontextId(gs.wm.conn)
	xproto.CreateGC(gs.wm.conn, gs.gc, xproto.Drawable(gs.window), 0, nil)

	// Map window and grab keyboard
	xproto.MapWindow(gs.wm.conn, gs.window)
	xproto.SetInputFocus(gs.wm.conn, xproto.InputFocusPointerRoot, gs.window, xproto.TimeCurrentTime)

	// Grab keyboard
	xproto.GrabKeyboard(
		gs.wm.conn,
		true,
		gs.window,
		xproto.TimeCurrentTime,
		xproto.GrabModeAsync,
		xproto.GrabModeAsync,
	)
}

// Hide closes the grid select window
func (gs *GridSelect) Hide() {
	if !gs.visible {
		return
	}

	xproto.UngrabKeyboard(gs.wm.conn, xproto.TimeCurrentTime)
	xproto.FreeGC(gs.wm.conn, gs.gc)
	xproto.DestroyWindow(gs.wm.conn, gs.window)

	gs.visible = false
	gs.items = nil
	gs.filtered = nil
	gs.search = ""
}

// Draw renders the grid
func (gs *GridSelect) Draw() {
	if !gs.visible || len(gs.filtered) == 0 {
		return
	}

	// Clear background
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{gridColors.Base})
	geom, _ := xproto.GetGeometry(gs.wm.conn, xproto.Drawable(gs.window)).Reply()
	if geom != nil {
		xproto.PolyFillRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
			{X: 0, Y: 0, Width: geom.Width, Height: geom.Height},
		})
	}

	// Draw search bar
	gs.drawSearchBar()

	// Draw items
	for i, item := range gs.filtered {
		gs.drawItem(i, item)
	}
}

// drawSearchBar draws the search input area
func (gs *GridSelect) drawSearchBar() {
	// Search bar background
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{gridColors.Surface0})
	xproto.PolyFillRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
		{X: int16(gs.padding), Y: int16(gs.padding), Width: uint16(gs.cols*(gs.cellW+gs.padding) - gs.padding), Height: 22},
	})

	// Search text
	searchText := "Search: " + gs.search + "_"
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{gridColors.Text})
	xproto.ImageText8(
		gs.wm.conn,
		byte(len(searchText)),
		xproto.Drawable(gs.window),
		gs.gc,
		int16(gs.padding+8),
		int16(gs.padding+16),
		searchText,
	)
}

// drawItem draws a single grid item
func (gs *GridSelect) drawItem(idx int, item *GridItem) {
	isSelected := idx == gs.selected

	var bg, fg uint32
	if isSelected {
		bg = gridColors.Mauve // Selected uses Mauve (like xmonad's active bg)
		fg = gridColors.Crust
	} else {
		bg = item.BGColor
		fg = item.FGColor
	}

	// Draw cell background with rounded feel (fill main area)
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{bg})
	xproto.PolyFillRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
		{X: int16(item.X), Y: int16(item.Y), Width: uint16(item.W), Height: uint16(item.H)},
	})

	// Draw selection border
	if isSelected {
		xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{gridColors.Text})
		xproto.PolyRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
			{X: int16(item.X), Y: int16(item.Y), Width: uint16(item.W), Height: uint16(item.H)},
			{X: int16(item.X + 1), Y: int16(item.Y + 1), Width: uint16(item.W - 2), Height: uint16(item.H - 2)},
		})
	}

	// Draw text (centered vertically)
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{fg})
	textY := item.Y + item.H/2 + 5
	xproto.ImageText8(
		gs.wm.conn,
		byte(len(item.Label)),
		xproto.Drawable(gs.window),
		gs.gc,
		int16(item.X+12),
		int16(textY),
		item.Label,
	)
}

// HandleKeyPress handles keyboard input in grid select
func (gs *GridSelect) HandleKeyPress(e xproto.KeyPressEvent) bool {
	if !gs.visible {
		return false
	}

	keysym := gs.wm.keycodeToKeysym(e.Detail)

	switch keysym {
	case XK_Escape:
		gs.Hide()
		return true

	case XK_Return, XK_space:
		gs.SelectCurrent()
		return true

	case XK_h, XK_Left:
		gs.MoveSelection(-1, 0)
		return true

	case XK_l, XK_Right:
		gs.MoveSelection(1, 0)
		return true

	case XK_k, XK_Up:
		gs.MoveSelection(0, -1)
		return true

	case XK_j, XK_Down:
		gs.MoveSelection(0, 1)
		return true

	case XK_Tab:
		// Tab cycles through items
		if len(gs.filtered) > 0 {
			gs.selected = (gs.selected + 1) % len(gs.filtered)
			gs.Draw()
		}
		return true

	case XK_BackSpace:
		// Remove last character from search
		if len(gs.search) > 0 {
			gs.search = gs.search[:len(gs.search)-1]
			gs.updateFilter()
		}
		return true

	default:
		// Type to search (navNSearch)
		char := gs.keysymToChar(keysym)
		if char != 0 {
			gs.search += string(char)
			gs.updateFilter()
			return true
		}
	}

	return true // Consume all keys while grid is open
}

// keysymToChar converts a keysym to a character for typing
func (gs *GridSelect) keysymToChar(keysym xproto.Keysym) rune {
	// Letters a-z
	if keysym >= XK_a && keysym <= XK_z {
		return rune('a' + (keysym - XK_a))
	}
	// Numbers 0-9
	if keysym >= XK_0 && keysym <= XK_9 {
		return rune('0' + (keysym - XK_0))
	}
	// Space
	if keysym == XK_space {
		return ' '
	}
	// Common punctuation
	if keysym == XK_minus {
		return '-'
	}
	if keysym == XK_period {
		return '.'
	}
	return 0
}

// updateFilter filters items based on search string
func (gs *GridSelect) updateFilter() {
	if gs.search == "" {
		gs.filtered = gs.items
	} else {
		gs.filtered = nil
		searchLower := strings.ToLower(gs.search)
		for _, item := range gs.items {
			if strings.Contains(strings.ToLower(item.Label), searchLower) {
				gs.filtered = append(gs.filtered, item)
			}
		}
	}

	// Reset selection
	gs.selected = 0

	// Recreate window with new size
	if gs.visible {
		xproto.UngrabKeyboard(gs.wm.conn, xproto.TimeCurrentTime)
		xproto.FreeGC(gs.wm.conn, gs.gc)
		xproto.DestroyWindow(gs.wm.conn, gs.window)

		if len(gs.filtered) > 0 {
			gs.createWindow()
			gs.Draw()
		} else {
			// No matches, just show search bar
			gs.cols = 1
			gs.rows = 0
			gs.createWindow()
			gs.Draw()
		}
	}
}

// MoveSelection moves the selection in the grid
func (gs *GridSelect) MoveSelection(dx, dy int) {
	if len(gs.filtered) == 0 {
		return
	}

	col := gs.selected % gs.cols
	row := gs.selected / gs.cols

	newCol := col + dx
	newRow := row + dy

	// Wrap around
	if newCol < 0 {
		newCol = gs.cols - 1
		newRow--
	} else if newCol >= gs.cols {
		newCol = 0
		newRow++
	}

	if newRow < 0 {
		newRow = gs.rows - 1
	} else if newRow >= gs.rows {
		newRow = 0
	}

	newIdx := newRow*gs.cols + newCol
	if newIdx >= len(gs.filtered) {
		// Wrap to last item or first item
		if dx > 0 || dy > 0 {
			newIdx = 0
		} else {
			newIdx = len(gs.filtered) - 1
		}
	}

	gs.selected = newIdx
	gs.Draw()
}

// SelectCurrent selects the currently highlighted window
func (gs *GridSelect) SelectCurrent() {
	if len(gs.filtered) == 0 || gs.selected >= len(gs.filtered) {
		gs.Hide()
		return
	}

	item := gs.filtered[gs.selected]

	// Switch to workspace if needed
	if item.Workspace != gs.wm.current {
		gs.wm.switchToWorkspace(item.Workspace)
	}

	// Focus the window
	gs.wm.focus(item.Client)

	// Raise the window
	xproto.ConfigureWindow(gs.wm.conn, item.Client.Window,
		xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove})

	gs.Hide()
}

// Toggle shows or hides the grid select
func (gs *GridSelect) Toggle() {
	if gs.visible {
		gs.Hide()
	} else {
		gs.Show()
	}
}

// HandleExpose handles expose events for the grid window
func (gs *GridSelect) HandleExpose(e xproto.ExposeEvent) bool {
	if !gs.visible || e.Window != gs.window {
		return false
	}
	gs.Draw()
	return true
}
