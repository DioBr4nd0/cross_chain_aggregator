#!/bin/bash

IBC_PATHS_ENV_FILE="ibc_paths.env"

log_info() {
    echo "[INFO] $1" >&2  # Send logs to stderr to avoid capture
}

log_error() {
    echo "[ERROR] $1" >&2
}

log_success() {
    echo "[SUCCESS] $1" >&2
}

save_ibc_var() {
    local var_name="$1"
    local var_value="$2"
    echo "export $var_name=\"$var_value\"" >> "$IBC_PATHS_ENV_FILE"
    export "$var_name"="$var_value"
    log_success "Saved $var_name=$var_value"
}

load_ibc_var() {
    local var_name="$1"
    if [ -f "$IBC_PATHS_ENV_FILE" ]; then
        source "$IBC_PATHS_ENV_FILE"
        eval "echo \$$var_name"
    else
        echo ""
    fi
}

reset_ibc_env() {
    log_info "Resetting IBC environment file: $IBC_PATHS_ENV_FILE"
    > "$IBC_PATHS_ENV_FILE"
}

check_hermes() {
    if ! command -v hermes &> /dev/null; then
        log_error "Hermes not found in PATH"
        return 1
    fi
    log_success "Hermes found"
    return 0
}

check_jq() {
    if ! command -v jq &> /dev/null; then
        log_error "jq not found. Install with: sudo apt install jq"
        return 1
    fi
    log_success "jq found"
    return 0
}
