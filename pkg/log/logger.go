package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance for app logs
var Logger *logrus.Logger

// AccessLogger is the global logger instance for gin access logs
var AccessLogger *logrus.Logger

// Fields is a type alias for logrus.Fields
type Fields = logrus.Fields

func init() {
	// Initial default logger (stdout only)
	Logger = logrus.New()
	setupDefaultLogger(Logger)

	AccessLogger = logrus.New()
	setupDefaultLogger(AccessLogger)
}

func setupDefaultLogger(l *logrus.Logger) {
	l.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", path.Base(f.Function)), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})
	l.SetOutput(os.Stdout)
	l.SetReportCaller(true)
	l.SetLevel(logrus.InfoLevel)
}

// InitLoggers initializes app and access loggers with file rotation and dual output
func InitLoggers(cfg *config.LoggingConfig) {
	// 1. Setup App Logger (show caller info)
	setupLogger(Logger, cfg.Level, &cfg.AppLog, true, true)

	// 2. Setup Access Logger (hide caller info for cleaner access logs)
	setupLogger(AccessLogger, cfg.Level, &cfg.AccessLog, true, false)
}

func setupLogger(l *logrus.Logger, levelStr string, fileCfg *config.LogFileConfig, showOnStdout bool, reportCaller bool) {
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel
	}
	l.SetLevel(level)
	l.SetReportCaller(reportCaller)

	// Formatter for file (no color)
	fileFormatter := &logrus.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", path.Base(f.Function)), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	}

	// Formatter for stdout (with color)
	stdoutFormatter := &logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", path.Base(f.Function)), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	}

	if fileCfg.Filename != "" {
		// Ensure log directory exists
		logDir := path.Dir(fileCfg.Filename)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create log directory %s: %v\n", logDir, err)
		}

		// Setup lumberjack for rotation
		jack := &lumberjack.Logger{
			Filename:   fileCfg.Filename,
			MaxSize:    fileCfg.MaxSize,
			MaxBackups: fileCfg.MaxBackups,
			MaxAge:     fileCfg.MaxAge,
			Compress:   fileCfg.Compress,
		}

		if showOnStdout {
			// If we need both, we use a hook or multi-writer.
			// Logrus doesn't support different formatters for different writers in a single logger easily.
			// We'll use a custom hook to write plain text to file.
			l.SetFormatter(stdoutFormatter)
			l.SetOutput(os.Stdout)
			l.AddHook(&FileHook{
				Writer:    jack,
				Formatter: fileFormatter,
				LogLevels: logrus.AllLevels,
			})
		} else {
			l.SetFormatter(fileFormatter)
			l.SetOutput(jack)
		}
	} else {
		l.SetFormatter(stdoutFormatter)
		l.SetOutput(os.Stdout)
	}
}

// FileHook is a logrus hook for writing to a file with a different formatter
type FileHook struct {
	Writer    io.Writer
	Formatter logrus.Formatter
	LogLevels []logrus.Level
}

func (h *FileHook) Levels() []logrus.Level {
	return h.LogLevels
}

func (h *FileHook) Fire(entry *logrus.Entry) error {
	line, err := h.Formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.Writer.Write(line)
	return err
}

// SetLevel sets the logger level for the global Logger
func SetLevel(levelStr string) {
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		Logger.Warnf("Invalid log level '%s', defaulting to Info", levelStr)
		level = logrus.InfoLevel
	}
	Logger.SetLevel(level)
}

// GinMiddleware returns a gin handler to log requests using the custom access logger
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Create log entry with standard fields using AccessLogger
		entry := AccessLogger.WithFields(logrus.Fields{
			"status":  statusCode,
			"latency": latency,
			"ip":      clientIP,
			"method":  method,
			"path":    path,
		})

		if raw != "" {
			entry = entry.WithField("query", raw)
		}

		if len(c.Errors) > 0 {
			entry.Error(errorMessage)
		} else {
			// Select log level based on status code
			// Use empty message as fields contain all info
			if statusCode >= 500 {
				entry.Error("")
			} else if statusCode >= 400 {
				entry.Warn("")
			} else {
				entry.Info("")
			}
		}
	}
}
