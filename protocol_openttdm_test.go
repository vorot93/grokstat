package main

import "testing"

func TestOPENTTDMparseData(t *testing.T) {
	var err error
	var s1 = Packet{Id: "servers4", Data: []byte("\x42\x00\x07\x01\x0A\x00\x4A\xD0\x4B\xB7\x8B\x0F\xAC\xF9\xB0\x91\x8B\x0F\x53\xC7\x18\x16\x8B\x0F\x3E\x8F\x2E\x44\x8B\x0F\x79\x2A\xA0\x97\x3E\x0F\x5C\xDE\x6E\x7C\x8B\x0F\x6C\x34\xE4\x4C\x8B\x0F\xB2\xEB\xB2\x57\x8B\x0F\x80\x48\x4A\x71\x8B\x0F\x40\x8A\xE7\x36\x8B\x0F\x42\x00\x07\x01\x01\x00\x4A\xD0\x4B\xB7\x8C\x0F")}
	expectation := []string{"74.208.75.183:3979", "172.249.176.145:3979", "83.199.24.22:3979", "62.143.46.68:3979", "121.42.160.151:3902", "92.222.110.124:3979", "108.52.228.76:3979", "178.235.178.87:3979", "128.72.74.113:3979", "64.138.231.54:3979", "74.208.75.183:3980"}

	result, resultErr := OPENTTDMparseData(s1.Data)

	if resultErr != nil {
		t.Errorf(resultErr.Error())
	}

	if len(result) != len(expectation) {
		err = CompError
	} else {
		for i := range result {
			if result[i] != expectation[i] {
				err = CompError
				break
			}
		}
	}

	if err != nil {
		t.Errorf(ErrorOut(expectation, result))
	}
}
