package main

import (
	"context"
	"log"
	"net/http" // For http.StatusNoContent

	"github.com/gin-gonic/gin"
	"cosmos_defi_aggregator/api"
	"cosmos_defi_aggregator/config"
	"cosmos_defi_aggregator/cosmos"
)

func main() {
	// 1. Initialize Application Configuration (Chains, IBC, etc.)
	config.InitAppConfig()

	// 2. Initialize Ignite Cosmos Clients for all configured chains
	ctx := context.Background() // Use a background context for initialization
	if err := cosmos.InitializeIgniteClients(ctx); err != nil {
		log.Fatalf("CRITICAL: Failed to initialize Ignite Cosmos clients: %v", err)
	}
	log.Println("All Ignite Cosmos clients initialized successfully.")

	// 3. Set up Gin HTTP Server
	router := gin.Default()

	// Basic CORS Middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins for dev
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// 4. Setup API Routes
	api.SetupRoutes(router)

	// 5. Start Server
	port := "8080" // Make configurable if needed
	log.Printf("🚀 Aggregator API server starting on http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}
