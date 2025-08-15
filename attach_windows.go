//go:build windows

package attach

import (
	"errors"
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
	return nil, errors.ErrUnsupported
}
