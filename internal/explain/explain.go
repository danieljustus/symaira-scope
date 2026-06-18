package explain

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/ports"
)

type PortExplanation struct {
	Port       int              `json:"port"`
	Listening  *model.Port      `json:"listening,omitempty"`
	Processes  []ProcessInfo    `json:"processes,omitempty"`
	MCPServers []MCPServerInfo  `json:"mcp_servers,omitempty"`
	Conflicts  []model.Conflict `json:"conflicts,omitempty"`
	Suggested  bool             `json:"suggested_free_port"`
}

type ProcessInfo struct {
	PID    int    `json:"pid"`
	Name   string `json:"name"`
	Action string `json:"action"`
}

type MCPServerInfo struct {
	Name      string `json:"name"`
	Client    string `json:"client"`
	Transport string `json:"transport"`
	URL       string `json:"url,omitempty"`
	Command   string `json:"command,omitempty"`
	ConfigPath string `json:"config_path"`
}

type ServerExplanation struct {
	Name       string         `json:"name"`
	Client     string         `json:"client"`
	Server     model.MCPServer `json:"server"`
	Occupied   *model.Port    `json:"occupied_port,omitempty"`
	Conflict   *model.Conflict `json:"conflict,omitempty"`
}

func ExplainPort(portNum int) (*PortExplanation, error) {
	listening, err := ports.ListListening()
	if err != nil {
		return nil, fmt.Errorf("list ports: %w", err)
	}

	exp := &PortExplanation{Port: portNum}

	for _, p := range listening {
		if p.Port == portNum {
			exp.Listening = &p
			exp.Processes = append(exp.Processes, ProcessInfo{
				PID:    p.PID,
				Name:   p.Process,
				Action: "listening",
			})
		}
	}

	servers := mcpcfg.Discover(mcpcfg.DefaultSources())
	for _, s := range servers {
		if s.URL == "" {
			continue
		}
		u, err := url.Parse(s.URL)
		if err != nil {
			continue
		}
		p, _ := strconv.Atoi(u.Port())
		if p == portNum {
			exp.MCPServers = append(exp.MCPServers, MCPServerInfo{
				Name:       s.Name,
				Client:     s.Client,
				Transport:  s.Transport,
				URL:        s.URL,
				Command:    s.Command,
				ConfigPath: s.ConfigPath,
			})
		}
	}

	allConflicts := ports.Conflicts(listening)
	mcpConflicts := ports.MCPServerConflicts(servers, listening)
	allConflicts = append(allConflicts, mcpConflicts...)
	for _, c := range allConflicts {
		if c.Port == portNum {
			exp.Conflicts = append(exp.Conflicts, c)
		}
	}

	if exp.Listening == nil && len(exp.MCPServers) == 0 {
		exp.Suggested = true
	}

	return exp, nil
}

func ExplainServer(name string) (*ServerExplanation, error) {
	servers := mcpcfg.Discover(mcpcfg.DefaultSources())

	var found *model.MCPServer
	for _, s := range servers {
		if s.Name == name {
			found = &s
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("server %q not found in any AI client config", name)
	}

	exp := &ServerExplanation{
		Name:   found.Name,
		Client: found.Client,
		Server: *found,
	}

	if found.URL != "" {
		u, err := url.Parse(found.URL)
		if err == nil {
			port := 0
			if u.Port() != "" {
				port, _ = strconv.Atoi(u.Port())
			}
			if port > 0 {
				listening, err := ports.ListListening()
				if err == nil {
					for _, p := range listening {
						if p.Port == port {
							occ := p
							exp.Occupied = &occ
							break
						}
					}
				}
			}
		}
	}

	return exp, nil
}
