package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"wallet-api/internal/config"
)

// Init initializes the logger based on configuration
func Init(cfg config.LoggingConfig) {
	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
			},
		})
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z",
		})
	default:
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z",
		})
	}

	// Set output
	switch cfg.Output {
	case "file":
		if cfg.Filename != "" {
			logrus.SetOutput(getFileWriter(cfg))
		} else {
			logrus.SetOutput(os.Stdout)
		}
	case "both":
		if cfg.Filename != "" {
			multiWriter := io.MultiWriter(os.Stdout, getFileWriter(cfg))
			logrus.SetOutput(multiWriter)
		} else {
			logrus.SetOutput(os.Stdout)
		}
	default:
		logrus.SetOutput(os.Stdout)
	}

	// Add hooks if needed
	if cfg.EnableAudit {
		// TODO: Add audit hook
		logrus.Info("Audit logging enabled")
	}
}

// getFileWriter returns a file writer with rotation
func getFileWriter(cfg config.LoggingConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
	}
}

// AuditLogger creates a specialized logger for audit events
func AuditLogger(cfg config.LoggingConfig) *logrus.Logger {
	auditLogger := logrus.New()

	// Always use JSON format for audit logs
	auditLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Set audit log output
	if cfg.EnableAudit && cfg.AuditFile != "" {
		auditLogger.SetOutput(&lumberjack.Logger{
			Filename:   cfg.AuditFile,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge * 2, // Keep audit logs longer
			MaxBackups: cfg.MaxBackups * 2,
			Compress:   cfg.Compress,
		})
	} else {
		auditLogger.SetOutput(os.Stdout)
	}

	auditLogger.SetLevel(logrus.InfoLevel)

	return auditLogger
}