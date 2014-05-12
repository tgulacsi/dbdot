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
	"strings"

	"github.com/golang/glog"
)

var rLineComment = regexp.MustCompile("--[^\n]*(?:\n|$)")
var rComment = regexp.MustCompile("(?s)/[*].*?[*]/")

// stripComments strips the comments from the PL/SQL code
func stripComments(code string) string {
	ocode := code
	code = rLineComment.ReplaceAllStringFunc(code, func(needle string) string {
		n := len(needle)
		if needle[n-1] == '\n' {
			return strings.Repeat(" ", n-1) + "\n"
		}
		return strings.Repeat(" ", n)
	})
	code = rComment.ReplaceAllStringFunc(code, func(needle string) string {
		return strings.Map(
			func(r rune) rune {
				if r == '\n' {
					return '\n'
				}
				return ' '
			}, needle)
	})
	glog.V(2).Infof("%q => %q", ocode, code)
	return code
}

var rSelect = regexp.MustCompile(`(FOR\s+[^ ]+\s+IN\s*[(]|[^(]\s*)SELECT\s`)

func getSelects(code string) []string {
	code = stripComments(code)
	selects := make([]string, 0, 4)
	i := 0
	for {
		loc := rSelect.FindStringIndex(code[i:])
		if len(loc) == 0 {
			break
		}
		start := i + loc[1] - 7
		prefix := code[i+loc[0] : i+loc[0]+3]
		i = start
		glog.V(2).Infof("start=%d prefix=%q rest=%q", start, prefix, code[start:])
		var end int
		if prefix == "FOR" {
			end = findEndBracket("("+code[start:]) - 1
		} else {
			end = findEndSemi(code[start:])
		}
		if end < 0 {
			glog.Warningf("cannot find end of %q in %q", prefix, code[start:])
			break
		}
		selects = append(selects, code[start:start+end])
		i = start + end + 1
	}
	return selects
}

// selectGetLinks parses code (which should be a SELECT statement only)
// and returns the table1.field1 = table2.field2 pairs.
func selectGetLinks(code string) [][2]string {
	return nil
}

// findEndSemi returns the closing semicolon
func findEndSemi(code string) int {
	return findNonStrConst(code, ";")
}

func findNonStrConst(code, ending string) int {
	lastCount, lastPos := 0, 0
	for j := 0; j < len(code); j += len(ending) {
		k := strings.Index(code[j:], ending)
		if k < 0 {
			return -1
		}
		j += k
		glog.V(2).Infof("j=%d rest=%q", j, code[j:])
		cnt := lastCount + strings.Count(code[lastPos:j], "'")
		if cnt%2 == 0 {
			return j
		}
		lastCount, lastPos = cnt, j
	}
	return -1
}

var rStrConst = regexp.MustCompile("'[^']*'")

// findEndBracket returns the closing bracket
func findEndBracket(code string) int {
	code = rStrConst.ReplaceAllStringFunc(code, func(needle string) string {
		return "'" + strings.Repeat("_", len(needle)-2) + "'"
	})
	glog.V(2).Infof("findEndBracket(%q)", code)
	lastCount, lastPos := 0, 0
	for j := 0; j < len(code); j++ {
		k := strings.Index(code[j:], ")")
		if k < 0 {
			return -1
		}
		j += k
		glog.V(2).Infof("j=%d rest=%q", j, code[j:])
		cnt := lastCount + strings.Count(code[lastPos:j+1], "(") - strings.Count(code[lastPos:j+1], ")")
		if cnt == 0 {
			return j
		}
		lastCount, lastPos = cnt, j+1
	}
	return -1
}
