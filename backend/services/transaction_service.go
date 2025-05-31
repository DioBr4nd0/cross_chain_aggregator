package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	"cosmos_defi_aggregator/config"
	"cosmos_defi_aggregator/cosmos"
	"cosmos_defi_aggregator/models"
	// "cosmos_defi_aggregator/utils" // Not strictly needed here for MVP
)

// executeIBCTransferViaIgnite initiates an IBC token transfer.
func executeIBCTransferViaIgnite(
	ctx context.Context,
	fromChainID, toChainID,
	apiSenderAddress, recipientOnTargetChain string, // apiSenderAddress used to find operator key
	coinToSend sdk.Coin,
) (string, error) { // Returns txHash

	fromChainCfg, found := config.GetChain(fromChainID)
	if !found {
		return "", fmt.Errorf("source chain %s not found for IBC", fromChainID)
	}
	ibcChannelCfg, found := config.GetIBCChannel(fromChainID, toChainID)
	if !found {
		return "", fmt.Errorf("IBC channel from %s to %s not configured", fromChainID, toChainID)
	}

	// The operatorKeyName from config will be used by Ignite client.
	// The apiSenderAddress from request should logically map to this operator.
	// For IBC, the actual sender (signer) is the operator on fromChain.
	igniteClient, err := cosmos.GetIgniteClient(fromChainID)
	if err != nil { return "", err }
	operatorAccount, err := igniteClient.Account(fromChainCfg.OperatorKeyName)
	if err != nil { return "", fmt.Errorf("could not get operator account %s for IBC on %s: %w", fromChainCfg.OperatorKeyName, fromChainID, err)}
	actualSenderOnFromChain, err := operatorAccount.Address(fromChainCfg.AccountPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get address: %v",err)
	}

	timeoutHeight := clienttypes.ZeroHeight()
	timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).UnixNano())

	msg := transfertypes.NewMsgTransfer(
		ibcChannelCfg.PortID,
		ibcChannelCfg.ChannelID,
		coinToSend,
		actualSenderOnFromChain, // Actual signer address
		recipientOnTargetChain,
		timeoutHeight,
		timeoutTimestamp,
		"AggregatorIBC",
	)

	if err := msg.ValidateBasic(); err != nil {
		return "", fmt.Errorf("invalid IBC MsgTransfer: %w", err)
	}

	log.Printf("Submitting IBC Transfer: From %s (%s) To %s (%s) Amount %s OperatorKey %s",
		fromChainID, actualSenderOnFromChain, toChainID, recipientOnTargetChain, coinToSend.String(), fromChainCfg.OperatorKeyName)

	return cosmos.BroadcastMessageWithIgnite(ctx, fromChainID, fromChainCfg.OperatorKeyName, msg)
}

// executeDexSwapViaIgnite executes a swap on a DEX contract.
func executeDexSwapViaIgnite(
	ctx context.Context,
	chainID, apiSenderAddress string, // apiSenderAddress maps to operator key
	coinToSendToDex sdk.Coin,
	tokenOutDenom string,
	// finalRecipientAddress string, // TODO: If DEX doesn't send to final recipient
) (models.ExecuteSwapResponse, error) {

	chainCfg, found := config.GetChain(chainID)
	if !found {
		return models.ExecuteSwapResponse{Status: "failed"}, fmt.Errorf("chain %s not found for DEX swap", chainID)
	}
	igniteClient, err := cosmos.GetIgniteClient(chainID)
	if err != nil { return models.ExecuteSwapResponse{Status: "failed"}, err }
	operatorAccount, err := igniteClient.Account(chainCfg.OperatorKeyName)
	if err != nil { return models.ExecuteSwapResponse{Status: "failed"}, fmt.Errorf("could not get operator account %s for DEX swap on %s: %w", chainCfg.OperatorKeyName, chainID, err)}
	actualSenderOnChain, err := operatorAccount.Address(chainCfg.AccountPrefix)
	if err != nil{
		return models.ExecuteSwapResponse{Status: "failed"}, fmt.Errorf("failed to get sender: %v",actualSenderOnChain)
	}

	swapPayload := struct {
		Swap struct {
			ToDenom   string `json:"to_denom"`
			MinOutput string `json:"min_output"`
		} `json:"swap"`
	}{
		Swap: struct {
			ToDenom   string `json:"to_denom"`
			MinOutput string `json:"min_output"`
		}{ToDenom: tokenOutDenom, MinOutput: "0"}, // MVP: min_output "0"
	}
	executeMsgJSON, err := json.Marshal(swapPayload)
	if err != nil {
		return models.ExecuteSwapResponse{Status: "failed"}, fmt.Errorf("failed to marshal DEX swap msg: %w", err)
	}

	msg := &wasmtypes.MsgExecuteContract{
		Sender:   actualSenderOnChain, // Actual signer
		Contract: chainCfg.DexContract,
		Msg:      wasmtypes.RawContractMessage(executeMsgJSON),
		Funds:    sdk.NewCoins(coinToSendToDex),
	}
	// msg := wasmtypes.MsgExecuteContractWrapper{
    // MsgExecuteContract: execMsg,
	// }
	if err := msg.ValidateBasic(); err != nil {
		return models.ExecuteSwapResponse{Status: "failed"}, fmt.Errorf("invalid MsgExecuteContract: %w", err)
	}

	log.Printf("Submitting DEX Swap: Chain %s Contract %s Sender %s Amount %s To %s OperatorKey %s",
		chainID, chainCfg.DexContract, actualSenderOnChain, coinToSendToDex.String(), tokenOutDenom, chainCfg.OperatorKeyName)

	txHash, err := cosmos.BroadcastMessageWithIgnite(ctx, chainID, chainCfg.OperatorKeyName, msg)
	if err != nil {
		return models.ExecuteSwapResponse{Status: "failed", Message: err.Error(), SwapTxHash: txHash}, err
	}
	// Actual output amount is not known from BroadcastTxSync.
	return models.ExecuteSwapResponse{
		Status: "submitted_dex_swap", SwapTxHash: txHash, FinalAmountOut: "unknown_query_tx_events",
		Message: fmt.Sprintf("DEX Swap tx submitted on %s. Hash: %s", chainID, txHash),
	}, nil
}

// ProcessSwapRequest orchestrates the full swap flow.
func ProcessSwapRequest(ctx context.Context, req models.ExecuteSwapRequest) (models.ExecuteSwapResponse, error) {
	log.Printf("Processing swap request: %+v", req)

	// 1. Find the best route using the aggregator service
	bestRouteReq := models.BestRouteRequest{
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		AmountIn:    req.AmountIn,
		FromChainID: req.FromChainID,
	}
	routeResult, err := FindBestSwapRoute(ctx, bestRouteReq)
	if err != nil {
		return models.ExecuteSwapResponse{Status: "failed", Message: "Failed to find best route: " + err.Error()}, err
	}
	chosenRoute := routeResult.BestRoute

	// Parse initial amount
	amountInCoin, err := sdk.ParseCoinNormalized(req.AmountIn + req.FromToken)
	if err != nil {
		return models.ExecuteSwapResponse{Status: "failed", Message: "Invalid input amount format"}, err
	}

	// Determine final recipient (defaults to userAddress if not specified)
	finalRecipient := req.RecipientAddress
	if finalRecipient == "" {
		finalRecipient = req.UserAddress
	}

	var ibcTxHash string
	currentChainIDForSwap := req.FromChainID
	coinForNextStep := amountInCoin

	// 2. Perform IBC transfer if needed
	if chosenRoute.NeedsIBC {
		log.Printf("Route requires IBC: %s -> %s", req.FromChainID, chosenRoute.ChainIDSwappingOn)
		// For MVP, the recipient of the IBC transfer is the UserAddress on the target chain.
		// The sender of the IBC message is the operator key on req.FromChainID.
		ibcTxHash, err = executeIBCTransferViaIgnite(ctx,
			req.FromChainID, chosenRoute.ChainIDSwappingOn,
			req.UserAddress, // This should map to OperatorKeyName on FromChainID
			req.UserAddress, // Tokens arrive at UserAddress on target chain
			amountInCoin,
		)
		if err != nil {
			// executeIBCTransferViaIgnite now returns txhash even on error if broadcast was attempted
			return models.ExecuteSwapResponse{Status: "ibc_failed", Message: "IBC transfer failed: " + err.Error(), IbcTxHash: ibcTxHash}, err
		}
		log.Printf("IBC transfer submitted. TxHash: %s. Waiting for relay (MVP sleep)...", ibcTxHash)
		time.Sleep(30 * time.Second) // MVP HACK: Wait for Hermes

		currentChainIDForSwap = chosenRoute.ChainIDSwappingOn
		// MVP Simplification: Use original FromToken denom. Real system needs ibc/HASH.
		coinForNextStep = sdk.NewCoin(req.FromToken, amountInCoin.Amount)
	}

	// 3. Perform DEX swap on the target chain (either source or IBC destination)
	log.Printf("Performing DEX swap on chain %s for token %s to %s", currentChainIDForSwap, coinForNextStep.Denom, req.ToToken)
	// The userAddress is who initiates the swap on the currentChainIDForSwap (using operator key for that chain)
	swapResp, err := executeDexSwapViaIgnite(ctx,
		currentChainIDForSwap,
		req.UserAddress, // Maps to operator key on currentChainIDForSwap
		coinForNextStep,
		req.ToToken,
		// finalRecipient, // TODO: MockDEX doesn't take recipient, need post-swap transfer if different
	)
	swapResp.IbcTxHash = ibcTxHash // Carry over IBC hash if it happened

	if err != nil {
		// swapResp already contains status and message from executeDexSwapViaIgnite
		return swapResp, err
	}

	// MVP: If finalRecipient is different from req.UserAddress, a final bank send would be needed here.
	// For now, assume swap result goes to req.UserAddress or DEX handles final recipient.
	if finalRecipient != req.UserAddress {
		swapResp.Message += fmt.Sprintf(" (Note: Final tokens at %s. Transfer to %s not yet implemented for MVP.)", req.UserAddress, finalRecipient)
	}

	log.Printf("Swap process finished. Final Status: %s, SwapTxHash: %s", swapResp.Status, swapResp.SwapTxHash)
	return swapResp, nil
}
