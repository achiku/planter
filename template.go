package main

const entryTmpl = `
entity "{{ .Name }}" {
{{- if .Comment.Valid }}
  {{ .Comment.String }}
  ..
{{- end }}
{{- range .Columns }}
  {{- if .IsPrimaryKey }}
  + {{ .Name }}:{{ .DataType }} [PK]{{- if .Comment.Valid }} : {{ .Comment.String }}{{- end }}
  {{- end }}
{{- end }}
  --
{{- range .Columns }}
  {{- if not .IsPrimaryKey }}
  {{if .NotNull}}*{{end}}{{ .Name }}:{{ .DataType }} {{- if .Comment.Valid }} : {{ .Comment.String }}{{- end }}
  {{- end }}
{{- end }}
}
`

const relationTmpl = `
{{ if .IsOneToOne }} {{ .SourceTableName }} ||-|| {{ .TargetTableName }}{{else}} {{ .SourceTableName }} }-- {{ .TargetTableName }}{{end}}
`
