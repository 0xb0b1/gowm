package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// StartupConfig holds startup applications and settings
type StartupConfig struct {
	// X11 settings
	SetWMName          string
	KeyboardLayout     string
	KeyboardOptions    string
	KeyRepeatDelay     int
	KeyRepeatRate      int
	DisableScreenSaver bool
	DisableDPMS        bool

	// Autostart applications (run once)
	AutostartOnce []string

	// Autostart applications (always run, kill existing first)
	AutostartAlways []string

	// Background
	WallpaperCommand string
}

// DefaultStartupConfig returns startup config matching your xmonad setup
func DefaultStartupConfig() *StartupConfig {
	return &StartupConfig{
		SetWMName:          "LG3D", // Java/game compatibility
		KeyboardLayout:     "us",
		KeyboardOptions:    "caps:escape",
		KeyRepeatDelay:     200,
		KeyRepeatRate:      40,
		DisableScreenSaver: true,
		DisableDPMS:        true,

		AutostartOnce: []string{
			"parcellite",
		},

		AutostartAlways: []string{
			"dunst",
			"picom --config ~/.config/picom/picom.conf",
			"redshift -l 0.01:-99.0 -g 0.8 -t 5200:5200 -r",
			"~/.config/eww/launch.sh start",
		},

		WallpaperCommand: "nitrogen --restore",
	}
}

// runStartup runs startup commands
func (wm *WindowManager) runStartup() {
	cfg := DefaultStartupConfig()

	log.Println("Running startup...")

	// X11 settings
	if cfg.DisableScreenSaver {
		spawn("xset s off")
	}
	if cfg.DisableDPMS {
		spawn("xset -dpms")
	}
	if cfg.KeyRepeatDelay > 0 && cfg.KeyRepeatRate > 0 {
		spawn("xset r rate %d %d", cfg.KeyRepeatDelay, cfg.KeyRepeatRate)
	}

	// Cursor
	spawn("xsetroot -cursor_name left_ptr")

	// WM name for Java/game compatibility
	if cfg.SetWMName != "" {
		spawn("wmname %s", cfg.SetWMName)
	}

	// Keyboard
	if cfg.KeyboardLayout != "" {
		if cfg.KeyboardOptions != "" {
			spawn("setxkbmap -option %s -layout %s", cfg.KeyboardOptions, cfg.KeyboardLayout)
		} else {
			spawn("setxkbmap -layout %s", cfg.KeyboardLayout)
		}
	}

	// Load Xresources
	if _, err := os.Stat(os.ExpandEnv("$HOME/.Xresources")); err == nil {
		spawn("xrdb -load ~/.Xresources")
	}

	// Display settings (xrandr) - using your config
	spawn("xrandr --output DP-2 --primary --mode 1920x1080 --rate 165.00")

	// Background
	if cfg.WallpaperCommand != "" {
		spawn(cfg.WallpaperCommand)
	}

	// Kill and restart always-run apps
	for _, cmd := range cfg.AutostartAlways {
		// Extract program name for killing
		progName := extractProgramName(cmd)
		if progName != "" {
			spawn("pkill -x %s", progName)
		}
	}

	// Small delay to let processes die
	time.Sleep(500 * time.Millisecond)

	// Start always-run apps
	for _, cmd := range cfg.AutostartAlways {
		spawnOnce(cmd)
	}

	// Start once-only apps (check if already running)
	for _, cmd := range cfg.AutostartOnce {
		progName := extractProgramName(cmd)
		if !isRunning(progName) {
			spawnOnce(cmd)
		}
	}

	log.Println("Startup complete")
}

// spawn runs a command and doesn't wait
func spawn(format string, args ...interface{}) {
	cmd := format
	if len(args) > 0 {
		cmd = fmt.Sprintf(format, args...)
	}

	c := exec.Command("sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		log.Printf("Failed to spawn '%s': %v", cmd, err)
	}
}

// spawnOnce spawns a command that should keep running
func spawnOnce(cmd string) {
	c := exec.Command("sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		log.Printf("Failed to spawn '%s': %v", cmd, err)
	} else {
		log.Printf("Started: %s", cmd)
	}
}

// extractProgramName gets the program name from a command
func extractProgramName(cmd string) string {
	// Handle paths like ~/.config/eww/launch.sh
	if cmd[0] == '~' || cmd[0] == '/' || cmd[0] == '.' {
		return ""
	}

	// Get first word
	for i, c := range cmd {
		if c == ' ' {
			return cmd[:i]
		}
	}
	return cmd
}

// isRunning checks if a program is already running
func isRunning(name string) bool {
	if name == "" {
		return false
	}
	cmd := exec.Command("pgrep", "-x", name)
	return cmd.Run() == nil
}
