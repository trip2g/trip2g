"use strict";(()=>{var S="mermaid-panzoom-style",B=`
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
`;function T(){if(document.getElementById(S))return;let t=document.createElement("style");t.id=S,t.textContent=B,document.head.appendChild(t)}function k(t){let o=t.querySelector("svg");if(!o)return;let i=t.__panzoom;if(i){i.rebind();return}T(),t.classList.add("mermaid-panzoom");let u=o,r=1,a=0,l=0,c=!1,y=()=>{u.style.transformOrigin="0 0",u.style.transform=`translate(${a}px, ${l}px) scale(${r})`,t.classList.toggle("mermaid-panzoom--zoomed",r>1),t.style.touchAction=c||r>1?"none":"pan-y"},v=()=>{r=1,a=0,l=0,y()},b=(e,n,s)=>{let f=Math.min(10,Math.max(1,r*s)),h=t.getBoundingClientRect(),m=e-h.left,M=n-h.top;a=m-(m-a)*(f/r),l=M-(M-l)*(f/r),r=f,r===1&&(a=0,l=0),y()},E=e=>{let n=t.getBoundingClientRect();b(n.left+n.width/2,n.top+n.height/2,e)},x=e=>{e.key==="Escape"&&z(!1)},z=e=>{c!==e&&(c=e,t.classList.toggle("mermaid-panzoom--full",e),document.documentElement.style.overflow=e?"hidden":"",e?document.addEventListener("keydown",x):document.removeEventListener("keydown",x),v())},_=()=>{let e=document.createElement("div");e.className="mermaid-panzoom__controls";let n=(s,f,h)=>{let m=document.createElement("button");m.type="button",m.className="mermaid-panzoom__btn",m.textContent=s,m.title=f,m.setAttribute("aria-label",f),m.addEventListener("click",h),e.appendChild(m)};n("+","Zoom in",()=>E(1.4)),n("\u2212","Zoom out",()=>E(1/1.4)),n("\u27F2","Reset zoom",v),n("\u26F6","Fullscreen",()=>z(!c)),t.appendChild(e)},d=new Map,p=null,g=0,L=()=>{let[e,n]=Array.from(d.values());return{mid:{x:(e.x+n.x)/2,y:(e.y+n.y)/2},dist:Math.hypot(e.x-n.x,e.y-n.y)}};t.addEventListener("pointerdown",e=>{if(!e.target.closest(".mermaid-panzoom__controls")){if(d.set(e.pointerId,{x:e.clientX,y:e.clientY}),t.setPointerCapture(e.pointerId),d.size===2){let n=L();p=n.mid,g=n.dist}e.pointerType==="mouse"&&(r>1||c)&&e.preventDefault()}}),t.addEventListener("pointermove",e=>{if(!d.has(e.pointerId))return;let n=d.get(e.pointerId);if(d.set(e.pointerId,{x:e.clientX,y:e.clientY}),d.size===2){let s=L();g>0&&b(s.mid.x,s.mid.y,s.dist/g),p&&(a+=s.mid.x-p.x,l+=s.mid.y-p.y,y()),p=s.mid,g=s.dist,e.preventDefault()}else d.size===1&&(r>1||c)&&(a+=e.clientX-n.x,l+=e.clientY-n.y,y(),e.preventDefault())});let C=e=>{d.delete(e.pointerId),p=null,g=0};t.addEventListener("pointerup",C),t.addEventListener("pointercancel",C),t.addEventListener("wheel",e=>{!e.ctrlKey&&!e.metaKey&&!c||(e.preventDefault(),b(e.clientX,e.clientY,Math.exp(-e.deltaY*.01)))},{passive:!1});let I=()=>{let e=t.querySelector("svg");e&&(u=e,z(!1),_(),v())};t.__panzoom={rebind:I},_(),y()}function D(){let t=Array.from(document.querySelectorAll("pre > code.language-mermaid")),o=[];for(let i of t){let u=i.parentElement;if(!u)continue;let r=document.createElement("div");r.className="mermaid";let a=i.textContent||"";r.textContent=a,u.replaceWith(r),o.push({el:r,src:a})}return o}function P(){return document.documentElement.classList.contains("dark")?"dark":"default"}function w(t){let o=window.mermaid;o.initialize({startOnLoad:!1,theme:P(),securityLevel:"strict"});for(let i of t)i.el.removeAttribute("data-processed"),i.el.textContent=i.src;Promise.resolve(o.run({nodes:t.map(i=>i.el)})).then(()=>{for(let i of t)k(i.el)})}function X(t){let o=window;(o.trip2g_theme_listeners||(o.trip2g_theme_listeners=[])).push(t)}function Y(t){if(window.mermaid){t();return}let o=document.getElementById("mermaid-lib");if(o){o.addEventListener("load",t);return}o=document.createElement("script"),o.id="mermaid-lib",o.src="/assets/mermaid.min.js",o.onload=t,document.head.appendChild(o)}function A(){let t=D();t.length!==0&&Y(()=>{w(t),X(()=>w(t))})}document.readyState==="loading"?document.addEventListener("DOMContentLoaded",A):A();})();
