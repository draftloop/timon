package ipcdaemon

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
	"timon/internal/ipc"
	handlerUtils "timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/daemon/handlers"
	"timon/internal/ipc/dto"
	"timon/internal/log"
)

type handlerFunc func([]byte) ([]byte, error, error)

var registry = map[string]handlerFunc{}

func register(types ...any) {
	for _, t := range types {
		gob.Register(t)
	}
}

func trimStrings(v any) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		if field.Kind() == reflect.String && field.CanSet() {
			field.SetString(strings.TrimSpace(field.String()))
		}
	}
}

func addHandler[Req, Resp any](handlerFn func(Req) handlerUtils.Response[Resp], reqType any, resType any) {
	var zero Req
	registry[ipc.MsgType(zero)] = func(data []byte) ([]byte, error, error) {
		var req Req
		if err := ipc.Unwrap(data, &req); err != nil {
			return nil, fmt.Errorf("malformed request: %w", err), nil
		}
		trimStrings(&req)
		resp := handlerFn(req)
		if resp.ClientError != nil || resp.DaemonError != nil {
			return nil, resp.ClientError, resp.DaemonError
		} else if resp.Data == nil {
			return nil, nil, fmt.Errorf("handler response is missing")
		}
		bytes, err := ipc.Wrap(resp.Data)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to wrap the response")
		}
		return bytes, resp.ClientError, resp.DaemonError
	}
	register(reqType, resType)
}

func dispatch(conn net.Conn) error {
	defer conn.Close()

	var env ipc.Envelope
	if err := gob.NewDecoder(conn).Decode(&env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}

	var payload []byte
	start := time.Now()

	var errClient, errDaemon error
	handler, handlerExists := registry[env.MessageType]
	if !handlerExists {
		errClient = fmt.Errorf("unknown request")
		payload, _ = ipc.Wrap(dto.ErrorResponse{
			Message: errClient.Error(),
		})
	} else {
		payload, errClient, errDaemon = handler(env.Payload)
		if errDaemon != nil {
			payload, _ = ipc.Wrap(dto.ErrorResponse{Message: "internal error"})
		} else if errClient != nil {
			payload, _ = ipc.Wrap(dto.ErrorResponse{Message: errClient.Error()})
		}
	}

	duration := time.Since(start)

	if errDaemon != nil {
		_ = log.IPC.Errorf(`%s FAIL (%s) "%s"`, strings.TrimSuffix(env.MessageType, "Request"), duration, errDaemon)
	} else if errClient != nil {
		log.IPC.Warnf(`%s INVALID (%s) "%s"`, strings.TrimSuffix(env.MessageType, "Request"), duration, errClient)
	} else {
		log.IPC.Debugf(`%s OK (%s)`, strings.TrimSuffix(env.MessageType, "Request"), duration)
	}

	return gob.NewEncoder(conn).Encode(ipc.EnvelopeResponse{Payload: payload})
}

func CreateServer(socketPath string, onReady func()) error {
	register(dto.ErrorResponse{})
	addHandler(handlers.PushIncidentHandler, dto.PushIncidentRequest{}, dto.PushIncidentResponse{})
	addHandler(handlers.PushProbeHandler, dto.PushProbeRequest{}, dto.PushProbeResponse{})
	addHandler(handlers.PushJobStartHandler, dto.PushJobStartRequest{}, dto.PushJobStartResponse{})
	addHandler(handlers.PushJobStepHandler, dto.PushJobStepRequest{}, dto.PushJobStepResponse{})
	addHandler(handlers.PushJobEndHandler, dto.PushJobEndRequest{}, dto.PushJobEndResponse{})
	addHandler(handlers.ShowHandler, dto.ShowRequest{}, dto.ShowResponse{})
	addHandler(handlers.DeleteHandler, dto.DeleteRequest{}, dto.DeleteResponse{})
	addHandler(handlers.AnnotateHandler, dto.AnnotateRequest{}, dto.AnnotateResponse{})
	addHandler(handlers.ResolveHandler, dto.ResolveRequest{}, dto.ResolveResponse{})
	addHandler(handlers.SummaryHandler, dto.SummaryRequest{}, dto.SummaryResponse{})
	addHandler(handlers.StatusHandler, dto.StatusRequest{}, dto.StatusResponse{})
	addHandler(handlers.TruncateHandler, dto.TruncateRequest{}, dto.TruncateResponse{})

	// Ensure socket directory exists
	if err := os.MkdirAll(filepath.Dir(socketPath), 0750); err != nil {
		return fmt.Errorf("mkdir: %v", err)
	}

	socketLockPath := socketPath + ".lock"
	lockFile, err := os.OpenFile(socketLockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock: %v", err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("already running (flock: %v)", err)
	}

	// Remove stale socket if it exists
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	defer listener.Close()

	// Restrict socket permissions to owner only
	if err := os.Chmod(socketPath, 0600); err != nil {
		return fmt.Errorf("chmod: %v", err)
	}

	// Clean up on exit
	defer os.Remove(socketPath)
	defer os.Remove(socketLockPath)

	log.Daemon.Infof("daemon started on unix socket %s", socketPath)
	if onReady != nil {
		onReady()
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			_ = log.Daemon.Error(err.Error())
			continue
		}

		go func(conn net.Conn) {
			err := dispatch(conn)
			if err != nil {
				_ = log.Daemon.Error(err.Error())
			}
		}(conn)
	}
}
