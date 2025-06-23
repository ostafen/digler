// Copyright (c) 2025 Stefano Scafiti
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
package logger

import (
	"fmt"
	"io"
	"sync"
)

// Level type for log levels
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func ParseLevel(level string) Level {
	switch level {
	case "INFO":
		return InfoLevel
	case "DEBUG":
		return DebugLevel
	case "WARN":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	}
	return InfoLevel
}

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger defines the logging structure
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	level Level
}

// New creates a new logger writing to a writer with minimum log level
func New(w io.Writer, level Level) *Logger {
	return &Logger{
		out:   w,
		level: level,
	}
}

// log is the internal formatter
func (l *Logger) log(level Level, msg string) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprintf(l.out, "[%s] %s\n", level.String(), msg)
}

// --- Logging Methods ---

func (l *Logger) Debug(msg string) { l.log(DebugLevel, msg) }
func (l *Logger) Info(msg string)  { l.log(InfoLevel, msg) }
func (l *Logger) Warn(msg string)  { l.log(WarnLevel, msg) }
func (l *Logger) Error(msg string) { l.log(ErrorLevel, msg) }

func (l *Logger) Debugf(format string, args ...any) { l.log(DebugLevel, fmt.Sprintf(format, args...)) }
func (l *Logger) Infof(format string, args ...any)  { l.log(InfoLevel, fmt.Sprintf(format, args...)) }
func (l *Logger) Warnf(format string, args ...any)  { l.log(WarnLevel, fmt.Sprintf(format, args...)) }
func (l *Logger) Errorf(format string, args ...any) { l.log(ErrorLevel, fmt.Sprintf(format, args...)) }
