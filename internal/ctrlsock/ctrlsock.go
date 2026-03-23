// Package ctrlsock implements a TCP-based control channel between
// outie (host) and innie (container). The protocol is line-based: each
// message is a single line terminated by '\n'.
//
// TCP is used instead of Unix sockets because Unix sockets don't work
// across the Docker VM boundary on macOS.
package ctrlsock

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// EnvVar is the environment variable that holds the control server address
// (host:port). Both the Go code in innie and the diffreviewer wrapper script
// read this.
const EnvVar = "GIVERNY_CTRL_SOCK"

// ContainerAddr returns the control server address from the environment,
// or empty string if not set.
func ContainerAddr() string {
	return os.Getenv(EnvVar)
}

// Listener listens on a TCP port and dispatches incoming messages.
type Listener struct {
	ln            net.Listener
	port          int
	containerName string // Docker container name, used for OrbStack URLs
	orbstack      bool   // true if running under OrbStack
	done          chan struct{}
	debug         bool
}

// Listen binds a TCP port on localhost (OS-allocated) and starts a goroutine
// that accepts connections and handles messages. containerName is the Docker
// container name, used to construct OrbStack URLs. The returned Listener must
// be closed with Close().
func Listen(containerName string, debug bool) (*Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	l := &Listener{
		ln:            ln,
		port:          port,
		containerName: containerName,
		orbstack:      isOrbStack(),
		done:          make(chan struct{}),
		debug:         debug,
	}

	go l.accept()
	return l, nil
}

// Port returns the TCP port the listener is bound to.
func (l *Listener) Port() int {
	return l.port
}

// Close stops the listener and closes the port.
func (l *Listener) Close() error {
	l.ln.Close()
	<-l.done
	return nil
}

func (l *Listener) accept() {
	defer close(l.done)
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			return // listener closed
		}
		go l.handleConn(conn)
	}
}

func (l *Listener) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if l.debug {
			fmt.Fprintf(os.Stderr, "[ctrlsock] received: %s\n", line)
		}
		l.dispatch(line)
	}
}

func (l *Listener) dispatch(msg string) {
	parts := strings.SplitN(msg, " ", 2)
	cmd := parts[0]

	switch cmd {
	case "OPEN-DIFFR":
		url := "http://localhost:8000"
		if len(parts) > 1 && parts[1] != "" {
			url = parts[1]
		}
		url = l.rewriteURL(url)
		if l.debug {
			fmt.Fprintf(os.Stderr, "[ctrlsock] opening browser: %s\n", url)
		}
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open browser for %s: %v\n", url, err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Warning: unknown control message: %s\n", msg)
	}
}

// rewriteURL converts a container-local URL to a host-accessible URL.
// For OrbStack: http://container-name.orb.local (port auto-detected by OrbStack)
// For regular Docker: the URL is returned unchanged (requires -p port mapping).
func (l *Listener) rewriteURL(containerURL string) string {
	if l.orbstack && l.containerName != "" {
		return fmt.Sprintf("http://%s.orb.local", l.containerName)
	}
	return containerURL
}

// isOrbStack returns true if the Docker runtime is OrbStack.
func isOrbStack() bool {
	_, err := exec.LookPath("orb")
	return err == nil
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// Send connects to the control server at the given address and sends a message.
// This is used by innie to send messages to outie.
func Send(addr, msg string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to control server: %w", err)
	}
	defer conn.Close()

	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	_, err = conn.Write([]byte(msg))
	return err
}
