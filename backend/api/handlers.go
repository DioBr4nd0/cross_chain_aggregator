package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"cosmos_defi_aggregator/models"
	"cosmos_defi_aggregator/services"
	"cosmos_defi_aggregator/config" // For GetChains for /chains endpoint
)

func GetChainsHandler(c *gin.Context) {
	var apiChains []config.ChainConfig // Expose configured chains
	for _, chainCfg := range config.GlobalAppConfig.Chains {
		apiChains = append(apiChains, chainCfg)
	}
	c.JSON(http.StatusOK, apiChains)
}

func GetTokensHandler(c *gin.Context) { // Simplified
	tokenMap := make(map[string]bool)
	for _, chain := range config.GlobalAppConfig.Chains {
		tokenMap[chain.FeeDenom] = true // Include native/fee denoms
		for _, t := range chain.SupportedTokens {
			tokenMap[t] = true
		}
	}
	var tokens []string
	for token := range tokenMap {
		tokens = append(tokens, token)
	}
	c.JSON(http.StatusOK, gin.H{"supportedTokens": tokens})
}


func GetRatesHandler(c *gin.Context) {
	fromToken := c.Query("fromToken")
	toToken := c.Query("toToken")
	if fromToken == "" || toToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fromToken and toToken query parameters are required"})
		return
	}
	rates, err := services.GetRatesForPairAcrossChains(c.Request.Context(), fromToken, toToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rates)
}

func FindBestRouteHandler(c *gin.Context) {
	var req models.BestRouteRequest
	// For GET requests, bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}
	if req.FromToken == "" || req.ToToken == "" || req.AmountIn == "" || req.FromChainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fromToken, toToken, amountIn, and fromChainID are required"})
		return
	}
	routeResponse, err := services.FindBestSwapRoute(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find best route: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, routeResponse)
}

func ExecuteSwapHandler(c *gin.Context) {
	var req models.ExecuteSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
    if req.FromToken == "" || req.ToToken == "" || req.AmountIn == "" || req.FromChainID == "" || req.UserAddress == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "fromToken, toToken, amountIn, fromChainID, and userAddress are required"})
        return
    }

	log.Printf("API: Received swap request: %+v", req)
	result, err := services.ProcessSwapRequest(c.Request.Context(), req)
	if err != nil {
		// ProcessSwapRequest now includes txhash in result even on error for broadcast attempts
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Swap execution failed: " + err.Error(), "details": result})
		return
	}
	c.JSON(http.StatusOK, result)
}
