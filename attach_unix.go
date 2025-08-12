//go:build unix

package attach

import (
	"fmt"
	"maps"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() {
	provider = &UnixProvider{}
}

type UnixProvider struct {
}

// listPids returns a list of process IDs found in the hsperfdata directories.
// The PIDs may or may not exist or may not be from Java processes.
func (up *UnixProvider) listPids() ([]int, error) {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return nil, err
	}

	pids := map[int]struct{}{}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "hsperfdata_") || !entry.IsDir() {
			continue
		}

		subEntries, err := os.ReadDir(filepath.Join(os.TempDir(), entry.Name()))
		if err != nil {
			continue
		}

		for _, subEntry := range subEntries {
			if pid, err := strconv.Atoi(subEntry.Name()); err == nil {
				pids[pid] = struct{}{}
			}
		}
	}

	return slices.Collect(maps.Keys(pids)), nil
}

func (up *UnixProvider) List() ([]*VMDescriptor, error) {
	pids, err := up.listPids()
	if err != nil {
		return nil, fmt.Errorf("error listing PIDs: %v", err)
	}

	var descs []*VMDescriptor
	for _, pid := range pids {
		if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil {
			descs = append(descs, &VMDescriptor{
				ID:          strconv.Itoa(pid),
				DisplayName: strings.ReplaceAll(string(data), "\u0000", " "),
				Provider:    up,
			})
		}
	}

	return descs, nil
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

	return &stdVM{conn}, nil
}

func (up *UnixProvider) connect(pid int) (*net.UnixConn, error) {
	if _, err := os.Stat(fmt.Sprintf("/proc/%d/status", pid)); err != nil {
		return nil, fmt.Errorf("error stating process status %d: %v", pid, err)
	}

	attachFile := fmt.Sprintf("/proc/%d/cwd/.attach_pid%d", pid, pid)
	if err := os.WriteFile(attachFile, nil, 0660); err != nil {
		return nil, fmt.Errorf("error creating file %s: %w", attachFile, err)
	}

	defer func() {
		_ = os.Remove(attachFile)
	}()

	err := syscall.Kill(pid, syscall.SIGQUIT)
	if err != nil {
		return nil, fmt.Errorf("error sending SIGQUIT to %v: %v", pid, err)
	}

	sockFile := fmt.Sprintf("/proc/%d/root/tmp/.java_pid%d", pid, pid)
	for i := 0; i < 10; i++ {
		var conn *net.UnixConn
		conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: sockFile, Net: "unix"})
		if err != nil {
			time.Sleep(time.Duration(1<<uint(i)) * time.Millisecond)
			continue
		}

		return conn, nil
	}
	return nil, fmt.Errorf("error attaching to process %d: %v", pid, err)
}
