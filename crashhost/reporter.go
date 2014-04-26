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
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const kGoCrashHost = "GO_CRASH_REPORTER_HOST"

type biwriter struct {
	crashbuf bytes.Buffer
	realw    io.Writer
}

func newBiwriter(w io.Writer) *biwriter {
	return &biwriter{
		realw: w,
	}
}

func (w *biwriter) Write(p []byte) (int, error) {
	w.crashbuf.Write(p)
	return w.realw.Write(p)
}

func EnableCrashReporting() {
	// Already running under crash reporter.
	if crashHostPid, _ := strconv.Atoi(os.Getenv(kGoCrashHost)); crashHostPid == os.Getppid() {
		return
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)

	// Not running under crash reporter, so re-exec, specifying this as the crash host pid.
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d", kGoCrashHost, os.Getpid()))

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
