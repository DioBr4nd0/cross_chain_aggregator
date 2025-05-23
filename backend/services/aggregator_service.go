package services

import (
	"fmt"
	"sort"

	"cosmos_defi_aggregator/config" // Adjust import path
	"cosmos_defi_aggregator/models" // Adjust import path
	"cosmos_defi_aggregator/utils"  // Adjust import path

	"cosmossdk.io/math"
)

// SimulatedDexRate represents a fixed rate for a pair on a simulated DEX
type SimulatedDexRate struct {
	FromDenom string
	ToDenom   string
	Rate      string // e.g., "10.5" (meaning 1 FromDenom = 10.5 ToDenom)
}

// SimulatedDexRates holds all predefined rates for all DEXes
// Key: chainID, Value: map of "from:to" -> rate_string
var simulatedChainDexRates = map[string]map[string]string{
	"chain-a": {
		"uwasma:tokenb": "10.0", // 1 uwasma = 10 tokenb on chain-a
		"tokenb:uwasma": "0.1",
		"uwasma:tokenc": "5.0",
		"tokenc:uwasma": "0.2",
	},
	"chain-b": {
		"uwasmb:tokena": "0.09", // 1 uwasmb = 0.09 tokena (uwasma) on chain-b
		"tokena:uwasmb": "11.0", // IBC'd tokena (uwasma)
		"uwasmb:tokenc": "2.0",
		"tokenc:uwasmb": "0.5",
	},
	"chain-c": {
		"uwasmc:tokena": "0.15", // 1 uwasmc = 0.15 tokena (uwasma)
		"tokena:uwasmc": "6.5",  // IBC'd tokena (uwasma)
		"uwasmc:tokenb": "0.4",  // 1 uwasmc = 0.4 tokenb
		"tokenb:uwasmc": "2.5",  // IBC'd tokenb
	},
}

// QueryDexRate SIMULATES querying a specific DEX contract for an exchange rate.
// In a real implementation, this would make an RPC/gRPC call to the chain.
func QueryDexRate(chainID, dexContractAddress, fromToken, toToken string) (math.LegacyDec, error) {
	chainRates, ok := simulatedChainDexRates[chainID]
	if !ok {
		return math.LegacyDec{}, fmt.Errorf("no DEX rates defined for chain %s", chainID)
	}
	rateKey := fmt.Sprintf("%s:%s", fromToken, toToken)
	rateStr, ok := chainRates[rateKey]
	if !ok {
		return math.LegacyDec{}, fmt.Errorf("no rate found for %s to %s on chain %s DEX", fromToken, toToken, chainID)
	}

	rate, err := utils.StringToSDKDec(rateStr)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("invalid rate format '%s' for %s to %s on chain %s: %w", rateStr, fromToken, toToken, chainID, err)
	}
	return rate, nil
}

// GetRatesForPairAcrossChains queries all configured DEXes for a given token pair.
func GetRatesForPairAcrossChains(fromToken, toToken string) ([]models.DexRateInfo, error) {
	var availableRates []models.DexRateInfo

	for _, chain :=       range config.GlobalChains {
		// Check if the chain's DEX *could* support this swap
		// (Simple check: does it support fromToken and toToken individually?
		// A real DEX might not have a direct pool.)
		supportsFrom := false
		supportsTo := false
		for _, supported := range chain.SupportedTokens {
			if supported == fromToken {
				supportsFrom = true
			}
			if supported == toToken {
				supportsTo = true
			}
		}

		if supportsFrom && supportsTo {
			rate, err := QueryDexRate(chain.ID, chain.DexContract, fromToken, toToken)
			if err == nil { // If a rate exists for this direct pair on this DEX
				availableRates = append(availableRates, models.DexRateInfo{
					ChainID:    chain.ID,
					DexAddress: chain.DexContract,
					FromToken:  fromToken,
					ToToken:    toToken,
					Rate:       rate.String(),
				})
			}
			// For a hackathon, we are only considering direct swaps on each DEX.
			// Multi-hop on a single DEX is out of scope for this simplification.
		}
	}

	if len(availableRates) == 0 {
		return nil, fmt.Errorf("no direct swap rates found for %s to %s on any configured chain", fromToken, toToken)
	}
	return availableRates, nil
}


// FindBestRouteForSwap calculates the best way to swap fromToken to toToken.
// For this hackathon version, it will consider:
// 1. Direct swap on fromChainID (if possible).
// 2. IBC transfer fromToken to another chain, then swap there.
// It does NOT do multi-hop swaps (A -> B -> C) for this simplified version.
func FindBestRouteForSwap(req models.BestRouteRequest) (models.BestRouteResponse, error) {
	fromChain, fromChainExists := config.GetChainByID(req.FromChainID)
	if !fromChainExists {
		return models.BestRouteResponse{}, fmt.Errorf("source chain ID '%s' not configured", req.FromChainID)
	}

	amountInDec, err := utils.StringToSDKDec(req.AmountIn) // Using Dec for intermediate calcs
	if err != nil {
		return models.BestRouteResponse{}, fmt.Errorf("invalid amountIn: %w", err)
	}

	var possibleRoutes []models.RouteDetail

	// Option 1: Swap directly on the fromChainID
	// Check if fromChain's DEX supports this direct swap
	directSwapRate, err := QueryDexRate(fromChain.ID, fromChain.DexContract, req.FromToken, req.ToToken)
	if err == nil { // If direct swap is possible
		amountOut := amountInDec.Mul(directSwapRate)
		possibleRoutes = append(possibleRoutes, models.RouteDetail{
			ChainIDSwappingOn: fromChain.ID,
			Rate:              directSwapRate.String(),
			AmountOut:         amountOut.TruncateInt().String(),
			Steps:             []string{fmt.Sprintf("Swap %s for %s on %s DEX (%s)", req.FromToken, req.ToToken, fromChain.Name, fromChain.DexContract)},
			NeedsIBC:          false,
		})
	}

	// Option 2: IBC transfer fromToken to other chains and swap there
	for _, targetChain := range config.GlobalChains {
		if targetChain.ID == fromChain.ID {
			continue // Already handled by Option 1
		}

		// Check if targetChain's DEX supports swapping the (potentially IBC'd) fromToken for toToken
		// For IBC, the fromToken on targetChain would have an IBC denom.
		// Simplified: assume if targetChain DEX supports `req.FromToken` (as an IBC voucher) and `req.ToToken`.
		// A real version would construct the IBC denom: `ibc/.../<fromTokenDenom>`
		ibcFromTokenDenom := req.FromToken // Simplified: actual IBC denom is complex
		if targetChain.ID != req.FromChainID {
			// Construct a hypothetical IBC denom if we were to send it.
			// For this simulation, we'll just check if target DEX supports original `fromToken` and `toToken`
			// This is a MAJOR simplification for the hackathon.
			// A real system would have to look up the IBC denom of fromToken on targetChain.
		}


		swapRateOnTargetChain, err := QueryDexRate(targetChain.ID, targetChain.DexContract, ibcFromTokenDenom, req.ToToken)
		if err == nil { // If swap is possible on targetChain
			// Assume IBC transfer fee is negligible or handled by user separately for this simulation
			amountOut := amountInDec.Mul(swapRateOnTargetChain) // AmountIn is the same after IBC (ignoring fees)
			possibleRoutes = append(possibleRoutes, models.RouteDetail{
				ChainIDSwappingOn: targetChain.ID,
				Rate:              swapRateOnTargetChain.String(), // This is the rate on the target DEX
				AmountOut:         amountOut.TruncateInt().String(),
				Steps: []string{
					fmt.Sprintf("IBC Transfer %s from %s to %s", req.FromToken, fromChain.Name, targetChain.Name),
					fmt.Sprintf("Swap %s for %s on %s DEX (%s)", ibcFromTokenDenom, req.ToToken, targetChain.Name, targetChain.DexContract),
				},
				NeedsIBC: true,
			})
		}
	}

	if len(possibleRoutes) == 0 {
		return models.BestRouteResponse{}, fmt.Errorf("no viable swap routes found for %s to %s starting from %s", req.FromToken, req.ToToken, req.FromChainID)
	}

	// Sort routes to find the one that yields the most AmountOut
	sort.Slice(possibleRoutes, func(i, j int) bool {
		amountOutI, _ := utils.StringToSDKInt(possibleRoutes[i].AmountOut)
		amountOutJ, _ := utils.StringToSDKInt(possibleRoutes[j].AmountOut)
		return amountOutI.GT(amountOutJ) // Sort descending by AmountOut
	})

	bestRoute := possibleRoutes[0]
	otherRoutes := []models.RouteDetail{}
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
