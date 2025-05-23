#!/bin/bash

# This script sets up three local wasmd chains for the Cross-Chain DeFi Aggregator

set -e

# Configuration
CHAIN_DIR_ROOT="$HOME/.aggregator_chains"

# Chain A Configuration
CHAIN_A_ID="chain-a"
CHAIN_A_DIR="$CHAIN_DIR_ROOT/$CHAIN_A_ID"
CHAIN_A_DENOM="tokena"
CHAIN_A_RPC_PORT="26657"
CHAIN_A_P2P_PORT="26656"
CHAIN_A_GRPC_PORT="9090"
CHAIN_A_API_PORT="1317"

# Chain B Configuration
CHAIN_B_ID="chain-b"
CHAIN_B_DIR="$CHAIN_DIR_ROOT/$CHAIN_B_ID"
CHAIN_B_DENOM="tokenb"
CHAIN_B_RPC_PORT="27657"
CHAIN_B_P2P_PORT="27656"
CHAIN_B_GRPC_PORT="9190"
CHAIN_B_API_PORT="1318"

# Chain C Configuration
CHAIN_C_ID="chain-c"
CHAIN_C_DIR="$CHAIN_DIR_ROOT/$CHAIN_C_ID"
CHAIN_C_DENOM="tokenc"
CHAIN_C_RPC_PORT="28657"
CHAIN_C_P2P_PORT="28656"
CHAIN_C_GRPC_PORT="9290"
CHAIN_C_API_PORT="1319"

# Key names
VALIDATOR_KEY="validator"
USER_KEY="user"
ADMIN_KEY="admin"
RELAYER_KEY="relayer"

# Function to setup a single chain
setup_chain() {
    local CHAIN_ID=$1
    local CHAIN_DIR=$2
    local DENOM=$3
    local RPC_PORT=$4
    local P2P_PORT=$5
    local GRPC_PORT=$6
    local API_PORT=$7

    echo "--- Setting up chain $CHAIN_ID ---"
    
    # Clean up previous data
    rm -rf "$CHAIN_DIR"
    mkdir -p "$CHAIN_DIR"

    # Initialize chain
    wasmd init "node-$CHAIN_ID" --chain-id "$CHAIN_ID" --home "$CHAIN_DIR"

    # Modify genesis to add custom tokens
    # This is simplified - in a real setup, you'd need to properly modify the genesis
    # to include your custom tokens
    sed -i.bak "s/\"stake\"/\"$DENOM\"/g" "$CHAIN_DIR/config/genesis.json"

    # Configure ports
    sed -i.bak "s/\"tcp:\/\/127.0.0.1:26657\"/\"tcp:\/\/0.0.0.0:$RPC_PORT\"/g" "$CHAIN_DIR/config/config.toml"
    sed -i.bak "s/\"tcp:\/\/127.0.0.1:26656\"/\"tcp:\/\/0.0.0.0:$P2P_PORT\"/g" "$CHAIN_DIR/config/config.toml"
    sed -i.bak "s/\"tcp:\/\/localhost:1317\"/\"tcp:\/\/0.0.0.0:$API_PORT\"/g" "$CHAIN_DIR/config/app.toml"
    sed -i.bak "s/\"localhost:9090\"/\"0.0.0.0:$GRPC_PORT\"/g" "$CHAIN_DIR/config/app.toml"

    # Enable API and gRPC
    sed -i.bak 's/enable = false/enable = true/g' "$CHAIN_DIR/config/app.toml"

    # Add keys
    wasmd keys add "$VALIDATOR_KEY" --keyring-backend test --home "$CHAIN_DIR"
    wasmd keys add "$USER_KEY" --keyring-backend test --home "$CHAIN_DIR"
    wasmd keys add "$ADMIN_KEY" --keyring-backend test --home "$CHAIN_DIR"
    wasmd keys add "$RELAYER_KEY" --keyring-backend test --home "$CHAIN_DIR"

    # Get addresses
    VALIDATOR_ADDR=$(wasmd keys show "$VALIDATOR_KEY" -a --keyring-backend test --home "$CHAIN_DIR")
    USER_ADDR=$(wasmd keys show "$USER_KEY" -a --keyring-backend test --home "$CHAIN_DIR")
    ADMIN_ADDR=$(wasmd keys show "$ADMIN_KEY" -a --keyring-backend test --home "$CHAIN_DIR")
    RELAYER_ADDR=$(wasmd keys show "$RELAYER_KEY" -a --keyring-backend test --home "$CHAIN_DIR")

    # Add genesis accounts
    wasmd genesis add-genesis-account "$VALIDATOR_ADDR" "1000000000$DENOM,1000000000stake" --home "$CHAIN_DIR"
    wasmd genesis add-genesis-account "$USER_ADDR" "1000000000$DENOM,1000000000stake" --home "$CHAIN_DIR"
    wasmd genesis add-genesis-account "$ADMIN_ADDR" "1000000000$DENOM,1000000000stake" --home "$CHAIN_DIR"
    wasmd genesis add-genesis-account "$RELAYER_ADDR" "1000000000$DENOM,1000000000stake" --home "$CHAIN_DIR"

    # Create validator gentx
    wasmd genesis gentx "$VALIDATOR_KEY" "100000000stake" --chain-id "$CHAIN_ID" --keyring-backend test --home "$CHAIN_DIR"
    
    # Collect gentxs
    wasmd genesis collect-gentxs --home "$CHAIN_DIR"

    echo "Chain $CHAIN_ID setup complete."
    echo "Validator address: $VALIDATOR_ADDR"
    echo "User address: $USER_ADDR"
    echo "Admin address: $ADMIN_ADDR"
    echo "Relayer address: $RELAYER_ADDR"
    echo
}

# Setup all three chains
setup_chain "$CHAIN_A_ID" "$CHAIN_A_DIR" "$CHAIN_A_DENOM" "$CHAIN_A_RPC_PORT" "$CHAIN_A_P2P_PORT" "$CHAIN_A_GRPC_PORT" "$CHAIN_A_API_PORT"
setup_chain "$CHAIN_B_ID" "$CHAIN_B_DIR" "$CHAIN_B_DENOM" "$CHAIN_B_RPC_PORT" "$CHAIN_B_P2P_PORT" "$CHAIN_B_GRPC_PORT" "$CHAIN_B_API_PORT"
setup_chain "$CHAIN_C_ID" "$CHAIN_C_DIR" "$CHAIN_C_DENOM" "$CHAIN_C_RPC_PORT" "$CHAIN_C_P2P_PORT" "$CHAIN_C_GRPC_PORT" "$CHAIN_C_API_PORT"

echo "All chains setup complete."
echo
echo "To start Chain A: wasmd start --home \"$CHAIN_A_DIR\""
echo "To start Chain B: wasmd start --home \"$CHAIN_B_DIR\""
echo "To start Chain C: wasmd start --home \"$CHAIN_C_DIR\""
echo
echo "Next: Run setup_ibc_channels.sh to establish IBC connections between chains."
echo "Then: Run deploy_contracts.sh to deploy DEX contracts."
