//go:build windows

package attach

import (
	"strings"
	"testing"
)

func TestWindowsProvider_New(t *testing.T) {
	provider := &WindowsProvider{}
	
	// Test that attachFilePath works correctly
	path, err := provider.attachFilePath(1234)
	if err != nil {
		t.Fatalf("attachFilePath failed: %v", err)
	}
	
	if path == "" {
		t.Fatal("attachFilePath returned empty path")
	}
	
	t.Logf("Attach file path for PID 1234: %s", path)
}

func TestWindowsProvider_AttachID_Invalid(t *testing.T) {
	provider := &WindowsProvider{}
	
	// Test with invalid PID
	_, err := provider.AttachID("invalid")
	if err == nil {
		t.Fatal("Expected error for invalid PID")
	}
	
	t.Logf("Expected error for invalid PID: %v", err)
}

func TestWindowsProvider_AttachID_ProcessInjectionNotImplemented(t *testing.T) {
	provider := &WindowsProvider{}
	
	// Test that attach correctly returns error about process injection
	_, err := provider.AttachID("1234")
	if err == nil {
		t.Fatal("Expected error indicating process injection not implemented")
	}
	
	// Verify the error mentions the complexity of Windows attach
	if !strings.Contains(err.Error(), "process injection") {
		t.Fatalf("Expected error to mention process injection, got: %v", err)
	}
	
	t.Logf("Expected error about Windows complexity: %v", err)
}