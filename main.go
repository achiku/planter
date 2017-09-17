package main

import (
	"io"
	"log"
	"os"

	"github.com/alecthomas/kingpin"
)

var (
	connStr = kingpin.Arg(
		"conn", "PostgreSQL connection string in URL format").Required().String()
	schema = kingpin.Flag(
		"schema", "PostgreSQL schema name").Default("public").Short('s').String()
	outFile    = kingpin.Flag("output", "output file path").Short('o').String()
	targetTbls = kingpin.Flag("table", "target tales").Short('t').Strings()
)

func main() {
	kingpin.Parse()

	db, err := OpenDB(*connStr)
	if err != nil {
		log.Fatal(err)
	}

	ts, err := PgLoadTableDef(db, *schema)
	if err != nil {
		log.Fatal(err)
	}

	var tbls []*PgTable
	if len(*targetTbls) != 0 {
		tbls = FilterTables(ts, *targetTbls)
	} else {
		tbls = ts
	}
	entry, err := PgTableToUMLEntry(tbls)
	if err != nil {
		log.Fatal(err)
	}
	rel, err := PgForeignKeyToUMLRelation(tbls)
	if err != nil {
		log.Fatal(err)
	}
	var src []byte
	src = append([]byte("@startuml"), entry...)
	src = append(src, rel...)
	src = append(src, []byte("@enduml")...)

	var out io.Writer
	if *outFile != "" {
		out, err = os.Create(*outFile)
		if err != nil {
			log.Fatalf("failed to create output file %s: %s", *outFile, err)
		}
	} else {
		out = os.Stdout
	}
	if _, err := out.Write(src); err != nil {
		log.Fatal(err)
	}
}
