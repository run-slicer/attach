//go:build windows

package attach

import (
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