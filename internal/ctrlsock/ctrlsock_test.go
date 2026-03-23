package ctrlsock

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if testEnvDir := os.Getenv("GIV_TEST_ENV_DIR"); testEnvDir != "" {
		if err := os.Chdir(testEnvDir); err != nil {
			panic("failed to change to test environment directory: " + err.Error())
		}
	}

	m.Run()
}

func TestListenAndSend(t *testing.T) {
	l, err := Listen("test-container", false)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	if l.Port() == 0 {
		t.Fatal("expected non-zero port")
	}

	addr := fmt.Sprintf("127.0.0.1:%d", l.Port())

	// Send a message (unknown command — just verify no crash)
	if err := Send(addr, "HELLO world"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Give the goroutine a moment to process
	time.Sleep(50 * time.Millisecond)
}

func TestListenAndClose(t *testing.T) {
	l, err := Listen("test-container", false)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Sending after close should fail
	addr := fmt.Sprintf("127.0.0.1:%d", l.Port())
	if err := Send(addr, "HELLO"); err == nil {
		t.Fatal("expected error when sending to closed listener")
	}
}

func TestSendToMissing(t *testing.T) {
	err := Send("127.0.0.1:1", "HELLO")
	if err == nil {
		t.Fatal("expected error when sending to unreachable address")
	}
}

func TestRewriteURL_OrbStack(t *testing.T) {
	l := &Listener{containerName: "giverny-my-task", orbstack: true}
	got := l.rewriteURL("http://localhost:8000")
	if got != "http://giverny-my-task.orb.local" {
		t.Fatalf("expected OrbStack URL, got: %s", got)
	}
}

func TestRewriteURL_RegularDocker(t *testing.T) {
	l := &Listener{containerName: "giverny-my-task", orbstack: false}
	got := l.rewriteURL("http://localhost:8000")
	if got != "http://localhost:8000" {
		t.Fatalf("expected unchanged URL, got: %s", got)
	}
}

func TestContainerAddr(t *testing.T) {
	t.Setenv(EnvVar, "host.docker.internal:12345")
	a := ContainerAddr()
	if a != "host.docker.internal:12345" {
		t.Fatalf("unexpected addr: %s", a)
	}
}

func TestContainerAddrEmpty(t *testing.T) {
	t.Setenv(EnvVar, "")
	a := ContainerAddr()
	if a != "" {
		t.Fatalf("expected empty addr, got: %s", a)
	}
}
