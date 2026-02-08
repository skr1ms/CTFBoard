package logger

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Success(t *testing.T) {
	opts := &Options{Level: InfoLevel, Output: ConsoleOutput}
	l := New(opts)
	require.NotNil(t, l)
	l.Info("test")
}

func TestNew_DefaultOutput(t *testing.T) {
	opts := &Options{Level: DebugLevel, Output: OutputType(99)}
	l := New(opts)
	require.NotNil(t, l)
	l.Debug("test")
}

func TestZerologLogger_WithError_Success(t *testing.T) {
	opts := &Options{Level: InfoLevel, Output: ConsoleOutput}
	l := New(opts).WithError(errors.New("err"))
	require.NotNil(t, l)
	l.Info("with error")
}

func TestZerologLogger_WithError_Error(t *testing.T) {
	opts := &Options{Level: WarnLevel, Output: ConsoleOutput}
	l := New(opts).WithError(errors.New("warn err"))
	require.NotNil(t, l)
	l.Warn("warn")
}

func TestZerologLogger_WithFields_Success(t *testing.T) {
	opts := &Options{Level: InfoLevel, Output: ConsoleOutput}
	l := New(opts).WithFields(Fields{"k": "v"})
	require.NotNil(t, l)
	l.Info("with fields")
}

func TestConvertLogLevel_Default(t *testing.T) {
	opts := &Options{Level: Level(100), Output: ConsoleOutput}
	l := New(opts)
	require.NotNil(t, l)
	l.Info("default level")
}

func TestOptions_Apply(t *testing.T) {
	opts := &Options{}
	WithLevel(ErrorLevel)(opts)
	WithOutput(FileOutput)(opts)
	assert.Equal(t, ErrorLevel, opts.Level)
	assert.Equal(t, FileOutput, opts.Output)
}
