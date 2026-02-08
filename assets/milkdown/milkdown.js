"use strict";
var $trip2g_milkdown_bundle = (() => {
  var __create = Object.create;
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __getProtoOf = Object.getPrototypeOf;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __typeError = (msg) => {
    throw TypeError(msg);
  };
  var __commonJS = (cb, mod) => function __require() {
    return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
  };
  var __export = (target, all2) => {
    for (var name in all2)
      __defProp(target, name, { get: all2[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key3 of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key3) && key3 !== except)
          __defProp(to, key3, { get: () => from[key3], enumerable: !(desc = __getOwnPropDesc(from, key3)) || desc.enumerable });
    }
    return to;
  };
  var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
    // If the importer is in node compatibility mode or this is not an ESM
    // file that has been converted to a CommonJS file using a Babel-
    // compatible transform (i.e. "__esModule" has not been set), then set
    // "default" to the CommonJS "module.exports" for node compatibility.
    isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
    mod
  ));
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);
  var __accessCheck = (obj, member, msg) => member.has(obj) || __typeError("Cannot " + msg);
  var __privateGet = (obj, member, getter) => (__accessCheck(obj, member, "read from private field"), getter ? getter.call(obj) : member.get(obj));
  var __privateAdd = (obj, member, value) => member.has(obj) ? __typeError("Cannot add the same private member more than once") : member instanceof WeakSet ? member.add(obj) : member.set(obj, value);
  var __privateSet = (obj, member, value, setter) => (__accessCheck(obj, member, "write to private field"), setter ? setter.call(obj, value) : member.set(obj, value), value);

  // node_modules/extend/index.js
  var require_extend = __commonJS({
    "node_modules/extend/index.js"(exports, module) {
      "use strict";
      var hasOwn = Object.prototype.hasOwnProperty;
      var toStr = Object.prototype.toString;
      var defineProperty = Object.defineProperty;
      var gOPD = Object.getOwnPropertyDescriptor;
      var isArray = function isArray2(arr) {
        if (typeof Array.isArray === "function") {
          return Array.isArray(arr);
        }
        return toStr.call(arr) === "[object Array]";
      };
      var isPlainObject2 = function isPlainObject3(obj) {
        if (!obj || toStr.call(obj) !== "[object Object]") {
          return false;
        }
        var hasOwnConstructor = hasOwn.call(obj, "constructor");
        var hasIsPrototypeOf = obj.constructor && obj.constructor.prototype && hasOwn.call(obj.constructor.prototype, "isPrototypeOf");
        if (obj.constructor && !hasOwnConstructor && !hasIsPrototypeOf) {
          return false;
        }
        var key3;
        for (key3 in obj) {
        }
        return typeof key3 === "undefined" || hasOwn.call(obj, key3);
      };
      var setProperty = function setProperty2(target, options) {
        if (defineProperty && options.name === "__proto__") {
          defineProperty(target, options.name, {
            enumerable: true,
            configurable: true,
            value: options.newValue,
            writable: true
          });
        } else {
          target[options.name] = options.newValue;
        }
      };
      var getProperty = function getProperty2(obj, name) {
        if (name === "__proto__") {
          if (!hasOwn.call(obj, name)) {
            return void 0;
          } else if (gOPD) {
            return gOPD(obj, name).value;
          }
        }
        return obj[name];
      };
      module.exports = function extend2() {
        var options, name, src, copy2, copyIsArray, clone;
        var target = arguments[0];
        var i2 = 1;
        var length = arguments.length;
        var deep = false;
        if (typeof target === "boolean") {
          deep = target;
          target = arguments[1] || {};
          i2 = 2;
        }
        if (target == null || typeof target !== "object" && typeof target !== "function") {
          target = {};
        }
        for (; i2 < length; ++i2) {
          options = arguments[i2];
          if (options != null) {
            for (name in options) {
              src = getProperty(target, name);
              copy2 = getProperty(options, name);
              if (target !== copy2) {
                if (deep && copy2 && (isPlainObject2(copy2) || (copyIsArray = isArray(copy2)))) {
                  if (copyIsArray) {
                    copyIsArray = false;
                    clone = src && isArray(src) ? src : [];
                  } else {
                    clone = src && isPlainObject2(src) ? src : {};
                  }
                  setProperty(target, { name, newValue: extend2(deep, clone, copy2) });
                } else if (typeof copy2 !== "undefined") {
                  setProperty(target, { name, newValue: copy2 });
                }
              }
            }
          }
        }
        return target;
      };
    }
  });

  // src/index.ts
  var index_exports = {};
  __export(index_exports, {
    createMilkdown: () => createMilkdown
  });

  // node_modules/@milkdown/exception/lib/index.js
  var ErrorCode = /* @__PURE__ */ ((ErrorCode2) => {
    ErrorCode2["docTypeError"] = "docTypeError";
    ErrorCode2["contextNotFound"] = "contextNotFound";
    ErrorCode2["timerNotFound"] = "timerNotFound";
    ErrorCode2["ctxCallOutOfScope"] = "ctxCallOutOfScope";
    ErrorCode2["createNodeInParserFail"] = "createNodeInParserFail";
    ErrorCode2["stackOverFlow"] = "stackOverFlow";
    ErrorCode2["parserMatchError"] = "parserMatchError";
    ErrorCode2["serializerMatchError"] = "serializerMatchError";
    ErrorCode2["getAtomFromSchemaFail"] = "getAtomFromSchemaFail";
    ErrorCode2["expectDomTypeError"] = "expectDomTypeError";
    ErrorCode2["callCommandBeforeEditorView"] = "callCommandBeforeEditorView";
    ErrorCode2["missingRootElement"] = "missingRootElement";
    ErrorCode2["missingNodeInSchema"] = "missingNodeInSchema";
    ErrorCode2["missingMarkInSchema"] = "missingMarkInSchema";
    ErrorCode2["ctxNotBind"] = "ctxNotBind";
    ErrorCode2["missingYjsDoc"] = "missingYjsDoc";
    return ErrorCode2;
  })(ErrorCode || {});
  var MilkdownError = class extends Error {
    constructor(code2, message) {
      super(message);
      this.name = "MilkdownError";
      this.code = code2;
    }
  };
  var functionReplacer = (_, value) => typeof value === "function" ? "[Function]" : value;
  var stringify = (x) => JSON.stringify(x, functionReplacer);
  function docTypeError(type) {
    return new MilkdownError(
      ErrorCode.docTypeError,
      `Doc type error, unsupported type: ${stringify(type)}`
    );
  }
  function contextNotFound(name) {
    return new MilkdownError(
      ErrorCode.contextNotFound,
      `Context "${name}" not found, do you forget to inject it?`
    );
  }
  function timerNotFound(name) {
    return new MilkdownError(
      ErrorCode.timerNotFound,
      `Timer "${name}" not found, do you forget to record it?`
    );
  }
  function ctxCallOutOfScope() {
    return new MilkdownError(
      ErrorCode.ctxCallOutOfScope,
      "Should not call a context out of the plugin."
    );
  }
  function createNodeInParserFail(nodeType, attrs, content3) {
    const nodeTypeName = "name" in nodeType ? nodeType.name : nodeType;
    const heading2 = `Cannot create node for ${nodeTypeName}`;
    const serialize = (x) => {
      if (x == null) return "null";
      if (Array.isArray(x)) {
        return `[${x.map(serialize).join(", ")}]`;
      }
      if (typeof x === "object") {
        if ("toJSON" in x && typeof x.toJSON === "function") {
          return JSON.stringify(x.toJSON());
        }
        if ("spec" in x) {
          return JSON.stringify(x.spec);
        }
        return JSON.stringify(x);
      }
      if (typeof x === "string" || typeof x === "number" || typeof x === "boolean") {
        return JSON.stringify(x);
      }
      if (typeof x === "function") {
        return `[Function: ${x.name || "anonymous"}]`;
      }
      try {
        return String(x);
      } catch {
        return "[Unserializable]";
      }
    };
    const headingMessage = ["[Description]", heading2];
    const attrsMessage = ["[Attributes]", attrs];
    const contentMessage = [
      "[Content]",
      (content3 ?? []).map((node2) => {
        if (!node2) return "null";
        if (typeof node2 === "object" && "type" in node2) {
          return `${node2}`;
        }
        return serialize(node2);
      })
    ];
    const messages = [headingMessage, attrsMessage, contentMessage].reduce(
      (acc, [title, value]) => {
        const message = `${title}: ${serialize(value)}.`;
        return acc.concat(message);
      },
      []
    );
    return new MilkdownError(
      ErrorCode.createNodeInParserFail,
      messages.join("\n")
    );
  }
  function stackOverFlow() {
    return new MilkdownError(
      ErrorCode.stackOverFlow,
      "Stack over flow, cannot pop on an empty stack."
    );
  }
  function parserMatchError(node2) {
    return new MilkdownError(
      ErrorCode.parserMatchError,
      `Cannot match target parser for node: ${stringify(node2)}.`
    );
  }
  function serializerMatchError(node2) {
    return new MilkdownError(
      ErrorCode.serializerMatchError,
      `Cannot match target serializer for node: ${stringify(node2)}.`
    );
  }
  function expectDomTypeError(node2) {
    return new MilkdownError(
      ErrorCode.expectDomTypeError,
      `Expect to be a dom, but get: ${stringify(node2)}.`
    );
  }
  function callCommandBeforeEditorView() {
    return new MilkdownError(
      ErrorCode.callCommandBeforeEditorView,
      "You're trying to call a command before editor view initialized, make sure to get commandManager from ctx after editor view has been initialized"
    );
  }
  function missingNodeInSchema(name) {
    return new MilkdownError(
      ErrorCode.missingNodeInSchema,
      `Missing node in schema, milkdown cannot find "${name}" in schema.`
    );
  }
  function missingMarkInSchema(name) {
    return new MilkdownError(
      ErrorCode.missingMarkInSchema,
      `Missing mark in schema, milkdown cannot find "${name}" in schema.`
    );
  }

  // node_modules/@milkdown/ctx/lib/index.js
  var Container = class {
    constructor() {
      this.sliceMap = /* @__PURE__ */ new Map();
      this.get = (slice) => {
        const context = typeof slice === "string" ? [...this.sliceMap.values()].find((x) => x.type.name === slice) : this.sliceMap.get(slice.id);
        if (!context) {
          const name = typeof slice === "string" ? slice : slice.name;
          throw contextNotFound(name);
        }
        return context;
      };
      this.remove = (slice) => {
        const context = typeof slice === "string" ? [...this.sliceMap.values()].find((x) => x.type.name === slice) : this.sliceMap.get(slice.id);
        if (!context) return;
        this.sliceMap.delete(context.type.id);
      };
      this.has = (slice) => {
        if (typeof slice === "string")
          return [...this.sliceMap.values()].some((x) => x.type.name === slice);
        return this.sliceMap.has(slice.id);
      };
    }
  };
  var _watchers, _value, _emit;
  var Slice = class {
    /// @internal
    constructor(container, value, type) {
      __privateAdd(this, _watchers);
      /// @internal
      __privateAdd(this, _value);
      __privateAdd(this, _emit);
      __privateSet(this, _watchers, []);
      __privateSet(this, _emit, () => {
        __privateGet(this, _watchers).forEach((watcher) => watcher(__privateGet(this, _value)));
      });
      this.set = (value2) => {
        __privateSet(this, _value, value2);
        __privateGet(this, _emit).call(this);
      };
      this.get = () => __privateGet(this, _value);
      this.update = (updater) => {
        __privateSet(this, _value, updater(__privateGet(this, _value)));
        __privateGet(this, _emit).call(this);
      };
      this.type = type;
      __privateSet(this, _value, value);
      container.set(type.id, this);
    }
    /// Add a watcher for changes in the slice.
    /// Returns a function to remove the watcher.
    on(watcher) {
      __privateGet(this, _watchers).push(watcher);
      return () => {
        __privateSet(this, _watchers, __privateGet(this, _watchers).filter((w) => w !== watcher));
      };
    }
    /// Add a one-time watcher for changes in the slice.
    /// The watcher will be removed after it is called.
    /// Returns a function to remove the watcher.
    once(watcher) {
      const off = this.on((value) => {
        watcher(value);
        off();
      });
      return off;
    }
    /// Remove a watcher.
    off(watcher) {
      __privateSet(this, _watchers, __privateGet(this, _watchers).filter((w) => w !== watcher));
    }
    /// Remove all watchers.
    offAll() {
      __privateSet(this, _watchers, []);
    }
  };
  _watchers = new WeakMap();
  _value = new WeakMap();
  _emit = new WeakMap();
  var SliceType = class {
    /// Create a slice type with a default value and a name.
    /// The name should be unique in the container.
    constructor(value, name) {
      this.id = /* @__PURE__ */ Symbol(`Context-${name}`);
      this.name = name;
      this._defaultValue = value;
      this._typeInfo = () => {
        throw ctxCallOutOfScope();
      };
    }
    /// Create a slice with a container.
    /// You can also pass a value to override the default value.
    create(container, value = this._defaultValue) {
      return new Slice(container, value, this);
    }
  };
  var createSlice = (value, name) => new SliceType(value, name);
  var _meta, _container, _clock, _injectedSlices, _consumedSlices, _recordedTimers, _waitTimers, _getSlice, _getTimer;
  var Inspector = class {
    /// Create an inspector with container, clock and metadata.
    constructor(container, clock, meta) {
      /// @internal
      __privateAdd(this, _meta);
      /// @internal
      __privateAdd(this, _container);
      /// @internal
      __privateAdd(this, _clock);
      __privateAdd(this, _injectedSlices);
      __privateAdd(this, _consumedSlices);
      __privateAdd(this, _recordedTimers);
      __privateAdd(this, _waitTimers);
      __privateAdd(this, _getSlice);
      __privateAdd(this, _getTimer);
      __privateSet(this, _injectedSlices, /* @__PURE__ */ new Set());
      __privateSet(this, _consumedSlices, /* @__PURE__ */ new Set());
      __privateSet(this, _recordedTimers, /* @__PURE__ */ new Map());
      __privateSet(this, _waitTimers, /* @__PURE__ */ new Map());
      this.read = () => {
        return {
          metadata: __privateGet(this, _meta),
          injectedSlices: [...__privateGet(this, _injectedSlices)].map((slice) => ({
            name: typeof slice === "string" ? slice : slice.name,
            value: __privateGet(this, _getSlice).call(this, slice)
          })),
          consumedSlices: [...__privateGet(this, _consumedSlices)].map((slice) => ({
            name: typeof slice === "string" ? slice : slice.name,
            value: __privateGet(this, _getSlice).call(this, slice)
          })),
          recordedTimers: [...__privateGet(this, _recordedTimers)].map(
            ([timer, { duration }]) => ({
              name: timer.name,
              duration,
              status: __privateGet(this, _getTimer).call(this, timer)
            })
          ),
          waitTimers: [...__privateGet(this, _waitTimers)].map(([timer, { duration }]) => ({
            name: timer.name,
            duration,
            status: __privateGet(this, _getTimer).call(this, timer)
          }))
        };
      };
      this.onRecord = (timerType) => {
        __privateGet(this, _recordedTimers).set(timerType, { start: Date.now(), duration: 0 });
      };
      this.onClear = (timerType) => {
        __privateGet(this, _recordedTimers).delete(timerType);
      };
      this.onDone = (timerType) => {
        const timer = __privateGet(this, _recordedTimers).get(timerType);
        if (!timer) return;
        timer.duration = Date.now() - timer.start;
      };
      this.onWait = (timerType, promise) => {
        const start = Date.now();
        promise.finally(() => {
          __privateGet(this, _waitTimers).set(timerType, { duration: Date.now() - start });
        }).catch(console.error);
      };
      this.onInject = (sliceType) => {
        __privateGet(this, _injectedSlices).add(sliceType);
      };
      this.onRemove = (sliceType) => {
        __privateGet(this, _injectedSlices).delete(sliceType);
      };
      this.onUse = (sliceType) => {
        __privateGet(this, _consumedSlices).add(sliceType);
      };
      __privateSet(this, _getSlice, (sliceType) => {
        return __privateGet(this, _container).get(sliceType).get();
      });
      __privateSet(this, _getTimer, (timerType) => {
        return __privateGet(this, _clock).get(timerType).status;
      });
      __privateSet(this, _container, container);
      __privateSet(this, _clock, clock);
      __privateSet(this, _meta, meta);
    }
  };
  _meta = new WeakMap();
  _container = new WeakMap();
  _clock = new WeakMap();
  _injectedSlices = new WeakMap();
  _consumedSlices = new WeakMap();
  _recordedTimers = new WeakMap();
  _waitTimers = new WeakMap();
  _getSlice = new WeakMap();
  _getTimer = new WeakMap();
  var _container2, _clock2, _meta2, _inspector;
  var _Ctx = class _Ctx {
    /// Create a ctx object with container and clock.
    constructor(container, clock, meta) {
      /// @internal
      __privateAdd(this, _container2);
      /// @internal
      __privateAdd(this, _clock2);
      /// @internal
      __privateAdd(this, _meta2);
      /// @internal
      __privateAdd(this, _inspector);
      this.produce = (meta2) => {
        if (meta2 && Object.keys(meta2).length)
          return new _Ctx(__privateGet(this, _container2), __privateGet(this, _clock2), { ...meta2 });
        return this;
      };
      this.inject = (sliceType, value) => {
        const slice = sliceType.create(__privateGet(this, _container2).sliceMap);
        if (value != null) slice.set(value);
        __privateGet(this, _inspector)?.onInject(sliceType);
        return this;
      };
      this.remove = (sliceType) => {
        __privateGet(this, _container2).remove(sliceType);
        __privateGet(this, _inspector)?.onRemove(sliceType);
        return this;
      };
      this.record = (timerType) => {
        timerType.create(__privateGet(this, _clock2).store);
        __privateGet(this, _inspector)?.onRecord(timerType);
        return this;
      };
      this.clearTimer = (timerType) => {
        __privateGet(this, _clock2).remove(timerType);
        __privateGet(this, _inspector)?.onClear(timerType);
        return this;
      };
      this.isInjected = (sliceType) => __privateGet(this, _container2).has(sliceType);
      this.isRecorded = (timerType) => __privateGet(this, _clock2).has(timerType);
      this.use = (sliceType) => {
        __privateGet(this, _inspector)?.onUse(sliceType);
        return __privateGet(this, _container2).get(sliceType);
      };
      this.get = (sliceType) => this.use(sliceType).get();
      this.set = (sliceType, value) => this.use(sliceType).set(value);
      this.update = (sliceType, updater) => this.use(sliceType).update(updater);
      this.timer = (timer) => __privateGet(this, _clock2).get(timer);
      this.done = (timer) => {
        this.timer(timer).done();
        __privateGet(this, _inspector)?.onDone(timer);
      };
      this.wait = (timer) => {
        const promise = this.timer(timer).start();
        __privateGet(this, _inspector)?.onWait(timer, promise);
        return promise;
      };
      this.waitTimers = async (slice) => {
        await Promise.all(this.get(slice).map((x) => this.wait(x)));
      };
      __privateSet(this, _container2, container);
      __privateSet(this, _clock2, clock);
      __privateSet(this, _meta2, meta);
      if (meta) __privateSet(this, _inspector, new Inspector(container, clock, meta));
    }
    /// Get metadata of the ctx.
    get meta() {
      return __privateGet(this, _meta2);
    }
    /// Get the inspector of the ctx.
    get inspector() {
      return __privateGet(this, _inspector);
    }
  };
  _container2 = new WeakMap();
  _clock2 = new WeakMap();
  _meta2 = new WeakMap();
  _inspector = new WeakMap();
  var Ctx = _Ctx;
  var Clock = class {
    constructor() {
      this.store = /* @__PURE__ */ new Map();
      this.get = (timer) => {
        const meta = this.store.get(timer.id);
        if (!meta) throw timerNotFound(timer.name);
        return meta;
      };
      this.remove = (timer) => {
        this.store.delete(timer.id);
      };
      this.has = (timer) => {
        return this.store.has(timer.id);
      };
    }
  };
  var _promise, _listener, _eventUniqId, _status, _removeListener, _waitTimeout;
  var Timer = class {
    /// @internal
    constructor(clock, type) {
      __privateAdd(this, _promise);
      __privateAdd(this, _listener);
      /// @internal
      __privateAdd(this, _eventUniqId);
      __privateAdd(this, _status);
      __privateAdd(this, _removeListener);
      __privateAdd(this, _waitTimeout);
      __privateSet(this, _promise, null);
      __privateSet(this, _listener, null);
      __privateSet(this, _status, "pending");
      this.start = () => {
        __privateGet(this, _promise) ?? __privateSet(this, _promise, new Promise((resolve, reject) => {
          __privateSet(this, _listener, (e2) => {
            if (!(e2 instanceof CustomEvent)) return;
            if (e2.detail.id === __privateGet(this, _eventUniqId)) {
              __privateSet(this, _status, "resolved");
              __privateGet(this, _removeListener).call(this);
              e2.stopImmediatePropagation();
              resolve();
            }
          });
          __privateGet(this, _waitTimeout).call(this, () => {
            if (__privateGet(this, _status) === "pending") __privateSet(this, _status, "rejected");
            __privateGet(this, _removeListener).call(this);
            reject(new Error(`Timing ${this.type.name} timeout.`));
          });
          __privateSet(this, _status, "pending");
          addEventListener(this.type.name, __privateGet(this, _listener));
        }));
        return __privateGet(this, _promise);
      };
      this.done = () => {
        const event = new CustomEvent(this.type.name, {
          detail: { id: __privateGet(this, _eventUniqId) }
        });
        dispatchEvent(event);
      };
      __privateSet(this, _removeListener, () => {
        if (__privateGet(this, _listener)) removeEventListener(this.type.name, __privateGet(this, _listener));
      });
      __privateSet(this, _waitTimeout, (ifTimeout) => {
        setTimeout(() => {
          ifTimeout();
        }, this.type.timeout);
      });
      __privateSet(this, _eventUniqId, Symbol(type.name));
      this.type = type;
      clock.set(type.id, this);
    }
    /// The status of the timer.
    /// Can be `pending`, `resolved` or `rejected`.
    get status() {
      return __privateGet(this, _status);
    }
  };
  _promise = new WeakMap();
  _listener = new WeakMap();
  _eventUniqId = new WeakMap();
  _status = new WeakMap();
  _removeListener = new WeakMap();
  _waitTimeout = new WeakMap();
  var TimerType = class {
    /// Create a timer type with a name and a timeout.
    /// The name should be unique in the clock.
    constructor(name, timeout = 3e3) {
      this.create = (clock) => {
        return new Timer(clock, this);
      };
      this.id = /* @__PURE__ */ Symbol(`Timer-${name}`);
      this.name = name;
      this.timeout = timeout;
    }
  };
  var createTimer = (name, timeout = 3e3) => new TimerType(name, timeout);

  // node_modules/orderedmap/dist/index.js
  function OrderedMap(content3) {
    this.content = content3;
  }
  OrderedMap.prototype = {
    constructor: OrderedMap,
    find: function(key3) {
      for (var i2 = 0; i2 < this.content.length; i2 += 2)
        if (this.content[i2] === key3) return i2;
      return -1;
    },
    // :: (string) → ?any
    // Retrieve the value stored under `key`, or return undefined when
    // no such key exists.
    get: function(key3) {
      var found2 = this.find(key3);
      return found2 == -1 ? void 0 : this.content[found2 + 1];
    },
    // :: (string, any, ?string) → OrderedMap
    // Create a new map by replacing the value of `key` with a new
    // value, or adding a binding to the end of the map. If `newKey` is
    // given, the key of the binding will be replaced with that key.
    update: function(key3, value, newKey) {
      var self2 = newKey && newKey != key3 ? this.remove(newKey) : this;
      var found2 = self2.find(key3), content3 = self2.content.slice();
      if (found2 == -1) {
        content3.push(newKey || key3, value);
      } else {
        content3[found2 + 1] = value;
        if (newKey) content3[found2] = newKey;
      }
      return new OrderedMap(content3);
    },
    // :: (string) → OrderedMap
    // Return a map with the given key removed, if it existed.
    remove: function(key3) {
      var found2 = this.find(key3);
      if (found2 == -1) return this;
      var content3 = this.content.slice();
      content3.splice(found2, 2);
      return new OrderedMap(content3);
    },
    // :: (string, any) → OrderedMap
    // Add a new key to the start of the map.
    addToStart: function(key3, value) {
      return new OrderedMap([key3, value].concat(this.remove(key3).content));
    },
    // :: (string, any) → OrderedMap
    // Add a new key to the end of the map.
    addToEnd: function(key3, value) {
      var content3 = this.remove(key3).content.slice();
      content3.push(key3, value);
      return new OrderedMap(content3);
    },
    // :: (string, string, any) → OrderedMap
    // Add a key after the given key. If `place` is not found, the new
    // key is added to the end.
    addBefore: function(place, key3, value) {
      var without = this.remove(key3), content3 = without.content.slice();
      var found2 = without.find(place);
      content3.splice(found2 == -1 ? content3.length : found2, 0, key3, value);
      return new OrderedMap(content3);
    },
    // :: ((key: string, value: any))
    // Call the given function for each key/value pair in the map, in
    // order.
    forEach: function(f2) {
      for (var i2 = 0; i2 < this.content.length; i2 += 2)
        f2(this.content[i2], this.content[i2 + 1]);
    },
    // :: (union<Object, OrderedMap>) → OrderedMap
    // Create a new map by prepending the keys in this map that don't
    // appear in `map` before the keys in `map`.
    prepend: function(map4) {
      map4 = OrderedMap.from(map4);
      if (!map4.size) return this;
      return new OrderedMap(map4.content.concat(this.subtract(map4).content));
    },
    // :: (union<Object, OrderedMap>) → OrderedMap
    // Create a new map by appending the keys in this map that don't
    // appear in `map` after the keys in `map`.
    append: function(map4) {
      map4 = OrderedMap.from(map4);
      if (!map4.size) return this;
      return new OrderedMap(this.subtract(map4).content.concat(map4.content));
    },
    // :: (union<Object, OrderedMap>) → OrderedMap
    // Create a map containing all the keys in this map that don't
    // appear in `map`.
    subtract: function(map4) {
      var result = this;
      map4 = OrderedMap.from(map4);
      for (var i2 = 0; i2 < map4.content.length; i2 += 2)
        result = result.remove(map4.content[i2]);
      return result;
    },
    // :: () → Object
    // Turn ordered map into a plain object.
    toObject: function() {
      var result = {};
      this.forEach(function(key3, value) {
        result[key3] = value;
      });
      return result;
    },
    // :: number
    // The amount of keys in this map.
    get size() {
      return this.content.length >> 1;
    }
  };
  OrderedMap.from = function(value) {
    if (value instanceof OrderedMap) return value;
    var content3 = [];
    if (value) for (var prop in value) content3.push(prop, value[prop]);
    return new OrderedMap(content3);
  };
  var dist_default = OrderedMap;

  // node_modules/prosemirror-model/dist/index.js
  function findDiffStart(a2, b, pos) {
    for (let i2 = 0; ; i2++) {
      if (i2 == a2.childCount || i2 == b.childCount)
        return a2.childCount == b.childCount ? null : pos;
      let childA = a2.child(i2), childB = b.child(i2);
      if (childA == childB) {
        pos += childA.nodeSize;
        continue;
      }
      if (!childA.sameMarkup(childB))
        return pos;
      if (childA.isText && childA.text != childB.text) {
        for (let j = 0; childA.text[j] == childB.text[j]; j++)
          pos++;
        return pos;
      }
      if (childA.content.size || childB.content.size) {
        let inner = findDiffStart(childA.content, childB.content, pos + 1);
        if (inner != null)
          return inner;
      }
      pos += childA.nodeSize;
    }
  }
  function findDiffEnd(a2, b, posA, posB) {
    for (let iA = a2.childCount, iB = b.childCount; ; ) {
      if (iA == 0 || iB == 0)
        return iA == iB ? null : { a: posA, b: posB };
      let childA = a2.child(--iA), childB = b.child(--iB), size = childA.nodeSize;
      if (childA == childB) {
        posA -= size;
        posB -= size;
        continue;
      }
      if (!childA.sameMarkup(childB))
        return { a: posA, b: posB };
      if (childA.isText && childA.text != childB.text) {
        let same = 0, minSize = Math.min(childA.text.length, childB.text.length);
        while (same < minSize && childA.text[childA.text.length - same - 1] == childB.text[childB.text.length - same - 1]) {
          same++;
          posA--;
          posB--;
        }
        return { a: posA, b: posB };
      }
      if (childA.content.size || childB.content.size) {
        let inner = findDiffEnd(childA.content, childB.content, posA - 1, posB - 1);
        if (inner)
          return inner;
      }
      posA -= size;
      posB -= size;
    }
  }
  var Fragment = class _Fragment {
    /**
    @internal
    */
    constructor(content3, size) {
      this.content = content3;
      this.size = size || 0;
      if (size == null)
        for (let i2 = 0; i2 < content3.length; i2++)
          this.size += content3[i2].nodeSize;
    }
    /**
    Invoke a callback for all descendant nodes between the given two
    positions (relative to start of this fragment). Doesn't descend
    into a node when the callback returns `false`.
    */
    nodesBetween(from, to, f2, nodeStart = 0, parent) {
      for (let i2 = 0, pos = 0; pos < to; i2++) {
        let child = this.content[i2], end = pos + child.nodeSize;
        if (end > from && f2(child, nodeStart + pos, parent || null, i2) !== false && child.content.size) {
          let start = pos + 1;
          child.nodesBetween(Math.max(0, from - start), Math.min(child.content.size, to - start), f2, nodeStart + start);
        }
        pos = end;
      }
    }
    /**
    Call the given callback for every descendant node. `pos` will be
    relative to the start of the fragment. The callback may return
    `false` to prevent traversal of a given node's children.
    */
    descendants(f2) {
      this.nodesBetween(0, this.size, f2);
    }
    /**
    Extract the text between `from` and `to`. See the same method on
    [`Node`](https://prosemirror.net/docs/ref/#model.Node.textBetween).
    */
    textBetween(from, to, blockSeparator, leafText) {
      let text4 = "", first = true;
      this.nodesBetween(from, to, (node2, pos) => {
        let nodeText = node2.isText ? node2.text.slice(Math.max(from, pos) - pos, to - pos) : !node2.isLeaf ? "" : leafText ? typeof leafText === "function" ? leafText(node2) : leafText : node2.type.spec.leafText ? node2.type.spec.leafText(node2) : "";
        if (node2.isBlock && (node2.isLeaf && nodeText || node2.isTextblock) && blockSeparator) {
          if (first)
            first = false;
          else
            text4 += blockSeparator;
        }
        text4 += nodeText;
      }, 0);
      return text4;
    }
    /**
    Create a new fragment containing the combined content of this
    fragment and the other.
    */
    append(other) {
      if (!other.size)
        return this;
      if (!this.size)
        return other;
      let last = this.lastChild, first = other.firstChild, content3 = this.content.slice(), i2 = 0;
      if (last.isText && last.sameMarkup(first)) {
        content3[content3.length - 1] = last.withText(last.text + first.text);
        i2 = 1;
      }
      for (; i2 < other.content.length; i2++)
        content3.push(other.content[i2]);
      return new _Fragment(content3, this.size + other.size);
    }
    /**
    Cut out the sub-fragment between the two given positions.
    */
    cut(from, to = this.size) {
      if (from == 0 && to == this.size)
        return this;
      let result = [], size = 0;
      if (to > from)
        for (let i2 = 0, pos = 0; pos < to; i2++) {
          let child = this.content[i2], end = pos + child.nodeSize;
          if (end > from) {
            if (pos < from || end > to) {
              if (child.isText)
                child = child.cut(Math.max(0, from - pos), Math.min(child.text.length, to - pos));
              else
                child = child.cut(Math.max(0, from - pos - 1), Math.min(child.content.size, to - pos - 1));
            }
            result.push(child);
            size += child.nodeSize;
          }
          pos = end;
        }
      return new _Fragment(result, size);
    }
    /**
    @internal
    */
    cutByIndex(from, to) {
      if (from == to)
        return _Fragment.empty;
      if (from == 0 && to == this.content.length)
        return this;
      return new _Fragment(this.content.slice(from, to));
    }
    /**
    Create a new fragment in which the node at the given index is
    replaced by the given node.
    */
    replaceChild(index2, node2) {
      let current = this.content[index2];
      if (current == node2)
        return this;
      let copy2 = this.content.slice();
      let size = this.size + node2.nodeSize - current.nodeSize;
      copy2[index2] = node2;
      return new _Fragment(copy2, size);
    }
    /**
    Create a new fragment by prepending the given node to this
    fragment.
    */
    addToStart(node2) {
      return new _Fragment([node2].concat(this.content), this.size + node2.nodeSize);
    }
    /**
    Create a new fragment by appending the given node to this
    fragment.
    */
    addToEnd(node2) {
      return new _Fragment(this.content.concat(node2), this.size + node2.nodeSize);
    }
    /**
    Compare this fragment to another one.
    */
    eq(other) {
      if (this.content.length != other.content.length)
        return false;
      for (let i2 = 0; i2 < this.content.length; i2++)
        if (!this.content[i2].eq(other.content[i2]))
          return false;
      return true;
    }
    /**
    The first child of the fragment, or `null` if it is empty.
    */
    get firstChild() {
      return this.content.length ? this.content[0] : null;
    }
    /**
    The last child of the fragment, or `null` if it is empty.
    */
    get lastChild() {
      return this.content.length ? this.content[this.content.length - 1] : null;
    }
    /**
    The number of child nodes in this fragment.
    */
    get childCount() {
      return this.content.length;
    }
    /**
    Get the child node at the given index. Raise an error when the
    index is out of range.
    */
    child(index2) {
      let found2 = this.content[index2];
      if (!found2)
        throw new RangeError("Index " + index2 + " out of range for " + this);
      return found2;
    }
    /**
    Get the child node at the given index, if it exists.
    */
    maybeChild(index2) {
      return this.content[index2] || null;
    }
    /**
    Call `f` for every child node, passing the node, its offset
    into this parent node, and its index.
    */
    forEach(f2) {
      for (let i2 = 0, p2 = 0; i2 < this.content.length; i2++) {
        let child = this.content[i2];
        f2(child, p2, i2);
        p2 += child.nodeSize;
      }
    }
    /**
    Find the first position at which this fragment and another
    fragment differ, or `null` if they are the same.
    */
    findDiffStart(other, pos = 0) {
      return findDiffStart(this, other, pos);
    }
    /**
    Find the first position, searching from the end, at which this
    fragment and the given fragment differ, or `null` if they are
    the same. Since this position will not be the same in both
    nodes, an object with two separate positions is returned.
    */
    findDiffEnd(other, pos = this.size, otherPos = other.size) {
      return findDiffEnd(this, other, pos, otherPos);
    }
    /**
    Find the index and inner offset corresponding to a given relative
    position in this fragment. The result object will be reused
    (overwritten) the next time the function is called. @internal
    */
    findIndex(pos) {
      if (pos == 0)
        return retIndex(0, pos);
      if (pos == this.size)
        return retIndex(this.content.length, pos);
      if (pos > this.size || pos < 0)
        throw new RangeError(`Position ${pos} outside of fragment (${this})`);
      for (let i2 = 0, curPos = 0; ; i2++) {
        let cur = this.child(i2), end = curPos + cur.nodeSize;
        if (end >= pos) {
          if (end == pos)
            return retIndex(i2 + 1, end);
          return retIndex(i2, curPos);
        }
        curPos = end;
      }
    }
    /**
    Return a debugging string that describes this fragment.
    */
    toString() {
      return "<" + this.toStringInner() + ">";
    }
    /**
    @internal
    */
    toStringInner() {
      return this.content.join(", ");
    }
    /**
    Create a JSON-serializeable representation of this fragment.
    */
    toJSON() {
      return this.content.length ? this.content.map((n2) => n2.toJSON()) : null;
    }
    /**
    Deserialize a fragment from its JSON representation.
    */
    static fromJSON(schema3, value) {
      if (!value)
        return _Fragment.empty;
      if (!Array.isArray(value))
        throw new RangeError("Invalid input for Fragment.fromJSON");
      return new _Fragment(value.map(schema3.nodeFromJSON));
    }
    /**
    Build a fragment from an array of nodes. Ensures that adjacent
    text nodes with the same marks are joined together.
    */
    static fromArray(array) {
      if (!array.length)
        return _Fragment.empty;
      let joined, size = 0;
      for (let i2 = 0; i2 < array.length; i2++) {
        let node2 = array[i2];
        size += node2.nodeSize;
        if (i2 && node2.isText && array[i2 - 1].sameMarkup(node2)) {
          if (!joined)
            joined = array.slice(0, i2);
          joined[joined.length - 1] = node2.withText(joined[joined.length - 1].text + node2.text);
        } else if (joined) {
          joined.push(node2);
        }
      }
      return new _Fragment(joined || array, size);
    }
    /**
    Create a fragment from something that can be interpreted as a
    set of nodes. For `null`, it returns the empty fragment. For a
    fragment, the fragment itself. For a node or array of nodes, a
    fragment containing those nodes.
    */
    static from(nodes) {
      if (!nodes)
        return _Fragment.empty;
      if (nodes instanceof _Fragment)
        return nodes;
      if (Array.isArray(nodes))
        return this.fromArray(nodes);
      if (nodes.attrs)
        return new _Fragment([nodes], nodes.nodeSize);
      throw new RangeError("Can not convert " + nodes + " to a Fragment" + (nodes.nodesBetween ? " (looks like multiple versions of prosemirror-model were loaded)" : ""));
    }
  };
  Fragment.empty = new Fragment([], 0);
  var found = { index: 0, offset: 0 };
  function retIndex(index2, offset) {
    found.index = index2;
    found.offset = offset;
    return found;
  }
  function compareDeep(a2, b) {
    if (a2 === b)
      return true;
    if (!(a2 && typeof a2 == "object") || !(b && typeof b == "object"))
      return false;
    let array = Array.isArray(a2);
    if (Array.isArray(b) != array)
      return false;
    if (array) {
      if (a2.length != b.length)
        return false;
      for (let i2 = 0; i2 < a2.length; i2++)
        if (!compareDeep(a2[i2], b[i2]))
          return false;
    } else {
      for (let p2 in a2)
        if (!(p2 in b) || !compareDeep(a2[p2], b[p2]))
          return false;
      for (let p2 in b)
        if (!(p2 in a2))
          return false;
    }
    return true;
  }
  var Mark = class _Mark {
    /**
    @internal
    */
    constructor(type, attrs) {
      this.type = type;
      this.attrs = attrs;
    }
    /**
    Given a set of marks, create a new set which contains this one as
    well, in the right position. If this mark is already in the set,
    the set itself is returned. If any marks that are set to be
    [exclusive](https://prosemirror.net/docs/ref/#model.MarkSpec.excludes) with this mark are present,
    those are replaced by this one.
    */
    addToSet(set) {
      let copy2, placed = false;
      for (let i2 = 0; i2 < set.length; i2++) {
        let other = set[i2];
        if (this.eq(other))
          return set;
        if (this.type.excludes(other.type)) {
          if (!copy2)
            copy2 = set.slice(0, i2);
        } else if (other.type.excludes(this.type)) {
          return set;
        } else {
          if (!placed && other.type.rank > this.type.rank) {
            if (!copy2)
              copy2 = set.slice(0, i2);
            copy2.push(this);
            placed = true;
          }
          if (copy2)
            copy2.push(other);
        }
      }
      if (!copy2)
        copy2 = set.slice();
      if (!placed)
        copy2.push(this);
      return copy2;
    }
    /**
    Remove this mark from the given set, returning a new set. If this
    mark is not in the set, the set itself is returned.
    */
    removeFromSet(set) {
      for (let i2 = 0; i2 < set.length; i2++)
        if (this.eq(set[i2]))
          return set.slice(0, i2).concat(set.slice(i2 + 1));
      return set;
    }
    /**
    Test whether this mark is in the given set of marks.
    */
    isInSet(set) {
      for (let i2 = 0; i2 < set.length; i2++)
        if (this.eq(set[i2]))
          return true;
      return false;
    }
    /**
    Test whether this mark has the same type and attributes as
    another mark.
    */
    eq(other) {
      return this == other || this.type == other.type && compareDeep(this.attrs, other.attrs);
    }
    /**
    Convert this mark to a JSON-serializeable representation.
    */
    toJSON() {
      let obj = { type: this.type.name };
      for (let _ in this.attrs) {
        obj.attrs = this.attrs;
        break;
      }
      return obj;
    }
    /**
    Deserialize a mark from JSON.
    */
    static fromJSON(schema3, json) {
      if (!json)
        throw new RangeError("Invalid input for Mark.fromJSON");
      let type = schema3.marks[json.type];
      if (!type)
        throw new RangeError(`There is no mark type ${json.type} in this schema`);
      let mark = type.create(json.attrs);
      type.checkAttrs(mark.attrs);
      return mark;
    }
    /**
    Test whether two sets of marks are identical.
    */
    static sameSet(a2, b) {
      if (a2 == b)
        return true;
      if (a2.length != b.length)
        return false;
      for (let i2 = 0; i2 < a2.length; i2++)
        if (!a2[i2].eq(b[i2]))
          return false;
      return true;
    }
    /**
    Create a properly sorted mark set from null, a single mark, or an
    unsorted array of marks.
    */
    static setFrom(marks) {
      if (!marks || Array.isArray(marks) && marks.length == 0)
        return _Mark.none;
      if (marks instanceof _Mark)
        return [marks];
      let copy2 = marks.slice();
      copy2.sort((a2, b) => a2.type.rank - b.type.rank);
      return copy2;
    }
  };
  Mark.none = [];
  var ReplaceError = class extends Error {
  };
  var Slice2 = class _Slice {
    /**
    Create a slice. When specifying a non-zero open depth, you must
    make sure that there are nodes of at least that depth at the
    appropriate side of the fragment—i.e. if the fragment is an
    empty paragraph node, `openStart` and `openEnd` can't be greater
    than 1.
    
    It is not necessary for the content of open nodes to conform to
    the schema's content constraints, though it should be a valid
    start/end/middle for such a node, depending on which sides are
    open.
    */
    constructor(content3, openStart, openEnd) {
      this.content = content3;
      this.openStart = openStart;
      this.openEnd = openEnd;
    }
    /**
    The size this slice would add when inserted into a document.
    */
    get size() {
      return this.content.size - this.openStart - this.openEnd;
    }
    /**
    @internal
    */
    insertAt(pos, fragment) {
      let content3 = insertInto(this.content, pos + this.openStart, fragment);
      return content3 && new _Slice(content3, this.openStart, this.openEnd);
    }
    /**
    @internal
    */
    removeBetween(from, to) {
      return new _Slice(removeRange(this.content, from + this.openStart, to + this.openStart), this.openStart, this.openEnd);
    }
    /**
    Tests whether this slice is equal to another slice.
    */
    eq(other) {
      return this.content.eq(other.content) && this.openStart == other.openStart && this.openEnd == other.openEnd;
    }
    /**
    @internal
    */
    toString() {
      return this.content + "(" + this.openStart + "," + this.openEnd + ")";
    }
    /**
    Convert a slice to a JSON-serializable representation.
    */
    toJSON() {
      if (!this.content.size)
        return null;
      let json = { content: this.content.toJSON() };
      if (this.openStart > 0)
        json.openStart = this.openStart;
      if (this.openEnd > 0)
        json.openEnd = this.openEnd;
      return json;
    }
    /**
    Deserialize a slice from its JSON representation.
    */
    static fromJSON(schema3, json) {
      if (!json)
        return _Slice.empty;
      let openStart = json.openStart || 0, openEnd = json.openEnd || 0;
      if (typeof openStart != "number" || typeof openEnd != "number")
        throw new RangeError("Invalid input for Slice.fromJSON");
      return new _Slice(Fragment.fromJSON(schema3, json.content), openStart, openEnd);
    }
    /**
    Create a slice from a fragment by taking the maximum possible
    open value on both side of the fragment.
    */
    static maxOpen(fragment, openIsolating = true) {
      let openStart = 0, openEnd = 0;
      for (let n2 = fragment.firstChild; n2 && !n2.isLeaf && (openIsolating || !n2.type.spec.isolating); n2 = n2.firstChild)
        openStart++;
      for (let n2 = fragment.lastChild; n2 && !n2.isLeaf && (openIsolating || !n2.type.spec.isolating); n2 = n2.lastChild)
        openEnd++;
      return new _Slice(fragment, openStart, openEnd);
    }
  };
  Slice2.empty = new Slice2(Fragment.empty, 0, 0);
  function removeRange(content3, from, to) {
    let { index: index2, offset } = content3.findIndex(from), child = content3.maybeChild(index2);
    let { index: indexTo, offset: offsetTo } = content3.findIndex(to);
    if (offset == from || child.isText) {
      if (offsetTo != to && !content3.child(indexTo).isText)
        throw new RangeError("Removing non-flat range");
      return content3.cut(0, from).append(content3.cut(to));
    }
    if (index2 != indexTo)
      throw new RangeError("Removing non-flat range");
    return content3.replaceChild(index2, child.copy(removeRange(child.content, from - offset - 1, to - offset - 1)));
  }
  function insertInto(content3, dist, insert, parent) {
    let { index: index2, offset } = content3.findIndex(dist), child = content3.maybeChild(index2);
    if (offset == dist || child.isText) {
      if (parent && !parent.canReplace(index2, index2, insert))
        return null;
      return content3.cut(0, dist).append(insert).append(content3.cut(dist));
    }
    let inner = insertInto(child.content, dist - offset - 1, insert, child);
    return inner && content3.replaceChild(index2, child.copy(inner));
  }
  function replace($from, $to, slice) {
    if (slice.openStart > $from.depth)
      throw new ReplaceError("Inserted content deeper than insertion position");
    if ($from.depth - slice.openStart != $to.depth - slice.openEnd)
      throw new ReplaceError("Inconsistent open depths");
    return replaceOuter($from, $to, slice, 0);
  }
  function replaceOuter($from, $to, slice, depth) {
    let index2 = $from.index(depth), node2 = $from.node(depth);
    if (index2 == $to.index(depth) && depth < $from.depth - slice.openStart) {
      let inner = replaceOuter($from, $to, slice, depth + 1);
      return node2.copy(node2.content.replaceChild(index2, inner));
    } else if (!slice.content.size) {
      return close(node2, replaceTwoWay($from, $to, depth));
    } else if (!slice.openStart && !slice.openEnd && $from.depth == depth && $to.depth == depth) {
      let parent = $from.parent, content3 = parent.content;
      return close(parent, content3.cut(0, $from.parentOffset).append(slice.content).append(content3.cut($to.parentOffset)));
    } else {
      let { start, end } = prepareSliceForReplace(slice, $from);
      return close(node2, replaceThreeWay($from, start, end, $to, depth));
    }
  }
  function checkJoin(main, sub) {
    if (!sub.type.compatibleContent(main.type))
      throw new ReplaceError("Cannot join " + sub.type.name + " onto " + main.type.name);
  }
  function joinable($before, $after, depth) {
    let node2 = $before.node(depth);
    checkJoin(node2, $after.node(depth));
    return node2;
  }
  function addNode(child, target) {
    let last = target.length - 1;
    if (last >= 0 && child.isText && child.sameMarkup(target[last]))
      target[last] = child.withText(target[last].text + child.text);
    else
      target.push(child);
  }
  function addRange($start, $end, depth, target) {
    let node2 = ($end || $start).node(depth);
    let startIndex = 0, endIndex = $end ? $end.index(depth) : node2.childCount;
    if ($start) {
      startIndex = $start.index(depth);
      if ($start.depth > depth) {
        startIndex++;
      } else if ($start.textOffset) {
        addNode($start.nodeAfter, target);
        startIndex++;
      }
    }
    for (let i2 = startIndex; i2 < endIndex; i2++)
      addNode(node2.child(i2), target);
    if ($end && $end.depth == depth && $end.textOffset)
      addNode($end.nodeBefore, target);
  }
  function close(node2, content3) {
    node2.type.checkContent(content3);
    return node2.copy(content3);
  }
  function replaceThreeWay($from, $start, $end, $to, depth) {
    let openStart = $from.depth > depth && joinable($from, $start, depth + 1);
    let openEnd = $to.depth > depth && joinable($end, $to, depth + 1);
    let content3 = [];
    addRange(null, $from, depth, content3);
    if (openStart && openEnd && $start.index(depth) == $end.index(depth)) {
      checkJoin(openStart, openEnd);
      addNode(close(openStart, replaceThreeWay($from, $start, $end, $to, depth + 1)), content3);
    } else {
      if (openStart)
        addNode(close(openStart, replaceTwoWay($from, $start, depth + 1)), content3);
      addRange($start, $end, depth, content3);
      if (openEnd)
        addNode(close(openEnd, replaceTwoWay($end, $to, depth + 1)), content3);
    }
    addRange($to, null, depth, content3);
    return new Fragment(content3);
  }
  function replaceTwoWay($from, $to, depth) {
    let content3 = [];
    addRange(null, $from, depth, content3);
    if ($from.depth > depth) {
      let type = joinable($from, $to, depth + 1);
      addNode(close(type, replaceTwoWay($from, $to, depth + 1)), content3);
    }
    addRange($to, null, depth, content3);
    return new Fragment(content3);
  }
  function prepareSliceForReplace(slice, $along) {
    let extra = $along.depth - slice.openStart, parent = $along.node(extra);
    let node2 = parent.copy(slice.content);
    for (let i2 = extra - 1; i2 >= 0; i2--)
      node2 = $along.node(i2).copy(Fragment.from(node2));
    return {
      start: node2.resolveNoCache(slice.openStart + extra),
      end: node2.resolveNoCache(node2.content.size - slice.openEnd - extra)
    };
  }
  var ResolvedPos = class _ResolvedPos {
    /**
    @internal
    */
    constructor(pos, path, parentOffset) {
      this.pos = pos;
      this.path = path;
      this.parentOffset = parentOffset;
      this.depth = path.length / 3 - 1;
    }
    /**
    @internal
    */
    resolveDepth(val) {
      if (val == null)
        return this.depth;
      if (val < 0)
        return this.depth + val;
      return val;
    }
    /**
    The parent node that the position points into. Note that even if
    a position points into a text node, that node is not considered
    the parent—text nodes are ‘flat’ in this model, and have no content.
    */
    get parent() {
      return this.node(this.depth);
    }
    /**
    The root node in which the position was resolved.
    */
    get doc() {
      return this.node(0);
    }
    /**
    The ancestor node at the given level. `p.node(p.depth)` is the
    same as `p.parent`.
    */
    node(depth) {
      return this.path[this.resolveDepth(depth) * 3];
    }
    /**
    The index into the ancestor at the given level. If this points
    at the 3rd node in the 2nd paragraph on the top level, for
    example, `p.index(0)` is 1 and `p.index(1)` is 2.
    */
    index(depth) {
      return this.path[this.resolveDepth(depth) * 3 + 1];
    }
    /**
    The index pointing after this position into the ancestor at the
    given level.
    */
    indexAfter(depth) {
      depth = this.resolveDepth(depth);
      return this.index(depth) + (depth == this.depth && !this.textOffset ? 0 : 1);
    }
    /**
    The (absolute) position at the start of the node at the given
    level.
    */
    start(depth) {
      depth = this.resolveDepth(depth);
      return depth == 0 ? 0 : this.path[depth * 3 - 1] + 1;
    }
    /**
    The (absolute) position at the end of the node at the given
    level.
    */
    end(depth) {
      depth = this.resolveDepth(depth);
      return this.start(depth) + this.node(depth).content.size;
    }
    /**
    The (absolute) position directly before the wrapping node at the
    given level, or, when `depth` is `this.depth + 1`, the original
    position.
    */
    before(depth) {
      depth = this.resolveDepth(depth);
      if (!depth)
        throw new RangeError("There is no position before the top-level node");
      return depth == this.depth + 1 ? this.pos : this.path[depth * 3 - 1];
    }
    /**
    The (absolute) position directly after the wrapping node at the
    given level, or the original position when `depth` is `this.depth + 1`.
    */
    after(depth) {
      depth = this.resolveDepth(depth);
      if (!depth)
        throw new RangeError("There is no position after the top-level node");
      return depth == this.depth + 1 ? this.pos : this.path[depth * 3 - 1] + this.path[depth * 3].nodeSize;
    }
    /**
    When this position points into a text node, this returns the
    distance between the position and the start of the text node.
    Will be zero for positions that point between nodes.
    */
    get textOffset() {
      return this.pos - this.path[this.path.length - 1];
    }
    /**
    Get the node directly after the position, if any. If the position
    points into a text node, only the part of that node after the
    position is returned.
    */
    get nodeAfter() {
      let parent = this.parent, index2 = this.index(this.depth);
      if (index2 == parent.childCount)
        return null;
      let dOff = this.pos - this.path[this.path.length - 1], child = parent.child(index2);
      return dOff ? parent.child(index2).cut(dOff) : child;
    }
    /**
    Get the node directly before the position, if any. If the
    position points into a text node, only the part of that node
    before the position is returned.
    */
    get nodeBefore() {
      let index2 = this.index(this.depth);
      let dOff = this.pos - this.path[this.path.length - 1];
      if (dOff)
        return this.parent.child(index2).cut(0, dOff);
      return index2 == 0 ? null : this.parent.child(index2 - 1);
    }
    /**
    Get the position at the given index in the parent node at the
    given depth (which defaults to `this.depth`).
    */
    posAtIndex(index2, depth) {
      depth = this.resolveDepth(depth);
      let node2 = this.path[depth * 3], pos = depth == 0 ? 0 : this.path[depth * 3 - 1] + 1;
      for (let i2 = 0; i2 < index2; i2++)
        pos += node2.child(i2).nodeSize;
      return pos;
    }
    /**
    Get the marks at this position, factoring in the surrounding
    marks' [`inclusive`](https://prosemirror.net/docs/ref/#model.MarkSpec.inclusive) property. If the
    position is at the start of a non-empty node, the marks of the
    node after it (if any) are returned.
    */
    marks() {
      let parent = this.parent, index2 = this.index();
      if (parent.content.size == 0)
        return Mark.none;
      if (this.textOffset)
        return parent.child(index2).marks;
      let main = parent.maybeChild(index2 - 1), other = parent.maybeChild(index2);
      if (!main) {
        let tmp = main;
        main = other;
        other = tmp;
      }
      let marks = main.marks;
      for (var i2 = 0; i2 < marks.length; i2++)
        if (marks[i2].type.spec.inclusive === false && (!other || !marks[i2].isInSet(other.marks)))
          marks = marks[i2--].removeFromSet(marks);
      return marks;
    }
    /**
    Get the marks after the current position, if any, except those
    that are non-inclusive and not present at position `$end`. This
    is mostly useful for getting the set of marks to preserve after a
    deletion. Will return `null` if this position is at the end of
    its parent node or its parent node isn't a textblock (in which
    case no marks should be preserved).
    */
    marksAcross($end) {
      let after = this.parent.maybeChild(this.index());
      if (!after || !after.isInline)
        return null;
      let marks = after.marks, next = $end.parent.maybeChild($end.index());
      for (var i2 = 0; i2 < marks.length; i2++)
        if (marks[i2].type.spec.inclusive === false && (!next || !marks[i2].isInSet(next.marks)))
          marks = marks[i2--].removeFromSet(marks);
      return marks;
    }
    /**
    The depth up to which this position and the given (non-resolved)
    position share the same parent nodes.
    */
    sharedDepth(pos) {
      for (let depth = this.depth; depth > 0; depth--)
        if (this.start(depth) <= pos && this.end(depth) >= pos)
          return depth;
      return 0;
    }
    /**
    Returns a range based on the place where this position and the
    given position diverge around block content. If both point into
    the same textblock, for example, a range around that textblock
    will be returned. If they point into different blocks, the range
    around those blocks in their shared ancestor is returned. You can
    pass in an optional predicate that will be called with a parent
    node to see if a range into that parent is acceptable.
    */
    blockRange(other = this, pred) {
      if (other.pos < this.pos)
        return other.blockRange(this);
      for (let d2 = this.depth - (this.parent.inlineContent || this.pos == other.pos ? 1 : 0); d2 >= 0; d2--)
        if (other.pos <= this.end(d2) && (!pred || pred(this.node(d2))))
          return new NodeRange(this, other, d2);
      return null;
    }
    /**
    Query whether the given position shares the same parent node.
    */
    sameParent(other) {
      return this.pos - this.parentOffset == other.pos - other.parentOffset;
    }
    /**
    Return the greater of this and the given position.
    */
    max(other) {
      return other.pos > this.pos ? other : this;
    }
    /**
    Return the smaller of this and the given position.
    */
    min(other) {
      return other.pos < this.pos ? other : this;
    }
    /**
    @internal
    */
    toString() {
      let str = "";
      for (let i2 = 1; i2 <= this.depth; i2++)
        str += (str ? "/" : "") + this.node(i2).type.name + "_" + this.index(i2 - 1);
      return str + ":" + this.parentOffset;
    }
    /**
    @internal
    */
    static resolve(doc4, pos) {
      if (!(pos >= 0 && pos <= doc4.content.size))
        throw new RangeError("Position " + pos + " out of range");
      let path = [];
      let start = 0, parentOffset = pos;
      for (let node2 = doc4; ; ) {
        let { index: index2, offset } = node2.content.findIndex(parentOffset);
        let rem = parentOffset - offset;
        path.push(node2, index2, start + offset);
        if (!rem)
          break;
        node2 = node2.child(index2);
        if (node2.isText)
          break;
        parentOffset = rem - 1;
        start += offset + 1;
      }
      return new _ResolvedPos(pos, path, parentOffset);
    }
    /**
    @internal
    */
    static resolveCached(doc4, pos) {
      let cache = resolveCache.get(doc4);
      if (cache) {
        for (let i2 = 0; i2 < cache.elts.length; i2++) {
          let elt = cache.elts[i2];
          if (elt.pos == pos)
            return elt;
        }
      } else {
        resolveCache.set(doc4, cache = new ResolveCache());
      }
      let result = cache.elts[cache.i] = _ResolvedPos.resolve(doc4, pos);
      cache.i = (cache.i + 1) % resolveCacheSize;
      return result;
    }
  };
  var ResolveCache = class {
    constructor() {
      this.elts = [];
      this.i = 0;
    }
  };
  var resolveCacheSize = 12;
  var resolveCache = /* @__PURE__ */ new WeakMap();
  var NodeRange = class {
    /**
    Construct a node range. `$from` and `$to` should point into the
    same node until at least the given `depth`, since a node range
    denotes an adjacent set of nodes in a single parent node.
    */
    constructor($from, $to, depth) {
      this.$from = $from;
      this.$to = $to;
      this.depth = depth;
    }
    /**
    The position at the start of the range.
    */
    get start() {
      return this.$from.before(this.depth + 1);
    }
    /**
    The position at the end of the range.
    */
    get end() {
      return this.$to.after(this.depth + 1);
    }
    /**
    The parent node that the range points into.
    */
    get parent() {
      return this.$from.node(this.depth);
    }
    /**
    The start index of the range in the parent node.
    */
    get startIndex() {
      return this.$from.index(this.depth);
    }
    /**
    The end index of the range in the parent node.
    */
    get endIndex() {
      return this.$to.indexAfter(this.depth);
    }
  };
  var emptyAttrs = /* @__PURE__ */ Object.create(null);
  var Node = class _Node {
    /**
    @internal
    */
    constructor(type, attrs, content3, marks = Mark.none) {
      this.type = type;
      this.attrs = attrs;
      this.marks = marks;
      this.content = content3 || Fragment.empty;
    }
    /**
    The array of this node's child nodes.
    */
    get children() {
      return this.content.content;
    }
    /**
    The size of this node, as defined by the integer-based [indexing
    scheme](https://prosemirror.net/docs/guide/#doc.indexing). For text nodes, this is the
    amount of characters. For other leaf nodes, it is one. For
    non-leaf nodes, it is the size of the content plus two (the
    start and end token).
    */
    get nodeSize() {
      return this.isLeaf ? 1 : 2 + this.content.size;
    }
    /**
    The number of children that the node has.
    */
    get childCount() {
      return this.content.childCount;
    }
    /**
    Get the child node at the given index. Raises an error when the
    index is out of range.
    */
    child(index2) {
      return this.content.child(index2);
    }
    /**
    Get the child node at the given index, if it exists.
    */
    maybeChild(index2) {
      return this.content.maybeChild(index2);
    }
    /**
    Call `f` for every child node, passing the node, its offset
    into this parent node, and its index.
    */
    forEach(f2) {
      this.content.forEach(f2);
    }
    /**
    Invoke a callback for all descendant nodes recursively between
    the given two positions that are relative to start of this
    node's content. The callback is invoked with the node, its
    position relative to the original node (method receiver),
    its parent node, and its child index. When the callback returns
    false for a given node, that node's children will not be
    recursed over. The last parameter can be used to specify a
    starting position to count from.
    */
    nodesBetween(from, to, f2, startPos = 0) {
      this.content.nodesBetween(from, to, f2, startPos, this);
    }
    /**
    Call the given callback for every descendant node. Doesn't
    descend into a node when the callback returns `false`.
    */
    descendants(f2) {
      this.nodesBetween(0, this.content.size, f2);
    }
    /**
    Concatenates all the text nodes found in this fragment and its
    children.
    */
    get textContent() {
      return this.isLeaf && this.type.spec.leafText ? this.type.spec.leafText(this) : this.textBetween(0, this.content.size, "");
    }
    /**
    Get all text between positions `from` and `to`. When
    `blockSeparator` is given, it will be inserted to separate text
    from different block nodes. If `leafText` is given, it'll be
    inserted for every non-text leaf node encountered, otherwise
    [`leafText`](https://prosemirror.net/docs/ref/#model.NodeSpec.leafText) will be used.
    */
    textBetween(from, to, blockSeparator, leafText) {
      return this.content.textBetween(from, to, blockSeparator, leafText);
    }
    /**
    Returns this node's first child, or `null` if there are no
    children.
    */
    get firstChild() {
      return this.content.firstChild;
    }
    /**
    Returns this node's last child, or `null` if there are no
    children.
    */
    get lastChild() {
      return this.content.lastChild;
    }
    /**
    Test whether two nodes represent the same piece of document.
    */
    eq(other) {
      return this == other || this.sameMarkup(other) && this.content.eq(other.content);
    }
    /**
    Compare the markup (type, attributes, and marks) of this node to
    those of another. Returns `true` if both have the same markup.
    */
    sameMarkup(other) {
      return this.hasMarkup(other.type, other.attrs, other.marks);
    }
    /**
    Check whether this node's markup correspond to the given type,
    attributes, and marks.
    */
    hasMarkup(type, attrs, marks) {
      return this.type == type && compareDeep(this.attrs, attrs || type.defaultAttrs || emptyAttrs) && Mark.sameSet(this.marks, marks || Mark.none);
    }
    /**
    Create a new node with the same markup as this node, containing
    the given content (or empty, if no content is given).
    */
    copy(content3 = null) {
      if (content3 == this.content)
        return this;
      return new _Node(this.type, this.attrs, content3, this.marks);
    }
    /**
    Create a copy of this node, with the given set of marks instead
    of the node's own marks.
    */
    mark(marks) {
      return marks == this.marks ? this : new _Node(this.type, this.attrs, this.content, marks);
    }
    /**
    Create a copy of this node with only the content between the
    given positions. If `to` is not given, it defaults to the end of
    the node.
    */
    cut(from, to = this.content.size) {
      if (from == 0 && to == this.content.size)
        return this;
      return this.copy(this.content.cut(from, to));
    }
    /**
    Cut out the part of the document between the given positions, and
    return it as a `Slice` object.
    */
    slice(from, to = this.content.size, includeParents = false) {
      if (from == to)
        return Slice2.empty;
      let $from = this.resolve(from), $to = this.resolve(to);
      let depth = includeParents ? 0 : $from.sharedDepth(to);
      let start = $from.start(depth), node2 = $from.node(depth);
      let content3 = node2.content.cut($from.pos - start, $to.pos - start);
      return new Slice2(content3, $from.depth - depth, $to.depth - depth);
    }
    /**
    Replace the part of the document between the given positions with
    the given slice. The slice must 'fit', meaning its open sides
    must be able to connect to the surrounding content, and its
    content nodes must be valid children for the node they are placed
    into. If any of this is violated, an error of type
    [`ReplaceError`](https://prosemirror.net/docs/ref/#model.ReplaceError) is thrown.
    */
    replace(from, to, slice) {
      return replace(this.resolve(from), this.resolve(to), slice);
    }
    /**
    Find the node directly after the given position.
    */
    nodeAt(pos) {
      for (let node2 = this; ; ) {
        let { index: index2, offset } = node2.content.findIndex(pos);
        node2 = node2.maybeChild(index2);
        if (!node2)
          return null;
        if (offset == pos || node2.isText)
          return node2;
        pos -= offset + 1;
      }
    }
    /**
    Find the (direct) child node after the given offset, if any,
    and return it along with its index and offset relative to this
    node.
    */
    childAfter(pos) {
      let { index: index2, offset } = this.content.findIndex(pos);
      return { node: this.content.maybeChild(index2), index: index2, offset };
    }
    /**
    Find the (direct) child node before the given offset, if any,
    and return it along with its index and offset relative to this
    node.
    */
    childBefore(pos) {
      if (pos == 0)
        return { node: null, index: 0, offset: 0 };
      let { index: index2, offset } = this.content.findIndex(pos);
      if (offset < pos)
        return { node: this.content.child(index2), index: index2, offset };
      let node2 = this.content.child(index2 - 1);
      return { node: node2, index: index2 - 1, offset: offset - node2.nodeSize };
    }
    /**
    Resolve the given position in the document, returning an
    [object](https://prosemirror.net/docs/ref/#model.ResolvedPos) with information about its context.
    */
    resolve(pos) {
      return ResolvedPos.resolveCached(this, pos);
    }
    /**
    @internal
    */
    resolveNoCache(pos) {
      return ResolvedPos.resolve(this, pos);
    }
    /**
    Test whether a given mark or mark type occurs in this document
    between the two given positions.
    */
    rangeHasMark(from, to, type) {
      let found2 = false;
      if (to > from)
        this.nodesBetween(from, to, (node2) => {
          if (type.isInSet(node2.marks))
            found2 = true;
          return !found2;
        });
      return found2;
    }
    /**
    True when this is a block (non-inline node)
    */
    get isBlock() {
      return this.type.isBlock;
    }
    /**
    True when this is a textblock node, a block node with inline
    content.
    */
    get isTextblock() {
      return this.type.isTextblock;
    }
    /**
    True when this node allows inline content.
    */
    get inlineContent() {
      return this.type.inlineContent;
    }
    /**
    True when this is an inline node (a text node or a node that can
    appear among text).
    */
    get isInline() {
      return this.type.isInline;
    }
    /**
    True when this is a text node.
    */
    get isText() {
      return this.type.isText;
    }
    /**
    True when this is a leaf node.
    */
    get isLeaf() {
      return this.type.isLeaf;
    }
    /**
    True when this is an atom, i.e. when it does not have directly
    editable content. This is usually the same as `isLeaf`, but can
    be configured with the [`atom` property](https://prosemirror.net/docs/ref/#model.NodeSpec.atom)
    on a node's spec (typically used when the node is displayed as
    an uneditable [node view](https://prosemirror.net/docs/ref/#view.NodeView)).
    */
    get isAtom() {
      return this.type.isAtom;
    }
    /**
    Return a string representation of this node for debugging
    purposes.
    */
    toString() {
      if (this.type.spec.toDebugString)
        return this.type.spec.toDebugString(this);
      let name = this.type.name;
      if (this.content.size)
        name += "(" + this.content.toStringInner() + ")";
      return wrapMarks(this.marks, name);
    }
    /**
    Get the content match in this node at the given index.
    */
    contentMatchAt(index2) {
      let match = this.type.contentMatch.matchFragment(this.content, 0, index2);
      if (!match)
        throw new Error("Called contentMatchAt on a node with invalid content");
      return match;
    }
    /**
    Test whether replacing the range between `from` and `to` (by
    child index) with the given replacement fragment (which defaults
    to the empty fragment) would leave the node's content valid. You
    can optionally pass `start` and `end` indices into the
    replacement fragment.
    */
    canReplace(from, to, replacement = Fragment.empty, start = 0, end = replacement.childCount) {
      let one2 = this.contentMatchAt(from).matchFragment(replacement, start, end);
      let two = one2 && one2.matchFragment(this.content, to);
      if (!two || !two.validEnd)
        return false;
      for (let i2 = start; i2 < end; i2++)
        if (!this.type.allowsMarks(replacement.child(i2).marks))
          return false;
      return true;
    }
    /**
    Test whether replacing the range `from` to `to` (by index) with
    a node of the given type would leave the node's content valid.
    */
    canReplaceWith(from, to, type, marks) {
      if (marks && !this.type.allowsMarks(marks))
        return false;
      let start = this.contentMatchAt(from).matchType(type);
      let end = start && start.matchFragment(this.content, to);
      return end ? end.validEnd : false;
    }
    /**
    Test whether the given node's content could be appended to this
    node. If that node is empty, this will only return true if there
    is at least one node type that can appear in both nodes (to avoid
    merging completely incompatible nodes).
    */
    canAppend(other) {
      if (other.content.size)
        return this.canReplace(this.childCount, this.childCount, other.content);
      else
        return this.type.compatibleContent(other.type);
    }
    /**
    Check whether this node and its descendants conform to the
    schema, and raise an exception when they do not.
    */
    check() {
      this.type.checkContent(this.content);
      this.type.checkAttrs(this.attrs);
      let copy2 = Mark.none;
      for (let i2 = 0; i2 < this.marks.length; i2++) {
        let mark = this.marks[i2];
        mark.type.checkAttrs(mark.attrs);
        copy2 = mark.addToSet(copy2);
      }
      if (!Mark.sameSet(copy2, this.marks))
        throw new RangeError(`Invalid collection of marks for node ${this.type.name}: ${this.marks.map((m) => m.type.name)}`);
      this.content.forEach((node2) => node2.check());
    }
    /**
    Return a JSON-serializeable representation of this node.
    */
    toJSON() {
      let obj = { type: this.type.name };
      for (let _ in this.attrs) {
        obj.attrs = this.attrs;
        break;
      }
      if (this.content.size)
        obj.content = this.content.toJSON();
      if (this.marks.length)
        obj.marks = this.marks.map((n2) => n2.toJSON());
      return obj;
    }
    /**
    Deserialize a node from its JSON representation.
    */
    static fromJSON(schema3, json) {
      if (!json)
        throw new RangeError("Invalid input for Node.fromJSON");
      let marks = void 0;
      if (json.marks) {
        if (!Array.isArray(json.marks))
          throw new RangeError("Invalid mark data for Node.fromJSON");
        marks = json.marks.map(schema3.markFromJSON);
      }
      if (json.type == "text") {
        if (typeof json.text != "string")
          throw new RangeError("Invalid text node in JSON");
        return schema3.text(json.text, marks);
      }
      let content3 = Fragment.fromJSON(schema3, json.content);
      let node2 = schema3.nodeType(json.type).create(json.attrs, content3, marks);
      node2.type.checkAttrs(node2.attrs);
      return node2;
    }
  };
  Node.prototype.text = void 0;
  var TextNode = class _TextNode extends Node {
    /**
    @internal
    */
    constructor(type, attrs, content3, marks) {
      super(type, attrs, null, marks);
      if (!content3)
        throw new RangeError("Empty text nodes are not allowed");
      this.text = content3;
    }
    toString() {
      if (this.type.spec.toDebugString)
        return this.type.spec.toDebugString(this);
      return wrapMarks(this.marks, JSON.stringify(this.text));
    }
    get textContent() {
      return this.text;
    }
    textBetween(from, to) {
      return this.text.slice(from, to);
    }
    get nodeSize() {
      return this.text.length;
    }
    mark(marks) {
      return marks == this.marks ? this : new _TextNode(this.type, this.attrs, this.text, marks);
    }
    withText(text4) {
      if (text4 == this.text)
        return this;
      return new _TextNode(this.type, this.attrs, text4, this.marks);
    }
    cut(from = 0, to = this.text.length) {
      if (from == 0 && to == this.text.length)
        return this;
      return this.withText(this.text.slice(from, to));
    }
    eq(other) {
      return this.sameMarkup(other) && this.text == other.text;
    }
    toJSON() {
      let base2 = super.toJSON();
      base2.text = this.text;
      return base2;
    }
  };
  function wrapMarks(marks, str) {
    for (let i2 = marks.length - 1; i2 >= 0; i2--)
      str = marks[i2].type.name + "(" + str + ")";
    return str;
  }
  var ContentMatch = class _ContentMatch {
    /**
    @internal
    */
    constructor(validEnd) {
      this.validEnd = validEnd;
      this.next = [];
      this.wrapCache = [];
    }
    /**
    @internal
    */
    static parse(string3, nodeTypes) {
      let stream = new TokenStream(string3, nodeTypes);
      if (stream.next == null)
        return _ContentMatch.empty;
      let expr = parseExpr(stream);
      if (stream.next)
        stream.err("Unexpected trailing text");
      let match = dfa(nfa(expr));
      checkForDeadEnds(match, stream);
      return match;
    }
    /**
    Match a node type, returning a match after that node if
    successful.
    */
    matchType(type) {
      for (let i2 = 0; i2 < this.next.length; i2++)
        if (this.next[i2].type == type)
          return this.next[i2].next;
      return null;
    }
    /**
    Try to match a fragment. Returns the resulting match when
    successful.
    */
    matchFragment(frag, start = 0, end = frag.childCount) {
      let cur = this;
      for (let i2 = start; cur && i2 < end; i2++)
        cur = cur.matchType(frag.child(i2).type);
      return cur;
    }
    /**
    @internal
    */
    get inlineContent() {
      return this.next.length != 0 && this.next[0].type.isInline;
    }
    /**
    Get the first matching node type at this match position that can
    be generated.
    */
    get defaultType() {
      for (let i2 = 0; i2 < this.next.length; i2++) {
        let { type } = this.next[i2];
        if (!(type.isText || type.hasRequiredAttrs()))
          return type;
      }
      return null;
    }
    /**
    @internal
    */
    compatible(other) {
      for (let i2 = 0; i2 < this.next.length; i2++)
        for (let j = 0; j < other.next.length; j++)
          if (this.next[i2].type == other.next[j].type)
            return true;
      return false;
    }
    /**
    Try to match the given fragment, and if that fails, see if it can
    be made to match by inserting nodes in front of it. When
    successful, return a fragment of inserted nodes (which may be
    empty if nothing had to be inserted). When `toEnd` is true, only
    return a fragment if the resulting match goes to the end of the
    content expression.
    */
    fillBefore(after, toEnd = false, startIndex = 0) {
      let seen = [this];
      function search2(match, types) {
        let finished = match.matchFragment(after, startIndex);
        if (finished && (!toEnd || finished.validEnd))
          return Fragment.from(types.map((tp) => tp.createAndFill()));
        for (let i2 = 0; i2 < match.next.length; i2++) {
          let { type, next } = match.next[i2];
          if (!(type.isText || type.hasRequiredAttrs()) && seen.indexOf(next) == -1) {
            seen.push(next);
            let found2 = search2(next, types.concat(type));
            if (found2)
              return found2;
          }
        }
        return null;
      }
      return search2(this, []);
    }
    /**
    Find a set of wrapping node types that would allow a node of the
    given type to appear at this position. The result may be empty
    (when it fits directly) and will be null when no such wrapping
    exists.
    */
    findWrapping(target) {
      for (let i2 = 0; i2 < this.wrapCache.length; i2 += 2)
        if (this.wrapCache[i2] == target)
          return this.wrapCache[i2 + 1];
      let computed = this.computeWrapping(target);
      this.wrapCache.push(target, computed);
      return computed;
    }
    /**
    @internal
    */
    computeWrapping(target) {
      let seen = /* @__PURE__ */ Object.create(null), active = [{ match: this, type: null, via: null }];
      while (active.length) {
        let current = active.shift(), match = current.match;
        if (match.matchType(target)) {
          let result = [];
          for (let obj = current; obj.type; obj = obj.via)
            result.push(obj.type);
          return result.reverse();
        }
        for (let i2 = 0; i2 < match.next.length; i2++) {
          let { type, next } = match.next[i2];
          if (!type.isLeaf && !type.hasRequiredAttrs() && !(type.name in seen) && (!current.type || next.validEnd)) {
            active.push({ match: type.contentMatch, type, via: current });
            seen[type.name] = true;
          }
        }
      }
      return null;
    }
    /**
    The number of outgoing edges this node has in the finite
    automaton that describes the content expression.
    */
    get edgeCount() {
      return this.next.length;
    }
    /**
    Get the _n_​th outgoing edge from this node in the finite
    automaton that describes the content expression.
    */
    edge(n2) {
      if (n2 >= this.next.length)
        throw new RangeError(`There's no ${n2}th edge in this content match`);
      return this.next[n2];
    }
    /**
    @internal
    */
    toString() {
      let seen = [];
      function scan(m) {
        seen.push(m);
        for (let i2 = 0; i2 < m.next.length; i2++)
          if (seen.indexOf(m.next[i2].next) == -1)
            scan(m.next[i2].next);
      }
      scan(this);
      return seen.map((m, i2) => {
        let out = i2 + (m.validEnd ? "*" : " ") + " ";
        for (let i3 = 0; i3 < m.next.length; i3++)
          out += (i3 ? ", " : "") + m.next[i3].type.name + "->" + seen.indexOf(m.next[i3].next);
        return out;
      }).join("\n");
    }
  };
  ContentMatch.empty = new ContentMatch(true);
  var TokenStream = class {
    constructor(string3, nodeTypes) {
      this.string = string3;
      this.nodeTypes = nodeTypes;
      this.inline = null;
      this.pos = 0;
      this.tokens = string3.split(/\s*(?=\b|\W|$)/);
      if (this.tokens[this.tokens.length - 1] == "")
        this.tokens.pop();
      if (this.tokens[0] == "")
        this.tokens.shift();
    }
    get next() {
      return this.tokens[this.pos];
    }
    eat(tok) {
      return this.next == tok && (this.pos++ || true);
    }
    err(str) {
      throw new SyntaxError(str + " (in content expression '" + this.string + "')");
    }
  };
  function parseExpr(stream) {
    let exprs = [];
    do {
      exprs.push(parseExprSeq(stream));
    } while (stream.eat("|"));
    return exprs.length == 1 ? exprs[0] : { type: "choice", exprs };
  }
  function parseExprSeq(stream) {
    let exprs = [];
    do {
      exprs.push(parseExprSubscript(stream));
    } while (stream.next && stream.next != ")" && stream.next != "|");
    return exprs.length == 1 ? exprs[0] : { type: "seq", exprs };
  }
  function parseExprSubscript(stream) {
    let expr = parseExprAtom(stream);
    for (; ; ) {
      if (stream.eat("+"))
        expr = { type: "plus", expr };
      else if (stream.eat("*"))
        expr = { type: "star", expr };
      else if (stream.eat("?"))
        expr = { type: "opt", expr };
      else if (stream.eat("{"))
        expr = parseExprRange(stream, expr);
      else
        break;
    }
    return expr;
  }
  function parseNum(stream) {
    if (/\D/.test(stream.next))
      stream.err("Expected number, got '" + stream.next + "'");
    let result = Number(stream.next);
    stream.pos++;
    return result;
  }
  function parseExprRange(stream, expr) {
    let min = parseNum(stream), max = min;
    if (stream.eat(",")) {
      if (stream.next != "}")
        max = parseNum(stream);
      else
        max = -1;
    }
    if (!stream.eat("}"))
      stream.err("Unclosed braced range");
    return { type: "range", min, max, expr };
  }
  function resolveName(stream, name) {
    let types = stream.nodeTypes, type = types[name];
    if (type)
      return [type];
    let result = [];
    for (let typeName in types) {
      let type2 = types[typeName];
      if (type2.isInGroup(name))
        result.push(type2);
    }
    if (result.length == 0)
      stream.err("No node type or group '" + name + "' found");
    return result;
  }
  function parseExprAtom(stream) {
    if (stream.eat("(")) {
      let expr = parseExpr(stream);
      if (!stream.eat(")"))
        stream.err("Missing closing paren");
      return expr;
    } else if (!/\W/.test(stream.next)) {
      let exprs = resolveName(stream, stream.next).map((type) => {
        if (stream.inline == null)
          stream.inline = type.isInline;
        else if (stream.inline != type.isInline)
          stream.err("Mixing inline and block content");
        return { type: "name", value: type };
      });
      stream.pos++;
      return exprs.length == 1 ? exprs[0] : { type: "choice", exprs };
    } else {
      stream.err("Unexpected token '" + stream.next + "'");
    }
  }
  function nfa(expr) {
    let nfa2 = [[]];
    connect(compile(expr, 0), node2());
    return nfa2;
    function node2() {
      return nfa2.push([]) - 1;
    }
    function edge(from, to, term) {
      let edge2 = { term, to };
      nfa2[from].push(edge2);
      return edge2;
    }
    function connect(edges, to) {
      edges.forEach((edge2) => edge2.to = to);
    }
    function compile(expr2, from) {
      if (expr2.type == "choice") {
        return expr2.exprs.reduce((out, expr3) => out.concat(compile(expr3, from)), []);
      } else if (expr2.type == "seq") {
        for (let i2 = 0; ; i2++) {
          let next = compile(expr2.exprs[i2], from);
          if (i2 == expr2.exprs.length - 1)
            return next;
          connect(next, from = node2());
        }
      } else if (expr2.type == "star") {
        let loop = node2();
        edge(from, loop);
        connect(compile(expr2.expr, loop), loop);
        return [edge(loop)];
      } else if (expr2.type == "plus") {
        let loop = node2();
        connect(compile(expr2.expr, from), loop);
        connect(compile(expr2.expr, loop), loop);
        return [edge(loop)];
      } else if (expr2.type == "opt") {
        return [edge(from)].concat(compile(expr2.expr, from));
      } else if (expr2.type == "range") {
        let cur = from;
        for (let i2 = 0; i2 < expr2.min; i2++) {
          let next = node2();
          connect(compile(expr2.expr, cur), next);
          cur = next;
        }
        if (expr2.max == -1) {
          connect(compile(expr2.expr, cur), cur);
        } else {
          for (let i2 = expr2.min; i2 < expr2.max; i2++) {
            let next = node2();
            edge(cur, next);
            connect(compile(expr2.expr, cur), next);
            cur = next;
          }
        }
        return [edge(cur)];
      } else if (expr2.type == "name") {
        return [edge(from, void 0, expr2.value)];
      } else {
        throw new Error("Unknown expr type");
      }
    }
  }
  function cmp(a2, b) {
    return b - a2;
  }
  function nullFrom(nfa2, node2) {
    let result = [];
    scan(node2);
    return result.sort(cmp);
    function scan(node3) {
      let edges = nfa2[node3];
      if (edges.length == 1 && !edges[0].term)
        return scan(edges[0].to);
      result.push(node3);
      for (let i2 = 0; i2 < edges.length; i2++) {
        let { term, to } = edges[i2];
        if (!term && result.indexOf(to) == -1)
          scan(to);
      }
    }
  }
  function dfa(nfa2) {
    let labeled = /* @__PURE__ */ Object.create(null);
    return explore(nullFrom(nfa2, 0));
    function explore(states) {
      let out = [];
      states.forEach((node2) => {
        nfa2[node2].forEach(({ term, to }) => {
          if (!term)
            return;
          let set;
          for (let i2 = 0; i2 < out.length; i2++)
            if (out[i2][0] == term)
              set = out[i2][1];
          nullFrom(nfa2, to).forEach((node3) => {
            if (!set)
              out.push([term, set = []]);
            if (set.indexOf(node3) == -1)
              set.push(node3);
          });
        });
      });
      let state = labeled[states.join(",")] = new ContentMatch(states.indexOf(nfa2.length - 1) > -1);
      for (let i2 = 0; i2 < out.length; i2++) {
        let states2 = out[i2][1].sort(cmp);
        state.next.push({ type: out[i2][0], next: labeled[states2.join(",")] || explore(states2) });
      }
      return state;
    }
  }
  function checkForDeadEnds(match, stream) {
    for (let i2 = 0, work = [match]; i2 < work.length; i2++) {
      let state = work[i2], dead = !state.validEnd, nodes = [];
      for (let j = 0; j < state.next.length; j++) {
        let { type, next } = state.next[j];
        nodes.push(type.name);
        if (dead && !(type.isText || type.hasRequiredAttrs()))
          dead = false;
        if (work.indexOf(next) == -1)
          work.push(next);
      }
      if (dead)
        stream.err("Only non-generatable nodes (" + nodes.join(", ") + ") in a required position (see https://prosemirror.net/docs/guide/#generatable)");
    }
  }
  function defaultAttrs(attrs) {
    let defaults = /* @__PURE__ */ Object.create(null);
    for (let attrName in attrs) {
      let attr = attrs[attrName];
      if (!attr.hasDefault)
        return null;
      defaults[attrName] = attr.default;
    }
    return defaults;
  }
  function computeAttrs(attrs, value) {
    let built = /* @__PURE__ */ Object.create(null);
    for (let name in attrs) {
      let given = value && value[name];
      if (given === void 0) {
        let attr = attrs[name];
        if (attr.hasDefault)
          given = attr.default;
        else
          throw new RangeError("No value supplied for attribute " + name);
      }
      built[name] = given;
    }
    return built;
  }
  function checkAttrs(attrs, values, type, name) {
    for (let name2 in values)
      if (!(name2 in attrs))
        throw new RangeError(`Unsupported attribute ${name2} for ${type} of type ${name2}`);
    for (let name2 in attrs) {
      let attr = attrs[name2];
      if (attr.validate)
        attr.validate(values[name2]);
    }
  }
  function initAttrs(typeName, attrs) {
    let result = /* @__PURE__ */ Object.create(null);
    if (attrs)
      for (let name in attrs)
        result[name] = new Attribute(typeName, name, attrs[name]);
    return result;
  }
  var NodeType = class _NodeType {
    /**
    @internal
    */
    constructor(name, schema3, spec) {
      this.name = name;
      this.schema = schema3;
      this.spec = spec;
      this.markSet = null;
      this.groups = spec.group ? spec.group.split(" ") : [];
      this.attrs = initAttrs(name, spec.attrs);
      this.defaultAttrs = defaultAttrs(this.attrs);
      this.contentMatch = null;
      this.inlineContent = null;
      this.isBlock = !(spec.inline || name == "text");
      this.isText = name == "text";
    }
    /**
    True if this is an inline type.
    */
    get isInline() {
      return !this.isBlock;
    }
    /**
    True if this is a textblock type, a block that contains inline
    content.
    */
    get isTextblock() {
      return this.isBlock && this.inlineContent;
    }
    /**
    True for node types that allow no content.
    */
    get isLeaf() {
      return this.contentMatch == ContentMatch.empty;
    }
    /**
    True when this node is an atom, i.e. when it does not have
    directly editable content.
    */
    get isAtom() {
      return this.isLeaf || !!this.spec.atom;
    }
    /**
    Return true when this node type is part of the given
    [group](https://prosemirror.net/docs/ref/#model.NodeSpec.group).
    */
    isInGroup(group) {
      return this.groups.indexOf(group) > -1;
    }
    /**
    The node type's [whitespace](https://prosemirror.net/docs/ref/#model.NodeSpec.whitespace) option.
    */
    get whitespace() {
      return this.spec.whitespace || (this.spec.code ? "pre" : "normal");
    }
    /**
    Tells you whether this node type has any required attributes.
    */
    hasRequiredAttrs() {
      for (let n2 in this.attrs)
        if (this.attrs[n2].isRequired)
          return true;
      return false;
    }
    /**
    Indicates whether this node allows some of the same content as
    the given node type.
    */
    compatibleContent(other) {
      return this == other || this.contentMatch.compatible(other.contentMatch);
    }
    /**
    @internal
    */
    computeAttrs(attrs) {
      if (!attrs && this.defaultAttrs)
        return this.defaultAttrs;
      else
        return computeAttrs(this.attrs, attrs);
    }
    /**
    Create a `Node` of this type. The given attributes are
    checked and defaulted (you can pass `null` to use the type's
    defaults entirely, if no required attributes exist). `content`
    may be a `Fragment`, a node, an array of nodes, or
    `null`. Similarly `marks` may be `null` to default to the empty
    set of marks.
    */
    create(attrs = null, content3, marks) {
      if (this.isText)
        throw new Error("NodeType.create can't construct text nodes");
      return new Node(this, this.computeAttrs(attrs), Fragment.from(content3), Mark.setFrom(marks));
    }
    /**
    Like [`create`](https://prosemirror.net/docs/ref/#model.NodeType.create), but check the given content
    against the node type's content restrictions, and throw an error
    if it doesn't match.
    */
    createChecked(attrs = null, content3, marks) {
      content3 = Fragment.from(content3);
      this.checkContent(content3);
      return new Node(this, this.computeAttrs(attrs), content3, Mark.setFrom(marks));
    }
    /**
    Like [`create`](https://prosemirror.net/docs/ref/#model.NodeType.create), but see if it is
    necessary to add nodes to the start or end of the given fragment
    to make it fit the node. If no fitting wrapping can be found,
    return null. Note that, due to the fact that required nodes can
    always be created, this will always succeed if you pass null or
    `Fragment.empty` as content.
    */
    createAndFill(attrs = null, content3, marks) {
      attrs = this.computeAttrs(attrs);
      content3 = Fragment.from(content3);
      if (content3.size) {
        let before = this.contentMatch.fillBefore(content3);
        if (!before)
          return null;
        content3 = before.append(content3);
      }
      let matched = this.contentMatch.matchFragment(content3);
      let after = matched && matched.fillBefore(Fragment.empty, true);
      if (!after)
        return null;
      return new Node(this, attrs, content3.append(after), Mark.setFrom(marks));
    }
    /**
    Returns true if the given fragment is valid content for this node
    type.
    */
    validContent(content3) {
      let result = this.contentMatch.matchFragment(content3);
      if (!result || !result.validEnd)
        return false;
      for (let i2 = 0; i2 < content3.childCount; i2++)
        if (!this.allowsMarks(content3.child(i2).marks))
          return false;
      return true;
    }
    /**
    Throws a RangeError if the given fragment is not valid content for this
    node type.
    @internal
    */
    checkContent(content3) {
      if (!this.validContent(content3))
        throw new RangeError(`Invalid content for node ${this.name}: ${content3.toString().slice(0, 50)}`);
    }
    /**
    @internal
    */
    checkAttrs(attrs) {
      checkAttrs(this.attrs, attrs, "node", this.name);
    }
    /**
    Check whether the given mark type is allowed in this node.
    */
    allowsMarkType(markType) {
      return this.markSet == null || this.markSet.indexOf(markType) > -1;
    }
    /**
    Test whether the given set of marks are allowed in this node.
    */
    allowsMarks(marks) {
      if (this.markSet == null)
        return true;
      for (let i2 = 0; i2 < marks.length; i2++)
        if (!this.allowsMarkType(marks[i2].type))
          return false;
      return true;
    }
    /**
    Removes the marks that are not allowed in this node from the given set.
    */
    allowedMarks(marks) {
      if (this.markSet == null)
        return marks;
      let copy2;
      for (let i2 = 0; i2 < marks.length; i2++) {
        if (!this.allowsMarkType(marks[i2].type)) {
          if (!copy2)
            copy2 = marks.slice(0, i2);
        } else if (copy2) {
          copy2.push(marks[i2]);
        }
      }
      return !copy2 ? marks : copy2.length ? copy2 : Mark.none;
    }
    /**
    @internal
    */
    static compile(nodes, schema3) {
      let result = /* @__PURE__ */ Object.create(null);
      nodes.forEach((name, spec) => result[name] = new _NodeType(name, schema3, spec));
      let topType = schema3.spec.topNode || "doc";
      if (!result[topType])
        throw new RangeError("Schema is missing its top node type ('" + topType + "')");
      if (!result.text)
        throw new RangeError("Every schema needs a 'text' type");
      for (let _ in result.text.attrs)
        throw new RangeError("The text node type should not have attributes");
      return result;
    }
  };
  function validateType(typeName, attrName, type) {
    let types = type.split("|");
    return (value) => {
      let name = value === null ? "null" : typeof value;
      if (types.indexOf(name) < 0)
        throw new RangeError(`Expected value of type ${types} for attribute ${attrName} on type ${typeName}, got ${name}`);
    };
  }
  var Attribute = class {
    constructor(typeName, attrName, options) {
      this.hasDefault = Object.prototype.hasOwnProperty.call(options, "default");
      this.default = options.default;
      this.validate = typeof options.validate == "string" ? validateType(typeName, attrName, options.validate) : options.validate;
    }
    get isRequired() {
      return !this.hasDefault;
    }
  };
  var MarkType = class _MarkType {
    /**
    @internal
    */
    constructor(name, rank, schema3, spec) {
      this.name = name;
      this.rank = rank;
      this.schema = schema3;
      this.spec = spec;
      this.attrs = initAttrs(name, spec.attrs);
      this.excluded = null;
      let defaults = defaultAttrs(this.attrs);
      this.instance = defaults ? new Mark(this, defaults) : null;
    }
    /**
    Create a mark of this type. `attrs` may be `null` or an object
    containing only some of the mark's attributes. The others, if
    they have defaults, will be added.
    */
    create(attrs = null) {
      if (!attrs && this.instance)
        return this.instance;
      return new Mark(this, computeAttrs(this.attrs, attrs));
    }
    /**
    @internal
    */
    static compile(marks, schema3) {
      let result = /* @__PURE__ */ Object.create(null), rank = 0;
      marks.forEach((name, spec) => result[name] = new _MarkType(name, rank++, schema3, spec));
      return result;
    }
    /**
    When there is a mark of this type in the given set, a new set
    without it is returned. Otherwise, the input set is returned.
    */
    removeFromSet(set) {
      for (var i2 = 0; i2 < set.length; i2++)
        if (set[i2].type == this) {
          set = set.slice(0, i2).concat(set.slice(i2 + 1));
          i2--;
        }
      return set;
    }
    /**
    Tests whether there is a mark of this type in the given set.
    */
    isInSet(set) {
      for (let i2 = 0; i2 < set.length; i2++)
        if (set[i2].type == this)
          return set[i2];
    }
    /**
    @internal
    */
    checkAttrs(attrs) {
      checkAttrs(this.attrs, attrs, "mark", this.name);
    }
    /**
    Queries whether a given mark type is
    [excluded](https://prosemirror.net/docs/ref/#model.MarkSpec.excludes) by this one.
    */
    excludes(other) {
      return this.excluded.indexOf(other) > -1;
    }
  };
  var Schema = class {
    /**
    Construct a schema from a schema [specification](https://prosemirror.net/docs/ref/#model.SchemaSpec).
    */
    constructor(spec) {
      this.linebreakReplacement = null;
      this.cached = /* @__PURE__ */ Object.create(null);
      let instanceSpec = this.spec = {};
      for (let prop in spec)
        instanceSpec[prop] = spec[prop];
      instanceSpec.nodes = dist_default.from(spec.nodes), instanceSpec.marks = dist_default.from(spec.marks || {}), this.nodes = NodeType.compile(this.spec.nodes, this);
      this.marks = MarkType.compile(this.spec.marks, this);
      let contentExprCache = /* @__PURE__ */ Object.create(null);
      for (let prop in this.nodes) {
        if (prop in this.marks)
          throw new RangeError(prop + " can not be both a node and a mark");
        let type = this.nodes[prop], contentExpr = type.spec.content || "", markExpr = type.spec.marks;
        type.contentMatch = contentExprCache[contentExpr] || (contentExprCache[contentExpr] = ContentMatch.parse(contentExpr, this.nodes));
        type.inlineContent = type.contentMatch.inlineContent;
        if (type.spec.linebreakReplacement) {
          if (this.linebreakReplacement)
            throw new RangeError("Multiple linebreak nodes defined");
          if (!type.isInline || !type.isLeaf)
            throw new RangeError("Linebreak replacement nodes must be inline leaf nodes");
          this.linebreakReplacement = type;
        }
        type.markSet = markExpr == "_" ? null : markExpr ? gatherMarks(this, markExpr.split(" ")) : markExpr == "" || !type.inlineContent ? [] : null;
      }
      for (let prop in this.marks) {
        let type = this.marks[prop], excl = type.spec.excludes;
        type.excluded = excl == null ? [type] : excl == "" ? [] : gatherMarks(this, excl.split(" "));
      }
      this.nodeFromJSON = (json) => Node.fromJSON(this, json);
      this.markFromJSON = (json) => Mark.fromJSON(this, json);
      this.topNodeType = this.nodes[this.spec.topNode || "doc"];
      this.cached.wrappings = /* @__PURE__ */ Object.create(null);
    }
    /**
    Create a node in this schema. The `type` may be a string or a
    `NodeType` instance. Attributes will be extended with defaults,
    `content` may be a `Fragment`, `null`, a `Node`, or an array of
    nodes.
    */
    node(type, attrs = null, content3, marks) {
      if (typeof type == "string")
        type = this.nodeType(type);
      else if (!(type instanceof NodeType))
        throw new RangeError("Invalid node type: " + type);
      else if (type.schema != this)
        throw new RangeError("Node type from different schema used (" + type.name + ")");
      return type.createChecked(attrs, content3, marks);
    }
    /**
    Create a text node in the schema. Empty text nodes are not
    allowed.
    */
    text(text4, marks) {
      let type = this.nodes.text;
      return new TextNode(type, type.defaultAttrs, text4, Mark.setFrom(marks));
    }
    /**
    Create a mark with the given type and attributes.
    */
    mark(type, attrs) {
      if (typeof type == "string")
        type = this.marks[type];
      return type.create(attrs);
    }
    /**
    @internal
    */
    nodeType(name) {
      let found2 = this.nodes[name];
      if (!found2)
        throw new RangeError("Unknown node type: " + name);
      return found2;
    }
  };
  function gatherMarks(schema3, marks) {
    let found2 = [];
    for (let i2 = 0; i2 < marks.length; i2++) {
      let name = marks[i2], mark = schema3.marks[name], ok3 = mark;
      if (mark) {
        found2.push(mark);
      } else {
        for (let prop in schema3.marks) {
          let mark2 = schema3.marks[prop];
          if (name == "_" || mark2.spec.group && mark2.spec.group.split(" ").indexOf(name) > -1)
            found2.push(ok3 = mark2);
        }
      }
      if (!ok3)
        throw new SyntaxError("Unknown mark type: '" + marks[i2] + "'");
    }
    return found2;
  }
  function isTagRule(rule) {
    return rule.tag != null;
  }
  function isStyleRule(rule) {
    return rule.style != null;
  }
  var DOMParser = class _DOMParser {
    /**
    Create a parser that targets the given schema, using the given
    parsing rules.
    */
    constructor(schema3, rules) {
      this.schema = schema3;
      this.rules = rules;
      this.tags = [];
      this.styles = [];
      let matchedStyles = this.matchedStyles = [];
      rules.forEach((rule) => {
        if (isTagRule(rule)) {
          this.tags.push(rule);
        } else if (isStyleRule(rule)) {
          let prop = /[^=]*/.exec(rule.style)[0];
          if (matchedStyles.indexOf(prop) < 0)
            matchedStyles.push(prop);
          this.styles.push(rule);
        }
      });
      this.normalizeLists = !this.tags.some((r2) => {
        if (!/^(ul|ol)\b/.test(r2.tag) || !r2.node)
          return false;
        let node2 = schema3.nodes[r2.node];
        return node2.contentMatch.matchType(node2);
      });
    }
    /**
    Parse a document from the content of a DOM node.
    */
    parse(dom, options = {}) {
      let context = new ParseContext(this, options, false);
      context.addAll(dom, Mark.none, options.from, options.to);
      return context.finish();
    }
    /**
    Parses the content of the given DOM node, like
    [`parse`](https://prosemirror.net/docs/ref/#model.DOMParser.parse), and takes the same set of
    options. But unlike that method, which produces a whole node,
    this one returns a slice that is open at the sides, meaning that
    the schema constraints aren't applied to the start of nodes to
    the left of the input and the end of nodes at the end.
    */
    parseSlice(dom, options = {}) {
      let context = new ParseContext(this, options, true);
      context.addAll(dom, Mark.none, options.from, options.to);
      return Slice2.maxOpen(context.finish());
    }
    /**
    @internal
    */
    matchTag(dom, context, after) {
      for (let i2 = after ? this.tags.indexOf(after) + 1 : 0; i2 < this.tags.length; i2++) {
        let rule = this.tags[i2];
        if (matches(dom, rule.tag) && (rule.namespace === void 0 || dom.namespaceURI == rule.namespace) && (!rule.context || context.matchesContext(rule.context))) {
          if (rule.getAttrs) {
            let result = rule.getAttrs(dom);
            if (result === false)
              continue;
            rule.attrs = result || void 0;
          }
          return rule;
        }
      }
    }
    /**
    @internal
    */
    matchStyle(prop, value, context, after) {
      for (let i2 = after ? this.styles.indexOf(after) + 1 : 0; i2 < this.styles.length; i2++) {
        let rule = this.styles[i2], style = rule.style;
        if (style.indexOf(prop) != 0 || rule.context && !context.matchesContext(rule.context) || // Test that the style string either precisely matches the prop,
        // or has an '=' sign after the prop, followed by the given
        // value.
        style.length > prop.length && (style.charCodeAt(prop.length) != 61 || style.slice(prop.length + 1) != value))
          continue;
        if (rule.getAttrs) {
          let result = rule.getAttrs(value);
          if (result === false)
            continue;
          rule.attrs = result || void 0;
        }
        return rule;
      }
    }
    /**
    @internal
    */
    static schemaRules(schema3) {
      let result = [];
      function insert(rule) {
        let priority = rule.priority == null ? 50 : rule.priority, i2 = 0;
        for (; i2 < result.length; i2++) {
          let next = result[i2], nextPriority = next.priority == null ? 50 : next.priority;
          if (nextPriority < priority)
            break;
        }
        result.splice(i2, 0, rule);
      }
      for (let name in schema3.marks) {
        let rules = schema3.marks[name].spec.parseDOM;
        if (rules)
          rules.forEach((rule) => {
            insert(rule = copy(rule));
            if (!(rule.mark || rule.ignore || rule.clearMark))
              rule.mark = name;
          });
      }
      for (let name in schema3.nodes) {
        let rules = schema3.nodes[name].spec.parseDOM;
        if (rules)
          rules.forEach((rule) => {
            insert(rule = copy(rule));
            if (!(rule.node || rule.ignore || rule.mark))
              rule.node = name;
          });
      }
      return result;
    }
    /**
    Construct a DOM parser using the parsing rules listed in a
    schema's [node specs](https://prosemirror.net/docs/ref/#model.NodeSpec.parseDOM), reordered by
    [priority](https://prosemirror.net/docs/ref/#model.GenericParseRule.priority).
    */
    static fromSchema(schema3) {
      return schema3.cached.domParser || (schema3.cached.domParser = new _DOMParser(schema3, _DOMParser.schemaRules(schema3)));
    }
  };
  var blockTags = {
    address: true,
    article: true,
    aside: true,
    blockquote: true,
    canvas: true,
    dd: true,
    div: true,
    dl: true,
    fieldset: true,
    figcaption: true,
    figure: true,
    footer: true,
    form: true,
    h1: true,
    h2: true,
    h3: true,
    h4: true,
    h5: true,
    h6: true,
    header: true,
    hgroup: true,
    hr: true,
    li: true,
    noscript: true,
    ol: true,
    output: true,
    p: true,
    pre: true,
    section: true,
    table: true,
    tfoot: true,
    ul: true
  };
  var ignoreTags = {
    head: true,
    noscript: true,
    object: true,
    script: true,
    style: true,
    title: true
  };
  var listTags = { ol: true, ul: true };
  var OPT_PRESERVE_WS = 1;
  var OPT_PRESERVE_WS_FULL = 2;
  var OPT_OPEN_LEFT = 4;
  function wsOptionsFor(type, preserveWhitespace, base2) {
    if (preserveWhitespace != null)
      return (preserveWhitespace ? OPT_PRESERVE_WS : 0) | (preserveWhitespace === "full" ? OPT_PRESERVE_WS_FULL : 0);
    return type && type.whitespace == "pre" ? OPT_PRESERVE_WS | OPT_PRESERVE_WS_FULL : base2 & ~OPT_OPEN_LEFT;
  }
  var NodeContext = class {
    constructor(type, attrs, marks, solid, match, options) {
      this.type = type;
      this.attrs = attrs;
      this.marks = marks;
      this.solid = solid;
      this.options = options;
      this.content = [];
      this.activeMarks = Mark.none;
      this.match = match || (options & OPT_OPEN_LEFT ? null : type.contentMatch);
    }
    findWrapping(node2) {
      if (!this.match) {
        if (!this.type)
          return [];
        let fill = this.type.contentMatch.fillBefore(Fragment.from(node2));
        if (fill) {
          this.match = this.type.contentMatch.matchFragment(fill);
        } else {
          let start = this.type.contentMatch, wrap3;
          if (wrap3 = start.findWrapping(node2.type)) {
            this.match = start;
            return wrap3;
          } else {
            return null;
          }
        }
      }
      return this.match.findWrapping(node2.type);
    }
    finish(openEnd) {
      if (!(this.options & OPT_PRESERVE_WS)) {
        let last = this.content[this.content.length - 1], m;
        if (last && last.isText && (m = /[ \t\r\n\u000c]+$/.exec(last.text))) {
          let text4 = last;
          if (last.text.length == m[0].length)
            this.content.pop();
          else
            this.content[this.content.length - 1] = text4.withText(text4.text.slice(0, text4.text.length - m[0].length));
        }
      }
      let content3 = Fragment.from(this.content);
      if (!openEnd && this.match)
        content3 = content3.append(this.match.fillBefore(Fragment.empty, true));
      return this.type ? this.type.create(this.attrs, content3, this.marks) : content3;
    }
    inlineContext(node2) {
      if (this.type)
        return this.type.inlineContent;
      if (this.content.length)
        return this.content[0].isInline;
      return node2.parentNode && !blockTags.hasOwnProperty(node2.parentNode.nodeName.toLowerCase());
    }
  };
  var ParseContext = class {
    constructor(parser2, options, isOpen) {
      this.parser = parser2;
      this.options = options;
      this.isOpen = isOpen;
      this.open = 0;
      this.localPreserveWS = false;
      let topNode = options.topNode, topContext;
      let topOptions = wsOptionsFor(null, options.preserveWhitespace, 0) | (isOpen ? OPT_OPEN_LEFT : 0);
      if (topNode)
        topContext = new NodeContext(topNode.type, topNode.attrs, Mark.none, true, options.topMatch || topNode.type.contentMatch, topOptions);
      else if (isOpen)
        topContext = new NodeContext(null, null, Mark.none, true, null, topOptions);
      else
        topContext = new NodeContext(parser2.schema.topNodeType, null, Mark.none, true, null, topOptions);
      this.nodes = [topContext];
      this.find = options.findPositions;
      this.needsBlock = false;
    }
    get top() {
      return this.nodes[this.open];
    }
    // Add a DOM node to the content. Text is inserted as text node,
    // otherwise, the node is passed to `addElement` or, if it has a
    // `style` attribute, `addElementWithStyles`.
    addDOM(dom, marks) {
      if (dom.nodeType == 3)
        this.addTextNode(dom, marks);
      else if (dom.nodeType == 1)
        this.addElement(dom, marks);
    }
    addTextNode(dom, marks) {
      let value = dom.nodeValue;
      let top = this.top, preserveWS = top.options & OPT_PRESERVE_WS_FULL ? "full" : this.localPreserveWS || (top.options & OPT_PRESERVE_WS) > 0;
      let { schema: schema3 } = this.parser;
      if (preserveWS === "full" || top.inlineContext(dom) || /[^ \t\r\n\u000c]/.test(value)) {
        if (!preserveWS) {
          value = value.replace(/[ \t\r\n\u000c]+/g, " ");
          if (/^[ \t\r\n\u000c]/.test(value) && this.open == this.nodes.length - 1) {
            let nodeBefore = top.content[top.content.length - 1];
            let domNodeBefore = dom.previousSibling;
            if (!nodeBefore || domNodeBefore && domNodeBefore.nodeName == "BR" || nodeBefore.isText && /[ \t\r\n\u000c]$/.test(nodeBefore.text))
              value = value.slice(1);
          }
        } else if (preserveWS === "full") {
          value = value.replace(/\r\n?/g, "\n");
        } else if (schema3.linebreakReplacement && /[\r\n]/.test(value) && this.top.findWrapping(schema3.linebreakReplacement.create())) {
          let lines = value.split(/\r?\n|\r/);
          for (let i2 = 0; i2 < lines.length; i2++) {
            if (i2)
              this.insertNode(schema3.linebreakReplacement.create(), marks, true);
            if (lines[i2])
              this.insertNode(schema3.text(lines[i2]), marks, !/\S/.test(lines[i2]));
          }
          value = "";
        } else {
          value = value.replace(/\r?\n|\r/g, " ");
        }
        if (value)
          this.insertNode(schema3.text(value), marks, !/\S/.test(value));
        this.findInText(dom);
      } else {
        this.findInside(dom);
      }
    }
    // Try to find a handler for the given tag and use that to parse. If
    // none is found, the element's content nodes are added directly.
    addElement(dom, marks, matchAfter) {
      let outerWS = this.localPreserveWS, top = this.top;
      if (dom.tagName == "PRE" || /pre/.test(dom.style && dom.style.whiteSpace))
        this.localPreserveWS = true;
      let name = dom.nodeName.toLowerCase(), ruleID;
      if (listTags.hasOwnProperty(name) && this.parser.normalizeLists)
        normalizeList(dom);
      let rule = this.options.ruleFromNode && this.options.ruleFromNode(dom) || (ruleID = this.parser.matchTag(dom, this, matchAfter));
      out: if (rule ? rule.ignore : ignoreTags.hasOwnProperty(name)) {
        this.findInside(dom);
        this.ignoreFallback(dom, marks);
      } else if (!rule || rule.skip || rule.closeParent) {
        if (rule && rule.closeParent)
          this.open = Math.max(0, this.open - 1);
        else if (rule && rule.skip.nodeType)
          dom = rule.skip;
        let sync, oldNeedsBlock = this.needsBlock;
        if (blockTags.hasOwnProperty(name)) {
          if (top.content.length && top.content[0].isInline && this.open) {
            this.open--;
            top = this.top;
          }
          sync = true;
          if (!top.type)
            this.needsBlock = true;
        } else if (!dom.firstChild) {
          this.leafFallback(dom, marks);
          break out;
        }
        let innerMarks = rule && rule.skip ? marks : this.readStyles(dom, marks);
        if (innerMarks)
          this.addAll(dom, innerMarks);
        if (sync)
          this.sync(top);
        this.needsBlock = oldNeedsBlock;
      } else {
        let innerMarks = this.readStyles(dom, marks);
        if (innerMarks)
          this.addElementByRule(dom, rule, innerMarks, rule.consuming === false ? ruleID : void 0);
      }
      this.localPreserveWS = outerWS;
    }
    // Called for leaf DOM nodes that would otherwise be ignored
    leafFallback(dom, marks) {
      if (dom.nodeName == "BR" && this.top.type && this.top.type.inlineContent)
        this.addTextNode(dom.ownerDocument.createTextNode("\n"), marks);
    }
    // Called for ignored nodes
    ignoreFallback(dom, marks) {
      if (dom.nodeName == "BR" && (!this.top.type || !this.top.type.inlineContent))
        this.findPlace(this.parser.schema.text("-"), marks, true);
    }
    // Run any style parser associated with the node's styles. Either
    // return an updated array of marks, or null to indicate some of the
    // styles had a rule with `ignore` set.
    readStyles(dom, marks) {
      let styles = dom.style;
      if (styles && styles.length)
        for (let i2 = 0; i2 < this.parser.matchedStyles.length; i2++) {
          let name = this.parser.matchedStyles[i2], value = styles.getPropertyValue(name);
          if (value)
            for (let after = void 0; ; ) {
              let rule = this.parser.matchStyle(name, value, this, after);
              if (!rule)
                break;
              if (rule.ignore)
                return null;
              if (rule.clearMark)
                marks = marks.filter((m) => !rule.clearMark(m));
              else
                marks = marks.concat(this.parser.schema.marks[rule.mark].create(rule.attrs));
              if (rule.consuming === false)
                after = rule;
              else
                break;
            }
        }
      return marks;
    }
    // Look up a handler for the given node. If none are found, return
    // false. Otherwise, apply it, use its return value to drive the way
    // the node's content is wrapped, and return true.
    addElementByRule(dom, rule, marks, continueAfter) {
      let sync, nodeType;
      if (rule.node) {
        nodeType = this.parser.schema.nodes[rule.node];
        if (!nodeType.isLeaf) {
          let inner = this.enter(nodeType, rule.attrs || null, marks, rule.preserveWhitespace);
          if (inner) {
            sync = true;
            marks = inner;
          }
        } else if (!this.insertNode(nodeType.create(rule.attrs), marks, dom.nodeName == "BR")) {
          this.leafFallback(dom, marks);
        }
      } else {
        let markType = this.parser.schema.marks[rule.mark];
        marks = marks.concat(markType.create(rule.attrs));
      }
      let startIn = this.top;
      if (nodeType && nodeType.isLeaf) {
        this.findInside(dom);
      } else if (continueAfter) {
        this.addElement(dom, marks, continueAfter);
      } else if (rule.getContent) {
        this.findInside(dom);
        rule.getContent(dom, this.parser.schema).forEach((node2) => this.insertNode(node2, marks, false));
      } else {
        let contentDOM = dom;
        if (typeof rule.contentElement == "string")
          contentDOM = dom.querySelector(rule.contentElement);
        else if (typeof rule.contentElement == "function")
          contentDOM = rule.contentElement(dom);
        else if (rule.contentElement)
          contentDOM = rule.contentElement;
        this.findAround(dom, contentDOM, true);
        this.addAll(contentDOM, marks);
        this.findAround(dom, contentDOM, false);
      }
      if (sync && this.sync(startIn))
        this.open--;
    }
    // Add all child nodes between `startIndex` and `endIndex` (or the
    // whole node, if not given). If `sync` is passed, use it to
    // synchronize after every block element.
    addAll(parent, marks, startIndex, endIndex) {
      let index2 = startIndex || 0;
      for (let dom = startIndex ? parent.childNodes[startIndex] : parent.firstChild, end = endIndex == null ? null : parent.childNodes[endIndex]; dom != end; dom = dom.nextSibling, ++index2) {
        this.findAtPoint(parent, index2);
        this.addDOM(dom, marks);
      }
      this.findAtPoint(parent, index2);
    }
    // Try to find a way to fit the given node type into the current
    // context. May add intermediate wrappers and/or leave non-solid
    // nodes that we're in.
    findPlace(node2, marks, cautious) {
      let route, sync;
      for (let depth = this.open, penalty = 0; depth >= 0; depth--) {
        let cx = this.nodes[depth];
        let found2 = cx.findWrapping(node2);
        if (found2 && (!route || route.length > found2.length + penalty)) {
          route = found2;
          sync = cx;
          if (!found2.length)
            break;
        }
        if (cx.solid) {
          if (cautious)
            break;
          penalty += 2;
        }
      }
      if (!route)
        return null;
      this.sync(sync);
      for (let i2 = 0; i2 < route.length; i2++)
        marks = this.enterInner(route[i2], null, marks, false);
      return marks;
    }
    // Try to insert the given node, adjusting the context when needed.
    insertNode(node2, marks, cautious) {
      if (node2.isInline && this.needsBlock && !this.top.type) {
        let block = this.textblockFromContext();
        if (block)
          marks = this.enterInner(block, null, marks);
      }
      let innerMarks = this.findPlace(node2, marks, cautious);
      if (innerMarks) {
        this.closeExtra();
        let top = this.top;
        if (top.match)
          top.match = top.match.matchType(node2.type);
        let nodeMarks = Mark.none;
        for (let m of innerMarks.concat(node2.marks))
          if (top.type ? top.type.allowsMarkType(m.type) : markMayApply(m.type, node2.type))
            nodeMarks = m.addToSet(nodeMarks);
        top.content.push(node2.mark(nodeMarks));
        return true;
      }
      return false;
    }
    // Try to start a node of the given type, adjusting the context when
    // necessary.
    enter(type, attrs, marks, preserveWS) {
      let innerMarks = this.findPlace(type.create(attrs), marks, false);
      if (innerMarks)
        innerMarks = this.enterInner(type, attrs, marks, true, preserveWS);
      return innerMarks;
    }
    // Open a node of the given type
    enterInner(type, attrs, marks, solid = false, preserveWS) {
      this.closeExtra();
      let top = this.top;
      top.match = top.match && top.match.matchType(type);
      let options = wsOptionsFor(type, preserveWS, top.options);
      if (top.options & OPT_OPEN_LEFT && top.content.length == 0)
        options |= OPT_OPEN_LEFT;
      let applyMarks = Mark.none;
      marks = marks.filter((m) => {
        if (top.type ? top.type.allowsMarkType(m.type) : markMayApply(m.type, type)) {
          applyMarks = m.addToSet(applyMarks);
          return false;
        }
        return true;
      });
      this.nodes.push(new NodeContext(type, attrs, applyMarks, solid, null, options));
      this.open++;
      return marks;
    }
    // Make sure all nodes above this.open are finished and added to
    // their parents
    closeExtra(openEnd = false) {
      let i2 = this.nodes.length - 1;
      if (i2 > this.open) {
        for (; i2 > this.open; i2--)
          this.nodes[i2 - 1].content.push(this.nodes[i2].finish(openEnd));
        this.nodes.length = this.open + 1;
      }
    }
    finish() {
      this.open = 0;
      this.closeExtra(this.isOpen);
      return this.nodes[0].finish(!!(this.isOpen || this.options.topOpen));
    }
    sync(to) {
      for (let i2 = this.open; i2 >= 0; i2--) {
        if (this.nodes[i2] == to) {
          this.open = i2;
          return true;
        } else if (this.localPreserveWS) {
          this.nodes[i2].options |= OPT_PRESERVE_WS;
        }
      }
      return false;
    }
    get currentPos() {
      this.closeExtra();
      let pos = 0;
      for (let i2 = this.open; i2 >= 0; i2--) {
        let content3 = this.nodes[i2].content;
        for (let j = content3.length - 1; j >= 0; j--)
          pos += content3[j].nodeSize;
        if (i2)
          pos++;
      }
      return pos;
    }
    findAtPoint(parent, offset) {
      if (this.find)
        for (let i2 = 0; i2 < this.find.length; i2++) {
          if (this.find[i2].node == parent && this.find[i2].offset == offset)
            this.find[i2].pos = this.currentPos;
        }
    }
    findInside(parent) {
      if (this.find)
        for (let i2 = 0; i2 < this.find.length; i2++) {
          if (this.find[i2].pos == null && parent.nodeType == 1 && parent.contains(this.find[i2].node))
            this.find[i2].pos = this.currentPos;
        }
    }
    findAround(parent, content3, before) {
      if (parent != content3 && this.find)
        for (let i2 = 0; i2 < this.find.length; i2++) {
          if (this.find[i2].pos == null && parent.nodeType == 1 && parent.contains(this.find[i2].node)) {
            let pos = content3.compareDocumentPosition(this.find[i2].node);
            if (pos & (before ? 2 : 4))
              this.find[i2].pos = this.currentPos;
          }
        }
    }
    findInText(textNode) {
      if (this.find)
        for (let i2 = 0; i2 < this.find.length; i2++) {
          if (this.find[i2].node == textNode)
            this.find[i2].pos = this.currentPos - (textNode.nodeValue.length - this.find[i2].offset);
        }
    }
    // Determines whether the given context string matches this context.
    matchesContext(context) {
      if (context.indexOf("|") > -1)
        return context.split(/\s*\|\s*/).some(this.matchesContext, this);
      let parts = context.split("/");
      let option = this.options.context;
      let useRoot = !this.isOpen && (!option || option.parent.type == this.nodes[0].type);
      let minDepth = -(option ? option.depth + 1 : 0) + (useRoot ? 0 : 1);
      let match = (i2, depth) => {
        for (; i2 >= 0; i2--) {
          let part = parts[i2];
          if (part == "") {
            if (i2 == parts.length - 1 || i2 == 0)
              continue;
            for (; depth >= minDepth; depth--)
              if (match(i2 - 1, depth))
                return true;
            return false;
          } else {
            let next = depth > 0 || depth == 0 && useRoot ? this.nodes[depth].type : option && depth >= minDepth ? option.node(depth - minDepth).type : null;
            if (!next || next.name != part && !next.isInGroup(part))
              return false;
            depth--;
          }
        }
        return true;
      };
      return match(parts.length - 1, this.open);
    }
    textblockFromContext() {
      let $context = this.options.context;
      if ($context)
        for (let d2 = $context.depth; d2 >= 0; d2--) {
          let deflt = $context.node(d2).contentMatchAt($context.indexAfter(d2)).defaultType;
          if (deflt && deflt.isTextblock && deflt.defaultAttrs)
            return deflt;
        }
      for (let name in this.parser.schema.nodes) {
        let type = this.parser.schema.nodes[name];
        if (type.isTextblock && type.defaultAttrs)
          return type;
      }
    }
  };
  function normalizeList(dom) {
    for (let child = dom.firstChild, prevItem = null; child; child = child.nextSibling) {
      let name = child.nodeType == 1 ? child.nodeName.toLowerCase() : null;
      if (name && listTags.hasOwnProperty(name) && prevItem) {
        prevItem.appendChild(child);
        child = prevItem;
      } else if (name == "li") {
        prevItem = child;
      } else if (name) {
        prevItem = null;
      }
    }
  }
  function matches(dom, selector) {
    return (dom.matches || dom.msMatchesSelector || dom.webkitMatchesSelector || dom.mozMatchesSelector).call(dom, selector);
  }
  function copy(obj) {
    let copy2 = {};
    for (let prop in obj)
      copy2[prop] = obj[prop];
    return copy2;
  }
  function markMayApply(markType, nodeType) {
    let nodes = nodeType.schema.nodes;
    for (let name in nodes) {
      let parent = nodes[name];
      if (!parent.allowsMarkType(markType))
        continue;
      let seen = [], scan = (match) => {
        seen.push(match);
        for (let i2 = 0; i2 < match.edgeCount; i2++) {
          let { type, next } = match.edge(i2);
          if (type == nodeType)
            return true;
          if (seen.indexOf(next) < 0 && scan(next))
            return true;
        }
      };
      if (scan(parent.contentMatch))
        return true;
    }
  }
  var DOMSerializer = class _DOMSerializer {
    /**
    Create a serializer. `nodes` should map node names to functions
    that take a node and return a description of the corresponding
    DOM. `marks` does the same for mark names, but also gets an
    argument that tells it whether the mark's content is block or
    inline content (for typical use, it'll always be inline). A mark
    serializer may be `null` to indicate that marks of that type
    should not be serialized.
    */
    constructor(nodes, marks) {
      this.nodes = nodes;
      this.marks = marks;
    }
    /**
    Serialize the content of this fragment to a DOM fragment. When
    not in the browser, the `document` option, containing a DOM
    document, should be passed so that the serializer can create
    nodes.
    */
    serializeFragment(fragment, options = {}, target) {
      if (!target)
        target = doc(options).createDocumentFragment();
      let top = target, active = [];
      fragment.forEach((node2) => {
        if (active.length || node2.marks.length) {
          let keep = 0, rendered = 0;
          while (keep < active.length && rendered < node2.marks.length) {
            let next = node2.marks[rendered];
            if (!this.marks[next.type.name]) {
              rendered++;
              continue;
            }
            if (!next.eq(active[keep][0]) || next.type.spec.spanning === false)
              break;
            keep++;
            rendered++;
          }
          while (keep < active.length)
            top = active.pop()[1];
          while (rendered < node2.marks.length) {
            let add = node2.marks[rendered++];
            let markDOM = this.serializeMark(add, node2.isInline, options);
            if (markDOM) {
              active.push([add, top]);
              top.appendChild(markDOM.dom);
              top = markDOM.contentDOM || markDOM.dom;
            }
          }
        }
        top.appendChild(this.serializeNodeInner(node2, options));
      });
      return target;
    }
    /**
    @internal
    */
    serializeNodeInner(node2, options) {
      let { dom, contentDOM } = renderSpec(doc(options), this.nodes[node2.type.name](node2), null, node2.attrs);
      if (contentDOM) {
        if (node2.isLeaf)
          throw new RangeError("Content hole not allowed in a leaf node spec");
        this.serializeFragment(node2.content, options, contentDOM);
      }
      return dom;
    }
    /**
    Serialize this node to a DOM node. This can be useful when you
    need to serialize a part of a document, as opposed to the whole
    document. To serialize a whole document, use
    [`serializeFragment`](https://prosemirror.net/docs/ref/#model.DOMSerializer.serializeFragment) on
    its [content](https://prosemirror.net/docs/ref/#model.Node.content).
    */
    serializeNode(node2, options = {}) {
      let dom = this.serializeNodeInner(node2, options);
      for (let i2 = node2.marks.length - 1; i2 >= 0; i2--) {
        let wrap3 = this.serializeMark(node2.marks[i2], node2.isInline, options);
        if (wrap3) {
          (wrap3.contentDOM || wrap3.dom).appendChild(dom);
          dom = wrap3.dom;
        }
      }
      return dom;
    }
    /**
    @internal
    */
    serializeMark(mark, inline, options = {}) {
      let toDOM = this.marks[mark.type.name];
      return toDOM && renderSpec(doc(options), toDOM(mark, inline), null, mark.attrs);
    }
    static renderSpec(doc4, structure, xmlNS = null, blockArraysIn) {
      return renderSpec(doc4, structure, xmlNS, blockArraysIn);
    }
    /**
    Build a serializer using the [`toDOM`](https://prosemirror.net/docs/ref/#model.NodeSpec.toDOM)
    properties in a schema's node and mark specs.
    */
    static fromSchema(schema3) {
      return schema3.cached.domSerializer || (schema3.cached.domSerializer = new _DOMSerializer(this.nodesFromSchema(schema3), this.marksFromSchema(schema3)));
    }
    /**
    Gather the serializers in a schema's node specs into an object.
    This can be useful as a base to build a custom serializer from.
    */
    static nodesFromSchema(schema3) {
      let result = gatherToDOM(schema3.nodes);
      if (!result.text)
        result.text = (node2) => node2.text;
      return result;
    }
    /**
    Gather the serializers in a schema's mark specs into an object.
    */
    static marksFromSchema(schema3) {
      return gatherToDOM(schema3.marks);
    }
  };
  function gatherToDOM(obj) {
    let result = {};
    for (let name in obj) {
      let toDOM = obj[name].spec.toDOM;
      if (toDOM)
        result[name] = toDOM;
    }
    return result;
  }
  function doc(options) {
    return options.document || window.document;
  }
  var suspiciousAttributeCache = /* @__PURE__ */ new WeakMap();
  function suspiciousAttributes(attrs) {
    let value = suspiciousAttributeCache.get(attrs);
    if (value === void 0)
      suspiciousAttributeCache.set(attrs, value = suspiciousAttributesInner(attrs));
    return value;
  }
  function suspiciousAttributesInner(attrs) {
    let result = null;
    function scan(value) {
      if (value && typeof value == "object") {
        if (Array.isArray(value)) {
          if (typeof value[0] == "string") {
            if (!result)
              result = [];
            result.push(value);
          } else {
            for (let i2 = 0; i2 < value.length; i2++)
              scan(value[i2]);
          }
        } else {
          for (let prop in value)
            scan(value[prop]);
        }
      }
    }
    scan(attrs);
    return result;
  }
  function renderSpec(doc4, structure, xmlNS, blockArraysIn) {
    if (typeof structure == "string")
      return { dom: doc4.createTextNode(structure) };
    if (structure.nodeType != null)
      return { dom: structure };
    if (structure.dom && structure.dom.nodeType != null)
      return structure;
    let tagName = structure[0], suspicious;
    if (typeof tagName != "string")
      throw new RangeError("Invalid array passed to renderSpec");
    if (blockArraysIn && (suspicious = suspiciousAttributes(blockArraysIn)) && suspicious.indexOf(structure) > -1)
      throw new RangeError("Using an array from an attribute object as a DOM spec. This may be an attempted cross site scripting attack.");
    let space = tagName.indexOf(" ");
    if (space > 0) {
      xmlNS = tagName.slice(0, space);
      tagName = tagName.slice(space + 1);
    }
    let contentDOM;
    let dom = xmlNS ? doc4.createElementNS(xmlNS, tagName) : doc4.createElement(tagName);
    let attrs = structure[1], start = 1;
    if (attrs && typeof attrs == "object" && attrs.nodeType == null && !Array.isArray(attrs)) {
      start = 2;
      for (let name in attrs)
        if (attrs[name] != null) {
          let space2 = name.indexOf(" ");
          if (space2 > 0)
            dom.setAttributeNS(name.slice(0, space2), name.slice(space2 + 1), attrs[name]);
          else if (name == "style" && dom.style)
            dom.style.cssText = attrs[name];
          else
            dom.setAttribute(name, attrs[name]);
        }
    }
    for (let i2 = start; i2 < structure.length; i2++) {
      let child = structure[i2];
      if (child === 0) {
        if (i2 < structure.length - 1 || i2 > start)
          throw new RangeError("Content hole must be the only child of its parent node");
        return { dom, contentDOM: dom };
      } else {
        let { dom: inner, contentDOM: innerContent } = renderSpec(doc4, child, xmlNS, blockArraysIn);
        dom.appendChild(inner);
        if (innerContent) {
          if (contentDOM)
            throw new RangeError("Multiple content holes");
          contentDOM = innerContent;
        }
      }
    }
    return { dom, contentDOM };
  }

  // node_modules/mdast-util-to-string/lib/index.js
  var emptyOptions = {};
  function toString(value, options) {
    const settings = options || emptyOptions;
    const includeImageAlt = typeof settings.includeImageAlt === "boolean" ? settings.includeImageAlt : true;
    const includeHtml = typeof settings.includeHtml === "boolean" ? settings.includeHtml : true;
    return one(value, includeImageAlt, includeHtml);
  }
  function one(value, includeImageAlt, includeHtml) {
    if (node(value)) {
      if ("value" in value) {
        return value.type === "html" && !includeHtml ? "" : value.value;
      }
      if (includeImageAlt && "alt" in value && value.alt) {
        return value.alt;
      }
      if ("children" in value) {
        return all(value.children, includeImageAlt, includeHtml);
      }
    }
    if (Array.isArray(value)) {
      return all(value, includeImageAlt, includeHtml);
    }
    return "";
  }
  function all(values, includeImageAlt, includeHtml) {
    const result = [];
    let index2 = -1;
    while (++index2 < values.length) {
      result[index2] = one(values[index2], includeImageAlt, includeHtml);
    }
    return result.join("");
  }
  function node(value) {
    return Boolean(value && typeof value === "object");
  }

  // node_modules/decode-named-character-reference/index.dom.js
  var element = document.createElement("i");
  function decodeNamedCharacterReference(value) {
    const characterReference2 = "&" + value + ";";
    element.innerHTML = characterReference2;
    const character = element.textContent;
    if (character.charCodeAt(character.length - 1) === 59 && value !== "semi") {
      return false;
    }
    return character === characterReference2 ? false : character;
  }

  // node_modules/micromark-util-chunked/index.js
  function splice(list4, start, remove, items) {
    const end = list4.length;
    let chunkStart = 0;
    let parameters;
    if (start < 0) {
      start = -start > end ? 0 : end + start;
    } else {
      start = start > end ? end : start;
    }
    remove = remove > 0 ? remove : 0;
    if (items.length < 1e4) {
      parameters = Array.from(items);
      parameters.unshift(start, remove);
      list4.splice(...parameters);
    } else {
      if (remove) list4.splice(start, remove);
      while (chunkStart < items.length) {
        parameters = items.slice(chunkStart, chunkStart + 1e4);
        parameters.unshift(start, 0);
        list4.splice(...parameters);
        chunkStart += 1e4;
        start += 1e4;
      }
    }
  }
  function push(list4, items) {
    if (list4.length > 0) {
      splice(list4, list4.length, 0, items);
      return list4;
    }
    return items;
  }

  // node_modules/micromark-util-combine-extensions/index.js
  var hasOwnProperty = {}.hasOwnProperty;
  function combineExtensions(extensions) {
    const all2 = {};
    let index2 = -1;
    while (++index2 < extensions.length) {
      syntaxExtension(all2, extensions[index2]);
    }
    return all2;
  }
  function syntaxExtension(all2, extension2) {
    let hook;
    for (hook in extension2) {
      const maybe = hasOwnProperty.call(all2, hook) ? all2[hook] : void 0;
      const left = maybe || (all2[hook] = {});
      const right = extension2[hook];
      let code2;
      if (right) {
        for (code2 in right) {
          if (!hasOwnProperty.call(left, code2)) left[code2] = [];
          const value = right[code2];
          constructs(
            // @ts-expect-error Looks like a list.
            left[code2],
            Array.isArray(value) ? value : value ? [value] : []
          );
        }
      }
    }
  }
  function constructs(existing, list4) {
    let index2 = -1;
    const before = [];
    while (++index2 < list4.length) {
      ;
      (list4[index2].add === "after" ? existing : before).push(list4[index2]);
    }
    splice(existing, 0, 0, before);
  }

  // node_modules/micromark-util-decode-numeric-character-reference/index.js
  function decodeNumericCharacterReference(value, base2) {
    const code2 = Number.parseInt(value, base2);
    if (
      // C0 except for HT, LF, FF, CR, space.
      code2 < 9 || code2 === 11 || code2 > 13 && code2 < 32 || // Control character (DEL) of C0, and C1 controls.
      code2 > 126 && code2 < 160 || // Lone high surrogates and low surrogates.
      code2 > 55295 && code2 < 57344 || // Noncharacters.
      code2 > 64975 && code2 < 65008 || /* eslint-disable no-bitwise */
      (code2 & 65535) === 65535 || (code2 & 65535) === 65534 || /* eslint-enable no-bitwise */
      // Out of range
      code2 > 1114111
    ) {
      return "\uFFFD";
    }
    return String.fromCodePoint(code2);
  }

  // node_modules/micromark-util-normalize-identifier/index.js
  function normalizeIdentifier(value) {
    return value.replace(/[\t\n\r ]+/g, " ").replace(/^ | $/g, "").toLowerCase().toUpperCase();
  }

  // node_modules/micromark-util-character/index.js
  var asciiAlpha = regexCheck(/[A-Za-z]/);
  var asciiAlphanumeric = regexCheck(/[\dA-Za-z]/);
  var asciiAtext = regexCheck(/[#-'*+\--9=?A-Z^-~]/);
  function asciiControl(code2) {
    return (
      // Special whitespace codes (which have negative values), C0 and Control
      // character DEL
      code2 !== null && (code2 < 32 || code2 === 127)
    );
  }
  var asciiDigit = regexCheck(/\d/);
  var asciiHexDigit = regexCheck(/[\dA-Fa-f]/);
  var asciiPunctuation = regexCheck(/[!-/:-@[-`{-~]/);
  function markdownLineEnding(code2) {
    return code2 !== null && code2 < -2;
  }
  function markdownLineEndingOrSpace(code2) {
    return code2 !== null && (code2 < 0 || code2 === 32);
  }
  function markdownSpace(code2) {
    return code2 === -2 || code2 === -1 || code2 === 32;
  }
  var unicodePunctuation = regexCheck(/\p{P}|\p{S}/u);
  var unicodeWhitespace = regexCheck(/\s/);
  function regexCheck(regex) {
    return check;
    function check(code2) {
      return code2 !== null && code2 > -1 && regex.test(String.fromCharCode(code2));
    }
  }

  // node_modules/micromark-factory-space/index.js
  function factorySpace(effects, ok3, type, max) {
    const limit = max ? max - 1 : Number.POSITIVE_INFINITY;
    let size = 0;
    return start;
    function start(code2) {
      if (markdownSpace(code2)) {
        effects.enter(type);
        return prefix(code2);
      }
      return ok3(code2);
    }
    function prefix(code2) {
      if (markdownSpace(code2) && size++ < limit) {
        effects.consume(code2);
        return prefix;
      }
      effects.exit(type);
      return ok3(code2);
    }
  }

  // node_modules/micromark/lib/initialize/content.js
  var content = {
    tokenize: initializeContent
  };
  function initializeContent(effects) {
    const contentStart = effects.attempt(this.parser.constructs.contentInitial, afterContentStartConstruct, paragraphInitial);
    let previous2;
    return contentStart;
    function afterContentStartConstruct(code2) {
      if (code2 === null) {
        effects.consume(code2);
        return;
      }
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return factorySpace(effects, contentStart, "linePrefix");
    }
    function paragraphInitial(code2) {
      effects.enter("paragraph");
      return lineStart(code2);
    }
    function lineStart(code2) {
      const token = effects.enter("chunkText", {
        contentType: "text",
        previous: previous2
      });
      if (previous2) {
        previous2.next = token;
      }
      previous2 = token;
      return data(code2);
    }
    function data(code2) {
      if (code2 === null) {
        effects.exit("chunkText");
        effects.exit("paragraph");
        effects.consume(code2);
        return;
      }
      if (markdownLineEnding(code2)) {
        effects.consume(code2);
        effects.exit("chunkText");
        return lineStart;
      }
      effects.consume(code2);
      return data;
    }
  }

  // node_modules/micromark/lib/initialize/document.js
  var document2 = {
    tokenize: initializeDocument
  };
  var containerConstruct = {
    tokenize: tokenizeContainer
  };
  function initializeDocument(effects) {
    const self2 = this;
    const stack = [];
    let continued = 0;
    let childFlow;
    let childToken;
    let lineStartOffset;
    return start;
    function start(code2) {
      if (continued < stack.length) {
        const item = stack[continued];
        self2.containerState = item[1];
        return effects.attempt(item[0].continuation, documentContinue, checkNewContainers)(code2);
      }
      return checkNewContainers(code2);
    }
    function documentContinue(code2) {
      continued++;
      if (self2.containerState._closeFlow) {
        self2.containerState._closeFlow = void 0;
        if (childFlow) {
          closeFlow();
        }
        const indexBeforeExits = self2.events.length;
        let indexBeforeFlow = indexBeforeExits;
        let point3;
        while (indexBeforeFlow--) {
          if (self2.events[indexBeforeFlow][0] === "exit" && self2.events[indexBeforeFlow][1].type === "chunkFlow") {
            point3 = self2.events[indexBeforeFlow][1].end;
            break;
          }
        }
        exitContainers(continued);
        let index2 = indexBeforeExits;
        while (index2 < self2.events.length) {
          self2.events[index2][1].end = {
            ...point3
          };
          index2++;
        }
        splice(self2.events, indexBeforeFlow + 1, 0, self2.events.slice(indexBeforeExits));
        self2.events.length = index2;
        return checkNewContainers(code2);
      }
      return start(code2);
    }
    function checkNewContainers(code2) {
      if (continued === stack.length) {
        if (!childFlow) {
          return documentContinued(code2);
        }
        if (childFlow.currentConstruct && childFlow.currentConstruct.concrete) {
          return flowStart(code2);
        }
        self2.interrupt = Boolean(childFlow.currentConstruct && !childFlow._gfmTableDynamicInterruptHack);
      }
      self2.containerState = {};
      return effects.check(containerConstruct, thereIsANewContainer, thereIsNoNewContainer)(code2);
    }
    function thereIsANewContainer(code2) {
      if (childFlow) closeFlow();
      exitContainers(continued);
      return documentContinued(code2);
    }
    function thereIsNoNewContainer(code2) {
      self2.parser.lazy[self2.now().line] = continued !== stack.length;
      lineStartOffset = self2.now().offset;
      return flowStart(code2);
    }
    function documentContinued(code2) {
      self2.containerState = {};
      return effects.attempt(containerConstruct, containerContinue, flowStart)(code2);
    }
    function containerContinue(code2) {
      continued++;
      stack.push([self2.currentConstruct, self2.containerState]);
      return documentContinued(code2);
    }
    function flowStart(code2) {
      if (code2 === null) {
        if (childFlow) closeFlow();
        exitContainers(0);
        effects.consume(code2);
        return;
      }
      childFlow = childFlow || self2.parser.flow(self2.now());
      effects.enter("chunkFlow", {
        _tokenizer: childFlow,
        contentType: "flow",
        previous: childToken
      });
      return flowContinue(code2);
    }
    function flowContinue(code2) {
      if (code2 === null) {
        writeToChild(effects.exit("chunkFlow"), true);
        exitContainers(0);
        effects.consume(code2);
        return;
      }
      if (markdownLineEnding(code2)) {
        effects.consume(code2);
        writeToChild(effects.exit("chunkFlow"));
        continued = 0;
        self2.interrupt = void 0;
        return start;
      }
      effects.consume(code2);
      return flowContinue;
    }
    function writeToChild(token, endOfFile) {
      const stream = self2.sliceStream(token);
      if (endOfFile) stream.push(null);
      token.previous = childToken;
      if (childToken) childToken.next = token;
      childToken = token;
      childFlow.defineSkip(token.start);
      childFlow.write(stream);
      if (self2.parser.lazy[token.start.line]) {
        let index2 = childFlow.events.length;
        while (index2--) {
          if (
            // The token starts before the line ending…
            childFlow.events[index2][1].start.offset < lineStartOffset && // …and either is not ended yet…
            (!childFlow.events[index2][1].end || // …or ends after it.
            childFlow.events[index2][1].end.offset > lineStartOffset)
          ) {
            return;
          }
        }
        const indexBeforeExits = self2.events.length;
        let indexBeforeFlow = indexBeforeExits;
        let seen;
        let point3;
        while (indexBeforeFlow--) {
          if (self2.events[indexBeforeFlow][0] === "exit" && self2.events[indexBeforeFlow][1].type === "chunkFlow") {
            if (seen) {
              point3 = self2.events[indexBeforeFlow][1].end;
              break;
            }
            seen = true;
          }
        }
        exitContainers(continued);
        index2 = indexBeforeExits;
        while (index2 < self2.events.length) {
          self2.events[index2][1].end = {
            ...point3
          };
          index2++;
        }
        splice(self2.events, indexBeforeFlow + 1, 0, self2.events.slice(indexBeforeExits));
        self2.events.length = index2;
      }
    }
    function exitContainers(size) {
      let index2 = stack.length;
      while (index2-- > size) {
        const entry = stack[index2];
        self2.containerState = entry[1];
        entry[0].exit.call(self2, effects);
      }
      stack.length = size;
    }
    function closeFlow() {
      childFlow.write([null]);
      childToken = void 0;
      childFlow = void 0;
      self2.containerState._closeFlow = void 0;
    }
  }
  function tokenizeContainer(effects, ok3, nok) {
    return factorySpace(effects, effects.attempt(this.parser.constructs.document, ok3, nok), "linePrefix", this.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4);
  }

  // node_modules/micromark-util-classify-character/index.js
  function classifyCharacter(code2) {
    if (code2 === null || markdownLineEndingOrSpace(code2) || unicodeWhitespace(code2)) {
      return 1;
    }
    if (unicodePunctuation(code2)) {
      return 2;
    }
  }

  // node_modules/micromark-util-resolve-all/index.js
  function resolveAll(constructs2, events, context) {
    const called = [];
    let index2 = -1;
    while (++index2 < constructs2.length) {
      const resolve = constructs2[index2].resolveAll;
      if (resolve && !called.includes(resolve)) {
        events = resolve(events, context);
        called.push(resolve);
      }
    }
    return events;
  }

  // node_modules/micromark-core-commonmark/lib/attention.js
  var attention = {
    name: "attention",
    resolveAll: resolveAllAttention,
    tokenize: tokenizeAttention
  };
  function resolveAllAttention(events, context) {
    let index2 = -1;
    let open;
    let group;
    let text4;
    let openingSequence;
    let closingSequence;
    let use;
    let nextEvents;
    let offset;
    while (++index2 < events.length) {
      if (events[index2][0] === "enter" && events[index2][1].type === "attentionSequence" && events[index2][1]._close) {
        open = index2;
        while (open--) {
          if (events[open][0] === "exit" && events[open][1].type === "attentionSequence" && events[open][1]._open && // If the markers are the same:
          context.sliceSerialize(events[open][1]).charCodeAt(0) === context.sliceSerialize(events[index2][1]).charCodeAt(0)) {
            if ((events[open][1]._close || events[index2][1]._open) && (events[index2][1].end.offset - events[index2][1].start.offset) % 3 && !((events[open][1].end.offset - events[open][1].start.offset + events[index2][1].end.offset - events[index2][1].start.offset) % 3)) {
              continue;
            }
            use = events[open][1].end.offset - events[open][1].start.offset > 1 && events[index2][1].end.offset - events[index2][1].start.offset > 1 ? 2 : 1;
            const start = {
              ...events[open][1].end
            };
            const end = {
              ...events[index2][1].start
            };
            movePoint(start, -use);
            movePoint(end, use);
            openingSequence = {
              type: use > 1 ? "strongSequence" : "emphasisSequence",
              start,
              end: {
                ...events[open][1].end
              }
            };
            closingSequence = {
              type: use > 1 ? "strongSequence" : "emphasisSequence",
              start: {
                ...events[index2][1].start
              },
              end
            };
            text4 = {
              type: use > 1 ? "strongText" : "emphasisText",
              start: {
                ...events[open][1].end
              },
              end: {
                ...events[index2][1].start
              }
            };
            group = {
              type: use > 1 ? "strong" : "emphasis",
              start: {
                ...openingSequence.start
              },
              end: {
                ...closingSequence.end
              }
            };
            events[open][1].end = {
              ...openingSequence.start
            };
            events[index2][1].start = {
              ...closingSequence.end
            };
            nextEvents = [];
            if (events[open][1].end.offset - events[open][1].start.offset) {
              nextEvents = push(nextEvents, [["enter", events[open][1], context], ["exit", events[open][1], context]]);
            }
            nextEvents = push(nextEvents, [["enter", group, context], ["enter", openingSequence, context], ["exit", openingSequence, context], ["enter", text4, context]]);
            nextEvents = push(nextEvents, resolveAll(context.parser.constructs.insideSpan.null, events.slice(open + 1, index2), context));
            nextEvents = push(nextEvents, [["exit", text4, context], ["enter", closingSequence, context], ["exit", closingSequence, context], ["exit", group, context]]);
            if (events[index2][1].end.offset - events[index2][1].start.offset) {
              offset = 2;
              nextEvents = push(nextEvents, [["enter", events[index2][1], context], ["exit", events[index2][1], context]]);
            } else {
              offset = 0;
            }
            splice(events, open - 1, index2 - open + 3, nextEvents);
            index2 = open + nextEvents.length - offset - 2;
            break;
          }
        }
      }
    }
    index2 = -1;
    while (++index2 < events.length) {
      if (events[index2][1].type === "attentionSequence") {
        events[index2][1].type = "data";
      }
    }
    return events;
  }
  function tokenizeAttention(effects, ok3) {
    const attentionMarkers2 = this.parser.constructs.attentionMarkers.null;
    const previous2 = this.previous;
    const before = classifyCharacter(previous2);
    let marker;
    return start;
    function start(code2) {
      marker = code2;
      effects.enter("attentionSequence");
      return inside(code2);
    }
    function inside(code2) {
      if (code2 === marker) {
        effects.consume(code2);
        return inside;
      }
      const token = effects.exit("attentionSequence");
      const after = classifyCharacter(code2);
      const open = !after || after === 2 && before || attentionMarkers2.includes(code2);
      const close2 = !before || before === 2 && after || attentionMarkers2.includes(previous2);
      token._open = Boolean(marker === 42 ? open : open && (before || !close2));
      token._close = Boolean(marker === 42 ? close2 : close2 && (after || !open));
      return ok3(code2);
    }
  }
  function movePoint(point3, offset) {
    point3.column += offset;
    point3.offset += offset;
    point3._bufferIndex += offset;
  }

  // node_modules/micromark-core-commonmark/lib/autolink.js
  var autolink = {
    name: "autolink",
    tokenize: tokenizeAutolink
  };
  function tokenizeAutolink(effects, ok3, nok) {
    let size = 0;
    return start;
    function start(code2) {
      effects.enter("autolink");
      effects.enter("autolinkMarker");
      effects.consume(code2);
      effects.exit("autolinkMarker");
      effects.enter("autolinkProtocol");
      return open;
    }
    function open(code2) {
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        return schemeOrEmailAtext;
      }
      if (code2 === 64) {
        return nok(code2);
      }
      return emailAtext(code2);
    }
    function schemeOrEmailAtext(code2) {
      if (code2 === 43 || code2 === 45 || code2 === 46 || asciiAlphanumeric(code2)) {
        size = 1;
        return schemeInsideOrEmailAtext(code2);
      }
      return emailAtext(code2);
    }
    function schemeInsideOrEmailAtext(code2) {
      if (code2 === 58) {
        effects.consume(code2);
        size = 0;
        return urlInside;
      }
      if ((code2 === 43 || code2 === 45 || code2 === 46 || asciiAlphanumeric(code2)) && size++ < 32) {
        effects.consume(code2);
        return schemeInsideOrEmailAtext;
      }
      size = 0;
      return emailAtext(code2);
    }
    function urlInside(code2) {
      if (code2 === 62) {
        effects.exit("autolinkProtocol");
        effects.enter("autolinkMarker");
        effects.consume(code2);
        effects.exit("autolinkMarker");
        effects.exit("autolink");
        return ok3;
      }
      if (code2 === null || code2 === 32 || code2 === 60 || asciiControl(code2)) {
        return nok(code2);
      }
      effects.consume(code2);
      return urlInside;
    }
    function emailAtext(code2) {
      if (code2 === 64) {
        effects.consume(code2);
        return emailAtSignOrDot;
      }
      if (asciiAtext(code2)) {
        effects.consume(code2);
        return emailAtext;
      }
      return nok(code2);
    }
    function emailAtSignOrDot(code2) {
      return asciiAlphanumeric(code2) ? emailLabel(code2) : nok(code2);
    }
    function emailLabel(code2) {
      if (code2 === 46) {
        effects.consume(code2);
        size = 0;
        return emailAtSignOrDot;
      }
      if (code2 === 62) {
        effects.exit("autolinkProtocol").type = "autolinkEmail";
        effects.enter("autolinkMarker");
        effects.consume(code2);
        effects.exit("autolinkMarker");
        effects.exit("autolink");
        return ok3;
      }
      return emailValue(code2);
    }
    function emailValue(code2) {
      if ((code2 === 45 || asciiAlphanumeric(code2)) && size++ < 63) {
        const next = code2 === 45 ? emailValue : emailLabel;
        effects.consume(code2);
        return next;
      }
      return nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/blank-line.js
  var blankLine = {
    partial: true,
    tokenize: tokenizeBlankLine
  };
  function tokenizeBlankLine(effects, ok3, nok) {
    return start;
    function start(code2) {
      return markdownSpace(code2) ? factorySpace(effects, after, "linePrefix")(code2) : after(code2);
    }
    function after(code2) {
      return code2 === null || markdownLineEnding(code2) ? ok3(code2) : nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/block-quote.js
  var blockQuote = {
    continuation: {
      tokenize: tokenizeBlockQuoteContinuation
    },
    exit,
    name: "blockQuote",
    tokenize: tokenizeBlockQuoteStart
  };
  function tokenizeBlockQuoteStart(effects, ok3, nok) {
    const self2 = this;
    return start;
    function start(code2) {
      if (code2 === 62) {
        const state = self2.containerState;
        if (!state.open) {
          effects.enter("blockQuote", {
            _container: true
          });
          state.open = true;
        }
        effects.enter("blockQuotePrefix");
        effects.enter("blockQuoteMarker");
        effects.consume(code2);
        effects.exit("blockQuoteMarker");
        return after;
      }
      return nok(code2);
    }
    function after(code2) {
      if (markdownSpace(code2)) {
        effects.enter("blockQuotePrefixWhitespace");
        effects.consume(code2);
        effects.exit("blockQuotePrefixWhitespace");
        effects.exit("blockQuotePrefix");
        return ok3;
      }
      effects.exit("blockQuotePrefix");
      return ok3(code2);
    }
  }
  function tokenizeBlockQuoteContinuation(effects, ok3, nok) {
    const self2 = this;
    return contStart;
    function contStart(code2) {
      if (markdownSpace(code2)) {
        return factorySpace(effects, contBefore, "linePrefix", self2.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4)(code2);
      }
      return contBefore(code2);
    }
    function contBefore(code2) {
      return effects.attempt(blockQuote, ok3, nok)(code2);
    }
  }
  function exit(effects) {
    effects.exit("blockQuote");
  }

  // node_modules/micromark-core-commonmark/lib/character-escape.js
  var characterEscape = {
    name: "characterEscape",
    tokenize: tokenizeCharacterEscape
  };
  function tokenizeCharacterEscape(effects, ok3, nok) {
    return start;
    function start(code2) {
      effects.enter("characterEscape");
      effects.enter("escapeMarker");
      effects.consume(code2);
      effects.exit("escapeMarker");
      return inside;
    }
    function inside(code2) {
      if (asciiPunctuation(code2)) {
        effects.enter("characterEscapeValue");
        effects.consume(code2);
        effects.exit("characterEscapeValue");
        effects.exit("characterEscape");
        return ok3;
      }
      return nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/character-reference.js
  var characterReference = {
    name: "characterReference",
    tokenize: tokenizeCharacterReference
  };
  function tokenizeCharacterReference(effects, ok3, nok) {
    const self2 = this;
    let size = 0;
    let max;
    let test;
    return start;
    function start(code2) {
      effects.enter("characterReference");
      effects.enter("characterReferenceMarker");
      effects.consume(code2);
      effects.exit("characterReferenceMarker");
      return open;
    }
    function open(code2) {
      if (code2 === 35) {
        effects.enter("characterReferenceMarkerNumeric");
        effects.consume(code2);
        effects.exit("characterReferenceMarkerNumeric");
        return numeric;
      }
      effects.enter("characterReferenceValue");
      max = 31;
      test = asciiAlphanumeric;
      return value(code2);
    }
    function numeric(code2) {
      if (code2 === 88 || code2 === 120) {
        effects.enter("characterReferenceMarkerHexadecimal");
        effects.consume(code2);
        effects.exit("characterReferenceMarkerHexadecimal");
        effects.enter("characterReferenceValue");
        max = 6;
        test = asciiHexDigit;
        return value;
      }
      effects.enter("characterReferenceValue");
      max = 7;
      test = asciiDigit;
      return value(code2);
    }
    function value(code2) {
      if (code2 === 59 && size) {
        const token = effects.exit("characterReferenceValue");
        if (test === asciiAlphanumeric && !decodeNamedCharacterReference(self2.sliceSerialize(token))) {
          return nok(code2);
        }
        effects.enter("characterReferenceMarker");
        effects.consume(code2);
        effects.exit("characterReferenceMarker");
        effects.exit("characterReference");
        return ok3;
      }
      if (test(code2) && size++ < max) {
        effects.consume(code2);
        return value;
      }
      return nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/code-fenced.js
  var nonLazyContinuation = {
    partial: true,
    tokenize: tokenizeNonLazyContinuation
  };
  var codeFenced = {
    concrete: true,
    name: "codeFenced",
    tokenize: tokenizeCodeFenced
  };
  function tokenizeCodeFenced(effects, ok3, nok) {
    const self2 = this;
    const closeStart = {
      partial: true,
      tokenize: tokenizeCloseStart
    };
    let initialPrefix = 0;
    let sizeOpen = 0;
    let marker;
    return start;
    function start(code2) {
      return beforeSequenceOpen(code2);
    }
    function beforeSequenceOpen(code2) {
      const tail = self2.events[self2.events.length - 1];
      initialPrefix = tail && tail[1].type === "linePrefix" ? tail[2].sliceSerialize(tail[1], true).length : 0;
      marker = code2;
      effects.enter("codeFenced");
      effects.enter("codeFencedFence");
      effects.enter("codeFencedFenceSequence");
      return sequenceOpen(code2);
    }
    function sequenceOpen(code2) {
      if (code2 === marker) {
        sizeOpen++;
        effects.consume(code2);
        return sequenceOpen;
      }
      if (sizeOpen < 3) {
        return nok(code2);
      }
      effects.exit("codeFencedFenceSequence");
      return markdownSpace(code2) ? factorySpace(effects, infoBefore, "whitespace")(code2) : infoBefore(code2);
    }
    function infoBefore(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("codeFencedFence");
        return self2.interrupt ? ok3(code2) : effects.check(nonLazyContinuation, atNonLazyBreak, after)(code2);
      }
      effects.enter("codeFencedFenceInfo");
      effects.enter("chunkString", {
        contentType: "string"
      });
      return info(code2);
    }
    function info(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("chunkString");
        effects.exit("codeFencedFenceInfo");
        return infoBefore(code2);
      }
      if (markdownSpace(code2)) {
        effects.exit("chunkString");
        effects.exit("codeFencedFenceInfo");
        return factorySpace(effects, metaBefore, "whitespace")(code2);
      }
      if (code2 === 96 && code2 === marker) {
        return nok(code2);
      }
      effects.consume(code2);
      return info;
    }
    function metaBefore(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        return infoBefore(code2);
      }
      effects.enter("codeFencedFenceMeta");
      effects.enter("chunkString", {
        contentType: "string"
      });
      return meta(code2);
    }
    function meta(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("chunkString");
        effects.exit("codeFencedFenceMeta");
        return infoBefore(code2);
      }
      if (code2 === 96 && code2 === marker) {
        return nok(code2);
      }
      effects.consume(code2);
      return meta;
    }
    function atNonLazyBreak(code2) {
      return effects.attempt(closeStart, after, contentBefore)(code2);
    }
    function contentBefore(code2) {
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return contentStart;
    }
    function contentStart(code2) {
      return initialPrefix > 0 && markdownSpace(code2) ? factorySpace(effects, beforeContentChunk, "linePrefix", initialPrefix + 1)(code2) : beforeContentChunk(code2);
    }
    function beforeContentChunk(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        return effects.check(nonLazyContinuation, atNonLazyBreak, after)(code2);
      }
      effects.enter("codeFlowValue");
      return contentChunk(code2);
    }
    function contentChunk(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("codeFlowValue");
        return beforeContentChunk(code2);
      }
      effects.consume(code2);
      return contentChunk;
    }
    function after(code2) {
      effects.exit("codeFenced");
      return ok3(code2);
    }
    function tokenizeCloseStart(effects2, ok4, nok2) {
      let size = 0;
      return startBefore;
      function startBefore(code2) {
        effects2.enter("lineEnding");
        effects2.consume(code2);
        effects2.exit("lineEnding");
        return start2;
      }
      function start2(code2) {
        effects2.enter("codeFencedFence");
        return markdownSpace(code2) ? factorySpace(effects2, beforeSequenceClose, "linePrefix", self2.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4)(code2) : beforeSequenceClose(code2);
      }
      function beforeSequenceClose(code2) {
        if (code2 === marker) {
          effects2.enter("codeFencedFenceSequence");
          return sequenceClose(code2);
        }
        return nok2(code2);
      }
      function sequenceClose(code2) {
        if (code2 === marker) {
          size++;
          effects2.consume(code2);
          return sequenceClose;
        }
        if (size >= sizeOpen) {
          effects2.exit("codeFencedFenceSequence");
          return markdownSpace(code2) ? factorySpace(effects2, sequenceCloseAfter, "whitespace")(code2) : sequenceCloseAfter(code2);
        }
        return nok2(code2);
      }
      function sequenceCloseAfter(code2) {
        if (code2 === null || markdownLineEnding(code2)) {
          effects2.exit("codeFencedFence");
          return ok4(code2);
        }
        return nok2(code2);
      }
    }
  }
  function tokenizeNonLazyContinuation(effects, ok3, nok) {
    const self2 = this;
    return start;
    function start(code2) {
      if (code2 === null) {
        return nok(code2);
      }
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return lineStart;
    }
    function lineStart(code2) {
      return self2.parser.lazy[self2.now().line] ? nok(code2) : ok3(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/code-indented.js
  var codeIndented = {
    name: "codeIndented",
    tokenize: tokenizeCodeIndented
  };
  var furtherStart = {
    partial: true,
    tokenize: tokenizeFurtherStart
  };
  function tokenizeCodeIndented(effects, ok3, nok) {
    const self2 = this;
    return start;
    function start(code2) {
      effects.enter("codeIndented");
      return factorySpace(effects, afterPrefix, "linePrefix", 4 + 1)(code2);
    }
    function afterPrefix(code2) {
      const tail = self2.events[self2.events.length - 1];
      return tail && tail[1].type === "linePrefix" && tail[2].sliceSerialize(tail[1], true).length >= 4 ? atBreak(code2) : nok(code2);
    }
    function atBreak(code2) {
      if (code2 === null) {
        return after(code2);
      }
      if (markdownLineEnding(code2)) {
        return effects.attempt(furtherStart, atBreak, after)(code2);
      }
      effects.enter("codeFlowValue");
      return inside(code2);
    }
    function inside(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("codeFlowValue");
        return atBreak(code2);
      }
      effects.consume(code2);
      return inside;
    }
    function after(code2) {
      effects.exit("codeIndented");
      return ok3(code2);
    }
  }
  function tokenizeFurtherStart(effects, ok3, nok) {
    const self2 = this;
    return furtherStart2;
    function furtherStart2(code2) {
      if (self2.parser.lazy[self2.now().line]) {
        return nok(code2);
      }
      if (markdownLineEnding(code2)) {
        effects.enter("lineEnding");
        effects.consume(code2);
        effects.exit("lineEnding");
        return furtherStart2;
      }
      return factorySpace(effects, afterPrefix, "linePrefix", 4 + 1)(code2);
    }
    function afterPrefix(code2) {
      const tail = self2.events[self2.events.length - 1];
      return tail && tail[1].type === "linePrefix" && tail[2].sliceSerialize(tail[1], true).length >= 4 ? ok3(code2) : markdownLineEnding(code2) ? furtherStart2(code2) : nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/code-text.js
  var codeText = {
    name: "codeText",
    previous,
    resolve: resolveCodeText,
    tokenize: tokenizeCodeText
  };
  function resolveCodeText(events) {
    let tailExitIndex = events.length - 4;
    let headEnterIndex = 3;
    let index2;
    let enter;
    if ((events[headEnterIndex][1].type === "lineEnding" || events[headEnterIndex][1].type === "space") && (events[tailExitIndex][1].type === "lineEnding" || events[tailExitIndex][1].type === "space")) {
      index2 = headEnterIndex;
      while (++index2 < tailExitIndex) {
        if (events[index2][1].type === "codeTextData") {
          events[headEnterIndex][1].type = "codeTextPadding";
          events[tailExitIndex][1].type = "codeTextPadding";
          headEnterIndex += 2;
          tailExitIndex -= 2;
          break;
        }
      }
    }
    index2 = headEnterIndex - 1;
    tailExitIndex++;
    while (++index2 <= tailExitIndex) {
      if (enter === void 0) {
        if (index2 !== tailExitIndex && events[index2][1].type !== "lineEnding") {
          enter = index2;
        }
      } else if (index2 === tailExitIndex || events[index2][1].type === "lineEnding") {
        events[enter][1].type = "codeTextData";
        if (index2 !== enter + 2) {
          events[enter][1].end = events[index2 - 1][1].end;
          events.splice(enter + 2, index2 - enter - 2);
          tailExitIndex -= index2 - enter - 2;
          index2 = enter + 2;
        }
        enter = void 0;
      }
    }
    return events;
  }
  function previous(code2) {
    return code2 !== 96 || this.events[this.events.length - 1][1].type === "characterEscape";
  }
  function tokenizeCodeText(effects, ok3, nok) {
    const self2 = this;
    let sizeOpen = 0;
    let size;
    let token;
    return start;
    function start(code2) {
      effects.enter("codeText");
      effects.enter("codeTextSequence");
      return sequenceOpen(code2);
    }
    function sequenceOpen(code2) {
      if (code2 === 96) {
        effects.consume(code2);
        sizeOpen++;
        return sequenceOpen;
      }
      effects.exit("codeTextSequence");
      return between2(code2);
    }
    function between2(code2) {
      if (code2 === null) {
        return nok(code2);
      }
      if (code2 === 32) {
        effects.enter("space");
        effects.consume(code2);
        effects.exit("space");
        return between2;
      }
      if (code2 === 96) {
        token = effects.enter("codeTextSequence");
        size = 0;
        return sequenceClose(code2);
      }
      if (markdownLineEnding(code2)) {
        effects.enter("lineEnding");
        effects.consume(code2);
        effects.exit("lineEnding");
        return between2;
      }
      effects.enter("codeTextData");
      return data(code2);
    }
    function data(code2) {
      if (code2 === null || code2 === 32 || code2 === 96 || markdownLineEnding(code2)) {
        effects.exit("codeTextData");
        return between2(code2);
      }
      effects.consume(code2);
      return data;
    }
    function sequenceClose(code2) {
      if (code2 === 96) {
        effects.consume(code2);
        size++;
        return sequenceClose;
      }
      if (size === sizeOpen) {
        effects.exit("codeTextSequence");
        effects.exit("codeText");
        return ok3(code2);
      }
      token.type = "codeTextData";
      return data(code2);
    }
  }

  // node_modules/micromark-util-subtokenize/lib/splice-buffer.js
  var SpliceBuffer = class {
    /**
     * @param {ReadonlyArray<T> | null | undefined} [initial]
     *   Initial items (optional).
     * @returns
     *   Splice buffer.
     */
    constructor(initial) {
      this.left = initial ? [...initial] : [];
      this.right = [];
    }
    /**
     * Array access;
     * does not move the cursor.
     *
     * @param {number} index
     *   Index.
     * @return {T}
     *   Item.
     */
    get(index2) {
      if (index2 < 0 || index2 >= this.left.length + this.right.length) {
        throw new RangeError("Cannot access index `" + index2 + "` in a splice buffer of size `" + (this.left.length + this.right.length) + "`");
      }
      if (index2 < this.left.length) return this.left[index2];
      return this.right[this.right.length - index2 + this.left.length - 1];
    }
    /**
     * The length of the splice buffer, one greater than the largest index in the
     * array.
     */
    get length() {
      return this.left.length + this.right.length;
    }
    /**
     * Remove and return `list[0]`;
     * moves the cursor to `0`.
     *
     * @returns {T | undefined}
     *   Item, optional.
     */
    shift() {
      this.setCursor(0);
      return this.right.pop();
    }
    /**
     * Slice the buffer to get an array;
     * does not move the cursor.
     *
     * @param {number} start
     *   Start.
     * @param {number | null | undefined} [end]
     *   End (optional).
     * @returns {Array<T>}
     *   Array of items.
     */
    slice(start, end) {
      const stop = end === null || end === void 0 ? Number.POSITIVE_INFINITY : end;
      if (stop < this.left.length) {
        return this.left.slice(start, stop);
      }
      if (start > this.left.length) {
        return this.right.slice(this.right.length - stop + this.left.length, this.right.length - start + this.left.length).reverse();
      }
      return this.left.slice(start).concat(this.right.slice(this.right.length - stop + this.left.length).reverse());
    }
    /**
     * Mimics the behavior of Array.prototype.splice() except for the change of
     * interface necessary to avoid segfaults when patching in very large arrays.
     *
     * This operation moves cursor is moved to `start` and results in the cursor
     * placed after any inserted items.
     *
     * @param {number} start
     *   Start;
     *   zero-based index at which to start changing the array;
     *   negative numbers count backwards from the end of the array and values
     *   that are out-of bounds are clamped to the appropriate end of the array.
     * @param {number | null | undefined} [deleteCount=0]
     *   Delete count (default: `0`);
     *   maximum number of elements to delete, starting from start.
     * @param {Array<T> | null | undefined} [items=[]]
     *   Items to include in place of the deleted items (default: `[]`).
     * @return {Array<T>}
     *   Any removed items.
     */
    splice(start, deleteCount, items) {
      const count = deleteCount || 0;
      this.setCursor(Math.trunc(start));
      const removed = this.right.splice(this.right.length - count, Number.POSITIVE_INFINITY);
      if (items) chunkedPush(this.left, items);
      return removed.reverse();
    }
    /**
     * Remove and return the highest-numbered item in the array, so
     * `list[list.length - 1]`;
     * Moves the cursor to `length`.
     *
     * @returns {T | undefined}
     *   Item, optional.
     */
    pop() {
      this.setCursor(Number.POSITIVE_INFINITY);
      return this.left.pop();
    }
    /**
     * Inserts a single item to the high-numbered side of the array;
     * moves the cursor to `length`.
     *
     * @param {T} item
     *   Item.
     * @returns {undefined}
     *   Nothing.
     */
    push(item) {
      this.setCursor(Number.POSITIVE_INFINITY);
      this.left.push(item);
    }
    /**
     * Inserts many items to the high-numbered side of the array.
     * Moves the cursor to `length`.
     *
     * @param {Array<T>} items
     *   Items.
     * @returns {undefined}
     *   Nothing.
     */
    pushMany(items) {
      this.setCursor(Number.POSITIVE_INFINITY);
      chunkedPush(this.left, items);
    }
    /**
     * Inserts a single item to the low-numbered side of the array;
     * Moves the cursor to `0`.
     *
     * @param {T} item
     *   Item.
     * @returns {undefined}
     *   Nothing.
     */
    unshift(item) {
      this.setCursor(0);
      this.right.push(item);
    }
    /**
     * Inserts many items to the low-numbered side of the array;
     * moves the cursor to `0`.
     *
     * @param {Array<T>} items
     *   Items.
     * @returns {undefined}
     *   Nothing.
     */
    unshiftMany(items) {
      this.setCursor(0);
      chunkedPush(this.right, items.reverse());
    }
    /**
     * Move the cursor to a specific position in the array. Requires
     * time proportional to the distance moved.
     *
     * If `n < 0`, the cursor will end up at the beginning.
     * If `n > length`, the cursor will end up at the end.
     *
     * @param {number} n
     *   Position.
     * @return {undefined}
     *   Nothing.
     */
    setCursor(n2) {
      if (n2 === this.left.length || n2 > this.left.length && this.right.length === 0 || n2 < 0 && this.left.length === 0) return;
      if (n2 < this.left.length) {
        const removed = this.left.splice(n2, Number.POSITIVE_INFINITY);
        chunkedPush(this.right, removed.reverse());
      } else {
        const removed = this.right.splice(this.left.length + this.right.length - n2, Number.POSITIVE_INFINITY);
        chunkedPush(this.left, removed.reverse());
      }
    }
  };
  function chunkedPush(list4, right) {
    let chunkStart = 0;
    if (right.length < 1e4) {
      list4.push(...right);
    } else {
      while (chunkStart < right.length) {
        list4.push(...right.slice(chunkStart, chunkStart + 1e4));
        chunkStart += 1e4;
      }
    }
  }

  // node_modules/micromark-util-subtokenize/index.js
  function subtokenize(eventsArray) {
    const jumps = {};
    let index2 = -1;
    let event;
    let lineIndex;
    let otherIndex;
    let otherEvent;
    let parameters;
    let subevents;
    let more;
    const events = new SpliceBuffer(eventsArray);
    while (++index2 < events.length) {
      while (index2 in jumps) {
        index2 = jumps[index2];
      }
      event = events.get(index2);
      if (index2 && event[1].type === "chunkFlow" && events.get(index2 - 1)[1].type === "listItemPrefix") {
        subevents = event[1]._tokenizer.events;
        otherIndex = 0;
        if (otherIndex < subevents.length && subevents[otherIndex][1].type === "lineEndingBlank") {
          otherIndex += 2;
        }
        if (otherIndex < subevents.length && subevents[otherIndex][1].type === "content") {
          while (++otherIndex < subevents.length) {
            if (subevents[otherIndex][1].type === "content") {
              break;
            }
            if (subevents[otherIndex][1].type === "chunkText") {
              subevents[otherIndex][1]._isInFirstContentOfListItem = true;
              otherIndex++;
            }
          }
        }
      }
      if (event[0] === "enter") {
        if (event[1].contentType) {
          Object.assign(jumps, subcontent(events, index2));
          index2 = jumps[index2];
          more = true;
        }
      } else if (event[1]._container) {
        otherIndex = index2;
        lineIndex = void 0;
        while (otherIndex--) {
          otherEvent = events.get(otherIndex);
          if (otherEvent[1].type === "lineEnding" || otherEvent[1].type === "lineEndingBlank") {
            if (otherEvent[0] === "enter") {
              if (lineIndex) {
                events.get(lineIndex)[1].type = "lineEndingBlank";
              }
              otherEvent[1].type = "lineEnding";
              lineIndex = otherIndex;
            }
          } else if (otherEvent[1].type === "linePrefix" || otherEvent[1].type === "listItemIndent") {
          } else {
            break;
          }
        }
        if (lineIndex) {
          event[1].end = {
            ...events.get(lineIndex)[1].start
          };
          parameters = events.slice(lineIndex, index2);
          parameters.unshift(event);
          events.splice(lineIndex, index2 - lineIndex + 1, parameters);
        }
      }
    }
    splice(eventsArray, 0, Number.POSITIVE_INFINITY, events.slice(0));
    return !more;
  }
  function subcontent(events, eventIndex) {
    const token = events.get(eventIndex)[1];
    const context = events.get(eventIndex)[2];
    let startPosition = eventIndex - 1;
    const startPositions = [];
    let tokenizer = token._tokenizer;
    if (!tokenizer) {
      tokenizer = context.parser[token.contentType](token.start);
      if (token._contentTypeTextTrailing) {
        tokenizer._contentTypeTextTrailing = true;
      }
    }
    const childEvents = tokenizer.events;
    const jumps = [];
    const gaps = {};
    let stream;
    let previous2;
    let index2 = -1;
    let current = token;
    let adjust = 0;
    let start = 0;
    const breaks = [start];
    while (current) {
      while (events.get(++startPosition)[1] !== current) {
      }
      startPositions.push(startPosition);
      if (!current._tokenizer) {
        stream = context.sliceStream(current);
        if (!current.next) {
          stream.push(null);
        }
        if (previous2) {
          tokenizer.defineSkip(current.start);
        }
        if (current._isInFirstContentOfListItem) {
          tokenizer._gfmTasklistFirstContentOfListItem = true;
        }
        tokenizer.write(stream);
        if (current._isInFirstContentOfListItem) {
          tokenizer._gfmTasklistFirstContentOfListItem = void 0;
        }
      }
      previous2 = current;
      current = current.next;
    }
    current = token;
    while (++index2 < childEvents.length) {
      if (
        // Find a void token that includes a break.
        childEvents[index2][0] === "exit" && childEvents[index2 - 1][0] === "enter" && childEvents[index2][1].type === childEvents[index2 - 1][1].type && childEvents[index2][1].start.line !== childEvents[index2][1].end.line
      ) {
        start = index2 + 1;
        breaks.push(start);
        current._tokenizer = void 0;
        current.previous = void 0;
        current = current.next;
      }
    }
    tokenizer.events = [];
    if (current) {
      current._tokenizer = void 0;
      current.previous = void 0;
    } else {
      breaks.pop();
    }
    index2 = breaks.length;
    while (index2--) {
      const slice = childEvents.slice(breaks[index2], breaks[index2 + 1]);
      const start2 = startPositions.pop();
      jumps.push([start2, start2 + slice.length - 1]);
      events.splice(start2, 2, slice);
    }
    jumps.reverse();
    index2 = -1;
    while (++index2 < jumps.length) {
      gaps[adjust + jumps[index2][0]] = adjust + jumps[index2][1];
      adjust += jumps[index2][1] - jumps[index2][0] - 1;
    }
    return gaps;
  }

  // node_modules/micromark-core-commonmark/lib/content.js
  var content2 = {
    resolve: resolveContent,
    tokenize: tokenizeContent
  };
  var continuationConstruct = {
    partial: true,
    tokenize: tokenizeContinuation
  };
  function resolveContent(events) {
    subtokenize(events);
    return events;
  }
  function tokenizeContent(effects, ok3) {
    let previous2;
    return chunkStart;
    function chunkStart(code2) {
      effects.enter("content");
      previous2 = effects.enter("chunkContent", {
        contentType: "content"
      });
      return chunkInside(code2);
    }
    function chunkInside(code2) {
      if (code2 === null) {
        return contentEnd(code2);
      }
      if (markdownLineEnding(code2)) {
        return effects.check(continuationConstruct, contentContinue, contentEnd)(code2);
      }
      effects.consume(code2);
      return chunkInside;
    }
    function contentEnd(code2) {
      effects.exit("chunkContent");
      effects.exit("content");
      return ok3(code2);
    }
    function contentContinue(code2) {
      effects.consume(code2);
      effects.exit("chunkContent");
      previous2.next = effects.enter("chunkContent", {
        contentType: "content",
        previous: previous2
      });
      previous2 = previous2.next;
      return chunkInside;
    }
  }
  function tokenizeContinuation(effects, ok3, nok) {
    const self2 = this;
    return startLookahead;
    function startLookahead(code2) {
      effects.exit("chunkContent");
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return factorySpace(effects, prefixed, "linePrefix");
    }
    function prefixed(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        return nok(code2);
      }
      const tail = self2.events[self2.events.length - 1];
      if (!self2.parser.constructs.disable.null.includes("codeIndented") && tail && tail[1].type === "linePrefix" && tail[2].sliceSerialize(tail[1], true).length >= 4) {
        return ok3(code2);
      }
      return effects.interrupt(self2.parser.constructs.flow, nok, ok3)(code2);
    }
  }

  // node_modules/micromark-factory-destination/index.js
  function factoryDestination(effects, ok3, nok, type, literalType, literalMarkerType, rawType, stringType, max) {
    const limit = max || Number.POSITIVE_INFINITY;
    let balance = 0;
    return start;
    function start(code2) {
      if (code2 === 60) {
        effects.enter(type);
        effects.enter(literalType);
        effects.enter(literalMarkerType);
        effects.consume(code2);
        effects.exit(literalMarkerType);
        return enclosedBefore;
      }
      if (code2 === null || code2 === 32 || code2 === 41 || asciiControl(code2)) {
        return nok(code2);
      }
      effects.enter(type);
      effects.enter(rawType);
      effects.enter(stringType);
      effects.enter("chunkString", {
        contentType: "string"
      });
      return raw(code2);
    }
    function enclosedBefore(code2) {
      if (code2 === 62) {
        effects.enter(literalMarkerType);
        effects.consume(code2);
        effects.exit(literalMarkerType);
        effects.exit(literalType);
        effects.exit(type);
        return ok3;
      }
      effects.enter(stringType);
      effects.enter("chunkString", {
        contentType: "string"
      });
      return enclosed(code2);
    }
    function enclosed(code2) {
      if (code2 === 62) {
        effects.exit("chunkString");
        effects.exit(stringType);
        return enclosedBefore(code2);
      }
      if (code2 === null || code2 === 60 || markdownLineEnding(code2)) {
        return nok(code2);
      }
      effects.consume(code2);
      return code2 === 92 ? enclosedEscape : enclosed;
    }
    function enclosedEscape(code2) {
      if (code2 === 60 || code2 === 62 || code2 === 92) {
        effects.consume(code2);
        return enclosed;
      }
      return enclosed(code2);
    }
    function raw(code2) {
      if (!balance && (code2 === null || code2 === 41 || markdownLineEndingOrSpace(code2))) {
        effects.exit("chunkString");
        effects.exit(stringType);
        effects.exit(rawType);
        effects.exit(type);
        return ok3(code2);
      }
      if (balance < limit && code2 === 40) {
        effects.consume(code2);
        balance++;
        return raw;
      }
      if (code2 === 41) {
        effects.consume(code2);
        balance--;
        return raw;
      }
      if (code2 === null || code2 === 32 || code2 === 40 || asciiControl(code2)) {
        return nok(code2);
      }
      effects.consume(code2);
      return code2 === 92 ? rawEscape : raw;
    }
    function rawEscape(code2) {
      if (code2 === 40 || code2 === 41 || code2 === 92) {
        effects.consume(code2);
        return raw;
      }
      return raw(code2);
    }
  }

  // node_modules/micromark-factory-label/index.js
  function factoryLabel(effects, ok3, nok, type, markerType, stringType) {
    const self2 = this;
    let size = 0;
    let seen;
    return start;
    function start(code2) {
      effects.enter(type);
      effects.enter(markerType);
      effects.consume(code2);
      effects.exit(markerType);
      effects.enter(stringType);
      return atBreak;
    }
    function atBreak(code2) {
      if (size > 999 || code2 === null || code2 === 91 || code2 === 93 && !seen || // To do: remove in the future once we’ve switched from
      // `micromark-extension-footnote` to `micromark-extension-gfm-footnote`,
      // which doesn’t need this.
      // Hidden footnotes hook.
      /* c8 ignore next 3 */
      code2 === 94 && !size && "_hiddenFootnoteSupport" in self2.parser.constructs) {
        return nok(code2);
      }
      if (code2 === 93) {
        effects.exit(stringType);
        effects.enter(markerType);
        effects.consume(code2);
        effects.exit(markerType);
        effects.exit(type);
        return ok3;
      }
      if (markdownLineEnding(code2)) {
        effects.enter("lineEnding");
        effects.consume(code2);
        effects.exit("lineEnding");
        return atBreak;
      }
      effects.enter("chunkString", {
        contentType: "string"
      });
      return labelInside(code2);
    }
    function labelInside(code2) {
      if (code2 === null || code2 === 91 || code2 === 93 || markdownLineEnding(code2) || size++ > 999) {
        effects.exit("chunkString");
        return atBreak(code2);
      }
      effects.consume(code2);
      if (!seen) seen = !markdownSpace(code2);
      return code2 === 92 ? labelEscape : labelInside;
    }
    function labelEscape(code2) {
      if (code2 === 91 || code2 === 92 || code2 === 93) {
        effects.consume(code2);
        size++;
        return labelInside;
      }
      return labelInside(code2);
    }
  }

  // node_modules/micromark-factory-title/index.js
  function factoryTitle(effects, ok3, nok, type, markerType, stringType) {
    let marker;
    return start;
    function start(code2) {
      if (code2 === 34 || code2 === 39 || code2 === 40) {
        effects.enter(type);
        effects.enter(markerType);
        effects.consume(code2);
        effects.exit(markerType);
        marker = code2 === 40 ? 41 : code2;
        return begin;
      }
      return nok(code2);
    }
    function begin(code2) {
      if (code2 === marker) {
        effects.enter(markerType);
        effects.consume(code2);
        effects.exit(markerType);
        effects.exit(type);
        return ok3;
      }
      effects.enter(stringType);
      return atBreak(code2);
    }
    function atBreak(code2) {
      if (code2 === marker) {
        effects.exit(stringType);
        return begin(marker);
      }
      if (code2 === null) {
        return nok(code2);
      }
      if (markdownLineEnding(code2)) {
        effects.enter("lineEnding");
        effects.consume(code2);
        effects.exit("lineEnding");
        return factorySpace(effects, atBreak, "linePrefix");
      }
      effects.enter("chunkString", {
        contentType: "string"
      });
      return inside(code2);
    }
    function inside(code2) {
      if (code2 === marker || code2 === null || markdownLineEnding(code2)) {
        effects.exit("chunkString");
        return atBreak(code2);
      }
      effects.consume(code2);
      return code2 === 92 ? escape : inside;
    }
    function escape(code2) {
      if (code2 === marker || code2 === 92) {
        effects.consume(code2);
        return inside;
      }
      return inside(code2);
    }
  }

  // node_modules/micromark-factory-whitespace/index.js
  function factoryWhitespace(effects, ok3) {
    let seen;
    return start;
    function start(code2) {
      if (markdownLineEnding(code2)) {
        effects.enter("lineEnding");
        effects.consume(code2);
        effects.exit("lineEnding");
        seen = true;
        return start;
      }
      if (markdownSpace(code2)) {
        return factorySpace(effects, start, seen ? "linePrefix" : "lineSuffix")(code2);
      }
      return ok3(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/definition.js
  var definition = {
    name: "definition",
    tokenize: tokenizeDefinition
  };
  var titleBefore = {
    partial: true,
    tokenize: tokenizeTitleBefore
  };
  function tokenizeDefinition(effects, ok3, nok) {
    const self2 = this;
    let identifier;
    return start;
    function start(code2) {
      effects.enter("definition");
      return before(code2);
    }
    function before(code2) {
      return factoryLabel.call(
        self2,
        effects,
        labelAfter,
        // Note: we don’t need to reset the way `markdown-rs` does.
        nok,
        "definitionLabel",
        "definitionLabelMarker",
        "definitionLabelString"
      )(code2);
    }
    function labelAfter(code2) {
      identifier = normalizeIdentifier(self2.sliceSerialize(self2.events[self2.events.length - 1][1]).slice(1, -1));
      if (code2 === 58) {
        effects.enter("definitionMarker");
        effects.consume(code2);
        effects.exit("definitionMarker");
        return markerAfter;
      }
      return nok(code2);
    }
    function markerAfter(code2) {
      return markdownLineEndingOrSpace(code2) ? factoryWhitespace(effects, destinationBefore)(code2) : destinationBefore(code2);
    }
    function destinationBefore(code2) {
      return factoryDestination(
        effects,
        destinationAfter,
        // Note: we don’t need to reset the way `markdown-rs` does.
        nok,
        "definitionDestination",
        "definitionDestinationLiteral",
        "definitionDestinationLiteralMarker",
        "definitionDestinationRaw",
        "definitionDestinationString"
      )(code2);
    }
    function destinationAfter(code2) {
      return effects.attempt(titleBefore, after, after)(code2);
    }
    function after(code2) {
      return markdownSpace(code2) ? factorySpace(effects, afterWhitespace, "whitespace")(code2) : afterWhitespace(code2);
    }
    function afterWhitespace(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("definition");
        self2.parser.defined.push(identifier);
        return ok3(code2);
      }
      return nok(code2);
    }
  }
  function tokenizeTitleBefore(effects, ok3, nok) {
    return titleBefore2;
    function titleBefore2(code2) {
      return markdownLineEndingOrSpace(code2) ? factoryWhitespace(effects, beforeMarker)(code2) : nok(code2);
    }
    function beforeMarker(code2) {
      return factoryTitle(effects, titleAfter, nok, "definitionTitle", "definitionTitleMarker", "definitionTitleString")(code2);
    }
    function titleAfter(code2) {
      return markdownSpace(code2) ? factorySpace(effects, titleAfterOptionalWhitespace, "whitespace")(code2) : titleAfterOptionalWhitespace(code2);
    }
    function titleAfterOptionalWhitespace(code2) {
      return code2 === null || markdownLineEnding(code2) ? ok3(code2) : nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/hard-break-escape.js
  var hardBreakEscape = {
    name: "hardBreakEscape",
    tokenize: tokenizeHardBreakEscape
  };
  function tokenizeHardBreakEscape(effects, ok3, nok) {
    return start;
    function start(code2) {
      effects.enter("hardBreakEscape");
      effects.consume(code2);
      return after;
    }
    function after(code2) {
      if (markdownLineEnding(code2)) {
        effects.exit("hardBreakEscape");
        return ok3(code2);
      }
      return nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/heading-atx.js
  var headingAtx = {
    name: "headingAtx",
    resolve: resolveHeadingAtx,
    tokenize: tokenizeHeadingAtx
  };
  function resolveHeadingAtx(events, context) {
    let contentEnd = events.length - 2;
    let contentStart = 3;
    let content3;
    let text4;
    if (events[contentStart][1].type === "whitespace") {
      contentStart += 2;
    }
    if (contentEnd - 2 > contentStart && events[contentEnd][1].type === "whitespace") {
      contentEnd -= 2;
    }
    if (events[contentEnd][1].type === "atxHeadingSequence" && (contentStart === contentEnd - 1 || contentEnd - 4 > contentStart && events[contentEnd - 2][1].type === "whitespace")) {
      contentEnd -= contentStart + 1 === contentEnd ? 2 : 4;
    }
    if (contentEnd > contentStart) {
      content3 = {
        type: "atxHeadingText",
        start: events[contentStart][1].start,
        end: events[contentEnd][1].end
      };
      text4 = {
        type: "chunkText",
        start: events[contentStart][1].start,
        end: events[contentEnd][1].end,
        contentType: "text"
      };
      splice(events, contentStart, contentEnd - contentStart + 1, [["enter", content3, context], ["enter", text4, context], ["exit", text4, context], ["exit", content3, context]]);
    }
    return events;
  }
  function tokenizeHeadingAtx(effects, ok3, nok) {
    let size = 0;
    return start;
    function start(code2) {
      effects.enter("atxHeading");
      return before(code2);
    }
    function before(code2) {
      effects.enter("atxHeadingSequence");
      return sequenceOpen(code2);
    }
    function sequenceOpen(code2) {
      if (code2 === 35 && size++ < 6) {
        effects.consume(code2);
        return sequenceOpen;
      }
      if (code2 === null || markdownLineEndingOrSpace(code2)) {
        effects.exit("atxHeadingSequence");
        return atBreak(code2);
      }
      return nok(code2);
    }
    function atBreak(code2) {
      if (code2 === 35) {
        effects.enter("atxHeadingSequence");
        return sequenceFurther(code2);
      }
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("atxHeading");
        return ok3(code2);
      }
      if (markdownSpace(code2)) {
        return factorySpace(effects, atBreak, "whitespace")(code2);
      }
      effects.enter("atxHeadingText");
      return data(code2);
    }
    function sequenceFurther(code2) {
      if (code2 === 35) {
        effects.consume(code2);
        return sequenceFurther;
      }
      effects.exit("atxHeadingSequence");
      return atBreak(code2);
    }
    function data(code2) {
      if (code2 === null || code2 === 35 || markdownLineEndingOrSpace(code2)) {
        effects.exit("atxHeadingText");
        return atBreak(code2);
      }
      effects.consume(code2);
      return data;
    }
  }

  // node_modules/micromark-util-html-tag-name/index.js
  var htmlBlockNames = [
    "address",
    "article",
    "aside",
    "base",
    "basefont",
    "blockquote",
    "body",
    "caption",
    "center",
    "col",
    "colgroup",
    "dd",
    "details",
    "dialog",
    "dir",
    "div",
    "dl",
    "dt",
    "fieldset",
    "figcaption",
    "figure",
    "footer",
    "form",
    "frame",
    "frameset",
    "h1",
    "h2",
    "h3",
    "h4",
    "h5",
    "h6",
    "head",
    "header",
    "hr",
    "html",
    "iframe",
    "legend",
    "li",
    "link",
    "main",
    "menu",
    "menuitem",
    "nav",
    "noframes",
    "ol",
    "optgroup",
    "option",
    "p",
    "param",
    "search",
    "section",
    "summary",
    "table",
    "tbody",
    "td",
    "tfoot",
    "th",
    "thead",
    "title",
    "tr",
    "track",
    "ul"
  ];
  var htmlRawNames = ["pre", "script", "style", "textarea"];

  // node_modules/micromark-core-commonmark/lib/html-flow.js
  var htmlFlow = {
    concrete: true,
    name: "htmlFlow",
    resolveTo: resolveToHtmlFlow,
    tokenize: tokenizeHtmlFlow
  };
  var blankLineBefore = {
    partial: true,
    tokenize: tokenizeBlankLineBefore
  };
  var nonLazyContinuationStart = {
    partial: true,
    tokenize: tokenizeNonLazyContinuationStart
  };
  function resolveToHtmlFlow(events) {
    let index2 = events.length;
    while (index2--) {
      if (events[index2][0] === "enter" && events[index2][1].type === "htmlFlow") {
        break;
      }
    }
    if (index2 > 1 && events[index2 - 2][1].type === "linePrefix") {
      events[index2][1].start = events[index2 - 2][1].start;
      events[index2 + 1][1].start = events[index2 - 2][1].start;
      events.splice(index2 - 2, 2);
    }
    return events;
  }
  function tokenizeHtmlFlow(effects, ok3, nok) {
    const self2 = this;
    let marker;
    let closingTag;
    let buffer;
    let index2;
    let markerB;
    return start;
    function start(code2) {
      return before(code2);
    }
    function before(code2) {
      effects.enter("htmlFlow");
      effects.enter("htmlFlowData");
      effects.consume(code2);
      return open;
    }
    function open(code2) {
      if (code2 === 33) {
        effects.consume(code2);
        return declarationOpen;
      }
      if (code2 === 47) {
        effects.consume(code2);
        closingTag = true;
        return tagCloseStart;
      }
      if (code2 === 63) {
        effects.consume(code2);
        marker = 3;
        return self2.interrupt ? ok3 : continuationDeclarationInside;
      }
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        buffer = String.fromCharCode(code2);
        return tagName;
      }
      return nok(code2);
    }
    function declarationOpen(code2) {
      if (code2 === 45) {
        effects.consume(code2);
        marker = 2;
        return commentOpenInside;
      }
      if (code2 === 91) {
        effects.consume(code2);
        marker = 5;
        index2 = 0;
        return cdataOpenInside;
      }
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        marker = 4;
        return self2.interrupt ? ok3 : continuationDeclarationInside;
      }
      return nok(code2);
    }
    function commentOpenInside(code2) {
      if (code2 === 45) {
        effects.consume(code2);
        return self2.interrupt ? ok3 : continuationDeclarationInside;
      }
      return nok(code2);
    }
    function cdataOpenInside(code2) {
      const value = "CDATA[";
      if (code2 === value.charCodeAt(index2++)) {
        effects.consume(code2);
        if (index2 === value.length) {
          return self2.interrupt ? ok3 : continuation;
        }
        return cdataOpenInside;
      }
      return nok(code2);
    }
    function tagCloseStart(code2) {
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        buffer = String.fromCharCode(code2);
        return tagName;
      }
      return nok(code2);
    }
    function tagName(code2) {
      if (code2 === null || code2 === 47 || code2 === 62 || markdownLineEndingOrSpace(code2)) {
        const slash = code2 === 47;
        const name = buffer.toLowerCase();
        if (!slash && !closingTag && htmlRawNames.includes(name)) {
          marker = 1;
          return self2.interrupt ? ok3(code2) : continuation(code2);
        }
        if (htmlBlockNames.includes(buffer.toLowerCase())) {
          marker = 6;
          if (slash) {
            effects.consume(code2);
            return basicSelfClosing;
          }
          return self2.interrupt ? ok3(code2) : continuation(code2);
        }
        marker = 7;
        return self2.interrupt && !self2.parser.lazy[self2.now().line] ? nok(code2) : closingTag ? completeClosingTagAfter(code2) : completeAttributeNameBefore(code2);
      }
      if (code2 === 45 || asciiAlphanumeric(code2)) {
        effects.consume(code2);
        buffer += String.fromCharCode(code2);
        return tagName;
      }
      return nok(code2);
    }
    function basicSelfClosing(code2) {
      if (code2 === 62) {
        effects.consume(code2);
        return self2.interrupt ? ok3 : continuation;
      }
      return nok(code2);
    }
    function completeClosingTagAfter(code2) {
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return completeClosingTagAfter;
      }
      return completeEnd(code2);
    }
    function completeAttributeNameBefore(code2) {
      if (code2 === 47) {
        effects.consume(code2);
        return completeEnd;
      }
      if (code2 === 58 || code2 === 95 || asciiAlpha(code2)) {
        effects.consume(code2);
        return completeAttributeName;
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return completeAttributeNameBefore;
      }
      return completeEnd(code2);
    }
    function completeAttributeName(code2) {
      if (code2 === 45 || code2 === 46 || code2 === 58 || code2 === 95 || asciiAlphanumeric(code2)) {
        effects.consume(code2);
        return completeAttributeName;
      }
      return completeAttributeNameAfter(code2);
    }
    function completeAttributeNameAfter(code2) {
      if (code2 === 61) {
        effects.consume(code2);
        return completeAttributeValueBefore;
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return completeAttributeNameAfter;
      }
      return completeAttributeNameBefore(code2);
    }
    function completeAttributeValueBefore(code2) {
      if (code2 === null || code2 === 60 || code2 === 61 || code2 === 62 || code2 === 96) {
        return nok(code2);
      }
      if (code2 === 34 || code2 === 39) {
        effects.consume(code2);
        markerB = code2;
        return completeAttributeValueQuoted;
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return completeAttributeValueBefore;
      }
      return completeAttributeValueUnquoted(code2);
    }
    function completeAttributeValueQuoted(code2) {
      if (code2 === markerB) {
        effects.consume(code2);
        markerB = null;
        return completeAttributeValueQuotedAfter;
      }
      if (code2 === null || markdownLineEnding(code2)) {
        return nok(code2);
      }
      effects.consume(code2);
      return completeAttributeValueQuoted;
    }
    function completeAttributeValueUnquoted(code2) {
      if (code2 === null || code2 === 34 || code2 === 39 || code2 === 47 || code2 === 60 || code2 === 61 || code2 === 62 || code2 === 96 || markdownLineEndingOrSpace(code2)) {
        return completeAttributeNameAfter(code2);
      }
      effects.consume(code2);
      return completeAttributeValueUnquoted;
    }
    function completeAttributeValueQuotedAfter(code2) {
      if (code2 === 47 || code2 === 62 || markdownSpace(code2)) {
        return completeAttributeNameBefore(code2);
      }
      return nok(code2);
    }
    function completeEnd(code2) {
      if (code2 === 62) {
        effects.consume(code2);
        return completeAfter;
      }
      return nok(code2);
    }
    function completeAfter(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        return continuation(code2);
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return completeAfter;
      }
      return nok(code2);
    }
    function continuation(code2) {
      if (code2 === 45 && marker === 2) {
        effects.consume(code2);
        return continuationCommentInside;
      }
      if (code2 === 60 && marker === 1) {
        effects.consume(code2);
        return continuationRawTagOpen;
      }
      if (code2 === 62 && marker === 4) {
        effects.consume(code2);
        return continuationClose;
      }
      if (code2 === 63 && marker === 3) {
        effects.consume(code2);
        return continuationDeclarationInside;
      }
      if (code2 === 93 && marker === 5) {
        effects.consume(code2);
        return continuationCdataInside;
      }
      if (markdownLineEnding(code2) && (marker === 6 || marker === 7)) {
        effects.exit("htmlFlowData");
        return effects.check(blankLineBefore, continuationAfter, continuationStart)(code2);
      }
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("htmlFlowData");
        return continuationStart(code2);
      }
      effects.consume(code2);
      return continuation;
    }
    function continuationStart(code2) {
      return effects.check(nonLazyContinuationStart, continuationStartNonLazy, continuationAfter)(code2);
    }
    function continuationStartNonLazy(code2) {
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return continuationBefore;
    }
    function continuationBefore(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        return continuationStart(code2);
      }
      effects.enter("htmlFlowData");
      return continuation(code2);
    }
    function continuationCommentInside(code2) {
      if (code2 === 45) {
        effects.consume(code2);
        return continuationDeclarationInside;
      }
      return continuation(code2);
    }
    function continuationRawTagOpen(code2) {
      if (code2 === 47) {
        effects.consume(code2);
        buffer = "";
        return continuationRawEndTag;
      }
      return continuation(code2);
    }
    function continuationRawEndTag(code2) {
      if (code2 === 62) {
        const name = buffer.toLowerCase();
        if (htmlRawNames.includes(name)) {
          effects.consume(code2);
          return continuationClose;
        }
        return continuation(code2);
      }
      if (asciiAlpha(code2) && buffer.length < 8) {
        effects.consume(code2);
        buffer += String.fromCharCode(code2);
        return continuationRawEndTag;
      }
      return continuation(code2);
    }
    function continuationCdataInside(code2) {
      if (code2 === 93) {
        effects.consume(code2);
        return continuationDeclarationInside;
      }
      return continuation(code2);
    }
    function continuationDeclarationInside(code2) {
      if (code2 === 62) {
        effects.consume(code2);
        return continuationClose;
      }
      if (code2 === 45 && marker === 2) {
        effects.consume(code2);
        return continuationDeclarationInside;
      }
      return continuation(code2);
    }
    function continuationClose(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("htmlFlowData");
        return continuationAfter(code2);
      }
      effects.consume(code2);
      return continuationClose;
    }
    function continuationAfter(code2) {
      effects.exit("htmlFlow");
      return ok3(code2);
    }
  }
  function tokenizeNonLazyContinuationStart(effects, ok3, nok) {
    const self2 = this;
    return start;
    function start(code2) {
      if (markdownLineEnding(code2)) {
        effects.enter("lineEnding");
        effects.consume(code2);
        effects.exit("lineEnding");
        return after;
      }
      return nok(code2);
    }
    function after(code2) {
      return self2.parser.lazy[self2.now().line] ? nok(code2) : ok3(code2);
    }
  }
  function tokenizeBlankLineBefore(effects, ok3, nok) {
    return start;
    function start(code2) {
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return effects.attempt(blankLine, ok3, nok);
    }
  }

  // node_modules/micromark-core-commonmark/lib/html-text.js
  var htmlText = {
    name: "htmlText",
    tokenize: tokenizeHtmlText
  };
  function tokenizeHtmlText(effects, ok3, nok) {
    const self2 = this;
    let marker;
    let index2;
    let returnState;
    return start;
    function start(code2) {
      effects.enter("htmlText");
      effects.enter("htmlTextData");
      effects.consume(code2);
      return open;
    }
    function open(code2) {
      if (code2 === 33) {
        effects.consume(code2);
        return declarationOpen;
      }
      if (code2 === 47) {
        effects.consume(code2);
        return tagCloseStart;
      }
      if (code2 === 63) {
        effects.consume(code2);
        return instruction;
      }
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        return tagOpen;
      }
      return nok(code2);
    }
    function declarationOpen(code2) {
      if (code2 === 45) {
        effects.consume(code2);
        return commentOpenInside;
      }
      if (code2 === 91) {
        effects.consume(code2);
        index2 = 0;
        return cdataOpenInside;
      }
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        return declaration;
      }
      return nok(code2);
    }
    function commentOpenInside(code2) {
      if (code2 === 45) {
        effects.consume(code2);
        return commentEnd;
      }
      return nok(code2);
    }
    function comment(code2) {
      if (code2 === null) {
        return nok(code2);
      }
      if (code2 === 45) {
        effects.consume(code2);
        return commentClose;
      }
      if (markdownLineEnding(code2)) {
        returnState = comment;
        return lineEndingBefore(code2);
      }
      effects.consume(code2);
      return comment;
    }
    function commentClose(code2) {
      if (code2 === 45) {
        effects.consume(code2);
        return commentEnd;
      }
      return comment(code2);
    }
    function commentEnd(code2) {
      return code2 === 62 ? end(code2) : code2 === 45 ? commentClose(code2) : comment(code2);
    }
    function cdataOpenInside(code2) {
      const value = "CDATA[";
      if (code2 === value.charCodeAt(index2++)) {
        effects.consume(code2);
        return index2 === value.length ? cdata : cdataOpenInside;
      }
      return nok(code2);
    }
    function cdata(code2) {
      if (code2 === null) {
        return nok(code2);
      }
      if (code2 === 93) {
        effects.consume(code2);
        return cdataClose;
      }
      if (markdownLineEnding(code2)) {
        returnState = cdata;
        return lineEndingBefore(code2);
      }
      effects.consume(code2);
      return cdata;
    }
    function cdataClose(code2) {
      if (code2 === 93) {
        effects.consume(code2);
        return cdataEnd;
      }
      return cdata(code2);
    }
    function cdataEnd(code2) {
      if (code2 === 62) {
        return end(code2);
      }
      if (code2 === 93) {
        effects.consume(code2);
        return cdataEnd;
      }
      return cdata(code2);
    }
    function declaration(code2) {
      if (code2 === null || code2 === 62) {
        return end(code2);
      }
      if (markdownLineEnding(code2)) {
        returnState = declaration;
        return lineEndingBefore(code2);
      }
      effects.consume(code2);
      return declaration;
    }
    function instruction(code2) {
      if (code2 === null) {
        return nok(code2);
      }
      if (code2 === 63) {
        effects.consume(code2);
        return instructionClose;
      }
      if (markdownLineEnding(code2)) {
        returnState = instruction;
        return lineEndingBefore(code2);
      }
      effects.consume(code2);
      return instruction;
    }
    function instructionClose(code2) {
      return code2 === 62 ? end(code2) : instruction(code2);
    }
    function tagCloseStart(code2) {
      if (asciiAlpha(code2)) {
        effects.consume(code2);
        return tagClose;
      }
      return nok(code2);
    }
    function tagClose(code2) {
      if (code2 === 45 || asciiAlphanumeric(code2)) {
        effects.consume(code2);
        return tagClose;
      }
      return tagCloseBetween(code2);
    }
    function tagCloseBetween(code2) {
      if (markdownLineEnding(code2)) {
        returnState = tagCloseBetween;
        return lineEndingBefore(code2);
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return tagCloseBetween;
      }
      return end(code2);
    }
    function tagOpen(code2) {
      if (code2 === 45 || asciiAlphanumeric(code2)) {
        effects.consume(code2);
        return tagOpen;
      }
      if (code2 === 47 || code2 === 62 || markdownLineEndingOrSpace(code2)) {
        return tagOpenBetween(code2);
      }
      return nok(code2);
    }
    function tagOpenBetween(code2) {
      if (code2 === 47) {
        effects.consume(code2);
        return end;
      }
      if (code2 === 58 || code2 === 95 || asciiAlpha(code2)) {
        effects.consume(code2);
        return tagOpenAttributeName;
      }
      if (markdownLineEnding(code2)) {
        returnState = tagOpenBetween;
        return lineEndingBefore(code2);
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return tagOpenBetween;
      }
      return end(code2);
    }
    function tagOpenAttributeName(code2) {
      if (code2 === 45 || code2 === 46 || code2 === 58 || code2 === 95 || asciiAlphanumeric(code2)) {
        effects.consume(code2);
        return tagOpenAttributeName;
      }
      return tagOpenAttributeNameAfter(code2);
    }
    function tagOpenAttributeNameAfter(code2) {
      if (code2 === 61) {
        effects.consume(code2);
        return tagOpenAttributeValueBefore;
      }
      if (markdownLineEnding(code2)) {
        returnState = tagOpenAttributeNameAfter;
        return lineEndingBefore(code2);
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return tagOpenAttributeNameAfter;
      }
      return tagOpenBetween(code2);
    }
    function tagOpenAttributeValueBefore(code2) {
      if (code2 === null || code2 === 60 || code2 === 61 || code2 === 62 || code2 === 96) {
        return nok(code2);
      }
      if (code2 === 34 || code2 === 39) {
        effects.consume(code2);
        marker = code2;
        return tagOpenAttributeValueQuoted;
      }
      if (markdownLineEnding(code2)) {
        returnState = tagOpenAttributeValueBefore;
        return lineEndingBefore(code2);
      }
      if (markdownSpace(code2)) {
        effects.consume(code2);
        return tagOpenAttributeValueBefore;
      }
      effects.consume(code2);
      return tagOpenAttributeValueUnquoted;
    }
    function tagOpenAttributeValueQuoted(code2) {
      if (code2 === marker) {
        effects.consume(code2);
        marker = void 0;
        return tagOpenAttributeValueQuotedAfter;
      }
      if (code2 === null) {
        return nok(code2);
      }
      if (markdownLineEnding(code2)) {
        returnState = tagOpenAttributeValueQuoted;
        return lineEndingBefore(code2);
      }
      effects.consume(code2);
      return tagOpenAttributeValueQuoted;
    }
    function tagOpenAttributeValueUnquoted(code2) {
      if (code2 === null || code2 === 34 || code2 === 39 || code2 === 60 || code2 === 61 || code2 === 96) {
        return nok(code2);
      }
      if (code2 === 47 || code2 === 62 || markdownLineEndingOrSpace(code2)) {
        return tagOpenBetween(code2);
      }
      effects.consume(code2);
      return tagOpenAttributeValueUnquoted;
    }
    function tagOpenAttributeValueQuotedAfter(code2) {
      if (code2 === 47 || code2 === 62 || markdownLineEndingOrSpace(code2)) {
        return tagOpenBetween(code2);
      }
      return nok(code2);
    }
    function end(code2) {
      if (code2 === 62) {
        effects.consume(code2);
        effects.exit("htmlTextData");
        effects.exit("htmlText");
        return ok3;
      }
      return nok(code2);
    }
    function lineEndingBefore(code2) {
      effects.exit("htmlTextData");
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return lineEndingAfter;
    }
    function lineEndingAfter(code2) {
      return markdownSpace(code2) ? factorySpace(effects, lineEndingAfterPrefix, "linePrefix", self2.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4)(code2) : lineEndingAfterPrefix(code2);
    }
    function lineEndingAfterPrefix(code2) {
      effects.enter("htmlTextData");
      return returnState(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/label-end.js
  var labelEnd = {
    name: "labelEnd",
    resolveAll: resolveAllLabelEnd,
    resolveTo: resolveToLabelEnd,
    tokenize: tokenizeLabelEnd
  };
  var resourceConstruct = {
    tokenize: tokenizeResource
  };
  var referenceFullConstruct = {
    tokenize: tokenizeReferenceFull
  };
  var referenceCollapsedConstruct = {
    tokenize: tokenizeReferenceCollapsed
  };
  function resolveAllLabelEnd(events) {
    let index2 = -1;
    const newEvents = [];
    while (++index2 < events.length) {
      const token = events[index2][1];
      newEvents.push(events[index2]);
      if (token.type === "labelImage" || token.type === "labelLink" || token.type === "labelEnd") {
        const offset = token.type === "labelImage" ? 4 : 2;
        token.type = "data";
        index2 += offset;
      }
    }
    if (events.length !== newEvents.length) {
      splice(events, 0, events.length, newEvents);
    }
    return events;
  }
  function resolveToLabelEnd(events, context) {
    let index2 = events.length;
    let offset = 0;
    let token;
    let open;
    let close2;
    let media;
    while (index2--) {
      token = events[index2][1];
      if (open) {
        if (token.type === "link" || token.type === "labelLink" && token._inactive) {
          break;
        }
        if (events[index2][0] === "enter" && token.type === "labelLink") {
          token._inactive = true;
        }
      } else if (close2) {
        if (events[index2][0] === "enter" && (token.type === "labelImage" || token.type === "labelLink") && !token._balanced) {
          open = index2;
          if (token.type !== "labelLink") {
            offset = 2;
            break;
          }
        }
      } else if (token.type === "labelEnd") {
        close2 = index2;
      }
    }
    const group = {
      type: events[open][1].type === "labelLink" ? "link" : "image",
      start: {
        ...events[open][1].start
      },
      end: {
        ...events[events.length - 1][1].end
      }
    };
    const label = {
      type: "label",
      start: {
        ...events[open][1].start
      },
      end: {
        ...events[close2][1].end
      }
    };
    const text4 = {
      type: "labelText",
      start: {
        ...events[open + offset + 2][1].end
      },
      end: {
        ...events[close2 - 2][1].start
      }
    };
    media = [["enter", group, context], ["enter", label, context]];
    media = push(media, events.slice(open + 1, open + offset + 3));
    media = push(media, [["enter", text4, context]]);
    media = push(media, resolveAll(context.parser.constructs.insideSpan.null, events.slice(open + offset + 4, close2 - 3), context));
    media = push(media, [["exit", text4, context], events[close2 - 2], events[close2 - 1], ["exit", label, context]]);
    media = push(media, events.slice(close2 + 1));
    media = push(media, [["exit", group, context]]);
    splice(events, open, events.length, media);
    return events;
  }
  function tokenizeLabelEnd(effects, ok3, nok) {
    const self2 = this;
    let index2 = self2.events.length;
    let labelStart;
    let defined;
    while (index2--) {
      if ((self2.events[index2][1].type === "labelImage" || self2.events[index2][1].type === "labelLink") && !self2.events[index2][1]._balanced) {
        labelStart = self2.events[index2][1];
        break;
      }
    }
    return start;
    function start(code2) {
      if (!labelStart) {
        return nok(code2);
      }
      if (labelStart._inactive) {
        return labelEndNok(code2);
      }
      defined = self2.parser.defined.includes(normalizeIdentifier(self2.sliceSerialize({
        start: labelStart.end,
        end: self2.now()
      })));
      effects.enter("labelEnd");
      effects.enter("labelMarker");
      effects.consume(code2);
      effects.exit("labelMarker");
      effects.exit("labelEnd");
      return after;
    }
    function after(code2) {
      if (code2 === 40) {
        return effects.attempt(resourceConstruct, labelEndOk, defined ? labelEndOk : labelEndNok)(code2);
      }
      if (code2 === 91) {
        return effects.attempt(referenceFullConstruct, labelEndOk, defined ? referenceNotFull : labelEndNok)(code2);
      }
      return defined ? labelEndOk(code2) : labelEndNok(code2);
    }
    function referenceNotFull(code2) {
      return effects.attempt(referenceCollapsedConstruct, labelEndOk, labelEndNok)(code2);
    }
    function labelEndOk(code2) {
      return ok3(code2);
    }
    function labelEndNok(code2) {
      labelStart._balanced = true;
      return nok(code2);
    }
  }
  function tokenizeResource(effects, ok3, nok) {
    return resourceStart;
    function resourceStart(code2) {
      effects.enter("resource");
      effects.enter("resourceMarker");
      effects.consume(code2);
      effects.exit("resourceMarker");
      return resourceBefore;
    }
    function resourceBefore(code2) {
      return markdownLineEndingOrSpace(code2) ? factoryWhitespace(effects, resourceOpen)(code2) : resourceOpen(code2);
    }
    function resourceOpen(code2) {
      if (code2 === 41) {
        return resourceEnd(code2);
      }
      return factoryDestination(effects, resourceDestinationAfter, resourceDestinationMissing, "resourceDestination", "resourceDestinationLiteral", "resourceDestinationLiteralMarker", "resourceDestinationRaw", "resourceDestinationString", 32)(code2);
    }
    function resourceDestinationAfter(code2) {
      return markdownLineEndingOrSpace(code2) ? factoryWhitespace(effects, resourceBetween)(code2) : resourceEnd(code2);
    }
    function resourceDestinationMissing(code2) {
      return nok(code2);
    }
    function resourceBetween(code2) {
      if (code2 === 34 || code2 === 39 || code2 === 40) {
        return factoryTitle(effects, resourceTitleAfter, nok, "resourceTitle", "resourceTitleMarker", "resourceTitleString")(code2);
      }
      return resourceEnd(code2);
    }
    function resourceTitleAfter(code2) {
      return markdownLineEndingOrSpace(code2) ? factoryWhitespace(effects, resourceEnd)(code2) : resourceEnd(code2);
    }
    function resourceEnd(code2) {
      if (code2 === 41) {
        effects.enter("resourceMarker");
        effects.consume(code2);
        effects.exit("resourceMarker");
        effects.exit("resource");
        return ok3;
      }
      return nok(code2);
    }
  }
  function tokenizeReferenceFull(effects, ok3, nok) {
    const self2 = this;
    return referenceFull;
    function referenceFull(code2) {
      return factoryLabel.call(self2, effects, referenceFullAfter, referenceFullMissing, "reference", "referenceMarker", "referenceString")(code2);
    }
    function referenceFullAfter(code2) {
      return self2.parser.defined.includes(normalizeIdentifier(self2.sliceSerialize(self2.events[self2.events.length - 1][1]).slice(1, -1))) ? ok3(code2) : nok(code2);
    }
    function referenceFullMissing(code2) {
      return nok(code2);
    }
  }
  function tokenizeReferenceCollapsed(effects, ok3, nok) {
    return referenceCollapsedStart;
    function referenceCollapsedStart(code2) {
      effects.enter("reference");
      effects.enter("referenceMarker");
      effects.consume(code2);
      effects.exit("referenceMarker");
      return referenceCollapsedOpen;
    }
    function referenceCollapsedOpen(code2) {
      if (code2 === 93) {
        effects.enter("referenceMarker");
        effects.consume(code2);
        effects.exit("referenceMarker");
        effects.exit("reference");
        return ok3;
      }
      return nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/label-start-image.js
  var labelStartImage = {
    name: "labelStartImage",
    resolveAll: labelEnd.resolveAll,
    tokenize: tokenizeLabelStartImage
  };
  function tokenizeLabelStartImage(effects, ok3, nok) {
    const self2 = this;
    return start;
    function start(code2) {
      effects.enter("labelImage");
      effects.enter("labelImageMarker");
      effects.consume(code2);
      effects.exit("labelImageMarker");
      return open;
    }
    function open(code2) {
      if (code2 === 91) {
        effects.enter("labelMarker");
        effects.consume(code2);
        effects.exit("labelMarker");
        effects.exit("labelImage");
        return after;
      }
      return nok(code2);
    }
    function after(code2) {
      return code2 === 94 && "_hiddenFootnoteSupport" in self2.parser.constructs ? nok(code2) : ok3(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/label-start-link.js
  var labelStartLink = {
    name: "labelStartLink",
    resolveAll: labelEnd.resolveAll,
    tokenize: tokenizeLabelStartLink
  };
  function tokenizeLabelStartLink(effects, ok3, nok) {
    const self2 = this;
    return start;
    function start(code2) {
      effects.enter("labelLink");
      effects.enter("labelMarker");
      effects.consume(code2);
      effects.exit("labelMarker");
      effects.exit("labelLink");
      return after;
    }
    function after(code2) {
      return code2 === 94 && "_hiddenFootnoteSupport" in self2.parser.constructs ? nok(code2) : ok3(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/line-ending.js
  var lineEnding = {
    name: "lineEnding",
    tokenize: tokenizeLineEnding
  };
  function tokenizeLineEnding(effects, ok3) {
    return start;
    function start(code2) {
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      return factorySpace(effects, ok3, "linePrefix");
    }
  }

  // node_modules/micromark-core-commonmark/lib/thematic-break.js
  var thematicBreak = {
    name: "thematicBreak",
    tokenize: tokenizeThematicBreak
  };
  function tokenizeThematicBreak(effects, ok3, nok) {
    let size = 0;
    let marker;
    return start;
    function start(code2) {
      effects.enter("thematicBreak");
      return before(code2);
    }
    function before(code2) {
      marker = code2;
      return atBreak(code2);
    }
    function atBreak(code2) {
      if (code2 === marker) {
        effects.enter("thematicBreakSequence");
        return sequence(code2);
      }
      if (size >= 3 && (code2 === null || markdownLineEnding(code2))) {
        effects.exit("thematicBreak");
        return ok3(code2);
      }
      return nok(code2);
    }
    function sequence(code2) {
      if (code2 === marker) {
        effects.consume(code2);
        size++;
        return sequence;
      }
      effects.exit("thematicBreakSequence");
      return markdownSpace(code2) ? factorySpace(effects, atBreak, "whitespace")(code2) : atBreak(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/list.js
  var list = {
    continuation: {
      tokenize: tokenizeListContinuation
    },
    exit: tokenizeListEnd,
    name: "list",
    tokenize: tokenizeListStart
  };
  var listItemPrefixWhitespaceConstruct = {
    partial: true,
    tokenize: tokenizeListItemPrefixWhitespace
  };
  var indentConstruct = {
    partial: true,
    tokenize: tokenizeIndent
  };
  function tokenizeListStart(effects, ok3, nok) {
    const self2 = this;
    const tail = self2.events[self2.events.length - 1];
    let initialSize = tail && tail[1].type === "linePrefix" ? tail[2].sliceSerialize(tail[1], true).length : 0;
    let size = 0;
    return start;
    function start(code2) {
      const kind = self2.containerState.type || (code2 === 42 || code2 === 43 || code2 === 45 ? "listUnordered" : "listOrdered");
      if (kind === "listUnordered" ? !self2.containerState.marker || code2 === self2.containerState.marker : asciiDigit(code2)) {
        if (!self2.containerState.type) {
          self2.containerState.type = kind;
          effects.enter(kind, {
            _container: true
          });
        }
        if (kind === "listUnordered") {
          effects.enter("listItemPrefix");
          return code2 === 42 || code2 === 45 ? effects.check(thematicBreak, nok, atMarker)(code2) : atMarker(code2);
        }
        if (!self2.interrupt || code2 === 49) {
          effects.enter("listItemPrefix");
          effects.enter("listItemValue");
          return inside(code2);
        }
      }
      return nok(code2);
    }
    function inside(code2) {
      if (asciiDigit(code2) && ++size < 10) {
        effects.consume(code2);
        return inside;
      }
      if ((!self2.interrupt || size < 2) && (self2.containerState.marker ? code2 === self2.containerState.marker : code2 === 41 || code2 === 46)) {
        effects.exit("listItemValue");
        return atMarker(code2);
      }
      return nok(code2);
    }
    function atMarker(code2) {
      effects.enter("listItemMarker");
      effects.consume(code2);
      effects.exit("listItemMarker");
      self2.containerState.marker = self2.containerState.marker || code2;
      return effects.check(
        blankLine,
        // Can’t be empty when interrupting.
        self2.interrupt ? nok : onBlank,
        effects.attempt(listItemPrefixWhitespaceConstruct, endOfPrefix, otherPrefix)
      );
    }
    function onBlank(code2) {
      self2.containerState.initialBlankLine = true;
      initialSize++;
      return endOfPrefix(code2);
    }
    function otherPrefix(code2) {
      if (markdownSpace(code2)) {
        effects.enter("listItemPrefixWhitespace");
        effects.consume(code2);
        effects.exit("listItemPrefixWhitespace");
        return endOfPrefix;
      }
      return nok(code2);
    }
    function endOfPrefix(code2) {
      self2.containerState.size = initialSize + self2.sliceSerialize(effects.exit("listItemPrefix"), true).length;
      return ok3(code2);
    }
  }
  function tokenizeListContinuation(effects, ok3, nok) {
    const self2 = this;
    self2.containerState._closeFlow = void 0;
    return effects.check(blankLine, onBlank, notBlank);
    function onBlank(code2) {
      self2.containerState.furtherBlankLines = self2.containerState.furtherBlankLines || self2.containerState.initialBlankLine;
      return factorySpace(effects, ok3, "listItemIndent", self2.containerState.size + 1)(code2);
    }
    function notBlank(code2) {
      if (self2.containerState.furtherBlankLines || !markdownSpace(code2)) {
        self2.containerState.furtherBlankLines = void 0;
        self2.containerState.initialBlankLine = void 0;
        return notInCurrentItem(code2);
      }
      self2.containerState.furtherBlankLines = void 0;
      self2.containerState.initialBlankLine = void 0;
      return effects.attempt(indentConstruct, ok3, notInCurrentItem)(code2);
    }
    function notInCurrentItem(code2) {
      self2.containerState._closeFlow = true;
      self2.interrupt = void 0;
      return factorySpace(effects, effects.attempt(list, ok3, nok), "linePrefix", self2.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4)(code2);
    }
  }
  function tokenizeIndent(effects, ok3, nok) {
    const self2 = this;
    return factorySpace(effects, afterPrefix, "listItemIndent", self2.containerState.size + 1);
    function afterPrefix(code2) {
      const tail = self2.events[self2.events.length - 1];
      return tail && tail[1].type === "listItemIndent" && tail[2].sliceSerialize(tail[1], true).length === self2.containerState.size ? ok3(code2) : nok(code2);
    }
  }
  function tokenizeListEnd(effects) {
    effects.exit(this.containerState.type);
  }
  function tokenizeListItemPrefixWhitespace(effects, ok3, nok) {
    const self2 = this;
    return factorySpace(effects, afterPrefix, "listItemPrefixWhitespace", self2.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4 + 1);
    function afterPrefix(code2) {
      const tail = self2.events[self2.events.length - 1];
      return !markdownSpace(code2) && tail && tail[1].type === "listItemPrefixWhitespace" ? ok3(code2) : nok(code2);
    }
  }

  // node_modules/micromark-core-commonmark/lib/setext-underline.js
  var setextUnderline = {
    name: "setextUnderline",
    resolveTo: resolveToSetextUnderline,
    tokenize: tokenizeSetextUnderline
  };
  function resolveToSetextUnderline(events, context) {
    let index2 = events.length;
    let content3;
    let text4;
    let definition3;
    while (index2--) {
      if (events[index2][0] === "enter") {
        if (events[index2][1].type === "content") {
          content3 = index2;
          break;
        }
        if (events[index2][1].type === "paragraph") {
          text4 = index2;
        }
      } else {
        if (events[index2][1].type === "content") {
          events.splice(index2, 1);
        }
        if (!definition3 && events[index2][1].type === "definition") {
          definition3 = index2;
        }
      }
    }
    const heading2 = {
      type: "setextHeading",
      start: {
        ...events[content3][1].start
      },
      end: {
        ...events[events.length - 1][1].end
      }
    };
    events[text4][1].type = "setextHeadingText";
    if (definition3) {
      events.splice(text4, 0, ["enter", heading2, context]);
      events.splice(definition3 + 1, 0, ["exit", events[content3][1], context]);
      events[content3][1].end = {
        ...events[definition3][1].end
      };
    } else {
      events[content3][1] = heading2;
    }
    events.push(["exit", heading2, context]);
    return events;
  }
  function tokenizeSetextUnderline(effects, ok3, nok) {
    const self2 = this;
    let marker;
    return start;
    function start(code2) {
      let index2 = self2.events.length;
      let paragraph2;
      while (index2--) {
        if (self2.events[index2][1].type !== "lineEnding" && self2.events[index2][1].type !== "linePrefix" && self2.events[index2][1].type !== "content") {
          paragraph2 = self2.events[index2][1].type === "paragraph";
          break;
        }
      }
      if (!self2.parser.lazy[self2.now().line] && (self2.interrupt || paragraph2)) {
        effects.enter("setextHeadingLine");
        marker = code2;
        return before(code2);
      }
      return nok(code2);
    }
    function before(code2) {
      effects.enter("setextHeadingLineSequence");
      return inside(code2);
    }
    function inside(code2) {
      if (code2 === marker) {
        effects.consume(code2);
        return inside;
      }
      effects.exit("setextHeadingLineSequence");
      return markdownSpace(code2) ? factorySpace(effects, after, "lineSuffix")(code2) : after(code2);
    }
    function after(code2) {
      if (code2 === null || markdownLineEnding(code2)) {
        effects.exit("setextHeadingLine");
        return ok3(code2);
      }
      return nok(code2);
    }
  }

  // node_modules/micromark/lib/initialize/flow.js
  var flow = {
    tokenize: initializeFlow
  };
  function initializeFlow(effects) {
    const self2 = this;
    const initial = effects.attempt(
      // Try to parse a blank line.
      blankLine,
      atBlankEnding,
      // Try to parse initial flow (essentially, only code).
      effects.attempt(this.parser.constructs.flowInitial, afterConstruct, factorySpace(effects, effects.attempt(this.parser.constructs.flow, afterConstruct, effects.attempt(content2, afterConstruct)), "linePrefix"))
    );
    return initial;
    function atBlankEnding(code2) {
      if (code2 === null) {
        effects.consume(code2);
        return;
      }
      effects.enter("lineEndingBlank");
      effects.consume(code2);
      effects.exit("lineEndingBlank");
      self2.currentConstruct = void 0;
      return initial;
    }
    function afterConstruct(code2) {
      if (code2 === null) {
        effects.consume(code2);
        return;
      }
      effects.enter("lineEnding");
      effects.consume(code2);
      effects.exit("lineEnding");
      self2.currentConstruct = void 0;
      return initial;
    }
  }

  // node_modules/micromark/lib/initialize/text.js
  var resolver = {
    resolveAll: createResolver()
  };
  var string = initializeFactory("string");
  var text = initializeFactory("text");
  function initializeFactory(field) {
    return {
      resolveAll: createResolver(field === "text" ? resolveAllLineSuffixes : void 0),
      tokenize: initializeText
    };
    function initializeText(effects) {
      const self2 = this;
      const constructs2 = this.parser.constructs[field];
      const text4 = effects.attempt(constructs2, start, notText);
      return start;
      function start(code2) {
        return atBreak(code2) ? text4(code2) : notText(code2);
      }
      function notText(code2) {
        if (code2 === null) {
          effects.consume(code2);
          return;
        }
        effects.enter("data");
        effects.consume(code2);
        return data;
      }
      function data(code2) {
        if (atBreak(code2)) {
          effects.exit("data");
          return text4(code2);
        }
        effects.consume(code2);
        return data;
      }
      function atBreak(code2) {
        if (code2 === null) {
          return true;
        }
        const list4 = constructs2[code2];
        let index2 = -1;
        if (list4) {
          while (++index2 < list4.length) {
            const item = list4[index2];
            if (!item.previous || item.previous.call(self2, self2.previous)) {
              return true;
            }
          }
        }
        return false;
      }
    }
  }
  function createResolver(extraResolver) {
    return resolveAllText;
    function resolveAllText(events, context) {
      let index2 = -1;
      let enter;
      while (++index2 <= events.length) {
        if (enter === void 0) {
          if (events[index2] && events[index2][1].type === "data") {
            enter = index2;
            index2++;
          }
        } else if (!events[index2] || events[index2][1].type !== "data") {
          if (index2 !== enter + 2) {
            events[enter][1].end = events[index2 - 1][1].end;
            events.splice(enter + 2, index2 - enter - 2);
            index2 = enter + 2;
          }
          enter = void 0;
        }
      }
      return extraResolver ? extraResolver(events, context) : events;
    }
  }
  function resolveAllLineSuffixes(events, context) {
    let eventIndex = 0;
    while (++eventIndex <= events.length) {
      if ((eventIndex === events.length || events[eventIndex][1].type === "lineEnding") && events[eventIndex - 1][1].type === "data") {
        const data = events[eventIndex - 1][1];
        const chunks = context.sliceStream(data);
        let index2 = chunks.length;
        let bufferIndex = -1;
        let size = 0;
        let tabs;
        while (index2--) {
          const chunk = chunks[index2];
          if (typeof chunk === "string") {
            bufferIndex = chunk.length;
            while (chunk.charCodeAt(bufferIndex - 1) === 32) {
              size++;
              bufferIndex--;
            }
            if (bufferIndex) break;
            bufferIndex = -1;
          } else if (chunk === -2) {
            tabs = true;
            size++;
          } else if (chunk === -1) {
          } else {
            index2++;
            break;
          }
        }
        if (context._contentTypeTextTrailing && eventIndex === events.length) {
          size = 0;
        }
        if (size) {
          const token = {
            type: eventIndex === events.length || tabs || size < 2 ? "lineSuffix" : "hardBreakTrailing",
            start: {
              _bufferIndex: index2 ? bufferIndex : data.start._bufferIndex + bufferIndex,
              _index: data.start._index + index2,
              line: data.end.line,
              column: data.end.column - size,
              offset: data.end.offset - size
            },
            end: {
              ...data.end
            }
          };
          data.end = {
            ...token.start
          };
          if (data.start.offset === data.end.offset) {
            Object.assign(data, token);
          } else {
            events.splice(eventIndex, 0, ["enter", token, context], ["exit", token, context]);
            eventIndex += 2;
          }
        }
        eventIndex++;
      }
    }
    return events;
  }

  // node_modules/micromark/lib/constructs.js
  var constructs_exports = {};
  __export(constructs_exports, {
    attentionMarkers: () => attentionMarkers,
    contentInitial: () => contentInitial,
    disable: () => disable,
    document: () => document3,
    flow: () => flow2,
    flowInitial: () => flowInitial,
    insideSpan: () => insideSpan,
    string: () => string2,
    text: () => text2
  });
  var document3 = {
    [42]: list,
    [43]: list,
    [45]: list,
    [48]: list,
    [49]: list,
    [50]: list,
    [51]: list,
    [52]: list,
    [53]: list,
    [54]: list,
    [55]: list,
    [56]: list,
    [57]: list,
    [62]: blockQuote
  };
  var contentInitial = {
    [91]: definition
  };
  var flowInitial = {
    [-2]: codeIndented,
    [-1]: codeIndented,
    [32]: codeIndented
  };
  var flow2 = {
    [35]: headingAtx,
    [42]: thematicBreak,
    [45]: [setextUnderline, thematicBreak],
    [60]: htmlFlow,
    [61]: setextUnderline,
    [95]: thematicBreak,
    [96]: codeFenced,
    [126]: codeFenced
  };
  var string2 = {
    [38]: characterReference,
    [92]: characterEscape
  };
  var text2 = {
    [-5]: lineEnding,
    [-4]: lineEnding,
    [-3]: lineEnding,
    [33]: labelStartImage,
    [38]: characterReference,
    [42]: attention,
    [60]: [autolink, htmlText],
    [91]: labelStartLink,
    [92]: [hardBreakEscape, characterEscape],
    [93]: labelEnd,
    [95]: attention,
    [96]: codeText
  };
  var insideSpan = {
    null: [attention, resolver]
  };
  var attentionMarkers = {
    null: [42, 95]
  };
  var disable = {
    null: []
  };

  // node_modules/micromark/lib/create-tokenizer.js
  function createTokenizer(parser2, initialize, from) {
    let point3 = {
      _bufferIndex: -1,
      _index: 0,
      line: from && from.line || 1,
      column: from && from.column || 1,
      offset: from && from.offset || 0
    };
    const columnStart = {};
    const resolveAllConstructs = [];
    let chunks = [];
    let stack = [];
    let consumed = true;
    const effects = {
      attempt: constructFactory(onsuccessfulconstruct),
      check: constructFactory(onsuccessfulcheck),
      consume,
      enter,
      exit: exit2,
      interrupt: constructFactory(onsuccessfulcheck, {
        interrupt: true
      })
    };
    const context = {
      code: null,
      containerState: {},
      defineSkip,
      events: [],
      now: now2,
      parser: parser2,
      previous: null,
      sliceSerialize,
      sliceStream,
      write
    };
    let state = initialize.tokenize.call(context, effects);
    let expectedCode;
    if (initialize.resolveAll) {
      resolveAllConstructs.push(initialize);
    }
    return context;
    function write(slice) {
      chunks = push(chunks, slice);
      main();
      if (chunks[chunks.length - 1] !== null) {
        return [];
      }
      addResult(initialize, 0);
      context.events = resolveAll(resolveAllConstructs, context.events, context);
      return context.events;
    }
    function sliceSerialize(token, expandTabs) {
      return serializeChunks(sliceStream(token), expandTabs);
    }
    function sliceStream(token) {
      return sliceChunks(chunks, token);
    }
    function now2() {
      const {
        _bufferIndex,
        _index,
        line,
        column,
        offset
      } = point3;
      return {
        _bufferIndex,
        _index,
        line,
        column,
        offset
      };
    }
    function defineSkip(value) {
      columnStart[value.line] = value.column;
      accountForPotentialSkip();
    }
    function main() {
      let chunkIndex;
      while (point3._index < chunks.length) {
        const chunk = chunks[point3._index];
        if (typeof chunk === "string") {
          chunkIndex = point3._index;
          if (point3._bufferIndex < 0) {
            point3._bufferIndex = 0;
          }
          while (point3._index === chunkIndex && point3._bufferIndex < chunk.length) {
            go(chunk.charCodeAt(point3._bufferIndex));
          }
        } else {
          go(chunk);
        }
      }
    }
    function go(code2) {
      consumed = void 0;
      expectedCode = code2;
      state = state(code2);
    }
    function consume(code2) {
      if (markdownLineEnding(code2)) {
        point3.line++;
        point3.column = 1;
        point3.offset += code2 === -3 ? 2 : 1;
        accountForPotentialSkip();
      } else if (code2 !== -1) {
        point3.column++;
        point3.offset++;
      }
      if (point3._bufferIndex < 0) {
        point3._index++;
      } else {
        point3._bufferIndex++;
        if (point3._bufferIndex === // Points w/ non-negative `_bufferIndex` reference
        // strings.
        /** @type {string} */
        chunks[point3._index].length) {
          point3._bufferIndex = -1;
          point3._index++;
        }
      }
      context.previous = code2;
      consumed = true;
    }
    function enter(type, fields) {
      const token = fields || {};
      token.type = type;
      token.start = now2();
      context.events.push(["enter", token, context]);
      stack.push(token);
      return token;
    }
    function exit2(type) {
      const token = stack.pop();
      token.end = now2();
      context.events.push(["exit", token, context]);
      return token;
    }
    function onsuccessfulconstruct(construct, info) {
      addResult(construct, info.from);
    }
    function onsuccessfulcheck(_, info) {
      info.restore();
    }
    function constructFactory(onreturn, fields) {
      return hook;
      function hook(constructs2, returnState, bogusState) {
        let listOfConstructs;
        let constructIndex;
        let currentConstruct;
        let info;
        return Array.isArray(constructs2) ? (
          /* c8 ignore next 1 */
          handleListOfConstructs(constructs2)
        ) : "tokenize" in constructs2 ? (
          // Looks like a construct.
          handleListOfConstructs([
            /** @type {Construct} */
            constructs2
          ])
        ) : handleMapOfConstructs(constructs2);
        function handleMapOfConstructs(map4) {
          return start;
          function start(code2) {
            const left = code2 !== null && map4[code2];
            const all2 = code2 !== null && map4.null;
            const list4 = [
              // To do: add more extension tests.
              /* c8 ignore next 2 */
              ...Array.isArray(left) ? left : left ? [left] : [],
              ...Array.isArray(all2) ? all2 : all2 ? [all2] : []
            ];
            return handleListOfConstructs(list4)(code2);
          }
        }
        function handleListOfConstructs(list4) {
          listOfConstructs = list4;
          constructIndex = 0;
          if (list4.length === 0) {
            return bogusState;
          }
          return handleConstruct(list4[constructIndex]);
        }
        function handleConstruct(construct) {
          return start;
          function start(code2) {
            info = store();
            currentConstruct = construct;
            if (!construct.partial) {
              context.currentConstruct = construct;
            }
            if (construct.name && context.parser.constructs.disable.null.includes(construct.name)) {
              return nok(code2);
            }
            return construct.tokenize.call(
              // If we do have fields, create an object w/ `context` as its
              // prototype.
              // This allows a “live binding”, which is needed for `interrupt`.
              fields ? Object.assign(Object.create(context), fields) : context,
              effects,
              ok3,
              nok
            )(code2);
          }
        }
        function ok3(code2) {
          consumed = true;
          onreturn(currentConstruct, info);
          return returnState;
        }
        function nok(code2) {
          consumed = true;
          info.restore();
          if (++constructIndex < listOfConstructs.length) {
            return handleConstruct(listOfConstructs[constructIndex]);
          }
          return bogusState;
        }
      }
    }
    function addResult(construct, from2) {
      if (construct.resolveAll && !resolveAllConstructs.includes(construct)) {
        resolveAllConstructs.push(construct);
      }
      if (construct.resolve) {
        splice(context.events, from2, context.events.length - from2, construct.resolve(context.events.slice(from2), context));
      }
      if (construct.resolveTo) {
        context.events = construct.resolveTo(context.events, context);
      }
    }
    function store() {
      const startPoint = now2();
      const startPrevious = context.previous;
      const startCurrentConstruct = context.currentConstruct;
      const startEventsIndex = context.events.length;
      const startStack = Array.from(stack);
      return {
        from: startEventsIndex,
        restore
      };
      function restore() {
        point3 = startPoint;
        context.previous = startPrevious;
        context.currentConstruct = startCurrentConstruct;
        context.events.length = startEventsIndex;
        stack = startStack;
        accountForPotentialSkip();
      }
    }
    function accountForPotentialSkip() {
      if (point3.line in columnStart && point3.column < 2) {
        point3.column = columnStart[point3.line];
        point3.offset += columnStart[point3.line] - 1;
      }
    }
  }
  function sliceChunks(chunks, token) {
    const startIndex = token.start._index;
    const startBufferIndex = token.start._bufferIndex;
    const endIndex = token.end._index;
    const endBufferIndex = token.end._bufferIndex;
    let view;
    if (startIndex === endIndex) {
      view = [chunks[startIndex].slice(startBufferIndex, endBufferIndex)];
    } else {
      view = chunks.slice(startIndex, endIndex);
      if (startBufferIndex > -1) {
        const head = view[0];
        if (typeof head === "string") {
          view[0] = head.slice(startBufferIndex);
        } else {
          view.shift();
        }
      }
      if (endBufferIndex > 0) {
        view.push(chunks[endIndex].slice(0, endBufferIndex));
      }
    }
    return view;
  }
  function serializeChunks(chunks, expandTabs) {
    let index2 = -1;
    const result = [];
    let atTab;
    while (++index2 < chunks.length) {
      const chunk = chunks[index2];
      let value;
      if (typeof chunk === "string") {
        value = chunk;
      } else switch (chunk) {
        case -5: {
          value = "\r";
          break;
        }
        case -4: {
          value = "\n";
          break;
        }
        case -3: {
          value = "\r\n";
          break;
        }
        case -2: {
          value = expandTabs ? " " : "	";
          break;
        }
        case -1: {
          if (!expandTabs && atTab) continue;
          value = " ";
          break;
        }
        default: {
          value = String.fromCharCode(chunk);
        }
      }
      atTab = chunk === -2;
      result.push(value);
    }
    return result.join("");
  }

  // node_modules/micromark/lib/parse.js
  function parse(options) {
    const settings = options || {};
    const constructs2 = (
      /** @type {FullNormalizedExtension} */
      combineExtensions([constructs_exports, ...settings.extensions || []])
    );
    const parser2 = {
      constructs: constructs2,
      content: create(content),
      defined: [],
      document: create(document2),
      flow: create(flow),
      lazy: {},
      string: create(string),
      text: create(text)
    };
    return parser2;
    function create(initial) {
      return creator;
      function creator(from) {
        return createTokenizer(parser2, initial, from);
      }
    }
  }

  // node_modules/micromark/lib/postprocess.js
  function postprocess(events) {
    while (!subtokenize(events)) {
    }
    return events;
  }

  // node_modules/micromark/lib/preprocess.js
  var search = /[\0\t\n\r]/g;
  function preprocess() {
    let column = 1;
    let buffer = "";
    let start = true;
    let atCarriageReturn;
    return preprocessor;
    function preprocessor(value, encoding, end) {
      const chunks = [];
      let match;
      let next;
      let startPosition;
      let endPosition;
      let code2;
      value = buffer + (typeof value === "string" ? value.toString() : new TextDecoder(encoding || void 0).decode(value));
      startPosition = 0;
      buffer = "";
      if (start) {
        if (value.charCodeAt(0) === 65279) {
          startPosition++;
        }
        start = void 0;
      }
      while (startPosition < value.length) {
        search.lastIndex = startPosition;
        match = search.exec(value);
        endPosition = match && match.index !== void 0 ? match.index : value.length;
        code2 = value.charCodeAt(endPosition);
        if (!match) {
          buffer = value.slice(startPosition);
          break;
        }
        if (code2 === 10 && startPosition === endPosition && atCarriageReturn) {
          chunks.push(-3);
          atCarriageReturn = void 0;
        } else {
          if (atCarriageReturn) {
            chunks.push(-5);
            atCarriageReturn = void 0;
          }
          if (startPosition < endPosition) {
            chunks.push(value.slice(startPosition, endPosition));
            column += endPosition - startPosition;
          }
          switch (code2) {
            case 0: {
              chunks.push(65533);
              column++;
              break;
            }
            case 9: {
              next = Math.ceil(column / 4) * 4;
              chunks.push(-2);
              while (column++ < next) chunks.push(-1);
              break;
            }
            case 10: {
              chunks.push(-4);
              column = 1;
              break;
            }
            default: {
              atCarriageReturn = true;
              column = 1;
            }
          }
        }
        startPosition = endPosition + 1;
      }
      if (end) {
        if (atCarriageReturn) chunks.push(-5);
        if (buffer) chunks.push(buffer);
        chunks.push(null);
      }
      return chunks;
    }
  }

  // node_modules/micromark-util-decode-string/index.js
  var characterEscapeOrReference = /\\([!-/:-@[-`{-~])|&(#(?:\d{1,7}|x[\da-f]{1,6})|[\da-z]{1,31});/gi;
  function decodeString(value) {
    return value.replace(characterEscapeOrReference, decode);
  }
  function decode($0, $1, $2) {
    if ($1) {
      return $1;
    }
    const head = $2.charCodeAt(0);
    if (head === 35) {
      const head2 = $2.charCodeAt(1);
      const hex = head2 === 120 || head2 === 88;
      return decodeNumericCharacterReference($2.slice(hex ? 2 : 1), hex ? 16 : 10);
    }
    return decodeNamedCharacterReference($2) || $0;
  }

  // node_modules/unist-util-stringify-position/lib/index.js
  function stringifyPosition(value) {
    if (!value || typeof value !== "object") {
      return "";
    }
    if ("position" in value || "type" in value) {
      return position(value.position);
    }
    if ("start" in value || "end" in value) {
      return position(value);
    }
    if ("line" in value || "column" in value) {
      return point(value);
    }
    return "";
  }
  function point(point3) {
    return index(point3 && point3.line) + ":" + index(point3 && point3.column);
  }
  function position(pos) {
    return point(pos && pos.start) + "-" + point(pos && pos.end);
  }
  function index(value) {
    return value && typeof value === "number" ? value : 1;
  }

  // node_modules/mdast-util-from-markdown/lib/index.js
  var own = {}.hasOwnProperty;
  function fromMarkdown(value, encoding, options) {
    if (typeof encoding !== "string") {
      options = encoding;
      encoding = void 0;
    }
    return compiler(options)(postprocess(parse(options).document().write(preprocess()(value, encoding, true))));
  }
  function compiler(options) {
    const config2 = {
      transforms: [],
      canContainEols: ["emphasis", "fragment", "heading", "paragraph", "strong"],
      enter: {
        autolink: opener(link2),
        autolinkProtocol: onenterdata,
        autolinkEmail: onenterdata,
        atxHeading: opener(heading2),
        blockQuote: opener(blockQuote2),
        characterEscape: onenterdata,
        characterReference: onenterdata,
        codeFenced: opener(codeFlow),
        codeFencedFenceInfo: buffer,
        codeFencedFenceMeta: buffer,
        codeIndented: opener(codeFlow, buffer),
        codeText: opener(codeText2, buffer),
        codeTextData: onenterdata,
        data: onenterdata,
        codeFlowValue: onenterdata,
        definition: opener(definition3),
        definitionDestinationString: buffer,
        definitionLabelString: buffer,
        definitionTitleString: buffer,
        emphasis: opener(emphasis2),
        hardBreakEscape: opener(hardBreak2),
        hardBreakTrailing: opener(hardBreak2),
        htmlFlow: opener(html2, buffer),
        htmlFlowData: onenterdata,
        htmlText: opener(html2, buffer),
        htmlTextData: onenterdata,
        image: opener(image2),
        label: buffer,
        link: opener(link2),
        listItem: opener(listItem2),
        listItemValue: onenterlistitemvalue,
        listOrdered: opener(list4, onenterlistordered),
        listUnordered: opener(list4),
        paragraph: opener(paragraph2),
        reference: onenterreference,
        referenceString: buffer,
        resourceDestinationString: buffer,
        resourceTitleString: buffer,
        setextHeading: opener(heading2),
        strong: opener(strong2),
        thematicBreak: opener(thematicBreak3)
      },
      exit: {
        atxHeading: closer(),
        atxHeadingSequence: onexitatxheadingsequence,
        autolink: closer(),
        autolinkEmail: onexitautolinkemail,
        autolinkProtocol: onexitautolinkprotocol,
        blockQuote: closer(),
        characterEscapeValue: onexitdata,
        characterReferenceMarkerHexadecimal: onexitcharacterreferencemarker,
        characterReferenceMarkerNumeric: onexitcharacterreferencemarker,
        characterReferenceValue: onexitcharacterreferencevalue,
        characterReference: onexitcharacterreference,
        codeFenced: closer(onexitcodefenced),
        codeFencedFence: onexitcodefencedfence,
        codeFencedFenceInfo: onexitcodefencedfenceinfo,
        codeFencedFenceMeta: onexitcodefencedfencemeta,
        codeFlowValue: onexitdata,
        codeIndented: closer(onexitcodeindented),
        codeText: closer(onexitcodetext),
        codeTextData: onexitdata,
        data: onexitdata,
        definition: closer(),
        definitionDestinationString: onexitdefinitiondestinationstring,
        definitionLabelString: onexitdefinitionlabelstring,
        definitionTitleString: onexitdefinitiontitlestring,
        emphasis: closer(),
        hardBreakEscape: closer(onexithardbreak),
        hardBreakTrailing: closer(onexithardbreak),
        htmlFlow: closer(onexithtmlflow),
        htmlFlowData: onexitdata,
        htmlText: closer(onexithtmltext),
        htmlTextData: onexitdata,
        image: closer(onexitimage),
        label: onexitlabel,
        labelText: onexitlabeltext,
        lineEnding: onexitlineending,
        link: closer(onexitlink),
        listItem: closer(),
        listOrdered: closer(),
        listUnordered: closer(),
        paragraph: closer(),
        referenceString: onexitreferencestring,
        resourceDestinationString: onexitresourcedestinationstring,
        resourceTitleString: onexitresourcetitlestring,
        resource: onexitresource,
        setextHeading: closer(onexitsetextheading),
        setextHeadingLineSequence: onexitsetextheadinglinesequence,
        setextHeadingText: onexitsetextheadingtext,
        strong: closer(),
        thematicBreak: closer()
      }
    };
    configure(config2, (options || {}).mdastExtensions || []);
    const data = {};
    return compile;
    function compile(events) {
      let tree = {
        type: "root",
        children: []
      };
      const context = {
        stack: [tree],
        tokenStack: [],
        config: config2,
        enter,
        exit: exit2,
        buffer,
        resume,
        data
      };
      const listStack = [];
      let index2 = -1;
      while (++index2 < events.length) {
        if (events[index2][1].type === "listOrdered" || events[index2][1].type === "listUnordered") {
          if (events[index2][0] === "enter") {
            listStack.push(index2);
          } else {
            const tail = listStack.pop();
            index2 = prepareList(events, tail, index2);
          }
        }
      }
      index2 = -1;
      while (++index2 < events.length) {
        const handler = config2[events[index2][0]];
        if (own.call(handler, events[index2][1].type)) {
          handler[events[index2][1].type].call(Object.assign({
            sliceSerialize: events[index2][2].sliceSerialize
          }, context), events[index2][1]);
        }
      }
      if (context.tokenStack.length > 0) {
        const tail = context.tokenStack[context.tokenStack.length - 1];
        const handler = tail[1] || defaultOnError;
        handler.call(context, void 0, tail[0]);
      }
      tree.position = {
        start: point2(events.length > 0 ? events[0][1].start : {
          line: 1,
          column: 1,
          offset: 0
        }),
        end: point2(events.length > 0 ? events[events.length - 2][1].end : {
          line: 1,
          column: 1,
          offset: 0
        })
      };
      index2 = -1;
      while (++index2 < config2.transforms.length) {
        tree = config2.transforms[index2](tree) || tree;
      }
      return tree;
    }
    function prepareList(events, start, length) {
      let index2 = start - 1;
      let containerBalance = -1;
      let listSpread = false;
      let listItem3;
      let lineIndex;
      let firstBlankLineIndex;
      let atMarker;
      while (++index2 <= length) {
        const event = events[index2];
        switch (event[1].type) {
          case "listUnordered":
          case "listOrdered":
          case "blockQuote": {
            if (event[0] === "enter") {
              containerBalance++;
            } else {
              containerBalance--;
            }
            atMarker = void 0;
            break;
          }
          case "lineEndingBlank": {
            if (event[0] === "enter") {
              if (listItem3 && !atMarker && !containerBalance && !firstBlankLineIndex) {
                firstBlankLineIndex = index2;
              }
              atMarker = void 0;
            }
            break;
          }
          case "linePrefix":
          case "listItemValue":
          case "listItemMarker":
          case "listItemPrefix":
          case "listItemPrefixWhitespace": {
            break;
          }
          default: {
            atMarker = void 0;
          }
        }
        if (!containerBalance && event[0] === "enter" && event[1].type === "listItemPrefix" || containerBalance === -1 && event[0] === "exit" && (event[1].type === "listUnordered" || event[1].type === "listOrdered")) {
          if (listItem3) {
            let tailIndex = index2;
            lineIndex = void 0;
            while (tailIndex--) {
              const tailEvent = events[tailIndex];
              if (tailEvent[1].type === "lineEnding" || tailEvent[1].type === "lineEndingBlank") {
                if (tailEvent[0] === "exit") continue;
                if (lineIndex) {
                  events[lineIndex][1].type = "lineEndingBlank";
                  listSpread = true;
                }
                tailEvent[1].type = "lineEnding";
                lineIndex = tailIndex;
              } else if (tailEvent[1].type === "linePrefix" || tailEvent[1].type === "blockQuotePrefix" || tailEvent[1].type === "blockQuotePrefixWhitespace" || tailEvent[1].type === "blockQuoteMarker" || tailEvent[1].type === "listItemIndent") {
              } else {
                break;
              }
            }
            if (firstBlankLineIndex && (!lineIndex || firstBlankLineIndex < lineIndex)) {
              listItem3._spread = true;
            }
            listItem3.end = Object.assign({}, lineIndex ? events[lineIndex][1].start : event[1].end);
            events.splice(lineIndex || index2, 0, ["exit", listItem3, event[2]]);
            index2++;
            length++;
          }
          if (event[1].type === "listItemPrefix") {
            const item = {
              type: "listItem",
              _spread: false,
              start: Object.assign({}, event[1].start),
              // @ts-expect-error: we’ll add `end` in a second.
              end: void 0
            };
            listItem3 = item;
            events.splice(index2, 0, ["enter", item, event[2]]);
            index2++;
            length++;
            firstBlankLineIndex = void 0;
            atMarker = true;
          }
        }
      }
      events[start][1]._spread = listSpread;
      return length;
    }
    function opener(create, and) {
      return open;
      function open(token) {
        enter.call(this, create(token), token);
        if (and) and.call(this, token);
      }
    }
    function buffer() {
      this.stack.push({
        type: "fragment",
        children: []
      });
    }
    function enter(node2, token, errorHandler) {
      const parent = this.stack[this.stack.length - 1];
      const siblings = parent.children;
      siblings.push(node2);
      this.stack.push(node2);
      this.tokenStack.push([token, errorHandler || void 0]);
      node2.position = {
        start: point2(token.start),
        // @ts-expect-error: `end` will be patched later.
        end: void 0
      };
    }
    function closer(and) {
      return close2;
      function close2(token) {
        if (and) and.call(this, token);
        exit2.call(this, token);
      }
    }
    function exit2(token, onExitError) {
      const node2 = this.stack.pop();
      const open = this.tokenStack.pop();
      if (!open) {
        throw new Error("Cannot close `" + token.type + "` (" + stringifyPosition({
          start: token.start,
          end: token.end
        }) + "): it\u2019s not open");
      } else if (open[0].type !== token.type) {
        if (onExitError) {
          onExitError.call(this, token, open[0]);
        } else {
          const handler = open[1] || defaultOnError;
          handler.call(this, token, open[0]);
        }
      }
      node2.position.end = point2(token.end);
    }
    function resume() {
      return toString(this.stack.pop());
    }
    function onenterlistordered() {
      this.data.expectingFirstListItemValue = true;
    }
    function onenterlistitemvalue(token) {
      if (this.data.expectingFirstListItemValue) {
        const ancestor = this.stack[this.stack.length - 2];
        ancestor.start = Number.parseInt(this.sliceSerialize(token), 10);
        this.data.expectingFirstListItemValue = void 0;
      }
    }
    function onexitcodefencedfenceinfo() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.lang = data2;
    }
    function onexitcodefencedfencemeta() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.meta = data2;
    }
    function onexitcodefencedfence() {
      if (this.data.flowCodeInside) return;
      this.buffer();
      this.data.flowCodeInside = true;
    }
    function onexitcodefenced() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.value = data2.replace(/^(\r?\n|\r)|(\r?\n|\r)$/g, "");
      this.data.flowCodeInside = void 0;
    }
    function onexitcodeindented() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.value = data2.replace(/(\r?\n|\r)$/g, "");
    }
    function onexitdefinitionlabelstring(token) {
      const label = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.label = label;
      node2.identifier = normalizeIdentifier(this.sliceSerialize(token)).toLowerCase();
    }
    function onexitdefinitiontitlestring() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.title = data2;
    }
    function onexitdefinitiondestinationstring() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.url = data2;
    }
    function onexitatxheadingsequence(token) {
      const node2 = this.stack[this.stack.length - 1];
      if (!node2.depth) {
        const depth = this.sliceSerialize(token).length;
        node2.depth = depth;
      }
    }
    function onexitsetextheadingtext() {
      this.data.setextHeadingSlurpLineEnding = true;
    }
    function onexitsetextheadinglinesequence(token) {
      const node2 = this.stack[this.stack.length - 1];
      node2.depth = this.sliceSerialize(token).codePointAt(0) === 61 ? 1 : 2;
    }
    function onexitsetextheading() {
      this.data.setextHeadingSlurpLineEnding = void 0;
    }
    function onenterdata(token) {
      const node2 = this.stack[this.stack.length - 1];
      const siblings = node2.children;
      let tail = siblings[siblings.length - 1];
      if (!tail || tail.type !== "text") {
        tail = text4();
        tail.position = {
          start: point2(token.start),
          // @ts-expect-error: we’ll add `end` later.
          end: void 0
        };
        siblings.push(tail);
      }
      this.stack.push(tail);
    }
    function onexitdata(token) {
      const tail = this.stack.pop();
      tail.value += this.sliceSerialize(token);
      tail.position.end = point2(token.end);
    }
    function onexitlineending(token) {
      const context = this.stack[this.stack.length - 1];
      if (this.data.atHardBreak) {
        const tail = context.children[context.children.length - 1];
        tail.position.end = point2(token.end);
        this.data.atHardBreak = void 0;
        return;
      }
      if (!this.data.setextHeadingSlurpLineEnding && config2.canContainEols.includes(context.type)) {
        onenterdata.call(this, token);
        onexitdata.call(this, token);
      }
    }
    function onexithardbreak() {
      this.data.atHardBreak = true;
    }
    function onexithtmlflow() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.value = data2;
    }
    function onexithtmltext() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.value = data2;
    }
    function onexitcodetext() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.value = data2;
    }
    function onexitlink() {
      const node2 = this.stack[this.stack.length - 1];
      if (this.data.inReference) {
        const referenceType = this.data.referenceType || "shortcut";
        node2.type += "Reference";
        node2.referenceType = referenceType;
        delete node2.url;
        delete node2.title;
      } else {
        delete node2.identifier;
        delete node2.label;
      }
      this.data.referenceType = void 0;
    }
    function onexitimage() {
      const node2 = this.stack[this.stack.length - 1];
      if (this.data.inReference) {
        const referenceType = this.data.referenceType || "shortcut";
        node2.type += "Reference";
        node2.referenceType = referenceType;
        delete node2.url;
        delete node2.title;
      } else {
        delete node2.identifier;
        delete node2.label;
      }
      this.data.referenceType = void 0;
    }
    function onexitlabeltext(token) {
      const string3 = this.sliceSerialize(token);
      const ancestor = this.stack[this.stack.length - 2];
      ancestor.label = decodeString(string3);
      ancestor.identifier = normalizeIdentifier(string3).toLowerCase();
    }
    function onexitlabel() {
      const fragment = this.stack[this.stack.length - 1];
      const value = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      this.data.inReference = true;
      if (node2.type === "link") {
        const children = fragment.children;
        node2.children = children;
      } else {
        node2.alt = value;
      }
    }
    function onexitresourcedestinationstring() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.url = data2;
    }
    function onexitresourcetitlestring() {
      const data2 = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.title = data2;
    }
    function onexitresource() {
      this.data.inReference = void 0;
    }
    function onenterreference() {
      this.data.referenceType = "collapsed";
    }
    function onexitreferencestring(token) {
      const label = this.resume();
      const node2 = this.stack[this.stack.length - 1];
      node2.label = label;
      node2.identifier = normalizeIdentifier(this.sliceSerialize(token)).toLowerCase();
      this.data.referenceType = "full";
    }
    function onexitcharacterreferencemarker(token) {
      this.data.characterReferenceType = token.type;
    }
    function onexitcharacterreferencevalue(token) {
      const data2 = this.sliceSerialize(token);
      const type = this.data.characterReferenceType;
      let value;
      if (type) {
        value = decodeNumericCharacterReference(data2, type === "characterReferenceMarkerNumeric" ? 10 : 16);
        this.data.characterReferenceType = void 0;
      } else {
        const result = decodeNamedCharacterReference(data2);
        value = result;
      }
      const tail = this.stack[this.stack.length - 1];
      tail.value += value;
    }
    function onexitcharacterreference(token) {
      const tail = this.stack.pop();
      tail.position.end = point2(token.end);
    }
    function onexitautolinkprotocol(token) {
      onexitdata.call(this, token);
      const node2 = this.stack[this.stack.length - 1];
      node2.url = this.sliceSerialize(token);
    }
    function onexitautolinkemail(token) {
      onexitdata.call(this, token);
      const node2 = this.stack[this.stack.length - 1];
      node2.url = "mailto:" + this.sliceSerialize(token);
    }
    function blockQuote2() {
      return {
        type: "blockquote",
        children: []
      };
    }
    function codeFlow() {
      return {
        type: "code",
        lang: null,
        meta: null,
        value: ""
      };
    }
    function codeText2() {
      return {
        type: "inlineCode",
        value: ""
      };
    }
    function definition3() {
      return {
        type: "definition",
        identifier: "",
        label: null,
        title: null,
        url: ""
      };
    }
    function emphasis2() {
      return {
        type: "emphasis",
        children: []
      };
    }
    function heading2() {
      return {
        type: "heading",
        // @ts-expect-error `depth` will be set later.
        depth: 0,
        children: []
      };
    }
    function hardBreak2() {
      return {
        type: "break"
      };
    }
    function html2() {
      return {
        type: "html",
        value: ""
      };
    }
    function image2() {
      return {
        type: "image",
        title: null,
        url: "",
        alt: null
      };
    }
    function link2() {
      return {
        type: "link",
        title: null,
        url: "",
        children: []
      };
    }
    function list4(token) {
      return {
        type: "list",
        ordered: token.type === "listOrdered",
        start: null,
        spread: token._spread,
        children: []
      };
    }
    function listItem2(token) {
      return {
        type: "listItem",
        spread: token._spread,
        checked: null,
        children: []
      };
    }
    function paragraph2() {
      return {
        type: "paragraph",
        children: []
      };
    }
    function strong2() {
      return {
        type: "strong",
        children: []
      };
    }
    function text4() {
      return {
        type: "text",
        value: ""
      };
    }
    function thematicBreak3() {
      return {
        type: "thematicBreak"
      };
    }
  }
  function point2(d2) {
    return {
      line: d2.line,
      column: d2.column,
      offset: d2.offset
    };
  }
  function configure(combined, extensions) {
    let index2 = -1;
    while (++index2 < extensions.length) {
      const value = extensions[index2];
      if (Array.isArray(value)) {
        configure(combined, value);
      } else {
        extension(combined, value);
      }
    }
  }
  function extension(combined, extension2) {
    let key3;
    for (key3 in extension2) {
      if (own.call(extension2, key3)) {
        switch (key3) {
          case "canContainEols": {
            const right = extension2[key3];
            if (right) {
              combined[key3].push(...right);
            }
            break;
          }
          case "transforms": {
            const right = extension2[key3];
            if (right) {
              combined[key3].push(...right);
            }
            break;
          }
          case "enter":
          case "exit": {
            const right = extension2[key3];
            if (right) {
              Object.assign(combined[key3], right);
            }
            break;
          }
        }
      }
    }
  }
  function defaultOnError(left, right) {
    if (left) {
      throw new Error("Cannot close `" + left.type + "` (" + stringifyPosition({
        start: left.start,
        end: left.end
      }) + "): a different token (`" + right.type + "`, " + stringifyPosition({
        start: right.start,
        end: right.end
      }) + ") is open");
    } else {
      throw new Error("Cannot close document, a token (`" + right.type + "`, " + stringifyPosition({
        start: right.start,
        end: right.end
      }) + ") is still open");
    }
  }

  // node_modules/remark-parse/lib/index.js
  function remarkParse(options) {
    const self2 = this;
    self2.parser = parser2;
    function parser2(doc4) {
      return fromMarkdown(doc4, {
        ...self2.data("settings"),
        ...options,
        // Note: these options are not in the readme.
        // The goal is for them to be set by plugins on `data` instead of being
        // passed by users.
        extensions: self2.data("micromarkExtensions") || [],
        mdastExtensions: self2.data("fromMarkdownExtensions") || []
      });
    }
  }

  // node_modules/zwitch/index.js
  var own2 = {}.hasOwnProperty;
  function zwitch(key3, options) {
    const settings = options || {};
    function one2(value, ...parameters) {
      let fn = one2.invalid;
      const handlers2 = one2.handlers;
      if (value && own2.call(value, key3)) {
        const id = String(value[key3]);
        fn = own2.call(handlers2, id) ? handlers2[id] : one2.unknown;
      }
      if (fn) {
        return fn.call(this, value, ...parameters);
      }
    }
    one2.handlers = settings.handlers || {};
    one2.invalid = settings.invalid;
    one2.unknown = settings.unknown;
    return one2;
  }

  // node_modules/mdast-util-to-markdown/lib/configure.js
  var own3 = {}.hasOwnProperty;
  function configure2(base2, extension2) {
    let index2 = -1;
    let key3;
    if (extension2.extensions) {
      while (++index2 < extension2.extensions.length) {
        configure2(base2, extension2.extensions[index2]);
      }
    }
    for (key3 in extension2) {
      if (own3.call(extension2, key3)) {
        switch (key3) {
          case "extensions": {
            break;
          }
          /* c8 ignore next 4 */
          case "unsafe": {
            list2(base2[key3], extension2[key3]);
            break;
          }
          case "join": {
            list2(base2[key3], extension2[key3]);
            break;
          }
          case "handlers": {
            map(base2[key3], extension2[key3]);
            break;
          }
          default: {
            base2.options[key3] = extension2[key3];
          }
        }
      }
    }
    return base2;
  }
  function list2(left, right) {
    if (right) {
      left.push(...right);
    }
  }
  function map(left, right) {
    if (right) {
      Object.assign(left, right);
    }
  }

  // node_modules/mdast-util-to-markdown/lib/handle/blockquote.js
  function blockquote(node2, _, state, info) {
    const exit2 = state.enter("blockquote");
    const tracker = state.createTracker(info);
    tracker.move("> ");
    tracker.shift(2);
    const value = state.indentLines(
      state.containerFlow(node2, tracker.current()),
      map2
    );
    exit2();
    return value;
  }
  function map2(line, _, blank) {
    return ">" + (blank ? "" : " ") + line;
  }

  // node_modules/mdast-util-to-markdown/lib/util/pattern-in-scope.js
  function patternInScope(stack, pattern) {
    return listInScope(stack, pattern.inConstruct, true) && !listInScope(stack, pattern.notInConstruct, false);
  }
  function listInScope(stack, list4, none2) {
    if (typeof list4 === "string") {
      list4 = [list4];
    }
    if (!list4 || list4.length === 0) {
      return none2;
    }
    let index2 = -1;
    while (++index2 < list4.length) {
      if (stack.includes(list4[index2])) {
        return true;
      }
    }
    return false;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/break.js
  function hardBreak(_, _1, state, info) {
    let index2 = -1;
    while (++index2 < state.unsafe.length) {
      if (state.unsafe[index2].character === "\n" && patternInScope(state.stack, state.unsafe[index2])) {
        return /[ \t]/.test(info.before) ? "" : " ";
      }
    }
    return "\\\n";
  }

  // node_modules/longest-streak/index.js
  function longestStreak(value, substring) {
    const source = String(value);
    let index2 = source.indexOf(substring);
    let expected = index2;
    let count = 0;
    let max = 0;
    if (typeof substring !== "string") {
      throw new TypeError("Expected substring");
    }
    while (index2 !== -1) {
      if (index2 === expected) {
        if (++count > max) {
          max = count;
        }
      } else {
        count = 1;
      }
      expected = index2 + substring.length;
      index2 = source.indexOf(substring, expected);
    }
    return max;
  }

  // node_modules/mdast-util-to-markdown/lib/util/format-code-as-indented.js
  function formatCodeAsIndented(node2, state) {
    return Boolean(
      state.options.fences === false && node2.value && // If there’s no info…
      !node2.lang && // And there’s a non-whitespace character…
      /[^ \r\n]/.test(node2.value) && // And the value doesn’t start or end in a blank…
      !/^[\t ]*(?:[\r\n]|$)|(?:^|[\r\n])[\t ]*$/.test(node2.value)
    );
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-fence.js
  function checkFence(state) {
    const marker = state.options.fence || "`";
    if (marker !== "`" && marker !== "~") {
      throw new Error(
        "Cannot serialize code with `" + marker + "` for `options.fence`, expected `` ` `` or `~`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/code.js
  function code(node2, _, state, info) {
    const marker = checkFence(state);
    const raw = node2.value || "";
    const suffix = marker === "`" ? "GraveAccent" : "Tilde";
    if (formatCodeAsIndented(node2, state)) {
      const exit3 = state.enter("codeIndented");
      const value2 = state.indentLines(raw, map3);
      exit3();
      return value2;
    }
    const tracker = state.createTracker(info);
    const sequence = marker.repeat(Math.max(longestStreak(raw, marker) + 1, 3));
    const exit2 = state.enter("codeFenced");
    let value = tracker.move(sequence);
    if (node2.lang) {
      const subexit = state.enter(`codeFencedLang${suffix}`);
      value += tracker.move(
        state.safe(node2.lang, {
          before: value,
          after: " ",
          encode: ["`"],
          ...tracker.current()
        })
      );
      subexit();
    }
    if (node2.lang && node2.meta) {
      const subexit = state.enter(`codeFencedMeta${suffix}`);
      value += tracker.move(" ");
      value += tracker.move(
        state.safe(node2.meta, {
          before: value,
          after: "\n",
          encode: ["`"],
          ...tracker.current()
        })
      );
      subexit();
    }
    value += tracker.move("\n");
    if (raw) {
      value += tracker.move(raw + "\n");
    }
    value += tracker.move(sequence);
    exit2();
    return value;
  }
  function map3(line, _, blank) {
    return (blank ? "" : "    ") + line;
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-quote.js
  function checkQuote(state) {
    const marker = state.options.quote || '"';
    if (marker !== '"' && marker !== "'") {
      throw new Error(
        "Cannot serialize title with `" + marker + "` for `options.quote`, expected `\"`, or `'`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/definition.js
  function definition2(node2, _, state, info) {
    const quote = checkQuote(state);
    const suffix = quote === '"' ? "Quote" : "Apostrophe";
    const exit2 = state.enter("definition");
    let subexit = state.enter("label");
    const tracker = state.createTracker(info);
    let value = tracker.move("[");
    value += tracker.move(
      state.safe(state.associationId(node2), {
        before: value,
        after: "]",
        ...tracker.current()
      })
    );
    value += tracker.move("]: ");
    subexit();
    if (
      // If there’s no url, or…
      !node2.url || // If there are control characters or whitespace.
      /[\0- \u007F]/.test(node2.url)
    ) {
      subexit = state.enter("destinationLiteral");
      value += tracker.move("<");
      value += tracker.move(
        state.safe(node2.url, { before: value, after: ">", ...tracker.current() })
      );
      value += tracker.move(">");
    } else {
      subexit = state.enter("destinationRaw");
      value += tracker.move(
        state.safe(node2.url, {
          before: value,
          after: node2.title ? " " : "\n",
          ...tracker.current()
        })
      );
    }
    subexit();
    if (node2.title) {
      subexit = state.enter(`title${suffix}`);
      value += tracker.move(" " + quote);
      value += tracker.move(
        state.safe(node2.title, {
          before: value,
          after: quote,
          ...tracker.current()
        })
      );
      value += tracker.move(quote);
      subexit();
    }
    exit2();
    return value;
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-emphasis.js
  function checkEmphasis(state) {
    const marker = state.options.emphasis || "*";
    if (marker !== "*" && marker !== "_") {
      throw new Error(
        "Cannot serialize emphasis with `" + marker + "` for `options.emphasis`, expected `*`, or `_`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/util/encode-character-reference.js
  function encodeCharacterReference(code2) {
    return "&#x" + code2.toString(16).toUpperCase() + ";";
  }

  // node_modules/mdast-util-to-markdown/lib/util/encode-info.js
  function encodeInfo(outside, inside, marker) {
    const outsideKind = classifyCharacter(outside);
    const insideKind = classifyCharacter(inside);
    if (outsideKind === void 0) {
      return insideKind === void 0 ? (
        // Letter inside:
        // we have to encode *both* letters for `_` as it is looser.
        // it already forms for `*` (and GFMs `~`).
        marker === "_" ? { inside: true, outside: true } : { inside: false, outside: false }
      ) : insideKind === 1 ? (
        // Whitespace inside: encode both (letter, whitespace).
        { inside: true, outside: true }
      ) : (
        // Punctuation inside: encode outer (letter)
        { inside: false, outside: true }
      );
    }
    if (outsideKind === 1) {
      return insideKind === void 0 ? (
        // Letter inside: already forms.
        { inside: false, outside: false }
      ) : insideKind === 1 ? (
        // Whitespace inside: encode both (whitespace).
        { inside: true, outside: true }
      ) : (
        // Punctuation inside: already forms.
        { inside: false, outside: false }
      );
    }
    return insideKind === void 0 ? (
      // Letter inside: already forms.
      { inside: false, outside: false }
    ) : insideKind === 1 ? (
      // Whitespace inside: encode inner (whitespace).
      { inside: true, outside: false }
    ) : (
      // Punctuation inside: already forms.
      { inside: false, outside: false }
    );
  }

  // node_modules/mdast-util-to-markdown/lib/handle/emphasis.js
  emphasis.peek = emphasisPeek;
  function emphasis(node2, _, state, info) {
    const marker = checkEmphasis(state);
    const exit2 = state.enter("emphasis");
    const tracker = state.createTracker(info);
    const before = tracker.move(marker);
    let between2 = tracker.move(
      state.containerPhrasing(node2, {
        after: marker,
        before,
        ...tracker.current()
      })
    );
    const betweenHead = between2.charCodeAt(0);
    const open = encodeInfo(
      info.before.charCodeAt(info.before.length - 1),
      betweenHead,
      marker
    );
    if (open.inside) {
      between2 = encodeCharacterReference(betweenHead) + between2.slice(1);
    }
    const betweenTail = between2.charCodeAt(between2.length - 1);
    const close2 = encodeInfo(info.after.charCodeAt(0), betweenTail, marker);
    if (close2.inside) {
      between2 = between2.slice(0, -1) + encodeCharacterReference(betweenTail);
    }
    const after = tracker.move(marker);
    exit2();
    state.attentionEncodeSurroundingInfo = {
      after: close2.outside,
      before: open.outside
    };
    return before + between2 + after;
  }
  function emphasisPeek(_, _1, state) {
    return state.options.emphasis || "*";
  }

  // node_modules/unist-util-is/lib/index.js
  var convert = (
    // Note: overloads in JSDoc can’t yet use different `@template`s.
    /**
     * @type {(
     *   (<Condition extends string>(test: Condition) => (node: unknown, index?: number | null | undefined, parent?: Parent | null | undefined, context?: unknown) => node is Node & {type: Condition}) &
     *   (<Condition extends Props>(test: Condition) => (node: unknown, index?: number | null | undefined, parent?: Parent | null | undefined, context?: unknown) => node is Node & Condition) &
     *   (<Condition extends TestFunction>(test: Condition) => (node: unknown, index?: number | null | undefined, parent?: Parent | null | undefined, context?: unknown) => node is Node & Predicate<Condition, Node>) &
     *   ((test?: null | undefined) => (node?: unknown, index?: number | null | undefined, parent?: Parent | null | undefined, context?: unknown) => node is Node) &
     *   ((test?: Test) => Check)
     * )}
     */
    /**
     * @param {Test} [test]
     * @returns {Check}
     */
    (function(test) {
      if (test === null || test === void 0) {
        return ok;
      }
      if (typeof test === "function") {
        return castFactory(test);
      }
      if (typeof test === "object") {
        return Array.isArray(test) ? anyFactory(test) : (
          // Cast because `ReadonlyArray` goes into the above but `isArray`
          // narrows to `Array`.
          propertiesFactory(
            /** @type {Props} */
            test
          )
        );
      }
      if (typeof test === "string") {
        return typeFactory(test);
      }
      throw new Error("Expected function, string, or object as test");
    })
  );
  function anyFactory(tests) {
    const checks = [];
    let index2 = -1;
    while (++index2 < tests.length) {
      checks[index2] = convert(tests[index2]);
    }
    return castFactory(any);
    function any(...parameters) {
      let index3 = -1;
      while (++index3 < checks.length) {
        if (checks[index3].apply(this, parameters)) return true;
      }
      return false;
    }
  }
  function propertiesFactory(check) {
    const checkAsRecord = (
      /** @type {Record<string, unknown>} */
      check
    );
    return castFactory(all2);
    function all2(node2) {
      const nodeAsRecord = (
        /** @type {Record<string, unknown>} */
        /** @type {unknown} */
        node2
      );
      let key3;
      for (key3 in check) {
        if (nodeAsRecord[key3] !== checkAsRecord[key3]) return false;
      }
      return true;
    }
  }
  function typeFactory(check) {
    return castFactory(type);
    function type(node2) {
      return node2 && node2.type === check;
    }
  }
  function castFactory(testFunction) {
    return check;
    function check(value, index2, parent) {
      return Boolean(
        looksLikeANode(value) && testFunction.call(
          this,
          value,
          typeof index2 === "number" ? index2 : void 0,
          parent || void 0
        )
      );
    }
  }
  function ok() {
    return true;
  }
  function looksLikeANode(value) {
    return value !== null && typeof value === "object" && "type" in value;
  }

  // node_modules/unist-util-visit-parents/lib/color.js
  function color(d2) {
    return d2;
  }

  // node_modules/unist-util-visit-parents/lib/index.js
  var empty = [];
  var CONTINUE = true;
  var EXIT = false;
  var SKIP = "skip";
  function visitParents(tree, test, visitor, reverse) {
    let check;
    if (typeof test === "function" && typeof visitor !== "function") {
      reverse = visitor;
      visitor = test;
    } else {
      check = test;
    }
    const is2 = convert(check);
    const step = reverse ? -1 : 1;
    factory(tree, void 0, [])();
    function factory(node2, index2, parents) {
      const value = (
        /** @type {Record<string, unknown>} */
        node2 && typeof node2 === "object" ? node2 : {}
      );
      if (typeof value.type === "string") {
        const name = (
          // `hast`
          typeof value.tagName === "string" ? value.tagName : (
            // `xast`
            typeof value.name === "string" ? value.name : void 0
          )
        );
        Object.defineProperty(visit2, "name", {
          value: "node (" + color(node2.type + (name ? "<" + name + ">" : "")) + ")"
        });
      }
      return visit2;
      function visit2() {
        let result = empty;
        let subresult;
        let offset;
        let grandparents;
        if (!test || is2(node2, index2, parents[parents.length - 1] || void 0)) {
          result = toResult(visitor(node2, parents));
          if (result[0] === EXIT) {
            return result;
          }
        }
        if ("children" in node2 && node2.children) {
          const nodeAsParent = (
            /** @type {UnistParent} */
            node2
          );
          if (nodeAsParent.children && result[0] !== SKIP) {
            offset = (reverse ? nodeAsParent.children.length : -1) + step;
            grandparents = parents.concat(nodeAsParent);
            while (offset > -1 && offset < nodeAsParent.children.length) {
              const child = nodeAsParent.children[offset];
              subresult = factory(child, offset, grandparents)();
              if (subresult[0] === EXIT) {
                return subresult;
              }
              offset = typeof subresult[1] === "number" ? subresult[1] : offset + step;
            }
          }
        }
        return result;
      }
    }
  }
  function toResult(value) {
    if (Array.isArray(value)) {
      return value;
    }
    if (typeof value === "number") {
      return [CONTINUE, value];
    }
    return value === null || value === void 0 ? empty : [value];
  }

  // node_modules/unist-util-visit/lib/index.js
  function visit(tree, testOrVisitor, visitorOrReverse, maybeReverse) {
    let reverse;
    let test;
    let visitor;
    if (typeof testOrVisitor === "function" && typeof visitorOrReverse !== "function") {
      test = void 0;
      visitor = testOrVisitor;
      reverse = visitorOrReverse;
    } else {
      test = testOrVisitor;
      visitor = visitorOrReverse;
      reverse = maybeReverse;
    }
    visitParents(tree, test, overload, reverse);
    function overload(node2, parents) {
      const parent = parents[parents.length - 1];
      const index2 = parent ? parent.children.indexOf(node2) : void 0;
      return visitor(node2, index2, parent);
    }
  }

  // node_modules/mdast-util-to-markdown/lib/util/format-heading-as-setext.js
  function formatHeadingAsSetext(node2, state) {
    let literalWithBreak = false;
    visit(node2, function(node3) {
      if ("value" in node3 && /\r?\n|\r/.test(node3.value) || node3.type === "break") {
        literalWithBreak = true;
        return EXIT;
      }
    });
    return Boolean(
      (!node2.depth || node2.depth < 3) && toString(node2) && (state.options.setext || literalWithBreak)
    );
  }

  // node_modules/mdast-util-to-markdown/lib/handle/heading.js
  function heading(node2, _, state, info) {
    const rank = Math.max(Math.min(6, node2.depth || 1), 1);
    const tracker = state.createTracker(info);
    if (formatHeadingAsSetext(node2, state)) {
      const exit3 = state.enter("headingSetext");
      const subexit2 = state.enter("phrasing");
      const value2 = state.containerPhrasing(node2, {
        ...tracker.current(),
        before: "\n",
        after: "\n"
      });
      subexit2();
      exit3();
      return value2 + "\n" + (rank === 1 ? "=" : "-").repeat(
        // The whole size…
        value2.length - // Minus the position of the character after the last EOL (or
        // 0 if there is none)…
        (Math.max(value2.lastIndexOf("\r"), value2.lastIndexOf("\n")) + 1)
      );
    }
    const sequence = "#".repeat(rank);
    const exit2 = state.enter("headingAtx");
    const subexit = state.enter("phrasing");
    tracker.move(sequence + " ");
    let value = state.containerPhrasing(node2, {
      before: "# ",
      after: "\n",
      ...tracker.current()
    });
    if (/^[\t ]/.test(value)) {
      value = encodeCharacterReference(value.charCodeAt(0)) + value.slice(1);
    }
    value = value ? sequence + " " + value : sequence;
    if (state.options.closeAtx) {
      value += " " + sequence;
    }
    subexit();
    exit2();
    return value;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/html.js
  html.peek = htmlPeek;
  function html(node2) {
    return node2.value || "";
  }
  function htmlPeek() {
    return "<";
  }

  // node_modules/mdast-util-to-markdown/lib/handle/image.js
  image.peek = imagePeek;
  function image(node2, _, state, info) {
    const quote = checkQuote(state);
    const suffix = quote === '"' ? "Quote" : "Apostrophe";
    const exit2 = state.enter("image");
    let subexit = state.enter("label");
    const tracker = state.createTracker(info);
    let value = tracker.move("![");
    value += tracker.move(
      state.safe(node2.alt, { before: value, after: "]", ...tracker.current() })
    );
    value += tracker.move("](");
    subexit();
    if (
      // If there’s no url but there is a title…
      !node2.url && node2.title || // If there are control characters or whitespace.
      /[\0- \u007F]/.test(node2.url)
    ) {
      subexit = state.enter("destinationLiteral");
      value += tracker.move("<");
      value += tracker.move(
        state.safe(node2.url, { before: value, after: ">", ...tracker.current() })
      );
      value += tracker.move(">");
    } else {
      subexit = state.enter("destinationRaw");
      value += tracker.move(
        state.safe(node2.url, {
          before: value,
          after: node2.title ? " " : ")",
          ...tracker.current()
        })
      );
    }
    subexit();
    if (node2.title) {
      subexit = state.enter(`title${suffix}`);
      value += tracker.move(" " + quote);
      value += tracker.move(
        state.safe(node2.title, {
          before: value,
          after: quote,
          ...tracker.current()
        })
      );
      value += tracker.move(quote);
      subexit();
    }
    value += tracker.move(")");
    exit2();
    return value;
  }
  function imagePeek() {
    return "!";
  }

  // node_modules/mdast-util-to-markdown/lib/handle/image-reference.js
  imageReference.peek = imageReferencePeek;
  function imageReference(node2, _, state, info) {
    const type = node2.referenceType;
    const exit2 = state.enter("imageReference");
    let subexit = state.enter("label");
    const tracker = state.createTracker(info);
    let value = tracker.move("![");
    const alt = state.safe(node2.alt, {
      before: value,
      after: "]",
      ...tracker.current()
    });
    value += tracker.move(alt + "][");
    subexit();
    const stack = state.stack;
    state.stack = [];
    subexit = state.enter("reference");
    const reference = state.safe(state.associationId(node2), {
      before: value,
      after: "]",
      ...tracker.current()
    });
    subexit();
    state.stack = stack;
    exit2();
    if (type === "full" || !alt || alt !== reference) {
      value += tracker.move(reference + "]");
    } else if (type === "shortcut") {
      value = value.slice(0, -1);
    } else {
      value += tracker.move("]");
    }
    return value;
  }
  function imageReferencePeek() {
    return "!";
  }

  // node_modules/mdast-util-to-markdown/lib/handle/inline-code.js
  inlineCode.peek = inlineCodePeek;
  function inlineCode(node2, _, state) {
    let value = node2.value || "";
    let sequence = "`";
    let index2 = -1;
    while (new RegExp("(^|[^`])" + sequence + "([^`]|$)").test(value)) {
      sequence += "`";
    }
    if (/[^ \r\n]/.test(value) && (/^[ \r\n]/.test(value) && /[ \r\n]$/.test(value) || /^`|`$/.test(value))) {
      value = " " + value + " ";
    }
    while (++index2 < state.unsafe.length) {
      const pattern = state.unsafe[index2];
      const expression = state.compilePattern(pattern);
      let match;
      if (!pattern.atBreak) continue;
      while (match = expression.exec(value)) {
        let position2 = match.index;
        if (value.charCodeAt(position2) === 10 && value.charCodeAt(position2 - 1) === 13) {
          position2--;
        }
        value = value.slice(0, position2) + " " + value.slice(match.index + 1);
      }
    }
    return sequence + value + sequence;
  }
  function inlineCodePeek() {
    return "`";
  }

  // node_modules/mdast-util-to-markdown/lib/util/format-link-as-autolink.js
  function formatLinkAsAutolink(node2, state) {
    const raw = toString(node2);
    return Boolean(
      !state.options.resourceLink && // If there’s a url…
      node2.url && // And there’s a no title…
      !node2.title && // And the content of `node` is a single text node…
      node2.children && node2.children.length === 1 && node2.children[0].type === "text" && // And if the url is the same as the content…
      (raw === node2.url || "mailto:" + raw === node2.url) && // And that starts w/ a protocol…
      /^[a-z][a-z+.-]+:/i.test(node2.url) && // And that doesn’t contain ASCII control codes (character escapes and
      // references don’t work), space, or angle brackets…
      !/[\0- <>\u007F]/.test(node2.url)
    );
  }

  // node_modules/mdast-util-to-markdown/lib/handle/link.js
  link.peek = linkPeek;
  function link(node2, _, state, info) {
    const quote = checkQuote(state);
    const suffix = quote === '"' ? "Quote" : "Apostrophe";
    const tracker = state.createTracker(info);
    let exit2;
    let subexit;
    if (formatLinkAsAutolink(node2, state)) {
      const stack = state.stack;
      state.stack = [];
      exit2 = state.enter("autolink");
      let value2 = tracker.move("<");
      value2 += tracker.move(
        state.containerPhrasing(node2, {
          before: value2,
          after: ">",
          ...tracker.current()
        })
      );
      value2 += tracker.move(">");
      exit2();
      state.stack = stack;
      return value2;
    }
    exit2 = state.enter("link");
    subexit = state.enter("label");
    let value = tracker.move("[");
    value += tracker.move(
      state.containerPhrasing(node2, {
        before: value,
        after: "](",
        ...tracker.current()
      })
    );
    value += tracker.move("](");
    subexit();
    if (
      // If there’s no url but there is a title…
      !node2.url && node2.title || // If there are control characters or whitespace.
      /[\0- \u007F]/.test(node2.url)
    ) {
      subexit = state.enter("destinationLiteral");
      value += tracker.move("<");
      value += tracker.move(
        state.safe(node2.url, { before: value, after: ">", ...tracker.current() })
      );
      value += tracker.move(">");
    } else {
      subexit = state.enter("destinationRaw");
      value += tracker.move(
        state.safe(node2.url, {
          before: value,
          after: node2.title ? " " : ")",
          ...tracker.current()
        })
      );
    }
    subexit();
    if (node2.title) {
      subexit = state.enter(`title${suffix}`);
      value += tracker.move(" " + quote);
      value += tracker.move(
        state.safe(node2.title, {
          before: value,
          after: quote,
          ...tracker.current()
        })
      );
      value += tracker.move(quote);
      subexit();
    }
    value += tracker.move(")");
    exit2();
    return value;
  }
  function linkPeek(node2, _, state) {
    return formatLinkAsAutolink(node2, state) ? "<" : "[";
  }

  // node_modules/mdast-util-to-markdown/lib/handle/link-reference.js
  linkReference.peek = linkReferencePeek;
  function linkReference(node2, _, state, info) {
    const type = node2.referenceType;
    const exit2 = state.enter("linkReference");
    let subexit = state.enter("label");
    const tracker = state.createTracker(info);
    let value = tracker.move("[");
    const text4 = state.containerPhrasing(node2, {
      before: value,
      after: "]",
      ...tracker.current()
    });
    value += tracker.move(text4 + "][");
    subexit();
    const stack = state.stack;
    state.stack = [];
    subexit = state.enter("reference");
    const reference = state.safe(state.associationId(node2), {
      before: value,
      after: "]",
      ...tracker.current()
    });
    subexit();
    state.stack = stack;
    exit2();
    if (type === "full" || !text4 || text4 !== reference) {
      value += tracker.move(reference + "]");
    } else if (type === "shortcut") {
      value = value.slice(0, -1);
    } else {
      value += tracker.move("]");
    }
    return value;
  }
  function linkReferencePeek() {
    return "[";
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-bullet.js
  function checkBullet(state) {
    const marker = state.options.bullet || "*";
    if (marker !== "*" && marker !== "+" && marker !== "-") {
      throw new Error(
        "Cannot serialize items with `" + marker + "` for `options.bullet`, expected `*`, `+`, or `-`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-bullet-other.js
  function checkBulletOther(state) {
    const bullet = checkBullet(state);
    const bulletOther = state.options.bulletOther;
    if (!bulletOther) {
      return bullet === "*" ? "-" : "*";
    }
    if (bulletOther !== "*" && bulletOther !== "+" && bulletOther !== "-") {
      throw new Error(
        "Cannot serialize items with `" + bulletOther + "` for `options.bulletOther`, expected `*`, `+`, or `-`"
      );
    }
    if (bulletOther === bullet) {
      throw new Error(
        "Expected `bullet` (`" + bullet + "`) and `bulletOther` (`" + bulletOther + "`) to be different"
      );
    }
    return bulletOther;
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-bullet-ordered.js
  function checkBulletOrdered(state) {
    const marker = state.options.bulletOrdered || ".";
    if (marker !== "." && marker !== ")") {
      throw new Error(
        "Cannot serialize items with `" + marker + "` for `options.bulletOrdered`, expected `.` or `)`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-rule.js
  function checkRule(state) {
    const marker = state.options.rule || "*";
    if (marker !== "*" && marker !== "-" && marker !== "_") {
      throw new Error(
        "Cannot serialize rules with `" + marker + "` for `options.rule`, expected `*`, `-`, or `_`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/list.js
  function list3(node2, parent, state, info) {
    const exit2 = state.enter("list");
    const bulletCurrent = state.bulletCurrent;
    let bullet = node2.ordered ? checkBulletOrdered(state) : checkBullet(state);
    const bulletOther = node2.ordered ? bullet === "." ? ")" : "." : checkBulletOther(state);
    let useDifferentMarker = parent && state.bulletLastUsed ? bullet === state.bulletLastUsed : false;
    if (!node2.ordered) {
      const firstListItem = node2.children ? node2.children[0] : void 0;
      if (
        // Bullet could be used as a thematic break marker:
        (bullet === "*" || bullet === "-") && // Empty first list item:
        firstListItem && (!firstListItem.children || !firstListItem.children[0]) && // Directly in two other list items:
        state.stack[state.stack.length - 1] === "list" && state.stack[state.stack.length - 2] === "listItem" && state.stack[state.stack.length - 3] === "list" && state.stack[state.stack.length - 4] === "listItem" && // That are each the first child.
        state.indexStack[state.indexStack.length - 1] === 0 && state.indexStack[state.indexStack.length - 2] === 0 && state.indexStack[state.indexStack.length - 3] === 0
      ) {
        useDifferentMarker = true;
      }
      if (checkRule(state) === bullet && firstListItem) {
        let index2 = -1;
        while (++index2 < node2.children.length) {
          const item = node2.children[index2];
          if (item && item.type === "listItem" && item.children && item.children[0] && item.children[0].type === "thematicBreak") {
            useDifferentMarker = true;
            break;
          }
        }
      }
    }
    if (useDifferentMarker) {
      bullet = bulletOther;
    }
    state.bulletCurrent = bullet;
    const value = state.containerFlow(node2, info);
    state.bulletLastUsed = bullet;
    state.bulletCurrent = bulletCurrent;
    exit2();
    return value;
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-list-item-indent.js
  function checkListItemIndent(state) {
    const style = state.options.listItemIndent || "one";
    if (style !== "tab" && style !== "one" && style !== "mixed") {
      throw new Error(
        "Cannot serialize items with `" + style + "` for `options.listItemIndent`, expected `tab`, `one`, or `mixed`"
      );
    }
    return style;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/list-item.js
  function listItem(node2, parent, state, info) {
    const listItemIndent = checkListItemIndent(state);
    let bullet = state.bulletCurrent || checkBullet(state);
    if (parent && parent.type === "list" && parent.ordered) {
      bullet = (typeof parent.start === "number" && parent.start > -1 ? parent.start : 1) + (state.options.incrementListMarker === false ? 0 : parent.children.indexOf(node2)) + bullet;
    }
    let size = bullet.length + 1;
    if (listItemIndent === "tab" || listItemIndent === "mixed" && (parent && parent.type === "list" && parent.spread || node2.spread)) {
      size = Math.ceil(size / 4) * 4;
    }
    const tracker = state.createTracker(info);
    tracker.move(bullet + " ".repeat(size - bullet.length));
    tracker.shift(size);
    const exit2 = state.enter("listItem");
    const value = state.indentLines(
      state.containerFlow(node2, tracker.current()),
      map4
    );
    exit2();
    return value;
    function map4(line, index2, blank) {
      if (index2) {
        return (blank ? "" : " ".repeat(size)) + line;
      }
      return (blank ? bullet : bullet + " ".repeat(size - bullet.length)) + line;
    }
  }

  // node_modules/mdast-util-to-markdown/lib/handle/paragraph.js
  function paragraph(node2, _, state, info) {
    const exit2 = state.enter("paragraph");
    const subexit = state.enter("phrasing");
    const value = state.containerPhrasing(node2, info);
    subexit();
    exit2();
    return value;
  }

  // node_modules/mdast-util-phrasing/lib/index.js
  var phrasing = (
    /** @type {(node?: unknown) => node is Exclude<PhrasingContent, Html>} */
    convert([
      "break",
      "delete",
      "emphasis",
      // To do: next major: removed since footnotes were added to GFM.
      "footnote",
      "footnoteReference",
      "image",
      "imageReference",
      "inlineCode",
      // Enabled by `mdast-util-math`:
      "inlineMath",
      "link",
      "linkReference",
      // Enabled by `mdast-util-mdx`:
      "mdxJsxTextElement",
      // Enabled by `mdast-util-mdx`:
      "mdxTextExpression",
      "strong",
      "text",
      // Enabled by `mdast-util-directive`:
      "textDirective"
    ])
  );

  // node_modules/mdast-util-to-markdown/lib/handle/root.js
  function root(node2, _, state, info) {
    const hasPhrasing = node2.children.some(function(d2) {
      return phrasing(d2);
    });
    const container = hasPhrasing ? state.containerPhrasing : state.containerFlow;
    return container.call(state, node2, info);
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-strong.js
  function checkStrong(state) {
    const marker = state.options.strong || "*";
    if (marker !== "*" && marker !== "_") {
      throw new Error(
        "Cannot serialize strong with `" + marker + "` for `options.strong`, expected `*`, or `_`"
      );
    }
    return marker;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/strong.js
  strong.peek = strongPeek;
  function strong(node2, _, state, info) {
    const marker = checkStrong(state);
    const exit2 = state.enter("strong");
    const tracker = state.createTracker(info);
    const before = tracker.move(marker + marker);
    let between2 = tracker.move(
      state.containerPhrasing(node2, {
        after: marker,
        before,
        ...tracker.current()
      })
    );
    const betweenHead = between2.charCodeAt(0);
    const open = encodeInfo(
      info.before.charCodeAt(info.before.length - 1),
      betweenHead,
      marker
    );
    if (open.inside) {
      between2 = encodeCharacterReference(betweenHead) + between2.slice(1);
    }
    const betweenTail = between2.charCodeAt(between2.length - 1);
    const close2 = encodeInfo(info.after.charCodeAt(0), betweenTail, marker);
    if (close2.inside) {
      between2 = between2.slice(0, -1) + encodeCharacterReference(betweenTail);
    }
    const after = tracker.move(marker + marker);
    exit2();
    state.attentionEncodeSurroundingInfo = {
      after: close2.outside,
      before: open.outside
    };
    return before + between2 + after;
  }
  function strongPeek(_, _1, state) {
    return state.options.strong || "*";
  }

  // node_modules/mdast-util-to-markdown/lib/handle/text.js
  function text3(node2, _, state, info) {
    return state.safe(node2.value, info);
  }

  // node_modules/mdast-util-to-markdown/lib/util/check-rule-repetition.js
  function checkRuleRepetition(state) {
    const repetition = state.options.ruleRepetition || 3;
    if (repetition < 3) {
      throw new Error(
        "Cannot serialize rules with repetition `" + repetition + "` for `options.ruleRepetition`, expected `3` or more"
      );
    }
    return repetition;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/thematic-break.js
  function thematicBreak2(_, _1, state) {
    const value = (checkRule(state) + (state.options.ruleSpaces ? " " : "")).repeat(checkRuleRepetition(state));
    return state.options.ruleSpaces ? value.slice(0, -1) : value;
  }

  // node_modules/mdast-util-to-markdown/lib/handle/index.js
  var handle = {
    blockquote,
    break: hardBreak,
    code,
    definition: definition2,
    emphasis,
    hardBreak,
    heading,
    html,
    image,
    imageReference,
    inlineCode,
    link,
    linkReference,
    list: list3,
    listItem,
    paragraph,
    root,
    strong,
    text: text3,
    thematicBreak: thematicBreak2
  };

  // node_modules/mdast-util-to-markdown/lib/join.js
  var join = [joinDefaults];
  function joinDefaults(left, right, parent, state) {
    if (right.type === "code" && formatCodeAsIndented(right, state) && (left.type === "list" || left.type === right.type && formatCodeAsIndented(left, state))) {
      return false;
    }
    if ("spread" in parent && typeof parent.spread === "boolean") {
      if (left.type === "paragraph" && // Two paragraphs.
      (left.type === right.type || right.type === "definition" || // Paragraph followed by a setext heading.
      right.type === "heading" && formatHeadingAsSetext(right, state))) {
        return;
      }
      return parent.spread ? 1 : 0;
    }
  }

  // node_modules/mdast-util-to-markdown/lib/unsafe.js
  var fullPhrasingSpans = [
    "autolink",
    "destinationLiteral",
    "destinationRaw",
    "reference",
    "titleQuote",
    "titleApostrophe"
  ];
  var unsafe = [
    { character: "	", after: "[\\r\\n]", inConstruct: "phrasing" },
    { character: "	", before: "[\\r\\n]", inConstruct: "phrasing" },
    {
      character: "	",
      inConstruct: ["codeFencedLangGraveAccent", "codeFencedLangTilde"]
    },
    {
      character: "\r",
      inConstruct: [
        "codeFencedLangGraveAccent",
        "codeFencedLangTilde",
        "codeFencedMetaGraveAccent",
        "codeFencedMetaTilde",
        "destinationLiteral",
        "headingAtx"
      ]
    },
    {
      character: "\n",
      inConstruct: [
        "codeFencedLangGraveAccent",
        "codeFencedLangTilde",
        "codeFencedMetaGraveAccent",
        "codeFencedMetaTilde",
        "destinationLiteral",
        "headingAtx"
      ]
    },
    { character: " ", after: "[\\r\\n]", inConstruct: "phrasing" },
    { character: " ", before: "[\\r\\n]", inConstruct: "phrasing" },
    {
      character: " ",
      inConstruct: ["codeFencedLangGraveAccent", "codeFencedLangTilde"]
    },
    // An exclamation mark can start an image, if it is followed by a link or
    // a link reference.
    {
      character: "!",
      after: "\\[",
      inConstruct: "phrasing",
      notInConstruct: fullPhrasingSpans
    },
    // A quote can break out of a title.
    { character: '"', inConstruct: "titleQuote" },
    // A number sign could start an ATX heading if it starts a line.
    { atBreak: true, character: "#" },
    { character: "#", inConstruct: "headingAtx", after: "(?:[\r\n]|$)" },
    // Dollar sign and percentage are not used in markdown.
    // An ampersand could start a character reference.
    { character: "&", after: "[#A-Za-z]", inConstruct: "phrasing" },
    // An apostrophe can break out of a title.
    { character: "'", inConstruct: "titleApostrophe" },
    // A left paren could break out of a destination raw.
    { character: "(", inConstruct: "destinationRaw" },
    // A left paren followed by `]` could make something into a link or image.
    {
      before: "\\]",
      character: "(",
      inConstruct: "phrasing",
      notInConstruct: fullPhrasingSpans
    },
    // A right paren could start a list item or break out of a destination
    // raw.
    { atBreak: true, before: "\\d+", character: ")" },
    { character: ")", inConstruct: "destinationRaw" },
    // An asterisk can start thematic breaks, list items, emphasis, strong.
    { atBreak: true, character: "*", after: "(?:[ 	\r\n*])" },
    { character: "*", inConstruct: "phrasing", notInConstruct: fullPhrasingSpans },
    // A plus sign could start a list item.
    { atBreak: true, character: "+", after: "(?:[ 	\r\n])" },
    // A dash can start thematic breaks, list items, and setext heading
    // underlines.
    { atBreak: true, character: "-", after: "(?:[ 	\r\n-])" },
    // A dot could start a list item.
    { atBreak: true, before: "\\d+", character: ".", after: "(?:[ 	\r\n]|$)" },
    // Slash, colon, and semicolon are not used in markdown for constructs.
    // A less than can start html (flow or text) or an autolink.
    // HTML could start with an exclamation mark (declaration, cdata, comment),
    // slash (closing tag), question mark (instruction), or a letter (tag).
    // An autolink also starts with a letter.
    // Finally, it could break out of a destination literal.
    { atBreak: true, character: "<", after: "[!/?A-Za-z]" },
    {
      character: "<",
      after: "[!/?A-Za-z]",
      inConstruct: "phrasing",
      notInConstruct: fullPhrasingSpans
    },
    { character: "<", inConstruct: "destinationLiteral" },
    // An equals to can start setext heading underlines.
    { atBreak: true, character: "=" },
    // A greater than can start block quotes and it can break out of a
    // destination literal.
    { atBreak: true, character: ">" },
    { character: ">", inConstruct: "destinationLiteral" },
    // Question mark and at sign are not used in markdown for constructs.
    // A left bracket can start definitions, references, labels,
    { atBreak: true, character: "[" },
    { character: "[", inConstruct: "phrasing", notInConstruct: fullPhrasingSpans },
    { character: "[", inConstruct: ["label", "reference"] },
    // A backslash can start an escape (when followed by punctuation) or a
    // hard break (when followed by an eol).
    // Note: typical escapes are handled in `safe`!
    { character: "\\", after: "[\\r\\n]", inConstruct: "phrasing" },
    // A right bracket can exit labels.
    { character: "]", inConstruct: ["label", "reference"] },
    // Caret is not used in markdown for constructs.
    // An underscore can start emphasis, strong, or a thematic break.
    { atBreak: true, character: "_" },
    { character: "_", inConstruct: "phrasing", notInConstruct: fullPhrasingSpans },
    // A grave accent can start code (fenced or text), or it can break out of
    // a grave accent code fence.
    { atBreak: true, character: "`" },
    {
      character: "`",
      inConstruct: ["codeFencedLangGraveAccent", "codeFencedMetaGraveAccent"]
    },
    { character: "`", inConstruct: "phrasing", notInConstruct: fullPhrasingSpans },
    // Left brace, vertical bar, right brace are not used in markdown for
    // constructs.
    // A tilde can start code (fenced).
    { atBreak: true, character: "~" }
  ];

  // node_modules/mdast-util-to-markdown/lib/util/association.js
  function association(node2) {
    if (node2.label || !node2.identifier) {
      return node2.label || "";
    }
    return decodeString(node2.identifier);
  }

  // node_modules/mdast-util-to-markdown/lib/util/compile-pattern.js
  function compilePattern(pattern) {
    if (!pattern._compiled) {
      const before = (pattern.atBreak ? "[\\r\\n][\\t ]*" : "") + (pattern.before ? "(?:" + pattern.before + ")" : "");
      pattern._compiled = new RegExp(
        (before ? "(" + before + ")" : "") + (/[|\\{}()[\]^$+*?.-]/.test(pattern.character) ? "\\" : "") + pattern.character + (pattern.after ? "(?:" + pattern.after + ")" : ""),
        "g"
      );
    }
    return pattern._compiled;
  }

  // node_modules/mdast-util-to-markdown/lib/util/container-phrasing.js
  function containerPhrasing(parent, state, info) {
    const indexStack = state.indexStack;
    const children = parent.children || [];
    const results = [];
    let index2 = -1;
    let before = info.before;
    let encodeAfter;
    indexStack.push(-1);
    let tracker = state.createTracker(info);
    while (++index2 < children.length) {
      const child = children[index2];
      let after;
      indexStack[indexStack.length - 1] = index2;
      if (index2 + 1 < children.length) {
        let handle2 = state.handle.handlers[children[index2 + 1].type];
        if (handle2 && handle2.peek) handle2 = handle2.peek;
        after = handle2 ? handle2(children[index2 + 1], parent, state, {
          before: "",
          after: "",
          ...tracker.current()
        }).charAt(0) : "";
      } else {
        after = info.after;
      }
      if (results.length > 0 && (before === "\r" || before === "\n") && child.type === "html") {
        results[results.length - 1] = results[results.length - 1].replace(
          /(\r?\n|\r)$/,
          " "
        );
        before = " ";
        tracker = state.createTracker(info);
        tracker.move(results.join(""));
      }
      let value = state.handle(child, parent, state, {
        ...tracker.current(),
        after,
        before
      });
      if (encodeAfter && encodeAfter === value.slice(0, 1)) {
        value = encodeCharacterReference(encodeAfter.charCodeAt(0)) + value.slice(1);
      }
      const encodingInfo = state.attentionEncodeSurroundingInfo;
      state.attentionEncodeSurroundingInfo = void 0;
      encodeAfter = void 0;
      if (encodingInfo) {
        if (results.length > 0 && encodingInfo.before && before === results[results.length - 1].slice(-1)) {
          results[results.length - 1] = results[results.length - 1].slice(0, -1) + encodeCharacterReference(before.charCodeAt(0));
        }
        if (encodingInfo.after) encodeAfter = after;
      }
      tracker.move(value);
      results.push(value);
      before = value.slice(-1);
    }
    indexStack.pop();
    return results.join("");
  }

  // node_modules/mdast-util-to-markdown/lib/util/container-flow.js
  function containerFlow(parent, state, info) {
    const indexStack = state.indexStack;
    const children = parent.children || [];
    const tracker = state.createTracker(info);
    const results = [];
    let index2 = -1;
    indexStack.push(-1);
    while (++index2 < children.length) {
      const child = children[index2];
      indexStack[indexStack.length - 1] = index2;
      results.push(
        tracker.move(
          state.handle(child, parent, state, {
            before: "\n",
            after: "\n",
            ...tracker.current()
          })
        )
      );
      if (child.type !== "list") {
        state.bulletLastUsed = void 0;
      }
      if (index2 < children.length - 1) {
        results.push(
          tracker.move(between(child, children[index2 + 1], parent, state))
        );
      }
    }
    indexStack.pop();
    return results.join("");
  }
  function between(left, right, parent, state) {
    let index2 = state.join.length;
    while (index2--) {
      const result = state.join[index2](left, right, parent, state);
      if (result === true || result === 1) {
        break;
      }
      if (typeof result === "number") {
        return "\n".repeat(1 + result);
      }
      if (result === false) {
        return "\n\n<!---->\n\n";
      }
    }
    return "\n\n";
  }

  // node_modules/mdast-util-to-markdown/lib/util/indent-lines.js
  var eol = /\r?\n|\r/g;
  function indentLines(value, map4) {
    const result = [];
    let start = 0;
    let line = 0;
    let match;
    while (match = eol.exec(value)) {
      one2(value.slice(start, match.index));
      result.push(match[0]);
      start = match.index + match[0].length;
      line++;
    }
    one2(value.slice(start));
    return result.join("");
    function one2(value2) {
      result.push(map4(value2, line, !value2));
    }
  }

  // node_modules/mdast-util-to-markdown/lib/util/safe.js
  function safe(state, input, config2) {
    const value = (config2.before || "") + (input || "") + (config2.after || "");
    const positions = [];
    const result = [];
    const infos = {};
    let index2 = -1;
    while (++index2 < state.unsafe.length) {
      const pattern = state.unsafe[index2];
      if (!patternInScope(state.stack, pattern)) {
        continue;
      }
      const expression = state.compilePattern(pattern);
      let match;
      while (match = expression.exec(value)) {
        const before = "before" in pattern || Boolean(pattern.atBreak);
        const after = "after" in pattern;
        const position2 = match.index + (before ? match[1].length : 0);
        if (positions.includes(position2)) {
          if (infos[position2].before && !before) {
            infos[position2].before = false;
          }
          if (infos[position2].after && !after) {
            infos[position2].after = false;
          }
        } else {
          positions.push(position2);
          infos[position2] = { before, after };
        }
      }
    }
    positions.sort(numerical);
    let start = config2.before ? config2.before.length : 0;
    const end = value.length - (config2.after ? config2.after.length : 0);
    index2 = -1;
    while (++index2 < positions.length) {
      const position2 = positions[index2];
      if (position2 < start || position2 >= end) {
        continue;
      }
      if (position2 + 1 < end && positions[index2 + 1] === position2 + 1 && infos[position2].after && !infos[position2 + 1].before && !infos[position2 + 1].after || positions[index2 - 1] === position2 - 1 && infos[position2].before && !infos[position2 - 1].before && !infos[position2 - 1].after) {
        continue;
      }
      if (start !== position2) {
        result.push(escapeBackslashes(value.slice(start, position2), "\\"));
      }
      start = position2;
      if (/[!-/:-@[-`{-~]/.test(value.charAt(position2)) && (!config2.encode || !config2.encode.includes(value.charAt(position2)))) {
        result.push("\\");
      } else {
        result.push(encodeCharacterReference(value.charCodeAt(position2)));
        start++;
      }
    }
    result.push(escapeBackslashes(value.slice(start, end), config2.after));
    return result.join("");
  }
  function numerical(a2, b) {
    return a2 - b;
  }
  function escapeBackslashes(value, after) {
    const expression = /\\(?=[!-/:-@[-`{-~])/g;
    const positions = [];
    const results = [];
    const whole = value + after;
    let index2 = -1;
    let start = 0;
    let match;
    while (match = expression.exec(whole)) {
      positions.push(match.index);
    }
    while (++index2 < positions.length) {
      if (start !== positions[index2]) {
        results.push(value.slice(start, positions[index2]));
      }
      results.push("\\");
      start = positions[index2];
    }
    results.push(value.slice(start));
    return results.join("");
  }

  // node_modules/mdast-util-to-markdown/lib/util/track.js
  function track(config2) {
    const options = config2 || {};
    const now2 = options.now || {};
    let lineShift = options.lineShift || 0;
    let line = now2.line || 1;
    let column = now2.column || 1;
    return { move, current, shift: shift2 };
    function current() {
      return { now: { line, column }, lineShift };
    }
    function shift2(value) {
      lineShift += value;
    }
    function move(input) {
      const value = input || "";
      const chunks = value.split(/\r?\n|\r/g);
      const tail = chunks[chunks.length - 1];
      line += chunks.length - 1;
      column = chunks.length === 1 ? column + tail.length : 1 + tail.length + lineShift;
      return value;
    }
  }

  // node_modules/mdast-util-to-markdown/lib/index.js
  function toMarkdown(tree, options) {
    const settings = options || {};
    const state = {
      associationId: association,
      containerPhrasing: containerPhrasingBound,
      containerFlow: containerFlowBound,
      createTracker: track,
      compilePattern,
      enter,
      // @ts-expect-error: GFM / frontmatter are typed in `mdast` but not defined
      // here.
      handlers: { ...handle },
      // @ts-expect-error: add `handle` in a second.
      handle: void 0,
      indentLines,
      indexStack: [],
      join: [...join],
      options: {},
      safe: safeBound,
      stack: [],
      unsafe: [...unsafe]
    };
    configure2(state, settings);
    if (state.options.tightDefinitions) {
      state.join.push(joinDefinition);
    }
    state.handle = zwitch("type", {
      invalid,
      unknown,
      handlers: state.handlers
    });
    let result = state.handle(tree, void 0, state, {
      before: "\n",
      after: "\n",
      now: { line: 1, column: 1 },
      lineShift: 0
    });
    if (result && result.charCodeAt(result.length - 1) !== 10 && result.charCodeAt(result.length - 1) !== 13) {
      result += "\n";
    }
    return result;
    function enter(name) {
      state.stack.push(name);
      return exit2;
      function exit2() {
        state.stack.pop();
      }
    }
  }
  function invalid(value) {
    throw new Error("Cannot handle value `" + value + "`, expected node");
  }
  function unknown(value) {
    const node2 = (
      /** @type {Nodes} */
      value
    );
    throw new Error("Cannot handle unknown node `" + node2.type + "`");
  }
  function joinDefinition(left, right) {
    if (left.type === "definition" && left.type === right.type) {
      return 0;
    }
  }
  function containerPhrasingBound(parent, info) {
    return containerPhrasing(parent, this, info);
  }
  function containerFlowBound(parent, info) {
    return containerFlow(parent, this, info);
  }
  function safeBound(value, config2) {
    return safe(this, value, config2);
  }

  // node_modules/remark-stringify/lib/index.js
  function remarkStringify(options) {
    const self2 = this;
    self2.compiler = compiler2;
    function compiler2(tree) {
      return toMarkdown(tree, {
        ...self2.data("settings"),
        ...options,
        // Note: this option is not in the readme.
        // The goal is for it to be set by plugins on `data` instead of being
        // passed by users.
        extensions: self2.data("toMarkdownExtensions") || []
      });
    }
  }

  // node_modules/bail/index.js
  function bail(error) {
    if (error) {
      throw error;
    }
  }

  // node_modules/unified/lib/index.js
  var import_extend = __toESM(require_extend(), 1);

  // node_modules/devlop/lib/default.js
  function ok2() {
  }

  // node_modules/is-plain-obj/index.js
  function isPlainObject(value) {
    if (typeof value !== "object" || value === null) {
      return false;
    }
    const prototype = Object.getPrototypeOf(value);
    return (prototype === null || prototype === Object.prototype || Object.getPrototypeOf(prototype) === null) && !(Symbol.toStringTag in value) && !(Symbol.iterator in value);
  }

  // node_modules/trough/lib/index.js
  function trough() {
    const fns = [];
    const pipeline = { run: run2, use };
    return pipeline;
    function run2(...values) {
      let middlewareIndex = -1;
      const callback = values.pop();
      if (typeof callback !== "function") {
        throw new TypeError("Expected function as last argument, not " + callback);
      }
      next(null, ...values);
      function next(error, ...output) {
        const fn = fns[++middlewareIndex];
        let index2 = -1;
        if (error) {
          callback(error);
          return;
        }
        while (++index2 < values.length) {
          if (output[index2] === null || output[index2] === void 0) {
            output[index2] = values[index2];
          }
        }
        values = output;
        if (fn) {
          wrap(fn, next)(...output);
        } else {
          callback(null, ...output);
        }
      }
    }
    function use(middelware) {
      if (typeof middelware !== "function") {
        throw new TypeError(
          "Expected `middelware` to be a function, not " + middelware
        );
      }
      fns.push(middelware);
      return pipeline;
    }
  }
  function wrap(middleware, callback) {
    let called;
    return wrapped;
    function wrapped(...parameters) {
      const fnExpectsCallback = middleware.length > parameters.length;
      let result;
      if (fnExpectsCallback) {
        parameters.push(done);
      }
      try {
        result = middleware.apply(this, parameters);
      } catch (error) {
        const exception = (
          /** @type {Error} */
          error
        );
        if (fnExpectsCallback && called) {
          throw exception;
        }
        return done(exception);
      }
      if (!fnExpectsCallback) {
        if (result && result.then && typeof result.then === "function") {
          result.then(then, done);
        } else if (result instanceof Error) {
          done(result);
        } else {
          then(result);
        }
      }
    }
    function done(error, ...output) {
      if (!called) {
        called = true;
        callback(error, ...output);
      }
    }
    function then(value) {
      done(null, value);
    }
  }

  // node_modules/vfile-message/lib/index.js
  var VFileMessage = class extends Error {
    /**
     * Create a message for `reason`.
     *
     * > 🪦 **Note**: also has obsolete signatures.
     *
     * @overload
     * @param {string} reason
     * @param {Options | null | undefined} [options]
     * @returns
     *
     * @overload
     * @param {string} reason
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns
     *
     * @overload
     * @param {string} reason
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns
     *
     * @overload
     * @param {string} reason
     * @param {string | null | undefined} [origin]
     * @returns
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {string | null | undefined} [origin]
     * @returns
     *
     * @param {Error | VFileMessage | string} causeOrReason
     *   Reason for message, should use markdown.
     * @param {Node | NodeLike | Options | Point | Position | string | null | undefined} [optionsOrParentOrPlace]
     *   Configuration (optional).
     * @param {string | null | undefined} [origin]
     *   Place in code where the message originates (example:
     *   `'my-package:my-rule'` or `'my-rule'`).
     * @returns
     *   Instance of `VFileMessage`.
     */
    // eslint-disable-next-line complexity
    constructor(causeOrReason, optionsOrParentOrPlace, origin) {
      super();
      if (typeof optionsOrParentOrPlace === "string") {
        origin = optionsOrParentOrPlace;
        optionsOrParentOrPlace = void 0;
      }
      let reason = "";
      let options = {};
      let legacyCause = false;
      if (optionsOrParentOrPlace) {
        if ("line" in optionsOrParentOrPlace && "column" in optionsOrParentOrPlace) {
          options = { place: optionsOrParentOrPlace };
        } else if ("start" in optionsOrParentOrPlace && "end" in optionsOrParentOrPlace) {
          options = { place: optionsOrParentOrPlace };
        } else if ("type" in optionsOrParentOrPlace) {
          options = {
            ancestors: [optionsOrParentOrPlace],
            place: optionsOrParentOrPlace.position
          };
        } else {
          options = { ...optionsOrParentOrPlace };
        }
      }
      if (typeof causeOrReason === "string") {
        reason = causeOrReason;
      } else if (!options.cause && causeOrReason) {
        legacyCause = true;
        reason = causeOrReason.message;
        options.cause = causeOrReason;
      }
      if (!options.ruleId && !options.source && typeof origin === "string") {
        const index2 = origin.indexOf(":");
        if (index2 === -1) {
          options.ruleId = origin;
        } else {
          options.source = origin.slice(0, index2);
          options.ruleId = origin.slice(index2 + 1);
        }
      }
      if (!options.place && options.ancestors && options.ancestors) {
        const parent = options.ancestors[options.ancestors.length - 1];
        if (parent) {
          options.place = parent.position;
        }
      }
      const start = options.place && "start" in options.place ? options.place.start : options.place;
      this.ancestors = options.ancestors || void 0;
      this.cause = options.cause || void 0;
      this.column = start ? start.column : void 0;
      this.fatal = void 0;
      this.file = "";
      this.message = reason;
      this.line = start ? start.line : void 0;
      this.name = stringifyPosition(options.place) || "1:1";
      this.place = options.place || void 0;
      this.reason = this.message;
      this.ruleId = options.ruleId || void 0;
      this.source = options.source || void 0;
      this.stack = legacyCause && options.cause && typeof options.cause.stack === "string" ? options.cause.stack : "";
      this.actual = void 0;
      this.expected = void 0;
      this.note = void 0;
      this.url = void 0;
    }
  };
  VFileMessage.prototype.file = "";
  VFileMessage.prototype.name = "";
  VFileMessage.prototype.reason = "";
  VFileMessage.prototype.message = "";
  VFileMessage.prototype.stack = "";
  VFileMessage.prototype.column = void 0;
  VFileMessage.prototype.line = void 0;
  VFileMessage.prototype.ancestors = void 0;
  VFileMessage.prototype.cause = void 0;
  VFileMessage.prototype.fatal = void 0;
  VFileMessage.prototype.place = void 0;
  VFileMessage.prototype.ruleId = void 0;
  VFileMessage.prototype.source = void 0;

  // node_modules/vfile/lib/minpath.browser.js
  var minpath = { basename, dirname, extname, join: join2, sep: "/" };
  function basename(path, extname2) {
    if (extname2 !== void 0 && typeof extname2 !== "string") {
      throw new TypeError('"ext" argument must be a string');
    }
    assertPath(path);
    let start = 0;
    let end = -1;
    let index2 = path.length;
    let seenNonSlash;
    if (extname2 === void 0 || extname2.length === 0 || extname2.length > path.length) {
      while (index2--) {
        if (path.codePointAt(index2) === 47) {
          if (seenNonSlash) {
            start = index2 + 1;
            break;
          }
        } else if (end < 0) {
          seenNonSlash = true;
          end = index2 + 1;
        }
      }
      return end < 0 ? "" : path.slice(start, end);
    }
    if (extname2 === path) {
      return "";
    }
    let firstNonSlashEnd = -1;
    let extnameIndex = extname2.length - 1;
    while (index2--) {
      if (path.codePointAt(index2) === 47) {
        if (seenNonSlash) {
          start = index2 + 1;
          break;
        }
      } else {
        if (firstNonSlashEnd < 0) {
          seenNonSlash = true;
          firstNonSlashEnd = index2 + 1;
        }
        if (extnameIndex > -1) {
          if (path.codePointAt(index2) === extname2.codePointAt(extnameIndex--)) {
            if (extnameIndex < 0) {
              end = index2;
            }
          } else {
            extnameIndex = -1;
            end = firstNonSlashEnd;
          }
        }
      }
    }
    if (start === end) {
      end = firstNonSlashEnd;
    } else if (end < 0) {
      end = path.length;
    }
    return path.slice(start, end);
  }
  function dirname(path) {
    assertPath(path);
    if (path.length === 0) {
      return ".";
    }
    let end = -1;
    let index2 = path.length;
    let unmatchedSlash;
    while (--index2) {
      if (path.codePointAt(index2) === 47) {
        if (unmatchedSlash) {
          end = index2;
          break;
        }
      } else if (!unmatchedSlash) {
        unmatchedSlash = true;
      }
    }
    return end < 0 ? path.codePointAt(0) === 47 ? "/" : "." : end === 1 && path.codePointAt(0) === 47 ? "//" : path.slice(0, end);
  }
  function extname(path) {
    assertPath(path);
    let index2 = path.length;
    let end = -1;
    let startPart = 0;
    let startDot = -1;
    let preDotState = 0;
    let unmatchedSlash;
    while (index2--) {
      const code2 = path.codePointAt(index2);
      if (code2 === 47) {
        if (unmatchedSlash) {
          startPart = index2 + 1;
          break;
        }
        continue;
      }
      if (end < 0) {
        unmatchedSlash = true;
        end = index2 + 1;
      }
      if (code2 === 46) {
        if (startDot < 0) {
          startDot = index2;
        } else if (preDotState !== 1) {
          preDotState = 1;
        }
      } else if (startDot > -1) {
        preDotState = -1;
      }
    }
    if (startDot < 0 || end < 0 || // We saw a non-dot character immediately before the dot.
    preDotState === 0 || // The (right-most) trimmed path component is exactly `..`.
    preDotState === 1 && startDot === end - 1 && startDot === startPart + 1) {
      return "";
    }
    return path.slice(startDot, end);
  }
  function join2(...segments) {
    let index2 = -1;
    let joined;
    while (++index2 < segments.length) {
      assertPath(segments[index2]);
      if (segments[index2]) {
        joined = joined === void 0 ? segments[index2] : joined + "/" + segments[index2];
      }
    }
    return joined === void 0 ? "." : normalize(joined);
  }
  function normalize(path) {
    assertPath(path);
    const absolute = path.codePointAt(0) === 47;
    let value = normalizeString(path, !absolute);
    if (value.length === 0 && !absolute) {
      value = ".";
    }
    if (value.length > 0 && path.codePointAt(path.length - 1) === 47) {
      value += "/";
    }
    return absolute ? "/" + value : value;
  }
  function normalizeString(path, allowAboveRoot) {
    let result = "";
    let lastSegmentLength = 0;
    let lastSlash = -1;
    let dots = 0;
    let index2 = -1;
    let code2;
    let lastSlashIndex;
    while (++index2 <= path.length) {
      if (index2 < path.length) {
        code2 = path.codePointAt(index2);
      } else if (code2 === 47) {
        break;
      } else {
        code2 = 47;
      }
      if (code2 === 47) {
        if (lastSlash === index2 - 1 || dots === 1) {
        } else if (lastSlash !== index2 - 1 && dots === 2) {
          if (result.length < 2 || lastSegmentLength !== 2 || result.codePointAt(result.length - 1) !== 46 || result.codePointAt(result.length - 2) !== 46) {
            if (result.length > 2) {
              lastSlashIndex = result.lastIndexOf("/");
              if (lastSlashIndex !== result.length - 1) {
                if (lastSlashIndex < 0) {
                  result = "";
                  lastSegmentLength = 0;
                } else {
                  result = result.slice(0, lastSlashIndex);
                  lastSegmentLength = result.length - 1 - result.lastIndexOf("/");
                }
                lastSlash = index2;
                dots = 0;
                continue;
              }
            } else if (result.length > 0) {
              result = "";
              lastSegmentLength = 0;
              lastSlash = index2;
              dots = 0;
              continue;
            }
          }
          if (allowAboveRoot) {
            result = result.length > 0 ? result + "/.." : "..";
            lastSegmentLength = 2;
          }
        } else {
          if (result.length > 0) {
            result += "/" + path.slice(lastSlash + 1, index2);
          } else {
            result = path.slice(lastSlash + 1, index2);
          }
          lastSegmentLength = index2 - lastSlash - 1;
        }
        lastSlash = index2;
        dots = 0;
      } else if (code2 === 46 && dots > -1) {
        dots++;
      } else {
        dots = -1;
      }
    }
    return result;
  }
  function assertPath(path) {
    if (typeof path !== "string") {
      throw new TypeError(
        "Path must be a string. Received " + JSON.stringify(path)
      );
    }
  }

  // node_modules/vfile/lib/minproc.browser.js
  var minproc = { cwd };
  function cwd() {
    return "/";
  }

  // node_modules/vfile/lib/minurl.shared.js
  function isUrl(fileUrlOrPath) {
    return Boolean(
      fileUrlOrPath !== null && typeof fileUrlOrPath === "object" && "href" in fileUrlOrPath && fileUrlOrPath.href && "protocol" in fileUrlOrPath && fileUrlOrPath.protocol && // @ts-expect-error: indexing is fine.
      fileUrlOrPath.auth === void 0
    );
  }

  // node_modules/vfile/lib/minurl.browser.js
  function urlToPath(path) {
    if (typeof path === "string") {
      path = new URL(path);
    } else if (!isUrl(path)) {
      const error = new TypeError(
        'The "path" argument must be of type string or an instance of URL. Received `' + path + "`"
      );
      error.code = "ERR_INVALID_ARG_TYPE";
      throw error;
    }
    if (path.protocol !== "file:") {
      const error = new TypeError("The URL must be of scheme file");
      error.code = "ERR_INVALID_URL_SCHEME";
      throw error;
    }
    return getPathFromURLPosix(path);
  }
  function getPathFromURLPosix(url) {
    if (url.hostname !== "") {
      const error = new TypeError(
        'File URL host must be "localhost" or empty on darwin'
      );
      error.code = "ERR_INVALID_FILE_URL_HOST";
      throw error;
    }
    const pathname = url.pathname;
    let index2 = -1;
    while (++index2 < pathname.length) {
      if (pathname.codePointAt(index2) === 37 && pathname.codePointAt(index2 + 1) === 50) {
        const third = pathname.codePointAt(index2 + 2);
        if (third === 70 || third === 102) {
          const error = new TypeError(
            "File URL path must not include encoded / characters"
          );
          error.code = "ERR_INVALID_FILE_URL_PATH";
          throw error;
        }
      }
    }
    return decodeURIComponent(pathname);
  }

  // node_modules/vfile/lib/index.js
  var order = (
    /** @type {const} */
    [
      "history",
      "path",
      "basename",
      "stem",
      "extname",
      "dirname"
    ]
  );
  var VFile = class {
    /**
     * Create a new virtual file.
     *
     * `options` is treated as:
     *
     * *   `string` or `Uint8Array` — `{value: options}`
     * *   `URL` — `{path: options}`
     * *   `VFile` — shallow copies its data over to the new file
     * *   `object` — all fields are shallow copied over to the new file
     *
     * Path related fields are set in the following order (least specific to
     * most specific): `history`, `path`, `basename`, `stem`, `extname`,
     * `dirname`.
     *
     * You cannot set `dirname` or `extname` without setting either `history`,
     * `path`, `basename`, or `stem` too.
     *
     * @param {Compatible | null | undefined} [value]
     *   File value.
     * @returns
     *   New instance.
     */
    constructor(value) {
      let options;
      if (!value) {
        options = {};
      } else if (isUrl(value)) {
        options = { path: value };
      } else if (typeof value === "string" || isUint8Array(value)) {
        options = { value };
      } else {
        options = value;
      }
      this.cwd = "cwd" in options ? "" : minproc.cwd();
      this.data = {};
      this.history = [];
      this.messages = [];
      this.value;
      this.map;
      this.result;
      this.stored;
      let index2 = -1;
      while (++index2 < order.length) {
        const field2 = order[index2];
        if (field2 in options && options[field2] !== void 0 && options[field2] !== null) {
          this[field2] = field2 === "history" ? [...options[field2]] : options[field2];
        }
      }
      let field;
      for (field in options) {
        if (!order.includes(field)) {
          this[field] = options[field];
        }
      }
    }
    /**
     * Get the basename (including extname) (example: `'index.min.js'`).
     *
     * @returns {string | undefined}
     *   Basename.
     */
    get basename() {
      return typeof this.path === "string" ? minpath.basename(this.path) : void 0;
    }
    /**
     * Set basename (including extname) (`'index.min.js'`).
     *
     * Cannot contain path separators (`'/'` on unix, macOS, and browsers, `'\'`
     * on windows).
     * Cannot be nullified (use `file.path = file.dirname` instead).
     *
     * @param {string} basename
     *   Basename.
     * @returns {undefined}
     *   Nothing.
     */
    set basename(basename2) {
      assertNonEmpty(basename2, "basename");
      assertPart(basename2, "basename");
      this.path = minpath.join(this.dirname || "", basename2);
    }
    /**
     * Get the parent path (example: `'~'`).
     *
     * @returns {string | undefined}
     *   Dirname.
     */
    get dirname() {
      return typeof this.path === "string" ? minpath.dirname(this.path) : void 0;
    }
    /**
     * Set the parent path (example: `'~'`).
     *
     * Cannot be set if there’s no `path` yet.
     *
     * @param {string | undefined} dirname
     *   Dirname.
     * @returns {undefined}
     *   Nothing.
     */
    set dirname(dirname2) {
      assertPath2(this.basename, "dirname");
      this.path = minpath.join(dirname2 || "", this.basename);
    }
    /**
     * Get the extname (including dot) (example: `'.js'`).
     *
     * @returns {string | undefined}
     *   Extname.
     */
    get extname() {
      return typeof this.path === "string" ? minpath.extname(this.path) : void 0;
    }
    /**
     * Set the extname (including dot) (example: `'.js'`).
     *
     * Cannot contain path separators (`'/'` on unix, macOS, and browsers, `'\'`
     * on windows).
     * Cannot be set if there’s no `path` yet.
     *
     * @param {string | undefined} extname
     *   Extname.
     * @returns {undefined}
     *   Nothing.
     */
    set extname(extname2) {
      assertPart(extname2, "extname");
      assertPath2(this.dirname, "extname");
      if (extname2) {
        if (extname2.codePointAt(0) !== 46) {
          throw new Error("`extname` must start with `.`");
        }
        if (extname2.includes(".", 1)) {
          throw new Error("`extname` cannot contain multiple dots");
        }
      }
      this.path = minpath.join(this.dirname, this.stem + (extname2 || ""));
    }
    /**
     * Get the full path (example: `'~/index.min.js'`).
     *
     * @returns {string}
     *   Path.
     */
    get path() {
      return this.history[this.history.length - 1];
    }
    /**
     * Set the full path (example: `'~/index.min.js'`).
     *
     * Cannot be nullified.
     * You can set a file URL (a `URL` object with a `file:` protocol) which will
     * be turned into a path with `url.fileURLToPath`.
     *
     * @param {URL | string} path
     *   Path.
     * @returns {undefined}
     *   Nothing.
     */
    set path(path) {
      if (isUrl(path)) {
        path = urlToPath(path);
      }
      assertNonEmpty(path, "path");
      if (this.path !== path) {
        this.history.push(path);
      }
    }
    /**
     * Get the stem (basename w/o extname) (example: `'index.min'`).
     *
     * @returns {string | undefined}
     *   Stem.
     */
    get stem() {
      return typeof this.path === "string" ? minpath.basename(this.path, this.extname) : void 0;
    }
    /**
     * Set the stem (basename w/o extname) (example: `'index.min'`).
     *
     * Cannot contain path separators (`'/'` on unix, macOS, and browsers, `'\'`
     * on windows).
     * Cannot be nullified (use `file.path = file.dirname` instead).
     *
     * @param {string} stem
     *   Stem.
     * @returns {undefined}
     *   Nothing.
     */
    set stem(stem) {
      assertNonEmpty(stem, "stem");
      assertPart(stem, "stem");
      this.path = minpath.join(this.dirname || "", stem + (this.extname || ""));
    }
    // Normal prototypal methods.
    /**
     * Create a fatal message for `reason` associated with the file.
     *
     * The `fatal` field of the message is set to `true` (error; file not usable)
     * and the `file` field is set to the current file path.
     * The message is added to the `messages` field on `file`.
     *
     * > 🪦 **Note**: also has obsolete signatures.
     *
     * @overload
     * @param {string} reason
     * @param {MessageOptions | null | undefined} [options]
     * @returns {never}
     *
     * @overload
     * @param {string} reason
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns {never}
     *
     * @overload
     * @param {string} reason
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns {never}
     *
     * @overload
     * @param {string} reason
     * @param {string | null | undefined} [origin]
     * @returns {never}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns {never}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns {never}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {string | null | undefined} [origin]
     * @returns {never}
     *
     * @param {Error | VFileMessage | string} causeOrReason
     *   Reason for message, should use markdown.
     * @param {Node | NodeLike | MessageOptions | Point | Position | string | null | undefined} [optionsOrParentOrPlace]
     *   Configuration (optional).
     * @param {string | null | undefined} [origin]
     *   Place in code where the message originates (example:
     *   `'my-package:my-rule'` or `'my-rule'`).
     * @returns {never}
     *   Never.
     * @throws {VFileMessage}
     *   Message.
     */
    fail(causeOrReason, optionsOrParentOrPlace, origin) {
      const message = this.message(causeOrReason, optionsOrParentOrPlace, origin);
      message.fatal = true;
      throw message;
    }
    /**
     * Create an info message for `reason` associated with the file.
     *
     * The `fatal` field of the message is set to `undefined` (info; change
     * likely not needed) and the `file` field is set to the current file path.
     * The message is added to the `messages` field on `file`.
     *
     * > 🪦 **Note**: also has obsolete signatures.
     *
     * @overload
     * @param {string} reason
     * @param {MessageOptions | null | undefined} [options]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {string} reason
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {string} reason
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {string} reason
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @param {Error | VFileMessage | string} causeOrReason
     *   Reason for message, should use markdown.
     * @param {Node | NodeLike | MessageOptions | Point | Position | string | null | undefined} [optionsOrParentOrPlace]
     *   Configuration (optional).
     * @param {string | null | undefined} [origin]
     *   Place in code where the message originates (example:
     *   `'my-package:my-rule'` or `'my-rule'`).
     * @returns {VFileMessage}
     *   Message.
     */
    info(causeOrReason, optionsOrParentOrPlace, origin) {
      const message = this.message(causeOrReason, optionsOrParentOrPlace, origin);
      message.fatal = void 0;
      return message;
    }
    /**
     * Create a message for `reason` associated with the file.
     *
     * The `fatal` field of the message is set to `false` (warning; change may be
     * needed) and the `file` field is set to the current file path.
     * The message is added to the `messages` field on `file`.
     *
     * > 🪦 **Note**: also has obsolete signatures.
     *
     * @overload
     * @param {string} reason
     * @param {MessageOptions | null | undefined} [options]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {string} reason
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {string} reason
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {string} reason
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Node | NodeLike | null | undefined} parent
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {Point | Position | null | undefined} place
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @overload
     * @param {Error | VFileMessage} cause
     * @param {string | null | undefined} [origin]
     * @returns {VFileMessage}
     *
     * @param {Error | VFileMessage | string} causeOrReason
     *   Reason for message, should use markdown.
     * @param {Node | NodeLike | MessageOptions | Point | Position | string | null | undefined} [optionsOrParentOrPlace]
     *   Configuration (optional).
     * @param {string | null | undefined} [origin]
     *   Place in code where the message originates (example:
     *   `'my-package:my-rule'` or `'my-rule'`).
     * @returns {VFileMessage}
     *   Message.
     */
    message(causeOrReason, optionsOrParentOrPlace, origin) {
      const message = new VFileMessage(
        // @ts-expect-error: the overloads are fine.
        causeOrReason,
        optionsOrParentOrPlace,
        origin
      );
      if (this.path) {
        message.name = this.path + ":" + message.name;
        message.file = this.path;
      }
      message.fatal = false;
      this.messages.push(message);
      return message;
    }
    /**
     * Serialize the file.
     *
     * > **Note**: which encodings are supported depends on the engine.
     * > For info on Node.js, see:
     * > <https://nodejs.org/api/util.html#whatwg-supported-encodings>.
     *
     * @param {string | null | undefined} [encoding='utf8']
     *   Character encoding to understand `value` as when it’s a `Uint8Array`
     *   (default: `'utf-8'`).
     * @returns {string}
     *   Serialized file.
     */
    toString(encoding) {
      if (this.value === void 0) {
        return "";
      }
      if (typeof this.value === "string") {
        return this.value;
      }
      const decoder = new TextDecoder(encoding || void 0);
      return decoder.decode(this.value);
    }
  };
  function assertPart(part, name) {
    if (part && part.includes(minpath.sep)) {
      throw new Error(
        "`" + name + "` cannot be a path: did not expect `" + minpath.sep + "`"
      );
    }
  }
  function assertNonEmpty(part, name) {
    if (!part) {
      throw new Error("`" + name + "` cannot be empty");
    }
  }
  function assertPath2(path, name) {
    if (!path) {
      throw new Error("Setting `" + name + "` requires `path` to be set too");
    }
  }
  function isUint8Array(value) {
    return Boolean(
      value && typeof value === "object" && "byteLength" in value && "byteOffset" in value
    );
  }

  // node_modules/unified/lib/callable-instance.js
  var CallableInstance = (
    /**
     * @type {new <Parameters extends Array<unknown>, Result>(property: string | symbol) => (...parameters: Parameters) => Result}
     */
    /** @type {unknown} */
    /**
     * @this {Function}
     * @param {string | symbol} property
     * @returns {(...parameters: Array<unknown>) => unknown}
     */
    (function(property) {
      const self2 = this;
      const constr = self2.constructor;
      const proto = (
        /** @type {Record<string | symbol, Function>} */
        // Prototypes do exist.
        // type-coverage:ignore-next-line
        constr.prototype
      );
      const value = proto[property];
      const apply2 = function() {
        return value.apply(apply2, arguments);
      };
      Object.setPrototypeOf(apply2, proto);
      return apply2;
    })
  );

  // node_modules/unified/lib/index.js
  var own4 = {}.hasOwnProperty;
  var Processor = class _Processor extends CallableInstance {
    /**
     * Create a processor.
     */
    constructor() {
      super("copy");
      this.Compiler = void 0;
      this.Parser = void 0;
      this.attachers = [];
      this.compiler = void 0;
      this.freezeIndex = -1;
      this.frozen = void 0;
      this.namespace = {};
      this.parser = void 0;
      this.transformers = trough();
    }
    /**
     * Copy a processor.
     *
     * @deprecated
     *   This is a private internal method and should not be used.
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *   New *unfrozen* processor ({@linkcode Processor}) that is
     *   configured to work the same as its ancestor.
     *   When the descendant processor is configured in the future it does not
     *   affect the ancestral processor.
     */
    copy() {
      const destination = (
        /** @type {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>} */
        new _Processor()
      );
      let index2 = -1;
      while (++index2 < this.attachers.length) {
        const attacher = this.attachers[index2];
        destination.use(...attacher);
      }
      destination.data((0, import_extend.default)(true, {}, this.namespace));
      return destination;
    }
    /**
     * Configure the processor with info available to all plugins.
     * Information is stored in an object.
     *
     * Typically, options can be given to a specific plugin, but sometimes it
     * makes sense to have information shared with several plugins.
     * For example, a list of HTML elements that are self-closing, which is
     * needed during all phases.
     *
     * > **Note**: setting information cannot occur on *frozen* processors.
     * > Call the processor first to create a new unfrozen processor.
     *
     * > **Note**: to register custom data in TypeScript, augment the
     * > {@linkcode Data} interface.
     *
     * @example
     *   This example show how to get and set info:
     *
     *   ```js
     *   import {unified} from 'unified'
     *
     *   const processor = unified().data('alpha', 'bravo')
     *
     *   processor.data('alpha') // => 'bravo'
     *
     *   processor.data() // => {alpha: 'bravo'}
     *
     *   processor.data({charlie: 'delta'})
     *
     *   processor.data() // => {charlie: 'delta'}
     *   ```
     *
     * @template {keyof Data} Key
     *
     * @overload
     * @returns {Data}
     *
     * @overload
     * @param {Data} dataset
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *
     * @overload
     * @param {Key} key
     * @returns {Data[Key]}
     *
     * @overload
     * @param {Key} key
     * @param {Data[Key]} value
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *
     * @param {Data | Key} [key]
     *   Key to get or set, or entire dataset to set, or nothing to get the
     *   entire dataset (optional).
     * @param {Data[Key]} [value]
     *   Value to set (optional).
     * @returns {unknown}
     *   The current processor when setting, the value at `key` when getting, or
     *   the entire dataset when getting without key.
     */
    data(key3, value) {
      if (typeof key3 === "string") {
        if (arguments.length === 2) {
          assertUnfrozen("data", this.frozen);
          this.namespace[key3] = value;
          return this;
        }
        return own4.call(this.namespace, key3) && this.namespace[key3] || void 0;
      }
      if (key3) {
        assertUnfrozen("data", this.frozen);
        this.namespace = key3;
        return this;
      }
      return this.namespace;
    }
    /**
     * Freeze a processor.
     *
     * Frozen processors are meant to be extended and not to be configured
     * directly.
     *
     * When a processor is frozen it cannot be unfrozen.
     * New processors working the same way can be created by calling the
     * processor.
     *
     * It’s possible to freeze processors explicitly by calling `.freeze()`.
     * Processors freeze automatically when `.parse()`, `.run()`, `.runSync()`,
     * `.stringify()`, `.process()`, or `.processSync()` are called.
     *
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *   The current processor.
     */
    freeze() {
      if (this.frozen) {
        return this;
      }
      const self2 = (
        /** @type {Processor} */
        /** @type {unknown} */
        this
      );
      while (++this.freezeIndex < this.attachers.length) {
        const [attacher, ...options] = this.attachers[this.freezeIndex];
        if (options[0] === false) {
          continue;
        }
        if (options[0] === true) {
          options[0] = void 0;
        }
        const transformer = attacher.call(self2, ...options);
        if (typeof transformer === "function") {
          this.transformers.use(transformer);
        }
      }
      this.frozen = true;
      this.freezeIndex = Number.POSITIVE_INFINITY;
      return this;
    }
    /**
     * Parse text to a syntax tree.
     *
     * > **Note**: `parse` freezes the processor if not already *frozen*.
     *
     * > **Note**: `parse` performs the parse phase, not the run phase or other
     * > phases.
     *
     * @param {Compatible | undefined} [file]
     *   file to parse (optional); typically `string` or `VFile`; any value
     *   accepted as `x` in `new VFile(x)`.
     * @returns {ParseTree extends undefined ? Node : ParseTree}
     *   Syntax tree representing `file`.
     */
    parse(file) {
      this.freeze();
      const realFile = vfile(file);
      const parser2 = this.parser || this.Parser;
      assertParser("parse", parser2);
      return parser2(String(realFile), realFile);
    }
    /**
     * Process the given file as configured on the processor.
     *
     * > **Note**: `process` freezes the processor if not already *frozen*.
     *
     * > **Note**: `process` performs the parse, run, and stringify phases.
     *
     * @overload
     * @param {Compatible | undefined} file
     * @param {ProcessCallback<VFileWithOutput<CompileResult>>} done
     * @returns {undefined}
     *
     * @overload
     * @param {Compatible | undefined} [file]
     * @returns {Promise<VFileWithOutput<CompileResult>>}
     *
     * @param {Compatible | undefined} [file]
     *   File (optional); typically `string` or `VFile`]; any value accepted as
     *   `x` in `new VFile(x)`.
     * @param {ProcessCallback<VFileWithOutput<CompileResult>> | undefined} [done]
     *   Callback (optional).
     * @returns {Promise<VFile> | undefined}
     *   Nothing if `done` is given.
     *   Otherwise a promise, rejected with a fatal error or resolved with the
     *   processed file.
     *
     *   The parsed, transformed, and compiled value is available at
     *   `file.value` (see note).
     *
     *   > **Note**: unified typically compiles by serializing: most
     *   > compilers return `string` (or `Uint8Array`).
     *   > Some compilers, such as the one configured with
     *   > [`rehype-react`][rehype-react], return other values (in this case, a
     *   > React tree).
     *   > If you’re using a compiler that doesn’t serialize, expect different
     *   > result values.
     *   >
     *   > To register custom results in TypeScript, add them to
     *   > {@linkcode CompileResultMap}.
     *
     *   [rehype-react]: https://github.com/rehypejs/rehype-react
     */
    process(file, done) {
      const self2 = this;
      this.freeze();
      assertParser("process", this.parser || this.Parser);
      assertCompiler("process", this.compiler || this.Compiler);
      return done ? executor(void 0, done) : new Promise(executor);
      function executor(resolve, reject) {
        const realFile = vfile(file);
        const parseTree = (
          /** @type {HeadTree extends undefined ? Node : HeadTree} */
          /** @type {unknown} */
          self2.parse(realFile)
        );
        self2.run(parseTree, realFile, function(error, tree, file2) {
          if (error || !tree || !file2) {
            return realDone(error);
          }
          const compileTree = (
            /** @type {CompileTree extends undefined ? Node : CompileTree} */
            /** @type {unknown} */
            tree
          );
          const compileResult = self2.stringify(compileTree, file2);
          if (looksLikeAValue(compileResult)) {
            file2.value = compileResult;
          } else {
            file2.result = compileResult;
          }
          realDone(
            error,
            /** @type {VFileWithOutput<CompileResult>} */
            file2
          );
        });
        function realDone(error, file2) {
          if (error || !file2) {
            reject(error);
          } else if (resolve) {
            resolve(file2);
          } else {
            ok2(done, "`done` is defined if `resolve` is not");
            done(void 0, file2);
          }
        }
      }
    }
    /**
     * Process the given file as configured on the processor.
     *
     * An error is thrown if asynchronous transforms are configured.
     *
     * > **Note**: `processSync` freezes the processor if not already *frozen*.
     *
     * > **Note**: `processSync` performs the parse, run, and stringify phases.
     *
     * @param {Compatible | undefined} [file]
     *   File (optional); typically `string` or `VFile`; any value accepted as
     *   `x` in `new VFile(x)`.
     * @returns {VFileWithOutput<CompileResult>}
     *   The processed file.
     *
     *   The parsed, transformed, and compiled value is available at
     *   `file.value` (see note).
     *
     *   > **Note**: unified typically compiles by serializing: most
     *   > compilers return `string` (or `Uint8Array`).
     *   > Some compilers, such as the one configured with
     *   > [`rehype-react`][rehype-react], return other values (in this case, a
     *   > React tree).
     *   > If you’re using a compiler that doesn’t serialize, expect different
     *   > result values.
     *   >
     *   > To register custom results in TypeScript, add them to
     *   > {@linkcode CompileResultMap}.
     *
     *   [rehype-react]: https://github.com/rehypejs/rehype-react
     */
    processSync(file) {
      let complete = false;
      let result;
      this.freeze();
      assertParser("processSync", this.parser || this.Parser);
      assertCompiler("processSync", this.compiler || this.Compiler);
      this.process(file, realDone);
      assertDone("processSync", "process", complete);
      ok2(result, "we either bailed on an error or have a tree");
      return result;
      function realDone(error, file2) {
        complete = true;
        bail(error);
        result = file2;
      }
    }
    /**
     * Run *transformers* on a syntax tree.
     *
     * > **Note**: `run` freezes the processor if not already *frozen*.
     *
     * > **Note**: `run` performs the run phase, not other phases.
     *
     * @overload
     * @param {HeadTree extends undefined ? Node : HeadTree} tree
     * @param {RunCallback<TailTree extends undefined ? Node : TailTree>} done
     * @returns {undefined}
     *
     * @overload
     * @param {HeadTree extends undefined ? Node : HeadTree} tree
     * @param {Compatible | undefined} file
     * @param {RunCallback<TailTree extends undefined ? Node : TailTree>} done
     * @returns {undefined}
     *
     * @overload
     * @param {HeadTree extends undefined ? Node : HeadTree} tree
     * @param {Compatible | undefined} [file]
     * @returns {Promise<TailTree extends undefined ? Node : TailTree>}
     *
     * @param {HeadTree extends undefined ? Node : HeadTree} tree
     *   Tree to transform and inspect.
     * @param {(
     *   RunCallback<TailTree extends undefined ? Node : TailTree> |
     *   Compatible
     * )} [file]
     *   File associated with `node` (optional); any value accepted as `x` in
     *   `new VFile(x)`.
     * @param {RunCallback<TailTree extends undefined ? Node : TailTree>} [done]
     *   Callback (optional).
     * @returns {Promise<TailTree extends undefined ? Node : TailTree> | undefined}
     *   Nothing if `done` is given.
     *   Otherwise, a promise rejected with a fatal error or resolved with the
     *   transformed tree.
     */
    run(tree, file, done) {
      assertNode(tree);
      this.freeze();
      const transformers = this.transformers;
      if (!done && typeof file === "function") {
        done = file;
        file = void 0;
      }
      return done ? executor(void 0, done) : new Promise(executor);
      function executor(resolve, reject) {
        ok2(
          typeof file !== "function",
          "`file` can\u2019t be a `done` anymore, we checked"
        );
        const realFile = vfile(file);
        transformers.run(tree, realFile, realDone);
        function realDone(error, outputTree, file2) {
          const resultingTree = (
            /** @type {TailTree extends undefined ? Node : TailTree} */
            outputTree || tree
          );
          if (error) {
            reject(error);
          } else if (resolve) {
            resolve(resultingTree);
          } else {
            ok2(done, "`done` is defined if `resolve` is not");
            done(void 0, resultingTree, file2);
          }
        }
      }
    }
    /**
     * Run *transformers* on a syntax tree.
     *
     * An error is thrown if asynchronous transforms are configured.
     *
     * > **Note**: `runSync` freezes the processor if not already *frozen*.
     *
     * > **Note**: `runSync` performs the run phase, not other phases.
     *
     * @param {HeadTree extends undefined ? Node : HeadTree} tree
     *   Tree to transform and inspect.
     * @param {Compatible | undefined} [file]
     *   File associated with `node` (optional); any value accepted as `x` in
     *   `new VFile(x)`.
     * @returns {TailTree extends undefined ? Node : TailTree}
     *   Transformed tree.
     */
    runSync(tree, file) {
      let complete = false;
      let result;
      this.run(tree, file, realDone);
      assertDone("runSync", "run", complete);
      ok2(result, "we either bailed on an error or have a tree");
      return result;
      function realDone(error, tree2) {
        bail(error);
        result = tree2;
        complete = true;
      }
    }
    /**
     * Compile a syntax tree.
     *
     * > **Note**: `stringify` freezes the processor if not already *frozen*.
     *
     * > **Note**: `stringify` performs the stringify phase, not the run phase
     * > or other phases.
     *
     * @param {CompileTree extends undefined ? Node : CompileTree} tree
     *   Tree to compile.
     * @param {Compatible | undefined} [file]
     *   File associated with `node` (optional); any value accepted as `x` in
     *   `new VFile(x)`.
     * @returns {CompileResult extends undefined ? Value : CompileResult}
     *   Textual representation of the tree (see note).
     *
     *   > **Note**: unified typically compiles by serializing: most compilers
     *   > return `string` (or `Uint8Array`).
     *   > Some compilers, such as the one configured with
     *   > [`rehype-react`][rehype-react], return other values (in this case, a
     *   > React tree).
     *   > If you’re using a compiler that doesn’t serialize, expect different
     *   > result values.
     *   >
     *   > To register custom results in TypeScript, add them to
     *   > {@linkcode CompileResultMap}.
     *
     *   [rehype-react]: https://github.com/rehypejs/rehype-react
     */
    stringify(tree, file) {
      this.freeze();
      const realFile = vfile(file);
      const compiler2 = this.compiler || this.Compiler;
      assertCompiler("stringify", compiler2);
      assertNode(tree);
      return compiler2(tree, realFile);
    }
    /**
     * Configure the processor to use a plugin, a list of usable values, or a
     * preset.
     *
     * If the processor is already using a plugin, the previous plugin
     * configuration is changed based on the options that are passed in.
     * In other words, the plugin is not added a second time.
     *
     * > **Note**: `use` cannot be called on *frozen* processors.
     * > Call the processor first to create a new unfrozen processor.
     *
     * @example
     *   There are many ways to pass plugins to `.use()`.
     *   This example gives an overview:
     *
     *   ```js
     *   import {unified} from 'unified'
     *
     *   unified()
     *     // Plugin with options:
     *     .use(pluginA, {x: true, y: true})
     *     // Passing the same plugin again merges configuration (to `{x: true, y: false, z: true}`):
     *     .use(pluginA, {y: false, z: true})
     *     // Plugins:
     *     .use([pluginB, pluginC])
     *     // Two plugins, the second with options:
     *     .use([pluginD, [pluginE, {}]])
     *     // Preset with plugins and settings:
     *     .use({plugins: [pluginF, [pluginG, {}]], settings: {position: false}})
     *     // Settings only:
     *     .use({settings: {position: false}})
     *   ```
     *
     * @template {Array<unknown>} [Parameters=[]]
     * @template {Node | string | undefined} [Input=undefined]
     * @template [Output=Input]
     *
     * @overload
     * @param {Preset | null | undefined} [preset]
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *
     * @overload
     * @param {PluggableList} list
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *
     * @overload
     * @param {Plugin<Parameters, Input, Output>} plugin
     * @param {...(Parameters | [boolean])} parameters
     * @returns {UsePlugin<ParseTree, HeadTree, TailTree, CompileTree, CompileResult, Input, Output>}
     *
     * @param {PluggableList | Plugin | Preset | null | undefined} value
     *   Usable value.
     * @param {...unknown} parameters
     *   Parameters, when a plugin is given as a usable value.
     * @returns {Processor<ParseTree, HeadTree, TailTree, CompileTree, CompileResult>}
     *   Current processor.
     */
    use(value, ...parameters) {
      const attachers = this.attachers;
      const namespace = this.namespace;
      assertUnfrozen("use", this.frozen);
      if (value === null || value === void 0) {
      } else if (typeof value === "function") {
        addPlugin(value, parameters);
      } else if (typeof value === "object") {
        if (Array.isArray(value)) {
          addList(value);
        } else {
          addPreset(value);
        }
      } else {
        throw new TypeError("Expected usable value, not `" + value + "`");
      }
      return this;
      function add(value2) {
        if (typeof value2 === "function") {
          addPlugin(value2, []);
        } else if (typeof value2 === "object") {
          if (Array.isArray(value2)) {
            const [plugin, ...parameters2] = (
              /** @type {PluginTuple<Array<unknown>>} */
              value2
            );
            addPlugin(plugin, parameters2);
          } else {
            addPreset(value2);
          }
        } else {
          throw new TypeError("Expected usable value, not `" + value2 + "`");
        }
      }
      function addPreset(result) {
        if (!("plugins" in result) && !("settings" in result)) {
          throw new Error(
            "Expected usable value but received an empty preset, which is probably a mistake: presets typically come with `plugins` and sometimes with `settings`, but this has neither"
          );
        }
        addList(result.plugins);
        if (result.settings) {
          namespace.settings = (0, import_extend.default)(true, namespace.settings, result.settings);
        }
      }
      function addList(plugins2) {
        let index2 = -1;
        if (plugins2 === null || plugins2 === void 0) {
        } else if (Array.isArray(plugins2)) {
          while (++index2 < plugins2.length) {
            const thing = plugins2[index2];
            add(thing);
          }
        } else {
          throw new TypeError("Expected a list of plugins, not `" + plugins2 + "`");
        }
      }
      function addPlugin(plugin, parameters2) {
        let index2 = -1;
        let entryIndex = -1;
        while (++index2 < attachers.length) {
          if (attachers[index2][0] === plugin) {
            entryIndex = index2;
            break;
          }
        }
        if (entryIndex === -1) {
          attachers.push([plugin, ...parameters2]);
        } else if (parameters2.length > 0) {
          let [primary, ...rest] = parameters2;
          const currentPrimary = attachers[entryIndex][1];
          if (isPlainObject(currentPrimary) && isPlainObject(primary)) {
            primary = (0, import_extend.default)(true, currentPrimary, primary);
          }
          attachers[entryIndex] = [plugin, primary, ...rest];
        }
      }
    }
  };
  var unified = new Processor().freeze();
  function assertParser(name, value) {
    if (typeof value !== "function") {
      throw new TypeError("Cannot `" + name + "` without `parser`");
    }
  }
  function assertCompiler(name, value) {
    if (typeof value !== "function") {
      throw new TypeError("Cannot `" + name + "` without `compiler`");
    }
  }
  function assertUnfrozen(name, frozen) {
    if (frozen) {
      throw new Error(
        "Cannot call `" + name + "` on a frozen processor.\nCreate a new processor first, by calling it: use `processor()` instead of `processor`."
      );
    }
  }
  function assertNode(node2) {
    if (!isPlainObject(node2) || typeof node2.type !== "string") {
      throw new TypeError("Expected node, got `" + node2 + "`");
    }
  }
  function assertDone(name, asyncName, complete) {
    if (!complete) {
      throw new Error(
        "`" + name + "` finished async. Use `" + asyncName + "` instead"
      );
    }
  }
  function vfile(value) {
    return looksLikeAVFile(value) ? value : new VFile(value);
  }
  function looksLikeAVFile(value) {
    return Boolean(
      value && typeof value === "object" && "message" in value && "messages" in value
    );
  }
  function looksLikeAValue(value) {
    return typeof value === "string" || isUint8Array2(value);
  }
  function isUint8Array2(value) {
    return Boolean(
      value && typeof value === "object" && "byteLength" in value && "byteOffset" in value
    );
  }

  // node_modules/@milkdown/transformer/lib/index.js
  var __typeError2 = (msg) => {
    throw TypeError(msg);
  };
  var __accessCheck2 = (obj, member, msg) => member.has(obj) || __typeError2("Cannot " + msg);
  var __privateGet2 = (obj, member, getter) => (__accessCheck2(obj, member, "read from private field"), getter ? getter.call(obj) : member.get(obj));
  var __privateAdd2 = (obj, member, value) => member.has(obj) ? __typeError2("Cannot add the same private member more than once") : member instanceof WeakSet ? member.add(obj) : member.set(obj, value);
  var __privateSet2 = (obj, member, value, setter) => (__accessCheck2(obj, member, "write to private field"), setter ? setter.call(obj, value) : member.set(obj, value), value);
  var _marks;
  var _hasText;
  var _maybeMerge;
  var _matchTarget;
  var _runNode;
  var _closeNodeAndPush;
  var _addNodeAndPush;
  var _marks2;
  var _matchTarget2;
  var _runProseNode;
  var _runProseMark;
  var _runNode2;
  var _searchType;
  var _maybeMergeChildren;
  var _createMarkdownNode;
  var _moveSpaces;
  var _closeNodeAndPush2;
  var _addNodeAndPush2;
  var _openMark;
  var _closeMark;
  var StackElement = class {
  };
  var Stack = class {
    constructor() {
      this.elements = [];
      this.size = () => {
        return this.elements.length;
      };
      this.top = () => {
        return this.elements.at(-1);
      };
      this.push = (node2) => {
        this.top()?.push(node2);
      };
      this.open = (node2) => {
        this.elements.push(node2);
      };
      this.close = () => {
        const el = this.elements.pop();
        if (!el) throw stackOverFlow();
        return el;
      };
    }
  };
  var ParserStackElement = class _ParserStackElement extends StackElement {
    constructor(type, content3, attrs) {
      super();
      this.type = type;
      this.content = content3;
      this.attrs = attrs;
    }
    push(node2, ...rest) {
      this.content.push(node2, ...rest);
    }
    pop() {
      return this.content.pop();
    }
    static create(type, content3, attrs) {
      return new _ParserStackElement(type, content3, attrs);
    }
  };
  var _ParserState = class _ParserState2 extends Stack {
    /// @internal
    constructor(schema3) {
      super();
      __privateAdd2(this, _marks);
      __privateAdd2(this, _hasText);
      __privateAdd2(this, _maybeMerge);
      __privateAdd2(this, _matchTarget);
      __privateAdd2(this, _runNode);
      __privateAdd2(this, _closeNodeAndPush);
      __privateAdd2(this, _addNodeAndPush);
      __privateSet2(this, _marks, Mark.none);
      __privateSet2(this, _hasText, (node2) => node2.isText);
      __privateSet2(this, _maybeMerge, (a2, b) => {
        if (__privateGet2(this, _hasText).call(this, a2) && __privateGet2(this, _hasText).call(this, b) && Mark.sameSet(a2.marks, b.marks))
          return this.schema.text(a2.text + b.text, a2.marks);
        return void 0;
      });
      __privateSet2(this, _matchTarget, (node2) => {
        const result = Object.values({
          ...this.schema.nodes,
          ...this.schema.marks
        }).find((x) => {
          const spec = x.spec;
          return spec.parseMarkdown.match(node2);
        });
        if (!result) throw parserMatchError(node2);
        return result;
      });
      __privateSet2(this, _runNode, (node2) => {
        const type = __privateGet2(this, _matchTarget).call(this, node2);
        const spec = type.spec;
        spec.parseMarkdown.runner(this, node2, type);
      });
      this.injectRoot = (node2, nodeType, attrs) => {
        this.openNode(nodeType, attrs);
        this.next(node2.children);
        return this;
      };
      this.openNode = (nodeType, attrs) => {
        this.open(ParserStackElement.create(nodeType, [], attrs));
        return this;
      };
      __privateSet2(this, _closeNodeAndPush, () => {
        __privateSet2(this, _marks, Mark.none);
        const element2 = this.close();
        return __privateGet2(this, _addNodeAndPush).call(this, element2.type, element2.attrs, element2.content);
      });
      this.closeNode = () => {
        try {
          __privateGet2(this, _closeNodeAndPush).call(this);
        } catch (e2) {
          console.error(e2);
        }
        return this;
      };
      __privateSet2(this, _addNodeAndPush, (nodeType, attrs, content3) => {
        const node2 = nodeType.createAndFill(attrs, content3, __privateGet2(this, _marks));
        if (!node2) throw createNodeInParserFail(nodeType, attrs, content3);
        this.push(node2);
        return node2;
      });
      this.addNode = (nodeType, attrs, content3) => {
        try {
          __privateGet2(this, _addNodeAndPush).call(this, nodeType, attrs, content3);
        } catch (e2) {
          console.error(e2);
        }
        return this;
      };
      this.openMark = (markType, attrs) => {
        const mark = markType.create(attrs);
        __privateSet2(this, _marks, mark.addToSet(__privateGet2(this, _marks)));
        return this;
      };
      this.closeMark = (markType) => {
        __privateSet2(this, _marks, markType.removeFromSet(__privateGet2(this, _marks)));
        return this;
      };
      this.addText = (text4) => {
        try {
          const topElement = this.top();
          if (!topElement) throw stackOverFlow();
          const prevNode = topElement.pop();
          const currNode = this.schema.text(text4, __privateGet2(this, _marks));
          if (!prevNode) {
            topElement.push(currNode);
            return this;
          }
          const merged = __privateGet2(this, _maybeMerge).call(this, prevNode, currNode);
          if (merged) {
            topElement.push(merged);
            return this;
          }
          topElement.push(prevNode, currNode);
          return this;
        } catch (e2) {
          console.error(e2);
          return this;
        }
      };
      this.build = () => {
        let doc4;
        do
          doc4 = __privateGet2(this, _closeNodeAndPush).call(this);
        while (this.size());
        return doc4;
      };
      this.next = (nodes = []) => {
        [nodes].flat().forEach((node2) => __privateGet2(this, _runNode).call(this, node2));
        return this;
      };
      this.toDoc = () => this.build();
      this.run = (remark, markdown) => {
        const tree = remark.runSync(
          remark.parse(markdown),
          markdown
        );
        this.next(tree);
        return this;
      };
      this.schema = schema3;
    }
  };
  _marks = /* @__PURE__ */ new WeakMap();
  _hasText = /* @__PURE__ */ new WeakMap();
  _maybeMerge = /* @__PURE__ */ new WeakMap();
  _matchTarget = /* @__PURE__ */ new WeakMap();
  _runNode = /* @__PURE__ */ new WeakMap();
  _closeNodeAndPush = /* @__PURE__ */ new WeakMap();
  _addNodeAndPush = /* @__PURE__ */ new WeakMap();
  _ParserState.create = (schema3, remark) => {
    const state = new _ParserState(schema3);
    return (text4) => {
      state.run(remark, text4);
      return state.toDoc();
    };
  };
  var ParserState = _ParserState;
  var _SerializerStackElement = class _SerializerStackElement2 extends StackElement {
    constructor(type, children, value, props = {}) {
      super();
      this.type = type;
      this.children = children;
      this.value = value;
      this.props = props;
      this.push = (node2, ...rest) => {
        if (!this.children) this.children = [];
        this.children.push(node2, ...rest);
      };
      this.pop = () => this.children?.pop();
    }
  };
  _SerializerStackElement.create = (type, children, value, props = {}) => new _SerializerStackElement(type, children, value, props);
  var SerializerStackElement = _SerializerStackElement;
  var isFragment = (x) => Object.prototype.hasOwnProperty.call(x, "size");
  var _SerializerState = class _SerializerState2 extends Stack {
    /// @internal
    constructor(schema3) {
      super();
      __privateAdd2(this, _marks2);
      __privateAdd2(this, _matchTarget2);
      __privateAdd2(this, _runProseNode);
      __privateAdd2(this, _runProseMark);
      __privateAdd2(this, _runNode2);
      __privateAdd2(this, _searchType);
      __privateAdd2(this, _maybeMergeChildren);
      __privateAdd2(this, _createMarkdownNode);
      __privateAdd2(this, _moveSpaces);
      __privateAdd2(this, _closeNodeAndPush2);
      __privateAdd2(this, _addNodeAndPush2);
      __privateAdd2(this, _openMark);
      __privateAdd2(this, _closeMark);
      __privateSet2(this, _marks2, Mark.none);
      __privateSet2(this, _matchTarget2, (node2) => {
        const result = Object.values({
          ...this.schema.nodes,
          ...this.schema.marks
        }).find((x) => {
          const spec = x.spec;
          return spec.toMarkdown.match(node2);
        });
        if (!result) throw serializerMatchError(node2.type);
        return result;
      });
      __privateSet2(this, _runProseNode, (node2) => {
        const type = __privateGet2(this, _matchTarget2).call(this, node2);
        const spec = type.spec;
        return spec.toMarkdown.runner(this, node2);
      });
      __privateSet2(this, _runProseMark, (mark, node2) => {
        const type = __privateGet2(this, _matchTarget2).call(this, mark);
        const spec = type.spec;
        return spec.toMarkdown.runner(this, mark, node2);
      });
      __privateSet2(this, _runNode2, (node2) => {
        const { marks } = node2;
        const getPriority = (x) => x.type.spec.priority ?? 50;
        const tmp = [...marks].sort((a2, b) => getPriority(a2) - getPriority(b));
        const unPreventNext = tmp.every((mark) => !__privateGet2(this, _runProseMark).call(this, mark, node2));
        if (unPreventNext) __privateGet2(this, _runProseNode).call(this, node2);
        marks.forEach((mark) => __privateGet2(this, _closeMark).call(this, mark));
      });
      __privateSet2(this, _searchType, (child, type) => {
        if (child.type === type) return child;
        if (child.children?.length !== 1) return child;
        const searchNode = (node22) => {
          if (node22.type === type) return node22;
          if (node22.children?.length !== 1) return null;
          const [firstChild] = node22.children;
          if (!firstChild) return null;
          return searchNode(firstChild);
        };
        const target = searchNode(child);
        if (!target) return child;
        const tmp = target.children ? [...target.children] : void 0;
        const node2 = { ...child, children: tmp };
        node2.children = tmp;
        target.children = [node2];
        return target;
      });
      __privateSet2(this, _maybeMergeChildren, (node2) => {
        const { children } = node2;
        if (!children) return node2;
        node2.children = children.reduce((nextChildren, child, index2) => {
          if (index2 === 0) return [child];
          const last = nextChildren.at(-1);
          if (last && last.isMark && child.isMark) {
            child = __privateGet2(this, _searchType).call(this, child, last.type);
            const { children: currChildren, ...currRest } = child;
            const { children: prevChildren, ...prevRest } = last;
            if (child.type === last.type && currChildren && prevChildren && JSON.stringify(currRest) === JSON.stringify(prevRest)) {
              const next = {
                ...prevRest,
                children: [...prevChildren, ...currChildren]
              };
              return nextChildren.slice(0, -1).concat(__privateGet2(this, _maybeMergeChildren).call(this, next));
            }
          }
          return nextChildren.concat(child);
        }, []);
        return node2;
      });
      __privateSet2(this, _createMarkdownNode, (element2) => {
        const node2 = {
          ...element2.props,
          type: element2.type
        };
        if (element2.children) node2.children = element2.children;
        if (element2.value) node2.value = element2.value;
        return node2;
      });
      this.openNode = (type, value, props) => {
        this.open(SerializerStackElement.create(type, void 0, value, props));
        return this;
      };
      __privateSet2(this, _moveSpaces, (element2, onPush) => {
        let startSpaces = "";
        let endSpaces = "";
        const children = element2.children;
        let first = -1;
        let last = -1;
        const findIndex = (node2) => {
          if (!node2) return;
          node2.forEach((child, index2) => {
            if (child.type === "text" && child.value) {
              if (first < 0) first = index2;
              last = index2;
            }
          });
        };
        if (children) {
          findIndex(children);
          const lastChild = children?.[last];
          const firstChild = children?.[first];
          if (lastChild && lastChild.value.endsWith(" ")) {
            const text4 = lastChild.value;
            const trimmed = text4.trimEnd();
            endSpaces = text4.slice(trimmed.length);
            lastChild.value = trimmed;
          }
          if (firstChild && firstChild.value.startsWith(" ")) {
            const text4 = firstChild.value;
            const trimmed = text4.trimStart();
            startSpaces = text4.slice(0, text4.length - trimmed.length);
            firstChild.value = trimmed;
          }
        }
        if (startSpaces.length) __privateGet2(this, _addNodeAndPush2).call(this, "text", void 0, startSpaces);
        const result = onPush();
        if (endSpaces.length) __privateGet2(this, _addNodeAndPush2).call(this, "text", void 0, endSpaces);
        return result;
      });
      __privateSet2(this, _closeNodeAndPush2, (trim = false) => {
        const element2 = this.close();
        const onPush = () => __privateGet2(this, _addNodeAndPush2).call(this, element2.type, element2.children, element2.value, element2.props);
        if (trim) return __privateGet2(this, _moveSpaces).call(this, element2, onPush);
        return onPush();
      });
      this.closeNode = () => {
        __privateGet2(this, _closeNodeAndPush2).call(this);
        return this;
      };
      __privateSet2(this, _addNodeAndPush2, (type, children, value, props) => {
        const element2 = SerializerStackElement.create(type, children, value, props);
        const node2 = __privateGet2(this, _maybeMergeChildren).call(this, __privateGet2(this, _createMarkdownNode).call(this, element2));
        this.push(node2);
        return node2;
      });
      this.addNode = (type, children, value, props) => {
        __privateGet2(this, _addNodeAndPush2).call(this, type, children, value, props);
        return this;
      };
      __privateSet2(this, _openMark, (mark, type, value, props) => {
        const isIn = mark.isInSet(__privateGet2(this, _marks2));
        if (isIn) return this;
        __privateSet2(this, _marks2, mark.addToSet(__privateGet2(this, _marks2)));
        return this.openNode(type, value, { ...props, isMark: true });
      });
      __privateSet2(this, _closeMark, (mark) => {
        const isIn = mark.isInSet(__privateGet2(this, _marks2));
        if (!isIn) return;
        __privateSet2(this, _marks2, mark.type.removeFromSet(__privateGet2(this, _marks2)));
        __privateGet2(this, _closeNodeAndPush2).call(this, true);
      });
      this.withMark = (mark, type, value, props) => {
        __privateGet2(this, _openMark).call(this, mark, type, value, props);
        return this;
      };
      this.closeMark = (mark) => {
        __privateGet2(this, _closeMark).call(this, mark);
        return this;
      };
      this.build = () => {
        let doc4 = null;
        do
          doc4 = __privateGet2(this, _closeNodeAndPush2).call(this);
        while (this.size());
        return doc4;
      };
      this.next = (nodes) => {
        if (isFragment(nodes)) {
          nodes.forEach((node2) => {
            __privateGet2(this, _runNode2).call(this, node2);
          });
          return this;
        }
        __privateGet2(this, _runNode2).call(this, nodes);
        return this;
      };
      this.toString = (remark) => remark.stringify(this.build());
      this.run = (tree) => {
        this.next(tree);
        return this;
      };
      this.schema = schema3;
    }
  };
  _marks2 = /* @__PURE__ */ new WeakMap();
  _matchTarget2 = /* @__PURE__ */ new WeakMap();
  _runProseNode = /* @__PURE__ */ new WeakMap();
  _runProseMark = /* @__PURE__ */ new WeakMap();
  _runNode2 = /* @__PURE__ */ new WeakMap();
  _searchType = /* @__PURE__ */ new WeakMap();
  _maybeMergeChildren = /* @__PURE__ */ new WeakMap();
  _createMarkdownNode = /* @__PURE__ */ new WeakMap();
  _moveSpaces = /* @__PURE__ */ new WeakMap();
  _closeNodeAndPush2 = /* @__PURE__ */ new WeakMap();
  _addNodeAndPush2 = /* @__PURE__ */ new WeakMap();
  _openMark = /* @__PURE__ */ new WeakMap();
  _closeMark = /* @__PURE__ */ new WeakMap();
  _SerializerState.create = (schema3, remark) => {
    const state = new _SerializerState(schema3);
    return (content3) => {
      state.run(content3);
      return state.toString(remark);
    };
  };
  var SerializerState = _SerializerState;

  // node_modules/prosemirror-transform/dist/index.js
  var lower16 = 65535;
  var factor16 = Math.pow(2, 16);
  function makeRecover(index2, offset) {
    return index2 + offset * factor16;
  }
  function recoverIndex(value) {
    return value & lower16;
  }
  function recoverOffset(value) {
    return (value - (value & lower16)) / factor16;
  }
  var DEL_BEFORE = 1;
  var DEL_AFTER = 2;
  var DEL_ACROSS = 4;
  var DEL_SIDE = 8;
  var MapResult = class {
    /**
    @internal
    */
    constructor(pos, delInfo, recover) {
      this.pos = pos;
      this.delInfo = delInfo;
      this.recover = recover;
    }
    /**
    Tells you whether the position was deleted, that is, whether the
    step removed the token on the side queried (via the `assoc`)
    argument from the document.
    */
    get deleted() {
      return (this.delInfo & DEL_SIDE) > 0;
    }
    /**
    Tells you whether the token before the mapped position was deleted.
    */
    get deletedBefore() {
      return (this.delInfo & (DEL_BEFORE | DEL_ACROSS)) > 0;
    }
    /**
    True when the token after the mapped position was deleted.
    */
    get deletedAfter() {
      return (this.delInfo & (DEL_AFTER | DEL_ACROSS)) > 0;
    }
    /**
    Tells whether any of the steps mapped through deletes across the
    position (including both the token before and after the
    position).
    */
    get deletedAcross() {
      return (this.delInfo & DEL_ACROSS) > 0;
    }
  };
  var StepMap = class _StepMap {
    /**
    Create a position map. The modifications to the document are
    represented as an array of numbers, in which each group of three
    represents a modified chunk as `[start, oldSize, newSize]`.
    */
    constructor(ranges, inverted = false) {
      this.ranges = ranges;
      this.inverted = inverted;
      if (!ranges.length && _StepMap.empty)
        return _StepMap.empty;
    }
    /**
    @internal
    */
    recover(value) {
      let diff = 0, index2 = recoverIndex(value);
      if (!this.inverted)
        for (let i2 = 0; i2 < index2; i2++)
          diff += this.ranges[i2 * 3 + 2] - this.ranges[i2 * 3 + 1];
      return this.ranges[index2 * 3] + diff + recoverOffset(value);
    }
    mapResult(pos, assoc = 1) {
      return this._map(pos, assoc, false);
    }
    map(pos, assoc = 1) {
      return this._map(pos, assoc, true);
    }
    /**
    @internal
    */
    _map(pos, assoc, simple) {
      let diff = 0, oldIndex = this.inverted ? 2 : 1, newIndex = this.inverted ? 1 : 2;
      for (let i2 = 0; i2 < this.ranges.length; i2 += 3) {
        let start = this.ranges[i2] - (this.inverted ? diff : 0);
        if (start > pos)
          break;
        let oldSize = this.ranges[i2 + oldIndex], newSize = this.ranges[i2 + newIndex], end = start + oldSize;
        if (pos <= end) {
          let side = !oldSize ? assoc : pos == start ? -1 : pos == end ? 1 : assoc;
          let result = start + diff + (side < 0 ? 0 : newSize);
          if (simple)
            return result;
          let recover = pos == (assoc < 0 ? start : end) ? null : makeRecover(i2 / 3, pos - start);
          let del2 = pos == start ? DEL_AFTER : pos == end ? DEL_BEFORE : DEL_ACROSS;
          if (assoc < 0 ? pos != start : pos != end)
            del2 |= DEL_SIDE;
          return new MapResult(result, del2, recover);
        }
        diff += newSize - oldSize;
      }
      return simple ? pos + diff : new MapResult(pos + diff, 0, null);
    }
    /**
    @internal
    */
    touches(pos, recover) {
      let diff = 0, index2 = recoverIndex(recover);
      let oldIndex = this.inverted ? 2 : 1, newIndex = this.inverted ? 1 : 2;
      for (let i2 = 0; i2 < this.ranges.length; i2 += 3) {
        let start = this.ranges[i2] - (this.inverted ? diff : 0);
        if (start > pos)
          break;
        let oldSize = this.ranges[i2 + oldIndex], end = start + oldSize;
        if (pos <= end && i2 == index2 * 3)
          return true;
        diff += this.ranges[i2 + newIndex] - oldSize;
      }
      return false;
    }
    /**
    Calls the given function on each of the changed ranges included in
    this map.
    */
    forEach(f2) {
      let oldIndex = this.inverted ? 2 : 1, newIndex = this.inverted ? 1 : 2;
      for (let i2 = 0, diff = 0; i2 < this.ranges.length; i2 += 3) {
        let start = this.ranges[i2], oldStart = start - (this.inverted ? diff : 0), newStart = start + (this.inverted ? 0 : diff);
        let oldSize = this.ranges[i2 + oldIndex], newSize = this.ranges[i2 + newIndex];
        f2(oldStart, oldStart + oldSize, newStart, newStart + newSize);
        diff += newSize - oldSize;
      }
    }
    /**
    Create an inverted version of this map. The result can be used to
    map positions in the post-step document to the pre-step document.
    */
    invert() {
      return new _StepMap(this.ranges, !this.inverted);
    }
    /**
    @internal
    */
    toString() {
      return (this.inverted ? "-" : "") + JSON.stringify(this.ranges);
    }
    /**
    Create a map that moves all positions by offset `n` (which may be
    negative). This can be useful when applying steps meant for a
    sub-document to a larger document, or vice-versa.
    */
    static offset(n2) {
      return n2 == 0 ? _StepMap.empty : new _StepMap(n2 < 0 ? [0, -n2, 0] : [0, 0, n2]);
    }
  };
  StepMap.empty = new StepMap([]);
  var Mapping = class _Mapping {
    /**
    Create a new mapping with the given position maps.
    */
    constructor(maps, mirror, from = 0, to = maps ? maps.length : 0) {
      this.mirror = mirror;
      this.from = from;
      this.to = to;
      this._maps = maps || [];
      this.ownData = !(maps || mirror);
    }
    /**
    The step maps in this mapping.
    */
    get maps() {
      return this._maps;
    }
    /**
    Create a mapping that maps only through a part of this one.
    */
    slice(from = 0, to = this.maps.length) {
      return new _Mapping(this._maps, this.mirror, from, to);
    }
    /**
    Add a step map to the end of this mapping. If `mirrors` is
    given, it should be the index of the step map that is the mirror
    image of this one.
    */
    appendMap(map4, mirrors) {
      if (!this.ownData) {
        this._maps = this._maps.slice();
        this.mirror = this.mirror && this.mirror.slice();
        this.ownData = true;
      }
      this.to = this._maps.push(map4);
      if (mirrors != null)
        this.setMirror(this._maps.length - 1, mirrors);
    }
    /**
    Add all the step maps in a given mapping to this one (preserving
    mirroring information).
    */
    appendMapping(mapping) {
      for (let i2 = 0, startSize = this._maps.length; i2 < mapping._maps.length; i2++) {
        let mirr = mapping.getMirror(i2);
        this.appendMap(mapping._maps[i2], mirr != null && mirr < i2 ? startSize + mirr : void 0);
      }
    }
    /**
    Finds the offset of the step map that mirrors the map at the
    given offset, in this mapping (as per the second argument to
    `appendMap`).
    */
    getMirror(n2) {
      if (this.mirror) {
        for (let i2 = 0; i2 < this.mirror.length; i2++)
          if (this.mirror[i2] == n2)
            return this.mirror[i2 + (i2 % 2 ? -1 : 1)];
      }
    }
    /**
    @internal
    */
    setMirror(n2, m) {
      if (!this.mirror)
        this.mirror = [];
      this.mirror.push(n2, m);
    }
    /**
    Append the inverse of the given mapping to this one.
    */
    appendMappingInverted(mapping) {
      for (let i2 = mapping.maps.length - 1, totalSize = this._maps.length + mapping._maps.length; i2 >= 0; i2--) {
        let mirr = mapping.getMirror(i2);
        this.appendMap(mapping._maps[i2].invert(), mirr != null && mirr > i2 ? totalSize - mirr - 1 : void 0);
      }
    }
    /**
    Create an inverted version of this mapping.
    */
    invert() {
      let inverse = new _Mapping();
      inverse.appendMappingInverted(this);
      return inverse;
    }
    /**
    Map a position through this mapping.
    */
    map(pos, assoc = 1) {
      if (this.mirror)
        return this._map(pos, assoc, true);
      for (let i2 = this.from; i2 < this.to; i2++)
        pos = this._maps[i2].map(pos, assoc);
      return pos;
    }
    /**
    Map a position through this mapping, returning a mapping
    result.
    */
    mapResult(pos, assoc = 1) {
      return this._map(pos, assoc, false);
    }
    /**
    @internal
    */
    _map(pos, assoc, simple) {
      let delInfo = 0;
      for (let i2 = this.from; i2 < this.to; i2++) {
        let map4 = this._maps[i2], result = map4.mapResult(pos, assoc);
        if (result.recover != null) {
          let corr = this.getMirror(i2);
          if (corr != null && corr > i2 && corr < this.to) {
            i2 = corr;
            pos = this._maps[corr].recover(result.recover);
            continue;
          }
        }
        delInfo |= result.delInfo;
        pos = result.pos;
      }
      return simple ? pos : new MapResult(pos, delInfo, null);
    }
  };
  var stepsByID = /* @__PURE__ */ Object.create(null);
  var Step = class {
    /**
    Get the step map that represents the changes made by this step,
    and which can be used to transform between positions in the old
    and the new document.
    */
    getMap() {
      return StepMap.empty;
    }
    /**
    Try to merge this step with another one, to be applied directly
    after it. Returns the merged step when possible, null if the
    steps can't be merged.
    */
    merge(other) {
      return null;
    }
    /**
    Deserialize a step from its JSON representation. Will call
    through to the step class' own implementation of this method.
    */
    static fromJSON(schema3, json) {
      if (!json || !json.stepType)
        throw new RangeError("Invalid input for Step.fromJSON");
      let type = stepsByID[json.stepType];
      if (!type)
        throw new RangeError(`No step type ${json.stepType} defined`);
      return type.fromJSON(schema3, json);
    }
    /**
    To be able to serialize steps to JSON, each step needs a string
    ID to attach to its JSON representation. Use this method to
    register an ID for your step classes. Try to pick something
    that's unlikely to clash with steps from other modules.
    */
    static jsonID(id, stepClass) {
      if (id in stepsByID)
        throw new RangeError("Duplicate use of step JSON ID " + id);
      stepsByID[id] = stepClass;
      stepClass.prototype.jsonID = id;
      return stepClass;
    }
  };
  var StepResult = class _StepResult {
    /**
    @internal
    */
    constructor(doc4, failed) {
      this.doc = doc4;
      this.failed = failed;
    }
    /**
    Create a successful step result.
    */
    static ok(doc4) {
      return new _StepResult(doc4, null);
    }
    /**
    Create a failed step result.
    */
    static fail(message) {
      return new _StepResult(null, message);
    }
    /**
    Call [`Node.replace`](https://prosemirror.net/docs/ref/#model.Node.replace) with the given
    arguments. Create a successful result if it succeeds, and a
    failed one if it throws a `ReplaceError`.
    */
    static fromReplace(doc4, from, to, slice) {
      try {
        return _StepResult.ok(doc4.replace(from, to, slice));
      } catch (e2) {
        if (e2 instanceof ReplaceError)
          return _StepResult.fail(e2.message);
        throw e2;
      }
    }
  };
  function mapFragment(fragment, f2, parent) {
    let mapped = [];
    for (let i2 = 0; i2 < fragment.childCount; i2++) {
      let child = fragment.child(i2);
      if (child.content.size)
        child = child.copy(mapFragment(child.content, f2, child));
      if (child.isInline)
        child = f2(child, parent, i2);
      mapped.push(child);
    }
    return Fragment.fromArray(mapped);
  }
  var AddMarkStep = class _AddMarkStep extends Step {
    /**
    Create a mark step.
    */
    constructor(from, to, mark) {
      super();
      this.from = from;
      this.to = to;
      this.mark = mark;
    }
    apply(doc4) {
      let oldSlice = doc4.slice(this.from, this.to), $from = doc4.resolve(this.from);
      let parent = $from.node($from.sharedDepth(this.to));
      let slice = new Slice2(mapFragment(oldSlice.content, (node2, parent2) => {
        if (!node2.isAtom || !parent2.type.allowsMarkType(this.mark.type))
          return node2;
        return node2.mark(this.mark.addToSet(node2.marks));
      }, parent), oldSlice.openStart, oldSlice.openEnd);
      return StepResult.fromReplace(doc4, this.from, this.to, slice);
    }
    invert() {
      return new RemoveMarkStep(this.from, this.to, this.mark);
    }
    map(mapping) {
      let from = mapping.mapResult(this.from, 1), to = mapping.mapResult(this.to, -1);
      if (from.deleted && to.deleted || from.pos >= to.pos)
        return null;
      return new _AddMarkStep(from.pos, to.pos, this.mark);
    }
    merge(other) {
      if (other instanceof _AddMarkStep && other.mark.eq(this.mark) && this.from <= other.to && this.to >= other.from)
        return new _AddMarkStep(Math.min(this.from, other.from), Math.max(this.to, other.to), this.mark);
      return null;
    }
    toJSON() {
      return {
        stepType: "addMark",
        mark: this.mark.toJSON(),
        from: this.from,
        to: this.to
      };
    }
    /**
    @internal
    */
    static fromJSON(schema3, json) {
      if (typeof json.from != "number" || typeof json.to != "number")
        throw new RangeError("Invalid input for AddMarkStep.fromJSON");
      return new _AddMarkStep(json.from, json.to, schema3.markFromJSON(json.mark));
    }
  };
  Step.jsonID("addMark", AddMarkStep);
  var RemoveMarkStep = class _RemoveMarkStep extends Step {
    /**
    Create a mark-removing step.
    */
    constructor(from, to, mark) {
      super();
      this.from = from;
      this.to = to;
      this.mark = mark;
    }
    apply(doc4) {
      let oldSlice = doc4.slice(this.from, this.to);
      let slice = new Slice2(mapFragment(oldSlice.content, (node2) => {
        return node2.mark(this.mark.removeFromSet(node2.marks));
      }, doc4), oldSlice.openStart, oldSlice.openEnd);
      return StepResult.fromReplace(doc4, this.from, this.to, slice);
    }
    invert() {
      return new AddMarkStep(this.from, this.to, this.mark);
    }
    map(mapping) {
      let from = mapping.mapResult(this.from, 1), to = mapping.mapResult(this.to, -1);
      if (from.deleted && to.deleted || from.pos >= to.pos)
        return null;
      return new _RemoveMarkStep(from.pos, to.pos, this.mark);
    }
    merge(other) {
      if (other instanceof _RemoveMarkStep && other.mark.eq(this.mark) && this.from <= other.to && this.to >= other.from)
        return new _RemoveMarkStep(Math.min(this.from, other.from), Math.max(this.to, other.to), this.mark);
      return null;
    }
    toJSON() {
      return {
        stepType: "removeMark",
        mark: this.mark.toJSON(),
        from: this.from,
        to: this.to
      };
    }
    /**
    @internal
    */
    static fromJSON(schema3, json) {
      if (typeof json.from != "number" || typeof json.to != "number")
        throw new RangeError("Invalid input for RemoveMarkStep.fromJSON");
      return new _RemoveMarkStep(json.from, json.to, schema3.markFromJSON(json.mark));
    }
  };
  Step.jsonID("removeMark", RemoveMarkStep);
  var AddNodeMarkStep = class _AddNodeMarkStep extends Step {
    /**
    Create a node mark step.
    */
    constructor(pos, mark) {
      super();
      this.pos = pos;
      this.mark = mark;
    }
    apply(doc4) {
      let node2 = doc4.nodeAt(this.pos);
      if (!node2)
        return StepResult.fail("No node at mark step's position");
      let updated = node2.type.create(node2.attrs, null, this.mark.addToSet(node2.marks));
      return StepResult.fromReplace(doc4, this.pos, this.pos + 1, new Slice2(Fragment.from(updated), 0, node2.isLeaf ? 0 : 1));
    }
    invert(doc4) {
      let node2 = doc4.nodeAt(this.pos);
      if (node2) {
        let newSet = this.mark.addToSet(node2.marks);
        if (newSet.length == node2.marks.length) {
          for (let i2 = 0; i2 < node2.marks.length; i2++)
            if (!node2.marks[i2].isInSet(newSet))
              return new _AddNodeMarkStep(this.pos, node2.marks[i2]);
          return new _AddNodeMarkStep(this.pos, this.mark);
        }
      }
      return new RemoveNodeMarkStep(this.pos, this.mark);
    }
    map(mapping) {
      let pos = mapping.mapResult(this.pos, 1);
      return pos.deletedAfter ? null : new _AddNodeMarkStep(pos.pos, this.mark);
    }
    toJSON() {
      return { stepType: "addNodeMark", pos: this.pos, mark: this.mark.toJSON() };
    }
    /**
    @internal
    */
    static fromJSON(schema3, json) {
      if (typeof json.pos != "number")
        throw new RangeError("Invalid input for AddNodeMarkStep.fromJSON");
      return new _AddNodeMarkStep(json.pos, schema3.markFromJSON(json.mark));
    }
  };
  Step.jsonID("addNodeMark", AddNodeMarkStep);
  var RemoveNodeMarkStep = class _RemoveNodeMarkStep extends Step {
    /**
    Create a mark-removing step.
    */
    constructor(pos, mark) {
      super();
      this.pos = pos;
      this.mark = mark;
    }
    apply(doc4) {
      let node2 = doc4.nodeAt(this.pos);
      if (!node2)
        return StepResult.fail("No node at mark step's position");
      let updated = node2.type.create(node2.attrs, null, this.mark.removeFromSet(node2.marks));
      return StepResult.fromReplace(doc4, this.pos, this.pos + 1, new Slice2(Fragment.from(updated), 0, node2.isLeaf ? 0 : 1));
    }
    invert(doc4) {
      let node2 = doc4.nodeAt(this.pos);
      if (!node2 || !this.mark.isInSet(node2.marks))
        return this;
      return new AddNodeMarkStep(this.pos, this.mark);
    }
    map(mapping) {
      let pos = mapping.mapResult(this.pos, 1);
      return pos.deletedAfter ? null : new _RemoveNodeMarkStep(pos.pos, this.mark);
    }
    toJSON() {
      return { stepType: "removeNodeMark", pos: this.pos, mark: this.mark.toJSON() };
    }
    /**
    @internal
    */
    static fromJSON(schema3, json) {
      if (typeof json.pos != "number")
        throw new RangeError("Invalid input for RemoveNodeMarkStep.fromJSON");
      return new _RemoveNodeMarkStep(json.pos, schema3.markFromJSON(json.mark));
    }
  };
  Step.jsonID("removeNodeMark", RemoveNodeMarkStep);
  var ReplaceStep = class _ReplaceStep extends Step {
    /**
    The given `slice` should fit the 'gap' between `from` and
    `to`—the depths must line up, and the surrounding nodes must be
    able to be joined with the open sides of the slice. When
    `structure` is true, the step will fail if the content between
    from and to is not just a sequence of closing and then opening
    tokens (this is to guard against rebased replace steps
    overwriting something they weren't supposed to).
    */
    constructor(from, to, slice, structure = false) {
      super();
      this.from = from;
      this.to = to;
      this.slice = slice;
      this.structure = structure;
    }
    apply(doc4) {
      if (this.structure && contentBetween(doc4, this.from, this.to))
        return StepResult.fail("Structure replace would overwrite content");
      return StepResult.fromReplace(doc4, this.from, this.to, this.slice);
    }
    getMap() {
      return new StepMap([this.from, this.to - this.from, this.slice.size]);
    }
    invert(doc4) {
      return new _ReplaceStep(this.from, this.from + this.slice.size, doc4.slice(this.from, this.to));
    }
    map(mapping) {
      let from = mapping.mapResult(this.from, 1), to = mapping.mapResult(this.to, -1);
      if (from.deletedAcross && to.deletedAcross)
        return null;
      return new _ReplaceStep(from.pos, Math.max(from.pos, to.pos), this.slice, this.structure);
    }
    merge(other) {
      if (!(other instanceof _ReplaceStep) || other.structure || this.structure)
        return null;
      if (this.from + this.slice.size == other.from && !this.slice.openEnd && !other.slice.openStart) {
        let slice = this.slice.size + other.slice.size == 0 ? Slice2.empty : new Slice2(this.slice.content.append(other.slice.content), this.slice.openStart, other.slice.openEnd);
        return new _ReplaceStep(this.from, this.to + (other.to - other.from), slice, this.structure);
      } else if (other.to == this.from && !this.slice.openStart && !other.slice.openEnd) {
        let slice = this.slice.size + other.slice.size == 0 ? Slice2.empty : new Slice2(other.slice.content.append(this.slice.content), other.slice.openStart, this.slice.openEnd);
        return new _ReplaceStep(other.from, this.to, slice, this.structure);
      } else {
        return null;
      }
    }
    toJSON() {
      let json = { stepType: "replace", from: this.from, to: this.to };
      if (this.slice.size)
        json.slice = this.slice.toJSON();
      if (this.structure)
        json.structure = true;
      return json;
    }
    /**
    @internal
    */
    static fromJSON(schema3, json) {
      if (typeof json.from != "number" || typeof json.to != "number")
        throw new RangeError("Invalid input for ReplaceStep.fromJSON");
      return new _ReplaceStep(json.from, json.to, Slice2.fromJSON(schema3, json.slice), !!json.structure);
    }
  };
  Step.jsonID("replace", ReplaceStep);
  var ReplaceAroundStep = class _ReplaceAroundStep extends Step {
    /**
    Create a replace-around step with the given range and gap.
    `insert` should be the point in the slice into which the content
    of the gap should be moved. `structure` has the same meaning as
    it has in the [`ReplaceStep`](https://prosemirror.net/docs/ref/#transform.ReplaceStep) class.
    */
    constructor(from, to, gapFrom, gapTo, slice, insert, structure = false) {
      super();
      this.from = from;
      this.to = to;
      this.gapFrom = gapFrom;
      this.gapTo = gapTo;
      this.slice = slice;
      this.insert = insert;
      this.structure = structure;
    }
    apply(doc4) {
      if (this.structure && (contentBetween(doc4, this.from, this.gapFrom) || contentBetween(doc4, this.gapTo, this.to)))
        return StepResult.fail("Structure gap-replace would overwrite content");
      let gap = doc4.slice(this.gapFrom, this.gapTo);
      if (gap.openStart || gap.openEnd)
        return StepResult.fail("Gap is not a flat range");
      let inserted = this.slice.insertAt(this.insert, gap.content);
      if (!inserted)
        return StepResult.fail("Content does not fit in gap");
      return StepResult.fromReplace(doc4, this.from, this.to, inserted);
    }
    getMap() {
      return new StepMap([
        this.from,
        this.gapFrom - this.from,
        this.insert,
        this.gapTo,
        this.to - this.gapTo,
        this.slice.size - this.insert
      ]);
    }
    invert(doc4) {
      let gap = this.gapTo - this.gapFrom;
      return new _ReplaceAroundStep(this.from, this.from + this.slice.size + gap, this.from + this.insert, this.from + this.insert + gap, doc4.slice(this.from, this.to).removeBetween(this.gapFrom - this.from, this.gapTo - this.from), this.gapFrom - this.from, this.structure);
    }
    map(mapping) {
      let from = mapping.mapResult(this.from, 1), to = mapping.mapResult(this.to, -1);
      let gapFrom = this.from == this.gapFrom ? from.pos : mapping.map(this.gapFrom, -1);
      let gapTo = this.to == this.gapTo ? to.pos : mapping.map(this.gapTo, 1);
      if (from.deletedAcross && to.deletedAcross || gapFrom < from.pos || gapTo > to.pos)
        return null;
      return new _ReplaceAroundStep(from.pos, to.pos, gapFrom, gapTo, this.slice, this.insert, this.structure);
    }
    toJSON() {
      let json = {
        stepType: "replaceAround",
        from: this.from,
        to: this.to,
        gapFrom: this.gapFrom,
        gapTo: this.gapTo,
        insert: this.insert
      };
      if (this.slice.size)
        json.slice = this.slice.toJSON();
      if (this.structure)
        json.structure = true;
      return json;
    }
    /**
    @internal
    */
    static fromJSON(schema3, json) {
      if (typeof json.from != "number" || typeof json.to != "number" || typeof json.gapFrom != "number" || typeof json.gapTo != "number" || typeof json.insert != "number")
        throw new RangeError("Invalid input for ReplaceAroundStep.fromJSON");
      return new _ReplaceAroundStep(json.from, json.to, json.gapFrom, json.gapTo, Slice2.fromJSON(schema3, json.slice), json.insert, !!json.structure);
    }
  };
  Step.jsonID("replaceAround", ReplaceAroundStep);
  function contentBetween(doc4, from, to) {
    let $from = doc4.resolve(from), dist = to - from, depth = $from.depth;
    while (dist > 0 && depth > 0 && $from.indexAfter(depth) == $from.node(depth).childCount) {
      depth--;
      dist--;
    }
    if (dist > 0) {
      let next = $from.node(depth).maybeChild($from.indexAfter(depth));
      while (dist > 0) {
        if (!next || next.isLeaf)
          return true;
        next = next.firstChild;
        dist--;
      }
    }
    return false;
  }
  function addMark(tr, from, to, mark) {
    let removed = [], added = [];
    let removing, adding;
    tr.doc.nodesBetween(from, to, (node2, pos, parent) => {
      if (!node2.isInline)
        return;
      let marks = node2.marks;
      if (!mark.isInSet(marks) && parent.type.allowsMarkType(mark.type)) {
        let start = Math.max(pos, from), end = Math.min(pos + node2.nodeSize, to);
        let newSet = mark.addToSet(marks);
        for (let i2 = 0; i2 < marks.length; i2++) {
          if (!marks[i2].isInSet(newSet)) {
            if (removing && removing.to == start && removing.mark.eq(marks[i2]))
              removing.to = end;
            else
              removed.push(removing = new RemoveMarkStep(start, end, marks[i2]));
          }
        }
        if (adding && adding.to == start)
          adding.to = end;
        else
          added.push(adding = new AddMarkStep(start, end, mark));
      }
    });
    removed.forEach((s2) => tr.step(s2));
    added.forEach((s2) => tr.step(s2));
  }
  function removeMark(tr, from, to, mark) {
    let matched = [], step = 0;
    tr.doc.nodesBetween(from, to, (node2, pos) => {
      if (!node2.isInline)
        return;
      step++;
      let toRemove = null;
      if (mark instanceof MarkType) {
        let set = node2.marks, found2;
        while (found2 = mark.isInSet(set)) {
          (toRemove || (toRemove = [])).push(found2);
          set = found2.removeFromSet(set);
        }
      } else if (mark) {
        if (mark.isInSet(node2.marks))
          toRemove = [mark];
      } else {
        toRemove = node2.marks;
      }
      if (toRemove && toRemove.length) {
        let end = Math.min(pos + node2.nodeSize, to);
        for (let i2 = 0; i2 < toRemove.length; i2++) {
          let style = toRemove[i2], found2;
          for (let j = 0; j < matched.length; j++) {
            let m = matched[j];
            if (m.step == step - 1 && style.eq(matched[j].style))
              found2 = m;
          }
          if (found2) {
            found2.to = end;
            found2.step = step;
          } else {
            matched.push({ style, from: Math.max(pos, from), to: end, step });
          }
        }
      }
    });
    matched.forEach((m) => tr.step(new RemoveMarkStep(m.from, m.to, m.style)));
  }
  function clearIncompatible(tr, pos, parentType, match = parentType.contentMatch, clearNewlines = true) {
    let node2 = tr.doc.nodeAt(pos);
    let replSteps = [], cur = pos + 1;
    for (let i2 = 0; i2 < node2.childCount; i2++) {
      let child = node2.child(i2), end = cur + child.nodeSize;
      let allowed = match.matchType(child.type);
      if (!allowed) {
        replSteps.push(new ReplaceStep(cur, end, Slice2.empty));
      } else {
        match = allowed;
        for (let j = 0; j < child.marks.length; j++)
          if (!parentType.allowsMarkType(child.marks[j].type))
            tr.step(new RemoveMarkStep(cur, end, child.marks[j]));
        if (clearNewlines && child.isText && parentType.whitespace != "pre") {
          let m, newline = /\r?\n|\r/g, slice;
          while (m = newline.exec(child.text)) {
            if (!slice)
              slice = new Slice2(Fragment.from(parentType.schema.text(" ", parentType.allowedMarks(child.marks))), 0, 0);
            replSteps.push(new ReplaceStep(cur + m.index, cur + m.index + m[0].length, slice));
          }
        }
      }
      cur = end;
    }
    if (!match.validEnd) {
      let fill = match.fillBefore(Fragment.empty, true);
      tr.replace(cur, cur, new Slice2(fill, 0, 0));
    }
    for (let i2 = replSteps.length - 1; i2 >= 0; i2--)
      tr.step(replSteps[i2]);
  }
  function canCut(node2, start, end) {
    return (start == 0 || node2.canReplace(start, node2.childCount)) && (end == node2.childCount || node2.canReplace(0, end));
  }
  function liftTarget(range) {
    let parent = range.parent;
    let content3 = parent.content.cutByIndex(range.startIndex, range.endIndex);
    for (let depth = range.depth, contentBefore = 0, contentAfter = 0; ; --depth) {
      let node2 = range.$from.node(depth);
      let index2 = range.$from.index(depth) + contentBefore, endIndex = range.$to.indexAfter(depth) - contentAfter;
      if (depth < range.depth && node2.canReplace(index2, endIndex, content3))
        return depth;
      if (depth == 0 || node2.type.spec.isolating || !canCut(node2, index2, endIndex))
        break;
      if (index2)
        contentBefore = 1;
      if (endIndex < node2.childCount)
        contentAfter = 1;
    }
    return null;
  }
  function lift(tr, range, target) {
    let { $from, $to, depth } = range;
    let gapStart = $from.before(depth + 1), gapEnd = $to.after(depth + 1);
    let start = gapStart, end = gapEnd;
    let before = Fragment.empty, openStart = 0;
    for (let d2 = depth, splitting = false; d2 > target; d2--)
      if (splitting || $from.index(d2) > 0) {
        splitting = true;
        before = Fragment.from($from.node(d2).copy(before));
        openStart++;
      } else {
        start--;
      }
    let after = Fragment.empty, openEnd = 0;
    for (let d2 = depth, splitting = false; d2 > target; d2--)
      if (splitting || $to.after(d2 + 1) < $to.end(d2)) {
        splitting = true;
        after = Fragment.from($to.node(d2).copy(after));
        openEnd++;
      } else {
        end++;
      }
    tr.step(new ReplaceAroundStep(start, end, gapStart, gapEnd, new Slice2(before.append(after), openStart, openEnd), before.size - openStart, true));
  }
  function findWrapping(range, nodeType, attrs = null, innerRange = range) {
    let around = findWrappingOutside(range, nodeType);
    let inner = around && findWrappingInside(innerRange, nodeType);
    if (!inner)
      return null;
    return around.map(withAttrs).concat({ type: nodeType, attrs }).concat(inner.map(withAttrs));
  }
  function withAttrs(type) {
    return { type, attrs: null };
  }
  function findWrappingOutside(range, type) {
    let { parent, startIndex, endIndex } = range;
    let around = parent.contentMatchAt(startIndex).findWrapping(type);
    if (!around)
      return null;
    let outer = around.length ? around[0] : type;
    return parent.canReplaceWith(startIndex, endIndex, outer) ? around : null;
  }
  function findWrappingInside(range, type) {
    let { parent, startIndex, endIndex } = range;
    let inner = parent.child(startIndex);
    let inside = type.contentMatch.findWrapping(inner.type);
    if (!inside)
      return null;
    let lastType = inside.length ? inside[inside.length - 1] : type;
    let innerMatch = lastType.contentMatch;
    for (let i2 = startIndex; innerMatch && i2 < endIndex; i2++)
      innerMatch = innerMatch.matchType(parent.child(i2).type);
    if (!innerMatch || !innerMatch.validEnd)
      return null;
    return inside;
  }
  function wrap2(tr, range, wrappers) {
    let content3 = Fragment.empty;
    for (let i2 = wrappers.length - 1; i2 >= 0; i2--) {
      if (content3.size) {
        let match = wrappers[i2].type.contentMatch.matchFragment(content3);
        if (!match || !match.validEnd)
          throw new RangeError("Wrapper type given to Transform.wrap does not form valid content of its parent wrapper");
      }
      content3 = Fragment.from(wrappers[i2].type.create(wrappers[i2].attrs, content3));
    }
    let start = range.start, end = range.end;
    tr.step(new ReplaceAroundStep(start, end, start, end, new Slice2(content3, 0, 0), wrappers.length, true));
  }
  function setBlockType(tr, from, to, type, attrs) {
    if (!type.isTextblock)
      throw new RangeError("Type given to setBlockType should be a textblock");
    let mapFrom = tr.steps.length;
    tr.doc.nodesBetween(from, to, (node2, pos) => {
      let attrsHere = typeof attrs == "function" ? attrs(node2) : attrs;
      if (node2.isTextblock && !node2.hasMarkup(type, attrsHere) && canChangeType(tr.doc, tr.mapping.slice(mapFrom).map(pos), type)) {
        let convertNewlines = null;
        if (type.schema.linebreakReplacement) {
          let pre = type.whitespace == "pre", supportLinebreak = !!type.contentMatch.matchType(type.schema.linebreakReplacement);
          if (pre && !supportLinebreak)
            convertNewlines = false;
          else if (!pre && supportLinebreak)
            convertNewlines = true;
        }
        if (convertNewlines === false)
          replaceLinebreaks(tr, node2, pos, mapFrom);
        clearIncompatible(tr, tr.mapping.slice(mapFrom).map(pos, 1), type, void 0, convertNewlines === null);
        let mapping = tr.mapping.slice(mapFrom);
        let startM = mapping.map(pos, 1), endM = mapping.map(pos + node2.nodeSize, 1);
        tr.step(new ReplaceAroundStep(startM, endM, startM + 1, endM - 1, new Slice2(Fragment.from(type.create(attrsHere, null, node2.marks)), 0, 0), 1, true));
        if (convertNewlines === true)
          replaceNewlines(tr, node2, pos, mapFrom);
        return false;
      }
    });
  }
  function replaceNewlines(tr, node2, pos, mapFrom) {
    node2.forEach((child, offset) => {
      if (child.isText) {
        let m, newline = /\r?\n|\r/g;
        while (m = newline.exec(child.text)) {
          let start = tr.mapping.slice(mapFrom).map(pos + 1 + offset + m.index);
          tr.replaceWith(start, start + 1, node2.type.schema.linebreakReplacement.create());
        }
      }
    });
  }
  function replaceLinebreaks(tr, node2, pos, mapFrom) {
    node2.forEach((child, offset) => {
      if (child.type == child.type.schema.linebreakReplacement) {
        let start = tr.mapping.slice(mapFrom).map(pos + 1 + offset);
        tr.replaceWith(start, start + 1, node2.type.schema.text("\n"));
      }
    });
  }
  function canChangeType(doc4, pos, type) {
    let $pos = doc4.resolve(pos), index2 = $pos.index();
    return $pos.parent.canReplaceWith(index2, index2 + 1, type);
  }
  function setNodeMarkup(tr, pos, type, attrs, marks) {
    let node2 = tr.doc.nodeAt(pos);
    if (!node2)
      throw new RangeError("No node at given position");
    if (!type)
      type = node2.type;
    let newNode = type.create(attrs, null, marks || node2.marks);
    if (node2.isLeaf)
      return tr.replaceWith(pos, pos + node2.nodeSize, newNode);
    if (!type.validContent(node2.content))
      throw new RangeError("Invalid content for node type " + type.name);
    tr.step(new ReplaceAroundStep(pos, pos + node2.nodeSize, pos + 1, pos + node2.nodeSize - 1, new Slice2(Fragment.from(newNode), 0, 0), 1, true));
  }
  function canSplit(doc4, pos, depth = 1, typesAfter) {
    let $pos = doc4.resolve(pos), base2 = $pos.depth - depth;
    let innerType = typesAfter && typesAfter[typesAfter.length - 1] || $pos.parent;
    if (base2 < 0 || $pos.parent.type.spec.isolating || !$pos.parent.canReplace($pos.index(), $pos.parent.childCount) || !innerType.type.validContent($pos.parent.content.cutByIndex($pos.index(), $pos.parent.childCount)))
      return false;
    for (let d2 = $pos.depth - 1, i2 = depth - 2; d2 > base2; d2--, i2--) {
      let node2 = $pos.node(d2), index3 = $pos.index(d2);
      if (node2.type.spec.isolating)
        return false;
      let rest = node2.content.cutByIndex(index3, node2.childCount);
      let overrideChild = typesAfter && typesAfter[i2 + 1];
      if (overrideChild)
        rest = rest.replaceChild(0, overrideChild.type.create(overrideChild.attrs));
      let after = typesAfter && typesAfter[i2] || node2;
      if (!node2.canReplace(index3 + 1, node2.childCount) || !after.type.validContent(rest))
        return false;
    }
    let index2 = $pos.indexAfter(base2);
    let baseType = typesAfter && typesAfter[0];
    return $pos.node(base2).canReplaceWith(index2, index2, baseType ? baseType.type : $pos.node(base2 + 1).type);
  }
  function split(tr, pos, depth = 1, typesAfter) {
    let $pos = tr.doc.resolve(pos), before = Fragment.empty, after = Fragment.empty;
    for (let d2 = $pos.depth, e2 = $pos.depth - depth, i2 = depth - 1; d2 > e2; d2--, i2--) {
      before = Fragment.from($pos.node(d2).copy(before));
      let typeAfter = typesAfter && typesAfter[i2];
      after = Fragment.from(typeAfter ? typeAfter.type.create(typeAfter.attrs, after) : $pos.node(d2).copy(after));
    }
    tr.step(new ReplaceStep(pos, pos, new Slice2(before.append(after), depth, depth), true));
  }
  function canJoin(doc4, pos) {
    let $pos = doc4.resolve(pos), index2 = $pos.index();
    return joinable2($pos.nodeBefore, $pos.nodeAfter) && $pos.parent.canReplace(index2, index2 + 1);
  }
  function canAppendWithSubstitutedLinebreaks(a2, b) {
    if (!b.content.size)
      a2.type.compatibleContent(b.type);
    let match = a2.contentMatchAt(a2.childCount);
    let { linebreakReplacement } = a2.type.schema;
    for (let i2 = 0; i2 < b.childCount; i2++) {
      let child = b.child(i2);
      let type = child.type == linebreakReplacement ? a2.type.schema.nodes.text : child.type;
      match = match.matchType(type);
      if (!match)
        return false;
      if (!a2.type.allowsMarks(child.marks))
        return false;
    }
    return match.validEnd;
  }
  function joinable2(a2, b) {
    return !!(a2 && b && !a2.isLeaf && canAppendWithSubstitutedLinebreaks(a2, b));
  }
  function join3(tr, pos, depth) {
    let convertNewlines = null;
    let { linebreakReplacement } = tr.doc.type.schema;
    let $before = tr.doc.resolve(pos - depth), beforeType = $before.node().type;
    if (linebreakReplacement && beforeType.inlineContent) {
      let pre = beforeType.whitespace == "pre";
      let supportLinebreak = !!beforeType.contentMatch.matchType(linebreakReplacement);
      if (pre && !supportLinebreak)
        convertNewlines = false;
      else if (!pre && supportLinebreak)
        convertNewlines = true;
    }
    let mapFrom = tr.steps.length;
    if (convertNewlines === false) {
      let $after = tr.doc.resolve(pos + depth);
      replaceLinebreaks(tr, $after.node(), $after.before(), mapFrom);
    }
    if (beforeType.inlineContent)
      clearIncompatible(tr, pos + depth - 1, beforeType, $before.node().contentMatchAt($before.index()), convertNewlines == null);
    let mapping = tr.mapping.slice(mapFrom), start = mapping.map(pos - depth);
    tr.step(new ReplaceStep(start, mapping.map(pos + depth, -1), Slice2.empty, true));
    if (convertNewlines === true) {
      let $full = tr.doc.resolve(start);
      replaceNewlines(tr, $full.node(), $full.before(), tr.steps.length);
    }
    return tr;
  }
  function insertPoint(doc4, pos, nodeType) {
    let $pos = doc4.resolve(pos);
    if ($pos.parent.canReplaceWith($pos.index(), $pos.index(), nodeType))
      return pos;
    if ($pos.parentOffset == 0)
      for (let d2 = $pos.depth - 1; d2 >= 0; d2--) {
        let index2 = $pos.index(d2);
        if ($pos.node(d2).canReplaceWith(index2, index2, nodeType))
          return $pos.before(d2 + 1);
        if (index2 > 0)
          return null;
      }
    if ($pos.parentOffset == $pos.parent.content.size)
      for (let d2 = $pos.depth - 1; d2 >= 0; d2--) {
        let index2 = $pos.indexAfter(d2);
        if ($pos.node(d2).canReplaceWith(index2, index2, nodeType))
          return $pos.after(d2 + 1);
        if (index2 < $pos.node(d2).childCount)
          return null;
      }
    return null;
  }
  function dropPoint(doc4, pos, slice) {
    let $pos = doc4.resolve(pos);
    if (!slice.content.size)
      return pos;
    let content3 = slice.content;
    for (let i2 = 0; i2 < slice.openStart; i2++)
      content3 = content3.firstChild.content;
    for (let pass = 1; pass <= (slice.openStart == 0 && slice.size ? 2 : 1); pass++) {
      for (let d2 = $pos.depth; d2 >= 0; d2--) {
        let bias = d2 == $pos.depth ? 0 : $pos.pos <= ($pos.start(d2 + 1) + $pos.end(d2 + 1)) / 2 ? -1 : 1;
        let insertPos = $pos.index(d2) + (bias > 0 ? 1 : 0);
        let parent = $pos.node(d2), fits = false;
        if (pass == 1) {
          fits = parent.canReplace(insertPos, insertPos, content3);
        } else {
          let wrapping = parent.contentMatchAt(insertPos).findWrapping(content3.firstChild.type);
          fits = wrapping && parent.canReplaceWith(insertPos, insertPos, wrapping[0]);
        }
        if (fits)
          return bias == 0 ? $pos.pos : bias < 0 ? $pos.before(d2 + 1) : $pos.after(d2 + 1);
      }
    }
    return null;
  }
  function replaceStep(doc4, from, to = from, slice = Slice2.empty) {
    if (from == to && !slice.size)
      return null;
    let $from = doc4.resolve(from), $to = doc4.resolve(to);
    if (fitsTrivially($from, $to, slice))
      return new ReplaceStep(from, to, slice);
    return new Fitter($from, $to, slice).fit();
  }
  function fitsTrivially($from, $to, slice) {
    return !slice.openStart && !slice.openEnd && $from.start() == $to.start() && $from.parent.canReplace($from.index(), $to.index(), slice.content);
  }
  var Fitter = class {
    constructor($from, $to, unplaced) {
      this.$from = $from;
      this.$to = $to;
      this.unplaced = unplaced;
      this.frontier = [];
      this.placed = Fragment.empty;
      for (let i2 = 0; i2 <= $from.depth; i2++) {
        let node2 = $from.node(i2);
        this.frontier.push({
          type: node2.type,
          match: node2.contentMatchAt($from.indexAfter(i2))
        });
      }
      for (let i2 = $from.depth; i2 > 0; i2--)
        this.placed = Fragment.from($from.node(i2).copy(this.placed));
    }
    get depth() {
      return this.frontier.length - 1;
    }
    fit() {
      while (this.unplaced.size) {
        let fit = this.findFittable();
        if (fit)
          this.placeNodes(fit);
        else
          this.openMore() || this.dropNode();
      }
      let moveInline = this.mustMoveInline(), placedSize = this.placed.size - this.depth - this.$from.depth;
      let $from = this.$from, $to = this.close(moveInline < 0 ? this.$to : $from.doc.resolve(moveInline));
      if (!$to)
        return null;
      let content3 = this.placed, openStart = $from.depth, openEnd = $to.depth;
      while (openStart && openEnd && content3.childCount == 1) {
        content3 = content3.firstChild.content;
        openStart--;
        openEnd--;
      }
      let slice = new Slice2(content3, openStart, openEnd);
      if (moveInline > -1)
        return new ReplaceAroundStep($from.pos, moveInline, this.$to.pos, this.$to.end(), slice, placedSize);
      if (slice.size || $from.pos != this.$to.pos)
        return new ReplaceStep($from.pos, $to.pos, slice);
      return null;
    }
    // Find a position on the start spine of `this.unplaced` that has
    // content that can be moved somewhere on the frontier. Returns two
    // depths, one for the slice and one for the frontier.
    findFittable() {
      let startDepth = this.unplaced.openStart;
      for (let cur = this.unplaced.content, d2 = 0, openEnd = this.unplaced.openEnd; d2 < startDepth; d2++) {
        let node2 = cur.firstChild;
        if (cur.childCount > 1)
          openEnd = 0;
        if (node2.type.spec.isolating && openEnd <= d2) {
          startDepth = d2;
          break;
        }
        cur = node2.content;
      }
      for (let pass = 1; pass <= 2; pass++) {
        for (let sliceDepth = pass == 1 ? startDepth : this.unplaced.openStart; sliceDepth >= 0; sliceDepth--) {
          let fragment, parent = null;
          if (sliceDepth) {
            parent = contentAt(this.unplaced.content, sliceDepth - 1).firstChild;
            fragment = parent.content;
          } else {
            fragment = this.unplaced.content;
          }
          let first = fragment.firstChild;
          for (let frontierDepth = this.depth; frontierDepth >= 0; frontierDepth--) {
            let { type, match } = this.frontier[frontierDepth], wrap3, inject = null;
            if (pass == 1 && (first ? match.matchType(first.type) || (inject = match.fillBefore(Fragment.from(first), false)) : parent && type.compatibleContent(parent.type)))
              return { sliceDepth, frontierDepth, parent, inject };
            else if (pass == 2 && first && (wrap3 = match.findWrapping(first.type)))
              return { sliceDepth, frontierDepth, parent, wrap: wrap3 };
            if (parent && match.matchType(parent.type))
              break;
          }
        }
      }
    }
    openMore() {
      let { content: content3, openStart, openEnd } = this.unplaced;
      let inner = contentAt(content3, openStart);
      if (!inner.childCount || inner.firstChild.isLeaf)
        return false;
      this.unplaced = new Slice2(content3, openStart + 1, Math.max(openEnd, inner.size + openStart >= content3.size - openEnd ? openStart + 1 : 0));
      return true;
    }
    dropNode() {
      let { content: content3, openStart, openEnd } = this.unplaced;
      let inner = contentAt(content3, openStart);
      if (inner.childCount <= 1 && openStart > 0) {
        let openAtEnd = content3.size - openStart <= openStart + inner.size;
        this.unplaced = new Slice2(dropFromFragment(content3, openStart - 1, 1), openStart - 1, openAtEnd ? openStart - 1 : openEnd);
      } else {
        this.unplaced = new Slice2(dropFromFragment(content3, openStart, 1), openStart, openEnd);
      }
    }
    // Move content from the unplaced slice at `sliceDepth` to the
    // frontier node at `frontierDepth`. Close that frontier node when
    // applicable.
    placeNodes({ sliceDepth, frontierDepth, parent, inject, wrap: wrap3 }) {
      while (this.depth > frontierDepth)
        this.closeFrontierNode();
      if (wrap3)
        for (let i2 = 0; i2 < wrap3.length; i2++)
          this.openFrontierNode(wrap3[i2]);
      let slice = this.unplaced, fragment = parent ? parent.content : slice.content;
      let openStart = slice.openStart - sliceDepth;
      let taken = 0, add = [];
      let { match, type } = this.frontier[frontierDepth];
      if (inject) {
        for (let i2 = 0; i2 < inject.childCount; i2++)
          add.push(inject.child(i2));
        match = match.matchFragment(inject);
      }
      let openEndCount = fragment.size + sliceDepth - (slice.content.size - slice.openEnd);
      while (taken < fragment.childCount) {
        let next = fragment.child(taken), matches2 = match.matchType(next.type);
        if (!matches2)
          break;
        taken++;
        if (taken > 1 || openStart == 0 || next.content.size) {
          match = matches2;
          add.push(closeNodeStart(next.mark(type.allowedMarks(next.marks)), taken == 1 ? openStart : 0, taken == fragment.childCount ? openEndCount : -1));
        }
      }
      let toEnd = taken == fragment.childCount;
      if (!toEnd)
        openEndCount = -1;
      this.placed = addToFragment(this.placed, frontierDepth, Fragment.from(add));
      this.frontier[frontierDepth].match = match;
      if (toEnd && openEndCount < 0 && parent && parent.type == this.frontier[this.depth].type && this.frontier.length > 1)
        this.closeFrontierNode();
      for (let i2 = 0, cur = fragment; i2 < openEndCount; i2++) {
        let node2 = cur.lastChild;
        this.frontier.push({ type: node2.type, match: node2.contentMatchAt(node2.childCount) });
        cur = node2.content;
      }
      this.unplaced = !toEnd ? new Slice2(dropFromFragment(slice.content, sliceDepth, taken), slice.openStart, slice.openEnd) : sliceDepth == 0 ? Slice2.empty : new Slice2(dropFromFragment(slice.content, sliceDepth - 1, 1), sliceDepth - 1, openEndCount < 0 ? slice.openEnd : sliceDepth - 1);
    }
    mustMoveInline() {
      if (!this.$to.parent.isTextblock)
        return -1;
      let top = this.frontier[this.depth], level;
      if (!top.type.isTextblock || !contentAfterFits(this.$to, this.$to.depth, top.type, top.match, false) || this.$to.depth == this.depth && (level = this.findCloseLevel(this.$to)) && level.depth == this.depth)
        return -1;
      let { depth } = this.$to, after = this.$to.after(depth);
      while (depth > 1 && after == this.$to.end(--depth))
        ++after;
      return after;
    }
    findCloseLevel($to) {
      scan: for (let i2 = Math.min(this.depth, $to.depth); i2 >= 0; i2--) {
        let { match, type } = this.frontier[i2];
        let dropInner = i2 < $to.depth && $to.end(i2 + 1) == $to.pos + ($to.depth - (i2 + 1));
        let fit = contentAfterFits($to, i2, type, match, dropInner);
        if (!fit)
          continue;
        for (let d2 = i2 - 1; d2 >= 0; d2--) {
          let { match: match2, type: type2 } = this.frontier[d2];
          let matches2 = contentAfterFits($to, d2, type2, match2, true);
          if (!matches2 || matches2.childCount)
            continue scan;
        }
        return { depth: i2, fit, move: dropInner ? $to.doc.resolve($to.after(i2 + 1)) : $to };
      }
    }
    close($to) {
      let close2 = this.findCloseLevel($to);
      if (!close2)
        return null;
      while (this.depth > close2.depth)
        this.closeFrontierNode();
      if (close2.fit.childCount)
        this.placed = addToFragment(this.placed, close2.depth, close2.fit);
      $to = close2.move;
      for (let d2 = close2.depth + 1; d2 <= $to.depth; d2++) {
        let node2 = $to.node(d2), add = node2.type.contentMatch.fillBefore(node2.content, true, $to.index(d2));
        this.openFrontierNode(node2.type, node2.attrs, add);
      }
      return $to;
    }
    openFrontierNode(type, attrs = null, content3) {
      let top = this.frontier[this.depth];
      top.match = top.match.matchType(type);
      this.placed = addToFragment(this.placed, this.depth, Fragment.from(type.create(attrs, content3)));
      this.frontier.push({ type, match: type.contentMatch });
    }
    closeFrontierNode() {
      let open = this.frontier.pop();
      let add = open.match.fillBefore(Fragment.empty, true);
      if (add.childCount)
        this.placed = addToFragment(this.placed, this.frontier.length, add);
    }
  };
  function dropFromFragment(fragment, depth, count) {
    if (depth == 0)
      return fragment.cutByIndex(count, fragment.childCount);
    return fragment.replaceChild(0, fragment.firstChild.copy(dropFromFragment(fragment.firstChild.content, depth - 1, count)));
  }
  function addToFragment(fragment, depth, content3) {
    if (depth == 0)
      return fragment.append(content3);
    return fragment.replaceChild(fragment.childCount - 1, fragment.lastChild.copy(addToFragment(fragment.lastChild.content, depth - 1, content3)));
  }
  function contentAt(fragment, depth) {
    for (let i2 = 0; i2 < depth; i2++)
      fragment = fragment.firstChild.content;
    return fragment;
  }
  function closeNodeStart(node2, openStart, openEnd) {
    if (openStart <= 0)
      return node2;
    let frag = node2.content;
    if (openStart > 1)
      frag = frag.replaceChild(0, closeNodeStart(frag.firstChild, openStart - 1, frag.childCount == 1 ? openEnd - 1 : 0));
    if (openStart > 0) {
      frag = node2.type.contentMatch.fillBefore(frag).append(frag);
      if (openEnd <= 0)
        frag = frag.append(node2.type.contentMatch.matchFragment(frag).fillBefore(Fragment.empty, true));
    }
    return node2.copy(frag);
  }
  function contentAfterFits($to, depth, type, match, open) {
    let node2 = $to.node(depth), index2 = open ? $to.indexAfter(depth) : $to.index(depth);
    if (index2 == node2.childCount && !type.compatibleContent(node2.type))
      return null;
    let fit = match.fillBefore(node2.content, true, index2);
    return fit && !invalidMarks(type, node2.content, index2) ? fit : null;
  }
  function invalidMarks(type, fragment, start) {
    for (let i2 = start; i2 < fragment.childCount; i2++)
      if (!type.allowsMarks(fragment.child(i2).marks))
        return true;
    return false;
  }
  function definesContent(type) {
    return type.spec.defining || type.spec.definingForContent;
  }
  function replaceRange(tr, from, to, slice) {
    if (!slice.size)
      return tr.deleteRange(from, to);
    let $from = tr.doc.resolve(from), $to = tr.doc.resolve(to);
    if (fitsTrivially($from, $to, slice))
      return tr.step(new ReplaceStep(from, to, slice));
    let targetDepths = coveredDepths($from, $to);
    if (targetDepths[targetDepths.length - 1] == 0)
      targetDepths.pop();
    let preferredTarget = -($from.depth + 1);
    targetDepths.unshift(preferredTarget);
    for (let d2 = $from.depth, pos = $from.pos - 1; d2 > 0; d2--, pos--) {
      let spec = $from.node(d2).type.spec;
      if (spec.defining || spec.definingAsContext || spec.isolating)
        break;
      if (targetDepths.indexOf(d2) > -1)
        preferredTarget = d2;
      else if ($from.before(d2) == pos)
        targetDepths.splice(1, 0, -d2);
    }
    let preferredTargetIndex = targetDepths.indexOf(preferredTarget);
    let leftNodes = [], preferredDepth = slice.openStart;
    for (let content3 = slice.content, i2 = 0; ; i2++) {
      let node2 = content3.firstChild;
      leftNodes.push(node2);
      if (i2 == slice.openStart)
        break;
      content3 = node2.content;
    }
    for (let d2 = preferredDepth - 1; d2 >= 0; d2--) {
      let leftNode = leftNodes[d2], def = definesContent(leftNode.type);
      if (def && !leftNode.sameMarkup($from.node(Math.abs(preferredTarget) - 1)))
        preferredDepth = d2;
      else if (def || !leftNode.type.isTextblock)
        break;
    }
    for (let j = slice.openStart; j >= 0; j--) {
      let openDepth = (j + preferredDepth + 1) % (slice.openStart + 1);
      let insert = leftNodes[openDepth];
      if (!insert)
        continue;
      for (let i2 = 0; i2 < targetDepths.length; i2++) {
        let targetDepth = targetDepths[(i2 + preferredTargetIndex) % targetDepths.length], expand = true;
        if (targetDepth < 0) {
          expand = false;
          targetDepth = -targetDepth;
        }
        let parent = $from.node(targetDepth - 1), index2 = $from.index(targetDepth - 1);
        if (parent.canReplaceWith(index2, index2, insert.type, insert.marks))
          return tr.replace($from.before(targetDepth), expand ? $to.after(targetDepth) : to, new Slice2(closeFragment(slice.content, 0, slice.openStart, openDepth), openDepth, slice.openEnd));
      }
    }
    let startSteps = tr.steps.length;
    for (let i2 = targetDepths.length - 1; i2 >= 0; i2--) {
      tr.replace(from, to, slice);
      if (tr.steps.length > startSteps)
        break;
      let depth = targetDepths[i2];
      if (depth < 0)
        continue;
      from = $from.before(depth);
      to = $to.after(depth);
    }
  }
  function closeFragment(fragment, depth, oldOpen, newOpen, parent) {
    if (depth < oldOpen) {
      let first = fragment.firstChild;
      fragment = fragment.replaceChild(0, first.copy(closeFragment(first.content, depth + 1, oldOpen, newOpen, first)));
    }
    if (depth > newOpen) {
      let match = parent.contentMatchAt(0);
      let start = match.fillBefore(fragment).append(fragment);
      fragment = start.append(match.matchFragment(start).fillBefore(Fragment.empty, true));
    }
    return fragment;
  }
  function replaceRangeWith(tr, from, to, node2) {
    if (!node2.isInline && from == to && tr.doc.resolve(from).parent.content.size) {
      let point3 = insertPoint(tr.doc, from, node2.type);
      if (point3 != null)
        from = to = point3;
    }
    tr.replaceRange(from, to, new Slice2(Fragment.from(node2), 0, 0));
  }
  function deleteRange(tr, from, to) {
    let $from = tr.doc.resolve(from), $to = tr.doc.resolve(to);
    let covered = coveredDepths($from, $to);
    for (let i2 = 0; i2 < covered.length; i2++) {
      let depth = covered[i2], last = i2 == covered.length - 1;
      if (last && depth == 0 || $from.node(depth).type.contentMatch.validEnd)
        return tr.delete($from.start(depth), $to.end(depth));
      if (depth > 0 && (last || $from.node(depth - 1).canReplace($from.index(depth - 1), $to.indexAfter(depth - 1))))
        return tr.delete($from.before(depth), $to.after(depth));
    }
    for (let d2 = 1; d2 <= $from.depth && d2 <= $to.depth; d2++) {
      if (from - $from.start(d2) == $from.depth - d2 && to > $from.end(d2) && $to.end(d2) - to != $to.depth - d2 && $from.start(d2 - 1) == $to.start(d2 - 1) && $from.node(d2 - 1).canReplace($from.index(d2 - 1), $to.index(d2 - 1)))
        return tr.delete($from.before(d2), to);
    }
    tr.delete(from, to);
  }
  function coveredDepths($from, $to) {
    let result = [], minDepth = Math.min($from.depth, $to.depth);
    for (let d2 = minDepth; d2 >= 0; d2--) {
      let start = $from.start(d2);
      if (start < $from.pos - ($from.depth - d2) || $to.end(d2) > $to.pos + ($to.depth - d2) || $from.node(d2).type.spec.isolating || $to.node(d2).type.spec.isolating)
        break;
      if (start == $to.start(d2) || d2 == $from.depth && d2 == $to.depth && $from.parent.inlineContent && $to.parent.inlineContent && d2 && $to.start(d2 - 1) == start - 1)
        result.push(d2);
    }
    return result;
  }
  var AttrStep = class _AttrStep extends Step {
    /**
    Construct an attribute step.
    */
    constructor(pos, attr, value) {
      super();
      this.pos = pos;
      this.attr = attr;
      this.value = value;
    }
    apply(doc4) {
      let node2 = doc4.nodeAt(this.pos);
      if (!node2)
        return StepResult.fail("No node at attribute step's position");
      let attrs = /* @__PURE__ */ Object.create(null);
      for (let name in node2.attrs)
        attrs[name] = node2.attrs[name];
      attrs[this.attr] = this.value;
      let updated = node2.type.create(attrs, null, node2.marks);
      return StepResult.fromReplace(doc4, this.pos, this.pos + 1, new Slice2(Fragment.from(updated), 0, node2.isLeaf ? 0 : 1));
    }
    getMap() {
      return StepMap.empty;
    }
    invert(doc4) {
      return new _AttrStep(this.pos, this.attr, doc4.nodeAt(this.pos).attrs[this.attr]);
    }
    map(mapping) {
      let pos = mapping.mapResult(this.pos, 1);
      return pos.deletedAfter ? null : new _AttrStep(pos.pos, this.attr, this.value);
    }
    toJSON() {
      return { stepType: "attr", pos: this.pos, attr: this.attr, value: this.value };
    }
    static fromJSON(schema3, json) {
      if (typeof json.pos != "number" || typeof json.attr != "string")
        throw new RangeError("Invalid input for AttrStep.fromJSON");
      return new _AttrStep(json.pos, json.attr, json.value);
    }
  };
  Step.jsonID("attr", AttrStep);
  var DocAttrStep = class _DocAttrStep extends Step {
    /**
    Construct an attribute step.
    */
    constructor(attr, value) {
      super();
      this.attr = attr;
      this.value = value;
    }
    apply(doc4) {
      let attrs = /* @__PURE__ */ Object.create(null);
      for (let name in doc4.attrs)
        attrs[name] = doc4.attrs[name];
      attrs[this.attr] = this.value;
      let updated = doc4.type.create(attrs, doc4.content, doc4.marks);
      return StepResult.ok(updated);
    }
    getMap() {
      return StepMap.empty;
    }
    invert(doc4) {
      return new _DocAttrStep(this.attr, doc4.attrs[this.attr]);
    }
    map(mapping) {
      return this;
    }
    toJSON() {
      return { stepType: "docAttr", attr: this.attr, value: this.value };
    }
    static fromJSON(schema3, json) {
      if (typeof json.attr != "string")
        throw new RangeError("Invalid input for DocAttrStep.fromJSON");
      return new _DocAttrStep(json.attr, json.value);
    }
  };
  Step.jsonID("docAttr", DocAttrStep);
  var TransformError = class extends Error {
  };
  TransformError = function TransformError2(message) {
    let err = Error.call(this, message);
    err.__proto__ = TransformError2.prototype;
    return err;
  };
  TransformError.prototype = Object.create(Error.prototype);
  TransformError.prototype.constructor = TransformError;
  TransformError.prototype.name = "TransformError";
  var Transform = class {
    /**
    Create a transform that starts with the given document.
    */
    constructor(doc4) {
      this.doc = doc4;
      this.steps = [];
      this.docs = [];
      this.mapping = new Mapping();
    }
    /**
    The starting document.
    */
    get before() {
      return this.docs.length ? this.docs[0] : this.doc;
    }
    /**
    Apply a new step in this transform, saving the result. Throws an
    error when the step fails.
    */
    step(step) {
      let result = this.maybeStep(step);
      if (result.failed)
        throw new TransformError(result.failed);
      return this;
    }
    /**
    Try to apply a step in this transformation, ignoring it if it
    fails. Returns the step result.
    */
    maybeStep(step) {
      let result = step.apply(this.doc);
      if (!result.failed)
        this.addStep(step, result.doc);
      return result;
    }
    /**
    True when the document has been changed (when there are any
    steps).
    */
    get docChanged() {
      return this.steps.length > 0;
    }
    /**
    Return a single range, in post-transform document positions,
    that covers all content changed by this transform. Returns null
    if no replacements are made. Note that this will ignore changes
    that add/remove marks without replacing the underlying content.
    */
    changedRange() {
      let from = 1e9, to = -1e9;
      for (let i2 = 0; i2 < this.mapping.maps.length; i2++) {
        let map4 = this.mapping.maps[i2];
        if (i2) {
          from = map4.map(from, 1);
          to = map4.map(to, -1);
        }
        map4.forEach((_f, _t, fromB, toB) => {
          from = Math.min(from, fromB);
          to = Math.max(to, toB);
        });
      }
      return from == 1e9 ? null : { from, to };
    }
    /**
    @internal
    */
    addStep(step, doc4) {
      this.docs.push(this.doc);
      this.steps.push(step);
      this.mapping.appendMap(step.getMap());
      this.doc = doc4;
    }
    /**
    Replace the part of the document between `from` and `to` with the
    given `slice`.
    */
    replace(from, to = from, slice = Slice2.empty) {
      let step = replaceStep(this.doc, from, to, slice);
      if (step)
        this.step(step);
      return this;
    }
    /**
    Replace the given range with the given content, which may be a
    fragment, node, or array of nodes.
    */
    replaceWith(from, to, content3) {
      return this.replace(from, to, new Slice2(Fragment.from(content3), 0, 0));
    }
    /**
    Delete the content between the given positions.
    */
    delete(from, to) {
      return this.replace(from, to, Slice2.empty);
    }
    /**
    Insert the given content at the given position.
    */
    insert(pos, content3) {
      return this.replaceWith(pos, pos, content3);
    }
    /**
    Replace a range of the document with a given slice, using
    `from`, `to`, and the slice's
    [`openStart`](https://prosemirror.net/docs/ref/#model.Slice.openStart) property as hints, rather
    than fixed start and end points. This method may grow the
    replaced area or close open nodes in the slice in order to get a
    fit that is more in line with WYSIWYG expectations, by dropping
    fully covered parent nodes of the replaced region when they are
    marked [non-defining as
    context](https://prosemirror.net/docs/ref/#model.NodeSpec.definingAsContext), or including an
    open parent node from the slice that _is_ marked as [defining
    its content](https://prosemirror.net/docs/ref/#model.NodeSpec.definingForContent).
    
    This is the method, for example, to handle paste. The similar
    [`replace`](https://prosemirror.net/docs/ref/#transform.Transform.replace) method is a more
    primitive tool which will _not_ move the start and end of its given
    range, and is useful in situations where you need more precise
    control over what happens.
    */
    replaceRange(from, to, slice) {
      replaceRange(this, from, to, slice);
      return this;
    }
    /**
    Replace the given range with a node, but use `from` and `to` as
    hints, rather than precise positions. When from and to are the same
    and are at the start or end of a parent node in which the given
    node doesn't fit, this method may _move_ them out towards a parent
    that does allow the given node to be placed. When the given range
    completely covers a parent node, this method may completely replace
    that parent node.
    */
    replaceRangeWith(from, to, node2) {
      replaceRangeWith(this, from, to, node2);
      return this;
    }
    /**
    Delete the given range, expanding it to cover fully covered
    parent nodes until a valid replace is found.
    */
    deleteRange(from, to) {
      deleteRange(this, from, to);
      return this;
    }
    /**
    Split the content in the given range off from its parent, if there
    is sibling content before or after it, and move it up the tree to
    the depth specified by `target`. You'll probably want to use
    [`liftTarget`](https://prosemirror.net/docs/ref/#transform.liftTarget) to compute `target`, to make
    sure the lift is valid.
    */
    lift(range, target) {
      lift(this, range, target);
      return this;
    }
    /**
    Join the blocks around the given position. If depth is 2, their
    last and first siblings are also joined, and so on.
    */
    join(pos, depth = 1) {
      join3(this, pos, depth);
      return this;
    }
    /**
    Wrap the given [range](https://prosemirror.net/docs/ref/#model.NodeRange) in the given set of wrappers.
    The wrappers are assumed to be valid in this position, and should
    probably be computed with [`findWrapping`](https://prosemirror.net/docs/ref/#transform.findWrapping).
    */
    wrap(range, wrappers) {
      wrap2(this, range, wrappers);
      return this;
    }
    /**
    Set the type of all textblocks (partly) between `from` and `to` to
    the given node type with the given attributes.
    */
    setBlockType(from, to = from, type, attrs = null) {
      setBlockType(this, from, to, type, attrs);
      return this;
    }
    /**
    Change the type, attributes, and/or marks of the node at `pos`.
    When `type` isn't given, the existing node type is preserved,
    */
    setNodeMarkup(pos, type, attrs = null, marks) {
      setNodeMarkup(this, pos, type, attrs, marks);
      return this;
    }
    /**
    Set a single attribute on a given node to a new value.
    The `pos` addresses the document content. Use `setDocAttribute`
    to set attributes on the document itself.
    */
    setNodeAttribute(pos, attr, value) {
      this.step(new AttrStep(pos, attr, value));
      return this;
    }
    /**
    Set a single attribute on the document to a new value.
    */
    setDocAttribute(attr, value) {
      this.step(new DocAttrStep(attr, value));
      return this;
    }
    /**
    Add a mark to the node at position `pos`.
    */
    addNodeMark(pos, mark) {
      this.step(new AddNodeMarkStep(pos, mark));
      return this;
    }
    /**
    Remove a mark (or all marks of the given type) from the node at
    position `pos`.
    */
    removeNodeMark(pos, mark) {
      let node2 = this.doc.nodeAt(pos);
      if (!node2)
        throw new RangeError("No node at position " + pos);
      if (mark instanceof Mark) {
        if (mark.isInSet(node2.marks))
          this.step(new RemoveNodeMarkStep(pos, mark));
      } else {
        let set = node2.marks, found2, steps = [];
        while (found2 = mark.isInSet(set)) {
          steps.push(new RemoveNodeMarkStep(pos, found2));
          set = found2.removeFromSet(set);
        }
        for (let i2 = steps.length - 1; i2 >= 0; i2--)
          this.step(steps[i2]);
      }
      return this;
    }
    /**
    Split the node at the given position, and optionally, if `depth` is
    greater than one, any number of nodes above that. By default, the
    parts split off will inherit the node type of the original node.
    This can be changed by passing an array of types and attributes to
    use after the split (with the outermost nodes coming first).
    */
    split(pos, depth = 1, typesAfter) {
      split(this, pos, depth, typesAfter);
      return this;
    }
    /**
    Add the given mark to the inline content between `from` and `to`.
    */
    addMark(from, to, mark) {
      addMark(this, from, to, mark);
      return this;
    }
    /**
    Remove marks from inline nodes between `from` and `to`. When
    `mark` is a single mark, remove precisely that mark. When it is
    a mark type, remove all marks of that type. When it is null,
    remove all marks of any type.
    */
    removeMark(from, to, mark) {
      removeMark(this, from, to, mark);
      return this;
    }
    /**
    Removes all marks and nodes from the content of the node at
    `pos` that don't match the given new parent node type. Accepts
    an optional starting [content match](https://prosemirror.net/docs/ref/#model.ContentMatch) as
    third argument.
    */
    clearIncompatible(pos, parentType, match) {
      clearIncompatible(this, pos, parentType, match);
      return this;
    }
  };

  // node_modules/prosemirror-state/dist/index.js
  var classesById = /* @__PURE__ */ Object.create(null);
  var Selection = class {
    /**
    Initialize a selection with the head and anchor and ranges. If no
    ranges are given, constructs a single range across `$anchor` and
    `$head`.
    */
    constructor($anchor, $head, ranges) {
      this.$anchor = $anchor;
      this.$head = $head;
      this.ranges = ranges || [new SelectionRange($anchor.min($head), $anchor.max($head))];
    }
    /**
    The selection's anchor, as an unresolved position.
    */
    get anchor() {
      return this.$anchor.pos;
    }
    /**
    The selection's head.
    */
    get head() {
      return this.$head.pos;
    }
    /**
    The lower bound of the selection's main range.
    */
    get from() {
      return this.$from.pos;
    }
    /**
    The upper bound of the selection's main range.
    */
    get to() {
      return this.$to.pos;
    }
    /**
    The resolved lower  bound of the selection's main range.
    */
    get $from() {
      return this.ranges[0].$from;
    }
    /**
    The resolved upper bound of the selection's main range.
    */
    get $to() {
      return this.ranges[0].$to;
    }
    /**
    Indicates whether the selection contains any content.
    */
    get empty() {
      let ranges = this.ranges;
      for (let i2 = 0; i2 < ranges.length; i2++)
        if (ranges[i2].$from.pos != ranges[i2].$to.pos)
          return false;
      return true;
    }
    /**
    Get the content of this selection as a slice.
    */
    content() {
      return this.$from.doc.slice(this.from, this.to, true);
    }
    /**
    Replace the selection with a slice or, if no slice is given,
    delete the selection. Will append to the given transaction.
    */
    replace(tr, content3 = Slice2.empty) {
      let lastNode = content3.content.lastChild, lastParent = null;
      for (let i2 = 0; i2 < content3.openEnd; i2++) {
        lastParent = lastNode;
        lastNode = lastNode.lastChild;
      }
      let mapFrom = tr.steps.length, ranges = this.ranges;
      for (let i2 = 0; i2 < ranges.length; i2++) {
        let { $from, $to } = ranges[i2], mapping = tr.mapping.slice(mapFrom);
        tr.replaceRange(mapping.map($from.pos), mapping.map($to.pos), i2 ? Slice2.empty : content3);
        if (i2 == 0)
          selectionToInsertionEnd(tr, mapFrom, (lastNode ? lastNode.isInline : lastParent && lastParent.isTextblock) ? -1 : 1);
      }
    }
    /**
    Replace the selection with the given node, appending the changes
    to the given transaction.
    */
    replaceWith(tr, node2) {
      let mapFrom = tr.steps.length, ranges = this.ranges;
      for (let i2 = 0; i2 < ranges.length; i2++) {
        let { $from, $to } = ranges[i2], mapping = tr.mapping.slice(mapFrom);
        let from = mapping.map($from.pos), to = mapping.map($to.pos);
        if (i2) {
          tr.deleteRange(from, to);
        } else {
          tr.replaceRangeWith(from, to, node2);
          selectionToInsertionEnd(tr, mapFrom, node2.isInline ? -1 : 1);
        }
      }
    }
    /**
    Find a valid cursor or leaf node selection starting at the given
    position and searching back if `dir` is negative, and forward if
    positive. When `textOnly` is true, only consider cursor
    selections. Will return null when no valid selection position is
    found.
    */
    static findFrom($pos, dir, textOnly = false) {
      let inner = $pos.parent.inlineContent ? new TextSelection($pos) : findSelectionIn($pos.node(0), $pos.parent, $pos.pos, $pos.index(), dir, textOnly);
      if (inner)
        return inner;
      for (let depth = $pos.depth - 1; depth >= 0; depth--) {
        let found2 = dir < 0 ? findSelectionIn($pos.node(0), $pos.node(depth), $pos.before(depth + 1), $pos.index(depth), dir, textOnly) : findSelectionIn($pos.node(0), $pos.node(depth), $pos.after(depth + 1), $pos.index(depth) + 1, dir, textOnly);
        if (found2)
          return found2;
      }
      return null;
    }
    /**
    Find a valid cursor or leaf node selection near the given
    position. Searches forward first by default, but if `bias` is
    negative, it will search backwards first.
    */
    static near($pos, bias = 1) {
      return this.findFrom($pos, bias) || this.findFrom($pos, -bias) || new AllSelection($pos.node(0));
    }
    /**
    Find the cursor or leaf node selection closest to the start of
    the given document. Will return an
    [`AllSelection`](https://prosemirror.net/docs/ref/#state.AllSelection) if no valid position
    exists.
    */
    static atStart(doc4) {
      return findSelectionIn(doc4, doc4, 0, 0, 1) || new AllSelection(doc4);
    }
    /**
    Find the cursor or leaf node selection closest to the end of the
    given document.
    */
    static atEnd(doc4) {
      return findSelectionIn(doc4, doc4, doc4.content.size, doc4.childCount, -1) || new AllSelection(doc4);
    }
    /**
    Deserialize the JSON representation of a selection. Must be
    implemented for custom classes (as a static class method).
    */
    static fromJSON(doc4, json) {
      if (!json || !json.type)
        throw new RangeError("Invalid input for Selection.fromJSON");
      let cls = classesById[json.type];
      if (!cls)
        throw new RangeError(`No selection type ${json.type} defined`);
      return cls.fromJSON(doc4, json);
    }
    /**
    To be able to deserialize selections from JSON, custom selection
    classes must register themselves with an ID string, so that they
    can be disambiguated. Try to pick something that's unlikely to
    clash with classes from other modules.
    */
    static jsonID(id, selectionClass) {
      if (id in classesById)
        throw new RangeError("Duplicate use of selection JSON ID " + id);
      classesById[id] = selectionClass;
      selectionClass.prototype.jsonID = id;
      return selectionClass;
    }
    /**
    Get a [bookmark](https://prosemirror.net/docs/ref/#state.SelectionBookmark) for this selection,
    which is a value that can be mapped without having access to a
    current document, and later resolved to a real selection for a
    given document again. (This is used mostly by the history to
    track and restore old selections.) The default implementation of
    this method just converts the selection to a text selection and
    returns the bookmark for that.
    */
    getBookmark() {
      return TextSelection.between(this.$anchor, this.$head).getBookmark();
    }
  };
  Selection.prototype.visible = true;
  var SelectionRange = class {
    /**
    Create a range.
    */
    constructor($from, $to) {
      this.$from = $from;
      this.$to = $to;
    }
  };
  var warnedAboutTextSelection = false;
  function checkTextSelection($pos) {
    if (!warnedAboutTextSelection && !$pos.parent.inlineContent) {
      warnedAboutTextSelection = true;
      console["warn"]("TextSelection endpoint not pointing into a node with inline content (" + $pos.parent.type.name + ")");
    }
  }
  var TextSelection = class _TextSelection extends Selection {
    /**
    Construct a text selection between the given points.
    */
    constructor($anchor, $head = $anchor) {
      checkTextSelection($anchor);
      checkTextSelection($head);
      super($anchor, $head);
    }
    /**
    Returns a resolved position if this is a cursor selection (an
    empty text selection), and null otherwise.
    */
    get $cursor() {
      return this.$anchor.pos == this.$head.pos ? this.$head : null;
    }
    map(doc4, mapping) {
      let $head = doc4.resolve(mapping.map(this.head));
      if (!$head.parent.inlineContent)
        return Selection.near($head);
      let $anchor = doc4.resolve(mapping.map(this.anchor));
      return new _TextSelection($anchor.parent.inlineContent ? $anchor : $head, $head);
    }
    replace(tr, content3 = Slice2.empty) {
      super.replace(tr, content3);
      if (content3 == Slice2.empty) {
        let marks = this.$from.marksAcross(this.$to);
        if (marks)
          tr.ensureMarks(marks);
      }
    }
    eq(other) {
      return other instanceof _TextSelection && other.anchor == this.anchor && other.head == this.head;
    }
    getBookmark() {
      return new TextBookmark(this.anchor, this.head);
    }
    toJSON() {
      return { type: "text", anchor: this.anchor, head: this.head };
    }
    /**
    @internal
    */
    static fromJSON(doc4, json) {
      if (typeof json.anchor != "number" || typeof json.head != "number")
        throw new RangeError("Invalid input for TextSelection.fromJSON");
      return new _TextSelection(doc4.resolve(json.anchor), doc4.resolve(json.head));
    }
    /**
    Create a text selection from non-resolved positions.
    */
    static create(doc4, anchor, head = anchor) {
      let $anchor = doc4.resolve(anchor);
      return new this($anchor, head == anchor ? $anchor : doc4.resolve(head));
    }
    /**
    Return a text selection that spans the given positions or, if
    they aren't text positions, find a text selection near them.
    `bias` determines whether the method searches forward (default)
    or backwards (negative number) first. Will fall back to calling
    [`Selection.near`](https://prosemirror.net/docs/ref/#state.Selection^near) when the document
    doesn't contain a valid text position.
    */
    static between($anchor, $head, bias) {
      let dPos = $anchor.pos - $head.pos;
      if (!bias || dPos)
        bias = dPos >= 0 ? 1 : -1;
      if (!$head.parent.inlineContent) {
        let found2 = Selection.findFrom($head, bias, true) || Selection.findFrom($head, -bias, true);
        if (found2)
          $head = found2.$head;
        else
          return Selection.near($head, bias);
      }
      if (!$anchor.parent.inlineContent) {
        if (dPos == 0) {
          $anchor = $head;
        } else {
          $anchor = (Selection.findFrom($anchor, -bias, true) || Selection.findFrom($anchor, bias, true)).$anchor;
          if ($anchor.pos < $head.pos != dPos < 0)
            $anchor = $head;
        }
      }
      return new _TextSelection($anchor, $head);
    }
  };
  Selection.jsonID("text", TextSelection);
  var TextBookmark = class _TextBookmark {
    constructor(anchor, head) {
      this.anchor = anchor;
      this.head = head;
    }
    map(mapping) {
      return new _TextBookmark(mapping.map(this.anchor), mapping.map(this.head));
    }
    resolve(doc4) {
      return TextSelection.between(doc4.resolve(this.anchor), doc4.resolve(this.head));
    }
  };
  var NodeSelection = class _NodeSelection extends Selection {
    /**
    Create a node selection. Does not verify the validity of its
    argument.
    */
    constructor($pos) {
      let node2 = $pos.nodeAfter;
      let $end = $pos.node(0).resolve($pos.pos + node2.nodeSize);
      super($pos, $end);
      this.node = node2;
    }
    map(doc4, mapping) {
      let { deleted, pos } = mapping.mapResult(this.anchor);
      let $pos = doc4.resolve(pos);
      if (deleted)
        return Selection.near($pos);
      return new _NodeSelection($pos);
    }
    content() {
      return new Slice2(Fragment.from(this.node), 0, 0);
    }
    eq(other) {
      return other instanceof _NodeSelection && other.anchor == this.anchor;
    }
    toJSON() {
      return { type: "node", anchor: this.anchor };
    }
    getBookmark() {
      return new NodeBookmark(this.anchor);
    }
    /**
    @internal
    */
    static fromJSON(doc4, json) {
      if (typeof json.anchor != "number")
        throw new RangeError("Invalid input for NodeSelection.fromJSON");
      return new _NodeSelection(doc4.resolve(json.anchor));
    }
    /**
    Create a node selection from non-resolved positions.
    */
    static create(doc4, from) {
      return new _NodeSelection(doc4.resolve(from));
    }
    /**
    Determines whether the given node may be selected as a node
    selection.
    */
    static isSelectable(node2) {
      return !node2.isText && node2.type.spec.selectable !== false;
    }
  };
  NodeSelection.prototype.visible = false;
  Selection.jsonID("node", NodeSelection);
  var NodeBookmark = class _NodeBookmark {
    constructor(anchor) {
      this.anchor = anchor;
    }
    map(mapping) {
      let { deleted, pos } = mapping.mapResult(this.anchor);
      return deleted ? new TextBookmark(pos, pos) : new _NodeBookmark(pos);
    }
    resolve(doc4) {
      let $pos = doc4.resolve(this.anchor), node2 = $pos.nodeAfter;
      if (node2 && NodeSelection.isSelectable(node2))
        return new NodeSelection($pos);
      return Selection.near($pos);
    }
  };
  var AllSelection = class _AllSelection extends Selection {
    /**
    Create an all-selection over the given document.
    */
    constructor(doc4) {
      super(doc4.resolve(0), doc4.resolve(doc4.content.size));
    }
    replace(tr, content3 = Slice2.empty) {
      if (content3 == Slice2.empty) {
        tr.delete(0, tr.doc.content.size);
        let sel = Selection.atStart(tr.doc);
        if (!sel.eq(tr.selection))
          tr.setSelection(sel);
      } else {
        super.replace(tr, content3);
      }
    }
    toJSON() {
      return { type: "all" };
    }
    /**
    @internal
    */
    static fromJSON(doc4) {
      return new _AllSelection(doc4);
    }
    map(doc4) {
      return new _AllSelection(doc4);
    }
    eq(other) {
      return other instanceof _AllSelection;
    }
    getBookmark() {
      return AllBookmark;
    }
  };
  Selection.jsonID("all", AllSelection);
  var AllBookmark = {
    map() {
      return this;
    },
    resolve(doc4) {
      return new AllSelection(doc4);
    }
  };
  function findSelectionIn(doc4, node2, pos, index2, dir, text4 = false) {
    if (node2.inlineContent)
      return TextSelection.create(doc4, pos);
    for (let i2 = index2 - (dir > 0 ? 0 : 1); dir > 0 ? i2 < node2.childCount : i2 >= 0; i2 += dir) {
      let child = node2.child(i2);
      if (!child.isAtom) {
        let inner = findSelectionIn(doc4, child, pos + dir, dir < 0 ? child.childCount : 0, dir, text4);
        if (inner)
          return inner;
      } else if (!text4 && NodeSelection.isSelectable(child)) {
        return NodeSelection.create(doc4, pos - (dir < 0 ? child.nodeSize : 0));
      }
      pos += child.nodeSize * dir;
    }
    return null;
  }
  function selectionToInsertionEnd(tr, startLen, bias) {
    let last = tr.steps.length - 1;
    if (last < startLen)
      return;
    let step = tr.steps[last];
    if (!(step instanceof ReplaceStep || step instanceof ReplaceAroundStep))
      return;
    let map4 = tr.mapping.maps[last], end;
    map4.forEach((_from, _to, _newFrom, newTo) => {
      if (end == null)
        end = newTo;
    });
    tr.setSelection(Selection.near(tr.doc.resolve(end), bias));
  }
  var UPDATED_SEL = 1;
  var UPDATED_MARKS = 2;
  var UPDATED_SCROLL = 4;
  var Transaction = class extends Transform {
    /**
    @internal
    */
    constructor(state) {
      super(state.doc);
      this.curSelectionFor = 0;
      this.updated = 0;
      this.meta = /* @__PURE__ */ Object.create(null);
      this.time = Date.now();
      this.curSelection = state.selection;
      this.storedMarks = state.storedMarks;
    }
    /**
    The transaction's current selection. This defaults to the editor
    selection [mapped](https://prosemirror.net/docs/ref/#state.Selection.map) through the steps in the
    transaction, but can be overwritten with
    [`setSelection`](https://prosemirror.net/docs/ref/#state.Transaction.setSelection).
    */
    get selection() {
      if (this.curSelectionFor < this.steps.length) {
        this.curSelection = this.curSelection.map(this.doc, this.mapping.slice(this.curSelectionFor));
        this.curSelectionFor = this.steps.length;
      }
      return this.curSelection;
    }
    /**
    Update the transaction's current selection. Will determine the
    selection that the editor gets when the transaction is applied.
    */
    setSelection(selection) {
      if (selection.$from.doc != this.doc)
        throw new RangeError("Selection passed to setSelection must point at the current document");
      this.curSelection = selection;
      this.curSelectionFor = this.steps.length;
      this.updated = (this.updated | UPDATED_SEL) & ~UPDATED_MARKS;
      this.storedMarks = null;
      return this;
    }
    /**
    Whether the selection was explicitly updated by this transaction.
    */
    get selectionSet() {
      return (this.updated & UPDATED_SEL) > 0;
    }
    /**
    Set the current stored marks.
    */
    setStoredMarks(marks) {
      this.storedMarks = marks;
      this.updated |= UPDATED_MARKS;
      return this;
    }
    /**
    Make sure the current stored marks or, if that is null, the marks
    at the selection, match the given set of marks. Does nothing if
    this is already the case.
    */
    ensureMarks(marks) {
      if (!Mark.sameSet(this.storedMarks || this.selection.$from.marks(), marks))
        this.setStoredMarks(marks);
      return this;
    }
    /**
    Add a mark to the set of stored marks.
    */
    addStoredMark(mark) {
      return this.ensureMarks(mark.addToSet(this.storedMarks || this.selection.$head.marks()));
    }
    /**
    Remove a mark or mark type from the set of stored marks.
    */
    removeStoredMark(mark) {
      return this.ensureMarks(mark.removeFromSet(this.storedMarks || this.selection.$head.marks()));
    }
    /**
    Whether the stored marks were explicitly set for this transaction.
    */
    get storedMarksSet() {
      return (this.updated & UPDATED_MARKS) > 0;
    }
    /**
    @internal
    */
    addStep(step, doc4) {
      super.addStep(step, doc4);
      this.updated = this.updated & ~UPDATED_MARKS;
      this.storedMarks = null;
    }
    /**
    Update the timestamp for the transaction.
    */
    setTime(time) {
      this.time = time;
      return this;
    }
    /**
    Replace the current selection with the given slice.
    */
    replaceSelection(slice) {
      this.selection.replace(this, slice);
      return this;
    }
    /**
    Replace the selection with the given node. When `inheritMarks` is
    true and the content is inline, it inherits the marks from the
    place where it is inserted.
    */
    replaceSelectionWith(node2, inheritMarks = true) {
      let selection = this.selection;
      if (inheritMarks)
        node2 = node2.mark(this.storedMarks || (selection.empty ? selection.$from.marks() : selection.$from.marksAcross(selection.$to) || Mark.none));
      selection.replaceWith(this, node2);
      return this;
    }
    /**
    Delete the selection.
    */
    deleteSelection() {
      this.selection.replace(this);
      return this;
    }
    /**
    Replace the given range, or the selection if no range is given,
    with a text node containing the given string.
    */
    insertText(text4, from, to) {
      let schema3 = this.doc.type.schema;
      if (from == null) {
        if (!text4)
          return this.deleteSelection();
        return this.replaceSelectionWith(schema3.text(text4), true);
      } else {
        if (to == null)
          to = from;
        if (!text4)
          return this.deleteRange(from, to);
        let marks = this.storedMarks;
        if (!marks) {
          let $from = this.doc.resolve(from);
          marks = to == from ? $from.marks() : $from.marksAcross(this.doc.resolve(to));
        }
        this.replaceRangeWith(from, to, schema3.text(text4, marks));
        if (!this.selection.empty && this.selection.to == from + text4.length)
          this.setSelection(Selection.near(this.selection.$to));
        return this;
      }
    }
    /**
    Store a metadata property in this transaction, keyed either by
    name or by plugin.
    */
    setMeta(key3, value) {
      this.meta[typeof key3 == "string" ? key3 : key3.key] = value;
      return this;
    }
    /**
    Retrieve a metadata property for a given name or plugin.
    */
    getMeta(key3) {
      return this.meta[typeof key3 == "string" ? key3 : key3.key];
    }
    /**
    Returns true if this transaction doesn't contain any metadata,
    and can thus safely be extended.
    */
    get isGeneric() {
      for (let _ in this.meta)
        return false;
      return true;
    }
    /**
    Indicate that the editor should scroll the selection into view
    when updated to the state produced by this transaction.
    */
    scrollIntoView() {
      this.updated |= UPDATED_SCROLL;
      return this;
    }
    /**
    True when this transaction has had `scrollIntoView` called on it.
    */
    get scrolledIntoView() {
      return (this.updated & UPDATED_SCROLL) > 0;
    }
  };
  function bind(f2, self2) {
    return !self2 || !f2 ? f2 : f2.bind(self2);
  }
  var FieldDesc = class {
    constructor(name, desc, self2) {
      this.name = name;
      this.init = bind(desc.init, self2);
      this.apply = bind(desc.apply, self2);
    }
  };
  var baseFields = [
    new FieldDesc("doc", {
      init(config2) {
        return config2.doc || config2.schema.topNodeType.createAndFill();
      },
      apply(tr) {
        return tr.doc;
      }
    }),
    new FieldDesc("selection", {
      init(config2, instance) {
        return config2.selection || Selection.atStart(instance.doc);
      },
      apply(tr) {
        return tr.selection;
      }
    }),
    new FieldDesc("storedMarks", {
      init(config2) {
        return config2.storedMarks || null;
      },
      apply(tr, _marks3, _old, state) {
        return state.selection.$cursor ? tr.storedMarks : null;
      }
    }),
    new FieldDesc("scrollToSelection", {
      init() {
        return 0;
      },
      apply(tr, prev) {
        return tr.scrolledIntoView ? prev + 1 : prev;
      }
    })
  ];
  var Configuration = class {
    constructor(schema3, plugins2) {
      this.schema = schema3;
      this.plugins = [];
      this.pluginsByKey = /* @__PURE__ */ Object.create(null);
      this.fields = baseFields.slice();
      if (plugins2)
        plugins2.forEach((plugin) => {
          if (this.pluginsByKey[plugin.key])
            throw new RangeError("Adding different instances of a keyed plugin (" + plugin.key + ")");
          this.plugins.push(plugin);
          this.pluginsByKey[plugin.key] = plugin;
          if (plugin.spec.state)
            this.fields.push(new FieldDesc(plugin.key, plugin.spec.state, plugin));
        });
    }
  };
  var EditorState = class _EditorState {
    /**
    @internal
    */
    constructor(config2) {
      this.config = config2;
    }
    /**
    The schema of the state's document.
    */
    get schema() {
      return this.config.schema;
    }
    /**
    The plugins that are active in this state.
    */
    get plugins() {
      return this.config.plugins;
    }
    /**
    Apply the given transaction to produce a new state.
    */
    apply(tr) {
      return this.applyTransaction(tr).state;
    }
    /**
    @internal
    */
    filterTransaction(tr, ignore = -1) {
      for (let i2 = 0; i2 < this.config.plugins.length; i2++)
        if (i2 != ignore) {
          let plugin = this.config.plugins[i2];
          if (plugin.spec.filterTransaction && !plugin.spec.filterTransaction.call(plugin, tr, this))
            return false;
        }
      return true;
    }
    /**
    Verbose variant of [`apply`](https://prosemirror.net/docs/ref/#state.EditorState.apply) that
    returns the precise transactions that were applied (which might
    be influenced by the [transaction
    hooks](https://prosemirror.net/docs/ref/#state.PluginSpec.filterTransaction) of
    plugins) along with the new state.
    */
    applyTransaction(rootTr) {
      if (!this.filterTransaction(rootTr))
        return { state: this, transactions: [] };
      let trs = [rootTr], newState = this.applyInner(rootTr), seen = null;
      for (; ; ) {
        let haveNew = false;
        for (let i2 = 0; i2 < this.config.plugins.length; i2++) {
          let plugin = this.config.plugins[i2];
          if (plugin.spec.appendTransaction) {
            let n2 = seen ? seen[i2].n : 0, oldState = seen ? seen[i2].state : this;
            let tr = n2 < trs.length && plugin.spec.appendTransaction.call(plugin, n2 ? trs.slice(n2) : trs, oldState, newState);
            if (tr && newState.filterTransaction(tr, i2)) {
              tr.setMeta("appendedTransaction", rootTr);
              if (!seen) {
                seen = [];
                for (let j = 0; j < this.config.plugins.length; j++)
                  seen.push(j < i2 ? { state: newState, n: trs.length } : { state: this, n: 0 });
              }
              trs.push(tr);
              newState = newState.applyInner(tr);
              haveNew = true;
            }
            if (seen)
              seen[i2] = { state: newState, n: trs.length };
          }
        }
        if (!haveNew)
          return { state: newState, transactions: trs };
      }
    }
    /**
    @internal
    */
    applyInner(tr) {
      if (!tr.before.eq(this.doc))
        throw new RangeError("Applying a mismatched transaction");
      let newInstance = new _EditorState(this.config), fields = this.config.fields;
      for (let i2 = 0; i2 < fields.length; i2++) {
        let field = fields[i2];
        newInstance[field.name] = field.apply(tr, this[field.name], this, newInstance);
      }
      return newInstance;
    }
    /**
    Accessor that constructs and returns a new [transaction](https://prosemirror.net/docs/ref/#state.Transaction) from this state.
    */
    get tr() {
      return new Transaction(this);
    }
    /**
    Create a new state.
    */
    static create(config2) {
      let $config = new Configuration(config2.doc ? config2.doc.type.schema : config2.schema, config2.plugins);
      let instance = new _EditorState($config);
      for (let i2 = 0; i2 < $config.fields.length; i2++)
        instance[$config.fields[i2].name] = $config.fields[i2].init(config2, instance);
      return instance;
    }
    /**
    Create a new state based on this one, but with an adjusted set
    of active plugins. State fields that exist in both sets of
    plugins are kept unchanged. Those that no longer exist are
    dropped, and those that are new are initialized using their
    [`init`](https://prosemirror.net/docs/ref/#state.StateField.init) method, passing in the new
    configuration object..
    */
    reconfigure(config2) {
      let $config = new Configuration(this.schema, config2.plugins);
      let fields = $config.fields, instance = new _EditorState($config);
      for (let i2 = 0; i2 < fields.length; i2++) {
        let name = fields[i2].name;
        instance[name] = this.hasOwnProperty(name) ? this[name] : fields[i2].init(config2, instance);
      }
      return instance;
    }
    /**
    Serialize this state to JSON. If you want to serialize the state
    of plugins, pass an object mapping property names to use in the
    resulting JSON object to plugin objects. The argument may also be
    a string or number, in which case it is ignored, to support the
    way `JSON.stringify` calls `toString` methods.
    */
    toJSON(pluginFields) {
      let result = { doc: this.doc.toJSON(), selection: this.selection.toJSON() };
      if (this.storedMarks)
        result.storedMarks = this.storedMarks.map((m) => m.toJSON());
      if (pluginFields && typeof pluginFields == "object")
        for (let prop in pluginFields) {
          if (prop == "doc" || prop == "selection")
            throw new RangeError("The JSON fields `doc` and `selection` are reserved");
          let plugin = pluginFields[prop], state = plugin.spec.state;
          if (state && state.toJSON)
            result[prop] = state.toJSON.call(plugin, this[plugin.key]);
        }
      return result;
    }
    /**
    Deserialize a JSON representation of a state. `config` should
    have at least a `schema` field, and should contain array of
    plugins to initialize the state with. `pluginFields` can be used
    to deserialize the state of plugins, by associating plugin
    instances with the property names they use in the JSON object.
    */
    static fromJSON(config2, json, pluginFields) {
      if (!json)
        throw new RangeError("Invalid input for EditorState.fromJSON");
      if (!config2.schema)
        throw new RangeError("Required config field 'schema' missing");
      let $config = new Configuration(config2.schema, config2.plugins);
      let instance = new _EditorState($config);
      $config.fields.forEach((field) => {
        if (field.name == "doc") {
          instance.doc = Node.fromJSON(config2.schema, json.doc);
        } else if (field.name == "selection") {
          instance.selection = Selection.fromJSON(instance.doc, json.selection);
        } else if (field.name == "storedMarks") {
          if (json.storedMarks)
            instance.storedMarks = json.storedMarks.map(config2.schema.markFromJSON);
        } else {
          if (pluginFields)
            for (let prop in pluginFields) {
              let plugin = pluginFields[prop], state = plugin.spec.state;
              if (plugin.key == field.name && state && state.fromJSON && Object.prototype.hasOwnProperty.call(json, prop)) {
                instance[field.name] = state.fromJSON.call(plugin, config2, json[prop], instance);
                return;
              }
            }
          instance[field.name] = field.init(config2, instance);
        }
      });
      return instance;
    }
  };
  function bindProps(obj, self2, target) {
    for (let prop in obj) {
      let val = obj[prop];
      if (val instanceof Function)
        val = val.bind(self2);
      else if (prop == "handleDOMEvents")
        val = bindProps(val, self2, {});
      target[prop] = val;
    }
    return target;
  }
  var Plugin = class {
    /**
    Create a plugin.
    */
    constructor(spec) {
      this.spec = spec;
      this.props = {};
      if (spec.props)
        bindProps(spec.props, this, this.props);
      this.key = spec.key ? spec.key.key : createKey("plugin");
    }
    /**
    Extract the plugin's state field from an editor state.
    */
    getState(state) {
      return state[this.key];
    }
  };
  var keys = /* @__PURE__ */ Object.create(null);
  function createKey(name) {
    if (name in keys)
      return name + "$" + ++keys[name];
    keys[name] = 0;
    return name + "$";
  }
  var PluginKey = class {
    /**
    Create a plugin key.
    */
    constructor(name = "key") {
      this.key = createKey(name);
    }
    /**
    Get the active plugin with this key, if any, from an editor
    state.
    */
    get(state) {
      return state.config.pluginsByKey[this.key];
    }
    /**
    Get the plugin's state from an editor state.
    */
    getState(state) {
      return state[this.key];
    }
  };

  // node_modules/prosemirror-commands/dist/index.js
  var deleteSelection = (state, dispatch) => {
    if (state.selection.empty)
      return false;
    if (dispatch)
      dispatch(state.tr.deleteSelection().scrollIntoView());
    return true;
  };
  function atBlockStart(state, view) {
    let { $cursor } = state.selection;
    if (!$cursor || (view ? !view.endOfTextblock("backward", state) : $cursor.parentOffset > 0))
      return null;
    return $cursor;
  }
  var joinBackward = (state, dispatch, view) => {
    let $cursor = atBlockStart(state, view);
    if (!$cursor)
      return false;
    let $cut = findCutBefore($cursor);
    if (!$cut) {
      let range = $cursor.blockRange(), target = range && liftTarget(range);
      if (target == null)
        return false;
      if (dispatch)
        dispatch(state.tr.lift(range, target).scrollIntoView());
      return true;
    }
    let before = $cut.nodeBefore;
    if (deleteBarrier(state, $cut, dispatch, -1))
      return true;
    if ($cursor.parent.content.size == 0 && (textblockAt(before, "end") || NodeSelection.isSelectable(before))) {
      for (let depth = $cursor.depth; ; depth--) {
        let delStep = replaceStep(state.doc, $cursor.before(depth), $cursor.after(depth), Slice2.empty);
        if (delStep && delStep.slice.size < delStep.to - delStep.from) {
          if (dispatch) {
            let tr = state.tr.step(delStep);
            tr.setSelection(textblockAt(before, "end") ? Selection.findFrom(tr.doc.resolve(tr.mapping.map($cut.pos, -1)), -1) : NodeSelection.create(tr.doc, $cut.pos - before.nodeSize));
            dispatch(tr.scrollIntoView());
          }
          return true;
        }
        if (depth == 1 || $cursor.node(depth - 1).childCount > 1)
          break;
      }
    }
    if (before.isAtom && $cut.depth == $cursor.depth - 1) {
      if (dispatch)
        dispatch(state.tr.delete($cut.pos - before.nodeSize, $cut.pos).scrollIntoView());
      return true;
    }
    return false;
  };
  var joinTextblockBackward = (state, dispatch, view) => {
    let $cursor = atBlockStart(state, view);
    if (!$cursor)
      return false;
    let $cut = findCutBefore($cursor);
    return $cut ? joinTextblocksAround(state, $cut, dispatch) : false;
  };
  function joinTextblocksAround(state, $cut, dispatch) {
    let before = $cut.nodeBefore, beforeText = before, beforePos = $cut.pos - 1;
    for (; !beforeText.isTextblock; beforePos--) {
      if (beforeText.type.spec.isolating)
        return false;
      let child = beforeText.lastChild;
      if (!child)
        return false;
      beforeText = child;
    }
    let after = $cut.nodeAfter, afterText = after, afterPos = $cut.pos + 1;
    for (; !afterText.isTextblock; afterPos++) {
      if (afterText.type.spec.isolating)
        return false;
      let child = afterText.firstChild;
      if (!child)
        return false;
      afterText = child;
    }
    let step = replaceStep(state.doc, beforePos, afterPos, Slice2.empty);
    if (!step || step.from != beforePos || step instanceof ReplaceStep && step.slice.size >= afterPos - beforePos)
      return false;
    if (dispatch) {
      let tr = state.tr.step(step);
      tr.setSelection(TextSelection.create(tr.doc, beforePos));
      dispatch(tr.scrollIntoView());
    }
    return true;
  }
  function textblockAt(node2, side, only = false) {
    for (let scan = node2; scan; scan = side == "start" ? scan.firstChild : scan.lastChild) {
      if (scan.isTextblock)
        return true;
      if (only && scan.childCount != 1)
        return false;
    }
    return false;
  }
  var selectNodeBackward = (state, dispatch, view) => {
    let { $head, empty: empty3 } = state.selection, $cut = $head;
    if (!empty3)
      return false;
    if ($head.parent.isTextblock) {
      if (view ? !view.endOfTextblock("backward", state) : $head.parentOffset > 0)
        return false;
      $cut = findCutBefore($head);
    }
    let node2 = $cut && $cut.nodeBefore;
    if (!node2 || !NodeSelection.isSelectable(node2))
      return false;
    if (dispatch)
      dispatch(state.tr.setSelection(NodeSelection.create(state.doc, $cut.pos - node2.nodeSize)).scrollIntoView());
    return true;
  };
  function findCutBefore($pos) {
    if (!$pos.parent.type.spec.isolating)
      for (let i2 = $pos.depth - 1; i2 >= 0; i2--) {
        if ($pos.index(i2) > 0)
          return $pos.doc.resolve($pos.before(i2 + 1));
        if ($pos.node(i2).type.spec.isolating)
          break;
      }
    return null;
  }
  function atBlockEnd(state, view) {
    let { $cursor } = state.selection;
    if (!$cursor || (view ? !view.endOfTextblock("forward", state) : $cursor.parentOffset < $cursor.parent.content.size))
      return null;
    return $cursor;
  }
  var joinForward = (state, dispatch, view) => {
    let $cursor = atBlockEnd(state, view);
    if (!$cursor)
      return false;
    let $cut = findCutAfter($cursor);
    if (!$cut)
      return false;
    let after = $cut.nodeAfter;
    if (deleteBarrier(state, $cut, dispatch, 1))
      return true;
    if ($cursor.parent.content.size == 0 && (textblockAt(after, "start") || NodeSelection.isSelectable(after))) {
      let delStep = replaceStep(state.doc, $cursor.before(), $cursor.after(), Slice2.empty);
      if (delStep && delStep.slice.size < delStep.to - delStep.from) {
        if (dispatch) {
          let tr = state.tr.step(delStep);
          tr.setSelection(textblockAt(after, "start") ? Selection.findFrom(tr.doc.resolve(tr.mapping.map($cut.pos)), 1) : NodeSelection.create(tr.doc, tr.mapping.map($cut.pos)));
          dispatch(tr.scrollIntoView());
        }
        return true;
      }
    }
    if (after.isAtom && $cut.depth == $cursor.depth - 1) {
      if (dispatch)
        dispatch(state.tr.delete($cut.pos, $cut.pos + after.nodeSize).scrollIntoView());
      return true;
    }
    return false;
  };
  var selectNodeForward = (state, dispatch, view) => {
    let { $head, empty: empty3 } = state.selection, $cut = $head;
    if (!empty3)
      return false;
    if ($head.parent.isTextblock) {
      if (view ? !view.endOfTextblock("forward", state) : $head.parentOffset < $head.parent.content.size)
        return false;
      $cut = findCutAfter($head);
    }
    let node2 = $cut && $cut.nodeAfter;
    if (!node2 || !NodeSelection.isSelectable(node2))
      return false;
    if (dispatch)
      dispatch(state.tr.setSelection(NodeSelection.create(state.doc, $cut.pos)).scrollIntoView());
    return true;
  };
  function findCutAfter($pos) {
    if (!$pos.parent.type.spec.isolating)
      for (let i2 = $pos.depth - 1; i2 >= 0; i2--) {
        let parent = $pos.node(i2);
        if ($pos.index(i2) + 1 < parent.childCount)
          return $pos.doc.resolve($pos.after(i2 + 1));
        if (parent.type.spec.isolating)
          break;
      }
    return null;
  }
  var newlineInCode = (state, dispatch) => {
    let { $head, $anchor } = state.selection;
    if (!$head.parent.type.spec.code || !$head.sameParent($anchor))
      return false;
    if (dispatch)
      dispatch(state.tr.insertText("\n").scrollIntoView());
    return true;
  };
  function defaultBlockAt(match) {
    for (let i2 = 0; i2 < match.edgeCount; i2++) {
      let { type } = match.edge(i2);
      if (type.isTextblock && !type.hasRequiredAttrs())
        return type;
    }
    return null;
  }
  var exitCode = (state, dispatch) => {
    let { $head, $anchor } = state.selection;
    if (!$head.parent.type.spec.code || !$head.sameParent($anchor))
      return false;
    let above = $head.node(-1), after = $head.indexAfter(-1), type = defaultBlockAt(above.contentMatchAt(after));
    if (!type || !above.canReplaceWith(after, after, type))
      return false;
    if (dispatch) {
      let pos = $head.after(), tr = state.tr.replaceWith(pos, pos, type.createAndFill());
      tr.setSelection(Selection.near(tr.doc.resolve(pos), 1));
      dispatch(tr.scrollIntoView());
    }
    return true;
  };
  var createParagraphNear = (state, dispatch) => {
    let sel = state.selection, { $from, $to } = sel;
    if (sel instanceof AllSelection || $from.parent.inlineContent || $to.parent.inlineContent)
      return false;
    let type = defaultBlockAt($to.parent.contentMatchAt($to.indexAfter()));
    if (!type || !type.isTextblock)
      return false;
    if (dispatch) {
      let side = (!$from.parentOffset && $to.index() < $to.parent.childCount ? $from : $to).pos;
      let tr = state.tr.insert(side, type.createAndFill());
      tr.setSelection(TextSelection.create(tr.doc, side + 1));
      dispatch(tr.scrollIntoView());
    }
    return true;
  };
  var liftEmptyBlock = (state, dispatch) => {
    let { $cursor } = state.selection;
    if (!$cursor || $cursor.parent.content.size)
      return false;
    if ($cursor.depth > 1 && $cursor.after() != $cursor.end(-1)) {
      let before = $cursor.before();
      if (canSplit(state.doc, before)) {
        if (dispatch)
          dispatch(state.tr.split(before).scrollIntoView());
        return true;
      }
    }
    let range = $cursor.blockRange(), target = range && liftTarget(range);
    if (target == null)
      return false;
    if (dispatch)
      dispatch(state.tr.lift(range, target).scrollIntoView());
    return true;
  };
  function splitBlockAs(splitNode) {
    return (state, dispatch) => {
      let { $from, $to } = state.selection;
      if (state.selection instanceof NodeSelection && state.selection.node.isBlock) {
        if (!$from.parentOffset || !canSplit(state.doc, $from.pos))
          return false;
        if (dispatch)
          dispatch(state.tr.split($from.pos).scrollIntoView());
        return true;
      }
      if (!$from.depth)
        return false;
      let types = [];
      let splitDepth, deflt, atEnd = false, atStart = false;
      for (let d2 = $from.depth; ; d2--) {
        let node2 = $from.node(d2);
        if (node2.isBlock) {
          atEnd = $from.end(d2) == $from.pos + ($from.depth - d2);
          atStart = $from.start(d2) == $from.pos - ($from.depth - d2);
          deflt = defaultBlockAt($from.node(d2 - 1).contentMatchAt($from.indexAfter(d2 - 1)));
          let splitType = splitNode && splitNode($to.parent, atEnd, $from);
          types.unshift(splitType || (atEnd && deflt ? { type: deflt } : null));
          splitDepth = d2;
          break;
        } else {
          if (d2 == 1)
            return false;
          types.unshift(null);
        }
      }
      let tr = state.tr;
      if (state.selection instanceof TextSelection || state.selection instanceof AllSelection)
        tr.deleteSelection();
      let splitPos = tr.mapping.map($from.pos);
      let can = canSplit(tr.doc, splitPos, types.length, types);
      if (!can) {
        types[0] = deflt ? { type: deflt } : null;
        can = canSplit(tr.doc, splitPos, types.length, types);
      }
      if (!can)
        return false;
      tr.split(splitPos, types.length, types);
      if (!atEnd && atStart && $from.node(splitDepth).type != deflt) {
        let first = tr.mapping.map($from.before(splitDepth)), $first = tr.doc.resolve(first);
        if (deflt && $from.node(splitDepth - 1).canReplaceWith($first.index(), $first.index() + 1, deflt))
          tr.setNodeMarkup(tr.mapping.map($from.before(splitDepth)), deflt);
      }
      if (dispatch)
        dispatch(tr.scrollIntoView());
      return true;
    };
  }
  var splitBlock = splitBlockAs();
  var selectAll = (state, dispatch) => {
    if (dispatch)
      dispatch(state.tr.setSelection(new AllSelection(state.doc)));
    return true;
  };
  function joinMaybeClear(state, $pos, dispatch) {
    let before = $pos.nodeBefore, after = $pos.nodeAfter, index2 = $pos.index();
    if (!before || !after || !before.type.compatibleContent(after.type))
      return false;
    if (!before.content.size && $pos.parent.canReplace(index2 - 1, index2)) {
      if (dispatch)
        dispatch(state.tr.delete($pos.pos - before.nodeSize, $pos.pos).scrollIntoView());
      return true;
    }
    if (!$pos.parent.canReplace(index2, index2 + 1) || !(after.isTextblock || canJoin(state.doc, $pos.pos)))
      return false;
    if (dispatch)
      dispatch(state.tr.join($pos.pos).scrollIntoView());
    return true;
  }
  function deleteBarrier(state, $cut, dispatch, dir) {
    let before = $cut.nodeBefore, after = $cut.nodeAfter, conn, match;
    let isolated = before.type.spec.isolating || after.type.spec.isolating;
    if (!isolated && joinMaybeClear(state, $cut, dispatch))
      return true;
    let canDelAfter = !isolated && $cut.parent.canReplace($cut.index(), $cut.index() + 1);
    if (canDelAfter && (conn = (match = before.contentMatchAt(before.childCount)).findWrapping(after.type)) && match.matchType(conn[0] || after.type).validEnd) {
      if (dispatch) {
        let end = $cut.pos + after.nodeSize, wrap3 = Fragment.empty;
        for (let i2 = conn.length - 1; i2 >= 0; i2--)
          wrap3 = Fragment.from(conn[i2].create(null, wrap3));
        wrap3 = Fragment.from(before.copy(wrap3));
        let tr = state.tr.step(new ReplaceAroundStep($cut.pos - 1, end, $cut.pos, end, new Slice2(wrap3, 1, 0), conn.length, true));
        let $joinAt = tr.doc.resolve(end + 2 * conn.length);
        if ($joinAt.nodeAfter && $joinAt.nodeAfter.type == before.type && canJoin(tr.doc, $joinAt.pos))
          tr.join($joinAt.pos);
        dispatch(tr.scrollIntoView());
      }
      return true;
    }
    let selAfter = after.type.spec.isolating || dir > 0 && isolated ? null : Selection.findFrom($cut, 1);
    let range = selAfter && selAfter.$from.blockRange(selAfter.$to), target = range && liftTarget(range);
    if (target != null && target >= $cut.depth) {
      if (dispatch)
        dispatch(state.tr.lift(range, target).scrollIntoView());
      return true;
    }
    if (canDelAfter && textblockAt(after, "start", true) && textblockAt(before, "end")) {
      let at = before, wrap3 = [];
      for (; ; ) {
        wrap3.push(at);
        if (at.isTextblock)
          break;
        at = at.lastChild;
      }
      let afterText = after, afterDepth = 1;
      for (; !afterText.isTextblock; afterText = afterText.firstChild)
        afterDepth++;
      if (at.canReplace(at.childCount, at.childCount, afterText.content)) {
        if (dispatch) {
          let end = Fragment.empty;
          for (let i2 = wrap3.length - 1; i2 >= 0; i2--)
            end = Fragment.from(wrap3[i2].copy(end));
          let tr = state.tr.step(new ReplaceAroundStep($cut.pos - wrap3.length, $cut.pos + after.nodeSize, $cut.pos + afterDepth, $cut.pos + after.nodeSize - afterDepth, new Slice2(end, wrap3.length, 0), 0, true));
          dispatch(tr.scrollIntoView());
        }
        return true;
      }
    }
    return false;
  }
  function selectTextblockSide(side) {
    return function(state, dispatch) {
      let sel = state.selection, $pos = side < 0 ? sel.$from : sel.$to;
      let depth = $pos.depth;
      while ($pos.node(depth).isInline) {
        if (!depth)
          return false;
        depth--;
      }
      if (!$pos.node(depth).isTextblock)
        return false;
      if (dispatch)
        dispatch(state.tr.setSelection(TextSelection.create(state.doc, side < 0 ? $pos.start(depth) : $pos.end(depth))));
      return true;
    };
  }
  var selectTextblockStart = selectTextblockSide(-1);
  var selectTextblockEnd = selectTextblockSide(1);
  function wrapIn(nodeType, attrs = null) {
    return function(state, dispatch) {
      let { $from, $to } = state.selection;
      let range = $from.blockRange($to), wrapping = range && findWrapping(range, nodeType, attrs);
      if (!wrapping)
        return false;
      if (dispatch)
        dispatch(state.tr.wrap(range, wrapping).scrollIntoView());
      return true;
    };
  }
  function setBlockType2(nodeType, attrs = null) {
    return function(state, dispatch) {
      let applicable = false;
      for (let i2 = 0; i2 < state.selection.ranges.length && !applicable; i2++) {
        let { $from: { pos: from }, $to: { pos: to } } = state.selection.ranges[i2];
        state.doc.nodesBetween(from, to, (node2, pos) => {
          if (applicable)
            return false;
          if (!node2.isTextblock || node2.hasMarkup(nodeType, attrs))
            return;
          if (node2.type == nodeType) {
            applicable = true;
          } else {
            let $pos = state.doc.resolve(pos), index2 = $pos.index();
            applicable = $pos.parent.canReplaceWith(index2, index2 + 1, nodeType);
          }
        });
      }
      if (!applicable)
        return false;
      if (dispatch) {
        let tr = state.tr;
        for (let i2 = 0; i2 < state.selection.ranges.length; i2++) {
          let { $from: { pos: from }, $to: { pos: to } } = state.selection.ranges[i2];
          tr.setBlockType(from, to, nodeType, attrs);
        }
        dispatch(tr.scrollIntoView());
      }
      return true;
    };
  }
  function markApplies(doc4, ranges, type, enterAtoms) {
    for (let i2 = 0; i2 < ranges.length; i2++) {
      let { $from, $to } = ranges[i2];
      let can = $from.depth == 0 ? doc4.inlineContent && doc4.type.allowsMarkType(type) : false;
      doc4.nodesBetween($from.pos, $to.pos, (node2, pos) => {
        if (can || !enterAtoms && node2.isAtom && node2.isInline && pos >= $from.pos && pos + node2.nodeSize <= $to.pos)
          return false;
        can = node2.inlineContent && node2.type.allowsMarkType(type);
      });
      if (can)
        return true;
    }
    return false;
  }
  function removeInlineAtoms(ranges) {
    let result = [];
    for (let i2 = 0; i2 < ranges.length; i2++) {
      let { $from, $to } = ranges[i2];
      $from.doc.nodesBetween($from.pos, $to.pos, (node2, pos) => {
        if (node2.isAtom && node2.content.size && node2.isInline && pos >= $from.pos && pos + node2.nodeSize <= $to.pos) {
          if (pos + 1 > $from.pos)
            result.push(new SelectionRange($from, $from.doc.resolve(pos + 1)));
          $from = $from.doc.resolve(pos + 1 + node2.content.size);
          return false;
        }
      });
      if ($from.pos < $to.pos)
        result.push(new SelectionRange($from, $to));
    }
    return result;
  }
  function toggleMark(markType, attrs = null, options) {
    let removeWhenPresent = (options && options.removeWhenPresent) !== false;
    let enterAtoms = (options && options.enterInlineAtoms) !== false;
    let dropSpace = !(options && options.includeWhitespace);
    return function(state, dispatch) {
      let { empty: empty3, $cursor, ranges } = state.selection;
      if (empty3 && !$cursor || !markApplies(state.doc, ranges, markType, enterAtoms))
        return false;
      if (dispatch) {
        if ($cursor) {
          if (markType.isInSet(state.storedMarks || $cursor.marks()))
            dispatch(state.tr.removeStoredMark(markType));
          else
            dispatch(state.tr.addStoredMark(markType.create(attrs)));
        } else {
          let add, tr = state.tr;
          if (!enterAtoms)
            ranges = removeInlineAtoms(ranges);
          if (removeWhenPresent) {
            add = !ranges.some((r2) => state.doc.rangeHasMark(r2.$from.pos, r2.$to.pos, markType));
          } else {
            add = !ranges.every((r2) => {
              let missing = false;
              tr.doc.nodesBetween(r2.$from.pos, r2.$to.pos, (node2, pos, parent) => {
                if (missing)
                  return false;
                missing = !markType.isInSet(node2.marks) && !!parent && parent.type.allowsMarkType(markType) && !(node2.isText && /^\s*$/.test(node2.textBetween(Math.max(0, r2.$from.pos - pos), Math.min(node2.nodeSize, r2.$to.pos - pos))));
              });
              return !missing;
            });
          }
          for (let i2 = 0; i2 < ranges.length; i2++) {
            let { $from, $to } = ranges[i2];
            if (!add) {
              tr.removeMark($from.pos, $to.pos, markType);
            } else {
              let from = $from.pos, to = $to.pos, start = $from.nodeAfter, end = $to.nodeBefore;
              let spaceStart = dropSpace && start && start.isText ? /^\s*/.exec(start.text)[0].length : 0;
              let spaceEnd = dropSpace && end && end.isText ? /\s*$/.exec(end.text)[0].length : 0;
              if (from + spaceStart < to) {
                from += spaceStart;
                to -= spaceEnd;
              }
              tr.addMark(from, to, markType.create(attrs));
            }
          }
          dispatch(tr.scrollIntoView());
        }
      }
      return true;
    };
  }
  function chainCommands(...commands3) {
    return function(state, dispatch, view) {
      for (let i2 = 0; i2 < commands3.length; i2++)
        if (commands3[i2](state, dispatch, view))
          return true;
      return false;
    };
  }
  var backspace = chainCommands(deleteSelection, joinBackward, selectNodeBackward);
  var del = chainCommands(deleteSelection, joinForward, selectNodeForward);
  var pcBaseKeymap = {
    "Enter": chainCommands(newlineInCode, createParagraphNear, liftEmptyBlock, splitBlock),
    "Mod-Enter": exitCode,
    "Backspace": backspace,
    "Mod-Backspace": backspace,
    "Shift-Backspace": backspace,
    "Delete": del,
    "Mod-Delete": del,
    "Mod-a": selectAll
  };
  var macBaseKeymap = {
    "Ctrl-h": pcBaseKeymap["Backspace"],
    "Alt-Backspace": pcBaseKeymap["Mod-Backspace"],
    "Ctrl-d": pcBaseKeymap["Delete"],
    "Ctrl-Alt-Backspace": pcBaseKeymap["Mod-Delete"],
    "Alt-Delete": pcBaseKeymap["Mod-Delete"],
    "Alt-d": pcBaseKeymap["Mod-Delete"],
    "Ctrl-a": selectTextblockStart,
    "Ctrl-e": selectTextblockEnd
  };
  for (let key3 in pcBaseKeymap)
    macBaseKeymap[key3] = pcBaseKeymap[key3];
  var mac = typeof navigator != "undefined" ? /Mac|iP(hone|[oa]d)/.test(navigator.platform) : typeof os != "undefined" && os.platform ? os.platform() == "darwin" : false;
  var baseKeymap = mac ? macBaseKeymap : pcBaseKeymap;

  // node_modules/prosemirror-inputrules/dist/index.js
  var InputRule = class {
    /**
    Create an input rule. The rule applies when the user typed
    something and the text directly in front of the cursor matches
    `match`, which should end with `$`.
    
    The `handler` can be a string, in which case the matched text, or
    the first matched group in the regexp, is replaced by that
    string.
    
    Or a it can be a function, which will be called with the match
    array produced by
    [`RegExp.exec`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/RegExp/exec),
    as well as the start and end of the matched range, and which can
    return a [transaction](https://prosemirror.net/docs/ref/#state.Transaction) that describes the
    rule's effect, or null to indicate the input was not handled.
    */
    constructor(match, handler, options = {}) {
      this.match = match;
      this.match = match;
      this.handler = typeof handler == "string" ? stringHandler(handler) : handler;
      this.undoable = options.undoable !== false;
      this.inCode = options.inCode || false;
      this.inCodeMark = options.inCodeMark !== false;
    }
  };
  function stringHandler(string3) {
    return function(state, match, start, end) {
      let insert = string3;
      if (match[1]) {
        let offset = match[0].lastIndexOf(match[1]);
        insert += match[0].slice(offset + match[1].length);
        start += offset;
        let cutOff = start - end;
        if (cutOff > 0) {
          insert = match[0].slice(offset - cutOff, offset) + insert;
          start = end;
        }
      }
      return state.tr.insertText(insert, start, end);
    };
  }
  var undoInputRule = (state, dispatch) => {
    let plugins2 = state.plugins;
    for (let i2 = 0; i2 < plugins2.length; i2++) {
      let plugin = plugins2[i2], undoable;
      if (plugin.spec.isInputRules && (undoable = plugin.getState(state))) {
        if (dispatch) {
          let tr = state.tr, toUndo = undoable.transform;
          for (let j = toUndo.steps.length - 1; j >= 0; j--)
            tr.step(toUndo.steps[j].invert(toUndo.docs[j]));
          if (undoable.text) {
            let marks = tr.doc.resolve(undoable.from).marks();
            tr.replaceWith(undoable.from, undoable.to, state.schema.text(undoable.text, marks));
          } else {
            tr.delete(undoable.from, undoable.to);
          }
          dispatch(tr);
        }
        return true;
      }
    }
    return false;
  };
  var emDash = new InputRule(/--$/, "\u2014", { inCodeMark: false });
  var ellipsis = new InputRule(/\.\.\.$/, "\u2026", { inCodeMark: false });
  var openDoubleQuote = new InputRule(/(?:^|[\s\{\[\(\<'"\u2018\u201C])(")$/, "\u201C", { inCodeMark: false });
  var closeDoubleQuote = new InputRule(/"$/, "\u201D", { inCodeMark: false });
  var openSingleQuote = new InputRule(/(?:^|[\s\{\[\(\<'"\u2018\u201C])(')$/, "\u2018", { inCodeMark: false });
  var closeSingleQuote = new InputRule(/'$/, "\u2019", { inCodeMark: false });
  function wrappingInputRule(regexp, nodeType, getAttrs = null, joinPredicate) {
    return new InputRule(regexp, (state, match, start, end) => {
      let attrs = getAttrs instanceof Function ? getAttrs(match) : getAttrs;
      let tr = state.tr.delete(start, end);
      let $start = tr.doc.resolve(start), range = $start.blockRange(), wrapping = range && findWrapping(range, nodeType, attrs);
      if (!wrapping)
        return null;
      tr.wrap(range, wrapping);
      let before = tr.doc.resolve(start - 1).nodeBefore;
      if (before && before.type == nodeType && canJoin(tr.doc, start - 1) && (!joinPredicate || joinPredicate(match, before)))
        tr.join(start - 1);
      return tr;
    });
  }
  function textblockTypeInputRule(regexp, nodeType, getAttrs = null) {
    return new InputRule(regexp, (state, match, start, end) => {
      let $start = state.doc.resolve(start);
      let attrs = getAttrs instanceof Function ? getAttrs(match) : getAttrs;
      if (!$start.node(-1).canReplaceWith($start.index(-1), $start.indexAfter(-1), nodeType))
        return null;
      return state.tr.delete(start, end).setBlockType(start, start, nodeType, attrs);
    });
  }

  // node_modules/@milkdown/prose/lib/index.js
  var nav = typeof navigator != "undefined" ? navigator : null;
  var doc2 = typeof document != "undefined" ? document : null;
  var agent = nav && nav.userAgent || "";
  var ie_edge = /Edge\/(\d+)/.exec(agent);
  var ie_upto10 = /MSIE \d/.exec(agent);
  var ie_11up = /Trident\/(?:[7-9]|\d{2,})\..*rv:(\d+)/.exec(agent);
  var ie = !!(ie_upto10 || ie_11up || ie_edge);
  var ie_version = ie_upto10 ? document.documentMode : ie_11up ? +ie_11up[1] : ie_edge ? +ie_edge[1] : 0;
  var gecko = !ie && /gecko\/(\d+)/i.test(agent);
  var gecko_version = gecko && +(/Firefox\/(\d+)/.exec(agent) || [0, 0])[1];
  var _chrome = !ie && /Chrome\/(\d+)/.exec(agent);
  var chrome_version = _chrome ? +_chrome[1] : 0;
  var safari = !ie && !!nav && /Apple Computer/.test(nav.vendor);
  var ios = safari && (/Mobile\/\w+/.test(agent) || !!nav && nav.maxTouchPoints > 2);
  var mac2 = ios || (nav ? /Mac/.test(nav.platform) : false);
  var android = /Android \d/.test(agent);
  var webkit = !!doc2 && "webkitFontSmoothing" in doc2.documentElement.style;
  var webkit_version = webkit ? +(/\bAppleWebKit\/(\d+)/.exec(navigator.userAgent) || [0, 0])[1] : 0;
  function run(view, from, to, text4, rules, plugin) {
    if (view.composing) return false;
    const state = view.state;
    const $from = state.doc.resolve(from);
    if ($from.parent.type.spec.code) return false;
    const textBefore = $from.parent.textBetween(
      Math.max(0, $from.parentOffset - 500),
      $from.parentOffset,
      void 0,
      "\uFFFC"
    ) + text4;
    for (let _matcher of rules) {
      const matcher = _matcher;
      const match = matcher.match.exec(textBefore);
      const tr = match && match[0] && matcher.handler(state, match, from - (match[0].length - text4.length), to);
      if (!tr) continue;
      if (matcher.undoable !== false)
        tr.setMeta(plugin, { transform: tr, from, to, text: text4 });
      view.dispatch(tr);
      return true;
    }
    return false;
  }
  var customInputRulesKey = new PluginKey("MILKDOWN_CUSTOM_INPUTRULES");
  function customInputRules({ rules }) {
    const plugin = new Plugin({
      key: customInputRulesKey,
      isInputRules: true,
      state: {
        init() {
          return null;
        },
        apply(tr, prev) {
          const stored = tr.getMeta(this);
          if (stored) return stored;
          return tr.selectionSet || tr.docChanged ? null : prev;
        }
      },
      props: {
        handleTextInput(view, from, to, text4) {
          return run(view, from, to, text4, rules, plugin);
        },
        handleDOMEvents: {
          compositionend: (view) => {
            setTimeout(() => {
              const { $cursor } = view.state.selection;
              if ($cursor) run(view, $cursor.pos, $cursor.pos, "", rules, plugin);
            });
            return false;
          }
        },
        handleKeyDown(view, event) {
          if (event.key !== "Enter") return false;
          const { $cursor } = view.state.selection;
          if ($cursor)
            return run(view, $cursor.pos, $cursor.pos, "\n", rules, plugin);
          return false;
        }
      }
    });
    return plugin;
  }
  function markRule(regexp, markType, options = {}) {
    return new InputRule(regexp, (state, match, start, end) => {
      var _a, _b, _c, _d;
      const { tr } = state;
      const matchLength = match.length;
      let group = match[matchLength - 1];
      let fullMatch = match[0];
      let initialStoredMarks = [];
      let markEnd = end;
      const captured = {
        group,
        fullMatch,
        start,
        end
      };
      const result = (_a = options.updateCaptured) == null ? void 0 : _a.call(options, captured);
      Object.assign(captured, result);
      ({ group, fullMatch, start, end } = captured);
      if (fullMatch === null) return null;
      if ((group == null ? void 0 : group.trim()) === "") return null;
      if (group) {
        const startSpaces = fullMatch.search(/\S/);
        const textStart = start + fullMatch.indexOf(group);
        const textEnd = textStart + group.length;
        initialStoredMarks = (_b = tr.storedMarks) != null ? _b : [];
        if (textEnd < end) tr.delete(textEnd, end);
        if (textStart > start) tr.delete(start + startSpaces, textStart);
        markEnd = start + startSpaces + group.length;
        const attrs = (_c = options.getAttr) == null ? void 0 : _c.call(options, match);
        tr.addMark(start, markEnd, markType.create(attrs));
        tr.setStoredMarks(initialStoredMarks);
        (_d = options.beforeDispatch) == null ? void 0 : _d.call(options, { match, start, end, tr });
      }
      return tr;
    });
  }
  function equalNodeType(nodeType, node2) {
    return Array.isArray(nodeType) && nodeType.includes(node2.type) || node2.type === nodeType;
  }
  function findSelectedNodeOfType(selection, nodeType) {
    if (!(selection instanceof NodeSelection)) return;
    const { node: node2, $from } = selection;
    if (equalNodeType(nodeType, node2))
      return {
        node: node2,
        pos: $from.pos,
        start: $from.start($from.depth),
        depth: $from.depth
      };
    return void 0;
  }
  var findNodeInSelection = (state, node2) => {
    const { selection, doc: doc4 } = state;
    if (selection instanceof NodeSelection) {
      return {
        hasNode: selection.node.type === node2,
        pos: selection.from,
        target: selection.node
      };
    }
    const { from, to } = selection;
    let hasNode = false;
    let pos = -1;
    let target = null;
    doc4.nodesBetween(from, to, (n2, p2) => {
      if (target) return false;
      if (n2.type === node2) {
        hasNode = true;
        pos = p2;
        target = n2;
        return false;
      }
      return true;
    });
    return {
      hasNode,
      pos,
      target
    };
  };

  // node_modules/w3c-keyname/index.js
  var base = {
    8: "Backspace",
    9: "Tab",
    10: "Enter",
    12: "NumLock",
    13: "Enter",
    16: "Shift",
    17: "Control",
    18: "Alt",
    20: "CapsLock",
    27: "Escape",
    32: " ",
    33: "PageUp",
    34: "PageDown",
    35: "End",
    36: "Home",
    37: "ArrowLeft",
    38: "ArrowUp",
    39: "ArrowRight",
    40: "ArrowDown",
    44: "PrintScreen",
    45: "Insert",
    46: "Delete",
    59: ";",
    61: "=",
    91: "Meta",
    92: "Meta",
    106: "*",
    107: "+",
    108: ",",
    109: "-",
    110: ".",
    111: "/",
    144: "NumLock",
    145: "ScrollLock",
    160: "Shift",
    161: "Shift",
    162: "Control",
    163: "Control",
    164: "Alt",
    165: "Alt",
    173: "-",
    186: ";",
    187: "=",
    188: ",",
    189: "-",
    190: ".",
    191: "/",
    192: "`",
    219: "[",
    220: "\\",
    221: "]",
    222: "'"
  };
  var shift = {
    48: ")",
    49: "!",
    50: "@",
    51: "#",
    52: "$",
    53: "%",
    54: "^",
    55: "&",
    56: "*",
    57: "(",
    59: ":",
    61: "+",
    173: "_",
    186: ":",
    187: "+",
    188: "<",
    189: "_",
    190: ">",
    191: "?",
    192: "~",
    219: "{",
    220: "|",
    221: "}",
    222: '"'
  };
  var mac3 = typeof navigator != "undefined" && /Mac/.test(navigator.platform);
  var ie2 = typeof navigator != "undefined" && /MSIE \d|Trident\/(?:[7-9]|\d{2,})\..*rv:(\d+)/.exec(navigator.userAgent);
  for (i2 = 0; i2 < 10; i2++) base[48 + i2] = base[96 + i2] = String(i2);
  var i2;
  for (i2 = 1; i2 <= 24; i2++) base[i2 + 111] = "F" + i2;
  var i2;
  for (i2 = 65; i2 <= 90; i2++) {
    base[i2] = String.fromCharCode(i2 + 32);
    shift[i2] = String.fromCharCode(i2);
  }
  var i2;
  for (code2 in base) if (!shift.hasOwnProperty(code2)) shift[code2] = base[code2];
  var code2;
  function keyName(event) {
    var ignoreKey = mac3 && event.metaKey && event.shiftKey && !event.ctrlKey && !event.altKey || ie2 && event.shiftKey && event.key && event.key.length == 1 || event.key == "Unidentified";
    var name = !ignoreKey && event.key || (event.shiftKey ? shift : base)[event.keyCode] || event.key || "Unidentified";
    if (name == "Esc") name = "Escape";
    if (name == "Del") name = "Delete";
    if (name == "Left") name = "ArrowLeft";
    if (name == "Up") name = "ArrowUp";
    if (name == "Right") name = "ArrowRight";
    if (name == "Down") name = "ArrowDown";
    return name;
  }

  // node_modules/prosemirror-keymap/dist/index.js
  var mac4 = typeof navigator != "undefined" && /Mac|iP(hone|[oa]d)/.test(navigator.platform);
  var windows = typeof navigator != "undefined" && /Win/.test(navigator.platform);
  function normalizeKeyName(name) {
    let parts = name.split(/-(?!$)/), result = parts[parts.length - 1];
    if (result == "Space")
      result = " ";
    let alt, ctrl, shift2, meta;
    for (let i2 = 0; i2 < parts.length - 1; i2++) {
      let mod = parts[i2];
      if (/^(cmd|meta|m)$/i.test(mod))
        meta = true;
      else if (/^a(lt)?$/i.test(mod))
        alt = true;
      else if (/^(c|ctrl|control)$/i.test(mod))
        ctrl = true;
      else if (/^s(hift)?$/i.test(mod))
        shift2 = true;
      else if (/^mod$/i.test(mod)) {
        if (mac4)
          meta = true;
        else
          ctrl = true;
      } else
        throw new Error("Unrecognized modifier name: " + mod);
    }
    if (alt)
      result = "Alt-" + result;
    if (ctrl)
      result = "Ctrl-" + result;
    if (meta)
      result = "Meta-" + result;
    if (shift2)
      result = "Shift-" + result;
    return result;
  }
  function normalize2(map4) {
    let copy2 = /* @__PURE__ */ Object.create(null);
    for (let prop in map4)
      copy2[normalizeKeyName(prop)] = map4[prop];
    return copy2;
  }
  function modifiers(name, event, shift2 = true) {
    if (event.altKey)
      name = "Alt-" + name;
    if (event.ctrlKey)
      name = "Ctrl-" + name;
    if (event.metaKey)
      name = "Meta-" + name;
    if (shift2 && event.shiftKey)
      name = "Shift-" + name;
    return name;
  }
  function keymap(bindings) {
    return new Plugin({ props: { handleKeyDown: keydownHandler(bindings) } });
  }
  function keydownHandler(bindings) {
    let map4 = normalize2(bindings);
    return function(view, event) {
      let name = keyName(event), baseName, direct = map4[modifiers(name, event)];
      if (direct && direct(view.state, view.dispatch, view))
        return true;
      if (name.length == 1 && name != " ") {
        if (event.shiftKey) {
          let noShift = map4[modifiers(name, event, false)];
          if (noShift && noShift(view.state, view.dispatch, view))
            return true;
        }
        if ((event.altKey || event.metaKey || event.ctrlKey) && // Ctrl-Alt may be used for AltGr on Windows
        !(windows && event.ctrlKey && event.altKey) && (baseName = base[event.keyCode]) && baseName != name) {
          let fromCode = map4[modifiers(baseName, event)];
          if (fromCode && fromCode(view.state, view.dispatch, view))
            return true;
        }
      }
      return false;
    };
  }

  // node_modules/prosemirror-view/dist/index.js
  var domIndex = function(node2) {
    for (var index2 = 0; ; index2++) {
      node2 = node2.previousSibling;
      if (!node2)
        return index2;
    }
  };
  var parentNode = function(node2) {
    let parent = node2.assignedSlot || node2.parentNode;
    return parent && parent.nodeType == 11 ? parent.host : parent;
  };
  var reusedRange = null;
  var textRange = function(node2, from, to) {
    let range = reusedRange || (reusedRange = document.createRange());
    range.setEnd(node2, to == null ? node2.nodeValue.length : to);
    range.setStart(node2, from || 0);
    return range;
  };
  var clearReusedRange = function() {
    reusedRange = null;
  };
  var isEquivalentPosition = function(node2, off, targetNode, targetOff) {
    return targetNode && (scanFor(node2, off, targetNode, targetOff, -1) || scanFor(node2, off, targetNode, targetOff, 1));
  };
  var atomElements = /^(img|br|input|textarea|hr)$/i;
  function scanFor(node2, off, targetNode, targetOff, dir) {
    var _a;
    for (; ; ) {
      if (node2 == targetNode && off == targetOff)
        return true;
      if (off == (dir < 0 ? 0 : nodeSize(node2))) {
        let parent = node2.parentNode;
        if (!parent || parent.nodeType != 1 || hasBlockDesc(node2) || atomElements.test(node2.nodeName) || node2.contentEditable == "false")
          return false;
        off = domIndex(node2) + (dir < 0 ? 0 : 1);
        node2 = parent;
      } else if (node2.nodeType == 1) {
        let child = node2.childNodes[off + (dir < 0 ? -1 : 0)];
        if (child.nodeType == 1 && child.contentEditable == "false") {
          if ((_a = child.pmViewDesc) === null || _a === void 0 ? void 0 : _a.ignoreForSelection)
            off += dir;
          else
            return false;
        } else {
          node2 = child;
          off = dir < 0 ? nodeSize(node2) : 0;
        }
      } else {
        return false;
      }
    }
  }
  function nodeSize(node2) {
    return node2.nodeType == 3 ? node2.nodeValue.length : node2.childNodes.length;
  }
  function textNodeBefore$1(node2, offset) {
    for (; ; ) {
      if (node2.nodeType == 3 && offset)
        return node2;
      if (node2.nodeType == 1 && offset > 0) {
        if (node2.contentEditable == "false")
          return null;
        node2 = node2.childNodes[offset - 1];
        offset = nodeSize(node2);
      } else if (node2.parentNode && !hasBlockDesc(node2)) {
        offset = domIndex(node2);
        node2 = node2.parentNode;
      } else {
        return null;
      }
    }
  }
  function textNodeAfter$1(node2, offset) {
    for (; ; ) {
      if (node2.nodeType == 3 && offset < node2.nodeValue.length)
        return node2;
      if (node2.nodeType == 1 && offset < node2.childNodes.length) {
        if (node2.contentEditable == "false")
          return null;
        node2 = node2.childNodes[offset];
        offset = 0;
      } else if (node2.parentNode && !hasBlockDesc(node2)) {
        offset = domIndex(node2) + 1;
        node2 = node2.parentNode;
      } else {
        return null;
      }
    }
  }
  function isOnEdge(node2, offset, parent) {
    for (let atStart = offset == 0, atEnd = offset == nodeSize(node2); atStart || atEnd; ) {
      if (node2 == parent)
        return true;
      let index2 = domIndex(node2);
      node2 = node2.parentNode;
      if (!node2)
        return false;
      atStart = atStart && index2 == 0;
      atEnd = atEnd && index2 == nodeSize(node2);
    }
  }
  function hasBlockDesc(dom) {
    let desc;
    for (let cur = dom; cur; cur = cur.parentNode)
      if (desc = cur.pmViewDesc)
        break;
    return desc && desc.node && desc.node.isBlock && (desc.dom == dom || desc.contentDOM == dom);
  }
  var selectionCollapsed = function(domSel) {
    return domSel.focusNode && isEquivalentPosition(domSel.focusNode, domSel.focusOffset, domSel.anchorNode, domSel.anchorOffset);
  };
  function keyEvent(keyCode, key3) {
    let event = document.createEvent("Event");
    event.initEvent("keydown", true, true);
    event.keyCode = keyCode;
    event.key = event.code = key3;
    return event;
  }
  function deepActiveElement(doc4) {
    let elt = doc4.activeElement;
    while (elt && elt.shadowRoot)
      elt = elt.shadowRoot.activeElement;
    return elt;
  }
  function caretFromPoint(doc4, x, y) {
    if (doc4.caretPositionFromPoint) {
      try {
        let pos = doc4.caretPositionFromPoint(x, y);
        if (pos)
          return { node: pos.offsetNode, offset: Math.min(nodeSize(pos.offsetNode), pos.offset) };
      } catch (_) {
      }
    }
    if (doc4.caretRangeFromPoint) {
      let range = doc4.caretRangeFromPoint(x, y);
      if (range)
        return { node: range.startContainer, offset: Math.min(nodeSize(range.startContainer), range.startOffset) };
    }
  }
  var nav2 = typeof navigator != "undefined" ? navigator : null;
  var doc3 = typeof document != "undefined" ? document : null;
  var agent2 = nav2 && nav2.userAgent || "";
  var ie_edge2 = /Edge\/(\d+)/.exec(agent2);
  var ie_upto102 = /MSIE \d/.exec(agent2);
  var ie_11up2 = /Trident\/(?:[7-9]|\d{2,})\..*rv:(\d+)/.exec(agent2);
  var ie3 = !!(ie_upto102 || ie_11up2 || ie_edge2);
  var ie_version2 = ie_upto102 ? document.documentMode : ie_11up2 ? +ie_11up2[1] : ie_edge2 ? +ie_edge2[1] : 0;
  var gecko2 = !ie3 && /gecko\/(\d+)/i.test(agent2);
  gecko2 && +(/Firefox\/(\d+)/.exec(agent2) || [0, 0])[1];
  var _chrome2 = !ie3 && /Chrome\/(\d+)/.exec(agent2);
  var chrome = !!_chrome2;
  var chrome_version2 = _chrome2 ? +_chrome2[1] : 0;
  var safari2 = !ie3 && !!nav2 && /Apple Computer/.test(nav2.vendor);
  var ios2 = safari2 && (/Mobile\/\w+/.test(agent2) || !!nav2 && nav2.maxTouchPoints > 2);
  var mac5 = ios2 || (nav2 ? /Mac/.test(nav2.platform) : false);
  var windows2 = nav2 ? /Win/.test(nav2.platform) : false;
  var android2 = /Android \d/.test(agent2);
  var webkit2 = !!doc3 && "webkitFontSmoothing" in doc3.documentElement.style;
  var webkit_version2 = webkit2 ? +(/\bAppleWebKit\/(\d+)/.exec(navigator.userAgent) || [0, 0])[1] : 0;
  function windowRect(doc4) {
    let vp = doc4.defaultView && doc4.defaultView.visualViewport;
    if (vp)
      return {
        left: 0,
        right: vp.width,
        top: 0,
        bottom: vp.height
      };
    return {
      left: 0,
      right: doc4.documentElement.clientWidth,
      top: 0,
      bottom: doc4.documentElement.clientHeight
    };
  }
  function getSide(value, side) {
    return typeof value == "number" ? value : value[side];
  }
  function clientRect(node2) {
    let rect = node2.getBoundingClientRect();
    let scaleX = rect.width / node2.offsetWidth || 1;
    let scaleY = rect.height / node2.offsetHeight || 1;
    return {
      left: rect.left,
      right: rect.left + node2.clientWidth * scaleX,
      top: rect.top,
      bottom: rect.top + node2.clientHeight * scaleY
    };
  }
  function scrollRectIntoView(view, rect, startDOM) {
    let scrollThreshold = view.someProp("scrollThreshold") || 0, scrollMargin = view.someProp("scrollMargin") || 5;
    let doc4 = view.dom.ownerDocument;
    for (let parent = startDOM || view.dom; ; ) {
      if (!parent)
        break;
      if (parent.nodeType != 1) {
        parent = parentNode(parent);
        continue;
      }
      let elt = parent;
      let atTop = elt == doc4.body;
      let bounding = atTop ? windowRect(doc4) : clientRect(elt);
      let moveX = 0, moveY = 0;
      if (rect.top < bounding.top + getSide(scrollThreshold, "top"))
        moveY = -(bounding.top - rect.top + getSide(scrollMargin, "top"));
      else if (rect.bottom > bounding.bottom - getSide(scrollThreshold, "bottom"))
        moveY = rect.bottom - rect.top > bounding.bottom - bounding.top ? rect.top + getSide(scrollMargin, "top") - bounding.top : rect.bottom - bounding.bottom + getSide(scrollMargin, "bottom");
      if (rect.left < bounding.left + getSide(scrollThreshold, "left"))
        moveX = -(bounding.left - rect.left + getSide(scrollMargin, "left"));
      else if (rect.right > bounding.right - getSide(scrollThreshold, "right"))
        moveX = rect.right - bounding.right + getSide(scrollMargin, "right");
      if (moveX || moveY) {
        if (atTop) {
          doc4.defaultView.scrollBy(moveX, moveY);
        } else {
          let startX = elt.scrollLeft, startY = elt.scrollTop;
          if (moveY)
            elt.scrollTop += moveY;
          if (moveX)
            elt.scrollLeft += moveX;
          let dX = elt.scrollLeft - startX, dY = elt.scrollTop - startY;
          rect = { left: rect.left - dX, top: rect.top - dY, right: rect.right - dX, bottom: rect.bottom - dY };
        }
      }
      let pos = atTop ? "fixed" : getComputedStyle(parent).position;
      if (/^(fixed|sticky)$/.test(pos))
        break;
      parent = pos == "absolute" ? parent.offsetParent : parentNode(parent);
    }
  }
  function storeScrollPos(view) {
    let rect = view.dom.getBoundingClientRect(), startY = Math.max(0, rect.top);
    let refDOM, refTop;
    for (let x = (rect.left + rect.right) / 2, y = startY + 1; y < Math.min(innerHeight, rect.bottom); y += 5) {
      let dom = view.root.elementFromPoint(x, y);
      if (!dom || dom == view.dom || !view.dom.contains(dom))
        continue;
      let localRect = dom.getBoundingClientRect();
      if (localRect.top >= startY - 20) {
        refDOM = dom;
        refTop = localRect.top;
        break;
      }
    }
    return { refDOM, refTop, stack: scrollStack(view.dom) };
  }
  function scrollStack(dom) {
    let stack = [], doc4 = dom.ownerDocument;
    for (let cur = dom; cur; cur = parentNode(cur)) {
      stack.push({ dom: cur, top: cur.scrollTop, left: cur.scrollLeft });
      if (dom == doc4)
        break;
    }
    return stack;
  }
  function resetScrollPos({ refDOM, refTop, stack }) {
    let newRefTop = refDOM ? refDOM.getBoundingClientRect().top : 0;
    restoreScrollStack(stack, newRefTop == 0 ? 0 : newRefTop - refTop);
  }
  function restoreScrollStack(stack, dTop) {
    for (let i2 = 0; i2 < stack.length; i2++) {
      let { dom, top, left } = stack[i2];
      if (dom.scrollTop != top + dTop)
        dom.scrollTop = top + dTop;
      if (dom.scrollLeft != left)
        dom.scrollLeft = left;
    }
  }
  var preventScrollSupported = null;
  function focusPreventScroll(dom) {
    if (dom.setActive)
      return dom.setActive();
    if (preventScrollSupported)
      return dom.focus(preventScrollSupported);
    let stored = scrollStack(dom);
    dom.focus(preventScrollSupported == null ? {
      get preventScroll() {
        preventScrollSupported = { preventScroll: true };
        return true;
      }
    } : void 0);
    if (!preventScrollSupported) {
      preventScrollSupported = false;
      restoreScrollStack(stored, 0);
    }
  }
  function findOffsetInNode(node2, coords) {
    let closest, dxClosest = 2e8, coordsClosest, offset = 0;
    let rowBot = coords.top, rowTop = coords.top;
    let firstBelow, coordsBelow;
    for (let child = node2.firstChild, childIndex = 0; child; child = child.nextSibling, childIndex++) {
      let rects;
      if (child.nodeType == 1)
        rects = child.getClientRects();
      else if (child.nodeType == 3)
        rects = textRange(child).getClientRects();
      else
        continue;
      for (let i2 = 0; i2 < rects.length; i2++) {
        let rect = rects[i2];
        if (rect.top <= rowBot && rect.bottom >= rowTop) {
          rowBot = Math.max(rect.bottom, rowBot);
          rowTop = Math.min(rect.top, rowTop);
          let dx = rect.left > coords.left ? rect.left - coords.left : rect.right < coords.left ? coords.left - rect.right : 0;
          if (dx < dxClosest) {
            closest = child;
            dxClosest = dx;
            coordsClosest = dx && closest.nodeType == 3 ? {
              left: rect.right < coords.left ? rect.right : rect.left,
              top: coords.top
            } : coords;
            if (child.nodeType == 1 && dx)
              offset = childIndex + (coords.left >= (rect.left + rect.right) / 2 ? 1 : 0);
            continue;
          }
        } else if (rect.top > coords.top && !firstBelow && rect.left <= coords.left && rect.right >= coords.left) {
          firstBelow = child;
          coordsBelow = { left: Math.max(rect.left, Math.min(rect.right, coords.left)), top: rect.top };
        }
        if (!closest && (coords.left >= rect.right && coords.top >= rect.top || coords.left >= rect.left && coords.top >= rect.bottom))
          offset = childIndex + 1;
      }
    }
    if (!closest && firstBelow) {
      closest = firstBelow;
      coordsClosest = coordsBelow;
      dxClosest = 0;
    }
    if (closest && closest.nodeType == 3)
      return findOffsetInText(closest, coordsClosest);
    if (!closest || dxClosest && closest.nodeType == 1)
      return { node: node2, offset };
    return findOffsetInNode(closest, coordsClosest);
  }
  function findOffsetInText(node2, coords) {
    let len = node2.nodeValue.length;
    let range = document.createRange(), result;
    for (let i2 = 0; i2 < len; i2++) {
      range.setEnd(node2, i2 + 1);
      range.setStart(node2, i2);
      let rect = singleRect(range, 1);
      if (rect.top == rect.bottom)
        continue;
      if (inRect(coords, rect)) {
        result = { node: node2, offset: i2 + (coords.left >= (rect.left + rect.right) / 2 ? 1 : 0) };
        break;
      }
    }
    range.detach();
    return result || { node: node2, offset: 0 };
  }
  function inRect(coords, rect) {
    return coords.left >= rect.left - 1 && coords.left <= rect.right + 1 && coords.top >= rect.top - 1 && coords.top <= rect.bottom + 1;
  }
  function targetKludge(dom, coords) {
    let parent = dom.parentNode;
    if (parent && /^li$/i.test(parent.nodeName) && coords.left < dom.getBoundingClientRect().left)
      return parent;
    return dom;
  }
  function posFromElement(view, elt, coords) {
    let { node: node2, offset } = findOffsetInNode(elt, coords), bias = -1;
    if (node2.nodeType == 1 && !node2.firstChild) {
      let rect = node2.getBoundingClientRect();
      bias = rect.left != rect.right && coords.left > (rect.left + rect.right) / 2 ? 1 : -1;
    }
    return view.docView.posFromDOM(node2, offset, bias);
  }
  function posFromCaret(view, node2, offset, coords) {
    let outsideBlock = -1;
    for (let cur = node2, sawBlock = false; ; ) {
      if (cur == view.dom)
        break;
      let desc = view.docView.nearestDesc(cur, true), rect;
      if (!desc)
        return null;
      if (desc.dom.nodeType == 1 && (desc.node.isBlock && desc.parent || !desc.contentDOM) && // Ignore elements with zero-size bounding rectangles
      ((rect = desc.dom.getBoundingClientRect()).width || rect.height)) {
        if (desc.node.isBlock && desc.parent && !/^T(R|BODY|HEAD|FOOT)$/.test(desc.dom.nodeName)) {
          if (!sawBlock && rect.left > coords.left || rect.top > coords.top)
            outsideBlock = desc.posBefore;
          else if (!sawBlock && rect.right < coords.left || rect.bottom < coords.top)
            outsideBlock = desc.posAfter;
          sawBlock = true;
        }
        if (!desc.contentDOM && outsideBlock < 0 && !desc.node.isText) {
          let before = desc.node.isBlock ? coords.top < (rect.top + rect.bottom) / 2 : coords.left < (rect.left + rect.right) / 2;
          return before ? desc.posBefore : desc.posAfter;
        }
      }
      cur = desc.dom.parentNode;
    }
    return outsideBlock > -1 ? outsideBlock : view.docView.posFromDOM(node2, offset, -1);
  }
  function elementFromPoint(element2, coords, box) {
    let len = element2.childNodes.length;
    if (len && box.top < box.bottom) {
      for (let startI = Math.max(0, Math.min(len - 1, Math.floor(len * (coords.top - box.top) / (box.bottom - box.top)) - 2)), i2 = startI; ; ) {
        let child = element2.childNodes[i2];
        if (child.nodeType == 1) {
          let rects = child.getClientRects();
          for (let j = 0; j < rects.length; j++) {
            let rect = rects[j];
            if (inRect(coords, rect))
              return elementFromPoint(child, coords, rect);
          }
        }
        if ((i2 = (i2 + 1) % len) == startI)
          break;
      }
    }
    return element2;
  }
  function posAtCoords(view, coords) {
    let doc4 = view.dom.ownerDocument, node2, offset = 0;
    let caret = caretFromPoint(doc4, coords.left, coords.top);
    if (caret)
      ({ node: node2, offset } = caret);
    let elt = (view.root.elementFromPoint ? view.root : doc4).elementFromPoint(coords.left, coords.top);
    let pos;
    if (!elt || !view.dom.contains(elt.nodeType != 1 ? elt.parentNode : elt)) {
      let box = view.dom.getBoundingClientRect();
      if (!inRect(coords, box))
        return null;
      elt = elementFromPoint(view.dom, coords, box);
      if (!elt)
        return null;
    }
    if (safari2) {
      for (let p2 = elt; node2 && p2; p2 = parentNode(p2))
        if (p2.draggable)
          node2 = void 0;
    }
    elt = targetKludge(elt, coords);
    if (node2) {
      if (gecko2 && node2.nodeType == 1) {
        offset = Math.min(offset, node2.childNodes.length);
        if (offset < node2.childNodes.length) {
          let next = node2.childNodes[offset], box;
          if (next.nodeName == "IMG" && (box = next.getBoundingClientRect()).right <= coords.left && box.bottom > coords.top)
            offset++;
        }
      }
      let prev;
      if (webkit2 && offset && node2.nodeType == 1 && (prev = node2.childNodes[offset - 1]).nodeType == 1 && prev.contentEditable == "false" && prev.getBoundingClientRect().top >= coords.top)
        offset--;
      if (node2 == view.dom && offset == node2.childNodes.length - 1 && node2.lastChild.nodeType == 1 && coords.top > node2.lastChild.getBoundingClientRect().bottom)
        pos = view.state.doc.content.size;
      else if (offset == 0 || node2.nodeType != 1 || node2.childNodes[offset - 1].nodeName != "BR")
        pos = posFromCaret(view, node2, offset, coords);
    }
    if (pos == null)
      pos = posFromElement(view, elt, coords);
    let desc = view.docView.nearestDesc(elt, true);
    return { pos, inside: desc ? desc.posAtStart - desc.border : -1 };
  }
  function nonZero(rect) {
    return rect.top < rect.bottom || rect.left < rect.right;
  }
  function singleRect(target, bias) {
    let rects = target.getClientRects();
    if (rects.length) {
      let first = rects[bias < 0 ? 0 : rects.length - 1];
      if (nonZero(first))
        return first;
    }
    return Array.prototype.find.call(rects, nonZero) || target.getBoundingClientRect();
  }
  var BIDI = /[\u0590-\u05f4\u0600-\u06ff\u0700-\u08ac]/;
  function coordsAtPos(view, pos, side) {
    let { node: node2, offset, atom } = view.docView.domFromPos(pos, side < 0 ? -1 : 1);
    let supportEmptyRange = webkit2 || gecko2;
    if (node2.nodeType == 3) {
      if (supportEmptyRange && (BIDI.test(node2.nodeValue) || (side < 0 ? !offset : offset == node2.nodeValue.length))) {
        let rect = singleRect(textRange(node2, offset, offset), side);
        if (gecko2 && offset && /\s/.test(node2.nodeValue[offset - 1]) && offset < node2.nodeValue.length) {
          let rectBefore = singleRect(textRange(node2, offset - 1, offset - 1), -1);
          if (rectBefore.top == rect.top) {
            let rectAfter = singleRect(textRange(node2, offset, offset + 1), -1);
            if (rectAfter.top != rect.top)
              return flattenV(rectAfter, rectAfter.left < rectBefore.left);
          }
        }
        return rect;
      } else {
        let from = offset, to = offset, takeSide = side < 0 ? 1 : -1;
        if (side < 0 && !offset) {
          to++;
          takeSide = -1;
        } else if (side >= 0 && offset == node2.nodeValue.length) {
          from--;
          takeSide = 1;
        } else if (side < 0) {
          from--;
        } else {
          to++;
        }
        return flattenV(singleRect(textRange(node2, from, to), takeSide), takeSide < 0);
      }
    }
    let $dom = view.state.doc.resolve(pos - (atom || 0));
    if (!$dom.parent.inlineContent) {
      if (atom == null && offset && (side < 0 || offset == nodeSize(node2))) {
        let before = node2.childNodes[offset - 1];
        if (before.nodeType == 1)
          return flattenH(before.getBoundingClientRect(), false);
      }
      if (atom == null && offset < nodeSize(node2)) {
        let after = node2.childNodes[offset];
        if (after.nodeType == 1)
          return flattenH(after.getBoundingClientRect(), true);
      }
      return flattenH(node2.getBoundingClientRect(), side >= 0);
    }
    if (atom == null && offset && (side < 0 || offset == nodeSize(node2))) {
      let before = node2.childNodes[offset - 1];
      let target = before.nodeType == 3 ? textRange(before, nodeSize(before) - (supportEmptyRange ? 0 : 1)) : before.nodeType == 1 && (before.nodeName != "BR" || !before.nextSibling) ? before : null;
      if (target)
        return flattenV(singleRect(target, 1), false);
    }
    if (atom == null && offset < nodeSize(node2)) {
      let after = node2.childNodes[offset];
      while (after.pmViewDesc && after.pmViewDesc.ignoreForCoords)
        after = after.nextSibling;
      let target = !after ? null : after.nodeType == 3 ? textRange(after, 0, supportEmptyRange ? 0 : 1) : after.nodeType == 1 ? after : null;
      if (target)
        return flattenV(singleRect(target, -1), true);
    }
    return flattenV(singleRect(node2.nodeType == 3 ? textRange(node2) : node2, -side), side >= 0);
  }
  function flattenV(rect, left) {
    if (rect.width == 0)
      return rect;
    let x = left ? rect.left : rect.right;
    return { top: rect.top, bottom: rect.bottom, left: x, right: x };
  }
  function flattenH(rect, top) {
    if (rect.height == 0)
      return rect;
    let y = top ? rect.top : rect.bottom;
    return { top: y, bottom: y, left: rect.left, right: rect.right };
  }
  function withFlushedState(view, state, f2) {
    let viewState = view.state, active = view.root.activeElement;
    if (viewState != state)
      view.updateState(state);
    if (active != view.dom)
      view.focus();
    try {
      return f2();
    } finally {
      if (viewState != state)
        view.updateState(viewState);
      if (active != view.dom && active)
        active.focus();
    }
  }
  function endOfTextblockVertical(view, state, dir) {
    let sel = state.selection;
    let $pos = dir == "up" ? sel.$from : sel.$to;
    return withFlushedState(view, state, () => {
      let { node: dom } = view.docView.domFromPos($pos.pos, dir == "up" ? -1 : 1);
      for (; ; ) {
        let nearest = view.docView.nearestDesc(dom, true);
        if (!nearest)
          break;
        if (nearest.node.isBlock) {
          dom = nearest.contentDOM || nearest.dom;
          break;
        }
        dom = nearest.dom.parentNode;
      }
      let coords = coordsAtPos(view, $pos.pos, 1);
      for (let child = dom.firstChild; child; child = child.nextSibling) {
        let boxes;
        if (child.nodeType == 1)
          boxes = child.getClientRects();
        else if (child.nodeType == 3)
          boxes = textRange(child, 0, child.nodeValue.length).getClientRects();
        else
          continue;
        for (let i2 = 0; i2 < boxes.length; i2++) {
          let box = boxes[i2];
          if (box.bottom > box.top + 1 && (dir == "up" ? coords.top - box.top > (box.bottom - coords.top) * 2 : box.bottom - coords.bottom > (coords.bottom - box.top) * 2))
            return false;
        }
      }
      return true;
    });
  }
  var maybeRTL = /[\u0590-\u08ac]/;
  function endOfTextblockHorizontal(view, state, dir) {
    let { $head } = state.selection;
    if (!$head.parent.isTextblock)
      return false;
    let offset = $head.parentOffset, atStart = !offset, atEnd = offset == $head.parent.content.size;
    let sel = view.domSelection();
    if (!sel)
      return $head.pos == $head.start() || $head.pos == $head.end();
    if (!maybeRTL.test($head.parent.textContent) || !sel.modify)
      return dir == "left" || dir == "backward" ? atStart : atEnd;
    return withFlushedState(view, state, () => {
      let { focusNode: oldNode, focusOffset: oldOff, anchorNode, anchorOffset } = view.domSelectionRange();
      let oldBidiLevel = sel.caretBidiLevel;
      sel.modify("move", dir, "character");
      let parentDOM = $head.depth ? view.docView.domAfterPos($head.before()) : view.dom;
      let { focusNode: newNode, focusOffset: newOff } = view.domSelectionRange();
      let result = newNode && !parentDOM.contains(newNode.nodeType == 1 ? newNode : newNode.parentNode) || oldNode == newNode && oldOff == newOff;
      try {
        sel.collapse(anchorNode, anchorOffset);
        if (oldNode && (oldNode != anchorNode || oldOff != anchorOffset) && sel.extend)
          sel.extend(oldNode, oldOff);
      } catch (_) {
      }
      if (oldBidiLevel != null)
        sel.caretBidiLevel = oldBidiLevel;
      return result;
    });
  }
  var cachedState = null;
  var cachedDir = null;
  var cachedResult = false;
  function endOfTextblock(view, state, dir) {
    if (cachedState == state && cachedDir == dir)
      return cachedResult;
    cachedState = state;
    cachedDir = dir;
    return cachedResult = dir == "up" || dir == "down" ? endOfTextblockVertical(view, state, dir) : endOfTextblockHorizontal(view, state, dir);
  }
  var NOT_DIRTY = 0;
  var CHILD_DIRTY = 1;
  var CONTENT_DIRTY = 2;
  var NODE_DIRTY = 3;
  var ViewDesc = class {
    constructor(parent, children, dom, contentDOM) {
      this.parent = parent;
      this.children = children;
      this.dom = dom;
      this.contentDOM = contentDOM;
      this.dirty = NOT_DIRTY;
      dom.pmViewDesc = this;
    }
    // Used to check whether a given description corresponds to a
    // widget/mark/node.
    matchesWidget(widget) {
      return false;
    }
    matchesMark(mark) {
      return false;
    }
    matchesNode(node2, outerDeco, innerDeco) {
      return false;
    }
    matchesHack(nodeName) {
      return false;
    }
    // When parsing in-editor content (in domchange.js), we allow
    // descriptions to determine the parse rules that should be used to
    // parse them.
    parseRule() {
      return null;
    }
    // Used by the editor's event handler to ignore events that come
    // from certain descs.
    stopEvent(event) {
      return false;
    }
    // The size of the content represented by this desc.
    get size() {
      let size = 0;
      for (let i2 = 0; i2 < this.children.length; i2++)
        size += this.children[i2].size;
      return size;
    }
    // For block nodes, this represents the space taken up by their
    // start/end tokens.
    get border() {
      return 0;
    }
    destroy() {
      this.parent = void 0;
      if (this.dom.pmViewDesc == this)
        this.dom.pmViewDesc = void 0;
      for (let i2 = 0; i2 < this.children.length; i2++)
        this.children[i2].destroy();
    }
    posBeforeChild(child) {
      for (let i2 = 0, pos = this.posAtStart; ; i2++) {
        let cur = this.children[i2];
        if (cur == child)
          return pos;
        pos += cur.size;
      }
    }
    get posBefore() {
      return this.parent.posBeforeChild(this);
    }
    get posAtStart() {
      return this.parent ? this.parent.posBeforeChild(this) + this.border : 0;
    }
    get posAfter() {
      return this.posBefore + this.size;
    }
    get posAtEnd() {
      return this.posAtStart + this.size - 2 * this.border;
    }
    localPosFromDOM(dom, offset, bias) {
      if (this.contentDOM && this.contentDOM.contains(dom.nodeType == 1 ? dom : dom.parentNode)) {
        if (bias < 0) {
          let domBefore, desc;
          if (dom == this.contentDOM) {
            domBefore = dom.childNodes[offset - 1];
          } else {
            while (dom.parentNode != this.contentDOM)
              dom = dom.parentNode;
            domBefore = dom.previousSibling;
          }
          while (domBefore && !((desc = domBefore.pmViewDesc) && desc.parent == this))
            domBefore = domBefore.previousSibling;
          return domBefore ? this.posBeforeChild(desc) + desc.size : this.posAtStart;
        } else {
          let domAfter, desc;
          if (dom == this.contentDOM) {
            domAfter = dom.childNodes[offset];
          } else {
            while (dom.parentNode != this.contentDOM)
              dom = dom.parentNode;
            domAfter = dom.nextSibling;
          }
          while (domAfter && !((desc = domAfter.pmViewDesc) && desc.parent == this))
            domAfter = domAfter.nextSibling;
          return domAfter ? this.posBeforeChild(desc) : this.posAtEnd;
        }
      }
      let atEnd;
      if (dom == this.dom && this.contentDOM) {
        atEnd = offset > domIndex(this.contentDOM);
      } else if (this.contentDOM && this.contentDOM != this.dom && this.dom.contains(this.contentDOM)) {
        atEnd = dom.compareDocumentPosition(this.contentDOM) & 2;
      } else if (this.dom.firstChild) {
        if (offset == 0)
          for (let search2 = dom; ; search2 = search2.parentNode) {
            if (search2 == this.dom) {
              atEnd = false;
              break;
            }
            if (search2.previousSibling)
              break;
          }
        if (atEnd == null && offset == dom.childNodes.length)
          for (let search2 = dom; ; search2 = search2.parentNode) {
            if (search2 == this.dom) {
              atEnd = true;
              break;
            }
            if (search2.nextSibling)
              break;
          }
      }
      return (atEnd == null ? bias > 0 : atEnd) ? this.posAtEnd : this.posAtStart;
    }
    nearestDesc(dom, onlyNodes = false) {
      for (let first = true, cur = dom; cur; cur = cur.parentNode) {
        let desc = this.getDesc(cur), nodeDOM;
        if (desc && (!onlyNodes || desc.node)) {
          if (first && (nodeDOM = desc.nodeDOM) && !(nodeDOM.nodeType == 1 ? nodeDOM.contains(dom.nodeType == 1 ? dom : dom.parentNode) : nodeDOM == dom))
            first = false;
          else
            return desc;
        }
      }
    }
    getDesc(dom) {
      let desc = dom.pmViewDesc;
      for (let cur = desc; cur; cur = cur.parent)
        if (cur == this)
          return desc;
    }
    posFromDOM(dom, offset, bias) {
      for (let scan = dom; scan; scan = scan.parentNode) {
        let desc = this.getDesc(scan);
        if (desc)
          return desc.localPosFromDOM(dom, offset, bias);
      }
      return -1;
    }
    // Find the desc for the node after the given pos, if any. (When a
    // parent node overrode rendering, there might not be one.)
    descAt(pos) {
      for (let i2 = 0, offset = 0; i2 < this.children.length; i2++) {
        let child = this.children[i2], end = offset + child.size;
        if (offset == pos && end != offset) {
          while (!child.border && child.children.length) {
            for (let i3 = 0; i3 < child.children.length; i3++) {
              let inner = child.children[i3];
              if (inner.size) {
                child = inner;
                break;
              }
            }
          }
          return child;
        }
        if (pos < end)
          return child.descAt(pos - offset - child.border);
        offset = end;
      }
    }
    domFromPos(pos, side) {
      if (!this.contentDOM)
        return { node: this.dom, offset: 0, atom: pos + 1 };
      let i2 = 0, offset = 0;
      for (let curPos = 0; i2 < this.children.length; i2++) {
        let child = this.children[i2], end = curPos + child.size;
        if (end > pos || child instanceof TrailingHackViewDesc) {
          offset = pos - curPos;
          break;
        }
        curPos = end;
      }
      if (offset)
        return this.children[i2].domFromPos(offset - this.children[i2].border, side);
      for (let prev; i2 && !(prev = this.children[i2 - 1]).size && prev instanceof WidgetViewDesc && prev.side >= 0; i2--) {
      }
      if (side <= 0) {
        let prev, enter = true;
        for (; ; i2--, enter = false) {
          prev = i2 ? this.children[i2 - 1] : null;
          if (!prev || prev.dom.parentNode == this.contentDOM)
            break;
        }
        if (prev && side && enter && !prev.border && !prev.domAtom)
          return prev.domFromPos(prev.size, side);
        return { node: this.contentDOM, offset: prev ? domIndex(prev.dom) + 1 : 0 };
      } else {
        let next, enter = true;
        for (; ; i2++, enter = false) {
          next = i2 < this.children.length ? this.children[i2] : null;
          if (!next || next.dom.parentNode == this.contentDOM)
            break;
        }
        if (next && enter && !next.border && !next.domAtom)
          return next.domFromPos(0, side);
        return { node: this.contentDOM, offset: next ? domIndex(next.dom) : this.contentDOM.childNodes.length };
      }
    }
    // Used to find a DOM range in a single parent for a given changed
    // range.
    parseRange(from, to, base2 = 0) {
      if (this.children.length == 0)
        return { node: this.contentDOM, from, to, fromOffset: 0, toOffset: this.contentDOM.childNodes.length };
      let fromOffset = -1, toOffset = -1;
      for (let offset = base2, i2 = 0; ; i2++) {
        let child = this.children[i2], end = offset + child.size;
        if (fromOffset == -1 && from <= end) {
          let childBase = offset + child.border;
          if (from >= childBase && to <= end - child.border && child.node && child.contentDOM && this.contentDOM.contains(child.contentDOM))
            return child.parseRange(from, to, childBase);
          from = offset;
          for (let j = i2; j > 0; j--) {
            let prev = this.children[j - 1];
            if (prev.size && prev.dom.parentNode == this.contentDOM && !prev.emptyChildAt(1)) {
              fromOffset = domIndex(prev.dom) + 1;
              break;
            }
            from -= prev.size;
          }
          if (fromOffset == -1)
            fromOffset = 0;
        }
        if (fromOffset > -1 && (end > to || i2 == this.children.length - 1)) {
          to = end;
          for (let j = i2 + 1; j < this.children.length; j++) {
            let next = this.children[j];
            if (next.size && next.dom.parentNode == this.contentDOM && !next.emptyChildAt(-1)) {
              toOffset = domIndex(next.dom);
              break;
            }
            to += next.size;
          }
          if (toOffset == -1)
            toOffset = this.contentDOM.childNodes.length;
          break;
        }
        offset = end;
      }
      return { node: this.contentDOM, from, to, fromOffset, toOffset };
    }
    emptyChildAt(side) {
      if (this.border || !this.contentDOM || !this.children.length)
        return false;
      let child = this.children[side < 0 ? 0 : this.children.length - 1];
      return child.size == 0 || child.emptyChildAt(side);
    }
    domAfterPos(pos) {
      let { node: node2, offset } = this.domFromPos(pos, 0);
      if (node2.nodeType != 1 || offset == node2.childNodes.length)
        throw new RangeError("No node after pos " + pos);
      return node2.childNodes[offset];
    }
    // View descs are responsible for setting any selection that falls
    // entirely inside of them, so that custom implementations can do
    // custom things with the selection. Note that this falls apart when
    // a selection starts in such a node and ends in another, in which
    // case we just use whatever domFromPos produces as a best effort.
    setSelection(anchor, head, view, force = false) {
      let from = Math.min(anchor, head), to = Math.max(anchor, head);
      for (let i2 = 0, offset = 0; i2 < this.children.length; i2++) {
        let child = this.children[i2], end = offset + child.size;
        if (from > offset && to < end)
          return child.setSelection(anchor - offset - child.border, head - offset - child.border, view, force);
        offset = end;
      }
      let anchorDOM = this.domFromPos(anchor, anchor ? -1 : 1);
      let headDOM = head == anchor ? anchorDOM : this.domFromPos(head, head ? -1 : 1);
      let domSel = view.root.getSelection();
      let selRange = view.domSelectionRange();
      let brKludge = false;
      if ((gecko2 || safari2) && anchor == head) {
        let { node: node2, offset } = anchorDOM;
        if (node2.nodeType == 3) {
          brKludge = !!(offset && node2.nodeValue[offset - 1] == "\n");
          if (brKludge && offset == node2.nodeValue.length) {
            for (let scan = node2, after; scan; scan = scan.parentNode) {
              if (after = scan.nextSibling) {
                if (after.nodeName == "BR")
                  anchorDOM = headDOM = { node: after.parentNode, offset: domIndex(after) + 1 };
                break;
              }
              let desc = scan.pmViewDesc;
              if (desc && desc.node && desc.node.isBlock)
                break;
            }
          }
        } else {
          let prev = node2.childNodes[offset - 1];
          brKludge = prev && (prev.nodeName == "BR" || prev.contentEditable == "false");
        }
      }
      if (gecko2 && selRange.focusNode && selRange.focusNode != headDOM.node && selRange.focusNode.nodeType == 1) {
        let after = selRange.focusNode.childNodes[selRange.focusOffset];
        if (after && after.contentEditable == "false")
          force = true;
      }
      if (!(force || brKludge && safari2) && isEquivalentPosition(anchorDOM.node, anchorDOM.offset, selRange.anchorNode, selRange.anchorOffset) && isEquivalentPosition(headDOM.node, headDOM.offset, selRange.focusNode, selRange.focusOffset))
        return;
      let domSelExtended = false;
      if ((domSel.extend || anchor == head) && !(brKludge && gecko2)) {
        domSel.collapse(anchorDOM.node, anchorDOM.offset);
        try {
          if (anchor != head)
            domSel.extend(headDOM.node, headDOM.offset);
          domSelExtended = true;
        } catch (_) {
        }
      }
      if (!domSelExtended) {
        if (anchor > head) {
          let tmp = anchorDOM;
          anchorDOM = headDOM;
          headDOM = tmp;
        }
        let range = document.createRange();
        range.setEnd(headDOM.node, headDOM.offset);
        range.setStart(anchorDOM.node, anchorDOM.offset);
        domSel.removeAllRanges();
        domSel.addRange(range);
      }
    }
    ignoreMutation(mutation) {
      return !this.contentDOM && mutation.type != "selection";
    }
    get contentLost() {
      return this.contentDOM && this.contentDOM != this.dom && !this.dom.contains(this.contentDOM);
    }
    // Remove a subtree of the element tree that has been touched
    // by a DOM change, so that the next update will redraw it.
    markDirty(from, to) {
      for (let offset = 0, i2 = 0; i2 < this.children.length; i2++) {
        let child = this.children[i2], end = offset + child.size;
        if (offset == end ? from <= end && to >= offset : from < end && to > offset) {
          let startInside = offset + child.border, endInside = end - child.border;
          if (from >= startInside && to <= endInside) {
            this.dirty = from == offset || to == end ? CONTENT_DIRTY : CHILD_DIRTY;
            if (from == startInside && to == endInside && (child.contentLost || child.dom.parentNode != this.contentDOM))
              child.dirty = NODE_DIRTY;
            else
              child.markDirty(from - startInside, to - startInside);
            return;
          } else {
            child.dirty = child.dom == child.contentDOM && child.dom.parentNode == this.contentDOM && !child.children.length ? CONTENT_DIRTY : NODE_DIRTY;
          }
        }
        offset = end;
      }
      this.dirty = CONTENT_DIRTY;
    }
    markParentsDirty() {
      let level = 1;
      for (let node2 = this.parent; node2; node2 = node2.parent, level++) {
        let dirty = level == 1 ? CONTENT_DIRTY : CHILD_DIRTY;
        if (node2.dirty < dirty)
          node2.dirty = dirty;
      }
    }
    get domAtom() {
      return false;
    }
    get ignoreForCoords() {
      return false;
    }
    get ignoreForSelection() {
      return false;
    }
    isText(text4) {
      return false;
    }
  };
  var WidgetViewDesc = class extends ViewDesc {
    constructor(parent, widget, view, pos) {
      let self2, dom = widget.type.toDOM;
      if (typeof dom == "function")
        dom = dom(view, () => {
          if (!self2)
            return pos;
          if (self2.parent)
            return self2.parent.posBeforeChild(self2);
        });
      if (!widget.type.spec.raw) {
        if (dom.nodeType != 1) {
          let wrap3 = document.createElement("span");
          wrap3.appendChild(dom);
          dom = wrap3;
        }
        dom.contentEditable = "false";
        dom.classList.add("ProseMirror-widget");
      }
      super(parent, [], dom, null);
      this.widget = widget;
      this.widget = widget;
      self2 = this;
    }
    matchesWidget(widget) {
      return this.dirty == NOT_DIRTY && widget.type.eq(this.widget.type);
    }
    parseRule() {
      return { ignore: true };
    }
    stopEvent(event) {
      let stop = this.widget.spec.stopEvent;
      return stop ? stop(event) : false;
    }
    ignoreMutation(mutation) {
      return mutation.type != "selection" || this.widget.spec.ignoreSelection;
    }
    destroy() {
      this.widget.type.destroy(this.dom);
      super.destroy();
    }
    get domAtom() {
      return true;
    }
    get ignoreForSelection() {
      return !!this.widget.type.spec.relaxedSide;
    }
    get side() {
      return this.widget.type.side;
    }
  };
  var CompositionViewDesc = class extends ViewDesc {
    constructor(parent, dom, textDOM, text4) {
      super(parent, [], dom, null);
      this.textDOM = textDOM;
      this.text = text4;
    }
    get size() {
      return this.text.length;
    }
    localPosFromDOM(dom, offset) {
      if (dom != this.textDOM)
        return this.posAtStart + (offset ? this.size : 0);
      return this.posAtStart + offset;
    }
    domFromPos(pos) {
      return { node: this.textDOM, offset: pos };
    }
    ignoreMutation(mut) {
      return mut.type === "characterData" && mut.target.nodeValue == mut.oldValue;
    }
  };
  var MarkViewDesc = class _MarkViewDesc extends ViewDesc {
    constructor(parent, mark, dom, contentDOM, spec) {
      super(parent, [], dom, contentDOM);
      this.mark = mark;
      this.spec = spec;
    }
    static create(parent, mark, inline, view) {
      let custom = view.nodeViews[mark.type.name];
      let spec = custom && custom(mark, view, inline);
      if (!spec || !spec.dom)
        spec = DOMSerializer.renderSpec(document, mark.type.spec.toDOM(mark, inline), null, mark.attrs);
      return new _MarkViewDesc(parent, mark, spec.dom, spec.contentDOM || spec.dom, spec);
    }
    parseRule() {
      if (this.dirty & NODE_DIRTY || this.mark.type.spec.reparseInView)
        return null;
      return { mark: this.mark.type.name, attrs: this.mark.attrs, contentElement: this.contentDOM };
    }
    matchesMark(mark) {
      return this.dirty != NODE_DIRTY && this.mark.eq(mark);
    }
    markDirty(from, to) {
      super.markDirty(from, to);
      if (this.dirty != NOT_DIRTY) {
        let parent = this.parent;
        while (!parent.node)
          parent = parent.parent;
        if (parent.dirty < this.dirty)
          parent.dirty = this.dirty;
        this.dirty = NOT_DIRTY;
      }
    }
    slice(from, to, view) {
      let copy2 = _MarkViewDesc.create(this.parent, this.mark, true, view);
      let nodes = this.children, size = this.size;
      if (to < size)
        nodes = replaceNodes(nodes, to, size, view);
      if (from > 0)
        nodes = replaceNodes(nodes, 0, from, view);
      for (let i2 = 0; i2 < nodes.length; i2++)
        nodes[i2].parent = copy2;
      copy2.children = nodes;
      return copy2;
    }
    ignoreMutation(mutation) {
      return this.spec.ignoreMutation ? this.spec.ignoreMutation(mutation) : super.ignoreMutation(mutation);
    }
    destroy() {
      if (this.spec.destroy)
        this.spec.destroy();
      super.destroy();
    }
  };
  var NodeViewDesc = class _NodeViewDesc extends ViewDesc {
    constructor(parent, node2, outerDeco, innerDeco, dom, contentDOM, nodeDOM, view, pos) {
      super(parent, [], dom, contentDOM);
      this.node = node2;
      this.outerDeco = outerDeco;
      this.innerDeco = innerDeco;
      this.nodeDOM = nodeDOM;
    }
    // By default, a node is rendered using the `toDOM` method from the
    // node type spec. But client code can use the `nodeViews` spec to
    // supply a custom node view, which can influence various aspects of
    // the way the node works.
    //
    // (Using subclassing for this was intentionally decided against,
    // since it'd require exposing a whole slew of finicky
    // implementation details to the user code that they probably will
    // never need.)
    static create(parent, node2, outerDeco, innerDeco, view, pos) {
      let custom = view.nodeViews[node2.type.name], descObj;
      let spec = custom && custom(node2, view, () => {
        if (!descObj)
          return pos;
        if (descObj.parent)
          return descObj.parent.posBeforeChild(descObj);
      }, outerDeco, innerDeco);
      let dom = spec && spec.dom, contentDOM = spec && spec.contentDOM;
      if (node2.isText) {
        if (!dom)
          dom = document.createTextNode(node2.text);
        else if (dom.nodeType != 3)
          throw new RangeError("Text must be rendered as a DOM text node");
      } else if (!dom) {
        let spec2 = DOMSerializer.renderSpec(document, node2.type.spec.toDOM(node2), null, node2.attrs);
        ({ dom, contentDOM } = spec2);
      }
      if (!contentDOM && !node2.isText && dom.nodeName != "BR") {
        if (!dom.hasAttribute("contenteditable"))
          dom.contentEditable = "false";
        if (node2.type.spec.draggable)
          dom.draggable = true;
      }
      let nodeDOM = dom;
      dom = applyOuterDeco(dom, outerDeco, node2);
      if (spec)
        return descObj = new CustomNodeViewDesc(parent, node2, outerDeco, innerDeco, dom, contentDOM || null, nodeDOM, spec, view, pos + 1);
      else if (node2.isText)
        return new TextViewDesc(parent, node2, outerDeco, innerDeco, dom, nodeDOM, view);
      else
        return new _NodeViewDesc(parent, node2, outerDeco, innerDeco, dom, contentDOM || null, nodeDOM, view, pos + 1);
    }
    parseRule() {
      if (this.node.type.spec.reparseInView)
        return null;
      let rule = { node: this.node.type.name, attrs: this.node.attrs };
      if (this.node.type.whitespace == "pre")
        rule.preserveWhitespace = "full";
      if (!this.contentDOM) {
        rule.getContent = () => this.node.content;
      } else if (!this.contentLost) {
        rule.contentElement = this.contentDOM;
      } else {
        for (let i2 = this.children.length - 1; i2 >= 0; i2--) {
          let child = this.children[i2];
          if (this.dom.contains(child.dom.parentNode)) {
            rule.contentElement = child.dom.parentNode;
            break;
          }
        }
        if (!rule.contentElement)
          rule.getContent = () => Fragment.empty;
      }
      return rule;
    }
    matchesNode(node2, outerDeco, innerDeco) {
      return this.dirty == NOT_DIRTY && node2.eq(this.node) && sameOuterDeco(outerDeco, this.outerDeco) && innerDeco.eq(this.innerDeco);
    }
    get size() {
      return this.node.nodeSize;
    }
    get border() {
      return this.node.isLeaf ? 0 : 1;
    }
    // Syncs `this.children` to match `this.node.content` and the local
    // decorations, possibly introducing nesting for marks. Then, in a
    // separate step, syncs the DOM inside `this.contentDOM` to
    // `this.children`.
    updateChildren(view, pos) {
      let inline = this.node.inlineContent, off = pos;
      let composition = view.composing ? this.localCompositionInfo(view, pos) : null;
      let localComposition = composition && composition.pos > -1 ? composition : null;
      let compositionInChild = composition && composition.pos < 0;
      let updater = new ViewTreeUpdater(this, localComposition && localComposition.node, view);
      iterDeco(this.node, this.innerDeco, (widget, i2, insideNode) => {
        if (widget.spec.marks)
          updater.syncToMarks(widget.spec.marks, inline, view, i2);
        else if (widget.type.side >= 0 && !insideNode)
          updater.syncToMarks(i2 == this.node.childCount ? Mark.none : this.node.child(i2).marks, inline, view, i2);
        updater.placeWidget(widget, view, off);
      }, (child, outerDeco, innerDeco, i2) => {
        updater.syncToMarks(child.marks, inline, view, i2);
        let compIndex;
        if (updater.findNodeMatch(child, outerDeco, innerDeco, i2)) ;
        else if (compositionInChild && view.state.selection.from > off && view.state.selection.to < off + child.nodeSize && (compIndex = updater.findIndexWithChild(composition.node)) > -1 && updater.updateNodeAt(child, outerDeco, innerDeco, compIndex, view)) ;
        else if (updater.updateNextNode(child, outerDeco, innerDeco, view, i2, off)) ;
        else {
          updater.addNode(child, outerDeco, innerDeco, view, off);
        }
        off += child.nodeSize;
      });
      updater.syncToMarks([], inline, view, 0);
      if (this.node.isTextblock)
        updater.addTextblockHacks();
      updater.destroyRest();
      if (updater.changed || this.dirty == CONTENT_DIRTY) {
        if (localComposition)
          this.protectLocalComposition(view, localComposition);
        renderDescs(this.contentDOM, this.children, view);
        if (ios2)
          iosHacks(this.dom);
      }
    }
    localCompositionInfo(view, pos) {
      let { from, to } = view.state.selection;
      if (!(view.state.selection instanceof TextSelection) || from < pos || to > pos + this.node.content.size)
        return null;
      let textNode = view.input.compositionNode;
      if (!textNode || !this.dom.contains(textNode.parentNode))
        return null;
      if (this.node.inlineContent) {
        let text4 = textNode.nodeValue;
        let textPos = findTextInFragment(this.node.content, text4, from - pos, to - pos);
        return textPos < 0 ? null : { node: textNode, pos: textPos, text: text4 };
      } else {
        return { node: textNode, pos: -1, text: "" };
      }
    }
    protectLocalComposition(view, { node: node2, pos, text: text4 }) {
      if (this.getDesc(node2))
        return;
      let topNode = node2;
      for (; ; topNode = topNode.parentNode) {
        if (topNode.parentNode == this.contentDOM)
          break;
        while (topNode.previousSibling)
          topNode.parentNode.removeChild(topNode.previousSibling);
        while (topNode.nextSibling)
          topNode.parentNode.removeChild(topNode.nextSibling);
        if (topNode.pmViewDesc)
          topNode.pmViewDesc = void 0;
      }
      let desc = new CompositionViewDesc(this, topNode, node2, text4);
      view.input.compositionNodes.push(desc);
      this.children = replaceNodes(this.children, pos, pos + text4.length, view, desc);
    }
    // If this desc must be updated to match the given node decoration,
    // do so and return true.
    update(node2, outerDeco, innerDeco, view) {
      if (this.dirty == NODE_DIRTY || !node2.sameMarkup(this.node))
        return false;
      this.updateInner(node2, outerDeco, innerDeco, view);
      return true;
    }
    updateInner(node2, outerDeco, innerDeco, view) {
      this.updateOuterDeco(outerDeco);
      this.node = node2;
      this.innerDeco = innerDeco;
      if (this.contentDOM)
        this.updateChildren(view, this.posAtStart);
      this.dirty = NOT_DIRTY;
    }
    updateOuterDeco(outerDeco) {
      if (sameOuterDeco(outerDeco, this.outerDeco))
        return;
      let needsWrap = this.nodeDOM.nodeType != 1;
      let oldDOM = this.dom;
      this.dom = patchOuterDeco(this.dom, this.nodeDOM, computeOuterDeco(this.outerDeco, this.node, needsWrap), computeOuterDeco(outerDeco, this.node, needsWrap));
      if (this.dom != oldDOM) {
        oldDOM.pmViewDesc = void 0;
        this.dom.pmViewDesc = this;
      }
      this.outerDeco = outerDeco;
    }
    // Mark this node as being the selected node.
    selectNode() {
      if (this.nodeDOM.nodeType == 1) {
        this.nodeDOM.classList.add("ProseMirror-selectednode");
        if (this.contentDOM || !this.node.type.spec.draggable)
          this.nodeDOM.draggable = true;
      }
    }
    // Remove selected node marking from this node.
    deselectNode() {
      if (this.nodeDOM.nodeType == 1) {
        this.nodeDOM.classList.remove("ProseMirror-selectednode");
        if (this.contentDOM || !this.node.type.spec.draggable)
          this.nodeDOM.removeAttribute("draggable");
      }
    }
    get domAtom() {
      return this.node.isAtom;
    }
  };
  function docViewDesc(doc4, outerDeco, innerDeco, dom, view) {
    applyOuterDeco(dom, outerDeco, doc4);
    let docView = new NodeViewDesc(void 0, doc4, outerDeco, innerDeco, dom, dom, dom, view, 0);
    if (docView.contentDOM)
      docView.updateChildren(view, 0);
    return docView;
  }
  var TextViewDesc = class _TextViewDesc extends NodeViewDesc {
    constructor(parent, node2, outerDeco, innerDeco, dom, nodeDOM, view) {
      super(parent, node2, outerDeco, innerDeco, dom, null, nodeDOM, view, 0);
    }
    parseRule() {
      let skip = this.nodeDOM.parentNode;
      while (skip && skip != this.dom && !skip.pmIsDeco)
        skip = skip.parentNode;
      return { skip: skip || true };
    }
    update(node2, outerDeco, innerDeco, view) {
      if (this.dirty == NODE_DIRTY || this.dirty != NOT_DIRTY && !this.inParent() || !node2.sameMarkup(this.node))
        return false;
      this.updateOuterDeco(outerDeco);
      if ((this.dirty != NOT_DIRTY || node2.text != this.node.text) && node2.text != this.nodeDOM.nodeValue) {
        this.nodeDOM.nodeValue = node2.text;
        if (view.trackWrites == this.nodeDOM)
          view.trackWrites = null;
      }
      this.node = node2;
      this.dirty = NOT_DIRTY;
      return true;
    }
    inParent() {
      let parentDOM = this.parent.contentDOM;
      for (let n2 = this.nodeDOM; n2; n2 = n2.parentNode)
        if (n2 == parentDOM)
          return true;
      return false;
    }
    domFromPos(pos) {
      return { node: this.nodeDOM, offset: pos };
    }
    localPosFromDOM(dom, offset, bias) {
      if (dom == this.nodeDOM)
        return this.posAtStart + Math.min(offset, this.node.text.length);
      return super.localPosFromDOM(dom, offset, bias);
    }
    ignoreMutation(mutation) {
      return mutation.type != "characterData" && mutation.type != "selection";
    }
    slice(from, to, view) {
      let node2 = this.node.cut(from, to), dom = document.createTextNode(node2.text);
      return new _TextViewDesc(this.parent, node2, this.outerDeco, this.innerDeco, dom, dom, view);
    }
    markDirty(from, to) {
      super.markDirty(from, to);
      if (this.dom != this.nodeDOM && (from == 0 || to == this.nodeDOM.nodeValue.length))
        this.dirty = NODE_DIRTY;
    }
    get domAtom() {
      return false;
    }
    isText(text4) {
      return this.node.text == text4;
    }
  };
  var TrailingHackViewDesc = class extends ViewDesc {
    parseRule() {
      return { ignore: true };
    }
    matchesHack(nodeName) {
      return this.dirty == NOT_DIRTY && this.dom.nodeName == nodeName;
    }
    get domAtom() {
      return true;
    }
    get ignoreForCoords() {
      return this.dom.nodeName == "IMG";
    }
  };
  var CustomNodeViewDesc = class extends NodeViewDesc {
    constructor(parent, node2, outerDeco, innerDeco, dom, contentDOM, nodeDOM, spec, view, pos) {
      super(parent, node2, outerDeco, innerDeco, dom, contentDOM, nodeDOM, view, pos);
      this.spec = spec;
    }
    // A custom `update` method gets to decide whether the update goes
    // through. If it does, and there's a `contentDOM` node, our logic
    // updates the children.
    update(node2, outerDeco, innerDeco, view) {
      if (this.dirty == NODE_DIRTY)
        return false;
      if (this.spec.update && (this.node.type == node2.type || this.spec.multiType)) {
        let result = this.spec.update(node2, outerDeco, innerDeco);
        if (result)
          this.updateInner(node2, outerDeco, innerDeco, view);
        return result;
      } else if (!this.contentDOM && !node2.isLeaf) {
        return false;
      } else {
        return super.update(node2, outerDeco, innerDeco, view);
      }
    }
    selectNode() {
      this.spec.selectNode ? this.spec.selectNode() : super.selectNode();
    }
    deselectNode() {
      this.spec.deselectNode ? this.spec.deselectNode() : super.deselectNode();
    }
    setSelection(anchor, head, view, force) {
      this.spec.setSelection ? this.spec.setSelection(anchor, head, view.root) : super.setSelection(anchor, head, view, force);
    }
    destroy() {
      if (this.spec.destroy)
        this.spec.destroy();
      super.destroy();
    }
    stopEvent(event) {
      return this.spec.stopEvent ? this.spec.stopEvent(event) : false;
    }
    ignoreMutation(mutation) {
      return this.spec.ignoreMutation ? this.spec.ignoreMutation(mutation) : super.ignoreMutation(mutation);
    }
  };
  function renderDescs(parentDOM, descs, view) {
    let dom = parentDOM.firstChild, written = false;
    for (let i2 = 0; i2 < descs.length; i2++) {
      let desc = descs[i2], childDOM = desc.dom;
      if (childDOM.parentNode == parentDOM) {
        while (childDOM != dom) {
          dom = rm(dom);
          written = true;
        }
        dom = dom.nextSibling;
      } else {
        written = true;
        parentDOM.insertBefore(childDOM, dom);
      }
      if (desc instanceof MarkViewDesc) {
        let pos = dom ? dom.previousSibling : parentDOM.lastChild;
        renderDescs(desc.contentDOM, desc.children, view);
        dom = pos ? pos.nextSibling : parentDOM.firstChild;
      }
    }
    while (dom) {
      dom = rm(dom);
      written = true;
    }
    if (written && view.trackWrites == parentDOM)
      view.trackWrites = null;
  }
  var OuterDecoLevel = function(nodeName) {
    if (nodeName)
      this.nodeName = nodeName;
  };
  OuterDecoLevel.prototype = /* @__PURE__ */ Object.create(null);
  var noDeco = [new OuterDecoLevel()];
  function computeOuterDeco(outerDeco, node2, needsWrap) {
    if (outerDeco.length == 0)
      return noDeco;
    let top = needsWrap ? noDeco[0] : new OuterDecoLevel(), result = [top];
    for (let i2 = 0; i2 < outerDeco.length; i2++) {
      let attrs = outerDeco[i2].type.attrs;
      if (!attrs)
        continue;
      if (attrs.nodeName)
        result.push(top = new OuterDecoLevel(attrs.nodeName));
      for (let name in attrs) {
        let val = attrs[name];
        if (val == null)
          continue;
        if (needsWrap && result.length == 1)
          result.push(top = new OuterDecoLevel(node2.isInline ? "span" : "div"));
        if (name == "class")
          top.class = (top.class ? top.class + " " : "") + val;
        else if (name == "style")
          top.style = (top.style ? top.style + ";" : "") + val;
        else if (name != "nodeName")
          top[name] = val;
      }
    }
    return result;
  }
  function patchOuterDeco(outerDOM, nodeDOM, prevComputed, curComputed) {
    if (prevComputed == noDeco && curComputed == noDeco)
      return nodeDOM;
    let curDOM = nodeDOM;
    for (let i2 = 0; i2 < curComputed.length; i2++) {
      let deco = curComputed[i2], prev = prevComputed[i2];
      if (i2) {
        let parent;
        if (prev && prev.nodeName == deco.nodeName && curDOM != outerDOM && (parent = curDOM.parentNode) && parent.nodeName.toLowerCase() == deco.nodeName) {
          curDOM = parent;
        } else {
          parent = document.createElement(deco.nodeName);
          parent.pmIsDeco = true;
          parent.appendChild(curDOM);
          prev = noDeco[0];
          curDOM = parent;
        }
      }
      patchAttributes(curDOM, prev || noDeco[0], deco);
    }
    return curDOM;
  }
  function patchAttributes(dom, prev, cur) {
    for (let name in prev)
      if (name != "class" && name != "style" && name != "nodeName" && !(name in cur))
        dom.removeAttribute(name);
    for (let name in cur)
      if (name != "class" && name != "style" && name != "nodeName" && cur[name] != prev[name])
        dom.setAttribute(name, cur[name]);
    if (prev.class != cur.class) {
      let prevList = prev.class ? prev.class.split(" ").filter(Boolean) : [];
      let curList = cur.class ? cur.class.split(" ").filter(Boolean) : [];
      for (let i2 = 0; i2 < prevList.length; i2++)
        if (curList.indexOf(prevList[i2]) == -1)
          dom.classList.remove(prevList[i2]);
      for (let i2 = 0; i2 < curList.length; i2++)
        if (prevList.indexOf(curList[i2]) == -1)
          dom.classList.add(curList[i2]);
      if (dom.classList.length == 0)
        dom.removeAttribute("class");
    }
    if (prev.style != cur.style) {
      if (prev.style) {
        let prop = /\s*([\w\-\xa1-\uffff]+)\s*:(?:"(?:\\.|[^"])*"|'(?:\\.|[^'])*'|\(.*?\)|[^;])*/g, m;
        while (m = prop.exec(prev.style))
          dom.style.removeProperty(m[1]);
      }
      if (cur.style)
        dom.style.cssText += cur.style;
    }
  }
  function applyOuterDeco(dom, deco, node2) {
    return patchOuterDeco(dom, dom, noDeco, computeOuterDeco(deco, node2, dom.nodeType != 1));
  }
  function sameOuterDeco(a2, b) {
    if (a2.length != b.length)
      return false;
    for (let i2 = 0; i2 < a2.length; i2++)
      if (!a2[i2].type.eq(b[i2].type))
        return false;
    return true;
  }
  function rm(dom) {
    let next = dom.nextSibling;
    dom.parentNode.removeChild(dom);
    return next;
  }
  var ViewTreeUpdater = class {
    constructor(top, lock, view) {
      this.lock = lock;
      this.view = view;
      this.index = 0;
      this.stack = [];
      this.changed = false;
      this.top = top;
      this.preMatch = preMatch(top.node.content, top);
    }
    // Destroy and remove the children between the given indices in
    // `this.top`.
    destroyBetween(start, end) {
      if (start == end)
        return;
      for (let i2 = start; i2 < end; i2++)
        this.top.children[i2].destroy();
      this.top.children.splice(start, end - start);
      this.changed = true;
    }
    // Destroy all remaining children in `this.top`.
    destroyRest() {
      this.destroyBetween(this.index, this.top.children.length);
    }
    // Sync the current stack of mark descs with the given array of
    // marks, reusing existing mark descs when possible.
    syncToMarks(marks, inline, view, parentIndex) {
      let keep = 0, depth = this.stack.length >> 1;
      let maxKeep = Math.min(depth, marks.length);
      while (keep < maxKeep && (keep == depth - 1 ? this.top : this.stack[keep + 1 << 1]).matchesMark(marks[keep]) && marks[keep].type.spec.spanning !== false)
        keep++;
      while (keep < depth) {
        this.destroyRest();
        this.top.dirty = NOT_DIRTY;
        this.index = this.stack.pop();
        this.top = this.stack.pop();
        depth--;
      }
      while (depth < marks.length) {
        this.stack.push(this.top, this.index + 1);
        let found2 = -1, scanTo = this.top.children.length;
        if (parentIndex < this.preMatch.index)
          scanTo = Math.min(this.index + 3, scanTo);
        for (let i2 = this.index; i2 < scanTo; i2++) {
          let next = this.top.children[i2];
          if (next.matchesMark(marks[depth]) && !this.isLocked(next.dom)) {
            found2 = i2;
            break;
          }
        }
        if (found2 > -1) {
          if (found2 > this.index) {
            this.changed = true;
            this.destroyBetween(this.index, found2);
          }
          this.top = this.top.children[this.index];
        } else {
          let markDesc = MarkViewDesc.create(this.top, marks[depth], inline, view);
          this.top.children.splice(this.index, 0, markDesc);
          this.top = markDesc;
          this.changed = true;
        }
        this.index = 0;
        depth++;
      }
    }
    // Try to find a node desc matching the given data. Skip over it and
    // return true when successful.
    findNodeMatch(node2, outerDeco, innerDeco, index2) {
      let found2 = -1, targetDesc;
      if (index2 >= this.preMatch.index && (targetDesc = this.preMatch.matches[index2 - this.preMatch.index]).parent == this.top && targetDesc.matchesNode(node2, outerDeco, innerDeco)) {
        found2 = this.top.children.indexOf(targetDesc, this.index);
      } else {
        for (let i2 = this.index, e2 = Math.min(this.top.children.length, i2 + 5); i2 < e2; i2++) {
          let child = this.top.children[i2];
          if (child.matchesNode(node2, outerDeco, innerDeco) && !this.preMatch.matched.has(child)) {
            found2 = i2;
            break;
          }
        }
      }
      if (found2 < 0)
        return false;
      this.destroyBetween(this.index, found2);
      this.index++;
      return true;
    }
    updateNodeAt(node2, outerDeco, innerDeco, index2, view) {
      let child = this.top.children[index2];
      if (child.dirty == NODE_DIRTY && child.dom == child.contentDOM)
        child.dirty = CONTENT_DIRTY;
      if (!child.update(node2, outerDeco, innerDeco, view))
        return false;
      this.destroyBetween(this.index, index2);
      this.index++;
      return true;
    }
    findIndexWithChild(domNode) {
      for (; ; ) {
        let parent = domNode.parentNode;
        if (!parent)
          return -1;
        if (parent == this.top.contentDOM) {
          let desc = domNode.pmViewDesc;
          if (desc)
            for (let i2 = this.index; i2 < this.top.children.length; i2++) {
              if (this.top.children[i2] == desc)
                return i2;
            }
          return -1;
        }
        domNode = parent;
      }
    }
    // Try to update the next node, if any, to the given data. Checks
    // pre-matches to avoid overwriting nodes that could still be used.
    updateNextNode(node2, outerDeco, innerDeco, view, index2, pos) {
      for (let i2 = this.index; i2 < this.top.children.length; i2++) {
        let next = this.top.children[i2];
        if (next instanceof NodeViewDesc) {
          let preMatch2 = this.preMatch.matched.get(next);
          if (preMatch2 != null && preMatch2 != index2)
            return false;
          let nextDOM = next.dom, updated;
          let locked = this.isLocked(nextDOM) && !(node2.isText && next.node && next.node.isText && next.nodeDOM.nodeValue == node2.text && next.dirty != NODE_DIRTY && sameOuterDeco(outerDeco, next.outerDeco));
          if (!locked && next.update(node2, outerDeco, innerDeco, view)) {
            this.destroyBetween(this.index, i2);
            if (next.dom != nextDOM)
              this.changed = true;
            this.index++;
            return true;
          } else if (!locked && (updated = this.recreateWrapper(next, node2, outerDeco, innerDeco, view, pos))) {
            this.destroyBetween(this.index, i2);
            this.top.children[this.index] = updated;
            if (updated.contentDOM) {
              updated.dirty = CONTENT_DIRTY;
              updated.updateChildren(view, pos + 1);
              updated.dirty = NOT_DIRTY;
            }
            this.changed = true;
            this.index++;
            return true;
          }
          break;
        }
      }
      return false;
    }
    // When a node with content is replaced by a different node with
    // identical content, move over its children.
    recreateWrapper(next, node2, outerDeco, innerDeco, view, pos) {
      if (next.dirty || node2.isAtom || !next.children.length || !next.node.content.eq(node2.content) || !sameOuterDeco(outerDeco, next.outerDeco) || !innerDeco.eq(next.innerDeco))
        return null;
      let wrapper = NodeViewDesc.create(this.top, node2, outerDeco, innerDeco, view, pos);
      if (wrapper.contentDOM) {
        wrapper.children = next.children;
        next.children = [];
        for (let ch of wrapper.children)
          ch.parent = wrapper;
      }
      next.destroy();
      return wrapper;
    }
    // Insert the node as a newly created node desc.
    addNode(node2, outerDeco, innerDeco, view, pos) {
      let desc = NodeViewDesc.create(this.top, node2, outerDeco, innerDeco, view, pos);
      if (desc.contentDOM)
        desc.updateChildren(view, pos + 1);
      this.top.children.splice(this.index++, 0, desc);
      this.changed = true;
    }
    placeWidget(widget, view, pos) {
      let next = this.index < this.top.children.length ? this.top.children[this.index] : null;
      if (next && next.matchesWidget(widget) && (widget == next.widget || !next.widget.type.toDOM.parentNode)) {
        this.index++;
      } else {
        let desc = new WidgetViewDesc(this.top, widget, view, pos);
        this.top.children.splice(this.index++, 0, desc);
        this.changed = true;
      }
    }
    // Make sure a textblock looks and behaves correctly in
    // contentEditable.
    addTextblockHacks() {
      let lastChild = this.top.children[this.index - 1], parent = this.top;
      while (lastChild instanceof MarkViewDesc) {
        parent = lastChild;
        lastChild = parent.children[parent.children.length - 1];
      }
      if (!lastChild || // Empty textblock
      !(lastChild instanceof TextViewDesc) || /\n$/.test(lastChild.node.text) || this.view.requiresGeckoHackNode && /\s$/.test(lastChild.node.text)) {
        if ((safari2 || chrome) && lastChild && lastChild.dom.contentEditable == "false")
          this.addHackNode("IMG", parent);
        this.addHackNode("BR", this.top);
      }
    }
    addHackNode(nodeName, parent) {
      if (parent == this.top && this.index < parent.children.length && parent.children[this.index].matchesHack(nodeName)) {
        this.index++;
      } else {
        let dom = document.createElement(nodeName);
        if (nodeName == "IMG") {
          dom.className = "ProseMirror-separator";
          dom.alt = "";
        }
        if (nodeName == "BR")
          dom.className = "ProseMirror-trailingBreak";
        let hack = new TrailingHackViewDesc(this.top, [], dom, null);
        if (parent != this.top)
          parent.children.push(hack);
        else
          parent.children.splice(this.index++, 0, hack);
        this.changed = true;
      }
    }
    isLocked(node2) {
      return this.lock && (node2 == this.lock || node2.nodeType == 1 && node2.contains(this.lock.parentNode));
    }
  };
  function preMatch(frag, parentDesc) {
    let curDesc = parentDesc, descI = curDesc.children.length;
    let fI = frag.childCount, matched = /* @__PURE__ */ new Map(), matches2 = [];
    outer: while (fI > 0) {
      let desc;
      for (; ; ) {
        if (descI) {
          let next = curDesc.children[descI - 1];
          if (next instanceof MarkViewDesc) {
            curDesc = next;
            descI = next.children.length;
          } else {
            desc = next;
            descI--;
            break;
          }
        } else if (curDesc == parentDesc) {
          break outer;
        } else {
          descI = curDesc.parent.children.indexOf(curDesc);
          curDesc = curDesc.parent;
        }
      }
      let node2 = desc.node;
      if (!node2)
        continue;
      if (node2 != frag.child(fI - 1))
        break;
      --fI;
      matched.set(desc, fI);
      matches2.push(desc);
    }
    return { index: fI, matched, matches: matches2.reverse() };
  }
  function compareSide(a2, b) {
    return a2.type.side - b.type.side;
  }
  function iterDeco(parent, deco, onWidget, onNode) {
    let locals = deco.locals(parent), offset = 0;
    if (locals.length == 0) {
      for (let i2 = 0; i2 < parent.childCount; i2++) {
        let child = parent.child(i2);
        onNode(child, locals, deco.forChild(offset, child), i2);
        offset += child.nodeSize;
      }
      return;
    }
    let decoIndex = 0, active = [], restNode = null;
    for (let parentIndex = 0; ; ) {
      let widget, widgets;
      while (decoIndex < locals.length && locals[decoIndex].to == offset) {
        let next = locals[decoIndex++];
        if (next.widget) {
          if (!widget)
            widget = next;
          else
            (widgets || (widgets = [widget])).push(next);
        }
      }
      if (widget) {
        if (widgets) {
          widgets.sort(compareSide);
          for (let i2 = 0; i2 < widgets.length; i2++)
            onWidget(widgets[i2], parentIndex, !!restNode);
        } else {
          onWidget(widget, parentIndex, !!restNode);
        }
      }
      let child, index2;
      if (restNode) {
        index2 = -1;
        child = restNode;
        restNode = null;
      } else if (parentIndex < parent.childCount) {
        index2 = parentIndex;
        child = parent.child(parentIndex++);
      } else {
        break;
      }
      for (let i2 = 0; i2 < active.length; i2++)
        if (active[i2].to <= offset)
          active.splice(i2--, 1);
      while (decoIndex < locals.length && locals[decoIndex].from <= offset && locals[decoIndex].to > offset)
        active.push(locals[decoIndex++]);
      let end = offset + child.nodeSize;
      if (child.isText) {
        let cutAt = end;
        if (decoIndex < locals.length && locals[decoIndex].from < cutAt)
          cutAt = locals[decoIndex].from;
        for (let i2 = 0; i2 < active.length; i2++)
          if (active[i2].to < cutAt)
            cutAt = active[i2].to;
        if (cutAt < end) {
          restNode = child.cut(cutAt - offset);
          child = child.cut(0, cutAt - offset);
          end = cutAt;
          index2 = -1;
        }
      } else {
        while (decoIndex < locals.length && locals[decoIndex].to < end)
          decoIndex++;
      }
      let outerDeco = child.isInline && !child.isLeaf ? active.filter((d2) => !d2.inline) : active.slice();
      onNode(child, outerDeco, deco.forChild(offset, child), index2);
      offset = end;
    }
  }
  function iosHacks(dom) {
    if (dom.nodeName == "UL" || dom.nodeName == "OL") {
      let oldCSS = dom.style.cssText;
      dom.style.cssText = oldCSS + "; list-style: square !important";
      window.getComputedStyle(dom).listStyle;
      dom.style.cssText = oldCSS;
    }
  }
  function findTextInFragment(frag, text4, from, to) {
    for (let i2 = 0, pos = 0; i2 < frag.childCount && pos <= to; ) {
      let child = frag.child(i2++), childStart = pos;
      pos += child.nodeSize;
      if (!child.isText)
        continue;
      let str = child.text;
      while (i2 < frag.childCount) {
        let next = frag.child(i2++);
        pos += next.nodeSize;
        if (!next.isText)
          break;
        str += next.text;
      }
      if (pos >= from) {
        if (pos >= to && str.slice(to - text4.length - childStart, to - childStart) == text4)
          return to - text4.length;
        let found2 = childStart < to ? str.lastIndexOf(text4, to - childStart - 1) : -1;
        if (found2 >= 0 && found2 + text4.length + childStart >= from)
          return childStart + found2;
        if (from == to && str.length >= to + text4.length - childStart && str.slice(to - childStart, to - childStart + text4.length) == text4)
          return to;
      }
    }
    return -1;
  }
  function replaceNodes(nodes, from, to, view, replacement) {
    let result = [];
    for (let i2 = 0, off = 0; i2 < nodes.length; i2++) {
      let child = nodes[i2], start = off, end = off += child.size;
      if (start >= to || end <= from) {
        result.push(child);
      } else {
        if (start < from)
          result.push(child.slice(0, from - start, view));
        if (replacement) {
          result.push(replacement);
          replacement = void 0;
        }
        if (end > to)
          result.push(child.slice(to - start, child.size, view));
      }
    }
    return result;
  }
  function selectionFromDOM(view, origin = null) {
    let domSel = view.domSelectionRange(), doc4 = view.state.doc;
    if (!domSel.focusNode)
      return null;
    let nearestDesc = view.docView.nearestDesc(domSel.focusNode), inWidget = nearestDesc && nearestDesc.size == 0;
    let head = view.docView.posFromDOM(domSel.focusNode, domSel.focusOffset, 1);
    if (head < 0)
      return null;
    let $head = doc4.resolve(head), anchor, selection;
    if (selectionCollapsed(domSel)) {
      anchor = head;
      while (nearestDesc && !nearestDesc.node)
        nearestDesc = nearestDesc.parent;
      let nearestDescNode = nearestDesc.node;
      if (nearestDesc && nearestDescNode.isAtom && NodeSelection.isSelectable(nearestDescNode) && nearestDesc.parent && !(nearestDescNode.isInline && isOnEdge(domSel.focusNode, domSel.focusOffset, nearestDesc.dom))) {
        let pos = nearestDesc.posBefore;
        selection = new NodeSelection(head == pos ? $head : doc4.resolve(pos));
      }
    } else {
      if (domSel instanceof view.dom.ownerDocument.defaultView.Selection && domSel.rangeCount > 1) {
        let min = head, max = head;
        for (let i2 = 0; i2 < domSel.rangeCount; i2++) {
          let range = domSel.getRangeAt(i2);
          min = Math.min(min, view.docView.posFromDOM(range.startContainer, range.startOffset, 1));
          max = Math.max(max, view.docView.posFromDOM(range.endContainer, range.endOffset, -1));
        }
        if (min < 0)
          return null;
        [anchor, head] = max == view.state.selection.anchor ? [max, min] : [min, max];
        $head = doc4.resolve(head);
      } else {
        anchor = view.docView.posFromDOM(domSel.anchorNode, domSel.anchorOffset, 1);
      }
      if (anchor < 0)
        return null;
    }
    let $anchor = doc4.resolve(anchor);
    if (!selection) {
      let bias = origin == "pointer" || view.state.selection.head < $head.pos && !inWidget ? 1 : -1;
      selection = selectionBetween(view, $anchor, $head, bias);
    }
    return selection;
  }
  function editorOwnsSelection(view) {
    return view.editable ? view.hasFocus() : hasSelection(view) && document.activeElement && document.activeElement.contains(view.dom);
  }
  function selectionToDOM(view, force = false) {
    let sel = view.state.selection;
    syncNodeSelection(view, sel);
    if (!editorOwnsSelection(view))
      return;
    if (!force && view.input.mouseDown && view.input.mouseDown.allowDefault && chrome) {
      let domSel = view.domSelectionRange(), curSel = view.domObserver.currentSelection;
      if (domSel.anchorNode && curSel.anchorNode && isEquivalentPosition(domSel.anchorNode, domSel.anchorOffset, curSel.anchorNode, curSel.anchorOffset)) {
        view.input.mouseDown.delayedSelectionSync = true;
        view.domObserver.setCurSelection();
        return;
      }
    }
    view.domObserver.disconnectSelection();
    if (view.cursorWrapper) {
      selectCursorWrapper(view);
    } else {
      let { anchor, head } = sel, resetEditableFrom, resetEditableTo;
      if (brokenSelectBetweenUneditable && !(sel instanceof TextSelection)) {
        if (!sel.$from.parent.inlineContent)
          resetEditableFrom = temporarilyEditableNear(view, sel.from);
        if (!sel.empty && !sel.$from.parent.inlineContent)
          resetEditableTo = temporarilyEditableNear(view, sel.to);
      }
      view.docView.setSelection(anchor, head, view, force);
      if (brokenSelectBetweenUneditable) {
        if (resetEditableFrom)
          resetEditable(resetEditableFrom);
        if (resetEditableTo)
          resetEditable(resetEditableTo);
      }
      if (sel.visible) {
        view.dom.classList.remove("ProseMirror-hideselection");
      } else {
        view.dom.classList.add("ProseMirror-hideselection");
        if ("onselectionchange" in document)
          removeClassOnSelectionChange(view);
      }
    }
    view.domObserver.setCurSelection();
    view.domObserver.connectSelection();
  }
  var brokenSelectBetweenUneditable = safari2 || chrome && chrome_version2 < 63;
  function temporarilyEditableNear(view, pos) {
    let { node: node2, offset } = view.docView.domFromPos(pos, 0);
    let after = offset < node2.childNodes.length ? node2.childNodes[offset] : null;
    let before = offset ? node2.childNodes[offset - 1] : null;
    if (safari2 && after && after.contentEditable == "false")
      return setEditable(after);
    if ((!after || after.contentEditable == "false") && (!before || before.contentEditable == "false")) {
      if (after)
        return setEditable(after);
      else if (before)
        return setEditable(before);
    }
  }
  function setEditable(element2) {
    element2.contentEditable = "true";
    if (safari2 && element2.draggable) {
      element2.draggable = false;
      element2.wasDraggable = true;
    }
    return element2;
  }
  function resetEditable(element2) {
    element2.contentEditable = "false";
    if (element2.wasDraggable) {
      element2.draggable = true;
      element2.wasDraggable = null;
    }
  }
  function removeClassOnSelectionChange(view) {
    let doc4 = view.dom.ownerDocument;
    doc4.removeEventListener("selectionchange", view.input.hideSelectionGuard);
    let domSel = view.domSelectionRange();
    let node2 = domSel.anchorNode, offset = domSel.anchorOffset;
    doc4.addEventListener("selectionchange", view.input.hideSelectionGuard = () => {
      if (domSel.anchorNode != node2 || domSel.anchorOffset != offset) {
        doc4.removeEventListener("selectionchange", view.input.hideSelectionGuard);
        setTimeout(() => {
          if (!editorOwnsSelection(view) || view.state.selection.visible)
            view.dom.classList.remove("ProseMirror-hideselection");
        }, 20);
      }
    });
  }
  function selectCursorWrapper(view) {
    let domSel = view.domSelection();
    if (!domSel)
      return;
    let node2 = view.cursorWrapper.dom, img = node2.nodeName == "IMG";
    if (img)
      domSel.collapse(node2.parentNode, domIndex(node2) + 1);
    else
      domSel.collapse(node2, 0);
    if (!img && !view.state.selection.visible && ie3 && ie_version2 <= 11) {
      node2.disabled = true;
      node2.disabled = false;
    }
  }
  function syncNodeSelection(view, sel) {
    if (sel instanceof NodeSelection) {
      let desc = view.docView.descAt(sel.from);
      if (desc != view.lastSelectedViewDesc) {
        clearNodeSelection(view);
        if (desc)
          desc.selectNode();
        view.lastSelectedViewDesc = desc;
      }
    } else {
      clearNodeSelection(view);
    }
  }
  function clearNodeSelection(view) {
    if (view.lastSelectedViewDesc) {
      if (view.lastSelectedViewDesc.parent)
        view.lastSelectedViewDesc.deselectNode();
      view.lastSelectedViewDesc = void 0;
    }
  }
  function selectionBetween(view, $anchor, $head, bias) {
    return view.someProp("createSelectionBetween", (f2) => f2(view, $anchor, $head)) || TextSelection.between($anchor, $head, bias);
  }
  function hasFocusAndSelection(view) {
    if (view.editable && !view.hasFocus())
      return false;
    return hasSelection(view);
  }
  function hasSelection(view) {
    let sel = view.domSelectionRange();
    if (!sel.anchorNode)
      return false;
    try {
      return view.dom.contains(sel.anchorNode.nodeType == 3 ? sel.anchorNode.parentNode : sel.anchorNode) && (view.editable || view.dom.contains(sel.focusNode.nodeType == 3 ? sel.focusNode.parentNode : sel.focusNode));
    } catch (_) {
      return false;
    }
  }
  function anchorInRightPlace(view) {
    let anchorDOM = view.docView.domFromPos(view.state.selection.anchor, 0);
    let domSel = view.domSelectionRange();
    return isEquivalentPosition(anchorDOM.node, anchorDOM.offset, domSel.anchorNode, domSel.anchorOffset);
  }
  function moveSelectionBlock(state, dir) {
    let { $anchor, $head } = state.selection;
    let $side = dir > 0 ? $anchor.max($head) : $anchor.min($head);
    let $start = !$side.parent.inlineContent ? $side : $side.depth ? state.doc.resolve(dir > 0 ? $side.after() : $side.before()) : null;
    return $start && Selection.findFrom($start, dir);
  }
  function apply(view, sel) {
    view.dispatch(view.state.tr.setSelection(sel).scrollIntoView());
    return true;
  }
  function selectHorizontally(view, dir, mods) {
    let sel = view.state.selection;
    if (sel instanceof TextSelection) {
      if (mods.indexOf("s") > -1) {
        let { $head } = sel, node2 = $head.textOffset ? null : dir < 0 ? $head.nodeBefore : $head.nodeAfter;
        if (!node2 || node2.isText || !node2.isLeaf)
          return false;
        let $newHead = view.state.doc.resolve($head.pos + node2.nodeSize * (dir < 0 ? -1 : 1));
        return apply(view, new TextSelection(sel.$anchor, $newHead));
      } else if (!sel.empty) {
        return false;
      } else if (view.endOfTextblock(dir > 0 ? "forward" : "backward")) {
        let next = moveSelectionBlock(view.state, dir);
        if (next && next instanceof NodeSelection)
          return apply(view, next);
        return false;
      } else if (!(mac5 && mods.indexOf("m") > -1)) {
        let $head = sel.$head, node2 = $head.textOffset ? null : dir < 0 ? $head.nodeBefore : $head.nodeAfter, desc;
        if (!node2 || node2.isText)
          return false;
        let nodePos = dir < 0 ? $head.pos - node2.nodeSize : $head.pos;
        if (!(node2.isAtom || (desc = view.docView.descAt(nodePos)) && !desc.contentDOM))
          return false;
        if (NodeSelection.isSelectable(node2)) {
          return apply(view, new NodeSelection(dir < 0 ? view.state.doc.resolve($head.pos - node2.nodeSize) : $head));
        } else if (webkit2) {
          return apply(view, new TextSelection(view.state.doc.resolve(dir < 0 ? nodePos : nodePos + node2.nodeSize)));
        } else {
          return false;
        }
      }
    } else if (sel instanceof NodeSelection && sel.node.isInline) {
      return apply(view, new TextSelection(dir > 0 ? sel.$to : sel.$from));
    } else {
      let next = moveSelectionBlock(view.state, dir);
      if (next)
        return apply(view, next);
      return false;
    }
  }
  function nodeLen(node2) {
    return node2.nodeType == 3 ? node2.nodeValue.length : node2.childNodes.length;
  }
  function isIgnorable(dom, dir) {
    let desc = dom.pmViewDesc;
    return desc && desc.size == 0 && (dir < 0 || dom.nextSibling || dom.nodeName != "BR");
  }
  function skipIgnoredNodes(view, dir) {
    return dir < 0 ? skipIgnoredNodesBefore(view) : skipIgnoredNodesAfter(view);
  }
  function skipIgnoredNodesBefore(view) {
    let sel = view.domSelectionRange();
    let node2 = sel.focusNode, offset = sel.focusOffset;
    if (!node2)
      return;
    let moveNode, moveOffset, force = false;
    if (gecko2 && node2.nodeType == 1 && offset < nodeLen(node2) && isIgnorable(node2.childNodes[offset], -1))
      force = true;
    for (; ; ) {
      if (offset > 0) {
        if (node2.nodeType != 1) {
          break;
        } else {
          let before = node2.childNodes[offset - 1];
          if (isIgnorable(before, -1)) {
            moveNode = node2;
            moveOffset = --offset;
          } else if (before.nodeType == 3) {
            node2 = before;
            offset = node2.nodeValue.length;
          } else
            break;
        }
      } else if (isBlockNode(node2)) {
        break;
      } else {
        let prev = node2.previousSibling;
        while (prev && isIgnorable(prev, -1)) {
          moveNode = node2.parentNode;
          moveOffset = domIndex(prev);
          prev = prev.previousSibling;
        }
        if (!prev) {
          node2 = node2.parentNode;
          if (node2 == view.dom)
            break;
          offset = 0;
        } else {
          node2 = prev;
          offset = nodeLen(node2);
        }
      }
    }
    if (force)
      setSelFocus(view, node2, offset);
    else if (moveNode)
      setSelFocus(view, moveNode, moveOffset);
  }
  function skipIgnoredNodesAfter(view) {
    let sel = view.domSelectionRange();
    let node2 = sel.focusNode, offset = sel.focusOffset;
    if (!node2)
      return;
    let len = nodeLen(node2);
    let moveNode, moveOffset;
    for (; ; ) {
      if (offset < len) {
        if (node2.nodeType != 1)
          break;
        let after = node2.childNodes[offset];
        if (isIgnorable(after, 1)) {
          moveNode = node2;
          moveOffset = ++offset;
        } else
          break;
      } else if (isBlockNode(node2)) {
        break;
      } else {
        let next = node2.nextSibling;
        while (next && isIgnorable(next, 1)) {
          moveNode = next.parentNode;
          moveOffset = domIndex(next) + 1;
          next = next.nextSibling;
        }
        if (!next) {
          node2 = node2.parentNode;
          if (node2 == view.dom)
            break;
          offset = len = 0;
        } else {
          node2 = next;
          offset = 0;
          len = nodeLen(node2);
        }
      }
    }
    if (moveNode)
      setSelFocus(view, moveNode, moveOffset);
  }
  function isBlockNode(dom) {
    let desc = dom.pmViewDesc;
    return desc && desc.node && desc.node.isBlock;
  }
  function textNodeAfter(node2, offset) {
    while (node2 && offset == node2.childNodes.length && !hasBlockDesc(node2)) {
      offset = domIndex(node2) + 1;
      node2 = node2.parentNode;
    }
    while (node2 && offset < node2.childNodes.length) {
      let next = node2.childNodes[offset];
      if (next.nodeType == 3)
        return next;
      if (next.nodeType == 1 && next.contentEditable == "false")
        break;
      node2 = next;
      offset = 0;
    }
  }
  function textNodeBefore(node2, offset) {
    while (node2 && !offset && !hasBlockDesc(node2)) {
      offset = domIndex(node2);
      node2 = node2.parentNode;
    }
    while (node2 && offset) {
      let next = node2.childNodes[offset - 1];
      if (next.nodeType == 3)
        return next;
      if (next.nodeType == 1 && next.contentEditable == "false")
        break;
      node2 = next;
      offset = node2.childNodes.length;
    }
  }
  function setSelFocus(view, node2, offset) {
    if (node2.nodeType != 3) {
      let before, after;
      if (after = textNodeAfter(node2, offset)) {
        node2 = after;
        offset = 0;
      } else if (before = textNodeBefore(node2, offset)) {
        node2 = before;
        offset = before.nodeValue.length;
      }
    }
    let sel = view.domSelection();
    if (!sel)
      return;
    if (selectionCollapsed(sel)) {
      let range = document.createRange();
      range.setEnd(node2, offset);
      range.setStart(node2, offset);
      sel.removeAllRanges();
      sel.addRange(range);
    } else if (sel.extend) {
      sel.extend(node2, offset);
    }
    view.domObserver.setCurSelection();
    let { state } = view;
    setTimeout(() => {
      if (view.state == state)
        selectionToDOM(view);
    }, 50);
  }
  function findDirection(view, pos) {
    let $pos = view.state.doc.resolve(pos);
    if (!(chrome || windows2) && $pos.parent.inlineContent) {
      let coords = view.coordsAtPos(pos);
      if (pos > $pos.start()) {
        let before = view.coordsAtPos(pos - 1);
        let mid = (before.top + before.bottom) / 2;
        if (mid > coords.top && mid < coords.bottom && Math.abs(before.left - coords.left) > 1)
          return before.left < coords.left ? "ltr" : "rtl";
      }
      if (pos < $pos.end()) {
        let after = view.coordsAtPos(pos + 1);
        let mid = (after.top + after.bottom) / 2;
        if (mid > coords.top && mid < coords.bottom && Math.abs(after.left - coords.left) > 1)
          return after.left > coords.left ? "ltr" : "rtl";
      }
    }
    let computed = getComputedStyle(view.dom).direction;
    return computed == "rtl" ? "rtl" : "ltr";
  }
  function selectVertically(view, dir, mods) {
    let sel = view.state.selection;
    if (sel instanceof TextSelection && !sel.empty || mods.indexOf("s") > -1)
      return false;
    if (mac5 && mods.indexOf("m") > -1)
      return false;
    let { $from, $to } = sel;
    if (!$from.parent.inlineContent || view.endOfTextblock(dir < 0 ? "up" : "down")) {
      let next = moveSelectionBlock(view.state, dir);
      if (next && next instanceof NodeSelection)
        return apply(view, next);
    }
    if (!$from.parent.inlineContent) {
      let side = dir < 0 ? $from : $to;
      let beyond = sel instanceof AllSelection ? Selection.near(side, dir) : Selection.findFrom(side, dir);
      return beyond ? apply(view, beyond) : false;
    }
    return false;
  }
  function stopNativeHorizontalDelete(view, dir) {
    if (!(view.state.selection instanceof TextSelection))
      return true;
    let { $head, $anchor, empty: empty3 } = view.state.selection;
    if (!$head.sameParent($anchor))
      return true;
    if (!empty3)
      return false;
    if (view.endOfTextblock(dir > 0 ? "forward" : "backward"))
      return true;
    let nextNode = !$head.textOffset && (dir < 0 ? $head.nodeBefore : $head.nodeAfter);
    if (nextNode && !nextNode.isText) {
      let tr = view.state.tr;
      if (dir < 0)
        tr.delete($head.pos - nextNode.nodeSize, $head.pos);
      else
        tr.delete($head.pos, $head.pos + nextNode.nodeSize);
      view.dispatch(tr);
      return true;
    }
    return false;
  }
  function switchEditable(view, node2, state) {
    view.domObserver.stop();
    node2.contentEditable = state;
    view.domObserver.start();
  }
  function safariDownArrowBug(view) {
    if (!safari2 || view.state.selection.$head.parentOffset > 0)
      return false;
    let { focusNode, focusOffset } = view.domSelectionRange();
    if (focusNode && focusNode.nodeType == 1 && focusOffset == 0 && focusNode.firstChild && focusNode.firstChild.contentEditable == "false") {
      let child = focusNode.firstChild;
      switchEditable(view, child, "true");
      setTimeout(() => switchEditable(view, child, "false"), 20);
    }
    return false;
  }
  function getMods(event) {
    let result = "";
    if (event.ctrlKey)
      result += "c";
    if (event.metaKey)
      result += "m";
    if (event.altKey)
      result += "a";
    if (event.shiftKey)
      result += "s";
    return result;
  }
  function captureKeyDown(view, event) {
    let code2 = event.keyCode, mods = getMods(event);
    if (code2 == 8 || mac5 && code2 == 72 && mods == "c") {
      return stopNativeHorizontalDelete(view, -1) || skipIgnoredNodes(view, -1);
    } else if (code2 == 46 && !event.shiftKey || mac5 && code2 == 68 && mods == "c") {
      return stopNativeHorizontalDelete(view, 1) || skipIgnoredNodes(view, 1);
    } else if (code2 == 13 || code2 == 27) {
      return true;
    } else if (code2 == 37 || mac5 && code2 == 66 && mods == "c") {
      let dir = code2 == 37 ? findDirection(view, view.state.selection.from) == "ltr" ? -1 : 1 : -1;
      return selectHorizontally(view, dir, mods) || skipIgnoredNodes(view, dir);
    } else if (code2 == 39 || mac5 && code2 == 70 && mods == "c") {
      let dir = code2 == 39 ? findDirection(view, view.state.selection.from) == "ltr" ? 1 : -1 : 1;
      return selectHorizontally(view, dir, mods) || skipIgnoredNodes(view, dir);
    } else if (code2 == 38 || mac5 && code2 == 80 && mods == "c") {
      return selectVertically(view, -1, mods) || skipIgnoredNodes(view, -1);
    } else if (code2 == 40 || mac5 && code2 == 78 && mods == "c") {
      return safariDownArrowBug(view) || selectVertically(view, 1, mods) || skipIgnoredNodes(view, 1);
    } else if (mods == (mac5 ? "m" : "c") && (code2 == 66 || code2 == 73 || code2 == 89 || code2 == 90)) {
      return true;
    }
    return false;
  }
  function serializeForClipboard(view, slice) {
    view.someProp("transformCopied", (f2) => {
      slice = f2(slice, view);
    });
    let context = [], { content: content3, openStart, openEnd } = slice;
    while (openStart > 1 && openEnd > 1 && content3.childCount == 1 && content3.firstChild.childCount == 1) {
      openStart--;
      openEnd--;
      let node2 = content3.firstChild;
      context.push(node2.type.name, node2.attrs != node2.type.defaultAttrs ? node2.attrs : null);
      content3 = node2.content;
    }
    let serializer2 = view.someProp("clipboardSerializer") || DOMSerializer.fromSchema(view.state.schema);
    let doc4 = detachedDoc(), wrap3 = doc4.createElement("div");
    wrap3.appendChild(serializer2.serializeFragment(content3, { document: doc4 }));
    let firstChild = wrap3.firstChild, needsWrap, wrappers = 0;
    while (firstChild && firstChild.nodeType == 1 && (needsWrap = wrapMap[firstChild.nodeName.toLowerCase()])) {
      for (let i2 = needsWrap.length - 1; i2 >= 0; i2--) {
        let wrapper = doc4.createElement(needsWrap[i2]);
        while (wrap3.firstChild)
          wrapper.appendChild(wrap3.firstChild);
        wrap3.appendChild(wrapper);
        wrappers++;
      }
      firstChild = wrap3.firstChild;
    }
    if (firstChild && firstChild.nodeType == 1)
      firstChild.setAttribute("data-pm-slice", `${openStart} ${openEnd}${wrappers ? ` -${wrappers}` : ""} ${JSON.stringify(context)}`);
    let text4 = view.someProp("clipboardTextSerializer", (f2) => f2(slice, view)) || slice.content.textBetween(0, slice.content.size, "\n\n");
    return { dom: wrap3, text: text4, slice };
  }
  function parseFromClipboard(view, text4, html2, plainText, $context) {
    let inCode = $context.parent.type.spec.code;
    let dom, slice;
    if (!html2 && !text4)
      return null;
    let asText = !!text4 && (plainText || inCode || !html2);
    if (asText) {
      view.someProp("transformPastedText", (f2) => {
        text4 = f2(text4, inCode || plainText, view);
      });
      if (inCode) {
        slice = new Slice2(Fragment.from(view.state.schema.text(text4.replace(/\r\n?/g, "\n"))), 0, 0);
        view.someProp("transformPasted", (f2) => {
          slice = f2(slice, view, true);
        });
        return slice;
      }
      let parsed = view.someProp("clipboardTextParser", (f2) => f2(text4, $context, plainText, view));
      if (parsed) {
        slice = parsed;
      } else {
        let marks = $context.marks();
        let { schema: schema3 } = view.state, serializer2 = DOMSerializer.fromSchema(schema3);
        dom = document.createElement("div");
        text4.split(/(?:\r\n?|\n)+/).forEach((block) => {
          let p2 = dom.appendChild(document.createElement("p"));
          if (block)
            p2.appendChild(serializer2.serializeNode(schema3.text(block, marks)));
        });
      }
    } else {
      view.someProp("transformPastedHTML", (f2) => {
        html2 = f2(html2, view);
      });
      dom = readHTML(html2);
      if (webkit2)
        restoreReplacedSpaces(dom);
    }
    let contextNode = dom && dom.querySelector("[data-pm-slice]");
    let sliceData = contextNode && /^(\d+) (\d+)(?: -(\d+))? (.*)/.exec(contextNode.getAttribute("data-pm-slice") || "");
    if (sliceData && sliceData[3])
      for (let i2 = +sliceData[3]; i2 > 0; i2--) {
        let child = dom.firstChild;
        while (child && child.nodeType != 1)
          child = child.nextSibling;
        if (!child)
          break;
        dom = child;
      }
    if (!slice) {
      let parser2 = view.someProp("clipboardParser") || view.someProp("domParser") || DOMParser.fromSchema(view.state.schema);
      slice = parser2.parseSlice(dom, {
        preserveWhitespace: !!(asText || sliceData),
        context: $context,
        ruleFromNode(dom2) {
          if (dom2.nodeName == "BR" && !dom2.nextSibling && dom2.parentNode && !inlineParents.test(dom2.parentNode.nodeName))
            return { ignore: true };
          return null;
        }
      });
    }
    if (sliceData) {
      slice = addContext(closeSlice(slice, +sliceData[1], +sliceData[2]), sliceData[4]);
    } else {
      slice = Slice2.maxOpen(normalizeSiblings(slice.content, $context), true);
      if (slice.openStart || slice.openEnd) {
        let openStart = 0, openEnd = 0;
        for (let node2 = slice.content.firstChild; openStart < slice.openStart && !node2.type.spec.isolating; openStart++, node2 = node2.firstChild) {
        }
        for (let node2 = slice.content.lastChild; openEnd < slice.openEnd && !node2.type.spec.isolating; openEnd++, node2 = node2.lastChild) {
        }
        slice = closeSlice(slice, openStart, openEnd);
      }
    }
    view.someProp("transformPasted", (f2) => {
      slice = f2(slice, view, asText);
    });
    return slice;
  }
  var inlineParents = /^(a|abbr|acronym|b|cite|code|del|em|i|ins|kbd|label|output|q|ruby|s|samp|span|strong|sub|sup|time|u|tt|var)$/i;
  function normalizeSiblings(fragment, $context) {
    if (fragment.childCount < 2)
      return fragment;
    for (let d2 = $context.depth; d2 >= 0; d2--) {
      let parent = $context.node(d2);
      let match = parent.contentMatchAt($context.index(d2));
      let lastWrap, result = [];
      fragment.forEach((node2) => {
        if (!result)
          return;
        let wrap3 = match.findWrapping(node2.type), inLast;
        if (!wrap3)
          return result = null;
        if (inLast = result.length && lastWrap.length && addToSibling(wrap3, lastWrap, node2, result[result.length - 1], 0)) {
          result[result.length - 1] = inLast;
        } else {
          if (result.length)
            result[result.length - 1] = closeRight(result[result.length - 1], lastWrap.length);
          let wrapped = withWrappers(node2, wrap3);
          result.push(wrapped);
          match = match.matchType(wrapped.type);
          lastWrap = wrap3;
        }
      });
      if (result)
        return Fragment.from(result);
    }
    return fragment;
  }
  function withWrappers(node2, wrap3, from = 0) {
    for (let i2 = wrap3.length - 1; i2 >= from; i2--)
      node2 = wrap3[i2].create(null, Fragment.from(node2));
    return node2;
  }
  function addToSibling(wrap3, lastWrap, node2, sibling, depth) {
    if (depth < wrap3.length && depth < lastWrap.length && wrap3[depth] == lastWrap[depth]) {
      let inner = addToSibling(wrap3, lastWrap, node2, sibling.lastChild, depth + 1);
      if (inner)
        return sibling.copy(sibling.content.replaceChild(sibling.childCount - 1, inner));
      let match = sibling.contentMatchAt(sibling.childCount);
      if (match.matchType(depth == wrap3.length - 1 ? node2.type : wrap3[depth + 1]))
        return sibling.copy(sibling.content.append(Fragment.from(withWrappers(node2, wrap3, depth + 1))));
    }
  }
  function closeRight(node2, depth) {
    if (depth == 0)
      return node2;
    let fragment = node2.content.replaceChild(node2.childCount - 1, closeRight(node2.lastChild, depth - 1));
    let fill = node2.contentMatchAt(node2.childCount).fillBefore(Fragment.empty, true);
    return node2.copy(fragment.append(fill));
  }
  function closeRange(fragment, side, from, to, depth, openEnd) {
    let node2 = side < 0 ? fragment.firstChild : fragment.lastChild, inner = node2.content;
    if (fragment.childCount > 1)
      openEnd = 0;
    if (depth < to - 1)
      inner = closeRange(inner, side, from, to, depth + 1, openEnd);
    if (depth >= from)
      inner = side < 0 ? node2.contentMatchAt(0).fillBefore(inner, openEnd <= depth).append(inner) : inner.append(node2.contentMatchAt(node2.childCount).fillBefore(Fragment.empty, true));
    return fragment.replaceChild(side < 0 ? 0 : fragment.childCount - 1, node2.copy(inner));
  }
  function closeSlice(slice, openStart, openEnd) {
    if (openStart < slice.openStart)
      slice = new Slice2(closeRange(slice.content, -1, openStart, slice.openStart, 0, slice.openEnd), openStart, slice.openEnd);
    if (openEnd < slice.openEnd)
      slice = new Slice2(closeRange(slice.content, 1, openEnd, slice.openEnd, 0, 0), slice.openStart, openEnd);
    return slice;
  }
  var wrapMap = {
    thead: ["table"],
    tbody: ["table"],
    tfoot: ["table"],
    caption: ["table"],
    colgroup: ["table"],
    col: ["table", "colgroup"],
    tr: ["table", "tbody"],
    td: ["table", "tbody", "tr"],
    th: ["table", "tbody", "tr"]
  };
  var _detachedDoc = null;
  function detachedDoc() {
    return _detachedDoc || (_detachedDoc = document.implementation.createHTMLDocument("title"));
  }
  var _policy = null;
  function maybeWrapTrusted(html2) {
    let trustedTypes = window.trustedTypes;
    if (!trustedTypes)
      return html2;
    if (!_policy)
      _policy = trustedTypes.defaultPolicy || trustedTypes.createPolicy("ProseMirrorClipboard", { createHTML: (s2) => s2 });
    return _policy.createHTML(html2);
  }
  function readHTML(html2) {
    let metas = /^(\s*<meta [^>]*>)*/.exec(html2);
    if (metas)
      html2 = html2.slice(metas[0].length);
    let elt = detachedDoc().createElement("div");
    let firstTag = /<([a-z][^>\s]+)/i.exec(html2), wrap3;
    if (wrap3 = firstTag && wrapMap[firstTag[1].toLowerCase()])
      html2 = wrap3.map((n2) => "<" + n2 + ">").join("") + html2 + wrap3.map((n2) => "</" + n2 + ">").reverse().join("");
    elt.innerHTML = maybeWrapTrusted(html2);
    if (wrap3)
      for (let i2 = 0; i2 < wrap3.length; i2++)
        elt = elt.querySelector(wrap3[i2]) || elt;
    return elt;
  }
  function restoreReplacedSpaces(dom) {
    let nodes = dom.querySelectorAll(chrome ? "span:not([class]):not([style])" : "span.Apple-converted-space");
    for (let i2 = 0; i2 < nodes.length; i2++) {
      let node2 = nodes[i2];
      if (node2.childNodes.length == 1 && node2.textContent == "\xA0" && node2.parentNode)
        node2.parentNode.replaceChild(dom.ownerDocument.createTextNode(" "), node2);
    }
  }
  function addContext(slice, context) {
    if (!slice.size)
      return slice;
    let schema3 = slice.content.firstChild.type.schema, array;
    try {
      array = JSON.parse(context);
    } catch (e2) {
      return slice;
    }
    let { content: content3, openStart, openEnd } = slice;
    for (let i2 = array.length - 2; i2 >= 0; i2 -= 2) {
      let type = schema3.nodes[array[i2]];
      if (!type || type.hasRequiredAttrs())
        break;
      content3 = Fragment.from(type.create(array[i2 + 1], content3));
      openStart++;
      openEnd++;
    }
    return new Slice2(content3, openStart, openEnd);
  }
  var handlers = {};
  var editHandlers = {};
  var passiveHandlers = { touchstart: true, touchmove: true };
  var InputState = class {
    constructor() {
      this.shiftKey = false;
      this.mouseDown = null;
      this.lastKeyCode = null;
      this.lastKeyCodeTime = 0;
      this.lastClick = { time: 0, x: 0, y: 0, type: "", button: 0 };
      this.lastSelectionOrigin = null;
      this.lastSelectionTime = 0;
      this.lastIOSEnter = 0;
      this.lastIOSEnterFallbackTimeout = -1;
      this.lastFocus = 0;
      this.lastTouch = 0;
      this.lastChromeDelete = 0;
      this.composing = false;
      this.compositionNode = null;
      this.composingTimeout = -1;
      this.compositionNodes = [];
      this.compositionEndedAt = -2e8;
      this.compositionID = 1;
      this.badSafariComposition = false;
      this.compositionPendingChanges = 0;
      this.domChangeCount = 0;
      this.eventHandlers = /* @__PURE__ */ Object.create(null);
      this.hideSelectionGuard = null;
    }
  };
  function initInput(view) {
    for (let event in handlers) {
      let handler = handlers[event];
      view.dom.addEventListener(event, view.input.eventHandlers[event] = (event2) => {
        if (eventBelongsToView(view, event2) && !runCustomHandler(view, event2) && (view.editable || !(event2.type in editHandlers)))
          handler(view, event2);
      }, passiveHandlers[event] ? { passive: true } : void 0);
    }
    if (safari2)
      view.dom.addEventListener("input", () => null);
    ensureListeners(view);
  }
  function setSelectionOrigin(view, origin) {
    view.input.lastSelectionOrigin = origin;
    view.input.lastSelectionTime = Date.now();
  }
  function destroyInput(view) {
    view.domObserver.stop();
    for (let type in view.input.eventHandlers)
      view.dom.removeEventListener(type, view.input.eventHandlers[type]);
    clearTimeout(view.input.composingTimeout);
    clearTimeout(view.input.lastIOSEnterFallbackTimeout);
  }
  function ensureListeners(view) {
    view.someProp("handleDOMEvents", (currentHandlers) => {
      for (let type in currentHandlers)
        if (!view.input.eventHandlers[type])
          view.dom.addEventListener(type, view.input.eventHandlers[type] = (event) => runCustomHandler(view, event));
    });
  }
  function runCustomHandler(view, event) {
    return view.someProp("handleDOMEvents", (handlers2) => {
      let handler = handlers2[event.type];
      return handler ? handler(view, event) || event.defaultPrevented : false;
    });
  }
  function eventBelongsToView(view, event) {
    if (!event.bubbles)
      return true;
    if (event.defaultPrevented)
      return false;
    for (let node2 = event.target; node2 != view.dom; node2 = node2.parentNode)
      if (!node2 || node2.nodeType == 11 || node2.pmViewDesc && node2.pmViewDesc.stopEvent(event))
        return false;
    return true;
  }
  function dispatchEvent2(view, event) {
    if (!runCustomHandler(view, event) && handlers[event.type] && (view.editable || !(event.type in editHandlers)))
      handlers[event.type](view, event);
  }
  editHandlers.keydown = (view, _event) => {
    let event = _event;
    view.input.shiftKey = event.keyCode == 16 || event.shiftKey;
    if (inOrNearComposition(view, event))
      return;
    view.input.lastKeyCode = event.keyCode;
    view.input.lastKeyCodeTime = Date.now();
    if (android2 && chrome && event.keyCode == 13)
      return;
    if (event.keyCode != 229)
      view.domObserver.forceFlush();
    if (ios2 && event.keyCode == 13 && !event.ctrlKey && !event.altKey && !event.metaKey) {
      let now2 = Date.now();
      view.input.lastIOSEnter = now2;
      view.input.lastIOSEnterFallbackTimeout = setTimeout(() => {
        if (view.input.lastIOSEnter == now2) {
          view.someProp("handleKeyDown", (f2) => f2(view, keyEvent(13, "Enter")));
          view.input.lastIOSEnter = 0;
        }
      }, 200);
    } else if (view.someProp("handleKeyDown", (f2) => f2(view, event)) || captureKeyDown(view, event)) {
      event.preventDefault();
    } else {
      setSelectionOrigin(view, "key");
    }
  };
  editHandlers.keyup = (view, event) => {
    if (event.keyCode == 16)
      view.input.shiftKey = false;
  };
  editHandlers.keypress = (view, _event) => {
    let event = _event;
    if (inOrNearComposition(view, event) || !event.charCode || event.ctrlKey && !event.altKey || mac5 && event.metaKey)
      return;
    if (view.someProp("handleKeyPress", (f2) => f2(view, event))) {
      event.preventDefault();
      return;
    }
    let sel = view.state.selection;
    if (!(sel instanceof TextSelection) || !sel.$from.sameParent(sel.$to)) {
      let text4 = String.fromCharCode(event.charCode);
      let deflt = () => view.state.tr.insertText(text4).scrollIntoView();
      if (!/[\r\n]/.test(text4) && !view.someProp("handleTextInput", (f2) => f2(view, sel.$from.pos, sel.$to.pos, text4, deflt)))
        view.dispatch(deflt());
      event.preventDefault();
    }
  };
  function eventCoords(event) {
    return { left: event.clientX, top: event.clientY };
  }
  function isNear(event, click) {
    let dx = click.x - event.clientX, dy = click.y - event.clientY;
    return dx * dx + dy * dy < 100;
  }
  function runHandlerOnContext(view, propName, pos, inside, event) {
    if (inside == -1)
      return false;
    let $pos = view.state.doc.resolve(inside);
    for (let i2 = $pos.depth + 1; i2 > 0; i2--) {
      if (view.someProp(propName, (f2) => i2 > $pos.depth ? f2(view, pos, $pos.nodeAfter, $pos.before(i2), event, true) : f2(view, pos, $pos.node(i2), $pos.before(i2), event, false)))
        return true;
    }
    return false;
  }
  function updateSelection(view, selection, origin) {
    if (!view.focused)
      view.focus();
    if (view.state.selection.eq(selection))
      return;
    let tr = view.state.tr.setSelection(selection);
    if (origin == "pointer")
      tr.setMeta("pointer", true);
    view.dispatch(tr);
  }
  function selectClickedLeaf(view, inside) {
    if (inside == -1)
      return false;
    let $pos = view.state.doc.resolve(inside), node2 = $pos.nodeAfter;
    if (node2 && node2.isAtom && NodeSelection.isSelectable(node2)) {
      updateSelection(view, new NodeSelection($pos), "pointer");
      return true;
    }
    return false;
  }
  function selectClickedNode(view, inside) {
    if (inside == -1)
      return false;
    let sel = view.state.selection, selectedNode, selectAt;
    if (sel instanceof NodeSelection)
      selectedNode = sel.node;
    let $pos = view.state.doc.resolve(inside);
    for (let i2 = $pos.depth + 1; i2 > 0; i2--) {
      let node2 = i2 > $pos.depth ? $pos.nodeAfter : $pos.node(i2);
      if (NodeSelection.isSelectable(node2)) {
        if (selectedNode && sel.$from.depth > 0 && i2 >= sel.$from.depth && $pos.before(sel.$from.depth + 1) == sel.$from.pos)
          selectAt = $pos.before(sel.$from.depth);
        else
          selectAt = $pos.before(i2);
        break;
      }
    }
    if (selectAt != null) {
      updateSelection(view, NodeSelection.create(view.state.doc, selectAt), "pointer");
      return true;
    } else {
      return false;
    }
  }
  function handleSingleClick(view, pos, inside, event, selectNode) {
    return runHandlerOnContext(view, "handleClickOn", pos, inside, event) || view.someProp("handleClick", (f2) => f2(view, pos, event)) || (selectNode ? selectClickedNode(view, inside) : selectClickedLeaf(view, inside));
  }
  function handleDoubleClick(view, pos, inside, event) {
    return runHandlerOnContext(view, "handleDoubleClickOn", pos, inside, event) || view.someProp("handleDoubleClick", (f2) => f2(view, pos, event));
  }
  function handleTripleClick(view, pos, inside, event) {
    return runHandlerOnContext(view, "handleTripleClickOn", pos, inside, event) || view.someProp("handleTripleClick", (f2) => f2(view, pos, event)) || defaultTripleClick(view, inside, event);
  }
  function defaultTripleClick(view, inside, event) {
    if (event.button != 0)
      return false;
    let doc4 = view.state.doc;
    if (inside == -1) {
      if (doc4.inlineContent) {
        updateSelection(view, TextSelection.create(doc4, 0, doc4.content.size), "pointer");
        return true;
      }
      return false;
    }
    let $pos = doc4.resolve(inside);
    for (let i2 = $pos.depth + 1; i2 > 0; i2--) {
      let node2 = i2 > $pos.depth ? $pos.nodeAfter : $pos.node(i2);
      let nodePos = $pos.before(i2);
      if (node2.inlineContent)
        updateSelection(view, TextSelection.create(doc4, nodePos + 1, nodePos + 1 + node2.content.size), "pointer");
      else if (NodeSelection.isSelectable(node2))
        updateSelection(view, NodeSelection.create(doc4, nodePos), "pointer");
      else
        continue;
      return true;
    }
  }
  function forceDOMFlush(view) {
    return endComposition(view);
  }
  var selectNodeModifier = mac5 ? "metaKey" : "ctrlKey";
  handlers.mousedown = (view, _event) => {
    let event = _event;
    view.input.shiftKey = event.shiftKey;
    let flushed = forceDOMFlush(view);
    let now2 = Date.now(), type = "singleClick";
    if (now2 - view.input.lastClick.time < 500 && isNear(event, view.input.lastClick) && !event[selectNodeModifier] && view.input.lastClick.button == event.button) {
      if (view.input.lastClick.type == "singleClick")
        type = "doubleClick";
      else if (view.input.lastClick.type == "doubleClick")
        type = "tripleClick";
    }
    view.input.lastClick = { time: now2, x: event.clientX, y: event.clientY, type, button: event.button };
    let pos = view.posAtCoords(eventCoords(event));
    if (!pos)
      return;
    if (type == "singleClick") {
      if (view.input.mouseDown)
        view.input.mouseDown.done();
      view.input.mouseDown = new MouseDown(view, pos, event, !!flushed);
    } else if ((type == "doubleClick" ? handleDoubleClick : handleTripleClick)(view, pos.pos, pos.inside, event)) {
      event.preventDefault();
    } else {
      setSelectionOrigin(view, "pointer");
    }
  };
  var MouseDown = class {
    constructor(view, pos, event, flushed) {
      this.view = view;
      this.pos = pos;
      this.event = event;
      this.flushed = flushed;
      this.delayedSelectionSync = false;
      this.mightDrag = null;
      this.startDoc = view.state.doc;
      this.selectNode = !!event[selectNodeModifier];
      this.allowDefault = event.shiftKey;
      let targetNode, targetPos;
      if (pos.inside > -1) {
        targetNode = view.state.doc.nodeAt(pos.inside);
        targetPos = pos.inside;
      } else {
        let $pos = view.state.doc.resolve(pos.pos);
        targetNode = $pos.parent;
        targetPos = $pos.depth ? $pos.before() : 0;
      }
      const target = flushed ? null : event.target;
      const targetDesc = target ? view.docView.nearestDesc(target, true) : null;
      this.target = targetDesc && targetDesc.nodeDOM.nodeType == 1 ? targetDesc.nodeDOM : null;
      let { selection } = view.state;
      if (event.button == 0 && targetNode.type.spec.draggable && targetNode.type.spec.selectable !== false || selection instanceof NodeSelection && selection.from <= targetPos && selection.to > targetPos)
        this.mightDrag = {
          node: targetNode,
          pos: targetPos,
          addAttr: !!(this.target && !this.target.draggable),
          setUneditable: !!(this.target && gecko2 && !this.target.hasAttribute("contentEditable"))
        };
      if (this.target && this.mightDrag && (this.mightDrag.addAttr || this.mightDrag.setUneditable)) {
        this.view.domObserver.stop();
        if (this.mightDrag.addAttr)
          this.target.draggable = true;
        if (this.mightDrag.setUneditable)
          setTimeout(() => {
            if (this.view.input.mouseDown == this)
              this.target.setAttribute("contentEditable", "false");
          }, 20);
        this.view.domObserver.start();
      }
      view.root.addEventListener("mouseup", this.up = this.up.bind(this));
      view.root.addEventListener("mousemove", this.move = this.move.bind(this));
      setSelectionOrigin(view, "pointer");
    }
    done() {
      this.view.root.removeEventListener("mouseup", this.up);
      this.view.root.removeEventListener("mousemove", this.move);
      if (this.mightDrag && this.target) {
        this.view.domObserver.stop();
        if (this.mightDrag.addAttr)
          this.target.removeAttribute("draggable");
        if (this.mightDrag.setUneditable)
          this.target.removeAttribute("contentEditable");
        this.view.domObserver.start();
      }
      if (this.delayedSelectionSync)
        setTimeout(() => selectionToDOM(this.view));
      this.view.input.mouseDown = null;
    }
    up(event) {
      this.done();
      if (!this.view.dom.contains(event.target))
        return;
      let pos = this.pos;
      if (this.view.state.doc != this.startDoc)
        pos = this.view.posAtCoords(eventCoords(event));
      this.updateAllowDefault(event);
      if (this.allowDefault || !pos) {
        setSelectionOrigin(this.view, "pointer");
      } else if (handleSingleClick(this.view, pos.pos, pos.inside, event, this.selectNode)) {
        event.preventDefault();
      } else if (event.button == 0 && (this.flushed || // Safari ignores clicks on draggable elements
      safari2 && this.mightDrag && !this.mightDrag.node.isAtom || // Chrome will sometimes treat a node selection as a
      // cursor, but still report that the node is selected
      // when asked through getSelection. You'll then get a
      // situation where clicking at the point where that
      // (hidden) cursor is doesn't change the selection, and
      // thus doesn't get a reaction from ProseMirror. This
      // works around that.
      chrome && !this.view.state.selection.visible && Math.min(Math.abs(pos.pos - this.view.state.selection.from), Math.abs(pos.pos - this.view.state.selection.to)) <= 2)) {
        updateSelection(this.view, Selection.near(this.view.state.doc.resolve(pos.pos)), "pointer");
        event.preventDefault();
      } else {
        setSelectionOrigin(this.view, "pointer");
      }
    }
    move(event) {
      this.updateAllowDefault(event);
      setSelectionOrigin(this.view, "pointer");
      if (event.buttons == 0)
        this.done();
    }
    updateAllowDefault(event) {
      if (!this.allowDefault && (Math.abs(this.event.x - event.clientX) > 4 || Math.abs(this.event.y - event.clientY) > 4))
        this.allowDefault = true;
    }
  };
  handlers.touchstart = (view) => {
    view.input.lastTouch = Date.now();
    forceDOMFlush(view);
    setSelectionOrigin(view, "pointer");
  };
  handlers.touchmove = (view) => {
    view.input.lastTouch = Date.now();
    setSelectionOrigin(view, "pointer");
  };
  handlers.contextmenu = (view) => forceDOMFlush(view);
  function inOrNearComposition(view, event) {
    if (view.composing)
      return true;
    if (safari2 && Math.abs(event.timeStamp - view.input.compositionEndedAt) < 500) {
      view.input.compositionEndedAt = -2e8;
      return true;
    }
    return false;
  }
  var timeoutComposition = android2 ? 5e3 : -1;
  editHandlers.compositionstart = editHandlers.compositionupdate = (view) => {
    if (!view.composing) {
      view.domObserver.flush();
      let { state } = view, $pos = state.selection.$to;
      if (state.selection instanceof TextSelection && (state.storedMarks || !$pos.textOffset && $pos.parentOffset && $pos.nodeBefore.marks.some((m) => m.type.spec.inclusive === false) || chrome && windows2 && selectionBeforeUneditable(view))) {
        view.markCursor = view.state.storedMarks || $pos.marks();
        endComposition(view, true);
        view.markCursor = null;
      } else {
        endComposition(view, !state.selection.empty);
        if (gecko2 && state.selection.empty && $pos.parentOffset && !$pos.textOffset && $pos.nodeBefore.marks.length) {
          let sel = view.domSelectionRange();
          for (let node2 = sel.focusNode, offset = sel.focusOffset; node2 && node2.nodeType == 1 && offset != 0; ) {
            let before = offset < 0 ? node2.lastChild : node2.childNodes[offset - 1];
            if (!before)
              break;
            if (before.nodeType == 3) {
              let sel2 = view.domSelection();
              if (sel2)
                sel2.collapse(before, before.nodeValue.length);
              break;
            } else {
              node2 = before;
              offset = -1;
            }
          }
        }
      }
      view.input.composing = true;
    }
    scheduleComposeEnd(view, timeoutComposition);
  };
  function selectionBeforeUneditable(view) {
    let { focusNode, focusOffset } = view.domSelectionRange();
    if (!focusNode || focusNode.nodeType != 1 || focusOffset >= focusNode.childNodes.length)
      return false;
    let next = focusNode.childNodes[focusOffset];
    return next.nodeType == 1 && next.contentEditable == "false";
  }
  editHandlers.compositionend = (view, event) => {
    if (view.composing) {
      view.input.composing = false;
      view.input.compositionEndedAt = event.timeStamp;
      view.input.compositionPendingChanges = view.domObserver.pendingRecords().length ? view.input.compositionID : 0;
      view.input.compositionNode = null;
      if (view.input.badSafariComposition)
        view.domObserver.forceFlush();
      else if (view.input.compositionPendingChanges)
        Promise.resolve().then(() => view.domObserver.flush());
      view.input.compositionID++;
      scheduleComposeEnd(view, 20);
    }
  };
  function scheduleComposeEnd(view, delay) {
    clearTimeout(view.input.composingTimeout);
    if (delay > -1)
      view.input.composingTimeout = setTimeout(() => endComposition(view), delay);
  }
  function clearComposition(view) {
    if (view.composing) {
      view.input.composing = false;
      view.input.compositionEndedAt = timestampFromCustomEvent();
    }
    while (view.input.compositionNodes.length > 0)
      view.input.compositionNodes.pop().markParentsDirty();
  }
  function findCompositionNode(view) {
    let sel = view.domSelectionRange();
    if (!sel.focusNode)
      return null;
    let textBefore = textNodeBefore$1(sel.focusNode, sel.focusOffset);
    let textAfter = textNodeAfter$1(sel.focusNode, sel.focusOffset);
    if (textBefore && textAfter && textBefore != textAfter) {
      let descAfter = textAfter.pmViewDesc, lastChanged = view.domObserver.lastChangedTextNode;
      if (textBefore == lastChanged || textAfter == lastChanged)
        return lastChanged;
      if (!descAfter || !descAfter.isText(textAfter.nodeValue)) {
        return textAfter;
      } else if (view.input.compositionNode == textAfter) {
        let descBefore = textBefore.pmViewDesc;
        if (!(!descBefore || !descBefore.isText(textBefore.nodeValue)))
          return textAfter;
      }
    }
    return textBefore || textAfter;
  }
  function timestampFromCustomEvent() {
    let event = document.createEvent("Event");
    event.initEvent("event", true, true);
    return event.timeStamp;
  }
  function endComposition(view, restarting = false) {
    if (android2 && view.domObserver.flushingSoon >= 0)
      return;
    view.domObserver.forceFlush();
    clearComposition(view);
    if (restarting || view.docView && view.docView.dirty) {
      let sel = selectionFromDOM(view), cur = view.state.selection;
      if (sel && !sel.eq(cur))
        view.dispatch(view.state.tr.setSelection(sel));
      else if ((view.markCursor || restarting) && !cur.$from.node(cur.$from.sharedDepth(cur.to)).inlineContent)
        view.dispatch(view.state.tr.deleteSelection());
      else
        view.updateState(view.state);
      return true;
    }
    return false;
  }
  function captureCopy(view, dom) {
    if (!view.dom.parentNode)
      return;
    let wrap3 = view.dom.parentNode.appendChild(document.createElement("div"));
    wrap3.appendChild(dom);
    wrap3.style.cssText = "position: fixed; left: -10000px; top: 10px";
    let sel = getSelection(), range = document.createRange();
    range.selectNodeContents(dom);
    view.dom.blur();
    sel.removeAllRanges();
    sel.addRange(range);
    setTimeout(() => {
      if (wrap3.parentNode)
        wrap3.parentNode.removeChild(wrap3);
      view.focus();
    }, 50);
  }
  var brokenClipboardAPI = ie3 && ie_version2 < 15 || ios2 && webkit_version2 < 604;
  handlers.copy = editHandlers.cut = (view, _event) => {
    let event = _event;
    let sel = view.state.selection, cut = event.type == "cut";
    if (sel.empty)
      return;
    let data = brokenClipboardAPI ? null : event.clipboardData;
    let slice = sel.content(), { dom, text: text4 } = serializeForClipboard(view, slice);
    if (data) {
      event.preventDefault();
      data.clearData();
      data.setData("text/html", dom.innerHTML);
      data.setData("text/plain", text4);
    } else {
      captureCopy(view, dom);
    }
    if (cut)
      view.dispatch(view.state.tr.deleteSelection().scrollIntoView().setMeta("uiEvent", "cut"));
  };
  function sliceSingleNode(slice) {
    return slice.openStart == 0 && slice.openEnd == 0 && slice.content.childCount == 1 ? slice.content.firstChild : null;
  }
  function capturePaste(view, event) {
    if (!view.dom.parentNode)
      return;
    let plainText = view.input.shiftKey || view.state.selection.$from.parent.type.spec.code;
    let target = view.dom.parentNode.appendChild(document.createElement(plainText ? "textarea" : "div"));
    if (!plainText)
      target.contentEditable = "true";
    target.style.cssText = "position: fixed; left: -10000px; top: 10px";
    target.focus();
    let plain = view.input.shiftKey && view.input.lastKeyCode != 45;
    setTimeout(() => {
      view.focus();
      if (target.parentNode)
        target.parentNode.removeChild(target);
      if (plainText)
        doPaste(view, target.value, null, plain, event);
      else
        doPaste(view, target.textContent, target.innerHTML, plain, event);
    }, 50);
  }
  function doPaste(view, text4, html2, preferPlain, event) {
    let slice = parseFromClipboard(view, text4, html2, preferPlain, view.state.selection.$from);
    if (view.someProp("handlePaste", (f2) => f2(view, event, slice || Slice2.empty)))
      return true;
    if (!slice)
      return false;
    let singleNode = sliceSingleNode(slice);
    let tr = singleNode ? view.state.tr.replaceSelectionWith(singleNode, preferPlain) : view.state.tr.replaceSelection(slice);
    view.dispatch(tr.scrollIntoView().setMeta("paste", true).setMeta("uiEvent", "paste"));
    return true;
  }
  function getText(clipboardData) {
    let text4 = clipboardData.getData("text/plain") || clipboardData.getData("Text");
    if (text4)
      return text4;
    let uris = clipboardData.getData("text/uri-list");
    return uris ? uris.replace(/\r?\n/g, " ") : "";
  }
  editHandlers.paste = (view, _event) => {
    let event = _event;
    if (view.composing && !android2)
      return;
    let data = brokenClipboardAPI ? null : event.clipboardData;
    let plain = view.input.shiftKey && view.input.lastKeyCode != 45;
    if (data && doPaste(view, getText(data), data.getData("text/html"), plain, event))
      event.preventDefault();
    else
      capturePaste(view, event);
  };
  var Dragging = class {
    constructor(slice, move, node2) {
      this.slice = slice;
      this.move = move;
      this.node = node2;
    }
  };
  var dragCopyModifier = mac5 ? "altKey" : "ctrlKey";
  function dragMoves(view, event) {
    let moves = view.someProp("dragCopies", (test) => !test(event));
    return moves != null ? moves : !event[dragCopyModifier];
  }
  handlers.dragstart = (view, _event) => {
    let event = _event;
    let mouseDown = view.input.mouseDown;
    if (mouseDown)
      mouseDown.done();
    if (!event.dataTransfer)
      return;
    let sel = view.state.selection;
    let pos = sel.empty ? null : view.posAtCoords(eventCoords(event));
    let node2;
    if (pos && pos.pos >= sel.from && pos.pos <= (sel instanceof NodeSelection ? sel.to - 1 : sel.to)) ;
    else if (mouseDown && mouseDown.mightDrag) {
      node2 = NodeSelection.create(view.state.doc, mouseDown.mightDrag.pos);
    } else if (event.target && event.target.nodeType == 1) {
      let desc = view.docView.nearestDesc(event.target, true);
      if (desc && desc.node.type.spec.draggable && desc != view.docView)
        node2 = NodeSelection.create(view.state.doc, desc.posBefore);
    }
    let draggedSlice = (node2 || view.state.selection).content();
    let { dom, text: text4, slice } = serializeForClipboard(view, draggedSlice);
    if (!event.dataTransfer.files.length || !chrome || chrome_version2 > 120)
      event.dataTransfer.clearData();
    event.dataTransfer.setData(brokenClipboardAPI ? "Text" : "text/html", dom.innerHTML);
    event.dataTransfer.effectAllowed = "copyMove";
    if (!brokenClipboardAPI)
      event.dataTransfer.setData("text/plain", text4);
    view.dragging = new Dragging(slice, dragMoves(view, event), node2);
  };
  handlers.dragend = (view) => {
    let dragging = view.dragging;
    window.setTimeout(() => {
      if (view.dragging == dragging)
        view.dragging = null;
    }, 50);
  };
  editHandlers.dragover = editHandlers.dragenter = (_, e2) => e2.preventDefault();
  editHandlers.drop = (view, event) => {
    try {
      handleDrop(view, event, view.dragging);
    } finally {
      view.dragging = null;
    }
  };
  function handleDrop(view, event, dragging) {
    if (!event.dataTransfer)
      return;
    let eventPos = view.posAtCoords(eventCoords(event));
    if (!eventPos)
      return;
    let $mouse = view.state.doc.resolve(eventPos.pos);
    let slice = dragging && dragging.slice;
    if (slice) {
      view.someProp("transformPasted", (f2) => {
        slice = f2(slice, view, false);
      });
    } else {
      slice = parseFromClipboard(view, getText(event.dataTransfer), brokenClipboardAPI ? null : event.dataTransfer.getData("text/html"), false, $mouse);
    }
    let move = !!(dragging && dragMoves(view, event));
    if (view.someProp("handleDrop", (f2) => f2(view, event, slice || Slice2.empty, move))) {
      event.preventDefault();
      return;
    }
    if (!slice)
      return;
    event.preventDefault();
    let insertPos = slice ? dropPoint(view.state.doc, $mouse.pos, slice) : $mouse.pos;
    if (insertPos == null)
      insertPos = $mouse.pos;
    let tr = view.state.tr;
    if (move) {
      let { node: node2 } = dragging;
      if (node2)
        node2.replace(tr);
      else
        tr.deleteSelection();
    }
    let pos = tr.mapping.map(insertPos);
    let isNode = slice.openStart == 0 && slice.openEnd == 0 && slice.content.childCount == 1;
    let beforeInsert = tr.doc;
    if (isNode)
      tr.replaceRangeWith(pos, pos, slice.content.firstChild);
    else
      tr.replaceRange(pos, pos, slice);
    if (tr.doc.eq(beforeInsert))
      return;
    let $pos = tr.doc.resolve(pos);
    if (isNode && NodeSelection.isSelectable(slice.content.firstChild) && $pos.nodeAfter && $pos.nodeAfter.sameMarkup(slice.content.firstChild)) {
      tr.setSelection(new NodeSelection($pos));
    } else {
      let end = tr.mapping.map(insertPos);
      tr.mapping.maps[tr.mapping.maps.length - 1].forEach((_from, _to, _newFrom, newTo) => end = newTo);
      tr.setSelection(selectionBetween(view, $pos, tr.doc.resolve(end)));
    }
    view.focus();
    view.dispatch(tr.setMeta("uiEvent", "drop"));
  }
  handlers.focus = (view) => {
    view.input.lastFocus = Date.now();
    if (!view.focused) {
      view.domObserver.stop();
      view.dom.classList.add("ProseMirror-focused");
      view.domObserver.start();
      view.focused = true;
      setTimeout(() => {
        if (view.docView && view.hasFocus() && !view.domObserver.currentSelection.eq(view.domSelectionRange()))
          selectionToDOM(view);
      }, 20);
    }
  };
  handlers.blur = (view, _event) => {
    let event = _event;
    if (view.focused) {
      view.domObserver.stop();
      view.dom.classList.remove("ProseMirror-focused");
      view.domObserver.start();
      if (event.relatedTarget && view.dom.contains(event.relatedTarget))
        view.domObserver.currentSelection.clear();
      view.focused = false;
    }
  };
  handlers.beforeinput = (view, _event) => {
    let event = _event;
    if (chrome && android2 && event.inputType == "deleteContentBackward") {
      view.domObserver.flushSoon();
      let { domChangeCount } = view.input;
      setTimeout(() => {
        if (view.input.domChangeCount != domChangeCount)
          return;
        view.dom.blur();
        view.focus();
        if (view.someProp("handleKeyDown", (f2) => f2(view, keyEvent(8, "Backspace"))))
          return;
        let { $cursor } = view.state.selection;
        if ($cursor && $cursor.pos > 0)
          view.dispatch(view.state.tr.delete($cursor.pos - 1, $cursor.pos).scrollIntoView());
      }, 50);
    }
  };
  for (let prop in editHandlers)
    handlers[prop] = editHandlers[prop];
  function compareObjs(a2, b) {
    if (a2 == b)
      return true;
    for (let p2 in a2)
      if (a2[p2] !== b[p2])
        return false;
    for (let p2 in b)
      if (!(p2 in a2))
        return false;
    return true;
  }
  var WidgetType = class _WidgetType {
    constructor(toDOM, spec) {
      this.toDOM = toDOM;
      this.spec = spec || noSpec;
      this.side = this.spec.side || 0;
    }
    map(mapping, span, offset, oldOffset) {
      let { pos, deleted } = mapping.mapResult(span.from + oldOffset, this.side < 0 ? -1 : 1);
      return deleted ? null : new Decoration(pos - offset, pos - offset, this);
    }
    valid() {
      return true;
    }
    eq(other) {
      return this == other || other instanceof _WidgetType && (this.spec.key && this.spec.key == other.spec.key || this.toDOM == other.toDOM && compareObjs(this.spec, other.spec));
    }
    destroy(node2) {
      if (this.spec.destroy)
        this.spec.destroy(node2);
    }
  };
  var InlineType = class _InlineType {
    constructor(attrs, spec) {
      this.attrs = attrs;
      this.spec = spec || noSpec;
    }
    map(mapping, span, offset, oldOffset) {
      let from = mapping.map(span.from + oldOffset, this.spec.inclusiveStart ? -1 : 1) - offset;
      let to = mapping.map(span.to + oldOffset, this.spec.inclusiveEnd ? 1 : -1) - offset;
      return from >= to ? null : new Decoration(from, to, this);
    }
    valid(_, span) {
      return span.from < span.to;
    }
    eq(other) {
      return this == other || other instanceof _InlineType && compareObjs(this.attrs, other.attrs) && compareObjs(this.spec, other.spec);
    }
    static is(span) {
      return span.type instanceof _InlineType;
    }
    destroy() {
    }
  };
  var NodeType2 = class _NodeType {
    constructor(attrs, spec) {
      this.attrs = attrs;
      this.spec = spec || noSpec;
    }
    map(mapping, span, offset, oldOffset) {
      let from = mapping.mapResult(span.from + oldOffset, 1);
      if (from.deleted)
        return null;
      let to = mapping.mapResult(span.to + oldOffset, -1);
      if (to.deleted || to.pos <= from.pos)
        return null;
      return new Decoration(from.pos - offset, to.pos - offset, this);
    }
    valid(node2, span) {
      let { index: index2, offset } = node2.content.findIndex(span.from), child;
      return offset == span.from && !(child = node2.child(index2)).isText && offset + child.nodeSize == span.to;
    }
    eq(other) {
      return this == other || other instanceof _NodeType && compareObjs(this.attrs, other.attrs) && compareObjs(this.spec, other.spec);
    }
    destroy() {
    }
  };
  var Decoration = class _Decoration {
    /**
    @internal
    */
    constructor(from, to, type) {
      this.from = from;
      this.to = to;
      this.type = type;
    }
    /**
    @internal
    */
    copy(from, to) {
      return new _Decoration(from, to, this.type);
    }
    /**
    @internal
    */
    eq(other, offset = 0) {
      return this.type.eq(other.type) && this.from + offset == other.from && this.to + offset == other.to;
    }
    /**
    @internal
    */
    map(mapping, offset, oldOffset) {
      return this.type.map(mapping, this, offset, oldOffset);
    }
    /**
    Creates a widget decoration, which is a DOM node that's shown in
    the document at the given position. It is recommended that you
    delay rendering the widget by passing a function that will be
    called when the widget is actually drawn in a view, but you can
    also directly pass a DOM node. `getPos` can be used to find the
    widget's current document position.
    */
    static widget(pos, toDOM, spec) {
      return new _Decoration(pos, pos, new WidgetType(toDOM, spec));
    }
    /**
    Creates an inline decoration, which adds the given attributes to
    each inline node between `from` and `to`.
    */
    static inline(from, to, attrs, spec) {
      return new _Decoration(from, to, new InlineType(attrs, spec));
    }
    /**
    Creates a node decoration. `from` and `to` should point precisely
    before and after a node in the document. That node, and only that
    node, will receive the given attributes.
    */
    static node(from, to, attrs, spec) {
      return new _Decoration(from, to, new NodeType2(attrs, spec));
    }
    /**
    The spec provided when creating this decoration. Can be useful
    if you've stored extra information in that object.
    */
    get spec() {
      return this.type.spec;
    }
    /**
    @internal
    */
    get inline() {
      return this.type instanceof InlineType;
    }
    /**
    @internal
    */
    get widget() {
      return this.type instanceof WidgetType;
    }
  };
  var none = [];
  var noSpec = {};
  var DecorationSet = class _DecorationSet {
    /**
    @internal
    */
    constructor(local, children) {
      this.local = local.length ? local : none;
      this.children = children.length ? children : none;
    }
    /**
    Create a set of decorations, using the structure of the given
    document. This will consume (modify) the `decorations` array, so
    you must make a copy if you want need to preserve that.
    */
    static create(doc4, decorations) {
      return decorations.length ? buildTree(decorations, doc4, 0, noSpec) : empty2;
    }
    /**
    Find all decorations in this set which touch the given range
    (including decorations that start or end directly at the
    boundaries) and match the given predicate on their spec. When
    `start` and `end` are omitted, all decorations in the set are
    considered. When `predicate` isn't given, all decorations are
    assumed to match.
    */
    find(start, end, predicate) {
      let result = [];
      this.findInner(start == null ? 0 : start, end == null ? 1e9 : end, result, 0, predicate);
      return result;
    }
    findInner(start, end, result, offset, predicate) {
      for (let i2 = 0; i2 < this.local.length; i2++) {
        let span = this.local[i2];
        if (span.from <= end && span.to >= start && (!predicate || predicate(span.spec)))
          result.push(span.copy(span.from + offset, span.to + offset));
      }
      for (let i2 = 0; i2 < this.children.length; i2 += 3) {
        if (this.children[i2] < end && this.children[i2 + 1] > start) {
          let childOff = this.children[i2] + 1;
          this.children[i2 + 2].findInner(start - childOff, end - childOff, result, offset + childOff, predicate);
        }
      }
    }
    /**
    Map the set of decorations in response to a change in the
    document.
    */
    map(mapping, doc4, options) {
      if (this == empty2 || mapping.maps.length == 0)
        return this;
      return this.mapInner(mapping, doc4, 0, 0, options || noSpec);
    }
    /**
    @internal
    */
    mapInner(mapping, node2, offset, oldOffset, options) {
      let newLocal;
      for (let i2 = 0; i2 < this.local.length; i2++) {
        let mapped = this.local[i2].map(mapping, offset, oldOffset);
        if (mapped && mapped.type.valid(node2, mapped))
          (newLocal || (newLocal = [])).push(mapped);
        else if (options.onRemove)
          options.onRemove(this.local[i2].spec);
      }
      if (this.children.length)
        return mapChildren(this.children, newLocal || [], mapping, node2, offset, oldOffset, options);
      else
        return newLocal ? new _DecorationSet(newLocal.sort(byPos), none) : empty2;
    }
    /**
    Add the given array of decorations to the ones in the set,
    producing a new set. Consumes the `decorations` array. Needs
    access to the current document to create the appropriate tree
    structure.
    */
    add(doc4, decorations) {
      if (!decorations.length)
        return this;
      if (this == empty2)
        return _DecorationSet.create(doc4, decorations);
      return this.addInner(doc4, decorations, 0);
    }
    addInner(doc4, decorations, offset) {
      let children, childIndex = 0;
      doc4.forEach((childNode, childOffset) => {
        let baseOffset = childOffset + offset, found2;
        if (!(found2 = takeSpansForNode(decorations, childNode, baseOffset)))
          return;
        if (!children)
          children = this.children.slice();
        while (childIndex < children.length && children[childIndex] < childOffset)
          childIndex += 3;
        if (children[childIndex] == childOffset)
          children[childIndex + 2] = children[childIndex + 2].addInner(childNode, found2, baseOffset + 1);
        else
          children.splice(childIndex, 0, childOffset, childOffset + childNode.nodeSize, buildTree(found2, childNode, baseOffset + 1, noSpec));
        childIndex += 3;
      });
      let local = moveSpans(childIndex ? withoutNulls(decorations) : decorations, -offset);
      for (let i2 = 0; i2 < local.length; i2++)
        if (!local[i2].type.valid(doc4, local[i2]))
          local.splice(i2--, 1);
      return new _DecorationSet(local.length ? this.local.concat(local).sort(byPos) : this.local, children || this.children);
    }
    /**
    Create a new set that contains the decorations in this set, minus
    the ones in the given array.
    */
    remove(decorations) {
      if (decorations.length == 0 || this == empty2)
        return this;
      return this.removeInner(decorations, 0);
    }
    removeInner(decorations, offset) {
      let children = this.children, local = this.local;
      for (let i2 = 0; i2 < children.length; i2 += 3) {
        let found2;
        let from = children[i2] + offset, to = children[i2 + 1] + offset;
        for (let j = 0, span; j < decorations.length; j++)
          if (span = decorations[j]) {
            if (span.from > from && span.to < to) {
              decorations[j] = null;
              (found2 || (found2 = [])).push(span);
            }
          }
        if (!found2)
          continue;
        if (children == this.children)
          children = this.children.slice();
        let removed = children[i2 + 2].removeInner(found2, from + 1);
        if (removed != empty2) {
          children[i2 + 2] = removed;
        } else {
          children.splice(i2, 3);
          i2 -= 3;
        }
      }
      if (local.length) {
        for (let i2 = 0, span; i2 < decorations.length; i2++)
          if (span = decorations[i2]) {
            for (let j = 0; j < local.length; j++)
              if (local[j].eq(span, offset)) {
                if (local == this.local)
                  local = this.local.slice();
                local.splice(j--, 1);
              }
          }
      }
      if (children == this.children && local == this.local)
        return this;
      return local.length || children.length ? new _DecorationSet(local, children) : empty2;
    }
    forChild(offset, node2) {
      if (this == empty2)
        return this;
      if (node2.isLeaf)
        return _DecorationSet.empty;
      let child, local;
      for (let i2 = 0; i2 < this.children.length; i2 += 3)
        if (this.children[i2] >= offset) {
          if (this.children[i2] == offset)
            child = this.children[i2 + 2];
          break;
        }
      let start = offset + 1, end = start + node2.content.size;
      for (let i2 = 0; i2 < this.local.length; i2++) {
        let dec = this.local[i2];
        if (dec.from < end && dec.to > start && dec.type instanceof InlineType) {
          let from = Math.max(start, dec.from) - start, to = Math.min(end, dec.to) - start;
          if (from < to)
            (local || (local = [])).push(dec.copy(from, to));
        }
      }
      if (local) {
        let localSet = new _DecorationSet(local.sort(byPos), none);
        return child ? new DecorationGroup([localSet, child]) : localSet;
      }
      return child || empty2;
    }
    /**
    @internal
    */
    eq(other) {
      if (this == other)
        return true;
      if (!(other instanceof _DecorationSet) || this.local.length != other.local.length || this.children.length != other.children.length)
        return false;
      for (let i2 = 0; i2 < this.local.length; i2++)
        if (!this.local[i2].eq(other.local[i2]))
          return false;
      for (let i2 = 0; i2 < this.children.length; i2 += 3)
        if (this.children[i2] != other.children[i2] || this.children[i2 + 1] != other.children[i2 + 1] || !this.children[i2 + 2].eq(other.children[i2 + 2]))
          return false;
      return true;
    }
    /**
    @internal
    */
    locals(node2) {
      return removeOverlap(this.localsInner(node2));
    }
    /**
    @internal
    */
    localsInner(node2) {
      if (this == empty2)
        return none;
      if (node2.inlineContent || !this.local.some(InlineType.is))
        return this.local;
      let result = [];
      for (let i2 = 0; i2 < this.local.length; i2++) {
        if (!(this.local[i2].type instanceof InlineType))
          result.push(this.local[i2]);
      }
      return result;
    }
    forEachSet(f2) {
      f2(this);
    }
  };
  DecorationSet.empty = new DecorationSet([], []);
  DecorationSet.removeOverlap = removeOverlap;
  var empty2 = DecorationSet.empty;
  var DecorationGroup = class _DecorationGroup {
    constructor(members) {
      this.members = members;
    }
    map(mapping, doc4) {
      const mappedDecos = this.members.map((member) => member.map(mapping, doc4, noSpec));
      return _DecorationGroup.from(mappedDecos);
    }
    forChild(offset, child) {
      if (child.isLeaf)
        return DecorationSet.empty;
      let found2 = [];
      for (let i2 = 0; i2 < this.members.length; i2++) {
        let result = this.members[i2].forChild(offset, child);
        if (result == empty2)
          continue;
        if (result instanceof _DecorationGroup)
          found2 = found2.concat(result.members);
        else
          found2.push(result);
      }
      return _DecorationGroup.from(found2);
    }
    eq(other) {
      if (!(other instanceof _DecorationGroup) || other.members.length != this.members.length)
        return false;
      for (let i2 = 0; i2 < this.members.length; i2++)
        if (!this.members[i2].eq(other.members[i2]))
          return false;
      return true;
    }
    locals(node2) {
      let result, sorted = true;
      for (let i2 = 0; i2 < this.members.length; i2++) {
        let locals = this.members[i2].localsInner(node2);
        if (!locals.length)
          continue;
        if (!result) {
          result = locals;
        } else {
          if (sorted) {
            result = result.slice();
            sorted = false;
          }
          for (let j = 0; j < locals.length; j++)
            result.push(locals[j]);
        }
      }
      return result ? removeOverlap(sorted ? result : result.sort(byPos)) : none;
    }
    // Create a group for the given array of decoration sets, or return
    // a single set when possible.
    static from(members) {
      switch (members.length) {
        case 0:
          return empty2;
        case 1:
          return members[0];
        default:
          return new _DecorationGroup(members.every((m) => m instanceof DecorationSet) ? members : members.reduce((r2, m) => r2.concat(m instanceof DecorationSet ? m : m.members), []));
      }
    }
    forEachSet(f2) {
      for (let i2 = 0; i2 < this.members.length; i2++)
        this.members[i2].forEachSet(f2);
    }
  };
  function mapChildren(oldChildren, newLocal, mapping, node2, offset, oldOffset, options) {
    let children = oldChildren.slice();
    for (let i2 = 0, baseOffset = oldOffset; i2 < mapping.maps.length; i2++) {
      let moved = 0;
      mapping.maps[i2].forEach((oldStart, oldEnd, newStart, newEnd) => {
        let dSize = newEnd - newStart - (oldEnd - oldStart);
        for (let i3 = 0; i3 < children.length; i3 += 3) {
          let end = children[i3 + 1];
          if (end < 0 || oldStart > end + baseOffset - moved)
            continue;
          let start = children[i3] + baseOffset - moved;
          if (oldEnd >= start) {
            children[i3 + 1] = oldStart <= start ? -2 : -1;
          } else if (oldStart >= baseOffset && dSize) {
            children[i3] += dSize;
            children[i3 + 1] += dSize;
          }
        }
        moved += dSize;
      });
      baseOffset = mapping.maps[i2].map(baseOffset, -1);
    }
    let mustRebuild = false;
    for (let i2 = 0; i2 < children.length; i2 += 3)
      if (children[i2 + 1] < 0) {
        if (children[i2 + 1] == -2) {
          mustRebuild = true;
          children[i2 + 1] = -1;
          continue;
        }
        let from = mapping.map(oldChildren[i2] + oldOffset), fromLocal = from - offset;
        if (fromLocal < 0 || fromLocal >= node2.content.size) {
          mustRebuild = true;
          continue;
        }
        let to = mapping.map(oldChildren[i2 + 1] + oldOffset, -1), toLocal = to - offset;
        let { index: index2, offset: childOffset } = node2.content.findIndex(fromLocal);
        let childNode = node2.maybeChild(index2);
        if (childNode && childOffset == fromLocal && childOffset + childNode.nodeSize == toLocal) {
          let mapped = children[i2 + 2].mapInner(mapping, childNode, from + 1, oldChildren[i2] + oldOffset + 1, options);
          if (mapped != empty2) {
            children[i2] = fromLocal;
            children[i2 + 1] = toLocal;
            children[i2 + 2] = mapped;
          } else {
            children[i2 + 1] = -2;
            mustRebuild = true;
          }
        } else {
          mustRebuild = true;
        }
      }
    if (mustRebuild) {
      let decorations = mapAndGatherRemainingDecorations(children, oldChildren, newLocal, mapping, offset, oldOffset, options);
      let built = buildTree(decorations, node2, 0, options);
      newLocal = built.local;
      for (let i2 = 0; i2 < children.length; i2 += 3)
        if (children[i2 + 1] < 0) {
          children.splice(i2, 3);
          i2 -= 3;
        }
      for (let i2 = 0, j = 0; i2 < built.children.length; i2 += 3) {
        let from = built.children[i2];
        while (j < children.length && children[j] < from)
          j += 3;
        children.splice(j, 0, built.children[i2], built.children[i2 + 1], built.children[i2 + 2]);
      }
    }
    return new DecorationSet(newLocal.sort(byPos), children);
  }
  function moveSpans(spans, offset) {
    if (!offset || !spans.length)
      return spans;
    let result = [];
    for (let i2 = 0; i2 < spans.length; i2++) {
      let span = spans[i2];
      result.push(new Decoration(span.from + offset, span.to + offset, span.type));
    }
    return result;
  }
  function mapAndGatherRemainingDecorations(children, oldChildren, decorations, mapping, offset, oldOffset, options) {
    function gather(set, oldOffset2) {
      for (let i2 = 0; i2 < set.local.length; i2++) {
        let mapped = set.local[i2].map(mapping, offset, oldOffset2);
        if (mapped)
          decorations.push(mapped);
        else if (options.onRemove)
          options.onRemove(set.local[i2].spec);
      }
      for (let i2 = 0; i2 < set.children.length; i2 += 3)
        gather(set.children[i2 + 2], set.children[i2] + oldOffset2 + 1);
    }
    for (let i2 = 0; i2 < children.length; i2 += 3)
      if (children[i2 + 1] == -1)
        gather(children[i2 + 2], oldChildren[i2] + oldOffset + 1);
    return decorations;
  }
  function takeSpansForNode(spans, node2, offset) {
    if (node2.isLeaf)
      return null;
    let end = offset + node2.nodeSize, found2 = null;
    for (let i2 = 0, span; i2 < spans.length; i2++) {
      if ((span = spans[i2]) && span.from > offset && span.to < end) {
        (found2 || (found2 = [])).push(span);
        spans[i2] = null;
      }
    }
    return found2;
  }
  function withoutNulls(array) {
    let result = [];
    for (let i2 = 0; i2 < array.length; i2++)
      if (array[i2] != null)
        result.push(array[i2]);
    return result;
  }
  function buildTree(spans, node2, offset, options) {
    let children = [], hasNulls = false;
    node2.forEach((childNode, localStart) => {
      let found2 = takeSpansForNode(spans, childNode, localStart + offset);
      if (found2) {
        hasNulls = true;
        let subtree = buildTree(found2, childNode, offset + localStart + 1, options);
        if (subtree != empty2)
          children.push(localStart, localStart + childNode.nodeSize, subtree);
      }
    });
    let locals = moveSpans(hasNulls ? withoutNulls(spans) : spans, -offset).sort(byPos);
    for (let i2 = 0; i2 < locals.length; i2++)
      if (!locals[i2].type.valid(node2, locals[i2])) {
        if (options.onRemove)
          options.onRemove(locals[i2].spec);
        locals.splice(i2--, 1);
      }
    return locals.length || children.length ? new DecorationSet(locals, children) : empty2;
  }
  function byPos(a2, b) {
    return a2.from - b.from || a2.to - b.to;
  }
  function removeOverlap(spans) {
    let working = spans;
    for (let i2 = 0; i2 < working.length - 1; i2++) {
      let span = working[i2];
      if (span.from != span.to)
        for (let j = i2 + 1; j < working.length; j++) {
          let next = working[j];
          if (next.from == span.from) {
            if (next.to != span.to) {
              if (working == spans)
                working = spans.slice();
              working[j] = next.copy(next.from, span.to);
              insertAhead(working, j + 1, next.copy(span.to, next.to));
            }
            continue;
          } else {
            if (next.from < span.to) {
              if (working == spans)
                working = spans.slice();
              working[i2] = span.copy(span.from, next.from);
              insertAhead(working, j, span.copy(next.from, span.to));
            }
            break;
          }
        }
    }
    return working;
  }
  function insertAhead(array, i2, deco) {
    while (i2 < array.length && byPos(deco, array[i2]) > 0)
      i2++;
    array.splice(i2, 0, deco);
  }
  function viewDecorations(view) {
    let found2 = [];
    view.someProp("decorations", (f2) => {
      let result = f2(view.state);
      if (result && result != empty2)
        found2.push(result);
    });
    if (view.cursorWrapper)
      found2.push(DecorationSet.create(view.state.doc, [view.cursorWrapper.deco]));
    return DecorationGroup.from(found2);
  }
  var observeOptions = {
    childList: true,
    characterData: true,
    characterDataOldValue: true,
    attributes: true,
    attributeOldValue: true,
    subtree: true
  };
  var useCharData = ie3 && ie_version2 <= 11;
  var SelectionState = class {
    constructor() {
      this.anchorNode = null;
      this.anchorOffset = 0;
      this.focusNode = null;
      this.focusOffset = 0;
    }
    set(sel) {
      this.anchorNode = sel.anchorNode;
      this.anchorOffset = sel.anchorOffset;
      this.focusNode = sel.focusNode;
      this.focusOffset = sel.focusOffset;
    }
    clear() {
      this.anchorNode = this.focusNode = null;
    }
    eq(sel) {
      return sel.anchorNode == this.anchorNode && sel.anchorOffset == this.anchorOffset && sel.focusNode == this.focusNode && sel.focusOffset == this.focusOffset;
    }
  };
  var DOMObserver = class {
    constructor(view, handleDOMChange) {
      this.view = view;
      this.handleDOMChange = handleDOMChange;
      this.queue = [];
      this.flushingSoon = -1;
      this.observer = null;
      this.currentSelection = new SelectionState();
      this.onCharData = null;
      this.suppressingSelectionUpdates = false;
      this.lastChangedTextNode = null;
      this.observer = window.MutationObserver && new window.MutationObserver((mutations) => {
        for (let i2 = 0; i2 < mutations.length; i2++)
          this.queue.push(mutations[i2]);
        if (ie3 && ie_version2 <= 11 && mutations.some((m) => m.type == "childList" && m.removedNodes.length || m.type == "characterData" && m.oldValue.length > m.target.nodeValue.length)) {
          this.flushSoon();
        } else if (safari2 && view.composing && mutations.some((m) => m.type == "childList" && m.target.nodeName == "TR")) {
          view.input.badSafariComposition = true;
          this.flushSoon();
        } else {
          this.flush();
        }
      });
      if (useCharData) {
        this.onCharData = (e2) => {
          this.queue.push({ target: e2.target, type: "characterData", oldValue: e2.prevValue });
          this.flushSoon();
        };
      }
      this.onSelectionChange = this.onSelectionChange.bind(this);
    }
    flushSoon() {
      if (this.flushingSoon < 0)
        this.flushingSoon = window.setTimeout(() => {
          this.flushingSoon = -1;
          this.flush();
        }, 20);
    }
    forceFlush() {
      if (this.flushingSoon > -1) {
        window.clearTimeout(this.flushingSoon);
        this.flushingSoon = -1;
        this.flush();
      }
    }
    start() {
      if (this.observer) {
        this.observer.takeRecords();
        this.observer.observe(this.view.dom, observeOptions);
      }
      if (this.onCharData)
        this.view.dom.addEventListener("DOMCharacterDataModified", this.onCharData);
      this.connectSelection();
    }
    stop() {
      if (this.observer) {
        let take = this.observer.takeRecords();
        if (take.length) {
          for (let i2 = 0; i2 < take.length; i2++)
            this.queue.push(take[i2]);
          window.setTimeout(() => this.flush(), 20);
        }
        this.observer.disconnect();
      }
      if (this.onCharData)
        this.view.dom.removeEventListener("DOMCharacterDataModified", this.onCharData);
      this.disconnectSelection();
    }
    connectSelection() {
      this.view.dom.ownerDocument.addEventListener("selectionchange", this.onSelectionChange);
    }
    disconnectSelection() {
      this.view.dom.ownerDocument.removeEventListener("selectionchange", this.onSelectionChange);
    }
    suppressSelectionUpdates() {
      this.suppressingSelectionUpdates = true;
      setTimeout(() => this.suppressingSelectionUpdates = false, 50);
    }
    onSelectionChange() {
      if (!hasFocusAndSelection(this.view))
        return;
      if (this.suppressingSelectionUpdates)
        return selectionToDOM(this.view);
      if (ie3 && ie_version2 <= 11 && !this.view.state.selection.empty) {
        let sel = this.view.domSelectionRange();
        if (sel.focusNode && isEquivalentPosition(sel.focusNode, sel.focusOffset, sel.anchorNode, sel.anchorOffset))
          return this.flushSoon();
      }
      this.flush();
    }
    setCurSelection() {
      this.currentSelection.set(this.view.domSelectionRange());
    }
    ignoreSelectionChange(sel) {
      if (!sel.focusNode)
        return true;
      let ancestors = /* @__PURE__ */ new Set(), container;
      for (let scan = sel.focusNode; scan; scan = parentNode(scan))
        ancestors.add(scan);
      for (let scan = sel.anchorNode; scan; scan = parentNode(scan))
        if (ancestors.has(scan)) {
          container = scan;
          break;
        }
      let desc = container && this.view.docView.nearestDesc(container);
      if (desc && desc.ignoreMutation({
        type: "selection",
        target: container.nodeType == 3 ? container.parentNode : container
      })) {
        this.setCurSelection();
        return true;
      }
    }
    pendingRecords() {
      if (this.observer)
        for (let mut of this.observer.takeRecords())
          this.queue.push(mut);
      return this.queue;
    }
    flush() {
      let { view } = this;
      if (!view.docView || this.flushingSoon > -1)
        return;
      let mutations = this.pendingRecords();
      if (mutations.length)
        this.queue = [];
      let sel = view.domSelectionRange();
      let newSel = !this.suppressingSelectionUpdates && !this.currentSelection.eq(sel) && hasFocusAndSelection(view) && !this.ignoreSelectionChange(sel);
      let from = -1, to = -1, typeOver = false, added = [];
      if (view.editable) {
        for (let i2 = 0; i2 < mutations.length; i2++) {
          let result = this.registerMutation(mutations[i2], added);
          if (result) {
            from = from < 0 ? result.from : Math.min(result.from, from);
            to = to < 0 ? result.to : Math.max(result.to, to);
            if (result.typeOver)
              typeOver = true;
          }
        }
      }
      if (added.some((n2) => n2.nodeName == "BR") && (view.input.lastKeyCode == 8 || view.input.lastKeyCode == 46)) {
        for (let node2 of added)
          if (node2.nodeName == "BR" && node2.parentNode) {
            let after = node2.nextSibling;
            if (after && after.nodeType == 1 && after.contentEditable == "false")
              node2.parentNode.removeChild(node2);
          }
      } else if (gecko2 && added.length) {
        let brs = added.filter((n2) => n2.nodeName == "BR");
        if (brs.length == 2) {
          let [a2, b] = brs;
          if (a2.parentNode && a2.parentNode.parentNode == b.parentNode)
            b.remove();
          else
            a2.remove();
        } else {
          let { focusNode } = this.currentSelection;
          for (let br of brs) {
            let parent = br.parentNode;
            if (parent && parent.nodeName == "LI" && (!focusNode || blockParent(view, focusNode) != parent))
              br.remove();
          }
        }
      }
      let readSel = null;
      if (from < 0 && newSel && view.input.lastFocus > Date.now() - 200 && Math.max(view.input.lastTouch, view.input.lastClick.time) < Date.now() - 300 && selectionCollapsed(sel) && (readSel = selectionFromDOM(view)) && readSel.eq(Selection.near(view.state.doc.resolve(0), 1))) {
        view.input.lastFocus = 0;
        selectionToDOM(view);
        this.currentSelection.set(sel);
        view.scrollToSelection();
      } else if (from > -1 || newSel) {
        if (from > -1) {
          view.docView.markDirty(from, to);
          checkCSS(view);
        }
        if (view.input.badSafariComposition) {
          view.input.badSafariComposition = false;
          fixUpBadSafariComposition(view, added);
        }
        this.handleDOMChange(from, to, typeOver, added);
        if (view.docView && view.docView.dirty)
          view.updateState(view.state);
        else if (!this.currentSelection.eq(sel))
          selectionToDOM(view);
        this.currentSelection.set(sel);
      }
    }
    registerMutation(mut, added) {
      if (added.indexOf(mut.target) > -1)
        return null;
      let desc = this.view.docView.nearestDesc(mut.target);
      if (mut.type == "attributes" && (desc == this.view.docView || mut.attributeName == "contenteditable" || // Firefox sometimes fires spurious events for null/empty styles
      mut.attributeName == "style" && !mut.oldValue && !mut.target.getAttribute("style")))
        return null;
      if (!desc || desc.ignoreMutation(mut))
        return null;
      if (mut.type == "childList") {
        for (let i2 = 0; i2 < mut.addedNodes.length; i2++) {
          let node2 = mut.addedNodes[i2];
          added.push(node2);
          if (node2.nodeType == 3)
            this.lastChangedTextNode = node2;
        }
        if (desc.contentDOM && desc.contentDOM != desc.dom && !desc.contentDOM.contains(mut.target))
          return { from: desc.posBefore, to: desc.posAfter };
        let prev = mut.previousSibling, next = mut.nextSibling;
        if (ie3 && ie_version2 <= 11 && mut.addedNodes.length) {
          for (let i2 = 0; i2 < mut.addedNodes.length; i2++) {
            let { previousSibling, nextSibling } = mut.addedNodes[i2];
            if (!previousSibling || Array.prototype.indexOf.call(mut.addedNodes, previousSibling) < 0)
              prev = previousSibling;
            if (!nextSibling || Array.prototype.indexOf.call(mut.addedNodes, nextSibling) < 0)
              next = nextSibling;
          }
        }
        let fromOffset = prev && prev.parentNode == mut.target ? domIndex(prev) + 1 : 0;
        let from = desc.localPosFromDOM(mut.target, fromOffset, -1);
        let toOffset = next && next.parentNode == mut.target ? domIndex(next) : mut.target.childNodes.length;
        let to = desc.localPosFromDOM(mut.target, toOffset, 1);
        return { from, to };
      } else if (mut.type == "attributes") {
        return { from: desc.posAtStart - desc.border, to: desc.posAtEnd + desc.border };
      } else {
        this.lastChangedTextNode = mut.target;
        return {
          from: desc.posAtStart,
          to: desc.posAtEnd,
          // An event was generated for a text change that didn't change
          // any text. Mark the dom change to fall back to assuming the
          // selection was typed over with an identical value if it can't
          // find another change.
          typeOver: mut.target.nodeValue == mut.oldValue
        };
      }
    }
  };
  var cssChecked = /* @__PURE__ */ new WeakMap();
  var cssCheckWarned = false;
  function checkCSS(view) {
    if (cssChecked.has(view))
      return;
    cssChecked.set(view, null);
    if (["normal", "nowrap", "pre-line"].indexOf(getComputedStyle(view.dom).whiteSpace) !== -1) {
      view.requiresGeckoHackNode = gecko2;
      if (cssCheckWarned)
        return;
      console["warn"]("ProseMirror expects the CSS white-space property to be set, preferably to 'pre-wrap'. It is recommended to load style/prosemirror.css from the prosemirror-view package.");
      cssCheckWarned = true;
    }
  }
  function rangeToSelectionRange(view, range) {
    let anchorNode = range.startContainer, anchorOffset = range.startOffset;
    let focusNode = range.endContainer, focusOffset = range.endOffset;
    let currentAnchor = view.domAtPos(view.state.selection.anchor);
    if (isEquivalentPosition(currentAnchor.node, currentAnchor.offset, focusNode, focusOffset))
      [anchorNode, anchorOffset, focusNode, focusOffset] = [focusNode, focusOffset, anchorNode, anchorOffset];
    return { anchorNode, anchorOffset, focusNode, focusOffset };
  }
  function safariShadowSelectionRange(view, selection) {
    if (selection.getComposedRanges) {
      let range = selection.getComposedRanges(view.root)[0];
      if (range)
        return rangeToSelectionRange(view, range);
    }
    let found2;
    function read(event) {
      event.preventDefault();
      event.stopImmediatePropagation();
      found2 = event.getTargetRanges()[0];
    }
    view.dom.addEventListener("beforeinput", read, true);
    document.execCommand("indent");
    view.dom.removeEventListener("beforeinput", read, true);
    return found2 ? rangeToSelectionRange(view, found2) : null;
  }
  function blockParent(view, node2) {
    for (let p2 = node2.parentNode; p2 && p2 != view.dom; p2 = p2.parentNode) {
      let desc = view.docView.nearestDesc(p2, true);
      if (desc && desc.node.isBlock)
        return p2;
    }
    return null;
  }
  function fixUpBadSafariComposition(view, addedNodes) {
    var _a;
    let { focusNode, focusOffset } = view.domSelectionRange();
    for (let node2 of addedNodes) {
      if (((_a = node2.parentNode) === null || _a === void 0 ? void 0 : _a.nodeName) == "TR") {
        let nextCell = node2.nextSibling;
        while (nextCell && (nextCell.nodeName != "TD" && nextCell.nodeName != "TH"))
          nextCell = nextCell.nextSibling;
        if (nextCell) {
          let parent = nextCell;
          for (; ; ) {
            let first = parent.firstChild;
            if (!first || first.nodeType != 1 || first.contentEditable == "false" || /^(BR|IMG)$/.test(first.nodeName))
              break;
            parent = first;
          }
          parent.insertBefore(node2, parent.firstChild);
          if (focusNode == node2)
            view.domSelection().collapse(node2, focusOffset);
        } else {
          node2.parentNode.removeChild(node2);
        }
      }
    }
  }
  function parseBetween(view, from_, to_) {
    let { node: parent, fromOffset, toOffset, from, to } = view.docView.parseRange(from_, to_);
    let domSel = view.domSelectionRange();
    let find;
    let anchor = domSel.anchorNode;
    if (anchor && view.dom.contains(anchor.nodeType == 1 ? anchor : anchor.parentNode)) {
      find = [{ node: anchor, offset: domSel.anchorOffset }];
      if (!selectionCollapsed(domSel))
        find.push({ node: domSel.focusNode, offset: domSel.focusOffset });
    }
    if (chrome && view.input.lastKeyCode === 8) {
      for (let off = toOffset; off > fromOffset; off--) {
        let node2 = parent.childNodes[off - 1], desc = node2.pmViewDesc;
        if (node2.nodeName == "BR" && !desc) {
          toOffset = off;
          break;
        }
        if (!desc || desc.size)
          break;
      }
    }
    let startDoc = view.state.doc;
    let parser2 = view.someProp("domParser") || DOMParser.fromSchema(view.state.schema);
    let $from = startDoc.resolve(from);
    let sel = null, doc4 = parser2.parse(parent, {
      topNode: $from.parent,
      topMatch: $from.parent.contentMatchAt($from.index()),
      topOpen: true,
      from: fromOffset,
      to: toOffset,
      preserveWhitespace: $from.parent.type.whitespace == "pre" ? "full" : true,
      findPositions: find,
      ruleFromNode,
      context: $from
    });
    if (find && find[0].pos != null) {
      let anchor2 = find[0].pos, head = find[1] && find[1].pos;
      if (head == null)
        head = anchor2;
      sel = { anchor: anchor2 + from, head: head + from };
    }
    return { doc: doc4, sel, from, to };
  }
  function ruleFromNode(dom) {
    let desc = dom.pmViewDesc;
    if (desc) {
      return desc.parseRule();
    } else if (dom.nodeName == "BR" && dom.parentNode) {
      if (safari2 && /^(ul|ol)$/i.test(dom.parentNode.nodeName)) {
        let skip = document.createElement("div");
        skip.appendChild(document.createElement("li"));
        return { skip };
      } else if (dom.parentNode.lastChild == dom || safari2 && /^(tr|table)$/i.test(dom.parentNode.nodeName)) {
        return { ignore: true };
      }
    } else if (dom.nodeName == "IMG" && dom.getAttribute("mark-placeholder")) {
      return { ignore: true };
    }
    return null;
  }
  var isInline = /^(a|abbr|acronym|b|bd[io]|big|br|button|cite|code|data(list)?|del|dfn|em|i|img|ins|kbd|label|map|mark|meter|output|q|ruby|s|samp|small|span|strong|su[bp]|time|u|tt|var)$/i;
  function readDOMChange(view, from, to, typeOver, addedNodes) {
    let compositionID = view.input.compositionPendingChanges || (view.composing ? view.input.compositionID : 0);
    view.input.compositionPendingChanges = 0;
    if (from < 0) {
      let origin = view.input.lastSelectionTime > Date.now() - 50 ? view.input.lastSelectionOrigin : null;
      let newSel = selectionFromDOM(view, origin);
      if (newSel && !view.state.selection.eq(newSel)) {
        if (chrome && android2 && view.input.lastKeyCode === 13 && Date.now() - 100 < view.input.lastKeyCodeTime && view.someProp("handleKeyDown", (f2) => f2(view, keyEvent(13, "Enter"))))
          return;
        let tr = view.state.tr.setSelection(newSel);
        if (origin == "pointer")
          tr.setMeta("pointer", true);
        else if (origin == "key")
          tr.scrollIntoView();
        if (compositionID)
          tr.setMeta("composition", compositionID);
        view.dispatch(tr);
      }
      return;
    }
    let $before = view.state.doc.resolve(from);
    let shared = $before.sharedDepth(to);
    from = $before.before(shared + 1);
    to = view.state.doc.resolve(to).after(shared + 1);
    let sel = view.state.selection;
    let parse2 = parseBetween(view, from, to);
    let doc4 = view.state.doc, compare = doc4.slice(parse2.from, parse2.to);
    let preferredPos, preferredSide;
    if (view.input.lastKeyCode === 8 && Date.now() - 100 < view.input.lastKeyCodeTime) {
      preferredPos = view.state.selection.to;
      preferredSide = "end";
    } else {
      preferredPos = view.state.selection.from;
      preferredSide = "start";
    }
    view.input.lastKeyCode = null;
    let change = findDiff(compare.content, parse2.doc.content, parse2.from, preferredPos, preferredSide);
    if (change)
      view.input.domChangeCount++;
    if ((ios2 && view.input.lastIOSEnter > Date.now() - 225 || android2) && addedNodes.some((n2) => n2.nodeType == 1 && !isInline.test(n2.nodeName)) && (!change || change.endA >= change.endB) && view.someProp("handleKeyDown", (f2) => f2(view, keyEvent(13, "Enter")))) {
      view.input.lastIOSEnter = 0;
      return;
    }
    if (!change) {
      if (typeOver && sel instanceof TextSelection && !sel.empty && sel.$head.sameParent(sel.$anchor) && !view.composing && !(parse2.sel && parse2.sel.anchor != parse2.sel.head)) {
        change = { start: sel.from, endA: sel.to, endB: sel.to };
      } else {
        if (parse2.sel) {
          let sel2 = resolveSelection(view, view.state.doc, parse2.sel);
          if (sel2 && !sel2.eq(view.state.selection)) {
            let tr = view.state.tr.setSelection(sel2);
            if (compositionID)
              tr.setMeta("composition", compositionID);
            view.dispatch(tr);
          }
        }
        return;
      }
    }
    if (view.state.selection.from < view.state.selection.to && change.start == change.endB && view.state.selection instanceof TextSelection) {
      if (change.start > view.state.selection.from && change.start <= view.state.selection.from + 2 && view.state.selection.from >= parse2.from) {
        change.start = view.state.selection.from;
      } else if (change.endA < view.state.selection.to && change.endA >= view.state.selection.to - 2 && view.state.selection.to <= parse2.to) {
        change.endB += view.state.selection.to - change.endA;
        change.endA = view.state.selection.to;
      }
    }
    if (ie3 && ie_version2 <= 11 && change.endB == change.start + 1 && change.endA == change.start && change.start > parse2.from && parse2.doc.textBetween(change.start - parse2.from - 1, change.start - parse2.from + 1) == " \xA0") {
      change.start--;
      change.endA--;
      change.endB--;
    }
    let $from = parse2.doc.resolveNoCache(change.start - parse2.from);
    let $to = parse2.doc.resolveNoCache(change.endB - parse2.from);
    let $fromA = doc4.resolve(change.start);
    let inlineChange = $from.sameParent($to) && $from.parent.inlineContent && $fromA.end() >= change.endA;
    if ((ios2 && view.input.lastIOSEnter > Date.now() - 225 && (!inlineChange || addedNodes.some((n2) => n2.nodeName == "DIV" || n2.nodeName == "P")) || !inlineChange && $from.pos < parse2.doc.content.size && (!$from.sameParent($to) || !$from.parent.inlineContent) && $from.pos < $to.pos && !/\S/.test(parse2.doc.textBetween($from.pos, $to.pos, "", ""))) && view.someProp("handleKeyDown", (f2) => f2(view, keyEvent(13, "Enter")))) {
      view.input.lastIOSEnter = 0;
      return;
    }
    if (view.state.selection.anchor > change.start && looksLikeBackspace(doc4, change.start, change.endA, $from, $to) && view.someProp("handleKeyDown", (f2) => f2(view, keyEvent(8, "Backspace")))) {
      if (android2 && chrome)
        view.domObserver.suppressSelectionUpdates();
      return;
    }
    if (chrome && change.endB == change.start)
      view.input.lastChromeDelete = Date.now();
    if (android2 && !inlineChange && $from.start() != $to.start() && $to.parentOffset == 0 && $from.depth == $to.depth && parse2.sel && parse2.sel.anchor == parse2.sel.head && parse2.sel.head == change.endA) {
      change.endB -= 2;
      $to = parse2.doc.resolveNoCache(change.endB - parse2.from);
      setTimeout(() => {
        view.someProp("handleKeyDown", function(f2) {
          return f2(view, keyEvent(13, "Enter"));
        });
      }, 20);
    }
    let chFrom = change.start, chTo = change.endA;
    let mkTr = (base2) => {
      let tr = base2 || view.state.tr.replace(chFrom, chTo, parse2.doc.slice(change.start - parse2.from, change.endB - parse2.from));
      if (parse2.sel) {
        let sel2 = resolveSelection(view, tr.doc, parse2.sel);
        if (sel2 && !(chrome && view.composing && sel2.empty && (change.start != change.endB || view.input.lastChromeDelete < Date.now() - 100) && (sel2.head == chFrom || sel2.head == tr.mapping.map(chTo) - 1) || ie3 && sel2.empty && sel2.head == chFrom))
          tr.setSelection(sel2);
      }
      if (compositionID)
        tr.setMeta("composition", compositionID);
      return tr.scrollIntoView();
    };
    let markChange;
    if (inlineChange) {
      if ($from.pos == $to.pos) {
        if (ie3 && ie_version2 <= 11 && $from.parentOffset == 0) {
          view.domObserver.suppressSelectionUpdates();
          setTimeout(() => selectionToDOM(view), 20);
        }
        let tr = mkTr(view.state.tr.delete(chFrom, chTo));
        let marks = doc4.resolve(change.start).marksAcross(doc4.resolve(change.endA));
        if (marks)
          tr.ensureMarks(marks);
        view.dispatch(tr);
      } else if (
        // Adding or removing a mark
        change.endA == change.endB && (markChange = isMarkChange($from.parent.content.cut($from.parentOffset, $to.parentOffset), $fromA.parent.content.cut($fromA.parentOffset, change.endA - $fromA.start())))
      ) {
        let tr = mkTr(view.state.tr);
        if (markChange.type == "add")
          tr.addMark(chFrom, chTo, markChange.mark);
        else
          tr.removeMark(chFrom, chTo, markChange.mark);
        view.dispatch(tr);
      } else if ($from.parent.child($from.index()).isText && $from.index() == $to.index() - ($to.textOffset ? 0 : 1)) {
        let text4 = $from.parent.textBetween($from.parentOffset, $to.parentOffset);
        let deflt = () => mkTr(view.state.tr.insertText(text4, chFrom, chTo));
        if (!view.someProp("handleTextInput", (f2) => f2(view, chFrom, chTo, text4, deflt)))
          view.dispatch(deflt());
      } else {
        view.dispatch(mkTr());
      }
    } else {
      view.dispatch(mkTr());
    }
  }
  function resolveSelection(view, doc4, parsedSel) {
    if (Math.max(parsedSel.anchor, parsedSel.head) > doc4.content.size)
      return null;
    return selectionBetween(view, doc4.resolve(parsedSel.anchor), doc4.resolve(parsedSel.head));
  }
  function isMarkChange(cur, prev) {
    let curMarks = cur.firstChild.marks, prevMarks = prev.firstChild.marks;
    let added = curMarks, removed = prevMarks, type, mark, update;
    for (let i2 = 0; i2 < prevMarks.length; i2++)
      added = prevMarks[i2].removeFromSet(added);
    for (let i2 = 0; i2 < curMarks.length; i2++)
      removed = curMarks[i2].removeFromSet(removed);
    if (added.length == 1 && removed.length == 0) {
      mark = added[0];
      type = "add";
      update = (node2) => node2.mark(mark.addToSet(node2.marks));
    } else if (added.length == 0 && removed.length == 1) {
      mark = removed[0];
      type = "remove";
      update = (node2) => node2.mark(mark.removeFromSet(node2.marks));
    } else {
      return null;
    }
    let updated = [];
    for (let i2 = 0; i2 < prev.childCount; i2++)
      updated.push(update(prev.child(i2)));
    if (Fragment.from(updated).eq(cur))
      return { mark, type };
  }
  function looksLikeBackspace(old, start, end, $newStart, $newEnd) {
    if (
      // The content must have shrunk
      end - start <= $newEnd.pos - $newStart.pos || // newEnd must point directly at or after the end of the block that newStart points into
      skipClosingAndOpening($newStart, true, false) < $newEnd.pos
    )
      return false;
    let $start = old.resolve(start);
    if (!$newStart.parent.isTextblock) {
      let after = $start.nodeAfter;
      return after != null && end == start + after.nodeSize;
    }
    if ($start.parentOffset < $start.parent.content.size || !$start.parent.isTextblock)
      return false;
    let $next = old.resolve(skipClosingAndOpening($start, true, true));
    if (!$next.parent.isTextblock || $next.pos > end || skipClosingAndOpening($next, true, false) < end)
      return false;
    return $newStart.parent.content.cut($newStart.parentOffset).eq($next.parent.content);
  }
  function skipClosingAndOpening($pos, fromEnd, mayOpen) {
    let depth = $pos.depth, end = fromEnd ? $pos.end() : $pos.pos;
    while (depth > 0 && (fromEnd || $pos.indexAfter(depth) == $pos.node(depth).childCount)) {
      depth--;
      end++;
      fromEnd = false;
    }
    if (mayOpen) {
      let next = $pos.node(depth).maybeChild($pos.indexAfter(depth));
      while (next && !next.isLeaf) {
        next = next.firstChild;
        end++;
      }
    }
    return end;
  }
  function findDiff(a2, b, pos, preferredPos, preferredSide) {
    let start = a2.findDiffStart(b, pos);
    if (start == null)
      return null;
    let { a: endA, b: endB } = a2.findDiffEnd(b, pos + a2.size, pos + b.size);
    if (preferredSide == "end") {
      let adjust = Math.max(0, start - Math.min(endA, endB));
      preferredPos -= endA + adjust - start;
    }
    if (endA < start && a2.size < b.size) {
      let move = preferredPos <= start && preferredPos >= endA ? start - preferredPos : 0;
      start -= move;
      if (start && start < b.size && isSurrogatePair(b.textBetween(start - 1, start + 1)))
        start += move ? 1 : -1;
      endB = start + (endB - endA);
      endA = start;
    } else if (endB < start) {
      let move = preferredPos <= start && preferredPos >= endB ? start - preferredPos : 0;
      start -= move;
      if (start && start < a2.size && isSurrogatePair(a2.textBetween(start - 1, start + 1)))
        start += move ? 1 : -1;
      endA = start + (endA - endB);
      endB = start;
    }
    return { start, endA, endB };
  }
  function isSurrogatePair(str) {
    if (str.length != 2)
      return false;
    let a2 = str.charCodeAt(0), b = str.charCodeAt(1);
    return a2 >= 56320 && a2 <= 57343 && b >= 55296 && b <= 56319;
  }
  var EditorView = class {
    /**
    Create a view. `place` may be a DOM node that the editor should
    be appended to, a function that will place it into the document,
    or an object whose `mount` property holds the node to use as the
    document container. If it is `null`, the editor will not be
    added to the document.
    */
    constructor(place, props) {
      this._root = null;
      this.focused = false;
      this.trackWrites = null;
      this.mounted = false;
      this.markCursor = null;
      this.cursorWrapper = null;
      this.lastSelectedViewDesc = void 0;
      this.input = new InputState();
      this.prevDirectPlugins = [];
      this.pluginViews = [];
      this.requiresGeckoHackNode = false;
      this.dragging = null;
      this._props = props;
      this.state = props.state;
      this.directPlugins = props.plugins || [];
      this.directPlugins.forEach(checkStateComponent);
      this.dispatch = this.dispatch.bind(this);
      this.dom = place && place.mount || document.createElement("div");
      if (place) {
        if (place.appendChild)
          place.appendChild(this.dom);
        else if (typeof place == "function")
          place(this.dom);
        else if (place.mount)
          this.mounted = true;
      }
      this.editable = getEditable(this);
      updateCursorWrapper(this);
      this.nodeViews = buildNodeViews(this);
      this.docView = docViewDesc(this.state.doc, computeDocDeco(this), viewDecorations(this), this.dom, this);
      this.domObserver = new DOMObserver(this, (from, to, typeOver, added) => readDOMChange(this, from, to, typeOver, added));
      this.domObserver.start();
      initInput(this);
      this.updatePluginViews();
    }
    /**
    Holds `true` when a
    [composition](https://w3c.github.io/uievents/#events-compositionevents)
    is active.
    */
    get composing() {
      return this.input.composing;
    }
    /**
    The view's current [props](https://prosemirror.net/docs/ref/#view.EditorProps).
    */
    get props() {
      if (this._props.state != this.state) {
        let prev = this._props;
        this._props = {};
        for (let name in prev)
          this._props[name] = prev[name];
        this._props.state = this.state;
      }
      return this._props;
    }
    /**
    Update the view's props. Will immediately cause an update to
    the DOM.
    */
    update(props) {
      if (props.handleDOMEvents != this._props.handleDOMEvents)
        ensureListeners(this);
      let prevProps = this._props;
      this._props = props;
      if (props.plugins) {
        props.plugins.forEach(checkStateComponent);
        this.directPlugins = props.plugins;
      }
      this.updateStateInner(props.state, prevProps);
    }
    /**
    Update the view by updating existing props object with the object
    given as argument. Equivalent to `view.update(Object.assign({},
    view.props, props))`.
    */
    setProps(props) {
      let updated = {};
      for (let name in this._props)
        updated[name] = this._props[name];
      updated.state = this.state;
      for (let name in props)
        updated[name] = props[name];
      this.update(updated);
    }
    /**
    Update the editor's `state` prop, without touching any of the
    other props.
    */
    updateState(state) {
      this.updateStateInner(state, this._props);
    }
    updateStateInner(state, prevProps) {
      var _a;
      let prev = this.state, redraw = false, updateSel = false;
      if (state.storedMarks && this.composing) {
        clearComposition(this);
        updateSel = true;
      }
      this.state = state;
      let pluginsChanged = prev.plugins != state.plugins || this._props.plugins != prevProps.plugins;
      if (pluginsChanged || this._props.plugins != prevProps.plugins || this._props.nodeViews != prevProps.nodeViews) {
        let nodeViews = buildNodeViews(this);
        if (changedNodeViews(nodeViews, this.nodeViews)) {
          this.nodeViews = nodeViews;
          redraw = true;
        }
      }
      if (pluginsChanged || prevProps.handleDOMEvents != this._props.handleDOMEvents) {
        ensureListeners(this);
      }
      this.editable = getEditable(this);
      updateCursorWrapper(this);
      let innerDeco = viewDecorations(this), outerDeco = computeDocDeco(this);
      let scroll = prev.plugins != state.plugins && !prev.doc.eq(state.doc) ? "reset" : state.scrollToSelection > prev.scrollToSelection ? "to selection" : "preserve";
      let updateDoc = redraw || !this.docView.matchesNode(state.doc, outerDeco, innerDeco);
      if (updateDoc || !state.selection.eq(prev.selection))
        updateSel = true;
      let oldScrollPos = scroll == "preserve" && updateSel && this.dom.style.overflowAnchor == null && storeScrollPos(this);
      if (updateSel) {
        this.domObserver.stop();
        let forceSelUpdate = updateDoc && (ie3 || chrome) && !this.composing && !prev.selection.empty && !state.selection.empty && selectionContextChanged(prev.selection, state.selection);
        if (updateDoc) {
          let chromeKludge = chrome ? this.trackWrites = this.domSelectionRange().focusNode : null;
          if (this.composing)
            this.input.compositionNode = findCompositionNode(this);
          if (redraw || !this.docView.update(state.doc, outerDeco, innerDeco, this)) {
            this.docView.updateOuterDeco(outerDeco);
            this.docView.destroy();
            this.docView = docViewDesc(state.doc, outerDeco, innerDeco, this.dom, this);
          }
          if (chromeKludge && (!this.trackWrites || !this.dom.contains(this.trackWrites)))
            forceSelUpdate = true;
        }
        if (forceSelUpdate || !(this.input.mouseDown && this.domObserver.currentSelection.eq(this.domSelectionRange()) && anchorInRightPlace(this))) {
          selectionToDOM(this, forceSelUpdate);
        } else {
          syncNodeSelection(this, state.selection);
          this.domObserver.setCurSelection();
        }
        this.domObserver.start();
      }
      this.updatePluginViews(prev);
      if (((_a = this.dragging) === null || _a === void 0 ? void 0 : _a.node) && !prev.doc.eq(state.doc))
        this.updateDraggedNode(this.dragging, prev);
      if (scroll == "reset") {
        this.dom.scrollTop = 0;
      } else if (scroll == "to selection") {
        this.scrollToSelection();
      } else if (oldScrollPos) {
        resetScrollPos(oldScrollPos);
      }
    }
    /**
    @internal
    */
    scrollToSelection() {
      let startDOM = this.domSelectionRange().focusNode;
      if (!startDOM || !this.dom.contains(startDOM.nodeType == 1 ? startDOM : startDOM.parentNode)) ;
      else if (this.someProp("handleScrollToSelection", (f2) => f2(this))) ;
      else if (this.state.selection instanceof NodeSelection) {
        let target = this.docView.domAfterPos(this.state.selection.from);
        if (target.nodeType == 1)
          scrollRectIntoView(this, target.getBoundingClientRect(), startDOM);
      } else {
        scrollRectIntoView(this, this.coordsAtPos(this.state.selection.head, 1), startDOM);
      }
    }
    destroyPluginViews() {
      let view;
      while (view = this.pluginViews.pop())
        if (view.destroy)
          view.destroy();
    }
    updatePluginViews(prevState) {
      if (!prevState || prevState.plugins != this.state.plugins || this.directPlugins != this.prevDirectPlugins) {
        this.prevDirectPlugins = this.directPlugins;
        this.destroyPluginViews();
        for (let i2 = 0; i2 < this.directPlugins.length; i2++) {
          let plugin = this.directPlugins[i2];
          if (plugin.spec.view)
            this.pluginViews.push(plugin.spec.view(this));
        }
        for (let i2 = 0; i2 < this.state.plugins.length; i2++) {
          let plugin = this.state.plugins[i2];
          if (plugin.spec.view)
            this.pluginViews.push(plugin.spec.view(this));
        }
      } else {
        for (let i2 = 0; i2 < this.pluginViews.length; i2++) {
          let pluginView = this.pluginViews[i2];
          if (pluginView.update)
            pluginView.update(this, prevState);
        }
      }
    }
    updateDraggedNode(dragging, prev) {
      let sel = dragging.node, found2 = -1;
      if (this.state.doc.nodeAt(sel.from) == sel.node) {
        found2 = sel.from;
      } else {
        let movedPos = sel.from + (this.state.doc.content.size - prev.doc.content.size);
        let moved = movedPos > 0 && this.state.doc.nodeAt(movedPos);
        if (moved == sel.node)
          found2 = movedPos;
      }
      this.dragging = new Dragging(dragging.slice, dragging.move, found2 < 0 ? void 0 : NodeSelection.create(this.state.doc, found2));
    }
    someProp(propName, f2) {
      let prop = this._props && this._props[propName], value;
      if (prop != null && (value = f2 ? f2(prop) : prop))
        return value;
      for (let i2 = 0; i2 < this.directPlugins.length; i2++) {
        let prop2 = this.directPlugins[i2].props[propName];
        if (prop2 != null && (value = f2 ? f2(prop2) : prop2))
          return value;
      }
      let plugins2 = this.state.plugins;
      if (plugins2)
        for (let i2 = 0; i2 < plugins2.length; i2++) {
          let prop2 = plugins2[i2].props[propName];
          if (prop2 != null && (value = f2 ? f2(prop2) : prop2))
            return value;
        }
    }
    /**
    Query whether the view has focus.
    */
    hasFocus() {
      if (ie3) {
        let node2 = this.root.activeElement;
        if (node2 == this.dom)
          return true;
        if (!node2 || !this.dom.contains(node2))
          return false;
        while (node2 && this.dom != node2 && this.dom.contains(node2)) {
          if (node2.contentEditable == "false")
            return false;
          node2 = node2.parentElement;
        }
        return true;
      }
      return this.root.activeElement == this.dom;
    }
    /**
    Focus the editor.
    */
    focus() {
      this.domObserver.stop();
      if (this.editable)
        focusPreventScroll(this.dom);
      selectionToDOM(this);
      this.domObserver.start();
    }
    /**
    Get the document root in which the editor exists. This will
    usually be the top-level `document`, but might be a [shadow
    DOM](https://developer.mozilla.org/en-US/docs/Web/Web_Components/Shadow_DOM)
    root if the editor is inside one.
    */
    get root() {
      let cached = this._root;
      if (cached == null)
        for (let search2 = this.dom.parentNode; search2; search2 = search2.parentNode) {
          if (search2.nodeType == 9 || search2.nodeType == 11 && search2.host) {
            if (!search2.getSelection)
              Object.getPrototypeOf(search2).getSelection = () => search2.ownerDocument.getSelection();
            return this._root = search2;
          }
        }
      return cached || document;
    }
    /**
    When an existing editor view is moved to a new document or
    shadow tree, call this to make it recompute its root.
    */
    updateRoot() {
      this._root = null;
    }
    /**
    Given a pair of viewport coordinates, return the document
    position that corresponds to them. May return null if the given
    coordinates aren't inside of the editor. When an object is
    returned, its `pos` property is the position nearest to the
    coordinates, and its `inside` property holds the position of the
    inner node that the position falls inside of, or -1 if it is at
    the top level, not in any node.
    */
    posAtCoords(coords) {
      return posAtCoords(this, coords);
    }
    /**
    Returns the viewport rectangle at a given document position.
    `left` and `right` will be the same number, as this returns a
    flat cursor-ish rectangle. If the position is between two things
    that aren't directly adjacent, `side` determines which element
    is used. When < 0, the element before the position is used,
    otherwise the element after.
    */
    coordsAtPos(pos, side = 1) {
      return coordsAtPos(this, pos, side);
    }
    /**
    Find the DOM position that corresponds to the given document
    position. When `side` is negative, find the position as close as
    possible to the content before the position. When positive,
    prefer positions close to the content after the position. When
    zero, prefer as shallow a position as possible.
    
    Note that you should **not** mutate the editor's internal DOM,
    only inspect it (and even that is usually not necessary).
    */
    domAtPos(pos, side = 0) {
      return this.docView.domFromPos(pos, side);
    }
    /**
    Find the DOM node that represents the document node after the
    given position. May return `null` when the position doesn't point
    in front of a node or if the node is inside an opaque node view.
    
    This is intended to be able to call things like
    `getBoundingClientRect` on that DOM node. Do **not** mutate the
    editor DOM directly, or add styling this way, since that will be
    immediately overriden by the editor as it redraws the node.
    */
    nodeDOM(pos) {
      let desc = this.docView.descAt(pos);
      return desc ? desc.nodeDOM : null;
    }
    /**
    Find the document position that corresponds to a given DOM
    position. (Whenever possible, it is preferable to inspect the
    document structure directly, rather than poking around in the
    DOM, but sometimes—for example when interpreting an event
    target—you don't have a choice.)
    
    The `bias` parameter can be used to influence which side of a DOM
    node to use when the position is inside a leaf node.
    */
    posAtDOM(node2, offset, bias = -1) {
      let pos = this.docView.posFromDOM(node2, offset, bias);
      if (pos == null)
        throw new RangeError("DOM position not inside the editor");
      return pos;
    }
    /**
    Find out whether the selection is at the end of a textblock when
    moving in a given direction. When, for example, given `"left"`,
    it will return true if moving left from the current cursor
    position would leave that position's parent textblock. Will apply
    to the view's current state by default, but it is possible to
    pass a different state.
    */
    endOfTextblock(dir, state) {
      return endOfTextblock(this, state || this.state, dir);
    }
    /**
    Run the editor's paste logic with the given HTML string. The
    `event`, if given, will be passed to the
    [`handlePaste`](https://prosemirror.net/docs/ref/#view.EditorProps.handlePaste) hook.
    */
    pasteHTML(html2, event) {
      return doPaste(this, "", html2, false, event || new ClipboardEvent("paste"));
    }
    /**
    Run the editor's paste logic with the given plain-text input.
    */
    pasteText(text4, event) {
      return doPaste(this, text4, null, true, event || new ClipboardEvent("paste"));
    }
    /**
    Serialize the given slice as it would be if it was copied from
    this editor. Returns a DOM element that contains a
    representation of the slice as its children, a textual
    representation, and the transformed slice (which can be
    different from the given input due to hooks like
    [`transformCopied`](https://prosemirror.net/docs/ref/#view.EditorProps.transformCopied)).
    */
    serializeForClipboard(slice) {
      return serializeForClipboard(this, slice);
    }
    /**
    Removes the editor from the DOM and destroys all [node
    views](https://prosemirror.net/docs/ref/#view.NodeView).
    */
    destroy() {
      if (!this.docView)
        return;
      destroyInput(this);
      this.destroyPluginViews();
      if (this.mounted) {
        this.docView.update(this.state.doc, [], viewDecorations(this), this);
        this.dom.textContent = "";
      } else if (this.dom.parentNode) {
        this.dom.parentNode.removeChild(this.dom);
      }
      this.docView.destroy();
      this.docView = null;
      clearReusedRange();
    }
    /**
    This is true when the view has been
    [destroyed](https://prosemirror.net/docs/ref/#view.EditorView.destroy) (and thus should not be
    used anymore).
    */
    get isDestroyed() {
      return this.docView == null;
    }
    /**
    Used for testing.
    */
    dispatchEvent(event) {
      return dispatchEvent2(this, event);
    }
    /**
    @internal
    */
    domSelectionRange() {
      let sel = this.domSelection();
      if (!sel)
        return { focusNode: null, focusOffset: 0, anchorNode: null, anchorOffset: 0 };
      return safari2 && this.root.nodeType === 11 && deepActiveElement(this.dom.ownerDocument) == this.dom && safariShadowSelectionRange(this, sel) || sel;
    }
    /**
    @internal
    */
    domSelection() {
      return this.root.getSelection();
    }
  };
  EditorView.prototype.dispatch = function(tr) {
    let dispatchTransaction = this._props.dispatchTransaction;
    if (dispatchTransaction)
      dispatchTransaction.call(this, tr);
    else
      this.updateState(this.state.apply(tr));
  };
  function computeDocDeco(view) {
    let attrs = /* @__PURE__ */ Object.create(null);
    attrs.class = "ProseMirror";
    attrs.contenteditable = String(view.editable);
    view.someProp("attributes", (value) => {
      if (typeof value == "function")
        value = value(view.state);
      if (value)
        for (let attr in value) {
          if (attr == "class")
            attrs.class += " " + value[attr];
          else if (attr == "style")
            attrs.style = (attrs.style ? attrs.style + ";" : "") + value[attr];
          else if (!attrs[attr] && attr != "contenteditable" && attr != "nodeName")
            attrs[attr] = String(value[attr]);
        }
    });
    if (!attrs.translate)
      attrs.translate = "no";
    return [Decoration.node(0, view.state.doc.content.size, attrs)];
  }
  function updateCursorWrapper(view) {
    if (view.markCursor) {
      let dom = document.createElement("img");
      dom.className = "ProseMirror-separator";
      dom.setAttribute("mark-placeholder", "true");
      dom.setAttribute("alt", "");
      view.cursorWrapper = { dom, deco: Decoration.widget(view.state.selection.from, dom, { raw: true, marks: view.markCursor }) };
    } else {
      view.cursorWrapper = null;
    }
  }
  function getEditable(view) {
    return !view.someProp("editable", (value) => value(view.state) === false);
  }
  function selectionContextChanged(sel1, sel2) {
    let depth = Math.min(sel1.$anchor.sharedDepth(sel1.head), sel2.$anchor.sharedDepth(sel2.head));
    return sel1.$anchor.start(depth) != sel2.$anchor.start(depth);
  }
  function buildNodeViews(view) {
    let result = /* @__PURE__ */ Object.create(null);
    function add(obj) {
      for (let prop in obj)
        if (!Object.prototype.hasOwnProperty.call(result, prop))
          result[prop] = obj[prop];
    }
    view.someProp("nodeViews", add);
    view.someProp("markViews", add);
    return result;
  }
  function changedNodeViews(a2, b) {
    let nA = 0, nB = 0;
    for (let prop in a2) {
      if (a2[prop] != b[prop])
        return true;
      nA++;
    }
    for (let _ in b)
      nB++;
    return nA != nB;
  }
  function checkStateComponent(plugin) {
    if (plugin.spec.state || plugin.spec.filterTransaction || plugin.spec.appendTransaction)
      throw new RangeError("Plugins passed directly to the view must not have a state component");
  }

  // node_modules/@milkdown/core/lib/index.js
  function withMeta(plugin, meta) {
    plugin.meta = {
      package: "@milkdown/core",
      group: "System",
      ...meta
    };
    return plugin;
  }
  var remarkHandlers = {
    text: (node2, _, state, info) => {
      const value = node2.value;
      if (/^[^*_\\]*\s+$/.test(value)) {
        return value;
      }
      return state.safe(value, { ...info, encode: [] });
    },
    strong: (node2, _, state, info) => {
      const marker = node2.marker || state.options.strong || "*";
      const exit2 = state.enter("strong");
      const tracker = state.createTracker(info);
      let value = tracker.move(marker + marker);
      value += tracker.move(
        state.containerPhrasing(node2, {
          before: value,
          after: marker,
          ...tracker.current()
        })
      );
      value += tracker.move(marker + marker);
      exit2();
      return value;
    },
    emphasis: (node2, _, state, info) => {
      const marker = node2.marker || state.options.emphasis || "*";
      const exit2 = state.enter("emphasis");
      const tracker = state.createTracker(info);
      let value = tracker.move(marker);
      value += tracker.move(
        state.containerPhrasing(node2, {
          before: value,
          after: marker,
          ...tracker.current()
        })
      );
      value += tracker.move(marker);
      exit2();
      return value;
    }
  };
  var editorViewCtx = createSlice({}, "editorView");
  var editorStateCtx = createSlice({}, "editorState");
  var initTimerCtx = createSlice([], "initTimer");
  var editorCtx = createSlice({}, "editor");
  var inputRulesCtx = createSlice([], "inputRules");
  var prosePluginsCtx = createSlice([], "prosePlugins");
  var remarkPluginsCtx = createSlice(
    [],
    "remarkPlugins"
  );
  var nodeViewCtx = createSlice([], "nodeView");
  var markViewCtx = createSlice([], "markView");
  var remarkCtx = createSlice(
    unified().use(remarkParse).use(remarkStringify),
    "remark"
  );
  var remarkStringifyOptionsCtx = createSlice(
    {
      handlers: remarkHandlers,
      encode: []
    },
    "remarkStringifyOptions"
  );
  var ConfigReady = createTimer("ConfigReady");
  function config(configure3) {
    const plugin = (ctx) => {
      ctx.record(ConfigReady);
      return async () => {
        await configure3(ctx);
        ctx.done(ConfigReady);
        return () => {
          ctx.clearTimer(ConfigReady);
        };
      };
    };
    withMeta(plugin, {
      displayName: "Config"
    });
    return plugin;
  }
  var InitReady = createTimer("InitReady");
  function init(editor) {
    const plugin = (ctx) => {
      ctx.inject(editorCtx, editor).inject(prosePluginsCtx, []).inject(remarkPluginsCtx, []).inject(inputRulesCtx, []).inject(nodeViewCtx, []).inject(markViewCtx, []).inject(remarkStringifyOptionsCtx, {
        handlers: remarkHandlers,
        encode: []
      }).inject(remarkCtx, unified().use(remarkParse).use(remarkStringify)).inject(initTimerCtx, [ConfigReady]).record(InitReady);
      return async () => {
        await ctx.waitTimers(initTimerCtx);
        const options = ctx.get(remarkStringifyOptionsCtx);
        ctx.set(
          remarkCtx,
          unified().use(remarkParse).use(remarkStringify, options)
        );
        ctx.done(InitReady);
        return () => {
          ctx.remove(editorCtx).remove(prosePluginsCtx).remove(remarkPluginsCtx).remove(inputRulesCtx).remove(nodeViewCtx).remove(markViewCtx).remove(remarkStringifyOptionsCtx).remove(remarkCtx).remove(initTimerCtx).clearTimer(InitReady);
        };
      };
    };
    withMeta(plugin, {
      displayName: "Init"
    });
    return plugin;
  }
  var SchemaReady = createTimer("SchemaReady");
  var schemaTimerCtx = createSlice([], "schemaTimer");
  var schemaCtx = createSlice({}, "schema");
  var nodesCtx = createSlice([], "nodes");
  var marksCtx = createSlice([], "marks");
  function extendPriority(x) {
    return {
      ...x,
      parseDOM: x.parseDOM?.map((rule) => ({ priority: x.priority, ...rule }))
    };
  }
  var schema = (ctx) => {
    ctx.inject(schemaCtx, {}).inject(nodesCtx, []).inject(marksCtx, []).inject(schemaTimerCtx, [InitReady]).record(SchemaReady);
    return async () => {
      await ctx.waitTimers(schemaTimerCtx);
      const remark = ctx.get(remarkCtx);
      const remarkPlugins = ctx.get(remarkPluginsCtx);
      const processor = remarkPlugins.reduce(
        (acc, plug) => acc.use(plug.plugin, plug.options),
        remark
      );
      ctx.set(remarkCtx, processor);
      const nodes = Object.fromEntries(
        ctx.get(nodesCtx).map(([key22, x]) => [key22, extendPriority(x)])
      );
      const marks = Object.fromEntries(
        ctx.get(marksCtx).map(([key22, x]) => [key22, extendPriority(x)])
      );
      const schema22 = new Schema({ nodes, marks });
      ctx.set(schemaCtx, schema22);
      ctx.done(SchemaReady);
      return () => {
        ctx.remove(schemaCtx).remove(nodesCtx).remove(marksCtx).remove(schemaTimerCtx).clearTimer(SchemaReady);
      };
    };
  };
  withMeta(schema, {
    displayName: "Schema"
  });
  var _container3, _ctx;
  var CommandManager = class {
    constructor() {
      __privateAdd(this, _container3);
      __privateAdd(this, _ctx);
      __privateSet(this, _container3, new Container());
      __privateSet(this, _ctx, null);
      this.setCtx = (ctx) => {
        __privateSet(this, _ctx, ctx);
      };
      this.chain = () => {
        if (__privateGet(this, _ctx) == null) throw callCommandBeforeEditorView();
        const ctx = __privateGet(this, _ctx);
        const commands22 = [];
        const get = this.get.bind(this);
        const chains = {
          run: () => {
            const chained = chainCommands(...commands22);
            const view = ctx.get(editorViewCtx);
            return chained(view.state, view.dispatch, view);
          },
          inline: (command) => {
            commands22.push(command);
            return chains;
          },
          pipe: pipe.bind(this)
        };
        function pipe(slice, payload) {
          const cmd = get(slice);
          commands22.push(cmd(payload));
          return chains;
        }
        return chains;
      };
    }
    get ctx() {
      return __privateGet(this, _ctx);
    }
    /// Register a command into the manager.
    create(meta, value) {
      const slice = meta.create(__privateGet(this, _container3).sliceMap);
      slice.set(value);
      return slice;
    }
    get(slice) {
      return __privateGet(this, _container3).get(slice).get();
    }
    remove(slice) {
      return __privateGet(this, _container3).remove(slice);
    }
    call(slice, payload) {
      if (__privateGet(this, _ctx) == null) throw callCommandBeforeEditorView();
      const cmd = this.get(slice);
      const command = cmd(payload);
      const view = __privateGet(this, _ctx).get(editorViewCtx);
      return command(view.state, view.dispatch, view);
    }
    /// Call an inline command.
    inline(command) {
      if (__privateGet(this, _ctx) == null) throw callCommandBeforeEditorView();
      const view = __privateGet(this, _ctx).get(editorViewCtx);
      return command(view.state, view.dispatch, view);
    }
  };
  _container3 = new WeakMap();
  _ctx = new WeakMap();
  function createCmdKey(key22 = "cmdKey") {
    return createSlice((() => () => false), key22);
  }
  var commandsCtx = createSlice(new CommandManager(), "commands");
  var commandsTimerCtx = createSlice([SchemaReady], "commandsTimer");
  var CommandsReady = createTimer("CommandsReady");
  var commands = (ctx) => {
    const cmd = new CommandManager();
    cmd.setCtx(ctx);
    ctx.inject(commandsCtx, cmd).inject(commandsTimerCtx, [SchemaReady]).record(CommandsReady);
    return async () => {
      await ctx.waitTimers(commandsTimerCtx);
      ctx.done(CommandsReady);
      return () => {
        ctx.remove(commandsCtx).remove(commandsTimerCtx).clearTimer(CommandsReady);
      };
    };
  };
  withMeta(commands, {
    displayName: "Commands"
  });
  function overrideBaseKeymap(keymap22) {
    const handleBackspace = chainCommands(
      undoInputRule,
      deleteSelection,
      joinTextblockBackward,
      selectNodeBackward
    );
    keymap22.Backspace = handleBackspace;
    return keymap22;
  }
  var _ctx2, _keymap;
  var KeymapManager = class {
    constructor() {
      __privateAdd(this, _ctx2);
      __privateAdd(this, _keymap);
      __privateSet(this, _ctx2, null);
      __privateSet(this, _keymap, []);
      this.setCtx = (ctx) => {
        __privateSet(this, _ctx2, ctx);
      };
      this.add = (keymap22) => {
        __privateGet(this, _keymap).push(keymap22);
        return () => {
          __privateSet(this, _keymap, __privateGet(this, _keymap).filter((item) => item !== keymap22));
        };
      };
      this.addObjectKeymap = (keymaps) => {
        const remove = [];
        Object.entries(keymaps).forEach(([key22, command]) => {
          if (typeof command === "function") {
            const keymapItem = {
              key: key22,
              onRun: () => command
            };
            __privateGet(this, _keymap).push(keymapItem);
            remove.push(() => {
              __privateSet(this, _keymap, __privateGet(this, _keymap).filter((item) => item !== keymapItem));
            });
          } else {
            __privateGet(this, _keymap).push(command);
            remove.push(() => {
              __privateSet(this, _keymap, __privateGet(this, _keymap).filter((item) => item !== command));
            });
          }
        });
        return () => {
          remove.forEach((fn) => fn());
        };
      };
      this.addBaseKeymap = () => {
        const base2 = overrideBaseKeymap(baseKeymap);
        return this.addObjectKeymap(base2);
      };
      this.build = () => {
        const keymap22 = {};
        __privateGet(this, _keymap).forEach((item) => {
          keymap22[item.key] = [...keymap22[item.key] || [], item];
        });
        const output = Object.fromEntries(
          Object.entries(keymap22).map(([key22, items]) => {
            const sortedItems = items.sort(
              (a2, b) => (b.priority ?? 50) - (a2.priority ?? 50)
            );
            const command = (state, dispatch, view) => {
              const ctx = __privateGet(this, _ctx2);
              if (ctx == null) throw ctxCallOutOfScope();
              const commands22 = sortedItems.map((item) => item.onRun(ctx));
              const chained = chainCommands(...commands22);
              return chained(state, dispatch, view);
            };
            return [key22, command];
          })
        );
        return output;
      };
    }
    get ctx() {
      return __privateGet(this, _ctx2);
    }
  };
  _ctx2 = new WeakMap();
  _keymap = new WeakMap();
  var keymapCtx = createSlice(new KeymapManager(), "keymap");
  var keymapTimerCtx = createSlice([SchemaReady], "keymapTimer");
  var KeymapReady = createTimer("KeymapReady");
  var keymap2 = (ctx) => {
    const km = new KeymapManager();
    km.setCtx(ctx);
    ctx.inject(keymapCtx, km).inject(keymapTimerCtx, [SchemaReady]).record(KeymapReady);
    return async () => {
      await ctx.waitTimers(keymapTimerCtx);
      ctx.done(KeymapReady);
      return () => {
        ctx.remove(keymapCtx).remove(keymapTimerCtx).clearTimer(KeymapReady);
      };
    };
  };
  var ParserReady = createTimer("ParserReady");
  var outOfScope$1 = (() => {
    throw ctxCallOutOfScope();
  });
  var parserCtx = createSlice(outOfScope$1, "parser");
  var parserTimerCtx = createSlice([], "parserTimer");
  var parser = (ctx) => {
    ctx.inject(parserCtx, outOfScope$1).inject(parserTimerCtx, [SchemaReady]).record(ParserReady);
    return async () => {
      await ctx.waitTimers(parserTimerCtx);
      const remark = ctx.get(remarkCtx);
      const schema22 = ctx.get(schemaCtx);
      ctx.set(parserCtx, ParserState.create(schema22, remark));
      ctx.done(ParserReady);
      return () => {
        ctx.remove(parserCtx).remove(parserTimerCtx).clearTimer(ParserReady);
      };
    };
  };
  withMeta(parser, {
    displayName: "Parser"
  });
  var SerializerReady = createTimer("SerializerReady");
  var serializerTimerCtx = createSlice(
    [],
    "serializerTimer"
  );
  var outOfScope = (() => {
    throw ctxCallOutOfScope();
  });
  var serializerCtx = createSlice(
    outOfScope,
    "serializer"
  );
  var serializer = (ctx) => {
    ctx.inject(serializerCtx, outOfScope).inject(serializerTimerCtx, [SchemaReady]).record(SerializerReady);
    return async () => {
      await ctx.waitTimers(serializerTimerCtx);
      const remark = ctx.get(remarkCtx);
      const schema22 = ctx.get(schemaCtx);
      ctx.set(serializerCtx, SerializerState.create(schema22, remark));
      ctx.done(SerializerReady);
      return () => {
        ctx.remove(serializerCtx).remove(serializerTimerCtx).clearTimer(SerializerReady);
      };
    };
  };
  withMeta(serializer, {
    displayName: "Serializer"
  });
  var defaultValueCtx = createSlice("", "defaultValue");
  var editorStateOptionsCtx = createSlice(
    (x) => x,
    "stateOptions"
  );
  var editorStateTimerCtx = createSlice(
    [],
    "editorStateTimer"
  );
  var EditorStateReady = createTimer("EditorStateReady");
  function getDoc(defaultValue, parser2, schema22) {
    if (typeof defaultValue === "string") return parser2(defaultValue);
    if (defaultValue.type === "html")
      return DOMParser.fromSchema(schema22).parse(defaultValue.dom);
    if (defaultValue.type === "json")
      return Node.fromJSON(schema22, defaultValue.value);
    throw docTypeError(defaultValue);
  }
  var key$1 = new PluginKey("MILKDOWN_STATE_TRACKER");
  var editorState = (ctx) => {
    ctx.inject(defaultValueCtx, "").inject(editorStateCtx, {}).inject(editorStateOptionsCtx, (x) => x).inject(editorStateTimerCtx, [
      ParserReady,
      SerializerReady,
      CommandsReady,
      KeymapReady
    ]).record(EditorStateReady);
    return async () => {
      await ctx.waitTimers(editorStateTimerCtx);
      const schema22 = ctx.get(schemaCtx);
      const parser2 = ctx.get(parserCtx);
      const rules = ctx.get(inputRulesCtx);
      const optionsOverride = ctx.get(editorStateOptionsCtx);
      const prosePlugins = ctx.get(prosePluginsCtx);
      const defaultValue = ctx.get(defaultValueCtx);
      const doc4 = getDoc(defaultValue, parser2, schema22);
      const km = ctx.get(keymapCtx);
      const disposeBaseKeymap = km.addBaseKeymap();
      const plugins2 = [
        ...prosePlugins,
        new Plugin({
          key: key$1,
          state: {
            init: () => {
            },
            apply: (_tr, _value2, _oldState, newState) => {
              ctx.set(editorStateCtx, newState);
            }
          }
        }),
        customInputRules({ rules }),
        keymap(km.build())
      ];
      ctx.set(prosePluginsCtx, plugins2);
      const options = optionsOverride({
        schema: schema22,
        doc: doc4,
        plugins: plugins2
      });
      const state = EditorState.create(options);
      ctx.set(editorStateCtx, state);
      ctx.done(EditorStateReady);
      return () => {
        disposeBaseKeymap();
        ctx.remove(defaultValueCtx).remove(editorStateCtx).remove(editorStateOptionsCtx).remove(editorStateTimerCtx).clearTimer(EditorStateReady);
      };
    };
  };
  withMeta(editorState, {
    displayName: "EditorState"
  });
  var pasteRulesCtx = createSlice([], "pasteRule");
  var pasteRulesTimerCtx = createSlice([SchemaReady], "pasteRuleTimer");
  var PasteRulesReady = createTimer("PasteRuleReady");
  var pasteRule = (ctx) => {
    ctx.inject(pasteRulesCtx, []).inject(pasteRulesTimerCtx, [SchemaReady]).record(PasteRulesReady);
    return async () => {
      await ctx.waitTimers(pasteRulesTimerCtx);
      ctx.done(PasteRulesReady);
      return () => {
        ctx.remove(pasteRulesCtx).remove(pasteRulesTimerCtx).clearTimer(PasteRulesReady);
      };
    };
  };
  withMeta(pasteRule, {
    displayName: "PasteRule"
  });
  var EditorViewReady = createTimer("EditorViewReady");
  var editorViewTimerCtx = createSlice(
    [],
    "editorViewTimer"
  );
  var editorViewOptionsCtx = createSlice(
    {},
    "editorViewOptions"
  );
  var rootCtx = createSlice(null, "root");
  var rootDOMCtx = createSlice(null, "rootDOM");
  var rootAttrsCtx = createSlice(
    {},
    "rootAttrs"
  );
  function createViewContainer(root3, ctx) {
    const container = document.createElement("div");
    container.className = "milkdown";
    root3.appendChild(container);
    ctx.set(rootDOMCtx, container);
    const attrs = ctx.get(rootAttrsCtx);
    Object.entries(attrs).forEach(
      ([key22, value]) => container.setAttribute(key22, value)
    );
    return container;
  }
  function prepareViewDom(dom) {
    dom.classList.add("editor");
    dom.setAttribute("role", "textbox");
  }
  var key = new PluginKey("MILKDOWN_VIEW_CLEAR");
  var editorView = (ctx) => {
    ctx.inject(rootCtx, document.body).inject(editorViewCtx, {}).inject(editorViewOptionsCtx, {}).inject(rootDOMCtx, null).inject(rootAttrsCtx, {}).inject(editorViewTimerCtx, [EditorStateReady, PasteRulesReady]).record(EditorViewReady);
    return async () => {
      await ctx.wait(InitReady);
      const root3 = ctx.get(rootCtx) || document.body;
      const el = typeof root3 === "string" ? document.querySelector(root3) : root3;
      ctx.update(prosePluginsCtx, (xs) => [
        new Plugin({
          key,
          view: (editorView2) => {
            const container = el ? createViewContainer(el, ctx) : void 0;
            const handleDOM = () => {
              if (container && el) {
                const editor = editorView2.dom;
                el.replaceChild(container, editor);
                container.appendChild(editor);
              }
            };
            handleDOM();
            return {
              destroy: () => {
                if (container?.parentNode)
                  container?.parentNode.replaceChild(editorView2.dom, container);
                container?.remove();
              }
            };
          }
        }),
        ...xs
      ]);
      await ctx.waitTimers(editorViewTimerCtx);
      const state = ctx.get(editorStateCtx);
      const options = ctx.get(editorViewOptionsCtx);
      const nodeViews = Object.fromEntries(ctx.get(nodeViewCtx));
      const markViews = Object.fromEntries(ctx.get(markViewCtx));
      const view = new EditorView(el, {
        state,
        nodeViews,
        markViews,
        transformPasted: (slice, view2, isPlainText) => {
          ctx.get(pasteRulesCtx).sort((a2, b) => (b.priority ?? 50) - (a2.priority ?? 50)).map((rule) => rule.run).forEach((runner) => {
            slice = runner(slice, view2, isPlainText);
          });
          return slice;
        },
        ...options
      });
      prepareViewDom(view.dom);
      ctx.set(editorViewCtx, view);
      ctx.done(EditorViewReady);
      return () => {
        view?.destroy();
        ctx.remove(rootCtx).remove(editorViewCtx).remove(editorViewOptionsCtx).remove(rootDOMCtx).remove(rootAttrsCtx).remove(editorViewTimerCtx).clearTimer(EditorViewReady);
      };
    };
  };
  withMeta(editorView, {
    displayName: "EditorView"
  });
  var _enableInspector, _status2, _configureList, _onStatusChange, _container4, _clock3, _usrPluginStore, _sysPluginStore, _ctx3, _loadInternal, _prepare, _cleanup, _cleanupInternal, _setStatus, _loadPluginInStore;
  var _Editor = class _Editor {
    constructor() {
      __privateAdd(this, _enableInspector);
      __privateAdd(this, _status2);
      __privateAdd(this, _configureList);
      __privateAdd(this, _onStatusChange);
      __privateAdd(this, _container4);
      __privateAdd(this, _clock3);
      __privateAdd(this, _usrPluginStore);
      __privateAdd(this, _sysPluginStore);
      __privateAdd(this, _ctx3);
      __privateAdd(this, _loadInternal);
      __privateAdd(this, _prepare);
      __privateAdd(this, _cleanup);
      __privateAdd(this, _cleanupInternal);
      __privateAdd(this, _setStatus);
      __privateAdd(this, _loadPluginInStore);
      __privateSet(this, _enableInspector, false);
      __privateSet(this, _status2, "Idle");
      __privateSet(this, _configureList, []);
      __privateSet(this, _onStatusChange, () => void 0);
      __privateSet(this, _container4, new Container());
      __privateSet(this, _clock3, new Clock());
      __privateSet(this, _usrPluginStore, /* @__PURE__ */ new Map());
      __privateSet(this, _sysPluginStore, /* @__PURE__ */ new Map());
      __privateSet(this, _ctx3, new Ctx(__privateGet(this, _container4), __privateGet(this, _clock3)));
      __privateSet(this, _loadInternal, () => {
        const configPlugin = config(async (ctx) => {
          await Promise.all(
            __privateGet(this, _configureList).map((fn) => Promise.resolve(fn(ctx)))
          );
        });
        const internalPlugins = [
          schema,
          parser,
          serializer,
          commands,
          keymap2,
          pasteRule,
          editorState,
          editorView,
          init(this),
          configPlugin
        ];
        __privateGet(this, _prepare).call(this, internalPlugins, __privateGet(this, _sysPluginStore));
      });
      __privateSet(this, _prepare, (plugins2, store) => {
        plugins2.forEach((plugin) => {
          const ctx = __privateGet(this, _ctx3).produce(
            __privateGet(this, _enableInspector) ? plugin.meta : void 0
          );
          const handler = plugin(ctx);
          store.set(plugin, { ctx, handler, cleanup: void 0 });
        });
      });
      __privateSet(this, _cleanup, (plugins2, remove = false) => {
        return Promise.all(
          [plugins2].flat().map(async (plugin) => {
            const loader = __privateGet(this, _usrPluginStore).get(plugin);
            const cleanup = loader?.cleanup;
            if (remove) __privateGet(this, _usrPluginStore).delete(plugin);
            else
              __privateGet(this, _usrPluginStore).set(plugin, {
                ctx: void 0,
                handler: void 0,
                cleanup: void 0
              });
            if (typeof cleanup === "function") return cleanup();
            return cleanup;
          })
        );
      });
      __privateSet(this, _cleanupInternal, async () => {
        await Promise.all(
          [...__privateGet(this, _sysPluginStore).entries()].map(async ([_, { cleanup }]) => {
            if (typeof cleanup === "function") return cleanup();
            return cleanup;
          })
        );
        __privateGet(this, _sysPluginStore).clear();
      });
      __privateSet(this, _setStatus, (status) => {
        __privateSet(this, _status2, status);
        __privateGet(this, _onStatusChange).call(this, status);
      });
      __privateSet(this, _loadPluginInStore, (store) => {
        return [...store.entries()].map(async ([key22, loader]) => {
          const { ctx, handler } = loader;
          if (!handler) return;
          const cleanup = await handler();
          store.set(key22, { ctx, handler, cleanup });
        });
      });
      this.enableInspector = (enable = true) => {
        __privateSet(this, _enableInspector, enable);
        return this;
      };
      this.onStatusChange = (onChange) => {
        __privateSet(this, _onStatusChange, onChange);
        return this;
      };
      this.config = (configure3) => {
        __privateGet(this, _configureList).push(configure3);
        return this;
      };
      this.removeConfig = (configure3) => {
        __privateSet(this, _configureList, __privateGet(this, _configureList).filter((x) => x !== configure3));
        return this;
      };
      this.use = (plugins2) => {
        const _plugins = [plugins2].flat();
        _plugins.flat().forEach((plugin) => {
          __privateGet(this, _usrPluginStore).set(plugin, {
            ctx: void 0,
            handler: void 0,
            cleanup: void 0
          });
        });
        if (__privateGet(this, _status2) === "Created")
          __privateGet(this, _prepare).call(this, _plugins, __privateGet(this, _usrPluginStore));
        return this;
      };
      this.remove = async (plugins2) => {
        if (__privateGet(this, _status2) === "OnCreate") {
          console.warn(
            "[Milkdown]: You are trying to remove plugins when the editor is creating, this is not recommended, please check your code."
          );
          return new Promise((resolve) => {
            setTimeout(() => {
              resolve(this.remove(plugins2));
            }, 50);
          });
        }
        await __privateGet(this, _cleanup).call(this, [plugins2].flat(), true);
        return this;
      };
      this.create = async () => {
        if (__privateGet(this, _status2) === "OnCreate") return this;
        if (__privateGet(this, _status2) === "Created") await this.destroy();
        __privateGet(this, _setStatus).call(this, "OnCreate");
        __privateGet(this, _loadInternal).call(this);
        __privateGet(this, _prepare).call(this, [...__privateGet(this, _usrPluginStore).keys()], __privateGet(this, _usrPluginStore));
        await Promise.all(
          [
            __privateGet(this, _loadPluginInStore).call(this, __privateGet(this, _sysPluginStore)),
            __privateGet(this, _loadPluginInStore).call(this, __privateGet(this, _usrPluginStore))
          ].flat()
        );
        __privateGet(this, _setStatus).call(this, "Created");
        return this;
      };
      this.destroy = async (clearPlugins = false) => {
        if (__privateGet(this, _status2) === "Destroyed" || __privateGet(this, _status2) === "OnDestroy")
          return this;
        if (__privateGet(this, _status2) === "OnCreate") {
          return new Promise((resolve) => {
            setTimeout(() => {
              resolve(this.destroy(clearPlugins));
            }, 50);
          });
        }
        if (clearPlugins) __privateSet(this, _configureList, []);
        __privateGet(this, _setStatus).call(this, "OnDestroy");
        await __privateGet(this, _cleanup).call(this, [...__privateGet(this, _usrPluginStore).keys()], clearPlugins);
        await __privateGet(this, _cleanupInternal).call(this);
        __privateGet(this, _setStatus).call(this, "Destroyed");
        return this;
      };
      this.action = (action) => action(__privateGet(this, _ctx3));
      this.inspect = () => {
        if (!__privateGet(this, _enableInspector)) {
          console.warn(
            "[Milkdown]: You are trying to collect inspection when inspector is disabled, please enable inspector by `editor.enableInspector()` first."
          );
          return [];
        }
        return [...__privateGet(this, _sysPluginStore).values(), ...__privateGet(this, _usrPluginStore).values()].map(({ ctx }) => ctx?.inspector?.read()).filter((x) => Boolean(x));
      };
    }
    /// Create a new editor instance.
    static make() {
      return new _Editor();
    }
    /// Get the ctx of the editor.
    get ctx() {
      return __privateGet(this, _ctx3);
    }
    /// Get the status of the editor.
    get status() {
      return __privateGet(this, _status2);
    }
  };
  _enableInspector = new WeakMap();
  _status2 = new WeakMap();
  _configureList = new WeakMap();
  _onStatusChange = new WeakMap();
  _container4 = new WeakMap();
  _clock3 = new WeakMap();
  _usrPluginStore = new WeakMap();
  _sysPluginStore = new WeakMap();
  _ctx3 = new WeakMap();
  _loadInternal = new WeakMap();
  _prepare = new WeakMap();
  _cleanup = new WeakMap();
  _cleanupInternal = new WeakMap();
  _setStatus = new WeakMap();
  _loadPluginInStore = new WeakMap();
  var Editor = _Editor;

  // node_modules/nanoid/index.browser.js
  var random = (bytes) => crypto.getRandomValues(new Uint8Array(bytes));
  var customRandom = (alphabet, defaultSize, getRandom) => {
    let mask = (2 << Math.log2(alphabet.length - 1)) - 1;
    let step = -~(1.6 * mask * defaultSize / alphabet.length);
    return (size = defaultSize) => {
      let id = "";
      while (true) {
        let bytes = getRandom(step);
        let j = step | 0;
        while (j--) {
          id += alphabet[bytes[j] & mask] || "";
          if (id.length >= size) return id;
        }
      }
    };
  };
  var customAlphabet = (alphabet, size = 21) => customRandom(alphabet, size | 0, random);

  // node_modules/@milkdown/utils/lib/index.js
  var nanoid = customAlphabet("abcedfghicklmn", 10);
  function $command(key3, cmd) {
    const cmdKey = createCmdKey(key3);
    const plugin = (ctx) => async () => {
      plugin.key = cmdKey;
      await ctx.wait(CommandsReady);
      const command = cmd(ctx);
      ctx.get(commandsCtx).create(cmdKey, command);
      plugin.run = (payload) => ctx.get(commandsCtx).call(key3, payload);
      return () => {
        ctx.get(commandsCtx).remove(cmdKey);
      };
    };
    return plugin;
  }
  function $inputRule(inputRule) {
    const plugin = (ctx) => async () => {
      await ctx.wait(SchemaReady);
      const ir = inputRule(ctx);
      ctx.update(inputRulesCtx, (irs) => [...irs, ir]);
      plugin.inputRule = ir;
      return () => {
        ctx.update(inputRulesCtx, (irs) => irs.filter((x) => x !== ir));
      };
    };
    return plugin;
  }
  function $mark(id, schema3) {
    const plugin = (ctx) => async () => {
      const markSchema = schema3(ctx);
      ctx.update(marksCtx, (ns) => [
        ...ns.filter((n2) => n2[0] !== id),
        [id, markSchema]
      ]);
      plugin.id = id;
      plugin.schema = markSchema;
      return () => {
        ctx.update(marksCtx, (ns) => ns.filter(([x]) => x !== id));
      };
    };
    plugin.type = (ctx) => {
      const markType = ctx.get(schemaCtx).marks[id];
      if (!markType) throw missingMarkInSchema(id);
      return markType;
    };
    return plugin;
  }
  function $node(id, schema3) {
    const plugin = (ctx) => async () => {
      const nodeSchema = schema3(ctx);
      ctx.update(nodesCtx, (ns) => [
        ...ns.filter((n2) => n2[0] !== id),
        [id, nodeSchema]
      ]);
      plugin.id = id;
      plugin.schema = nodeSchema;
      return () => {
        ctx.update(nodesCtx, (ns) => ns.filter(([x]) => x !== id));
      };
    };
    plugin.type = (ctx) => {
      const nodeType = ctx.get(schemaCtx).nodes[id];
      if (!nodeType) throw missingNodeInSchema(id);
      return nodeType;
    };
    return plugin;
  }
  function $prose(prose) {
    let prosePlugin;
    const plugin = (ctx) => async () => {
      await ctx.wait(SchemaReady);
      prosePlugin = prose(ctx);
      ctx.update(prosePluginsCtx, (ps) => [...ps, prosePlugin]);
      return () => {
        ctx.update(prosePluginsCtx, (ps) => ps.filter((x) => x !== prosePlugin));
      };
    };
    plugin.plugin = () => prosePlugin;
    plugin.key = () => prosePlugin.spec.key;
    return plugin;
  }
  function $shortcut(shortcut) {
    const plugin = (ctx) => async () => {
      await ctx.wait(KeymapReady);
      const km = ctx.get(keymapCtx);
      const keymap4 = shortcut(ctx);
      const dispose = km.addObjectKeymap(keymap4);
      plugin.keymap = keymap4;
      return () => {
        dispose();
      };
    };
    return plugin;
  }
  function $ctx(value, name) {
    const slice = createSlice(value, name);
    const plugin = (ctx) => {
      ctx.inject(slice);
      return () => {
        return () => {
          ctx.remove(slice);
        };
      };
    };
    plugin.key = slice;
    return plugin;
  }
  function $nodeSchema(id, schema3) {
    const schemaCtx2 = $ctx(schema3, id);
    const nodeSchema = $node(id, (ctx) => {
      const userSchema = ctx.get(schemaCtx2.key);
      return userSchema(ctx);
    });
    const result = [schemaCtx2, nodeSchema];
    result.id = nodeSchema.id;
    result.node = nodeSchema;
    result.type = (ctx) => nodeSchema.type(ctx);
    result.ctx = schemaCtx2;
    result.key = schemaCtx2.key;
    result.extendSchema = (handler) => {
      const nextSchema = handler(schema3);
      return $nodeSchema(id, nextSchema);
    };
    return result;
  }
  function $markSchema(id, schema3) {
    const schemaCtx2 = $ctx(schema3, id);
    const markSchema = $mark(id, (ctx) => {
      const userSchema = ctx.get(schemaCtx2.key);
      return userSchema(ctx);
    });
    const result = [schemaCtx2, markSchema];
    result.id = markSchema.id;
    result.mark = markSchema;
    result.type = (ctx) => markSchema.type(ctx);
    result.ctx = schemaCtx2;
    result.key = schemaCtx2.key;
    result.extendSchema = (handler) => {
      const nextSchema = handler(schema3);
      return $markSchema(id, nextSchema);
    };
    return result;
  }
  function $useKeymap(name, userKeymap) {
    const key3 = Object.fromEntries(
      Object.entries(userKeymap).map(
        ([key22, { shortcuts: shortcuts2, priority }]) => {
          return [key22, { shortcuts: shortcuts2, priority }];
        }
      )
    );
    const keymapDef = $ctx(key3, `${name}Keymap`);
    const shortcuts = $shortcut((ctx) => {
      const keys2 = ctx.get(keymapDef.key);
      const keymapTuple = Object.entries(userKeymap).flatMap(
        ([key22, { command }]) => {
          const target = keys2[key22];
          const targetKeys = [target.shortcuts].flat();
          const priority = target.priority;
          return targetKeys.map(
            (targetKey) => [
              targetKey,
              {
                key: targetKey,
                onRun: command,
                priority
              }
            ]
          );
        }
      );
      return Object.fromEntries(keymapTuple);
    });
    const result = [keymapDef, shortcuts];
    result.ctx = keymapDef;
    result.shortcuts = shortcuts;
    result.key = keymapDef.key;
    result.keymap = shortcuts.keymap;
    return result;
  }
  var $nodeAttr = (name, value = () => ({})) => $ctx(value, `${name}Attr`);
  var $markAttr = (name, value = () => ({})) => $ctx(value, `${name}Attr`);
  function $remark(id, remark, initialOptions) {
    const options = $ctx(initialOptions ?? {}, id);
    const plugin = (ctx) => async () => {
      await ctx.wait(InitReady);
      const re = remark(ctx);
      const remarkPlugin = {
        plugin: re,
        options: ctx.get(options.key)
      };
      ctx.update(remarkPluginsCtx, (rp) => [...rp, remarkPlugin]);
      return () => {
        ctx.update(remarkPluginsCtx, (rp) => rp.filter((x) => x !== remarkPlugin));
      };
    };
    const result = [options, plugin];
    result.id = id;
    result.plugin = plugin;
    result.options = options;
    return result;
  }

  // node_modules/prosemirror-schema-list/dist/index.js
  function splitListItem(itemType, itemAttrs) {
    return function(state, dispatch) {
      let { $from, $to, node: node2 } = state.selection;
      if (node2 && node2.isBlock || $from.depth < 2 || !$from.sameParent($to))
        return false;
      let grandParent = $from.node(-1);
      if (grandParent.type != itemType)
        return false;
      if ($from.parent.content.size == 0 && $from.node(-1).childCount == $from.indexAfter(-1)) {
        if ($from.depth == 3 || $from.node(-3).type != itemType || $from.index(-2) != $from.node(-2).childCount - 1)
          return false;
        if (dispatch) {
          let wrap3 = Fragment.empty;
          let depthBefore = $from.index(-1) ? 1 : $from.index(-2) ? 2 : 3;
          for (let d2 = $from.depth - depthBefore; d2 >= $from.depth - 3; d2--)
            wrap3 = Fragment.from($from.node(d2).copy(wrap3));
          let depthAfter = $from.indexAfter(-1) < $from.node(-2).childCount ? 1 : $from.indexAfter(-2) < $from.node(-3).childCount ? 2 : 3;
          wrap3 = wrap3.append(Fragment.from(itemType.createAndFill()));
          let start = $from.before($from.depth - (depthBefore - 1));
          let tr2 = state.tr.replace(start, $from.after(-depthAfter), new Slice2(wrap3, 4 - depthBefore, 0));
          let sel = -1;
          tr2.doc.nodesBetween(start, tr2.doc.content.size, (node3, pos) => {
            if (sel > -1)
              return false;
            if (node3.isTextblock && node3.content.size == 0)
              sel = pos + 1;
          });
          if (sel > -1)
            tr2.setSelection(Selection.near(tr2.doc.resolve(sel)));
          dispatch(tr2.scrollIntoView());
        }
        return true;
      }
      let nextType = $to.pos == $from.end() ? grandParent.contentMatchAt(0).defaultType : null;
      let tr = state.tr.delete($from.pos, $to.pos);
      let types = nextType ? [itemAttrs ? { type: itemType, attrs: itemAttrs } : null, { type: nextType }] : void 0;
      if (!canSplit(tr.doc, $from.pos, 2, types))
        return false;
      if (dispatch)
        dispatch(tr.split($from.pos, 2, types).scrollIntoView());
      return true;
    };
  }
  function liftListItem(itemType) {
    return function(state, dispatch) {
      let { $from, $to } = state.selection;
      let range = $from.blockRange($to, (node2) => node2.childCount > 0 && node2.firstChild.type == itemType);
      if (!range)
        return false;
      if (!dispatch)
        return true;
      if ($from.node(range.depth - 1).type == itemType)
        return liftToOuterList(state, dispatch, itemType, range);
      else
        return liftOutOfList(state, dispatch, range);
    };
  }
  function liftToOuterList(state, dispatch, itemType, range) {
    let tr = state.tr, end = range.end, endOfList = range.$to.end(range.depth);
    if (end < endOfList) {
      tr.step(new ReplaceAroundStep(end - 1, endOfList, end, endOfList, new Slice2(Fragment.from(itemType.create(null, range.parent.copy())), 1, 0), 1, true));
      range = new NodeRange(tr.doc.resolve(range.$from.pos), tr.doc.resolve(endOfList), range.depth);
    }
    const target = liftTarget(range);
    if (target == null)
      return false;
    tr.lift(range, target);
    let $after = tr.doc.resolve(tr.mapping.map(end, -1) - 1);
    if (canJoin(tr.doc, $after.pos) && $after.nodeBefore.type == $after.nodeAfter.type)
      tr.join($after.pos);
    dispatch(tr.scrollIntoView());
    return true;
  }
  function liftOutOfList(state, dispatch, range) {
    let tr = state.tr, list4 = range.parent;
    for (let pos = range.end, i2 = range.endIndex - 1, e2 = range.startIndex; i2 > e2; i2--) {
      pos -= list4.child(i2).nodeSize;
      tr.delete(pos - 1, pos + 1);
    }
    let $start = tr.doc.resolve(range.start), item = $start.nodeAfter;
    if (tr.mapping.map(range.end) != range.start + $start.nodeAfter.nodeSize)
      return false;
    let atStart = range.startIndex == 0, atEnd = range.endIndex == list4.childCount;
    let parent = $start.node(-1), indexBefore = $start.index(-1);
    if (!parent.canReplace(indexBefore + (atStart ? 0 : 1), indexBefore + 1, item.content.append(atEnd ? Fragment.empty : Fragment.from(list4))))
      return false;
    let start = $start.pos, end = start + item.nodeSize;
    tr.step(new ReplaceAroundStep(start - (atStart ? 1 : 0), end + (atEnd ? 1 : 0), start + 1, end - 1, new Slice2((atStart ? Fragment.empty : Fragment.from(list4.copy(Fragment.empty))).append(atEnd ? Fragment.empty : Fragment.from(list4.copy(Fragment.empty))), atStart ? 0 : 1, atEnd ? 0 : 1), atStart ? 0 : 1));
    dispatch(tr.scrollIntoView());
    return true;
  }
  function sinkListItem(itemType) {
    return function(state, dispatch) {
      let { $from, $to } = state.selection;
      let range = $from.blockRange($to, (node2) => node2.childCount > 0 && node2.firstChild.type == itemType);
      if (!range)
        return false;
      let startIndex = range.startIndex;
      if (startIndex == 0)
        return false;
      let parent = range.parent, nodeBefore = parent.child(startIndex - 1);
      if (nodeBefore.type != itemType)
        return false;
      if (dispatch) {
        let nestedBefore = nodeBefore.lastChild && nodeBefore.lastChild.type == parent.type;
        let inner = Fragment.from(nestedBefore ? itemType.create() : null);
        let slice = new Slice2(Fragment.from(itemType.create(null, Fragment.from(parent.type.create(null, inner)))), nestedBefore ? 3 : 1, 0);
        let before = range.start, after = range.end;
        dispatch(state.tr.step(new ReplaceAroundStep(before - (nestedBefore ? 3 : 1), after, before, after, slice, 1, true)).scrollIntoView());
      }
      return true;
    };
  }

  // node_modules/mdast-util-definitions/lib/index.js
  function definitions(tree) {
    const cache = /* @__PURE__ */ new Map();
    if (!tree || !tree.type) {
      throw new Error("mdast-util-definitions expected node");
    }
    visit(tree, "definition", function(definition4) {
      const id = clean(definition4.identifier);
      if (id && !cache.get(id)) {
        cache.set(id, definition4);
      }
    });
    return definition3;
    function definition3(identifier) {
      const id = clean(identifier);
      return cache.get(id);
    }
  }
  function clean(value) {
    return String(value || "").toUpperCase();
  }

  // node_modules/remark-inline-links/lib/index.js
  function remarkInlineLinks() {
    return function(tree) {
      const definition3 = definitions(tree);
      visit(tree, function(node2, index2, parent) {
        if (node2.type === "definition" && parent !== void 0 && typeof index2 === "number") {
          parent.children.splice(index2, 1);
          return [SKIP, index2];
        }
        if (node2.type === "imageReference" || node2.type === "linkReference") {
          const def = definition3(node2.identifier);
          if (def && parent && typeof index2 === "number") {
            parent.children[index2] = node2.type === "imageReference" ? { type: "image", url: def.url, title: def.title, alt: node2.alt } : {
              type: "link",
              url: def.url,
              title: def.title,
              children: node2.children
            };
            return [SKIP, index2];
          }
        }
      });
    };
  }

  // node_modules/@milkdown/preset-commonmark/lib/index.js
  function serializeText(state, node2) {
    const lastIsHardBreak = node2.childCount >= 1 && node2.lastChild?.type.name === "hardbreak";
    if (!lastIsHardBreak) {
      state.next(node2.content);
      return;
    }
    const contentArr = [];
    node2.content.forEach((n2, _, i2) => {
      if (i2 === node2.childCount - 1) return;
      contentArr.push(n2);
    });
    state.next(Fragment.fromArray(contentArr));
  }
  function withMeta2(plugin, meta) {
    Object.assign(plugin, {
      meta: {
        package: "@milkdown/preset-commonmark",
        ...meta
      }
    });
    return plugin;
  }
  var emphasisAttr = $markAttr("emphasis");
  withMeta2(emphasisAttr, {
    displayName: "Attr<emphasis>",
    group: "Emphasis"
  });
  var emphasisSchema = $markSchema("emphasis", (ctx) => ({
    attrs: {
      marker: {
        default: ctx.get(remarkStringifyOptionsCtx).emphasis || "*",
        validate: "string"
      }
    },
    parseDOM: [
      { tag: "i" },
      { tag: "em" },
      { style: "font-style", getAttrs: (value) => value === "italic" }
    ],
    toDOM: (mark) => ["em", ctx.get(emphasisAttr.key)(mark)],
    parseMarkdown: {
      match: (node2) => node2.type === "emphasis",
      runner: (state, node2, markType) => {
        state.openMark(markType, { marker: node2.marker });
        state.next(node2.children);
        state.closeMark(markType);
      }
    },
    toMarkdown: {
      match: (mark) => mark.type.name === "emphasis",
      runner: (state, mark) => {
        state.withMark(mark, "emphasis", void 0, {
          marker: mark.attrs.marker
        });
      }
    }
  }));
  withMeta2(emphasisSchema.mark, {
    displayName: "MarkSchema<emphasis>",
    group: "Emphasis"
  });
  withMeta2(emphasisSchema.ctx, {
    displayName: "MarkSchemaCtx<emphasis>",
    group: "Emphasis"
  });
  var toggleEmphasisCommand = $command("ToggleEmphasis", (ctx) => () => {
    return toggleMark(emphasisSchema.type(ctx));
  });
  withMeta2(toggleEmphasisCommand, {
    displayName: "Command<toggleEmphasisCommand>",
    group: "Emphasis"
  });
  var emphasisStarInputRule = $inputRule((ctx) => {
    return markRule(/(?:^|[^*])\*([^*]+)\*$/, emphasisSchema.type(ctx), {
      getAttr: () => ({
        marker: "*"
      }),
      updateCaptured: ({ fullMatch, start }) => !fullMatch.startsWith("*") ? { fullMatch: fullMatch.slice(1), start: start + 1 } : {}
    });
  });
  withMeta2(emphasisStarInputRule, {
    displayName: "InputRule<emphasis>|Star",
    group: "Emphasis"
  });
  var emphasisUnderscoreInputRule = $inputRule((ctx) => {
    return markRule(/\b_(?![_\s])(.*?[^_\s])_\b/, emphasisSchema.type(ctx), {
      getAttr: () => ({
        marker: "_"
      }),
      updateCaptured: ({ fullMatch, start }) => !fullMatch.startsWith("_") ? { fullMatch: fullMatch.slice(1), start: start + 1 } : {}
    });
  });
  withMeta2(emphasisUnderscoreInputRule, {
    displayName: "InputRule<emphasis>|Underscore",
    group: "Emphasis"
  });
  var emphasisKeymap = $useKeymap("emphasisKeymap", {
    ToggleEmphasis: {
      shortcuts: "Mod-i",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(toggleEmphasisCommand.key);
      }
    }
  });
  withMeta2(emphasisKeymap.ctx, {
    displayName: "KeymapCtx<emphasis>",
    group: "Emphasis"
  });
  withMeta2(emphasisKeymap.shortcuts, {
    displayName: "Keymap<emphasis>",
    group: "Emphasis"
  });
  var strongAttr = $markAttr("strong");
  withMeta2(strongAttr, {
    displayName: "Attr<strong>",
    group: "Strong"
  });
  var strongSchema = $markSchema("strong", (ctx) => ({
    attrs: {
      marker: {
        default: ctx.get(remarkStringifyOptionsCtx).strong || "*",
        validate: "string"
      }
    },
    parseDOM: [
      // This works around a Google Docs misbehavior where
      // pasted content will be inexplicably wrapped in `<b>`
      // tags with a font-weight normal.
      {
        tag: "b",
        getAttrs: (node2) => node2.style.fontWeight != "normal" && null
      },
      { tag: "strong" },
      { style: "font-style", getAttrs: (value) => value === "bold" },
      { style: "font-weight=400", clearMark: (m) => m.type.name == "strong" },
      {
        style: "font-weight",
        getAttrs: (value) => /^(bold(er)?|[5-9]\d{2,})$/.test(value) && null
      }
    ],
    toDOM: (mark) => ["strong", ctx.get(strongAttr.key)(mark)],
    parseMarkdown: {
      match: (node2) => node2.type === "strong",
      runner: (state, node2, markType) => {
        state.openMark(markType, { marker: node2.marker });
        state.next(node2.children);
        state.closeMark(markType);
      }
    },
    toMarkdown: {
      match: (mark) => mark.type.name === "strong",
      runner: (state, mark) => {
        state.withMark(mark, "strong", void 0, {
          marker: mark.attrs.marker
        });
      }
    }
  }));
  withMeta2(strongSchema.mark, {
    displayName: "MarkSchema<strong>",
    group: "Strong"
  });
  withMeta2(strongSchema.ctx, {
    displayName: "MarkSchemaCtx<strong>",
    group: "Strong"
  });
  var toggleStrongCommand = $command("ToggleStrong", (ctx) => () => {
    return toggleMark(strongSchema.type(ctx));
  });
  withMeta2(toggleStrongCommand, {
    displayName: "Command<toggleStrongCommand>",
    group: "Strong"
  });
  var strongInputRule = $inputRule((ctx) => {
    return markRule(
      new RegExp("(?<![\\w:/])(?:\\*\\*|__)([^*_]+?)(?:\\*\\*|__)(?![\\w/])$"),
      strongSchema.type(ctx),
      {
        getAttr: (match) => {
          return {
            marker: match[0].startsWith("*") ? "*" : "_"
          };
        }
      }
    );
  });
  withMeta2(strongInputRule, {
    displayName: "InputRule<strong>",
    group: "Strong"
  });
  var strongKeymap = $useKeymap("strongKeymap", {
    ToggleBold: {
      shortcuts: ["Mod-b"],
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(toggleStrongCommand.key);
      }
    }
  });
  withMeta2(strongKeymap.ctx, {
    displayName: "KeymapCtx<strong>",
    group: "Strong"
  });
  withMeta2(strongKeymap.shortcuts, {
    displayName: "Keymap<strong>",
    group: "Strong"
  });
  var inlineCodeAttr = $markAttr("inlineCode");
  withMeta2(inlineCodeAttr, {
    displayName: "Attr<inlineCode>",
    group: "InlineCode"
  });
  var inlineCodeSchema = $markSchema("inlineCode", (ctx) => ({
    priority: 100,
    code: true,
    parseDOM: [{ tag: "code" }],
    toDOM: (mark) => ["code", ctx.get(inlineCodeAttr.key)(mark)],
    parseMarkdown: {
      match: (node2) => node2.type === "inlineCode",
      runner: (state, node2, markType) => {
        state.openMark(markType);
        state.addText(node2.value);
        state.closeMark(markType);
      }
    },
    toMarkdown: {
      match: (mark) => mark.type.name === "inlineCode",
      runner: (state, mark, node2) => {
        state.withMark(mark, "inlineCode", node2.text || "");
      }
    }
  }));
  withMeta2(inlineCodeSchema.mark, {
    displayName: "MarkSchema<inlineCode>",
    group: "InlineCode"
  });
  withMeta2(inlineCodeSchema.ctx, {
    displayName: "MarkSchemaCtx<inlineCode>",
    group: "InlineCode"
  });
  var toggleInlineCodeCommand = $command(
    "ToggleInlineCode",
    (ctx) => () => (state, dispatch) => {
      const { selection, tr } = state;
      if (selection.empty) return false;
      const { from, to } = selection;
      const has = state.doc.rangeHasMark(from, to, inlineCodeSchema.type(ctx));
      if (has) {
        dispatch?.(tr.removeMark(from, to, inlineCodeSchema.type(ctx)));
        return true;
      }
      const restMarksName = Object.keys(state.schema.marks).filter(
        (x) => x !== inlineCodeSchema.type.name
      );
      restMarksName.map((name) => state.schema.marks[name]).forEach((t2) => {
        tr.removeMark(from, to, t2);
      });
      dispatch?.(tr.addMark(from, to, inlineCodeSchema.type(ctx).create()));
      return true;
    }
  );
  withMeta2(toggleInlineCodeCommand, {
    displayName: "Command<toggleInlineCodeCommand>",
    group: "InlineCode"
  });
  var inlineCodeInputRule = $inputRule((ctx) => {
    return markRule(/(?:`)([^`]+)(?:`)$/, inlineCodeSchema.type(ctx));
  });
  withMeta2(inlineCodeInputRule, {
    displayName: "InputRule<inlineCodeInputRule>",
    group: "InlineCode"
  });
  var inlineCodeKeymap = $useKeymap("inlineCodeKeymap", {
    ToggleInlineCode: {
      shortcuts: "Mod-e",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(toggleInlineCodeCommand.key);
      }
    }
  });
  withMeta2(inlineCodeKeymap.ctx, {
    displayName: "KeymapCtx<inlineCode>",
    group: "InlineCode"
  });
  withMeta2(inlineCodeKeymap.shortcuts, {
    displayName: "Keymap<inlineCode>",
    group: "InlineCode"
  });
  var linkAttr = $markAttr("link");
  withMeta2(linkAttr, {
    displayName: "Attr<link>",
    group: "Link"
  });
  var linkSchema = $markSchema("link", (ctx) => ({
    attrs: {
      href: { validate: "string" },
      title: { default: null, validate: "string|null" }
    },
    parseDOM: [
      {
        tag: "a[href]",
        getAttrs: (dom) => {
          if (!(dom instanceof HTMLElement)) throw expectDomTypeError(dom);
          return {
            href: dom.getAttribute("href"),
            title: dom.getAttribute("title")
          };
        }
      }
    ],
    toDOM: (mark) => ["a", { ...ctx.get(linkAttr.key)(mark), ...mark.attrs }],
    parseMarkdown: {
      match: (node2) => node2.type === "link",
      runner: (state, node2, markType) => {
        const url = node2.url;
        const title = node2.title;
        state.openMark(markType, { href: url, title });
        state.next(node2.children);
        state.closeMark(markType);
      }
    },
    toMarkdown: {
      match: (mark) => mark.type.name === "link",
      runner: (state, mark) => {
        state.withMark(mark, "link", void 0, {
          title: mark.attrs.title,
          url: mark.attrs.href
        });
      }
    }
  }));
  withMeta2(linkSchema.mark, {
    displayName: "MarkSchema<link>",
    group: "Link"
  });
  var toggleLinkCommand = $command(
    "ToggleLink",
    (ctx) => (payload = {}) => toggleMark(linkSchema.type(ctx), payload)
  );
  withMeta2(toggleLinkCommand, {
    displayName: "Command<toggleLinkCommand>",
    group: "Link"
  });
  var updateLinkCommand = $command(
    "UpdateLink",
    (ctx) => (payload = {}) => (state, dispatch) => {
      if (!dispatch) return false;
      let node2;
      let pos = -1;
      const { selection } = state;
      const { from, to } = selection;
      state.doc.nodesBetween(from, from === to ? to + 1 : to, (n2, p2) => {
        if (linkSchema.type(ctx).isInSet(n2.marks)) {
          node2 = n2;
          pos = p2;
          return false;
        }
        return void 0;
      });
      if (!node2) return false;
      const mark = node2.marks.find(({ type }) => type === linkSchema.type(ctx));
      if (!mark) return false;
      const start = pos;
      const end = pos + node2.nodeSize;
      const { tr } = state;
      const linkMark = linkSchema.type(ctx).create({ ...mark.attrs, ...payload });
      if (!linkMark) return false;
      dispatch(
        tr.removeMark(start, end, mark).addMark(start, end, linkMark).setSelection(new TextSelection(tr.selection.$anchor)).scrollIntoView()
      );
      return true;
    }
  );
  withMeta2(updateLinkCommand, {
    displayName: "Command<updateLinkCommand>",
    group: "Link"
  });
  var docSchema = $node("doc", () => ({
    content: "block+",
    parseMarkdown: {
      match: ({ type }) => type === "root",
      runner: (state, node2, type) => {
        state.injectRoot(node2, type);
      }
    },
    toMarkdown: {
      match: (node2) => node2.type.name === "doc",
      runner: (state, node2) => {
        state.openNode("root");
        state.next(node2.content);
      }
    }
  }));
  withMeta2(docSchema, {
    displayName: "NodeSchema<doc>",
    group: "Doc"
  });
  function visitEmptyLine(ast) {
    return visitParents(
      ast,
      (node2) => node2.type === "html" && ["<br />", "<br>", "<br >", "<br/>"].includes(
        node2.value?.trim()
      ),
      (node2, parents) => {
        if (!parents.length) return;
        const parent = parents[parents.length - 1];
        if (!parent) return;
        const index2 = parent.children.indexOf(node2);
        if (index2 === -1) return;
        parent.children.splice(index2, 1);
      },
      true
    );
  }
  var remarkPreserveEmptyLinePlugin = $remark(
    "remark-preserve-empty-line",
    () => () => visitEmptyLine
  );
  withMeta2(remarkPreserveEmptyLinePlugin.plugin, {
    displayName: "Remark<remarkPreserveEmptyLine>",
    group: "Remark"
  });
  withMeta2(remarkPreserveEmptyLinePlugin.options, {
    displayName: "RemarkConfig<remarkPreserveEmptyLine>",
    group: "Remark"
  });
  var paragraphAttr = $nodeAttr("paragraph");
  withMeta2(paragraphAttr, {
    displayName: "Attr<paragraph>",
    group: "Paragraph"
  });
  var paragraphSchema = $nodeSchema("paragraph", (ctx) => ({
    content: "inline*",
    group: "block",
    parseDOM: [{ tag: "p" }],
    toDOM: (node2) => ["p", ctx.get(paragraphAttr.key)(node2), 0],
    parseMarkdown: {
      match: (node2) => node2.type === "paragraph",
      runner: (state, node2, type) => {
        state.openNode(type);
        if (node2.children) state.next(node2.children);
        else state.addText(node2.value || "");
        state.closeNode();
      }
    },
    toMarkdown: {
      match: (node2) => node2.type.name === "paragraph",
      runner: (state, node2) => {
        const view = ctx.get(editorViewCtx);
        const lastNode = view.state?.doc.lastChild;
        state.openNode("paragraph");
        if ((!node2.content || node2.content.size === 0) && node2 !== lastNode && shouldPreserveEmptyLine(ctx)) {
          state.addNode("html", void 0, "<br />");
        } else {
          serializeText(state, node2);
        }
        state.closeNode();
      }
    }
  }));
  function shouldPreserveEmptyLine(ctx) {
    let shouldPreserveEmptyLine2 = false;
    try {
      ctx.get(remarkPreserveEmptyLinePlugin.id);
      shouldPreserveEmptyLine2 = true;
    } catch {
      shouldPreserveEmptyLine2 = false;
    }
    return shouldPreserveEmptyLine2;
  }
  withMeta2(paragraphSchema.node, {
    displayName: "NodeSchema<paragraph>",
    group: "Paragraph"
  });
  withMeta2(paragraphSchema.ctx, {
    displayName: "NodeSchemaCtx<paragraph>",
    group: "Paragraph"
  });
  var turnIntoTextCommand = $command(
    "TurnIntoText",
    (ctx) => () => setBlockType2(paragraphSchema.type(ctx))
  );
  withMeta2(turnIntoTextCommand, {
    displayName: "Command<turnIntoTextCommand>",
    group: "Paragraph"
  });
  var paragraphKeymap = $useKeymap("paragraphKeymap", {
    TurnIntoText: {
      shortcuts: "Mod-Alt-0",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(turnIntoTextCommand.key);
      }
    }
  });
  withMeta2(paragraphKeymap.ctx, {
    displayName: "KeymapCtx<paragraph>",
    group: "Paragraph"
  });
  withMeta2(paragraphKeymap.shortcuts, {
    displayName: "Keymap<paragraph>",
    group: "Paragraph"
  });
  var headingIndex = Array(6).fill(0).map((_, i2) => i2 + 1);
  function defaultHeadingIdGenerator(node2) {
    return node2.textContent.toLowerCase().trim().replace(/\s+/g, "-");
  }
  var headingIdGenerator = $ctx(
    defaultHeadingIdGenerator,
    "headingIdGenerator"
  );
  withMeta2(headingIdGenerator, {
    displayName: "Ctx<HeadingIdGenerator>",
    group: "Heading"
  });
  var headingAttr = $nodeAttr("heading");
  withMeta2(headingAttr, {
    displayName: "Attr<heading>",
    group: "Heading"
  });
  var headingSchema = $nodeSchema("heading", (ctx) => {
    const getId = ctx.get(headingIdGenerator.key);
    return {
      content: "inline*",
      group: "block",
      defining: true,
      attrs: {
        id: {
          default: "",
          validate: "string"
        },
        level: {
          default: 1,
          validate: "number"
        }
      },
      parseDOM: headingIndex.map((x) => ({
        tag: `h${x}`,
        getAttrs: (node2) => {
          if (!(node2 instanceof HTMLElement)) throw expectDomTypeError(node2);
          return { level: x, id: node2.id };
        }
      })),
      toDOM: (node2) => {
        return [
          `h${node2.attrs.level}`,
          {
            ...ctx.get(headingAttr.key)(node2),
            id: node2.attrs.id || getId(node2)
          },
          0
        ];
      },
      parseMarkdown: {
        match: ({ type }) => type === "heading",
        runner: (state, node2, type) => {
          const depth = node2.depth;
          state.openNode(type, { level: depth });
          state.next(node2.children);
          state.closeNode();
        }
      },
      toMarkdown: {
        match: (node2) => node2.type.name === "heading",
        runner: (state, node2) => {
          state.openNode("heading", void 0, { depth: node2.attrs.level });
          serializeText(state, node2);
          state.closeNode();
        }
      }
    };
  });
  withMeta2(headingSchema.node, {
    displayName: "NodeSchema<heading>",
    group: "Heading"
  });
  withMeta2(headingSchema.ctx, {
    displayName: "NodeSchemaCtx<heading>",
    group: "Heading"
  });
  var wrapInHeadingInputRule = $inputRule((ctx) => {
    return textblockTypeInputRule(
      /^(?<hashes>#+)\s$/,
      headingSchema.type(ctx),
      (match) => {
        const x = match.groups?.hashes?.length || 0;
        const view = ctx.get(editorViewCtx);
        const { $from } = view.state.selection;
        const node2 = $from.node();
        if (node2.type.name === "heading") {
          let level = Number(node2.attrs.level) + Number(x);
          if (level > 6) level = 6;
          return { level };
        }
        return { level: x };
      }
    );
  });
  withMeta2(wrapInHeadingInputRule, {
    displayName: "InputRule<wrapInHeadingInputRule>",
    group: "Heading"
  });
  var wrapInHeadingCommand = $command("WrapInHeading", (ctx) => {
    return (level) => {
      level ?? (level = 1);
      if (level < 1) return setBlockType2(paragraphSchema.type(ctx));
      return setBlockType2(headingSchema.type(ctx), { level });
    };
  });
  withMeta2(wrapInHeadingCommand, {
    displayName: "Command<wrapInHeadingCommand>",
    group: "Heading"
  });
  var downgradeHeadingCommand = $command(
    "DowngradeHeading",
    (ctx) => () => (state, dispatch, view) => {
      const { $from } = state.selection;
      const node2 = $from.node();
      if (node2.type !== headingSchema.type(ctx) || !state.selection.empty || $from.parentOffset !== 0)
        return false;
      const level = node2.attrs.level - 1;
      if (!level)
        return setBlockType2(paragraphSchema.type(ctx))(state, dispatch, view);
      dispatch?.(
        state.tr.setNodeMarkup(state.selection.$from.before(), void 0, {
          ...node2.attrs,
          level
        })
      );
      return true;
    }
  );
  withMeta2(downgradeHeadingCommand, {
    displayName: "Command<downgradeHeadingCommand>",
    group: "Heading"
  });
  var headingKeymap = $useKeymap("headingKeymap", {
    TurnIntoH1: {
      shortcuts: "Mod-Alt-1",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInHeadingCommand.key, 1);
      }
    },
    TurnIntoH2: {
      shortcuts: "Mod-Alt-2",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInHeadingCommand.key, 2);
      }
    },
    TurnIntoH3: {
      shortcuts: "Mod-Alt-3",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInHeadingCommand.key, 3);
      }
    },
    TurnIntoH4: {
      shortcuts: "Mod-Alt-4",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInHeadingCommand.key, 4);
      }
    },
    TurnIntoH5: {
      shortcuts: "Mod-Alt-5",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInHeadingCommand.key, 5);
      }
    },
    TurnIntoH6: {
      shortcuts: "Mod-Alt-6",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInHeadingCommand.key, 6);
      }
    },
    DowngradeHeading: {
      shortcuts: ["Delete", "Backspace"],
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(downgradeHeadingCommand.key);
      }
    }
  });
  withMeta2(headingKeymap.ctx, {
    displayName: "KeymapCtx<heading>",
    group: "Heading"
  });
  withMeta2(headingKeymap.shortcuts, {
    displayName: "Keymap<heading>",
    group: "Heading"
  });
  var blockquoteAttr = $nodeAttr("blockquote");
  withMeta2(blockquoteAttr, {
    displayName: "Attr<blockquote>",
    group: "Blockquote"
  });
  var blockquoteSchema = $nodeSchema(
    "blockquote",
    (ctx) => ({
      content: "block+",
      group: "block",
      defining: true,
      parseDOM: [{ tag: "blockquote" }],
      toDOM: (node2) => ["blockquote", ctx.get(blockquoteAttr.key)(node2), 0],
      parseMarkdown: {
        match: ({ type }) => type === "blockquote",
        runner: (state, node2, type) => {
          state.openNode(type).next(node2.children).closeNode();
        }
      },
      toMarkdown: {
        match: (node2) => node2.type.name === "blockquote",
        runner: (state, node2) => {
          state.openNode("blockquote").next(node2.content).closeNode();
        }
      }
    })
  );
  withMeta2(blockquoteSchema.node, {
    displayName: "NodeSchema<blockquote>",
    group: "Blockquote"
  });
  withMeta2(blockquoteSchema.ctx, {
    displayName: "NodeSchemaCtx<blockquote>",
    group: "Blockquote"
  });
  var wrapInBlockquoteInputRule = $inputRule(
    (ctx) => wrappingInputRule(/^\s*>\s$/, blockquoteSchema.type(ctx))
  );
  withMeta2(wrapInBlockquoteInputRule, {
    displayName: "InputRule<wrapInBlockquoteInputRule>",
    group: "Blockquote"
  });
  var wrapInBlockquoteCommand = $command(
    "WrapInBlockquote",
    (ctx) => () => wrapIn(blockquoteSchema.type(ctx))
  );
  withMeta2(wrapInBlockquoteCommand, {
    displayName: "Command<wrapInBlockquoteCommand>",
    group: "Blockquote"
  });
  var blockquoteKeymap = $useKeymap("blockquoteKeymap", {
    WrapInBlockquote: {
      shortcuts: "Mod-Shift-b",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInBlockquoteCommand.key);
      }
    }
  });
  withMeta2(blockquoteKeymap.ctx, {
    displayName: "KeymapCtx<blockquote>",
    group: "Blockquote"
  });
  withMeta2(blockquoteKeymap.shortcuts, {
    displayName: "Keymap<blockquote>",
    group: "Blockquote"
  });
  var codeBlockAttr = $nodeAttr("codeBlock", () => ({
    pre: {},
    code: {}
  }));
  withMeta2(codeBlockAttr, {
    displayName: "Attr<codeBlock>",
    group: "CodeBlock"
  });
  var codeBlockSchema = $nodeSchema("code_block", (ctx) => {
    return {
      content: "text*",
      group: "block",
      marks: "",
      defining: true,
      code: true,
      attrs: {
        language: {
          default: "",
          validate: "string"
        }
      },
      parseDOM: [
        {
          tag: "pre",
          preserveWhitespace: "full",
          getAttrs: (dom) => {
            if (!(dom instanceof HTMLElement)) throw expectDomTypeError(dom);
            return { language: dom.dataset.language };
          }
        }
      ],
      toDOM: (node2) => {
        const attr = ctx.get(codeBlockAttr.key)(node2);
        const language = node2.attrs.language;
        const languageAttrs = language && language.length > 0 ? { "data-language": language } : void 0;
        return [
          "pre",
          {
            ...attr.pre,
            ...languageAttrs
          },
          ["code", attr.code, 0]
        ];
      },
      parseMarkdown: {
        match: ({ type }) => type === "code",
        runner: (state, node2, type) => {
          const language = node2.lang ?? "";
          const value = node2.value;
          state.openNode(type, { language });
          if (value) state.addText(value);
          state.closeNode();
        }
      },
      toMarkdown: {
        match: (node2) => node2.type.name === "code_block",
        runner: (state, node2) => {
          state.addNode("code", void 0, node2.content.firstChild?.text || "", {
            lang: node2.attrs.language
          });
        }
      }
    };
  });
  withMeta2(codeBlockSchema.node, {
    displayName: "NodeSchema<codeBlock>",
    group: "CodeBlock"
  });
  withMeta2(codeBlockSchema.ctx, {
    displayName: "NodeSchemaCtx<codeBlock>",
    group: "CodeBlock"
  });
  var createCodeBlockInputRule = $inputRule(
    (ctx) => textblockTypeInputRule(
      /^```(?<language>[a-z]*)?[\s\n]$/,
      codeBlockSchema.type(ctx),
      (match) => ({
        language: match.groups?.language ?? ""
      })
    )
  );
  withMeta2(createCodeBlockInputRule, {
    displayName: "InputRule<createCodeBlockInputRule>",
    group: "CodeBlock"
  });
  var createCodeBlockCommand = $command(
    "CreateCodeBlock",
    (ctx) => (language = "") => setBlockType2(codeBlockSchema.type(ctx), { language })
  );
  withMeta2(createCodeBlockCommand, {
    displayName: "Command<createCodeBlockCommand>",
    group: "CodeBlock"
  });
  var updateCodeBlockLanguageCommand = $command(
    "UpdateCodeBlockLanguage",
    () => ({ pos, language } = {
      pos: -1,
      language: ""
    }) => (state, dispatch) => {
      if (pos >= 0) {
        dispatch?.(state.tr.setNodeAttribute(pos, "language", language));
        return true;
      }
      return false;
    }
  );
  withMeta2(updateCodeBlockLanguageCommand, {
    displayName: "Command<updateCodeBlockLanguageCommand>",
    group: "CodeBlock"
  });
  var codeBlockKeymap = $useKeymap("codeBlockKeymap", {
    CreateCodeBlock: {
      shortcuts: "Mod-Alt-c",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(createCodeBlockCommand.key);
      }
    }
  });
  withMeta2(codeBlockKeymap.ctx, {
    displayName: "KeymapCtx<codeBlock>",
    group: "CodeBlock"
  });
  withMeta2(codeBlockKeymap.shortcuts, {
    displayName: "Keymap<codeBlock>",
    group: "CodeBlock"
  });
  var imageAttr = $nodeAttr("image");
  withMeta2(imageAttr, {
    displayName: "Attr<image>",
    group: "Image"
  });
  var imageSchema = $nodeSchema("image", (ctx) => {
    return {
      inline: true,
      group: "inline",
      selectable: true,
      draggable: true,
      marks: "",
      atom: true,
      defining: true,
      isolating: true,
      attrs: {
        src: { default: "", validate: "string" },
        alt: { default: "", validate: "string" },
        title: { default: "", validate: "string" }
      },
      parseDOM: [
        {
          tag: "img[src]",
          getAttrs: (dom) => {
            if (!(dom instanceof HTMLElement)) throw expectDomTypeError(dom);
            return {
              src: dom.getAttribute("src") || "",
              alt: dom.getAttribute("alt") || "",
              title: dom.getAttribute("title") || dom.getAttribute("alt") || ""
            };
          }
        }
      ],
      toDOM: (node2) => {
        return ["img", { ...ctx.get(imageAttr.key)(node2), ...node2.attrs }];
      },
      parseMarkdown: {
        match: ({ type }) => type === "image",
        runner: (state, node2, type) => {
          const url = node2.url;
          const alt = node2.alt;
          const title = node2.title;
          state.addNode(type, {
            src: url,
            alt,
            title
          });
        }
      },
      toMarkdown: {
        match: (node2) => node2.type.name === "image",
        runner: (state, node2) => {
          state.addNode("image", void 0, void 0, {
            title: node2.attrs.title,
            url: node2.attrs.src,
            alt: node2.attrs.alt
          });
        }
      }
    };
  });
  withMeta2(imageSchema.node, {
    displayName: "NodeSchema<image>",
    group: "Image"
  });
  withMeta2(imageSchema.ctx, {
    displayName: "NodeSchemaCtx<image>",
    group: "Image"
  });
  var insertImageCommand = $command(
    "InsertImage",
    (ctx) => (payload = {}) => (state, dispatch) => {
      if (!dispatch) return true;
      const { src = "", alt = "", title = "" } = payload;
      const node2 = imageSchema.type(ctx).create({ src, alt, title });
      if (!node2) return true;
      dispatch(state.tr.replaceSelectionWith(node2).scrollIntoView());
      return true;
    }
  );
  withMeta2(insertImageCommand, {
    displayName: "Command<insertImageCommand>",
    group: "Image"
  });
  var updateImageCommand = $command(
    "UpdateImage",
    (ctx) => (payload = {}) => (state, dispatch) => {
      const nodeWithPos = findSelectedNodeOfType(
        state.selection,
        imageSchema.type(ctx)
      );
      if (!nodeWithPos) return false;
      const { node: node2, pos } = nodeWithPos;
      const newAttrs = { ...node2.attrs };
      const { src, alt, title } = payload;
      if (src !== void 0) newAttrs.src = src;
      if (alt !== void 0) newAttrs.alt = alt;
      if (title !== void 0) newAttrs.title = title;
      dispatch?.(
        state.tr.setNodeMarkup(pos, void 0, newAttrs).scrollIntoView()
      );
      return true;
    }
  );
  withMeta2(updateImageCommand, {
    displayName: "Command<updateImageCommand>",
    group: "Image"
  });
  var insertImageInputRule = $inputRule(
    (ctx) => new InputRule(
      /!\[(?<alt>.*?)]\((?<filename>.*?)\s*(?="|\))"?(?<title>[^"]+)?"?\)/,
      (state, match, start, end) => {
        const [matched, alt, src = "", title] = match;
        if (matched)
          return state.tr.replaceWith(
            start,
            end,
            imageSchema.type(ctx).create({ src, alt, title })
          );
        return null;
      }
    )
  );
  withMeta2(insertImageInputRule, {
    displayName: "InputRule<insertImageInputRule>",
    group: "Image"
  });
  var hardbreakAttr = $nodeAttr("hardbreak", (node2) => {
    return {
      "data-type": "hardbreak",
      "data-is-inline": node2.attrs.isInline
    };
  });
  withMeta2(hardbreakAttr, {
    displayName: "Attr<hardbreak>",
    group: "Hardbreak"
  });
  var hardbreakSchema = $nodeSchema("hardbreak", (ctx) => ({
    inline: true,
    group: "inline",
    attrs: {
      isInline: {
        default: false,
        validate: "boolean"
      }
    },
    selectable: false,
    parseDOM: [
      { tag: "br" },
      {
        tag: 'span[data-type="hardbreak"]',
        getAttrs: () => ({ isInline: true })
      }
    ],
    toDOM: (node2) => node2.attrs.isInline ? ["span", ctx.get(hardbreakAttr.key)(node2), " "] : ["br", ctx.get(hardbreakAttr.key)(node2)],
    parseMarkdown: {
      match: ({ type }) => type === "break",
      runner: (state, node2, type) => {
        state.addNode(type, {
          isInline: Boolean(
            node2.data?.isInline
          )
        });
      }
    },
    leafText: () => "\n",
    toMarkdown: {
      match: (node2) => node2.type.name === "hardbreak",
      runner: (state, node2) => {
        if (node2.attrs.isInline) state.addNode("text", void 0, "\n");
        else state.addNode("break");
      }
    }
  }));
  withMeta2(hardbreakSchema.node, {
    displayName: "NodeSchema<hardbreak>",
    group: "Hardbreak"
  });
  withMeta2(hardbreakSchema.ctx, {
    displayName: "NodeSchemaCtx<hardbreak>",
    group: "Hardbreak"
  });
  var insertHardbreakCommand = $command(
    "InsertHardbreak",
    (ctx) => () => (state, dispatch) => {
      const { selection, tr } = state;
      if (!(selection instanceof TextSelection)) return false;
      if (selection.empty) {
        const node2 = selection.$from.node();
        if (node2.childCount > 0 && node2.lastChild?.type.name === "hardbreak") {
          dispatch?.(
            tr.replaceRangeWith(
              selection.to - 1,
              selection.to,
              state.schema.node("paragraph")
            ).setSelection(Selection.near(tr.doc.resolve(selection.to))).scrollIntoView()
          );
          return true;
        }
      }
      dispatch?.(
        tr.setMeta("hardbreak", true).replaceSelectionWith(hardbreakSchema.type(ctx).create()).scrollIntoView()
      );
      return true;
    }
  );
  withMeta2(insertHardbreakCommand, {
    displayName: "Command<insertHardbreakCommand>",
    group: "Hardbreak"
  });
  var hardbreakKeymap = $useKeymap("hardbreakKeymap", {
    InsertHardbreak: {
      shortcuts: "Shift-Enter",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(insertHardbreakCommand.key);
      }
    }
  });
  withMeta2(hardbreakKeymap.ctx, {
    displayName: "KeymapCtx<hardbreak>",
    group: "Hardbreak"
  });
  withMeta2(hardbreakKeymap.shortcuts, {
    displayName: "Keymap<hardbreak>",
    group: "Hardbreak"
  });
  var hrAttr = $nodeAttr("hr");
  withMeta2(hrAttr, {
    displayName: "Attr<hr>",
    group: "Hr"
  });
  var hrSchema = $nodeSchema("hr", (ctx) => ({
    group: "block",
    parseDOM: [{ tag: "hr" }],
    toDOM: (node2) => ["hr", ctx.get(hrAttr.key)(node2)],
    parseMarkdown: {
      match: ({ type }) => type === "thematicBreak",
      runner: (state, _, type) => {
        state.addNode(type);
      }
    },
    toMarkdown: {
      match: (node2) => node2.type.name === "hr",
      runner: (state) => {
        state.addNode("thematicBreak");
      }
    }
  }));
  withMeta2(hrSchema.node, {
    displayName: "NodeSchema<hr>",
    group: "Hr"
  });
  withMeta2(hrSchema.ctx, {
    displayName: "NodeSchemaCtx<hr>",
    group: "Hr"
  });
  var insertHrInputRule = $inputRule(
    (ctx) => new InputRule(/^(?:---|___\s|\*\*\*\s)$/, (state, match, start, end) => {
      const { tr } = state;
      if (match[0]) tr.replaceWith(start - 1, end, hrSchema.type(ctx).create());
      return tr;
    })
  );
  withMeta2(insertHrInputRule, {
    displayName: "InputRule<insertHrInputRule>",
    group: "Hr"
  });
  var insertHrCommand = $command(
    "InsertHr",
    (ctx) => () => (state, dispatch) => {
      if (!dispatch) return true;
      const paragraph2 = paragraphSchema.node.type(ctx).create();
      const { tr, selection } = state;
      const { from } = selection;
      const node2 = hrSchema.type(ctx).create();
      if (!node2) return true;
      const _tr = tr.replaceSelectionWith(node2).insert(from, paragraph2);
      const sel = Selection.findFrom(_tr.doc.resolve(from), 1, true);
      if (!sel) return true;
      dispatch(_tr.setSelection(sel).scrollIntoView());
      return true;
    }
  );
  withMeta2(insertHrCommand, {
    displayName: "Command<insertHrCommand>",
    group: "Hr"
  });
  var bulletListAttr = $nodeAttr("bulletList");
  withMeta2(bulletListAttr, {
    displayName: "Attr<bulletList>",
    group: "BulletList"
  });
  var bulletListSchema = $nodeSchema("bullet_list", (ctx) => {
    return {
      content: "listItem+",
      group: "block",
      attrs: {
        spread: {
          default: false,
          validate: "boolean"
        }
      },
      parseDOM: [
        {
          tag: "ul",
          getAttrs: (dom) => {
            if (!(dom instanceof HTMLElement)) throw expectDomTypeError(dom);
            return {
              spread: dom.dataset.spread === "true"
            };
          }
        }
      ],
      toDOM: (node2) => {
        return [
          "ul",
          {
            ...ctx.get(bulletListAttr.key)(node2),
            "data-spread": node2.attrs.spread
          },
          0
        ];
      },
      parseMarkdown: {
        match: ({ type, ordered }) => type === "list" && !ordered,
        runner: (state, node2, type) => {
          const spread = node2.spread != null ? `${node2.spread}` : "false";
          state.openNode(type, { spread }).next(node2.children).closeNode();
        }
      },
      toMarkdown: {
        match: (node2) => node2.type.name === "bullet_list",
        runner: (state, node2) => {
          state.openNode("list", void 0, {
            ordered: false,
            spread: node2.attrs.spread
          }).next(node2.content).closeNode();
        }
      }
    };
  });
  withMeta2(bulletListSchema.node, {
    displayName: "NodeSchema<bulletList>",
    group: "BulletList"
  });
  withMeta2(bulletListSchema.ctx, {
    displayName: "NodeSchemaCtx<bulletList>",
    group: "BulletList"
  });
  var wrapInBulletListInputRule = $inputRule(
    (ctx) => wrappingInputRule(/^\s*([-+*])\s$/, bulletListSchema.type(ctx))
  );
  withMeta2(wrapInBulletListInputRule, {
    displayName: "InputRule<wrapInBulletListInputRule>",
    group: "BulletList"
  });
  var wrapInBulletListCommand = $command(
    "WrapInBulletList",
    (ctx) => () => wrapIn(bulletListSchema.type(ctx))
  );
  withMeta2(wrapInBulletListCommand, {
    displayName: "Command<wrapInBulletListCommand>",
    group: "BulletList"
  });
  var bulletListKeymap = $useKeymap("bulletListKeymap", {
    WrapInBulletList: {
      shortcuts: "Mod-Alt-8",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInBulletListCommand.key);
      }
    }
  });
  withMeta2(bulletListKeymap.ctx, {
    displayName: "KeymapCtx<bulletListKeymap>",
    group: "BulletList"
  });
  withMeta2(bulletListKeymap.shortcuts, {
    displayName: "Keymap<bulletListKeymap>",
    group: "BulletList"
  });
  var orderedListAttr = $nodeAttr("orderedList");
  withMeta2(orderedListAttr, {
    displayName: "Attr<orderedList>",
    group: "OrderedList"
  });
  var orderedListSchema = $nodeSchema("ordered_list", (ctx) => ({
    content: "listItem+",
    group: "block",
    attrs: {
      order: {
        default: 1,
        validate: "number"
      },
      spread: {
        default: false,
        validate: "boolean"
      }
    },
    parseDOM: [
      {
        tag: "ol",
        getAttrs: (dom) => {
          if (!(dom instanceof HTMLElement)) throw expectDomTypeError(dom);
          return {
            spread: dom.dataset.spread,
            order: dom.hasAttribute("start") ? Number(dom.getAttribute("start")) : 1
          };
        }
      }
    ],
    toDOM: (node2) => [
      "ol",
      {
        ...ctx.get(orderedListAttr.key)(node2),
        ...node2.attrs.order === 1 ? {} : node2.attrs.order,
        "data-spread": node2.attrs.spread
      },
      0
    ],
    parseMarkdown: {
      match: ({ type, ordered }) => type === "list" && !!ordered,
      runner: (state, node2, type) => {
        const spread = node2.spread != null ? `${node2.spread}` : "true";
        state.openNode(type, { spread }).next(node2.children).closeNode();
      }
    },
    toMarkdown: {
      match: (node2) => node2.type.name === "ordered_list",
      runner: (state, node2) => {
        state.openNode("list", void 0, {
          ordered: true,
          start: 1,
          spread: node2.attrs.spread === "true"
        });
        state.next(node2.content);
        state.closeNode();
      }
    }
  }));
  withMeta2(orderedListSchema.node, {
    displayName: "NodeSchema<orderedList>",
    group: "OrderedList"
  });
  withMeta2(orderedListSchema.ctx, {
    displayName: "NodeSchemaCtx<orderedList>",
    group: "OrderedList"
  });
  var wrapInOrderedListInputRule = $inputRule(
    (ctx) => wrappingInputRule(
      /^\s*(\d+)\.\s$/,
      orderedListSchema.type(ctx),
      (match) => ({ order: Number(match[1]) }),
      (match, node2) => node2.childCount + node2.attrs.order === Number(match[1])
    )
  );
  withMeta2(wrapInOrderedListInputRule, {
    displayName: "InputRule<wrapInOrderedListInputRule>",
    group: "OrderedList"
  });
  var wrapInOrderedListCommand = $command(
    "WrapInOrderedList",
    (ctx) => () => wrapIn(orderedListSchema.type(ctx))
  );
  withMeta2(wrapInOrderedListCommand, {
    displayName: "Command<wrapInOrderedListCommand>",
    group: "OrderedList"
  });
  var orderedListKeymap = $useKeymap("orderedListKeymap", {
    WrapInOrderedList: {
      shortcuts: "Mod-Alt-7",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(wrapInOrderedListCommand.key);
      }
    }
  });
  withMeta2(orderedListKeymap.ctx, {
    displayName: "KeymapCtx<orderedList>",
    group: "OrderedList"
  });
  withMeta2(orderedListKeymap.shortcuts, {
    displayName: "Keymap<orderedList>",
    group: "OrderedList"
  });
  var listItemAttr = $nodeAttr("listItem");
  withMeta2(listItemAttr, {
    displayName: "Attr<listItem>",
    group: "ListItem"
  });
  var listItemSchema = $nodeSchema("list_item", (ctx) => ({
    group: "listItem",
    content: "paragraph block*",
    attrs: {
      label: {
        default: "\u2022",
        validate: "string"
      },
      listType: {
        default: "bullet",
        validate: "string"
      },
      spread: {
        default: true,
        validate: "boolean"
      }
    },
    defining: true,
    parseDOM: [
      {
        tag: "li",
        getAttrs: (dom) => {
          if (!(dom instanceof HTMLElement)) throw expectDomTypeError(dom);
          return {
            label: dom.dataset.label,
            listType: dom.dataset.listType,
            spread: dom.dataset.spread === "true"
          };
        }
      }
    ],
    toDOM: (node2) => [
      "li",
      {
        ...ctx.get(listItemAttr.key)(node2),
        "data-label": node2.attrs.label,
        "data-list-type": node2.attrs.listType,
        "data-spread": node2.attrs.spread
      },
      0
    ],
    parseMarkdown: {
      match: ({ type }) => type === "listItem",
      runner: (state, node2, type) => {
        const label = node2.label != null ? `${node2.label}.` : "\u2022";
        const listType = node2.label != null ? "ordered" : "bullet";
        const spread = node2.spread != null ? `${node2.spread}` : "true";
        state.openNode(type, { label, listType, spread });
        state.next(node2.children);
        state.closeNode();
      }
    },
    toMarkdown: {
      match: (node2) => node2.type.name === "list_item",
      runner: (state, node2) => {
        state.openNode("listItem", void 0, {
          spread: node2.attrs.spread
        });
        state.next(node2.content);
        state.closeNode();
      }
    }
  }));
  withMeta2(listItemSchema.node, {
    displayName: "NodeSchema<listItem>",
    group: "ListItem"
  });
  withMeta2(listItemSchema.ctx, {
    displayName: "NodeSchemaCtx<listItem>",
    group: "ListItem"
  });
  var sinkListItemCommand = $command(
    "SinkListItem",
    (ctx) => () => sinkListItem(listItemSchema.type(ctx))
  );
  withMeta2(sinkListItemCommand, {
    displayName: "Command<sinkListItemCommand>",
    group: "ListItem"
  });
  var liftListItemCommand = $command(
    "LiftListItem",
    (ctx) => () => liftListItem(listItemSchema.type(ctx))
  );
  withMeta2(liftListItemCommand, {
    displayName: "Command<liftListItemCommand>",
    group: "ListItem"
  });
  var splitListItemCommand = $command(
    "SplitListItem",
    (ctx) => () => splitListItem(listItemSchema.type(ctx))
  );
  withMeta2(splitListItemCommand, {
    displayName: "Command<splitListItemCommand>",
    group: "ListItem"
  });
  function liftFirstListItem(ctx) {
    return (state, dispatch, view) => {
      const { selection } = state;
      if (!(selection instanceof TextSelection)) return false;
      const { empty: empty3, $from } = selection;
      if (!empty3 || $from.parentOffset !== 0) return false;
      const parentItem = $from.node(-1);
      if (parentItem.type !== listItemSchema.type(ctx)) return false;
      return joinBackward(state, dispatch, view);
    };
  }
  var liftFirstListItemCommand = $command(
    "LiftFirstListItem",
    (ctx) => () => liftFirstListItem(ctx)
  );
  withMeta2(liftFirstListItemCommand, {
    displayName: "Command<liftFirstListItemCommand>",
    group: "ListItem"
  });
  var listItemKeymap = $useKeymap("listItemKeymap", {
    NextListItem: {
      shortcuts: "Enter",
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(splitListItemCommand.key);
      }
    },
    SinkListItem: {
      shortcuts: ["Tab", "Mod-]"],
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(sinkListItemCommand.key);
      }
    },
    LiftListItem: {
      shortcuts: ["Shift-Tab", "Mod-["],
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(liftListItemCommand.key);
      }
    },
    LiftFirstListItem: {
      shortcuts: ["Backspace", "Delete"],
      command: (ctx) => {
        const commands22 = ctx.get(commandsCtx);
        return () => commands22.call(liftFirstListItemCommand.key);
      }
    }
  });
  withMeta2(listItemKeymap.ctx, {
    displayName: "KeymapCtx<listItem>",
    group: "ListItem"
  });
  withMeta2(listItemKeymap.shortcuts, {
    displayName: "Keymap<listItem>",
    group: "ListItem"
  });
  var textSchema = $node("text", () => ({
    group: "inline",
    parseMarkdown: {
      match: ({ type }) => type === "text",
      runner: (state, node2) => {
        state.addText(node2.value);
      }
    },
    toMarkdown: {
      match: (node2) => node2.type.name === "text",
      runner: (state, node2) => {
        state.addNode("text", void 0, node2.text);
      }
    }
  }));
  withMeta2(textSchema, {
    displayName: "NodeSchema<text>",
    group: "Text"
  });
  var htmlAttr = $nodeAttr("html");
  withMeta2(htmlAttr, {
    displayName: "Attr<html>",
    group: "Html"
  });
  var htmlSchema = $nodeSchema("html", (ctx) => {
    return {
      atom: true,
      group: "inline",
      inline: true,
      attrs: {
        value: {
          default: "",
          validate: "string"
        }
      },
      toDOM: (node2) => {
        const span = document.createElement("span");
        const attr = {
          ...ctx.get(htmlAttr.key)(node2),
          "data-value": node2.attrs.value,
          "data-type": "html"
        };
        span.textContent = node2.attrs.value;
        return ["span", attr, node2.attrs.value];
      },
      parseDOM: [
        {
          tag: 'span[data-type="html"]',
          getAttrs: (dom) => {
            return {
              value: dom.dataset.value ?? ""
            };
          }
        }
      ],
      parseMarkdown: {
        match: ({ type }) => Boolean(type === "html"),
        runner: (state, node2, type) => {
          state.addNode(type, { value: node2.value });
        }
      },
      toMarkdown: {
        match: (node2) => node2.type.name === "html",
        runner: (state, node2) => {
          state.addNode("html", void 0, node2.attrs.value);
        }
      }
    };
  });
  withMeta2(htmlSchema.node, {
    displayName: "NodeSchema<html>",
    group: "Html"
  });
  withMeta2(htmlSchema.ctx, {
    displayName: "NodeSchemaCtx<html>",
    group: "Html"
  });
  var schema2 = [
    docSchema,
    paragraphAttr,
    paragraphSchema,
    headingIdGenerator,
    headingAttr,
    headingSchema,
    hardbreakAttr,
    hardbreakSchema,
    blockquoteAttr,
    blockquoteSchema,
    codeBlockAttr,
    codeBlockSchema,
    hrAttr,
    hrSchema,
    imageAttr,
    imageSchema,
    bulletListAttr,
    bulletListSchema,
    orderedListAttr,
    orderedListSchema,
    listItemAttr,
    listItemSchema,
    emphasisAttr,
    emphasisSchema,
    strongAttr,
    strongSchema,
    inlineCodeAttr,
    inlineCodeSchema,
    linkAttr,
    linkSchema,
    htmlAttr,
    htmlSchema,
    textSchema
  ].flat();
  var inputRules = [
    wrapInBlockquoteInputRule,
    wrapInBulletListInputRule,
    wrapInOrderedListInputRule,
    createCodeBlockInputRule,
    insertHrInputRule,
    wrapInHeadingInputRule
  ].flat();
  var markInputRules = [
    emphasisStarInputRule,
    emphasisUnderscoreInputRule,
    inlineCodeInputRule,
    strongInputRule
  ];
  var isMarkSelectedCommand = $command(
    "IsMarkSelected",
    () => (markType) => (state) => {
      if (!markType) return false;
      const { doc: doc4, selection } = state;
      const hasLink = doc4.rangeHasMark(selection.from, selection.to, markType);
      return hasLink;
    }
  );
  var isNodeSelectedCommand = $command(
    "IsNoteSelected",
    () => (nodeType) => (state) => {
      if (!nodeType) return false;
      const result = findNodeInSelection(state, nodeType);
      return result.hasNode;
    }
  );
  var clearTextInCurrentBlockCommand = $command(
    "ClearTextInCurrentBlock",
    () => () => (state, dispatch) => {
      let tr = state.tr;
      const { $from, $to } = tr.selection;
      const { pos: from } = $from;
      const { pos: right } = $to;
      const left = from - $from.node().content.size;
      if (left < 0) return false;
      tr = tr.deleteRange(left, right);
      dispatch?.(tr);
      return true;
    }
  );
  var setBlockTypeCommand = $command(
    "SetBlockType",
    () => (payload) => (state, dispatch) => {
      const { nodeType, attrs = null } = payload ?? {};
      if (!nodeType) return false;
      const tr = state.tr;
      const { from, to } = tr.selection;
      try {
        tr.setBlockType(from, to, nodeType, attrs);
      } catch {
        return false;
      }
      dispatch?.(tr);
      return true;
    }
  );
  var wrapInBlockTypeCommand = $command(
    "WrapInBlockType",
    () => (payload) => (state, dispatch) => {
      const { nodeType, attrs = null } = payload ?? {};
      if (!nodeType) return false;
      let tr = state.tr;
      try {
        const { $from, $to } = tr.selection;
        const blockRange = $from.blockRange($to);
        const wrapping = blockRange && findWrapping(blockRange, nodeType, attrs);
        if (!wrapping) return false;
        tr = tr.wrap(blockRange, wrapping);
      } catch {
        return false;
      }
      dispatch?.(tr);
      return true;
    }
  );
  var addBlockTypeCommand = $command(
    "AddBlockType",
    () => (payload) => (state, dispatch) => {
      const { nodeType, attrs = null } = payload ?? {};
      if (!nodeType) return false;
      const tr = state.tr;
      try {
        const node2 = nodeType instanceof Node ? nodeType : nodeType.createAndFill(attrs);
        if (!node2) return false;
        tr.replaceSelectionWith(node2);
      } catch {
        return false;
      }
      dispatch?.(tr);
      return true;
    }
  );
  var selectTextNearPosCommand = $command(
    "SelectTextNearPos",
    () => (payload) => (state, dispatch) => {
      const { pos } = payload ?? {};
      if (pos == null) return false;
      const clamp = (value, min, max) => Math.min(Math.max(value, min), max);
      const tr = state.tr;
      try {
        const $pos = state.doc.resolve(clamp(pos, 0, state.doc.content.size));
        tr.setSelection(TextSelection.near($pos));
      } catch {
        return false;
      }
      dispatch?.(tr.scrollIntoView());
      return true;
    }
  );
  var commands2 = [
    turnIntoTextCommand,
    wrapInBlockquoteCommand,
    wrapInHeadingCommand,
    downgradeHeadingCommand,
    createCodeBlockCommand,
    insertHardbreakCommand,
    insertHrCommand,
    insertImageCommand,
    updateImageCommand,
    wrapInOrderedListCommand,
    wrapInBulletListCommand,
    sinkListItemCommand,
    splitListItemCommand,
    liftListItemCommand,
    liftFirstListItemCommand,
    toggleEmphasisCommand,
    toggleInlineCodeCommand,
    toggleStrongCommand,
    toggleLinkCommand,
    updateLinkCommand,
    isMarkSelectedCommand,
    isNodeSelectedCommand,
    clearTextInCurrentBlockCommand,
    setBlockTypeCommand,
    wrapInBlockTypeCommand,
    addBlockTypeCommand,
    selectTextNearPosCommand
  ];
  var keymap3 = [
    blockquoteKeymap,
    codeBlockKeymap,
    hardbreakKeymap,
    headingKeymap,
    listItemKeymap,
    orderedListKeymap,
    bulletListKeymap,
    paragraphKeymap,
    emphasisKeymap,
    inlineCodeKeymap,
    strongKeymap
  ].flat();
  var remarkAddOrderInListPlugin = $remark(
    "remarkAddOrderInList",
    () => () => (tree) => {
      visit(tree, "list", (node2) => {
        if (node2.ordered) {
          const start = node2.start ?? 1;
          node2.children.forEach((child, index2) => {
            child.label = index2 + start;
          });
        }
      });
    }
  );
  withMeta2(remarkAddOrderInListPlugin.plugin, {
    displayName: "Remark<remarkAddOrderInListPlugin>",
    group: "Remark"
  });
  withMeta2(remarkAddOrderInListPlugin.options, {
    displayName: "RemarkConfig<remarkAddOrderInListPlugin>",
    group: "Remark"
  });
  var remarkLineBreak = $remark(
    "remarkLineBreak",
    () => () => (tree) => {
      const find = /[\t ]*(?:\r?\n|\r)/g;
      visit(
        tree,
        "text",
        (node2, index2, parent) => {
          if (!node2.value || typeof node2.value !== "string") return;
          const result = [];
          let start = 0;
          find.lastIndex = 0;
          let match = find.exec(node2.value);
          while (match) {
            const position2 = match.index;
            if (start !== position2)
              result.push({
                type: "text",
                value: node2.value.slice(start, position2)
              });
            result.push({ type: "break", data: { isInline: true } });
            start = position2 + match[0].length;
            match = find.exec(node2.value);
          }
          const hasResultAndIndex = result.length > 0 && parent && typeof index2 === "number";
          if (!hasResultAndIndex) return;
          if (start < node2.value.length)
            result.push({ type: "text", value: node2.value.slice(start) });
          parent.children.splice(index2, 1, ...result);
          return index2 + result.length;
        }
      );
    }
  );
  withMeta2(remarkLineBreak.plugin, {
    displayName: "Remark<remarkLineBreak>",
    group: "Remark"
  });
  withMeta2(remarkLineBreak.options, {
    displayName: "RemarkConfig<remarkLineBreak>",
    group: "Remark"
  });
  var remarkInlineLinkPlugin = $remark(
    "remarkInlineLink",
    () => remarkInlineLinks
  );
  withMeta2(remarkInlineLinkPlugin.plugin, {
    displayName: "Remark<remarkInlineLinkPlugin>",
    group: "Remark"
  });
  withMeta2(remarkInlineLinkPlugin.options, {
    displayName: "RemarkConfig<remarkInlineLinkPlugin>",
    group: "Remark"
  });
  var isParent = (node2) => !!node2.children;
  var isHTML = (node2) => node2.type === "html";
  function flatMapWithDepth(ast, fn) {
    return transform(ast, 0, null)[0];
    function transform(node2, index2, parent) {
      if (isParent(node2)) {
        const out = [];
        for (let i2 = 0, n2 = node2.children.length; i2 < n2; i2++) {
          const nthChild = node2.children[i2];
          if (nthChild) {
            const xs = transform(nthChild, i2, node2);
            if (xs) {
              for (let j = 0, m = xs.length; j < m; j++) {
                const item = xs[j];
                if (item) out.push(item);
              }
            }
          }
        }
        node2.children = out;
      }
      return fn(node2, index2, parent);
    }
  }
  var BLOCK_CONTAINER_TYPES = ["root", "blockquote", "listItem"];
  var remarkHtmlTransformer = $remark(
    "remarkHTMLTransformer",
    () => () => (tree) => {
      flatMapWithDepth(tree, (node2, _index, parent) => {
        if (!isHTML(node2)) return [node2];
        if (parent && BLOCK_CONTAINER_TYPES.includes(parent.type)) {
          node2.children = [{ ...node2 }];
          delete node2.value;
          node2.type = "paragraph";
        }
        return [node2];
      });
    }
  );
  withMeta2(remarkHtmlTransformer.plugin, {
    displayName: "Remark<remarkHtmlTransformer>",
    group: "Remark"
  });
  withMeta2(remarkHtmlTransformer.options, {
    displayName: "RemarkConfig<remarkHtmlTransformer>",
    group: "Remark"
  });
  var remarkMarker = $remark(
    "remarkMarker",
    () => () => (tree, file) => {
      const getMarker = (node2) => {
        return file.value.charAt(node2.position.start.offset);
      };
      visit(
        tree,
        (node2) => ["strong", "emphasis"].includes(node2.type),
        (node2) => {
          node2.marker = getMarker(node2);
        }
      );
    }
  );
  withMeta2(remarkMarker.plugin, {
    displayName: "Remark<remarkMarker>",
    group: "Remark"
  });
  withMeta2(remarkMarker.options, {
    displayName: "RemarkConfig<remarkMarker>",
    group: "Remark"
  });
  var inlineNodesCursorPlugin = $prose(() => {
    let lock = false;
    const inlineNodesCursorPluginKey = new PluginKey(
      "MILKDOWN_INLINE_NODES_CURSOR"
    );
    const inlineNodesCursorPlugin2 = new Plugin({
      key: inlineNodesCursorPluginKey,
      state: {
        init() {
          return false;
        },
        apply(tr) {
          if (!tr.selection.empty) return false;
          const pos = tr.selection.$from;
          const left = pos.nodeBefore;
          const right = pos.nodeAfter;
          if (left && right && left.isInline && !left.isText && right.isInline && !right.isText)
            return true;
          return false;
        }
      },
      props: {
        handleDOMEvents: {
          compositionend: (view, e2) => {
            if (lock) {
              lock = false;
              requestAnimationFrame(() => {
                const active = inlineNodesCursorPlugin2.getState(view.state);
                if (active) {
                  const from = view.state.selection.from;
                  e2.preventDefault();
                  view.dispatch(view.state.tr.insertText(e2.data || "", from));
                }
              });
              return true;
            }
            return false;
          },
          compositionstart: (view) => {
            const active = inlineNodesCursorPlugin2.getState(view.state);
            if (active) lock = true;
            return false;
          },
          beforeinput: (view, e2) => {
            const active = inlineNodesCursorPlugin2.getState(view.state);
            if (active && e2 instanceof InputEvent && e2.data && !lock) {
              const from = view.state.selection.from;
              e2.preventDefault();
              view.dispatch(view.state.tr.insertText(e2.data || "", from));
              return true;
            }
            return false;
          }
        },
        decorations(state) {
          const active = inlineNodesCursorPlugin2.getState(state);
          if (active) {
            const pos = state.selection.$from;
            const position2 = pos.pos;
            const left = document.createElement("span");
            const leftDec = Decoration.widget(position2, left, {
              side: -1
            });
            const right = document.createElement("span");
            const rightDec = Decoration.widget(position2, right);
            setTimeout(() => {
              left.contentEditable = "true";
              right.contentEditable = "true";
            });
            return DecorationSet.create(state.doc, [leftDec, rightDec]);
          }
          return DecorationSet.empty;
        }
      }
    });
    return inlineNodesCursorPlugin2;
  });
  withMeta2(inlineNodesCursorPlugin, {
    displayName: "Prose<inlineNodesCursorPlugin>",
    group: "Prose"
  });
  var hardbreakClearMarkPlugin = $prose((ctx) => {
    return new Plugin({
      key: new PluginKey("MILKDOWN_HARDBREAK_MARKS"),
      appendTransaction: (trs, _oldState, newState) => {
        if (!trs.length) return;
        const [tr] = trs;
        if (!tr) return;
        const [step] = tr.steps;
        const isInsertHr = tr.getMeta("hardbreak");
        if (isInsertHr) {
          if (!(step instanceof ReplaceStep)) return;
          const { from } = step;
          return newState.tr.setNodeMarkup(
            from,
            hardbreakSchema.type(ctx),
            void 0,
            []
          );
        }
        const isAddMarkStep = step instanceof AddMarkStep;
        if (isAddMarkStep) {
          let _tr = newState.tr;
          const { from, to } = step;
          newState.doc.nodesBetween(from, to, (node2, pos) => {
            if (node2.type === hardbreakSchema.type(ctx))
              _tr = _tr.setNodeMarkup(
                pos,
                hardbreakSchema.type(ctx),
                void 0,
                []
              );
          });
          return _tr;
        }
        return void 0;
      }
    });
  });
  withMeta2(hardbreakClearMarkPlugin, {
    displayName: "Prose<hardbreakClearMarkPlugin>",
    group: "Prose"
  });
  var hardbreakFilterNodes = $ctx(
    ["table", "code_block"],
    "hardbreakFilterNodes"
  );
  withMeta2(hardbreakFilterNodes, {
    displayName: "Ctx<hardbreakFilterNodes>",
    group: "Prose"
  });
  var hardbreakFilterPlugin = $prose((ctx) => {
    const notIn = ctx.get(hardbreakFilterNodes.key);
    return new Plugin({
      key: new PluginKey("MILKDOWN_HARDBREAK_FILTER"),
      filterTransaction: (tr, state) => {
        const isInsertHr = tr.getMeta("hardbreak");
        const [step] = tr.steps;
        if (isInsertHr && step) {
          const { from } = step;
          const $from = state.doc.resolve(from);
          let curDepth = $from.depth;
          let canApply = true;
          while (curDepth > 0) {
            if (notIn.includes($from.node(curDepth).type.name)) canApply = false;
            curDepth--;
          }
          return canApply;
        }
        return true;
      }
    });
  });
  withMeta2(hardbreakFilterPlugin, {
    displayName: "Prose<hardbreakFilterPlugin>",
    group: "Prose"
  });
  var syncHeadingIdPlugin = $prose((ctx) => {
    const headingIdPluginKey = new PluginKey("MILKDOWN_HEADING_ID");
    const updateId = (view) => {
      if (view.composing) return;
      const getId = ctx.get(headingIdGenerator.key);
      const tr = view.state.tr.setMeta("addToHistory", false);
      let found2 = false;
      const idMap = {};
      view.state.doc.descendants((node2, pos) => {
        if (node2.type === headingSchema.type(ctx)) {
          if (node2.textContent.trim().length === 0) return;
          const attrs = node2.attrs;
          let id = getId(node2);
          if (idMap[id]) {
            idMap[id] += 1;
            id += `-#${idMap[id]}`;
          } else {
            idMap[id] = 1;
          }
          if (attrs.id !== id) {
            found2 = true;
            tr.setMeta(headingIdPluginKey, true).setNodeMarkup(pos, void 0, {
              ...attrs,
              id
            });
          }
        }
      });
      if (found2) view.dispatch(tr);
    };
    return new Plugin({
      key: headingIdPluginKey,
      view: (view) => {
        updateId(view);
        return {
          update: (view2, prevState) => {
            if (view2.state.doc.eq(prevState.doc)) return;
            updateId(view2);
          }
        };
      }
    });
  });
  withMeta2(syncHeadingIdPlugin, {
    displayName: "Prose<syncHeadingIdPlugin>",
    group: "Prose"
  });
  var syncListOrderPlugin = $prose((ctx) => {
    const syncOrderLabel = (transactions, _oldState, newState) => {
      if (!newState.selection || transactions.some(
        (tr2) => tr2.getMeta("addToHistory") === false || !tr2.isGeneric
      ))
        return null;
      const orderedListType = orderedListSchema.type(ctx);
      const bulletListType = bulletListSchema.type(ctx);
      const listItemType = listItemSchema.type(ctx);
      const handleNodeItem = (attrs, index2) => {
        let changed = false;
        const expectedLabel = `${index2 + 1}.`;
        if (attrs.label !== expectedLabel) {
          attrs.label = expectedLabel;
          changed = true;
        }
        return changed;
      };
      let tr = newState.tr;
      let needDispatch = false;
      newState.doc.descendants(
        (node2, pos, parent, index2) => {
          if (node2.type === bulletListType) {
            const base2 = node2.maybeChild(0);
            if (base2?.type === listItemType && base2.attrs.listType === "ordered") {
              needDispatch = true;
              tr.setNodeMarkup(pos, orderedListType, { spread: "true" });
              node2.descendants(
                (child, pos2, _parent, index22) => {
                  if (child.type === listItemType) {
                    const attrs = { ...child.attrs };
                    const changed = handleNodeItem(attrs, index22);
                    if (changed) tr = tr.setNodeMarkup(pos2, void 0, attrs);
                  }
                  return false;
                }
              );
            }
          } else if (node2.type === listItemType && parent?.type === orderedListType) {
            const attrs = { ...node2.attrs };
            let changed = false;
            if (attrs.listType !== "ordered") {
              attrs.listType = "ordered";
              changed = true;
            }
            const base2 = parent?.maybeChild(0);
            if (base2) changed = handleNodeItem(attrs, index2);
            if (changed) {
              tr = tr.setNodeMarkup(pos, void 0, attrs);
              needDispatch = true;
            }
          }
        }
      );
      return needDispatch ? tr.setMeta("addToHistory", false) : null;
    };
    return new Plugin({
      key: new PluginKey("MILKDOWN_KEEP_LIST_ORDER"),
      appendTransaction: syncOrderLabel
    });
  });
  withMeta2(syncListOrderPlugin, {
    displayName: "Prose<syncListOrderPlugin>",
    group: "Prose"
  });
  var plugins = [
    hardbreakClearMarkPlugin,
    hardbreakFilterNodes,
    hardbreakFilterPlugin,
    inlineNodesCursorPlugin,
    remarkAddOrderInListPlugin,
    remarkInlineLinkPlugin,
    remarkLineBreak,
    remarkHtmlTransformer,
    remarkMarker,
    remarkPreserveEmptyLinePlugin,
    syncHeadingIdPlugin,
    syncListOrderPlugin
  ].flat();
  var commonmark = [
    schema2,
    inputRules,
    markInputRules,
    commands2,
    keymap3,
    plugins
  ].flat();

  // node_modules/lodash-es/_freeGlobal.js
  var freeGlobal = typeof global == "object" && global && global.Object === Object && global;
  var freeGlobal_default = freeGlobal;

  // node_modules/lodash-es/_root.js
  var freeSelf = typeof self == "object" && self && self.Object === Object && self;
  var root2 = freeGlobal_default || freeSelf || Function("return this")();
  var root_default = root2;

  // node_modules/lodash-es/_Symbol.js
  var Symbol2 = root_default.Symbol;
  var Symbol_default = Symbol2;

  // node_modules/lodash-es/_getRawTag.js
  var objectProto = Object.prototype;
  var hasOwnProperty2 = objectProto.hasOwnProperty;
  var nativeObjectToString = objectProto.toString;
  var symToStringTag = Symbol_default ? Symbol_default.toStringTag : void 0;
  function getRawTag(value) {
    var isOwn = hasOwnProperty2.call(value, symToStringTag), tag = value[symToStringTag];
    try {
      value[symToStringTag] = void 0;
      var unmasked = true;
    } catch (e2) {
    }
    var result = nativeObjectToString.call(value);
    if (unmasked) {
      if (isOwn) {
        value[symToStringTag] = tag;
      } else {
        delete value[symToStringTag];
      }
    }
    return result;
  }
  var getRawTag_default = getRawTag;

  // node_modules/lodash-es/_objectToString.js
  var objectProto2 = Object.prototype;
  var nativeObjectToString2 = objectProto2.toString;
  function objectToString(value) {
    return nativeObjectToString2.call(value);
  }
  var objectToString_default = objectToString;

  // node_modules/lodash-es/_baseGetTag.js
  var nullTag = "[object Null]";
  var undefinedTag = "[object Undefined]";
  var symToStringTag2 = Symbol_default ? Symbol_default.toStringTag : void 0;
  function baseGetTag(value) {
    if (value == null) {
      return value === void 0 ? undefinedTag : nullTag;
    }
    return symToStringTag2 && symToStringTag2 in Object(value) ? getRawTag_default(value) : objectToString_default(value);
  }
  var baseGetTag_default = baseGetTag;

  // node_modules/lodash-es/isObjectLike.js
  function isObjectLike(value) {
    return value != null && typeof value == "object";
  }
  var isObjectLike_default = isObjectLike;

  // node_modules/lodash-es/isSymbol.js
  var symbolTag = "[object Symbol]";
  function isSymbol(value) {
    return typeof value == "symbol" || isObjectLike_default(value) && baseGetTag_default(value) == symbolTag;
  }
  var isSymbol_default = isSymbol;

  // node_modules/lodash-es/_trimmedEndIndex.js
  var reWhitespace = /\s/;
  function trimmedEndIndex(string3) {
    var index2 = string3.length;
    while (index2-- && reWhitespace.test(string3.charAt(index2))) {
    }
    return index2;
  }
  var trimmedEndIndex_default = trimmedEndIndex;

  // node_modules/lodash-es/_baseTrim.js
  var reTrimStart = /^\s+/;
  function baseTrim(string3) {
    return string3 ? string3.slice(0, trimmedEndIndex_default(string3) + 1).replace(reTrimStart, "") : string3;
  }
  var baseTrim_default = baseTrim;

  // node_modules/lodash-es/isObject.js
  function isObject(value) {
    var type = typeof value;
    return value != null && (type == "object" || type == "function");
  }
  var isObject_default = isObject;

  // node_modules/lodash-es/toNumber.js
  var NAN = 0 / 0;
  var reIsBadHex = /^[-+]0x[0-9a-f]+$/i;
  var reIsBinary = /^0b[01]+$/i;
  var reIsOctal = /^0o[0-7]+$/i;
  var freeParseInt = parseInt;
  function toNumber(value) {
    if (typeof value == "number") {
      return value;
    }
    if (isSymbol_default(value)) {
      return NAN;
    }
    if (isObject_default(value)) {
      var other = typeof value.valueOf == "function" ? value.valueOf() : value;
      value = isObject_default(other) ? other + "" : other;
    }
    if (typeof value != "string") {
      return value === 0 ? value : +value;
    }
    value = baseTrim_default(value);
    var isBinary = reIsBinary.test(value);
    return isBinary || reIsOctal.test(value) ? freeParseInt(value.slice(2), isBinary ? 2 : 8) : reIsBadHex.test(value) ? NAN : +value;
  }
  var toNumber_default = toNumber;

  // node_modules/lodash-es/now.js
  var now = function() {
    return root_default.Date.now();
  };
  var now_default = now;

  // node_modules/lodash-es/debounce.js
  var FUNC_ERROR_TEXT = "Expected a function";
  var nativeMax = Math.max;
  var nativeMin = Math.min;
  function debounce(func, wait, options) {
    var lastArgs, lastThis, maxWait, result, timerId, lastCallTime, lastInvokeTime = 0, leading = false, maxing = false, trailing = true;
    if (typeof func != "function") {
      throw new TypeError(FUNC_ERROR_TEXT);
    }
    wait = toNumber_default(wait) || 0;
    if (isObject_default(options)) {
      leading = !!options.leading;
      maxing = "maxWait" in options;
      maxWait = maxing ? nativeMax(toNumber_default(options.maxWait) || 0, wait) : maxWait;
      trailing = "trailing" in options ? !!options.trailing : trailing;
    }
    function invokeFunc(time) {
      var args = lastArgs, thisArg = lastThis;
      lastArgs = lastThis = void 0;
      lastInvokeTime = time;
      result = func.apply(thisArg, args);
      return result;
    }
    function leadingEdge(time) {
      lastInvokeTime = time;
      timerId = setTimeout(timerExpired, wait);
      return leading ? invokeFunc(time) : result;
    }
    function remainingWait(time) {
      var timeSinceLastCall = time - lastCallTime, timeSinceLastInvoke = time - lastInvokeTime, timeWaiting = wait - timeSinceLastCall;
      return maxing ? nativeMin(timeWaiting, maxWait - timeSinceLastInvoke) : timeWaiting;
    }
    function shouldInvoke(time) {
      var timeSinceLastCall = time - lastCallTime, timeSinceLastInvoke = time - lastInvokeTime;
      return lastCallTime === void 0 || timeSinceLastCall >= wait || timeSinceLastCall < 0 || maxing && timeSinceLastInvoke >= maxWait;
    }
    function timerExpired() {
      var time = now_default();
      if (shouldInvoke(time)) {
        return trailingEdge(time);
      }
      timerId = setTimeout(timerExpired, remainingWait(time));
    }
    function trailingEdge(time) {
      timerId = void 0;
      if (trailing && lastArgs) {
        return invokeFunc(time);
      }
      lastArgs = lastThis = void 0;
      return result;
    }
    function cancel() {
      if (timerId !== void 0) {
        clearTimeout(timerId);
      }
      lastInvokeTime = 0;
      lastArgs = lastCallTime = lastThis = timerId = void 0;
    }
    function flush() {
      return timerId === void 0 ? result : trailingEdge(now_default());
    }
    function debounced() {
      var time = now_default(), isInvoking = shouldInvoke(time);
      lastArgs = arguments;
      lastThis = this;
      lastCallTime = time;
      if (isInvoking) {
        if (timerId === void 0) {
          return leadingEdge(lastCallTime);
        }
        if (maxing) {
          clearTimeout(timerId);
          timerId = setTimeout(timerExpired, wait);
          return invokeFunc(lastCallTime);
        }
      }
      if (timerId === void 0) {
        timerId = setTimeout(timerExpired, wait);
      }
      return result;
    }
    debounced.cancel = cancel;
    debounced.flush = flush;
    return debounced;
  }
  var debounce_default = debounce;

  // node_modules/@milkdown/plugin-listener/lib/index.js
  var ListenerManager = class {
    constructor() {
      this.beforeMountedListeners = [];
      this.mountedListeners = [];
      this.updatedListeners = [];
      this.selectionUpdatedListeners = [];
      this.markdownUpdatedListeners = [];
      this.blurListeners = [];
      this.focusListeners = [];
      this.destroyListeners = [];
      this.beforeMount = (fn) => {
        this.beforeMountedListeners.push(fn);
        return this;
      };
      this.mounted = (fn) => {
        this.mountedListeners.push(fn);
        return this;
      };
      this.updated = (fn) => {
        this.updatedListeners.push(fn);
        return this;
      };
    }
    /// A getter to get all [subscribers](#interface-subscribers). You should not use this method directly.
    get listeners() {
      return {
        beforeMount: this.beforeMountedListeners,
        mounted: this.mountedListeners,
        updated: this.updatedListeners,
        markdownUpdated: this.markdownUpdatedListeners,
        blur: this.blurListeners,
        focus: this.focusListeners,
        destroy: this.destroyListeners,
        selectionUpdated: this.selectionUpdatedListeners
      };
    }
    /// Subscribe to the markdownUpdated event.
    /// This event will be triggered after the editor state is updated and **the document is changed**.
    /// The second parameter is the current markdown and the third parameter is the previous markdown.
    markdownUpdated(fn) {
      this.markdownUpdatedListeners.push(fn);
      return this;
    }
    /// Subscribe to the blur event.
    /// This event will be triggered when the editor is blurred.
    blur(fn) {
      this.blurListeners.push(fn);
      return this;
    }
    /// Subscribe to the focus event.
    /// This event will be triggered when the editor is focused.
    focus(fn) {
      this.focusListeners.push(fn);
      return this;
    }
    /// Subscribe to the destroy event.
    /// This event will be triggered before the editor is destroyed.
    destroy(fn) {
      this.destroyListeners.push(fn);
      return this;
    }
    /// Subscribe to the selectionUpdated event.
    /// This event will be triggered when the editor selection is updated.
    selectionUpdated(fn) {
      this.selectionUpdatedListeners.push(fn);
      return this;
    }
  };
  var listenerCtx = createSlice(
    new ListenerManager(),
    "listener"
  );
  var key2 = new PluginKey("MILKDOWN_LISTENER");
  var listener = (ctx) => {
    ctx.inject(listenerCtx, new ListenerManager());
    return async () => {
      await ctx.wait(InitReady);
      const listener2 = ctx.get(listenerCtx);
      const { listeners } = listener2;
      listeners.beforeMount.forEach((fn) => fn(ctx));
      await ctx.wait(SerializerReady);
      const serializer2 = ctx.get(serializerCtx);
      let prevDoc = null;
      let prevMarkdown = null;
      let prevSelection = null;
      const plugin = new Plugin({
        key: key2,
        view: () => {
          return {
            destroy: () => {
              listeners.destroy.forEach((fn) => fn(ctx));
            }
          };
        },
        props: {
          handleDOMEvents: {
            focus: () => {
              listeners.focus.forEach((fn) => fn(ctx));
              return false;
            },
            blur: () => {
              listeners.blur.forEach((fn) => fn(ctx));
              return false;
            }
          }
        },
        state: {
          init: (_, instance) => {
            prevDoc = instance.doc;
            prevMarkdown = serializer2(instance.doc);
          },
          apply: (tr) => {
            const currentSelection = tr.selection;
            if (!prevSelection && currentSelection || prevSelection && !currentSelection.eq(prevSelection)) {
              listeners.selectionUpdated.forEach((fn) => {
                fn(ctx, currentSelection, prevSelection);
              });
              prevSelection = currentSelection;
            }
            if (!(tr.docChanged || tr.storedMarksSet) || tr.getMeta("addToHistory") === false)
              return;
            const handler = debounce_default(() => {
              const { doc: doc4 } = tr;
              if (listeners.updated.length > 0 && prevDoc && !prevDoc.eq(doc4)) {
                listeners.updated.forEach((fn) => {
                  fn(ctx, doc4, prevDoc);
                });
              }
              if (listeners.markdownUpdated.length > 0 && prevDoc && !prevDoc.eq(doc4)) {
                const markdown = serializer2(doc4);
                listeners.markdownUpdated.forEach((fn) => {
                  fn(ctx, markdown, prevMarkdown);
                });
                prevMarkdown = markdown;
              }
              prevDoc = doc4;
            }, 200);
            return handler();
          }
        }
      });
      ctx.update(prosePluginsCtx, (x) => x.concat(plugin));
      await ctx.wait(EditorViewReady);
      listeners.mounted.forEach((fn) => fn(ctx));
    };
  };
  listener.meta = {
    package: "@milkdown/plugin-listener",
    displayName: "Listener"
  };

  // node_modules/remark-wiki-link/dist/index.js
  var e = { d: (t2, r2) => {
    for (var n2 in r2) e.o(r2, n2) && !e.o(t2, n2) && Object.defineProperty(t2, n2, { enumerable: true, get: r2[n2] });
  }, o: (e2, t2) => Object.prototype.hasOwnProperty.call(e2, t2) };
  var t = {};
  e.d(t, { Z: () => h, $: () => k });
  var r = { horizontalTab: -2, virtualSpace: -1, nul: 0, eof: null, space: 32 };
  function n(e2) {
    return e2 < r.nul || e2 === r.space;
  }
  function i(e2) {
    return e2 < r.horizontalTab;
  }
  var a = { 553: (e2) => {
    e2.exports = function(e3) {
      var t2, r2;
      return e3._compiled || (t2 = e3.before ? "(?:" + e3.before + ")" : "", r2 = e3.after ? "(?:" + e3.after + ")" : "", e3.atBreak && (t2 = "[\\r\\n][\\t ]*" + t2), e3._compiled = new RegExp((t2 ? "(" + t2 + ")" : "") + (/[|\\{}()[\]^$+*?.-]/.test(e3.character) ? "\\" : "") + e3.character + (r2 || ""), "g")), e3._compiled;
    };
  }, 112: (e2) => {
    function t2(e3, t3, r2) {
      var n2;
      if (!t3) return r2;
      for ("string" == typeof t3 && (t3 = [t3]), n2 = -1; ++n2 < t3.length; ) if (-1 !== e3.indexOf(t3[n2])) return true;
      return false;
    }
    e2.exports = function(e3, r2) {
      return t2(e3, r2.inConstruct, true) && !t2(e3, r2.notInConstruct);
    };
  }, 113: (e2, t2, r2) => {
    e2.exports = function(e3, t3, r3) {
      for (var s2, u2, l2, c2, f2, k2, h2, p2, d2 = (r3.before || "") + (t3 || "") + (r3.after || ""), x = [], w = [], v = {}, g = -1; ++g < e3.unsafe.length; ) if (c2 = e3.unsafe[g], i2(e3.stack, c2)) for (f2 = n2(c2); k2 = f2.exec(d2); ) s2 = "before" in c2 || c2.atBreak, u2 = "after" in c2, l2 = k2.index + (s2 ? k2[1].length : 0), -1 === x.indexOf(l2) ? (x.push(l2), v[l2] = { before: s2, after: u2 }) : (v[l2].before && !s2 && (v[l2].before = false), v[l2].after && !u2 && (v[l2].after = false));
      for (x.sort(a2), h2 = r3.before ? r3.before.length : 0, p2 = d2.length - (r3.after ? r3.after.length : 0), g = -1; ++g < x.length; ) (l2 = x[g]) < h2 || l2 >= p2 || l2 + 1 < p2 && x[g + 1] === l2 + 1 && v[l2].after && !v[l2 + 1].before && !v[l2 + 1].after || (h2 !== l2 && w.push(o2(d2.slice(h2, l2), "\\")), h2 = l2, !/[!-/:-@[-`{-~]/.test(d2.charAt(l2)) || r3.encode && -1 !== r3.encode.indexOf(d2.charAt(l2)) ? (w.push("&#x" + d2.charCodeAt(l2).toString(16).toUpperCase() + ";"), h2++) : w.push("\\"));
      return w.push(o2(d2.slice(h2, p2), r3.after)), w.join("");
    };
    var n2 = r2(553), i2 = r2(112);
    function a2(e3, t3) {
      return e3 - t3;
    }
    function o2(e3, t3) {
      for (var r3, n3 = /\\(?=[!-/:-@[-`{-~])/g, i3 = [], a3 = [], o3 = -1, s2 = 0, u2 = e3 + t3; r3 = n3.exec(u2); ) i3.push(r3.index);
      for (; ++o3 < i3.length; ) s2 !== i3[o3] && a3.push(e3.slice(s2, i3[o3])), a3.push("\\"), s2 = i3[o3];
      return a3.push(e3.slice(s2)), a3.join("");
    }
  } };
  var o = {};
  function s(e2) {
    var t2 = o[e2];
    if (void 0 !== t2) return t2.exports;
    var r2 = o[e2] = { exports: {} };
    return a[e2](r2, r2.exports, s), r2.exports;
  }
  s.n = (e2) => {
    var t2 = e2 && e2.__esModule ? () => e2.default : () => e2;
    return s.d(t2, { a: t2 }), t2;
  }, s.d = (e2, t2) => {
    for (var r2 in t2) s.o(t2, r2) && !s.o(e2, r2) && Object.defineProperty(e2, r2, { enumerable: true, get: t2[r2] });
  }, s.o = (e2, t2) => Object.prototype.hasOwnProperty.call(e2, t2);
  var u = {};
  (() => {
    function e2(e3 = {}) {
      const t3 = e3.permalinks || [], r3 = e3.pageResolver || ((e4) => [e4.replace(/ /g, "_").toLowerCase()]), n3 = e3.newClassName || "new", i2 = e3.wikiLinkClassName || "internal", a2 = e3.hrefTemplate || ((e4) => `#/page/${e4}`);
      let o2;
      function s2(e4) {
        return e4[e4.length - 1];
      }
      return { enter: { wikiLink: function(e4) {
        o2 = { type: "wikiLink", value: null, data: { alias: null, permalink: null, exists: null } }, this.enter(o2, e4);
      } }, exit: { wikiLinkTarget: function(e4) {
        const t4 = this.sliceSerialize(e4);
        s2(this.stack).value = t4;
      }, wikiLinkAlias: function(e4) {
        const t4 = this.sliceSerialize(e4);
        s2(this.stack).data.alias = t4;
      }, wikiLink: function(e4) {
        this.exit(e4);
        const s3 = o2, u2 = r3(s3.value), l2 = u2.find(((e5) => -1 !== t3.indexOf(e5))), c2 = void 0 !== l2;
        let f2;
        f2 = c2 ? l2 : u2[0] || "";
        let k2 = s3.value;
        s3.data.alias && (k2 = s3.data.alias);
        let h2 = i2;
        c2 || (h2 += " " + n3), s3.data.alias = k2, s3.data.permalink = f2, s3.data.exists = c2, s3.data.hName = "a", s3.data.hProperties = { className: h2, href: a2(f2) }, s3.data.hChildren = [{ type: "text", value: k2 }];
      } } };
    }
    s.d(u, { V: () => e2, x: () => n2 });
    var t2 = s(113), r2 = s.n(t2);
    function n2(e3 = {}) {
      const t3 = e3.aliasDivider || ":";
      return { unsafe: [{ character: "[", inConstruct: ["phrasing", "label", "reference"] }, { character: "]", inConstruct: ["label", "reference"] }], handlers: { wikiLink: function(e4, n3, i2) {
        const a2 = i2.enter("wikiLink"), o2 = r2()(i2, e4.value, { before: "[", after: "]" }), s2 = r2()(i2, e4.data.alias, { before: "[", after: "]" });
        let u2;
        return u2 = s2 !== o2 ? `[[${o2}${t3}${s2}]]` : `[[${o2}]]`, a2(), u2;
      } } };
    }
  })();
  var l = u.V;
  var c = u.x;
  var f = false;
  function k(e2 = {}) {
    const t2 = this.data();
    function a2(e3, r2) {
      t2[e3] ? t2[e3].push(r2) : t2[e3] = [r2];
    }
    !f && (this.Parser && this.Parser.prototype && this.Parser.prototype.blockTokenizers || this.Compiler && this.Compiler.prototype && this.Compiler.prototype.visitors) && (f = true, console.warn("[remark-wiki-link] Warning: please upgrade to remark 13 to use this plugin")), a2("micromarkExtensions", (function() {
      var e3 = (arguments.length > 0 && void 0 !== arguments[0] ? arguments[0] : {}).aliasDivider || ":", t3 = "]]";
      return { text: { 91: { tokenize: function(a3, o2, s2) {
        var u2, l2, c2 = 0, f2 = 0, k2 = 0;
        return function(e4) {
          return e4 !== "[[".charCodeAt(f2) ? s2(e4) : (a3.enter("wikiLink"), a3.enter("wikiLinkMarker"), h2(e4));
        };
        function h2(e4) {
          return 2 === f2 ? (a3.exit("wikiLinkMarker"), (function(e5) {
            return i(e5) || e5 === r.eof ? s2(e5) : (a3.enter("wikiLinkData"), a3.enter("wikiLinkTarget"), p2(e5));
          })(e4)) : e4 !== "[[".charCodeAt(f2) ? s2(e4) : (a3.consume(e4), f2++, h2);
        }
        function p2(o3) {
          return o3 === e3.charCodeAt(c2) ? u2 ? (a3.exit("wikiLinkTarget"), a3.enter("wikiLinkAliasMarker"), d2(o3)) : s2(o3) : o3 === t3.charCodeAt(k2) ? u2 ? (a3.exit("wikiLinkTarget"), a3.exit("wikiLinkData"), a3.enter("wikiLinkMarker"), w(o3)) : s2(o3) : i(o3) || o3 === r.eof ? s2(o3) : (n(o3) || (u2 = true), a3.consume(o3), p2);
        }
        function d2(t4) {
          return c2 === e3.length ? (a3.exit("wikiLinkAliasMarker"), a3.enter("wikiLinkAlias"), x(t4)) : t4 !== e3.charCodeAt(c2) ? s2(t4) : (a3.consume(t4), c2++, d2);
        }
        function x(e4) {
          return e4 === t3.charCodeAt(k2) ? l2 ? (a3.exit("wikiLinkAlias"), a3.exit("wikiLinkData"), a3.enter("wikiLinkMarker"), w(e4)) : s2(e4) : i(e4) || e4 === r.eof ? s2(e4) : (n(e4) || (l2 = true), a3.consume(e4), x);
        }
        function w(e4) {
          return 2 === k2 ? (a3.exit("wikiLinkMarker"), a3.exit("wikiLink"), o2(e4)) : e4 !== t3.charCodeAt(k2) ? s2(e4) : (a3.consume(e4), k2++, w);
        }
      } } } };
    })(e2)), a2("fromMarkdownExtensions", l(e2)), a2("toMarkdownExtensions", c(e2));
  }
  var h = k;
  var p = t.Z;
  var d = t.$;

  // src/index.ts
  var wikiLinkRemark = $remark(
    "wikiLink",
    () => p({
      aliasDivider: "|",
      hrefTemplate: (permalink) => `/${permalink}`,
      wikiLinkClassName: "wikilink",
      newClassName: "wikilink-new"
    })
  );
  function createMilkdown() {
    let editor = null;
    let changeCallback = null;
    return {
      async create(root3, defaultValue = "") {
        editor = await Editor.make().config((ctx) => {
          ctx.set(rootCtx, root3);
          ctx.set(defaultValueCtx, defaultValue);
          ctx.set(listenerCtx, {
            markdown: [(getMarkdown) => {
              if (changeCallback) {
                changeCallback(getMarkdown());
              }
            }]
          });
        }).use(commonmark).use(wikiLinkRemark).use(listener).create();
      },
      async destroy() {
        if (editor) {
          await editor.destroy();
          editor = null;
        }
      },
      getMarkdown() {
        if (!editor) return "";
        return editor.action((ctx) => {
          const view = ctx.get(editorViewCtx);
          const serializer2 = ctx.get(serializerCtx);
          return serializer2(view.state.doc);
        });
      },
      setMarkdown(markdown) {
        if (!editor) return;
        editor.action((ctx) => {
          const view = ctx.get(editorViewCtx);
          const parser2 = ctx.get(parserCtx);
          const doc4 = parser2(markdown);
          if (!doc4) return;
          const state = view.state;
          view.dispatch(
            state.tr.replace(
              0,
              state.doc.content.size,
              new Slice2(doc4.content, 0, 0)
            )
          );
        });
      },
      onChange(callback) {
        changeCallback = callback;
      }
    };
  }
  return __toCommonJS(index_exports);
})();
/*! Bundled license information:

lodash-es/lodash.js:
  (**
   * @license
   * Lodash (Custom Build) <https://lodash.com/>
   * Build: `lodash modularize exports="es" -o ./`
   * Copyright OpenJS Foundation and other contributors <https://openjsf.org/>
   * Released under MIT license <https://lodash.com/license>
   * Based on Underscore.js 1.8.3 <http://underscorejs.org/LICENSE>
   * Copyright Jeremy Ashkenas, DocumentCloud and Investigative Reporters & Editors
   *)
*/
