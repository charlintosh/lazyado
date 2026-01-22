package debug

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var logger *log.Logger
var enabled bool

func Enable() {
	enabled = true
	logPath := filepath.Join(os.TempDir(), "lazyado-debug.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		logger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags)
		return
	}
	logger = log.New(file, "[DEBUG] ", log.LstdFlags)
	logger.Printf("Debug log started - %s\n", logPath)
}

type Logger struct{ Scope string }

func Scope(scope string) *Logger { return &Logger{Scope: scope} }

func (l *Logger) Debugf(format string, args ...interface{}) {
	if !enabled || logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logger.Printf("%s: %s", l.Scope, msg)
}

func (l *Logger) Debug(format string, args ...interface{}) { l.Debugf(format, args...) }
