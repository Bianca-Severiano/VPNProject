package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

// Handler para gerar VPN e executar o script
func generateVPNAndExecuteScript(w http.ResponseWriter, r *http.Request) {
	// Parse do corpo da requisição
	var req VPNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Erro no JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validação básica
	if len(req.Clients) == 0 {
		http.Error(w, "Pelo menos um cliente deve ser fornecido", http.StatusBadRequest)
		return
	}

	usuario := fmt.Sprintf("usuario@%s", req.ServerIP)
	senha := fmt.Sprintf("echo %s", req.Password)
	// Prepara o comando SSH
	cmd := exec.Command("sshpass", "-p", senha,
	"ssh", usuario,
	"-o", "StrictHostKeyChecking=no",
	"sudo", "bash", "-c",
		fmt.Sprintf(`	
			export CLIENTS="%s"
			export SERVER_IP="%s"
			export SERVER_PORT="%s"

			cat > /tmp/generate_vpn.sh << 'EOF'
			#!/bin/bash

			if [ "$EUID" -ne 0 ]; then
				echo "Por favor, execute como root"
				exit 1
			fi

			EASYRSA_DIR="/usr/share/easy-rsa"
			OUTPUT_DIR="/home/$(logname)"
			VPN_BASE_DIR="/etc/openvpn/client"

			mkdir -p "$VPN_BASE_DIR"
			mkdir -p "$OUTPUT_DIR"

			IFS=',' read -r -a CLIENTES <<< "$CLIENTS"
			SERVER_IP="$SERVER_IP"
			SERVER_PORT="$SERVER_PORT"

			for CLIENTE in "${CLIENTES[@]}"; do
				echo "=============================="
				echo "Gerando VPN para: $CLIENTE"
				echo "=============================="

				cd "$EASYRSA_DIR" || exit 1
				./easyrsa gen-req "$CLIENTE" nopass
				echo "yes" | ./easyrsa sign-req client "$CLIENTE"

				CLIENT_DIR="$VPN_BASE_DIR/$CLIENTE"
				mkdir -p "$CLIENT_DIR"

				cp pki/ca.crt "$CLIENT_DIR/"
				cp "pki/issued/$CLIENTE.crt" "$CLIENT_DIR/"
				cp "pki/private/$CLIENTE.key" "$CLIENT_DIR/"
				cp pki/dh.pem "$CLIENT_DIR/" 2>/dev/null || echo "Arquivo dh.pem não encontrado (opcional)"

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

				OUTPUT_CLIENT_DIR="$OUTPUT_DIR/$CLIENTE"
				cp -r "$CLIENT_DIR" "$OUTPUT_CLIENT_DIR"
				chown -R $(logname):$(logname) "$OUTPUT_CLIENT_DIR"

				cd "$OUTPUT_DIR" || exit 1
				zip -r "${CLIENTE}.zip" "$CLIENTE"
				rm -rf "$OUTPUT_CLIENT_DIR"

				echo "VPN para $CLIENTE gerada em $OUTPUT_DIR/${CLIENTE}.zip"
			done

			echo "Processo concluído para todos os clientes."
			EOF

			chmod +x /tmp/generate_vpn.sh
			sudo /tmp/generate_vpn.sh
			rm -f /tmp/generate_vpn.sh
		`,
			strings.Join(req.Clients, ","),
			req.ServerIP,
			req.ServerPort,
		),
	)

	// Executa o comando
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		log.Printf("Erro ao executar o comando: %v\nSaída: %s", err, string(output))
		http.Error(w, fmt.Sprintf("Erro ao executar o script na VM: %v\nSaída: %s", err, string(output)), http.StatusInternalServerError)
		return
	}

	// Resposta de sucesso
	response := VPNResponse{
		Message: "Certificados gerados com sucesso",
		Clients: req.Clients,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}