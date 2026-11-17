package debug

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

var logger *log.Logger
var enabled bool

func Enable() {
	enabled = true
	logPath, err := LogPath()
	if err == nil {
		err = os.MkdirAll(filepath.Dir(logPath), 0o700)
	}
	var file *os.File
	if err == nil {
		_ = os.Remove(logPath)
		file, err = os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	}
	if err != nil {
		logger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags)
		return
	}
	logger = log.New(file, "[DEBUG] ", log.LstdFlags)
	logger.Printf("Debug log started - %s\n", logPath)
}

func LogPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazyado", "debug.log"), nil
}

func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<unparseable url>"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
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
