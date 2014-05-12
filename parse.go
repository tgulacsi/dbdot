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

var rFrom = regexp.MustCompile(`\sFROM\s`)
var rWhere = regexp.MustCompile(`\sWHERE\s`)

type linkField struct {
	Table, Field string
}

type link struct {
	A, B linkField
}

// selectGetLinks parses code (which should be a SELECT statement only)
// and returns the table1.field1 = table2.field2 pairs.
func selectGetLinks(code string) []link {
	fromI := findRe(code, rFrom)
	if fromI < 0 {
		glog.V(1).Infof("cannot find FROM in %q", code)
		return nil
	}
	fromI += 6
	glog.V(2).Infof("fromI=%d => code[fromI:]=%q", fromI, code[fromI:])
	whereI := findRe(code[fromI:], rWhere)
	if whereI < 0 {
		glog.V(1).Infof("cannot find WHERE in %q", code[fromI:])
		return nil
	}
	glog.V(2).Infof("fromI=%d whereI=%d (%q)", fromI, fromI+whereI, code)
	from := code[fromI : fromI+whereI]
	whereI += fromI + 7
	where := code[whereI:]
	glog.V(2).Infof("fromI=%d whereI=%d => from=%q where=%q", fromI, whereI, from, where)

	tables := fromTables(from)
	glog.V(1).Infof("tables=%#v", tables)

	equations := whereEquations(where, tables)
	if len(equations) == 0 {
		glog.V(1).Infof("no eqs in %q (where=%q tables=%q)", code, where, tables)
		return nil
	}
	links := make([]link, 0, len(equations))
	var lnkScratch [2]linkField
	for _, eq := range equations {
		var lnk link
		for i, fld := range eq {
			j := strings.IndexByte(fld, '.')
			tbl, ok := tables[fld[:j]]
			if !ok {
				glog.V(1).Infof("cannot find table for field %q.", fld)
				goto Next
			}
			lnkScratch[i] = linkField{Table: strings.ToUpper(tbl), Field: strings.ToUpper(fld[j+1:])}
		}
		switch {
		case lnkScratch[0].Table < lnkScratch[1].Table:
			lnk = link{A: lnkScratch[0], B: lnkScratch[1]}
		case lnkScratch[0].Table > lnkScratch[1].Table:
			lnk = link{A: lnkScratch[1], B: lnkScratch[0]}
		default: // equals
			continue
		}
		links = append(links, lnk)
	Next:
	}
	return links
}

var rField = regexp.MustCompile("\b[A-Za-z][A-Za-z0-9_]*[.][A-Za-z][A-Za-z0-9_]*\b")

// where Equations returns the table1.field1 = table2.field2 pairs
func whereEquations(where string, tables map[string]string) [][2]string {
	keys := make([]string, 0, len(tables))
	for k := range tables {
		keys = append(keys, k)
	}

	rField := regexp.MustCompile("(?:(?i)(?:" + strings.Join(keys, "|") + "))[.][A-Za-z][A-Za-z0-9_]*\\b")
	glog.V(2).Infof("rField=%s", rField)
	locs := rField.FindAllStringIndex(where, -1)
	if len(locs) == 0 {
		glog.V(1).Infof("cannot find fields in %q", where)
		return nil
	}
	eqs := make([][2]string, 0, len(locs)/2)
	for i := 0; i < len(locs)-1; i++ {
		act, nxt := locs[i], locs[i+1]
		i := strings.IndexByte(where[act[1]:nxt[0]], '=')
		if i < 0 {
			i = strings.Index(where[act[1]:nxt[0]], "LIKE")
			if i < 0 {
				glog.V(1).Infof("cannot find . in %q", where[act[1]:nxt[0]])
				continue
			}
		}
		i += act[1]
		eqs = append(eqs, [2]string{where[act[0]:act[1]], where[nxt[0]:nxt[1]]})
		i++
	}
	return eqs
}

// fromTables returns the sign->table mappings from the from string
func fromTables(from string) map[string]string {
	tables := make(map[string]string, 2)
	for _, part := range strings.Split(from, ",") {
		part = strings.TrimSpace(part)
		if len(part) == 0 || strings.IndexAny(part, "()") >= 0 {
			continue
		}
		i := strings.LastIndex(part, " ")
		if i < 0 {
			tables[strings.ToUpper(part)] = part
			continue
		}
		tables[strings.ToUpper(part[i+1:])] = part[:i]
	}
	return tables
}

// findRe finds the pattern in code and ensures that it is not inside a () bracket pair.
func findRe(code string, pattern *regexp.Regexp) int {
	code = stripStrConsts(code)
	lastCount, lastPos := 0, 0
	for j := 0; j < len(code); j++ {
		loc := pattern.FindStringIndex(code[j:])
		if len(loc) == 0 {
			return -1
		}
		j += loc[0]
		cnt := lastCount + strings.Count(code[lastPos:j+1], "(") - strings.Count(code[lastPos:j], ")")
		if cnt == 0 {
			return j
		}
		lastCount, lastPos = cnt, loc[1]
	}
	return -1
}

var rSelect = regexp.MustCompile(`(FOR\s+[^ ]+\s+IN\s*[(]|[^(]\s*)SELECT\s`)

// getSelects returns the select statements from the code
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
			glog.V(1).Infof("cannot find end of %q in %q", prefix, code[start:])
			break
		}
		selects = append(selects, code[start:start+end])
		i = start + end + 1
	}
	return selects
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

// findEndBracket returns the closing bracket
func findEndBracket(code string) int {
	code = stripStrConsts(code)
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

var rStrConst = regexp.MustCompile("'[^']*'")

// stripStrConsts replaces all 'xxsd' strings with '____' (equal length)
func stripStrConsts(code string) string {
	return rStrConst.ReplaceAllStringFunc(code, func(needle string) string {
		return "'" + strings.Repeat("_", len(needle)-2) + "'"
	})
}

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
