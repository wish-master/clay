package genhandler

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	"github.com/pkg/errors"
)

var (
	errNoTargetService = errors.New("no target service defined in the file")
)

type param struct {
	*descriptor.File
	Imports    []descriptor.GoPackage
	SwagBuffer []byte
}

func applyTemplate(p param) (string, error) {
	// r := &http.Request{}
	// r.URL.Query()
	w := bytes.NewBuffer(nil)
	if err := headerTemplate.Execute(w, p); err != nil {
		return "", err
	}

	if err := regTemplate.ExecuteTemplate(w, "base", p); err != nil {
		return "", err
	}

	type swaggerTmpl struct {
		FileName string
		Swagger  string
	}

	if err := footerTemplate.Execute(w, p); err != nil {
		return "", err
	}

	if err := patternsTemplate.ExecuteTemplate(w, "base", p); err != nil {
		return "", err
	}
	//spew.Dump(p.Services[0].Methods[0].Bindings)

	return w.String(), nil
}

var (
	funcMap = template.FuncMap{
		"dotToUnderscore": func(s string) string { return strings.Replace(strings.Replace(s, ".", "_", -1), "/", "_", -1) },
		"byteStr":         func(b []byte) string { return string(b) },
		"escapeBackTicks": func(s string) string { return strings.Replace(s, "`", "` + \"``\" + `", -1) },
	}

	headerTemplate = template.Must(template.New("header").Parse(`
// Code generated by protoc-gen-goclay
// source: {{.GetName}}
// DO NOT EDIT!

/*
Package {{.GoPkg.Name}} is a self-registering gRPC and JSON+Swagger service definition.

It conforms to the github.com/utrack/clay Service interface.
*/
package {{.GoPkg.Name}}
import (
	{{range $i := .Imports}}{{if $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}

	{{range $i := .Imports}}{{if not $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}
)

// Update your shared lib or downgrade generator to v1 if there's an error
var _ = transport.IsVersion2

var _ chi.Router
var _ runtime.Marshaler
`))
	regTemplate = template.Must(template.New("svc-reg").Funcs(funcMap).Parse(`
{{define "base"}}
{{range $svc := .Services}}
// {{$svc.GetName}}Desc is a descriptor/registrator for the {{$svc.GetName}}Server.
type {{$svc.GetName}}Desc struct {
      svc {{$svc.GetName}}Server
}

// New{{$svc.GetName}}ServiceDesc creates new registrator for the {{$svc.GetName}}Server.
func New{{$svc.GetName}}ServiceDesc(svc {{$svc.GetName}}Server) *{{$svc.GetName}}Desc {
      return &{{$svc.GetName}}Desc{svc:svc}
}

// RegisterGRPC implements service registrator interface.
func (d *{{$svc.GetName}}Desc) RegisterGRPC(s *grpc.Server) {
      Register{{$svc.GetName}}Server(s,d.svc)
}

// SwaggerDef returns this file's Swagger definition.
func (d *{{$svc.GetName}}Desc) SwaggerDef() []byte {
      return _swaggerDef_{{dotToUnderscore $.GetName}}
}

// RegisterHTTP registers this service's HTTP handlers/bindings.
func (d *{{$svc.GetName}}Desc) RegisterHTTP(mux transport.Router) {
	{{range $m := $svc.Methods}}
	// Handlers for {{$m.GetName}}
	{{range $b := $m.Bindings}}
	mux.MethodFunc("{{$b.HTTPMethod}}",pattern_goclay_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}}, func(w http.ResponseWriter, r *http.Request) {
          defer r.Body.Close()

	  var req {{$m.RequestType.GetName}}
          err := unmarshaler_goclay_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}}(r,&req)
	  if err != nil {
	    httpruntime.SetError(r.Context(),r,w,errors.Wrap(err,"couldn't parse request"))
	    return
	  }

	  ret,err := d.svc.{{$m.GetName}}(r.Context(),&req)
	  if err != nil {
	    httpruntime.SetError(r.Context(),r,w,errors.Wrap(err,"returned from handler"))
	    return
	  }

          _,outbound := httpruntime.MarshalerForRequest(r)
          w.Header().Set("Content-Type", outbound.ContentType())
	  err = outbound.Marshal(w, ret)
	  if err != nil {
	    httpruntime.SetError(r.Context(),r,w,errors.Wrap(err,"couldn't write response"))
	    return
	  }
      })
      {{end}}
      {{end}}
}
{{end}}
{{end}} // base service handler ended
`))

	footerTemplate = template.Must(template.New("footer").Funcs(funcMap).Parse(`
var _swaggerDef_{{dotToUnderscore .GetName}} = []byte(` + "`" + `{{escapeBackTicks (byteStr .SwagBuffer)}}` + `
` + "`)" + `
`))

	patternsTemplate = template.Must(template.New("patterns").Parse(`
{{define "base"}}
var (
{{range $svc := .Services}}
{{range $m := $svc.Methods}}
{{range $b := $m.Bindings}}
	pattern_goclay_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}} = "{{$b.PathTmpl.Template}}"
        unmarshaler_goclay_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}} = func(r *http.Request,req *{{$m.RequestType.GetName}}) error {

        var err error
        {{if $b.Body}}
          {{template "unmbody" .}}
        {{end}}
        {{if $b.PathParams}}
          {{template "unmpath" .}}
        {{end}}

        return err
        }
{{end}}
{{end}}
{{end}}
)
{{end}}
{{define "unmbody"}}
          inbound,_ := httpruntime.MarshalerForRequest(r)
	  err = errors.Wrap(inbound.Unmarshal(r.Body,req),"couldn't read request JSON")
          if err != nil {
            return err
          }
{{end}}
{{define "unmpath"}}
	  rctx := chi.RouteContext(r.Context())
          if rctx == nil {
            panic("Only chi router is supported for GETs atm")
	  }
          for pos,k := range rctx.URLParams.Keys {
	    runtime.PopulateFieldFromPath(req, k, rctx.URLParams.Values[pos])
          }
{{end}}
`))
)
