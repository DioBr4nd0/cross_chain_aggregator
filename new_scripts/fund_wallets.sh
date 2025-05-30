echo "--- Funding relayer accounts ---"

# Define variables for your chain setup (adjust if needed)
CHAIN_A_HOME_DIR="$HOME/.alphanet"
CHAIN_A_RPC="tcp://localhost:26657"
CHAIN_A_NATIVE_DENOM="ualpha"
FUNDER_KEY_A="backendop" # Key on AlphaNet with funds

CHAIN_B_HOME_DIR="$HOME/.betanet"
CHAIN_B_RPC="tcp://localhost:27657"
CHAIN_B_NATIVE_DENOM="ubeta"
FUNDER_KEY_B="backendop" # Key on BetaNet with funds

CHAIN_C_HOME_DIR="$HOME/.gammanet"
CHAIN_C_RPC="tcp://localhost:28657"
CHAIN_C_NATIVE_DENOM="ugamma"
FUNDER_KEY_C="backendop" # Key on GammaNet with funds

FUND_AMOUNT_NATIVE="50000000" # e.g., 50 of the native token (50,000,000 micro units)
COMMON_TX_SEND_FLAGS="--keyring-backend test --gas auto --gas-adjustment 1.5 -y -o json"

# Get relayer addresses (you should have these from Step 3, or re-run list)
# For scripting, you might parse them, but for manual steps, use the noted addresses.
# Example (replace with actual addresses):
RELAYER_ALPHA_ADDR="wasm1msq9cmxmwa3qqthsws3gjde6gstkw8fhj20j6p"
RELAYER_BETA_ADDR="wasm187cjkyky525d0jpd58f58vtum4fdxn7jyfchf2"
RELAYER_GAMMA_ADDR="wasm1f0sfpfgqv8khjlrpmxqxwlt356jgy8su3dn47e"

# Fund relayer on AlphaNet
echo "Funding relayer on AlphaNet ($RELAYER_ALPHA_ADDR)..."
wasmd tx bank send "$FUNDER_KEY_A" "$RELAYER_ALPHA_ADDR" "${FUND_AMOUNT_NATIVE}${CHAIN_A_NATIVE_DENOM}" \
    --chain-id alphanet-1 --node "$CHAIN_A_RPC" --home "$CHAIN_A_HOME_DIR" \
    --fees "5000${CHAIN_A_NATIVE_DENOM}" $COMMON_TX_SEND_FLAGS

sleep 5

# Fund relayer on BetaNet
echo "Funding relayer on BetaNet ($RELAYER_BETA_ADDR)..."
wasmd tx bank send "$FUNDER_KEY_B" "$RELAYER_BETA_ADDR" "${FUND_AMOUNT_NATIVE}${CHAIN_B_NATIVE_DENOM}" \
    --chain-id betanet-1 --node "$CHAIN_B_RPC" --home "$CHAIN_B_HOME_DIR" \
    --fees "5000${CHAIN_B_NATIVE_DENOM}" $COMMON_TX_SEND_FLAGS

sleep 5

# Fund relayer on GammaNet
echo "Funding relayer on GammaNet ($RELAYER_GAMMA_ADDR)..."
wasmd tx bank send "$FUNDER_KEY_C" "$RELAYER_GAMMA_ADDR" "${FUND_AMOUNT_NATIVE}${CHAIN_C_NATIVE_DENOM}" \
    --chain-id gammanet-1 --node "$CHAIN_C_RPC" --home "$CHAIN_C_HOME_DIR" \
    --fees "5000${CHAIN_C_NATIVE_DENOM}" $COMMON_TX_SEND_FLAGS

sleep 5
echo "Relayer accounts funding transactions submitted."
