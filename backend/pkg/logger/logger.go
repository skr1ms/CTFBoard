package logger

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var timeFormatOnce sync.Once

func initTimeFormat() {
	timeFormatOnce.Do(func() {
		zerolog.TimeFieldFormat = time.RFC3339
	})
}

type zerologLogger struct {
	zl zerolog.Logger
}

func New(opts *Options) Logger {
	var output io.Writer

	switch opts.Output {
	case ConsoleOutput:
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	case FileOutput:
		output = &lumberjack.Logger{
			Filename:   opts.FileOptions.Filename,
			MaxSize:    opts.FileOptions.MaxSize,
			MaxBackups: opts.FileOptions.MaxBackups,
			MaxAge:     opts.FileOptions.MaxAge,
			Compress:   opts.FileOptions.Compress,
		}
	case BothOutput:
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		fileWriter := &lumberjack.Logger{
			Filename:   opts.FileOptions.Filename,
			MaxSize:    opts.FileOptions.MaxSize,
			MaxBackups: opts.FileOptions.MaxBackups,
			MaxAge:     opts.FileOptions.MaxAge,
			Compress:   opts.FileOptions.Compress,
		}
		output = zerolog.MultiLevelWriter(consoleWriter, fileWriter)
	default:
		output = os.Stdout
	}

	initTimeFormat()

	zl := zerolog.New(output).With().Timestamp().Caller().Logger()

	zl = zl.Level(convertLogLevel(opts.Level))

	return &zerologLogger{zl: zl}
}

func (l *zerologLogger) Debug(msg string, fields ...Fields) {
	l.log(l.zl.Debug(), msg, fields...)
}

func (l *zerologLogger) Info(msg string, fields ...Fields) {
	l.log(l.zl.Info(), msg, fields...)
}

func (l *zerologLogger) Warn(msg string, fields ...Fields) {
	l.log(l.zl.Warn(), msg, fields...)
}

func (l *zerologLogger) Error(msg string, fields ...Fields) {
	l.log(l.zl.Error(), msg, fields...)
}

func (l *zerologLogger) Fatal(msg string, fields ...Fields) {
	l.log(l.zl.Fatal(), msg, fields...)
}

func (l *zerologLogger) WithFields(fields Fields) Logger {
	return &zerologLogger{zl: l.zl.With().Fields(fields).Logger()}
}

func (l *zerologLogger) WithError(err error) Logger {
	return &zerologLogger{zl: l.zl.With().Err(err).Logger()}
}

func (l *zerologLogger) log(event *zerolog.Event, msg string, fields ...Fields) {
	if len(fields) > 0 {
		event.Fields(fields[0])
	}
	event.Msg(msg)
}

func convertLogLevel(level Level) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// noopLogger discards all log output. Used as a safe fallback when no logger is provided.
type noopLogger struct{}

// Noop returns a Logger that silently discards all log output.
func Noop() Logger { return &noopLogger{} }

func (noopLogger) Debug(_ string, _ ...Fields) {}
func (noopLogger) Info(_ string, _ ...Fields)  {}
func (noopLogger) Warn(_ string, _ ...Fields)  {}
func (noopLogger) Error(_ string, _ ...Fields) {}
func (noopLogger) Fatal(_ string, _ ...Fields) {}
func (noopLogger) WithFields(_ Fields) Logger  { return &noopLogger{} }
func (noopLogger) WithError(_ error) Logger    { return &noopLogger{} }
