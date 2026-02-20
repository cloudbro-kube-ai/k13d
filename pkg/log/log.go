package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var logger *log.Logger

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

func Infof(format string, v ...any) {
	if logger != nil {
		_ = logger.Output(2, fmt.Sprintf("[INFO] "+format, v...))
	}
}

func Errorf(format string, v ...any) {
	if logger != nil {
		_ = logger.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
	}
}

func Debugf(format string, v ...any) {
	if logger != nil {
		_ = logger.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
	}
}

func Warnf(format string, v ...any) {
	if logger != nil {
		_ = logger.Output(2, fmt.Sprintf("[WARN] "+format, v...))
	}
}
