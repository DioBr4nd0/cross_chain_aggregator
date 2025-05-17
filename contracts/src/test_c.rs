use cosmwasm_std::{entry_point, to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use cw_storage_plus::{Item, Map};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Person {
    pub name: String,
    pub favourite_number: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg{}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum ExecuteMsg{
    SetFavouriteNumber {number: u64},
    AddPerson {name: String, favourite_number: u64}
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum QueryMsg{
    GetFavouriteNumber {},
    GetPerson {name: String},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct FavouriteNumberResponse {
    pub favourite_number : u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct PersonResponse {
    pub person: Option<Person>,
}

const FAVOURITE_NUMBER: Item<u64> = Item::new("favourite_number");
pub const LIST_OF_PEOPLE: Map<String, Person> = Map::new("list_of_people");

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response>{
    FAVOURITE_NUMBER.save(deps.storage, &0)?;
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
        ExecuteMsg::SetFavouriteNumber { number } => set_favourite_number(deps, number),
        ExecuteMsg::AddPerson { name, favourite_number } => add_person(deps, name, favourite_number),
    }
}

fn set_favourite_number(deps: DepsMut, number: u64) -> StdResult<Response> {
    FAVOURITE_NUMBER.save(deps.storage, &number)?;
    Ok(Response::new().add_attribute("method", "set_favourite_number"))
}

fn add_person(deps: DepsMut, name: String, favourite_number: u64) -> StdResult<Response> {
    let person = Person { name: name.clone(), favourite_number };
    LIST_OF_PEOPLE.save(deps.storage, &name, &person)?;
    Ok(Response::new().add_attribute("method", "add_person"))
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetFavouriteNumber {} => to_json_binary(&query_favourite_number(deps)?),
        QueryMsg::GetPerson { name } => to_json_binary(&query_person(deps, name)?),
    }
}

fn query_favourite_number(deps: Deps) -> StdResult<FavouriteNumberResponse> {
    let favourite_number = FAVOURITE_NUMBER.load(deps.storage)?;
    Ok(FavouriteNumberResponse { favourite_number })
}

fn query_person(deps: Deps, name: String) -> StdResult<PersonResponse> {
    let person = LIST_OF_PEOPLE.may_load(deps.storage, &name)?;
    Ok(PersonResponse { person })
}