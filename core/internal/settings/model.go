package settings

type LocalGatewaySettings struct {
	Enabled    bool   `json:"enabled"`
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port"`
}

type SelectedModel struct {
	ModelID  string `json:"model_id"`
	Position int    `json:"position"`
}

type AppSettings struct {
	LocalGateway         LocalGatewaySettings `json:"local_gateway"`
	LocalGatewaySelected []SelectedModel      `json:"local_gateway_selected_models"`
}
