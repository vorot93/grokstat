package mumbles

import (
	"fmt"
	"testing"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func TestParsePacket(t *testing.T) {
	var err error
	s1 := models.Packet{Id: "ping", Data: []byte("\x00\x01\x02\x05\x67\x72\x6F\x6B\x73\x74\x61\x74\x00\x00\x00\x02\x00\x00\x02\x00\x00\x01\x19\x40")}
	s2 := make(map[string]string)
	expectation := models.ServerEntry{NumClients: int64(2), MaxClients: int64(512), NeedPass: false, Players: []models.PlayerEntry{}, Rules: map[string]string{"protocol-version": "1.2.5", "current-clients": "2", "max-clients": "512", "max-bandwidth": "72000", "challenge": "grokstat"}}

	result, resultErr := parsePacket(s1, s2)

	if resultErr != nil {
		t.Errorf(resultErr.Error())
	}

	if len(result.Rules) != len(expectation.Rules) {
		err = grokstaterrors.CompError
	}

	for i, _ := range result.Rules {
		if result.Rules[i] != expectation.Rules[i] {
			err = grokstaterrors.CompError
		}
	}

	if err != nil {
		fmt.Println(util.MapComparison(expectation.Rules, result.Rules))
		t.Errorf(util.ErrorOut(expectation.Rules, result.Rules))
	}
}
