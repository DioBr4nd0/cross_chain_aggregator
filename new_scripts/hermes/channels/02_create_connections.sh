#!/bin/bash
set -e
source ./ibc_env_vars.sh

log_info "=== Starting IBC Connection Creation ==="

check_hermes || exit 1
check_jq || exit 1

query_existing_connection() {
    local chain_a="$1"
    local chain_b="$2"
    
    # Query without logging to avoid capturing log messages
    local query_output
    query_output=$(hermes --json query connections --chain "$chain_a" 2>/dev/null || echo '{"connections": []}')
    
    if [ -z "$query_output" ] || [ "$query_output" = "null" ]; then
        query_output='{"connections": []}'
    fi
    
    # Return only the connection ID, no log messages
    echo "$query_output" | jq -r ".connections[]? | select(.counterparty.chain_id == \"$chain_b\") | .id" 2>/dev/null | head -n1
}

create_connection_pair() {
    local chain_a="$1"
    local chain_b="$2"
    
    log_info "Processing connection: $chain_a <-> $chain_b"
    
    # Check for existing connection - this should return only ID or empty string
    local existing_conn
    existing_conn=$(query_existing_connection "$chain_a" "$chain_b")
    
    if [ -n "$existing_conn" ] && [ "$existing_conn" != "null" ]; then
        log_info "Found existing connection: $existing_conn"
        save_ibc_var "CONN_${chain_a//-/_}_${chain_b//-/_}" "$existing_conn"
        return 0
    fi
    
    log_info "No existing connection found. Creating new connection..."
    
    local output
    local exit_code
    
    output=$(hermes --json create connection --a-chain "$chain_a" --b-chain "$chain_b" 2>&1)
    exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        log_error "Failed to create connection $chain_a <-> $chain_b"
        log_error "Hermes output: $output"
        return 1
    fi
    
    # Parse connection IDs
    local conn_id_a conn_id_b
    conn_id_a=$(echo "$output" | jq -r '.result.a_side.connection_id // empty' 2>/dev/null)
    conn_id_b=$(echo "$output" | jq -r '.result.b_side.connection_id // empty' 2>/dev/null)
    
    if [ -z "$conn_id_a" ] || [ -z "$conn_id_b" ] || [ "$conn_id_a" = "null" ] || [ "$conn_id_b" = "null" ]; then
        log_error "Failed to parse connection IDs from output"
        log_error "Raw output: $output"
        return 1
    fi
    
    log_success "Connection created successfully!"
    log_success "  $chain_a: $conn_id_a"
    log_success "  $chain_b: $conn_id_b"
    
    # Save only the actual connection IDs
    save_ibc_var "CONN_${chain_a//-/_}_${chain_b//-/_}" "$conn_id_a"
    save_ibc_var "CONN_${chain_b//-/_}_${chain_a//-/_}" "$conn_id_b"
    
    log_info "Waiting for connection handshake (30 seconds)..."
    sleep 30
    
    return 0
}

log_info "Creating connections between chain pairs..."

create_connection_pair "alphanet-1" "betanet-1"
create_connection_pair "alphanet-1" "gammanet-1"
create_connection_pair "betanet-1" "gammanet-1"

log_success "=== IBC Connection Creation Completed ==="
