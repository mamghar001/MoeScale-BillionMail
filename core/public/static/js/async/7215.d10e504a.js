"use strict";(self.webpackChunkfrontend=self.webpackChunkfrontend||[]).push([["7215"],{66236:function(e,t,a){a.d(t,{W3:()=>d,ZP:()=>v});var r=a(58786),l=a(56946),n=a(54470),i=a(53198),s=a(19050),o=a(55456),c=a(71309);let u=(0,c.cB)("breadcrumb",`
 white-space: nowrap;
 cursor: default;
 line-height: var(--n-item-line-height);
`,[(0,c.c)("ul",`
 list-style: none;
 padding: 0;
 margin: 0;
 `),(0,c.c)("a",`
 color: inherit;
 text-decoration: inherit;
 `),(0,c.cB)("breadcrumb-item",`
 font-size: var(--n-font-size);
 transition: color .3s var(--n-bezier);
 display: inline-flex;
 align-items: center;
 `,[(0,c.cB)("icon",`
 font-size: 18px;
 vertical-align: -.2em;
 transition: color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 `),(0,c.c)("&:not(:last-child)",[(0,c.cM)("clickable",[(0,c.cE)("link",`
 cursor: pointer;
 `,[(0,c.c)("&:hover",`
 background-color: var(--n-item-color-hover);
 `),(0,c.c)("&:active",`
 background-color: var(--n-item-color-pressed); 
 `)])])]),(0,c.cE)("link",`
 padding: 4px;
 border-radius: var(--n-item-border-radius);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 position: relative;
 `,[(0,c.c)("&:hover",`
 color: var(--n-item-text-color-hover);
 `,[(0,c.cB)("icon",`
 color: var(--n-item-text-color-hover);
 `)]),(0,c.c)("&:active",`
 color: var(--n-item-text-color-pressed);
 `,[(0,c.cB)("icon",`
 color: var(--n-item-text-color-pressed);
 `)])]),(0,c.cE)("separator",`
 margin: 0 8px;
 color: var(--n-separator-color);
 transition: color .3s var(--n-bezier);
 user-select: none;
 -webkit-user-select: none;
 `),(0,c.c)("&:last-child",[(0,c.cE)("link",`
 font-weight: var(--n-font-weight-active);
 cursor: unset;
 color: var(--n-item-text-color-active);
 `,[(0,c.cB)("icon",`
 color: var(--n-item-text-color-active);
 `)]),(0,c.cE)("separator",`
 display: none;
 `)])])]),d=(0,s.U)("n-breadcrumb"),m=Object.assign(Object.assign({},l.Z.props),{separator:{type:String,default:"/"}}),v=(0,r.aZ)({name:"Breadcrumb",props:m,setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:a}=(0,n.ZP)(e),s=(0,l.Z)("Breadcrumb","-breadcrumb",u,o.Z,e,t);(0,r.JJ)(d,{separatorRef:(0,r.Vh)(e,"separator"),mergedClsPrefixRef:t});let c=(0,r.Fl)(()=>{let{common:{cubicBezierEaseInOut:e},self:{separatorColor:t,itemTextColor:a,itemTextColorHover:r,itemTextColorPressed:l,itemTextColorActive:n,fontSize:i,fontWeightActive:o,itemBorderRadius:c,itemColorHover:u,itemColorPressed:d,itemLineHeight:m}}=s.value;return{"--n-font-size":i,"--n-bezier":e,"--n-item-text-color":a,"--n-item-text-color-hover":r,"--n-item-text-color-pressed":l,"--n-item-text-color-active":n,"--n-separator-color":t,"--n-item-color-hover":u,"--n-item-color-pressed":d,"--n-item-border-radius":c,"--n-font-weight-active":o,"--n-item-line-height":m}}),m=a?(0,i.F)("breadcrumb",void 0,c,e):void 0;return{mergedClsPrefix:t,cssVars:a?void 0:c,themeClass:null==m?void 0:m.themeClass,onRender:null==m?void 0:m.onRender}},render(){var e;return null==(e=this.onRender)||e.call(this),(0,r.h)("nav",{class:[`${this.mergedClsPrefix}-breadcrumb`,this.themeClass],style:this.cssVars,"aria-label":"Breadcrumb"},(0,r.h)("ul",null,this.$slots))}})},70958:function(e,t,a){a.d(t,{Z:()=>s});var r=a(58786),l=a(93950),n=a(68574),i=a(66236);let s=(0,r.aZ)({name:"BreadcrumbItem",props:{separator:String,href:String,clickable:{type:Boolean,default:!0},onClick:Function},slots:Object,setup(e,{slots:t}){let a=(0,r.f3)(i.W3,null);if(!a)return()=>null;let{separatorRef:s,mergedClsPrefixRef:o}=a,c=function(e=n.j?window:null){let t=()=>{let{hash:t,host:a,hostname:r,href:l,origin:n,pathname:i,port:s,protocol:o,search:c}=(null==e?void 0:e.location)||{};return{hash:t,host:a,hostname:r,href:l,origin:n,pathname:i,port:s,protocol:o,search:c}},a=(0,r.iH)(t()),l=()=>{a.value=t()};return(0,r.bv)(()=>{e&&(e.addEventListener("popstate",l),e.addEventListener("hashchange",l))}),(0,r.SK)(()=>{e&&(e.removeEventListener("popstate",l),e.removeEventListener("hashchange",l))}),a}(),u=(0,r.Fl)(()=>e.href?"a":"span"),d=(0,r.Fl)(()=>c.value.href===e.href?"location":null);return()=>{let{value:a}=o;return(0,r.h)("li",{class:[`${a}-breadcrumb-item`,e.clickable&&`${a}-breadcrumb-item--clickable`]},(0,r.h)(u.value,{class:`${a}-breadcrumb-item__link`,"aria-current":d.value,href:e.href,onClick:e.onClick},t),(0,r.h)("span",{class:`${a}-breadcrumb-item__separator`,"aria-hidden":"true"},(0,l.gI)(t.separator,()=>{var t;return[null!=(t=e.separator)?t:s.value]})))}}})},55855:function(e,t,a){a.d(t,{Ed:()=>g,Gk:()=>s,MB:()=>_,NE:()=>x,Q5:()=>v,Xr:()=>m,ZS:()=>i,_5:()=>d,eA:()=>k,gI:()=>c,jC:()=>p,ke:()=>h,mQ:()=>o,o$:()=>y,oS:()=>b,pA:()=>f,xJ:()=>u,zl:()=>w});var r=a(76882),l=a(34541);let{t:n}=r.ag.global;function i(e){return l.e.get("/batch_mail/task/list",{params:e})}function s(e){return l.e.get("/batch_mail/task/stat_chart",{params:e})}function o(e){return l.e.get("/batch_mail/task/find",{params:e})}function c(e){return l.e.post("/batch_mail/task/create",e,{fetchOptions:{loading:n("market.task.loading.creating"),successMessage:!0}})}function u(e){return l.e.post("/batch_mail/task/update",e,{fetchOptions:{loading:n("market.task.loading.updating"),successMessage:!0}})}function d(e){return l.e.post("/batch_mail/task/delete",e,{fetchOptions:{loading:n("market.task.loading.deleting"),successMessage:!0}})}function m(e){return l.e.post("/batch_mail/task/pause",e,{fetchOptions:{loading:n("market.task.loading.pausing"),successMessage:!0}})}function v(e){return l.e.post("/batch_mail/task/resume",e,{fetchOptions:{loading:n("market.task.loading.resuming"),successMessage:!0}})}function p(e){return l.e.post("/batch_mail/task/send_test",e,{fetchOptions:{loading:n("market.task.loading.sendingTest"),successMessage:!0}})}function h(e){return l.e.get("/batch_mail/tracking/mail_provider",{params:e})}function b(e){return l.e.get("/batch_mail/tracking/logs",{params:e})}function f(e){return l.e.get("/batch_mail/task/sample_preview",{params:e})}function g(){return l.e.get("/batch_mail/workspace/list")}function _(e){return l.e.post("/batch_mail/workspace/create",e,{fetchOptions:{successMessage:!0}})}function k(e){return l.e.post("/batch_mail/workspace/update",e,{fetchOptions:{successMessage:!0}})}function w(e){return l.e.post("/batch_mail/workspace/delete",e,{fetchOptions:{successMessage:!0}})}function y(e){return l.e.post("/batch_mail/task/batch_move",e,{fetchOptions:{loading:"Moving campaigns...",successMessage:!0}})}function x(e){return l.e.post("/batch_mail/task/batch_delete",e,{fetchOptions:{loading:"Deleting campaigns...",successMessage:!0}})}},9393:function(e,t,a){a.r(t),a.d(t,{default:()=>B});var r=a(57869),l=a(17977),n=a(16715),i=a(77227),s=a(58786),o=a(49468),c=a(36760),u=a(17987),d=a(51628);let m={class:"bt-time-range"},v=(0,s.aZ)({__name:"index",props:(0,s.Vf)({defaultType:{type:String,default:"today"}},{value:{},valueModifiers:{}}),emits:(0,s.Vf)(["change"],["update:value"]),setup(e,t){let{emit:a}=t,r=(0,s.tT)(e,"value"),v=(0,s.iH)(e.defaultType),p=()=>{let e=new Date,t=(0,o.i)(e);return[(0,c.b)((0,u.E)(e,-6)).getTime(),t.getTime()]},h=e=>{switch(e){case"today":r.value=(0,d.wb)();break;case"yesterday":r.value=(0,d.wb)((0,u.E)(new Date,-1));break;case"last7days":r.value=p()}a("change")},b=e=>{let t=new Date,a=(0,u.E)(t,-30);return(0,c.b)(a).getTime()>e||(0,o.i)(t).getTime()<e},f=e=>{v.value="custom",r.value=[(0,c.b)(e[0]).getTime(),(0,o.i)(e[1]).getTime()],a("change")};return h(e.defaultType),(e,t)=>{let a=i.Z,o=n.Z,c=l.Z;return(0,s.wg)(),(0,s.iD)("div",m,[(0,s.Wm)(o,{value:(0,s.SU)(v),"onUpdate:value":[t[0]||(t[0]=e=>(0,s.dq)(v)?v.value=e:null),h]},{default:(0,s.w5)(()=>[(0,s.Wm)(a,{value:"today"},{default:(0,s.w5)(()=>[(0,s.Uk)((0,s.zw)(e.$t("common.time.today")),1)]),_:1}),(0,s.Wm)(a,{value:"yesterday"},{default:(0,s.w5)(()=>[(0,s.Uk)((0,s.zw)(e.$t("common.time.yesterday")),1)]),_:1}),(0,s.Wm)(a,{value:"last7days"},{default:(0,s.w5)(()=>[(0,s.Uk)((0,s.zw)(e.$t("common.time.last7days")),1)]),_:1})]),_:1},8,["value"]),(0,s.Wm)(c,{value:r.value,type:"daterange","is-date-disabled":b,"onUpdate:value":f},null,8,["value"])])}}});var p=a(89544);let h=(0,p.default)(v,[["__scopeId","data-v-65f017db"]]);var b=a(66236),f=a(35409),g=a(70958),_=a(55855),k=a(35617),w=a(85046),y=a(15244),x=a(77337),Z=a(65965),U=a(52495);let W={class:"p-20px"},z={class:"flex justify-between items-center mb-20px"},M={class:"font-bold text-basic"},S={class:"metrics-cards"},E={class:"detail-row"},O={class:"rate-charts-card"},$=(0,s.aZ)({__name:"analytics",setup(e){let t=(0,Z.yj)(),{t:a}=(0,U.QT)(),l=(0,s.Fl)(()=>(0,d.Dx)(t.params.id||"0")),n=(0,s.iH)((0,d.wb)()),i=(0,s.iH)([]),o=(0,s.qj)({delivery_rate:{label:a("overview.delivered"),value:0,unit:"%"},open_rate:{label:a("overview.opened"),value:0,unit:"%"},click_rate:{label:a("overview.clicked"),value:0,unit:"%"},bounce_rate:{label:a("overview.bounced"),value:0,unit:"%"}}),c=(0,s.iH)({column_type:"hourly",dashboard:{delivered:0,delivery_rate:0,failed:0,failure_rate:0,sends:0},data:[]}),u=(0,s.iH)({column_type:"hourly",data:[]}),m=(0,s.iH)({column_type:"hourly",data:[]}),v=(0,s.iH)({column_type:"hourly",data:[]}),p=e=>{Object.entries(e).forEach(e=>{let[t,a]=e;t in o&&(o[t].value=a)})};async function $(){let e=await (0,_.Gk)({task_id:l.value,start_time:Math.floor(n.value[0]/1e3),end_time:Math.floor(n.value[1]/1e3)});(0,d.Kn)(e)&&(p(e.dashboard),i.value=(0,d.kJ)(e.mail_providers)?e.mail_providers:[],c.value=e.send_mail_chart,u.value=e.bounce_rate_chart,m.value=e.click_rate_chart,v.value=e.open_rate_chart)}let B=(0,s.iH)("");return(async()=>{let e=await (0,_.mQ)({id:l.value});(0,d.Kn)(e)&&(B.value=e.subject)})(),(e,t)=>{let a=(0,s.up)("router-link"),l=g.Z,d=f.ZP,p=b.ZP,_=r.ZP;return(0,s.wg)(),(0,s.iD)("div",W,[(0,s._)("div",z,[(0,s.Wm)(p,null,{default:(0,s.w5)(()=>[(0,s.Wm)(l,null,{default:(0,s.w5)(()=>[(0,s.Wm)(a,{to:"/market/task"},{default:(0,s.w5)(()=>[(0,s.Uk)((0,s.zw)(e.$t("market.task.title")),1)]),_:1})]),_:1}),(0,s.Wm)(l,null,{default:(0,s.w5)(()=>[(0,s.Uk)((0,s.zw)(e.$t("market.task.actions.analytics")),1)]),_:1}),(0,s.Wm)(l,null,{default:(0,s.w5)(()=>[(0,s.Wm)(d,{style:{"max-width":"300px"}},{default:(0,s.w5)(()=>[(0,s._)("span",M,(0,s.zw)((0,s.SU)(B)||"--"),1)]),_:1})]),_:1})]),_:1}),(0,s.Wm)(h,{value:(0,s.SU)(n),"onUpdate:value":t[0]||(t[0]=e=>(0,s.dq)(n)?n.value=e:null),"default-type":"last7days",onChange:$},null,8,["value"])]),(0,s._)("div",S,[((0,s.wg)(!0),(0,s.iD)(s.HY,null,(0,s.Ko)((0,s.SU)(o),(e,t)=>((0,s.wg)(),(0,s.j4)(k.Z,{key:t,title:e.label,value:e.value,unit:e.unit},null,8,["title","value","unit"]))),128))]),(0,s._)("div",E,[(0,s.Wm)(_,{class:"provider-table-card",title:e.$t("overview.mailProviders")},{default:(0,s.w5)(()=>[(0,s.Wm)(w.Z,{value:(0,s.SU)(i),"onUpdate:value":t[1]||(t[1]=e=>(0,s.dq)(i)?i.value=e:null)},null,8,["value"])]),_:1},8,["title"]),(0,s.Wm)(_,{class:"send-today-card",title:e.$t("overview.sendStats")},{default:(0,s.w5)(()=>[(0,s.Wm)(y.Z,{data:(0,s.SU)(c)},null,8,["data"])]),_:1},8,["title"])]),(0,s._)("div",O,[(0,s.Wm)(x.Z,{bounce:(0,s.SU)(u),click:(0,s.SU)(m),open:(0,s.SU)(v)},null,8,["bounce","click","open"])])])}}}),B=(0,p.default)($,[["__scopeId","data-v-11ae91fa"]])}}]);