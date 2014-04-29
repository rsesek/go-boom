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
	"encoding/json"
	"io"
	"mime/multipart"
	"net/textproto"
	"time"
)

type ProcessRecord struct {
	Timestamp    time.Time
	Pid          int
	CommandLine  []string
	ExitCode     int
	Signaled     bool
	Signal       int `json:",omitempty"`
	StatusString string
	Uptime       time.Duration

	Stdout bytes.Buffer `json:"-"`
	Stderr bytes.Buffer `json:"-"`
}

func (pr *ProcessRecord) WriteTo(w io.Writer) error {
	mp := multipart.NewWriter(w)
	defer mp.Close()

	io.WriteString(w, "MIME-Version: 1.0\r\n")
	io.WriteString(w, "Content-Type: multipart/mixed; boundary="+mp.Boundary())
	io.WriteString(w, "\r\n\r\n")

	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Disposition", "attachment; filename=ProcessRecord.json")
	headers.Set("Content-Type", "application/json")
	part, err := mp.CreatePart(headers)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(part).Encode(pr); err != nil {
		return err
	}

	files := map[string]io.Reader{
		"stdout": &pr.Stdout,
		"stderr": &pr.Stderr,
	}
	for filename, buf := range files {
		headers := make(textproto.MIMEHeader)
		headers.Set("Content-Disposition", "attachment; filename="+filename)
		headers.Set("Content-Type", "text/plain")
		part, err := mp.CreatePart(headers)
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, buf); err != nil {
			return err
		}
	}

	return nil
}
