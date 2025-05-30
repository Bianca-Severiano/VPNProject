# Gerador de Certificados OpenVPN

Este serviço fornece uma API para gerar certificados OpenVPN em uma VM remota.

## Requisitos

- Go 1.21 ou superior
- Acesso SSH à VM onde o Easy-RSA está instalado
- Permissões de superusuário (sudo) na VM

## Configuração

1. Clone o repositório:
   ```bash
   git clone <url-do-repositorio>
   cd vpn-cert-generator
   ```

2. Instale as dependências:
   ```bash
   go mod tidy
   ```

3. Configure as variáveis de ambiente (opcional):
   ```bash
   export PORT=8080
   ```

## Uso

1. Inicie o servidor:
   ```bash
   go run .
   ```

2. Envie uma requisição POST para gerar os certificados:
   ```bash
   curl -X POST http://localhost:8080/generate-vpn \
     -H "Content-Type: application/json" \
     -d '{
       "clients": ["cliente1", "cliente2"],
       "server_ip": "10.68.76.165",
       "server_port": "1194"
     }'
   ```

## Estrutura da Requisição

```json
{
  "clients": ["usuario1", "usuario2"],
  "server_ip": "10.68.76.165",
  "server_port": "1194"
}
```

## Configuração da VM

A VM remota deve ter:
- OpenVPN instalado
- Easy-RSA configurado
- Acesso SSH configurado com chaves
- Permissões de sudo sem senha para o usuário que executará o script

## Segurança

- Configure autenticação por chave SSH
- Use HTTPS para a API em produção
- Reforce autenticação na API
- Mantenha o servidor e as dependências atualizadas
