package attach

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type ErrParse struct {
	Data string
}

func (ep *ErrParse) Error() string {
	return fmt.Sprintf("invalid response from attach protocol: %s", ep.Data)
}

type ErrResponse struct {
	Code int
	Data string
}

func (er *ErrResponse) Error() string {
	return fmt.Sprintf("non-zero response code %d from attach protocol, response: %s", er.Code, er.Data)
}

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

// response processes the response from the attach protocol.
func response(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var builder strings.Builder
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			break
		}
		builder.WriteByte(data[i])
	}

	code, err := strconv.Atoi(builder.String())
	if err != nil {
		return nil, &ErrParse{string(data)}
	}

	length := builder.Len() + 1
	if code != 0 {
		return nil, &ErrResponse{code, string(data[length:])}
	}

	return data[length:], nil
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
