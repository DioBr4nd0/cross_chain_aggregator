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
	// ACTION: Replace DexContract addresses with your actual deployed contract addresses.
	// ACTION: Ensure HomeDir paths are correct for your environment.
	// ACTION: Ensure OperatorKeyName is the key you created (e.g., "backendop").
	// ACTION: Ensure GasPrices and FeeDenom are appropriate for your chains.
	GlobalAppConfig.Chains = map[string]ChainConfig{
		"alphanet-1": {
			ID: "alphanet-1", Name: "AlphaNet", RPCEndpoint: "http://localhost:26657", GRPCEndpoint: "localhost:9090",
			AccountPrefix: "wasm", HomeDir: "/home/rupesh/.alphanet", OperatorKeyName: "backendop", FeeDenom: "ualpha", GasPrices: "0.025ualpha",
			DexContract:   "wasm1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrqr5j2ht", // EXAMPLE - UPDATE!
			SupportedTokens: []string{"ualpha", "ibc/BetaOnAlpha", "ibc/GammaOnAlpha"},
		},
		"betanet-1": {
			ID: "betanet-1", Name: "BetaNet", RPCEndpoint: "http://localhost:27657", GRPCEndpoint: "localhost:9190",
			AccountPrefix: "wasm", HomeDir: "/home/rupesh/.betanet", OperatorKeyName: "backendop", FeeDenom: "ubeta", GasPrices: "0.025ubeta",
			DexContract:   "wasm1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrqr5j2ht", // EXAMPLE - UPDATE!
			SupportedTokens: []string{"ubeta", "ibc/AlphaOnBeta", "ibc/GammaOnBeta"},
		},
		"gammanet-1": {
			ID: "gammanet-1", Name: "GammaNet", RPCEndpoint: "http://localhost:28657", GRPCEndpoint: "localhost:9290",
			AccountPrefix: "wasm", HomeDir: "/home/rupesh/.gammanet", OperatorKeyName: "backendop", FeeDenom: "ugamma", GasPrices: "0.025ugamma",
			DexContract:   "wasm1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrqr5j2ht", // EXAMPLE - UPDATE!
			SupportedTokens: []string{"ugamma", "ibc/AlphaOnGamma", "ibc/BetaOnGamma"},
		},
	}

	// ACTION: Update ChannelIDs after Hermes setup.
	GlobalAppConfig.IBCChannels = []IBCChannelConfig{
		{FromChainID: "alphanet-1", ToChainID: "betanet-1", ChannelID: "channel-0", PortID: "transfer"},
		{FromChainID: "betanet-1", ToChainID: "alphanet-1", ChannelID: "channel-0", PortID: "transfer"},
		{FromChainID: "betanet-1", ToChainID: "gammanet-1", ChannelID: "channel-1", PortID: "transfer"},
		{FromChainID: "gammanet-1", ToChainID: "betanet-1", ChannelID: "channel-1", PortID: "transfer"},
		{FromChainID: "alphanet-1", ToChainID: "gammanet-1", ChannelID: "channel-2", PortID: "transfer"},
		{FromChainID: "gammanet-1", ToChainID: "alphanet-1", ChannelID: "channel-2", PortID: "transfer"},
	}
	fmt.Println("Application configuration initialized.")
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
