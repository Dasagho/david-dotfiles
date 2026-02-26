package catalog

// Bin represents a single binary to symlink from the extracted archive.
type Bin struct {
	Src string `toml:"src"`
	Dst string `toml:"dst"`
}

// Program is a single installable entry from catalog.toml.
type Program struct {
	Name         string   // populated from the TOML table key
	Repo         string   `toml:"repo"`
	AssetPattern string   `toml:"asset_pattern"`
	Packages     []string `toml:"packages"`
	Bin          []Bin    `toml:"bin"`
}

// Catalog is the parsed catalog.toml.
type Catalog struct {
	Programs map[string]Program `toml:"programs"`
}
