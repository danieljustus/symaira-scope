//go:build windows

package mcphealth

import "os/exec"

func setProcAttr(c *exec.Cmd) {}

func killProcess(c *exec.Cmd) {
	if c.Process != nil {
		c.Process.Kill()
	}
	c.Wait()
}
