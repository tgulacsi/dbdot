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
	// nodes are the tables
	for _, t := range tables {
		//fmt.Fprintf(bw, "\tgraph %s {\n", t.Name)
		//bw.WriteString("\t\tnode [shape=record];\n\t\trankdir=LR;\n")
		fmt.Fprintf(bw, "\ttable_%s [label=\"", t.Name)
		for i, f := range t.Fields {
			if i > 0 {
				bw.WriteByte('|')
			}
			fmt.Fprintf(bw, "<%s> %s %s", f.Name, f.Name, f.Type)
		}
		bw.WriteString("\"];\n")
		//bw.WriteString("\"];\n\t}\n")
	}
	fmt.Fprintln(bw, "}")
	return nil
}
