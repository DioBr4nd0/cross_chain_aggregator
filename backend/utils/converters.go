package utils

import (
	"fmt"

	mathtypes "cosmos_defi_aggregator/math"
)

// StringToSDKDec converts a string to sdk.Dec.
func StringToSDKDec(s string) (mathtypes.LegacyDec, error) {
	dec, err := mathtypes.LegacyNewDecFromStr(s)
	// dec, err := sdk.NewDecFromStr(s)
	if err != nil {
		return mathtypes.LegacyDec{}, err
	}
	return dec, nil
}

// StringToSDKInt converts a string to sdk.Int.
func StringToSDKInt(s string) (mathtypes.Int, error) {
	val, ok := mathtypes.NewIntFromString(s)
	if !ok {
		return mathtypes.Int{}, fmt.Errorf("failed to convert string '%s' to sdk.Int", s)
	}
	return val, nil
}
