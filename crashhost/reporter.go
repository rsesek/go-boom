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
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const kGoCrashHost = "GO_CRASH_REPORTER_HOST"

func EnableCrashReporting(uploadUrl string) {
	// Already running under crash reporter.
	if crashHostPid, _ := strconv.Atoi(os.Getenv(kGoCrashHost)); crashHostPid == os.Getppid() {
		return
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)

	// Not running under crash reporter, so re-exec, specifying this as the crash host pid.
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d", kGoCrashHost, os.Getpid()))

	var pr ProcessRecord
	pr.CommandLine = os.Args
	start := time.Now()

	// Make sure that standard fds are given back to the controlling tty,
	// but also caputre stdout/err for uploading with the crash report.
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.MultiWriter(&pr.Stdout, os.Stdout)
	cmd.Stderr = io.MultiWriter(&pr.Stderr, os.Stderr)
	cmd.Run()
	if cmd.ProcessState.Success() {
		// The actual process has finished, so exit cleanly.
		os.Exit(0)
	}

	// The process did not exit cleanly, so start building a crash report.
	pr.Timestamp = time.Now()
	pr.Uptime = pr.Timestamp.Sub(start)
	pr.Pid = cmd.ProcessState.Pid()

	// TODO(rsesek): Is this OK on Windows?
	waitpid := cmd.ProcessState.Sys().(syscall.WaitStatus)
	pr.StatusString = cmd.ProcessState.String()
	pr.ExitCode = waitpid.ExitStatus()
	pr.Signaled = waitpid.Signaled()
	if pr.Signaled {
		pr.Signal = int(waitpid.Signal())
	}

	var buf bytes.Buffer
	if err := pr.WriteTo(&buf); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write crash report:", err)
		os.Exit(waitpid.ExitStatus())
		return
	}

	if err := uploadReport(uploadUrl, &buf); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to upload crash report:", err)
		os.Exit(waitpid.ExitStatus())
		return
	}

	os.Exit(waitpid.ExitStatus())
}

func uploadReport(uploadUrl string, body io.Reader) error {
	req, err := http.NewRequest("POST", uploadUrl, body)
	if err != nil {
		return err
	}

	if req.URL.Scheme != "https" {
		fmt.Fprintln(os.Stderr, "\n\n*** WARNING: Submitting privacy-sensitive crash report over clear-text HTTP, not secure HTTPS! ***\n\n")
	}

	_, err = http.DefaultClient.Do(req)
	return err
}
