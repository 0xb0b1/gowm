# gowm

A minimal, pure Go tiling window manager for X11, inspired by [xmonad](https://xmonad.org/).

```
┌────────────────────────────────────────────────────────┐
│                      eww bar                           │
├───────────────────────────┬────────────────────────────┤
│                           │                            │
│                           │         terminal 2         │
│        terminal 1         ├────────────────────────────┤
│         (master)          │                            │
│                           │         terminal 3         │
│                           │                            │
└───────────────────────────┴────────────────────────────┘
```

## Features

- **Tiling Layouts** - Tall (master/stack), Full (monocle), Grid
- **9 Workspaces** - Quick switching with `Super+1-9`
- **EWMH Compliant** - Works with panels, bars, and pagers
- **Strut Support** - Automatically tiles around eww, polybar, etc.
- **Scratchpad** - Toggle-able floating terminal with `Super+``
- **GridSelect** - Visual window picker with Xft fonts and search (`Super+g`)
- **Mouse Support** - Move/resize floating windows with Super+drag
- **Window Rules** - Auto-float and workspace assignment by WM_CLASS
- **Urgent Hints** - Red border for windows requesting attention
- **IPC Socket** - External control via `gowmctl` commands
- **Compile-time Config** - Edit `config.go` and rebuild (like xmonad)
- **Catppuccin Theme** - Frappe color palette built-in
- **Autostart** - Launch compositor, bar, and apps on startup
- **Restart Persistence** - Windows stay on their workspaces after restart
- **~4400 lines of Go** - Simple, hackable codebase

## Installation

### From Source

```bash
git clone https://github.com/0xb0b1/gowm.git
cd gowm
go build -o gowm .
sudo cp gowm /usr/local/bin/
```

### Desktop Entry

Create `/usr/share/xsessions/gowm.desktop`:

```ini
[Desktop Entry]
Name=gowm
Comment=Pure Go Tiling Window Manager
Exec=/usr/local/bin/gowm
Type=Application
```

### With startx

Add to `~/.xinitrc`:

```bash
exec /usr/local/bin/gowm
```

## Configuration

Edit `config.go` and rebuild. Configuration is compile-time for type safety and performance.

### Appearance

```go
BorderWidth:          2,
GapWidth:             2,
FocusedBorderColor:   ColorLavender,  // #babbf1
UnfocusedBorderColor: ColorSurface0,  // #414559
```

### Default Applications

```go
Terminal: "kitty",
Launcher: "rofi -show drun",
```

### Startup Applications

Edit `startup.go` to customize autostart:

```go
AutostartAlways: []string{
    "dunst",
    "picom --config ~/.config/picom/picom.conf",
    "~/.config/eww/launch.sh start",
},
```

## Keybindings

### Window Management

| Key | Action |
|-----|--------|
| `Super+Return` | Launch terminal |
| `Super+Shift+Return` | Launch floating terminal |
| `Super+d` / `Super+r` | Application launcher |
| `Super+q` | Close window |
| `Super+Shift+q` | Close all windows |
| `Super+g` | GridSelect (visual window picker) |

### Focus & Movement

| Key | Action |
|-----|--------|
| `Super+j` | Focus next |
| `Super+k` | Focus previous |
| `Super+m` | Focus master |
| `Super+Shift+j` | Swap with next |
| `Super+Shift+k` | Swap with previous |

### Layout

| Key | Action |
|-----|--------|
| `Super+Space` | Next layout |
| `Super+Shift+Space` | Reset to tall |
| `Super+h` | Shrink master |
| `Super+l` | Expand master |
| `Super+,` | Add master window |
| `Super+.` | Remove master window |
| `Super+s` | Sink floating window |

### Workspaces

| Key | Action |
|-----|--------|
| `Super+1-9` | Switch to workspace |
| `Super+Shift+1-9` | Move window to workspace |

### Media Keys

| Key | Action |
|-----|--------|
| `XF86AudioMute` | Toggle mute |
| `XF86AudioLower/Raise` | Volume down/up |
| `XF86AudioPlay/Next/Prev` | Media controls |
| `XF86MonBrightness*` | Brightness controls |

### Scratchpad & Floating

| Key | Action |
|-----|--------|
| `Super+`` | Toggle scratchpad terminal |
| `Super+s` | Sink floating window to tiled |
| `Super+Button1` | Move floating window |
| `Super+Button3` | Resize floating window |

### System

| Key | Action |
|-----|--------|
| `Super+Shift+r` | Restart gowm |
| `Super+Ctrl+q` | Quit gowm |

## Layouts

### Tall (Default)

Master window on the left, stack on the right. Adjustable ratio with `Super+h/l`.

```
┌──────────┬───────┐
│          │   2   │
│    1     ├───────┤
│ (master) │   3   │
│          ├───────┤
│          │   4   │
└──────────┴───────┘
```

### Full

All windows maximized, cycle with focus keys.

```
┌─────────────────┐
│                 │
│        1        │
│   (fullscreen)  │
│                 │
└─────────────────┘
```

### Grid

Equal-sized grid arrangement.

```
┌────────┬────────┐
│   1    │   2    │
├────────┼────────┤
│   3    │   4    │
└────────┴────────┘
```

## Testing

Test safely in a nested X server:

```bash
# Install Xephyr if needed
# Arch: sudo pacman -S xorg-server-xephyr
# Debian/Ubuntu: sudo apt install xserver-xephyr

# Start nested X server
Xephyr :1 -screen 1920x1080 -ac &

# Run gowm in it
DISPLAY=:1 ./gowm &

# Launch test applications
DISPLAY=:1 alacritty &
DISPLAY=:1 thunar &
```

## Dependencies

- Go 1.21+
- X11 server
- libxft-dev (for anti-aliased fonts in GridSelect)
- [jezek/xgb](https://github.com/jezek/xgb) - X11 protocol bindings
- [jezek/xgbutil](https://github.com/jezek/xgbutil) - X11 utilities

## IPC Control

Control gowm externally using the `gowmctl` script:

```bash
# Copy gowmctl to your PATH
sudo cp gowmctl /usr/local/bin/

# Switch workspace
gowmctl workspace switch 3

# Query windows
gowmctl query windows

# Close focused window
gowmctl window close

# Toggle scratchpad
gowmctl action scratchpad

# See all commands
gowmctl help
```

## Window Rules

Edit `rules.go` to customize auto-float and workspace assignment:

```go
// Auto-float specific applications
{Class: "pavucontrol", Floating: &floating},
{Class: "steam", Title: "Friends List", Floating: &floating},

// Assign apps to specific workspaces
{Class: "discord", Workspace: intPtr(8)},
{Class: "spotify", Workspace: intPtr(9)},
```

## Project Structure

```
gowm/
├── main.go          # Entry point, event loop
├── wm.go            # WindowManager core logic
├── client.go        # Window management
├── workspace.go     # Workspace handling
├── layout.go        # Layout interface
├── layout_tall.go   # Master/stack layout
├── layout_full.go   # Monocle layout
├── layout_grid.go   # Grid layout
├── config.go        # Configuration & keybindings
├── keysym.go        # X11 keysym definitions
├── actions.go       # Keybinding actions
├── startup.go       # Autostart handling
├── atoms.go         # X11 atom management
├── ewmh.go          # EWMH compliance
├── scratchpad.go    # Scratchpad functionality
├── gridselect.go    # GridSelect window picker
├── mouse.go         # Mouse move/resize
├── rules.go         # Window rules
├── urgent.go        # Urgent hints handling
├── ipc.go           # IPC socket server
├── gowmctl          # IPC client script
└── rect.go          # Geometry utilities
```

## Inspiration

- [xmonad](https://xmonad.org/) - The original Haskell tiling WM
- [dwm](https://dwm.suckless.org/) - Suckless dynamic window manager
- [wingo](https://github.com/BurntSushi/wingo) - Go window manager by xgb author

## License

MIT
