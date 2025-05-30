package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Configuração das rotas
	r.POST("/generate-vpn", generateVPNAndExecuteScript)

	// Inicia o servidor na porta 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor rodando na porta %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
