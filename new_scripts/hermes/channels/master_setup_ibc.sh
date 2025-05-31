#!/bin/bash
set -e
source ./ibc_env_vars.sh

log_info "===== STARTING COMPLETE IBC SETUP ====="

# Make all scripts executable
chmod +x ibc_env_vars.sh
chmod +x 01_create_clients.sh
chmod +x 02_create_connections.sh
chmod +x 03_create_channels.sh

# Reset environment file for fresh start
reset_ibc_env

log_info "Step 1: Creating IBC Clients"
log_info "=============================="
./01_create_clients.sh

log_info ""
log_info "Step 2: Creating IBC Connections"
log_info "================================="
./02_create_connections.sh

log_info ""
log_info "Step 3: Creating IBC Channels"
log_info "=============================="
./03_create_channels.sh

log_info ""
log_success "===== IBC SETUP COMPLETED SUCCESSFULLY ====="
log_info ""
log_info "Generated environment file: $IBC_PATHS_ENV_FILE"
log_info "Contents:"
log_info "--------"
cat "$IBC_PATHS_ENV_FILE"
log_info "--------"
log_info ""
log_info "NEXT STEPS:"
log_info "1. Update your backend/config/app_config.go with the channel IDs above"
log_info "2. Example for backend config:"
log_info "   {FromChainID: \"alphanet-1\", ToChainID: \"betanet-1\", ChannelID: \"$(load_ibc_var CHAN_alphanet_1_betanet_1_transfer)\", PortID: \"transfer\"}"
log_info ""
log_success "Setup completed successfully!"
