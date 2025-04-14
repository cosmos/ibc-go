"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[97587],{99074:(e,n,a)=>{a.r(n),a.d(n,{assets:()=>s,contentTitle:()=>o,default:()=>p,frontMatter:()=>r,metadata:()=>c,toc:()=>l});var i=a(85893),t=a(11151);const r={title:"IBC middleware",sidebar_label:"IBC middleware",sidebar_position:1,slug:"/ibc/middleware/develop"},o="IBC middleware",c={id:"ibc/middleware/develop",title:"IBC middleware",description:"Learn how to write your own custom middleware to wrap an IBC application, and understand how to hook different middleware to IBC base applications to form different IBC application stacks",source:"@site/versioned_docs/version-v7.8.x/01-ibc/04-middleware/01-develop.md",sourceDirName:"01-ibc/04-middleware",slug:"/ibc/middleware/develop",permalink:"/v7/ibc/middleware/develop",draft:!1,unlisted:!1,tags:[],version:"v7.8.x",sidebarPosition:1,frontMatter:{title:"IBC middleware",sidebar_label:"IBC middleware",sidebar_position:1,slug:"/ibc/middleware/develop"},sidebar:"defaultSidebar",previous:{title:"Routing",permalink:"/v7/ibc/apps/routing"},next:{title:"Integrating IBC middleware into a chain",permalink:"/v7/ibc/middleware/integration"}},s={},l=[{value:"Pre-requisite readings",id:"pre-requisite-readings",level:2},{value:"Definitions",id:"definitions",level:2},{value:"Create a custom IBC middleware",id:"create-a-custom-ibc-middleware",level:2},{value:"Interfaces",id:"interfaces",level:3},{value:"Implement <code>IBCModule</code> interface and callbacks",id:"implement-ibcmodule-interface-and-callbacks",level:3},{value:"Handshake callbacks",id:"handshake-callbacks",level:3},{value:"<code>OnChanOpenInit</code>",id:"onchanopeninit",level:4},{value:"<code>OnChanOpenTry</code>",id:"onchanopentry",level:4},{value:"<code>OnChanOpenAck</code>",id:"onchanopenack",level:4},{value:"<code>OnChanOpenConfirm</code>",id:"onchanopenconfirm",level:3},{value:"<code>OnChanCloseInit</code>",id:"onchancloseinit",level:4},{value:"<code>OnChanCloseConfirm</code>",id:"onchancloseconfirm",level:4},{value:"Packet callbacks",id:"packet-callbacks",level:3},{value:"<code>OnRecvPacket</code>",id:"onrecvpacket",level:4},{value:"<code>OnAcknowledgementPacket</code>",id:"onacknowledgementpacket",level:4},{value:"<code>OnTimeoutPacket</code>",id:"ontimeoutpacket",level:4},{value:"ICS-4 wrappers",id:"ics-4-wrappers",level:3},{value:"<code>SendPacket</code>",id:"sendpacket",level:4},{value:"<code>WriteAcknowledgement</code>",id:"writeacknowledgement",level:4},{value:"<code>GetAppVersion</code>",id:"getappversion",level:4}];function d(e){const n={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",li:"li",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,t.a)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.h1,{id:"ibc-middleware",children:"IBC middleware"}),"\n",(0,i.jsxs)(n.admonition,{title:"Synopsis",type:"note",children:[(0,i.jsx)(n.p,{children:"Learn how to write your own custom middleware to wrap an IBC application, and understand how to hook different middleware to IBC base applications to form different IBC application stacks\n:::."}),(0,i.jsx)(n.p,{children:"This document serves as a guide for middleware developers who want to write their own middleware and for chain developers who want to use IBC middleware on their chains."}),(0,i.jsx)(n.p,{children:"IBC applications are designed to be self-contained modules that implement their own application-specific logic through a set of interfaces with the core IBC handlers. These core IBC handlers, in turn, are designed to enforce the correctness properties of IBC (transport, authentication, ordering) while delegating all application-specific handling to the IBC application modules. However, there are cases where some functionality may be desired by many applications, yet not appropriate to place in core IBC."}),(0,i.jsx)(n.p,{children:"Middleware allows developers to define the extensions as separate modules that can wrap over the base application. This middleware can thus perform its own custom logic, and pass data into the application so that it may run its logic without being aware of the middleware's existence. This allows both the application and the middleware to implement its own isolated logic while still being able to run as part of a single packet flow."}),(0,i.jsxs)(n.admonition,{type:"note",children:[(0,i.jsx)(n.h2,{id:"pre-requisite-readings",children:"Pre-requisite readings"}),(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"/v7/ibc/overview",children:"IBC Overview"})}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"/v7/ibc/integration",children:"IBC Integration"})}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"/v7/ibc/apps/apps",children:"IBC Application Developer Guide"})}),"\n"]})]})]}),"\n",(0,i.jsx)(n.h2,{id:"definitions",children:"Definitions"}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.code,{children:"Middleware"}),": A self-contained module that sits between core IBC and an underlying IBC application during packet execution. All messages between core IBC and underlying application must flow through middleware, which may perform its own custom logic."]}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.code,{children:"Underlying Application"}),": An underlying application is the application that is directly connected to the middleware in question. This underlying application may itself be middleware that is chained to a base application."]}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.code,{children:"Base Application"}),": A base application is an IBC application that does not contain any middleware. It may be nested by 0 or multiple middleware to form an application stack."]}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.code,{children:"Application Stack (or stack)"}),": A stack is the complete set of application logic (middleware(s) + base application) that gets connected to core IBC. A stack may be just a base application, or it may be a series of middlewares that nest a base application."]}),"\n",(0,i.jsx)(n.h2,{id:"create-a-custom-ibc-middleware",children:"Create a custom IBC middleware"}),"\n",(0,i.jsx)(n.p,{children:"IBC middleware will wrap over an underlying IBC application and sits between core IBC and the application. It has complete control in modifying any message coming from IBC to the application, and any message coming from the application to core IBC. Thus, middleware must be completely trusted by chain developers who wish to integrate them, however this gives them complete flexibility in modifying the application(s) they wrap."}),"\n",(0,i.jsx)(n.admonition,{type:"warning",children:(0,i.jsx)(n.p,{children:"middleware developers must use the same serialization and deserialization method as in ibc-go's codec: transfertypes.ModuleCdc.[Must]MarshalJSON"})}),"\n",(0,i.jsx)(n.p,{children:"For middleware builders this means:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'import transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"\ntransfertypes.ModuleCdc.[Must]MarshalJSON\nfunc MarshalAsIBCDoes(ack channeltypes.Acknowledgement) ([]byte, error) {\n\treturn transfertypes.ModuleCdc.MarshalJSON(&ack)\n}\n'})}),"\n",(0,i.jsx)(n.h3,{id:"interfaces",children:"Interfaces"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"// Middleware implements the ICS26 Module interface\ntype Middleware interface {\n    porttypes.IBCModule // middleware has access to an underlying application which may be wrapped by more middleware\n    ics4Wrapper: ICS4Wrapper // middleware has access to ICS4Wrapper which may be core IBC Channel Handler or a higher-level middleware that wraps this middleware.\n}\n"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-typescript",children:"// This is implemented by ICS4 and all middleware that are wrapping base application.\n// The base application will call `sendPacket` or `writeAcknowledgement` of the middleware directly above them\n// which will call the next middleware until it reaches the core IBC handler.\ntype ICS4Wrapper interface {\n    SendPacket(\n        ctx sdk.Context,\n        chanCap *capabilitytypes.Capability,\n        sourcePort string,\n        sourceChannel string,\n        timeoutHeight clienttypes.Height,\n        timeoutTimestamp uint64,\n        data []byte,\n    ) (sequence uint64, err error)\n\n    WriteAcknowledgement(\n        ctx sdk.Context,\n        chanCap *capabilitytypes.Capability,\n        packet exported.PacketI,\n        ack exported.Acknowledgement,\n    ) error\n\n    GetAppVersion(\n        ctx sdk.Context,\n        portID,\n        channelID string,\n    ) (string, bool)\n}\n"})}),"\n",(0,i.jsxs)(n.h3,{id:"implement-ibcmodule-interface-and-callbacks",children:["Implement ",(0,i.jsx)(n.code,{children:"IBCModule"})," interface and callbacks"]}),"\n",(0,i.jsxs)(n.p,{children:["The ",(0,i.jsx)(n.code,{children:"IBCModule"})," is a struct that implements the ",(0,i.jsxs)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/main/modules/core/05-port/types/module.go#L11-L106",children:["ICS-26 interface (",(0,i.jsx)(n.code,{children:"porttypes.IBCModule"}),")"]}),". It is recommended to separate these callbacks into a separate file ",(0,i.jsx)(n.code,{children:"ibc_module.go"}),". As will be mentioned in the ",(0,i.jsx)(n.a,{href:"/v7/ibc/middleware/integration",children:"integration section"}),", this struct should be different than the struct that implements ",(0,i.jsx)(n.code,{children:"AppModule"})," in case the middleware maintains its own internal state and processes separate SDK messages."]}),"\n",(0,i.jsxs)(n.p,{children:["The middleware must have access to the underlying application, and be called before during all ICS-26 callbacks. It may execute custom logic during these callbacks, and then call the underlying application's callback. Middleware ",(0,i.jsx)(n.strong,{children:"may"})," choose not to call the underlying application's callback at all. Though these should generally be limited to error cases."]}),"\n",(0,i.jsx)(n.p,{children:"In the case where the IBC middleware expects to speak to a compatible IBC middleware on the counterparty chain, they must use the channel handshake to negotiate the middleware version without interfering in the version negotiation of the underlying application."}),"\n",(0,i.jsx)(n.p,{children:"Middleware accomplishes this by formatting the version in a JSON-encoded string containing the middleware version and the application version. The application version may as well be a JSON-encoded string, possibly including further middleware and app versions, if the application stack consists of multiple milddlewares wrapping a base application. The format of the version is specified in ICS-30 as the following:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-json",children:'{\n  "<middleware_version_key>": "<middleware_version_value>",\n  "app_version": "<application_version_value>"\n}\n'})}),"\n",(0,i.jsxs)(n.p,{children:["The ",(0,i.jsx)(n.code,{children:"<middleware_version_key>"})," key in the JSON struct should be replaced by the actual name of the key for the corresponding middleware (e.g. ",(0,i.jsx)(n.code,{children:"fee_version"}),")."]}),"\n",(0,i.jsxs)(n.p,{children:["During the handshake callbacks, the middleware can unmarshal the version string and retrieve the middleware and application versions. It can do its negotiation logic on ",(0,i.jsx)(n.code,{children:"<middleware_version_value>"}),", and pass the ",(0,i.jsx)(n.code,{children:"<application_version_value>"})," to the underlying application."]}),"\n",(0,i.jsx)(n.p,{children:"The middleware should simply pass the capability in the callback arguments along to the underlying application so that it may be claimed by the base application. The base application will then pass the capability up the stack in order to authenticate an outgoing packet/acknowledgement."}),"\n",(0,i.jsxs)(n.p,{children:["In the case where the middleware wishes to send a packet or acknowledgment without the involvement of the underlying application, it should be given access to the same ",(0,i.jsx)(n.code,{children:"scopedKeeper"})," as the base application so that it can retrieve the capabilities by itself."]}),"\n",(0,i.jsx)(n.h3,{id:"handshake-callbacks",children:"Handshake callbacks"}),"\n",(0,i.jsx)(n.h4,{id:"onchanopeninit",children:(0,i.jsx)(n.code,{children:"OnChanOpenInit"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'func (im IBCModule) OnChanOpenInit(\n    ctx sdk.Context,\n    order channeltypes.Order,\n    connectionHops []string,\n    portID string,\n    channelID string,\n    channelCap *capabilitytypes.Capability,\n    counterparty channeltypes.Counterparty,\n    version string,\n) (string, error) {\n    if version != "" {\n        // try to unmarshal JSON-encoded version string and pass\n        // the app-specific version to app callback.\n        // otherwise, pass version directly to app callback.\n        metadata, err := Unmarshal(version)\n        if err != nil {\n            // Since it is valid for fee version to not be specified,\n            // the above middleware version may be for another middleware.\n            // Pass the entire version string onto the underlying application.\n            return im.app.OnChanOpenInit(\n                ctx,\n                order,\n                connectionHops,\n                portID,\n                channelID,\n                channelCap,\n                counterparty,\n                version,\n            )\n        }\n    else {\n        metadata = {\n            // set middleware version to default value\n            MiddlewareVersion: defaultMiddlewareVersion,\n            // allow application to return its default version\n            AppVersion: "",\n        }\n    }\n\n    doCustomLogic()\n\n    // if the version string is empty, OnChanOpenInit is expected to return\n    // a default version string representing the version(s) it supports\n    appVersion, err := im.app.OnChanOpenInit(\n        ctx,\n        order,\n        connectionHops,\n        portID,\n        channelID,\n        channelCap,\n        counterparty,\n        metadata.AppVersion, // note we only pass app version here\n    )\n    if err != nil {\n        return "", err\n    }\n\n    version := constructVersion(metadata.MiddlewareVersion, appVersion)\n\n    return version, nil\n}\n'})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L34-L82",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"onchanopentry",children:(0,i.jsx)(n.code,{children:"OnChanOpenTry"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'func OnChanOpenTry(\n    ctx sdk.Context,\n    order channeltypes.Order,\n    connectionHops []string,\n    portID,\n    channelID string,\n    channelCap *capabilitytypes.Capability,\n    counterparty channeltypes.Counterparty,\n    counterpartyVersion string,\n) (string, error) {\n    // try to unmarshal JSON-encoded version string and pass\n    // the app-specific version to app callback.\n    // otherwise, pass version directly to app callback.\n    cpMetadata, err := Unmarshal(counterpartyVersion)\n    if err != nil {\n        return app.OnChanOpenTry(\n            ctx,\n            order,\n            connectionHops,\n            portID,\n            channelID,\n            channelCap,\n            counterparty,\n            counterpartyVersion,\n        )\n    }\n\n    doCustomLogic()\n\n    // Call the underlying application\'s OnChanOpenTry callback.\n    // The try callback must select the final app-specific version string and return it.\n    appVersion, err := app.OnChanOpenTry(\n        ctx,\n        order,\n        connectionHops,\n        portID,\n        channelID,\n        channelCap,\n        counterparty,\n        cpMetadata.AppVersion, // note we only pass counterparty app version here\n    )\n    if err != nil {\n        return "", err\n    }\n\n    // negotiate final middleware version\n    middlewareVersion := negotiateMiddlewareVersion(cpMetadata.MiddlewareVersion)\n    version := constructVersion(middlewareVersion, appVersion)\n\n    return version, nil\n}\n'})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L84-L124",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"onchanopenack",children:(0,i.jsx)(n.code,{children:"OnChanOpenAck"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnChanOpenAck(\n    ctx sdk.Context,\n    portID,\n    channelID string,\n    counterpartyChannelID string,\n    counterpartyVersion string,\n) error {\n    // try to unmarshal JSON-encoded version string and pass\n    // the app-specific version to app callback.\n    // otherwise, pass version directly to app callback.\n    cpMetadata, err = UnmarshalJSON(counterpartyVersion)\n    if err != nil {\n        return app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)\n    }\n\n    if !isCompatible(cpMetadata.MiddlewareVersion) {\n        return error\n    }\n    doCustomLogic()\n\n    // call the underlying application's OnChanOpenTry callback\n    return app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, cpMetadata.AppVersion)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L126-L152",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h3,{id:"onchanopenconfirm",children:(0,i.jsx)(n.code,{children:"OnChanOpenConfirm"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnChanOpenConfirm(\n    ctx sdk.Context,\n    portID,\n    channelID string,\n) error {\n    doCustomLogic()\n\n    return app.OnChanOpenConfirm(ctx, portID, channelID)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L154-L162",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"onchancloseinit",children:(0,i.jsx)(n.code,{children:"OnChanCloseInit"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnChanCloseInit(\n    ctx sdk.Context,\n    portID,\n    channelID string,\n) error {\n    doCustomLogic()\n\n    return app.OnChanCloseInit(ctx, portID, channelID)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L164-L187",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"onchancloseconfirm",children:(0,i.jsx)(n.code,{children:"OnChanCloseConfirm"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnChanCloseConfirm(\n    ctx sdk.Context,\n    portID,\n    channelID string,\n) error {\n    doCustomLogic()\n\n    return app.OnChanCloseConfirm(ctx, portID, channelID)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L189-L212",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.strong,{children:"NOTE"}),": Middleware that does not need to negotiate with a counterparty middleware on the remote stack will not implement the version unmarshalling and negotiation, and will simply perform its own custom logic on the callbacks without relying on the counterparty behaving similarly."]}),"\n",(0,i.jsx)(n.h3,{id:"packet-callbacks",children:"Packet callbacks"}),"\n",(0,i.jsx)(n.p,{children:"The packet callbacks just like the handshake callbacks wrap the application's packet callbacks. The packet callbacks are where the middleware performs most of its custom logic. The middleware may read the packet flow data and perform some additional packet handling, or it may modify the incoming data before it reaches the underlying application. This enables a wide degree of usecases, as a simple base application like token-transfer can be transformed for a variety of usecases by combining it with custom middleware."}),"\n",(0,i.jsx)(n.h4,{id:"onrecvpacket",children:(0,i.jsx)(n.code,{children:"OnRecvPacket"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnRecvPacket(\n    ctx sdk.Context,\n    packet channeltypes.Packet,\n    relayer sdk.AccAddress,\n) ibcexported.Acknowledgement {\n    doCustomLogic(packet)\n\n    ack := app.OnRecvPacket(ctx, packet, relayer)\n\n    doCustomLogic(ack) // middleware may modify outgoing ack\n    return ack\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L214-L237",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"onacknowledgementpacket",children:(0,i.jsx)(n.code,{children:"OnAcknowledgementPacket"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnAcknowledgementPacket(\n    ctx sdk.Context,\n    packet channeltypes.Packet,\n    acknowledgement []byte,\n    relayer sdk.AccAddress,\n) error {\n    doCustomLogic(packet, ack)\n\n    return app.OnAcknowledgementPacket(ctx, packet, ack, relayer)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L239-L292",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"ontimeoutpacket",children:(0,i.jsx)(n.code,{children:"OnTimeoutPacket"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func OnTimeoutPacket(\n    ctx sdk.Context,\n    packet channeltypes.Packet,\n    relayer sdk.AccAddress,\n) error {\n    doCustomLogic(packet)\n\n    return app.OnTimeoutPacket(ctx, packet, relayer)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L294-L334",children:"here"})," an example implementation of this callback for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h3,{id:"ics-4-wrappers",children:"ICS-4 wrappers"}),"\n",(0,i.jsxs)(n.p,{children:["Middleware must also wrap ICS-4 so that any communication from the application to the ",(0,i.jsx)(n.code,{children:"channelKeeper"})," goes through the middleware first. Similar to the packet callbacks, the middleware may modify outgoing acknowledgements and packets in any way it wishes."]}),"\n",(0,i.jsx)(n.h4,{id:"sendpacket",children:(0,i.jsx)(n.code,{children:"SendPacket"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"func SendPacket(\n    ctx sdk.Context,\n    chanCap *capabilitytypes.Capability,\n    sourcePort string,\n    sourceChannel string,\n    timeoutHeight clienttypes.Height,\n    timeoutTimestamp uint64,\n    appData []byte,\n) {\n    // middleware may modify data\n    data = doCustomLogic(appData)\n\n    return ics4Keeper.SendPacket(\n        ctx,\n        chanCap,\n        sourcePort,\n        sourceChannel,\n        timeoutHeight,\n        timeoutTimestamp,\n        data,\n    )\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L336-L343",children:"here"})," an example implementation of this function for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"writeacknowledgement",children:(0,i.jsx)(n.code,{children:"WriteAcknowledgement"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"// only called for async acks\nfunc WriteAcknowledgement(\n    ctx sdk.Context,\n    chanCap *capabilitytypes.Capability,\n    packet exported.PacketI,\n    ack exported.Acknowledgement,\n) {\n    // middleware may modify acknowledgement\n    ack_bytes = doCustomLogic(ack)\n\n    return ics4Keeper.WriteAcknowledgement(packet, ack_bytes)\n}\n"})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L345-L353",children:"here"})," an example implementation of this function for the ICS29 Fee Middleware module."]}),"\n",(0,i.jsx)(n.h4,{id:"getappversion",children:(0,i.jsx)(n.code,{children:"GetAppVersion"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:'// middleware must return the underlying application version\nfunc GetAppVersion(\n    ctx sdk.Context,\n    portID,\n    channelID string,\n) (string, bool) {\n    version, found := ics4Keeper.GetAppVersion(ctx, portID, channelID)\n    if !found {\n        return "", false\n    }\n\n    if !MiddlewareEnabled {\n        return version, true\n    }\n\n    // unwrap channel version\n    metadata, err := Unmarshal(version)\n    if err != nil {\n        panic(fmt.Errof("unable to unmarshal version: %w", err))\n    }\n\n    return metadata.AppVersion, true\n}\n\n// middleware must return the underlying application version\nfunc GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {\n    version, found := ics4Keeper.GetAppVersion(ctx, portID, channelID)\n    if !found {\n        return "", false\n    }\n\n    if !MiddlewareEnabled {\n        return version, true\n    }\n\n    // unwrap channel version\n    metadata, err := Unmarshal(version)\n    if err != nil {\n        panic(fmt.Errof("unable to unmarshal version: %w", err))\n    }\n\n    return metadata.AppVersion, true\n}\n'})}),"\n",(0,i.jsxs)(n.p,{children:["See ",(0,i.jsx)(n.a,{href:"https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L355-L358",children:"here"})," an example implementation of this function for the ICS29 Fee Middleware module."]})]})}function p(e={}){const{wrapper:n}={...(0,t.a)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(d,{...e})}):d(e)}},11151:(e,n,a)=>{a.d(n,{Z:()=>c,a:()=>o});var i=a(67294);const t={},r=i.createContext(t);function o(e){const n=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function c(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(t):e.components||t:o(e.components),i.createElement(r.Provider,{value:n},e.children)}}}]);