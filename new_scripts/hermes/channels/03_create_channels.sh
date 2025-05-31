#!/bin/bash
set -e
source ./ibc_env_vars.sh

log_info "=== Starting IBC Channel Creation ==="

check_hermes || exit 1
check_jq || exit 1

if [ -f "$IBC_PATHS_ENV_FILE" ]; then
    source "$IBC_PATHS_ENV_FILE"
    log_info "Loaded existing IBC environment variables"
else
    log_error "IBC environment file not found. Run connection creation first."
    exit 1
fi

query_existing_channel() {
    local chain_a="$1"
    local chain_b="$2"
    
    # Query without logging to avoid capturing log messages
    local query_output
    query_output=$(hermes --json query channels --chain "$chain_a" 2>/dev/null || echo '{"channels": []}')
    
    if [ -z "$query_output" ] || [ "$query_output" = "null" ]; then
        query_output='{"channels": []}'
    fi
    
    # Return only the channel ID, no log messages
    echo "$query_output" | jq -r ".channels[]? | select(.counterparty.chain_id == \"$chain_b\" and .port_id == \"transfer\") | .channel_id" 2>/dev/null | head -n1
}

create_channel_pair() {
    local chain_a="$1"
    local chain_b="$2"
    
    log_info "Processing channel: $chain_a <-> $chain_b (port: transfer)"
    
    # Get connection ID
    local conn_var="CONN_${chain_a//-/_}_${chain_b//-/_}"
    local conn_id
    conn_id=$(load_ibc_var "$conn_var")
    
    if [ -z "$conn_id" ]; then
        log_error "Connection ID not found for $chain_a -> $chain_b (variable: $conn_var)"
        return 1
    fi
    
    log_info "Using connection ID: $conn_id"
    
    # Check for existing channel - this should return only ID or empty string
    local existing_chan
    existing_chan=$(query_existing_channel "$chain_a" "$chain_b")
    
    if [ -n "$existing_chan" ] && [ "$existing_chan" != "null" ]; then
        log_info "Found existing channel: $existing_chan"
        save_ibc_var "CHAN_${chain_a//-/_}_${chain_b//-/_}_transfer" "$existing_chan"
        return 0
    fi
    
    log_info "No existing channel found. Creating new channel..."
    
    local output
    local exit_code
    
    output=$(hermes --json create channel --a-chain "$chain_a" --a-connection "$conn_id" --a-port transfer --b-port transfer --channel-version ics20-1 2>&1)
    exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        log_error "Failed to create channel $chain_a <-> $chain_b"
        log_error "Hermes output: $output"
        return 1
    fi
    
    # Parse channel IDs
    local chan_id_a chan_id_b
    chan_id_a=$(echo "$output" | jq -r '.result.a_side.channel_id // empty' 2>/dev/null)
    chan_id_b=$(echo "$output" | jq -r '.result.b_side.channel_id // empty' 2>/dev/null)
    
    if [ -z "$chan_id_a" ] || [ -z "$chan_id_b" ] || [ "$chan_id_a" = "null" ] || [ "$chan_id_b" = "null" ]; then
        log_error "Failed to parse channel IDs from output"
        log_error "Raw output: $output"
        return 1
    fi
    
    log_success "Channel created successfully!"
    log_success "  $chain_a: $chan_id_a"
    log_success "  $chain_b: $chan_id_b"
    
    # Save only the actual channel IDs
    save_ibc_var "CHAN_${chain_a//-/_}_${chain_b//-/_}_transfer" "$chan_id_a"
    save_ibc_var "CHAN_${chain_b//-/_}_${chain_a//-/_}_transfer" "$chan_id_b"
    
    log_info "Waiting for channel handshake (15 seconds)..."
    sleep 15
    
    return 0
}

log_info "Creating transfer channels between chain pairs..."

create_channel_pair "alphanet-1" "betanet-1"
create_channel_pair "alphanet-1" "gammanet-1"
create_channel_pair "betanet-1" "gammanet-1"

log_success "=== IBC Channel Creation Completed ==="
