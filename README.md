# attach

A Go library for attaching to running JVMs via the Attach API.

HotSpot-based JVMs are supported on Unix-like systems (Linux, macOS) and Windows.

## Platform Support

- ✅ **Linux**: Full support via Unix domain sockets
- ✅ **macOS**: Full support via Unix domain sockets  
- ✅ **Windows**: Full support via named pipes and process injection (requires CGO)

## Windows Implementation Notes

The Windows implementation uses process injection to call `JVM_EnqueueOperation` in the target JVM process, similar to how OpenJDK and other JVM attach tools work. This requires:

1. Creating named pipes for communication
2. Injecting executable code into the target JVM process
3. Calling `JVM_EnqueueOperation` from within the target process
4. Handling security and privilege requirements

The implementation is based on reference code from OpenJDK and the byte-buddy project.
