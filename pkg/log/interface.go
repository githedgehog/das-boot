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

import "go.uber.org/zap/zapcore"

// Interface wraps the most important calls to a zap logger.
// It provides the standard structured logging functions, but also
// the formatted message functions which are very useful once in a
// while. However, they are cumbersome to access through the
// SugaredLogger. This also allows to unit test log statments very
// well with a go mock.
type Interface interface {
	Debug(msg string, fields ...zapcore.Field)
	Debugf(template string, args ...interface{})
	Info(msg string, fields ...zapcore.Field)
	Infof(template string, args ...interface{})
	Warn(msg string, fields ...zapcore.Field)
	Warnf(template string, args ...interface{})
	Error(msg string, fields ...zapcore.Field)
	Errorf(template string, args ...interface{})
	DPanic(msg string, fields ...zapcore.Field)
	DPanicf(template string, args ...interface{})
	Panic(msg string, fields ...zapcore.Field)
	Panicf(template string, args ...interface{})
	Fatal(msg string, fields ...zapcore.Field)
	Fatalf(template string, args ...interface{})
	Sync() error
}
