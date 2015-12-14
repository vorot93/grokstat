package models

type PlayerEntry struct {
	Name string
	Info map[string]string
}

type ServerEntry struct {
	Host       string `json:"host"`
	Name       string `json:"host"`
	Protocol   string `json:"host"`
	NumClients string `json:"host"`
	NaxClients string `json:"host"`
	Ping       int    `json:"ping"`
	Players    []PlayerEntry
	Rules      map[string]string
}
