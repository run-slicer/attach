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

func TestWindowsProvider_AttachID_ProcessNotFound(t *testing.T) {
	provider := &WindowsProvider{}
	
	// Test with a PID that doesn't exist
	_, err := provider.AttachID("99999")
	if err == nil {
		t.Fatal("Expected error for non-existent PID")
	}
	
	// Should get an error about not being able to attach to the process
	t.Logf("Expected error for non-existent PID: %v", err)
}

func TestWindowsProvider_GeneratePipeName(t *testing.T) {
	provider := &WindowsProvider{}
	
	// Test pipe name generation
	name1, err := provider.generatePipeName()
	if err != nil {
		t.Fatalf("generatePipeName failed: %v", err)
	}
	
	name2, err := provider.generatePipeName()
	if err != nil {
		t.Fatalf("generatePipeName failed: %v", err)
	}
	
	if name1 == name2 {
		t.Fatal("generatePipeName should generate unique names")
	}
	
	if !strings.HasPrefix(name1, "javatool") {
		t.Fatalf("Pipe name should start with 'javatool', got: %s", name1)
	}
	
	t.Logf("Generated pipe names: %s, %s", name1, name2)
}