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

func RemoveDuplicates(ListA []string) []string {
	tempDict := make(map[string]bool, len(ListA))
	for _, entry := range ListA {
		tempDict[entry] = true
	}

	ListB := make([]string, 0, len(tempDict))
	for entry, _ := range tempDict {
		ListB = append(ListB, entry)
	}
	return ListB
}