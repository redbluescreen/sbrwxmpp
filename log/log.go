// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package log

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
	verbose bool
}

func (l *Logger) Debugln(v ...interface{}) {
	if l.verbose {
		l.Println(v...)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if l.verbose {
		l.Print(v...)
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.verbose {
		l.Printf(format, v...)
	}
}

func New(prefix string, verbose bool) *Logger {
	return &Logger{
		Logger:  log.New(os.Stderr, prefix, log.LstdFlags),
		verbose: verbose,
	}
}
