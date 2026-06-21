#!/usr/bin/env node
import*as U from"fs";import*as V from"path";import*as S from"fs";import*as A from"path";import*as F from"crypto";function B(a){return!!(a.startsWith("_layouts/")&&(a.endsWith(".html")||a.endsWith(".html.json")))}import*as C from"path";function R(a,t,e){if(t.startsWith("./")){let s=C.dirname(e),i=C.join(s,t.slice(2));return a.fileExistsSync(i)?i:null}if(t.startsWith("/")){let s=t.slice(1);return a.fileExistsSync(s)?s:null}if(t.includes("/"))return a.fileExistsSync(t)?t:null;if(a.fileExistsSync(t))return t;let r=C.posix.join("assets",t);if(a.fileExistsSync(r))return r;let o=C.dirname(e);if(o&&o!=="."){let s=C.posix.join(o,t);if(a.fileExistsSync(s))return s}return null}var O=class{constructor(t,e={}){this.url=t;this.options=e}async request(t){let e=typeof t.document=="string"?t.document:t.document.loc?.source.body;if(!e)throw new Error("Invalid GraphQL document: no query string found");let r=await fetch(this.url,{method:"POST",headers:{"Content-Type":"application/json",...this.options.headers,...t.requestHeaders},body:JSON.stringify({query:e,variables:t.variables}),signal:t.signal});if(!r.ok){let s=await r.text().catch(()=>"");throw new Error(`HTTP ${r.status}: ${r.statusText}${s?`
${s}`:""}`)}let o=await r.json();if(o.errors?.length){let s=o.errors.map(n=>n.message).join("; "),i=new Error(`GraphQL Error: ${s}`);throw i.graphqlErrors=o.errors,i.response=o,i}if(!o.data)throw new Error("GraphQL response missing data");return o.data}};function b(a,...t){let e=a[0];for(let r=0;r<t.length;r++)e+=String(t[r])+a[r+1];return{loc:{source:{body:e}}}}var K=b`
    query FetchServerHashes {
  notePaths {
    path: value
    hash: latestContentHash
  }
}
    `,_=b`
    query FetchPublishedUrls {
  notePaths {
    path: value
    latestNoteView {
      url
    }
  }
}
    `,j=b`
    query FetchAllWarnings {
  notePaths {
    path: value
    latestNoteView {
      url
      warnings {
        level
        message
      }
    }
  }
}
    `,J=b`
    query FetchNoteContents($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    path: value
    content
  }
}
    `,Y=b`
    query FetchNoteAssets($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    path: value
    assetReplaces {
      id
      url
      hash
      absolutePath
    }
  }
}
    `,z=b`
    mutation PushNotes($input: PushNotesInput!) {
  pushNotes(input: $input) {
    ... on ErrorPayload {
      message
    }
    ... on PushNotesPayload {
      notes {
        id
        path
        assets {
          path
          sha256Hash
          absolutePath
          url
        }
      }
      updated {
        path
        url
      }
    }
  }
}
    `,X=b`
    mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input) {
    ... on HideNotesPayload {
      success
    }
    ... on ErrorPayload {
      message
    }
  }
}
    `,Z=b`
    mutation UploadNoteAsset($input: UploadNoteAssetInput!) {
  uploadNoteAsset(input: $input) {
    ... on ErrorPayload {
      __typename
      message
    }
    ... on UploadNoteAssetPayload {
      __typename
      uploadSkipped
    }
  }
}
    `,tt=b`
    mutation CommitNotes {
  commitNotes {
    ... on CommitNotesPayload {
      success
      updated {
        path
        url
        warnings {
          level
          message
        }
      }
    }
    ... on ErrorPayload {
      message
    }
  }
}
    `,et=(a,t,e,r)=>a();function w(a,t=et){return{FetchServerHashes(e,r,o){return t(s=>a.request({document:K,variables:e,requestHeaders:{...r,...s},signal:o}),"FetchServerHashes","query",e)},FetchPublishedUrls(e,r,o){return t(s=>a.request({document:_,variables:e,requestHeaders:{...r,...s},signal:o}),"FetchPublishedUrls","query",e)},FetchAllWarnings(e,r,o){return t(s=>a.request({document:j,variables:e,requestHeaders:{...r,...s},signal:o}),"FetchAllWarnings","query",e)},FetchNoteContents(e,r,o){return t(s=>a.request({document:J,variables:e,requestHeaders:{...r,...s},signal:o}),"FetchNoteContents","query",e)},FetchNoteAssets(e,r,o){return t(s=>a.request({document:Y,variables:e,requestHeaders:{...r,...s},signal:o}),"FetchNoteAssets","query",e)},PushNotes(e,r,o){return t(s=>a.request({document:z,variables:e,requestHeaders:{...r,...s},signal:o}),"PushNotes","mutation",e)},HideNotes(e,r,o){return t(s=>a.request({document:X,variables:e,requestHeaders:{...r,...s},signal:o}),"HideNotes","mutation",e)},UploadNoteAsset(e,r,o){return t(s=>a.request({document:Z,variables:e,requestHeaders:{...r,...s},signal:o}),"UploadNoteAsset","mutation",e)},CommitNotes(e,r,o){return t(s=>a.request({document:tt,variables:e,requestHeaders:{...r,...s},signal:o}),"CommitNotes","mutation",e)}}}function N(a){let t=new O(a.apiUrl,{headers:{"X-API-Key":a.apiKey}});return w(t)}var v=".sync-state.json",k=class{constructor(t){this.pushBatchSize=100;this.folder=A.resolve(t.folder),this.prefix=t.prefix?t.prefix.replace(/\/$/,""):"",this.twoWaySync=t.twoWaySync,this.verbose=t.verbose??!1,this.conflictResolution=t.conflictResolution??"local",this.publishField=t.publishField??"",this.meta=t.meta??{},this.syncState=this.loadSyncState(),this.apiUrl=t.apiUrl,this.apiKey=t.apiKey,this.sdk=N({apiUrl:t.apiUrl,apiKey:t.apiKey})}toRemotePath(t){return this.prefix?`${this.prefix}/${t}`:t}toLocalPath(t){return this.prefix&&t.startsWith(this.prefix+"/")?t.substring(this.prefix.length+1):t}matchesPrefix(t){return this.prefix?t.startsWith(this.prefix+"/"):!0}loadSyncState(){let t=A.join(this.folder,v);try{if(S.existsSync(t)){let e=S.readFileSync(t,"utf-8");return JSON.parse(e)}}catch(e){this.log(`Warning: Could not load sync state: ${e}`)}return{files:{}}}log(t){this.verbose&&console.log(t)}async getLocalFiles(){let t=[],e=r=>{let o=S.readdirSync(r,{withFileTypes:!0});for(let s of o){if(s.name.startsWith(".")||s.name==="node_modules")continue;let i=A.join(r,s.name);if(s.isDirectory())e(i);else if(s.isFile()){let n=A.extname(s.name).toLowerCase();if(n===".md"||n===".html"||n===".canvas"||n===".base"||n===".excalidraw"||s.name.endsWith(".html.json")){let u=S.statSync(i),l=A.relative(this.folder,i);t.push({path:this.toRemotePath(l),mtime:u.mtimeMs})}}}};return e(this.folder),t}async getServerHashes(){try{return(await this.sdk.FetchServerHashes()).notePaths.filter(e=>this.matchesPrefix(e.path)).map(e=>({path:e.path,hash:e.hash}))}catch(t){return console.error(`\u274C Failed to fetch server hashes: ${t}`),[]}}getSyncState(){return this.syncState}async computeHash(t){return F.createHash("sha256").update(t,"utf-8").digest().toString("base64").replace(/\+/g,"-").replace(/\//g,"_")}async readFileContent(t){let e=this.toLocalPath(t),r=A.join(this.folder,e);return S.readFileSync(r,"utf-8")}async writeFile(t,e){let r=A.join(this.folder,t);S.writeFileSync(r,e,"utf-8")}async writeBinaryFile(t,e){let r=A.join(this.folder,t);S.writeFileSync(r,Buffer.from(e))}async readBinaryFile(t){let e=A.join(this.folder,t),r=S.readFileSync(e);return r.buffer.slice(r.byteOffset,r.byteOffset+r.byteLength)}async deleteFile(t){let e=A.join(this.folder,t);S.existsSync(e)&&S.unlinkSync(e)}async createFolder(t){let e=A.join(this.folder,t);S.mkdirSync(e,{recursive:!0})}async fileExists(t){return this.fileExistsSync(t)}fileExistsSync(t){let e=A.join(this.folder,t);return S.existsSync(e)}async pushNotes(t,e){if(t.length===0)return[];let r=t.map(o=>({path:o.path,content:this.injectMeta(o.content)}));if(this.publishField){for(let o of r)if(!this.hasPublishFieldInContent(o.content,o.path))throw new Error(`[Security] Attempted to push note "${o.path}" without publish field "${this.publishField}". This is a bug in the sync logic - please report it.`)}try{let o=await this.sdk.PushNotes({input:{updates:r.map(i=>({path:i.path,content:i.content})),skipCommit:e}});if("message"in o.pushNotes)throw new Error(`Push failed: ${o.pushNotes.message}`);console.log(`\u2705 Pushed ${t.length} notes`);let s=new Map((o.pushNotes.updated??[]).map(i=>[i.path,i.url??null]));return o.pushNotes.notes.map(i=>({id:String(i.id),path:i.path,assets:i.assets.map(n=>({path:n.path,sha256Hash:n.sha256Hash??null,absolutePath:n.absolutePath??null,url:n.url??null})),url:s.get(i.path)??null}))}catch(o){let s=r.map(n=>n.path).join(", ");console.error(`\u274C Failed to push notes (batch paths: ${s}):`),console.error(o);let i=o;return i.response&&console.error("   response:",JSON.stringify(i.response,null,2)),i.request&&console.error("   request:",JSON.stringify(i.request,null,2)),console.error("   own props:",Object.getOwnPropertyNames(o)),[]}}async hideNotes(t){if(t.length!==0)try{let e=await this.sdk.HideNotes({input:{paths:t}});if("message"in e.hideNotes)throw new Error(`Hide failed: ${e.hideNotes.message}`);console.log(`\u2705 Hidden ${t.length} notes`)}catch(e){console.error(`\u274C Failed to hide notes: ${e}`)}}async fetchNoteContents(t){if(t.length===0)return[];try{return(await this.sdk.FetchNoteContents({filter:{paths:t}})).notePaths.map(r=>({path:r.path,content:r.content}))}catch(e){return console.error(`\u274C Failed to fetch note contents: ${e}`),[]}}async fetchNoteAssets(t){if(t.length===0)return[];try{let e=await this.sdk.PushNotes({input:{updates:[]}});if("message"in e.pushNotes)return console.error(`\u274C Failed to fetch note assets: ${e.pushNotes.message}`),[];let r=new Set(t);return e.pushNotes.notes.filter(o=>r.has(o.path)).map(o=>({path:o.path,noteId:String(o.id),assets:o.assets.map(s=>({id:s.path,url:s.url,hash:s.sha256Hash??"",absolutePath:s.absolutePath}))}))}catch(e){return console.error(`\u274C Failed to fetch note assets: ${e}`),[]}}async uploadAsset(t){for(let r=1;r<=10;r++)try{if(await this.uploadAssetOnce(t))return!0}catch(o){if(r<10){this.log(`\u26A0\uFE0F Upload attempt ${r} failed, retrying: ${t.relativePath}`);continue}return console.error(`\u274C Failed to upload asset ${t.relativePath} after 10 attempts: ${o}`),!1}return!1}async uploadAssetOnce(t){let r=JSON.stringify({query:`mutation UploadNoteAsset($input: UploadNoteAssetInput!) {
	uploadNoteAsset(input: $input) {
		... on ErrorPayload {
			__typename
			message
		}
		... on UploadNoteAssetPayload {
			__typename
			uploadSkipped
		}
	}
}`,variables:{input:{file:null,noteId:parseInt(t.noteId),sha256Hash:t.sha256Hash,path:t.relativePath,absolutePath:t.absolutePath}}}),o=JSON.stringify({0:["variables.input.file"]}),s=new FormData;s.append("operations",r),s.append("map",o),s.append("0",t.blob,t.fileName);let i=await fetch(this.apiUrl,{method:"POST",headers:{"X-API-Key":this.apiKey},body:s});if(!i.ok){let l=await i.text();throw new Error(`HTTP ${i.status}: ${i.statusText}
${l}`)}let n=await i.json();if(n.errors)throw new Error(n.errors[0]?.message||"Unknown GraphQL error");let u=n.data?.uploadNoteAsset;if(u?.__typename==="ErrorPayload")throw new Error(`Upload failed: ${u.message}`);return u?.uploadSkipped?this.log(`\u23E9 Asset skipped (already exists): ${t.relativePath}`):console.log(`\u2705 Asset uploaded: ${t.relativePath}`),!0}async downloadAsset(t){try{let e=await fetch(t);return e.ok?await e.arrayBuffer():(console.error(`\u274C Failed to download asset: HTTP ${e.status}`),null)}catch(e){return console.error(`\u274C Failed to download asset from ${t}: ${e}`),null}}async commitNotes(){try{let t=await this.sdk.CommitNotes();if("message"in t.commitNotes)throw new Error(`Commit failed: ${t.commitNotes.message}`);return console.log("\u2705 Notes committed"),{updated:(t.commitNotes.updated??[]).map(e=>({path:e.path,url:e.url??"",warnings:(e.warnings??[]).map(r=>({level:r.level,message:r.message}))}))}}catch(t){return console.error(`\u274C Failed to commit notes: ${t}`),{updated:[]}}}async saveSyncState(t){let e=A.join(this.folder,v);t.lastSyncedAt=Date.now(),S.writeFileSync(e,JSON.stringify(t,null,2),"utf-8"),this.syncState=t}async computeBinaryHash(t){return F.createHash("sha256").update(Buffer.from(t)).digest("hex")}async resolveAssetPath(t,e){return R(this,t,e)}onProgress(t){this.verbose&&console.log(`  [${t.step}] ${t.current}/${t.total}: ${t.path??""}`)}async onConflict(t){if(this.conflictResolution==="fail"){console.error(`\u274C ${t.length} conflicts detected:`);for(let r of t)console.error(`   - ${r.path}`);throw new Error("Conflicts detected and --conflict-resolution=fail is set")}let e=this.cliToConflictResolution(this.conflictResolution);return console.log(`\u26A0\uFE0F ${t.length} conflicts detected, resolving with: ${this.conflictResolution}`),t.map(()=>e)}async onAssetConflict(t){if(this.conflictResolution==="fail"){console.error(`\u274C ${t.length} asset conflicts detected:`);for(let r of t)console.error(`   - ${r.path}`);throw new Error("Asset conflicts detected and --conflict-resolution=fail is set")}let e=this.cliToAssetConflictResolution(this.conflictResolution);return console.log(`\u26A0\uFE0F ${t.length} asset conflicts detected, resolving with: ${this.conflictResolution}`),t.map(()=>e)}cliToConflictResolution(t){switch(t){case"local":return"keep_local";case"remote":return"keep_remote";case"skip":return"skip";default:return"keep_local"}}cliToAssetConflictResolution(t){switch(t){case"local":return"keep_local";case"remote":return"keep_remote";case"skip":return"skip";default:return"keep_local"}}async onServerDeleted(t){return console.log(`\u26A0\uFE0F ${t.length} files deleted on server, keeping local copies`),!1}async confirmPush(t){return console.log(`\u{1F4E4} Pushing ${t.length} files...`),!0}injectMeta(t){if(Object.keys(this.meta).length===0)return t;if(t.startsWith("---")){let r=t.indexOf(`
---`,3);if(r!==-1){let o=t.slice(4,r),s=t.slice(r+4);for(let[i,n]of Object.entries(this.meta)){let u=new RegExp(`^${i}\\s*:.*$`,"m");u.test(o)?o=o.replace(u,`${i}: ${n}`):o=o.trimEnd()+`
${i}: ${n}`}return`---
${o}
---${s}`}}return`---
${Object.entries(this.meta).map(([r,o])=>`${r}: ${o}`).join(`
`)}
---
${t}`}hasPublishFieldInContent(t,e){if(!this.publishField||B(e))return!0;if(!t.startsWith("---"))return!1;let r=t.indexOf(`
---`,3);if(r===-1)return!1;let o=t.slice(4,r),s=this.publishField.split(",").map(i=>i.trim()).filter(i=>i);for(let i of s){let n=new RegExp(`^${i}\\s*:\\s*(.+)$`,"m"),u=o.match(n);if(u){let l=u[1].trim().toLowerCase();if(l==="true"||l==="yes"||l==="1"||l==='"true"'||l==="'true'")return!0}}return!1}};function at(a,t,e){return a===null&&t===null||a===t?"unchanged":a!==null&&t===null?e?"server_deleted":"local_only":a===null&&t!==null?e?"local_deleted":"remote_only":e?a===e?"pull":t===e?"push":"conflict":"conflict"}async function D(a){let t=a.getSyncState(),[e,r]=await Promise.all([a.getLocalFiles(),a.getServerHashes()]),o=new Map;for(let h of r)o.set(h.path,h.hash);let s=new Map,i=t.mtimes||{},n=t.localHashes||{};for(let h of e){let T=i[h.path],x=n[h.path];if(T===h.mtime&&x)s.set(h.path,x);else{let E=await a.readFileContent(h.path),M=await a.computeHash(E);s.set(h.path,M)}}let u=new Set([...s.keys(),...o.keys()]),l=[],p=[],c=[],d=[],m=[],y=[],P=[],g=[],I=0;for(let h of u){let T=s.get(h)||null,x=o.get(h)||null,E=t.files[h]||null,M=at(T,x,E),f={path:h,action:M,localHash:T,remoteHash:x,lastSyncedHash:E};switch(l.push(f),M){case"unchanged":I++;break;case"pull":p.push(f);break;case"push":c.push(f);break;case"conflict":d.push(f);break;case"local_only":m.push(f);break;case"remote_only":y.push(f);break;case"local_deleted":P.push(f);break;case"server_deleted":g.push(f);break}}return{classifications:l,pulls:p,pushes:c,conflicts:d,localOnly:m,remoteOnly:y,localDeleted:P,serverDeleted:g,unchanged:I}}function W(a,t){let{twoWaySync:e,hasPublishFields:r,isExcluded:o}=t,s=g=>r?r(g):!0,i=g=>o?o(g):!1,n=[],u=[],l=[],p=[],c=[],d=[],m=[],y=[],P=0;for(let g of a.classifications){if(i(g.path)){if(g.remoteHash!==null){let h=g.action==="local_deleted"?g:{...g,action:"local_deleted"};n.push(h),m.push(h)}continue}let I=s(g.path);switch(g.action){case"unchanged":n.push(g),P++;break;case"pull":e&&I&&(n.push(g),u.push(g));break;case"push":I&&(n.push(g),l.push(g));break;case"conflict":if(e)I&&(n.push(g),p.push(g));else if(I){let h={...g,action:"push"};n.push(h),l.push(h)}break;case"local_only":I&&(n.push(g),c.push(g));break;case"remote_only":e&&(n.push(g),d.push(g));break;case"local_deleted":I&&(n.push(g),m.push(g));break;case"server_deleted":e&&(n.push(g),y.push(g));break}}return{classifications:n,pulls:u,pushes:l,conflicts:p,localOnly:c,remoteOnly:d,localDeleted:m,serverDeleted:y,unchanged:P}}async function $(a,t,e={twoWaySync:!1}){let r={pulled:0,pushed:0,conflictsResolved:0,assetsUploaded:0,assetsDownloaded:0,errors:[],updatedUrls:[],warnings:[]},o=a.getSyncState(),s=[];if(t.pulls.length>0||t.remoteOnly.length>0){let l=[...t.pulls,...t.remoteOnly],p=await rt(a,l,o);r.pulled=p.count,r.errors.push(...p.errors),s.push(...p.pulledPaths)}if(s.length>0){let l=await H(a,s);r.assetsDownloaded+=l.downloaded,r.errors.push(...l.errors)}if(e.twoWaySync){let l=t.classifications.filter(p=>p.action==="unchanged"&&p.remoteHash!==null).map(p=>p.path);if(l.length>0){let p=await H(a,l);r.assetsDownloaded+=p.downloaded,r.errors.push(...p.errors)}}if(t.serverDeleted.length>0&&await it(a,t.serverDeleted,o),t.conflicts.length>0){let l=await nt(a,t.conflicts,o);r.conflictsResolved=l.resolved,r.errors.push(...l.errors)}let i=[...t.pushes,...t.localOnly],n=[];if(i.length>0&&await a.confirmPush(i.map(p=>p.path))){let p=await ot(a,i,o);r.pushed=p.count,r.errors.push(...p.errors),n=p.pushedNotes}if(t.localDeleted.length>0&&await lt(a,t.localDeleted,o),n.length>0){let l=await ut(a,n,e.twoWaySync);r.assetsUploaded=l.uploaded,r.assetsDownloaded=l.downloaded,r.errors.push(...l.errors)}let u=t.classifications.filter(l=>l.action==="unchanged"&&l.remoteHash!==null).map(l=>l.path);if(u.length>0){let l=await ct(a,u);r.assetsUploaded+=l.uploaded,r.errors.push(...l.errors)}if(r.pushed>0||r.assetsUploaded>0){let l=await a.commitNotes();r.updatedUrls=l.updated.map(({path:p,url:c})=>({path:p,url:c}));for(let p of l.updated)for(let c of p.warnings)r.warnings.push({path:p.path,level:c.level,message:c.message})}return await a.saveSyncState(o),r}async function rt(a,t,e){if(t.length===0)return{count:0,errors:[],pulledPaths:[]};let r=t.map(c=>c.path),o=[],s=[],i=0,n=await a.fetchNoteContents(r),u=new Map(n.map(c=>[c.path,c.content])),l=t.length,p=0;for(let c of t){p++,a.onProgress({step:"pull",current:p,total:l,path:c.path});let d=u.get(c.path);if(d===void 0){o.push(`Failed to fetch: ${c.path}`);continue}try{let m=c.path.substring(0,c.path.lastIndexOf("/"));m&&await a.createFolder(m),await a.writeFile(c.path,d);let y=await a.computeHash(d);e.files[c.path]=y,i++,s.push(c.path)}catch(m){o.push(`Failed to write ${c.path}: ${m}`)}}return{count:i,errors:o,pulledPaths:s}}async function ot(a,t,e){if(t.length===0)return{count:0,errors:[],pushedNotes:[],urls:[]};let r=[],o=[],s=t.length,i=0;for(let y of t){i++,a.onProgress({step:"push",current:i,total:s,path:y.path});try{let P=await a.readFileContent(y.path);o.push({path:y.path,content:P})}catch(P){r.push(`Failed to read ${y.path}: ${P}`)}}if(o.length===0)return{count:0,errors:r,pushedNotes:[],urls:[]};let n=new Set(o.map(y=>y.path)),u=a.pushBatchSize||100,l=[];for(let y=0;y<o.length;y+=u){let P=o.slice(y,y+u),g=await a.pushNotes(P,!0);l.push(...g)}let p=new Set(l.map(y=>y.path)),c=0;for(let y of o)if(p.has(y.path)){let P=await a.computeHash(y.content);e.files[y.path]=P,c++}let d=l.filter(y=>n.has(y.path)),m=d.filter(y=>typeof y.url=="string").map(y=>({path:y.path,url:y.url}));return{count:c,errors:r,pushedNotes:d,urls:m}}async function nt(a,t,e){if(t.length===0)return{resolved:0,errors:[]};let r=[],o=t.map(p=>p.path),s=await a.fetchNoteContents(o),i=new Map(s.map(p=>[p.path,p.content])),n=[];for(let p of t){let c=i.get(p.path);if(c!==void 0)try{let d=await a.readFileContent(p.path);n.push({path:p.path,localContent:d,remoteContent:c,localHash:p.localHash,remoteHash:p.remoteHash})}catch(d){console.warn(`Failed to read local file for conflict ${p.path}:`,d),r.push(`Failed to read local file for conflict: ${p.path}`)}}if(n.length===0)return{resolved:0,errors:r};let u=await a.onConflict(n),l=0;for(let p=0;p<n.length;p++){let c=n[p],d=u[p]||"skip";try{await st(a,c,d,e),d!=="skip"&&l++}catch(m){r.push(`Failed to resolve conflict for ${c.path}: ${m}`)}}return{resolved:l,errors:r}}async function st(a,t,e,r){switch(e){case"keep_local":await a.pushNotes([{path:t.path,content:t.localContent}],!0),r.files[t.path]=t.localHash;break;case"keep_remote":await a.writeFile(t.path,t.remoteContent),r.files[t.path]=t.remoteHash;break;case"keep_both":{let o=t.path.substring(t.path.lastIndexOf(".")),i=`${t.path.substring(0,t.path.lastIndexOf("."))} (server)${o}`;await a.writeFile(i,t.remoteContent),r.files[t.path]=t.localHash;let n=await a.computeHash(t.remoteContent);r.files[i]=n;break}case"skip":break}}async function it(a,t,e){if(t.length===0)return;let r=t.map(s=>s.path);if(await a.onServerDeleted(r))for(let s of t)try{await a.deleteFile(s.path),delete e.files[s.path]}catch(i){console.warn(`Failed to delete file ${s.path}:`,i)}else for(let s of t)s.localHash&&(e.files[s.path]=s.localHash)}async function lt(a,t,e){if(t.length===0)return;let r=t.map(o=>o.path);await a.hideNotes(r);for(let o of r)delete e.files[o]}async function ut(a,t,e){console.log(`[Trip2g Sync] syncAssets called with ${t.length} notes, twoWaySync=${e}`);let r={uploaded:0,downloaded:0,conflictsResolved:0,errors:[]};if(t.length===0)return r;let o=[],s=[],i=[];for(let n of t)if(console.log(`[Trip2g Sync] Processing assets for note: ${n.path}, assets count: ${n.assets?.length??0}`),!(!n.assets||n.assets.length===0))for(let u of n.assets){let l=await a.resolveAssetPath(u.path,n.path);if(console.log(`[Trip2g Sync] Asset "${u.path}" -> localPath: ${l??"NOT FOUND"}, sha256Hash: ${u.sha256Hash??"null"}`),!l)continue;if(!u.sha256Hash||!u.absolutePath||!u.url){console.log(`[Trip2g Sync] Queuing upload: ${u.path} (no hash on server)`),o.push({noteId:n.id,notePath:n.path,asset:u,localPath:l});continue}if(await a.fileExists(l))try{let c=await a.readBinaryFile(l),d=await a.computeBinaryHash(c);if(d===u.sha256Hash)continue;i.push({path:u.path,absolutePath:l,noteId:n.id,localHash:d,remoteHash:u.sha256Hash,remoteUrl:u.url})}catch(c){r.errors.push(`Failed to read local asset ${l}: ${c}`)}else e&&s.push({asset:u,localPath:l})}if(console.log(`[Trip2g Sync] Assets to upload: ${o.length}, to download: ${s.length}, conflicts: ${i.length}`),o.length>0){let n=new Map;for(let c of o){let d=`${c.noteId}:${c.localPath}`;n.has(d)||n.set(d,c)}let u=Array.from(n.values()),l=u.length,p=0;console.log(`[Trip2g Sync] Uploading ${l} unique (note, asset) pairs`);for(let c of u){p++,console.log(`[Trip2g Sync] Uploading asset ${p}/${l}: ${c.localPath}`),a.onProgress({step:"upload_asset",current:p,total:l,path:c.asset.path});try{let d=await a.readBinaryFile(c.localPath),m=await a.computeBinaryHash(d),y=new Blob([d]),P=c.localPath.substring(c.localPath.lastIndexOf("/")+1);await a.uploadAsset({noteId:c.noteId,blob:y,fileName:P,relativePath:c.asset.path,absolutePath:c.localPath,sha256Hash:m})&&r.uploaded++}catch(d){r.errors.push(`Failed to upload asset ${c.asset.path}: ${d}`)}}}if(s.length>0){let n=s.length,u=0;for(let l of s)if(u++,a.onProgress({step:"download_asset",current:u,total:n,path:l.asset.path}),!!l.asset.url)try{let p=await a.downloadAsset(l.asset.url);if(!p){r.errors.push(`Failed to download asset ${l.asset.path}`);continue}let c=l.localPath.substring(0,l.localPath.lastIndexOf("/"));c&&await a.createFolder(c),await a.writeBinaryFile(l.localPath,p),r.downloaded++}catch(p){r.errors.push(`Failed to download asset ${l.asset.path}: ${p}`)}}if(i.length>0){let n=await pt(a,i,e);r.uploaded+=n.uploaded,r.downloaded+=n.downloaded,r.conflictsResolved=n.conflictsResolved,r.errors.push(...n.errors)}return r}async function pt(a,t,e){let r={uploaded:0,downloaded:0,conflictsResolved:0,errors:[]};if(t.length===0)return r;let o;e?o=await a.onAssetConflict(t):o=t.map(()=>"keep_local");for(let s=0;s<t.length;s++){let i=t[s],n=o[s]||"skip";try{if(n==="keep_local"){let u=await a.readBinaryFile(i.absolutePath),l=new Blob([u]),p=i.absolutePath.substring(i.absolutePath.lastIndexOf("/")+1);await a.uploadAsset({noteId:i.noteId,blob:l,fileName:p,relativePath:i.path,absolutePath:i.absolutePath,sha256Hash:i.localHash})&&(r.uploaded++,r.conflictsResolved++)}else if(n==="keep_remote"){let u=await a.downloadAsset(i.remoteUrl);u?(await a.writeBinaryFile(i.absolutePath,u),r.downloaded++,r.conflictsResolved++):r.errors.push(`Failed to download asset ${i.path}`)}}catch(u){r.errors.push(`Failed to resolve asset conflict for ${i.path}: ${u}`)}}return r}async function H(a,t){let e={downloaded:0,errors:[]};if(t.length===0)return e;let r=await a.fetchNoteAssets(t);if(r.length===0)return e;let o=new Map;for(let n of r)for(let u of n.assets){let l=u.absolutePath.replace(/^\//,"");o.has(l)||await a.fileExists(l)||o.set(l,{url:u.url,hash:u.hash})}if(o.size===0)return e;let s=o.size,i=0;for(let[n,{url:u}]of o){i++,a.onProgress({step:"download_asset",current:i,total:s,path:n});try{let l=await a.downloadAsset(u);if(!l){e.errors.push(`Failed to download asset ${n}`);continue}let p=n.substring(0,n.lastIndexOf("/"));p&&await a.createFolder(p),await a.writeBinaryFile(n,l),e.downloaded++}catch(l){e.errors.push(`Failed to download asset ${n}: ${l}`)}}return e}async function ct(a,t){let e={uploaded:0,errors:[]};if(t.length===0)return e;let r=await a.fetchNoteAssets(t);if(r.length===0)return e;let o=[];for(let n of r)for(let u of n.assets){let l=u.absolutePath?.replace(/^\//,"");if(!l&&u.id){let c=n.path.includes("/")?n.path.substring(0,n.path.lastIndexOf("/")):"",d=u.id.replace(/^\.\//,"");l=c?`${c}/${d}`:d}if(!(!l||!await a.fileExists(l)))try{let c=await a.readBinaryFile(l),d=await a.computeBinaryHash(c);if(d===u.hash)continue;o.push({noteId:n.noteId,notePath:n.path,assetPath:u.id,localPath:l,localHash:d})}catch(c){e.errors.push(`Failed to read local asset ${l}: ${c}`)}}if(o.length===0)return e;let s=o.length,i=0;for(let n of o){i++,a.onProgress({step:"upload_asset",current:i,total:s,path:n.assetPath});try{let u=await a.readBinaryFile(n.localPath),l=new Blob([u]),p=n.localPath.substring(n.localPath.lastIndexOf("/")+1);await a.uploadAsset({noteId:n.noteId,blob:l,fileName:p,relativePath:n.assetPath,absolutePath:n.localPath,sha256Hash:n.localHash})&&e.uploaded++}catch(u){e.errors.push(`Failed to upload asset ${n.assetPath}: ${u}`)}}return e}function G(a){let t=a.map(r=>r.trim().replace(/\/+$/,"")).filter(r=>r.length>0);if(t.length===0)return()=>!1;let e=t.map(dt);return r=>e.some(o=>o.test(r))}function dt(a){return/[*?]/.test(a)?new RegExp("^"+yt(a)+"$"):new RegExp("^"+L(a)+"(?:/.*)?$")}function yt(a){let t="";for(let e=0;e<a.length;e++){let r=a[e];r==="*"?a[e+1]==="*"?(t+=".*",e++):t+="[^/]*":r==="?"?t+="[^/]":t+=L(r)}return t}function L(a){return a.replace(/[.*+?^${}()|[\]\\]/g,"\\$&")}function Q(){try{let a=V.join(process.cwd(),".obsidian","plugins","trip2g","data.json"),e=JSON.parse(U.readFileSync(a,"utf8"))?.syncDirs?.[0];return e?{apiUrl:e.apiUrl?`${e.apiUrl}/_system/graphql`:void 0,apiKey:e.apiKey||void 0}:{}}catch{return{}}}function gt(){let a=process.argv.slice(2),t=Q(),e={folder:"",prefix:"",apiUrl:process.env.TRIP2G_ENDPOINT||process.env.ENDPOINT||t.apiUrl||"http://localhost:8081/_system/graphql",apiKey:process.env.TRIP2G_API_KEY||process.env.API_KEY||t.apiKey||"",twoWaySync:!1,verbose:!1,dryRun:!1,conflictResolution:"local",meta:{},updatedOutput:"",exclude:[]},r=[];for(let o=0;o<a.length;o++){let s=a[o],i;if(s.includes("=")&&s.startsWith("-")){let n=s.indexOf("=");i=s.substring(n+1),s=s.substring(0,n)}switch(s){case"--api-url":case"-u":e.apiUrl=i??a[++o];break;case"--api-key":case"-k":e.apiKey=i??a[++o];break;case"--two-way":case"-2":e.twoWaySync=!0;break;case"--verbose":case"-v":e.verbose=!0;break;case"--dry-run":case"-n":e.dryRun=!0;break;case"--conflict-resolution":case"-c":{let n=i??a[++o];n==="local"||n==="remote"||n==="skip"||n==="fail"?e.conflictResolution=n:(console.error(`\u274C Invalid conflict resolution: ${n}. Use: local, remote, skip, fail`),process.exit(1));break}case"--meta":case"-m":{let n=i??a[++o];if(n&&n.includes("=")){let u=n.indexOf("="),l=n.substring(0,u),p=n.substring(u+1);e.meta[l]=p}else console.error(`\u274C Invalid --meta format: ${n}. Use: --meta key=value`),process.exit(1);break}case"--updated-output":case"-o":e.updatedOutput=i??a[++o];break;case"--exclude":case"-x":{let n=i??a[++o];n&&e.exclude.push(n);break}case"--help":case"-h":q(),process.exit(0);break;default:s.startsWith("-")||r.push(s)}}return r.length>=1&&(e.folder=r[0]),r.length>=2&&(e.prefix=r[1]),e}function q(){console.log(`
obsidian-sync CLI

Usage:
  npx ts-node src/sync/cli/cmd.ts [options] <folder> [prefix]

Arguments:
  folder                   Local folder to sync (required)
  prefix                   Remote path prefix (optional, for multi-repo setups)

Options:
  -u, --api-url <url>      GraphQL endpoint (default: $ENDPOINT or .obsidian/plugins/trip2g/data.json or http://localhost:8081/_system/graphql)
  -k, --api-key <key>      API key (default: $API_KEY)
  -2, --two-way            Enable two-way sync (pull changes from server)
  -c, --conflict-resolution <mode>
                           How to resolve conflicts (default: local)
                           - local:  Keep local version, push to server
                           - remote: Keep remote version, overwrite local
                           - skip:   Skip conflicting files
                           - fail:   Exit with error on first conflict
  -m, --meta <key=value>   Add/override frontmatter field for all files (can be repeated)
  -o, --updated-output <file>
                           Write pushed notes as JSON [{path, url}] to file after sync
  -x, --exclude <glob>     Exclude paths from sync (can be repeated). Excluded
                           paths are never pushed; if they exist on the server
                           they are hidden. A bare name like "dev" matches that
                           directory and everything under it. Default: none.
  -v, --verbose            Verbose output
  -n, --dry-run            Show what would be done without making changes
  -h, --help               Show this help

Environment Variables:
  TRIP2G_ENDPOINT    GraphQL endpoint URL
  TRIP2G_API_KEY     API key for authentication
  ENDPOINT           Fallback for TRIP2G_ENDPOINT
  API_KEY            Fallback for TRIP2G_API_KEY

Examples:
  # Push-only sync
  trip2g-sync ./vault --api-key xxx

  # Two-way sync
  trip2g-sync ./vault --api-key xxx --two-way

  # Exclude folders from a publish (they get hidden on the server if present)
  trip2g-sync ./docs --exclude dev --exclude demo

  # Multi-repo setup: each repo pushes to different folder with different meta
  trip2g-sync ./docs docs --meta subgraph=docs
  trip2g-sync ./blog blog --meta subgraph=blog
  trip2g-sync ./wiki wiki --meta subgraph=team-wiki
`)}async function ht(){let a=Q(),t=process.env.TRIP2G_ENDPOINT||process.env.ENDPOINT||a.apiUrl||"http://localhost:8081/_system/graphql",e=process.env.TRIP2G_API_KEY||process.env.API_KEY||a.apiKey||"";e||(console.error("\u274C TRIP2G_API_KEY or API_KEY required"),process.exit(1));let o=await N({apiUrl:t,apiKey:e}).FetchAllWarnings(),s=[];for(let i of o.notePaths){let n=i.latestNoteView;if(n)for(let u of n.warnings??[])s.push({path:i.path,level:u.level,message:u.message,url:n.url??""})}console.log(JSON.stringify(s,null,2))}async function St(){if(process.argv[2]==="warnings"){await ht();return}let a=gt();a.folder||(console.error("\u274C Error: --folder is required"),q(),process.exit(1)),a.apiKey||(console.error("\u274C Error: --api-key or API_KEY environment variable is required"),process.exit(1)),a.prefix&&a.twoWaySync&&(console.error("\u274C Error: prefix is not supported with --two-way sync"),process.exit(1)),a.dryRun&&console.log(`[dry-run] folder=${a.folder}${a.prefix?` prefix=${a.prefix}`:""}`);let t=new k({folder:a.folder,prefix:a.prefix,apiUrl:a.apiUrl,apiKey:a.apiKey,twoWaySync:a.twoWaySync,verbose:a.verbose,conflictResolution:a.conflictResolution,meta:a.meta});console.log(`
\u{1F4CA} Classifying files...`);let e=await D(t),r=G(a.exclude),o=W(e,{twoWaySync:a.twoWaySync,isExcluded:r});if(a.exclude.length>0&&console.log(`\u{1F6AB} Excluding: ${a.exclude.join(", ")}`),console.log(`
\u{1F4CB} Sync Plan:`),console.log("-".repeat(40)),console.log(`  Unchanged:      ${o.unchanged}`),console.log(`  To push:        ${o.pushes.length}`),console.log(`  Local only:     ${o.localOnly.length}`),console.log(`  To pull:        ${o.pulls.length}`),console.log(`  Remote only:    ${o.remoteOnly.length}`),console.log(`  Conflicts:      ${o.conflicts.length}`),console.log(`  Local deleted:  ${o.localDeleted.length}`),console.log(`  Server deleted: ${o.serverDeleted.length}`),console.log("-".repeat(40)),a.verbose){if(o.pushes.length>0){console.log(`
\u{1F4E4} Files to push:`);for(let u of o.pushes)console.log(`  ${u.path}`)}if(o.localOnly.length>0){console.log(`
\u{1F195} New local files:`);for(let u of o.localOnly)console.log(`  ${u.path}`)}if(o.pulls.length>0){console.log(`
\u{1F4E5} Files to pull:`);for(let u of o.pulls)console.log(`  ${u.path}`)}if(o.remoteOnly.length>0){console.log(`
\u{1F310} New remote files:`);for(let u of o.remoteOnly)console.log(`  ${u.path}`)}if(o.localDeleted.length>0){console.log(`
\u{1F5D1}\uFE0F To hide on server:`);for(let u of o.localDeleted)console.log(`  ${u.path}`)}}if(a.dryRun){console.log(`
\u23F8\uFE0F Dry run - no changes made`);return}let s=o.pushes.length+o.localOnly.length+o.pulls.length+o.remoteOnly.length+o.conflicts.length+o.localDeleted.length+o.serverDeleted.length;console.log(`
\u{1F680} Executing sync...`);let i=await $(t,o,{twoWaySync:a.twoWaySync});if(s===0&&i.assetsUploaded===0&&i.assetsDownloaded===0){console.log(`
\u2705 Everything is up to date!`);return}if(console.log(`
`+"=".repeat(60)),console.log("\u{1F4CA} SYNC RESULTS:"),console.log("=".repeat(60)),console.log(`  Pushed:             ${i.pushed}`),console.log(`  Pulled:             ${i.pulled}`),console.log(`  Conflicts resolved: ${i.conflictsResolved}`),console.log(`  Assets uploaded:    ${i.assetsUploaded}`),console.log(`  Assets downloaded:  ${i.assetsDownloaded}`),i.errors.length>0){console.log(`  Errors:             ${i.errors.length}`);for(let u of i.errors)console.log(`    \u274C ${u}`)}if(i.warnings.length>0){console.log(`  Warnings:           ${i.warnings.length}`);for(let u of i.warnings)console.log(`    \u26A0\uFE0F  [${u.level}] ${u.path}: ${u.message}`)}console.log("=".repeat(60));let n=i.updatedUrls??[];if(n.length>0){if(console.log(`
\u{1F4CE} Published:`),n.length<=20)for(let{path:u,url:l}of n)console.log(`  ${u} \u2192 ${l}`);a.updatedOutput?(U.writeFileSync(a.updatedOutput,JSON.stringify(n,null,2)),console.log(`\u{1F4BE} Saved to ${a.updatedOutput}`)):console.log("\u{1F4A1} --updated-output $(mktemp /tmp/updated-XXXXXX.json)")}}St().catch(a=>{console.error("\u274C Fatal error:",a),process.exit(1)});
