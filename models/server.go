package models

type PlayerEntry struct {
	Name string
	Ping int64
	Info map[string]string
}

type ServerEntry struct {
	Protocol   string            `json:"protocol"`
	Status     int               `json:"status"`
	Message    string            `json:"message"`
	Host       string            `json:"host"`
	Name       string            `json:"name"`
	ModName    string            `json:"modname"`
	GameType   string            `json:"gametype"`
	Terrain    string            `json:"terrain"`
	NumClients int64             `json:"numclients"`
	MaxClients int64             `json:"maxclients"`
	Secure     bool              `json:"secure"`
	Ping       int64             `json:"ping"`
	Players    []PlayerEntry     `json:"players"`
	Rules      map[string]string `json:"rules"`
}
