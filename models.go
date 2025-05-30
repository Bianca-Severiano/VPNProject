package main

// VPNRequest representa a estrutura da requisição para gerar certificados VPN
type VPNRequest struct {
	Clients    []string `json:"clients" binding:"required"`
	ServerIP   string   `json:"server_ip" binding:"required"`
	ServerPort string   `json:"server_port" binding:"required"`
}

// VPNResponse representa a resposta da API
type VPNResponse struct {
	Message string   `json:"message"`
	Clients []string `json:"clients"`
}
