package conf

// conf js action param collection
type Conf struct {
	MetaFile string `env:"METAFILE"`
	WorkDir  string `env:"WORKDIR"`
	// user action params
	Assert []Assert `env:"ACTION_ASSERTS"`
}

type Assert struct {
	Value       string `json:"value"`
	Assert      string `json:"assert"`
	ActualValue string `json:"actualValue"`
}
