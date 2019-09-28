// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package api

import (
	"log"
	"net/http"
	"time"
)

type loggerWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggerWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggerWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

type loggerHandler struct {
	http.Handler
	Logger *log.Logger
}

func (h loggerHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	lw := &loggerWriter{ResponseWriter: rw}
	t := time.Now()
	h.Handler.ServeHTTP(lw, r)
	if lw.status == 0 {
		lw.status = 200
	}
	h.Logger.Printf("[%v] %v %v %v\n", r.Method, r.URL.RequestURI(), lw.status, time.Since(t).String())
}

func loggerMiddleware(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return loggerHandler{
			Handler: next,
			Logger:  logger,
		}
	}
}

func authMiddleware(token string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != token {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(rw, r)
		})
	}
}
