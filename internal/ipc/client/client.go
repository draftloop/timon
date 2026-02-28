package ipcclient

import (
	"encoding/gob"
	"fmt"
	"net"
	"timon/internal/config"
	"timon/internal/ipc"
	"timon/internal/ipc/dto"
)

func Connect() (net.Conn, error) {
	return net.Dial("unix", config.DefaultSocketPath)
}

func Send[Req, Resp any](conn net.Conn, req Req) (*Resp, error) {
	payload, err := ipc.Wrap(req)
	if err != nil {
		return nil, fmt.Errorf("request error — encode request: %w", err)
	}

	env := ipc.Envelope{MessageType: ipc.MsgType(req), Payload: payload}
	if err := gob.NewEncoder(conn).Encode(env); err != nil {
		return nil, fmt.Errorf("request error — send envelope: %w", err)
	}

	var resp ipc.EnvelopeResponse
	if err := gob.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("request error — receive response: %w", err)
	}

	var errResp dto.ErrorResponse
	if err := ipc.Unwrap(resp.Payload, &errResp); err == nil && errResp.Message != "" {
		return nil, fmt.Errorf("daemon error: %s", errResp.Message)
	}

	var result Resp
	if err := ipc.Unwrap(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("response error — decode response: %w", err)
	}
	return &result, nil
}
