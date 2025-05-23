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
			ID:            "chain-a",
			Name:          "Chain Alpha (wasmd)",
			RPCEndpoint:   "http://localhost:26657",
			GRPCEndpoint:  "localhost:9090",
			DexContract:   "wasm1...", // Update this after deploying mock_dex to chain-a
			NativeToken:   "uwasma",
			FeeDenom:      "uwasma",
			SupportedTokens: []string{"uwasma", "tokenb", "tokenc"}, // Example tokens chain-a DEX can trade
		},
		{
			ID:            "chain-b",
			Name:          "Chain Beta (wasmd)",
			RPCEndpoint:   "http://localhost:27657",
			GRPCEndpoint:  "localhost:9190",
			DexContract:   "wasm1...", // Update this after deploying mock_dex to chain-b
			NativeToken:   "uwasmb",
			FeeDenom:      "uwasmb",
			SupportedTokens: []string{"uwasmb", "tokena", "tokenc"},
		},
		{
			ID:            "chain-c",
			Name:          "Chain Gamma (wasmd)",
			RPCEndpoint:   "http://localhost:28657",
			GRPCEndpoint:  "localhost:9290",
			DexContract:   "wasm1...", // Update this after deploying mock_dex to chain-c
			NativeToken:   "uwasmc",
			FeeDenom:      "uwasmc",
			SupportedTokens: []string{"uwasmc", "tokena", "tokenb"},
		},
	}

	GlobalIBCChannels = []IBCChannelInfo{
		{FromChainID: "chain-a", ToChainID: "chain-b", ChannelID: "channel-0", PortID: "transfer"}, // Example
		{FromChainID: "chain-b", ToChainID: "chain-a", ChannelID: "channel-0", PortID: "transfer"},
		{FromChainID: "chain-b", ToChainID: "chain-c", ChannelID: "channel-1", PortID: "transfer"},
		{FromChainID: "chain-c", ToChainID: "chain-b", ChannelID: "channel-1", PortID: "transfer"},
		{FromChainID: "chain-a", ToChainID: "chain-c", ChannelID: "channel-2", PortID: "transfer"},
		{FromChainID: "chain-c", ToChainID: "chain-a", ChannelID: "channel-2", PortID: "transfer"},
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
