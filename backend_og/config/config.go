package config

import "fmt"

// Chain represents a blockchain configuration
type Chain struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	RPCEndpoint   string   `json:"rpcEndpoint"`  // For queries
	GRPCEndpoint  string   `json:"grpcEndpoint"` // For queries/tx (if using gRPC client)
	DexContract   string   `json:"dexContract"`  // Placeholder, update after deployment
	NativeToken   string   `json:"nativeToken"`
	FeeDenom      string   `json:"feeDenom"` // Denom for transaction fees on this chain
	SupportedTokens []string `json:"supportedTokens"` // All tokens available for swap on this chain's DEX
}

// GlobalChains holds the configuration for all chains
var GlobalChains []Chain

// IBCChannelInfo stores the channel ID for a path between two chains
type IBCChannelInfo struct {
	FromChainID string
	ToChainID   string
	ChannelID   string // Channel ID on FromChainID to reach ToChainID
	PortID      string // Usually "transfer" for ICS20
}

// GlobalIBCChannels holds IBC channel configurations
var GlobalIBCChannels []IBCChannelInfo

// Init initializes the chain and IBC configurations
func Init() {
	GlobalChains = []Chain{
		{
			ID:            "alphanet-1",
			Name:          "Chain Alpha (wasmd)",
			RPCEndpoint:   "http://localhost:26657",
			GRPCEndpoint:  "localhost:9090",
			DexContract:   "wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d", // Update this after deploying mock_dex to chain-a
			NativeToken:   "ualpha",
			FeeDenom:      "ualpha",
			SupportedTokens: []string{"ualpha", "tokenb", "tokenc"}, // Example tokens chain-a DEX can trade
		},
		{
			ID:            "betanet-1",
			Name:          "Chain Beta (wasmd)",
			RPCEndpoint:   "http://localhost:27657",
			GRPCEndpoint:  "localhost:9190",
			DexContract:   "wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d", // Update this after deploying mock_dex to chain-b
			NativeToken:   "ubeta",
			FeeDenom:      "ubeta",
			SupportedTokens: []string{"ubeta", "tokena", "tokenc"},
		},
		{
			ID:            "gammanet-1",
			Name:          "Chain Gamma (wasmd)",
			RPCEndpoint:   "http://localhost:28657",
			GRPCEndpoint:  "localhost:9290",
			DexContract:   "wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d", // Update this after deploying mock_dex to chain-c
			NativeToken:   "ugamma",
			FeeDenom:      "ugamma",
			SupportedTokens: []string{"ugamma", "tokena", "tokenb"},
		},
	}

	GlobalIBCChannels = []IBCChannelInfo{
		{FromChainID: "alphanet-1", ToChainID: "betanet-1", ChannelID: "channel-0", PortID: "transfer"}, // Example
		{FromChainID: "betanet-1", ToChainID: "alphanet-1", ChannelID: "channel-0", PortID: "transfer"},
		{FromChainID: "betanet-1", ToChainID: "gammanet-1", ChannelID: "channel-1", PortID: "transfer"},
		{FromChainID: "gammanet-1", ToChainID: "betanet-1", ChannelID: "channel-1", PortID: "transfer"},
		{FromChainID: "alphanet-1", ToChainID: "gammanet-1", ChannelID: "channel-2", PortID: "transfer"},
		{FromChainID: "gammanet-1", ToChainID: "alphanet-1", ChannelID: "channel-2", PortID: "transfer"},
	}
	fmt.Println("Chain configurations initialized.")
}

// GetChainByID retrieves a chain configuration by its ID
func GetChainByID(id string) (Chain, bool) {
	for _, chain := range GlobalChains {
		if chain.ID == id {
			return chain, true
		}
	}
	return Chain{}, false
}

// GetIBCChannel retrieves the IBC channel information for a path
func GetIBCChannel(fromChainID, toChainID string) (IBCChannelInfo, bool) {
	for _, ch := range GlobalIBCChannels {
		if ch.FromChainID == fromChainID && ch.ToChainID == toChainID {
			return ch, true
		}
	}
	return IBCChannelInfo{}, false
}
