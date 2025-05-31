package types

import (
	"fmt"

	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
)
type RawContractMessage []byte

type MsgExecuteContract struct {
	// Sender is the that actor that signed the messages
	Sender string `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	// Contract is the address of the smart contract
	Contract string `protobuf:"bytes,2,opt,name=contract,proto3" json:"contract,omitempty"`
	// Msg json encoded message to be passed to the contract
	Msg RawContractMessage `protobuf:"bytes,3,opt,name=msg,proto3,casttype=RawContractMessage" json:"msg,omitempty"`
	// Funds coins that are transferred to the contract on execution
	Funds github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,5,rep,name=funds,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"funds"`
}

func (msg *MsgExecuteContract) ValidateBasic() error {
	return nil
}


// ProtoMessage implements proto.Message
func (m MsgExecuteContract) ProtoMessage() {}

// Reset implements proto.Message
func (m MsgExecuteContract) Reset() {}

// String implements proto.Message
func (m MsgExecuteContract) String() string {
    return fmt.Sprintf("MsgExecuteContract{Sender: %s, Contract: %s, Funds: %s}", 
        m.Sender, m.Contract, m.Funds.String())
}

// ValidateBasic implements sdk.Msg