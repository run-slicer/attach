package attach

import (
	"errors"
	"fmt"
	"io"
	"net"
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
	// Provider is the provider that created this descriptor.
	Provider Provider
}

// VM represents an attached JVM instance.
type VM interface {
	io.Closer

	// Load loads a Java agent from the specified path into the attached JVM.
	Load(path string, options ...string) (string, error)
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

type stdVM struct {
	conn net.Conn
}

func (vm *stdVM) Load(path string, options ...string) (string, error) {
	resp, err := vm.send("load", append([]string{path}, options...)...)
	if err != nil {
		return "", err
	}

	return string(resp), nil
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

	return resp, nil
}

func (vm *stdVM) Close() error {
	return vm.conn.Close()
}
