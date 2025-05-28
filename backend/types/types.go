package types

import (
	"context"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	grpc "google.golang.org/grpc"
)

type QueryClient interface {
	// ContractInfo gets the contract meta data
	ContractInfo(ctx context.Context, in *QueryContractInfoRequest, opts ...grpc.CallOption) (*QueryContractInfoResponse, error)
	// ContractHistory gets the contract code history
	ContractHistory(ctx context.Context, in *QueryContractHistoryRequest, opts ...grpc.CallOption) (*QueryContractHistoryResponse, error)
	// ContractsByCode lists all smart contracts for a code id
	ContractsByCode(ctx context.Context, in *QueryContractsByCodeRequest, opts ...grpc.CallOption) (*QueryContractsByCodeResponse, error)
	// AllContractState gets all raw store data for a single contract
	AllContractState(ctx context.Context, in *QueryAllContractStateRequest, opts ...grpc.CallOption) (*QueryAllContractStateResponse, error)
	// RawContractState gets single key from the raw store data of a contract
	RawContractState(ctx context.Context, in *QueryRawContractStateRequest, opts ...grpc.CallOption) (*QueryRawContractStateResponse, error)
	// SmartContractState get smart query result from the contract
	SmartContractState(ctx context.Context, in *QuerySmartContractStateRequest, opts ...grpc.CallOption) (*QuerySmartContractStateResponse, error)
	// Code gets the binary code and metadata for a single wasm code
	Code(ctx context.Context, in *QueryCodeRequest, opts ...grpc.CallOption) (*QueryCodeResponse, error)
	// Codes gets the metadata for all stored wasm codes
	Codes(ctx context.Context, in *QueryCodesRequest, opts ...grpc.CallOption) (*QueryCodesResponse, error)
	// CodeInfo gets the metadata for a single wasm code
	CodeInfo(ctx context.Context, in *QueryCodeInfoRequest, opts ...grpc.CallOption) (*QueryCodeInfoResponse, error)
	// PinnedCodes gets the pinned code ids
	PinnedCodes(ctx context.Context, in *QueryPinnedCodesRequest, opts ...grpc.CallOption) (*QueryPinnedCodesResponse, error)
	// Params gets the module params
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// ContractsByCreator gets the contracts by creator
	ContractsByCreator(ctx context.Context, in *QueryContractsByCreatorRequest, opts ...grpc.CallOption) (*QueryContractsByCreatorResponse, error)
	// WasmLimitsConfig gets the configured limits for static validation of Wasm
	// files, encoded in JSON.
	WasmLimitsConfig(ctx context.Context, in *QueryWasmLimitsConfigRequest, opts ...grpc.CallOption) (*QueryWasmLimitsConfigResponse, error)
	// BuildAddress builds a contract address
	BuildAddress(ctx context.Context, in *QueryBuildAddressRequest, opts ...grpc.CallOption) (*QueryBuildAddressResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}