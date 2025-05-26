package utils

import (
	"fmt"

	"cosmossdk.io/math"
)

// StringToSDKInt converts a string amount to sdk.Int.
func StringToSDKInt(amountStr string) (math.Int, error) {
	i, ok := math.NewIntFromString(amountStr)
	if !ok {
		return math.Int{}, fmt.Errorf("failed to convert string to sdk Int")
	}
	
	return i, nil
}

// StringToSDKDec converts a string rate to sdk.Dec.
func StringToSDKDec(rateStr string) (math.LegacyDec, error) {
	d, err := math.LegacyNewDecFromStr(rateStr)
	if err != nil {
		return math.LegacyDec{}, err
	}
	return d, nil
}
