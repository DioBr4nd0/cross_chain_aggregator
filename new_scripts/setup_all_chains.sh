#!/bin/bash

set -e

# Path to the template script
TEMPLATE_SCRIPT="./setup_chain_template.sh" # Assuming it's in the same directory

# Make the template script executable
chmod +x "$TEMPLATE_SCRIPT"

# --- Chain Alpha (AlphaNet) ---
CHAIN_A_ID="alphanet-1"
CHAIN_A_MONIKER="alpha"
CHAIN_A_HOME_DIR="$HOME/.alphanet"
CHAIN_A_NATIVE_DENOM="ualpha"
CHAIN_A_RPC_PORT="26657"
CHAIN_A_P2P_PORT="26656"
CHAIN_A_GRPC_PORT="9090"
CHAIN_A_API_PORT="1317"

"$TEMPLATE_SCRIPT" "$CHAIN_A_ID" "$CHAIN_A_MONIKER" "$CHAIN_A_HOME_DIR" \
                   "$CHAIN_A_NATIVE_DENOM" \
                   "$CHAIN_A_RPC_PORT" "$CHAIN_A_P2P_PORT" "$CHAIN_A_GRPC_PORT" "$CHAIN_A_API_PORT"

# --- Chain Beta (BetaNet) ---
CHAIN_B_ID="betanet-1"
CHAIN_B_MONIKER="beta"
CHAIN_B_HOME_DIR="$HOME/.betanet"
CHAIN_B_NATIVE_DENOM="ubeta"
CHAIN_B_RPC_PORT="27657"
CHAIN_B_P2P_PORT="27656" # P2P must be unique if running on same host
CHAIN_B_GRPC_PORT="9190"
CHAIN_B_API_PORT="1318"

"$TEMPLATE_SCRIPT" "$CHAIN_B_ID" "$CHAIN_B_MONIKER" "$CHAIN_B_HOME_DIR" \
                   "$CHAIN_B_NATIVE_DENOM" \
                   "$CHAIN_B_RPC_PORT" "$CHAIN_B_P2P_PORT" "$CHAIN_B_GRPC_PORT" "$CHAIN_B_API_PORT"

# --- Chain Gamma (GammaNet) ---
CHAIN_C_ID="gammanet-1"
CHAIN_C_MONIKER="gamma"
CHAIN_C_HOME_DIR="$HOME/.gammanet"
CHAIN_C_NATIVE_DENOM="ugamma"
CHAIN_C_RPC_PORT="28657"
CHAIN_C_P2P_PORT="28656" # P2P must be unique
CHAIN_C_GRPC_PORT="9290"
CHAIN_C_API_PORT="1319"

"$TEMPLATE_SCRIPT" "$CHAIN_C_ID" "$CHAIN_C_MONIKER" "$CHAIN_C_HOME_DIR" \
                   "$CHAIN_C_NATIVE_DENOM" \
                   "$CHAIN_C_RPC_PORT" "$CHAIN_C_P2P_PORT" "$CHAIN_C_GRPC_PORT" "$CHAIN_C_API_PORT"

echo "--- All three chains configured! ---"
echo "You can now start them in separate terminals, e.g.:"
echo "wasmd start --home \"$CHAIN_A_HOME_DIR\" --log_level info"
echo "wasmd start --home \"$CHAIN_B_HOME_DIR\" --log_level info"
echo "wasmd start --home \"$CHAIN_C_HOME_DIR\" --log_level info"
