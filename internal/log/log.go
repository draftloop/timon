package log

import (
	"fmt"
	"io"
	golog "log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Level int

const (
	LevelSilent Level = iota
	LevelFatal
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

func (l Level) String() string {
	switch l {
	case LevelFatal:
		return "FATAL"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	default:
		return ""
	}
}

const colorReset = "\033[0m"

func (l Level) color() string {
	switch l {
	case LevelFatal:
		return "\033[1;38;2;235;47;6m"
	case LevelError:
		return "\033[38;2;229;80;57m"
	case LevelWarn:
		return "\033[38;2;246;185;59m"
	case LevelInfo:
		return "\033[38;2;74;105;189m"
	case LevelDebug:
		return "\033[38;2;120;224;143m"
	default:
		return colorReset
	}
}

type Logger struct {
	source string
}

var CurrentLevel = LevelSilent
var currentLogPath = ""
var multiWriter io.Writer = os.Stdout

func SetLevel(l Level, logPath string) {
	CurrentLevel = l
	currentLogPath = logPath
	if currentLogPath != "" {
		if err := os.MkdirAll(filepath.Dir(currentLogPath), 0640); err != nil {
			golog.Fatalf("error creating file folder: %v", err)
		}

		file, err := os.OpenFile(currentLogPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			golog.Fatalf("error creating file: %v", err)
		}

		multiWriter = io.MultiWriter(os.Stdout, file)
	} else {
		multiWriter = os.Stdout
	}
}

func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "silent":
		return LevelSilent, nil
	case "fatal":
		return LevelFatal, nil
	case "error":
		return LevelError, nil
	case "warn":
		return LevelWarn, nil
	case "info":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	default:
		return 0, fmt.Errorf("unknown level %q, must be one of: silent, fatal, error, warn, info, debug", s)
	}
}

func (lg *Logger) log(l Level, msg string) {
	if CurrentLevel < l {
		return
	}
	if lg.source != "" {
		msg = fmt.Sprintf("[%s] %s", lg.source, msg)
	}
	_, _ = fmt.Fprintf(multiWriter, "%s %s[%s]%s %s\n",
		time.Now().Format(time.RFC3339),
		l.color(),
		l,
		colorReset,
		msg,
	)
}

func (lg *Logger) Debug(msg string) { lg.log(LevelDebug, msg) }
func (lg *Logger) Debugf(format string, args ...any) {
	lg.log(LevelDebug, fmt.Sprintf(format, args...))
}
func (lg *Logger) Info(msg string) { lg.log(LevelInfo, msg) }
func (lg *Logger) Infof(format string, args ...any) {
	lg.log(LevelInfo, fmt.Sprintf(format, args...))
}
func (lg *Logger) Warn(msg string) { lg.log(LevelWarn, msg) }
func (lg *Logger) Warnf(format string, args ...any) {
	lg.log(LevelWarn, fmt.Sprintf(format, args...))
}
func (lg *Logger) Error(msg string) error {
	lastError := fmt.Errorf(msg)
	lg.log(LevelError, msg)
	return lastError
}
func (lg *Logger) Errorf(format string, args ...any) error {
	lastError := fmt.Errorf(format, args...)
	lg.log(LevelError, lastError.Error())
	return lastError
}
func (lg *Logger) Fatal(msg string) {
	lg.log(LevelFatal, msg)
	os.Exit(1)
}
func (lg *Logger) Fatalf(format string, args ...any) {
	lg.log(LevelFatal, fmt.Errorf(format, args...).Error())
	os.Exit(1)
}

var (
	Client = Logger{}
	Daemon = Logger{}
	SQL    = Logger{"SQL"}
	IPC    = Logger{"IPC"}
	Cron   = Logger{"Cron"}
)
