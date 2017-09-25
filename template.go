package main

const entryTmpl = `
entity "{{ .Name }}" {
{{- if .Comment.Valid }}
  {{ .Comment.String }}
  ..
{{- end }}
{{- range .Columns }}
  {{- if .IsPrimaryKey }}
  + {{ .Name }} [PK]{{- if .Comment.Valid }} : {{ .Comment.String }}{{- end }}
  {{- end }}
{{- end }}
  --
{{- range .Columns }}
  {{- if not .IsPrimaryKey }}
  {{ .Name }} {{- if .Comment.Valid }} : {{ .Comment.String }}{{- end }}
  {{- end }}
{{- end }}
}
`

const relationTmpl = `
{{ if .IsOneToOne }} {{ .ChildTableName }} ||-|| {{ .ParentTableName }} {{else}} {{ .ChildTableName }} }-- {{ .ParentTableName }} {{end}}
`
