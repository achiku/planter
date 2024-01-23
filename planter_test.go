package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

// before running test, create user and database
// CREATE USER planter;
// CREATE DATABASE planter OWNER planter;

func testPgSetup(t *testing.T) (*sql.DB, func()) {
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf("user=planter port=%s dbname=planter sslmode=disable", port)
	conn, err := sql.Open("postgres", dsn)
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

func TestLoadColumnDef(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	table := "customer"
	cols, err := LoadColumnDef(conn, schema, table)
	if err != nil {
		t.Fatal(err)
	}
	expected := []*Column{
		&Column{
			FieldOrdinal: 1,
			Name:         "id",
			Comment:      sql.NullString{},
			DataType:     "bigint",
			DDLType:      "bigserial",
			NotNull:      true,
			IsPrimaryKey: true,
		},
		&Column{
			FieldOrdinal: 2,
			Name:         "name",
			Comment:      sql.NullString{String: "Customer Name", Valid: true},
			DataType:     "text",
			DDLType:      "text",
			NotNull:      true,
			IsPrimaryKey: false,
		},
		&Column{
			FieldOrdinal: 3,
			Name:         "zip_code",
			Comment:      sql.NullString{String: "Customer Zip Code", Valid: true},
			DataType:     "text",
			DDLType:      "text",
			NotNull:      true,
			IsPrimaryKey: false,
		},
		&Column{
			FieldOrdinal: 4,
			Name:         "address",
			Comment:      sql.NullString{String: "Customer Address", Valid: true},
			DataType:     "text",
			DDLType:      "text",
			NotNull:      true,
			IsPrimaryKey: false,
		},
		&Column{
			FieldOrdinal: 5,
			Name:         "phone_number",
			Comment:      sql.NullString{String: "Customer Phone Number", Valid: true},
			DataType:     "text",
			DDLType:      "text",
			NotNull:      true,
			IsPrimaryKey: false,
		},
		&Column{
			FieldOrdinal: 6,
			Name:         "registered_at",
			Comment:      sql.NullString{},
			DataType:     "timestamp with time zone",
			DDLType:      "timestamp with time zone",
			NotNull:      true,
			IsPrimaryKey: false,
		},
	}
	for i := range cols {
		if !reflect.DeepEqual(cols[i], expected[i]) {
			t.Errorf("\n%+v\n%+v", cols[i], expected[i])
		}
	}
}

func TestFindTableByName(t *testing.T) {
	tbls := []*Table{
		&Table{Name: "t1"},
		&Table{Name: "t2"},
	}
	name := "t2"
	tbl, found := FindTableByName(tbls, name)
	if !found {
		t.Fatalf("%s not found", name)
	}
	if tbl.Name != name {
		t.Errorf("want %s got %s", name, tbl.Name)
	}
}

func TestLoadForeignKeyDef(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := LoadTableDef(conn, schema)
	if err != nil {
		t.Fatal(err)
	}
	n := "order_detail"
	tbl, found := FindTableByName(tbls, n)
	if !found {
		t.Fatalf("%s not found", n)
	}
	fks, err := LoadForeignKeyDef(conn, schema, tbls, tbl)
	if err != nil {
		t.Fatal(err)
	}
	expected := []*ForeignKey{
		&ForeignKey{
			ConstraintName:        "order_detail_customer_order_id_fkey",
			SourceTableName:       "order_detail",
			SourceColName:         "customer_order_id",
			IsSourceColPrimaryKey: true,
			TargetTableName:       "customer_order",
			TargetColName:         "id",
			IsTargetColPrimaryKey: true,
		},
		&ForeignKey{
			ConstraintName:        "order_detail_sku_id_fkey",
			SourceTableName:       "order_detail",
			SourceColName:         "sku_id",
			IsSourceColPrimaryKey: false,
			TargetTableName:       "sku",
			TargetColName:         "id",
			IsTargetColPrimaryKey: false,
		},
	}
	for i := range fks {
		fk, exp := fks[i], expected[i]
		if fk.ConstraintName != exp.ConstraintName {
			t.Errorf("wnat %s got %s", exp.ConstraintName, fk.ConstraintName)
		}
		if fk.SourceTableName != exp.SourceTableName {
			t.Errorf("wnat %s got %s", exp.SourceTableName, fk.SourceTableName)
		}
		if fk.SourceColName != exp.SourceColName {
			t.Errorf("wnat %s got %s", exp.SourceColName, fk.SourceColName)
		}
	}
}

func TestLoadTableDef(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := LoadTableDef(conn, schema)
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

func TestTableToUMLEntry(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := LoadTableDef(conn, schema)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := TableToUMLEntry(tbls)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", buf)
}

func TestForeignKeyToUMLRelation(t *testing.T) {
	conn, cleanup := testPgSetup(t)
	defer cleanup()

	schema := "public"
	tbls, err := LoadTableDef(conn, schema)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := ForeignKeyToUMLRelation(tbls)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", buf)
}

func TestFilterTables(t *testing.T) {
	tables := []*Table{
		{Name: "table1"}, {Name: "table2"},
	}

	t.Run("match = true", func(t *testing.T) {
		match := true

		t.Run("filtered by table1", func(t *testing.T) {
			filters := []string{"table1"}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 1 {
				t.Errorf("want %d got %d", 1, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
		})
		t.Run("filtered by table2", func(t *testing.T) {
			filters := []string{"table2"}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 1 {
				t.Errorf("want %d got %d", 1, len(retval))
			}
			if retval[0].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[0].Name)
			}
		})
		t.Run("filtered by t", func(t *testing.T) {
			filters := []string{"t"}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 2 {
				t.Errorf("want %d got %d", 2, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
			if retval[1].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[1].Name)
			}
		})
		t.Run(`filtered by table\d`, func(t *testing.T) {
			filters := []string{`table\d`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 2 {
				t.Errorf("want %d got %d", 2, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
			if retval[1].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[1].Name)
			}
		})
		t.Run(`filtered by ta*`, func(t *testing.T) {
			filters := []string{`ta*`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 2 {
				t.Errorf("want %d got %d", 2, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
			if retval[1].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[1].Name)
			}
		})
		t.Run(`filtered by [a-z].*1`, func(t *testing.T) {
			filters := []string{`[a-z].*1`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 1 {
				t.Errorf("want %d got %d", 1, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
		})
		t.Run(`filtered by ^table$`, func(t *testing.T) {
			filters := []string{`^t$`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 0 {
				t.Errorf("want %d got %d", 0, len(retval))
			}
		})
	})

	t.Run("match = false", func(t *testing.T) {
		match := false

		t.Run("filtered by table1", func(t *testing.T) {
			filters := []string{"table1"}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 1 {
				t.Errorf("want %d got %d", 1, len(retval))
			}
			if retval[0].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[0].Name)
			}
		})
		t.Run("filtered by table2 xxx", func(t *testing.T) {
			filters := []string{"table2"}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 1 {
				t.Errorf("want %d got %d", 1, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
		})
		t.Run("filtered by t", func(t *testing.T) {
			filters := []string{"t"}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 0 {
				t.Errorf("want %d got %d", 0, len(retval))
			}
		})
		t.Run(`filtered by table\d`, func(t *testing.T) {
			filters := []string{`table\d`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 0 {
				t.Errorf("want %d got %d", 0, len(retval))
			}
		})
		t.Run(`filtered by ta*`, func(t *testing.T) {
			filters := []string{`ta*`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 0 {
				t.Errorf("want %d got %d", 0, len(retval))
			}
		})
		t.Run(`filtered by [a-z].*1`, func(t *testing.T) {
			filters := []string{`[a-z].*1`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 1 {
				t.Errorf("want %d got %d", 1, len(retval))
			}
			if retval[0].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[0].Name)
			}
		})
		t.Run(`filtered by ^table$`, func(t *testing.T) {
			filters := []string{`^t$`}

			retval := FilterTables(match, tables, filters)
			if len(retval) != 2 {
				t.Errorf("want %d got %d", 2, len(retval))
			}
			if retval[0].Name != "table1" {
				t.Errorf("want %s got %s", "table1", retval[0].Name)
			}
			if retval[1].Name != "table2" {
				t.Errorf("want %s got %s", "table2", retval[1].Name)
			}
		})
	})
}
