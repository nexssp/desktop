package desktop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// ErrPortInUse identifies a persisted listener port that could not be rebound.
var ErrPortInUse = errors.New("persisted loopback port is unavailable")

// ListenerOptions describes a loopback listener whose selected port is persisted
// in a caller-owned file. Address must contain a literal loopback IP and port;
// use port 0 to request an ephemeral port on first launch.
type ListenerOptions struct {
	Address  string
	PortFile string
}

// PortInUseError reports a saved port that could not be rebound.
type PortInUseError struct {
	Port int
	Err  error
}

func (e *PortInUseError) Error() string {
	return fmt.Sprintf("%v: %d: %v", ErrPortInUse, e.Port, e.Err)
}

// Unwrap preserves the underlying socket error and ErrPortInUse classification.
func (e *PortInUseError) Unwrap() []error { return []error{ErrPortInUse, e.Err} }

// ListenLoopback listens only on an explicit loopback IP address. If PortFile
// contains a valid previously selected port, it is reused; failure to bind that
// saved port is returned instead of silently selecting another port.
func ListenLoopback(ctx context.Context, options ListenerOptions) (net.Listener, error) {
	if ctx == nil {
		return nil, errors.New("listener context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(options.PortFile) == "" {
		return nil, errors.New("port file path is required")
	}
	host, configuredPort, err := net.SplitHostPort(options.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid listener address: %w", err)
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil || !ip.IsLoopback() {
		return nil, errors.New("listener address must use a literal loopback IP")
	}
	port, err := strconv.Atoi(configuredPort)
	if err != nil || port < 0 || port > 65535 {
		return nil, errors.New("listener port must be between 0 and 65535")
	}

	var listenConfig net.ListenConfig
	if saved, readErr := os.ReadFile(options.PortFile); readErr == nil {
		if savedPort, parseErr := strconv.Atoi(strings.TrimSpace(string(saved))); parseErr == nil && savedPort > 0 && savedPort <= 65535 {
			address := net.JoinHostPort(ip.String(), strconv.Itoa(savedPort))
			listener, listenErr := listenConfig.Listen(ctx, "tcp", address)
			if listenErr != nil {
				return nil, &PortInUseError{Port: savedPort, Err: listenErr}
			}
			return listener, nil
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return nil, fmt.Errorf("read persisted port: %w", readErr)
	}

	listener, err := listenConfig.Listen(ctx, "tcp", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("listen on loopback address: %w", err)
	}
	tcpAddress, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, errors.New("TCP listener returned a non-TCP address")
	}
	if err := os.WriteFile(options.PortFile, []byte(strconv.Itoa(tcpAddress.Port)+"\n"), 0o600); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("persist selected port: %w", err)
	}
	return listener, nil
}
