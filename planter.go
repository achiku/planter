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
	"golang.org/x/sync/errgroup"
)

// Queryer database/sql compatible query interface
type Queryer interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

// OpenDB opens database connection
func OpenDB(connStr string) (*sql.DB, string, error) {
	typ := "postgres"
	if i := strings.Index(connStr, "://"); i > 0 {
		typ, connStr = connStr[:i], connStr[i+3:]
	}
	conn, err := sql.Open(typ, connStr)
	if err != nil {
		return nil, typ, errors.Wrapf(err, "failed to connect to %q database %q", typ, connStr)
	}
	switch typ {
	case "goracle":
		typ = "oracle"
	}
	return conn, typ, nil
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
	ChildTableName        string
	ChildColName          string
	IsChildColPrimaryKey  bool
	ParentTableName       string
	ParentColName         string
	IsParentColPrimaryKey bool
}

// IsOneToMany returns true if one to many relation
func (k ForeignKey) IsOneToMany() bool {
	if k.IsChildColPrimaryKey && !k.IsParentColPrimaryKey {
		return true
	}
	return false
}

// IsOneToOne returns true if one to one relation
func (k ForeignKey) IsOneToOne() bool {
	if k.IsChildColPrimaryKey && k.IsParentColPrimaryKey {
		return true
	}
	return false
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

func stripCommentSuffix(s string) string {
	if tok := strings.SplitN(s, "\t", 2); len(tok) == 2 {
		return tok[0]
	}
	return s
}

// LoadColumnDef load column definition
func (def QueryDef) LoadColumnDef(db Queryer, schema, table string) ([]*Column, error) {
	colDefs, err := db.Query(def.Column, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def:\n"+def.Column)
	}
	var cols []*Column
	for colDefs.Next() {
		var c Column
		var notNull, isPrimaryKey int
		err := colDefs.Scan(
			&c.FieldOrdinal,
			&c.Name,
			&c.Comment,
			&c.DataType,
			&notNull,
			&isPrimaryKey,
			&c.DDLType,
		)
		c.NotNull, c.IsPrimaryKey = notNull != 0, isPrimaryKey != 0
		c.Comment.String = stripCommentSuffix(c.Comment.String)
		if err != nil {
			var i [7]interface{}
			colDefs.Scan(&i[0], &i[1], &i[2], &i[3], &i[4], &i[5], &i[6])
			return nil, errors.Wrapf(err, "failed to scan %q", i)
		}
		cols = append(cols, &c)
	}
	return cols, nil
}

// LoadForeignKeyDef load fk definition
func (def QueryDef) LoadForeignKeyDef(db Queryer, schema, table string) ([]*ForeignKey, error) {
	fkDefs, err := db.Query(def.ForeignKey, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fk def")
	}
	var fks []*ForeignKey
	for fkDefs.Next() {
		fk := ForeignKey{
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

// LoadTableDef load table definition
func (def QueryDef) LoadTableDef(db Queryer, schema string) ([]*Table, error) {
	tbDefs, err := db.Query(def.Table, schema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def:\n"+def.Table)
	}
	var tbs []*Table
	var grp errgroup.Group
	limits := make(chan struct{}, 16)
	var token struct{}
	for tbDefs.Next() {
		t := &Table{Schema: schema}
		err := tbDefs.Scan(
			&t.Name,
			&t.Comment,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		tbs = append(tbs, t)

		grp.Go(func() error {
			limits <- token
			defer func() { <-limits }()
			cols, err := def.LoadColumnDef(db, schema, t.Name)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to get columns of %s", t.Name))
			}
			t.Columns = cols
			fks, err := def.LoadForeignKeyDef(db, schema, t.Name)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to get fks of %s", t.Name))
			}
			t.ForeingKeys = fks
			return nil
		})
	}
	return tbs, grp.Wait()
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
