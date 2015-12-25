package models

type PlayerEntry struct {
	Name string
	Ping int64
	Info map[string]string
}

type ServerEntry struct {
	Host       string `json:"host"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	NumClients int64  `json:"numclients"`
	MaxClients int64  `json:"maxclients"`
	Secure     bool   `json:"secure"`
	Ping       int64    `json:"ping"`
	Players    []PlayerEntry `json:"players"`
	Rules      map[string]string `json:"rules"`
}
