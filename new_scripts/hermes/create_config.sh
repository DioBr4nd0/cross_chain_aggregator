mkdir -p $HOME/.hermes
cat <<EOF > $HOME/.hermes/config.toml
[global]
log_level = 'info' # Use 'trace' for more verbose debugging if needed

# Define for Chain Alpha (alphanet-1)
[[chains]]
id = 'alphanet-1'
rpc_addr = 'http://localhost:26657'
grpc_addr = 'http://localhost:9090'
event_source = { mode = 'push', url = 'ws://localhost:26657/websocket', batch_delay = '50ms' }
rpc_timeout = '15s'
account_prefix = 'wasm'
key_name = 'relayer-alpha' # This name will be used when adding the key to Hermes
store_prefix = 'ibc'
gas_price = { price = 0.025, denom = 'ualpha' }
gas_multiplier = 1.2 # Replaces gas_adjustment
max_gas = 600000
clock_drift = '15s'
max_block_time = '30s'
trusting_period = '10hours' # Shorter for local testnets, e.g., '1hour' or '30minutes'
memo_prefix = 'hermes-alphanet'
[chains.trust_threshold]
numerator = '1'
denominator = '3'

# Define for Chain Beta (betanet-1)
[[chains]]
id = 'betanet-1'
rpc_addr = 'http://localhost:27657'
grpc_addr = 'http://localhost:9190'
event_source = { mode = 'push', url = 'ws://localhost:27657/websocket', batch_delay = '50ms' }
rpc_timeout = '15s'
account_prefix = 'wasm'
key_name = 'relayer-beta'
store_prefix = 'ibc'
gas_price = { price = 0.025, denom = 'ubeta' }
gas_multiplier = 1.2
max_gas = 600000
clock_drift = '15s'
max_block_time = '30s'
trusting_period = '10hours'
memo_prefix = 'hermes-betanet'
[chains.trust_threshold]
numerator = '1'
denominator = '3'

# Define for Chain Gamma (gammanet-1)
[[chains]]
id = 'gammanet-1'
rpc_addr = 'http://localhost:28657'
grpc_addr = 'http://localhost:9290'
event_source = { mode = 'push', url = 'ws://localhost:28657/websocket', batch_delay = '50ms' }
rpc_timeout = '15s'
account_prefix = 'wasm'
key_name = 'relayer-gamma'
store_prefix = 'ibc'
gas_price = { price = 0.025, denom = 'ugamma' }
gas_multiplier = 1.2
max_gas = 600000
clock_drift = '15s'
max_block_time = '30s'
trusting_period = '10hours'
memo_prefix = 'hermes-gammanet'
[chains.trust_threshold]
numerator = '1'
denominator = '3'

# Mode configuration (enable all necessary components)
[mode]
[mode.clients]
enabled = true
refresh = true
misbehaviour = true
[mode.connections]
enabled = true
[mode.channels]
enabled = true
[mode.packets]
enabled = true
clear_interval = 100
clear_on_start = true
tx_confirmation = true # Wait for txs to be confirmed
auto_register_counterparty_payee = false
EOF

echo "Hermes config.toml created at $HOME/.hermes/config.toml"
