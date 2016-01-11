package util

import (
	"bytes"
	"fmt"
	"os"
	"sort"
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

func ConvertByteArray32to8(byteArray []byte) []byte {
	newByteArray := make([]byte, len(byteArray)*2)
	i := 0
	for _, v := range byteArray {
		newByteArray[i] = v / 16
		i += 1
		newByteArray[i] = v - (v * (v / 16))
		i += 1
	}
	return newByteArray
}

func Clamp(value, min, max int) int {
	clamp := sort.IntSlice([]int{value, min, max})
	clamp.Sort()
	return clamp[1]
}

func GetByteString(byteArray []byte) string {
	return fmt.Sprintf("%x", byteArray)
}

func Print(enabled bool, data interface{}) {
	if enabled != true {
		return
	}
	fmt.Print(data)
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

func ErrorOut(expectation interface{}, result interface{}) string {
	return fmt.Sprintf("\nExpected:\n%+v\n\nReceived:\n%+v\n", expectation, result)
}

func MapComparison(A, B map[string]string) (length, keys string) {
	length = fmt.Sprintf("Length:\nA:%d\nB:%d\n\n", len(A), len(B))
	for k, _ := range A {
		keys = keys + fmt.Sprintf("\nKey: %s\nA: %s\nB: %s\n", k, A[k], B[k])
	}
	return length, keys
}
