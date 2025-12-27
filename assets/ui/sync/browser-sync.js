var $trip2g_sync_bundle = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // src/sync/browser/index.ts
  var index_exports = {};
  __export(index_exports, {
    BrowserEnv: () => BrowserEnv,
    checkPermission: () => checkPermission,
    classifySync: () => classifySync,
    clearDirectoryHandle: () => clearDirectoryHandle,
    clearSyncState: () => clearSyncState,
    configureStorage: () => configureStorage,
    executePlan: () => executePlan,
    filterPlan: () => filterPlan,
    loadDirectoryHandle: () => loadDirectoryHandle,
    loadSyncState: () => loadSyncState,
    requestPermission: () => requestPermission,
    saveDirectoryHandle: () => saveDirectoryHandle,
    saveSyncState: () => saveSyncState
  });

  // node_modules/browser-fs-access/dist/index.modern.js
  var e = (() => {
    if ("undefined" == typeof self) return false;
    if ("top" in self && self !== top) try {
      top.window.document._ = 0;
    } catch (e2) {
      return false;
    }
    return "showOpenFilePicker" in self;
  })();
  var t = e ? Promise.resolve().then(function() {
    return l;
  }) : Promise.resolve().then(function() {
    return v;
  });
  var r = e ? Promise.resolve().then(function() {
    return y;
  }) : Promise.resolve().then(function() {
    return b;
  });
  async function i(...e2) {
    return (await r).default(...e2);
  }
  var a = e ? Promise.resolve().then(function() {
    return m;
  }) : Promise.resolve().then(function() {
    return k;
  });
  var s = async (e2) => {
    const t2 = await e2.getFile();
    return t2.handle = e2, t2;
  };
  var c = async (e2 = [{}]) => {
    Array.isArray(e2) || (e2 = [e2]);
    const t2 = [];
    e2.forEach((e3, n2) => {
      t2[n2] = { description: e3.description || "Files", accept: {} }, e3.mimeTypes ? e3.mimeTypes.map((r3) => {
        t2[n2].accept[r3] = e3.extensions || [];
      }) : t2[n2].accept["*/*"] = e3.extensions || [];
    });
    const n = await window.showOpenFilePicker({ id: e2[0].id, startIn: e2[0].startIn, types: t2, multiple: e2[0].multiple || false, excludeAcceptAllOption: e2[0].excludeAcceptAllOption || false }), r2 = await Promise.all(n.map(s));
    return e2[0].multiple ? r2 : r2[0];
  };
  var l = { __proto__: null, default: c };
  function u(e2) {
    function t2(e3) {
      if (Object(e3) !== e3) return Promise.reject(new TypeError(e3 + " is not an object."));
      var t3 = e3.done;
      return Promise.resolve(e3.value).then(function(e4) {
        return { value: e4, done: t3 };
      });
    }
    return u = function(e3) {
      this.s = e3, this.n = e3.next;
    }, u.prototype = { s: null, n: null, next: function() {
      return t2(this.n.apply(this.s, arguments));
    }, return: function(e3) {
      var n = this.s.return;
      return void 0 === n ? Promise.resolve({ value: e3, done: true }) : t2(n.apply(this.s, arguments));
    }, throw: function(e3) {
      var n = this.s.return;
      return void 0 === n ? Promise.reject(e3) : t2(n.apply(this.s, arguments));
    } }, new u(e2);
  }
  var p = async (e2, t2, n = e2.name, r2) => {
    const i2 = [], a2 = [];
    var o, s2 = false, c2 = false;
    try {
      for (var l2, d2 = (function(e3) {
        var t3, n2, r3, i3 = 2;
        for ("undefined" != typeof Symbol && (n2 = Symbol.asyncIterator, r3 = Symbol.iterator); i3--; ) {
          if (n2 && null != (t3 = e3[n2])) return t3.call(e3);
          if (r3 && null != (t3 = e3[r3])) return new u(t3.call(e3));
          n2 = "@@asyncIterator", r3 = "@@iterator";
        }
        throw new TypeError("Object is not async iterable");
      })(e2.values()); s2 = !(l2 = await d2.next()).done; s2 = false) {
        const o2 = l2.value, s3 = `${n}/${o2.name}`;
        "file" === o2.kind ? a2.push(o2.getFile().then((t3) => (t3.directoryHandle = e2, t3.handle = o2, Object.defineProperty(t3, "webkitRelativePath", { configurable: true, enumerable: true, get: () => s3 })))) : "directory" !== o2.kind || !t2 || r2 && r2(o2) || i2.push(p(o2, t2, s3, r2));
      }
    } catch (e3) {
      c2 = true, o = e3;
    } finally {
      try {
        s2 && null != d2.return && await d2.return();
      } finally {
        if (c2) throw o;
      }
    }
    return [...(await Promise.all(i2)).flat(), ...await Promise.all(a2)];
  };
  var d = async (e2 = {}) => {
    e2.recursive = e2.recursive || false, e2.mode = e2.mode || "read";
    const t2 = await window.showDirectoryPicker({ id: e2.id, startIn: e2.startIn, mode: e2.mode });
    return (await (await t2.values()).next()).done ? [t2] : p(t2, e2.recursive, void 0, e2.skipDirectory);
  };
  var y = { __proto__: null, default: d };
  var f = async (e2, t2 = [{}], n = null, r2 = false, i2 = null) => {
    Array.isArray(t2) || (t2 = [t2]), t2[0].fileName = t2[0].fileName || "Untitled";
    const a2 = [];
    let o = null;
    if (e2 instanceof Blob && e2.type ? o = e2.type : e2.headers && e2.headers.get("content-type") && (o = e2.headers.get("content-type")), t2.forEach((e3, t3) => {
      a2[t3] = { description: e3.description || "Files", accept: {} }, e3.mimeTypes ? (0 === t3 && o && e3.mimeTypes.push(o), e3.mimeTypes.map((n2) => {
        a2[t3].accept[n2] = e3.extensions || [];
      })) : o ? a2[t3].accept[o] = e3.extensions || [] : a2[t3].accept["*/*"] = e3.extensions || [];
    }), n) try {
      await n.getFile();
    } catch (e3) {
      if (n = null, r2) throw e3;
    }
    const s2 = n || await window.showSaveFilePicker({ suggestedName: t2[0].fileName, id: t2[0].id, startIn: t2[0].startIn, types: a2, excludeAcceptAllOption: t2[0].excludeAcceptAllOption || false });
    !n && i2 && i2(s2);
    const c2 = await s2.createWritable();
    if ("stream" in e2) {
      const t3 = e2.stream();
      return await t3.pipeTo(c2), s2;
    }
    return "body" in e2 ? (await e2.body.pipeTo(c2), s2) : (await c2.write(await e2), await c2.close(), s2);
  };
  var m = { __proto__: null, default: f };
  var w = async (e2 = [{}]) => (Array.isArray(e2) || (e2 = [e2]), new Promise((t2, n) => {
    const r2 = document.createElement("input");
    r2.type = "file";
    const i2 = [...e2.map((e3) => e3.mimeTypes || []), ...e2.map((e3) => e3.extensions || [])].join();
    r2.multiple = e2[0].multiple || false, r2.accept = i2 || "", r2.style.display = "none", document.body.append(r2);
    const a2 = (e3) => {
      "function" == typeof o && o(), t2(e3);
    }, o = e2[0].legacySetup && e2[0].legacySetup(a2, () => o(n), r2), s2 = () => {
      window.removeEventListener("focus", s2), r2.remove();
    };
    r2.addEventListener("click", () => {
      window.addEventListener("focus", s2);
    }), r2.addEventListener("change", () => {
      window.removeEventListener("focus", s2), r2.remove(), a2(r2.multiple ? Array.from(r2.files) : r2.files[0]);
    }), "showPicker" in HTMLInputElement.prototype ? r2.showPicker() : r2.click();
  }));
  var v = { __proto__: null, default: w };
  var h = async (e2 = [{}]) => (Array.isArray(e2) || (e2 = [e2]), e2[0].recursive = e2[0].recursive || false, new Promise((t2, n) => {
    const r2 = document.createElement("input");
    r2.type = "file", r2.webkitdirectory = true;
    const i2 = (e3) => {
      "function" == typeof a2 && a2(), t2(e3);
    }, a2 = e2[0].legacySetup && e2[0].legacySetup(i2, () => a2(n), r2);
    r2.addEventListener("change", () => {
      let t3 = Array.from(r2.files);
      e2[0].recursive ? e2[0].recursive && e2[0].skipDirectory && (t3 = t3.filter((t4) => t4.webkitRelativePath.split("/").every((t5) => !e2[0].skipDirectory({ name: t5, kind: "directory" })))) : t3 = t3.filter((e3) => 2 === e3.webkitRelativePath.split("/").length), i2(t3);
    }), "showPicker" in HTMLInputElement.prototype ? r2.showPicker() : r2.click();
  }));
  var b = { __proto__: null, default: h };
  var P = async (e2, t2 = {}) => {
    Array.isArray(t2) && (t2 = t2[0]);
    const n = document.createElement("a");
    let r2 = e2;
    "body" in e2 && (r2 = await (async function(e3, t3) {
      const n2 = e3.getReader(), r3 = new ReadableStream({ start: (e4) => (async function t4() {
        return n2.read().then(({ done: n3, value: r4 }) => {
          if (!n3) return e4.enqueue(r4), t4();
          e4.close();
        });
      })() }), i3 = new Response(r3), a3 = await i3.blob();
      return n2.releaseLock(), new Blob([a3], { type: t3 });
    })(e2.body, e2.headers.get("content-type"))), n.download = t2.fileName || "Untitled", n.href = URL.createObjectURL(await r2);
    const i2 = () => {
      "function" == typeof a2 && a2();
    }, a2 = t2.legacySetup && t2.legacySetup(i2, () => a2(), n);
    return n.addEventListener("click", () => {
      setTimeout(() => URL.revokeObjectURL(n.href), 3e4), i2();
    }), n.click(), null;
  };
  var k = { __proto__: null, default: P };

  // src/sync/classify.ts
  function classifyFile(localHash, remoteHash, lastSyncedHash) {
    if (localHash === null && remoteHash === null) {
      return "unchanged";
    }
    if (localHash === remoteHash) {
      return "unchanged";
    }
    if (localHash !== null && remoteHash === null) {
      if (lastSyncedHash) {
        return "server_deleted";
      }
      return "local_only";
    }
    if (localHash === null && remoteHash !== null) {
      if (lastSyncedHash) {
        return "local_deleted";
      }
      return "remote_only";
    }
    if (!lastSyncedHash) {
      return "conflict";
    }
    if (localHash === lastSyncedHash) {
      return "pull";
    }
    if (remoteHash === lastSyncedHash) {
      return "push";
    }
    return "conflict";
  }
  async function classifySync(env) {
    const syncState = env.getSyncState();
    const [localFiles, serverHashes] = await Promise.all([
      env.getLocalFiles(),
      env.getServerHashes()
    ]);
    const serverHashMap = /* @__PURE__ */ new Map();
    for (const item of serverHashes) {
      serverHashMap.set(item.path, item.hash);
    }
    const localHashes = /* @__PURE__ */ new Map();
    const cachedMtimes = syncState.mtimes || {};
    const cachedLocalHashes = syncState.localHashes || {};
    for (const file of localFiles) {
      const cachedMtime = cachedMtimes[file.path];
      const cachedHash = cachedLocalHashes[file.path];
      if (cachedMtime === file.mtime && cachedHash) {
        localHashes.set(file.path, cachedHash);
      } else {
        const content = await env.readFileContent(file.path);
        const hash = await env.computeHash(content);
        localHashes.set(file.path, hash);
      }
    }
    const allPaths = /* @__PURE__ */ new Set([
      ...localHashes.keys(),
      ...serverHashMap.keys()
    ]);
    const classifications = [];
    const pulls = [];
    const pushes = [];
    const conflicts = [];
    const localOnly = [];
    const remoteOnly = [];
    const localDeleted = [];
    const serverDeleted = [];
    let unchanged = 0;
    for (const path of allPaths) {
      const localHash = localHashes.get(path) || null;
      const remoteHash = serverHashMap.get(path) || null;
      const lastSyncedHash = syncState.files[path] || null;
      const action = classifyFile(localHash, remoteHash, lastSyncedHash);
      const classification = {
        path,
        action,
        localHash,
        remoteHash,
        lastSyncedHash
      };
      classifications.push(classification);
      switch (action) {
        case "unchanged":
          unchanged++;
          break;
        case "pull":
          pulls.push(classification);
          break;
        case "push":
          pushes.push(classification);
          break;
        case "conflict":
          conflicts.push(classification);
          break;
        case "local_only":
          localOnly.push(classification);
          break;
        case "remote_only":
          remoteOnly.push(classification);
          break;
        case "local_deleted":
          localDeleted.push(classification);
          break;
        case "server_deleted":
          serverDeleted.push(classification);
          break;
      }
    }
    return {
      classifications,
      pulls,
      pushes,
      conflicts,
      localOnly,
      remoteOnly,
      localDeleted,
      serverDeleted,
      unchanged
    };
  }

  // src/sync/filter.ts
  function filterPlan(plan, options) {
    const { twoWaySync, hasPublishFields } = options;
    const isPublishable = (path) => {
      if (!hasPublishFields) return true;
      return hasPublishFields(path);
    };
    const filteredClassifications = [];
    const pulls = [];
    const pushes = [];
    const conflicts = [];
    const localOnly = [];
    const remoteOnly = [];
    const localDeleted = [];
    const serverDeleted = [];
    let unchanged = 0;
    for (const c2 of plan.classifications) {
      const publishable = isPublishable(c2.path);
      switch (c2.action) {
        case "unchanged":
          filteredClassifications.push(c2);
          unchanged++;
          break;
        case "pull":
          if (twoWaySync && publishable) {
            filteredClassifications.push(c2);
            pulls.push(c2);
          }
          break;
        case "push":
          if (publishable) {
            filteredClassifications.push(c2);
            pushes.push(c2);
          }
          break;
        case "conflict":
          if (!twoWaySync) {
            if (publishable) {
              const asPush = { ...c2, action: "push" };
              filteredClassifications.push(asPush);
              pushes.push(asPush);
            }
          } else if (publishable) {
            filteredClassifications.push(c2);
            conflicts.push(c2);
          }
          break;
        case "local_only":
          if (publishable) {
            filteredClassifications.push(c2);
            localOnly.push(c2);
          }
          break;
        case "remote_only":
          if (twoWaySync) {
            filteredClassifications.push(c2);
            remoteOnly.push(c2);
          }
          break;
        case "local_deleted":
          if (publishable) {
            filteredClassifications.push(c2);
            localDeleted.push(c2);
          }
          break;
        case "server_deleted":
          if (twoWaySync) {
            filteredClassifications.push(c2);
            serverDeleted.push(c2);
          }
          break;
      }
    }
    return {
      classifications: filteredClassifications,
      pulls,
      pushes,
      conflicts,
      localOnly,
      remoteOnly,
      localDeleted,
      serverDeleted,
      unchanged
    };
  }

  // src/sync/execute.ts
  async function executePlan(env, plan, options = { twoWaySync: false }) {
    const result = {
      pulled: 0,
      pushed: 0,
      conflictsResolved: 0,
      assetsUploaded: 0,
      assetsDownloaded: 0,
      errors: []
    };
    const syncState = env.getSyncState();
    const pulledPaths = [];
    if (plan.pulls.length > 0 || plan.remoteOnly.length > 0) {
      const toPull = [...plan.pulls, ...plan.remoteOnly];
      const pullResult = await executePulls(env, toPull, syncState);
      result.pulled = pullResult.count;
      result.errors.push(...pullResult.errors);
      pulledPaths.push(...pullResult.pulledPaths);
    }
    if (pulledPaths.length > 0) {
      const assetResult = await downloadAssetsForNotes(env, pulledPaths);
      result.assetsDownloaded += assetResult.downloaded;
      result.errors.push(...assetResult.errors);
    }
    if (options.twoWaySync) {
      const unchangedServerPaths = plan.classifications.filter((c2) => c2.action === "unchanged" && c2.remoteHash !== null).map((c2) => c2.path);
      if (unchangedServerPaths.length > 0) {
        const assetResult = await downloadAssetsForNotes(env, unchangedServerPaths);
        result.assetsDownloaded += assetResult.downloaded;
        result.errors.push(...assetResult.errors);
      }
    }
    if (plan.serverDeleted.length > 0) {
      await handleServerDeleted(env, plan.serverDeleted, syncState);
    }
    if (plan.conflicts.length > 0) {
      const conflictResult = await handleConflicts(env, plan.conflicts, syncState);
      result.conflictsResolved = conflictResult.resolved;
      result.errors.push(...conflictResult.errors);
    }
    const toPush = [...plan.pushes, ...plan.localOnly];
    let pushedNotes = [];
    if (toPush.length > 0) {
      const confirmed = await env.confirmPush(toPush.map((c2) => c2.path));
      if (confirmed) {
        const pushResult = await executePushes(env, toPush, syncState);
        result.pushed = pushResult.count;
        result.errors.push(...pushResult.errors);
        pushedNotes = pushResult.pushedNotes;
      }
    }
    if (plan.localDeleted.length > 0) {
      await handleLocalDeleted(env, plan.localDeleted, syncState);
    }
    if (pushedNotes.length > 0) {
      const assetResult = await syncAssets(env, pushedNotes, options.twoWaySync);
      result.assetsUploaded = assetResult.uploaded;
      result.assetsDownloaded = assetResult.downloaded;
      result.errors.push(...assetResult.errors);
    }
    const unchangedPaths = plan.classifications.filter((c2) => c2.action === "unchanged" && c2.remoteHash !== null).map((c2) => c2.path);
    if (unchangedPaths.length > 0) {
      const assetResult = await uploadMissingAssetsForNotes(env, unchangedPaths);
      result.assetsUploaded += assetResult.uploaded;
      result.errors.push(...assetResult.errors);
    }
    if (result.pushed > 0 || result.assetsUploaded > 0) {
      await env.commitNotes();
    }
    await env.saveSyncState(syncState);
    return result;
  }
  async function executePulls(env, pulls, syncState) {
    if (pulls.length === 0) {
      return { count: 0, errors: [], pulledPaths: [] };
    }
    const paths = pulls.map((p2) => p2.path);
    const errors = [];
    const pulledPaths = [];
    let count = 0;
    const contents = await env.fetchNoteContents(paths);
    const contentMap = new Map(contents.map((c2) => [c2.path, c2.content]));
    const total = pulls.length;
    let current = 0;
    for (const pull of pulls) {
      current++;
      env.onProgress({ step: "pull", current, total, path: pull.path });
      const content = contentMap.get(pull.path);
      if (content === void 0) {
        errors.push(`Failed to fetch: ${pull.path}`);
        continue;
      }
      try {
        const dirPath = pull.path.substring(0, pull.path.lastIndexOf("/"));
        if (dirPath) {
          await env.createFolder(dirPath);
        }
        await env.writeFile(pull.path, content);
        const hash = await env.computeHash(content);
        syncState.files[pull.path] = hash;
        count++;
        pulledPaths.push(pull.path);
      } catch (e2) {
        errors.push(`Failed to write ${pull.path}: ${e2}`);
      }
    }
    return { count, errors, pulledPaths };
  }
  async function executePushes(env, pushes, syncState) {
    if (pushes.length === 0) {
      return { count: 0, errors: [], pushedNotes: [] };
    }
    const errors = [];
    const updates = [];
    const total = pushes.length;
    let current = 0;
    for (const push of pushes) {
      current++;
      env.onProgress({ step: "push", current, total, path: push.path });
      try {
        const content = await env.readFileContent(push.path);
        updates.push({ path: push.path, content });
      } catch (e2) {
        errors.push(`Failed to read ${push.path}: ${e2}`);
      }
    }
    if (updates.length === 0) {
      return { count: 0, errors, pushedNotes: [] };
    }
    const updatePaths = new Set(updates.map((u2) => u2.path));
    const batchSize = env.pushBatchSize || 100;
    const pushedNotes = [];
    for (let i2 = 0; i2 < updates.length; i2 += batchSize) {
      const batch = updates.slice(i2, i2 + batchSize);
      const batchNotes = await env.pushNotes(batch, true);
      pushedNotes.push(...batchNotes);
    }
    const serverPaths = new Set(pushedNotes.map((n) => n.path));
    let pushedCount = 0;
    for (const update of updates) {
      if (serverPaths.has(update.path)) {
        const hash = await env.computeHash(update.content);
        syncState.files[update.path] = hash;
        pushedCount++;
      }
    }
    const filteredNotes = pushedNotes.filter((n) => updatePaths.has(n.path));
    return { count: pushedCount, errors, pushedNotes: filteredNotes };
  }
  async function handleConflicts(env, conflicts, syncState) {
    if (conflicts.length === 0) {
      return { resolved: 0, errors: [] };
    }
    const errors = [];
    const paths = conflicts.map((c2) => c2.path);
    const remoteContents = await env.fetchNoteContents(paths);
    const remoteMap = new Map(remoteContents.map((c2) => [c2.path, c2.content]));
    const conflictInfos = [];
    for (const conflict of conflicts) {
      const remoteContent = remoteMap.get(conflict.path);
      if (remoteContent === void 0) {
        continue;
      }
      try {
        const localContent = await env.readFileContent(conflict.path);
        conflictInfos.push({
          path: conflict.path,
          localContent,
          remoteContent,
          localHash: conflict.localHash,
          remoteHash: conflict.remoteHash
        });
      } catch (e2) {
        console.warn(`Failed to read local file for conflict ${conflict.path}:`, e2);
        errors.push(`Failed to read local file for conflict: ${conflict.path}`);
      }
    }
    if (conflictInfos.length === 0) {
      return { resolved: 0, errors };
    }
    const resolutions = await env.onConflict(conflictInfos);
    let resolved = 0;
    for (let i2 = 0; i2 < conflictInfos.length; i2++) {
      const info = conflictInfos[i2];
      const resolution = resolutions[i2] || "skip";
      try {
        await resolveConflict(env, info, resolution, syncState);
        if (resolution !== "skip") {
          resolved++;
        }
      } catch (e2) {
        errors.push(`Failed to resolve conflict for ${info.path}: ${e2}`);
      }
    }
    return { resolved, errors };
  }
  async function resolveConflict(env, conflict, resolution, syncState) {
    switch (resolution) {
      case "keep_local":
        await env.pushNotes([{ path: conflict.path, content: conflict.localContent }], true);
        syncState.files[conflict.path] = conflict.localHash;
        break;
      case "keep_remote":
        await env.writeFile(conflict.path, conflict.remoteContent);
        syncState.files[conflict.path] = conflict.remoteHash;
        break;
      case "keep_both": {
        const ext = conflict.path.substring(conflict.path.lastIndexOf("."));
        const baseName = conflict.path.substring(0, conflict.path.lastIndexOf("."));
        const newPath = `${baseName} (server)${ext}`;
        await env.writeFile(newPath, conflict.remoteContent);
        syncState.files[conflict.path] = conflict.localHash;
        const remoteHash = await env.computeHash(conflict.remoteContent);
        syncState.files[newPath] = remoteHash;
        break;
      }
      // Stryker disable next-line StringLiteral,ConditionalExpression: skip case is intentionally empty
      case "skip":
        break;
    }
  }
  async function handleServerDeleted(env, serverDeleted, syncState) {
    if (serverDeleted.length === 0) {
      return;
    }
    const paths = serverDeleted.map((c2) => c2.path);
    const deleteLocally = await env.onServerDeleted(paths);
    if (deleteLocally) {
      for (const c2 of serverDeleted) {
        try {
          await env.deleteFile(c2.path);
          delete syncState.files[c2.path];
        } catch (e2) {
          console.warn(`Failed to delete file ${c2.path}:`, e2);
        }
      }
    } else {
      for (const c2 of serverDeleted) {
        if (c2.localHash) {
          syncState.files[c2.path] = c2.localHash;
        }
      }
    }
  }
  async function handleLocalDeleted(env, localDeleted, syncState) {
    if (localDeleted.length === 0) {
      return;
    }
    const paths = localDeleted.map((c2) => c2.path);
    await env.hideNotes(paths);
    for (const path of paths) {
      delete syncState.files[path];
    }
  }
  async function syncAssets(env, pushedNotes, twoWaySync) {
    console.log(`[Trip2g Sync] syncAssets called with ${pushedNotes.length} notes, twoWaySync=${twoWaySync}`);
    const result = {
      uploaded: 0,
      downloaded: 0,
      conflictsResolved: 0,
      errors: []
    };
    if (pushedNotes.length === 0) {
      return result;
    }
    const toUpload = [];
    const toDownload = [];
    const conflicts = [];
    for (const note of pushedNotes) {
      console.log(`[Trip2g Sync] Processing assets for note: ${note.path}, assets count: ${note.assets?.length ?? 0}`);
      if (!note.assets || note.assets.length === 0) {
        continue;
      }
      for (const asset of note.assets) {
        const localPath = await env.resolveAssetPath(asset.path, note.path);
        console.log(`[Trip2g Sync] Asset "${asset.path}" -> localPath: ${localPath ?? "NOT FOUND"}, sha256Hash: ${asset.sha256Hash ?? "null"}`);
        if (!localPath) {
          continue;
        }
        if (!asset.sha256Hash || !asset.absolutePath || !asset.url) {
          console.log(`[Trip2g Sync] Queuing upload: ${asset.path} (no hash on server)`);
          toUpload.push({ noteId: note.id, notePath: note.path, asset, localPath });
          continue;
        }
        const exists = await env.fileExists(localPath);
        if (exists) {
          try {
            const localData = await env.readBinaryFile(localPath);
            const localHash = await env.computeBinaryHash(localData);
            if (localHash === asset.sha256Hash) {
              continue;
            }
            conflicts.push({
              path: asset.path,
              absolutePath: localPath,
              noteId: note.id,
              localHash,
              remoteHash: asset.sha256Hash,
              remoteUrl: asset.url
            });
          } catch (e2) {
            result.errors.push(`Failed to read local asset ${localPath}: ${e2}`);
          }
        } else if (twoWaySync) {
          toDownload.push({ asset, localPath });
        }
      }
    }
    console.log(`[Trip2g Sync] Assets to upload: ${toUpload.length}, to download: ${toDownload.length}, conflicts: ${conflicts.length}`);
    if (toUpload.length > 0) {
      const uniqueUploads = /* @__PURE__ */ new Map();
      for (const item of toUpload) {
        const key = `${item.noteId}:${item.localPath}`;
        if (!uniqueUploads.has(key)) {
          uniqueUploads.set(key, item);
        }
      }
      const deduped = Array.from(uniqueUploads.values());
      const uploadTotal = deduped.length;
      let uploadCurrent = 0;
      console.log(`[Trip2g Sync] Uploading ${uploadTotal} unique (note, asset) pairs`);
      for (const item of deduped) {
        uploadCurrent++;
        console.log(`[Trip2g Sync] Uploading asset ${uploadCurrent}/${uploadTotal}: ${item.localPath}`);
        env.onProgress({ step: "upload_asset", current: uploadCurrent, total: uploadTotal, path: item.asset.path });
        try {
          const localData = await env.readBinaryFile(item.localPath);
          const localHash = await env.computeBinaryHash(localData);
          const blob = new Blob([localData]);
          const fileName = item.localPath.substring(item.localPath.lastIndexOf("/") + 1);
          const success = await env.uploadAsset({
            noteId: item.noteId,
            blob,
            fileName,
            relativePath: item.asset.path,
            absolutePath: item.localPath,
            sha256Hash: localHash
          });
          if (success) {
            result.uploaded++;
          }
        } catch (e2) {
          result.errors.push(`Failed to upload asset ${item.asset.path}: ${e2}`);
        }
      }
    }
    if (toDownload.length > 0) {
      const downloadTotal = toDownload.length;
      let downloadCurrent = 0;
      for (const item of toDownload) {
        downloadCurrent++;
        env.onProgress({ step: "download_asset", current: downloadCurrent, total: downloadTotal, path: item.asset.path });
        if (!item.asset.url) {
          continue;
        }
        try {
          const data = await env.downloadAsset(item.asset.url);
          if (!data) {
            result.errors.push(`Failed to download asset ${item.asset.path}`);
            continue;
          }
          const dirPath = item.localPath.substring(0, item.localPath.lastIndexOf("/"));
          if (dirPath) {
            await env.createFolder(dirPath);
          }
          await env.writeBinaryFile(item.localPath, data);
          result.downloaded++;
        } catch (e2) {
          result.errors.push(`Failed to download asset ${item.asset.path}: ${e2}`);
        }
      }
    }
    if (conflicts.length > 0) {
      const assetResult = await handleAssetConflicts(env, conflicts, twoWaySync);
      result.uploaded += assetResult.uploaded;
      result.downloaded += assetResult.downloaded;
      result.conflictsResolved = assetResult.conflictsResolved;
      result.errors.push(...assetResult.errors);
    }
    return result;
  }
  async function handleAssetConflicts(env, conflicts, twoWaySync) {
    const result = {
      uploaded: 0,
      downloaded: 0,
      conflictsResolved: 0,
      errors: []
    };
    if (conflicts.length === 0) {
      return result;
    }
    let resolutions;
    if (twoWaySync) {
      resolutions = await env.onAssetConflict(conflicts);
    } else {
      resolutions = conflicts.map(() => "keep_local");
    }
    for (let i2 = 0; i2 < conflicts.length; i2++) {
      const conflict = conflicts[i2];
      const resolution = resolutions[i2] || "skip";
      try {
        if (resolution === "keep_local") {
          const localData = await env.readBinaryFile(conflict.absolutePath);
          const blob = new Blob([localData]);
          const fileName = conflict.absolutePath.substring(conflict.absolutePath.lastIndexOf("/") + 1);
          const success = await env.uploadAsset({
            noteId: conflict.noteId,
            blob,
            fileName,
            relativePath: conflict.path,
            absolutePath: conflict.absolutePath,
            sha256Hash: conflict.localHash
          });
          if (success) {
            result.uploaded++;
            result.conflictsResolved++;
          }
        } else if (resolution === "keep_remote") {
          const data = await env.downloadAsset(conflict.remoteUrl);
          if (data) {
            await env.writeBinaryFile(conflict.absolutePath, data);
            result.downloaded++;
            result.conflictsResolved++;
          } else {
            result.errors.push(`Failed to download asset ${conflict.path}`);
          }
        }
      } catch (e2) {
        result.errors.push(`Failed to resolve asset conflict for ${conflict.path}: ${e2}`);
      }
    }
    return result;
  }
  async function downloadAssetsForNotes(env, notePaths) {
    const result = { downloaded: 0, errors: [] };
    if (notePaths.length === 0) {
      return result;
    }
    const noteAssets = await env.fetchNoteAssets(notePaths);
    if (noteAssets.length === 0) {
      return result;
    }
    const toDownload = /* @__PURE__ */ new Map();
    for (const note of noteAssets) {
      for (const asset of note.assets) {
        const absolutePath = asset.absolutePath.replace(/^\//, "");
        if (!toDownload.has(absolutePath)) {
          const exists = await env.fileExists(absolutePath);
          if (!exists) {
            toDownload.set(absolutePath, { url: asset.url, hash: asset.hash });
          }
        }
      }
    }
    if (toDownload.size === 0) {
      return result;
    }
    const total = toDownload.size;
    let current = 0;
    for (const [absolutePath, { url }] of toDownload) {
      current++;
      env.onProgress({ step: "download_asset", current, total, path: absolutePath });
      try {
        const data = await env.downloadAsset(url);
        if (!data) {
          result.errors.push(`Failed to download asset ${absolutePath}`);
          continue;
        }
        const dirPath = absolutePath.substring(0, absolutePath.lastIndexOf("/"));
        if (dirPath) {
          await env.createFolder(dirPath);
        }
        await env.writeBinaryFile(absolutePath, data);
        result.downloaded++;
      } catch (e2) {
        result.errors.push(`Failed to download asset ${absolutePath}: ${e2}`);
      }
    }
    return result;
  }
  async function uploadMissingAssetsForNotes(env, notePaths) {
    const result = { uploaded: 0, errors: [] };
    if (notePaths.length === 0) {
      return result;
    }
    const noteAssets = await env.fetchNoteAssets(notePaths);
    if (noteAssets.length === 0) {
      return result;
    }
    const toUpload = [];
    for (const note of noteAssets) {
      for (const asset of note.assets) {
        let localPath = asset.absolutePath?.replace(/^\//, "");
        if (!localPath && asset.id) {
          const noteDir = note.path.includes("/") ? note.path.substring(0, note.path.lastIndexOf("/")) : "";
          const assetPath = asset.id.replace(/^\.\//, "");
          localPath = noteDir ? `${noteDir}/${assetPath}` : assetPath;
        }
        if (!localPath) {
          continue;
        }
        const exists = await env.fileExists(localPath);
        if (!exists) {
          continue;
        }
        try {
          const localData = await env.readBinaryFile(localPath);
          const localHash = await env.computeBinaryHash(localData);
          if (localHash === asset.hash) {
            continue;
          }
          toUpload.push({
            noteId: note.noteId,
            // version ID from server
            notePath: note.path,
            assetPath: asset.id,
            localPath,
            localHash
          });
        } catch (e2) {
          result.errors.push(`Failed to read local asset ${localPath}: ${e2}`);
        }
      }
    }
    if (toUpload.length === 0) {
      return result;
    }
    const total = toUpload.length;
    let current = 0;
    for (const item of toUpload) {
      current++;
      env.onProgress({ step: "upload_asset", current, total, path: item.assetPath });
      try {
        const localData = await env.readBinaryFile(item.localPath);
        const blob = new Blob([localData]);
        const fileName = item.localPath.substring(item.localPath.lastIndexOf("/") + 1);
        const success = await env.uploadAsset({
          noteId: item.noteId,
          blob,
          fileName,
          relativePath: item.assetPath,
          absolutePath: item.localPath,
          sha256Hash: item.localHash
          // Use pre-computed hash
        });
        if (success) {
          result.uploaded++;
        }
      } catch (e2) {
        result.errors.push(`Failed to upload asset ${item.assetPath}: ${e2}`);
      }
    }
    return result;
  }

  // src/sync/browser/storage.ts
  var DEFAULT_CONFIG = {
    dbName: "trip2g-sync"
  };
  var currentConfig = DEFAULT_CONFIG;
  var DB_VERSION = 1;
  var HANDLE_STORE = "handles";
  var STATE_STORE = "state";
  var dbPromise = null;
  function configureStorage(config) {
    if (dbPromise && config.dbName && config.dbName !== currentConfig.dbName) {
      dbPromise.then((db) => db.close()).catch(() => {
      });
      dbPromise = null;
    }
    currentConfig = { ...currentConfig, ...config };
  }
  function openDB() {
    if (dbPromise) return dbPromise;
    dbPromise = new Promise((resolve, reject) => {
      const request = indexedDB.open(currentConfig.dbName, DB_VERSION);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result);
      request.onupgradeneeded = (event) => {
        const db = event.target.result;
        if (!db.objectStoreNames.contains(HANDLE_STORE)) {
          db.createObjectStore(HANDLE_STORE);
        }
        if (!db.objectStoreNames.contains(STATE_STORE)) {
          db.createObjectStore(STATE_STORE);
        }
      };
    });
    return dbPromise;
  }
  async function saveDirectoryHandle(handle) {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(HANDLE_STORE, "readwrite");
      const store = tx.objectStore(HANDLE_STORE);
      const request = store.put(handle, "directory");
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }
  async function loadDirectoryHandle() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(HANDLE_STORE, "readonly");
      const store = tx.objectStore(HANDLE_STORE);
      const request = store.get("directory");
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result ?? null);
    });
  }
  async function clearDirectoryHandle() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(HANDLE_STORE, "readwrite");
      const store = tx.objectStore(HANDLE_STORE);
      const request = store.delete("directory");
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }
  async function requestPermission(handle) {
    const permission = await handle.requestPermission({ mode: "readwrite" });
    return permission === "granted";
  }
  async function checkPermission(handle) {
    const permission = await handle.queryPermission({ mode: "readwrite" });
    return permission === "granted";
  }
  async function saveSyncState(state) {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STATE_STORE, "readwrite");
      const store = tx.objectStore(STATE_STORE);
      const request = store.put(state, "syncState");
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }
  async function loadSyncState() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STATE_STORE, "readonly");
      const store = tx.objectStore(STATE_STORE);
      const request = store.get("syncState");
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result ?? { files: {} });
    });
  }
  async function clearSyncState() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const tx = db.transaction(STATE_STORE, "readwrite");
      const store = tx.objectStore(STATE_STORE);
      const request = store.delete("syncState");
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  // src/sync/browser/index.ts
  var BrowserEnv = class {
    constructor(options, callbacks = {}) {
      this.directoryHandle = null;
      this.syncState = { files: {} };
      this.fileCache = /* @__PURE__ */ new Map();
      this.existsCache = /* @__PURE__ */ new Map();
      this.pushBatchSize = 100;
      this.options = options;
      this.callbacks = callbacks;
    }
    log(message, level = "info") {
      this.callbacks.onLog?.(message, level);
    }
    // ============ Directory Management ============
    /**
     * Check if a directory is already saved in IndexedDB and has permission.
     */
    async hasStoredDirectory() {
      const handle = await loadDirectoryHandle();
      if (!handle) return false;
      const hasPermission = await checkPermission(handle);
      if (hasPermission) {
        this.directoryHandle = handle;
        return true;
      }
      return false;
    }
    /**
     * Request permission for the stored directory handle.
     * Must be called on user gesture (button click).
     */
    async requestStoredPermission() {
      const handle = await loadDirectoryHandle();
      if (!handle) return false;
      const granted = await requestPermission(handle);
      if (granted) {
        this.directoryHandle = handle;
      }
      return granted;
    }
    /**
     * Open a directory picker and store the handle.
     * Must be called on user gesture (button click).
     */
    async selectDirectory() {
      try {
        const blobs = await i({
          recursive: true,
          mode: "readwrite"
        });
        if (blobs.length > 0 && blobs[0].directoryHandle) {
          this.directoryHandle = blobs[0].directoryHandle;
          await saveDirectoryHandle(this.directoryHandle);
          this.log(`Directory selected: ${this.directoryHandle.name}`);
          return true;
        }
        return false;
      } catch (e2) {
        if (e2.name === "AbortError") {
          return false;
        }
        throw e2;
      }
    }
    /**
     * Clear stored directory handle and sync state.
     */
    async clearDirectory() {
      this.directoryHandle = null;
      this.fileCache.clear();
      this.existsCache.clear();
      await clearDirectoryHandle();
      await clearSyncState();
    }
    /**
     * Get the name of the current directory.
     */
    getDirectoryName() {
      return this.directoryHandle?.name ?? null;
    }
    /**
     * Check if directory is selected and ready.
     */
    isReady() {
      return this.directoryHandle !== null;
    }
    // ============ Sync Operations ============
    /**
     * Initialize the environment (load sync state).
     */
    async init() {
      this.syncState = await loadSyncState();
      const hasDir = await this.hasStoredDirectory();
      if (!hasDir) {
        this.log("No stored directory or permission lost", "warn");
      }
    }
    /**
     * Run full sync operation.
     */
    async sync() {
      if (!this.directoryHandle) {
        throw new Error("No directory selected. Call selectDirectory() first.");
      }
      this.fileCache.clear();
      this.existsCache.clear();
      this.onProgress({ step: "classify", current: 0, total: 1 });
      const plan = await classifySync(this);
      const filterOptions = {
        twoWaySync: this.options.twoWaySync ?? false,
        hasPublishFields: this.options.publishField ? (path) => this.hasPublishFieldSync(path) : void 0
      };
      const filteredPlan = filterPlan(plan, filterOptions);
      const result = await executePlan(this, filteredPlan);
      return result;
    }
    /**
     * Get sync plan without executing (for preview).
     */
    async getSyncPlan() {
      if (!this.directoryHandle) {
        throw new Error("No directory selected. Call selectDirectory() first.");
      }
      this.fileCache.clear();
      this.existsCache.clear();
      const plan = await classifySync(this);
      const filterOptions = {
        twoWaySync: this.options.twoWaySync ?? false,
        hasPublishFields: this.options.publishField ? (path) => this.hasPublishFieldSync(path) : void 0
      };
      return filterPlan(plan, filterOptions);
    }
    // ============ ClassifyEnv Implementation ============
    async getLocalFiles() {
      if (!this.directoryHandle) {
        throw new Error("No directory selected");
      }
      const files = [];
      await this.walkDirectory(this.directoryHandle, "", files);
      return files;
    }
    async walkDirectory(dir, basePath, files) {
      for await (const [name, handle] of dir.entries()) {
        if (name.startsWith(".")) continue;
        const path = basePath ? `${basePath}/${name}` : name;
        if (handle.kind === "directory") {
          await this.walkDirectory(handle, path, files);
        } else if (handle.kind === "file") {
          const ext = name.split(".").pop()?.toLowerCase();
          if (ext === "md" || ext === "html") {
            const file = await handle.getFile();
            files.push({
              path,
              mtime: file.lastModified
            });
            this.fileCache.set(path, handle);
          }
        }
      }
    }
    async getServerHashes() {
      const query = `query FetchServerHashes {
			notePaths {
				path: value
				hash: latestContentHash
			}
		}`;
      const result = await this.graphqlRequest(query);
      return result.notePaths.map((np) => ({
        path: np.path,
        hash: np.hash
      }));
    }
    getSyncState() {
      return this.syncState;
    }
    async computeHash(content) {
      const encoder = new TextEncoder();
      const data = encoder.encode(content);
      const hashBuffer = await crypto.subtle.digest("SHA-256", data);
      return this.arrayBufferToBase64Url(hashBuffer);
    }
    async readFileContent(path) {
      const handle = await this.getFileHandle(path);
      const file = await handle.getFile();
      return file.text();
    }
    // ============ File Operations ============
    async writeFile(path, content) {
      const handle = await this.getOrCreateFileHandle(path);
      const writable = await handle.createWritable();
      await writable.write(content);
      await writable.close();
    }
    async writeBinaryFile(path, data) {
      const handle = await this.getOrCreateFileHandle(path);
      const writable = await handle.createWritable();
      await writable.write(data);
      await writable.close();
    }
    async readBinaryFile(path) {
      const handle = await this.getFileHandle(path);
      const file = await handle.getFile();
      return file.arrayBuffer();
    }
    async deleteFile(path) {
      const parts = path.split("/");
      const fileName = parts.pop();
      const dir = await this.navigateToDir(parts.join("/"), false);
      if (dir) {
        await dir.removeEntry(fileName);
        this.fileCache.delete(path);
        this.existsCache.delete(path);
      }
    }
    async createFolder(path) {
      await this.navigateToDir(path, true);
    }
    async fileExists(path) {
      return this.fileExistsSync(path);
    }
    fileExistsSync(path) {
      if (this.existsCache.has(path)) {
        return this.existsCache.get(path);
      }
      const exists = this.fileCache.has(path);
      this.existsCache.set(path, exists);
      return exists;
    }
    // ============ Server Operations ============
    async pushNotes(updates, skipCommit) {
      if (updates.length === 0) return [];
      if (this.options.publishField) {
        for (const update of updates) {
          if (!this.hasPublishFieldInContent(update.content)) {
            throw new Error(
              `[Security] Attempted to push note "${update.path}" without publish field "${this.options.publishField}". This is a bug in the sync logic - please report it.`
            );
          }
        }
      }
      const query = `mutation PushNotes($input: PushNotesInput!) {
			pushNotes(input: $input) {
				... on ErrorPayload {
					__typename
					message
				}
				... on PushNotesPayload {
					__typename
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
				}
			}
		}`;
      const result = await this.graphqlRequest(query, {
        input: {
          updates: updates.map((u2) => ({ path: u2.path, content: u2.content })),
          skipCommit
        }
      });
      if (result.pushNotes.__typename === "ErrorPayload") {
        throw new Error(`Push failed: ${result.pushNotes.message}`);
      }
      this.log(`Pushed ${updates.length} notes`);
      return result.pushNotes.notes.map((n) => ({
        id: String(n.id),
        path: n.path,
        assets: n.assets.map((a2) => ({
          path: a2.path,
          sha256Hash: a2.sha256Hash,
          absolutePath: a2.absolutePath,
          url: a2.url
        }))
      }));
    }
    async hideNotes(paths) {
      if (paths.length === 0) return;
      const query = `mutation HideNotes($input: HideNotesInput!) {
			hideNotes(input: $input) {
				... on ErrorPayload {
					__typename
					message
				}
				... on HideNotesPayload {
					__typename
					success
				}
			}
		}`;
      const result = await this.graphqlRequest(query, { input: { paths } });
      if (result.hideNotes.__typename === "ErrorPayload") {
        throw new Error(`Hide failed: ${result.hideNotes.message}`);
      }
      this.log(`Hidden ${paths.length} notes`);
    }
    async fetchNoteContents(paths) {
      if (paths.length === 0) return [];
      const query = `query FetchNoteContents($filter: NotePathsFilter) {
			notePaths(filter: $filter) {
				path: value
				content: latestContent
			}
		}`;
      const result = await this.graphqlRequest(query, { filter: { paths } });
      return result.notePaths.map((np) => ({
        path: np.path,
        content: np.content
      }));
    }
    async fetchNoteAssets(paths) {
      if (paths.length === 0) return [];
      const query = `mutation FetchNoteAssets($input: PushNotesInput!) {
			pushNotes(input: $input) {
				... on ErrorPayload {
					__typename
					message
				}
				... on PushNotesPayload {
					__typename
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
				}
			}
		}`;
      const result = await this.graphqlRequest(query, { input: { updates: [] } });
      if (result.pushNotes.__typename === "ErrorPayload") {
        this.log(`Failed to fetch note assets: ${result.pushNotes.message}`, "error");
        return [];
      }
      const pathSet = new Set(paths);
      return result.pushNotes.notes.filter((note) => pathSet.has(note.path)).map((note) => ({
        path: note.path,
        noteId: String(note.id),
        assets: note.assets.map((a2) => ({
          id: a2.path,
          url: a2.url ?? "",
          hash: a2.sha256Hash ?? "",
          absolutePath: a2.absolutePath ?? ""
        }))
      }));
    }
    async uploadAsset(params) {
      const query = `mutation UploadNoteAsset($input: UploadNoteAssetInput!) {
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
		}`;
      const operations = JSON.stringify({
        query,
        variables: {
          input: {
            file: null,
            noteId: parseInt(params.noteId),
            sha256Hash: params.sha256Hash,
            path: params.relativePath,
            absolutePath: params.absolutePath
          }
        }
      });
      const map = JSON.stringify({
        "0": ["variables.input.file"]
      });
      const formData = new FormData();
      formData.append("operations", operations);
      formData.append("map", map);
      formData.append("0", params.blob, params.fileName);
      const response = await fetch(this.options.apiUrl, {
        method: "POST",
        headers: {
          "X-API-Key": this.options.apiKey
        },
        body: formData
      });
      if (!response.ok) {
        const body = await response.text();
        throw new Error(`HTTP ${response.status}: ${response.statusText}
${body}`);
      }
      const result = await response.json();
      if (result.errors) {
        throw new Error(result.errors[0]?.message || "Unknown GraphQL error");
      }
      const payload = result.data?.uploadNoteAsset;
      if (payload?.__typename === "ErrorPayload") {
        throw new Error(`Upload failed: ${payload.message}`);
      }
      if (payload?.uploadSkipped) {
        this.log(`Asset skipped (already exists): ${params.relativePath}`);
      } else {
        this.log(`Asset uploaded: ${params.relativePath}`);
      }
      return true;
    }
    async downloadAsset(url) {
      try {
        const response = await fetch(url);
        if (!response.ok) {
          this.log(`Failed to download asset: HTTP ${response.status}`, "error");
          return null;
        }
        return response.arrayBuffer();
      } catch (e2) {
        this.log(`Failed to download asset from ${url}: ${e2}`, "error");
        return null;
      }
    }
    async commitNotes() {
      const query = `mutation CommitNotes {
			commitNotes {
				... on ErrorPayload {
					__typename
					message
				}
				... on CommitNotesPayload {
					__typename
					success
				}
			}
		}`;
      const result = await this.graphqlRequest(query);
      if (result.commitNotes.__typename === "ErrorPayload") {
        throw new Error(`Commit failed: ${result.commitNotes.message}`);
      }
      this.log("Notes committed");
    }
    // ============ Asset Operations ============
    async computeBinaryHash(data) {
      const hashBuffer = await crypto.subtle.digest("SHA-256", data);
      return this.arrayBufferToHex(hashBuffer);
    }
    async resolveAssetPath(assetPath, notePath) {
      if (assetPath.startsWith("./")) {
        const noteDir2 = this.dirname(notePath);
        const relativePath = this.joinPath(noteDir2, assetPath.slice(2));
        if (await this.fileExistsAsync(relativePath)) {
          return relativePath;
        }
        return null;
      }
      if (assetPath.startsWith("/")) {
        const absolutePath = assetPath.slice(1);
        if (await this.fileExistsAsync(absolutePath)) {
          return absolutePath;
        }
        return null;
      }
      if (assetPath.includes("/")) {
        if (await this.fileExistsAsync(assetPath)) {
          return assetPath;
        }
        return null;
      }
      if (await this.fileExistsAsync(assetPath)) {
        return assetPath;
      }
      const assetsPath = `assets/${assetPath}`;
      if (await this.fileExistsAsync(assetsPath)) {
        return assetsPath;
      }
      const noteDir = this.dirname(notePath);
      if (noteDir && noteDir !== ".") {
        const relativePath = this.joinPath(noteDir, assetPath);
        if (await this.fileExistsAsync(relativePath)) {
          return relativePath;
        }
      }
      return null;
    }
    // ============ State ============
    async saveSyncState(state) {
      state.lastSyncedAt = Date.now();
      await saveSyncState(state);
      this.syncState = state;
    }
    // ============ UI Callbacks ============
    onProgress(progress) {
      this.callbacks.onProgress?.(progress);
    }
    async onConflict(conflicts) {
      if (this.callbacks.onConflict) {
        return this.callbacks.onConflict(conflicts);
      }
      return conflicts.map(() => "keep_local");
    }
    async onAssetConflict(conflicts) {
      if (this.callbacks.onAssetConflict) {
        return this.callbacks.onAssetConflict(conflicts);
      }
      return conflicts.map(() => "keep_local");
    }
    async onServerDeleted(paths) {
      if (this.callbacks.onServerDeleted) {
        return this.callbacks.onServerDeleted(paths);
      }
      return false;
    }
    async confirmPush(paths) {
      if (this.callbacks.confirmPush) {
        return this.callbacks.confirmPush(paths);
      }
      return true;
    }
    // ============ Private Helpers ============
    async graphqlRequest(query, variables) {
      const response = await fetch(this.options.apiUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": this.options.apiKey
        },
        body: JSON.stringify({ query, variables })
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const json = await response.json();
      if (json.errors?.length) {
        throw new Error(`GraphQL Error: ${json.errors[0].message}`);
      }
      if (!json.data) {
        throw new Error("GraphQL response missing data");
      }
      return json.data;
    }
    async getFileHandle(path) {
      const cached = this.fileCache.get(path);
      if (cached) return cached;
      const parts = path.split("/");
      const fileName = parts.pop();
      const dir = await this.navigateToDir(parts.join("/"), false);
      if (!dir) {
        throw new Error(`Directory not found for: ${path}`);
      }
      const handle = await dir.getFileHandle(fileName);
      this.fileCache.set(path, handle);
      return handle;
    }
    async getOrCreateFileHandle(path) {
      const parts = path.split("/");
      const fileName = parts.pop();
      const dir = await this.navigateToDir(parts.join("/"), true);
      if (!dir) {
        throw new Error(`Could not create directory for: ${path}`);
      }
      const handle = await dir.getFileHandle(fileName, { create: true });
      this.fileCache.set(path, handle);
      return handle;
    }
    async navigateToDir(path, create) {
      if (!this.directoryHandle) return null;
      if (!path || path === ".") return this.directoryHandle;
      let current = this.directoryHandle;
      const parts = path.split("/").filter((p2) => p2 && p2 !== ".");
      for (const part of parts) {
        try {
          current = await current.getDirectoryHandle(part, { create });
        } catch {
          if (create) throw new Error(`Could not create directory: ${part}`);
          return null;
        }
      }
      return current;
    }
    async fileExistsAsync(path) {
      if (this.existsCache.has(path)) {
        return this.existsCache.get(path);
      }
      try {
        await this.getFileHandle(path);
        this.existsCache.set(path, true);
        return true;
      } catch {
        this.existsCache.set(path, false);
        return false;
      }
    }
    arrayBufferToBase64Url(buffer) {
      const bytes = new Uint8Array(buffer);
      let binary = "";
      for (let i2 = 0; i2 < bytes.byteLength; i2++) {
        binary += String.fromCharCode(bytes[i2]);
      }
      const base64 = btoa(binary);
      return base64.replace(/\+/g, "-").replace(/\//g, "_");
    }
    arrayBufferToHex(buffer) {
      const bytes = new Uint8Array(buffer);
      let hex = "";
      for (let i2 = 0; i2 < bytes.byteLength; i2++) {
        hex += bytes[i2].toString(16).padStart(2, "0");
      }
      return hex;
    }
    dirname(path) {
      const lastSlash = path.lastIndexOf("/");
      if (lastSlash === -1) return "";
      return path.slice(0, lastSlash);
    }
    joinPath(...parts) {
      return parts.filter((p2) => p2).join("/");
    }
    hasPublishFieldSync(path) {
      return true;
    }
    hasPublishFieldInContent(content) {
      if (!this.options.publishField) return true;
      if (!content.startsWith("---")) return false;
      const endIndex = content.indexOf("\n---", 3);
      if (endIndex === -1) return false;
      const frontmatterText = content.slice(4, endIndex);
      const fields = this.options.publishField.split(",").map((f2) => f2.trim()).filter((f2) => f2);
      for (const field of fields) {
        const regex = new RegExp(`^${field}\\s*:\\s*(.+)$`, "m");
        const match = frontmatterText.match(regex);
        if (match) {
          const value = match[1].trim().toLowerCase();
          if (value === "true" || value === "yes" || value === "1" || value === '"true"' || value === "'true'") {
            return true;
          }
        }
      }
      return false;
    }
  };
  return __toCommonJS(index_exports);
})();
