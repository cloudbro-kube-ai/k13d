package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	logger       *log.Logger
	currentLevel Level = LevelInfo // Default level
)

func Init(appName string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	logDir := filepath.Join(configDir, appName, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// System-wide log
	sysLogFile, err := os.OpenFile(filepath.Join(logDir, "k13d.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Local log (optional, but requested for easy dev access)
	localLogFile, err := os.OpenFile("k13d.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Just use system log if local fails
		logger = log.New(sysLogFile, "", log.LstdFlags|log.Lshortfile)
		return nil
	}

	multi := io.MultiWriter(sysLogFile, localLogFile)
	logger = log.New(multi, "", log.LstdFlags|log.Lshortfile)
	return nil
}

// SetLevel sets the current logging level from a string (debug, info, warn, error)
func SetLevel(levelStr string) {
	switch strings.ToLower(levelStr) {
	case "debug":
		currentLevel = LevelDebug
	case "info":
		currentLevel = LevelInfo
	case "warn", "warning":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}
}

func Infof(format string, v ...any) {
	if logger != nil && currentLevel <= LevelInfo {
		_ = logger.Output(2, fmt.Sprintf("[INFO] "+format, v...))
	}
}

func Errorf(format string, v ...any) {
	if logger != nil && currentLevel <= LevelError {
		_ = logger.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
	}
}

func Debugf(format string, v ...any) {
	if logger != nil && currentLevel <= LevelDebug {
		_ = logger.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
	}
}

func Warnf(format string, v ...any) {
	if logger != nil && currentLevel <= LevelWarn {
		_ = logger.Output(2, fmt.Sprintf("[WARN] "+format, v...))
	}
}
