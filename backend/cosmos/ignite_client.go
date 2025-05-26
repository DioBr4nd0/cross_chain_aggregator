package cosmos

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cosmos_defi_aggregator/config" // Adjust
)

var igniteClients map[string]cosmosclient.Client

// InitializeIgniteClients sets up cosmosclient instances for all configured chains.
func InitializeIgniteClients(ctx context.Context) error {
	igniteClients = make(map[string]cosmosclient.Client)
	for chainID, chainCfg := range config.GlobalAppConfig.Chains {
		clientOpts := []cosmosclient.Option{
			cosmosclient.WithAddressPrefix(chainCfg.AccountPrefix),
			cosmosclient.WithNodeAddress(chainCfg.RPCEndpoint), // Tendermint RPC for tx broadcasting
			cosmosclient.WithHome(chainCfg.HomeDir),           // For keyring access
			cosmosclient.WithKeyringBackend(cosmosclient.KeyringTest),
			cosmosclient.WithGas("auto"), // Default gas estimation
			cosmosclient.WithGasAdjustment(1.5),
			// Fees can be tricky with auto gas. If BroadcastTx fails on fees, set them here.
			// cosmosclient.WithFees(fmt.Sprintf("10000%s", chainCfg.FeeDenom)),
		}

		client, err := cosmosclient.New(ctx, clientOpts...)
		if err != nil {
			return fmt.Errorf("failed to create Ignite client for chain %s: %w", chainID, err)
		}
		igniteClients[chainID] = client
		log.Printf("Ignite client initialized for chain: %s (RPC: %s, Home: %s)", chainID, chainCfg.RPCEndpoint, chainCfg.HomeDir)
	}
	return nil
}

// GetIgniteClient retrieves an initialized client for a chain.
func GetIgniteClient(chainID string) (cosmosclient.Client, error) {
	client, ok := igniteClients[chainID]
	if !ok {
		return cosmosclient.Client{}, fmt.Errorf("Ignite client not initialized for chain %s", chainID)
	}
	return client, nil
}

// QueryContractWithIgnite queries a smart contract.
func QueryContractWithIgnite(ctx context.Context, chainID, contractAddr string, queryMsgJSON []byte) ([]byte, error) {
	// For queries, we often use gRPC directly if cosmosclient's query abstraction is limited
	// or if we need specific query clients (like wasmtypes.QueryClient).
	chainCfg, found := config.GetChain(chainID)
	if !found {
		return nil, fmt.Errorf("chain config not found for query: %s", chainID)
	}

	conn, err := grpc.DialContext(ctx, chainCfg.GRPCEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("gRPC dial failed for chain %s query: %w", chainID, err)
	}
	defer conn.Close()

	wasmQueryClient := wasmtypes.NewQueryClient(conn)
	res, err := wasmQueryClient.SmartContractState(ctx, &wasmtypes.QuerySmartContractStateRequest{
		Address:   contractAddr,
		QueryData: queryMsgJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("smart contract query failed on chain %s: %w", chainID, err)
	}
	return res.Data, nil
}

// BroadcastMessageWithIgnite broadcasts messages using the Ignite client.
func BroadcastMessageWithIgnite(ctx context.Context, chainID, operatorKeyName string, msgs ...sdk.Msg) (string, error) {
	client, err := GetIgniteClient(chainID)
	if err != nil {
		return "", err
	}

	account, err := client.Account(operatorKeyName)
	if err != nil {
		return "", fmt.Errorf("failed to get account '%s' for chain %s: %w", operatorKeyName, chainID, err)
	}

	// BroadcastTx automatically handles gas, fees (if configured in client), acc num/seq.
	resp, err := client.BroadcastTx(ctx, account, msgs...)
	if err != nil {
		// Try to get TxHash even if there's an error (e.g., tx failed in block)
		if resp.TxResponse != nil && resp.TxResponse.TxHash != "" {
			return resp.TxResponse.TxHash, fmt.Errorf("broadcast error (txhash: %s) on chain %s: %w. RawLog: %s", resp.TxResponse.TxHash, chainID, err, resp.TxResponse.RawLog)
		}
		return "", fmt.Errorf("broadcast error on chain %s: %w", chainID, err)
	}

	if resp.TxResponse.Code != 0 { // ABCI code
		return resp.TxResponse.TxHash, fmt.Errorf("tx failed with code %d on chain %s: %s", resp.TxResponse.Code, chainID, resp.TxResponse.RawLog)
	}

	log.Printf("Tx broadcasted via Ignite on %s (Key: %s). Hash: %s", chainID, operatorKeyName, resp.TxResponse.TxHash)
	return resp.TxResponse.TxHash, nil
}
