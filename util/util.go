package util

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
	"time"
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

func PrintWait(enabled bool, interval int, active chan struct{}) {
	if enabled != true {
		return
	}
	for {
		select {
		default:
			os.Stdout.Write([]byte("."))
			os.Stdout.Sync()
			time.Sleep(time.Duration(interval) * time.Millisecond)
		case <-active:
			return
		}
	}
}

func PrintReplace(enabled bool, data string, replaceLength int) (newLength int) {
	if enabled != true {
		return
	}
	for i := 0; i < replaceLength; i++ {
		fmt.Print("\033[D")
	}
	fmt.Print(data)
	newLength = len(data)
	return newLength
}

func PrintEmptyLine(enabled bool) {
	if enabled != true {
		return
	}
	fmt.Print("\n")
}
