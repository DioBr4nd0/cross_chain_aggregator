package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the API routes for the Gin engine
func SetupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		v1.GET("/chains", GetChainsHandler)
		v1.GET("/tokens", GetTokensHandler)
		v1.GET("/rates", GetRatesHandler)             // Query params: fromToken, toToken
		v1.GET("/best-route", FindBestRouteHandler) // Query params: fromToken, toToken, amountIn, fromChainID
		v1.POST("/swap", ExecuteSwapHandler)        // JSON body: ExecuteSwapRequest
	}
}
