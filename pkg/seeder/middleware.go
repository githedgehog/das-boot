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

package seeder

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.uber.org/zap"
)

// SetHeader is a convenience handler to set a response header key/value
func AddResponseRequestID() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			if reqID != "" {
				w.Header().Set("Request-ID", reqID)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func RequestLogger(l log.Interface) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&requestLogFormatter{l: l})
}

// DefaultLogFormatter is a simple logger that implements a LogFormatter.
type requestLogFormatter struct {
	l log.Interface
}

var _ middleware.LogFormatter = &requestLogFormatter{}

// NewLogEntry creates a new LogEntry for the request.
func (l *requestLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	req := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
	reqid := middleware.GetReqID(r.Context())
	verb := r.Method
	from := r.RemoteAddr
	proto := r.Proto
	return &requestLogger{
		l:     l.l,
		verb:  verb,
		req:   req,
		reqid: reqid,
		from:  from,
		proto: proto,
	}
}

type requestLogger struct {
	l     log.Interface
	verb  string
	req   string
	reqid string
	from  string
	proto string
}

func (l *requestLogger) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.l.Info(
		"request",
		zap.String("method", l.verb),
		zap.String("url", l.req),
		zap.String("reqID", l.reqid),
		zap.String("proto", l.proto),
		zap.String("from", l.from),
		zap.Int("status", status),
		zap.Int("bytes", bytes),
		zap.Duration("elapsed", elapsed),
		zap.Reflect("extra", extra),
	)
}

func (l *requestLogger) Panic(v interface{}, stack []byte) {
	l.l.DPanic("panic", zap.Reflect("v", v), zap.ByteString("stack", stack))
}
