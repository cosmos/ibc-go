"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[72162],{1110:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>a,default:()=>p,frontMatter:()=>o,metadata:()=>i,toc:()=>d});var s=n(85893),r=n(11151);const o={title:"State",sidebar_label:"State",sidebar_position:2,slug:"/apps/transfer/ics20-v1/state"},a="State",i={id:"apps/transfer/state",title:"State",description:"The IBC transfer application module keeps state of the port to which the module is binded and the denomination trace information.",source:"@site/versioned_docs/version-v10.1.x/02-apps/01-transfer/02-state.md",sourceDirName:"02-apps/01-transfer",slug:"/apps/transfer/ics20-v1/state",permalink:"/v10/apps/transfer/ics20-v1/state",draft:!1,unlisted:!1,tags:[],version:"v10.1.x",sidebarPosition:2,frontMatter:{title:"State",sidebar_label:"State",sidebar_position:2,slug:"/apps/transfer/ics20-v1/state"},sidebar:"defaultSidebar",previous:{title:"Overview",permalink:"/v10/apps/transfer/ics20-v1/overview"},next:{title:"State Transitions",permalink:"/v10/apps/transfer/ics20-v1/state-transitions"}},c={},d=[];function l(e){const t={code:"code",h1:"h1",li:"li",p:"p",ul:"ul",...(0,r.a)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.h1,{id:"state",children:"State"}),"\n",(0,s.jsx)(t.p,{children:"The IBC transfer application module keeps state of the port to which the module is binded and the denomination trace information."}),"\n",(0,s.jsxs)(t.ul,{children:["\n",(0,s.jsxs)(t.li,{children:[(0,s.jsx)(t.code,{children:"PortKey"}),": ",(0,s.jsx)(t.code,{children:"0x01 -> ProtocolBuffer(string)"})]}),"\n",(0,s.jsxs)(t.li,{children:[(0,s.jsx)(t.code,{children:"DenomTraceKey"}),": ",(0,s.jsx)(t.code,{children:"0x02 | []bytes(traceHash) -> ProtocolBuffer(Denom)"})]}),"\n",(0,s.jsxs)(t.li,{children:[(0,s.jsx)(t.code,{children:"DenomKey"})," : ",(0,s.jsx)(t.code,{children:"0x03 | []bytes(traceHash) -> ProtocolBuffer(Denom)"})]}),"\n"]})]})}function p(e={}){const{wrapper:t}={...(0,r.a)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(l,{...e})}):l(e)}},11151:(e,t,n)=>{n.d(t,{Z:()=>i,a:()=>a});var s=n(67294);const r={},o=s.createContext(r);function a(e){const t=s.useContext(o);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function i(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:a(e.components),s.createElement(o.Provider,{value:t},e.children)}}}]);