package main

// Workspace represents a virtual desktop with its own set of windows and layout
type Workspace struct {
	ID      int
	Name    string
	Clients []*Client
	Layout  Layout
	Focused *Client
}

// NewWorkspace creates a new workspace with the given ID and name
func NewWorkspace(id int, name string) *Workspace {
	return &Workspace{
		ID:      id,
		Name:    name,
		Clients: make([]*Client, 0),
		Layout:  NewTallLayout(),
	}
}

// Add adds a client to this workspace
func (ws *Workspace) Add(c *Client) {
	ws.Clients = append(ws.Clients, c)
	c.Workspace = ws.ID
}

// Remove removes a client from this workspace
func (ws *Workspace) Remove(c *Client) {
	for i, client := range ws.Clients {
		if client == c {
			ws.Clients = append(ws.Clients[:i], ws.Clients[i+1:]...)
			if ws.Focused == c {
				ws.Focused = nil
				// Focus next available client
				if len(ws.Clients) > 0 {
					if i < len(ws.Clients) {
						ws.Focused = ws.Clients[i]
					} else {
						ws.Focused = ws.Clients[len(ws.Clients)-1]
					}
				}
			}
			break
		}
	}
}

// TiledClients returns only the non-floating clients
func (ws *Workspace) TiledClients() []*Client {
	var tiled []*Client
	for _, c := range ws.Clients {
		if !c.Floating {
			tiled = append(tiled, c)
		}
	}
	return tiled
}

// FocusNext focuses the next client in the list
func (ws *Workspace) FocusNext() *Client {
	if len(ws.Clients) == 0 {
		return nil
	}

	if ws.Focused == nil {
		ws.Focused = ws.Clients[0]
		return ws.Focused
	}

	for i, c := range ws.Clients {
		if c == ws.Focused {
			ws.Focused = ws.Clients[(i+1)%len(ws.Clients)]
			return ws.Focused
		}
	}

	ws.Focused = ws.Clients[0]
	return ws.Focused
}

// FocusPrev focuses the previous client in the list
func (ws *Workspace) FocusPrev() *Client {
	if len(ws.Clients) == 0 {
		return nil
	}

	if ws.Focused == nil {
		ws.Focused = ws.Clients[len(ws.Clients)-1]
		return ws.Focused
	}

	for i, c := range ws.Clients {
		if c == ws.Focused {
			idx := i - 1
			if idx < 0 {
				idx = len(ws.Clients) - 1
			}
			ws.Focused = ws.Clients[idx]
			return ws.Focused
		}
	}

	ws.Focused = ws.Clients[0]
	return ws.Focused
}

// SwapNext swaps the focused client with the next one
func (ws *Workspace) SwapNext() {
	if len(ws.Clients) < 2 || ws.Focused == nil {
		return
	}

	for i, c := range ws.Clients {
		if c == ws.Focused {
			next := (i + 1) % len(ws.Clients)
			ws.Clients[i], ws.Clients[next] = ws.Clients[next], ws.Clients[i]
			return
		}
	}
}

// SwapPrev swaps the focused client with the previous one
func (ws *Workspace) SwapPrev() {
	if len(ws.Clients) < 2 || ws.Focused == nil {
		return
	}

	for i, c := range ws.Clients {
		if c == ws.Focused {
			prev := i - 1
			if prev < 0 {
				prev = len(ws.Clients) - 1
			}
			ws.Clients[i], ws.Clients[prev] = ws.Clients[prev], ws.Clients[i]
			return
		}
	}
}

// FocusMaster focuses the first (master) client
func (ws *Workspace) FocusMaster() *Client {
	if len(ws.Clients) == 0 {
		return nil
	}
	ws.Focused = ws.Clients[0]
	return ws.Focused
}

// SwapMaster swaps the focused client with the master
func (ws *Workspace) SwapMaster() {
	if len(ws.Clients) < 2 || ws.Focused == nil {
		return
	}

	for i, c := range ws.Clients {
		if c == ws.Focused && i != 0 {
			ws.Clients[0], ws.Clients[i] = ws.Clients[i], ws.Clients[0]
			return
		}
	}
}

// SetLayout sets the workspace layout
func (ws *Workspace) SetLayout(layout Layout) {
	ws.Layout = layout
}

// NextLayout cycles to the next layout in the given list
func (ws *Workspace) NextLayout(layouts []Layout) {
	currentName := ws.Layout.Name()
	for i, l := range layouts {
		if l.Name() == currentName {
			ws.Layout = layouts[(i+1)%len(layouts)]
			return
		}
	}
	// Default to first layout if current not found
	if len(layouts) > 0 {
		ws.Layout = layouts[0]
	}
}
