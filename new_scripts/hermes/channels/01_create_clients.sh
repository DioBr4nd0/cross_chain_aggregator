#!/bin/bash
set -e
source ./ibc_env_vars.sh

log_info "=== Starting IBC Client Creation ==="

# Check prerequisites
check_hermes || exit 1
check_jq || exit 1

CHAINS=("alphanet-1" "betanet-1" "gammanet-1")

create_client_pair() {
    local host_chain="$1"
    local reference_chain="$2"
    
    log_info "Creating client on $host_chain for $reference_chain"
    
    # Execute hermes command and capture output
    local output
    local exit_code
    
    output=$(hermes --json create client --host-chain "$host_chain" --reference-chain "$reference_chain" 2>&1)
    exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        # Parse client ID from successful output
        local client_id
        client_id=$(echo "$output" | jq -r '.result.client_id // .client_id // empty' 2>/dev/null)
        
        if [ -n "$client_id" ]; then
            log_success "Client created: $host_chain -> $reference_chain (ID: $client_id)"
            save_ibc_var "CLIENT_${host_chain//-/_}_${reference_chain//-/_}" "$client_id"
        else
            log_success "Client created: $host_chain -> $reference_chain (ID parsing failed, but command succeeded)"
        fi
    else
        # Check if client already exists
        if echo "$output" | grep -qi "client already exists\|already created"; then
            log_info "Client already exists: $host_chain -> $reference_chain"
        else
            log_error "Failed to create client $host_chain -> $reference_chain"
            log_error "Output: $output"
            return 1
        fi
    fi
    
    return 0
}

# Create all client pairs
log_info "Creating clients between all chain pairs..."

for i in "${!CHAINS[@]}"; do
    for j in "${!CHAINS[@]}"; do
        if [ $i -ne $j ]; then
            create_client_pair "${CHAINS[$i]}" "${CHAINS[$j]}"
            sleep 2  # Brief pause between client creations
        fi
    done
done

log_success "=== IBC Client Creation Completed ==="
