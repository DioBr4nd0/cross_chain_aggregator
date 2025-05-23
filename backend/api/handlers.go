package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"cosmos_defi_aggregator/models"   // Adjust
	"cosmos_defi_aggregator/services" // Adjust
	"cosmos_defi_aggregator/config"
)

// GetChainsHandler returns a list of configured chains
func GetChainsHandler(c *gin.Context) {
	// In a real app, this might come from config or a dynamic discovery
	// For now, using the config.GlobalChains directly.
	// We need to filter what we expose via API from the internal config.Chain struct
	type APIChainInfo struct {
		ID string `json:"id"`
		Name string `json:"name"`
		NativeToken string `json:"nativeToken"`
		SupportedTokens []string `json:"supportedTokens"`
	}
	var apiChains []APIChainInfo
	for _, chain := range config.GlobalChains {
		apiChains = append(apiChains, APIChainInfo{
			ID: chain.ID,
			Name: chain.Name,
			NativeToken: chain.NativeToken,
			SupportedTokens: chain.SupportedTokens,
		})
	}
	c.JSON(http.StatusOK, apiChains)
}


// GetTokensHandler returns a list of all unique tokens supported across chains
func GetTokensHandler(c *gin.Context) {
	tokenMap := make(map[string]bool)
	for _, chain := range config.GlobalChains {
		tokenMap[chain.NativeToken] = true
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

// GetRatesHandler returns available direct rates for a token pair across all chains
func GetRatesHandler(c *gin.Context) {
	fromToken := c.Query("fromToken")
	toToken := c.Query("toToken")

	if fromToken == "" || toToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fromToken and toToken query parameters are required"})
		return
	}

	rates, err := services.GetRatesForPairAcrossChains(fromToken, toToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rates)
}


// FindBestRouteHandler handles requests to find the best swap route
func FindBestRouteHandler(c *gin.Context) {
	var req models.BestRouteRequest
	if err := c.ShouldBindQuery(&req); err != nil { // Use ShouldBindQuery for GET params
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}
	// Basic validation
	if req.FromToken == "" || req.ToToken == "" || req.AmountIn == "" || req.FromChainID == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "fromToken, toToken, amountIn, and fromChainID are required"})
        return
	}


	routeResponse, err := services.FindBestRouteForSwap(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find best route: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, routeResponse)
}

// ExecuteSwapHandler handles requests to execute a swap
func ExecuteSwapHandler(c *gin.Context) {
	var req models.ExecuteSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Basic validation
    if req.FromToken == "" || req.ToToken == "" || req.AmountIn == "" || req.FromChainID == "" || req.SenderAddress == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "fromToken, toToken, amountIn, fromChainID, and senderAddress are required"})
        return
    }

	result, err := services.ExecuteFullSwapRoute(req)
	if err != nil {
		// Determine appropriate status code based on error type if needed
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Swap execution failed: " + err.Error(), "details": result})
		return
	}
	c.JSON(http.StatusOK, result)
}
