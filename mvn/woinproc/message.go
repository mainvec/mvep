//https://github.com/leandro-lugaresi/hub
package woinproc

import (
	"fmt"
	"sort"
	"strings"
)

type (
	// Fields is a [key]value storage for Messages values.
	Fields map[string]interface{}

	// Message represent some message/event passed into the hub
	// It also contain some helper functions to convert the fields to primitive types.
	Message struct {
		Name   string
		Body   []byte
		Fields Fields
		respCh chan []byte
	}
)

// Topic return the message topic used when the message was sended.
func (m *Message) Topic() string {
	return m.Name
}

func (m *Message) GetSubject() string {
	return m.Topic()
}

func (m *Message) GetData() []byte {
	return m.Body
}

func (m *Message) Respond(data []byte) error {
	if m.respCh != nil {
		m.respCh <- data
		close(m.respCh)
	}
	return nil
}

func (f Fields) String() string {
	if len(f) == 0 {
		return "Fields(<empty>)"
	}

	fields := make([]string, 0, len(f))
	for k, v := range f {
		fields = append(fields, fmt.Sprintf("[%s]%v", k, v))
	}

	sort.Strings(fields)

	return "Fields( " + strings.Join(fields, ", ") + " )"
}
