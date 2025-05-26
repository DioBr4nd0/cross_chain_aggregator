package services

import (
	"encoding/json"
	"fmt"

	"cosmos_defi_aggregator/config" // Adjust
	"cosmos_defi_aggregator/models" // Adjust

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// ExecuteDexSwapOnChain executes a real swap on the DEX contract
func ExecuteDexSwapOnChain(chainID, sender, recipient string, amount types.Coin, minOut math.Int) (string, error) {
	chain, err := config.GetChainByID(chainID)
	if !err {
		return "", fmt.Errorf("chain %s not found", chainID)
	}

	// Set up gRPC connection
	grpcConn, err := grpc.Dial(
		chain.GRPCEndpoint,
		grpc.WithInsecure(),
	)
	if !err {
		return "", fmt.Errorf("failed to connect to gRPC: %w", err)
	}
	defer grpcConn.Close()

	// Create swap message
	swapMsg := struct {
		Swap struct {
			ToDenom   string `json:"to_denom"`
			MinOutput string `json:"min_output"`
		} `json:"swap"`
	}{
		Swap: struct {
			ToDenom   string `json:"to_denom"`
			MinOutput string `json:"min_output"`
		}{
			ToDenom:   amount.Denom, // Replace with target denom
			MinOutput: minOut.String(),
		},
	}

	msgBz, err := json.Marshal(swapMsg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal swap msg: %w", err)
	}

	// Build transaction
	msg := &wasmtypes.MsgExecuteContract{
		Sender:   sender,
		Contract: chain.DexContract,
		Msg:      msgBz,
		Funds:    []types.Coin{amount},
	}

	// Broadcast transaction (implementation depends on your signing setup)
	txHash, err := broadcastTx(chain, msg)
	if err != nil {
		return "", fmt.Errorf("tx failed: %w", err)
	}

	return txHash, nil
}

// ExecuteDexSwapOnChain SIMULATES executing a swap on a specific DEX contract.
// In a real implementation, this would build and broadcast a MsgExecuteContract.
// func ExecuteDexSwapOnChain(
// 	chainID string,
// 	dexContractAddress string,
// 	senderAddress string, // User initiating the swap on this chain
// 	amountIn string,      // e.g., "1000000" (after potential IBC)
// 	tokenInDenom string,  // Denom of the token being swapped (could be an IBC voucher)
// 	tokenOutDenom string, // Desired output token
// 	minAmountOut string,  // Minimum expected output
// ) (models.ExecuteSwapResponse, error) {

// 	fmt.Printf("SIMULATING: DEX Swap on %s (DEX: %s) from %s\n", chainID, dexContractAddress, senderAddress)
// 	fmt.Printf("  Swapping: %s %s for %s (min out: %s)\n", amountIn, tokenInDenom, tokenOutDenom, minAmountOut)

// 	// Simulate network delay and transaction processing
// 	time.Sleep(time.Duration(rand.Intn(2)+1) * time.Second) // Simulate 1-2 second delay

// 	// Simulate getting the rate again just before swap (could differ due to activity)
// 	rate, err := QueryDexRate(chainID, dexContractAddress, tokenInDenom, tokenOutDenom)
// 	if err != nil {
// 		return models.ExecuteSwapResponse{Status: "failed", Message: fmt.Sprintf("Failed to get rate for swap on %s: %s", chainID, err.Error())}, err
// 	}

// 	amountInInt, _ := utils.StringToSDKInt(amountIn)
// 	// minAmountOutInt, _ := utils.StringToSDKInt(minAmountOut) // Not used in this simulation's success path
	
// 	actualAmountOutDec := math.LegacyNewDecFromInt(amountInInt).Mul(rate)
// 	actualAmountOut := actualAmountOutDec.TruncateInt().String()


// 	simulatedTxHash := fmt.Sprintf("simulated_dex_swap_tx_%x", time.Now().UnixNano())

// 	return models.ExecuteSwapResponse{
// 		Status:        "completed",
// 		Message:       fmt.Sprintf("Swap of %s %s for %s %s on %s completed.", amountIn, tokenInDenom, actualAmountOut, tokenOutDenom, chainID),
// 		SwapTxHash:    simulatedTxHash,
// 		FinalAmountOut: actualAmountOut,
// 	}, nil
// }

// ExecuteFullSwapRoute orchestrates the entire swap, including potential IBC.
func ExecuteFullSwapRoute(req models.ExecuteSwapRequest) (models.ExecuteSwapResponse, error) {
	fmt.Printf("Executing full swap route: %+v\n", req)

	bestRouteResponse, err := FindBestRouteForSwap(models.BestRouteRequest{
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		AmountIn:    req.AmountIn,
		FromChainID: req.FromChainID,
	})
	if err != nil {
		return models.ExecuteSwapResponse{Status: "failed", Message: fmt.Sprintf("Could not find route: %s", err.Error())}, err
	}

	bestRoute := bestRouteResponse.BestRoute
	recipient := req.RecipientAddress
	if recipient == "" {
		recipient = req.SenderAddress // Default to sender if no specific recipient
	}

	// If the best chain for swapping is different from where the user's tokens are
	if bestRoute.NeedsIBC {
		fmt.Printf("Route requires IBC. Best swap on chain %s, tokens currently on %s.\n", bestRoute.ChainIDSwappingOn, req.FromChainID)
		// Step 1: IBC Transfer
		ibcResponse, err := ExecuteIBCTransfer(
			req.FromChainID,
			bestRoute.ChainIDSwappingOn,
			req.SenderAddress,          // Sender on source chain
			req.SenderAddress,          // Recipient on destination chain (the user themselves, to then swap)
			req.AmountIn,
			req.FromToken,
		)
		if err != nil {
			ibcResponse.Status = "failed"
			ibcResponse.Message = fmt.Sprintf("IBC transfer failed: %s", err.Error())
			return ibcResponse, err
		}
		fmt.Printf("IBC Transfer Simulation: TxHash %s, Status: %s\n", ibcResponse.IbcTxHash, ibcResponse.Status)

		// For hackathon: Assume IBC completes and tokens are now on bestRoute.ChainIDSwappingOn
		// The amountIn for the DEX swap is the original amountIn (ignoring IBC fees for simplicity)
		// The tokenInDenom for the DEX swap is now the IBC representation of req.FromToken on bestRoute.ChainIDSwappingOn
		// This is a MAJOR simplification. A real app needs the IBC denom.
		// For simulation, we'll just use the original fromToken.
		tokenInDenomForDexSwap := req.FromToken
		chainForDexSwap, _ := config.GetChainByID(bestRoute.ChainIDSwappingOn)

		// Step 2: DEX Swap on the destination chain
		swapResponse, err := ExecuteDexSwapOnChain(
			bestRoute.ChainIDSwappingOn,
			chainForDexSwap.DexContract,
			req.SenderAddress, // User (who now "has" IBC'd tokens on this chain) initiates the swap
			req.AmountIn,
			tokenInDenomForDexSwap, // Simplified: should be IBC denom of req.FromToken
			req.ToToken,
			"0", // Min amount out, could be derived from bestRoute.AmountOut with slippage
		)
		if err != nil {
			swapResponse.Status = "failed"
			swapResponse.Message = fmt.Sprintf("DEX swap on %s failed: %s", bestRoute.ChainIDSwappingOn, err.Error())
			swapResponse.IbcTxHash = ibcResponse.IbcTxHash // Carry over IBC hash
			return swapResponse, err
		}
		swapResponse.IbcTxHash = ibcResponse.IbcTxHash
		fmt.Printf("DEX Swap Simulation: TxHash %s, Status: %s, Final Amount Out: %s\n", swapResponse.SwapTxHash, swapResponse.Status, swapResponse.FinalAmountOut)
		return swapResponse, nil

	} else { // Direct swap on the source chain
		fmt.Printf("Route is direct swap on source chain %s.\n", req.FromChainID)
		chainForDexSwap, _ := config.GetChainByID(req.FromChainID)
		swapResponse, err := ExecuteDexSwapOnChain(
			req.FromChainID,
			chainForDexSwap.DexContract,
			req.SenderAddress,
			req.AmountIn,
			req.FromToken,
			req.ToToken,
			"0", // Min amount out
		)
		if err != nil {
			swapResponse.Status = "failed"
			swapResponse.Message = fmt.Sprintf("DEX swap on %s failed: %s", req.FromChainID, err.Error())
			return swapResponse, err
		}
		fmt.Printf("DEX Swap Simulation: TxHash %s, Status: %s, Final Amount Out: %s\n", swapResponse.SwapTxHash, swapResponse.Status, swapResponse.FinalAmountOut)
		return swapResponse, nil
	}
}
