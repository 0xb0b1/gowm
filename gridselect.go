package main

/*
#cgo pkg-config: xft x11
#include <X11/Xlib.h>
#include <X11/Xft/Xft.h>
#include <stdlib.h>
#include <string.h>

// Helper functions to work with Xft from Go

static XftFont* openFont(Display *dpy, int screen, const char *name) {
    return XftFontOpenName(dpy, screen, name);
}

static void closeFont(Display *dpy, XftFont *font) {
    if (font) XftFontClose(dpy, font);
}

static XftDraw* createDraw(Display *dpy, Drawable d, Visual *visual, Colormap cmap) {
    return XftDrawCreate(dpy, d, visual, cmap);
}

static void destroyDraw(XftDraw *draw) {
    if (draw) XftDrawDestroy(draw);
}

static void drawString(XftDraw *draw, XftColor *color, XftFont *font, int x, int y, const char *str, int len) {
    XftDrawStringUtf8(draw, color, font, x, y, (const FcChar8*)str, len);
}

static void allocColor(Display *dpy, Visual *visual, Colormap cmap, XftColor *color, unsigned int rgb) {
    XRenderColor xrc;
    xrc.red = ((rgb >> 16) & 0xFF) * 257;
    xrc.green = ((rgb >> 8) & 0xFF) * 257;
    xrc.blue = (rgb & 0xFF) * 257;
    xrc.alpha = 0xFFFF;
    XftColorAllocValue(dpy, visual, cmap, &xrc, color);
}

static void freeColor(Display *dpy, Visual *visual, Colormap cmap, XftColor *color) {
    XftColorFree(dpy, visual, cmap, color);
}

static int fontHeight(XftFont *font) {
    return font->ascent + font->descent;
}

static int fontAscent(XftFont *font) {
    return font->ascent;
}

static int textWidth(Display *dpy, XftFont *font, const char *str, int len) {
    XGlyphInfo extents;
    XftTextExtentsUtf8(dpy, font, (const FcChar8*)str, len, &extents);
    return extents.xOff;
}
*/
import "C"

import (
	"fmt"
	"log"
	"strings"
	"unsafe"

	"github.com/jezek/xgb/xproto"
)

// GridSelectMode determines the grid content type
type GridSelectMode int

const (
	GridModeWindows GridSelectMode = iota
	GridModeWorkspaces
	GridModeSpawn
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
	mode     GridSelectMode

	// Config matching xmonad
	cellW        int
	cellH        int
	padding      int
	originFractX float64 // 0.0 = left, 0.5 = center, 1.0 = right
	originFractY float64 // 0.0 = top, 0.5 = center, 1.0 = bottom

	// Xft resources for font rendering
	xftFont *C.XftFont
	xftDraw *C.XftDraw
	display *C.Display
}

// GridItem represents an item in the grid
type GridItem struct {
	Client    *Client // For window mode
	Workspace int     // Workspace index
	Label     string
	Action    string // Command to run (for spawn mode)
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

// colorRangeFromClassName creates a gradient color based on window class hash
// Mimics xmonad's colorRangeFromClassName with:
//   lowest bg: Base (0x303446)
//   highest bg: Surface0 (0x414559)
func colorFromClassHash(className string) uint32 {
	if className == "" {
		return gridColors.Base
	}

	// Simple hash of class name
	var hash uint32 = 0
	for _, c := range className {
		hash = hash*31 + uint32(c)
	}

	// Interpolate between Base and Surface0
	// Base:     0x303446 -> R=0x30, G=0x34, B=0x46
	// Surface0: 0x414559 -> R=0x41, G=0x45, B=0x59
	t := float64(hash%256) / 255.0

	r1, g1, b1 := uint32(0x30), uint32(0x34), uint32(0x46)
	r2, g2, b2 := uint32(0x41), uint32(0x45), uint32(0x59)

	r := uint32(float64(r1) + t*float64(r2-r1))
	g := uint32(float64(g1) + t*float64(g2-g1))
	b := uint32(float64(b1) + t*float64(b2-b1))

	return (r << 16) | (g << 8) | b
}

// GridSelect font configuration - using Vanilla Caramel size 12
const gridFontName = "Vanilla Caramel:size=12"

// Fallback fonts if Vanilla Caramel is not installed
var gridFontFallbacks = []string{
	"DejaVu Sans:style=Bold:size=12",
	"Liberation Sans:style=Bold:size=12",
	"Sans:style=Bold:size=12",
}

// NewGridSelect creates a new grid select instance
func NewGridSelect(wm *WindowManager) *GridSelect {
	gs := &GridSelect{
		wm:           wm,
		cellW:        280,  // Match xmonad: gs_cellwidth = 280
		cellH:        80,   // Match xmonad: gs_cellheight = 80
		padding:      8,    // Match xmonad: gs_cellpadding = 8
		originFractX: 0.5,  // Center horizontally
		originFractY: 0.5,  // Center vertically
		mode:         GridModeWindows,
	}

	// Open Xlib display connection for Xft
	displayName := C.CString("")
	defer C.free(unsafe.Pointer(displayName))
	gs.display = C.XOpenDisplay(displayName)
	if gs.display == nil {
		log.Println("GridSelect: Failed to open X display for Xft")
		return gs
	}

	// Load font - try primary first, then fallbacks
	// Use default screen (0) for font loading
	screenNum := C.XDefaultScreen(gs.display)
	fontName := C.CString(gridFontName)
	gs.xftFont = C.openFont(gs.display, screenNum, fontName)
	C.free(unsafe.Pointer(fontName))

	if gs.xftFont == nil {
		for _, fallback := range gridFontFallbacks {
			fontName = C.CString(fallback)
			gs.xftFont = C.openFont(gs.display, screenNum, fontName)
			C.free(unsafe.Pointer(fontName))
			if gs.xftFont != nil {
				log.Printf("GridSelect: Using fallback font: %s", fallback)
				break
			}
		}
	}

	if gs.xftFont == nil {
		log.Println("GridSelect: Failed to load any font, text will use default")
	} else {
		log.Printf("GridSelect: Loaded font, height=%d", int(C.fontHeight(gs.xftFont)))
	}

	return gs
}

// Show displays the grid select window
func (gs *GridSelect) Show() {
	if gs.visible {
		return
	}

	// Collect all windows from all workspaces
	gs.items = nil
	gs.search = ""
	gs.mode = GridModeWindows

	for wsIdx, ws := range gs.wm.workspaces {
		for _, client := range ws.Clients {
			title := gs.wm.getWindowTitle(client.Window)
			className := gs.wm.getWMClass(client.Window)
			if title == "" {
				title = className
			}
			if title == "" {
				title = "Unknown"
			}

			// Add workspace indicator
			label := ws.Name + ": " + title
			if len(label) > 40 {
				label = label[:37] + "..."
			}

			// Use colorRangeFromClassName style:
			// - Inactive bg: gradient from Base to Surface0 based on class hash
			// - Inactive fg: Text
			bgColor := colorFromClassHash(className)

			gs.items = append(gs.items, &GridItem{
				Client:    client,
				Workspace: wsIdx,
				Label:     label,
				BGColor:   bgColor,
				FGColor:   gridColors.Text, // Inactive uses Text color
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

// ShowWorkspaces displays a grid of workspaces for selection
func (gs *GridSelect) ShowWorkspaces() {
	if gs.visible {
		return
	}

	gs.items = nil
	gs.search = ""
	gs.mode = GridModeWorkspaces

	for i, ws := range gs.wm.workspaces {
		label := ws.Name
		windowCount := len(ws.Clients)
		if windowCount > 0 {
			label = fmt.Sprintf("%s (%d)", ws.Name, windowCount)
		}

		// Color based on workspace state
		var bgColor uint32
		if i == gs.wm.current {
			bgColor = gridColors.Surface1 // Current workspace
		} else if windowCount > 0 {
			bgColor = gridColors.Surface0 // Has windows
		} else {
			bgColor = gridColors.Base // Empty
		}

		gs.items = append(gs.items, &GridItem{
			Workspace: i,
			Label:     label,
			BGColor:   bgColor,
			FGColor:   gridColors.Text,
		})
	}

	gs.filtered = gs.items
	gs.selected = gs.wm.current
	gs.visible = true

	gs.createWindow()
	gs.Draw()
}

// SpawnItem represents an application to launch
type SpawnItem struct {
	Name    string
	Command string
}

// Default spawn items - can be customized in config
var defaultSpawnItems = []SpawnItem{
	{"Terminal", "alacritty"},
	{"Browser", "firefox"},
	{"File Manager", "thunar"},
	{"Editor", "code"},
	{"Discord", "discord"},
	{"Spotify", "spotify"},
	{"Steam", "steam"},
	{"OBS", "obs"},
}

// ShowSpawn displays a grid of applications to launch
func (gs *GridSelect) ShowSpawn() {
	if gs.visible {
		return
	}

	gs.items = nil
	gs.search = ""
	gs.mode = GridModeSpawn

	for i, app := range defaultSpawnItems {
		bgColor := colorFromClassHash(app.Name)
		gs.items = append(gs.items, &GridItem{
			Workspace: i, // Reuse for index
			Label:     app.Name,
			Action:    app.Command,
			BGColor:   bgColor,
			FGColor:   gridColors.Text,
		})
	}

	if len(gs.items) == 0 {
		log.Println("GridSelect: No spawn items configured")
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
		n = 1 // At least show search bar
	}

	// Calculate optimal grid dimensions (as square as possible, like xmonad)
	// Find the most square-like arrangement
	gs.cols = 1
	gs.rows = n
	bestDiff := gs.rows - gs.cols

	for c := 1; c <= n; c++ {
		r := (n + c - 1) / c
		diff := r - c
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff || (diff == bestDiff && c > gs.cols) {
			bestDiff = diff
			gs.cols = c
			gs.rows = r
		}
	}

	// Limit max columns based on screen width
	screenW := int(gs.wm.screen.WidthInPixels)
	maxCols := (screenW - gs.padding*2) / (gs.cellW + gs.padding)
	if gs.cols > maxCols {
		gs.cols = maxCols
		gs.rows = (n + gs.cols - 1) / gs.cols
	}

	gridW := gs.cols*(gs.cellW+gs.padding) + gs.padding
	gridH := gs.rows*(gs.cellH+gs.padding) + gs.padding

	// Position using originFractX/Y (0.5 = center)
	screenH := int(gs.wm.screen.HeightInPixels)
	x := int(float64(screenW-gridW) * gs.originFractX)
	y := int(float64(screenH-gridH) * gs.originFractY)

	// Calculate item positions in grid
	for i, item := range gs.filtered {
		col := i % gs.cols
		row := i / gs.cols
		item.X = gs.padding + col*(gs.cellW+gs.padding)
		item.Y = gs.padding + row*(gs.cellH+gs.padding)
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
			xproto.EventMaskExposure | xproto.EventMaskKeyPress |
				xproto.EventMaskButtonPress | xproto.EventMaskPointerMotion,
		},
	)

	// Create graphics context
	gs.gc, _ = xproto.NewGcontextId(gs.wm.conn)
	xproto.CreateGC(gs.wm.conn, gs.gc, xproto.Drawable(gs.window), 0, nil)

	// Map window and grab keyboard
	xproto.MapWindow(gs.wm.conn, gs.window)
	xproto.SetInputFocus(gs.wm.conn, xproto.InputFocusPointerRoot, gs.window, xproto.TimeCurrentTime)

	// Create XftDraw for the window (must be done after MapWindow)
	if gs.display != nil && gs.xftFont != nil {
		screen := C.XDefaultScreenOfDisplay(gs.display)
		visual := C.XDefaultVisualOfScreen(screen)
		colormap := C.XDefaultColormapOfScreen(screen)

		// Sync xgb connection to ensure window is visible to Xlib
		gs.wm.conn.Sync()

		gs.xftDraw = C.createDraw(gs.display, C.Drawable(gs.window), visual, colormap)
	}

	// Grab keyboard
	xproto.GrabKeyboard(
		gs.wm.conn,
		true,
		gs.window,
		xproto.TimeCurrentTime,
		xproto.GrabModeAsync,
		xproto.GrabModeAsync,
	)

	// Grab pointer for mouse selection
	xproto.GrabPointer(
		gs.wm.conn,
		true,
		gs.window,
		xproto.EventMaskButtonPress|xproto.EventMaskPointerMotion,
		xproto.GrabModeAsync,
		xproto.GrabModeAsync,
		gs.window,
		xproto.CursorNone,
		xproto.TimeCurrentTime,
	)
}

// Hide closes the grid select window
func (gs *GridSelect) Hide() {
	if !gs.visible {
		return
	}

	// Clean up XftDraw
	if gs.xftDraw != nil {
		C.destroyDraw(gs.xftDraw)
		gs.xftDraw = nil
	}

	xproto.UngrabPointer(gs.wm.conn, xproto.TimeCurrentTime)
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
	if !gs.visible {
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

	// Draw items
	for i, item := range gs.filtered {
		gs.drawItem(i, item)
	}

	// If searching, show search text at bottom
	if gs.search != "" && geom != nil {
		searchText := "/" + gs.search
		searchY := int(geom.Height) - gs.padding
		gs.drawText(searchText, gs.padding, searchY, gridColors.Text)
	}
}

// drawText draws text using Xft if available, otherwise falls back to X11 core
func (gs *GridSelect) drawText(text string, x, y int, color uint32) {
	if gs.xftDraw != nil && gs.xftFont != nil && gs.display != nil {
		// Use Xft for anti-aliased text
		screen := C.XDefaultScreenOfDisplay(gs.display)
		visual := C.XDefaultVisualOfScreen(screen)
		colormap := C.XDefaultColormapOfScreen(screen)

		var xftColor C.XftColor
		C.allocColor(gs.display, visual, colormap, &xftColor, C.uint(color))

		cstr := C.CString(text)
		C.drawString(gs.xftDraw, &xftColor, gs.xftFont, C.int(x), C.int(y), cstr, C.int(len(text)))
		C.free(unsafe.Pointer(cstr))

		C.freeColor(gs.display, visual, colormap, &xftColor)

		// Flush to ensure drawing is visible
		C.XFlush(gs.display)
	} else {
		// Fallback to X11 core text
		xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{color})
		xproto.ImageText8(
			gs.wm.conn,
			byte(len(text)),
			xproto.Drawable(gs.window),
			gs.gc,
			int16(x),
			int16(y),
			text,
		)
	}
}

// getTextWidth returns the width of text in pixels
func (gs *GridSelect) getTextWidth(text string) int {
	if gs.xftFont != nil && gs.display != nil {
		cstr := C.CString(text)
		width := int(C.textWidth(gs.display, gs.xftFont, cstr, C.int(len(text))))
		C.free(unsafe.Pointer(cstr))
		return width
	}
	// Fallback: approximate 7 pixels per character
	return len(text) * 7
}

// getFontHeight returns the font height in pixels
func (gs *GridSelect) getFontHeight() int {
	if gs.xftFont != nil {
		return int(C.fontHeight(gs.xftFont))
	}
	return 14 // Default height
}

// drawItem draws a single grid item
func (gs *GridSelect) drawItem(idx int, item *GridItem) {
	isSelected := idx == gs.selected

	var bg, fg uint32
	if isSelected {
		// Active: Mauve bg (0xca9ee6), Crust fg (0x232634)
		bg = gridColors.Mauve
		fg = gridColors.Crust
	} else {
		// Inactive: gradient bg (Base to Surface0), Text fg (0xc6d0f5)
		bg = item.BGColor
		fg = item.FGColor
	}

	// Draw cell background
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{bg})
	xproto.PolyFillRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
		{X: int16(item.X), Y: int16(item.Y), Width: uint16(item.W), Height: uint16(item.H)},
	})

	// Draw border (thicker for selected)
	borderColor := gridColors.Surface1
	if isSelected {
		borderColor = gridColors.Text
	}
	xproto.ChangeGC(gs.wm.conn, gs.gc, xproto.GcForeground, []uint32{borderColor})
	xproto.PolyRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
		{X: int16(item.X), Y: int16(item.Y), Width: uint16(item.W - 1), Height: uint16(item.H - 1)},
	})
	if isSelected {
		xproto.PolyRectangle(gs.wm.conn, xproto.Drawable(gs.window), gs.gc, []xproto.Rectangle{
			{X: int16(item.X + 1), Y: int16(item.Y + 1), Width: uint16(item.W - 3), Height: uint16(item.H - 3)},
		})
	}

	// Draw text (centered in cell)
	textW := gs.getTextWidth(item.Label)
	textX := item.X + (item.W-textW)/2
	if textX < item.X+8 {
		textX = item.X + 8
	}

	// Vertical centering: baseline at cell center + ascent/2
	fontH := gs.getFontHeight()
	textY := item.Y + item.H/2 + fontH/4

	gs.drawText(item.Label, textX, textY, fg)
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

	// Reset selection if out of bounds
	if gs.selected >= len(gs.filtered) {
		gs.selected = 0
	}

	// Recreate window with new size
	if gs.visible {
		// Clean up XftDraw before destroying window
		if gs.xftDraw != nil {
			C.destroyDraw(gs.xftDraw)
			gs.xftDraw = nil
		}

		xproto.UngrabPointer(gs.wm.conn, xproto.TimeCurrentTime)
		xproto.UngrabKeyboard(gs.wm.conn, xproto.TimeCurrentTime)
		xproto.FreeGC(gs.wm.conn, gs.gc)
		xproto.DestroyWindow(gs.wm.conn, gs.window)

		if len(gs.filtered) > 0 {
			gs.createWindow()
		}
		gs.Draw()
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

// SelectCurrent selects the currently highlighted item based on mode
func (gs *GridSelect) SelectCurrent() {
	if len(gs.filtered) == 0 || gs.selected >= len(gs.filtered) {
		gs.Hide()
		return
	}

	item := gs.filtered[gs.selected]

	switch gs.mode {
	case GridModeWindows:
		// Switch to workspace if needed
		if item.Workspace != gs.wm.current {
			gs.wm.switchToWorkspace(item.Workspace)
		}
		// Focus the window
		if item.Client != nil {
			gs.wm.focus(item.Client)
			// Raise the window
			xproto.ConfigureWindow(gs.wm.conn, item.Client.Window,
				xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove})
		}

	case GridModeWorkspaces:
		// Switch to selected workspace
		gs.wm.switchToWorkspace(item.Workspace)

	case GridModeSpawn:
		// Launch the application
		if item.Action != "" {
			spawn(item.Action)
		}
	}

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

// HandleButtonPress handles mouse clicks in the grid
func (gs *GridSelect) HandleButtonPress(e xproto.ButtonPressEvent) bool {
	if !gs.visible || e.Event != gs.window {
		return false
	}

	x, y := int(e.EventX), int(e.EventY)

	// Check if click is on an item
	for i, item := range gs.filtered {
		if x >= item.X && x < item.X+item.W && y >= item.Y && y < item.Y+item.H {
			gs.selected = i
			if e.Detail == xproto.ButtonIndex1 {
				// Left click - select
				gs.SelectCurrent()
			} else {
				// Other buttons - just highlight
				gs.Draw()
			}
			return true
		}
	}

	// Click on empty space - close grid (like xmonad's gs_cancelOnEmptyClick)
	if e.Detail == xproto.ButtonIndex1 {
		gs.Hide()
	}
	return true
}

// HandleMotionNotify handles mouse movement for hover highlighting
func (gs *GridSelect) HandleMotionNotify(e xproto.MotionNotifyEvent) bool {
	if !gs.visible || e.Event != gs.window {
		return false
	}

	x, y := int(e.EventX), int(e.EventY)

	// Check if hovering over an item
	for i, item := range gs.filtered {
		if x >= item.X && x < item.X+item.W && y >= item.Y && y < item.Y+item.H {
			if gs.selected != i {
				gs.selected = i
				gs.Draw()
			}
			return true
		}
	}
	return true
}

// Close cleans up Xft resources
func (gs *GridSelect) Close() {
	gs.Hide()

	if gs.xftFont != nil && gs.display != nil {
		C.closeFont(gs.display, gs.xftFont)
		gs.xftFont = nil
	}

	if gs.display != nil {
		C.XCloseDisplay(gs.display)
		gs.display = nil
	}
}
