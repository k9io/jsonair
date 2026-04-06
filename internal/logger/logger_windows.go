//go:build windows

/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

/** Ported from Corium; re-licensed by the author under MIT. **/

package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	//"golang.org/x/sys/windows/svc/eventlog"
	"github.com/fatih/color"
)

const (
	INFO   = 1
	WARN   = 2
	NOTICE = 3
	DEBUG  = 4
	ERROR  = 5
)

func Init_Logger(host string, proto string) {

	/* Nothing to init under Windows (yet) */

}

func Logger(log_type int, format string, args ...interface{}) {

	var __FILE__ string /* Use old school __LINE__ and __FILE__ variables */

	Message := fmt.Sprintf(format, args...)

	/* Grab runtime information of the caller for logging */

	_, file, __LINE__, _ := runtime.Caller(1)
	__FILE__ = filepath.Base(file)

	self := filepath.Base(os.Args[0])

	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	switch log_type {

	case INFO:

		fmt.Printf("%s    :%s:%s:%s:\t%s\n", white("Info"), cyan(self), green(__FILE__), green(__LINE__), white(Message))

	case WARN:

		fmt.Printf("%s :%s:%s:%s:\t%s\n", yellow("Warning"), cyan(self), green(__FILE__), green(__LINE__), yellow(Message))

	case NOTICE:

		fmt.Printf("%s  :%s:%s:%s:\t%s\n", cyan("Notice"), cyan(self), green(__FILE__), green(__LINE__), cyan(Message))

	case ERROR:

		fmt.Printf("%s   :%s:%s:%s:\t%s\n", red("Error"), cyan(self), green(__FILE__), green(__LINE__), red(Message))

	case DEBUG:

		fmt.Printf("%s   :%s:%s:%s:\t%s\n", blue("Debug"), cyan(self), green(__FILE__), green(__LINE__), blue(Message))

	case BANNER:

		fmt.Printf("%s    :%s:%s:%s:\t%s\n", white("Info"), cyan(self), green(__FILE__), green(__LINE__), magenta(Message))

	default:

		log.Printf("%s %s", red("!! Unknown logging type: %s !!"), log_type)

	}

}
