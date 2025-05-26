package api

import "github.com/gin-gonic/gin"

func SetupRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	v1 := router.Group("/api/v1")
	{
		v1.GET("/chains", GetChainsHandler)
		v1.GET("/tokens", GetTokensHandler)
		v1.GET("/rates", GetRatesHandler)
		v1.GET("/best-route", FindBestRouteHandler)
		v1.POST("/swap", ExecuteSwapHandler)
	}
}
