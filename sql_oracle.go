// +build oracle

package main

import (
	_ "gopkg.in/goracle.v2"
)

func init() {
	DefinitionQueries["oracle"] = QueryDef{
		Column: `
SELECT A.*, A.data_type
  FROM (
SELECT
  NVL(A.column_id, 0) AS field_ordinal,
  A.column_name,
  B.COMMENTS AS DESCRIPTION,
  CASE A.data_type
    WHEN 'DATE' THEN 'DATE'
    WHEN 'NUMBER' THEN
      CASE NVL(A.data_precision, 0)
        WHEN 0 THEN 'NUMBER'
        ELSE
          CASE NVL(A.data_scale, 0) WHEN 0 THEN 'NUMBER('||a.data_precision||')'
               ELSE 'NUMBER('||a.data_precision||','||A.data_scale||')' END
        END
    ELSE CASE WHEN a.data_length IS NOT NULL THEN a.data_type||'('||a.data_length||')'
              ELSE a.data_type END
    END AS data_type,
  CASE A.nullable WHEN 'Y' THEN 0 ELSE 1 END AS not_null,
  (SELECT DECODE(COUNT(0), 0,0, 1)
     FROM all_constraints Y, all_cons_columns X
     WHERE Y.CONSTRAINT_TYPE IN ('P', 'U') AND Y.constraint_name = X.constraint_name AND Y.owner = X.owner AND
           X.column_name = A.column_name AND X.table_name = A.table_name AND X.owner = A.owner) is_primary_key
FROM all_col_comments B, all_tab_cols A
WHERE B.column_name(+) = A.column_name AND B.table_name(+) = A.table_name AND B.owner(+) = A.owner AND
      A.owner = UPPER(:1) AND A.table_name = UPPER(:2)
  ) A
ORDER BY 1`,

		Table: `
SELECT
  A.table_name, B.comments AS description
  FROM all_tab_comments B, all_tables A
  WHERE B.table_name(+) = A.table_name AND B.owner(+) = A.owner AND
        INSTR(A.table_name, '$') = 0 AND A.owner = UPPER(:1)
  ORDER BY A.table_name
`,

		ForeignKey: `
SELECT
  B.column_name AS child_column,
  A.table_name AS parent_table,
  C.column_name AS parent_column,
  C.constraint_name AS conname,
  '1' AS is_parent_pk,
  '0' AS is_child_pk
  FROM all_cons_columns C,
       all_cons_columns B, all_constraints A
  WHERE C.CONSTRAINT_NAME = B.constraint_name AND C.owner = B.owner AND
        B.CONSTRAINT_NAME = A.constraint_name AND B.owner = A.owner AND
        A.constraint_type = 'R' AND
		a.OWNER = UPPER(:1) AND A.table_name = UPPER(:2)
`,
	}
}
