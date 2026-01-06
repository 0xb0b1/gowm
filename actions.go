package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

// ActionSpawn returns an action that spawns a command
func ActionSpawn(cmd string) Action {
	return func(wm *WindowManager) {
		log.Printf("Spawning: %s", cmd)
		c := exec.Command("sh", "-c", cmd)
		c.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		if err := c.Start(); err != nil {
			log.Printf("Failed to spawn %s: %v", cmd, err)
		}
	}
}

// ActionKill closes the focused window
func ActionKill(wm *WindowManager) {
	if wm.focused == nil {
		return
	}

	// Try WM_DELETE_WINDOW first for graceful close
	if wm.supportsProtocol(wm.focused.Window, wm.atoms.WM_DELETE_WINDOW) {
		wm.sendDeleteWindow(wm.focused)
	} else {
		wm.destroyClient(wm.focused)
	}
}

// ActionKillAll closes all windows in the current workspace
func ActionKillAll(wm *WindowManager) {
	ws := wm.currentWorkspace()
	// Make a copy since we'll be modifying the slice
	clients := make([]*Client, len(ws.Clients))
	copy(clients, ws.Clients)

	for _, c := range clients {
		if wm.supportsProtocol(c.Window, wm.atoms.WM_DELETE_WINDOW) {
			wm.sendDeleteWindow(c)
		} else {
			wm.destroyClient(c)
		}
	}
}

// ActionFocusNext focuses the next window
func ActionFocusNext(wm *WindowManager) {
	ws := wm.currentWorkspace()
	if c := ws.FocusNext(); c != nil {
		wm.focus(c)
	}
}

// ActionFocusPrev focuses the previous window
func ActionFocusPrev(wm *WindowManager) {
	ws := wm.currentWorkspace()
	if c := ws.FocusPrev(); c != nil {
		wm.focus(c)
	}
}

// ActionFocusMaster focuses the master window
func ActionFocusMaster(wm *WindowManager) {
	ws := wm.currentWorkspace()
	if c := ws.FocusMaster(); c != nil {
		wm.focus(c)
	}
}

// ActionSwapNext swaps the focused window with the next
func ActionSwapNext(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.SwapNext()
	wm.tile()
}

// ActionSwapPrev swaps the focused window with the previous
func ActionSwapPrev(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.SwapPrev()
	wm.tile()
}

// ActionSwapMaster swaps the focused window with the master
func ActionSwapMaster(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.SwapMaster()
	wm.tile()
}

// ActionShrink shrinks the master area
func ActionShrink(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.Layout.HandleMessage(LayoutMsgShrink)
	wm.tile()
}

// ActionExpand expands the master area
func ActionExpand(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.Layout.HandleMessage(LayoutMsgExpand)
	wm.tile()
}

// ActionIncMaster increases the number of master windows
func ActionIncMaster(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.Layout.HandleMessage(LayoutMsgIncMaster)
	wm.tile()
}

// ActionDecMaster decreases the number of master windows
func ActionDecMaster(wm *WindowManager) {
	ws := wm.currentWorkspace()
	ws.Layout.HandleMessage(LayoutMsgDecMaster)
	wm.tile()
}

// ActionNextLayout cycles to the next layout
func ActionNextLayout(wm *WindowManager) {
	ws := wm.currentWorkspace()
	// Sink all floating windows back to tiled when switching layouts
	for _, c := range ws.Clients {
		c.Floating = false
	}
	ws.NextLayout(wm.layouts)
	wm.tile()
	log.Printf("Layout: %s", ws.Layout.Name())
	// Notify via dunst
	spawn("notify-send -t 1000 'Layout' '%s'", ws.Layout.Name())
}

// ActionResetLayout resets to the default layout
func ActionResetLayout(wm *WindowManager) {
	ws := wm.currentWorkspace()
	// Sink all floating windows back to tiled when resetting layout
	for _, c := range ws.Clients {
		c.Floating = false
	}
	ws.Layout = NewTallLayout()
	wm.tile()
	log.Printf("Layout reset: %s", ws.Layout.Name())
	// Notify via dunst
	spawn("notify-send -t 1000 'Layout' '%s'", ws.Layout.Name())
}

// ActionSink sinks a floating window back to tiled
func ActionSink(wm *WindowManager) {
	if wm.focused != nil && wm.focused.Floating {
		wm.focused.Floating = false
		wm.tile()
	}
}

// ActionToggleFloat toggles the focused window between floating and tiled
func ActionToggleFloat(wm *WindowManager) {
	if wm.focused != nil {
		wm.focused.Floating = !wm.focused.Floating
		wm.tile()
	}
}

// ActionGridSelect shows the grid select window picker
func ActionGridSelect(wm *WindowManager) {
	wm.gridSelect.Toggle()
}

// ActionGridSelectWorkspaces shows workspace grid selector
func ActionGridSelectWorkspaces(wm *WindowManager) {
	wm.gridSelect.ShowWorkspaces()
}

// ActionGridSelectSpawn shows application launcher grid
func ActionGridSelectSpawn(wm *WindowManager) {
	wm.gridSelect.ShowSpawn()
}

// ActionSwitchWorkspace returns an action that switches to workspace n
func ActionSwitchWorkspace(n int) Action {
	return func(wm *WindowManager) {
		wm.switchToWorkspace(n)
	}
}

// ActionMoveToWorkspace returns an action that moves the focused window to workspace n
func ActionMoveToWorkspace(n int) Action {
	return func(wm *WindowManager) {
		if wm.focused != nil {
			wm.moveToWorkspace(wm.focused, n)
		}
	}
}

// ActionRestart restarts the window manager
func ActionRestart(wm *WindowManager) {
	log.Println("Restarting...")
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		return
	}

	// Exec into new instance
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		log.Printf("Failed to restart: %v", err)
	}
}

// ActionQuit exits the window manager
func ActionQuit(wm *WindowManager) {
	log.Println("Quitting...")
	wm.running = false
}

// ActionToggleScratchpad toggles the scratchpad visibility
func ActionToggleScratchpad(wm *WindowManager) {
	wm.ToggleScratchpad()
}
