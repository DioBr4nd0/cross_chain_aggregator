package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"cosmos_defi_aggregator/config"
	"cosmos_defi_aggregator/cosmos"
	"cosmos_defi_aggregator/models"
	"cosmos_defi_aggregator/utils"
	"cosmos_defi_aggregator/math"

)

// GetDexRate queries a specific DEX contract for an exchange rate.
func GetDexRate(ctx context.Context, chainID, fromToken, toToken string) (math.LegacyDec, error) {
	chainCfg, found := config.GetChain(chainID)
	if !found {
		return math.LegacyDec{}, fmt.Errorf("chain config not found for ID: %s", chainID)
	}

	// Construct the query message for your mock_dex contract
	queryPayload := struct {
		GetRate struct {
			FromDenom string `json:"from_denom"`
			ToDenom   string `json:"to_denom"`
		} `json:"get_rate"`
	}{
		GetRate: struct {
			FromDenom string `json:"from_denom"`
			ToDenom   string `json:"to_denom"`
		}{FromDenom: fromToken, ToDenom: toToken},
	}
	queryBytes, err := json.Marshal(queryPayload)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("failed to marshal GetRate query: %w", err)
	}

	responseData, err := cosmos.QueryContractWithIgnite(ctx, chainID, chainCfg.DexContract, queryBytes)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("QueryContractWithIgnite failed for %s DEX: %w", chainID, err)
	}

	var rateResp struct {
		Rate string `json:"rate"`
	}
	if err := json.Unmarshal(responseData, &rateResp); err != nil {
		return math.LegacyDec{}, fmt.Errorf("failed to unmarshal rate response from %s DEX: %w. Data: %s", chainID, err, string(responseData))
	}

	rate, err := utils.StringToSDKDec(rateResp.Rate)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("invalid rate format '%s' from %s DEX: %w", rateResp.Rate, chainID, err)
	}
	return rate, nil
}


// GetRatesForPairAcrossChains gathers rates for a token pair from all configured DEXes.
func GetRatesForPairAcrossChains(ctx context.Context, fromToken, toToken string) ([]models.DexRateInfo, error) {
	var allRates []models.DexRateInfo
	for chainID := range config.GlobalAppConfig.Chains {
		rate, err := GetDexRate(ctx, chainID, fromToken, toToken)
		if err == nil { // If rate exists
			allRates = append(allRates, models.DexRateInfo{
				ChainID:   chainID,
				FromToken: fromToken,
				ToToken:   toToken,
				Rate:      rate.String(),
			})
		} else {
			log.Printf("Note: No rate for %s->%s on chain %s: %v\n", fromToken, toToken, chainID, err)
		}
	}
	if len(allRates) == 0 {
		return nil, fmt.Errorf("no DEXes found offering a rate for %s to %s", fromToken, toToken)
	}
	return allRates, nil
}


// FindBestSwapRoute determines the optimal path for a token swap.
// MVP: Considers direct swap on source chain OR (IBC transfer + swap on one other chain).
func FindBestSwapRoute(ctx context.Context, req models.BestRouteRequest) (models.BestRouteResponse, error) {
	sourceChainCfg, found := config.GetChain(req.FromChainID)
	if !found {
		return models.BestRouteResponse{}, fmt.Errorf("source chain %s not configured", req.FromChainID)
	}

	amountInDec, err := utils.StringToSDKDec(req.AmountIn) // Use Dec for precision in rate calcs
	if err != nil {
		return models.BestRouteResponse{}, fmt.Errorf("invalid amountIn '%s': %w", req.AmountIn, err)
	}

	var possibleRoutes []models.RouteDetail

	// Option 1: Direct swap on the source chain
	directRate, err := GetDexRate(ctx, sourceChainCfg.ID, req.FromToken, req.ToToken)
	if err == nil {
		amountOut := amountInDec.Mul(directRate)
		possibleRoutes = append(possibleRoutes, models.RouteDetail{
			ChainIDSwappingOn: sourceChainCfg.ID,
			Rate:              directRate.String(),
			AmountOut:         amountOut.TruncateInt().String(), // Convert to Int string for API
			Steps:             []string{fmt.Sprintf("Swap %s for %s on %s DEX", req.FromToken, req.ToToken, sourceChainCfg.Name)},
			NeedsIBC:          false,
		})
	}

	// Option 2: IBC transfer to another chain, then swap there
	for targetChainID, targetChainCfg := range config.GlobalAppConfig.Chains {
		if targetChainID == sourceChainCfg.ID {
			continue // Skip source chain, already handled
		}

		// Check if an IBC path is configured
		_, ibcPathExists := config.GetIBCChannel(sourceChainCfg.ID, targetChainID)
		if !ibcPathExists {
			continue
		}

		// MVP Simplification: Assume the target DEX has a rate for the *original* FromToken denom.
		// A real implementation needs to handle the `ibc/...` denom for the transferred token.
		tokenToSwapOnTargetDEX := req.FromToken

		rateOnTarget, err := GetDexRate(ctx, targetChainID, tokenToSwapOnTargetDEX, req.ToToken)
		if err == nil {
			// Assume IBC transfer itself has negligible impact on amount for rate comparison simplicity
			amountOut := amountInDec.Mul(rateOnTarget)
			possibleRoutes = append(possibleRoutes, models.RouteDetail{
				ChainIDSwappingOn: targetChainID,
				Rate:              rateOnTarget.String(), // This is the DEX rate on the target chain
				AmountOut:         amountOut.TruncateInt().String(),
				Steps: []string{
					fmt.Sprintf("IBC Transfer %s from %s to %s", req.FromToken, sourceChainCfg.Name, targetChainCfg.Name),
					fmt.Sprintf("Swap %s for %s on %s DEX", tokenToSwapOnTargetDEX, req.ToToken, targetChainCfg.Name),
				},
				NeedsIBC: true,
			})
		}
	}

	if len(possibleRoutes) == 0 {
		return models.BestRouteResponse{}, fmt.Errorf("no viable swap routes found for %s to %s from chain %s", req.FromToken, req.ToToken, req.FromChainID)
	}

	// Sort routes by the highest AmountOut
	sort.Slice(possibleRoutes, func(i, j int) bool {
		amountOutI, _ := utils.StringToSDKInt(possibleRoutes[i].AmountOut)
		amountOutJ, _ := utils.StringToSDKInt(possibleRoutes[j].AmountOut)
		return amountOutI.GT(amountOutJ) // GT for descending order (more is better)
	})

	bestRoute := possibleRoutes[0]
	var otherRoutes []models.RouteDetail
	if len(possibleRoutes) > 1 {
		otherRoutes = possibleRoutes[1:]
	}

	return models.BestRouteResponse{
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		AmountIn:    req.AmountIn,
		BestRoute:   bestRoute,
		OtherRoutes: otherRoutes,
	}, nil
}
