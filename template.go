package main

const entryTmpl = `
entity "{{ .Name }}{{- if .Comment }} - {{ .Comment }}{{- end }}" {
{{- range .Columns }}
  {{- if .IsPrimaryKey }}
  + {{ .Name }} [PK]{{- if .Comment }} : {{ .Comment }}{{- end }}
  {{- end }}
{{- end }}
  --
{{- range .Columns }}
  {{- if not .IsPrimaryKey }}
  + {{ .Name }} {{- if .Comment }} : {{ .Comment }}{{- end }}
  {{- end }}
{{- end }}
}
`

const relationTmpl = `
{{ if .IsOneToOne }} {{ .ChildTableName }} ||-|| {{ .ParentTableName }} {{else}} {{ .ChildTableName }} }-- {{ .ParentTableName }} {{end}}
`
