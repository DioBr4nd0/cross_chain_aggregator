use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{
    entry_point, to_json_binary, BankMsg, Binary, Coin, Decimal, Deps, DepsMut, Env,
    MessageInfo, Response, StdError, StdResult, Uint128,
};
use cw_storage_plus::{Item, Map};

// State storage
#[cw_serde]
pub struct Config {
    pub admin: String,
    pub dex_name: String,
}

// Exchange rate between two tokens
#[cw_serde]
pub struct ExchangeRate {
    pub from_denom: String,
    pub to_denom: String,
    pub rate: Decimal, // How many to_denom tokens for 1 from_denom token
}

// Store contract configuration
pub const CONFIG: Item<Config> = Item::new("config");
// Store exchange rates: (from_denom, to_denom) -> ExchangeRate
pub const EXCHANGE_RATES: Map<(&str, &str), ExchangeRate> = Map::new("exchange_rates");

// Contract messages
#[cw_serde]
pub struct InstantiateMsg {
    pub admin: String,
    pub dex_name: String,
    pub initial_rates: Vec<ExchangeRate>,
}

#[cw_serde]
pub enum ExecuteMsg {
    // Swap tokens based on predefined rates
    Swap {
        to_denom: String,
        min_output: Uint128,
    },
    // Admin function to update rates
    UpdateRate {
        from_denom: String,
        to_denom: String,
        new_rate: Decimal,
    },
    // Admin function to add liquidity (just sends tokens to contract)
    AddLiquidity {},
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    // Get the exchange rate between two tokens
    #[returns(RateResponse)]
    GetRate {
        from_denom: String,
        to_denom: String,
    },
    // Get all supported token pairs
    #[returns(PairsResponse)]
    GetPairs {},
    // Get contract config
    #[returns(ConfigResponse)]
    GetConfig {},
}

// Query responses
#[cw_serde]
pub struct RateResponse {
    pub rate: Decimal,
}

#[cw_serde]
pub struct PairsResponse {
    pub pairs: Vec<(String, String)>,
}

#[cw_serde]
pub struct ConfigResponse {
    pub admin: String,
    pub dex_name: String,
}

// Contract entry points
#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    // Save contract configuration
    let config = Config {
        admin: msg.admin.clone(),
        dex_name: msg.dex_name.clone(),
    };
    CONFIG.save(deps.storage, &config)?;

    // Save initial exchange rates
    for rate in msg.initial_rates {
        if rate.rate.is_zero() || rate.rate < Decimal::zero() {
            return Err(StdError::generic_err("Rate must be positive"));
        }
        EXCHANGE_RATES.save(
            deps.storage,
            (&rate.from_denom, &rate.to_denom),
            &rate,
        )?;
    }

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("admin", msg.admin)
        .add_attribute("dex_name", msg.dex_name))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::Swap { to_denom, min_output } => execute_swap(deps, info, to_denom, min_output),
        ExecuteMsg::UpdateRate { from_denom, to_denom, new_rate } => {
            execute_update_rate(deps, info, from_denom, to_denom, new_rate)
        }
        ExecuteMsg::AddLiquidity {} => execute_add_liquidity(deps, info),
    }
}

// Execute swap function
fn execute_swap(
    deps: DepsMut,
    info: MessageInfo,
    to_denom: String,
    min_output: Uint128,
) -> StdResult<Response> {
    // Ensure exactly one coin is sent
    if info.funds.len() != 1 {
        return Err(StdError::generic_err("Must send exactly one coin type"));
    }

    let input_coin = &info.funds[0];
    let from_denom = input_coin.denom.clone();
    let input_amount = input_coin.amount;

    // Get exchange rate
    let rate = EXCHANGE_RATES
        .may_load(deps.storage, (&from_denom, &to_denom))?
        .ok_or_else(|| StdError::generic_err(format!("No rate for {}-{}", from_denom, to_denom)))?;

    // Calculate output amount
    let output_amount = input_amount * rate.rate;

    // Check minimum output
    if output_amount < min_output {
        return Err(StdError::generic_err(format!(
            "Output amount {} less than minimum {}",
            output_amount, min_output
        )));
    }

    // Send output tokens to user
    let bank_msg = BankMsg::Send {
        to_address: info.sender.to_string(),
        amount: vec![Coin {
            denom: to_denom.clone(),
            amount: output_amount,
        }],
    };

    Ok(Response::new()
        .add_message(bank_msg)
        .add_attribute("action", "swap")
        .add_attribute("from_denom", from_denom)
        .add_attribute("to_denom", to_denom)
        .add_attribute("input_amount", input_amount.to_string())
        .add_attribute("output_amount", output_amount.to_string()))
}

// Admin function to update exchange rate
fn execute_update_rate(
    deps: DepsMut,
    info: MessageInfo,
    from_denom: String,
    to_denom: String,
    new_rate: Decimal,
) -> StdResult<Response> {
    // Check admin
    let config = CONFIG.load(deps.storage)?;
    if info.sender.to_string() != config.admin {
        return Err(StdError::generic_err("Unauthorized"));
    }

    // Validate rate
    if new_rate.is_zero() || new_rate < Decimal::zero() {
        return Err(StdError::generic_err("Rate must be positive"));
    }

    // Update rate
    let rate = ExchangeRate {
        from_denom: from_denom.clone(),
        to_denom: to_denom.clone(),
        rate: new_rate,
    };
    EXCHANGE_RATES.save(deps.storage, (&from_denom, &to_denom), &rate)?;

    Ok(Response::new()
        .add_attribute("action", "update_rate")
        .add_attribute("from_denom", from_denom)
        .add_attribute("to_denom", to_denom)
        .add_attribute("new_rate", new_rate.to_string()))
}

// Admin function to add liquidity
fn execute_add_liquidity(deps: DepsMut, info: MessageInfo) -> StdResult<Response> {
    // Check admin
    let config = CONFIG.load(deps.storage)?;
    if info.sender.to_string() != config.admin {
        return Err(StdError::generic_err("Unauthorized"));
    }

    // No action needed - funds are automatically sent to contract
    Ok(Response::new()
        .add_attribute("action", "add_liquidity")
        .add_attribute("amount", format!("{:?}", info.funds)))
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetRate { from_denom, to_denom } => to_json_binary(&query_rate(deps, from_denom, to_denom)?),
        QueryMsg::GetPairs {} => to_json_binary(&query_pairs(deps)?),
        QueryMsg::GetConfig {} => to_json_binary(&query_config(deps)?),
    }
}

// Query exchange rate
fn query_rate(deps: Deps, from_denom: String, to_denom: String) -> StdResult<RateResponse> {
    let rate = EXCHANGE_RATES
        .may_load(deps.storage, (&from_denom, &to_denom))?
        .ok_or_else(|| StdError::generic_err(format!("No rate for {}-{}", from_denom, to_denom)))?;

    Ok(RateResponse { rate: rate.rate })
}

// Query all supported pairs
fn query_pairs(deps: Deps) -> StdResult<PairsResponse> {
    let pairs: Vec<(String, String)> = EXCHANGE_RATES
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .map(|item| {
            let ((from, to), _) = item.unwrap();
            (from.to_string(), to.to_string())
        })
        .collect();

    Ok(PairsResponse { pairs })
}

// Query contract config
fn query_config(deps: Deps) -> StdResult<ConfigResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(ConfigResponse {
        admin: config.admin,
        dex_name: config.dex_name,
    })
}
