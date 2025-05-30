package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

// generateVPNAndExecuteScript lida com a requisição para gerar certificados VPN
func generateVPNAndExecuteScript(c *gin.Context) {
	var req VPNRequest

	// Faz o bind do JSON para a estrutura VPNRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Valida se há pelo menos um cliente
	if len(req.Clients) == 0 {
		c.JSON(400, gin.H{"error": "pelo menos um cliente deve ser fornecido"})
		return
	}

	// Prepara o comando para executar o script na VM
	// Nota: Você precisará configurar a autenticação adequada (como chaves SSH)
	// Este é um exemplo básico que precisa ser adaptado para seu ambiente
	cmd := exec.Command("ssh", "usuario@vm-endereco-ip", 
		"sudo", "bash", "-c", 
		fmt.Sprintf(`
			export CLIENTS="%s"
			export SERVER_IP="%s"
			export SERVER_PORT="%s"
			cat > /tmp/generate_vpn.sh << 'EOF'
			#!/bin/bash
			# Verifica se o usuário é root
			if [ "$EUID" -ne 0 ]; then
				echo "Por favor, execute como root"
				exit 1
			fi

			# Caminhos
			EASYRSA_DIR="/usr/share/easy-rsa"
			OUTPUT_DIR="/home/$(logname)"
			VPN_BASE_DIR="/etc/openvpn/client"

			# Cria diretórios, se não existirem
			mkdir -p "$VPN_BASE_DIR"
			mkdir -p "$OUTPUT_DIR"

			# Usa as variáveis de ambiente para clientes, IP e porta
			IFS=',' read -r -a CLIENTES <<< "$CLIENTS"
			SERVER_IP="$SERVER_IP"
			SERVER_PORT="$SERVER_PORT"

			# Loop para cada cliente
			for CLIENTE in "${CLIENTES[@]}"; do
				echo "=============================="
				echo "Gerando VPN para: $CLIENTE"
				echo "=============================="

				# Geração da chave e solicitação de certificado
				cd "$EASYRSA_DIR" || exit 1
				./easyrsa gen-req "$CLIENTE" nopass

				# Assinatura do certificado
				echo "yes" | ./easyrsa sign-req client "$CLIENTE"

				# Criação do diretório do cliente
				CLIENT_DIR="$VPN_BASE_DIR/$CLIENTE"
				mkdir -p "$CLIENT_DIR"

				# Cópia dos arquivos necessários
				cp pki/ca.crt "$CLIENT_DIR/"
				cp "pki/issued/$CLIENTE.crt" "$CLIENT_DIR/"
				cp "pki/private/$CLIENTE.key" "$CLIENT_DIR/"
				cp pki/dh.pem "$CLIENT_DIR/" 2>/dev/null || echo "Arquivo dh.pem não encontrado (opcional)"

				# Cria o arquivo de configuração .ovpn
				cat <<EOCONF > "$CLIENT_DIR/$CLIENTE.ovpn"
				client
				dev tun
				proto udp
				remote $SERVER_IP $SERVER_PORT
				ca ca.crt
				cert $CLIENTE.crt
				key $CLIENTE.key
				tls-client
				resolv-retry infinite
				nobind
				persist-key
				persist-tun
				EOCONF

				# Prepara o diretório de saída
				OUTPUT_CLIENT_DIR="$OUTPUT_DIR/$CLIENTE"
				cp -r "$CLIENT_DIR" "$OUTPUT_CLIENT_DIR"

				# Ajusta permissões
				chown -R $(logname):$(logname) "$OUTPUT_CLIENT_DIR"

				# Compacta em ZIP
				cd "$OUTPUT_DIR" || exit 1
				zip -r "${CLIENTE}.zip" "$CLIENTE"

				# Limpa diretório temporário de cliente
				rm -rf "$OUTPUT_CLIENT_DIR"
				echo "VPN para $CLIENTE gerada em $OUTPUT_DIR/${CLIENTE}.zip"
			done
				echo "Processo concluído para todos os clientes."
			EOF

			chmod +x /tmp/generate_vpn.sh
			sudo /tmp/generate_vpn.sh
			rm -f /tmp/generate_vpn.sh
		`),
		strings.Join(req.Clients, ","),
		req.ServerIP,
		req.ServerPort,
	)

	// Executa o comando
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Erro ao executar o comando: %v\nSaída: %s", err, string(output))
		c.JSON(500, gin.H{"error": "erro ao executar o script na VM", "details": string(output)})
		return
	}

	// Resposta de sucesso
	response := VPNResponse{
		Message: "Certificados gerados com sucesso",
		Clients: req.Clients,
	}

	c.JSON(200, response)
}
