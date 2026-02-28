package ipc

import (
	"bytes"
	"encoding/gob"
	"reflect"
)

type Envelope struct {
	MessageType string
	Payload     []byte
}

type EnvelopeResponse struct {
	Payload []byte
}

func Wrap(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unwrap(data []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}

func MsgType(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}
