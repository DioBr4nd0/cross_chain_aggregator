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
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	// Import your specific wasmd app for encoding config
	// Adjust this import if your wasmd app has a different module path
	wasmdapp "github.com/CosmWasm/wasmd/app"

	"cosmos_defi_aggregator/config" // Adjust to your project's module path
)

// broadcastTx handles the creation, signing, and broadcasting of a transaction.
func broadcastTx(chainConfig config.Chain, msg sdk.Msg, memo string) (string, error) {
	// 1. Create EncodingConfig
	// This typically comes from your app's specific initialization.
	// For wasmd, it's often `wasmdapp.MakeEncodingConfig()`.
	encodingConfig := wasmdapp.MakeEncodingConfig() // Or your app's equivalent

	// 2. Create a client.Context
	// For a backend, we typically don't use Viper for config, so we set fields manually.
	clientCtx := client.Context{}.
		WithChainID(chainConfig.ID).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(nil). // No input for backend service
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync). // Or BroadcastBlock / BroadcastAsync
		WithViper("")                    // Empty viper prefix
		// With темно(true). // Skip confirmation prompt
		// Note: NodeURI/Client needs to be set.
		// For gRPC based queries or txs, you might use a gRPC client.
		// For Tendermint RPC, you'd set up an RPC client.
		// Let's assume your chainConfig.RPCEndpoint is a Tendermint RPC endpoint.
		// Client needs to be initialized if tx.리와Factory requires it or if you query account seq.
	

	// Initialize an RPC client
	// Ensure your chainConfig.RPCEndpoint is the Tendermint RPC endpoint (e.g., "http://localhost:26657")
	rpcClient, err := client.NewClientFromNode(chainConfig.RPCEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to create RPC client for chain %s at %s: %w", chainConfig.ID, chainConfig.RPCEndpoint, err)
	}
	clientCtx = clientCtx.WithClient(rpcClient).WithNodeURI(chainConfig.RPCEndpoint)

	// 3. Prepare the Keyring and Signer Info
	// For a backend, we'll create an in-memory keyring and import the operator's key from mnemonic.
	operatorMnemonic, ok := config.OperatorMnemonics[chainConfig.ID]
	if !ok {
		return "", fmt.Errorf("operator mnemonic not found for chain %s in config", chainConfig.ID)
	}

	kr := keyring.NewInMemory(clientCtx.Codec())
	keyName := fmt.Sprintf("%s-operator", chainConfig.ID) // Unique key name for this in-memory instance

	// Import account from mnemonic
	// The HD path is the standard Cosmos one.
	hdPath := sdk.GetConfig().GetFullBIP44Path() // Or specific if your keys were derived differently
	_, err = kr.NewAccount(keyName, operatorMnemonic, keyring.DefaultBIP39Passphrase, hdPath, hd.Secp256k1)
	if err != nil {
		return "", fmt.Errorf("failed to import operator account for chain %s from mnemonic: %w", chainConfig.ID, err)
	}

	signerInfo, err := kr.Key(keyName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve signer info for key '%s' on chain %s: %w", keyName, chainConfig.ID, err)
	}
	signerAddress := signerInfo.GetAddress()
	clientCtx = clientCtx.WithKeyring(kr).WithFrom(keyName).WithFromAddress(signerAddress).WithFromName(keyName)

	// 4. Prepare TxFactory
	// This factory will help us build and sign the transaction.
	// We need to fetch the account number and sequence for the signer.
	txf := tx.NewFactoryCLI(clientCtx, nil).
		WithChainID(chainConfig.ID).
		WithKeybase(kr).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT). // Or your preferred sign mode
		WithGasAdjustment(1.5).                           // Default gas adjustment
		WithGasPrices(fmt.Sprintf("0.025%s", chainConfig.FeeDenom)) // Example gas prices
		// WithFees(...) or WithGas(...) can also be set if you have specific values

	// Populate account number and sequence
	// This requires an online query unless you manage these offline.
	accNum, accSeq, err := authtypes.NewAccountRetriever(clientCtx).GetAccountNumberSequence(clientCtx, signerAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get account number/sequence for %s on chain %s: %w", signerAddress.String(), chainConfig.ID, err)
	}
	txf = txf.WithAccountNumber(accNum).WithSequence(accSeq)

	if memo != "" {
		txf = txf.WithMemo(memo)
	}
	
	// Set gas. If not set, BuildUnsignedTx will simulate to estimate.
	// For simplicity, let's allow auto gas estimation.
	// txf = txf.WithGas(flags.DefaultGasLimit) // Or a specific gas limit

	// 5. Build and Sign the Transaction
	// Ensure clientCtx.TxConfig is set (done by `WithTxConfig(encodingConfig.TxConfig)`)
	unsignedTx, err := tx.BuildUnsignedTx(txf, msg)
	if err != nil {
		// If gas estimation failed, it might show up here.
		// Try building with a fixed gas limit if auto-gas fails.
		// For wasm execute, gas can be high.
		// txfGasSet := txf.WithGas(200000) // Example fixed gas, adjust as needed
		// unsignedTx, err = tx.BuildUnsignedTx(txfGasSet, msg)
		// if err != nil {
			return "", fmt.Errorf("failed to build unsigned tx for chain %s: %w", chainConfig.ID, err)
		// }
	}

	// Sign the transaction
	err = tx.Sign(context.Background(), txf, keyName, unsignedTx, true) // true for overwriting existing signatures
	if err != nil {
		return "", fmt.Errorf("failed to sign tx for chain %s: %w", chainConfig.ID, err)
	}

	// 6. Broadcast the Transaction
	// Get the transaction bytes
	txBytes, err := clientCtx.TxConfig.TxEncoder()(unsignedTx.GetTx())
	if err != nil {
		return "", fmt.Errorf("failed to encode tx for chain %s: %w", chainConfig.ID, err)
	}

	// Broadcast it
	// BroadcastModeSync returns tx hash immediately.
	// BroadcastModeBlock waits for the tx to be included in a block (or timeout).
	// BroadcastModeAsync also returns immediately.
	// For a backend, Sync is often preferred for initial acknowledgement.
	res, err := clientCtx.BroadcastTxSync(txBytes) // Or BroadcastTxCommit / BroadcastTxAsync
	if err != nil {
		return "", fmt.Errorf("failed to broadcast tx for chain %s: %w", chainConfig.ID, err)
	}

	if res.Code != 0 { // Check Tendermint result code
		return "", fmt.Errorf("tx broadcast failed on chain %s with code %d: %s", chainConfig.ID, res.Code, res.RawLog)
	}

	fmt.Printf("Successfully broadcasted tx on chain %s. TxHash: %s\n", chainConfig.ID, res.TxHash)
	return res.TxHash, nil
}
