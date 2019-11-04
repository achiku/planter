package main

const columDefSQL = `
SELECT
    a.attnum AS field_ordinal,
    a.attname AS column_name,
    pd.description AS description,
    format_type(a.atttypid, a.atttypmod) AS data_type,
    a.attnotnull AS not_null,
    COALESCE(ct.contype = 'p', false) AS  is_primary_key,
    CASE WHEN a.atttypid = ANY ('{int,int8,int2}'::regtype[])
      AND EXISTS (
         SELECT 1 FROM pg_attrdef ad
         WHERE  ad.adrelid = a.attrelid
         AND    ad.adnum   = a.attnum
         AND    ad.adbin   = 'nextval('''
            || (pg_get_serial_sequence (a.attrelid::regclass::text
                                      , a.attname))::regclass
            || '''::regclass)'
         )
    THEN CASE a.atttypid
            WHEN 'int'::regtype  THEN 'serial'
            WHEN 'int8'::regtype THEN 'bigserial'
            WHEN 'int2'::regtype THEN 'smallserial'
         END
    ELSE format_type(a.atttypid, a.atttypmod)
    END AS data_type
FROM pg_attribute a
JOIN ONLY pg_class c ON c.oid = a.attrelid
JOIN ONLY pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_constraint ct ON ct.conrelid = c.oid
AND a.attnum = ANY(ct.conkey) AND ct.contype IN ('p', 'u')
LEFT JOIN pg_attrdef ad ON ad.adrelid = c.oid AND ad.adnum = a.attnum
LEFT JOIN pg_description pd ON pd.objoid = a.attrelid AND pd.objsubid = a.attnum
WHERE a.attisdropped = false
AND n.nspname = $1
AND c.relname = $2
AND a.attnum > 0
ORDER BY a.attnum
`

const tableDefSQL = `
SELECT
  c.relname AS table_name,
  pd.description AS description
FROM pg_class c
JOIN ONLY pg_namespace n
ON n.oid = c.relnamespace
LEFT JOIN pg_description pd ON pd.objoid = c.oid AND pd.objsubid = 0
WHERE n.nspname = $1
AND c.relkind = 'r'
ORDER BY c.relname
`

const fkDefSQL = `
select
  att2.attname as "child_column"
  , cl.relname as "parent_table"
  , att.attname as "parent_column"
  , con.conname
  , case 
      when pi.indisprimary is null then false
      else pi.indisprimary
    end as "is_parent_pk"
  , case 
      when ci.indisprimary is null then false
      else ci.indisprimary
    end as "is_child_pk"
from (
  select 
    unnest(con1.conkey) as "parent"
    , unnest(con1.confkey) as "child"
    , con1.confrelid
    , con1.conrelid
    , con1.conname
  from pg_class cl
  join pg_namespace ns on cl.relnamespace = ns.oid
  join pg_constraint con1 on con1.conrelid = cl.oid
  where ns.nspname = $1
  and cl.relname = $2
  and con1.contype = 'f'
) con
join pg_attribute att
on att.attrelid = con.confrelid and att.attnum = con.child
left outer join pg_index pi
on att.attrelid = pi.indrelid and att.attnum = any(pi.indkey)
join pg_class cl
on cl.oid = con.confrelid
join pg_attribute att2
on att2.attrelid = con.conrelid and att2.attnum = con.parent
left outer join pg_index ci
on att2.attrelid = ci.indrelid and att2.attnum = any(ci.indkey)
order by con.conname
`
