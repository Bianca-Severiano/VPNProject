package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/ssh"
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

	fmt.Print(req.Clients)
	endereco := fmt.Sprintf("%s:%s", req.ServerIP, req.ServerPort)

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Conecta ao servidor
	client, err := ssh.Dial("tcp", endereco, config)
	if err != nil {
		log.Fatalf("Falha ao conectar: %s", err)
	}
	defer client.Close()

	// Cria uma nova sessão SSH
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Falha ao criar sessão: %s", err)
	}
	defer session.Close()

	// Captura stdout e stderr
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Prepara o comando SSH
	cmd := fmt.Sprintf(`

export CLIENTS="%s"
export SERVER_IP="%s"
export SERVER_PORT="%s"

cat > /tmp/generate_vpn.sh << 'EOF'
#!/bin/bash

set -e  # Sai imediatamente se algum comando falhar

CLIENTS="$CLIENTS"
EASYRSA_DIR="/usr/share/easy-rsa"
OUTPUT_DIR="/home/$CLIENTS"
VPN_BASE_DIR="/etc/openvpn/client"

# Verificar se easyrsa está instalado
if [ ! -f "$EASYRSA_DIR/easyrsa" ]; then
    echo "Erro: easyrsa não encontrado em $EASYRSA_DIR" >&2
    exit 1
fi

sudo mkdir -p "$VPN_BASE_DIR"
sudo mkdir -p "$OUTPUT_DIR"

SERVER_IP="$SERVER_IP"
SERVER_PORT="$SERVER_PORT"

# Validar entrada
if [ -z "$CLIENTS" ]; then
    echo "Erro: Nome do cliente não especificado - $CLIENTS" >&2
    exit 1
fi

if [ -z "$SERVER_IP" ]; then
    echo "Erro: IP do servidor não especificado" >&2
    exit 1
fi

echo "=============================="
echo "Gerando VPN para: $CLIENTS"
echo "=============================="

cd "$EASYRSA_DIR" || exit 1

# Inicializar PKI se necessário
if [ ! -d "pki" ]; then
    ./easyrsa init-pki
    ./easyrsa build-ca nopass
    ./easyrsa gen-dh
fi

# Gerar certificados (suprimindo saída interativa)
{
    echo "Gerando certificado para $CLIENTS..."
    ./easyrsa --batch gen-req "$CLIENTS" nopass
    echo "yes" | ./easyrsa --batch sign-req client "$CLIENTS"
} || {
    echo "Erro ao gerar certificados" >&2
    exit 1
}

CLIENT_DIR="$VPN_BASE_DIR/$CLIENTS"
sudo mkdir -p "$CLIENT_DIR"

# Copiar arquivos com verificação
sudo cp pki/ca.crt "$CLIENT_DIR/" || { echo "Erro ao copiar ca.crt" >&2; exit 1; }
sudo cp "pki/issued/$CLIENTS.crt" "$CLIENT_DIR/" || { echo "Erro ao copiar $CLIENTS.crt" >&2; exit 1; }
sudo cp "pki/private/$CLIENTS.key" "$CLIENT_DIR/" || { echo "Erro ao copiar $CLIENTS.key" >&2; exit 1; }
sudo cp pki/dh.pem "$CLIENT_DIR/" 2>/dev/null || echo "Arquivo dh.pem não encontrado (opcional)"

# Criar arquivo de configuração OVPN
sudo bash -c "cat > '$CLIENT_DIR/$CLIENTS.ovpn'" <<EOCONF
client
dev tun
proto udp
remote $SERVER_IP $SERVER_PORT
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
cipher AES-256-CBC
verb 3
<ca>
$(cat pki/ca.crt)
</ca>
<cert>
$(cat pki/issued/$CLIENTS.crt)
</cert>
<key>
$(cat pki/private/$CLIENTS.key)
</key>
EOCONF

# Preparar arquivos para download
OUTPUT_CLIENT_DIR="$OUTPUT_DIR/$CLIENTS"
sudo mkdir -p "$OUTPUT_CLIENT_DIR"
sudo cp -r "$CLIENT_DIR" "$(dirname "$OUTPUT_CLIENT_DIR")"
sudo chown -R $(logname):$(logname) "$OUTPUT_CLIENT_DIR"

# Compactar e limpar
cd "$OUTPUT_DIR" || exit 1
zip -r "${CLIENTS}.zip" "$CLIENTS" || { echo "Erro ao compactar arquivos" >&2; exit 1; }
sudo rm -rf "$OUTPUT_CLIENT_DIR"

echo "VPN para $CLIENTS gerada com sucesso em:"
echo "$OUTPUT_DIR/${CLIENTS}.zip"
EOF

sudo chmod +x /tmp/generate_vpn.sh
sudo /tmp/generate_vpn.sh || { echo "Falha na execução do script" >&2; exit 1; }
sudo rm -f /tmp/generate_vpn.sh
`,
    req.Clients,
    req.ServerIP,
    req.ServerPort,
)

	// Executa o comando
	err = session.Run(cmd)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro na execução remota: %s\nDetalhes: %s", err, stderrBuf.String()), http.StatusInternalServerError)
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
