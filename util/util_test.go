package util

import (
	"testing"

	"github.com/grokstat/grokstat/grokstaterrors"
)

func TestByteLEToInt64(t *testing.T) {
	var err error
	s1 := []byte{0x60, 0x09}
	expectation := int64(2400)

	result := ByteLEToInt64(s1)

	if result != expectation {
		err = grokstaterrors.CompError
	}

	if err != nil {
		t.Errorf(ErrorOut(expectation, result))
	}
}
