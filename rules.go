package main

import (
	"log"
	"strings"

	"github.com/jezek/xgb/xproto"
)

// WindowRule defines rules for matching and handling windows
type WindowRule struct {
	Class     string // WM_CLASS to match (case-insensitive, supports prefix)
	Instance  string // WM_CLASS instance to match (optional)
	Title     string // Window title to match (optional, prefix match)
	Floating  *bool  // Force floating if set
	Workspace *int   // Assign to workspace if set
}

// DefaultRules returns the default window rules
func DefaultRules() []WindowRule {
	floating := true
	return []WindowRule{
		// Dialogs and popups
		{Class: "floating", Floating: &floating},
		{Class: "dialog", Floating: &floating},
		{Class: "popup", Floating: &floating},

		// Common floating applications
		{Class: "pavucontrol", Floating: &floating},
		{Class: "nm-connection-editor", Floating: &floating},
		{Class: "blueman-manager", Floating: &floating},
		{Class: "lxappearance", Floating: &floating},
		{Class: "qt5ct", Floating: &floating},
		{Class: "nwg-look", Floating: &floating},
		{Class: "file-roller", Floating: &floating},
		{Class: "ark", Floating: &floating},
		{Class: "gnome-calculator", Floating: &floating},
		{Class: "galculator", Floating: &floating},

		// Steam
		{Class: "steam", Title: "Friends List", Floating: &floating},
		{Class: "steam", Title: "Steam - News", Floating: &floating},

		// Media viewers
		{Class: "mpv", Floating: &floating},
		{Class: "feh", Floating: &floating},
		{Class: "imv", Floating: &floating},
		{Class: "eog", Floating: &floating},
		{Class: "gpicview", Floating: &floating},
		{Class: "sxiv", Floating: &floating},

		// Settings
		{Class: "gnome-control-center", Floating: &floating},
		{Class: "xfce4-settings", Floating: &floating},

		// Pinentry (GPG)
		{Class: "pinentry", Floating: &floating},
		{Class: "gcr-prompter", Floating: &floating},

		// Zoom
		{Class: "zoom", Floating: &floating},

		// Workspace assignments (examples - customize as needed)
		// {Class: "firefox", Workspace: intPtr(1)},
		// {Class: "discord", Workspace: intPtr(8)},
		// {Class: "spotify", Workspace: intPtr(9)},
	}
}

// intPtr is a helper to create int pointer for workspace assignment
func intPtr(i int) *int {
	return &i
}

// applyRules applies window rules to a window and returns float/workspace settings
func (wm *WindowManager) applyRules(win xproto.Window) (shouldFloat bool, workspace *int) {
	class := strings.ToLower(wm.getWMClass(win))
	instance := strings.ToLower(wm.getWMInstance(win))
	title := strings.ToLower(wm.getWindowTitle(win))

	log.Printf("Checking rules for window: class=%q instance=%q title=%q", class, instance, title)

	for _, rule := range wm.rules {
		if !wm.matchRule(rule, class, instance, title) {
			continue
		}

		log.Printf("Rule matched: class=%q", rule.Class)

		if rule.Floating != nil {
			shouldFloat = *rule.Floating
		}
		if rule.Workspace != nil {
			workspace = rule.Workspace
		}
	}

	return shouldFloat, workspace
}

// matchRule checks if a window matches a rule
func (wm *WindowManager) matchRule(rule WindowRule, class, instance, title string) bool {
	// Class must match if specified
	if rule.Class != "" {
		ruleClass := strings.ToLower(rule.Class)
		if !strings.HasPrefix(class, ruleClass) && !strings.Contains(class, ruleClass) {
			return false
		}
	}

	// Instance must match if specified
	if rule.Instance != "" {
		ruleInstance := strings.ToLower(rule.Instance)
		if !strings.HasPrefix(instance, ruleInstance) && !strings.Contains(instance, ruleInstance) {
			return false
		}
	}

	// Title must match if specified
	if rule.Title != "" {
		ruleTitle := strings.ToLower(rule.Title)
		if !strings.Contains(title, ruleTitle) {
			return false
		}
	}

	return true
}

// getWMInstance returns the WM_CLASS instance name
func (wm *WindowManager) getWMInstance(win xproto.Window) string {
	reply, err := xproto.GetProperty(wm.conn, false, win,
		xproto.AtomWmClass, xproto.AtomString, 0, 256).Reply()
	if err != nil || reply == nil || len(reply.Value) == 0 {
		return ""
	}

	// WM_CLASS format: instance\0class\0
	parts := strings.Split(string(reply.Value), "\x00")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// getWindowTitle returns the window title
func (wm *WindowManager) getWindowTitle(win xproto.Window) string {
	// Try _NET_WM_NAME first
	reply, err := xproto.GetProperty(wm.conn, false, win,
		wm.atoms.NET_WM_NAME, wm.atoms.UTF8_STRING, 0, 1024).Reply()
	if err == nil && reply != nil && len(reply.Value) > 0 {
		return string(reply.Value)
	}

	// Fall back to WM_NAME
	reply, err = xproto.GetProperty(wm.conn, false, win,
		xproto.AtomWmName, xproto.AtomString, 0, 1024).Reply()
	if err == nil && reply != nil && len(reply.Value) > 0 {
		return string(reply.Value)
	}

	return ""
}
