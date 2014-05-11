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

func getSelects(code string) []string {
	code = stripComments(code)
	selects := make([]string, 0, 4)
	i := 0
Loop:
	for {
		start := strings.Index(code[i:], "SELECT ")
		if start < 0 {
			break
		}
		start += i
		i = start
		glog.V(2).Infof("start=%d rest=%q", start, code[start:])
		for j := start + 7; j < len(code); j++ {
			k := strings.Index(code[j:], ";")
			if k < 0 {
				break Loop
			}
			j += k
			glog.V(2).Infof("j=%d rest=%q", j, code[j:])
			if strings.Count(code[start:j], "'")%2 == 0 {
				selects = append(selects, code[start:j])
				break Loop
			}
		}
	}
	return selects
}

// selectGetLinks parses code (which should be a SELECT statement only)
// and returns the table1.field1 = table2.field2 pairs.
func selectGetLinks(code string) [][2]string {
	return nil
}
