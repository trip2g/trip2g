"use strict";(()=>{var M=`
  query TaskListNoteContent($filter: NotePathsFilter) {
    notePaths(filter: $filter) {
      content
    }
  }
`,v=`
  mutation TaskListUpdateNotes($input: UpdateNotesInput!) {
    updateNotes(input: $input) {
      __typename
      ... on UpdateNotesSuccessPayload {
        paths
        updated {
          path
          versionId
        }
      }
      ... on UpdateNotesHashMismatchPayload {
        path
        actualHash
      }
      ... on UpdateNotesPatchNotFoundPayload {
        path
        find
      }
      ... on ErrorPayload {
        message
      }
    }
  }
`;async function k(c,l){let a=await fetch("/graphql",{method:"POST",credentials:"include",headers:{"Content-Type":"application/json"},body:JSON.stringify({query:c,variables:l})});if(!a.ok)throw new Error(`GraphQL HTTP ${a.status}`);let e=await a.json();if(e.errors?.length)throw new Error(e.errors[0].message);return e.data}function I(c){let l=c.split(`
`),a=0,e=!1,f="",i=0;for(let u of l){let s=u.replace(/\r$/,"").replace(/^[ \t]+/,"");if(e){if(s.length>=i){let r=s[0];if(r===f){let n=0;for(;n<s.length&&s[n]===r;)n++;n>=i&&s.slice(n).trim()===""&&(e=!1)}}continue}else if(s.length>=3){let r=s[0];if(r==="`"||r==="~"){let n=0;for(;n<s.length&&s[n]===r;)n++;if(n>=3){e=!0,f=r,i=n;continue}}}let t=s;if(t.length<5)continue;let m=t[0];if(m!=="-"&&m!=="*"&&m!=="+"||t[1]!==" "&&t[1]!=="	")continue;let o=2;for(;o<t.length&&t[o]===" ";)o++;if(!(o+3>t.length)&&t[o]==="["){let r=t[o+1];t[o+2]==="]"&&(r===" "||r==="x"||r==="X")&&a++}}return a}function U(c,l,a){let e=c.split(`
`),f=0,i=!1,u="",p=0;for(let s=0;s<e.length;s++){let t=e[s],o=t.replace(/\r$/,"").replace(/^[ \t]+/,""),r=t.length-t.replace(/^[ \t]+/,"").length;if(i){if(o.length>=p){let h=o[0];if(h===u){let g=0;for(;g<o.length&&o[g]===h;)g++;g>=p&&o.slice(g).trim()===""&&(i=!1)}}continue}else if(o.length>=3){let h=o[0];if(h==="`"||h==="~"){let g=0;for(;g<o.length&&o[g]===h;)g++;if(g>=3){i=!0,u=h,p=g;continue}}}let n=o;if(n.length<5)continue;let w=n[0];if(w!=="-"&&w!=="*"&&w!=="+"||n[1]!==" "&&n[1]!=="	")continue;let d=2;for(;d<n.length&&n[d]===" ";)d++;if(!(d+3>n.length)&&n[d]==="["){let h=n[d+1];if(n[d+2]==="]"&&(h===" "||h==="x"||h==="X")){if(f===a){let N=t.indexOf(o)+d,L=h===" "?"x":" ",_=t.slice(0,N+1)+L+t.slice(N+2),E=[...e];return E[s]=_,E.join(`
`)}f++}}}return null}function S(c,l,a,e,f,i){return l.split(`
`).filter(p=>p.replace(/\r$/,"")===a.replace(/\r$/,"")).length===1?{changes:[{patch:{path:c,find:a,replace:e}}]}:{changes:[{upsert:{path:c,content:i,expectedHash:f}}]}}var y="";async function $(c,l,a,e){let f=c.checked;c.disabled=!0;try{let u=(await k(M,{filter:{paths:[e.path]}})).notePaths?.[0]?.content;if(!u)throw new Error("note content not found");let p=I(u);if(p!==a)throw new Error(`task count mismatch (DOM=${a}, src=${p}) \u2014 page may be stale, please reload`);let s=P(u);if(l>=s.length)throw new Error("index out of range");let t=s[l],m=U(u,a,l);if(!m)throw new Error("could not locate marker in source");let r=P(m)[l],n=S(e.path,u,t,r,y||e.versionId,m),d=(await k(v,{input:n})).updateNotes;if(d.__typename==="UpdateNotesSuccessPayload"){let h=d.updated.find(g=>g.path===e.path);h&&(y=h.versionId),c.checked=!f}else throw d.__typename==="UpdateNotesHashMismatchPayload"?new Error("note was modified by another client \u2014 please reload"):d.__typename==="UpdateNotesPatchNotFoundPayload"?new Error(`patch target not found in source: "${d.find}"`):new Error(d.message||"unknown error")}catch(i){c.checked=f;let u=i instanceof Error?i.message:String(i);console.error("[tasklist]",u),b(c,u)}finally{c.disabled=!1}}function P(c){let l=c.split(`
`),a=[],e=!1,f="",i=0;for(let u of l){let p=u.replace(/\r$/,""),s=p.replace(/^[ \t]+/,"");if(e){if(s.length>=i&&s[0]===f){let r=0;for(;r<s.length&&s[r]===f;)r++;r>=i&&s.slice(r).trim()===""&&(e=!1)}continue}else if(s.length>=3){let r=s[0];if(r==="`"||r==="~"){let n=0;for(;n<s.length&&s[n]===r;)n++;if(n>=3){e=!0,f=r,i=n;continue}}}let t=s;if(t.length<5)continue;let m=t[0];if(m!=="-"&&m!=="*"&&m!=="+"||t[1]!==" "&&t[1]!=="	")continue;let o=2;for(;o<t.length&&t[o]===" ";)o++;if(!(o+3>t.length)&&t[o]==="["){let r=t[o+1];t[o+2]==="]"&&(r===" "||r==="x"||r==="X")&&a.push(p)}}return a}function b(c,l){let a=c.parentElement?.querySelector(".tasklist-error");a&&a.remove();let e=document.createElement("span");e.className="tasklist-error",e.style.cssText="color:#c0392b;font-size:0.85em;margin-left:0.4em;",e.textContent="\u26A0 "+l,c.after(e),setTimeout(()=>e.remove(),6e3)}function T(){let c=document.getElementById("tasklist-meta");if(!c)return;let l;try{l=JSON.parse(c.textContent||"{}")}catch{return}if(!l.path||!l.versionId)return;y=l.versionId;let a=document.querySelector(".content__body");if(!a)return;let e=Array.from(a.querySelectorAll('li > input[type="checkbox"]'));if(e.length===0)return;let f=e.length;e.forEach((i,u)=>{i.removeAttribute("disabled"),i.style.cursor="pointer",i.addEventListener("change",p=>{p.preventDefault(),i.checked=!i.checked,$(i,u,f,l)})})}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",T):T();})();
