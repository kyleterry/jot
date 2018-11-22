package server

import (
	"bytes"
	"html/template"
	"net/http"
)

const indexTemplate = `Jot version {{ .Version }}

Usage: 
  Below examples use the curl command with -i set so we can see the headers.

  Creating a jot:
    Request:
      curl -i --data-binary @textfile.txt {{ .Host }}/
    Response:
      HTTP/1.1 201 Created
      Jot-Password: PE4VtqnNjrK3C07
      Date: Sat, 30 Jun 2018 19:09:03 GMT
      Content-Length: 32
      Content-Type: text/plain; charset=utf-8

      {{ .Host }}/LIU_JPnHp

  Getting a jot:
    Request:
      curl -i {{ .Host }}/LIU_JPnHp
	Response:
	  HTTP/1.1 200 OK
      Content-Type: text/plain; charset=utf-8
      Etag: 2018-06-30T19:09:03.735647737-07:00
      Date: Sat, 30 Jun 2018 19:10:13 GMT
      Content-Length: 38

	  here is my content from textfile.txt!

  Editing a jot:
    Request:
	  curl -i -H "If-Match: 2018-06-30T19:09:03.735647737-07:00" --data-binary @updated.txt {{ .Host }}/LIU_JPnHp?password=PE4VtqnNjrK3C07
    Response:
      HTTP/1.1 303 See Other
      Location: /LIU_JPnHp
      Date: Sat, 30 Jun 2018 19:14:26 GMT
      Content-Length: 0

Make note of the Jot-Password header as that's the password used to edit
your jot. ETag can be used in conjunction with If-None-Match and If-Match
for caching and collision prevention on PUT.

Source code: https://github.com/kyleterry/jot
`

// IndexTemplateContext is the data context to render the template with for the
// index response.
type IndexTemplateContext struct {
	Version string
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
