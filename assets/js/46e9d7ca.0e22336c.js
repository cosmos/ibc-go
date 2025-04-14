"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[69206],{86185:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>a,default:()=>u,frontMatter:()=>r,metadata:()=>s,toc:()=>l});var i=t(85893),o=t(11151);const r={title:"Routing",sidebar_label:"Routing",sidebar_position:6,slug:"/ibc/apps/routing"},a="Routing",s={id:"ibc/apps/routing",title:"Routing",description:"Pre-requisite readings",source:"@site/docs/01-ibc/03-apps/06-routing.md",sourceDirName:"01-ibc/03-apps",slug:"/ibc/apps/routing",permalink:"/main/ibc/apps/routing",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:6,frontMatter:{title:"Routing",sidebar_label:"Routing",sidebar_position:6,slug:"/ibc/apps/routing"},sidebar:"defaultSidebar",previous:{title:"Define packets and acks",permalink:"/main/ibc/apps/packets_acks"},next:{title:"IBC middleware",permalink:"/main/ibc/middleware/overview"}},c={},l=[{value:"Pre-requisite readings",id:"pre-requisite-readings",level:2}];function d(e){const n={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",li:"li",p:"p",pre:"pre",ul:"ul",...(0,o.a)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.h1,{id:"routing",children:"Routing"}),"\n",(0,i.jsxs)(n.admonition,{type:"note",children:[(0,i.jsx)(n.h2,{id:"pre-requisite-readings",children:"Pre-requisite readings"}),(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"/main/ibc/overview",children:"IBC Overview"})}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"/main/ibc/integration",children:"IBC default integration"})}),"\n"]})]}),"\n",(0,i.jsx)(n.admonition,{title:"Synopsis",type:"note",children:(0,i.jsx)(n.p,{children:"Learn how to hook a route to the IBC router for the custom IBC module."})}),"\n",(0,i.jsxs)(n.p,{children:["As mentioned above, modules must implement the ",(0,i.jsx)(n.code,{children:"IBCModule"})," interface (which contains both channel\nhandshake callbacks for IBC classic only, and packet handling callbacks for IBC classic and v2). The concrete implementation of this interface\nmust be registered with the module name as a route on the IBC ",(0,i.jsx)(n.code,{children:"Router"}),"."]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-go",children:"// app.go\nfunc NewApp(...args) *App {\n  // ...\n\n  // Create static IBC router, add module routes, then set and seal it\n  ibcRouter := port.NewRouter()\n\n  ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)\n  // Note: moduleCallbacks must implement IBCModule interface\n  ibcRouter.AddRoute(moduleName, moduleCallbacks)\n\n  // Setting Router will finalize all routes by sealing router\n  // No more routes can be added\n  app.IBCKeeper.SetRouter(ibcRouter)\n\n  // ...\n}\n"})})]})}function u(e={}){const{wrapper:n}={...(0,o.a)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(d,{...e})}):d(e)}},11151:(e,n,t)=>{t.d(n,{Z:()=>s,a:()=>a});var i=t(67294);const o={},r=i.createContext(o);function a(e){const n=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function s(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:a(e.components),i.createElement(r.Provider,{value:n},e.children)}}}]);