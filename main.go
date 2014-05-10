package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/golang/glog"
	"github.com/juju/errgo"
	_ "github.com/tgulacsi/gocilib/driver"
	//_ "github.com/tgulacsi/goracle/godrv"
)

func main() {
	flagDsn := flag.String("connect", "", "database connection string")
	flag.Parse()

	if *flagDsn == "" {
		log.Fatal("a database connection string is needed!")
	}

	db, err := sql.Open("gocilib", *flagDsn)
	if err != nil {
		log.Fatalf("error connecting to %q: %v", *flagDsn, err)
	}
	var errMu sync.Mutex
	tables := make(chan table, 8)
	go func() {
		if e := getTables(db, tables); e != nil {
			errMu.Lock()
			err = e
			errMu.Unlock()
		}
	}()
	for tbl := range tables {
		glog.V(1).Infof("table %#v", tbl)
		fmt.Printf("%#v", tbl)
	}
	errMu.Lock()
	if err != nil {
		glog.Errorf("ERROR: %v", err)
	}
	errMu.Unlock()
}

type table struct {
	Name, Comment string
	Fields        []field
}

type field struct {
	Name, Type, Comment string
}

func getTables(db *sql.DB, tables chan<- table) error {
	defer close(tables)
	var firstErr error
	errs := make(chan error, 8)
	var errWg sync.WaitGroup
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for e := range errs {
			if e != nil {
				log.Printf("error %v", e)
				if firstErr == nil {
					firstErr = e
				}
			}
		}
	}()
	partTables := make(chan table, 0)
	var workWg sync.WaitGroup
	workWg.Add(1)
	go func() {
		errs <- getTableNames(db, partTables)
		workWg.Done()
	}()

	for i := 0; i < 8; i++ {
		workWg.Add(1)
		go func() {
			defer workWg.Done()
			for tbl := range partTables {
				fields, err := getTableFields(db, tbl.Name)
				if err != nil {
					errs <- err
					return
				}
				tbl.Fields = fields
				tables <- tbl
			}
		}()
	}
	workWg.Wait()
	close(errs)
	errWg.Wait()
	return firstErr
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

func getTableNames(db *sql.DB, tables chan<- table) error {
	defer close(tables)
	qry := `SELECT A.table_name, NVL(B.comments, ' ')
              FROM user_tab_comments B, user_tables A
              WHERE B.table_name(+) = A.table_name AND
                    (A.table_name LIKE 'T_%' OR A.table_name LIKE 'R_%')`
	rows, err := db.Query(qry)
	if err != nil {
		return errgo.Notef(err, qry)
	}
	for rows.Next() {
		var tbl table
		if err = rows.Scan(&tbl.Name, &tbl.Comment); err != nil {
			log.Printf("error scanning table name: %v", err)
			continue
		}
		if tbl.Comment == " " {
			tbl.Comment = ""
		}
		glog.V(1).Infof("part %s", tbl)
		tables <- tbl
	}
	if err := rows.Err(); err != nil && err != io.EOF {
		return errgo.Mask(err)
	}
	return nil
}
