"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[26837],{66123:(e,s,n)=>{n.r(s),n.d(s,{assets:()=>a,contentTitle:()=>t,default:()=>h,frontMatter:()=>r,metadata:()=>c,toc:()=>d});var i=n(85893),o=n(11151);const r={title:"Migrations",sidebar_label:"Migrations",sidebar_position:9,slug:"/ibc/light-clients/wasm/migrations"},t="Migrations",c={id:"light-clients/wasm/migrations",title:"Migrations",description:"This guide provides instructions for migrating 08-wasm versions.",source:"@site/docs/03-light-clients/04-wasm/09-migrations.md",sourceDirName:"03-light-clients/04-wasm",slug:"/ibc/light-clients/wasm/migrations",permalink:"/main/ibc/light-clients/wasm/migrations",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:9,frontMatter:{title:"Migrations",sidebar_label:"Migrations",sidebar_position:9,slug:"/ibc/light-clients/wasm/migrations"},sidebar:"defaultSidebar",previous:{title:"Client",permalink:"/main/ibc/light-clients/wasm/client"},next:{title:"Overview",permalink:"/main/ibc/light-clients/tendermint/overview"}},a={},d=[{value:"From ibc-go v8.4.x to ibc-go v9.0.x",id:"from-ibc-go-v84x-to-ibc-go-v90x",level:2},{value:"Chains",id:"chains",level:3},{value:"From v0.3.0+ibc-go-v8.3-wasmvm-v2.0 to v0.4.1-ibc-go-v8.4-wasmvm-v2.0",id:"from-v030ibc-go-v83-wasmvm-v20-to-v041-ibc-go-v84-wasmvm-v20",level:2},{value:"Contract developers",id:"contract-developers",level:3},{value:"From v0.2.0+ibc-go-v7.3-wasmvm-v1.5 to v0.3.1-ibc-go-v7.4-wasmvm-v1.5",id:"from-v020ibc-go-v73-wasmvm-v15-to-v031-ibc-go-v74-wasmvm-v15",level:2},{value:"Contract developers",id:"contract-developers-1",level:3},{value:"From v0.2.0+ibc-go-v8.3-wasmvm-v2.0 to v0.3.0-ibc-go-v8.3-wasmvm-v2.0",id:"from-v020ibc-go-v83-wasmvm-v20-to-v030-ibc-go-v83-wasmvm-v20",level:2},{value:"Contract developers",id:"contract-developers-2",level:3},{value:"From v0.1.1+ibc-go-v7.3-wasmvm-v1.5 to v0.2.0-ibc-go-v7.3-wasmvm-v1.5",id:"from-v011ibc-go-v73-wasmvm-v15-to-v020-ibc-go-v73-wasmvm-v15",level:2},{value:"Contract developers",id:"contract-developers-3",level:3},{value:"From ibc-go v7.3.x to ibc-go v8.0.x",id:"from-ibc-go-v73x-to-ibc-go-v80x",level:2},{value:"Chains",id:"chains-1",level:3},{value:"From v0.1.0+ibc-go-v8.0-wasmvm-v1.5 to v0.2.0-ibc-go-v8.3-wasmvm-v2.0",id:"from-v010ibc-go-v80-wasmvm-v15-to-v020-ibc-go-v83-wasmvm-v20",level:2},{value:"Chains",id:"chains-2",level:3}];function l(e){const s={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",li:"li",p:"p",pre:"pre",ul:"ul",...(0,o.a)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(s.h1,{id:"migrations",children:"Migrations"}),"\n",(0,i.jsx)(s.p,{children:"This guide provides instructions for migrating 08-wasm versions."}),"\n",(0,i.jsx)(s.p,{children:"Please note that the following releases are retracted. Please refer to the appropriate migrations section for upgrading."}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-bash",children:"v0.3.1-0.20240717085919-bb71eef0f3bf => v0.3.0+ibc-go-v8.3-wasmvm-v2.0\nv0.2.1-0.20240717085554-570d057959e3 => v0.2.0+ibc-go-v7.6-wasmvm-v1.5\nv0.2.1-0.20240523101951-4b45d1822fb6 => v0.2.0+ibc-go-v8.3-wasmvm-v2.0\nv0.1.2-0.20240412103620-7ee2a2452b79 => v0.1.1+ibc-go-v7.3-wasmvm-v1.5\nv0.1.1-0.20231213092650-57fcdb9a9a9d => v0.1.0+ibc-go-v8.0-wasmvm-v1.5\nv0.1.1-0.20231213092633-b306e7a706e1 => v0.1.0+ibc-go-v7.3-wasmvm-v1.5\n"})}),"\n",(0,i.jsx)(s.h2,{id:"from-ibc-go-v84x-to-ibc-go-v90x",children:"From ibc-go v8.4.x to ibc-go v9.0.x"}),"\n",(0,i.jsx)(s.h3,{id:"chains",children:"Chains"}),"\n",(0,i.jsxs)(s.ul,{children:["\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"Initialize"}),", ",(0,i.jsx)(s.code,{children:"Status"}),", ",(0,i.jsx)(s.code,{children:"GetTimestampAtHeight"}),", ",(0,i.jsx)(s.code,{children:"GetLatestHeight"}),", ",(0,i.jsx)(s.code,{children:"VerifyMembership"}),", ",(0,i.jsx)(s.code,{children:"VerifyNonMembership"}),", ",(0,i.jsx)(s.code,{children:"VerifyClientMessage"}),", ",(0,i.jsx)(s.code,{children:"UpdateState"})," and ",(0,i.jsx)(s.code,{children:"UpdateStateOnMisbehaviour"})," functions in ",(0,i.jsx)(s.code,{children:"ClientState"})," have been removed and all their logic has been moved to functions of the ",(0,i.jsx)(s.code,{children:"LightClientModule"}),"."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"MigrateContract"})," function has been removed from ",(0,i.jsx)(s.code,{children:"ClientState"}),"."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"})," payloads for ",(0,i.jsx)(s.code,{children:"SudoMsg"})," have been modified. The ",(0,i.jsx)(s.code,{children:"Path"})," field of both structs has been updated from ",(0,i.jsx)(s.code,{children:"v1.MerklePath"})," to ",(0,i.jsx)(s.code,{children:"v2.MerklePath"}),". The new ",(0,i.jsx)(s.code,{children:"v2.MerklePath"})," field contains a ",(0,i.jsx)(s.code,{children:"KeyPath"})," of ",(0,i.jsx)(s.code,{children:"[][]byte"})," as opposed to ",(0,i.jsx)(s.code,{children:"[]string"}),", see ",(0,i.jsx)(s.a,{href:"../../05-migrations/13-v8-to-v9.md#23-commitment",children:"23-commitment"}),". This supports proving values stored under keys which contain non-utf8 encoded symbols. As a result, the JSON field ",(0,i.jsx)(s.code,{children:"path"})," containing ",(0,i.jsx)(s.code,{children:"key_path"})," of both messages will marshal elements as a base64 encoded bytestrings. This is a breaking change for 08-wasm client contracts and they should be migrated to correctly support deserialisation of the ",(0,i.jsx)(s.code,{children:"v2.MerklePath"})," field."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"ExportMetadataMsg"})," struct has been removed and is no longer required for contracts to implement. Core IBC will handle exporting all key/value's written to the store by a light client contract."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"ZeroCustomFields"})," interface function has been removed from the ",(0,i.jsx)(s.code,{children:"ClientState"})," interface. Core IBC only used this function to set tendermint client states when scheduling an IBC software upgrade. The interface function has been replaced by a type assertion."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"MaxWasmByteSize"})," function has been removed in favor of the ",(0,i.jsx)(s.code,{children:"MaxWasmSize"})," constant."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"HasChecksum"}),", ",(0,i.jsx)(s.code,{children:"GetAllChecksums"})," and ",(0,i.jsx)(s.code,{children:"Logger"})," functions have been moved from the ",(0,i.jsx)(s.code,{children:"types"})," package to a method on the ",(0,i.jsx)(s.code,{children:"Keeper"})," type in the ",(0,i.jsx)(s.code,{children:"keeper"})," package."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"InitializePinnedCodes"})," function has been moved to a method on the ",(0,i.jsx)(s.code,{children:"Keeper"})," type in the ",(0,i.jsx)(s.code,{children:"keeper"})," package."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"CustomQuerier"}),", ",(0,i.jsx)(s.code,{children:"StargateQuerier"})," and ",(0,i.jsx)(s.code,{children:"QueryPlugins"})," types have been moved from the ",(0,i.jsx)(s.code,{children:"types"})," package to the ",(0,i.jsx)(s.code,{children:"keeper"})," package."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"NewDefaultQueryPlugins"}),", ",(0,i.jsx)(s.code,{children:"AcceptListStargateQuerier"})," and ",(0,i.jsx)(s.code,{children:"RejectCustomQuerier"})," functions has been moved from the ",(0,i.jsx)(s.code,{children:"types"})," package to the ",(0,i.jsx)(s.code,{children:"keeper"})," package."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"NewDefaultQueryPlugins"})," function signature has changed to take an argument: ",(0,i.jsx)(s.code,{children:"queryRouter ibcwasm.QueryRouter"}),"."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"AcceptListStargateQuerier"})," function signature has changed to take an additional argument: ",(0,i.jsx)(s.code,{children:"queryRouter ibcwasm.QueryRouter"}),"."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"WithQueryPlugins"})," function signature has changed to take in the ",(0,i.jsx)(s.code,{children:"QueryPlugins"})," type from the ",(0,i.jsx)(s.code,{children:"keeper"})," package (previously from the ",(0,i.jsx)(s.code,{children:"types"})," package)."]}),"\n",(0,i.jsxs)(s.li,{children:["The ",(0,i.jsx)(s.code,{children:"VMGasRegister"})," variable has been moved from the ",(0,i.jsx)(s.code,{children:"types"})," package to the ",(0,i.jsx)(s.code,{children:"keeper"})," package."]}),"\n"]}),"\n",(0,i.jsx)(s.h2,{id:"from-v030ibc-go-v83-wasmvm-v20-to-v041-ibc-go-v84-wasmvm-v20",children:"From v0.3.0+ibc-go-v8.3-wasmvm-v2.0 to v0.4.1-ibc-go-v8.4-wasmvm-v2.0"}),"\n",(0,i.jsx)(s.h3,{id:"contract-developers",children:"Contract developers"}),"\n",(0,i.jsxs)(s.p,{children:["Contract developers are required to update their JSON API message structure for the ",(0,i.jsx)(s.code,{children:"SudoMsg"})," payloads ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"}),".\nThe ",(0,i.jsx)(s.code,{children:"path"})," field on both JSON API messages has been renamed to ",(0,i.jsx)(s.code,{children:"merkle_path"}),"."]}),"\n",(0,i.jsx)(s.p,{children:"A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of these fields."}),"\n",(0,i.jsx)(s.h2,{id:"from-v020ibc-go-v73-wasmvm-v15-to-v031-ibc-go-v74-wasmvm-v15",children:"From v0.2.0+ibc-go-v7.3-wasmvm-v1.5 to v0.3.1-ibc-go-v7.4-wasmvm-v1.5"}),"\n",(0,i.jsx)(s.h3,{id:"contract-developers-1",children:"Contract developers"}),"\n",(0,i.jsxs)(s.p,{children:["Contract developers are required to update their JSON API message structure for the ",(0,i.jsx)(s.code,{children:"SudoMsg"})," payloads ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"}),".\nThe ",(0,i.jsx)(s.code,{children:"path"})," field on both JSON API messages has been renamed to ",(0,i.jsx)(s.code,{children:"merkle_path"}),"."]}),"\n",(0,i.jsx)(s.p,{children:"A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of these fields."}),"\n",(0,i.jsx)(s.h2,{id:"from-v020ibc-go-v83-wasmvm-v20-to-v030-ibc-go-v83-wasmvm-v20",children:"From v0.2.0+ibc-go-v8.3-wasmvm-v2.0 to v0.3.0-ibc-go-v8.3-wasmvm-v2.0"}),"\n",(0,i.jsx)(s.h3,{id:"contract-developers-2",children:"Contract developers"}),"\n",(0,i.jsxs)(s.p,{children:["The ",(0,i.jsx)(s.code,{children:"v0.3.0"})," release of 08-wasm for ibc-go ",(0,i.jsx)(s.code,{children:"v8.3.x"})," and above introduces a breaking change for client contract developers."]}),"\n",(0,i.jsxs)(s.p,{children:["The contract API ",(0,i.jsx)(s.code,{children:"SudoMsg"})," payloads ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"})," have been modified.\nThe encoding of the ",(0,i.jsx)(s.code,{children:"Path"})," field of both structs has been updated from ",(0,i.jsx)(s.code,{children:"v1.MerklePath"})," to ",(0,i.jsx)(s.code,{children:"v2.MerklePath"})," to support proving values stored under keys which contain non-utf8 encoded symbols."]}),"\n",(0,i.jsxs)(s.p,{children:["As a result, the ",(0,i.jsx)(s.code,{children:"Path"})," field now contains a ",(0,i.jsx)(s.code,{children:"MerklePath"})," composed of ",(0,i.jsx)(s.code,{children:"key_path"})," of ",(0,i.jsx)(s.code,{children:"[][]byte"})," as opposed to ",(0,i.jsx)(s.code,{children:"[]string"}),". The JSON field ",(0,i.jsx)(s.code,{children:"path"})," containing ",(0,i.jsx)(s.code,{children:"key_path"})," of both ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"})," structs will now marshal elements as base64 encoded bytestrings. See below for example JSON diff."]}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-diff",children:'{\n  "verify_membership": {\n    "height": {\n      "revision_height": 1\n    },\n    "delay_time_period": 0,\n    "delay_block_period": 0,\n    "proof":"dmFsaWQgcHJvb2Y=",\n    "path": {\n+      "key_path":["L2liYw==","L2tleS9wYXRo"]\n-      "key_path":["/ibc","/key/path"]\n    },\n    "value":"dmFsdWU="\n  }\n}\n'})}),"\n",(0,i.jsxs)(s.p,{children:["A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of ",(0,i.jsx)(s.code,{children:"key_path"})," from ",(0,i.jsx)(s.code,{children:"[]string"})," to ",(0,i.jsx)(s.code,{children:"[][]byte"}),".\nContract developers should familiarise themselves with the migration path offered by 08-wasm ",(0,i.jsx)(s.a,{href:"/main/ibc/light-clients/wasm/governance#migrating-an-existing-wasm-light-client-contract",children:"here"}),"."]}),"\n",(0,i.jsx)(s.p,{children:"An example of the required changes in a client contract may look like:"}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-diff",children:"#[cw_serde]\npub struct MerklePath {\n+   pub key_path: Vec<cosmwasm_std::Binary>,\n-   pub key_path: Vec<String>,\n}\n"})}),"\n",(0,i.jsxs)(s.p,{children:["Please refer to the ",(0,i.jsx)(s.a,{href:"https://docs.rs/cosmwasm-std/2.0.4/cosmwasm_std/struct.Binary.html",children:(0,i.jsx)(s.code,{children:"cosmwasm_std"})})," documentation for more information."]}),"\n",(0,i.jsx)(s.h2,{id:"from-v011ibc-go-v73-wasmvm-v15-to-v020-ibc-go-v73-wasmvm-v15",children:"From v0.1.1+ibc-go-v7.3-wasmvm-v1.5 to v0.2.0-ibc-go-v7.3-wasmvm-v1.5"}),"\n",(0,i.jsx)(s.h3,{id:"contract-developers-3",children:"Contract developers"}),"\n",(0,i.jsxs)(s.p,{children:["The ",(0,i.jsx)(s.code,{children:"v0.2.0"})," release of 08-wasm for ibc-go ",(0,i.jsx)(s.code,{children:"v7.6.x"})," and above introduces a breaking change for client contract developers."]}),"\n",(0,i.jsxs)(s.p,{children:["The contract API ",(0,i.jsx)(s.code,{children:"SudoMsg"})," payloads ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"})," have been modified.\nThe encoding of the ",(0,i.jsx)(s.code,{children:"Path"})," field of both structs has been updated from ",(0,i.jsx)(s.code,{children:"v1.MerklePath"})," to ",(0,i.jsx)(s.code,{children:"v2.MerklePath"})," to support proving values stored under keys which contain non-utf8 encoded symbols."]}),"\n",(0,i.jsxs)(s.p,{children:["As a result, the ",(0,i.jsx)(s.code,{children:"Path"})," field now contains a ",(0,i.jsx)(s.code,{children:"MerklePath"})," composed of ",(0,i.jsx)(s.code,{children:"key_path"})," of ",(0,i.jsx)(s.code,{children:"[][]byte"})," as opposed to ",(0,i.jsx)(s.code,{children:"[]string"}),". The JSON field ",(0,i.jsx)(s.code,{children:"path"})," containing ",(0,i.jsx)(s.code,{children:"key_path"})," of both ",(0,i.jsx)(s.code,{children:"VerifyMembershipMsg"})," and ",(0,i.jsx)(s.code,{children:"VerifyNonMembershipMsg"})," structs will now marshal elements as base64 encoded bytestrings. See below for example JSON diff."]}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-diff",children:'{\n  "verify_membership": {\n    "height": {\n      "revision_height": 1\n    },\n    "delay_time_period": 0,\n    "delay_block_period": 0,\n    "proof":"dmFsaWQgcHJvb2Y=",\n    "path": {\n+      "key_path":["L2liYw==","L2tleS9wYXRo"]\n-      "key_path":["/ibc","/key/path"]\n    },\n    "value":"dmFsdWU="\n  }\n}\n'})}),"\n",(0,i.jsxs)(s.p,{children:["A migration is required for existing 08-wasm client contracts in order to correctly handle the deserialisation of ",(0,i.jsx)(s.code,{children:"key_path"})," from ",(0,i.jsx)(s.code,{children:"[]string"})," to ",(0,i.jsx)(s.code,{children:"[][]byte"}),".\nContract developers should familiarise themselves with the migration path offered by 08-wasm ",(0,i.jsx)(s.a,{href:"/main/ibc/light-clients/wasm/governance#migrating-an-existing-wasm-light-client-contract",children:"here"}),"."]}),"\n",(0,i.jsx)(s.p,{children:"An example of the required changes in a client contract may look like:"}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-diff",children:"#[cw_serde]\npub struct MerklePath {\n+   pub key_path: Vec<cosmwasm_std::Binary>,\n-   pub key_path: Vec<String>,\n}\n"})}),"\n",(0,i.jsxs)(s.p,{children:["Please refer to the ",(0,i.jsx)(s.a,{href:"https://docs.rs/cosmwasm-std/2.0.4/cosmwasm_std/struct.Binary.html",children:(0,i.jsx)(s.code,{children:"cosmwasm_std"})})," documentation for more information."]}),"\n",(0,i.jsx)(s.h2,{id:"from-ibc-go-v73x-to-ibc-go-v80x",children:"From ibc-go v7.3.x to ibc-go v8.0.x"}),"\n",(0,i.jsx)(s.h3,{id:"chains-1",children:"Chains"}),"\n",(0,i.jsxs)(s.p,{children:["In the 08-wasm versions compatible with ibc-go v7.3.x and above from the v7 release line, the checksums of the uploaded Wasm bytecodes are all stored under a single key. From ibc-go v8.0.x the checksums are stored using ",(0,i.jsx)(s.a,{href:"https://docs.cosmos.network/v0.50/build/packages/collections#keyset",children:(0,i.jsx)(s.code,{children:"collections.KeySet"})}),", whose full functionality became available in Cosmos SDK v0.50. There is therefore an ",(0,i.jsx)(s.a,{href:"https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/modules/light-clients/08-wasm/module.go#L115-L118",children:"automatic migration handler"})," configured in the 08-wasm module to migrate the stored checksums to ",(0,i.jsx)(s.code,{children:"collections.KeySet"}),"."]}),"\n",(0,i.jsx)(s.h2,{id:"from-v010ibc-go-v80-wasmvm-v15-to-v020-ibc-go-v83-wasmvm-v20",children:"From v0.1.0+ibc-go-v8.0-wasmvm-v1.5 to v0.2.0-ibc-go-v8.3-wasmvm-v2.0"}),"\n",(0,i.jsxs)(s.p,{children:["The ",(0,i.jsx)(s.code,{children:"WasmEngine"})," interface has been updated to reflect changes in the function signatures of Wasm VM:"]}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-diff",children:"type WasmEngine interface {\n- StoreCode(code wasmvm.WasmCode) (wasmvm.Checksum, error)\n+ StoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvmtypes.Checksum, uint64, error)\n\n  StoreCodeUnchecked(code wasmvm.WasmCode) (wasmvm.Checksum, error)\n\n  Instantiate(\n    checksum wasmvm.Checksum,\n    env wasmvmtypes.Env,\n    info wasmvmtypes.MessageInfo,\n    initMsg []byte,\n    store wasmvm.KVStore,\n    goapi wasmvm.GoAPI,\n    querier wasmvm.Querier,\n    gasMeter wasmvm.GasMeter,\n    gasLimit uint64,\n    deserCost wasmvmtypes.UFraction,\n- ) (*wasmvmtypes.Response, uint64, error)\n+ ) (*wasmvmtypes.ContractResult, uint64, error)\n\n  Query(\n    checksum wasmvm.Checksum,\n    env wasmvmtypes.Env,\n    queryMsg []byte,\n    store wasmvm.KVStore,\n    goapi wasmvm.GoAPI,\n    querier wasmvm.Querier,\n    gasMeter wasmvm.GasMeter,\n    gasLimit uint64,\n    deserCost wasmvmtypes.UFraction,\n- ) ([]byte, uint64, error)\n+ ) (*wasmvmtypes.QueryResult, uint64, error)\n\n  Migrate(\n    checksum wasmvm.Checksum,\n    env wasmvmtypes.Env,\n    migrateMsg []byte,\n    store wasmvm.KVStore,\n    goapi wasmvm.GoAPI,\n    querier wasmvm.Querier,\n    gasMeter wasmvm.GasMeter,\n    gasLimit uint64,\n    deserCost wasmvmtypes.UFraction,\n- ) (*wasmvmtypes.Response, uint64, error)\n+ ) (*wasmvmtypes.ContractResult, uint64, error)\n\n  Sudo(\n    checksum wasmvm.Checksum,\n    env wasmvmtypes.Env,\n    sudoMsg []byte,\n    store wasmvm.KVStore,\n    goapi wasmvm.GoAPI,\n    querier wasmvm.Querier,\n    gasMeter wasmvm.GasMeter,\n    gasLimit uint64,\n    deserCost wasmvmtypes.UFraction,\n- ) (*wasmvmtypes.Response, uint64, error)\n+ ) (*wasmvmtypes.ContractResult, uint64, error)\n\n  GetCode(checksum wasmvm.Checksum) (wasmvm.WasmCode, error)\n\n  Pin(checksum wasmvm.Checksum) error\n\n  Unpin(checksum wasmvm.Checksum) error\n}\n"})}),"\n",(0,i.jsxs)(s.p,{children:["Similar changes were required in the functions of ",(0,i.jsx)(s.code,{children:"MockWasmEngine"})," interface."]}),"\n",(0,i.jsx)(s.h3,{id:"chains-2",children:"Chains"}),"\n",(0,i.jsxs)(s.p,{children:["The ",(0,i.jsx)(s.code,{children:"SupportedCapabilities"})," field of ",(0,i.jsx)(s.code,{children:"WasmConfig"})," is now of type ",(0,i.jsx)(s.code,{children:"[]string"}),":"]}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-diff",children:"type WasmConfig struct {\n  DataDir string\n- SupportedCapabilities string\n+ SupportedCapabilities []string\n  ContractDebugMode bool\n}\n"})})]})}function h(e={}){const{wrapper:s}={...(0,o.a)(),...e.components};return s?(0,i.jsx)(s,{...e,children:(0,i.jsx)(l,{...e})}):l(e)}},11151:(e,s,n)=>{n.d(s,{Z:()=>c,a:()=>t});var i=n(67294);const o={},r=i.createContext(o);function t(e){const s=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(s):{...s,...e}}),[s,e])}function c(e){let s;return s=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:t(e.components),i.createElement(r.Provider,{value:s},e.children)}}}]);