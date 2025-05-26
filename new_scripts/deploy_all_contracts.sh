#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status.

# --- Configuration ---
WASM_FILE="../contracts/mock_dex/artifacts/mock_dex.wasm" # Adjust if your path is different
DEPLOYER_KEY_NAME="backendop"
KEYRING_BACKEND="test"
# Ensure COMMON_TX_FLAGS are appended correctly or integrated into TX_FLAGS_WASMD etc.
COMMON_TX_FLAGS_BASE="--gas auto --gas-adjustment 1.5 -y -o json --keyring-backend $KEYRING_BACKEND"


# --- Chain Alpha (AlphaNet) ---
CHAIN_A_ID="alphanet-1"
CHAIN_A_HOME_DIR="$HOME/.alphanet"
CHAIN_A_NODE_RPC="tcp://localhost:26657"
CHAIN_A_NATIVE_DENOM="ualpha"
CHAIN_A_DEX_NAME="AlphaDEX"
CHAIN_A_INITIAL_RATES_JSON='[
  {"from_denom":"ualpha","to_denom":"ibc/BetaOnAlpha","rate":"10.0"},
  {"from_denom":"ibc/BetaOnAlpha","to_denom":"ualpha","rate":"0.095"},
  {"from_denom":"ualpha","to_denom":"ibc/GammaOnAlpha","rate":"5.0"},
  {"from_denom":"ibc/GammaOnAlpha","to_denom":"ualpha","rate":"0.19"}
]'

# --- Chain Beta (BetaNet) ---
CHAIN_B_ID="betanet-1"
CHAIN_B_HOME_DIR="$HOME/.betanet"
CHAIN_B_NODE_RPC="tcp://localhost:27657"
CHAIN_B_NATIVE_DENOM="ubeta"
CHAIN_B_DEX_NAME="BetaDEX"
CHAIN_B_INITIAL_RATES_JSON='[
  {"from_denom":"ubeta","to_denom":"ibc/AlphaOnBeta","rate":"0.1"},
  {"from_denom":"ibc/AlphaOnBeta","to_denom":"ubeta","rate":"9.8"},
  {"from_denom":"ubeta","to_denom":"ibc/GammaOnBeta","rate":"2.0"},
  {"from_denom":"ibc/GammaOnBeta","to_denom":"ubeta","rate":"0.48"}
]'

# --- Chain Gamma (GammaNet) ---
CHAIN_C_ID="gammanet-1"
CHAIN_C_HOME_DIR="$HOME/.gammanet"
CHAIN_C_NODE_RPC="tcp://localhost:28657"
CHAIN_C_NATIVE_DENOM="ugamma"
CHAIN_C_DEX_NAME="GammaDEX"
CHAIN_C_INITIAL_RATES_JSON='[
  {"from_denom":"ugamma","to_denom":"ibc/AlphaOnGamma","rate":"0.2"},
  {"from_denom":"ibc/AlphaOnGamma","to_denom":"ugamma","rate":"4.9"},
  {"from_denom":"ugamma","to_denom":"ibc/BetaOnGamma","rate":"0.5"},
  {"from_denom":"ibc/BetaOnGamma","to_denom":"ugamma","rate":"1.95"}
]'


# --- Helper Function to Deploy to a Single Chain ---
deploy_to_chain() {
  local CHAIN_ID="$1"
  local CHAIN_HOME_DIR="$2"
  local NODE_RPC="$3"
  local NATIVE_DENOM="$4" # For calculating fees
  local DEX_NAME="$5"
  local INITIAL_RATES_JSON="$6"
  local DEPLOYER_ADDR=$(wasmd keys show "$DEPLOYER_KEY_NAME" -a --keyring-backend="$KEYRING_BACKEND" --home="$CHAIN_HOME_DIR")

  echo "--- [${CHAIN_ID}] Deploying ${DEX_NAME} ---"

  # 1. Store WASM Code
  echo "Storing WASM code on ${CHAIN_ID}..."
  # Construct TX_FLAGS for wasmd store command
  TX_FLAGS_STORE="--from $DEPLOYER_KEY_NAME --chain-id $CHAIN_ID --node $NODE_RPC --home $CHAIN_HOME_DIR --fees 1000000$NATIVE_DENOM $COMMON_TX_FLAGS_BASE"

  STORE_CMD_OUTPUT=$(wasmd tx wasm store "$WASM_FILE" $TX_FLAGS_STORE)

  # Check for valid JSON and successful transaction code (0)
  if ! echo "$STORE_CMD_OUTPUT" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd tx wasm store' command failed or did not return valid JSON for ${CHAIN_ID}."
    echo "Output: $STORE_CMD_OUTPUT"
    return 1
  fi
  TX_CODE=$(echo "$STORE_CMD_OUTPUT" | jq -r '.code')
  if [ "$TX_CODE" != "0" ]; then
    echo "Error storing WASM code on ${CHAIN_ID} (tx code: $TX_CODE). Raw log:"
    echo "$STORE_CMD_OUTPUT" | jq -r '.raw_log'
    return 1
  fi

  TXHASH_STORE=$(echo "$STORE_CMD_OUTPUT" | jq -r '.txhash')
  if [ -z "$TXHASH_STORE" ] || [ "$TXHASH_STORE" == "null" ]; then
    echo "Error: Could not extract TXHASH from store transaction output on ${CHAIN_ID}."
    echo "Output:"
    echo "$STORE_CMD_OUTPUT" | jq
    return 1
  fi
  echo "Store code transaction submitted. TXHASH: $TXHASH_STORE. Waiting for inclusion..."
  sleep 7 # Increased sleep as per docs suggestion

  # Query the transaction to get the CODE_ID from events
  echo "Querying transaction $TXHASH_STORE for CODE_ID..."
  # Query flags need --output json to be parsed by jq
  QUERY_TX_FLAGS="--node $NODE_RPC --output json --home $CHAIN_HOME_DIR"
  QUERY_TX_RESPONSE=$(wasmd query tx "$TXHASH_STORE" $QUERY_TX_FLAGS)
  
  if ! echo "$QUERY_TX_RESPONSE" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd query tx' did not return valid JSON for ${CHAIN_ID} tx $TXHASH_STORE."
    echo "Output: $QUERY_TX_RESPONSE"
    return 1
  fi
  
  # Extract CODE_ID using the jq path from official docs [3, 5]
  CODE_ID=$(echo "$QUERY_TX_RESPONSE" | jq -r '.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')

  if [ -z "$CODE_ID" ] || [ "$CODE_ID" == "null" ]; then
    # Fallback to logs if events structure is different, though `events` is standard
    CODE_ID_LOGS=$(echo "$QUERY_TX_RESPONSE" | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
    if [ -n "$CODE_ID_LOGS" ] && [ "$CODE_ID_LOGS" != "null" ]; then
        CODE_ID="$CODE_ID_LOGS"
    else
        echo "Error: Could not extract CODE_ID from transaction events or logs on ${CHAIN_ID} for tx $TXHASH_STORE."
        echo "Query TX Response:"
        echo "$QUERY_TX_RESPONSE" | jq
        return 1
    fi
  fi
  echo "Stored WASM code on ${CHAIN_ID} with CODE_ID: $CODE_ID"
  sleep 2

  # 2. Instantiate Contract
  INSTANTIATE_MSG=$(printf '{"admin":"%s","dex_name":"%s","initial_rates":%s}' "$DEPLOYER_ADDR" "$DEX_NAME" "$INITIAL_RATES_JSON")

  echo "Instantiating ${DEX_NAME} on ${CHAIN_ID} with CODE_ID ${CODE_ID}..."
  echo "Instantiate message: $INSTANTIATE_MSG"
  
  TX_FLAGS_INSTANTIATE="--from $DEPLOYER_KEY_NAME --label $DEX_NAME --chain-id $CHAIN_ID --node $NODE_RPC --home $CHAIN_HOME_DIR --admin $DEPLOYER_ADDR --fees 500000$NATIVE_DENOM $COMMON_TX_FLAGS_BASE"
  
  INSTANTIATE_CMD_OUTPUT=$(wasmd tx wasm instantiate "$CODE_ID" "$INSTANTIATE_MSG" $TX_FLAGS_INSTANTIATE)

  if ! echo "$INSTANTIATE_CMD_OUTPUT" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd tx wasm instantiate' command failed or did not return valid JSON for ${CHAIN_ID}."
    echo "Output: $INSTANTIATE_CMD_OUTPUT"
    return 1
  fi
  TX_CODE_INIT=$(echo "$INSTANTIATE_CMD_OUTPUT" | jq -r '.code')
  if [ "$TX_CODE_INIT" != "0" ]; then
    echo "Error instantiating contract on ${CHAIN_ID} (tx code: $TX_CODE_INIT). Raw log:"
    echo "$INSTANTIATE_CMD_OUTPUT" | jq -r '.raw_log'
    return 1
  fi

  TXHASH_INSTANTIATE=$(echo "$INSTANTIATE_CMD_OUTPUT" | jq -r '.txhash')
  if [ -z "$TXHASH_INSTANTIATE" ] || [ "$TXHASH_INSTANTIATE" == "null" ]; then
    echo "Error: Could not extract TXHASH from instantiate transaction output on ${CHAIN_ID}."
    echo "Output:"
    echo "$INSTANTIATE_CMD_OUTPUT" | jq
    return 1
  fi
  echo "Instantiate transaction submitted. TXHASH: $TXHASH_INSTANTIATE. Waiting for inclusion..."
  sleep 7 # Increased sleep

  # Query the instantiate transaction to get the contract address
  echo "Querying transaction $TXHASH_INSTANTIATE for CONTRACT_ADDRESS..."
  QUERY_INIT_TX_RESPONSE=$(wasmd query tx "$TXHASH_INSTANTIATE" $QUERY_TX_FLAGS)

  if ! echo "$QUERY_INIT_TX_RESPONSE" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd query tx' for instantiate did not return valid JSON for ${CHAIN_ID} tx $TXHASH_INSTANTIATE."
    echo "Output: $QUERY_INIT_TX_RESPONSE"
    return 1
  fi
  
  # Extract contract address using the jq path from official docs for instantiate event [3]
  CONTRACT_ADDRESS=$(echo "$QUERY_INIT_TX_RESPONSE" | jq -r '.events[] | select(.type=="instantiate") | .attributes[] | select(.key=="_contract_address") | .value')
  # Fallback for common variations
  if [ -z "$CONTRACT_ADDRESS" ] || [ "$CONTRACT_ADDRESS" == "null" ]; then
    CONTRACT_ADDRESS=$(echo "$QUERY_INIT_TX_RESPONSE" | jq -r '.events[] | select(.type=="instantiate_contract") | .attributes[] | select(.key=="contract_address") | .value')
  fi
  if [ -z "$CONTRACT_ADDRESS" ] || [ "$CONTRACT_ADDRESS" == "null" ]; then
    CONTRACT_ADDRESS=$(echo "$QUERY_INIT_TX_RESPONSE" | jq -r '.logs[0].events[] | select(.type=="instantiate" or .type=="instantiate_contract") | .attributes[] | select(.key=="_contract_address" or .key=="contract_address") | .value')
  fi

  if [ -z "$CONTRACT_ADDRESS" ] || [ "$CONTRACT_ADDRESS" == "null" ]; then
    echo "Error: Could not extract CONTRACT_ADDRESS from instantiate transaction events on ${CHAIN_ID} for tx $TXHASH_INSTANTIATE."
    echo "Query TX Response for Instantiate:"
    echo "$QUERY_INIT_TX_RESPONSE" | jq
    return 1
  fi
  echo "Instantiated ${DEX_NAME} on ${CHAIN_ID} at address: $CONTRACT_ADDRESS"
  sleep 2

  # 3. (Optional but Recommended) Fund the DEX Contract
  echo "Funding ${DEX_NAME} on ${CHAIN_ID} with some native tokens..."
  FUND_AMOUNT="100000000$NATIVE_DENOM" # Start with native token only for simplicity
  # For other tokens (e.g., 100000000ualpha on BetaNet), DEPLOYER_KEY_NAME on BetaNet must *have* ualpha (likely via IBC)
  # We'll handle multi-token funding in a separate step or assume DEX can operate with one-sided deposits initially
  
  TX_FLAGS_FUND="--from $DEPLOYER_KEY_NAME --chain-id $CHAIN_ID --node $NODE_RPC --home $CHAIN_HOME_DIR --fees 100000$NATIVE_DENOM $COMMON_TX_FLAGS_BASE"
  FUND_CMD_OUTPUT=$(wasmd tx bank send "$DEPLOYER_KEY_NAME" "$CONTRACT_ADDRESS" "$FUND_AMOUNT" $TX_FLAGS_FUND)

  if ! echo "$FUND_CMD_OUTPUT" | jq -e . > /dev/null 2>&1; then
    echo "Warning: 'wasmd tx bank send' for funding did not return valid JSON for ${CHAIN_ID}."
    echo "Output: $FUND_CMD_OUTPUT"
  else
    TX_CODE_FUND=$(echo "$FUND_CMD_OUTPUT" | jq -r '.code')
    if [ "$TX_CODE_FUND" != "0" ]; then
      echo "Warning: Error funding contract on ${CHAIN_ID} (tx code: $TX_CODE_FUND). Raw log:"
      echo "$FUND_CMD_OUTPUT" | jq -r '.raw_log'
    else
      echo "Attempted to fund ${DEX_NAME} on ${CHAIN_ID}."
    fi
  fi
  
  echo "Contract Address for ${CHAIN_ID} (${DEX_NAME}): ${CONTRACT_ADDRESS}"
  echo "${CHAIN_ID}_CONTRACT_ADDRESS=${CONTRACT_ADDRESS}" >> deployed_contracts.txt
  echo "--- [${CHAIN_ID}] ${DEX_NAME} deployment complete ---"
  echo ""
}

# --- Main Deployment ---
if [ ! -f "$WASM_FILE" ]; then
    echo "Error: WASM file not found at $WASM_FILE. Please compile the contract first."
    exit 1
fi

rm -f deployed_contracts.txt

echo "Starting contract deployments..."
deploy_to_chain "$CHAIN_A_ID" "$CHAIN_A_HOME_DIR" "$CHAIN_A_NODE_RPC" "$CHAIN_A_NATIVE_DENOM" "$CHAIN_A_DEX_NAME" "$CHAIN_A_INITIAL_RATES_JSON"
deploy_to_chain "$CHAIN_B_ID" "$CHAIN_B_HOME_DIR" "$CHAIN_B_NODE_RPC" "$CHAIN_B_NATIVE_DENOM" "$CHAIN_B_DEX_NAME" "$CHAIN_B_INITIAL_RATES_JSON"
deploy_to_chain "$CHAIN_C_ID" "$CHAIN_C_HOME_DIR" "$CHAIN_C_NODE_RPC" "$CHAIN_C_NATIVE_DENOM" "$CHAIN_C_DEX_NAME" "$CHAIN_C_INITIAL_RATES_JSON"

# echo "--- All contract deployments attempted. ---"
# echo "Deployed contract addresses are in 'deployed_contracts.txt' and printed above."
# echo "IMPORTANT: You will need to update your backend's config/config.go with these contract addresses."
# echo "IMPORTANT: The 'ibc/...' denoms in INITIAL_RATES_JSON are placeholders. After setting up IBC with Hermes, you will get actual IBC denoms. You might need to update the rates in your deployed contracts using an admin execute message (if your contract supports it) or re-instantiate with correct IBC denoms for the DEX to function with IBC tokens."
