//
// Go Crash Reporter
// Copyright 2014 Google Inc. All rights reserved.
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
//

package crashhost

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

const kCrasherFlag = "-run-under-crash-reporter-host"

// Define a flag so that it can be passed to the child, but it is not checked.
var _ = flag.Bool(kCrasherFlag[1:], false, "The process is running under a crash reporting host.")

type biwriter struct {
	crashbuf bytes.Buffer
	realw    io.Writer
}

func newBiwriter(w io.Writer) *biwriter {
	return &biwriter{
		realw:    w,
	}
}

func (w *biwriter) Write(p []byte) (int, error) {
	w.crashbuf.Write(p)
	return w.realw.Write(p)
}

func EnableCrashReporting() {
	// Already running under crash reporter.
	for _, arg := range os.Args {
		if arg == kCrasherFlag {
			return
		}
	}

	// Not running under crash reporter, so re-exec.
	args := append(os.Args, kCrasherFlag)

	cmd := exec.Command(args[0], args[1:]...)
	// Make sure that standard fds are given back to the controlling tty,
	// but also caputre stdout/err for uploading with the crash report.
	cmd.Stdin = os.Stdin
	cmd.Stdout = newBiwriter(os.Stdout)
	cmd.Stderr = newBiwriter(os.Stderr)
	cmd.Run()

	if !cmd.ProcessState.Success() {
		waitpid := cmd.ProcessState.Sys().(syscall.WaitStatus)
		fmt.Println("Process crashed with exit code", waitpid.ExitStatus())

		if waitpid.Signaled() {
			fmt.Println("... exited with signal", waitpid.Signal())
		}

		// TODO(rsesek): HTTP POST the report + log output.

		os.Exit(waitpid.ExitStatus())
	}

	// The actual process has finished, so exit cleanly.
	os.Exit(0)
}
