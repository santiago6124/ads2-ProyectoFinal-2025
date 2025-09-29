package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"portfolio-api/internal/config"
)

// Init initializes the logger based on configuration
func Init(cfg config.LoggerConfig) {
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
}

// getFileWriter returns a file writer with rotation
func getFileWriter(cfg config.LoggerConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
	}
}