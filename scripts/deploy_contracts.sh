#!/bin/bash

# This script compiles and deploys the Mock DEX contracts to all three chains

set -e

# Configuration
CHAIN_DIR_ROOT="$HOME/.aggregator_chains"
CONTRACT_DIR="../contracts/mock_dex"

# Chain A Configuration
CHAIN_A_ID="chain-a"
CHAIN_A_DIR="$CHAIN_DIR_ROOT/$CHAIN_A_ID"
CHAIN_A_RPC="http://localhost:26657"
CHAIN_A_DENOM="tokena"

# Chain B Configuration
CHAIN_B_ID="chain-b"
CHAIN_B_DIR="$CHAIN_DIR_ROOT/$CHAIN_B_ID"
CHAIN_B_RPC="http://localhost:27657"
CHAIN_B_DENOM="tokenb"

# Chain C Configuration
CHAIN_C_ID="chain-c"
CHAIN_C_DIR="$CHAIN_DIR_ROOT/$CHAIN_C_ID"
CHAIN_C_RPC="http://localhost:28657"
CHAIN_C_DENOM="tokenc"

# Key names
ADMIN_KEY="admin"

# Compile the contract
echo "--- Compiling Mock DEX contract ---"
(cd "$CONTRACT_DIR" && cargo wasm)

# Check if compilation succeeded
WASM_FILE="$CONTRACT_DIR/target/wasm32-unknown-unknown/release/mock_dex.wasm"
if [ ! -f "$WASM_FILE" ]; then
    echo "Error: Contract compilation failed. WASM file not found at $WASM_FILE"
    exit 1
fi

echo "Contract compiled successfully: $WASM_FILE"

# Function to deploy contract to a chain
deploy_contract() {
    local CHAIN_ID=$1
    local CHAIN_DIR=$2
    local RPC_URL=$3
    local DENOM=$4
    local DEX_NAME=$5

    echo "--- Deploying contract to $CHAIN_ID ---"
    
    # Get admin address
    ADMIN_ADDR=$(wasmd keys show "$ADMIN_KEY" -a --keyring-backend test --home "$CHAIN_DIR")
    
    # Store code
    echo "Storing contract code on $CHAIN_ID..."
    TX_STORE=$(wasmd tx wasm store "$WASM_FILE" \
        --from "$ADMIN_KEY" \
        --chain-id "$CHAIN_ID" \
        --node "$RPC_URL" \
        --gas auto --gas-adjustment 1.5 --fees "5000$DENOM" \
        --keyring-backend test \
        --home "$CHAIN_DIR" \
        -y --output json)
    
    # Extract code ID
    CODE_ID=$(echo "$TX_STORE" | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
    echo "Contract code stored with ID: $CODE_ID"
    
    # Prepare initial rates JSON
    # These rates are simplified for the demo
    # In a real DEX, rates would be determined by liquidity pools
    RATES_JSON='[
        {"from_denom":"tokena","to_denom":"tokenb","rate":"10.0"},
        {"from_denom":"tokenb","to_denom":"tokena","rate":"0.1"},
        {"from_denom":"tokena","to_denom":"tokenc","rate":"5.0"},
        {"from_denom":"tokenc","to_denom":"tokena","rate":"0.2"},
        {"from_denom":"tokenb","to_denom":"tokenc","rate":"0.5"},
        {"from_denom":"tokenc","to_denom":"tokenb","rate":"2.0"}
    ]'
    
    # Instantiate contract
    echo "Instantiating contract on $CHAIN_ID..."
    INSTANTIATE_MSG="{\"admin\":\"$ADMIN_ADDR\",\"dex_name\":\"$DEX_NAME\",\"initial_rates\":$RATES_JSON}"
    
    TX_INIT=$(wasmd tx wasm instantiate "$CODE_ID" "$INSTANTIATE_MSG" \
        --from "$ADMIN_KEY" \
        --chain-id "$CHAIN_ID" \
        --node "$RPC_URL" \
        --gas auto --gas-adjustment 1.5 --fees "5000$DENOM" \
        --keyring-backend test \
        --home "$CHAIN_DIR" \
        --label "$DEX_NAME" \
        --admin "$ADMIN_ADDR" \
        -y --output json)
    
    # Extract contract address
    CONTRACT_ADDR=$(echo "$TX_INIT" | jq -r '.logs[0].events[] | select(.type=="instantiate") | .attributes[] | select(.key=="_contract_address") | .value')
    echo "Contract instantiated at address: $CONTRACT_ADDR"
    
    # Fund the contract with tokens for swaps
    echo "Funding contract with tokens for swaps..."
    wasmd tx bank send "$ADMIN_KEY" "$CONTRACT_ADDR" "1000000$DENOM,1000000tokena,1000000tokenb,1000000tokenc" \
        --from "$ADMIN_KEY" \
        --chain-id "$CHAIN_ID" \
        --node "$RPC_URL" \
        --gas auto --gas-adjustment 1.5 --fees "5000$DENOM" \
        --keyring-backend test \
        --home "$CHAIN_DIR" \
        -y
    
    echo "Contract deployment to $CHAIN_ID complete."
    echo "Contract address: $CONTRACT_ADDR"
    echo
    
    # Return the contract address
    echo "$CONTRACT_ADDR"
}

# Deploy to all three chains
echo "Deploying contracts to all chains..."
CONTRACT_A=$(deploy_contract "$CHAIN_A_ID" "$CHAIN_A_DIR" "$CHAIN_A_RPC" "$CHAIN_A_DENOM" "DEX-A")
CONTRACT_B=$(deploy_contract "$CHAIN_B_ID" "$CHAIN_B_DIR" "$CHAIN_B_RPC" "$CHAIN_B_DENOM" "DEX-B")
CONTRACT_C=$(deploy_contract "$CHAIN_C_ID" "$CHAIN_C_DIR" "$CHAIN_C_RPC" "$CHAIN_C_DENOM" "DEX-C")

echo "All contracts deployed successfully."
echo
echo "Chain A contract: $CONTRACT_A"
echo "Chain B contract: $CONTRACT_B"
echo "Chain C contract: $CONTRACT_C"
echo
echo "Update the contract addresses in backend/config/chains.go with these values."
