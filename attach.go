package attach

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

// ErrNoProvider is an error returned when no provider is available, typically when the platform is not supported.
var ErrNoProvider = errors.New("no provider available")

var provider Provider = nil

// Default returns the provider for the current platform.
func Default() (Provider, error) {
	if provider == nil {
		return nil, ErrNoProvider
	}

	return provider, nil
}

// VMDescriptor is a descriptor for a JVM that can be attached to.
type VMDescriptor struct {
	// ID is the unique identifier for the JVM instance, usually the PID.
	ID string
	// DisplayName is a human-readable name for the JVM instance, may be empty, usually the process command line.
	DisplayName string
}

// VM represents an attached JVM instance.
type VM interface {
	io.Closer

	// Load loads a Java agent into the attached JVM.
	Load(agent string, options string) error
	// LoadLibrary loads a Java agent library from the specified path into the attached JVM.
	LoadLibrary(path string, absolute bool, options string) error
	// Properties retrieves the properties of the attached JVM.
	Properties() (map[string]string, error)
	// ThreadDump retrieves a thread dump of the attached JVM.
	ThreadDump() (string, error)
}

// Provider allows you to list and attach to JVMs.
// You can use the Default() function to get the current platform's provider.
type Provider interface {
	// List returns a list of available JVMs that can be attached to.
	List() ([]*VMDescriptor, error)
	// Attach attaches to a JVM using its descriptor.
	Attach(desc *VMDescriptor) (VM, error)
	// AttachID attaches to a JVM using its ID (usually the PID).
	AttachID(id string) (VM, error)
}

type stdProvider struct {
}

// listPids returns a list of process IDs found in the hsperfdata directories.
// The PIDs may or may not exist or may not be from Java processes.
func (sp stdProvider) listPids() ([]int, error) {
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

func (sp stdProvider) List() ([]*VMDescriptor, error) {
	pids, err := sp.listPids()
	if err != nil {
		return nil, fmt.Errorf("error listing PIDs: %v", err)
	}

	var descs []*VMDescriptor
	for _, pid := range pids {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			// exited?
			continue
		}

		desc := &VMDescriptor{ID: strconv.Itoa(pid)}
		descs = append(descs, desc)

		if cmdline, err := proc.Cmdline(); err == nil {
			desc.DisplayName = cmdline
		}
	}

	return descs, nil
}

type stdVM struct {
	conn net.Conn
}

const (
	jniENoMem            = -4
	attachErrorBadJar    = 100
	attachErrorNotOnCP   = 101
	attachErrorStartFail = 102
)

type ErrLoad struct {
	Code    int
	Message string
}

func (el *ErrLoad) Error() string {
	if el.Message == "" {
		return fmt.Sprintf("failed to load agent, non-zero code %d", el.Code)
	}

	return fmt.Sprintf("failed to load agent: %s", el.Message)
}

type ErrAgentLoad struct {
	*ErrLoad
}

func (eal *ErrAgentLoad) Error() string {
	if eal.Message == "" {
		switch eal.Code {
		case jniENoMem:
			return "insufficient memory"
		case attachErrorBadJar:
			return "agent JAR not found or no Agent-Class attribute"
		case attachErrorNotOnCP:
			return "unable to add JAR file to system class path"
		case attachErrorStartFail:
			return "agent JAR loaded but agent failed to initialize"
		}
	}

	return eal.ErrLoad.Error()
}

func (eal *ErrAgentLoad) Unwrap() error {
	return eal.ErrLoad
}

func (vm *stdVM) Load(agent string, options string) error {
	absAgent, err := filepath.Abs(agent)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for agent %s: %v", agent, err)
	}

	args := absAgent
	if options != "" {
		args = args + "=" + options
	}

	err = vm.LoadLibrary("instrument", false, args)
	if err != nil {
		var el *ErrLoad
		if errors.As(err, &el) {
			return &ErrAgentLoad{el}
		}
		return err
	}

	return nil
}

const retCodePrefix = "return code: "

func (vm *stdVM) LoadLibrary(path string, absolute bool, options string) error {
	args := []string{path, strconv.FormatBool(absolute)}
	if options != "" {
		args = append(args, options)
	}

	resp, err := vm.send("load", args...)
	if err != nil {
		return err
	}

	str := strings.TrimSpace(string(resp))
	if strings.HasPrefix(str, retCodePrefix) {
		code, err := strconv.Atoi(strings.TrimPrefix(str, retCodePrefix))
		if err != nil {
			return &ErrParse{str}
		}
		if code != 0 {
			return &ErrLoad{Code: code}
		}
	} else {
		return &ErrLoad{Message: str}
	}

	return nil
}

func (vm *stdVM) Properties() (map[string]string, error) {
	resp, err := vm.send("properties")
	if err != nil {
		return nil, err
	}

	return properties(resp), nil
}

func (vm *stdVM) ThreadDump() (string, error) {
	resp, err := vm.send("threaddump")
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (vm *stdVM) send(cmd string, args ...string) ([]byte, error) {
	data := request(cmd, args...)
	if _, err := vm.conn.Write(data); err != nil {
		return nil, fmt.Errorf("error writing to socket: %v", err)
	}

	resp, err := io.ReadAll(vm.conn)
	if err != nil {
		return nil, fmt.Errorf("error reading from socket: %v", err)
	}

	return response(resp)
}

func (vm *stdVM) Close() error {
	return vm.conn.Close()
}
