package rollkit

type CelestiaResponse struct {
	Result CelestiaBlockResult `json:"result"`
}

type CelestiaBlockResult struct {
	Block CelestiaBlock `json:"block"`
}

type CelestiaBlock struct {
	Header CelestiaBlockHeader `json:"header"`
}

type CelestiaBlockHeader struct {
	Height string `json:"height"`
}

type PrivValidatorKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type PrivValidatorKeyFile struct {
	Address string           `json:"address"`
	PubKey  PrivValidatorKey `json:"pub_key"`
	PrivKey PrivValidatorKey `json:"priv_key"`
}
