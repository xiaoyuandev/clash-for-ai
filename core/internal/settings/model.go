package settings

type LocalGatewaySettings struct {
	Enabled    bool   `json:"enabled"`
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port"`
}

type AppSettings struct {
	LocalGateway LocalGatewaySettings `json:"local_gateway"`
}
