"use strict";(()=>{var M="mermaid-panzoom-style",T=`
.mermaid-panzoom { position: relative; overflow: hidden; }
.mermaid-panzoom--zoomed { cursor: grab; }
.mermaid-panzoom--zoomed:active { cursor: grabbing; }
.mermaid-panzoom__controls {
  position: absolute; top: 0.4rem; right: 0.4rem;
  display: flex; gap: 0.3rem; z-index: 5;
  opacity: 0.55; transition: opacity 0.15s;
}
.mermaid-panzoom:hover .mermaid-panzoom__controls,
.mermaid-panzoom--zoomed .mermaid-panzoom__controls,
.mermaid-panzoom--full .mermaid-panzoom__controls { opacity: 1; }
.mermaid-panzoom__btn {
  width: 2.2rem; height: 2.2rem; padding: 0; margin: 0;
  display: flex; align-items: center; justify-content: center;
  font-size: 1.1rem; line-height: 1; color: inherit; cursor: pointer;
  border: 1px solid rgba(128, 128, 128, 0.4); border-radius: 0.3rem;
  background: rgba(128, 128, 128, 0.15);
}
.mermaid-panzoom__btn:hover { background: rgba(128, 128, 128, 0.3); }
.mermaid-panzoom--full {
  position: fixed; inset: 0; z-index: 1000; margin: 0;
  background: var(--pico-background-color, #fff);
}
`;function D(){if(document.getElementById(M))return;let t=document.createElement("style");t.id=M,t.textContent=T,document.head.appendChild(t)}function A(t){let o=t.querySelector("svg");if(!o)return;let r=t.__panzoom;if(r){r.rebind();return}D(),t.classList.add("mermaid-panzoom");let u=o,i=1,a=0,l=0,c=!1,y=()=>{u.style.transformOrigin="0 0",u.style.transform=`translate(${a}px, ${l}px) scale(${i})`,t.classList.toggle("mermaid-panzoom--zoomed",i>1),t.style.touchAction=c||i>1?"none":"pan-y"},h=()=>{i=1,a=0,l=0,y()},v=(e,n,s)=>{let f=Math.min(10,Math.max(1,i*s)),b=t.getBoundingClientRect(),m=e-b.left,S=n-b.top;a=m-(m-a)*(f/i),l=S-(S-l)*(f/i),i=f,i===1&&(a=0,l=0),y()},z=e=>{let n=t.getBoundingClientRect();v(n.left+n.width/2,n.top+n.height/2,e)},L=e=>{e.key==="Escape"&&E(!1)},E=e=>{c!==e&&(c=e,t.classList.toggle("mermaid-panzoom--full",e),document.documentElement.style.overflow=e?"hidden":"",e?document.addEventListener("keydown",L):document.removeEventListener("keydown",L),h())},x=()=>{let e=document.createElement("div");e.className="mermaid-panzoom__controls";let n=(s,f,b)=>{let m=document.createElement("button");m.type="button",m.className="mermaid-panzoom__btn",m.textContent=s,m.title=f,m.setAttribute("aria-label",f),m.addEventListener("click",b),e.appendChild(m)};n("+","Zoom in",()=>z(1.4)),n("\u2212","Zoom out",()=>z(1/1.4)),n("\u27F2","Reset zoom",h),n("\u26F6","Fullscreen",()=>E(!c)),t.appendChild(e)},d=new Map,p=null,g=0,_=()=>{let[e,n]=Array.from(d.values());return{mid:{x:(e.x+n.x)/2,y:(e.y+n.y)/2},dist:Math.hypot(e.x-n.x,e.y-n.y)}};t.addEventListener("pointerdown",e=>{if(!e.target.closest(".mermaid-panzoom__controls")){if(d.set(e.pointerId,{x:e.clientX,y:e.clientY}),t.setPointerCapture(e.pointerId),d.size===2){let n=_();p=n.mid,g=n.dist}e.pointerType==="mouse"&&(i>1||c)&&e.preventDefault()}}),t.addEventListener("pointermove",e=>{if(!d.has(e.pointerId))return;let n=d.get(e.pointerId);if(d.set(e.pointerId,{x:e.clientX,y:e.clientY}),d.size===2){let s=_();g>0&&v(s.mid.x,s.mid.y,s.dist/g),p&&(a+=s.mid.x-p.x,l+=s.mid.y-p.y,y()),p=s.mid,g=s.dist,e.preventDefault()}else d.size===1&&(i>1||c)&&(a+=e.clientX-n.x,l+=e.clientY-n.y,y(),e.preventDefault())});let C=e=>{d.delete(e.pointerId),p=null,g=0};t.addEventListener("pointerup",C),t.addEventListener("pointercancel",C),t.addEventListener("wheel",e=>{!e.ctrlKey&&!e.metaKey&&!c||(e.preventDefault(),v(e.clientX,e.clientY,Math.exp(-e.deltaY*.01)))},{passive:!1});let B=()=>{let e=t.querySelector("svg");e&&(u=e,E(!1),x(),h())};t.__panzoom={rebind:B},x(),y()}var k="mermaid-label-style",P=`
.mermaid .nodeLabel *,
.mermaid .edgeLabel *,
.mermaid .cluster-label * { color: inherit; }
`;function Y(){if(document.getElementById(k))return;let t=document.createElement("style");t.id=k,t.textContent=P,document.head.appendChild(t)}function X(){let t=Array.from(document.querySelectorAll("pre > code.language-mermaid")),o=[];for(let r of t){let u=r.parentElement;if(!u)continue;let i=document.createElement("div");i.className="mermaid";let a=r.textContent||"";i.textContent=a,u.replaceWith(i),o.push({el:i,src:a})}return o}function H(){return document.documentElement.classList.contains("dark")?"dark":"default"}function w(t){Y();let o=window.mermaid;o.initialize({startOnLoad:!1,theme:H(),securityLevel:"strict"});for(let r of t)r.el.removeAttribute("data-processed"),r.el.textContent=r.src;Promise.resolve(o.run({nodes:t.map(r=>r.el)})).then(()=>{for(let r of t)A(r.el)})}function N(t){let o=window;(o.trip2g_theme_listeners||(o.trip2g_theme_listeners=[])).push(t)}function j(t){if(window.mermaid){t();return}let o=document.getElementById("mermaid-lib");if(o){o.addEventListener("load",t);return}o=document.createElement("script"),o.id="mermaid-lib",o.src="/assets/mermaid.min.js",o.onload=t,document.head.appendChild(o)}function I(){let t=X();t.length!==0&&j(()=>{w(t),N(()=>w(t))})}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",I):I();})();
