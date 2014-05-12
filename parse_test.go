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
	"reflect"
	"regexp"
	"testing"
)

var rSpaces = regexp.MustCompile("[ \t\n]+")

func TestLinks(t *testing.T) {
	for i, c := range []struct {
		Code  string
		Links [][2]string
	}{
		{"aaa", nil},
		{"SELECT x FROM table A WHERE A.f= 1", nil},
		{"SELECT x FROM Btab B, Atab A WHERE A.f = B.c", [][2]string{{"A.f", "B.c"}}},
	} {
		got := selectGetLinks(c.Code)
		if len(got) != len(c.Links) {
			t.Errorf("%d. count mismatch: got %d, awaited %d (%q).", i, len(got), len(c.Links), c.Code)
			continue
		}
		for j, v := range got {
			if !(v[0] == c.Links[j][0] && v[1] == c.Links[j][1]) {
				t.Errorf("%d. %d mismatch: got %s, awaited %s (%q).", i, j, got, c.Links, c.Code)
			}
		}
	}
}

func TestFromTables(t *testing.T) {
	for i, c := range []struct {
		From   string
		Tables map[string]string
	}{
		{"aaa", map[string]string{"AAA": "aaa"}},
		{"table A", map[string]string{"A": "table"}},
		{"Btab B, Atab A, Ctab", map[string]string{"A": "Atab", "B": "Btab", "CTAB": "Ctab"}},
	} {
		got := fromTables(c.From)
		if len(got) != len(c.Tables) {
			t.Errorf("%d. count mismatch: got %d, awaited %d (%q).", i, len(got), len(c.Tables), c.From)
			continue
		}
		if !reflect.DeepEqual(got, c.Tables) {
			t.Errorf("%d. mismatch: got %s, awaited %s (%q).", i, got, c.Tables, c.From)
		}
	}
}

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
