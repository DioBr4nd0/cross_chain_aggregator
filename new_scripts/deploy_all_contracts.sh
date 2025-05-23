#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status.

# ... (Keep your existing configurations for WASM_FILE, DEPLOYER_KEY_NAME, chains, etc.) ...
# WASM_FILE="../contracts/mock_dex/artifacts/mock_dex.wasm"
# DEPLOYER_KEY_NAME="backendop"
# KEYRING_BACKEND="test"
# COMMON_TX_FLAGS="--gas auto --gas-adjustment 1.5 -y -o json --keyring-backend $KEYRING_BACKEND"
# ... (Chain A, B, C configs) ...

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
  # Common transaction flags specific to wasmd tx commands
  TX_FLAGS_WASMD="--from $DEPLOYER_KEY_NAME --chain-id $CHAIN_ID --node $NODE_RPC --home $CHAIN_HOME_DIR --fees 1000000$NATIVE_DENOM $COMMON_TX_FLAGS"

  # Execute the store command and capture its JSON output
  STORE_CMD_OUTPUT=$(wasmd tx wasm store "$WASM_FILE" $TX_FLAGS_WASMD)

  # Check if the command output is valid JSON and if the transaction was successful
  if ! echo "$STORE_CMD_OUTPUT" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd tx wasm store' did not return valid JSON for ${CHAIN_ID}."
    echo "Output: $STORE_CMD_OUTPUT"
    return 1
  fi
  
  TX_CODE=$(echo "$STORE_CMD_OUTPUT" | jq -r '.code')
  if [ "$TX_CODE" != "0" ]; then
    echo "Error storing WASM code on ${CHAIN_ID} (tx code: $TX_CODE):"
    echo "$STORE_CMD_OUTPUT" | jq
    return 1
  fi

  # Extract TXHASH from the store command output
  TXHASH_STORE=$(echo "$STORE_CMD_OUTPUT" | jq -r '.txhash')
  if [ -z "$TXHASH_STORE" ] || [ "$TXHASH_STORE" == "null" ]; then
    echo "Error: Could not extract TXHASH from store transaction output on ${CHAIN_ID}."
    echo "Output:"
    echo "$STORE_CMD_OUTPUT" | jq
    return 1
  fi
  echo "Store code transaction submitted. TXHASH: $TXHASH_STORE. Waiting for inclusion..."
  
  # Wait for the transaction to be included in a block (adjust sleep time as needed for your local chain)
  sleep 6 # As recommended in the CosmWasm docs [3]

  # 2. Query the transaction to get the CODE_ID from events [3]
  echo "Querying transaction $TXHASH_STORE for CODE_ID..."
  QUERY_TX_RESPONSE=$(wasmd query tx "$TXHASH_STORE" --node "$NODE_RPC" --output json --home "$CHAIN_HOME_DIR")
  
  # Check if the query tx response is valid JSON
  if ! echo "$QUERY_TX_RESPONSE" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd query tx' did not return valid JSON for ${CHAIN_ID} tx $TXHASH_STORE."
    echo "Output: $QUERY_TX_RESPONSE"
    return 1
  fi

  # Extract CODE_ID from the transaction events [3, 5]
  CODE_ID=$(echo "$QUERY_TX_RESPONSE" | jq -r '.logs[0].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')
  # A more robust jq for events that might not be in logs[0] or have different structures:
  # CODE_ID=$(echo "$QUERY_TX_RESPONSE" | jq -r 'first(.logs[].events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value)')
  # Or, as per the link:
  # CODE_ID=$(echo "$QUERY_TX_RESPONSE" | jq -r '.events[] | select(.type=="store_code").attributes[] | select(.key=="code_id").value')
  # Let's use the one from the official docs example directly [3]
  CODE_ID=$(echo "$QUERY_TX_RESPONSE" | jq -r '.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value')


  if [ -z "$CODE_ID" ] || [ "$CODE_ID" == "null" ]; then
    echo "Error: Could not extract CODE_ID from transaction events on ${CHAIN_ID} for tx $TXHASH_STORE."
    echo "Query TX Response:"
    echo "$QUERY_TX_RESPONSE" | jq
    # Fallback: try to get it from raw_log if events structure is different or tx failed subtly
    RAW_LOG_CODE_ID=$(echo "$QUERY_TX_RESPONSE" | jq -r '.raw_log | fromjson? | .[0].events[]? | select(.type=="store_code") | .attributes[]? | select(.key=="code_id") | .value')
    if [ -n "$RAW_LOG_CODE_ID" ] && [ "$RAW_LOG_CODE_ID" != "null" ]; then
        echo "Found CODE_ID in raw_log: $RAW_LOG_CODE_ID"
        CODE_ID="$RAW_LOG_CODE_ID"
    else
        echo "CODE_ID also not found in raw_log."
        return 1
    fi
  fi
  echo "Stored WASM code on ${CHAIN_ID} with CODE_ID: $CODE_ID"
  sleep 2

  # 3. Instantiate Contract
  INSTANTIATE_MSG=$(printf '{"admin":"%s","dex_name":"%s","initial_rates":%s}' "$DEPLOYER_ADDR" "$DEX_NAME" "$INITIAL_RATES_JSON")

  echo "Instantiating ${DEX_NAME} on ${CHAIN_ID} with CODE_ID ${CODE_ID}..."
  echo "Instantiate message: $INSTANTIATE_MSG"
  
  # Common flags for instantiate
  INSTANTIATE_TX_FLAGS="--from $DEPLOYER_KEY_NAME --label $DEX_NAME --chain-id $CHAIN_ID --node $NODE_RPC --home $CHAIN_HOME_DIR --admin $DEPLOYER_ADDR --fees 500000$NATIVE_DENOM $COMMON_TX_FLAGS"
  
  INSTANTIATE_CMD_OUTPUT=$(wasmd tx wasm instantiate "$CODE_ID" "$INSTANTIATE_MSG" $INSTANTIATE_TX_FLAGS)

  if ! echo "$INSTANTIATE_CMD_OUTPUT" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd tx wasm instantiate' did not return valid JSON for ${CHAIN_ID}."
    echo "Output: $INSTANTIATE_CMD_OUTPUT"
    return 1
  fi

  TX_CODE_INIT=$(echo "$INSTANTIATE_CMD_OUTPUT" | jq -r '.code')
  if [ "$TX_CODE_INIT" != "0" ]; then
    echo "Error instantiating contract on ${CHAIN_ID} (tx code: $TX_CODE_INIT):"
    echo "$INSTANTIATE_CMD_OUTPUT" | jq
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
  sleep 6

  # Query the instantiate transaction to get the contract address
  echo "Querying transaction $TXHASH_INSTANTIATE for CONTRACT_ADDRESS..."
  QUERY_INIT_TX_RESPONSE=$(wasmd query tx "$TXHASH_INSTANTIATE" --node "$NODE_RPC" --output json --home "$CHAIN_HOME_DIR")

  if ! echo "$QUERY_INIT_TX_RESPONSE" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd query tx' for instantiate did not return valid JSON for ${CHAIN_ID} tx $TXHASH_INSTANTIATE."
    echo "Output: $QUERY_INIT_TX_RESPONSE"
    return 1
  fi
  
  CONTRACT_ADDRESS=$(echo "$QUERY_INIT_TX_RESPONSE" | jq -r '.events[] | select(.type=="instantiate" or .type=="instantiate_contract") | .attributes[] | select(.key=="_contract_address" or .key=="contract_address") | .value')
  # Fallback for different event structures
  if [ -z "$CONTRACT_ADDRESS" ] || [ "$CONTRACT_ADDRESS" == "null" ]; then
    CONTRACT_ADDRESS=$(echo "$QUERY_INIT_TX_RESPONSE" | jq -r '.logs[0].events[] | select(.type=="instantiate" or .type=="instantiate_contract") | .attributes[] | select(.key=="_contract_address" or .key=="contract_address") | .value')
  fi
  if [ -z "$CONTRACT_ADDRESS" ] || [ "$CONTRACT_ADDRESS" == "null" ]; then
     RAW_LOG_CONTRACT_ADDRESS=$(echo "$QUERY_INIT_TX_RESPONSE" | jq -r '.raw_log | fromjson? | .[0].events[]? | select(.type=="instantiate") | .attributes[]? | select(.key=="_contract_address") | .value')
     if [ -n "$RAW_LOG_CONTRACT_ADDRESS" ] && [ "$RAW_LOG_CONTRACT_ADDRESS" != "null" ]; then
        echo "Found CONTRACT_ADDRESS in raw_log: $RAW_LOG_CONTRACT_ADDRESS"
        CONTRACT_ADDRESS="$RAW_LOG_CONTRACT_ADDRESS"
     else
        echo "Error: Could not extract CONTRACT_ADDRESS from instantiate transaction events on ${CHAIN_ID} for tx $TXHASH_INSTANTIATE."
        echo "Query TX Response for Instantiate:"
        echo "$QUERY_INIT_TX_RESPONSE" | jq
        return 1
    fi
  fi
  echo "Instantiated ${DEX_NAME} on ${CHAIN_ID} at address: $CONTRACT_ADDRESS"
  sleep 2

  # 4. (Optional but Recommended) Fund the DEX Contract
  echo "Funding ${DEX_NAME} on ${CHAIN_ID} with some native and other conceptual tokens..."
  # Note: The "other" tokens (like tokenb on chain-a) must exist in the DEPLOYER_KEY_NAME's account balance first.
  # Your setup_all_chains.sh script funds with native tokens. For non-native (conceptual or IBC'd later),
  # the deployer would need to acquire them first.
  # For this example, funding with just native and assuming other tokens for rates are conceptual for now.
  FUND_AMOUNT="100000000$NATIVE_DENOM,100000000ualpha,100000000ubeta,100000000ugamma" # Example funding, adjust based on actual available tokens
  
  # Trim funding string if some denoms are the same as NATIVE_DENOM
  FUND_AMOUNT_CLEANED=$(echo "$FUND_AMOUNT" | awk -F, -v RS=, -v native="$NATIVE_DENOM" '{ if ($0 ~ native && count++ > 0) {} else print }' ORS=, | sed 's/,$//')


  FUND_TX_FLAGS="--from $DEPLOYER_KEY_NAME --chain-id $CHAIN_ID --node $NODE_RPC --home $CHAIN_HOME_DIR --fees 100000$NATIVE_DENOM $COMMON_TX_FLAGS"
  FUND_CMD_OUTPUT=$(wasmd tx bank send "$DEPLOYER_KEY_NAME" "$CONTRACT_ADDRESS" "$FUND_AMOUNT_CLEANED" $FUND_TX_FLAGS)

  if ! echo "$FUND_CMD_OUTPUT" | jq -e . > /dev/null 2>&1; then
    echo "Error: 'wasmd tx bank send' for funding did not return valid JSON for ${CHAIN_ID}."
    echo "Output: $FUND_CMD_OUTPUT"
    # Don't exit, contract is deployed, funding might have failed due to insufficient balance of specific tokens
  else
    TX_CODE_FUND=$(echo "$FUND_CMD_OUTPUT" | jq -r '.code')
    if [ "$TX_CODE_FUND" != "0" ]; then
      echo "Error funding contract on ${CHAIN_ID} (tx code: $TX_CODE_FUND):"
      echo "$FUND_CMD_OUTPUT" | jq
    else
      echo "Funded ${DEX_NAME} on ${CHAIN_ID}."
    fi
  fi
  
  echo "Contract Address for ${CHAIN_ID} (${DEX_NAME}): ${CONTRACT_ADDRESS}"
  echo "${CHAIN_ID}_CONTRACT_ADDRESS=${CONTRACT_ADDRESS}" >> deployed_contracts.txt
  echo "--- [${CHAIN_ID}] ${DEX_NAME} deployment complete ---"
  echo ""
}

# --- Main Deployment ---
# (Keep the rest of your script: WASM_FILE check, rm deployed_contracts.txt, calls to deploy_to_chain)
# Ensure WASM file exists
if [ ! -f "$WASM_FILE" ]; then
    echo "Error: WASM file not found at $WASM_FILE. Please compile the contract first."
    exit 1
fi

# Clean up previous deployment info
rm -f deployed_contracts.txt

echo "Starting contract deployments..."
# (Calls to deploy_to_chain for Chain A, B, C as you had them)
deploy_to_chain "$CHAIN_A_ID" "$CHAIN_A_HOME_DIR" "$CHAIN_A_NODE_RPC" "$CHAIN_A_NATIVE_DENOM" "$CHAIN_A_DEX_NAME" "$CHAIN_A_INITIAL_RATES_JSON"
deploy_to_chain "$CHAIN_B_ID" "$CHAIN_B_HOME_DIR" "$CHAIN_B_NODE_RPC" "$CHAIN_B_NATIVE_DENOM" "$CHAIN_B_DEX_NAME" "$CHAIN_B_INITIAL_RATES_JSON"
deploy_to_chain "$CHAIN_C_ID" "$CHAIN_C_HOME_DIR" "$CHAIN_C_NODE_RPC" "$CHAIN_C_NATIVE_DENOM" "$CHAIN_C_DEX_NAME" "$CHAIN_C_INITIAL_RATES_JSON"


echo "--- All contract deployments attempted. ---"
echo "Deployed contract addresses are in 'deployed_contracts.txt' and printed above."
# ... (rest of your final messages) ...
