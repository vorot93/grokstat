package util

import (
    "bytes"
    "text/template"
)

func ParseTemplate(templ string, keys map[string]string) string {
    buf := new(bytes.Buffer)
	t, _ := template.New("New template").Parse(templ)
	t.Execute(buf, keys)
	return buf.String()
}
