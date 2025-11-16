//go:build windows

package attach

import "C"
import (
	"fmt"
	"strconv"
	"unsafe"
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

	return &hotSpotVM{c: &windowsConn{pid}}, nil
}

type windowsConn struct {
	pid int
}

func (wc *windowsConn) Close() error {
	return nil
}

func (wc *windowsConn) send(cmd string, args ...string) ([]byte, error) {
	argc := 1 + len(args)
	argv := make([]*C.char, argc)
	argv[0] = C.CString(cmd)
	defer C.free(unsafe.Pointer(argv[0]))

	for i, arg := range args {
		argv[i+1] = C.CString(arg)
		defer C.free(unsafe.Pointer(argv[i+1]))
	}

	var respBuf C.response_buffer_t
	respBuf.capacity = 8192
	respBuf.data = (*C.char)(C.malloc(C.size_t(respBuf.capacity)))
	if respBuf.data == nil {
		return nil, fmt.Errorf("failed to allocate response buffer")
	}
	defer C.free(unsafe.Pointer(respBuf.data))
	respBuf.length = 0

	var result C.attach_result_t
	retCode := C.attach(C.int(wc.pid), C.int(argc), &argv[0], &respBuf, &result)

	if result.success == 0 {
		errMsg := C.GoString(&result.error_msg[0])
		if errMsg == "" {
			errMsg = "unknown error"
		}
		if result.error_code != 0 {
			return nil, fmt.Errorf("attach failed: %s (error code: %d)", errMsg, result.error_code)
		}
		return nil, fmt.Errorf("attach failed: %s", errMsg)
	}

	resp, err := response(C.GoBytes(unsafe.Pointer(respBuf.data), respBuf.length))
	if err != nil {
		return nil, err
	}

	if retCode != 0 {
		return nil, &ErrResponse{Code: int(retCode), Data: string(resp)}
	}
	return resp, nil
}
