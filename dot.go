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
)

func makeDot(w io.Writer, tables []table) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	fmt.Fprintln(bw, "graph tables {")
	bw.WriteString("\tnode [shape=record];\n\trankdir=LR;\n")

	fields := make(map[string][]string, 64)
	// nodes are the tables
	for _, t := range tables {
		//fmt.Fprintf(bw, "\tgraph %s {\n", t.Name)
		//bw.WriteString("\t\tnode [shape=record];\n\t\trankdir=LR;\n")
		fmt.Fprintf(bw, "\ttable_%s [label=\"%s|", t.Name, t.Name)
		for i, f := range t.Fields {
			fields[f.Name] = append(fields[f.Name], t.Name)
			if i > 0 {
				bw.WriteByte('|')
			}
			fmt.Fprintf(bw, "<%s> %s %s", f.Name, f.Name, f.Type)
		}
		bw.WriteString("\"];\n")
		//bw.WriteString("\"];\n\t}\n")
	}
	bw.WriteByte('\n')

	// edges
	for f, tabs := range fields {
		for i := 1; i < len(tabs); i++ {
			fmt.Fprintf(bw, "table_%s:%s -- table_%s:%s;\n", tabs[0], f, tabs[i], f)
		}
	}

	fmt.Fprintln(bw, "}")
	return nil
}
