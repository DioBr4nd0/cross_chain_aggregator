package utils

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StringToSDKDec converts a string to sdk.Dec.
func StringToSDKDec(s string) (sdk.Dec, error) {
	dec, err := sdk.NewDecFromStr(s)
	if err != nil {
		return sdk.Dec{}, err
	}
	return dec, nil
}

// StringToSDKInt converts a string to sdk.Int.
func StringToSDKInt(s string) (sdk.Int, error) {
	val, ok := sdk.NewIntFromString(s)
	if !ok {
		return sdk.Int{}, fmt.Errorf("failed to convert string '%s' to sdk.Int", s)
	}
	return val, nil
}
