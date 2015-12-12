package protocols

type ProtocolConfig struct {
    Id string `toml:"Id"`
    Template string `toml:"Template"`
    Overrides map[string]string `toml:"Overrides"`
}
