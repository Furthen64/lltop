package runner

import (
	"os"
	"syscall"
)

func sendInterrupt(process *os.Process) error {
	return process.Signal(syscall.SIGINT)
}

func sendKill(process *os.Process) error {
	return process.Kill()
}
