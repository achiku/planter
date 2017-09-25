package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"sort"
	"strings"

	_ "github.com/lib/pq" // postgres
	"github.com/pkg/errors"
)

// Queryer database/sql compatible query interface
type Queryer interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

// OpenDB opens database connection
func OpenDB(connStr string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	return conn, nil
}

// PgColumn postgres columns
type PgColumn struct {
	FieldOrdinal int
	Name         string
	Comment      sql.NullString
	DataType     string
	DDLType      string
	NotNull      bool
	IsPrimaryKey bool
}

// PgForeignKey foreign key
type PgForeignKey struct {
	ConstraintName        string
	ChildTableName        string
	ChildColName          string
	IsChildColPrimaryKey  bool
	ParentTableName       string
	ParentColName         string
	IsParentColPrimaryKey bool
}

// IsOneToMany returns true if one to many relation
func (k PgForeignKey) IsOneToMany() bool {
	if k.IsChildColPrimaryKey && !k.IsParentColPrimaryKey {
		return true
	}
	return false
}

// IsOneToOne returns true if one to one relation
func (k PgForeignKey) IsOneToOne() bool {
	if k.IsChildColPrimaryKey && k.IsParentColPrimaryKey {
		return true
	}
	return false
}

// PgTable postgres table
type PgTable struct {
	Schema      string
	Name        string
	Comment     sql.NullString
	AutoGenPk   bool
	Columns     []*PgColumn
	ForeingKeys []*PgForeignKey
}

func stripCommentSuffix(s string) string {
	if tok := strings.SplitN(s, "\t", 2); len(tok) == 2 {
		return tok[0]
	}
	return s
}

// PgLoadColumnDef load Postgres column definition
func PgLoadColumnDef(db Queryer, schema, table string) ([]*PgColumn, error) {
	colDefs, err := db.Query(columDefSQL, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def")
	}
	var cols []*PgColumn
	for colDefs.Next() {
		var c PgColumn
		err := colDefs.Scan(
			&c.FieldOrdinal,
			&c.Name,
			&c.Comment,
			&c.DataType,
			&c.NotNull,
			&c.IsPrimaryKey,
			&c.DDLType,
		)
		c.Comment.String = stripCommentSuffix(c.Comment.String)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		cols = append(cols, &c)
	}
	return cols, nil
}

// PgLoadForeignKeyDef load Postgres fk definition
func PgLoadForeignKeyDef(db Queryer, schema, table string) ([]*PgForeignKey, error) {
	fkDefs, err := db.Query(fkDefSQL, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fk def")
	}
	var fks []*PgForeignKey
	for fkDefs.Next() {
		fk := PgForeignKey{
			ChildTableName: table,
		}
		err := fkDefs.Scan(
			&fk.ChildColName,
			&fk.ParentTableName,
			&fk.ParentColName,
			&fk.ConstraintName,
			&fk.IsParentColPrimaryKey,
			&fk.IsChildColPrimaryKey,
		)
		if err != nil {
			return nil, err
		}
		fks = append(fks, &fk)
	}
	return fks, nil
}

// PgLoadTableDef load Postgres table definition
func PgLoadTableDef(db Queryer, schema string) ([]*PgTable, error) {
	tbDefs, err := db.Query(tableDefSQL, schema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def")
	}
	var tbs []*PgTable
	for tbDefs.Next() {
		t := &PgTable{Schema: schema}
		err := tbDefs.Scan(
			&t.Name,
			&t.Comment,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		cols, err := PgLoadColumnDef(db, schema, t.Name)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get columns of %s", t.Name))
		}
		t.Columns = cols
		fks, err := PgLoadForeignKeyDef(db, schema, t.Name)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get fks of %s", t.Name))
		}
		t.ForeingKeys = fks
		tbs = append(tbs, t)
	}
	return tbs, nil
}

// PgTableToUMLEntry table entry
func PgTableToUMLEntry(tbls []*PgTable) ([]byte, error) {
	tpl, err := template.New("entry").Parse(entryTmpl)
	if err != nil {
		return nil, err
	}
	var src []byte
	for _, tbl := range tbls {
		buf := new(bytes.Buffer)
		if err := tpl.Execute(buf, tbl); err != nil {
			return nil, errors.Wrapf(err, "failed to execute template: %s", tbl.Name)
		}
		src = append(src, buf.Bytes()...)
	}
	return src, nil
}

// PgForeignKeyToUMLRelation relation
func PgForeignKeyToUMLRelation(tbls []*PgTable) ([]byte, error) {
	tpl, err := template.New("relation").Parse(relationTmpl)
	if err != nil {
		return nil, err
	}
	var src []byte
	for _, tbl := range tbls {
		for _, rel := range tbl.ForeingKeys {
			buf := new(bytes.Buffer)
			if err := tpl.Execute(buf, rel); err != nil {
				return nil, errors.Wrapf(err, "failed to execute template: %s", rel.ConstraintName)
			}
			src = append(src, buf.Bytes()...)
		}
	}
	return src, nil
}

func contains(v string, l []string) bool {
	i := sort.SearchStrings(l, v)
	if i < len(l) && l[i] == v {
		return true
	}
	return false
}

// FilterTables filter tables
func FilterTables(tbls []*PgTable, tblNames []string) []*PgTable {
	sort.Strings(tblNames)

	var target []*PgTable
	for _, tbl := range tbls {
		if contains(tbl.Name, tblNames) {
			var fks []*PgForeignKey
			for _, fk := range tbl.ForeingKeys {
				if contains(fk.ParentTableName, tblNames) {
					fks = append(fks, fk)
				}
			}
			tbl.ForeingKeys = fks
			target = append(target, tbl)
		}
	}
	return target
}
