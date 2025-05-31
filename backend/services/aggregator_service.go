package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log" // Added for debugging
	"sort"

	"cosmos_defi_aggregator/config"
	"cosmos_defi_aggregator/cosmos"
	"cosmos_defi_aggregator/models"
	"cosmos_defi_aggregator/utils"
	sdk "cosmos_defi_aggregator/math"
)

// GetDexRate remains the same (queries a specific DEX)
func GetDexRate(ctx context.Context, chainID, fromToken, toToken string) (sdk.LegacyDec, error) {
	chainCfg, found := config.GetChain(chainID)
	if !found {
		return sdk.LegacyDec{}, fmt.Errorf("chain config not found for ID: %s", chainID)
	}

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
		return sdk.LegacyDec{}, fmt.Errorf("failed to marshal GetRate query: %w", err)
	}

	log.Printf("[GetDexRate] Querying chain %s, contract %s, for %s -> %s", chainID, chainCfg.DexContract, fromToken, toToken)
	responseData, err := cosmos.QueryContractWithIgnite(ctx, chainID, chainCfg.DexContract, queryBytes)
	if err != nil {
		// Log the actual error from QueryContractWithIgnite, which includes "No rate for..." if that's the case
		log.Printf("[GetDexRate] Error from QueryContractWithIgnite for chain %s (%s -> %s): %v", chainID, fromToken, toToken, err)
		return sdk.LegacyDec{}, err // Propagate the error correctly
	}

	var rateResp struct {
		Rate string `json:"rate"`
	}
	if err := json.Unmarshal(responseData, &rateResp); err != nil {
		return sdk.LegacyDec{}, fmt.Errorf("failed to unmarshal rate response from %s DEX: %w. Data: %s", chainID, err, string(responseData))
	}

	rate, err := utils.StringToSDKDec(rateResp.Rate)
	if err != nil {
		return sdk.LegacyDec{}, fmt.Errorf("invalid rate format '%s' from %s DEX: %w", rateResp.Rate, chainID, err)
	}
	log.Printf("[GetDexRate] Success on chain %s for %s -> %s: Rate %s", chainID, fromToken, toToken, rate.String())
	return rate, nil
}


// GetRatesForPairAcrossChains remains the same
func GetRatesForPairAcrossChains(ctx context.Context, fromToken, toToken string) ([]models.DexRateInfo, error) {
	var allRates []models.DexRateInfo
	for chainID := range config.GlobalAppConfig.Chains {
		rate, err := GetDexRate(ctx, chainID, fromToken, toToken)
		if err == nil {
			allRates = append(allRates, models.DexRateInfo{
				ChainID:   chainID,
				FromToken: fromToken,
				ToToken:   toToken,
				Rate:      rate.String(),
			})
		} else {
			// This log is fine, it just means a direct rate wasn't found on this specific chain for this pair
			log.Printf("[GetRatesForPairAcrossChains] No direct rate for %s->%s on chain %s: %v\n", fromToken, toToken, chainID, err)
		}
	}
	if len(allRates) == 0 {
		// This error should only be returned if NO chain offers a direct rate for this pair.
		// It's okay for some chains not to have it.
		return nil, fmt.Errorf("no DEXes found offering a direct rate for %s to %s", fromToken, toToken)
	}
	return allRates, nil
}


// FindBestSwapRoute - MAJOR REVISIONS HERE
func FindBestSwapRoute(ctx context.Context, req models.BestRouteRequest) (models.BestRouteResponse, error) {
	log.Printf("[FindBestSwapRoute] Request: %+v", req)
	sourceChainCfg, found := config.GetChain(req.FromChainID)
	if !found {
		return models.BestRouteResponse{}, fmt.Errorf("source chain %s not configured", req.FromChainID)
	}

	amountInDec, err := utils.StringToSDKDec(req.AmountIn)
	if err != nil {
		return models.BestRouteResponse{}, fmt.Errorf("invalid amountIn '%s': %w", req.AmountIn, err)
	}

	var possibleRoutes []models.RouteDetail

	// --- Option 1: Direct Swap on the Source Chain ---
	// Can we swap req.FromToken for req.ToToken directly on sourceChainCfg?
	log.Printf("[FindBestSwapRoute] Trying direct swap on source chain %s for %s -> %s", sourceChainCfg.ID, req.FromToken, req.ToToken)
	directRateOnSource, err := GetDexRate(ctx, sourceChainCfg.ID, req.FromToken, req.ToToken)
	if err == nil {
		amountOut := amountInDec.Mul(directRateOnSource)
		possibleRoutes = append(possibleRoutes, models.RouteDetail{
			ChainIDSwappingOn: sourceChainCfg.ID,
			Rate:              directRateOnSource.String(),
			AmountOut:         amountOut.TruncateInt().String(),
			Steps:             []string{fmt.Sprintf("Swap %s for %s on %s DEX", req.FromToken, req.ToToken, sourceChainCfg.Name)},
			NeedsIBC:          false,
		})
		log.Printf("[FindBestSwapRoute] Found direct route on %s: Rate %s, Out %s", sourceChainCfg.ID, directRateOnSource.String(), amountOut.TruncateInt().String())
	} else {
		log.Printf("[FindBestSwapRoute] No direct rate on %s for %s -> %s: %v", sourceChainCfg.ID, req.FromToken, req.ToToken, err)
	}

	// --- Option 2: IBC Transfer to another chain, then Swap there ---
	// Iterate over ALL chains in your config as potential target chains for the swap.
	for targetChainIDForSwap, targetChainCfgForSwap := range config.GlobalAppConfig.Chains {
		if targetChainIDForSwap == sourceChainCfg.ID {
			// This case is covered by "Direct Swap on Source Chain" if req.ToToken is supported directly there.
			// However, if req.ToToken is NATIVE to sourceChainCfg, and req.FromToken is an IBC token on sourceChainCfg,
			// a direct swap is the only option.
			// The direct swap check above handles cases like:
			//   ualpha -> ibc/BetaOnAlpha (on AlphaNet)
			//   ibc/BetaOnAlpha -> ualpha (on AlphaNet)
			continue
		}

		// Check if an IBC path exists from sourceChainCfg.ID to targetChainIDForSwap
		_, ibcPathExists := config.GetIBCChannel(sourceChainCfg.ID, targetChainIDForSwap)
		if !ibcPathExists {
			log.Printf("[FindBestSwapRoute] No IBC path from %s to %s. Skipping this target.", sourceChainCfg.ID, targetChainIDForSwap)
			continue
		}

		// If we IBC req.FromToken from sourceChainCfg.ID to targetChainIDForSwap:
		// What will its denom be on targetChainIDForSwap?
		// For MVP, we use a conceptual mapping based on your config.
		// e.g., if req.FromToken="ualpha" (from "alphanet-1") and targetChainIDForSwap="betanet-1",
		// the conceptual denom on "betanet-1" is "ibc/AlphaOnBeta".
		var fromTokenDenomOnTargetDEX string
		// This mapping logic needs to be robust or your backend config needs to provide it.
		// Example conceptual mapping:
		if req.FromToken == "ualpha" && targetChainCfgForSwap.ID == "betanet-1" {
			fromTokenDenomOnTargetDEX = "ibc/AlphaOnBeta"
		} else if req.FromToken == "ualpha" && targetChainCfgForSwap.ID == "gammanet-1" {
			fromTokenDenomOnTargetDEX = "ibc/AlphaOnGamma"
		} else if req.FromToken == "ubeta" && targetChainCfgForSwap.ID == "alphanet-1" {
			fromTokenDenomOnTargetDEX = "ibc/BetaOnAlpha"
		} else if req.FromToken == "ubeta" && targetChainCfgForSwap.ID == "gammanet-1" {
			fromTokenDenomOnTargetDEX = "ibc/BetaOnGamma"
		} else if req.FromToken == "ugamma" && targetChainCfgForSwap.ID == "alphanet-1" {
			fromTokenDenomOnTargetDEX = "ibc/GammaOnAlpha"
		} else if req.FromToken == "ugamma" && targetChainCfgForSwap.ID == "betanet-1" {
			fromTokenDenomOnTargetDEX = "ibc/GammaOnBeta"
		} else {
			// If req.FromToken is already an IBC token (e.g., "ibc/BetaOnAlpha") and we are IBCing it *again*,
			// this simple conceptual mapping breaks. A real system handles multi-hop IBC vouchers.
			// For MVP, we assume req.FromToken is a NATIVE token of sourceChainCfg when considering IBC path.
			// If req.FromToken is itself an IBC token on the source chain, this path gets more complex.
			// Let's refine this: if fromToken is ALREADY an IBC token, and we are IBCing it *back* to its source,
			// it becomes native. If we are IBCing it to a *third* chain, that's complex.

			// Simplified for MVP: if fromToken is native to source, map it conceptually.
			// If fromToken is already an IBC denom on source, this logic path might not be suitable for simple aggregator.
			if (req.FromToken == sourceChainCfg.NativeToken) {
				// Create conceptual denom like "ibc/NativeTokenNameOnTargetChainName"
				// Example: "ibc/" + strings.Title(strings.TrimPrefix(req.FromToken, "u")) + "On" + strings.Title(targetChainCfgForSwap.Name)
				// This relies on consistent naming. Let's stick to explicit maps for clarity.
				log.Printf("[FindBestSwapRoute] Unhandled IBC denom mapping for FromToken '%s' being sent from '%s' to '%s'. Skipping.", req.FromToken, sourceChainCfg.ID, targetChainIDForSwap)
				continue
			} else {
				// If req.FromToken is an IBC token like "ibc/BetaOnAlpha" on "alphanet-1",
				// and we are trying to IBC it to "gammanet-1" (a third chain)...
				// This is a multi-hop IBC scenario for the original "ubeta".
				// An MVP aggregator might not support this complex path finding.
				// Or, if we IBC "ibc/BetaOnAlpha" back to "betanet-1", it becomes "ubeta".
				if targetChainIDForSwap == getOriginChainOfIBCToken(req.FromToken, sourceChainCfg.ID) { // pseudo-function
					fromTokenDenomOnTargetDEX = getBaseDenomOfIBCToken(req.FromToken) // pseudo-function, e.g. "ubeta"
				} else {
					log.Printf("[FindBestSwapRoute] Complex IBC hop for FromToken '%s' from '%s' to '%s'. Skipping for MVP.", req.FromToken, sourceChainCfg.ID, targetChainIDForSwap)
					continue
				}
			}
		}


		if fromTokenDenomOnTargetDEX == "" {
			log.Printf("[FindBestSwapRoute] Could not determine IBC denom for %s on %s. Skipping.", req.FromToken, targetChainIDForSwap)
			continue
		}
		
		log.Printf("[FindBestSwapRoute] Trying IBC path: %s (%s) -> %s (%s), then swap %s -> %s on %s DEX",
			req.FromToken, sourceChainCfg.Name,
			fromTokenDenomOnTargetDEX, targetChainCfgForSwap.Name,
			fromTokenDenomOnTargetDEX, req.ToToken, targetChainCfgForSwap.Name)

		// Now, can we swap fromTokenDenomOnTargetDEX for req.ToToken on targetChainCfgForSwap's DEX?
		rateOnTarget, err := GetDexRate(ctx, targetChainCfgForSwap.ID, fromTokenDenomOnTargetDEX, req.ToToken)
		if err == nil {
			// Assume IBC transfer fee is negligible for rate comparison (MVP)
			amountOut := amountInDec.Mul(rateOnTarget)
			possibleRoutes = append(possibleRoutes, models.RouteDetail{
				ChainIDSwappingOn: targetChainCfgForSwap.ID, // Swap happens on this target chain
				Rate:              rateOnTarget.String(),      // This is the DEX rate on the target chain
				AmountOut:         amountOut.TruncateInt().String(),
				Steps: []string{
					fmt.Sprintf("IBC Transfer %s from %s to %s (becomes %s)", req.FromToken, sourceChainCfg.Name, targetChainCfgForSwap.Name, fromTokenDenomOnTargetDEX),
					fmt.Sprintf("Swap %s for %s on %s DEX", fromTokenDenomOnTargetDEX, req.ToToken, targetChainCfgForSwap.Name),
				},
				NeedsIBC: true,
			})
			log.Printf("[FindBestSwapRoute] Found IBC route via %s: Rate %s, Out %s", targetChainCfgForSwap.ID, rateOnTarget.String(), amountOut.TruncateInt().String())
		} else {
			log.Printf("[FindBestSwapRoute] No rate on %s for %s -> %s: %v", targetChainCfgForSwap.ID, fromTokenDenomOnTargetDEX, req.ToToken, err)
		}
	}


	if len(possibleRoutes) == 0 {
		log.Printf("[FindBestSwapRoute] No viable routes found for %s -> %s from %s", req.FromToken, req.ToToken, req.FromChainID)
		return models.BestRouteResponse{}, fmt.Errorf("no viable swap routes found for %s to %s from chain %s", req.FromToken, req.ToToken, req.FromChainID)
	}

	// Sort routes by the highest AmountOut
	sort.Slice(possibleRoutes, func(i, j int) bool {
		amountOutI, _ := utils.StringToSDKInt(possibleRoutes[i].AmountOut) // Error ignored for sorting simplicity
		amountOutJ, _ := utils.StringToSDKInt(possibleRoutes[j].AmountOut)
		return amountOutI.GT(amountOutJ) // GT for descending order (more is better)
	})

	log.Printf("[FindBestSwapRoute] Found %d possible routes. Best route: %+v", len(possibleRoutes), possibleRoutes[0])

	return models.BestRouteResponse{
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		AmountIn:    req.AmountIn,
		BestRoute:   possibleRoutes[0],
		OtherRoutes: possibleRoutes[1:],
	}, nil
}

// Placeholder pseudo-functions, you'd need real logic if handling complex IBC denoms
func getOriginChainOfIBCToken(ibcTokenDenom string, currentChainId string) string {
    // e.g. "ibc/BetaOnAlpha" on "alphanet-1" originated from "betanet-1"
    // This needs a robust parsing or mapping.
    // For MVP, we might not even need this if we stick to native -> conceptual IBC denom.
    if ibcTokenDenom == "ibc/BetaOnAlpha" && currentChainId == "alphanet-1" { return "betanet-1" }
    if ibcTokenDenom == "ibc/GammaOnAlpha" && currentChainId == "alphanet-1" { return "gammanet-1" }
    // ... add all mappings from your config.SupportedTokens
    return ""
}

func getBaseDenomOfIBCToken(ibcTokenDenom string) string {
    // e.g. "ibc/BetaOnAlpha" -> "ubeta"
    if ibcTokenDenom == "ibc/BetaOnAlpha" { return "ubeta" }
    if ibcTokenDenom == "ibc/GammaOnAlpha" { return "ugamma" }
    // ... add all mappings
    return ""
}

