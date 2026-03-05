package server

import (
	"bytes"
	_ "embed"
	"net/http"
	"text/template"
)

//go:embed index.txt
var indexTemplate string

// IndexTemplateContext is the data context to render the template with for the
// index response.
type IndexTemplateContext struct {
	Version string
	Commit  string
	Host    string
}

func render(w http.ResponseWriter, content string, tplCtx interface{}) error {
	tpl, err := template.New("").Parse(content)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	if err := tpl.Execute(buf, tplCtx); err != nil {
		return err
	}

	buf.WriteTo(w)

	return nil
}
