import{_ as p,o as i,c as d,a as s,w as l,g as n,b as u,r as m,p as h,d as f,e as _,u as b}from "./index-57557fd5.js";const A= e=>(h("data-v-1cf5f789"),e=e(),f(),e),k={class:"transactions"},v=A(()=>_("div",{class:"header"},[_("h1",null,"Transactions")],-1)),y={__name:"TransactionsForm",setup(e){const t=b();async function c(a){const r={recipient_blockchain_address:a.recipientBlockchainAddress,amount:a.amount,sender_public_key:t.getters[n.PUBLIC_KEY],sender_private_key:t.getters[n.PRIVATE_KEY],sender_blockchain_address:t.getters[n.BLOCKCHAIN_ADDRESS]};await t.dispatch(u.TRANSACTION_CREATE,r)}return(a, r)=>{const o=m("FormKit");return i(),d("div",k,[v,s(o,{type:"form",onSubmit:c},{default:l(()=>[s(o,{type:"text",name:"recipientBlockchainAddress",label:"Recipient Blockchain Address",help:"Recipient blockchain address",validation:"required"}),s(o,{type:"text",name:"amount",label:"Amount",help:"Amount",validation:"required|number"})]),_:1})])}}},T=p(y,[["__scopeId","data-v-1cf5f789"]]),I={class:"about"},B={__name:"WalletInformation",setup(e){return(t, c)=>(i(),d("div",I,[s(T)]))}};export{B as default};
