//go:build unix

package attach

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func init() {
	provider = &UnixProvider{}
}

type UnixProvider struct {
	stdProvider
}

func (up *UnixProvider) Attach(desc *VMDescriptor) (VM, error) {
	return up.AttachID(desc.ID)
}

func (up *UnixProvider) AttachID(id string) (VM, error) {
	pid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid PID %s: %v", id, err)
	}

	conn, err := up.connect(pid)
	if err != nil {
		return nil, fmt.Errorf("error attaching to process %d: %v", pid, err)
	}

	return &hotSpotVM{c: &stdConn{conn}}, nil
}

func (up *UnixProvider) attachFilePath(pid int) (string, error) {
	file := fmt.Sprintf(".attach_pid%d", pid)
	if runtime.GOOS == "darwin" {
		// https://github.com/openjdk/jdk/blob/master/src/jdk.attach/macosx/classes/sun/tools/attach/VirtualMachineImpl.java#L214
		return filepath.Join(os.TempDir(), file), nil
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", fmt.Errorf("error getting process %d: %v", pid, err)
	}

	cwd, err := proc.Cwd()
	if err != nil {
		return "", fmt.Errorf("error getting current working directory for process %d: %v", pid, err)
	}

	return filepath.Join(cwd, file), nil
}

func (up *UnixProvider) connect(pid int) (*net.UnixConn, error) {
	attachPath, err := up.attachFilePath(pid)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(attachPath, nil, 0660); err != nil {
		return nil, fmt.Errorf("error creating attach file %s: %w", attachPath, err)
	}

	defer func() {
		_ = os.Remove(attachPath)
	}()

	err = syscall.Kill(pid, syscall.SIGQUIT)
	if err != nil {
		return nil, fmt.Errorf("error sending SIGQUIT to %v: %v", pid, err)
	}

	sockFile := fmt.Sprintf("/proc/%d/root/tmp/.java_pid%d", pid, pid)
	if _, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); errors.Is(err, os.ErrNotExist) {
		// presume procfs is not available, fallback to normal temp directory
		sockFile = filepath.Join(os.TempDir(), fmt.Sprintf(".java_pid%d", pid))
	}

	for i := 0; i < 10; i++ {
		var conn *net.UnixConn
		conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: sockFile, Net: "unix"})
		if err != nil {
			time.Sleep(time.Duration(1<<uint(i)) * time.Millisecond)
			continue
		}

		return conn, nil
	}
	return nil, err
}

type stdConn struct {
	net.Conn
}

func (sc *stdConn) send(cmd string, args ...string) ([]byte, error) {
	data := request(cmd, args...)
	if _, err := sc.Conn.Write(data); err != nil {
		return nil, fmt.Errorf("error writing to socket: %v", err)
	}

	resp, err := io.ReadAll(sc.Conn)
	if err != nil {
		return nil, fmt.Errorf("error reading from socket: %v", err)
	}

	return response(resp)
}
