package models

import (
	"testing"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/util"
)

func TestProtocolCollectionFindById(t *testing.T) {
	var err error
	s1 := MakeProtocolCollection()
	s1.AddEntry(ProtocolEntry{Id: "q3s"})
	s1.AddEntry(ProtocolEntry{Id: "teeworldss"})
	expA, expB := ProtocolEntry{Id: "q3s"}, true

	resA, resB := s1.FindById("q3s")

	if resA.Id != expA.Id || resB != expB {
		err = grokstaterrors.CompError
	}

	if err != nil {
		t.Errorf(util.ErrorOut(expA, resA))
	}
}
