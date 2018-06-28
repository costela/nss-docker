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
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func Test_removeCommentLines(t *testing.T) {
	tests := []struct {
		name    string
		args    io.Reader
		want    []byte
		wantErr bool
	}{
		{"no_comments_simple", strings.NewReader("a"), []byte("a"), false},
		{"no_comments_newline", strings.NewReader("a\nb\n"), []byte("a\nb\n"), false},
		{"simple_comments_start", strings.NewReader("//comment\na"), []byte("a"), false},
		{"simple_comments_middle", strings.NewReader("a\n//comment\nb"), []byte("a\nb"), false},
		{"simple_comments_end", strings.NewReader("a\n//comment"), []byte("a\n"), false},
		{"inline_comments_not_supported", strings.NewReader("a //comment"), []byte("a //comment"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := removeCommentLines(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeCommentLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			out, err := ioutil.ReadAll(got)
			if err != nil {
				t.Errorf("error reading from returned reader: %s", err)
				return
			}
			if !reflect.DeepEqual(out, tt.want) {
				t.Errorf("removeCommentLines() = '%s', want '%s'", out, tt.want)
			}
		})
	}
}

func Test_normalizeSuffix(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{"no_dot", "suffix", ".suffix"},
		{"noop", ".suffix", ".suffix"},
		{"two_dots", "..suffix", ".suffix"},
		{"dot_end", "suffix.", ".suffix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSuffix(tt.arg); got != tt.want {
				t.Errorf("normalizeSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}
