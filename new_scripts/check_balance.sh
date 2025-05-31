#!/bin/bash

# Define chains and their respective nodes and home paths
declare -A CHAINS=(
  ["alphanet"]="tcp://localhost:26657"
  ["betanet"]="tcp://localhost:27657"
  ["gammanet"]="tcp://localhost:28657"
)

declare -A HOMES=(
  ["alphanet"]="$HOME/.alphanet"
  ["betanet"]="$HOME/.betanet"
  ["gammanet"]="$HOME/.gammanet"
)

# Define the addresses to check for each chain
declare -A ADDRESSES=(
  ["alphanet"]="wasm1msq9cmxmwa3qqthsws3gjde6gstkw8fhj20j6p"
  ["betanet"]="wasm187cjkyky525d0jpd58f58vtum4fdxn7jyfchf2"
  ["gammanet"]="wasm1f0sfpfgqv8khjlrpmxqxwlt356jgy8su3dn47e"
)

# Define expected token denominations (e.g., ualpha, ubeta, ugamma)
declare -A DENOMS=(
  ["alphanet"]="ualpha"
  ["betanet"]="ubeta"
  ["gammanet"]="ugamma"
)

echo "🔍 Checking balances on each chain..."

for CHAIN in "${!CHAINS[@]}"; do
  NODE="${CHAINS[$CHAIN]}"
  HOME_DIR="${HOMES[$CHAIN]}"
  ADDRESS="${ADDRESSES[$CHAIN]}"
  DENOM="${DENOMS[$CHAIN]}"

  echo -e "\n🔗 Chain: $CHAIN"
  echo "📬 Address: $ADDRESS"

  BALANCE=$(wasmd q bank balances "$ADDRESS" --node "$NODE" --output json 2>/dev/null | jq -r --arg denom "$DENOM" '.balances[] | select(.denom==$denom) | .amount')

  if [[ -z "$BALANCE" ]]; then
    echo "❌ No balance found or address not funded yet."
  else
    echo "✅ Balance: $BALANCE $DENOM"
  fi
done
