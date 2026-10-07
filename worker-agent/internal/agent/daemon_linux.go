//go:build linux

package agent

import (
	"os"
	"syscall"
)

// daemonize forks the process into the background on Linux.
func daemonize() {
	// Check if we're already the child process
	if os.Getenv("WORKER_AGENT_DAEMON") == "1" {
		return // Already daemonized
	}

	// Fork: parent exits, child continues
	ret, _, errno := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		// Fork failed — just run in foreground
		return
	}

	if ret > 0 {
		// Parent process — exit
		os.Exit(0)
	}

	// Child process — create new session
	syscall.Setsid()

	// Set environment variable so we don't fork again
	os.Setenv("WORKER_AGENT_DAEMON", "1")

	// Redirect stdin/stdout/stderr to /null
	devNull, _ := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if devNull != nil {
		syscall.Dup2(int(devNull.Fd()), int(os.Stdin.Fd()))
		syscall.Dup2(int(devNull.Fd()), int(os.Stdout.Fd()))
		syscall.Dup2(int(devNull.Fd()), int(os.Stderr.Fd()))
		devNull.Close()
	}
}
