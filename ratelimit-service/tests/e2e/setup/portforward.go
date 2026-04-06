package setup

import (
	"fmt"
	"os/exec"
	"time"
)

type PortForward struct {
	cmd        *exec.Cmd
	localPort  string
	remotePort string
	resource   string
	namespace  string
}

func NewPortForward(namespace, resource, localPort, remotePort string) *PortForward {
	return &PortForward{
		namespace:  namespace,
		resource:   resource,
		localPort:  localPort,
		remotePort: remotePort,
	}
}

func (pf *PortForward) Start() error {
	args := []string{
		"port-forward",
		"-n", pf.namespace,
		pf.resource,
		fmt.Sprintf("%s:%s", pf.localPort, pf.remotePort),
	}

	pf.cmd = exec.Command("kubectl", args...)

	if err := pf.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start port-forward: %w", err)
	}

	time.Sleep(2 * time.Second)
	return nil
}

func (pf *PortForward) Stop() {
	if pf.cmd != nil && pf.cmd.Process != nil {
		pf.cmd.Process.Kill()
	}
}

func (pf *PortForward) URL() string {
	return fmt.Sprintf("http://localhost:%s", pf.localPort)
}
