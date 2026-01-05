package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IPCServer handles IPC communication via Unix socket
type IPCServer struct {
	wm       *WindowManager
	listener net.Listener
	sockPath string
}

// IPCResponse represents a response to an IPC command
type IPCResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// WorkspaceInfo represents workspace information for IPC
type WorkspaceInfo struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
	Windows int    `json:"windows"`
}

// WindowInfo represents window information for IPC
type WindowInfo struct {
	ID        uint32 `json:"id"`
	Title     string `json:"title"`
	Class     string `json:"class"`
	Workspace int    `json:"workspace"`
	Floating  bool   `json:"floating"`
	Focused   bool   `json:"focused"`
	Urgent    bool   `json:"urgent"`
}

// NewIPCServer creates a new IPC server
func NewIPCServer(wm *WindowManager) (*IPCServer, error) {
	// Create socket path in runtime dir or /tmp
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	sockPath := filepath.Join(runtimeDir, "gowm.sock")

	// Remove existing socket
	os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPC socket: %v", err)
	}

	// Set permissions (rw for user only)
	os.Chmod(sockPath, 0600)

	log.Printf("IPC socket created at %s", sockPath)

	return &IPCServer{
		wm:       wm,
		listener: listener,
		sockPath: sockPath,
	}, nil
}

// Start starts the IPC server in a goroutine
func (ipc *IPCServer) Start() {
	go func() {
		for {
			conn, err := ipc.listener.Accept()
			if err != nil {
				// Check if server was closed
				if strings.Contains(err.Error(), "use of closed") {
					return
				}
				log.Printf("IPC accept error: %v", err)
				continue
			}
			go ipc.handleConnection(conn)
		}
	}()
}

// Stop stops the IPC server
func (ipc *IPCServer) Stop() {
	if ipc.listener != nil {
		ipc.listener.Close()
	}
	os.Remove(ipc.sockPath)
}

// handleConnection handles a single IPC connection
func (ipc *IPCServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	cmd := strings.TrimSpace(line)
	response := ipc.handleCommand(cmd)

	// Send JSON response
	jsonResp, _ := json.Marshal(response)
	conn.Write(append(jsonResp, '\n'))
}

// handleCommand processes an IPC command and returns a response
func (ipc *IPCServer) handleCommand(cmd string) IPCResponse {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return IPCResponse{Success: false, Message: "empty command"}
	}

	action := strings.ToLower(parts[0])
	args := parts[1:]

	switch action {
	case "workspace":
		return ipc.cmdWorkspace(args)
	case "window":
		return ipc.cmdWindow(args)
	case "layout":
		return ipc.cmdLayout(args)
	case "query":
		return ipc.cmdQuery(args)
	case "action":
		return ipc.cmdAction(args)
	case "help":
		return ipc.cmdHelp()
	default:
		return IPCResponse{Success: false, Message: fmt.Sprintf("unknown command: %s", action)}
	}
}

// cmdWorkspace handles workspace commands
func (ipc *IPCServer) cmdWorkspace(args []string) IPCResponse {
	if len(args) == 0 {
		return IPCResponse{Success: false, Message: "usage: workspace <switch|move> <n>"}
	}

	switch args[0] {
	case "switch":
		if len(args) < 2 {
			return IPCResponse{Success: false, Message: "usage: workspace switch <n>"}
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 9 {
			return IPCResponse{Success: false, Message: "workspace must be 1-9"}
		}
		ipc.wm.switchToWorkspace(n - 1)
		return IPCResponse{Success: true, Message: fmt.Sprintf("switched to workspace %d", n)}

	case "move":
		if len(args) < 2 {
			return IPCResponse{Success: false, Message: "usage: workspace move <n>"}
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 9 {
			return IPCResponse{Success: false, Message: "workspace must be 1-9"}
		}
		if ipc.wm.focused != nil {
			ipc.wm.moveToWorkspace(ipc.wm.focused, n-1)
			return IPCResponse{Success: true, Message: fmt.Sprintf("moved window to workspace %d", n)}
		}
		return IPCResponse{Success: false, Message: "no focused window"}

	default:
		return IPCResponse{Success: false, Message: fmt.Sprintf("unknown workspace command: %s", args[0])}
	}
}

// cmdWindow handles window commands
func (ipc *IPCServer) cmdWindow(args []string) IPCResponse {
	if len(args) == 0 {
		return IPCResponse{Success: false, Message: "usage: window <close|focus|float|sink>"}
	}

	switch args[0] {
	case "close":
		if ipc.wm.focused != nil {
			ActionKill(ipc.wm)
			return IPCResponse{Success: true, Message: "window closed"}
		}
		return IPCResponse{Success: false, Message: "no focused window"}

	case "focus":
		if len(args) < 2 {
			return IPCResponse{Success: false, Message: "usage: window focus <next|prev|master>"}
		}
		switch args[1] {
		case "next":
			ActionFocusNext(ipc.wm)
		case "prev":
			ActionFocusPrev(ipc.wm)
		case "master":
			ActionFocusMaster(ipc.wm)
		default:
			return IPCResponse{Success: false, Message: fmt.Sprintf("unknown focus direction: %s", args[1])}
		}
		return IPCResponse{Success: true, Message: "focus changed"}

	case "float":
		if ipc.wm.focused != nil {
			ipc.wm.focused.Floating = true
			ipc.wm.tile()
			return IPCResponse{Success: true, Message: "window floating"}
		}
		return IPCResponse{Success: false, Message: "no focused window"}

	case "sink":
		if ipc.wm.focused != nil {
			ipc.wm.focused.Floating = false
			ipc.wm.tile()
			return IPCResponse{Success: true, Message: "window sunk"}
		}
		return IPCResponse{Success: false, Message: "no focused window"}

	case "swap":
		if len(args) < 2 {
			return IPCResponse{Success: false, Message: "usage: window swap <next|prev>"}
		}
		switch args[1] {
		case "next":
			ActionSwapNext(ipc.wm)
		case "prev":
			ActionSwapPrev(ipc.wm)
		default:
			return IPCResponse{Success: false, Message: fmt.Sprintf("unknown swap direction: %s", args[1])}
		}
		return IPCResponse{Success: true, Message: "window swapped"}

	default:
		return IPCResponse{Success: false, Message: fmt.Sprintf("unknown window command: %s", args[0])}
	}
}

// cmdLayout handles layout commands
func (ipc *IPCServer) cmdLayout(args []string) IPCResponse {
	if len(args) == 0 {
		return IPCResponse{Success: false, Message: "usage: layout <next|reset|shrink|expand>"}
	}

	switch args[0] {
	case "next":
		ActionNextLayout(ipc.wm)
		return IPCResponse{Success: true, Message: fmt.Sprintf("layout: %s", ipc.wm.currentWorkspace().Layout.Name())}

	case "reset":
		ActionResetLayout(ipc.wm)
		return IPCResponse{Success: true, Message: "layout reset to tall"}

	case "shrink":
		ActionShrink(ipc.wm)
		return IPCResponse{Success: true, Message: "master shrunk"}

	case "expand":
		ActionExpand(ipc.wm)
		return IPCResponse{Success: true, Message: "master expanded"}

	default:
		return IPCResponse{Success: false, Message: fmt.Sprintf("unknown layout command: %s", args[0])}
	}
}

// cmdQuery handles query commands
func (ipc *IPCServer) cmdQuery(args []string) IPCResponse {
	if len(args) == 0 {
		return IPCResponse{Success: false, Message: "usage: query <workspaces|windows|focused>"}
	}

	switch args[0] {
	case "workspaces":
		var workspaces []WorkspaceInfo
		for _, ws := range ipc.wm.workspaces {
			workspaces = append(workspaces, WorkspaceInfo{
				ID:      ws.ID + 1,
				Name:    ws.Name,
				Current: ws.ID == ipc.wm.current,
				Windows: len(ws.Clients),
			})
		}
		return IPCResponse{Success: true, Data: workspaces}

	case "windows":
		var windows []WindowInfo
		for _, c := range ipc.wm.clients {
			windows = append(windows, WindowInfo{
				ID:        uint32(c.Window),
				Title:     ipc.wm.getWindowTitle(c.Window),
				Class:     ipc.wm.getWMClass(c.Window),
				Workspace: c.Workspace + 1,
				Floating:  c.Floating,
				Focused:   c == ipc.wm.focused,
				Urgent:    c.Urgent,
			})
		}
		return IPCResponse{Success: true, Data: windows}

	case "focused":
		if ipc.wm.focused != nil {
			info := WindowInfo{
				ID:        uint32(ipc.wm.focused.Window),
				Title:     ipc.wm.getWindowTitle(ipc.wm.focused.Window),
				Class:     ipc.wm.getWMClass(ipc.wm.focused.Window),
				Workspace: ipc.wm.focused.Workspace + 1,
				Floating:  ipc.wm.focused.Floating,
				Focused:   true,
				Urgent:    ipc.wm.focused.Urgent,
			}
			return IPCResponse{Success: true, Data: info}
		}
		return IPCResponse{Success: false, Message: "no focused window"}

	case "layout":
		return IPCResponse{Success: true, Data: ipc.wm.currentWorkspace().Layout.Name()}

	default:
		return IPCResponse{Success: false, Message: fmt.Sprintf("unknown query: %s", args[0])}
	}
}

// cmdAction handles generic actions
func (ipc *IPCServer) cmdAction(args []string) IPCResponse {
	if len(args) == 0 {
		return IPCResponse{Success: false, Message: "usage: action <restart|quit|scratchpad>"}
	}

	switch args[0] {
	case "restart":
		ActionRestart(ipc.wm)
		return IPCResponse{Success: true, Message: "restarting"}

	case "quit":
		ActionQuit(ipc.wm)
		return IPCResponse{Success: true, Message: "quitting"}

	case "scratchpad":
		ActionToggleScratchpad(ipc.wm)
		return IPCResponse{Success: true, Message: "scratchpad toggled"}

	default:
		return IPCResponse{Success: false, Message: fmt.Sprintf("unknown action: %s", args[0])}
	}
}

// cmdHelp returns help information
func (ipc *IPCServer) cmdHelp() IPCResponse {
	help := `Available commands:
  workspace switch <1-9>    - Switch to workspace
  workspace move <1-9>      - Move focused window to workspace
  window close              - Close focused window
  window focus <next|prev|master> - Change focus
  window float              - Float focused window
  window sink               - Sink focused window to tiled
  window swap <next|prev>   - Swap focused window
  layout next               - Cycle to next layout
  layout reset              - Reset to tall layout
  layout shrink             - Shrink master area
  layout expand             - Expand master area
  query workspaces          - List all workspaces
  query windows             - List all windows
  query focused             - Get focused window info
  query layout              - Get current layout name
  action restart            - Restart window manager
  action quit               - Quit window manager
  action scratchpad         - Toggle scratchpad
  help                      - Show this help`
	return IPCResponse{Success: true, Message: help}
}
