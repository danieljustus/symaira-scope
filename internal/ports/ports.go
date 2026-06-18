// Package ports inventories local listening sockets and suggests free ports.
// Cross-platform via gopsutil — no shelling out to lsof/netstat.
package ports

import (
	"fmt"
	"net"
	"sort"
	"syscall"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// ListListening returns listening TCP sockets and bound UDP sockets, with the
// owning process where the OS exposes it (some PIDs require elevated rights).
func ListListening() ([]model.Port, error) {
	conns, err := psnet.Connections("inet")
	if err != nil {
		return nil, fmt.Errorf("enumerate sockets: %w", err)
	}

	names := map[int32]string{}
	seen := map[string]bool{}
	var out []model.Port
	for _, c := range conns {
		isUDP := c.Type == uint32(syscall.SOCK_DGRAM)
		if !isUDP && c.Status != "LISTEN" {
			continue
		}

		// Collapse identical IPv4/IPv6 entries for the same socket.
		key := fmt.Sprintf("%d/%d/%d/%s", c.Laddr.Port, c.Type, c.Pid, c.Laddr.IP)
		if seen[key] {
			continue
		}
		seen[key] = true

		name := ""
		if c.Pid != 0 {
			if cached, ok := names[c.Pid]; ok {
				name = cached
			} else if p, e := process.NewProcess(c.Pid); e == nil {
				name, _ = p.Name()
				names[c.Pid] = name
			}
		}

		proto := "tcp"
		if isUDP {
			proto = "udp"
		}
		out = append(out, model.Port{
			Port:     int(c.Laddr.Port),
			Protocol: proto,
			Address:  c.Laddr.IP,
			PID:      int(c.Pid),
			Process:  name,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].Protocol < out[j].Protocol
	})
	return out, nil
}

// SuggestFree returns up to count TCP ports in [from, to] that can be bound now.
func SuggestFree(count, from, to int) []int {
	if count <= 0 {
		count = 1
	}
	var free []int
	for p := from; p <= to && len(free) < count; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			_ = l.Close()
			free = append(free, p)
		}
	}
	return free
}

// Conflicts reports TCP ports listened on by more than one distinct process.
// PID 0 (process not exposed to us) is ignored to avoid false positives.
func Conflicts(ports []model.Port) []model.Conflict {
	byPort := map[int]map[int]string{}
	for _, p := range ports {
		if p.Protocol != "tcp" || p.PID == 0 {
			continue
		}
		if byPort[p.Port] == nil {
			byPort[p.Port] = map[int]string{}
		}
		byPort[p.Port][p.PID] = p.Process
	}

	var out []model.Conflict
	for port, holders := range byPort {
		if len(holders) < 2 {
			continue
		}
		hs := make([]string, 0, len(holders))
		for pid, name := range holders {
			hs = append(hs, fmt.Sprintf("%s(pid %d)", name, pid))
		}
		sort.Strings(hs)
		out = append(out, model.Conflict{Port: port, Holders: hs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out
}
