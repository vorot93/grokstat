package protocols

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/protocoltemplates"
	"github.com/imdario/mergo"
)

// Returns a map with protocols initialized
func MakeProtocolMap(configData []ProtocolConfig) map[string]models.ProtocolEntry {
	templates := make(map[string]models.ProtocolEntry)
	infoBase := models.ProtocolEntryInfo{`x20`: "\x20", `xFF`: "\xFF"}
	templates["Q3M"] = protocoltemplates.Q3Mtemplate
	templates["Q3S"] = protocoltemplates.Q3Stemplate
	templates["TEEWORLDSM"] = protocoltemplates.TEEWORLDSMtemplate
	templates["TEEWORLDSS"] = protocoltemplates.TEEWORLDSStemplate
	templates["OPENTTDM"] = protocoltemplates.OPENTTDMtemplate
	templates["OPENTTDS"] = protocoltemplates.OPENTTDStemplate

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

		protocolMap[entryId] = protocolEntry
	}

	return protocolMap
}
