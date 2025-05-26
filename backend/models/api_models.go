package models

// BestRouteRequest defines the structure for finding the best swap route.
type BestRouteRequest struct {
	FromToken   string `json:"fromToken" binding:"required"`
	ToToken     string `json:"toToken" binding:"required"`
	AmountIn    string `json:"amountIn" binding:"required"` // String to handle large numbers precisely
	FromChainID string `json:"fromChainID" binding:"required"`
}

// RouteDetail describes a single step or a full path for a swap.
type RouteDetail struct {
	ChainIDSwappingOn string   `json:"chainIDSwappingOn"`
	Rate              string   `json:"rate"`      // Effective rate for this step/path
	AmountOut         string   `json:"amountOut"` // Estimated output for this step/path
	Steps             []string `json:"steps"`     // Human-readable steps
	NeedsIBC          bool     `json:"needsIBC"`
}

// BestRouteResponse is the API response for the best route.
type BestRouteResponse struct {
	FromToken   string        `json:"fromToken"`
	ToToken     string        `json:"toToken"`
	AmountIn    string        `json:"amountIn"`
	BestRoute   RouteDetail   `json:"bestRoute"`
	OtherRoutes []RouteDetail `json:"otherRoutes,omitempty"`
}

// ExecuteSwapRequest defines the structure for executing a swap.
type ExecuteSwapRequest struct {
	FromToken       string `json:"fromToken" binding:"required"`
	ToToken         string `json:"toToken" binding:"required"`
	AmountIn        string `json:"amountIn" binding:"required"`
	FromChainID     string `json:"fromChainID" binding:"required"`       // Chain where user's FromToken currently is
	UserAddress     string `json:"userAddress" binding:"required"`       // User's address (for now, this will map to operator key)
	RecipientAddress string `json:"recipientAddress,omitempty"` // Optional: final recipient on target chain
}

// ExecuteSwapResponse is the API response for a swap execution attempt.
type ExecuteSwapResponse struct {
	Status         string `json:"status"` // e.g., "submitted", "ibc_initiated", "completed", "failed"
	Message        string `json:"message"`
	IbcTxHash      string `json:"ibcTxHash,omitempty"`
	SwapTxHash     string `json:"swapTxHash,omitempty"`
	FinalAmountOut string `json:"finalAmountOut,omitempty"` // If known
}

// DexRateInfo holds rate information from a specific DEX.
type DexRateInfo struct {
	ChainID   string `json:"chainID"`
	FromToken string `json:"fromToken"`
	ToToken   string `json:"toToken"`
	Rate      string `json:"rate"` // Exchange rate as string
}
