use cosmwasm_std::{ entry_point, to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult};
use serde::{Deserialize, Serialize};
use cw_storage_plus::{Item, Map};
use schemars::JsonSchema;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Token{
    pub from:String,
    pub to:String,
    pub rate:f64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg{
    pub admin_address:String,
    exchange_rates:Vec<Token>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum ExecuteMsg{
    SwapTokens { target_output_denom: String, min_output_amount: u128 },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum QueryMsg{
    GetExchangeRate { input_denom: String, output_denom: String },
    GetAdmin {}
}
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GetExchangeRateResponse {
    pub rate:f64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GetAdminResponse {
    admin_address:String,
}
const admin_address:Item<String> = Item::new("admin_address");
pub const EXCHANGE_RATES:Map<String, Token> = Map::new("exchange_rates"); 


#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response> {
    admin_address.save(deps.storage, &_msg.admin_address);
    for i in _msg.exchange_rates {
        EXCHANGE_RATES.save(deps.storage, i.from+"-"+i.to.as_str(), &Token { from: i.from, to: i.to, rate: i.rate });
    }
    Ok(Response::new().add_attribute("method", "instantiate"))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response>{
    match msg {
        ExecuteMsg::SwapTokens { target_output_denom, min_output_amount } => swap_tokens(deps,target_output_denom, min_output_amount),
    }
}

fn swap_tokens(deps:DepsMut, target_output_denom:String, min_output_amount:u128) -> StdResult<Response>{

    Ok(Response::new().add_attribute("tokens swapped", target_output_denom))
}

#[entry_point]
pub fn query(deps:Deps, _env:Env, msg:QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetAdmin { } => to_json_binary(&get_admin(deps),
        QueryMsg::GetExchangeRate { input_denom:String, output_denom:String } => to_json_binary(&get_exchange_rate(deps, input_denom, output_denom))
    }
}

pub fn get_admin(deps:Deps) ->StdResult<GetAdminResponse>{
    let adminAddress =  admin_address.load(deps)?;
    Ok(GetAdminResponse { admin_address: adminAddress })
}

pub fn get_exchange_rate(deps:Deps, from:String, to:String) -> StdResult<GetExchangeRateResponse> {
    let x = EXCHANGE_RATES.may_load(store, from+"-"+to.as_str())?;
    let rate =  x.unwrap();
    Ok(GetExchangeRateResponse { rate })
}