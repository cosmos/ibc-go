use cosmwasm_std::entry_point;
use cosmwasm_std::{to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response};
use ibc_go_proto::ibc::{
    core::client::v1::Height as IbcProtoHeight,
    lightclients::wasm::v1::ClientState as WasmClientState,
};
use prost::Message;
use tendermint_proto::google::protobuf::Any;

use crate::error::ContractError;
use crate::msg::{
    CheckForMisbehaviourResult, ExecuteMsg, ExportMetadataResult, Height, InstantiateMsg, QueryMsg,
    StatusResult, SudoMsg, TimestampAtHeightResult, UpdateStateResult,
};
use crate::state::{consensus_db_key, HOST_CLIENT_STATE_KEY};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let client_state = WasmClientState {
        checksum: msg.checksum.into(),
        data: msg.client_state.clone().into(),
        latest_height: Some(IbcProtoHeight {
            revision_number: 0,
            revision_height: 1,
        }),
    };

    let client_state_any = Any::from_msg(&client_state).unwrap();

    deps.storage.set(
        HOST_CLIENT_STATE_KEY.as_bytes(),
        client_state_any.encode_to_vec().as_slice(),
    );

    let height = Height {
        revision_number: 0,
        revision_height: 1,
    };
    deps.storage.set(
        consensus_db_key(&height).as_bytes(),
        msg.consensus_state.as_slice(),
    );

    Ok(Response::default())
}

#[entry_point]
pub fn sudo(_deps: DepsMut, _env: Env, msg: SudoMsg) -> Result<Response, ContractError> {
    let result = match msg {
        SudoMsg::VerifyMembership(_) => verify_membership()?,
        SudoMsg::VerifyNonMembership(_) => verify_non_membership()?,
        SudoMsg::UpdateState(_) => update_state()?,
        SudoMsg::UpdateStateOnMisbehaviour(_) => unimplemented!(),
        SudoMsg::VerifyUpgradeAndUpdateState(_) => unimplemented!(),
        SudoMsg::MigrateClientStore(_) => unimplemented!(),
    };

    Ok(Response::default().set_data(result))
}

pub fn verify_membership() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&Ok::<(), ()>(()))?)
}

pub fn verify_non_membership() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&Ok::<(), ()>(()))?)
}

pub fn update_state() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&UpdateStateResult { heights: vec![] })?)
}

#[entry_point]
pub fn execute(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    unimplemented!()
}

#[entry_point]
pub fn query(_deps: Deps, env: Env, msg: QueryMsg) -> Result<Binary, ContractError> {
    match msg {
        QueryMsg::VerifyClientMessage(_) => verify_client_message(),
        QueryMsg::CheckForMisbehaviour(_) => check_for_misbehaviour(),
        QueryMsg::TimestampAtHeight(_) => timestamp_at_height(env),
        QueryMsg::Status(_) => status(),
        QueryMsg::ExportMetadata(_) => export_metadata(),
    }
}

pub fn verify_client_message() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&Ok::<(), ()>(()))?)
}

pub fn check_for_misbehaviour() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&CheckForMisbehaviourResult {
        found_misbehaviour: false,
    })?)
}

pub fn timestamp_at_height(env: Env) -> Result<Binary, ContractError> {
    let now = env.block.time.seconds();
    Ok(to_json_binary(&TimestampAtHeightResult { timestamp: now })?)
}

pub fn status() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&StatusResult {
        status: "Active".to_string(),
    })?)
}

pub fn export_metadata() -> Result<Binary, ContractError> {
    Ok(to_json_binary(&ExportMetadataResult {
        genesis_metadata: vec![],
    })?)
}

#[cfg(test)]
mod tests {
    mod instantiate_tests {
        use cosmwasm_std::{
            coins,
            testing::{message_info, mock_dependencies, mock_env},
        };

        use crate::{contract::instantiate, msg::InstantiateMsg};

        #[test]
        fn test_instantiate() {
            let mut deps = mock_dependencies();
            let creator = deps.api.addr_make("creator");
            let info = message_info(&creator, &coins(1, "uatom"));

            let msg = InstantiateMsg {
                client_state: "does not matter yet".as_bytes().into(),
                consensus_state: "also does not matter yet".as_bytes().into(),
                checksum: "yet another that does not matter yet".as_bytes().into(),
            };

            let res = instantiate(deps.as_mut(), mock_env(), info, msg).unwrap();
            assert_eq!(0, res.messages.len());
        }
    }

    mod sudo_tests {
        use cosmwasm_std::{
            testing::{mock_dependencies, mock_env},
            Binary,
        };

        use crate::{
            contract::sudo,
            msg::{
                Height, MerklePath, SudoMsg, UpdateStateMsg, VerifyMembershipMsg,
                VerifyNonMembershipMsg,
            },
        };

        #[test]
        fn test_verify_membership() {
            let mut deps = mock_dependencies();
            let msg = SudoMsg::VerifyMembership(VerifyMembershipMsg {
                height: Height {
                    revision_number: 0,
                    revision_height: 1,
                },
                delay_time_period: 0,
                delay_block_period: 0,
                proof: Binary::default(),
                merkle_path: MerklePath { key_path: vec![] },
                value: Binary::default(),
            });
            let res = sudo(deps.as_mut(), mock_env(), msg).unwrap();
            assert_eq!(0, res.messages.len());
        }

        #[test]
        fn test_verify_non_membership() {
            let mut deps = mock_dependencies();
            let msg = SudoMsg::VerifyNonMembership(VerifyNonMembershipMsg {
                height: Height {
                    revision_number: 0,
                    revision_height: 1,
                },
                delay_time_period: 0,
                delay_block_period: 0,
                proof: Binary::default(),
                merkle_path: MerklePath { key_path: vec![] },
            });
            let res = sudo(deps.as_mut(), mock_env(), msg).unwrap();
            assert_eq!(0, res.messages.len());
        }

        #[test]
        fn test_update_state() {
            let mut deps = mock_dependencies();
            let msg = SudoMsg::UpdateState(UpdateStateMsg {
                client_message: Binary::default(),
            });
            let res = sudo(deps.as_mut(), mock_env(), msg).unwrap();
            assert_eq!(0, res.messages.len());
        }
    }

    mod query_tests {
        use cosmwasm_std::{
            from_json,
            testing::{mock_dependencies, mock_env},
            Binary,
        };

        use crate::{
            contract::query,
            msg::{
                CheckForMisbehaviourMsg, CheckForMisbehaviourResult, ExportMetadataMsg,
                ExportMetadataResult, Height, QueryMsg, StatusMsg, StatusResult,
                TimestampAtHeightMsg, TimestampAtHeightResult, VerifyClientMessageMsg,
            },
        };

        #[test]
        fn test_verify_client_message() {
            let deps = mock_dependencies();
            query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::VerifyClientMessage(VerifyClientMessageMsg {
                    client_message: Binary::default(),
                }),
            )
            .unwrap();
        }

        #[test]
        fn test_check_for_misbehaviour() {
            let deps = mock_dependencies();
            let res = query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::CheckForMisbehaviour(CheckForMisbehaviourMsg {
                    client_message: Binary::default(),
                }),
            )
            .unwrap();
            let misbehaviour_result: CheckForMisbehaviourResult = from_json(&res).unwrap();
            assert!(!misbehaviour_result.found_misbehaviour);
        }

        #[test]
        fn test_timestamp_at_height() {
            let deps = mock_dependencies();
            let res = query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::TimestampAtHeight(TimestampAtHeightMsg {
                    height: Height {
                        revision_number: 0,
                        revision_height: 1,
                    },
                }),
            )
            .unwrap();
            let timestamp_at_height_result: TimestampAtHeightResult = from_json(&res).unwrap();
            assert_eq!(
                mock_env().block.time.seconds(),
                timestamp_at_height_result.timestamp
            );
        }

        #[test]
        fn test_status() {
            let deps = mock_dependencies();
            let res = query(deps.as_ref(), mock_env(), QueryMsg::Status(StatusMsg {})).unwrap();
            let status_response: StatusResult = from_json(&res).unwrap();
            assert_eq!("Active", status_response.status);
        }

        #[test]
        fn test_export_metadata() {
            let deps = mock_dependencies();
            let res = query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::ExportMetadata(ExportMetadataMsg {}),
            )
            .unwrap();
            let export_metadata_result: ExportMetadataResult = from_json(&res).unwrap();
            assert_eq!(0, export_metadata_result.genesis_metadata.len());
        }
    }
}
