// Package plugin implements HashiCorp go-plugin lifecycle management (A15/A27).
package plugin

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"

	hclog "github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapHCLogAdapter wraps zap.Logger to implement hclog.Logger.
// Designed for use as go-plugin ClientConfig.Logger so plugin
// stderr output is captured and routed through the host's zap logger.
type zapHCLogAdapter struct {
	zap         *zap.Logger
	name        string
	lvl         atomic.Int64 // stores hclog.Level as int64
	impliedArgs []interface{}
}

// compile-time interface check
var _ hclog.Logger = (*zapHCLogAdapter)(nil)

// NewZapHCLogAdapter creates an hclog.Logger backed by zap.
func NewZapHCLogAdapter(z *zap.Logger) hclog.Logger {
	a := &zapHCLogAdapter{zap: z}
	a.lvl.Store(int64(hclog.Info))
	return a
}

// getLevel returns current log level.
func (a *zapHCLogAdapter) getLevel() hclog.Level {
	return hclog.Level(int(a.lvl.Load()))
}

// argsToFields converts hclog key-value args to zap.Field slice.
func argsToFields(args []interface{}) []zap.Field {
	if len(args) == 0 {
		return nil
	}
	fields := make([]zap.Field, 0, len(args)/2+1)
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			fields = append(fields, zap.Any(fmt.Sprintf("%v", args[i]), nil))
			break
		}
		fields = append(fields, zap.Any(fmt.Sprintf("%v", args[i]), args[i+1]))
	}
	return fields
}

func (a *zapHCLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	a.logZap(hclogToZapLevel(level), msg, args...)
}

// hclogToZapLevel converts hclog.Level to zapcore.Level.
func hclogToZapLevel(l hclog.Level) zapcore.Level {
	switch l {
	case hclog.Trace, hclog.Debug:
		return zapcore.DebugLevel
	case hclog.Info:
		return zapcore.InfoLevel
	case hclog.Warn:
		return zapcore.WarnLevel
	case hclog.Error:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// logZap logs at given zap level with message and optional hclog args.
func (a *zapHCLogAdapter) logZap(level zapcore.Level, msg string, args ...interface{}) {
	allArgs := append(a.impliedArgs, args...)
	fields := argsToFields(allArgs)
	if a.name != "" {
		fields = append(fields, zap.String("plugin", a.name))
	}
	if ce := a.zap.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}

// --- hclog.Logger implementation ---

func (a *zapHCLogAdapter) Trace(msg string, args ...interface{}) {
	a.logZap(zapcore.DebugLevel, msg, args...)
}

func (a *zapHCLogAdapter) Debug(msg string, args ...interface{}) {
	a.logZap(zapcore.DebugLevel, msg, args...)
}

func (a *zapHCLogAdapter) Info(msg string, args ...interface{}) {
	a.logZap(zapcore.InfoLevel, msg, args...)
}

func (a *zapHCLogAdapter) Warn(msg string, args ...interface{}) {
	a.logZap(zapcore.WarnLevel, msg, args...)
}

func (a *zapHCLogAdapter) Error(msg string, args ...interface{}) {
	a.logZap(zapcore.ErrorLevel, msg, args...)
}

func (a *zapHCLogAdapter) IsTrace() bool { return a.IsDebug() }

func (a *zapHCLogAdapter) IsDebug() bool {
	return a.zap.Core().Enabled(zapcore.DebugLevel)
}

func (a *zapHCLogAdapter) IsInfo() bool {
	return a.zap.Core().Enabled(zapcore.InfoLevel)
}

func (a *zapHCLogAdapter) IsWarn() bool {
	return a.zap.Core().Enabled(zapcore.WarnLevel)
}

func (a *zapHCLogAdapter) IsError() bool {
	return a.zap.Core().Enabled(zapcore.ErrorLevel)
}

func (a *zapHCLogAdapter) With(args ...interface{}) hclog.Logger {
	clone := a.clone()
	clone.impliedArgs = append(clone.impliedArgs, args...)
	return clone
}

func (a *zapHCLogAdapter) Named(name string) hclog.Logger {
	clone := a.clone()
	if clone.name == "" {
		clone.name = name
	} else {
		clone.name = clone.name + "." + name
	}
	return clone
}

func (a *zapHCLogAdapter) ResetNamed(name string) hclog.Logger {
	clone := a.clone()
	clone.name = name
	return clone
}

func (a *zapHCLogAdapter) SetLevel(level hclog.Level) {
	a.lvl.Store(int64(level))
}

func (a *zapHCLogAdapter) GetLevel() hclog.Level {
	return a.getLevel()
}

func (a *zapHCLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	if opts == nil {
		opts = &hclog.StandardLoggerOptions{}
	}
	return log.New(a.StandardWriter(opts), "", 0)
}

func (a *zapHCLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	level := hclog.Info
	if opts != nil && opts.ForceLevel != hclog.NoLevel {
		level = opts.ForceLevel
	}
	return &hclogWriter{adapter: a, level: level}
}

func (a *zapHCLogAdapter) ImpliedArgs() []interface{} {
	return a.impliedArgs
}

func (a *zapHCLogAdapter) Name() string {
	return a.name
}

// clone returns shallow copy preserving atomic level.
func (a *zapHCLogAdapter) clone() *zapHCLogAdapter {
	c := &zapHCLogAdapter{
		zap:         a.zap,
		name:        a.name,
		impliedArgs: append([]interface{}{}, a.impliedArgs...),
	}
	c.lvl.Store(a.lvl.Load())
	return c
}

// hclogWriter implements io.Writer by sending each Write through hclog adapter.
type hclogWriter struct {
	adapter *zapHCLogAdapter
	level   hclog.Level
}

var _ io.Writer = (*hclogWriter)(nil)

func (w *hclogWriter) Write(p []byte) (int, error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	if msg == "" {
		return len(p), nil
	}
	args := []interface{}{"line", msg}
	switch w.level {
	case hclog.Trace, hclog.Debug:
		w.adapter.Debug(msg, args...)
	case hclog.Info:
		w.adapter.Info(msg, args...)
	case hclog.Warn:
		w.adapter.Warn(msg, args...)
	case hclog.Error:
		w.adapter.Error(msg, args...)
	default:
		w.adapter.Info(msg, args...)
	}
	return len(p), nil
}
