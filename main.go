package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

func main() {
	r := chi.NewRouter()

	// Rotas
	r.Post("/generate-vpn", generateVPNAndExecuteScript)

	// Inicia o servidor na porta configurada
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor rodando na porta %s", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
