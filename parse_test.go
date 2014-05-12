/*
Copyright 2014 Tamás Gulácsi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"regexp"
	"testing"
)

var rSpaces = regexp.MustCompile("[ \t\n]+")

func TestSelects(t *testing.T) {
	for i, c := range []struct {
		Code    string
		Selects []string
	}{
		{"aaa", nil},
		{"aaa--SELECT;", nil},
		{"aa/*SELECT;*/bb", nil},
		{`aa/*SELECT
		;*/sds`, nil},
		{"dasd SELECT f; sdasd", []string{"SELECT f"}},
		{`dassd SELECT --
		from a /*
		sdasd;*/
		WHERE a=';'
		--;
		;aaa`, []string{`SELECT
		from a

		WHERE a=';'

		`},
		},
		{"FOR sor IN (SELECT A FROM (SELECT B))", []string{"SELECT A FROM (SELECT B)"}},
	} {
		got := getSelects(c.Code)
		if len(got) != len(c.Selects) {
			t.Errorf("%d. count mistmatch: got %d, awaited %d (%q).", i, len(got), len(c.Selects), c.Code)
			continue
		}
		if len(got) == 0 {
			continue
		}
		for j, txt := range got {
			if txt != c.Selects[j] && stripSpaces(txt) != stripSpaces(c.Selects[j]) {
				t.Errorf("%d. %d: got %q, awaited %q.", i, j, stripSpaces(txt), stripSpaces(c.Selects[j]))
			}
		}
	}
}
func TestStripComment(t *testing.T) {
	for i, c := range [][2]string{
		{"aaa", "aaa"},
		{"aaa--SELECT;", "aaa         "},
		{"aa/*SELECT;*/bb", "aa           bb"},
		{`aa/*SELECT
		;*/sds`, `aa      sds`},
		{`dassd SELECT --
		from a /*
		sdasd;*/
		WHERE a=';'
		--;
		;aaa`, `dassd SELECT
		from a

		WHERE a=';'

		;aaa`},
	} {
		got := stripComments(c[0])
		if got != c[1] && stripSpaces(got) != stripSpaces(c[1]) {
			t.Errorf("%d. got %q, awaited %q.", i, stripSpaces(got), stripSpaces(c[1]))
		}
	}
}

func TestFindEndSemi(t *testing.T) {
	for i, c := range []struct {
		Text string
		Pos  int
	}{
		{"abraka", -1},
		{"aaa;sdd", 3},
		{"aa';';", 5},
	} {
		got := findEndSemi(c.Text)
		if got != c.Pos {
			t.Errorf("%d. got %d, awaited %d (%q)", i, got, c.Pos, c.Text)
		}
	}
}

func TestFindEndBracket(t *testing.T) {
	for i, c := range []struct {
		Text string
		Pos  int
	}{
		{"(abraka", -1},
		{"(aaa)sdd", 4},
		{"(aa')')", 6},
		{"(aa()')')", 8},
	} {
		got := findEndBracket(c.Text)
		if got != c.Pos {
			t.Errorf("%d. got %d, awaited %d (%q)", i, got, c.Pos, c.Text)
		}
	}
}

func stripSpaces(text string) string {
	return rSpaces.ReplaceAllString(text, " ")
}
