#!/usr/bin/env node
import{fileURLToPath as __fU2P}from"node:url";import{dirname as __dn}from"node:path";const __filename=__fU2P(import.meta.url);const __dirname=__dn(__filename);
var __defProp = Object.defineProperty;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });
var __esm = (fn, res) => function __init() {
  return fn && (res = (0, fn[__getOwnPropNames(fn)[0]])(fn = 0)), res;
};
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};

// node_modules/readdirp/esm/index.js
import { stat, lstat, readdir, realpath } from "node:fs/promises";
import { Readable } from "node:stream";
import { resolve as presolve, relative as prelative, join as pjoin, sep as psep } from "node:path";
function readdirp(root, options = {}) {
  let type = options.entryType || options.type;
  if (type === "both")
    type = EntryTypes.FILE_DIR_TYPE;
  if (type)
    options.type = type;
  if (!root) {
    throw new Error("readdirp: root argument is required. Usage: readdirp(root, options)");
  } else if (typeof root !== "string") {
    throw new TypeError("readdirp: root argument must be a string. Usage: readdirp(root, options)");
  } else if (type && !ALL_TYPES.includes(type)) {
    throw new Error(`readdirp: Invalid type passed. Use one of ${ALL_TYPES.join(", ")}`);
  }
  options.root = root;
  return new ReaddirpStream(options);
}
var EntryTypes, defaultOptions, RECURSIVE_ERROR_CODE, NORMAL_FLOW_ERRORS, ALL_TYPES, DIR_TYPES, FILE_TYPES, isNormalFlowError, wantBigintFsStats, emptyFn, normalizeFilter, ReaddirpStream;
var init_esm = __esm({
  "node_modules/readdirp/esm/index.js"() {
    EntryTypes = {
      FILE_TYPE: "files",
      DIR_TYPE: "directories",
      FILE_DIR_TYPE: "files_directories",
      EVERYTHING_TYPE: "all"
    };
    defaultOptions = {
      root: ".",
      fileFilter: /* @__PURE__ */ __name((_entryInfo) => true, "fileFilter"),
      directoryFilter: /* @__PURE__ */ __name((_entryInfo) => true, "directoryFilter"),
      type: EntryTypes.FILE_TYPE,
      lstat: false,
      depth: 2147483648,
      alwaysStat: false,
      highWaterMark: 4096
    };
    Object.freeze(defaultOptions);
    RECURSIVE_ERROR_CODE = "READDIRP_RECURSIVE_ERROR";
    NORMAL_FLOW_ERRORS = /* @__PURE__ */ new Set(["ENOENT", "EPERM", "EACCES", "ELOOP", RECURSIVE_ERROR_CODE]);
    ALL_TYPES = [
      EntryTypes.DIR_TYPE,
      EntryTypes.EVERYTHING_TYPE,
      EntryTypes.FILE_DIR_TYPE,
      EntryTypes.FILE_TYPE
    ];
    DIR_TYPES = /* @__PURE__ */ new Set([
      EntryTypes.DIR_TYPE,
      EntryTypes.EVERYTHING_TYPE,
      EntryTypes.FILE_DIR_TYPE
    ]);
    FILE_TYPES = /* @__PURE__ */ new Set([
      EntryTypes.EVERYTHING_TYPE,
      EntryTypes.FILE_DIR_TYPE,
      EntryTypes.FILE_TYPE
    ]);
    isNormalFlowError = /* @__PURE__ */ __name((error) => NORMAL_FLOW_ERRORS.has(error.code), "isNormalFlowError");
    wantBigintFsStats = process.platform === "win32";
    emptyFn = /* @__PURE__ */ __name((_entryInfo) => true, "emptyFn");
    normalizeFilter = /* @__PURE__ */ __name((filter) => {
      if (filter === void 0)
        return emptyFn;
      if (typeof filter === "function")
        return filter;
      if (typeof filter === "string") {
        const fl = filter.trim();
        return (entry) => entry.basename === fl;
      }
      if (Array.isArray(filter)) {
        const trItems = filter.map((item) => item.trim());
        return (entry) => trItems.some((f) => entry.basename === f);
      }
      return emptyFn;
    }, "normalizeFilter");
    ReaddirpStream = class extends Readable {
      static {
        __name(this, "ReaddirpStream");
      }
      constructor(options = {}) {
        super({
          objectMode: true,
          autoDestroy: true,
          highWaterMark: options.highWaterMark
        });
        const opts = { ...defaultOptions, ...options };
        const { root, type } = opts;
        this._fileFilter = normalizeFilter(opts.fileFilter);
        this._directoryFilter = normalizeFilter(opts.directoryFilter);
        const statMethod = opts.lstat ? lstat : stat;
        if (wantBigintFsStats) {
          this._stat = (path5) => statMethod(path5, { bigint: true });
        } else {
          this._stat = statMethod;
        }
        this._maxDepth = opts.depth ?? defaultOptions.depth;
        this._wantsDir = type ? DIR_TYPES.has(type) : false;
        this._wantsFile = type ? FILE_TYPES.has(type) : false;
        this._wantsEverything = type === EntryTypes.EVERYTHING_TYPE;
        this._root = presolve(root);
        this._isDirent = !opts.alwaysStat;
        this._statsProp = this._isDirent ? "dirent" : "stats";
        this._rdOptions = { encoding: "utf8", withFileTypes: this._isDirent };
        this.parents = [this._exploreDir(root, 1)];
        this.reading = false;
        this.parent = void 0;
      }
      async _read(batch) {
        if (this.reading)
          return;
        this.reading = true;
        try {
          while (!this.destroyed && batch > 0) {
            const par = this.parent;
            const fil = par && par.files;
            if (fil && fil.length > 0) {
              const { path: path5, depth } = par;
              const slice = fil.splice(0, batch).map((dirent) => this._formatEntry(dirent, path5));
              const awaited = await Promise.all(slice);
              for (const entry of awaited) {
                if (!entry)
                  continue;
                if (this.destroyed)
                  return;
                const entryType = await this._getEntryType(entry);
                if (entryType === "directory" && this._directoryFilter(entry)) {
                  if (depth <= this._maxDepth) {
                    this.parents.push(this._exploreDir(entry.fullPath, depth + 1));
                  }
                  if (this._wantsDir) {
                    this.push(entry);
                    batch--;
                  }
                } else if ((entryType === "file" || this._includeAsFile(entry)) && this._fileFilter(entry)) {
                  if (this._wantsFile) {
                    this.push(entry);
                    batch--;
                  }
                }
              }
            } else {
              const parent = this.parents.pop();
              if (!parent) {
                this.push(null);
                break;
              }
              this.parent = await parent;
              if (this.destroyed)
                return;
            }
          }
        } catch (error) {
          this.destroy(error);
        } finally {
          this.reading = false;
        }
      }
      async _exploreDir(path5, depth) {
        let files;
        try {
          files = await readdir(path5, this._rdOptions);
        } catch (error) {
          this._onError(error);
        }
        return { files, depth, path: path5 };
      }
      async _formatEntry(dirent, path5) {
        let entry;
        const basename3 = this._isDirent ? dirent.name : dirent;
        try {
          const fullPath = presolve(pjoin(path5, basename3));
          entry = { path: prelative(this._root, fullPath), fullPath, basename: basename3 };
          entry[this._statsProp] = this._isDirent ? dirent : await this._stat(fullPath);
        } catch (err) {
          this._onError(err);
          return;
        }
        return entry;
      }
      _onError(err) {
        if (isNormalFlowError(err) && !this.destroyed) {
          this.emit("warn", err);
        } else {
          this.destroy(err);
        }
      }
      async _getEntryType(entry) {
        if (!entry && this._statsProp in entry) {
          return "";
        }
        const stats = entry[this._statsProp];
        if (stats.isFile())
          return "file";
        if (stats.isDirectory())
          return "directory";
        if (stats && stats.isSymbolicLink()) {
          const full = entry.fullPath;
          try {
            const entryRealPath = await realpath(full);
            const entryRealPathStats = await lstat(entryRealPath);
            if (entryRealPathStats.isFile()) {
              return "file";
            }
            if (entryRealPathStats.isDirectory()) {
              const len = entryRealPath.length;
              if (full.startsWith(entryRealPath) && full.substr(len, 1) === psep) {
                const recursiveError = new Error(`Circular symlink detected: "${full}" points to "${entryRealPath}"`);
                recursiveError.code = RECURSIVE_ERROR_CODE;
                return this._onError(recursiveError);
              }
              return "directory";
            }
          } catch (error) {
            this._onError(error);
            return "";
          }
        }
      }
      _includeAsFile(entry) {
        const stats = entry && entry[this._statsProp];
        return stats && this._wantsEverything && !stats.isDirectory();
      }
    };
    __name(readdirp, "readdirp");
  }
});

// node_modules/chokidar/esm/handler.js
import { watchFile, unwatchFile, watch as fs_watch } from "fs";
import { open, stat as stat2, lstat as lstat2, realpath as fsrealpath } from "fs/promises";
import * as sysPath from "path";
import { type as osType } from "os";
function createFsWatchInstance(path5, options, listener, errHandler, emitRaw) {
  const handleEvent = /* @__PURE__ */ __name((rawEvent, evPath) => {
    listener(path5);
    emitRaw(rawEvent, evPath, { watchedPath: path5 });
    if (evPath && path5 !== evPath) {
      fsWatchBroadcast(sysPath.resolve(path5, evPath), KEY_LISTENERS, sysPath.join(path5, evPath));
    }
  }, "handleEvent");
  try {
    return fs_watch(path5, {
      persistent: options.persistent
    }, handleEvent);
  } catch (error) {
    errHandler(error);
    return void 0;
  }
}
var STR_DATA, STR_END, STR_CLOSE, EMPTY_FN, pl, isWindows, isMacos, isLinux, isFreeBSD, isIBMi, EVENTS, EV, THROTTLE_MODE_WATCH, statMethods, KEY_LISTENERS, KEY_ERR, KEY_RAW, HANDLER_KEYS, binaryExtensions, isBinaryPath, foreach, addAndConvert, clearItem, delFromSet, isEmptySet, FsWatchInstances, fsWatchBroadcast, setFsWatchListener, FsWatchFileInstances, setFsWatchFileListener, NodeFsHandler;
var init_handler = __esm({
  "node_modules/chokidar/esm/handler.js"() {
    STR_DATA = "data";
    STR_END = "end";
    STR_CLOSE = "close";
    EMPTY_FN = /* @__PURE__ */ __name(() => {
    }, "EMPTY_FN");
    pl = process.platform;
    isWindows = pl === "win32";
    isMacos = pl === "darwin";
    isLinux = pl === "linux";
    isFreeBSD = pl === "freebsd";
    isIBMi = osType() === "OS400";
    EVENTS = {
      ALL: "all",
      READY: "ready",
      ADD: "add",
      CHANGE: "change",
      ADD_DIR: "addDir",
      UNLINK: "unlink",
      UNLINK_DIR: "unlinkDir",
      RAW: "raw",
      ERROR: "error"
    };
    EV = EVENTS;
    THROTTLE_MODE_WATCH = "watch";
    statMethods = { lstat: lstat2, stat: stat2 };
    KEY_LISTENERS = "listeners";
    KEY_ERR = "errHandlers";
    KEY_RAW = "rawEmitters";
    HANDLER_KEYS = [KEY_LISTENERS, KEY_ERR, KEY_RAW];
    binaryExtensions = /* @__PURE__ */ new Set([
      "3dm",
      "3ds",
      "3g2",
      "3gp",
      "7z",
      "a",
      "aac",
      "adp",
      "afdesign",
      "afphoto",
      "afpub",
      "ai",
      "aif",
      "aiff",
      "alz",
      "ape",
      "apk",
      "appimage",
      "ar",
      "arj",
      "asf",
      "au",
      "avi",
      "bak",
      "baml",
      "bh",
      "bin",
      "bk",
      "bmp",
      "btif",
      "bz2",
      "bzip2",
      "cab",
      "caf",
      "cgm",
      "class",
      "cmx",
      "cpio",
      "cr2",
      "cur",
      "dat",
      "dcm",
      "deb",
      "dex",
      "djvu",
      "dll",
      "dmg",
      "dng",
      "doc",
      "docm",
      "docx",
      "dot",
      "dotm",
      "dra",
      "DS_Store",
      "dsk",
      "dts",
      "dtshd",
      "dvb",
      "dwg",
      "dxf",
      "ecelp4800",
      "ecelp7470",
      "ecelp9600",
      "egg",
      "eol",
      "eot",
      "epub",
      "exe",
      "f4v",
      "fbs",
      "fh",
      "fla",
      "flac",
      "flatpak",
      "fli",
      "flv",
      "fpx",
      "fst",
      "fvt",
      "g3",
      "gh",
      "gif",
      "graffle",
      "gz",
      "gzip",
      "h261",
      "h263",
      "h264",
      "icns",
      "ico",
      "ief",
      "img",
      "ipa",
      "iso",
      "jar",
      "jpeg",
      "jpg",
      "jpgv",
      "jpm",
      "jxr",
      "key",
      "ktx",
      "lha",
      "lib",
      "lvp",
      "lz",
      "lzh",
      "lzma",
      "lzo",
      "m3u",
      "m4a",
      "m4v",
      "mar",
      "mdi",
      "mht",
      "mid",
      "midi",
      "mj2",
      "mka",
      "mkv",
      "mmr",
      "mng",
      "mobi",
      "mov",
      "movie",
      "mp3",
      "mp4",
      "mp4a",
      "mpeg",
      "mpg",
      "mpga",
      "mxu",
      "nef",
      "npx",
      "numbers",
      "nupkg",
      "o",
      "odp",
      "ods",
      "odt",
      "oga",
      "ogg",
      "ogv",
      "otf",
      "ott",
      "pages",
      "pbm",
      "pcx",
      "pdb",
      "pdf",
      "pea",
      "pgm",
      "pic",
      "png",
      "pnm",
      "pot",
      "potm",
      "potx",
      "ppa",
      "ppam",
      "ppm",
      "pps",
      "ppsm",
      "ppsx",
      "ppt",
      "pptm",
      "pptx",
      "psd",
      "pya",
      "pyc",
      "pyo",
      "pyv",
      "qt",
      "rar",
      "ras",
      "raw",
      "resources",
      "rgb",
      "rip",
      "rlc",
      "rmf",
      "rmvb",
      "rpm",
      "rtf",
      "rz",
      "s3m",
      "s7z",
      "scpt",
      "sgi",
      "shar",
      "snap",
      "sil",
      "sketch",
      "slk",
      "smv",
      "snk",
      "so",
      "stl",
      "suo",
      "sub",
      "swf",
      "tar",
      "tbz",
      "tbz2",
      "tga",
      "tgz",
      "thmx",
      "tif",
      "tiff",
      "tlz",
      "ttc",
      "ttf",
      "txz",
      "udf",
      "uvh",
      "uvi",
      "uvm",
      "uvp",
      "uvs",
      "uvu",
      "viv",
      "vob",
      "war",
      "wav",
      "wax",
      "wbmp",
      "wdp",
      "weba",
      "webm",
      "webp",
      "whl",
      "wim",
      "wm",
      "wma",
      "wmv",
      "wmx",
      "woff",
      "woff2",
      "wrm",
      "wvx",
      "xbm",
      "xif",
      "xla",
      "xlam",
      "xls",
      "xlsb",
      "xlsm",
      "xlsx",
      "xlt",
      "xltm",
      "xltx",
      "xm",
      "xmind",
      "xpi",
      "xpm",
      "xwd",
      "xz",
      "z",
      "zip",
      "zipx"
    ]);
    isBinaryPath = /* @__PURE__ */ __name((filePath) => binaryExtensions.has(sysPath.extname(filePath).slice(1).toLowerCase()), "isBinaryPath");
    foreach = /* @__PURE__ */ __name((val, fn) => {
      if (val instanceof Set) {
        val.forEach(fn);
      } else {
        fn(val);
      }
    }, "foreach");
    addAndConvert = /* @__PURE__ */ __name((main2, prop, item) => {
      let container = main2[prop];
      if (!(container instanceof Set)) {
        main2[prop] = container = /* @__PURE__ */ new Set([container]);
      }
      container.add(item);
    }, "addAndConvert");
    clearItem = /* @__PURE__ */ __name((cont) => (key) => {
      const set = cont[key];
      if (set instanceof Set) {
        set.clear();
      } else {
        delete cont[key];
      }
    }, "clearItem");
    delFromSet = /* @__PURE__ */ __name((main2, prop, item) => {
      const container = main2[prop];
      if (container instanceof Set) {
        container.delete(item);
      } else if (container === item) {
        delete main2[prop];
      }
    }, "delFromSet");
    isEmptySet = /* @__PURE__ */ __name((val) => val instanceof Set ? val.size === 0 : !val, "isEmptySet");
    FsWatchInstances = /* @__PURE__ */ new Map();
    __name(createFsWatchInstance, "createFsWatchInstance");
    fsWatchBroadcast = /* @__PURE__ */ __name((fullPath, listenerType, val1, val2, val3) => {
      const cont = FsWatchInstances.get(fullPath);
      if (!cont)
        return;
      foreach(cont[listenerType], (listener) => {
        listener(val1, val2, val3);
      });
    }, "fsWatchBroadcast");
    setFsWatchListener = /* @__PURE__ */ __name((path5, fullPath, options, handlers) => {
      const { listener, errHandler, rawEmitter } = handlers;
      let cont = FsWatchInstances.get(fullPath);
      let watcher;
      if (!options.persistent) {
        watcher = createFsWatchInstance(path5, options, listener, errHandler, rawEmitter);
        if (!watcher)
          return;
        return watcher.close.bind(watcher);
      }
      if (cont) {
        addAndConvert(cont, KEY_LISTENERS, listener);
        addAndConvert(cont, KEY_ERR, errHandler);
        addAndConvert(cont, KEY_RAW, rawEmitter);
      } else {
        watcher = createFsWatchInstance(
          path5,
          options,
          fsWatchBroadcast.bind(null, fullPath, KEY_LISTENERS),
          errHandler,
          // no need to use broadcast here
          fsWatchBroadcast.bind(null, fullPath, KEY_RAW)
        );
        if (!watcher)
          return;
        watcher.on(EV.ERROR, async (error) => {
          const broadcastErr = fsWatchBroadcast.bind(null, fullPath, KEY_ERR);
          if (cont)
            cont.watcherUnusable = true;
          if (isWindows && error.code === "EPERM") {
            try {
              const fd = await open(path5, "r");
              await fd.close();
              broadcastErr(error);
            } catch (err) {
            }
          } else {
            broadcastErr(error);
          }
        });
        cont = {
          listeners: listener,
          errHandlers: errHandler,
          rawEmitters: rawEmitter,
          watcher
        };
        FsWatchInstances.set(fullPath, cont);
      }
      return () => {
        delFromSet(cont, KEY_LISTENERS, listener);
        delFromSet(cont, KEY_ERR, errHandler);
        delFromSet(cont, KEY_RAW, rawEmitter);
        if (isEmptySet(cont.listeners)) {
          cont.watcher.close();
          FsWatchInstances.delete(fullPath);
          HANDLER_KEYS.forEach(clearItem(cont));
          cont.watcher = void 0;
          Object.freeze(cont);
        }
      };
    }, "setFsWatchListener");
    FsWatchFileInstances = /* @__PURE__ */ new Map();
    setFsWatchFileListener = /* @__PURE__ */ __name((path5, fullPath, options, handlers) => {
      const { listener, rawEmitter } = handlers;
      let cont = FsWatchFileInstances.get(fullPath);
      const copts = cont && cont.options;
      if (copts && (copts.persistent < options.persistent || copts.interval > options.interval)) {
        unwatchFile(fullPath);
        cont = void 0;
      }
      if (cont) {
        addAndConvert(cont, KEY_LISTENERS, listener);
        addAndConvert(cont, KEY_RAW, rawEmitter);
      } else {
        cont = {
          listeners: listener,
          rawEmitters: rawEmitter,
          options,
          watcher: watchFile(fullPath, options, (curr, prev) => {
            foreach(cont.rawEmitters, (rawEmitter2) => {
              rawEmitter2(EV.CHANGE, fullPath, { curr, prev });
            });
            const currmtime = curr.mtimeMs;
            if (curr.size !== prev.size || currmtime > prev.mtimeMs || currmtime === 0) {
              foreach(cont.listeners, (listener2) => listener2(path5, curr));
            }
          })
        };
        FsWatchFileInstances.set(fullPath, cont);
      }
      return () => {
        delFromSet(cont, KEY_LISTENERS, listener);
        delFromSet(cont, KEY_RAW, rawEmitter);
        if (isEmptySet(cont.listeners)) {
          FsWatchFileInstances.delete(fullPath);
          unwatchFile(fullPath);
          cont.options = cont.watcher = void 0;
          Object.freeze(cont);
        }
      };
    }, "setFsWatchFileListener");
    NodeFsHandler = class {
      static {
        __name(this, "NodeFsHandler");
      }
      constructor(fsW) {
        this.fsw = fsW;
        this._boundHandleError = (error) => fsW._handleError(error);
      }
      /**
       * Watch file for changes with fs_watchFile or fs_watch.
       * @param path to file or dir
       * @param listener on fs change
       * @returns closer for the watcher instance
       */
      _watchWithNodeFs(path5, listener) {
        const opts = this.fsw.options;
        const directory = sysPath.dirname(path5);
        const basename3 = sysPath.basename(path5);
        const parent = this.fsw._getWatchedDir(directory);
        parent.add(basename3);
        const absolutePath = sysPath.resolve(path5);
        const options = {
          persistent: opts.persistent
        };
        if (!listener)
          listener = EMPTY_FN;
        let closer;
        if (opts.usePolling) {
          const enableBin = opts.interval !== opts.binaryInterval;
          options.interval = enableBin && isBinaryPath(basename3) ? opts.binaryInterval : opts.interval;
          closer = setFsWatchFileListener(path5, absolutePath, options, {
            listener,
            rawEmitter: this.fsw._emitRaw
          });
        } else {
          closer = setFsWatchListener(path5, absolutePath, options, {
            listener,
            errHandler: this._boundHandleError,
            rawEmitter: this.fsw._emitRaw
          });
        }
        return closer;
      }
      /**
       * Watch a file and emit add event if warranted.
       * @returns closer for the watcher instance
       */
      _handleFile(file, stats, initialAdd) {
        if (this.fsw.closed) {
          return;
        }
        const dirname4 = sysPath.dirname(file);
        const basename3 = sysPath.basename(file);
        const parent = this.fsw._getWatchedDir(dirname4);
        let prevStats = stats;
        if (parent.has(basename3))
          return;
        const listener = /* @__PURE__ */ __name(async (path5, newStats) => {
          if (!this.fsw._throttle(THROTTLE_MODE_WATCH, file, 5))
            return;
          if (!newStats || newStats.mtimeMs === 0) {
            try {
              const newStats2 = await stat2(file);
              if (this.fsw.closed)
                return;
              const at = newStats2.atimeMs;
              const mt = newStats2.mtimeMs;
              if (!at || at <= mt || mt !== prevStats.mtimeMs) {
                this.fsw._emit(EV.CHANGE, file, newStats2);
              }
              if ((isMacos || isLinux || isFreeBSD) && prevStats.ino !== newStats2.ino) {
                this.fsw._closeFile(path5);
                prevStats = newStats2;
                const closer2 = this._watchWithNodeFs(file, listener);
                if (closer2)
                  this.fsw._addPathCloser(path5, closer2);
              } else {
                prevStats = newStats2;
              }
            } catch (error) {
              this.fsw._remove(dirname4, basename3);
            }
          } else if (parent.has(basename3)) {
            const at = newStats.atimeMs;
            const mt = newStats.mtimeMs;
            if (!at || at <= mt || mt !== prevStats.mtimeMs) {
              this.fsw._emit(EV.CHANGE, file, newStats);
            }
            prevStats = newStats;
          }
        }, "listener");
        const closer = this._watchWithNodeFs(file, listener);
        if (!(initialAdd && this.fsw.options.ignoreInitial) && this.fsw._isntIgnored(file)) {
          if (!this.fsw._throttle(EV.ADD, file, 0))
            return;
          this.fsw._emit(EV.ADD, file, stats);
        }
        return closer;
      }
      /**
       * Handle symlinks encountered while reading a dir.
       * @param entry returned by readdirp
       * @param directory path of dir being read
       * @param path of this item
       * @param item basename of this item
       * @returns true if no more processing is needed for this entry.
       */
      async _handleSymlink(entry, directory, path5, item) {
        if (this.fsw.closed) {
          return;
        }
        const full = entry.fullPath;
        const dir = this.fsw._getWatchedDir(directory);
        if (!this.fsw.options.followSymlinks) {
          this.fsw._incrReadyCount();
          let linkPath;
          try {
            linkPath = await fsrealpath(path5);
          } catch (e) {
            this.fsw._emitReady();
            return true;
          }
          if (this.fsw.closed)
            return;
          if (dir.has(item)) {
            if (this.fsw._symlinkPaths.get(full) !== linkPath) {
              this.fsw._symlinkPaths.set(full, linkPath);
              this.fsw._emit(EV.CHANGE, path5, entry.stats);
            }
          } else {
            dir.add(item);
            this.fsw._symlinkPaths.set(full, linkPath);
            this.fsw._emit(EV.ADD, path5, entry.stats);
          }
          this.fsw._emitReady();
          return true;
        }
        if (this.fsw._symlinkPaths.has(full)) {
          return true;
        }
        this.fsw._symlinkPaths.set(full, true);
      }
      _handleRead(directory, initialAdd, wh, target, dir, depth, throttler) {
        directory = sysPath.join(directory, "");
        throttler = this.fsw._throttle("readdir", directory, 1e3);
        if (!throttler)
          return;
        const previous = this.fsw._getWatchedDir(wh.path);
        const current = /* @__PURE__ */ new Set();
        let stream = this.fsw._readdirp(directory, {
          fileFilter: /* @__PURE__ */ __name((entry) => wh.filterPath(entry), "fileFilter"),
          directoryFilter: /* @__PURE__ */ __name((entry) => wh.filterDir(entry), "directoryFilter")
        });
        if (!stream)
          return;
        stream.on(STR_DATA, async (entry) => {
          if (this.fsw.closed) {
            stream = void 0;
            return;
          }
          const item = entry.path;
          let path5 = sysPath.join(directory, item);
          current.add(item);
          if (entry.stats.isSymbolicLink() && await this._handleSymlink(entry, directory, path5, item)) {
            return;
          }
          if (this.fsw.closed) {
            stream = void 0;
            return;
          }
          if (item === target || !target && !previous.has(item)) {
            this.fsw._incrReadyCount();
            path5 = sysPath.join(dir, sysPath.relative(dir, path5));
            this._addToNodeFs(path5, initialAdd, wh, depth + 1);
          }
        }).on(EV.ERROR, this._boundHandleError);
        return new Promise((resolve5, reject) => {
          if (!stream)
            return reject();
          stream.once(STR_END, () => {
            if (this.fsw.closed) {
              stream = void 0;
              return;
            }
            const wasThrottled = throttler ? throttler.clear() : false;
            resolve5(void 0);
            previous.getChildren().filter((item) => {
              return item !== directory && !current.has(item);
            }).forEach((item) => {
              this.fsw._remove(directory, item);
            });
            stream = void 0;
            if (wasThrottled)
              this._handleRead(directory, false, wh, target, dir, depth, throttler);
          });
        });
      }
      /**
       * Read directory to add / remove files from `@watched` list and re-read it on change.
       * @param dir fs path
       * @param stats
       * @param initialAdd
       * @param depth relative to user-supplied path
       * @param target child path targeted for watch
       * @param wh Common watch helpers for this path
       * @param realpath
       * @returns closer for the watcher instance.
       */
      async _handleDir(dir, stats, initialAdd, depth, target, wh, realpath2) {
        const parentDir = this.fsw._getWatchedDir(sysPath.dirname(dir));
        const tracked = parentDir.has(sysPath.basename(dir));
        if (!(initialAdd && this.fsw.options.ignoreInitial) && !target && !tracked) {
          this.fsw._emit(EV.ADD_DIR, dir, stats);
        }
        parentDir.add(sysPath.basename(dir));
        this.fsw._getWatchedDir(dir);
        let throttler;
        let closer;
        const oDepth = this.fsw.options.depth;
        if ((oDepth == null || depth <= oDepth) && !this.fsw._symlinkPaths.has(realpath2)) {
          if (!target) {
            await this._handleRead(dir, initialAdd, wh, target, dir, depth, throttler);
            if (this.fsw.closed)
              return;
          }
          closer = this._watchWithNodeFs(dir, (dirPath, stats2) => {
            if (stats2 && stats2.mtimeMs === 0)
              return;
            this._handleRead(dirPath, false, wh, target, dir, depth, throttler);
          });
        }
        return closer;
      }
      /**
       * Handle added file, directory, or glob pattern.
       * Delegates call to _handleFile / _handleDir after checks.
       * @param path to file or ir
       * @param initialAdd was the file added at watch instantiation?
       * @param priorWh depth relative to user-supplied path
       * @param depth Child path actually targeted for watch
       * @param target Child path actually targeted for watch
       */
      async _addToNodeFs(path5, initialAdd, priorWh, depth, target) {
        const ready = this.fsw._emitReady;
        if (this.fsw._isIgnored(path5) || this.fsw.closed) {
          ready();
          return false;
        }
        const wh = this.fsw._getWatchHelpers(path5);
        if (priorWh) {
          wh.filterPath = (entry) => priorWh.filterPath(entry);
          wh.filterDir = (entry) => priorWh.filterDir(entry);
        }
        try {
          const stats = await statMethods[wh.statMethod](wh.watchPath);
          if (this.fsw.closed)
            return;
          if (this.fsw._isIgnored(wh.watchPath, stats)) {
            ready();
            return false;
          }
          const follow = this.fsw.options.followSymlinks;
          let closer;
          if (stats.isDirectory()) {
            const absPath = sysPath.resolve(path5);
            const targetPath = follow ? await fsrealpath(path5) : path5;
            if (this.fsw.closed)
              return;
            closer = await this._handleDir(wh.watchPath, stats, initialAdd, depth, target, wh, targetPath);
            if (this.fsw.closed)
              return;
            if (absPath !== targetPath && targetPath !== void 0) {
              this.fsw._symlinkPaths.set(absPath, targetPath);
            }
          } else if (stats.isSymbolicLink()) {
            const targetPath = follow ? await fsrealpath(path5) : path5;
            if (this.fsw.closed)
              return;
            const parent = sysPath.dirname(wh.watchPath);
            this.fsw._getWatchedDir(parent).add(wh.watchPath);
            this.fsw._emit(EV.ADD, wh.watchPath, stats);
            closer = await this._handleDir(parent, stats, initialAdd, depth, path5, wh, targetPath);
            if (this.fsw.closed)
              return;
            if (targetPath !== void 0) {
              this.fsw._symlinkPaths.set(sysPath.resolve(path5), targetPath);
            }
          } else {
            closer = this._handleFile(wh.watchPath, stats, initialAdd);
          }
          ready();
          if (closer)
            this.fsw._addPathCloser(path5, closer);
          return false;
        } catch (error) {
          if (this.fsw._handleError(error)) {
            ready();
            return path5;
          }
        }
      }
    };
  }
});

// node_modules/chokidar/esm/index.js
var esm_exports = {};
__export(esm_exports, {
  FSWatcher: () => FSWatcher,
  WatchHelper: () => WatchHelper,
  default: () => esm_default,
  watch: () => watch
});
import { stat as statcb } from "fs";
import { stat as stat3, readdir as readdir2 } from "fs/promises";
import { EventEmitter } from "events";
import * as sysPath2 from "path";
function arrify(item) {
  return Array.isArray(item) ? item : [item];
}
function createPattern(matcher) {
  if (typeof matcher === "function")
    return matcher;
  if (typeof matcher === "string")
    return (string) => matcher === string;
  if (matcher instanceof RegExp)
    return (string) => matcher.test(string);
  if (typeof matcher === "object" && matcher !== null) {
    return (string) => {
      if (matcher.path === string)
        return true;
      if (matcher.recursive) {
        const relative5 = sysPath2.relative(matcher.path, string);
        if (!relative5) {
          return false;
        }
        return !relative5.startsWith("..") && !sysPath2.isAbsolute(relative5);
      }
      return false;
    };
  }
  return () => false;
}
function normalizePath(path5) {
  if (typeof path5 !== "string")
    throw new Error("string expected");
  path5 = sysPath2.normalize(path5);
  path5 = path5.replace(/\\/g, "/");
  let prepend = false;
  if (path5.startsWith("//"))
    prepend = true;
  const DOUBLE_SLASH_RE2 = /\/\//;
  while (path5.match(DOUBLE_SLASH_RE2))
    path5 = path5.replace(DOUBLE_SLASH_RE2, "/");
  if (prepend)
    path5 = "/" + path5;
  return path5;
}
function matchPatterns(patterns, testString, stats) {
  const path5 = normalizePath(testString);
  for (let index = 0; index < patterns.length; index++) {
    const pattern = patterns[index];
    if (pattern(path5, stats)) {
      return true;
    }
  }
  return false;
}
function anymatch(matchers, testString) {
  if (matchers == null) {
    throw new TypeError("anymatch: specify first argument");
  }
  const matchersArray = arrify(matchers);
  const patterns = matchersArray.map((matcher) => createPattern(matcher));
  if (testString == null) {
    return (testString2, stats) => {
      return matchPatterns(patterns, testString2, stats);
    };
  }
  return matchPatterns(patterns, testString);
}
function watch(paths, options = {}) {
  const watcher = new FSWatcher(options);
  watcher.add(paths);
  return watcher;
}
var SLASH, SLASH_SLASH, ONE_DOT, TWO_DOTS, STRING_TYPE, BACK_SLASH_RE, DOUBLE_SLASH_RE, DOT_RE, REPLACER_RE, isMatcherObject, unifyPaths, toUnix, normalizePathToUnix, normalizeIgnored, getAbsolutePath, EMPTY_SET, DirEntry, STAT_METHOD_F, STAT_METHOD_L, WatchHelper, FSWatcher, esm_default;
var init_esm2 = __esm({
  "node_modules/chokidar/esm/index.js"() {
    init_esm();
    init_handler();
    SLASH = "/";
    SLASH_SLASH = "//";
    ONE_DOT = ".";
    TWO_DOTS = "..";
    STRING_TYPE = "string";
    BACK_SLASH_RE = /\\/g;
    DOUBLE_SLASH_RE = /\/\//;
    DOT_RE = /\..*\.(sw[px])$|~$|\.subl.*\.tmp/;
    REPLACER_RE = /^\.[/\\]/;
    __name(arrify, "arrify");
    isMatcherObject = /* @__PURE__ */ __name((matcher) => typeof matcher === "object" && matcher !== null && !(matcher instanceof RegExp), "isMatcherObject");
    __name(createPattern, "createPattern");
    __name(normalizePath, "normalizePath");
    __name(matchPatterns, "matchPatterns");
    __name(anymatch, "anymatch");
    unifyPaths = /* @__PURE__ */ __name((paths_) => {
      const paths = arrify(paths_).flat();
      if (!paths.every((p) => typeof p === STRING_TYPE)) {
        throw new TypeError(`Non-string provided as watch path: ${paths}`);
      }
      return paths.map(normalizePathToUnix);
    }, "unifyPaths");
    toUnix = /* @__PURE__ */ __name((string) => {
      let str = string.replace(BACK_SLASH_RE, SLASH);
      let prepend = false;
      if (str.startsWith(SLASH_SLASH)) {
        prepend = true;
      }
      while (str.match(DOUBLE_SLASH_RE)) {
        str = str.replace(DOUBLE_SLASH_RE, SLASH);
      }
      if (prepend) {
        str = SLASH + str;
      }
      return str;
    }, "toUnix");
    normalizePathToUnix = /* @__PURE__ */ __name((path5) => toUnix(sysPath2.normalize(toUnix(path5))), "normalizePathToUnix");
    normalizeIgnored = /* @__PURE__ */ __name((cwd = "") => (path5) => {
      if (typeof path5 === "string") {
        return normalizePathToUnix(sysPath2.isAbsolute(path5) ? path5 : sysPath2.join(cwd, path5));
      } else {
        return path5;
      }
    }, "normalizeIgnored");
    getAbsolutePath = /* @__PURE__ */ __name((path5, cwd) => {
      if (sysPath2.isAbsolute(path5)) {
        return path5;
      }
      return sysPath2.join(cwd, path5);
    }, "getAbsolutePath");
    EMPTY_SET = Object.freeze(/* @__PURE__ */ new Set());
    DirEntry = class {
      static {
        __name(this, "DirEntry");
      }
      constructor(dir, removeWatcher) {
        this.path = dir;
        this._removeWatcher = removeWatcher;
        this.items = /* @__PURE__ */ new Set();
      }
      add(item) {
        const { items } = this;
        if (!items)
          return;
        if (item !== ONE_DOT && item !== TWO_DOTS)
          items.add(item);
      }
      async remove(item) {
        const { items } = this;
        if (!items)
          return;
        items.delete(item);
        if (items.size > 0)
          return;
        const dir = this.path;
        try {
          await readdir2(dir);
        } catch (err) {
          if (this._removeWatcher) {
            this._removeWatcher(sysPath2.dirname(dir), sysPath2.basename(dir));
          }
        }
      }
      has(item) {
        const { items } = this;
        if (!items)
          return;
        return items.has(item);
      }
      getChildren() {
        const { items } = this;
        if (!items)
          return [];
        return [...items.values()];
      }
      dispose() {
        this.items.clear();
        this.path = "";
        this._removeWatcher = EMPTY_FN;
        this.items = EMPTY_SET;
        Object.freeze(this);
      }
    };
    STAT_METHOD_F = "stat";
    STAT_METHOD_L = "lstat";
    WatchHelper = class {
      static {
        __name(this, "WatchHelper");
      }
      constructor(path5, follow, fsw) {
        this.fsw = fsw;
        const watchPath = path5;
        this.path = path5 = path5.replace(REPLACER_RE, "");
        this.watchPath = watchPath;
        this.fullWatchPath = sysPath2.resolve(watchPath);
        this.dirParts = [];
        this.dirParts.forEach((parts) => {
          if (parts.length > 1)
            parts.pop();
        });
        this.followSymlinks = follow;
        this.statMethod = follow ? STAT_METHOD_F : STAT_METHOD_L;
      }
      entryPath(entry) {
        return sysPath2.join(this.watchPath, sysPath2.relative(this.watchPath, entry.fullPath));
      }
      filterPath(entry) {
        const { stats } = entry;
        if (stats && stats.isSymbolicLink())
          return this.filterDir(entry);
        const resolvedPath = this.entryPath(entry);
        return this.fsw._isntIgnored(resolvedPath, stats) && this.fsw._hasReadPermissions(stats);
      }
      filterDir(entry) {
        return this.fsw._isntIgnored(this.entryPath(entry), entry.stats);
      }
    };
    FSWatcher = class extends EventEmitter {
      static {
        __name(this, "FSWatcher");
      }
      // Not indenting methods for history sake; for now.
      constructor(_opts = {}) {
        super();
        this.closed = false;
        this._closers = /* @__PURE__ */ new Map();
        this._ignoredPaths = /* @__PURE__ */ new Set();
        this._throttled = /* @__PURE__ */ new Map();
        this._streams = /* @__PURE__ */ new Set();
        this._symlinkPaths = /* @__PURE__ */ new Map();
        this._watched = /* @__PURE__ */ new Map();
        this._pendingWrites = /* @__PURE__ */ new Map();
        this._pendingUnlinks = /* @__PURE__ */ new Map();
        this._readyCount = 0;
        this._readyEmitted = false;
        const awf = _opts.awaitWriteFinish;
        const DEF_AWF = { stabilityThreshold: 2e3, pollInterval: 100 };
        const opts = {
          // Defaults
          persistent: true,
          ignoreInitial: false,
          ignorePermissionErrors: false,
          interval: 100,
          binaryInterval: 300,
          followSymlinks: true,
          usePolling: false,
          // useAsync: false,
          atomic: true,
          // NOTE: overwritten later (depends on usePolling)
          ..._opts,
          // Change format
          ignored: _opts.ignored ? arrify(_opts.ignored) : arrify([]),
          awaitWriteFinish: awf === true ? DEF_AWF : typeof awf === "object" ? { ...DEF_AWF, ...awf } : false
        };
        if (isIBMi)
          opts.usePolling = true;
        if (opts.atomic === void 0)
          opts.atomic = !opts.usePolling;
        const envPoll = process.env.CHOKIDAR_USEPOLLING;
        if (envPoll !== void 0) {
          const envLower = envPoll.toLowerCase();
          if (envLower === "false" || envLower === "0")
            opts.usePolling = false;
          else if (envLower === "true" || envLower === "1")
            opts.usePolling = true;
          else
            opts.usePolling = !!envLower;
        }
        const envInterval = process.env.CHOKIDAR_INTERVAL;
        if (envInterval)
          opts.interval = Number.parseInt(envInterval, 10);
        let readyCalls = 0;
        this._emitReady = () => {
          readyCalls++;
          if (readyCalls >= this._readyCount) {
            this._emitReady = EMPTY_FN;
            this._readyEmitted = true;
            process.nextTick(() => this.emit(EVENTS.READY));
          }
        };
        this._emitRaw = (...args) => this.emit(EVENTS.RAW, ...args);
        this._boundRemove = this._remove.bind(this);
        this.options = opts;
        this._nodeFsHandler = new NodeFsHandler(this);
        Object.freeze(opts);
      }
      _addIgnoredPath(matcher) {
        if (isMatcherObject(matcher)) {
          for (const ignored of this._ignoredPaths) {
            if (isMatcherObject(ignored) && ignored.path === matcher.path && ignored.recursive === matcher.recursive) {
              return;
            }
          }
        }
        this._ignoredPaths.add(matcher);
      }
      _removeIgnoredPath(matcher) {
        this._ignoredPaths.delete(matcher);
        if (typeof matcher === "string") {
          for (const ignored of this._ignoredPaths) {
            if (isMatcherObject(ignored) && ignored.path === matcher) {
              this._ignoredPaths.delete(ignored);
            }
          }
        }
      }
      // Public methods
      /**
       * Adds paths to be watched on an existing FSWatcher instance.
       * @param paths_ file or file list. Other arguments are unused
       */
      add(paths_, _origAdd, _internal) {
        const { cwd } = this.options;
        this.closed = false;
        this._closePromise = void 0;
        let paths = unifyPaths(paths_);
        if (cwd) {
          paths = paths.map((path5) => {
            const absPath = getAbsolutePath(path5, cwd);
            return absPath;
          });
        }
        paths.forEach((path5) => {
          this._removeIgnoredPath(path5);
        });
        this._userIgnored = void 0;
        if (!this._readyCount)
          this._readyCount = 0;
        this._readyCount += paths.length;
        Promise.all(paths.map(async (path5) => {
          const res = await this._nodeFsHandler._addToNodeFs(path5, !_internal, void 0, 0, _origAdd);
          if (res)
            this._emitReady();
          return res;
        })).then((results) => {
          if (this.closed)
            return;
          results.forEach((item) => {
            if (item)
              this.add(sysPath2.dirname(item), sysPath2.basename(_origAdd || item));
          });
        });
        return this;
      }
      /**
       * Close watchers or start ignoring events from specified paths.
       */
      unwatch(paths_) {
        if (this.closed)
          return this;
        const paths = unifyPaths(paths_);
        const { cwd } = this.options;
        paths.forEach((path5) => {
          if (!sysPath2.isAbsolute(path5) && !this._closers.has(path5)) {
            if (cwd)
              path5 = sysPath2.join(cwd, path5);
            path5 = sysPath2.resolve(path5);
          }
          this._closePath(path5);
          this._addIgnoredPath(path5);
          if (this._watched.has(path5)) {
            this._addIgnoredPath({
              path: path5,
              recursive: true
            });
          }
          this._userIgnored = void 0;
        });
        return this;
      }
      /**
       * Close watchers and remove all listeners from watched paths.
       */
      close() {
        if (this._closePromise) {
          return this._closePromise;
        }
        this.closed = true;
        this.removeAllListeners();
        const closers = [];
        this._closers.forEach((closerList) => closerList.forEach((closer) => {
          const promise = closer();
          if (promise instanceof Promise)
            closers.push(promise);
        }));
        this._streams.forEach((stream) => stream.destroy());
        this._userIgnored = void 0;
        this._readyCount = 0;
        this._readyEmitted = false;
        this._watched.forEach((dirent) => dirent.dispose());
        this._closers.clear();
        this._watched.clear();
        this._streams.clear();
        this._symlinkPaths.clear();
        this._throttled.clear();
        this._closePromise = closers.length ? Promise.all(closers).then(() => void 0) : Promise.resolve();
        return this._closePromise;
      }
      /**
       * Expose list of watched paths
       * @returns for chaining
       */
      getWatched() {
        const watchList = {};
        this._watched.forEach((entry, dir) => {
          const key = this.options.cwd ? sysPath2.relative(this.options.cwd, dir) : dir;
          const index = key || ONE_DOT;
          watchList[index] = entry.getChildren().sort();
        });
        return watchList;
      }
      emitWithAll(event, args) {
        this.emit(event, ...args);
        if (event !== EVENTS.ERROR)
          this.emit(EVENTS.ALL, event, ...args);
      }
      // Common helpers
      // --------------
      /**
       * Normalize and emit events.
       * Calling _emit DOES NOT MEAN emit() would be called!
       * @param event Type of event
       * @param path File or directory path
       * @param stats arguments to be passed with event
       * @returns the error if defined, otherwise the value of the FSWatcher instance's `closed` flag
       */
      async _emit(event, path5, stats) {
        if (this.closed)
          return;
        const opts = this.options;
        if (isWindows)
          path5 = sysPath2.normalize(path5);
        if (opts.cwd)
          path5 = sysPath2.relative(opts.cwd, path5);
        const args = [path5];
        if (stats != null)
          args.push(stats);
        const awf = opts.awaitWriteFinish;
        let pw;
        if (awf && (pw = this._pendingWrites.get(path5))) {
          pw.lastChange = /* @__PURE__ */ new Date();
          return this;
        }
        if (opts.atomic) {
          if (event === EVENTS.UNLINK) {
            this._pendingUnlinks.set(path5, [event, ...args]);
            setTimeout(() => {
              this._pendingUnlinks.forEach((entry, path6) => {
                this.emit(...entry);
                this.emit(EVENTS.ALL, ...entry);
                this._pendingUnlinks.delete(path6);
              });
            }, typeof opts.atomic === "number" ? opts.atomic : 100);
            return this;
          }
          if (event === EVENTS.ADD && this._pendingUnlinks.has(path5)) {
            event = EVENTS.CHANGE;
            this._pendingUnlinks.delete(path5);
          }
        }
        if (awf && (event === EVENTS.ADD || event === EVENTS.CHANGE) && this._readyEmitted) {
          const awfEmit = /* @__PURE__ */ __name((err, stats2) => {
            if (err) {
              event = EVENTS.ERROR;
              args[0] = err;
              this.emitWithAll(event, args);
            } else if (stats2) {
              if (args.length > 1) {
                args[1] = stats2;
              } else {
                args.push(stats2);
              }
              this.emitWithAll(event, args);
            }
          }, "awfEmit");
          this._awaitWriteFinish(path5, awf.stabilityThreshold, event, awfEmit);
          return this;
        }
        if (event === EVENTS.CHANGE) {
          const isThrottled = !this._throttle(EVENTS.CHANGE, path5, 50);
          if (isThrottled)
            return this;
        }
        if (opts.alwaysStat && stats === void 0 && (event === EVENTS.ADD || event === EVENTS.ADD_DIR || event === EVENTS.CHANGE)) {
          const fullPath = opts.cwd ? sysPath2.join(opts.cwd, path5) : path5;
          let stats2;
          try {
            stats2 = await stat3(fullPath);
          } catch (err) {
          }
          if (!stats2 || this.closed)
            return;
          args.push(stats2);
        }
        this.emitWithAll(event, args);
        return this;
      }
      /**
       * Common handler for errors
       * @returns The error if defined, otherwise the value of the FSWatcher instance's `closed` flag
       */
      _handleError(error) {
        const code = error && error.code;
        if (error && code !== "ENOENT" && code !== "ENOTDIR" && (!this.options.ignorePermissionErrors || code !== "EPERM" && code !== "EACCES")) {
          this.emit(EVENTS.ERROR, error);
        }
        return error || this.closed;
      }
      /**
       * Helper utility for throttling
       * @param actionType type being throttled
       * @param path being acted upon
       * @param timeout duration of time to suppress duplicate actions
       * @returns tracking object or false if action should be suppressed
       */
      _throttle(actionType, path5, timeout) {
        if (!this._throttled.has(actionType)) {
          this._throttled.set(actionType, /* @__PURE__ */ new Map());
        }
        const action = this._throttled.get(actionType);
        if (!action)
          throw new Error("invalid throttle");
        const actionPath = action.get(path5);
        if (actionPath) {
          actionPath.count++;
          return false;
        }
        let timeoutObject;
        const clear = /* @__PURE__ */ __name(() => {
          const item = action.get(path5);
          const count = item ? item.count : 0;
          action.delete(path5);
          clearTimeout(timeoutObject);
          if (item)
            clearTimeout(item.timeoutObject);
          return count;
        }, "clear");
        timeoutObject = setTimeout(clear, timeout);
        const thr = { timeoutObject, clear, count: 0 };
        action.set(path5, thr);
        return thr;
      }
      _incrReadyCount() {
        return this._readyCount++;
      }
      /**
       * Awaits write operation to finish.
       * Polls a newly created file for size variations. When files size does not change for 'threshold' milliseconds calls callback.
       * @param path being acted upon
       * @param threshold Time in milliseconds a file size must be fixed before acknowledging write OP is finished
       * @param event
       * @param awfEmit Callback to be called when ready for event to be emitted.
       */
      _awaitWriteFinish(path5, threshold, event, awfEmit) {
        const awf = this.options.awaitWriteFinish;
        if (typeof awf !== "object")
          return;
        const pollInterval = awf.pollInterval;
        let timeoutHandler;
        let fullPath = path5;
        if (this.options.cwd && !sysPath2.isAbsolute(path5)) {
          fullPath = sysPath2.join(this.options.cwd, path5);
        }
        const now = /* @__PURE__ */ new Date();
        const writes = this._pendingWrites;
        function awaitWriteFinishFn(prevStat) {
          statcb(fullPath, (err, curStat) => {
            if (err || !writes.has(path5)) {
              if (err && err.code !== "ENOENT")
                awfEmit(err);
              return;
            }
            const now2 = Number(/* @__PURE__ */ new Date());
            if (prevStat && curStat.size !== prevStat.size) {
              writes.get(path5).lastChange = now2;
            }
            const pw = writes.get(path5);
            const df = now2 - pw.lastChange;
            if (df >= threshold) {
              writes.delete(path5);
              awfEmit(void 0, curStat);
            } else {
              timeoutHandler = setTimeout(awaitWriteFinishFn, pollInterval, curStat);
            }
          });
        }
        __name(awaitWriteFinishFn, "awaitWriteFinishFn");
        if (!writes.has(path5)) {
          writes.set(path5, {
            lastChange: now,
            cancelWait: /* @__PURE__ */ __name(() => {
              writes.delete(path5);
              clearTimeout(timeoutHandler);
              return event;
            }, "cancelWait")
          });
          timeoutHandler = setTimeout(awaitWriteFinishFn, pollInterval);
        }
      }
      /**
       * Determines whether user has asked to ignore this path.
       */
      _isIgnored(path5, stats) {
        if (this.options.atomic && DOT_RE.test(path5))
          return true;
        if (!this._userIgnored) {
          const { cwd } = this.options;
          const ign = this.options.ignored;
          const ignored = (ign || []).map(normalizeIgnored(cwd));
          const ignoredPaths = [...this._ignoredPaths];
          const list = [...ignoredPaths.map(normalizeIgnored(cwd)), ...ignored];
          this._userIgnored = anymatch(list, void 0);
        }
        return this._userIgnored(path5, stats);
      }
      _isntIgnored(path5, stat4) {
        return !this._isIgnored(path5, stat4);
      }
      /**
       * Provides a set of common helpers and properties relating to symlink handling.
       * @param path file or directory pattern being watched
       */
      _getWatchHelpers(path5) {
        return new WatchHelper(path5, this.options.followSymlinks, this);
      }
      // Directory helpers
      // -----------------
      /**
       * Provides directory tracking objects
       * @param directory path of the directory
       */
      _getWatchedDir(directory) {
        const dir = sysPath2.resolve(directory);
        if (!this._watched.has(dir))
          this._watched.set(dir, new DirEntry(dir, this._boundRemove));
        return this._watched.get(dir);
      }
      // File helpers
      // ------------
      /**
       * Check for read permissions: https://stackoverflow.com/a/11781404/1358405
       */
      _hasReadPermissions(stats) {
        if (this.options.ignorePermissionErrors)
          return true;
        return Boolean(Number(stats.mode) & 256);
      }
      /**
       * Handles emitting unlink events for
       * files and directories, and via recursion, for
       * files and directories within directories that are unlinked
       * @param directory within which the following item is located
       * @param item      base path of item/directory
       */
      _remove(directory, item, isDirectory) {
        const path5 = sysPath2.join(directory, item);
        const fullPath = sysPath2.resolve(path5);
        isDirectory = isDirectory != null ? isDirectory : this._watched.has(path5) || this._watched.has(fullPath);
        if (!this._throttle("remove", path5, 100))
          return;
        if (!isDirectory && this._watched.size === 1) {
          this.add(directory, item, true);
        }
        const wp = this._getWatchedDir(path5);
        const nestedDirectoryChildren = wp.getChildren();
        nestedDirectoryChildren.forEach((nested) => this._remove(path5, nested));
        const parent = this._getWatchedDir(directory);
        const wasTracked = parent.has(item);
        parent.remove(item);
        if (this._symlinkPaths.has(fullPath)) {
          this._symlinkPaths.delete(fullPath);
        }
        let relPath = path5;
        if (this.options.cwd)
          relPath = sysPath2.relative(this.options.cwd, path5);
        if (this.options.awaitWriteFinish && this._pendingWrites.has(relPath)) {
          const event = this._pendingWrites.get(relPath).cancelWait();
          if (event === EVENTS.ADD)
            return;
        }
        this._watched.delete(path5);
        this._watched.delete(fullPath);
        const eventName = isDirectory ? EVENTS.UNLINK_DIR : EVENTS.UNLINK;
        if (wasTracked && !this._isIgnored(path5))
          this._emit(eventName, path5);
        this._closePath(path5);
      }
      /**
       * Closes all watchers for a path
       */
      _closePath(path5) {
        this._closeFile(path5);
        const dir = sysPath2.dirname(path5);
        this._getWatchedDir(dir).remove(sysPath2.basename(path5));
      }
      /**
       * Closes only file-specific watchers
       */
      _closeFile(path5) {
        const closers = this._closers.get(path5);
        if (!closers)
          return;
        closers.forEach((closer) => closer());
        this._closers.delete(path5);
      }
      _addPathCloser(path5, closer) {
        if (!closer)
          return;
        let list = this._closers.get(path5);
        if (!list) {
          list = [];
          this._closers.set(path5, list);
        }
        list.push(closer);
      }
      _readdirp(root, opts) {
        if (this.closed)
          return;
        const options = { type: EVENTS.ALL, alwaysStat: true, lstat: true, ...opts, depth: 0 };
        let stream = readdirp(root, options);
        this._streams.add(stream);
        stream.once(STR_CLOSE, () => {
          stream = void 0;
        });
        stream.once(STR_END, () => {
          if (stream) {
            this._streams.delete(stream);
            stream = void 0;
          }
        });
        return stream;
      }
    };
    __name(watch, "watch");
    esm_default = { watch, FSWatcher };
  }
});

// src/sync/cli/cmd.ts
import * as fs3 from "fs";
import * as path4 from "path";

// src/sync/cli/env.ts
import * as fs from "fs";
import * as path2 from "path";
import * as crypto from "crypto";

// src/sync/utils.ts
function isAlwaysPublishable(path5) {
  if (path5.startsWith("_layouts/") && (path5.endsWith(".html") || path5.endsWith(".html.json"))) {
    return true;
  }
  return false;
}
__name(isAlwaysPublishable, "isAlwaysPublishable");

// src/sync/resolve.ts
import * as path from "path";
function resolveAssetPath(env, assetPath, notePath) {
  if (assetPath.startsWith("./")) {
    const noteDir2 = path.dirname(notePath);
    const relativePath = path.join(noteDir2, assetPath.slice(2));
    if (env.fileExistsSync(relativePath)) {
      return relativePath;
    }
    return null;
  }
  if (assetPath.startsWith("/")) {
    const absolutePath = assetPath.slice(1);
    if (env.fileExistsSync(absolutePath)) {
      return absolutePath;
    }
    return null;
  }
  if (assetPath.includes("/")) {
    if (env.fileExistsSync(assetPath)) {
      return assetPath;
    }
    return null;
  }
  if (env.fileExistsSync(assetPath)) {
    return assetPath;
  }
  const assetsPath = path.posix.join("assets", assetPath);
  if (env.fileExistsSync(assetsPath)) {
    return assetsPath;
  }
  const noteDir = path.dirname(notePath);
  if (noteDir && noteDir !== ".") {
    const relativePath = path.posix.join(noteDir, assetPath);
    if (env.fileExistsSync(relativePath)) {
      return relativePath;
    }
  }
  return null;
}
__name(resolveAssetPath, "resolveAssetPath");

// src/i18n.ts
var en = {
  // General
  sync: "Sync",
  syncStarting: "Starting sync...",
  allFilesUpToDate: "All files are up to date",
  syncError: "Sync error",
  connectionSuccessful: "Connection successful",
  connectionFailed: "Connection failed",
  // Settings
  settingsHeading: "Sync directories",
  settingsDescription: "Configure directories to sync with remote servers. See onboarding guide for details.",
  addSyncDirectory: "Add sync directory",
  testAllConnections: "Test all connections",
  pathLabel: "Sync folder",
  pathPlaceholder: "Path to folder",
  pathDesc: "Folder to sync. Use / for vault root (all files will be synced).",
  apiUrlLabel: "API URL",
  apiUrlPlaceholder: "https://yoursite.trip2g.com",
  apiUrlDesc: "Your Trip2g site URL. Example: https://yoursite.trip2g.com",
  apiKeyLabel: "API Key",
  apiKeyPlaceholder: "API Key",
  apiKeyDesc: "API key from your Trip2g admin panel.",
  publishFieldLabel: "Publish fields",
  publishFieldPlaceholder: "publish, public",
  publishFieldDesc: "Only sync files with these frontmatter fields set to true. Comma-separated list. Leave empty to sync all files.",
  twoWaySyncLabel: "Two-way sync",
  twoWaySyncDesc: "Download updates from server. Enable for TG channel import or server-side automation.",
  livePullIncludeLabel: "Live-pull include patterns",
  livePullIncludeDesc: "Glob patterns (e.g. **) of notes to pull in real time as the server changes. Leave empty to disable live-pull. Requires two-way sync.",
  livePullIncludePlaceholder: "**",
  livePullExcludeLabel: "Live-pull exclude patterns",
  livePullExcludeDesc: "Glob patterns to exclude from live-pull. Applied after include patterns.",
  livePullExcludePlaceholder: "drafts/**",
  testConnection: "Test connection",
  resetSyncState: "Reset sync state",
  resetSyncStateConfirm: "Reset sync state? Next sync will re-download all files from server.",
  syncStateReset: "Sync state reset",
  removeDirectory: "Remove sync directory",
  removeDirectoryConfirm: "Remove this sync connection?",
  error: "Error",
  successfulConnections: /* @__PURE__ */ __name((success, fail) => fail === 0 ? `All connections successful (${success})` : `${success} successful, ${fail} failed`, "successfulConnections"),
  globalSettingsHeading: "Global settings",
  skipPushConfirmationLabel: "Skip push confirmation",
  skipPushConfirmationDesc: "Don't show confirmation dialog before uploading files to server",
  autoSyncOnSaveLabel: "Auto-sync on save",
  autoSyncOnSaveDesc: "Automatically push local changes a few seconds after you save. Conflicts are skipped and left for the sync badge \u2014 never overwritten.",
  hideSyncStatusLabel: "Hide sync badge",
  hideSyncStatusDesc: "Don't show the pending-changes indicator on the ribbon icon",
  showSyncWarningsLabel: "Show sync warnings",
  showSyncWarningsDesc: "Show a popup with warnings after sync (e.g. broken links, missing assets)",
  syncWarningsCount: /* @__PURE__ */ __name((count) => `\u26A0\uFE0F ${count} sync warning${count === 1 ? "" : "s"}`, "syncWarningsCount"),
  onboardingLink: "Onboarding guide",
  onboardingUrl: "https://trip2g.com/en/user/getting_started",
  // Sync actions
  pulledFiles: /* @__PURE__ */ __name((count) => `Pulled ${count} files from server`, "pulledFiles"),
  pushedFiles: /* @__PURE__ */ __name((count) => `Pushed ${count} files to server`, "pushedFiles"),
  hiddenNotes: /* @__PURE__ */ __name((count) => `Hid ${count} notes not found locally`, "hiddenNotes"),
  pushed: "Pushed",
  livePulledFiles: /* @__PURE__ */ __name((count) => `Live-pulled ${count} file${count === 1 ? "" : "s"} from server`, "livePulledFiles"),
  livePullConflict: /* @__PURE__ */ __name((count) => `${count} note${count === 1 ? "" : "s"} changed on server but also edited locally \u2014 sync to resolve`, "livePullConflict"),
  syncFailedAuth: "Trip2g: your changes aren't syncing \u2014 check your API key and site URL in settings.",
  syncFailedGeneric: "Trip2g: sync failed \u2014 couldn't reach the server. Your changes are saved locally and will retry.",
  // Status bar
  statusBarTooltip: "Click: open \xB7 Right-click: copy",
  urlCopied: "URL copied",
  urlCopyFailed: "Failed to copy URL",
  urlOpenFailed: "Failed to open URL",
  // Auto-sync status bar (autoSyncOnSave)
  autoStatusSynced: /* @__PURE__ */ __name((time) => `\u2713 synced ${time}`, "autoStatusSynced"),
  autoStatusPending: /* @__PURE__ */ __name((count) => count > 1 ? `\u25CF ${count} pending\u2026` : `\u25CF pending\u2026`, "autoStatusPending"),
  autoStatusSyncing: "\u21BB syncing\u2026",
  autoStatusError: "\u26A0 sync error",
  autoStatusTooltip: "Auto-sync on save",
  pushedFilesTo: /* @__PURE__ */ __name((count, host) => `Published ${count} file${count === 1 ? "" : "s"} to ${host}`, "pushedFilesTo"),
  pushError: /* @__PURE__ */ __name((message) => `Auto-sync failed: ${message}`, "pushError"),
  // Conflict view
  syncConflict: "Sync conflict",
  conflictProgress: /* @__PURE__ */ __name((current, total) => `${current} / ${total}`, "conflictProgress"),
  localVersion: "Local version",
  serverVersion: "Server version",
  localLines: /* @__PURE__ */ __name((count) => `Local: ${count} lines`, "localLines"),
  serverLines: /* @__PURE__ */ __name((count) => `Server: ${count} lines`, "serverLines"),
  linesChanged: /* @__PURE__ */ __name((added, removed, modified) => `+${added} -${removed} ~${modified}`, "linesChanged"),
  keepLocal: "Keep local",
  useServer: "Use server",
  keepBoth: "Keep both",
  skip: "Skip",
  skipAll: /* @__PURE__ */ __name((remaining) => `Skip all (${remaining} remaining)`, "skipAll"),
  noConflicts: "No conflicts to resolve",
  allConflictsResolved: "All conflicts resolved!",
  // Migration modal
  syncSystemUpdate: "Sync system update",
  migrationFoundFiles: /* @__PURE__ */ __name((count) => `Found ${count} files with differences between local and server.`, "migrationFoundFiles"),
  migrationDescription: "This is a one-time setup after the plugin update.",
  reviewEachConflict: "Review each conflict",
  trustServerForAll: "Trust server for all",
  // Directory selection
  selectSyncDirectory: "Select sync directory",
  syncThisDirectory: "Sync this directory",
  noSyncDirsConfigured: "No sync directories configured. Please add one in settings first.",
  openSettings: "Open Settings",
  // Badge tooltips
  pendingChanges: /* @__PURE__ */ __name((pull, push) => `Trip2g Sync (\u2193${pull} \u2191${push})`, "pendingChanges"),
  pendingPull: /* @__PURE__ */ __name((count) => `Trip2g Sync (\u2193${count} from server)`, "pendingPull"),
  pendingPush: /* @__PURE__ */ __name((count) => `Trip2g Sync (\u2191${count} to push)`, "pendingPush"),
  // Progress messages
  progressPulling: /* @__PURE__ */ __name((current, total) => `Pulling ${current}/${total}...`, "progressPulling"),
  progressPushing: /* @__PURE__ */ __name((current, total) => `Pushing ${current}/${total}...`, "progressPushing"),
  progressDownloadingAssets: /* @__PURE__ */ __name((current, total) => `Downloading assets ${current}/${total}...`, "progressDownloadingAssets"),
  progressUploadingAssets: /* @__PURE__ */ __name((current, total) => `Uploading assets ${current}/${total}...`, "progressUploadingAssets"),
  progressClassifying: "Analyzing files...",
  // Server deleted modal
  serverDeletedTitle: "Files deleted on server",
  serverDeletedDescription: /* @__PURE__ */ __name((count) => `${count} file(s) were deleted/hidden on the server but still exist locally. What would you like to do?`, "serverDeletedDescription"),
  serverDeletedFileList: "Affected files:",
  deleteLocally: "Delete locally",
  keepLocally: "Keep locally",
  deletedLocally: /* @__PURE__ */ __name((count) => `Deleted ${count} local files`, "deletedLocally"),
  keptLocally: /* @__PURE__ */ __name((count) => `Kept ${count} local files`, "keptLocally"),
  // Push confirmation modal
  pushConfirmTitle: "Confirm upload to server",
  pushConfirmDescription: /* @__PURE__ */ __name((count) => `${count} file(s) will be uploaded to the server. Continue?`, "pushConfirmDescription"),
  pushConfirmFileList: "Files to upload:",
  pushConfirmProceed: "Upload",
  pushConfirmCancel: "Cancel",
  pushConfirmDontAskAgain: "Don't ask again",
  // Asset conflict modal
  assetConflictTitle: "Asset conflict",
  assetConflictDescription: /* @__PURE__ */ __name((count) => `${count} asset(s) differ between local and server. Choose which version to keep:`, "assetConflictDescription"),
  assetConflictFileList: "Conflicting assets:",
  assetConflictKeepLocal: "Upload local",
  assetConflictKeepRemote: "Download from server",
  assetConflictSkip: "Skip",
  assetConflictApplyToAll: "Apply to all conflicts",
  assetUploaded: /* @__PURE__ */ __name((count) => `Uploaded ${count} assets`, "assetUploaded"),
  assetDownloaded: /* @__PURE__ */ __name((count) => `Downloaded ${count} assets`, "assetDownloaded"),
  assetTooLarge: /* @__PURE__ */ __name((fileName) => `File "${fileName}" is too large to upload`, "assetTooLarge")
};
var ru = {
  // General
  sync: "\u0421\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F",
  syncStarting: "\u041D\u0430\u0447\u0438\u043D\u0430\u044E \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044E...",
  allFilesUpToDate: "\u0412\u0441\u0435 \u0444\u0430\u0439\u043B\u044B \u0430\u043A\u0442\u0443\u0430\u043B\u044C\u043D\u044B",
  syncError: "\u041E\u0448\u0438\u0431\u043A\u0430 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  connectionSuccessful: "\u0421\u043E\u0435\u0434\u0438\u043D\u0435\u043D\u0438\u0435 \u0443\u0441\u043F\u0435\u0448\u043D\u043E",
  connectionFailed: "\u041E\u0448\u0438\u0431\u043A\u0430 \u0441\u043E\u0435\u0434\u0438\u043D\u0435\u043D\u0438\u044F",
  // Settings
  settingsHeading: "\u041F\u0430\u043F\u043A\u0438 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  settingsDescription: "\u041D\u0430\u0441\u0442\u0440\u043E\u0439\u0442\u0435 \u043F\u0430\u043F\u043A\u0438 \u0434\u043B\u044F \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438 \u0441 \u0443\u0434\u0430\u043B\u0451\u043D\u043D\u044B\u043C\u0438 \u0441\u0435\u0440\u0432\u0435\u0440\u0430\u043C\u0438. \u0421\u043C. \u0440\u0443\u043A\u043E\u0432\u043E\u0434\u0441\u0442\u0432\u043E \u043F\u043E \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0435.",
  addSyncDirectory: "\u0414\u043E\u0431\u0430\u0432\u0438\u0442\u044C \u043F\u0430\u043F\u043A\u0443",
  testAllConnections: "\u041F\u0440\u043E\u0432\u0435\u0440\u0438\u0442\u044C \u0432\u0441\u0435 \u0441\u043E\u0435\u0434\u0438\u043D\u0435\u043D\u0438\u044F",
  pathLabel: "\u041F\u0430\u043F\u043A\u0430 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  pathPlaceholder: "\u041F\u0443\u0442\u044C \u043A \u043F\u0430\u043F\u043A\u0435",
  pathDesc: "\u041F\u0430\u043F\u043A\u0430 \u0434\u043B\u044F \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438. \u0418\u0441\u043F\u043E\u043B\u044C\u0437\u0443\u0439\u0442\u0435 / \u0434\u043B\u044F \u043A\u043E\u0440\u043D\u044F \u0445\u0440\u0430\u043D\u0438\u043B\u0438\u0449\u0430 (\u0432\u0441\u0435 \u0444\u0430\u0439\u043B\u044B \u0431\u0443\u0434\u0443\u0442 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0438\u0440\u043E\u0432\u0430\u043D\u044B).",
  apiUrlLabel: "API URL",
  apiUrlPlaceholder: "https://yoursite.trip2g.com",
  apiUrlDesc: "URL \u0432\u0430\u0448\u0435\u0433\u043E \u0441\u0430\u0439\u0442\u0430 Trip2g. \u041F\u0440\u0438\u043C\u0435\u0440: https://yoursite.trip2g.com",
  apiKeyLabel: "API Key",
  apiKeyPlaceholder: "API Key",
  apiKeyDesc: "API \u043A\u043B\u044E\u0447 \u0438\u0437 \u0430\u0434\u043C\u0438\u043D-\u043F\u0430\u043D\u0435\u043B\u0438 Trip2g.",
  publishFieldLabel: "\u041F\u043E\u043B\u044F \u043F\u0443\u0431\u043B\u0438\u043A\u0430\u0446\u0438\u0438",
  publishFieldPlaceholder: "publish, public",
  publishFieldDesc: "\u0421\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0438\u0440\u043E\u0432\u0430\u0442\u044C \u0442\u043E\u043B\u044C\u043A\u043E \u0444\u0430\u0439\u043B\u044B \u0441 \u044D\u0442\u0438\u043C\u0438 \u043F\u043E\u043B\u044F\u043C\u0438 \u0432 frontmatter (\u0437\u043D\u0430\u0447\u0435\u043D\u0438\u0435 true). \u0427\u0435\u0440\u0435\u0437 \u0437\u0430\u043F\u044F\u0442\u0443\u044E. \u041E\u0441\u0442\u0430\u0432\u044C\u0442\u0435 \u043F\u0443\u0441\u0442\u044B\u043C \u0434\u043B\u044F \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438 \u0432\u0441\u0435\u0445 \u0444\u0430\u0439\u043B\u043E\u0432.",
  twoWaySyncLabel: "\u0414\u0432\u0443\u0441\u0442\u043E\u0440\u043E\u043D\u043D\u044F\u044F \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F",
  twoWaySyncDesc: "\u0417\u0430\u0433\u0440\u0443\u0436\u0430\u0442\u044C \u043E\u0431\u043D\u043E\u0432\u043B\u0435\u043D\u0438\u044F \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430. \u0412\u043A\u043B\u044E\u0447\u0438\u0442\u0435 \u0434\u043B\u044F \u0438\u043C\u043F\u043E\u0440\u0442\u0430 TG \u043A\u0430\u043D\u0430\u043B\u043E\u0432 \u0438\u043B\u0438 \u0441\u0435\u0440\u0432\u0435\u0440\u043D\u043E\u0439 \u0430\u0432\u0442\u043E\u043C\u0430\u0442\u0438\u0437\u0430\u0446\u0438\u0438.",
  livePullIncludeLabel: "\u0428\u0430\u0431\u043B\u043E\u043D\u044B live-pull (\u0432\u043A\u043B\u044E\u0447\u0438\u0442\u044C)",
  livePullIncludeDesc: "Glob-\u0448\u0430\u0431\u043B\u043E\u043D\u044B (\u043D\u0430\u043F\u0440\u0438\u043C\u0435\u0440, **) \u0437\u0430\u043C\u0435\u0442\u043E\u043A \u0434\u043B\u044F \u0437\u0430\u0433\u0440\u0443\u0437\u043A\u0438 \u0432 \u0440\u0435\u0430\u043B\u044C\u043D\u043E\u043C \u0432\u0440\u0435\u043C\u0435\u043D\u0438 \u043F\u0440\u0438 \u0438\u0437\u043C\u0435\u043D\u0435\u043D\u0438\u044F\u0445 \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440\u0435. \u041E\u0441\u0442\u0430\u0432\u044C\u0442\u0435 \u043F\u0443\u0441\u0442\u044B\u043C, \u0447\u0442\u043E\u0431\u044B \u043E\u0442\u043A\u043B\u044E\u0447\u0438\u0442\u044C live-pull. \u0422\u0440\u0435\u0431\u0443\u0435\u0442 \u0434\u0432\u0443\u0441\u0442\u043E\u0440\u043E\u043D\u043D\u0435\u0439 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438.",
  livePullIncludePlaceholder: "**",
  livePullExcludeLabel: "\u0428\u0430\u0431\u043B\u043E\u043D\u044B live-pull (\u0438\u0441\u043A\u043B\u044E\u0447\u0438\u0442\u044C)",
  livePullExcludeDesc: "Glob-\u0448\u0430\u0431\u043B\u043E\u043D\u044B \u0434\u043B\u044F \u0438\u0441\u043A\u043B\u044E\u0447\u0435\u043D\u0438\u044F \u0438\u0437 live-pull. \u041F\u0440\u0438\u043C\u0435\u043D\u044F\u044E\u0442\u0441\u044F \u043F\u043E\u0441\u043B\u0435 \u0432\u043A\u043B\u044E\u0447\u0430\u044E\u0449\u0438\u0445 \u0448\u0430\u0431\u043B\u043E\u043D\u043E\u0432.",
  livePullExcludePlaceholder: "drafts/**",
  testConnection: "\u041F\u0440\u043E\u0432\u0435\u0440\u0438\u0442\u044C \u0441\u043E\u0435\u0434\u0438\u043D\u0435\u043D\u0438\u0435",
  resetSyncState: "\u0421\u0431\u0440\u043E\u0441\u0438\u0442\u044C \u0441\u043E\u0441\u0442\u043E\u044F\u043D\u0438\u0435 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  resetSyncStateConfirm: "\u0421\u0431\u0440\u043E\u0441\u0438\u0442\u044C \u0441\u043E\u0441\u0442\u043E\u044F\u043D\u0438\u0435? \u041F\u0440\u0438 \u0441\u043B\u0435\u0434\u0443\u044E\u0449\u0435\u0439 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438 \u0432\u0441\u0435 \u0444\u0430\u0439\u043B\u044B \u0431\u0443\u0434\u0443\u0442 \u0437\u0430\u0433\u0440\u0443\u0436\u0435\u043D\u044B \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430.",
  syncStateReset: "\u0421\u043E\u0441\u0442\u043E\u044F\u043D\u0438\u0435 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438 \u0441\u0431\u0440\u043E\u0448\u0435\u043D\u043E",
  removeDirectory: "\u0423\u0434\u0430\u043B\u0438\u0442\u044C \u043F\u0430\u043F\u043A\u0443",
  removeDirectoryConfirm: "\u0423\u0434\u0430\u043B\u0438\u0442\u044C \u044D\u0442\u043E \u043F\u043E\u0434\u043A\u043B\u044E\u0447\u0435\u043D\u0438\u0435?",
  error: "\u041E\u0448\u0438\u0431\u043A\u0430",
  successfulConnections: /* @__PURE__ */ __name((success, fail) => fail === 0 ? `\u0412\u0441\u0435 \u0441\u043E\u0435\u0434\u0438\u043D\u0435\u043D\u0438\u044F \u0443\u0441\u043F\u0435\u0448\u043D\u044B (${success})` : `${success} \u0443\u0441\u043F\u0435\u0448\u043D\u043E, ${fail} \u0441 \u043E\u0448\u0438\u0431\u043A\u043E\u0439`, "successfulConnections"),
  globalSettingsHeading: "\u041E\u0431\u0449\u0438\u0435 \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0438",
  skipPushConfirmationLabel: "\u041D\u0435 \u0441\u043F\u0440\u0430\u0448\u0438\u0432\u0430\u0442\u044C \u043F\u043E\u0434\u0442\u0432\u0435\u0440\u0436\u0434\u0435\u043D\u0438\u0435",
  skipPushConfirmationDesc: "\u041D\u0435 \u043F\u043E\u043A\u0430\u0437\u044B\u0432\u0430\u0442\u044C \u0434\u0438\u0430\u043B\u043E\u0433 \u043F\u043E\u0434\u0442\u0432\u0435\u0440\u0436\u0434\u0435\u043D\u0438\u044F \u043F\u0435\u0440\u0435\u0434 \u0437\u0430\u0433\u0440\u0443\u0437\u043A\u043E\u0439 \u0444\u0430\u0439\u043B\u043E\u0432 \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440",
  autoSyncOnSaveLabel: "\u0410\u0432\u0442\u043E\u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F \u043F\u0440\u0438 \u0441\u043E\u0445\u0440\u0430\u043D\u0435\u043D\u0438\u0438",
  autoSyncOnSaveDesc: "\u0410\u0432\u0442\u043E\u043C\u0430\u0442\u0438\u0447\u0435\u0441\u043A\u0438 \u043E\u0442\u043F\u0440\u0430\u0432\u043B\u044F\u0442\u044C \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u044B\u0435 \u0438\u0437\u043C\u0435\u043D\u0435\u043D\u0438\u044F \u0447\u0435\u0440\u0435\u0437 \u043D\u0435\u0441\u043A\u043E\u043B\u044C\u043A\u043E \u0441\u0435\u043A\u0443\u043D\u0434 \u043F\u043E\u0441\u043B\u0435 \u0441\u043E\u0445\u0440\u0430\u043D\u0435\u043D\u0438\u044F. \u041A\u043E\u043D\u0444\u043B\u0438\u043A\u0442\u044B \u043F\u0440\u043E\u043F\u0443\u0441\u043A\u0430\u044E\u0442\u0441\u044F \u0438 \u043E\u0441\u0442\u0430\u044E\u0442\u0441\u044F \u043D\u0430 \u0431\u0435\u0439\u0434\u0436\u0435 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438 \u2014 \u043D\u0438\u043A\u043E\u0433\u0434\u0430 \u043D\u0435 \u043F\u0435\u0440\u0435\u0437\u0430\u043F\u0438\u0441\u044B\u0432\u0430\u044E\u0442\u0441\u044F.",
  hideSyncStatusLabel: "\u0421\u043A\u0440\u044B\u0442\u044C \u0431\u0435\u0439\u0434\u0436 \u0441\u0438\u043D\u043A\u0430",
  hideSyncStatusDesc: "\u041D\u0435 \u043F\u043E\u043A\u0430\u0437\u044B\u0432\u0430\u0442\u044C \u0438\u043D\u0434\u0438\u043A\u0430\u0442\u043E\u0440 \u043E\u0436\u0438\u0434\u0430\u044E\u0449\u0438\u0445 \u0438\u0437\u043C\u0435\u043D\u0435\u043D\u0438\u0439 \u043D\u0430 \u0438\u043A\u043E\u043D\u043A\u0435",
  showSyncWarningsLabel: "\u041F\u043E\u043A\u0430\u0437\u044B\u0432\u0430\u0442\u044C \u043F\u0440\u0435\u0434\u0443\u043F\u0440\u0435\u0436\u0434\u0435\u043D\u0438\u044F",
  showSyncWarningsDesc: "\u041F\u043E\u043A\u0430\u0437\u044B\u0432\u0430\u0442\u044C \u043E\u043A\u043D\u043E \u0441 \u043F\u0440\u0435\u0434\u0443\u043F\u0440\u0435\u0436\u0434\u0435\u043D\u0438\u044F\u043C\u0438 \u043F\u043E\u0441\u043B\u0435 \u0441\u0438\u043D\u043A\u0430 (\u0431\u0438\u0442\u044B\u0435 \u0441\u0441\u044B\u043B\u043A\u0438, \u043E\u0442\u0441\u0443\u0442\u0441\u0442\u0432\u0443\u044E\u0449\u0438\u0435 \u0444\u0430\u0439\u043B\u044B)",
  syncWarningsCount: /* @__PURE__ */ __name((count) => `\u26A0\uFE0F ${count} \u043F\u0440\u0435\u0434\u0443\u043F\u0440\u0435\u0436\u0434\u0435\u043D\u0438${count === 1 ? "\u0435" : "\u0439"}`, "syncWarningsCount"),
  onboardingLink: "\u0420\u0443\u043A\u043E\u0432\u043E\u0434\u0441\u0442\u0432\u043E \u043F\u043E \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0435",
  onboardingUrl: "https://trip2g.com/docs/onboarding",
  // Sync actions
  pulledFiles: /* @__PURE__ */ __name((count) => `\u041F\u043E\u043B\u0443\u0447\u0435\u043D\u043E ${count} \u0444\u0430\u0439\u043B\u043E\u0432 \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430`, "pulledFiles"),
  pushedFiles: /* @__PURE__ */ __name((count) => `\u041E\u0442\u043F\u0440\u0430\u0432\u043B\u0435\u043D\u043E ${count} \u0444\u0430\u0439\u043B\u043E\u0432 \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440`, "pushedFiles"),
  hiddenNotes: /* @__PURE__ */ __name((count) => `\u0421\u043A\u0440\u044B\u0442\u043E ${count} \u0437\u0430\u043C\u0435\u0442\u043E\u043A, \u043E\u0442\u0441\u0443\u0442\u0441\u0442\u0432\u0443\u044E\u0449\u0438\u0445 \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E`, "hiddenNotes"),
  pushed: "\u041E\u0442\u043F\u0440\u0430\u0432\u043B\u0435\u043D\u043E",
  livePulledFiles: /* @__PURE__ */ __name((count) => `Live-pull: \u043F\u043E\u043B\u0443\u0447\u0435\u043D\u043E ${count} \u0444\u0430\u0439\u043B\u043E\u0432 \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430`, "livePulledFiles"),
  livePullConflict: /* @__PURE__ */ __name((count) => `${count} \u0437\u0430\u043C\u0435\u0442\u043E\u043A \u0438\u0437\u043C\u0435\u043D\u0435\u043D\u044B \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440\u0435 \u0438 \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E \u2014 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0438\u0440\u0443\u0439\u0442\u0435 \u0434\u043B\u044F \u0440\u0430\u0437\u0440\u0435\u0448\u0435\u043D\u0438\u044F`, "livePullConflict"),
  syncFailedAuth: "Trip2g: \u0438\u0437\u043C\u0435\u043D\u0435\u043D\u0438\u044F \u043D\u0435 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0438\u0440\u0443\u044E\u0442\u0441\u044F \u2014 \u043F\u0440\u043E\u0432\u0435\u0440\u044C\u0442\u0435 API-\u043A\u043B\u044E\u0447 \u0438 URL \u0441\u0430\u0439\u0442\u0430 \u0432 \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0430\u0445.",
  syncFailedGeneric: "Trip2g: \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F \u043D\u0435 \u0443\u0434\u0430\u043B\u0430\u0441\u044C \u2014 \u0441\u0435\u0440\u0432\u0435\u0440 \u043D\u0435\u0434\u043E\u0441\u0442\u0443\u043F\u0435\u043D. \u0418\u0437\u043C\u0435\u043D\u0435\u043D\u0438\u044F \u0441\u043E\u0445\u0440\u0430\u043D\u0435\u043D\u044B \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E, \u043F\u043E\u043F\u0440\u043E\u0431\u0443\u0435\u043C \u0435\u0449\u0451 \u0440\u0430\u0437.",
  // Status bar
  statusBarTooltip: "\u041A\u043B\u0438\u043A: \u043E\u0442\u043A\u0440\u044B\u0442\u044C \xB7 \u041F\u0440\u0430\u0432\u044B\u0439 \u043A\u043B\u0438\u043A: \u043A\u043E\u043F\u0438\u0440\u043E\u0432\u0430\u0442\u044C",
  urlCopied: "URL \u0441\u043A\u043E\u043F\u0438\u0440\u043E\u0432\u0430\u043D",
  urlCopyFailed: "\u041D\u0435 \u0443\u0434\u0430\u043B\u043E\u0441\u044C \u0441\u043A\u043E\u043F\u0438\u0440\u043E\u0432\u0430\u0442\u044C URL",
  urlOpenFailed: "\u041D\u0435 \u0443\u0434\u0430\u043B\u043E\u0441\u044C \u043E\u0442\u043A\u0440\u044B\u0442\u044C URL",
  // Auto-sync status bar (autoSyncOnSave)
  autoStatusSynced: /* @__PURE__ */ __name((time) => `\u2713 \u0441\u0438\u043D\u0445\u0440. ${time}`, "autoStatusSynced"),
  autoStatusPending: /* @__PURE__ */ __name((count) => count > 1 ? `\u25CF ${count} \u0432 \u043E\u0447\u0435\u0440\u0435\u0434\u0438\u2026` : `\u25CF \u043E\u0436\u0438\u0434\u0430\u043D\u0438\u0435\u2026`, "autoStatusPending"),
  autoStatusSyncing: "\u21BB \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F\u2026",
  autoStatusError: "\u26A0 \u043E\u0448\u0438\u0431\u043A\u0430 \u0441\u0438\u043D\u0445\u0440.",
  autoStatusTooltip: "\u0410\u0432\u0442\u043E\u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F \u043F\u0440\u0438 \u0441\u043E\u0445\u0440\u0430\u043D\u0435\u043D\u0438\u0438",
  pushedFilesTo: /* @__PURE__ */ __name((count, host) => `\u041E\u043F\u0443\u0431\u043B\u0438\u043A\u043E\u0432\u0430\u043D\u043E ${count} \u0444\u0430\u0439\u043B(\u043E\u0432) \u043D\u0430 ${host}`, "pushedFilesTo"),
  pushError: /* @__PURE__ */ __name((message) => `\u0410\u0432\u0442\u043E\u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u044F \u043D\u0435 \u0443\u0434\u0430\u043B\u0430\u0441\u044C: ${message}`, "pushError"),
  // Conflict view
  syncConflict: "\u041A\u043E\u043D\u0444\u043B\u0438\u043A\u0442 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  conflictProgress: /* @__PURE__ */ __name((current, total) => `${current} / ${total}`, "conflictProgress"),
  localVersion: "\u041B\u043E\u043A\u0430\u043B\u044C\u043D\u0430\u044F \u0432\u0435\u0440\u0441\u0438\u044F",
  serverVersion: "\u0412\u0435\u0440\u0441\u0438\u044F \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440\u0435",
  localLines: /* @__PURE__ */ __name((count) => `\u041B\u043E\u043A\u0430\u043B\u044C\u043D\u043E: ${count} \u0441\u0442\u0440\u043E\u043A`, "localLines"),
  serverLines: /* @__PURE__ */ __name((count) => `\u0421\u0435\u0440\u0432\u0435\u0440: ${count} \u0441\u0442\u0440\u043E\u043A`, "serverLines"),
  linesChanged: /* @__PURE__ */ __name((added, removed, modified) => `+${added} -${removed} ~${modified}`, "linesChanged"),
  keepLocal: "\u041E\u0441\u0442\u0430\u0432\u0438\u0442\u044C \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u0443\u044E",
  useServer: "\u0412\u0437\u044F\u0442\u044C \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430",
  keepBoth: "\u0421\u043E\u0445\u0440\u0430\u043D\u0438\u0442\u044C \u043E\u0431\u0435",
  skip: "\u041F\u0440\u043E\u043F\u0443\u0441\u0442\u0438\u0442\u044C",
  skipAll: /* @__PURE__ */ __name((remaining) => `\u041F\u0440\u043E\u043F\u0443\u0441\u0442\u0438\u0442\u044C \u0432\u0441\u0435 (${remaining} \u043E\u0441\u0442\u0430\u043B\u043E\u0441\u044C)`, "skipAll"),
  noConflicts: "\u041D\u0435\u0442 \u043A\u043E\u043D\u0444\u043B\u0438\u043A\u0442\u043E\u0432 \u0434\u043B\u044F \u0440\u0430\u0437\u0440\u0435\u0448\u0435\u043D\u0438\u044F",
  allConflictsResolved: "\u0412\u0441\u0435 \u043A\u043E\u043D\u0444\u043B\u0438\u043A\u0442\u044B \u0440\u0430\u0437\u0440\u0435\u0448\u0435\u043D\u044B!",
  // Migration modal
  syncSystemUpdate: "\u041E\u0431\u043D\u043E\u0432\u043B\u0435\u043D\u0438\u0435 \u0441\u0438\u0441\u0442\u0435\u043C\u044B \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  migrationFoundFiles: /* @__PURE__ */ __name((count) => `\u041D\u0430\u0439\u0434\u0435\u043D\u043E ${count} \u0444\u0430\u0439\u043B\u043E\u0432 \u0441 \u0440\u0430\u0437\u043B\u0438\u0447\u0438\u044F\u043C\u0438 \u043C\u0435\u0436\u0434\u0443 \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E\u0439 \u0438 \u0441\u0435\u0440\u0432\u0435\u0440\u043D\u043E\u0439 \u0432\u0435\u0440\u0441\u0438\u044F\u043C\u0438.`, "migrationFoundFiles"),
  migrationDescription: "\u042D\u0442\u043E \u043E\u0434\u043D\u043E\u0440\u0430\u0437\u043E\u0432\u0430\u044F \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0430 \u043F\u043E\u0441\u043B\u0435 \u043E\u0431\u043D\u043E\u0432\u043B\u0435\u043D\u0438\u044F \u043F\u043B\u0430\u0433\u0438\u043D\u0430.",
  reviewEachConflict: "\u041F\u0440\u043E\u0432\u0435\u0440\u0438\u0442\u044C \u043A\u0430\u0436\u0434\u044B\u0439 \u043A\u043E\u043D\u0444\u043B\u0438\u043A\u0442",
  trustServerForAll: "\u0414\u043E\u0432\u0435\u0440\u044F\u0442\u044C \u0441\u0435\u0440\u0432\u0435\u0440\u0443 \u0434\u043B\u044F \u0432\u0441\u0435\u0445",
  // Directory selection
  selectSyncDirectory: "\u0412\u044B\u0431\u0435\u0440\u0438\u0442\u0435 \u043F\u0430\u043F\u043A\u0443 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438",
  syncThisDirectory: "\u0421\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0438\u0440\u043E\u0432\u0430\u0442\u044C \u044D\u0442\u0443 \u043F\u0430\u043F\u043A\u0443",
  noSyncDirsConfigured: "\u041F\u0430\u043F\u043A\u0438 \u0441\u0438\u043D\u0445\u0440\u043E\u043D\u0438\u0437\u0430\u0446\u0438\u0438 \u043D\u0435 \u043D\u0430\u0441\u0442\u0440\u043E\u0435\u043D\u044B. \u0414\u043E\u0431\u0430\u0432\u044C\u0442\u0435 \u0438\u0445 \u0432 \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0430\u0445.",
  openSettings: "\u041E\u0442\u043A\u0440\u044B\u0442\u044C \u043D\u0430\u0441\u0442\u0440\u043E\u0439\u043A\u0438",
  // Badge tooltips
  pendingChanges: /* @__PURE__ */ __name((pull, push) => `Trip2g Sync (\u2193${pull} \u2191${push})`, "pendingChanges"),
  pendingPull: /* @__PURE__ */ __name((count) => `Trip2g Sync (\u2193${count} \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430)`, "pendingPull"),
  pendingPush: /* @__PURE__ */ __name((count) => `Trip2g Sync (\u2191${count} \u043A \u043E\u0442\u043F\u0440\u0430\u0432\u043A\u0435)`, "pendingPush"),
  // Progress messages
  progressPulling: /* @__PURE__ */ __name((current, total) => `\u0417\u0430\u0433\u0440\u0443\u0437\u043A\u0430 ${current}/${total}...`, "progressPulling"),
  progressPushing: /* @__PURE__ */ __name((current, total) => `\u041E\u0442\u043F\u0440\u0430\u0432\u043A\u0430 ${current}/${total}...`, "progressPushing"),
  progressDownloadingAssets: /* @__PURE__ */ __name((current, total) => `\u0417\u0430\u0433\u0440\u0443\u0437\u043A\u0430 \u0430\u0441\u0441\u0435\u0442\u043E\u0432 ${current}/${total}...`, "progressDownloadingAssets"),
  progressUploadingAssets: /* @__PURE__ */ __name((current, total) => `\u041E\u0442\u043F\u0440\u0430\u0432\u043A\u0430 \u0430\u0441\u0441\u0435\u0442\u043E\u0432 ${current}/${total}...`, "progressUploadingAssets"),
  progressClassifying: "\u0410\u043D\u0430\u043B\u0438\u0437 \u0444\u0430\u0439\u043B\u043E\u0432...",
  // Server deleted modal
  serverDeletedTitle: "\u0424\u0430\u0439\u043B\u044B \u0443\u0434\u0430\u043B\u0435\u043D\u044B \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440\u0435",
  serverDeletedDescription: /* @__PURE__ */ __name((count) => `${count} \u0444\u0430\u0439\u043B(\u043E\u0432) \u0431\u044B\u043B\u0438 \u0443\u0434\u0430\u043B\u0435\u043D\u044B/\u0441\u043A\u0440\u044B\u0442\u044B \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440\u0435, \u043D\u043E \u0432\u0441\u0451 \u0435\u0449\u0451 \u0441\u0443\u0449\u0435\u0441\u0442\u0432\u0443\u044E\u0442 \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E. \u0427\u0442\u043E \u0441\u0434\u0435\u043B\u0430\u0442\u044C?`, "serverDeletedDescription"),
  serverDeletedFileList: "\u0417\u0430\u0442\u0440\u043E\u043D\u0443\u0442\u044B\u0435 \u0444\u0430\u0439\u043B\u044B:",
  deleteLocally: "\u0423\u0434\u0430\u043B\u0438\u0442\u044C \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E",
  keepLocally: "\u041E\u0441\u0442\u0430\u0432\u0438\u0442\u044C \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E",
  deletedLocally: /* @__PURE__ */ __name((count) => `\u0423\u0434\u0430\u043B\u0435\u043D\u043E ${count} \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u044B\u0445 \u0444\u0430\u0439\u043B\u043E\u0432`, "deletedLocally"),
  keptLocally: /* @__PURE__ */ __name((count) => `\u041E\u0441\u0442\u0430\u0432\u043B\u0435\u043D\u043E ${count} \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u044B\u0445 \u0444\u0430\u0439\u043B\u043E\u0432`, "keptLocally"),
  // Push confirmation modal
  pushConfirmTitle: "\u041F\u043E\u0434\u0442\u0432\u0435\u0440\u0434\u0438\u0442\u0435 \u0437\u0430\u0433\u0440\u0443\u0437\u043A\u0443 \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440",
  pushConfirmDescription: /* @__PURE__ */ __name((count) => `${count} \u0444\u0430\u0439\u043B(\u043E\u0432) \u0431\u0443\u0434\u0443\u0442 \u0437\u0430\u0433\u0440\u0443\u0436\u0435\u043D\u044B \u043D\u0430 \u0441\u0435\u0440\u0432\u0435\u0440. \u041F\u0440\u043E\u0434\u043E\u043B\u0436\u0438\u0442\u044C?`, "pushConfirmDescription"),
  pushConfirmFileList: "\u0424\u0430\u0439\u043B\u044B \u0434\u043B\u044F \u0437\u0430\u0433\u0440\u0443\u0437\u043A\u0438:",
  pushConfirmProceed: "\u0417\u0430\u0433\u0440\u0443\u0437\u0438\u0442\u044C",
  pushConfirmCancel: "\u041E\u0442\u043C\u0435\u043D\u0430",
  pushConfirmDontAskAgain: "\u0411\u043E\u043B\u044C\u0448\u0435 \u043D\u0435 \u0441\u043F\u0440\u0430\u0448\u0438\u0432\u0430\u0442\u044C",
  // Asset conflict modal
  assetConflictTitle: "\u041A\u043E\u043D\u0444\u043B\u0438\u043A\u0442 \u0430\u0441\u0441\u0435\u0442\u043E\u0432",
  assetConflictDescription: /* @__PURE__ */ __name((count) => `${count} \u0430\u0441\u0441\u0435\u0442(\u043E\u0432) \u043E\u0442\u043B\u0438\u0447\u0430\u044E\u0442\u0441\u044F \u043C\u0435\u0436\u0434\u0443 \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u043E\u0439 \u0438 \u0441\u0435\u0440\u0432\u0435\u0440\u043D\u043E\u0439 \u0432\u0435\u0440\u0441\u0438\u044F\u043C\u0438. \u0412\u044B\u0431\u0435\u0440\u0438\u0442\u0435 \u043A\u0430\u043A\u0443\u044E \u0432\u0435\u0440\u0441\u0438\u044E \u043E\u0441\u0442\u0430\u0432\u0438\u0442\u044C:`, "assetConflictDescription"),
  assetConflictFileList: "\u041A\u043E\u043D\u0444\u043B\u0438\u043A\u0442\u0443\u044E\u0449\u0438\u0435 \u0430\u0441\u0441\u0435\u0442\u044B:",
  assetConflictKeepLocal: "\u0417\u0430\u0433\u0440\u0443\u0437\u0438\u0442\u044C \u043B\u043E\u043A\u0430\u043B\u044C\u043D\u044B\u0435",
  assetConflictKeepRemote: "\u0421\u043A\u0430\u0447\u0430\u0442\u044C \u0441 \u0441\u0435\u0440\u0432\u0435\u0440\u0430",
  assetConflictSkip: "\u041F\u0440\u043E\u043F\u0443\u0441\u0442\u0438\u0442\u044C",
  assetConflictApplyToAll: "\u041F\u0440\u0438\u043C\u0435\u043D\u0438\u0442\u044C \u043A\u043E \u0432\u0441\u0435\u043C \u043A\u043E\u043D\u0444\u043B\u0438\u043A\u0442\u0430\u043C",
  assetUploaded: /* @__PURE__ */ __name((count) => `\u0417\u0430\u0433\u0440\u0443\u0436\u0435\u043D\u043E ${count} \u0430\u0441\u0441\u0435\u0442\u043E\u0432`, "assetUploaded"),
  assetDownloaded: /* @__PURE__ */ __name((count) => `\u0421\u043A\u0430\u0447\u0430\u043D\u043E ${count} \u0430\u0441\u0441\u0435\u0442\u043E\u0432`, "assetDownloaded"),
  assetTooLarge: /* @__PURE__ */ __name((fileName) => `\u0424\u0430\u0439\u043B "${fileName}" \u0441\u043B\u0438\u0448\u043A\u043E\u043C \u0431\u043E\u043B\u044C\u0448\u043E\u0439 \u0434\u043B\u044F \u0437\u0430\u0433\u0440\u0443\u0437\u043A\u0438`, "assetTooLarge")
};
var translations = { en, ru };
var currentLocale = "en";
function t() {
  return translations[currentLocale];
}
__name(t, "t");

// src/sync/upload-retry.ts
var AssetTooLargeError = class extends Error {
  static {
    __name(this, "AssetTooLargeError");
  }
  constructor(fileName) {
    super(t().assetTooLarge(fileName));
    this.name = "AssetTooLargeError";
    this.fileName = fileName;
  }
};
function isNonRetryableUploadError(e) {
  return e instanceof AssetTooLargeError;
}
__name(isNonRetryableUploadError, "isNonRetryableUploadError");
var defaultSleep = /* @__PURE__ */ __name((ms) => new Promise((resolve5) => setTimeout(resolve5, ms)), "defaultSleep");
async function uploadWithRetry(attempt, opts = {}) {
  const maxRetries = opts.maxRetries ?? 10;
  const sleep = opts.sleep ?? defaultSleep;
  for (let n = 1; n <= maxRetries; n++) {
    try {
      if (await attempt()) {
        return true;
      }
    } catch (e) {
      if (isNonRetryableUploadError(e)) {
        opts.onGiveUp?.(e, n);
        return false;
      }
      if (n < maxRetries) {
        opts.onRetry?.(n, e);
        await sleep(Math.pow(2, n - 1) * 1e3);
        continue;
      }
      opts.onGiveUp?.(e, n);
      return false;
    }
  }
  return false;
}
__name(uploadWithRetry, "uploadWithRetry");

// src/sync/cli/graphql-client.ts
var GraphQLClient = class {
  constructor(url, options = {}) {
    this.url = url;
    this.options = options;
  }
  static {
    __name(this, "GraphQLClient");
  }
  async request(params) {
    const query = typeof params.document === "string" ? params.document : params.document.loc?.source.body;
    if (!query) {
      throw new Error("Invalid GraphQL document: no query string found");
    }
    const response = await fetch(this.url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...this.options.headers,
        ...params.requestHeaders
      },
      body: JSON.stringify({ query, variables: params.variables }),
      signal: params.signal
    });
    if (!response.ok) {
      const body = await response.text().catch(() => "");
      throw new Error(`HTTP ${response.status}: ${response.statusText}${body ? `
${body}` : ""}`);
    }
    const json = await response.json();
    if (json.errors?.length) {
      const summary = json.errors.map((e) => e.message).join("; ");
      const err = new Error(`GraphQL Error: ${summary}`);
      err.graphqlErrors = json.errors;
      err.response = json;
      throw err;
    }
    if (!json.data) {
      throw new Error("GraphQL response missing data");
    }
    return json.data;
  }
};

// src/sync/cli/graphql-tag-shim.ts
function gql(strings, ...values) {
  let result = strings[0];
  for (let i = 0; i < values.length; i++) {
    result += String(values[i]) + strings[i + 1];
  }
  return {
    loc: {
      source: {
        body: result
      }
    }
  };
}
__name(gql, "gql");

// src/graphql.ts
var FetchServerHashesDocument = gql`
    query FetchServerHashes {
  notePaths {
    path: value
    hash: latestContentHash
  }
}
    `;
var FetchPublishedUrlsDocument = gql`
    query FetchPublishedUrls {
  notePaths {
    path: value
    latestNoteView {
      url
    }
  }
}
    `;
var FetchAllWarningsDocument = gql`
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
    `;
var FetchNoteContentsDocument = gql`
    query FetchNoteContents($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    path: value
    content
  }
}
    `;
var FetchNoteAssetsDocument = gql`
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
    `;
var PushNotesDocument = gql`
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
    `;
var HideNotesDocument = gql`
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
    `;
var UploadNoteAssetDocument = gql`
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
    `;
var CommitNotesDocument = gql`
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
    `;
var defaultWrapper = /* @__PURE__ */ __name((action, _operationName, _operationType, _variables) => action(), "defaultWrapper");
function getSdk(client, withWrapper = defaultWrapper) {
  return {
    FetchServerHashes(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: FetchServerHashesDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "FetchServerHashes", "query", variables);
    },
    FetchPublishedUrls(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: FetchPublishedUrlsDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "FetchPublishedUrls", "query", variables);
    },
    FetchAllWarnings(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: FetchAllWarningsDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "FetchAllWarnings", "query", variables);
    },
    FetchNoteContents(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: FetchNoteContentsDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "FetchNoteContents", "query", variables);
    },
    FetchNoteAssets(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: FetchNoteAssetsDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "FetchNoteAssets", "query", variables);
    },
    PushNotes(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: PushNotesDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "PushNotes", "mutation", variables);
    },
    HideNotes(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: HideNotesDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "HideNotes", "mutation", variables);
    },
    UploadNoteAsset(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: UploadNoteAssetDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "UploadNoteAsset", "mutation", variables);
    },
    CommitNotes(variables, requestHeaders, signal) {
      return withWrapper((wrappedRequestHeaders) => client.request({ document: CommitNotesDocument, variables, requestHeaders: { ...requestHeaders, ...wrappedRequestHeaders }, signal }), "CommitNotes", "mutation", variables);
    }
  };
}
__name(getSdk, "getSdk");

// src/sync/cli/client.ts
function createClient(options) {
  const client = new GraphQLClient(options.apiUrl, {
    headers: {
      "X-API-Key": options.apiKey
    }
  });
  return getSdk(client);
}
__name(createClient, "createClient");

// src/sync/cli/env.ts
var LEGACY_STATE_FILE = ".sync-state.json";
function stateFileNameForApiUrl(apiUrl) {
  if (!apiUrl) return LEGACY_STATE_FILE;
  try {
    const url = new URL(apiUrl);
    const sanitized = url.host.replace(/[^a-zA-Z0-9.-]/g, "_");
    if (!sanitized) return LEGACY_STATE_FILE;
    return `.sync-state.${sanitized}.json`;
  } catch {
    return LEGACY_STATE_FILE;
  }
}
__name(stateFileNameForApiUrl, "stateFileNameForApiUrl");
var NodeEnv = class {
  constructor(options) {
    this.pushBatchSize = 100;
    this.folder = path2.resolve(options.folder);
    this.prefix = options.prefix ? options.prefix.replace(/\/$/, "") : "";
    this.twoWaySync = options.twoWaySync;
    this.verbose = options.verbose ?? false;
    this.conflictResolution = options.conflictResolution ?? "local";
    this.publishField = options.publishField ?? "";
    this.meta = options.meta ?? {};
    this.apiUrl = options.apiUrl;
    this.apiKey = options.apiKey;
    if (options.stateFile) {
      this.statePath = path2.isAbsolute(options.stateFile) ? options.stateFile : path2.join(this.folder, options.stateFile);
    } else {
      this.statePath = path2.join(this.folder, stateFileNameForApiUrl(options.apiUrl));
    }
    this.syncState = this.loadSyncState();
    this.sdk = createClient({ apiUrl: options.apiUrl, apiKey: options.apiKey });
  }
  static {
    __name(this, "NodeEnv");
  }
  /** Add prefix to local path for remote path */
  toRemotePath(localPath) {
    return this.prefix ? `${this.prefix}/${localPath}` : localPath;
  }
  /** Remove prefix from remote path to get local path */
  toLocalPath(remotePath) {
    if (this.prefix && remotePath.startsWith(this.prefix + "/")) {
      return remotePath.substring(this.prefix.length + 1);
    }
    return remotePath;
  }
  /** Check if remote path belongs to this prefix */
  matchesPrefix(remotePath) {
    if (!this.prefix) return true;
    return remotePath.startsWith(this.prefix + "/");
  }
  loadSyncState() {
    try {
      if (fs.existsSync(this.statePath)) {
        const data = fs.readFileSync(this.statePath, "utf-8");
        return JSON.parse(data);
      }
    } catch (e) {
      this.log(`Warning: Could not load sync state: ${e}`);
    }
    return { files: {} };
  }
  log(message) {
    if (this.verbose) {
      console.log(message);
    }
  }
  // ============ ClassifyEnv ============
  async getLocalFiles() {
    const files = [];
    const walk = /* @__PURE__ */ __name((dir) => {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        if (entry.name.startsWith(".") || entry.name === "node_modules") {
          continue;
        }
        const fullPath = path2.join(dir, entry.name);
        if (entry.isDirectory()) {
          walk(fullPath);
        } else if (entry.isFile()) {
          const ext = path2.extname(entry.name).toLowerCase();
          if (ext === ".md" || ext === ".html" || ext === ".canvas" || ext === ".base" || ext === ".excalidraw" || entry.name.endsWith(".html.json")) {
            const stat4 = fs.statSync(fullPath);
            const relPath = path2.relative(this.folder, fullPath);
            files.push({
              // Use remote path with prefix for sync comparison
              path: this.toRemotePath(relPath),
              mtime: stat4.mtimeMs
            });
          }
        }
      }
    }, "walk");
    walk(this.folder);
    return files;
  }
  async getServerHashes() {
    try {
      const result = await this.sdk.FetchServerHashes();
      return result.notePaths.filter((np) => this.matchesPrefix(np.path)).map((np) => ({
        path: np.path,
        hash: np.hash
      }));
    } catch (e) {
      console.error(`\u274C Failed to fetch server hashes: ${e}`);
      return [];
    }
  }
  getSyncState() {
    return this.syncState;
  }
  async computeHash(content) {
    const hash = crypto.createHash("sha256").update(content, "utf-8").digest();
    const b64 = hash.toString("base64");
    return b64.replace(/\+/g, "-").replace(/\//g, "_");
  }
  async readFileContent(filePath) {
    const localPath = this.toLocalPath(filePath);
    const fullPath = path2.join(this.folder, localPath);
    return fs.readFileSync(fullPath, "utf-8");
  }
  // ============ File Operations ============
  async writeFile(filePath, content) {
    const fullPath = path2.join(this.folder, filePath);
    fs.writeFileSync(fullPath, content, "utf-8");
  }
  async writeBinaryFile(filePath, data) {
    const fullPath = path2.join(this.folder, filePath);
    fs.writeFileSync(fullPath, Buffer.from(data));
  }
  async readBinaryFile(filePath) {
    const fullPath = path2.join(this.folder, filePath);
    const buffer = fs.readFileSync(fullPath);
    return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
  }
  async deleteFile(filePath) {
    const fullPath = path2.join(this.folder, filePath);
    if (fs.existsSync(fullPath)) {
      fs.unlinkSync(fullPath);
    }
  }
  async createFolder(folderPath) {
    const fullPath = path2.join(this.folder, folderPath);
    fs.mkdirSync(fullPath, { recursive: true });
  }
  async fileExists(filePath) {
    return this.fileExistsSync(filePath);
  }
  fileExistsSync(filePath) {
    const fullPath = path2.join(this.folder, filePath);
    return fs.existsSync(fullPath);
  }
  // ============ Server Operations ============
  async pushNotes(updates, skipCommit) {
    if (updates.length === 0) {
      return [];
    }
    const processedUpdates = updates.map((u) => ({
      path: u.path,
      content: this.injectMeta(u.content)
    }));
    if (this.publishField) {
      for (const update of processedUpdates) {
        if (!this.hasPublishFieldInContent(update.content, update.path)) {
          throw new Error(
            `[Security] Attempted to push note "${update.path}" without publish field "${this.publishField}". This is a bug in the sync logic - please report it.`
          );
        }
      }
    }
    try {
      const result = await this.sdk.PushNotes({
        input: {
          updates: processedUpdates.map((u) => ({
            path: u.path,
            content: u.content
          })),
          skipCommit
        }
      });
      if ("message" in result.pushNotes) {
        throw new Error(`Push failed: ${result.pushNotes.message}`);
      }
      console.log(`\u2705 Pushed ${updates.length} notes`);
      const urlMap = new Map(
        (result.pushNotes.updated ?? []).map((u) => [u.path, u.url ?? null])
      );
      return result.pushNotes.notes.map((n) => ({
        id: String(n.id),
        path: n.path,
        assets: n.assets.map((a) => ({
          path: a.path,
          sha256Hash: a.sha256Hash ?? null,
          absolutePath: a.absolutePath ?? null,
          url: a.url ?? null
        })),
        url: urlMap.get(n.path) ?? null
      }));
    } catch (e) {
      const paths = processedUpdates.map((u) => u.path).join(", ");
      console.error(`\u274C Failed to push notes (batch paths: ${paths}):`);
      console.error(e);
      const anyE = e;
      if (anyE.response) {
        console.error("   response:", JSON.stringify(anyE.response, null, 2));
      }
      if (anyE.request) {
        console.error("   request:", JSON.stringify(anyE.request, null, 2));
      }
      console.error("   own props:", Object.getOwnPropertyNames(e));
      return [];
    }
  }
  async hideNotes(paths) {
    if (paths.length === 0) {
      return;
    }
    try {
      const result = await this.sdk.HideNotes({
        input: { paths }
      });
      if ("message" in result.hideNotes) {
        throw new Error(`Hide failed: ${result.hideNotes.message}`);
      }
      console.log(`\u2705 Hidden ${paths.length} notes`);
    } catch (e) {
      console.error(`\u274C Failed to hide notes: ${e}`);
    }
  }
  async fetchNoteContents(paths) {
    if (paths.length === 0) {
      return [];
    }
    try {
      const result = await this.sdk.FetchNoteContents({
        filter: { paths }
      });
      return result.notePaths.map((np) => ({
        path: np.path,
        content: np.content
      }));
    } catch (e) {
      console.error(`\u274C Failed to fetch note contents: ${e}`);
      return [];
    }
  }
  async fetchNoteAssets(paths) {
    if (paths.length === 0) {
      return [];
    }
    try {
      const result = await this.sdk.PushNotes({
        input: { updates: [] }
      });
      if ("message" in result.pushNotes) {
        console.error(`\u274C Failed to fetch note assets: ${result.pushNotes.message}`);
        return [];
      }
      const pathSet = new Set(paths);
      return result.pushNotes.notes.filter((note) => pathSet.has(note.path)).map((note) => ({
        path: note.path,
        noteId: String(note.id),
        // version ID for upload
        assets: note.assets.map((a) => ({
          id: a.path,
          // relative path used as asset identifier
          url: a.url,
          hash: a.sha256Hash ?? "",
          // empty string for null (not uploaded)
          absolutePath: a.absolutePath
        }))
      }));
    } catch (e) {
      console.error(`\u274C Failed to fetch note assets: ${e}`);
      return [];
    }
  }
  async uploadAsset(params) {
    return uploadWithRetry(() => this.uploadAssetOnce(params), {
      // Preserve the CLI's original no-delay retry behavior.
      sleep: /* @__PURE__ */ __name(async () => {
      }, "sleep"),
      onRetry: /* @__PURE__ */ __name((attempt) => this.log(`\u26A0\uFE0F Upload attempt ${attempt} failed, retrying: ${params.relativePath}`), "onRetry"),
      onGiveUp: /* @__PURE__ */ __name((e) => console.error(`\u274C Failed to upload asset ${params.relativePath}: ${e}`), "onGiveUp")
    });
  }
  async uploadAssetOnce(params) {
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
          skipCommit: true,
          // batch: skip per-upload PrepareLatestNotes; executePlan commits once at the end
          file: null,
          // Will be replaced by multipart map
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
    const response = await fetch(this.apiUrl, {
      method: "POST",
      headers: {
        "X-API-Key": this.apiKey
      },
      body: formData
    });
    if (response.status === 413) {
      throw new AssetTooLargeError(params.fileName);
    }
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
      this.log(`\u23E9 Asset skipped (already exists): ${params.relativePath}`);
    } else {
      console.log(`\u2705 Asset uploaded: ${params.relativePath}`);
    }
    return true;
  }
  async downloadAsset(url) {
    try {
      const response = await fetch(url);
      if (!response.ok) {
        console.error(`\u274C Failed to download asset: HTTP ${response.status}`);
        return null;
      }
      return await response.arrayBuffer();
    } catch (e) {
      console.error(`\u274C Failed to download asset from ${url}: ${e}`);
      return null;
    }
  }
  async commitNotes() {
    try {
      const result = await this.sdk.CommitNotes();
      if ("message" in result.commitNotes) {
        throw new Error(`Commit failed: ${result.commitNotes.message}`);
      }
      console.log("\u2705 Notes committed");
      return {
        updated: (result.commitNotes.updated ?? []).map((n) => ({
          path: n.path,
          url: n.url ?? "",
          warnings: (n.warnings ?? []).map((w) => ({ level: w.level, message: w.message }))
        }))
      };
    } catch (e) {
      console.error(`\u274C Failed to commit notes: ${e}`);
      return { updated: [] };
    }
  }
  // ============ State ============
  async saveSyncState(state) {
    state.lastSyncedAt = Date.now();
    fs.writeFileSync(this.statePath, JSON.stringify(state, null, 2), "utf-8");
    this.syncState = state;
  }
  // ============ Asset Operations ============
  async computeBinaryHash(data) {
    return crypto.createHash("sha256").update(Buffer.from(data)).digest("hex");
  }
  async resolveAssetPath(assetPath, notePath) {
    return resolveAssetPath(this, assetPath, notePath);
  }
  // ============ UI Callbacks (CLI versions) ============
  onProgress(progress) {
    if (this.verbose) {
      console.log(`  [${progress.step}] ${progress.current}/${progress.total}: ${progress.path ?? ""}`);
    }
  }
  async onConflict(conflicts) {
    if (this.conflictResolution === "fail") {
      console.error(`\u274C ${conflicts.length} conflicts detected:`);
      for (const c of conflicts) {
        console.error(`   - ${c.path}`);
      }
      throw new Error(`Conflicts detected and --conflict-resolution=fail is set`);
    }
    const resolution = this.cliToConflictResolution(this.conflictResolution);
    console.log(`\u26A0\uFE0F ${conflicts.length} conflicts detected, resolving with: ${this.conflictResolution}`);
    return conflicts.map(() => resolution);
  }
  async onAssetConflict(conflicts) {
    if (this.conflictResolution === "fail") {
      console.error(`\u274C ${conflicts.length} asset conflicts detected:`);
      for (const c of conflicts) {
        console.error(`   - ${c.path}`);
      }
      throw new Error(`Asset conflicts detected and --conflict-resolution=fail is set`);
    }
    const resolution = this.cliToAssetConflictResolution(this.conflictResolution);
    console.log(`\u26A0\uFE0F ${conflicts.length} asset conflicts detected, resolving with: ${this.conflictResolution}`);
    return conflicts.map(() => resolution);
  }
  cliToConflictResolution(cli) {
    switch (cli) {
      case "local":
        return "keep_local";
      case "remote":
        return "keep_remote";
      case "skip":
        return "skip";
      default:
        return "keep_local";
    }
  }
  cliToAssetConflictResolution(cli) {
    switch (cli) {
      case "local":
        return "keep_local";
      case "remote":
        return "keep_remote";
      case "skip":
        return "skip";
      default:
        return "keep_local";
    }
  }
  async onServerDeleted(paths) {
    console.log(`\u26A0\uFE0F ${paths.length} files deleted on server, keeping local copies`);
    return false;
  }
  async confirmPush(paths) {
    console.log(`\u{1F4E4} Pushing ${paths.length} files...`);
    return true;
  }
  /**
   * Inject meta fields into frontmatter.
   * - If no meta configured, returns content unchanged.
   * - If frontmatter exists, adds/updates fields.
   * - If no frontmatter, creates one with meta fields.
   */
  injectMeta(content) {
    if (Object.keys(this.meta).length === 0) {
      return content;
    }
    if (content.startsWith("---")) {
      const endIndex = content.indexOf("\n---", 3);
      if (endIndex !== -1) {
        let frontmatter = content.slice(4, endIndex);
        const afterFrontmatter = content.slice(endIndex + 4);
        for (const [key, value] of Object.entries(this.meta)) {
          const regex = new RegExp(`^${key}\\s*:.*$`, "m");
          if (regex.test(frontmatter)) {
            frontmatter = frontmatter.replace(regex, `${key}: ${value}`);
          } else {
            frontmatter = frontmatter.trimEnd() + `
${key}: ${value}`;
          }
        }
        return `---
${frontmatter}
---${afterFrontmatter}`;
      }
    }
    const metaLines = Object.entries(this.meta).map(([key, value]) => `${key}: ${value}`).join("\n");
    return `---
${metaLines}
---
${content}`;
  }
  /**
   * Check if content has any of the publish fields with a truthy value in frontmatter.
   * Parses YAML frontmatter from the content string.
   */
  hasPublishFieldInContent(content, path5) {
    if (!this.publishField) return true;
    if (isAlwaysPublishable(path5)) return true;
    if (!content.startsWith("---")) return false;
    const endIndex = content.indexOf("\n---", 3);
    if (endIndex === -1) return false;
    const frontmatterText = content.slice(4, endIndex);
    const fields = this.publishField.split(",").map((f) => f.trim()).filter((f) => f);
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
__name(classifyFile, "classifyFile");
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
  for (const path5 of allPaths) {
    const localHash = localHashes.get(path5) || null;
    const remoteHash = serverHashMap.get(path5) || null;
    const lastSyncedHash = syncState.files[path5] || null;
    const action = classifyFile(localHash, remoteHash, lastSyncedHash);
    const classification = {
      path: path5,
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
__name(classifySync, "classifySync");

// src/sync/filter.ts
function filterPlan(plan, options) {
  const { twoWaySync, prune, hasPublishFields, isExcluded } = options;
  const isPublishable = /* @__PURE__ */ __name((path5) => {
    if (!hasPublishFields) return true;
    return hasPublishFields(path5);
  }, "isPublishable");
  const excluded = /* @__PURE__ */ __name((path5) => {
    if (!isExcluded) return false;
    return isExcluded(path5);
  }, "excluded");
  const filteredClassifications = [];
  const pulls = [];
  const pushes = [];
  const conflicts = [];
  const localOnly = [];
  const remoteOnly = [];
  const localDeleted = [];
  const serverDeleted = [];
  let unchanged = 0;
  for (const c of plan.classifications) {
    if (excluded(c.path)) {
      if (c.remoteHash !== null) {
        const asHide = c.action === "local_deleted" ? c : { ...c, action: "local_deleted" };
        filteredClassifications.push(asHide);
        localDeleted.push(asHide);
      }
      continue;
    }
    const publishable = isPublishable(c.path);
    switch (c.action) {
      case "unchanged":
        filteredClassifications.push(c);
        unchanged++;
        break;
      case "pull":
        if (twoWaySync && publishable) {
          filteredClassifications.push(c);
          pulls.push(c);
        }
        break;
      case "push":
        if (publishable) {
          filteredClassifications.push(c);
          pushes.push(c);
        }
        break;
      case "conflict":
        if (!twoWaySync) {
          if (publishable) {
            const asPush = { ...c, action: "push" };
            filteredClassifications.push(asPush);
            pushes.push(asPush);
          }
        } else if (publishable) {
          filteredClassifications.push(c);
          conflicts.push(c);
        }
        break;
      case "local_only":
        if (publishable) {
          filteredClassifications.push(c);
          localOnly.push(c);
        }
        break;
      case "remote_only":
        if (twoWaySync) {
          filteredClassifications.push(c);
          remoteOnly.push(c);
        } else if (prune) {
          const asHide = { ...c, action: "local_deleted" };
          filteredClassifications.push(asHide);
          localDeleted.push(asHide);
        }
        break;
      case "local_deleted":
        if (publishable) {
          filteredClassifications.push(c);
          localDeleted.push(c);
        }
        break;
      case "server_deleted":
        if (twoWaySync) {
          filteredClassifications.push(c);
          serverDeleted.push(c);
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
__name(filterPlan, "filterPlan");

// src/sync/execute.ts
async function executePlan(env, plan, options = { twoWaySync: false }) {
  const result = {
    pulled: 0,
    pushed: 0,
    conflictsResolved: 0,
    assetsUploaded: 0,
    assetsDownloaded: 0,
    errors: [],
    updatedUrls: [],
    warnings: []
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
    const unchangedServerPaths = plan.classifications.filter((c) => c.action === "unchanged" && c.remoteHash !== null).map((c) => c.path);
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
    const confirmed = await env.confirmPush(toPush.map((c) => c.path));
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
    const notes = pushedNotes.map((note) => ({
      id: note.id,
      path: note.path,
      assets: (note.assets ?? []).map((a) => ({
        id: a.path,
        serverHash: a.sha256Hash,
        serverUrl: a.url
      }))
    }));
    const assetResult = await reconcileAssets(env, notes, options.twoWaySync);
    result.assetsUploaded = assetResult.uploaded;
    result.assetsDownloaded = assetResult.downloaded;
    result.errors.push(...assetResult.errors);
  }
  const unchangedPaths = plan.classifications.filter((c) => c.action === "unchanged" && c.remoteHash !== null).map((c) => c.path);
  if (unchangedPaths.length > 0) {
    const assetResult = await reconcileAssetsForUnchangedNotes(env, unchangedPaths);
    result.assetsUploaded += assetResult.uploaded;
    result.errors.push(...assetResult.errors);
  }
  if (result.pushed > 0 || result.assetsUploaded > 0) {
    const commitResult = await env.commitNotes();
    result.updatedUrls = commitResult.updated.map(({ path: path5, url }) => ({ path: path5, url }));
    for (const note of commitResult.updated) {
      for (const w of note.warnings) {
        result.warnings.push({ path: note.path, level: w.level, message: w.message });
      }
    }
  }
  await env.saveSyncState(syncState);
  return result;
}
__name(executePlan, "executePlan");
async function localFileIsNonEmpty(env, path5) {
  if (!await env.fileExists(path5)) {
    return false;
  }
  try {
    const local = await env.readFileContent(path5);
    return local.trim() !== "";
  } catch {
    return false;
  }
}
__name(localFileIsNonEmpty, "localFileIsNonEmpty");
async function executePulls(env, pulls, syncState) {
  if (pulls.length === 0) {
    return { count: 0, errors: [], pulledPaths: [] };
  }
  const paths = pulls.map((p) => p.path);
  const errors = [];
  const pulledPaths = [];
  let count = 0;
  const contents = await env.fetchNoteContents(paths);
  const contentMap = new Map(contents.map((c) => [c.path, c.content]));
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
    const fetchedHash = content.trim() === "" ? await env.computeHash(content) : null;
    if (fetchedHash !== null && fetchedHash !== pull.remoteHash && await localFileIsNonEmpty(env, pull.path)) {
      errors.push(`Refused to overwrite non-empty ${pull.path} with empty server content (hash mismatch)`);
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
    } catch (e) {
      errors.push(`Failed to write ${pull.path}: ${e}`);
    }
  }
  return { count, errors, pulledPaths };
}
__name(executePulls, "executePulls");
async function executePushes(env, pushes, syncState) {
  if (pushes.length === 0) {
    return { count: 0, errors: [], pushedNotes: [], urls: [] };
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
    } catch (e) {
      errors.push(`Failed to read ${push.path}: ${e}`);
    }
  }
  if (updates.length === 0) {
    return { count: 0, errors, pushedNotes: [], urls: [] };
  }
  const updatePaths = new Set(updates.map((u) => u.path));
  const batchSize = env.pushBatchSize || 100;
  const pushedNotes = [];
  for (let i = 0; i < updates.length; i += batchSize) {
    const batch = updates.slice(i, i + batchSize);
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
  const urls = filteredNotes.filter((n) => typeof n.url === "string").map((n) => ({ path: n.path, url: n.url }));
  return { count: pushedCount, errors, pushedNotes: filteredNotes, urls };
}
__name(executePushes, "executePushes");
async function handleConflicts(env, conflicts, syncState) {
  if (conflicts.length === 0) {
    return { resolved: 0, errors: [] };
  }
  const errors = [];
  const paths = conflicts.map((c) => c.path);
  const remoteContents = await env.fetchNoteContents(paths);
  const remoteMap = new Map(remoteContents.map((c) => [c.path, c.content]));
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
    } catch (e) {
      console.warn(`Failed to read local file for conflict ${conflict.path}:`, e);
      errors.push(`Failed to read local file for conflict: ${conflict.path}`);
    }
  }
  if (conflictInfos.length === 0) {
    return { resolved: 0, errors };
  }
  const resolutions = await env.onConflict(conflictInfos);
  let resolved = 0;
  for (let i = 0; i < conflictInfos.length; i++) {
    const info = conflictInfos[i];
    const resolution = resolutions[i] || "skip";
    try {
      await resolveConflict(env, info, resolution, syncState);
      if (resolution !== "skip") {
        resolved++;
      }
    } catch (e) {
      errors.push(`Failed to resolve conflict for ${info.path}: ${e}`);
    }
  }
  return { resolved, errors };
}
__name(handleConflicts, "handleConflicts");
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
__name(resolveConflict, "resolveConflict");
async function handleServerDeleted(env, serverDeleted, syncState) {
  if (serverDeleted.length === 0) {
    return;
  }
  const paths = serverDeleted.map((c) => c.path);
  const deleteLocally = await env.onServerDeleted(paths);
  if (deleteLocally) {
    for (const c of serverDeleted) {
      try {
        await env.deleteFile(c.path);
        delete syncState.files[c.path];
      } catch (e) {
        console.warn(`Failed to delete file ${c.path}:`, e);
      }
    }
  } else {
    for (const c of serverDeleted) {
      if (c.localHash) {
        syncState.files[c.path] = c.localHash;
      }
    }
  }
}
__name(handleServerDeleted, "handleServerDeleted");
async function handleLocalDeleted(env, localDeleted, syncState) {
  if (localDeleted.length === 0) {
    return;
  }
  const paths = localDeleted.map((c) => c.path);
  await env.hideNotes(paths);
  for (const path5 of paths) {
    delete syncState.files[path5];
  }
}
__name(handleLocalDeleted, "handleLocalDeleted");
async function reconcileAssets(env, notes, twoWaySync) {
  const result = {
    uploaded: 0,
    downloaded: 0,
    conflictsResolved: 0,
    errors: []
  };
  const toUpload = [];
  const toDownload = [];
  const conflicts = [];
  for (const note of notes) {
    for (const asset of note.assets) {
      const localPath = await env.resolveAssetPath(asset.id, note.path);
      if (!localPath) {
        continue;
      }
      const exists = await env.fileExists(localPath);
      const onServer = !!asset.serverHash && !!asset.serverUrl;
      if (!onServer) {
        if (!exists) {
          continue;
        }
        try {
          const localData = await env.readBinaryFile(localPath);
          const localHash = await env.computeBinaryHash(localData);
          toUpload.push({ noteId: note.id, assetId: asset.id, localPath, localHash });
        } catch (e) {
          result.errors.push(`Failed to read local asset ${localPath}: ${e}`);
        }
        continue;
      }
      if (!exists) {
        if (twoWaySync) {
          toDownload.push({ assetId: asset.id, url: asset.serverUrl, localPath });
        }
        continue;
      }
      try {
        const localData = await env.readBinaryFile(localPath);
        const localHash = await env.computeBinaryHash(localData);
        if (localHash === asset.serverHash) {
          continue;
        }
        conflicts.push({
          path: asset.id,
          absolutePath: localPath,
          noteId: note.id,
          localHash,
          remoteHash: asset.serverHash,
          remoteUrl: asset.serverUrl
        });
      } catch (e) {
        result.errors.push(`Failed to read local asset ${localPath}: ${e}`);
      }
    }
  }
  if (toUpload.length > 0) {
    const unique = /* @__PURE__ */ new Map();
    for (const item of toUpload) {
      const key = `${item.noteId}:${item.localPath}`;
      if (!unique.has(key)) {
        unique.set(key, item);
      }
    }
    const deduped = Array.from(unique.values());
    const uploadTotal = deduped.length;
    let uploadCurrent = 0;
    for (const item of deduped) {
      uploadCurrent++;
      env.onProgress({ step: "upload_asset", current: uploadCurrent, total: uploadTotal, path: item.assetId });
      try {
        const localData = await env.readBinaryFile(item.localPath);
        const blob = new Blob([localData]);
        const fileName = item.localPath.substring(item.localPath.lastIndexOf("/") + 1);
        const success = await env.uploadAsset({
          noteId: item.noteId,
          blob,
          fileName,
          relativePath: item.assetId,
          absolutePath: item.localPath,
          sha256Hash: item.localHash
        });
        if (success) {
          result.uploaded++;
        }
      } catch (e) {
        result.errors.push(`Failed to upload asset ${item.assetId}: ${e}`);
      }
    }
  }
  if (toDownload.length > 0) {
    const downloadTotal = toDownload.length;
    let downloadCurrent = 0;
    for (const item of toDownload) {
      downloadCurrent++;
      env.onProgress({ step: "download_asset", current: downloadCurrent, total: downloadTotal, path: item.assetId });
      try {
        const data = await env.downloadAsset(item.url);
        if (!data) {
          result.errors.push(`Failed to download asset ${item.assetId}`);
          continue;
        }
        const dirPath = item.localPath.substring(0, item.localPath.lastIndexOf("/"));
        if (dirPath) {
          await env.createFolder(dirPath);
        }
        await env.writeBinaryFile(item.localPath, data);
        result.downloaded++;
      } catch (e) {
        result.errors.push(`Failed to download asset ${item.assetId}: ${e}`);
      }
    }
  }
  if (conflicts.length > 0) {
    const assetResult = await handleAssetConflicts(env, conflicts, twoWaySync);
    result.uploaded += assetResult.uploaded;
    result.downloaded += assetResult.downloaded;
    result.conflictsResolved += assetResult.conflictsResolved;
    result.errors.push(...assetResult.errors);
  }
  return result;
}
__name(reconcileAssets, "reconcileAssets");
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
  for (let i = 0; i < conflicts.length; i++) {
    const conflict = conflicts[i];
    const resolution = resolutions[i] || "skip";
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
    } catch (e) {
      result.errors.push(`Failed to resolve asset conflict for ${conflict.path}: ${e}`);
    }
  }
  return result;
}
__name(handleAssetConflicts, "handleAssetConflicts");
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
    } catch (e) {
      result.errors.push(`Failed to download asset ${absolutePath}: ${e}`);
    }
  }
  return result;
}
__name(downloadAssetsForNotes, "downloadAssetsForNotes");
async function reconcileAssetsForUnchangedNotes(env, notePaths) {
  if (notePaths.length === 0) {
    return { uploaded: 0, downloaded: 0, conflictsResolved: 0, errors: [] };
  }
  const noteAssets = await env.fetchNoteAssets(notePaths);
  const notes = noteAssets.map((note) => ({
    id: note.noteId,
    path: note.path,
    assets: note.assets.map((a) => ({
      id: a.id,
      serverHash: a.hash || null,
      serverUrl: a.url || null
    }))
  }));
  return reconcileAssets(env, notes, false);
}
__name(reconcileAssetsForUnchangedNotes, "reconcileAssetsForUnchangedNotes");

// src/sync/exclude.ts
function makeExcludeMatcher(patterns) {
  const normalized = patterns.map((p) => p.trim().replace(/\/+$/, "")).filter((p) => p.length > 0);
  if (normalized.length === 0) {
    return () => false;
  }
  const regexes = normalized.map(patternToRegex);
  return (path5) => regexes.some((re) => re.test(path5));
}
__name(makeExcludeMatcher, "makeExcludeMatcher");
function patternToRegex(pattern) {
  if (/[*?]/.test(pattern)) {
    return new RegExp("^" + globToRegexSource(pattern) + "$");
  }
  return new RegExp("^" + escapeRegex(pattern) + "(?:/.*)?$");
}
__name(patternToRegex, "patternToRegex");
function globToRegexSource(pattern) {
  let out = "";
  for (let i = 0; i < pattern.length; i++) {
    const ch = pattern[i];
    if (ch === "*") {
      if (pattern[i + 1] === "*") {
        out += ".*";
        i++;
      } else {
        out += "[^/]*";
      }
    } else if (ch === "?") {
      out += "[^/]";
    } else {
      out += escapeRegex(ch);
    }
  }
  return out;
}
__name(globToRegexSource, "globToRegexSource");
function escapeRegex(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
__name(escapeRegex, "escapeRegex");

// src/sync/prune.ts
function summarizePrune(plan, prunedPlan) {
  return {
    paths: prunedPlan.localDeleted.map((c) => c.path),
    localPresent: plan.classifications.filter((c) => c.localHash !== null).length,
    serverPresent: plan.classifications.filter((c) => c.remoteHash !== null).length
  };
}
__name(summarizePrune, "summarizePrune");
function pruneNeedsForce(summary) {
  return summary.localPresent === 0 && summary.paths.length > 0;
}
__name(pruneNeedsForce, "pruneNeedsForce");

// src/sync/cli/watch.ts
import * as fs2 from "fs";
import * as path3 from "path";

// src/sync/live-apply.ts
async function applyLiveChanges(env, changes, syncState) {
  const pulledPaths = [];
  let conflictCount = 0;
  const hiddenPaths = [];
  const upserts = changes.filter(
    (c) => c.__typename === "NoteUpsertEvent"
  );
  const missingContentPaths = upserts.filter((c) => c.noteView === null).map((c) => c.path);
  const fetchedContent = /* @__PURE__ */ new Map();
  if (missingContentPaths.length > 0) {
    const contents = await env.fetchNoteContents(missingContentPaths);
    for (const c of contents) {
      fetchedContent.set(c.path, c.content);
    }
  }
  for (const change of changes) {
    if (change.__typename === "NoteHideEvent") {
      if (await env.fileExists(change.path) && syncState.files[change.path] !== void 0) {
        hiddenPaths.push(change.path);
      }
      continue;
    }
    const content = change.noteView?.content ?? fetchedContent.get(change.path);
    if (content === void 0) {
      continue;
    }
    const remoteHash = await env.computeHash(content);
    const exists = await env.fileExists(change.path);
    const localContent = exists ? await env.readFileContent(change.path) : null;
    const localHash = localContent !== null ? await env.computeHash(localContent) : null;
    const lastSynced = syncState.files[change.path] ?? null;
    if (isAlwaysPublishable(change.path) && content.trim() === "" && localContent !== null && localContent.trim() !== "") {
      conflictCount++;
      continue;
    }
    if (change.eventType === "create" && localHash === null) {
      await writeAndRecord(env, syncState, change.path, content, remoteHash, pulledPaths);
      continue;
    }
    const action = classifyFile(localHash, remoteHash, lastSynced);
    if (action === "pull") {
      await writeAndRecord(env, syncState, change.path, content, remoteHash, pulledPaths);
    } else if (action === "conflict") {
      conflictCount++;
    }
  }
  return { pulledPaths, conflictCount, hiddenPaths };
}
__name(applyLiveChanges, "applyLiveChanges");
async function writeAndRecord(env, syncState, path5, content, remoteHash, pulledPaths) {
  const dirPath = path5.substring(0, path5.lastIndexOf("/"));
  if (dirPath) {
    await env.createFolder(dirPath);
  }
  await env.writeFile(path5, content);
  syncState.files[path5] = remoteHash;
  pulledPaths.push(path5);
}
__name(writeAndRecord, "writeAndRecord");

// src/sync/LivePullConnection.ts
var SUBSCRIPTION_DOCUMENT = `subscription NoteChanges($filter: NoteChangesFilter!) {
	noteChanges(filter: $filter) {
		changes {
			__typename
			... on NoteUpsertEvent {
				path
				eventType
				versionId
				title
				noteView { path content }
			}
			... on NoteHideEvent {
				path
			}
		}
	}
}`;
var BACKOFF_MS = [3e3, 6e3, 12e3, 3e4];
var HEALTH_CHECK_INTERVAL_MS = 6e4;
var LivePullConnection = class {
  constructor(options) {
    this.abort = null;
    this.stopped = false;
    this.backoffIndex = 0;
    this.lastByteAt = 0;
    this.healthTimer = null;
    this.options = options;
  }
  static {
    __name(this, "LivePullConnection");
  }
  /** Start the self-reconnecting stream loop. Safe to call once. */
  connect() {
    if (!this.stopped && this.abort) {
      return;
    }
    this.stopped = false;
    this.startHealthCheck();
    void this.loop();
  }
  /** Stop the stream and cleanup. Safe to call multiple times. */
  disconnect() {
    this.stopped = true;
    this.stopHealthCheck();
    if (this.abort) {
      this.abort.abort();
      this.abort = null;
    }
  }
  /**
   * If the stream looks dead (no bytes for longer than the health interval),
   * abort the current connection so the loop reconnects. Used on window focus.
   */
  reconnectIfDead() {
    if (this.stopped) {
      return;
    }
    if (!this.abort) {
      this.connect();
      return;
    }
    if (Date.now() - this.lastByteAt > HEALTH_CHECK_INTERVAL_MS) {
      this.abort.abort();
    }
  }
  startHealthCheck() {
    this.stopHealthCheck();
    this.healthTimer = setInterval(() => {
      if (this.stopped || !this.abort) {
        return;
      }
      if (Date.now() - this.lastByteAt > HEALTH_CHECK_INTERVAL_MS) {
        this.abort.abort();
      }
    }, HEALTH_CHECK_INTERVAL_MS);
  }
  stopHealthCheck() {
    if (this.healthTimer !== null) {
      clearInterval(this.healthTimer);
      this.healthTimer = null;
    }
  }
  async loop() {
    while (!this.stopped) {
      try {
        await this.stream();
        this.backoffIndex = 0;
      } catch (e) {
        if (this.stopped) {
          return;
        }
        console.warn("[Trip2g Sync] Live-pull stream error:", e);
      }
      if (this.stopped) {
        return;
      }
      const delay = BACKOFF_MS[Math.min(this.backoffIndex, BACKOFF_MS.length - 1)];
      this.backoffIndex++;
      await new Promise((resolve5) => setTimeout(resolve5, delay));
    }
  }
  async stream() {
    const abort = new AbortController();
    this.abort = abort;
    this.lastByteAt = Date.now();
    const response = await fetch(this.options.endpoint ?? `${this.options.apiUrl}/_system/graphql`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
        "X-API-Key": this.options.apiKey,
        "X-Plugin-Version": this.options.pluginVersion
      },
      body: JSON.stringify({
        query: SUBSCRIPTION_DOCUMENT,
        variables: {
          filter: {
            includePatterns: this.options.includePatterns,
            excludePatterns: this.options.excludePatterns ?? []
          }
        }
      }),
      signal: abort.signal
    });
    if (!response.ok || !response.body) {
      throw new Error(`Live-pull: ${response.status} ${response.statusText}`);
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let eventType = "";
    this.backoffIndex = 0;
    this.lastByteAt = Date.now();
    this.options.onConnected();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }
        this.lastByteAt = Date.now();
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";
        for (const line of lines) {
          if (line.startsWith("event:")) {
            eventType = line.slice(6).trim();
          } else if (line.startsWith("data:")) {
            const payload = line.slice(5).trim();
            if (eventType === "next") {
              this.handleNext(payload);
            } else if (eventType === "complete") {
              return;
            }
            eventType = "";
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }
  handleNext(payload) {
    let parsed;
    try {
      parsed = JSON.parse(payload);
    } catch (e) {
      console.warn("[Trip2g Sync] Live-pull bad payload:", e);
      return;
    }
    if (parsed.errors) {
      console.warn("[Trip2g Sync] Live-pull subscription error:", parsed.errors);
      return;
    }
    const changes = parsed.data?.noteChanges?.changes;
    if (changes && changes.length > 0) {
      this.options.onChanges(changes);
    }
  }
};

// src/sync/cli/watch.ts
var FS_DEBOUNCE_MS = 500;
var FALLBACK_SWEEP_MS = 2e4;
var realClock = {
  setInterval: /* @__PURE__ */ __name((h, ms) => setInterval(h, ms), "setInterval"),
  clearInterval: /* @__PURE__ */ __name((h) => clearInterval(h), "clearInterval"),
  setTimeout: /* @__PURE__ */ __name((h, ms) => setTimeout(h, ms), "setTimeout"),
  clearTimeout: /* @__PURE__ */ __name((h) => clearTimeout(h), "clearTimeout")
};
var realSignals = /* @__PURE__ */ __name((handler) => {
  process.once("SIGINT", () => handler("SIGINT"));
  process.once("SIGTERM", () => handler("SIGTERM"));
}, "realSignals");
function log(message) {
  console.log(message);
}
__name(log, "log");
function readPluginVersion() {
  const candidates = [
    path3.join(process.cwd(), "manifest.json"),
    path3.join(__dirname, "..", "..", "..", "manifest.json")
  ];
  for (const file of candidates) {
    try {
      const data = JSON.parse(fs2.readFileSync(file, "utf-8"));
      if (typeof data?.version === "string" && data.version) {
        return data.version;
      }
    } catch {
    }
  }
  return "cli";
}
__name(readPluginVersion, "readPluginVersion");
function isWatchablePath(relPath) {
  const norm = relPath.split(path3.sep).join("/");
  if (norm.startsWith(".trip2g-memory/") || norm === ".trip2g-memory") {
    return false;
  }
  for (const segment of norm.split("/")) {
    if (segment === "node_modules") return false;
    if (segment.startsWith(".")) return false;
  }
  return true;
}
__name(isWatchablePath, "isWatchablePath");
function createLock() {
  let tail = Promise.resolve();
  return {
    run(fn) {
      const result = tail.then(() => fn());
      tail = result.then(
        () => void 0,
        () => void 0
      );
      return result;
    },
    /** Resolves when the current chain is drained. */
    idle() {
      return tail.then(() => void 0);
    }
  };
}
__name(createLock, "createLock");
async function runWatch(args, deps = {}) {
  const clock = deps.clock ?? realClock;
  const now = deps.now ?? Date.now;
  const env = deps.envFactory?.(args) ?? new NodeEnv({
    folder: args.folder,
    apiUrl: args.apiUrl,
    apiKey: args.apiKey,
    twoWaySync: true,
    verbose: args.verbose,
    conflictResolution: args.conflictResolution,
    meta: args.meta
  });
  const isExcluded = makeExcludeMatcher(args.exclude);
  const lock = createLock();
  let draining = false;
  let reconcileDone = false;
  let settled = false;
  const pendingSse = [];
  let livePullRef = null;
  let watcherRef = null;
  let debounceTimer = null;
  let sweepTimer = null;
  let resolveRun = null;
  log(`\u{1F440} watch: starting for ${args.folder}`);
  const drain = /* @__PURE__ */ __name(async (signal) => {
    if (settled) {
      return;
    }
    draining = true;
    log(`\u{1F6D1} watch: ${signal} received, draining\u2026`);
    if (debounceTimer !== null) {
      clock.clearTimeout(debounceTimer);
      debounceTimer = null;
    }
    if (sweepTimer !== null) {
      clock.clearInterval(sweepTimer);
      sweepTimer = null;
    }
    if (livePullRef) {
      livePullRef.disconnect();
    }
    if (watcherRef) {
      try {
        await watcherRef.close();
      } catch {
      }
    }
    await lock.idle();
    await env.saveSyncState(env.getSyncState());
    settled = true;
    log("\u2705 watch: drained, exiting");
    if (resolveRun) {
      resolveRun({ exitCode: 0 });
    }
  }, "drain");
  const registerSignals = deps.signals ?? realSignals;
  registerSignals((signal) => void drain(signal));
  const applyEnv = env;
  const handleChanges = /* @__PURE__ */ __name((changes) => {
    if (!reconcileDone) {
      pendingSse.push(changes);
      return;
    }
    if (draining) {
      return;
    }
    void lock.run(async () => {
      const syncState = env.getSyncState();
      const result = await applyLiveChanges(applyEnv, changes, syncState);
      for (const hidePath of result.hiddenPaths) {
        await env.deleteFile(hidePath);
        delete syncState.files[hidePath];
      }
      if (result.pulledPaths.length > 0) {
        log(`\u{1F4E5} watch: pulled ${result.pulledPaths.length} note(s) from server`);
      }
      if (result.hiddenPaths.length > 0) {
        log(`\u{1F5D1}\uFE0F  watch: removed ${result.hiddenPaths.length} hidden note(s)`);
      }
      await env.saveSyncState(syncState);
    });
  }, "handleChanges");
  const patterns = resolveWatchPatterns(args);
  const pluginVersion = readPluginVersion();
  const livePullFactory = deps.livePullFactory ?? ((opts) => new LivePullConnection(opts));
  const livePull = livePullFactory({
    apiUrl: args.apiUrl,
    apiKey: args.apiKey,
    pluginVersion,
    endpoint: args.apiUrl,
    includePatterns: patterns.include,
    excludePatterns: patterns.exclude,
    onConnected: /* @__PURE__ */ __name(() => log("\u{1F50C} watch: live-pull connected"), "onConnected"),
    onChanges: handleChanges
  });
  livePullRef = livePull;
  await lock.run(async () => {
    log("\u{1F4CA} watch: reconciling\u2026");
    await runOneShot(env, isExcluded);
    reconcileDone = true;
    log("\u2705 watch: reconcile complete");
  });
  if (draining) {
    return { exitCode: 0 };
  }
  livePull.connect();
  if (pendingSse.length > 0) {
    const queued = pendingSse.splice(0, pendingSse.length);
    for (const batch of queued) {
      handleChanges(batch);
    }
  }
  const dirtyPaths = /* @__PURE__ */ new Set();
  const flushBatch = /* @__PURE__ */ __name(() => {
    debounceTimer = null;
    if (draining) {
      return;
    }
    if (dirtyPaths.size === 0) {
      return;
    }
    const batch = Array.from(dirtyPaths);
    dirtyPaths.clear();
    void lock.run(async () => {
      log(`\u{1F4E4} watch: pushing ${batch.length} local change(s)`);
      await runOneShot(env, isExcluded);
    });
  }, "flushBatch");
  const onFsChange = /* @__PURE__ */ __name((filePath) => {
    if (draining) {
      return;
    }
    const rel = path3.isAbsolute(filePath) ? path3.relative(path3.resolve(args.folder), filePath) : filePath;
    if (!isWatchablePath(rel)) {
      return;
    }
    dirtyPaths.add(rel);
    if (debounceTimer !== null) {
      clock.clearTimeout(debounceTimer);
    }
    debounceTimer = clock.setTimeout(flushBatch, FS_DEBOUNCE_MS);
  }, "onFsChange");
  const { watcher, usingFallback } = await createWatcher(args.folder, deps);
  watcherRef = watcher;
  watcher.onChange(onFsChange);
  if (usingFallback) {
    log("\u26A0\uFE0F  watch: chokidar unavailable, using fs.watch + periodic sweep (best-effort)");
    sweepTimer = clock.setInterval(() => {
      if (draining) {
        return;
      }
      void lock.run(async () => {
        await runOneShot(env, isExcluded);
      });
    }, FALLBACK_SWEEP_MS);
  }
  void now;
  if (settled) {
    return { exitCode: 0 };
  }
  return await new Promise((resolve5) => {
    resolveRun = resolve5;
  });
}
__name(runWatch, "runWatch");
function resolveWatchPatterns(args) {
  const include = args.include.length > 0 ? args.include : args.dataInclude && args.dataInclude.length > 0 ? args.dataInclude : ["**"];
  const exclude = args.exclude.length > 0 ? args.exclude : args.dataExclude && args.dataExclude.length > 0 ? args.dataExclude : [];
  return { include, exclude };
}
__name(resolveWatchPatterns, "resolveWatchPatterns");
async function runOneShot(env, isExcluded) {
  const plan = await classifySync(env);
  const filtered = filterPlan(plan, { twoWaySync: true, isExcluded });
  await executePlan(env, filtered, { twoWaySync: true });
}
__name(runOneShot, "runOneShot");
async function createWatcher(folder, deps) {
  if (deps.watcherFactory) {
    return { watcher: deps.watcherFactory(folder), usingFallback: false };
  }
  try {
    const importChokidar = deps.importChokidar ?? (() => Promise.resolve().then(() => (init_esm2(), esm_exports)));
    const chokidarMod = await importChokidar();
    const chokidar = chokidarMod.default ?? chokidarMod;
    if (typeof chokidar?.watch !== "function") {
      throw new Error("chokidar.watch unavailable");
    }
    const fsw = chokidar.watch(folder, {
      ignoreInitial: true,
      ignored: /(^|[/\\])\../
      // dotfiles
    });
    const watcher = {
      onChange(listener) {
        fsw.on("add", listener);
        fsw.on("change", listener);
        fsw.on("unlink", listener);
      },
      close() {
        return fsw.close();
      }
    };
    return { watcher, usingFallback: false };
  } catch {
    const resolved = path3.resolve(folder);
    const listeners = [];
    const fsWatcher = fs2.watch(resolved, { recursive: true }, (_event, filename) => {
      if (!filename) return;
      const rel = typeof filename === "string" ? filename : filename.toString();
      for (const l of listeners) {
        l(rel);
      }
    });
    const watcher = {
      onChange(listener) {
        listeners.push(listener);
      },
      close() {
        fsWatcher.close();
      }
    };
    return { watcher, usingFallback: true };
  }
}
__name(createWatcher, "createWatcher");

// src/sync/cli/cmd.ts
function readDataJson() {
  try {
    const dataPath = path4.join(process.cwd(), ".obsidian", "plugins", "trip2g", "data.json");
    const data = JSON.parse(fs3.readFileSync(dataPath, "utf8"));
    const dir = data?.syncDirs?.[0];
    if (!dir) return {};
    return {
      apiUrl: dir.apiUrl ? `${dir.apiUrl}/_system/graphql` : void 0,
      apiKey: dir.apiKey || void 0,
      livePullIncludePatterns: dir.livePullIncludePatterns ?? void 0,
      livePullExcludePatterns: dir.livePullExcludePatterns ?? void 0
    };
  } catch {
    return {};
  }
}
__name(readDataJson, "readDataJson");
function parseArgs() {
  const args = process.argv.slice(2);
  const dataJson = readDataJson();
  const result = {
    folder: "",
    prefix: "",
    apiUrl: process.env.TRIP2G_ENDPOINT || process.env.ENDPOINT || dataJson.apiUrl || "http://localhost:8081/_system/graphql",
    apiKey: process.env.TRIP2G_API_KEY || process.env.API_KEY || dataJson.apiKey || "",
    twoWaySync: false,
    watch: false,
    verbose: false,
    dryRun: false,
    prune: false,
    force: false,
    conflictResolution: "local",
    meta: {},
    updatedOutput: "",
    exclude: [],
    include: [],
    stateFile: ""
  };
  const positionalArgs = [];
  for (let i = 0; i < args.length; i++) {
    let arg = args[i];
    let value;
    if (arg.includes("=") && arg.startsWith("-")) {
      const eqIndex = arg.indexOf("=");
      value = arg.substring(eqIndex + 1);
      arg = arg.substring(0, eqIndex);
    }
    switch (arg) {
      case "--api-url":
      case "-u":
        result.apiUrl = value ?? args[++i];
        break;
      case "--api-key":
      case "-k":
        result.apiKey = value ?? args[++i];
        break;
      case "--two-way":
      case "-2":
        result.twoWaySync = true;
        break;
      case "--watch":
      case "-w":
        result.watch = true;
        result.twoWaySync = true;
        break;
      case "--include":
      case "-i": {
        const includeValue = value ?? args[++i];
        if (includeValue) {
          result.include.push(includeValue);
        }
        break;
      }
      case "--verbose":
      case "-v":
        result.verbose = true;
        break;
      case "--dry-run":
      case "-n":
        result.dryRun = true;
        break;
      case "--prune":
      case "--mirror":
        result.prune = true;
        break;
      case "--force":
        result.force = true;
        break;
      case "--conflict-resolution":
      case "-c": {
        const crValue = value ?? args[++i];
        if (crValue === "local" || crValue === "remote" || crValue === "skip" || crValue === "fail") {
          result.conflictResolution = crValue;
        } else {
          console.error(`\u274C Invalid conflict resolution: ${crValue}. Use: local, remote, skip, fail`);
          process.exit(1);
        }
        break;
      }
      case "--meta":
      case "-m": {
        const metaValue = value ?? args[++i];
        if (metaValue && metaValue.includes("=")) {
          const eqIndex = metaValue.indexOf("=");
          const metaKey = metaValue.substring(0, eqIndex);
          const metaVal = metaValue.substring(eqIndex + 1);
          result.meta[metaKey] = metaVal;
        } else {
          console.error(`\u274C Invalid --meta format: ${metaValue}. Use: --meta key=value`);
          process.exit(1);
        }
        break;
      }
      case "--updated-output":
      case "-o":
        result.updatedOutput = value ?? args[++i];
        break;
      case "--state-file":
      case "-s":
        result.stateFile = value ?? args[++i];
        break;
      case "--exclude":
      case "-x": {
        const excludeValue = value ?? args[++i];
        if (excludeValue) {
          result.exclude.push(excludeValue);
        }
        break;
      }
      case "--help":
      case "-h":
        printHelp();
        process.exit(0);
        break;
      default:
        if (!arg.startsWith("-")) {
          positionalArgs.push(arg);
        }
    }
  }
  if (positionalArgs.length >= 1) {
    result.folder = positionalArgs[0];
  }
  if (positionalArgs.length >= 2) {
    result.prefix = positionalArgs[1];
  }
  return result;
}
__name(parseArgs, "parseArgs");
function printHelp() {
  console.log(`
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
  -w, --watch              Watch mode: stream live changes from server via SSE
                           (implies --two-way; prefix not allowed in this mode)
  -i, --include <glob>     Include only matching paths in live-pull (can be repeated).
                           Flags take priority over data.json livePullIncludePatterns.
                           Default when none specified: ** (follow everything)
  -c, --conflict-resolution <mode>
                           How to resolve conflicts (default: local)
                           - local:  Keep local version, push to server
                           - remote: Keep remote version, overwrite local
                           - skip:   Skip conflicting files
                           - fail:   Exit with error on first conflict
  -m, --meta <key=value>   Add/override frontmatter field for all files (can be repeated)
  -o, --updated-output <file>
                           Write pushed notes as JSON [{path, url}] to file after sync
  -s, --state-file <path>  Sync-state file path (default: .sync-state.<host>.json derived from --api-url)
  -x, --exclude <glob>     Exclude paths from sync (can be repeated). Excluded
                           paths are never pushed; if they exist on the server
                           they are hidden. A bare name like "dev" matches that
                           directory and everything under it. In --watch mode,
                           flags take priority over data.json livePullExcludePatterns.
                           Default: none.
      --prune, --mirror    Server-truth deletion (rsync --delete semantics):
                           hide every server note under the synced prefix that
                           is NOT present locally, even ones the local
                           sync-state has no record of. Fixes orphaned server
                           notes left behind after a sync-state reset/replace
                           (they are classified remote_only and normally
                           ignored, so they are never hidden). Opt-in; without
                           it behavior is 100% unchanged. Prints a loud summary
                           before hiding and honors --dry-run. Refuses to run
                           when the local tree is empty but the server has notes
                           (partial/reset copy) unless --force is also given.
      --force              Allow --prune even when the local tree looks empty.
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

  # Mirror: hide any server note not present locally (orphaned-note cleanup)
  trip2g-sync ./docs --prune --dry-run   # preview what would be hidden
  trip2g-sync ./docs --prune             # actually hide them

  # Multi-repo setup: each repo pushes to different folder with different meta
  trip2g-sync ./docs docs --meta subgraph=docs
  trip2g-sync ./blog blog --meta subgraph=blog
  trip2g-sync ./wiki wiki --meta subgraph=team-wiki
`);
}
__name(printHelp, "printHelp");
async function cmdWarnings() {
  const dataJson = readDataJson();
  const apiUrl = process.env.TRIP2G_ENDPOINT || process.env.ENDPOINT || dataJson.apiUrl || "http://localhost:8081/_system/graphql";
  const apiKey = process.env.TRIP2G_API_KEY || process.env.API_KEY || dataJson.apiKey || "";
  if (!apiKey) {
    console.error("\u274C TRIP2G_API_KEY or API_KEY required");
    process.exit(1);
  }
  const sdk = createClient({ apiUrl, apiKey });
  const data = await sdk.FetchAllWarnings();
  const result = [];
  for (const item of data.notePaths) {
    const view = item.latestNoteView;
    if (!view) continue;
    for (const w of view.warnings ?? []) {
      result.push({ path: item.path, level: w.level, message: w.message, url: view.url ?? "" });
    }
  }
  console.log(JSON.stringify(result, null, 2));
}
__name(cmdWarnings, "cmdWarnings");
async function main() {
  if (process.argv[2] === "warnings") {
    await cmdWarnings();
    return;
  }
  const args = parseArgs();
  if (!args.folder) {
    console.error("\u274C Error: --folder is required");
    printHelp();
    process.exit(1);
  }
  if (!args.apiKey) {
    console.error("\u274C Error: --api-key or API_KEY environment variable is required");
    process.exit(1);
  }
  if (args.prefix && args.twoWaySync) {
    console.error("\u274C Error: prefix is not supported with --two-way sync");
    process.exit(1);
  }
  if (args.watch) {
    const dataJson = readDataJson();
    const { exitCode } = await runWatch({
      folder: args.folder,
      apiUrl: args.apiUrl,
      apiKey: args.apiKey,
      include: args.include,
      exclude: args.exclude,
      conflictResolution: args.conflictResolution,
      meta: args.meta,
      verbose: args.verbose,
      dataInclude: dataJson.livePullIncludePatterns,
      dataExclude: dataJson.livePullExcludePatterns
    });
    process.exit(exitCode);
  }
  if (args.dryRun) {
    console.log(`[dry-run] folder=${args.folder}${args.prefix ? ` prefix=${args.prefix}` : ""}`);
  }
  const env = new NodeEnv({
    folder: args.folder,
    prefix: args.prefix,
    apiUrl: args.apiUrl,
    apiKey: args.apiKey,
    twoWaySync: args.twoWaySync,
    verbose: args.verbose,
    conflictResolution: args.conflictResolution,
    meta: args.meta,
    stateFile: args.stateFile || void 0
  });
  console.log("\n\u{1F4CA} Classifying files...");
  const plan = await classifySync(env);
  const isExcluded = makeExcludeMatcher(args.exclude);
  const filteredPlan = filterPlan(plan, {
    twoWaySync: args.twoWaySync,
    prune: args.prune,
    isExcluded
  });
  if (args.exclude.length > 0) {
    console.log(`\u{1F6AB} Excluding: ${args.exclude.join(", ")}`);
  }
  if (args.prune) {
    const summary = summarizePrune(plan, filteredPlan);
    console.log(`
\u{1FA93} PRUNE: ${summary.paths.length} server notes not present locally will be hidden:`);
    for (const p of summary.paths) {
      console.log(`  ${p}`);
    }
    if (!args.force && pruneNeedsForce(summary)) {
      console.error(
        `
\u274C Refusing to prune: 0 local notes under the synced prefix but ${summary.serverPresent} on the server.
   This looks like a partial or reset local copy \u2014 pruning would wipe the server.
   Re-run with --force if this is intentional.`
      );
      if (!args.dryRun) {
        process.exit(1);
      }
    }
  }
  console.log("\n\u{1F4CB} Sync Plan:");
  console.log("-".repeat(40));
  console.log(`  Unchanged:      ${filteredPlan.unchanged}`);
  console.log(`  To push:        ${filteredPlan.pushes.length}`);
  console.log(`  Local only:     ${filteredPlan.localOnly.length}`);
  console.log(`  To pull:        ${filteredPlan.pulls.length}`);
  console.log(`  Remote only:    ${filteredPlan.remoteOnly.length}`);
  console.log(`  Conflicts:      ${filteredPlan.conflicts.length}`);
  console.log(`  Local deleted:  ${filteredPlan.localDeleted.length}`);
  console.log(`  Server deleted: ${filteredPlan.serverDeleted.length}`);
  console.log("-".repeat(40));
  if (args.verbose) {
    if (filteredPlan.pushes.length > 0) {
      console.log("\n\u{1F4E4} Files to push:");
      for (const f of filteredPlan.pushes) {
        console.log(`  ${f.path}`);
      }
    }
    if (filteredPlan.localOnly.length > 0) {
      console.log("\n\u{1F195} New local files:");
      for (const f of filteredPlan.localOnly) {
        console.log(`  ${f.path}`);
      }
    }
    if (filteredPlan.pulls.length > 0) {
      console.log("\n\u{1F4E5} Files to pull:");
      for (const f of filteredPlan.pulls) {
        console.log(`  ${f.path}`);
      }
    }
    if (filteredPlan.remoteOnly.length > 0) {
      console.log("\n\u{1F310} New remote files:");
      for (const f of filteredPlan.remoteOnly) {
        console.log(`  ${f.path}`);
      }
    }
    if (filteredPlan.localDeleted.length > 0) {
      console.log("\n\u{1F5D1}\uFE0F To hide on server:");
      for (const f of filteredPlan.localDeleted) {
        console.log(`  ${f.path}`);
      }
    }
  }
  if (args.dryRun) {
    console.log("\n\u23F8\uFE0F Dry run - no changes made");
    return;
  }
  const totalActions = filteredPlan.pushes.length + filteredPlan.localOnly.length + filteredPlan.pulls.length + filteredPlan.remoteOnly.length + filteredPlan.conflicts.length + filteredPlan.localDeleted.length + filteredPlan.serverDeleted.length;
  console.log("\n\u{1F680} Executing sync...");
  const result = await executePlan(env, filteredPlan, { twoWaySync: args.twoWaySync });
  if (totalActions === 0 && result.assetsUploaded === 0 && result.assetsDownloaded === 0) {
    console.log("\n\u2705 Everything is up to date!");
    return;
  }
  console.log("\n" + "=".repeat(60));
  console.log("\u{1F4CA} SYNC RESULTS:");
  console.log("=".repeat(60));
  console.log(`  Pushed:             ${result.pushed}`);
  console.log(`  Pulled:             ${result.pulled}`);
  console.log(`  Conflicts resolved: ${result.conflictsResolved}`);
  console.log(`  Assets uploaded:    ${result.assetsUploaded}`);
  console.log(`  Assets downloaded:  ${result.assetsDownloaded}`);
  if (result.errors.length > 0) {
    console.log(`  Errors:             ${result.errors.length}`);
    for (const err of result.errors) {
      console.log(`    \u274C ${err}`);
    }
  }
  if (result.warnings.length > 0) {
    console.log(`  Warnings:           ${result.warnings.length}`);
    for (const w of result.warnings) {
      console.log(`    \u26A0\uFE0F  [${w.level}] ${w.path}: ${w.message}`);
    }
  }
  console.log("=".repeat(60));
  const updatedUrls = result.updatedUrls ?? [];
  if (updatedUrls.length > 0) {
    console.log("\n\u{1F4CE} Published:");
    if (updatedUrls.length <= 20) {
      for (const { path: path5, url } of updatedUrls) {
        console.log(`  ${path5} \u2192 ${url}`);
      }
    }
    if (args.updatedOutput) {
      fs3.writeFileSync(args.updatedOutput, JSON.stringify(updatedUrls, null, 2));
      console.log(`\u{1F4BE} Saved to ${args.updatedOutput}`);
    } else {
      console.log(`\u{1F4A1} --updated-output $(mktemp /tmp/updated-XXXXXX.json)`);
    }
  }
}
__name(main, "main");
main().catch((err) => {
  console.error("\u274C Fatal error:", err);
  process.exit(1);
});
/*! Bundled license information:

chokidar/esm/index.js:
  (*! chokidar - MIT License (c) 2012 Paul Miller (paulmillr.com) *)
*/
