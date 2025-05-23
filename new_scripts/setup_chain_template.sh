#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status.

# --- Parameters for this specific chain ---
CHAIN_ID="$1"       # e.g., alphanet-1
MONIKER_PREFIX="$2" # e.g., alpha
HOME_DIR="$3"       # e.g., $HOME/.alphanet
NATIVE_DENOM="$4"   # e.g., ualpha
RPC_PORT="$5"
P2P_PORT="$6"
GRPC_PORT="$7"
API_PORT="$8"
GRPC_WEB_PORT=$(($GRPC_PORT + 1)) # Calculate gRPC-Web port

# --- Common Configuration ---
KEYRING_BACKEND="test"
VALIDATOR_KEY_NAME="validator"
USER_KEY_NAME="user1"
BACKEND_OP_KEY_NAME="backendop"
RELAYER_KEY_NAME="relayer" # For Hermes later

VALIDATOR_STAKE_AMOUNT="250000000$NATIVE_DENOM" # Amount for validator gentx
GENESIS_ACCOUNT_BALANCE="1000000000000$NATIVE_DENOM" # Initial balance for accounts

# --- Cleanup and Initialization ---
echo "--- [${CHAIN_ID}] Removing previous data from $HOME_DIR (if any) ---"
rm -rf "$HOME_DIR"
mkdir -p "$HOME_DIR"

echo "--- [${CHAIN_ID}] Initializing node: ${MONIKER_PREFIX}-node with chain-id: $CHAIN_ID ---"
wasmd init "${MONIKER_PREFIX}-node" --chain-id="$CHAIN_ID" --home="$HOME_DIR" --default-denom="$NATIVE_DENOM"

# --- Genesis & Config Adjustments ---
GENESIS_FILE="$HOME_DIR/config/genesis.json"
CONFIG_FILE="$HOME_DIR/config/config.toml"
APP_TOML_FILE="$HOME_DIR/config/app.toml"

echo "--- [${CHAIN_ID}] Modifying genesis.json ---"
# Set native denom as the staking denom and gov deposit denom for simplicity
jq ".app_state.staking.params.bond_denom = \"$NATIVE_DENOM\" | \
    .app_state.gov.params.min_deposit[0].denom = \"$NATIVE_DENOM\" | \
    .app_state.crisis.constant_fee.denom = \"$NATIVE_DENOM\" | \
    .app_state.transfer.params.receive_enabled = true | \
    .app_state.transfer.params.send_enabled = true " \
    "$GENESIS_FILE" > tmp_genesis.json && mv tmp_genesis.json "$GENESIS_FILE"

sed -i.bak 's/"max_gas": "-1"/"max_gas": "30000000"/' "$GENESIS_FILE" # Increase gas limit

echo "--- [${CHAIN_ID}] Modifying config.toml ---"
sed -i.bak "s/laddr = \"tcp:\/\/127.0.0.1:26657\"/laddr = \"tcp:\/\/0.0.0.0:$RPC_PORT\"/" "$CONFIG_FILE"
sed -i.bak "s/laddr = \"tcp:\/\/127.0.0.1:26656\"/laddr = \"tcp:\/\/0.0.0.0:$P2P_PORT\"/" "$CONFIG_FILE"
sed -i.bak 's/timeout_commit = "5s"/timeout_commit = "2s"/' "$CONFIG_FILE" # Faster blocks
sed -i.bak 's/create_empty_blocks = true/create_empty_blocks = false/' "$CONFIG_FILE"
sed -i.bak 's/cors_allowed_origins = .*/cors_allowed_origins = ["*"]/' "$CONFIG_FILE"

echo "--- [${CHAIN_ID}] Modifying app.toml ---"
sed -i.bak -e '/\[api\]/,/\[store\]/{s/enable = false/enable = true/}' "$APP_TOML_FILE"
sed -i.bak -e "s|address = \"tcp://localhost:1317\"|address = \"tcp://0.0.0.0:$API_PORT\"|" "$APP_TOML_FILE"
sed -i.bak -e '/\[grpc\]/,/\[store\]/{s/enable = false/enable = true/}' "$APP_TOML_FILE"
sed -i.bak -e "s|address = \"localhost:9090\"|address = \"0.0.0.0:$GRPC_PORT\"|" "$APP_TOML_FILE"
sed -i.bak -e '/\[grpc-web\]/,/\[store\]/{s/enable = false/enable = true/}' "$APP_TOML_FILE"
sed -i.bak -e "s|address = \"localhost:9091\"|address = \"0.0.0.0:$GRPC_WEB_PORT\"|" "$APP_TOML_FILE"
sed -i.bak 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML_FILE"


# --- Key Management ---
echo "--- [${CHAIN_ID}] Adding keys ---"
wasmd keys add "$VALIDATOR_KEY_NAME" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"
wasmd keys add "$USER_KEY_NAME" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"
wasmd keys add "$BACKEND_OP_KEY_NAME" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"
wasmd keys add "$RELAYER_KEY_NAME" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"

VALIDATOR_ADDR=$(wasmd keys show "$VALIDATOR_KEY_NAME" -a --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR")
USER_ADDR=$(wasmd keys show "$USER_KEY_NAME" -a --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR")
BACKEND_OP_ADDR=$(wasmd keys show "$BACKEND_OP_KEY_NAME" -a --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR")
RELAYER_ADDR=$(wasmd keys show "$RELAYER_KEY_NAME" -a --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR")

# --- Genesis Accounts ---
echo "--- [${CHAIN_ID}] Adding genesis accounts ---"
wasmd genesis add-genesis-account "$VALIDATOR_KEY_NAME" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"
wasmd genesis add-genesis-account "$USER_KEY_NAME" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"
wasmd genesis add-genesis-account "$BACKEND_OP_KEY_NAME" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"
wasmd genesis add-genesis-account "$RELAYER_KEY_NAME" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"

# --- Gentx ---
echo "--- [${CHAIN_ID}] Creating genesis transaction for validator ---"
wasmd genesis gentx "$VALIDATOR_KEY_NAME" "$VALIDATOR_STAKE_AMOUNT" --chain-id="$CHAIN_ID" --amount="$VALIDATOR_STAKE_AMOUNT" --keyring-backend="$KEYRING_BACKEND" --home="$HOME_DIR"

echo "--- [${CHAIN_ID}] Collecting genesis transactions ---"
wasmd genesis collect-gentxs --home="$HOME_DIR"

# --- Final Instructions ---
echo ""
echo "✅ [${CHAIN_ID}] Setup complete in $HOME_DIR."
echo "  Validator ($VALIDATOR_KEY_NAME): $VALIDATOR_ADDR"
echo "  User ($USER_KEY_NAME): $USER_ADDR"
echo "  Backend Operator ($BACKEND_OP_KEY_NAME): $BACKEND_OP_ADDR"
echo "  Relayer ($RELAYER_KEY_NAME): $RELAYER_ADDR"
echo "  RPC: http://localhost:$RPC_PORT"
echo "  To start node: wasmd start --home \"$HOME_DIR\" --log_level info --trace"
echo ""
