package desktop

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestListenLoopbackPersistsAndReusesPort(t *testing.T) {
	portFile := filepath.Join(t.TempDir(), "host.port")
	first, err := ListenLoopback(context.Background(), ListenerOptions{Address: "127.0.0.1:0", PortFile: portFile})
	if err != nil {
		t.Fatal(err)
	}
	firstPort := listenerPort(t, first)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != strconv.Itoa(firstPort) {
		t.Fatalf("persisted port = %q, want %d", got, firstPort)
	}

	second, err := ListenLoopback(context.Background(), ListenerOptions{Address: "127.0.0.1:0", PortFile: portFile})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if got := listenerPort(t, second); got != firstPort {
		t.Fatalf("reused port = %d, want %d", got, firstPort)
	}
}

func TestListenLoopbackDoesNotFallBackWhenSavedPortIsBusy(t *testing.T) {
	var listenConfig net.ListenConfig
	reserved, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer reserved.Close()
	port := listenerPort(t, reserved)
	portFile := filepath.Join(t.TempDir(), "host.port")
	if err := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ListenLoopback(context.Background(), ListenerOptions{Address: "127.0.0.1:0", PortFile: portFile})
	if !errors.Is(err, ErrPortInUse) {
		t.Fatalf("ListenLoopback() error = %v, want ErrPortInUse", err)
	}
	var portErr *PortInUseError
	if !errors.As(err, &portErr) || portErr.Port != port {
		t.Fatalf("port error = %#v, want saved port %d", portErr, port)
	}
}

func TestListenLoopbackRejectsNonLoopbackMissingPolicyAndCanceledContext(t *testing.T) {
	portFile := filepath.Join(t.TempDir(), "port")
	if _, err := ListenLoopback(context.Background(), ListenerOptions{Address: "0.0.0.0:0", PortFile: portFile}); err == nil {
		t.Fatal("non-loopback listener address was accepted")
	}
	if _, err := ListenLoopback(context.Background(), ListenerOptions{Address: "127.0.0.1:0"}); err == nil {
		t.Fatal("missing port file path was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ListenLoopback(ctx, ListenerOptions{Address: "127.0.0.1:0", PortFile: portFile}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ListenLoopback() error = %v, want context.Canceled", err)
	}
}

func listenerPort(t *testing.T, listener net.Listener) int {
	t.Helper()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address has type %T, want *net.TCPAddr", listener.Addr())
	}
	return address.Port
}
