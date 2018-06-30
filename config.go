/*
Copyright © 2018 Leo Antunes <leo@costela.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type configStruct struct {
	Suffix                string `json:"suffix"`
	IncludeComposeProject bool   `json:"includeComposeProject"`
}

var config = configStruct{
	Suffix:                ".docker",
	IncludeComposeProject: true,
}

// used for tests; untouched by init()
var testConfig = config

func parseConfig(r io.Reader) error {
	cleaned, err := removeCommentLines(r)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(cleaned)
	err = dec.Decode(&config)
	config.Suffix = normalizeSuffix(config.Suffix)
	return err
}

func normalizeSuffix(suffix string) string {
	return fmt.Sprintf(".%s", strings.Trim(suffix, "."))
}

func removeCommentLines(r io.Reader) (io.Reader, error) {
	ur := bufio.NewReader(r)
	var buf bytes.Buffer
	for {
		line, err := ur.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			buf.Write([]byte(line))
		}
		if err == io.EOF {
			break
		}
	}
	return &buf, nil
}
