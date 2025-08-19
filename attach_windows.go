//go:build windows

package attach

/*
#include "attach_windows.c"
*/
import "C"

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"unsafe"

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
	// Generate a unique pipe name
	pipeName, err := wp.generatePipeName()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pipe name: %w", err)
	}

	// Create attach file to signal the JVM (this part is correct)
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

	// Create the named pipe
	pipeNameC := C.CString(pipeName)
	defer C.free(unsafe.Pointer(pipeNameC))

	hPipe := C.create_attach_pipe(pipeNameC)
	if hPipe == C.HANDLE(windows.InvalidHandle) {
		return nil, fmt.Errorf("failed to create named pipe: %v", windows.GetLastError())
	}

	// Inject into the target process to make it connect to our pipe
	pidC := C.DWORD(pid)
	arg0C := C.CString("JCMD")
	arg1C := C.CString("")
	arg2C := C.CString("")
	arg3C := C.CString("")
	defer C.free(unsafe.Pointer(arg0C))
	defer C.free(unsafe.Pointer(arg1C))
	defer C.free(unsafe.Pointer(arg2C))
	defer C.free(unsafe.Pointer(arg3C))

	result := C.attach_to_jvm(pidC, pipeNameC, arg0C, arg1C, arg2C, arg3C)
	if result != 0 {
		C.CloseHandle(hPipe)
		return nil, fmt.Errorf("failed to inject into target process: error code %d", result)
	}

	// Wait for the JVM to connect to our pipe
	err = wp.waitForConnection(windows.Handle(hPipe))
	if err != nil {
		C.CloseHandle(hPipe)
		return nil, fmt.Errorf("failed to wait for JVM connection: %w", err)
	}

	return &windowsPipeConn{handle: windows.Handle(hPipe)}, nil
}

func (wp *WindowsProvider) generatePipeName() (string, error) {
	// Generate a random pipe name similar to how OpenJDK does it
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("javatool%x", buf), nil
}

func (wp *WindowsProvider) waitForConnection(hPipe windows.Handle) error {
	// Wait for client to connect to the pipe
	err := windows.ConnectNamedPipe(hPipe, nil)
	if err != nil && err != windows.ERROR_PIPE_CONNECTED {
		return fmt.Errorf("ConnectNamedPipe failed: %w", err)
	}
	return nil
}

// windowsPipeConn implements net.Conn for Windows named pipes
type windowsPipeConn struct {
	handle windows.Handle
}

func (c *windowsPipeConn) Read(b []byte) (n int, err error) {
	var bytesRead uint32
	err = windows.ReadFile(c.handle, b, &bytesRead, nil)
	if err == windows.ERROR_BROKEN_PIPE {
		return int(bytesRead), fmt.Errorf("pipe broken")
	}
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
