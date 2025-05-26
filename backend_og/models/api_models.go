package models

// Request for finding the best swap route
type BestRouteRequest struct {
	FromToken string `json:"fromToken" binding:"required"` // e.g., "uwasma"
	ToToken   string `json:"toToken" binding:"required"`   // e.g., "tokenb"
	AmountIn  string `json:"amountIn" binding:"required"`  // e.g., "1000000" (as string to handle large numbers)
	FromChainID string `json:"fromChainID"` // Optional: if token exists on multiple chains, specify source
}

// Response for the best swap route
type BestRouteResponse struct {
	FromToken    string            `json:"fromToken"`
	ToToken      string            `json:"toToken"`
	AmountIn     string            `json:"amountIn"`
	BestRoute    RouteDetail       `json:"bestRoute"`
	OtherRoutes  []RouteDetail     `json:"otherRoutes,omitempty"`
}

// Detailed information about a potential route
type RouteDetail struct {
	ChainIDSwappingOn string   `json:"chainIDSwappingOn"` // The chain where the final swap happens
	Rate              string   `json:"rate"`              // Exchange rate as string
	AmountOut         string   `json:"amountOut"`         // Estimated amount out as string
	Steps             []string `json:"steps"`             // e.g., ["IBC chain-a to chain-b", "Swap uwasma for tokenb on chain-b DEX"]
	NeedsIBC          bool     `json:"needsIBC"`
}


// Request to execute a swap
type ExecuteSwapRequest struct {
	FromToken       string   `json:"fromToken" binding:"required"`
	ToToken         string   `json:"toToken" binding:"required"`
	AmountIn        string   `json:"amountIn" binding:"required"`
	FromChainID     string   `json:"fromChainID" binding:"required"` // Chain where user's FromToken currently resides
	SenderAddress   string   `json:"senderAddress" binding:"required"` // User's address on FromChainID
	RecipientAddress string  `json:"recipientAddress"` // Optional: defaults to SenderAddress if empty, on the ToToken's final chain
	Route           []string `json:"routeToExecute"` // Optional: if user wants to specify a pre-calculated route (not used yet)
}

// Response for swap execution
type ExecuteSwapResponse struct {
	Status        string `json:"status"` // e.g., "pending_ibc", "pending_swap", "completed", "failed"
	Message       string `json:"message"`
	IbcTxHash     string `json:"ibcTxHash,omitempty"`
	SwapTxHash    string `json:"swapTxHash,omitempty"`
	FinalAmountOut string `json:"finalAmountOut,omitempty"`
}

// Rate information from a specific DEX
type DexRateInfo struct {
	ChainID   string `json:"chainID"`
	DexAddress string `json:"dexAddress"`
	FromToken string `json:"fromToken"`
	ToToken   string `json:"toToken"`
	Rate      string `json:"rate"` // Exchange rate as string
}
