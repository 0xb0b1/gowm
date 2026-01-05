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
- **Compile-time Config** - Edit `config.go` and rebuild (like xmonad)
- **Catppuccin Theme** - Frappe color palette built-in
- **Autostart** - Launch compositor, bar, and apps on startup
- **~2500 lines of Go** - Simple, hackable, no runtime dependencies

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
- [jezek/xgb](https://github.com/jezek/xgb) - X11 protocol bindings
- [jezek/xgbutil](https://github.com/jezek/xgbutil) - X11 utilities

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
└── rect.go          # Geometry utilities
```

## Inspiration

- [xmonad](https://xmonad.org/) - The original Haskell tiling WM
- [dwm](https://dwm.suckless.org/) - Suckless dynamic window manager
- [wingo](https://github.com/BurntSushi/wingo) - Go window manager by xgb author

## License

MIT
