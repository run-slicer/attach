package attach

import "testing"

func TestProvider_List(t *testing.T) {
	provider, err := Default()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	descs, err := provider.List()
	if err != nil {
		t.Fatalf("failed to list VM descriptors: %v", err)
	}

	t.Logf("found %d VM descriptor(s)", len(descs))
	for _, desc := range descs {
		t.Logf("ID: %s", desc.ID)
		t.Logf("display name: %s", desc.DisplayName)
	}
}

func TestProvider_Attach(t *testing.T) {
	provider, err := Default()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	descs, err := provider.List()
	if err != nil {
		t.Fatalf("failed to list VM descriptors: %v", err)
	}

	if len(descs) == 0 {
		t.Skip("no VM descriptors found, skipping attach test")
	}

	vm, err := provider.Attach(descs[0])
	if err != nil {
		t.Fatalf("failed to attach to VM: %v", err)
	}

	_ = vm.Close()
}

func TestVM_Properties(t *testing.T) {
	provider, err := Default()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	descs, err := provider.List()
	if err != nil {
		t.Fatalf("failed to list VM descriptors: %v", err)
	}

	if len(descs) == 0 {
		t.Skip("no VM descriptors found, skipping properties test")
	}

	vm, err := provider.Attach(descs[0])
	if err != nil {
		t.Fatalf("failed to attach to VM: %v", err)
	}

	defer func() {
		_ = vm.Close()
	}()

	properties, err := vm.Properties()
	if err != nil {
		t.Fatalf("failed to get VM properties: %v", err)
	}

	t.Logf("VM properties: %s", properties)
}

func TestVM_ThreadDump(t *testing.T) {
	provider, err := Default()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	descs, err := provider.List()
	if err != nil {
		t.Fatalf("failed to list VM descriptors: %v", err)
	}

	if len(descs) == 0 {
		t.Skip("no VM descriptors found, skipping thread dump test")
	}

	vm, err := provider.Attach(descs[0])
	if err != nil {
		t.Fatalf("failed to attach to VM: %v", err)
	}

	defer func() {
		_ = vm.Close()
	}()

	threadDump, err := vm.ThreadDump()
	if err != nil {
		t.Fatalf("failed to get thread dump: %v", err)
	}

	t.Logf("Thread dump:\n%s", threadDump)
}

func TestVM_Load(t *testing.T) {
	provider, err := Default()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	descs, err := provider.List()
	if err != nil {
		t.Fatalf("failed to list VM descriptors: %v", err)
	}

	if len(descs) == 0 {
		t.Skip("no VM descriptors found, skipping load test")
	}

	vm, err := provider.Attach(descs[0])
	if err != nil {
		t.Fatalf("failed to attach to VM: %v", err)
	}

	defer func() {
		_ = vm.Close()
	}()

	err = vm.Load("./test/jolokia-agent-jvm-javaagent.jar", "")
	if err != nil {
		t.Fatalf("failed to load agent: %v", err)
	}

	t.Logf("Agent loaded successfully")
}
