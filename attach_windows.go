//go:build windows

package attach

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/windows"
)

func init() {
	provider = &WindowsProvider{}
}

type WindowsProvider struct {
	stdProvider
}

func (wp *WindowsProvider) Attach(desc *VMDescriptor) (VM, error) {
	return wp.AttachID(desc.ID)
}

func (wp *WindowsProvider) AttachID(id string) (VM, error) {
	pid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid PID %s: %v", id, err)
	}

	conn, err := wp.connect(pid)
	if err != nil {
		return nil, fmt.Errorf("error attaching to process %d: %v", pid, err)
	}

	return &stdVM{conn}, nil
}

func (wp *WindowsProvider) attachFilePath(pid int) (string, error) {
	file := fmt.Sprintf(".attach_pid%d", pid)
	return filepath.Join(os.TempDir(), file), nil
}

func (wp *WindowsProvider) connect(pid int) (net.Conn, error) {
	attachPath, err := wp.attachFilePath(pid)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(attachPath, nil, 0660); err != nil {
		return nil, fmt.Errorf("error creating attach file %s: %w", attachPath, err)
	}

	defer func() {
		_ = os.Remove(attachPath)
	}()

	// Signal the JVM process using Windows event mechanism
	err = wp.signalProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("error signaling process %v: %v", pid, err)
	}

	// Create named pipe path
	pipeName := fmt.Sprintf(`\\.\pipe\.java_pid%d`, pid)

	// Try to connect to the named pipe with retries
	for i := 0; i < 10; i++ {
		conn, err := wp.connectToPipe(pipeName)
		if err != nil {
			time.Sleep(time.Duration(1<<uint(i)) * time.Millisecond)
			continue
		}
		return conn, nil
	}
	return nil, err
}

func (wp *WindowsProvider) signalProcess(pid int) error {
	// Open the process with minimal required access
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("error opening process %d: %v", pid, err)
	}
	defer windows.CloseHandle(handle)

	// For Windows, we need to trigger the attach listener in the JVM
	// The JVM on Windows listens for attach requests by checking for attach files
	// and doesn't require a signal like Unix systems
	// The attach file creation is sufficient to trigger the attach mechanism
	return nil
}

func (wp *WindowsProvider) connectToPipe(pipeName string) (net.Conn, error) {
	// Convert Go string to UTF16 for Windows API
	pipeNameUTF16, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return nil, err
	}

	// Try to open the named pipe
	handle, err := windows.CreateFile(
		pipeNameUTF16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, err
	}

	// Wrap the Windows handle in a Go net.Conn interface
	return &windowsPipeConn{handle: handle}, nil
}

// windowsPipeConn implements net.Conn for Windows named pipes
type windowsPipeConn struct {
	handle windows.Handle
}

func (c *windowsPipeConn) Read(b []byte) (n int, err error) {
	var bytesRead uint32
	err = windows.ReadFile(c.handle, b, &bytesRead, nil)
	return int(bytesRead), err
}

func (c *windowsPipeConn) Write(b []byte) (n int, err error) {
	var bytesWritten uint32
	err = windows.WriteFile(c.handle, b, &bytesWritten, nil)
	return int(bytesWritten), err
}

func (c *windowsPipeConn) Close() error {
	return windows.CloseHandle(c.handle)
}

func (c *windowsPipeConn) LocalAddr() net.Addr {
	return &windowsPipeAddr{name: "local"}
}

func (c *windowsPipeConn) RemoteAddr() net.Addr {
	return &windowsPipeAddr{name: "remote"}
}

func (c *windowsPipeConn) SetDeadline(t time.Time) error {
	// Named pipes don't support deadlines in the same way as sockets
	return nil
}

func (c *windowsPipeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *windowsPipeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// windowsPipeAddr implements net.Addr for Windows named pipes
type windowsPipeAddr struct {
	name string
}

func (a *windowsPipeAddr) Network() string {
	return "pipe"
}

func (a *windowsPipeAddr) String() string {
	return a.name
}
