//go:build windows

package agent

// daemonize is a no-op on Windows — Windows services are handled differently.
// On Windows, the agent runs as a console application by default.
// For production, use: sc create WorkerAgent binPath= "C:\path\worker-agent.exe"
func daemonize() {
	// No-op on Windows
}
