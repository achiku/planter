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

// Column postgres columns
type Column struct {
	FieldOrdinal int
	Name         string
	Comment      sql.NullString
	DataType     string
	DDLType      string
	NotNull      bool
	IsPrimaryKey bool
}

// ForeignKey foreign key
type ForeignKey struct {
	ConstraintName        string
	SourceTableName       string
	SourceColName         string
	IsSourceColPrimaryKey bool
	TargetTableName       string
	TargetColName         string
	IsTargetColPrimaryKey bool
}

// IsOneToOne returns true if one to one relation
func (k *ForeignKey) IsOneToOne() bool {
	if k.IsSourceColPrimaryKey && k.IsTargetColPrimaryKey {
		return true
	}
	return false
}

// FindTableByName find table by name
func FindTableByName(tbls []*Table, name string) *Table {
	for _, tbl := range tbls {
		if tbl.Name == name {
			return tbl
		}
	}
	return nil
}

// Table postgres table
type Table struct {
	Schema      string
	Name        string
	Comment     sql.NullString
	AutoGenPk   bool
	Columns     []*Column
	ForeingKeys []*ForeignKey
}

// IsCompositePK check if table is composite pk
func (t *Table) IsCompositePK() bool {
	cnt := 0
	for _, c := range t.Columns {
		if c.IsPrimaryKey {
			cnt++
		}
		if cnt >= 2 {
			return true
		}
	}
	return false
}

func stripCommentSuffix(s string) string {
	if tok := strings.SplitN(s, "\t", 2); len(tok) == 2 {
		return tok[0]
	}
	return s
}

// LoadColumnDef load Postgres column definition
func LoadColumnDef(db Queryer, schema, table string) ([]*Column, error) {
	colDefs, err := db.Query(columDefSQL, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def")
	}
	var cols []*Column
	for colDefs.Next() {
		var c Column
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

// LoadForeignKeyDef load Postgres fk definition
func LoadForeignKeyDef(db Queryer, schema, table string) ([]*ForeignKey, error) {
	fkDefs, err := db.Query(fkDefSQL, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fk def")
	}
	var fks []*ForeignKey
	for fkDefs.Next() {
		fk := ForeignKey{
			SourceTableName: table,
		}
		err := fkDefs.Scan(
			&fk.SourceColName,
			&fk.TargetTableName,
			&fk.TargetColName,
			&fk.ConstraintName,
			&fk.IsTargetColPrimaryKey,
			&fk.IsSourceColPrimaryKey,
		)
		if err != nil {
			return nil, err
		}
		fks = append(fks, &fk)
	}
	return fks, nil
}

// LoadTableDef load Postgres table definition
func LoadTableDef(db Queryer, schema string) ([]*Table, error) {
	tbDefs, err := db.Query(tableDefSQL, schema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def")
	}
	var tbs []*Table
	for tbDefs.Next() {
		t := &Table{Schema: schema}
		err := tbDefs.Scan(
			&t.Name,
			&t.Comment,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		cols, err := LoadColumnDef(db, schema, t.Name)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get columns of %s", t.Name))
		}
		t.Columns = cols
		fks, err := LoadForeignKeyDef(db, schema, t.Name)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get fks of %s", t.Name))
		}
		t.ForeingKeys = fks
		tbs = append(tbs, t)
	}
	return tbs, nil
}

// TableToUMLEntry table entry
func TableToUMLEntry(tbls []*Table) ([]byte, error) {
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

// ForeignKeyToUMLRelation relation
func ForeignKeyToUMLRelation(tbls []*Table) ([]byte, error) {
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
func FilterTables(tbls []*Table, tblNames []string) []*Table {
	sort.Strings(tblNames)

	var target []*Table
	for _, tbl := range tbls {
		if contains(tbl.Name, tblNames) {
			var fks []*ForeignKey
			for _, fk := range tbl.ForeingKeys {
				if contains(fk.TargetTableName, tblNames) {
					fks = append(fks, fk)
				}
			}
			tbl.ForeingKeys = fks
			target = append(target, tbl)
		}
	}
	return target
}
