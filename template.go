package main

var tmpl = `package {{ .Package }}
import (
	"github.com/nicheinc/account/v6/internal/metrics"
	{{- range .Imports }}
	{{ . }}
	{{- end }}
)

// {{ .Name }}WithTracer is an implementation of the {{ .Name }}
// interface with all methods traced.
type {{ .Name }}WithTracer{{ .TypeParams }} struct {
	Base {{ .Name }}
}

// Verify that *{{ .Name }}WithTracer implements {{ .Name }}.
{{- if .TypeParams }}
func _{{ .TypeParams }}() {
    var _ {{ .Name }}{{ .TypeParams.Names }} = &{{ .Name }}WithTracer{{ .TypeParams.Names }}{}
}
{{ else }}
var _ {{ .Name }} = &{{ .Name }}WithTracer{}
{{ end }}

{{- range $method := .Methods }}

// {{ $method.Name}} wraps the original {{ $.Name }}.{{ $method.Name }}
// method and also conditionally starts a new tracing span.
func (t *{{ $.Name }}WithTracer{{ $.TypeParams.Names }}) {{ $method.Name }}({{ $method.Params.NamedString }}) {{ $method.Results }}{
	{{- range $param := .Params }}
    {{- if eq $param.Type "context.Context" }}
	tracer := metrics.TracerOrNoopFromCtx({{ $param.Name }})
    {{ $param.Name }}, span := tracer.Start({{ $param.Name }}, "{{ $.Name }}.{{ $method.Name }}")
	defer func() {
		span.End()
	}()
    {{ end }}
	{{ end }}
	{{- if gt (len .Results) 0 }}
	return t.Base.{{ .Name }}({{ .Params.ArgsString }})
	{{- else }}
	t.Base.{{ .Name }}({{ .Params.ArgsString }})
	{{- end }}
}
{{- end -}}
`
