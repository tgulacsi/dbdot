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
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/golang/glog"
	"github.com/juju/errgo"
	_ "github.com/tgulacsi/gocilib/driver"
	//_ "github.com/tgulacsi/goracle/godrv"
)

func main() {
	flagDsn := flag.String("connect", "", "database connection string")
	flagZip := flag.String("zip", "", "save here (if connect is specified), or load from here (if connect is empty)")
	flag.Parse()

	tables := make([]table, 0, 128)
	sources := make([]source, 0, 128)

	if *flagDsn == "" {
		if *flagZip == "" {
			log.Fatal("a database connection string or a specified zip is needed!")
		}
		zr, err := zip.OpenReader(*flagZip)
		if err != nil {
			log.Fatalf("error opening %q: %v", *flagZip, err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			if !(f.Name == "tables.json" || f.Name == "sources.json") {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				log.Fatal("error opening %q: %v", f.Name, err)
			}
			switch f.Name {
			case "tables.json":
				err = json.NewDecoder(rc).Decode(&tables)
				glog.Infof("read %d tables", len(tables))
			case "sources.json":
				err = json.NewDecoder(rc).Decode(&sources)
				glog.Infof("read %d sources", len(sources))
			}
			rc.Close()
			if err != nil {
				log.Fatalf("error decoding from %q: %v", f.Name, err)
			}
		}
	} else {
		db, err := sql.Open("gocilib", *flagDsn)
		if err != nil {
			log.Fatalf("error connecting to %q: %v", *flagDsn, err)
		}
		tables, err := getTables(db)
		if err != nil {
			log.Fatalf("error getting tables: %s", errgo.Details(err))
		}
		sources, err := getSources(db)
		if err != nil {
			log.Fatalf("error getting sources: %s", errgo.Details(err))
		}

		// save
		if *flagZip != "" {
			glog.Infof("saving data to %q", *flagZip)
			zfh, err := os.Create(*flagZip)
			if err != nil {
				log.Fatalf("error creating %q: %v", *flagZip, err)
			}
			defer zfh.Close()
			zw := zip.NewWriter(zfh)
			defer zw.Close()

			w, err := zw.Create("tables.json")
			if err != nil {
				log.Fatal("error creating tables.json: %v", err)
			}
			if err = json.NewEncoder(w).Encode(tables); err != nil {
				log.Fatal("error encoding tables: %v", err)
			}

			w, err = zw.Create("sources.json")
			if err != nil {
				log.Fatal("error creating sources.json")
			}
			if err = json.NewEncoder(w).Encode(sources); err != nil {
				log.Fatal("error encoding sources: %v", err)
			}
		}
	}

	defer os.Stdout.Close()
	if err := makeDot(os.Stdout, tables); err != nil {
		log.Fatalf("error creating dot: %v", err)
	}
}

type table struct {
	Name, Comment string
	Fields        []field
}

type field struct {
	Name, Type, Comment string
}

type source struct {
	Name, Type string
	Code       string
}

func getSources(db *sql.DB) ([]source, error) {
	sources := make([]source, 0, 64)
	qry := `SELECT name, type, text FROM user_source
	          ORDER BY name, type, line`
	rows, err := db.Query(qry)
	if err != nil {
		return nil, errgo.Notef(err, qry)
	}
	var s, t source
	lines := bytes.NewBuffer(make([]byte, 0, 1<<20))
	for rows.Next() {
		var line string
		if err = rows.Scan(&t.Name, &t.Type, &line); err != nil {
			log.Printf("error scanning source: %v", err)
			continue
		}
		if s.Name != t.Name {
			if s.Name != "" {
				s.Code = lines.String()
				sources = append(sources, s)
			}
			s = t
			lines.Reset()
		}
		lines.WriteString(line)
	}
	if t.Name != "" {
		t.Code = lines.String()
		sources = append(sources, t)
	}
	if err = rows.Err(); err != nil && err != io.EOF {
		return sources, errgo.Mask(err)
	}
	return sources, nil
}

func getTables(db *sql.DB) ([]table, error) {
	tableNames, err := getTableNames(db)
	if err != nil {
		return nil, errgo.Notef(err, "table names")
	}

	qry := `SELECT A.table_name, A.column_name, A.data_type, NVL(B.comments, ' ')
      FROM user_col_comments B, user_tab_cols A
        WHERE B.column_name(+) = A.column_name AND
              B.table_name(+) = A.table_name
	    ORDER BY A.table_name, A.column_id`
	rows, err := db.Query(qry)
	if err != nil {
		return nil, errgo.Notef(err, qry)
	}
	tables := make([]table, 0, len(tableNames))
	var t table
	var act, prev string
	for rows.Next() {
		var f field
		if err = rows.Scan(&act, &f.Name, &f.Type, &f.Comment); err != nil {
			glog.Warningf("error scanning field: %v", err)
			continue
		}
		if prev != act {
			if prev != "" {
				tables = append(tables, t)
			}
			prev = act
			t.Name = act
			t.Comment = tableNames[act]
			t.Fields = make([]field, 0, 8)
		}
		glog.V(2).Infof("field %s", f)
		t.Fields = append(t.Fields, f)
	}
	tables = append(tables, t)
	if err = rows.Err(); err != nil && err != io.EOF {
		return tables, errgo.Mask(err)
	}
	return tables, nil
}

func getTableFields(db *sql.DB, tbl string) ([]field, error) {
	qry := `SELECT A.column_name, A.data_type, NVL(B.comments, ' ')
      FROM user_col_comments B, user_tab_cols A
        WHERE B.column_name(+) = A.column_name AND
              B.table_name(+) = A.table_name AND
              A.table_name = :1`
	rows, err := db.Query(qry, tbl)
	if err != nil {
		return nil, errgo.Notef(err, qry)
	}
	fields := make([]field, 0, 8)
	for rows.Next() {
		var f field
		if err = rows.Scan(&f.Name, &f.Type, &f.Comment); err != nil {
			log.Printf("error scanning field: %v", err)
			continue
		}
		if f.Comment == " " {
			f.Comment = ""
		}
		glog.V(1).Infof("tbl %s field %s", tbl, f.Name)
		fields = append(fields, f)
	}
	if err = rows.Err(); err != nil && err != io.EOF {
		return fields, errgo.Mask(err)
	}
	return fields, nil
}

func getTableNames(db *sql.DB) (map[string]string, error) {
	qry := `SELECT A.table_name, NVL(B.comments, ' ')
              FROM user_tab_comments B, user_tables A
              WHERE B.table_name(+) = A.table_name AND
                    (A.table_name LIKE 'T_%' OR A.table_name LIKE 'R_%')`
	rows, err := db.Query(qry)
	if err != nil {
		return nil, errgo.Notef(err, "query %q", qry)
	}
	tables := make(map[string]string, 128)
	for rows.Next() {
		var name, comment string
		if err = rows.Scan(&name, &comment); err != nil {
			glog.Warningf("error scanning table name: %v", err)
			continue
		}
		if comment == " " {
			comment = ""
		}
		glog.V(1).Infof("table %s (%q)", name, comment)
		tables[name] = comment
	}
	if err := rows.Err(); err != nil && err != io.EOF {
		return tables, errgo.Notef(err, "rows")
	}
	return tables, nil
}
