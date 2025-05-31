package config

import "fmt"

// ChainConfig holds all necessary information for interacting with a chain.
type ChainConfig struct {
	ID            string   `json:"id"`              // e.g., "alphanet-1"
	Name          string   `json:"name"`            // e.g., "AlphaNet"
	RPCEndpoint   string   `json:"rpcEndpoint"`     // Tendermint RPC (e.g., "http://localhost:26657")
	GRPCEndpoint  string   `json:"grpcEndpoint"`    // gRPC endpoint (e.g., "localhost:9090")
	AccountPrefix string   `json:"accountPrefix"`   // Bech32 prefix (e.g., "wasm")
	HomeDir       string   `json:"-"`               // Path to the chain's data/keyring directory
	OperatorKeyName string   `json:"-"`               // Name of the key in the chain's keyring for operations
	FeeDenom      string   `json:"feeDenom"`        // Denom used for gas fees (e.g., "ualpha")
	GasPrices     string   `json:"gasPrices"`       // Gas prices for transactions (e.g., "0.025ualpha")
	DexContract   string   `json:"dexContract"`     // Address of the deployed MockDEX contract
	SupportedTokens []string `json:"supportedTokens"` // Tokens this chain's DEX can trade (including IBC vouchers)
	NativeToken   string   `json:"nativeToken"`
}

// IBCChannelConfig defines an IBC channel between two chains.
type IBCChannelConfig struct {
	FromChainID string `json:"fromChainID"`
	ToChainID   string `json:"toChainID"`
	ChannelID   string `json:"channelID"` // Channel ID on FromChainID to reach ToChainID
	PortID      string `json:"portID"`    // Usually "transfer" for ICS20
}

// AppConfiguration holds the overall application configuration.
type AppConfiguration struct {
	Chains      map[string]ChainConfig // Map chainID to ChainConfig
	IBCChannels []IBCChannelConfig
}

// GlobalAppConfig is the single instance of application configuration.
var GlobalAppConfig AppConfiguration

// InitAppConfig initializes the application configuration.
// This should be called once at application startup.
func InitAppConfig() {
	GlobalAppConfig.Chains = map[string]ChainConfig{
		"alphanet-1": {
			ID: "alphanet-1", Name: "AlphaNet", RPCEndpoint: "http://localhost:26657", GRPCEndpoint: "localhost:9090",
			AccountPrefix: "wasm", HomeDir: "/home/rupesh/.alphanet", OperatorKeyName: "backendop", FeeDenom: "ualpha", GasPrices: "0.025ualpha",
			DexContract:   "wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d", NativeToken:   "ualpha",// Use your actual deployed address
			SupportedTokens: []string{"ualpha", "ibc/BetaOnAlpha", "ibc/GammaOnAlpha"}, // Conceptual IBC denoms
		},
		"betanet-1": {
			ID: "betanet-1", Name: "BetaNet", RPCEndpoint: "http://localhost:27657", GRPCEndpoint: "localhost:9190",
			AccountPrefix: "wasm", HomeDir: "/home/rupesh/.betanet", OperatorKeyName: "backendop", FeeDenom: "ubeta", GasPrices: "0.025ubeta",
			DexContract:   "wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d", NativeToken:   "ubeta",// Use your actual deployed address
			SupportedTokens: []string{"ubeta", "ibc/AlphaOnBeta", "ibc/GammaOnBeta"},
		},
		"gammanet-1": {
			ID: "gammanet-1", Name: "GammaNet", RPCEndpoint: "http://localhost:28657", GRPCEndpoint: "localhost:9290",
			AccountPrefix: "wasm", HomeDir: "/home/rupesh/.gammanet", OperatorKeyName: "backendop", FeeDenom: "ugamma", GasPrices: "0.025ugamma",
			DexContract:   "wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d", NativeToken:   "ugamma",// Use your actual deployed address
			SupportedTokens: []string{"ugamma", "ibc/AlphaOnGamma", "ibc/BetaOnGamma"},
		},
	}

	// Corrected ChannelIDs based on typical Hermes behavior for the manual script
	GlobalAppConfig.IBCChannels = []IBCChannelConfig{
		// AlphaNet <-> BetaNet
		{FromChainID: "alphanet-1", ToChainID: "betanet-1", ChannelID: "channel-0", PortID: "transfer"},
		{FromChainID: "betanet-1", ToChainID: "alphanet-1", ChannelID: "channel-0", PortID: "transfer"},

		// BetaNet <-> GammaNet
		{FromChainID: "betanet-1", ToChainID: "gammanet-1", ChannelID: "channel-0", PortID: "transfer"}, // This is channel-0 on BetaNet's *new connection* to GammaNet
		{FromChainID: "gammanet-1", ToChainID: "betanet-1", ChannelID: "channel-0", PortID: "transfer"}, // This is channel-0 on GammaNet's *first connection* (to BetaNet)

		// AlphaNet <-> GammaNet
		{FromChainID: "alphanet-1", ToChainID: "gammanet-1", ChannelID: "channel-0", PortID: "transfer"}, // This is channel-0 on AlphaNet's *new connection* to GammaNet
		{FromChainID: "gammanet-1", ToChainID: "alphanet-1", ChannelID: "channel-0", PortID: "transfer"}, // This is channel-0 on GammaNet's *new connection* to AlphaNet
	}
	fmt.Println("Application configuration initialized with corrected IBC Channel IDs.")
}

// GetChain retrieves a chain configuration by its ID.
func GetChain(id string) (ChainConfig, bool) {
	chain, found := GlobalAppConfig.Chains[id]
	return chain, found
}

// GetIBCChannel retrieves IBC channel information.
func GetIBCChannel(fromChainID, toChainID string) (IBCChannelConfig, bool) {
	for _, ch := range GlobalAppConfig.IBCChannels {
		if ch.FromChainID == fromChainID && ch.ToChainID == toChainID {
			return ch, true
		}
	}
	return IBCChannelConfig{}, false
}
