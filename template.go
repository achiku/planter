package main

const entryTmpl = `
entity {{ .Name }} {
{{- range .Columns }}
  {{- if .IsPrimaryKey }}
  + {{ .Name }} [PK]
  {{- end }}
{{- end }}
  --
{{- range .Columns }}
  {{- if not .IsPrimaryKey }}
  {{ .Name }}
  {{- end }}
{{- end }}
}
`

const relationTmpl = `
{{ if .IsOneToOne }} {{ .ChildTableName }} ||-|| {{ .ParentTableName }} {{else}} {{ .ChildTableName }} }-- {{ .ParentTableName }} {{end}}
`
