// Package gormflag provides a [flag.Value] for GORM log levels.
//
//spellchecker:words gormflag
package gormflag

//spellchecker:words errors strings gorm logger Logger
import (
	"errors"
	"fmt"
	"strings"

	gormLogger "gorm.io/gorm/logger"
)

// LogLevel is a GORM [gormLogger.LogLevel] that implements [flag.Value].
type LogLevel gormLogger.LogLevel

// DefaultLogLevel is the default GORM log level used by CLI flags.
const DefaultLogLevel = LogLevel(gormLogger.Silent)

var errInvalidLogLevel = errors.New("invalid gorm log level")

// Level returns the underlying GORM log level.
func (l *LogLevel) Level() gormLogger.LogLevel {
	return gormLogger.LogLevel(*l)
}

func (l *LogLevel) String() string {
	switch gormLogger.LogLevel(*l) {
	case gormLogger.Silent:
		return "silent"
	case gormLogger.Error:
		return "error"
	case gormLogger.Warn:
		return "warn"
	case gormLogger.Info:
		return "info"
	default:
		return fmt.Sprintf("LogLevel(%d)", int(*l))
	}
}

func (l *LogLevel) Set(value string) error {
	switch strings.ToLower(value) {
	case "silent":
		*l = LogLevel(gormLogger.Silent)
	case "error":
		*l = LogLevel(gormLogger.Error)
	case "warn", "warning":
		*l = LogLevel(gormLogger.Warn)
	case "info":
		*l = LogLevel(gormLogger.Info)
	default:
		return fmt.Errorf("%w %q: (expected silent|error|warn|info)", errInvalidLogLevel, value)
	}
	return nil
}

// FlagUsage is the usage string for a GORM log-level flag.
const FlagUsage = "GORM log level: silent, error, warn, info"
