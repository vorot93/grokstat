package protocols

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/a2s"
	"github.com/grokstat/grokstat/protocols/openttdm"
	"github.com/grokstat/grokstat/protocols/openttds"
	"github.com/grokstat/grokstat/protocols/q3m"
	"github.com/grokstat/grokstat/protocols/q3s"
	"github.com/grokstat/grokstat/protocols/steam"
	"github.com/grokstat/grokstat/protocols/teeworldsm"
	"github.com/grokstat/grokstat/protocols/teeworldss"
	"github.com/imdario/mergo"
)

// Returns a map with protocols initialized
func LoadProtocolCollection(configData []ProtocolConfig) models.ProtocolCollection {
	templates := make(map[string]models.ProtocolEntry)
	infoBase := models.ProtocolEntryInfo{`x20`: "\x20", `xFF`: "\xFF"}
	templates["Q3M"] = q3m.ProtocolTemplate
	templates["Q3S"] = q3s.ProtocolTemplate
	templates["TEEWORLDSM"] = teeworldsm.ProtocolTemplate
	templates["TEEWORLDSS"] = teeworldss.ProtocolTemplate
	templates["OPENTTDM"] = openttdm.ProtocolTemplate
	templates["OPENTTDS"] = openttds.ProtocolTemplate
	templates["STEAM"] = steam.ProtocolTemplate
	templates["A2S"] = a2s.ProtocolTemplate

	for k, _ := range templates {
		entry := templates[k]
		mergo.Merge(&entry.Information, infoBase)
		templates[k] = entry
	}

	protocolMap := make(map[string]models.ProtocolEntry)

	for _, configEntry := range configData {
		entryId := configEntry.Id
		templateId := configEntry.Template
		overrides := configEntry.Overrides

		entryTemplate, eOk := templates[templateId]
		if eOk == false {
			continue
		}
		protocolEntry := models.MakeProtocolEntry(entryTemplate)
		for k, v := range overrides {
			protocolEntry.Information[k] = v
		}
		protocolEntry.Id = entryId
		protocolEntry.Information["Id"] = entryId

		protocolMap[entryId] = protocolEntry
	}

	protColl := models.MakeSharedProtocolCollection()

	for _, entry := range protocolMap {
		protColl.AddEntry(entry)
	}

	return protColl
}
