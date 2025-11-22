package attach

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type hotSpotVM struct {
	c conn
}

func (vm *hotSpotVM) Load(agent string, options string) error {
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

func (vm *hotSpotVM) LoadLibrary(path string, absolute bool, options string) error {
	args := []string{path, strconv.FormatBool(absolute)}
	if options != "" {
		args = append(args, options)
	}

	resp, err := vm.c.send("load", args...)
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

func (vm *hotSpotVM) Properties() (map[string]string, error) {
	resp, err := vm.c.send("properties")
	if err != nil {
		return nil, err
	}

	return properties(resp), nil
}

func (vm *hotSpotVM) ThreadDump() (string, error) {
	resp, err := vm.c.send("threaddump")
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (vm *hotSpotVM) Close() error {
	return vm.c.Close()
}
