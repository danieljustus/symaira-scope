// Package model holds the data types symscope produces. All fields use
// snake_case JSON tags for agent-friendly output.
package model

// Snapshot is the full inventory produced by `scan`.
type Snapshot struct {
	GeneratedAt string      `json:"generated_at"`
	Ports       []Port      `json:"ports"`
	MCPServers  []MCPServer `json:"mcp_servers"`
	Containers  []Container `json:"containers"`
	Notes       []string    `json:"notes,omitempty"`
}

// Port is a local listening socket.
type Port struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp | udp
	Address  string `json:"address"`
	PID      int    `json:"pid"`
	Process  string `json:"process"`
}

// MCPServer is one MCP server discovered in an AI client's configuration.
type MCPServer struct {
	Name       string   `json:"name"`
	Client     string   `json:"client"`    // claude-desktop, cursor, vscode, ...
	Transport  string   `json:"transport"` // stdio | http | sse
	Command    string   `json:"command,omitempty"`
	Args       []string `json:"args,omitempty"`
	URL        string   `json:"url,omitempty"`
	ConfigPath string   `json:"config_path"`
}

// Container is a running container and its published ports.
type Container struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	Ports []int  `json:"ports"`
}

// Conflict is a port bound by more than one process.
type Conflict struct {
	Port    int      `json:"port"`
	Holders []string `json:"holders"`
}

// ClientConfig reports whether a known AI client's MCP config file is present.
type ClientConfig struct {
	Client  string `json:"client"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}
