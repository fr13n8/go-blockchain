import{o as _,a as u,c as i,b as c,d as l,e as t,t as n,u as a,F as p,g as h,h as m,j as T,f}from"./index-9499f8a5.js";const b=t("div",{style:{"text-align":"center"}},[t("h1",null,"Transaction")],-1),g={class:"table table-bordered"},y=t("thead",null,[t("tr",null,[t("th",{scope:"col"},"Field"),t("th",{scope:"col"},"Value")])],-1),N=t("th",{scope:"row"},"Sender address",-1),A=t("th",{scope:"row"},"Recipient address",-1),S=t("th",{scope:"row"},"Amount",-1),w={__name:"TransactionData",setup(d){const e=h(),o=m();_(async()=>{const r=o.params.id;await e.dispatch(u.GET_TRANSACTION,r)});const s=i(()=>e.getters[T.GET_TRANSACTION]);return(r,x)=>(c(),l(p,null,[b,t("div",null,[t("table",g,[y,t("tbody",null,[t("tr",null,[N,t("td",null,n(a(s).sender_address),1)]),t("tr",null,[A,t("td",null,n(a(s).recipient_address),1)]),t("tr",null,[S,t("td",null,n(a(s).amount),1)])])])])],64))}},B={__name:"TransactionInformation",setup(d){return(e,o)=>(c(),l("main",null,[f(w)]))}};export{B as default};
