package models

type PlayerEntry struct {
	Name string            `json:"name"`
	Ping int64             `json:"ping"`
	Info map[string]string `json:"info"`
}

var MakePlayerEntry = func() PlayerEntry {
	return PlayerEntry{Info: make(map[string]string)}
}

type ServerEntry struct {
	Protocol   string            `json:"protocol"`
	Status     int               `json:"status"`
	Error      error             `json:"-"`
	Message    string            `json:"message"`
	Host       string            `json:"host"`
	Name       string            `json:"name"`
	NeedPass   bool              `json:"need-pass"`
	ModName    string            `json:"modname"`
	GameType   string            `json:"gametype"`
	Terrain    string            `json:"terrain"`
	NumClients int64             `json:"numclients"`
	MaxClients int64             `json:"maxclients"`
	NumBots    int64             `json:"numbots"`
	Secure     bool              `json:"secure"`
	Ping       int64             `json:"ping"`
	Players    []PlayerEntry     `json:"players"`
	Rules      map[string]string `json:"rules"`
}

var MakeServerEntry = func() ServerEntry {
	return ServerEntry{Players: make([]PlayerEntry, 0), Rules: make(map[string]string)}
}
