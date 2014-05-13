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
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
)

const html = false

func makeDot(w io.Writer, tables []table, sources []source) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	fmt.Fprintln(bw, "graph tables {")
	bw.WriteString("\tnode [shape=record];\n")

	tableNames := make(map[string]struct{}, len(tables))
	for _, t := range tables {
		//fmt.Fprintf(bw, "\tgraph %s {\n", t.Name)
		//bw.WriteString("\t\tnode [shape=record];\n\t\trankdir=LR;\n")
		tableNames[t.Name] = struct{}{}
	}
	usedTables := make(map[string][]string, len(tableNames))
	// edges
	edges := make(map[link]struct{}, 512)
	for _, src := range sources {
		code := src.Code
		for _, sel := range getSelects(code) {
			for _, lnk := range selectGetLinks(sel) {
				if _, ok := tableNames[lnk.A.Table]; !ok {
					glog.Infof("%q is not a table name.", lnk.A.Table)
					continue
				}
				if _, ok := tableNames[lnk.B.Table]; !ok {
					glog.Infof("%q is not a table name.", lnk.B.Table)
					continue
				}
				usedTables[lnk.A.Table] = addString(usedTables[lnk.A.Table], lnk.A.Field)
				usedTables[lnk.B.Table] = addString(usedTables[lnk.B.Table], lnk.B.Field)
				edges[lnk] = struct{}{}
			}
		}
	}

	// nodes are the tables
	if html {
		for _, t := range tables {
			//fmt.Fprintf(bw, "\tgraph %s {\n", t.Name)
			//bw.WriteString("\t\tnode [shape=record];\n\t\trankdir=LR;\n")
			fields, ok := usedTables[t.Name]
			if !ok {
				glog.Infof("%q not used, skipping.", t.Name)
				continue
			}
			fmt.Fprintf(bw, "\t"+`table_%s [style=none, label=<
<table border="0" cellborder="1" cellspacing="0">
  <tr><td align="center" bgcolor="BLACK"><font color="WHITE"><b>%s</b></font></td></tr>
`, t.Name, unocaps(t.Name))
			for _, fieldName := range fields {
				for _, f := range t.Fields {
					if f.Name != fieldName {
						continue
					}
					//fields[f.Name] = append(fields[f.Name], t.Name)
					fmt.Fprintf(bw, `  <tr><td align="left" PORT="%s">%s %s</td></tr>
`, f.Name, unocaps(f.Name), f.Type)
					break
				}
			}
			bw.WriteString("</table>\n>];\n")
			//bw.WriteString("\"];\n\t}\n")
		}
	} else {
		for _, t := range tables {
			//fmt.Fprintf(bw, "\tgraph %s {\n", t.Name)
			//bw.WriteString("\t\tnode [shape=record];\n\t\trankdir=LR;\n")
			fields, ok := usedTables[t.Name]
			if !ok {
				glog.Infof("%q not used, skipping.", t.Name)
				continue
			}
			fmt.Fprintf(bw, "\ttable_%s [label=\"{%s", t.Name, t.Name)
			for _, fieldName := range fields {
				for _, f := range t.Fields {
					if f.Name != fieldName {
						continue
					}
					fmt.Fprintf(bw, "|<%s> %s %s", f.Name, unocaps(f.Name), f.Type)
					break
				}
			}
			bw.WriteString("}\"];\n")
			//bw.WriteString("\"];\n\t}\n")
		}
	}
	bw.WriteByte('\n')

	// edges
	for lnk := range edges {
		fmt.Fprintf(bw, "\ttable_%s:%s -- table_%s:%s;\n",
			lnk.A.Table, lnk.A.Field,
			lnk.B.Table, lnk.B.Field,
		)
	}

	fmt.Fprintln(bw, "}")
	return nil
}

func unocaps(text string) string {
	i := strings.IndexByte(text, '_')
	if i < 0 {
		return text
	}
	return strings.ToUpper(text[:i]) + "_" + strings.ToLower(text[i+1:])
}

func addString(strings []string, elt string) []string {
	for _, v := range strings {
		if v == elt {
			return strings
		}
	}
	return append(strings, elt)
}
