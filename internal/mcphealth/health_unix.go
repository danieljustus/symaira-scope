//go:build !windows

package mcphealth

import (
	"os/exec"
	"syscall"
)

func setProcAttr(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcess(c *exec.Cmd) {
	if c.Process != nil {
		syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
	}
	c.Wait()
}
