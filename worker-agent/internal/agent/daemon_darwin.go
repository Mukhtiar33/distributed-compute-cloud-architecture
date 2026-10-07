//go:build darwin

package agent

import (
	"os"
	"syscall"
)

// daemonize forks the process into the background on macOS.
func daemonize() {
	if os.Getenv("WORKER_AGENT_DAEMON") == "1" {
		return
	}

	ret, _, errno := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		return
	}

	if ret > 0 {
		os.Exit(0)
	}

	syscall.Setsid()
	os.Setenv("WORKER_AGENT_DAEMON", "1")

	devNull, _ := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if devNull != nil {
		syscall.Dup2(int(devNull.Fd()), int(os.Stdin.Fd()))
		syscall.Dup2(int(devNull.Fd()), int(os.Stdout.Fd()))
		syscall.Dup2(int(devNull.Fd()), int(os.Stderr.Fd()))
		devNull.Close()
	}
}
