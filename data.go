package attach

import (
	"bytes"
	"strings"
)

// request creates a request byte slice for the attach protocol.
func request(cmd string, args ...string) []byte {
	var request bytes.Buffer
	request.WriteString("1")
	request.WriteByte(0)
	request.WriteString(cmd)
	request.WriteByte(0)
	for i := 0; i < 3; i++ {
		if i < len(args) {
			request.WriteString(args[i])
		}
		request.WriteByte(0)
	}
	return request.Bytes()
}

// properties reads the "properties" command response from the VM.
func properties(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	properties := make(map[string]string, len(lines))

	for _, line := range lines {
		line, _, _ = strings.Cut(line, "#")
		if index := strings.Index(line, "="); index != -1 {
			key := strings.TrimSpace(line[:index])
			value := strings.TrimSpace(line[index+1:])
			if key != "" && value != "" {
				properties[key] = value
			}
		}
	}
	return properties
}
