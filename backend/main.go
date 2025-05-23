package main

import (
	"fmt"
	"log"
	"net/http"
	// "os"

	"github.com/gin-gonic/gin"
	"cosmos_defi_aggregator/api"     // Adjust
	"cosmos_defi_aggregator/config"  // Adjust
	// sdk "github.com/cosmos/cosmos-sdk/types" // Not directly needed in main for this setup
	// wasmdapp "github.com/CosmWasm/wasmd/app" // Not directly needed in main for this setup
)

func main() {
	// Initialize configurations (chains, IBC channels, etc.)
	config.Init() // This will load your predefined chain and IBC channel info

	// Set up Gin router
	router := gin.Default()

	// CORS Middleware (allow all for development)
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent) // 204
			return
		}
		c.Next()
	})


	// Setup API routes
	api.SetupRoutes(router)

	// Start server
	port := "8080" // Default port, can be made configurable
	log.Printf("Starting API server on http://localhost:%s\n", port)
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
