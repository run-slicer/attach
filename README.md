# attach

A Go library for attaching to running JVMs via the Attach API.

Currently, HotSpot-based JVMs are supported on Unix-like systems (Linux, macOS). Windows support is partially implemented but requires complex process injection which is not yet available in pure Go.

## Platform Support

- ✅ **Linux**: Full support via Unix domain sockets
- ✅ **macOS**: Full support via Unix domain sockets  
- ⚠️ **Windows**: Partial implementation - Windows JVM attach requires process injection with assembly code generation, which is complex to implement in pure Go. For reference implementations, see OpenJDK's `VirtualMachineImpl.c` and the jattach project.

## Windows Implementation Notes

Windows JVM attach requires:
1. Creating named pipes for communication
2. Injecting assembly code into the target JVM process
3. Calling `JVM_EnqueueOperation` from within the target process
4. Complex memory management and security handling

This is significantly more complex than the Unix implementation which uses simple domain sockets and signals.
