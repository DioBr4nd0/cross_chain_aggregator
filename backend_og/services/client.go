package services

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client" // Used for TxFactory
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	// Adjust this import to your actual wasmd app's module path
	// This provides the application's encoding configuration.
	wasmdapp "github.com/CosmWasm/wasmd/app"

	"cosmos_defi_aggregator/config" // Adjust to your project's module path
)

// GetClientContextForChain creates and configures a client.Context for a given chain.
// It loads the operator key for that chain into an in-memory keyring.
func GetClientContextForChain(chainConfig config.Chain) (client.Context, keyring.Keyring, string, error) {
	var kr keyring.Keyring
	var keyName string

	encodingConfig := wasmdapp.MakeEncodingConfig()

	// Create an RPC client
	// Ensure chainConfig.RPCEndpoint is the Tendermint RPC (e.g., "http://localhost:26657")
	rpcClient, err := client.NewClientFromNode(chainConfig.RPCEndpoint)
	if err != nil {
		return client.Context{}, nil, "", fmt.Errorf("failed to create RPC client for chain %s: %w", chainConfig.ID, err)
	}

	clientCtx := client.Context{}.
		WithChainID(chainConfig.ID).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(nil).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync). // Use Sync for hackathon to get hash quickly
		WithViper("").
		// With темно(true).
		WithClient(rpcClient).
		WithNodeURI(chainConfig.RPCEndpoint)

	// Prepare Keyring and load operator key
	operatorMnemonic, ok := config.OperatorMnemonics[chainConfig.ID]
	if !ok {
		return client.Context{}, nil, "", fmt.Errorf("operator mnemonic not found for chain %s", chainConfig.ID)
	}

	kr = keyring.NewInMemory(clientCtx.Codec())
	keyName = fmt.Sprintf("%s-operator-key", chainConfig.ID) // Unique name for the key in this keyring instance

	// Standard Cosmos HD path
	hdPath := sdk.GetConfig().GetFullBIP44Path()
	_, err = kr.NewAccount(keyName, operatorMnemonic, keyring.DefaultBIP39Passphrase, hdPath, hd.Secp256k1)
	if err != nil {
		// Check if key already exists (e.g. if this function is called multiple times with same in-memory keyring)
		// This shouldn't happen if kr is new each time.
		// For simplicity, we assume it doesn't exist or NewAccount handles it.
		return client.Context{}, nil, "", fmt.Errorf("failed to import operator account for chain %s: %w", chainConfig.ID, err)
	}

	signerInfo, err := kr.Key(keyName)
	if err != nil {
		return client.Context{}, nil, "", fmt.Errorf("failed to get signer info for %s on %s: %w", keyName, chainConfig.ID, err)
	}

	clientCtx = clientCtx.WithKeyring(kr).
		WithFrom(keyName). // Set the "from" field for TxFactory
		WithFromAddress(signerInfo.GetAddress()).
		WithFromName(keyName)

	return clientCtx, kr, keyName, nil
}

// BroadcastTx prepares, signs, and broadcasts a transaction.
func BroadcastTx(clientCtx client.Context, kr keyring.Keyring, keyNameToSignWith string, msg sdk.Msg, memo string) (string, error) {
	// Create a TxFactory
	// Gas prices should match what your chain expects
	txf := tx.NewFactoryCLI(clientCtx, nil). // Pass clientCtx, not a new one
									WithChainID(clientCtx.ChainID).
									WithKeybase(kr). // Use the passed keyring
									WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
									WithGasAdjustment(1.5).
									WithGasPrices(fmt.Sprintf("0.025%s", config.MustGetChainByID(clientCtx.ChainID).FeeDenom)) // Assuming FeeDenom is set in config.Chain

	// Populate account number and sequence
	fromAddr := clientCtx.GetFromAddress()
	accNum, accSeq, err := authtypes.NewAccountRetriever(clientCtx).GetAccountNumberSequence(clientCtx, fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get account details for %s: %w", fromAddr, err)
	}
	txf = txf.WithAccountNumber(accNum).WithSequence(accSeq)

	if memo != "" {
		txf = txf.WithMemo(memo)
	}

	// Estimate gas or set a fixed gas limit
	// For Wasm execute, gas can be high. Simulation is preferred if it works.
	// _, gasUsed, err := tx.CalculateGas(clientCtx, txf, msg)
	// if err != nil {
	//     // Fallback to a higher fixed gas if simulation fails
	//     fmt.Printf("Gas simulation failed for %s, falling back to fixed gas: %v\n", clientCtx.ChainID, err)
	//     txf = txf.WithGas(300000) // Example: 300k gas, adjust as needed
	// } else {
	//     txf = txf.WithGas(gasUsed)
	// }
	// For MVP, let's try to let BuildUnsignedTx estimate, or set a generous fixed one if it fails often.
	// Many wasm operations may need more gas, e.g., 200k-500k or more.
	// For now, let BuildUnsignedTx try. Add WithGas if needed.
	// txf = txf.WithGas(300000) // Uncomment and adjust if auto-gas is problematic


	// Build and sign the transaction
	unsignedTx, err := tx.BuildUnsignedTx(txf, msg)
	if err != nil {
		return "", fmt.Errorf("failed to build unsigned tx: %w", err)
	}

	// Sign the transaction using the key name passed
	err = tx.Sign(context.Background(), txf, keyNameToSignWith, unsignedTx, true)
	if err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(unsignedTx.GetTx())
	if err != nil {
		return "", fmt.Errorf("failed to encode tx: %w", err)
	}

	// Broadcast it
	res, err := clientCtx.BroadcastTxSync(txBytes) // Using Sync for quick hash
	if err != nil {
		return "", fmt.Errorf("failed to broadcast tx: %w", err)
	}

	if res.Code != 0 {
		return res.TxHash, fmt.Errorf("tx broadcast failed with code %d: %s", res.Code, res.RawLog)
	}

	fmt.Printf("Tx broadcasted successfully on chain %s. Hash: %s\n", clientCtx.ChainID, res.TxHash)
	return res.TxHash, nil
}

// Helper to get chain config, panics if not found (use carefully or adapt)
func (c *config.Config) MustGetChainByID(id string) config.Chain {
    chain, found := c.GetChainByID(id)
    if !found {
        panic(fmt.Sprintf("Chain with ID %s not found in configuration", id))
    }
    return chain
}

