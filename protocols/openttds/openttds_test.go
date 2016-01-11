package openttds

import (
	"fmt"
	"testing"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func TestParseResponseMap(t *testing.T) {
	var err error
	s1 := map[string]models.Packet{"info": models.Packet{Id: "info", Data: []byte("\x86\x00\x01\x04\x03\x4D\x47\x03\x05\x2E\x96\xB9\xAB\x2B\xEA\x68\x6B\xFF\x94\x96\x1A\xD4\x33\xA7\x01\x32\x32\x33\x22\x31\x61\x80\xDA\x1B\xA6\x44\x4A\x06\xCD\x17\xF8\xFA\x79\xD6\x0A\x44\x4E\x07\x00\x48\xB3\xF9\xE4\xFD\x0D\xF2\xA7\x2B\x5F\x44\xD3\xC8\xA2\xF4\xA0\x63\xEC\x0A\x00\x63\xEC\x0A\x00\x0F\x00\x0A\x4F\x6E\x6C\x79\x46\x72\x69\x65\x6E\x64\x73\x20\x4F\x70\x65\x6E\x54\x54\x44\x20\x53\x65\x72\x76\x65\x72\x20\x23\x31\x00\x31\x2E\x35\x2E\x33\x00\x16\x00\x19\x00\x00\x52\x61\x6E\x64\x6F\x6D\x20\x4D\x61\x70\x00\x00\x04\x00\x04\x01\x01")}}
	s2 := make(map[string]string)
	expectation := models.ServerEntry{Name: "OnlyFriends OpenTTD Server #1", Terrain: "Random Map", NumClients: int64(0), MaxClients: int64(25), NeedPass: false, Players: []models.PlayerEntry{}, Rules: map[string]string{"protocol-version": "4", "active-newgrfs-num": "3", "active-newgrfs": "ID:4d470305/MD5:2e96b9ab2bea686bff94961ad433a701; ID:32323322/MD5:316180da1ba6444a06cd17f8fa79d60a; ID:444e0700/MD5:48b3f9e4fd0df2a72b5f44d3c8a2f4a0", "time-current": "1676413440", "time-start": "1676413440", "max-companies": "15", "current-companies": "0", "max-spectators": "10", "server-name": "OnlyFriends OpenTTD Server #1", "server-version": "1.5.3", "language-id": "22", "need-pass": "false", "max-clients": "25", "current-clients": "0", "current-spectators": "0", "map-name": "Random Map", "map-set": "1", "dedicated": "1"}}

	result, resultErr := ParseResponseMap(s1, s2)

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
