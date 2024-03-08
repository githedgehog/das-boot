// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"bufio"
	"context"
	"io"

	"go.uber.org/zap/zapcore"
)

// NewSinkWithLogger will create an io.Writer which can be used as a log sink for logger `l`. It will log
// every newline terminated string that gets written into it at log level `level`. Note that only log levels
// Debug through DPanic are supported. The log levels Panic or Fatal are being ignored (they don't make sense here)
// and will default to the Info level. All additional optional fields will be added to every log message.
func NewSinkWithLogger(ctx context.Context, l Interface, level zapcore.Level, fields ...zapcore.Field) io.Writer {
	reader, writer := io.Pipe()
	go runSink(ctx, l, reader, level, fields)
	return writer
}

func runSink(ctx context.Context, l Interface, buf io.Reader, level zapcore.Level, fields []zapcore.Field) {
	ch := make(chan string)
	defer close(ch)
	go func() {
		defer func() {
			recover() //nolint: errcheck
		}()
		s := bufio.NewScanner(buf)
		for s.Scan() {
			ch <- s.Text()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			// abort everything that we are doing
			return
		case line := <-ch:
			switch level { //nolint: exhaustive
			case zapcore.DebugLevel:
				l.Debug(line, fields...)
			case zapcore.InfoLevel:
				l.Info(line, fields...)
			case zapcore.WarnLevel:
				l.Warn(line, fields...)
			case zapcore.ErrorLevel:
				l.Error(line, fields...)
			case zapcore.DPanicLevel:
				l.DPanic(line, fields...)
			default:
				l.Info(line, fields...)
			}
		}
	}
}
