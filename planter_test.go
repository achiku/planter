package main

import (
	"database/sql"
	"io/ioutil"
	"testing"
)

// before running test, create user and database
// CREATE USER planter;
// CREATE DATABASE planter OWNER planter;

func testPgSetup(t *testing.T) (*sql.DB, func()) {
	conn, err := sql.Open("postgres", "user=planter dbname=planter sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	setupSQL, err := ioutil.ReadFile("./example/ddl.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(string(setupSQL))
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		conn.Close()
	}
	return conn, cleanup
}

func TestPgLoadColumnDef(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	table := "vendor"
	cols, err := PgLoadColumnDef(conn, schema, table)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cols {
		t.Logf("%+v", c)
	}
}

func TestPgLoadForeignKeyDef(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	table := "order_detail"
	fks, err := PgLoadForeignKeyDef(conn, schema, table)
	if err != nil {
		t.Fatal(err)
	}
	for _, fk := range fks {
		t.Logf("%+v", fk)
	}
}

func TestPgLoadTableDef(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := PgLoadTableDef(conn, schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, tbl := range tbls {
		t.Logf("%+v", tbl.Name)
		for _, c := range tbl.Columns {
			t.Logf("%+v", c)
		}
		for _, f := range tbl.ForeingKeys {
			t.Logf("%+v", f)
		}
	}
}

func TestPgTableToUMLEntry(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := PgLoadTableDef(conn, schema)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := PgTableToUMLEntry(tbls)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", buf)
}

func TestPgForeignKeyToUMLRelation(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := PgLoadTableDef(conn, schema)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := PgForeignKeyToUMLRelation(tbls)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", buf)
}
