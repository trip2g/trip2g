#!/usr/bin/env node
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });
var __commonJS = (cb, mod) => function __require() {
  try {
    return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
  } catch (e) {
    throw mod = 0, e;
  }
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
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

// ../../node_modules/graphql/version.js
var require_version = __commonJS({
  "../../node_modules/graphql/version.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.versionInfo = exports.version = void 0;
    var version = "16.10.0";
    exports.version = version;
    var versionInfo = Object.freeze({
      major: 16,
      minor: 10,
      patch: 0,
      preReleaseTag: null
    });
    exports.versionInfo = versionInfo;
  }
});

// ../../node_modules/graphql/jsutils/devAssert.js
var require_devAssert = __commonJS({
  "../../node_modules/graphql/jsutils/devAssert.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.devAssert = devAssert;
    function devAssert(condition, message) {
      const booleanCondition = Boolean(condition);
      if (!booleanCondition) {
        throw new Error(message);
      }
    }
    __name(devAssert, "devAssert");
  }
});

// ../../node_modules/graphql/jsutils/isPromise.js
var require_isPromise = __commonJS({
  "../../node_modules/graphql/jsutils/isPromise.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.isPromise = isPromise;
    function isPromise(value) {
      return typeof (value === null || value === void 0 ? void 0 : value.then) === "function";
    }
    __name(isPromise, "isPromise");
  }
});

// ../../node_modules/graphql/jsutils/isObjectLike.js
var require_isObjectLike = __commonJS({
  "../../node_modules/graphql/jsutils/isObjectLike.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.isObjectLike = isObjectLike;
    function isObjectLike(value) {
      return typeof value == "object" && value !== null;
    }
    __name(isObjectLike, "isObjectLike");
  }
});

// ../../node_modules/graphql/jsutils/invariant.js
var require_invariant = __commonJS({
  "../../node_modules/graphql/jsutils/invariant.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.invariant = invariant;
    function invariant(condition, message) {
      const booleanCondition = Boolean(condition);
      if (!booleanCondition) {
        throw new Error(
          message != null ? message : "Unexpected invariant triggered."
        );
      }
    }
    __name(invariant, "invariant");
  }
});

// ../../node_modules/graphql/language/location.js
var require_location = __commonJS({
  "../../node_modules/graphql/language/location.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.getLocation = getLocation;
    var _invariant = require_invariant();
    var LineRegExp = /\r\n|[\n\r]/g;
    function getLocation(source, position) {
      let lastLineStart = 0;
      let line = 1;
      for (const match of source.body.matchAll(LineRegExp)) {
        typeof match.index === "number" || (0, _invariant.invariant)(false);
        if (match.index >= position) {
          break;
        }
        lastLineStart = match.index + match[0].length;
        line += 1;
      }
      return {
        line,
        column: position + 1 - lastLineStart
      };
    }
    __name(getLocation, "getLocation");
  }
});

// ../../node_modules/graphql/language/printLocation.js
var require_printLocation = __commonJS({
  "../../node_modules/graphql/language/printLocation.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.printLocation = printLocation;
    exports.printSourceLocation = printSourceLocation;
    var _location = require_location();
    function printLocation(location) {
      return printSourceLocation(
        location.source,
        (0, _location.getLocation)(location.source, location.start)
      );
    }
    __name(printLocation, "printLocation");
    function printSourceLocation(source, sourceLocation) {
      const firstLineColumnOffset = source.locationOffset.column - 1;
      const body = "".padStart(firstLineColumnOffset) + source.body;
      const lineIndex = sourceLocation.line - 1;
      const lineOffset = source.locationOffset.line - 1;
      const lineNum = sourceLocation.line + lineOffset;
      const columnOffset = sourceLocation.line === 1 ? firstLineColumnOffset : 0;
      const columnNum = sourceLocation.column + columnOffset;
      const locationStr = `${source.name}:${lineNum}:${columnNum}
`;
      const lines = body.split(/\r\n|[\n\r]/g);
      const locationLine = lines[lineIndex];
      if (locationLine.length > 120) {
        const subLineIndex = Math.floor(columnNum / 80);
        const subLineColumnNum = columnNum % 80;
        const subLines = [];
        for (let i = 0; i < locationLine.length; i += 80) {
          subLines.push(locationLine.slice(i, i + 80));
        }
        return locationStr + printPrefixedLines([
          [`${lineNum} |`, subLines[0]],
          ...subLines.slice(1, subLineIndex + 1).map((subLine) => ["|", subLine]),
          ["|", "^".padStart(subLineColumnNum)],
          ["|", subLines[subLineIndex + 1]]
        ]);
      }
      return locationStr + printPrefixedLines([
        // Lines specified like this: ["prefix", "string"],
        [`${lineNum - 1} |`, lines[lineIndex - 1]],
        [`${lineNum} |`, locationLine],
        ["|", "^".padStart(columnNum)],
        [`${lineNum + 1} |`, lines[lineIndex + 1]]
      ]);
    }
    __name(printSourceLocation, "printSourceLocation");
    function printPrefixedLines(lines) {
      const existingLines = lines.filter(([_, line]) => line !== void 0);
      const padLen = Math.max(...existingLines.map(([prefix]) => prefix.length));
      return existingLines.map(([prefix, line]) => prefix.padStart(padLen) + (line ? " " + line : "")).join("\n");
    }
    __name(printPrefixedLines, "printPrefixedLines");
  }
});

// ../../node_modules/graphql/error/GraphQLError.js
var require_GraphQLError = __commonJS({
  "../../node_modules/graphql/error/GraphQLError.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.GraphQLError = void 0;
    exports.formatError = formatError;
    exports.printError = printError;
    var _isObjectLike = require_isObjectLike();
    var _location = require_location();
    var _printLocation = require_printLocation();
    function toNormalizedOptions(args) {
      const firstArg = args[0];
      if (firstArg == null || "kind" in firstArg || "length" in firstArg) {
        return {
          nodes: firstArg,
          source: args[1],
          positions: args[2],
          path: args[3],
          originalError: args[4],
          extensions: args[5]
        };
      }
      return firstArg;
    }
    __name(toNormalizedOptions, "toNormalizedOptions");
    var GraphQLError = class _GraphQLError extends Error {
      static {
        __name(this, "GraphQLError");
      }
      /**
       * An array of `{ line, column }` locations within the source GraphQL document
       * which correspond to this error.
       *
       * Errors during validation often contain multiple locations, for example to
       * point out two things with the same name. Errors during execution include a
       * single location, the field which produced the error.
       *
       * Enumerable, and appears in the result of JSON.stringify().
       */
      /**
       * An array describing the JSON-path into the execution response which
       * corresponds to this error. Only included for errors during execution.
       *
       * Enumerable, and appears in the result of JSON.stringify().
       */
      /**
       * An array of GraphQL AST Nodes corresponding to this error.
       */
      /**
       * The source GraphQL document for the first location of this error.
       *
       * Note that if this Error represents more than one node, the source may not
       * represent nodes after the first node.
       */
      /**
       * An array of character offsets within the source GraphQL document
       * which correspond to this error.
       */
      /**
       * The original error thrown from a field resolver during execution.
       */
      /**
       * Extension fields to add to the formatted error.
       */
      /**
       * @deprecated Please use the `GraphQLErrorOptions` constructor overload instead.
       */
      constructor(message, ...rawArgs) {
        var _this$nodes, _nodeLocations$, _ref;
        const { nodes, source, positions, path: path2, originalError, extensions } = toNormalizedOptions(rawArgs);
        super(message);
        this.name = "GraphQLError";
        this.path = path2 !== null && path2 !== void 0 ? path2 : void 0;
        this.originalError = originalError !== null && originalError !== void 0 ? originalError : void 0;
        this.nodes = undefinedIfEmpty(
          Array.isArray(nodes) ? nodes : nodes ? [nodes] : void 0
        );
        const nodeLocations = undefinedIfEmpty(
          (_this$nodes = this.nodes) === null || _this$nodes === void 0 ? void 0 : _this$nodes.map((node) => node.loc).filter((loc) => loc != null)
        );
        this.source = source !== null && source !== void 0 ? source : nodeLocations === null || nodeLocations === void 0 ? void 0 : (_nodeLocations$ = nodeLocations[0]) === null || _nodeLocations$ === void 0 ? void 0 : _nodeLocations$.source;
        this.positions = positions !== null && positions !== void 0 ? positions : nodeLocations === null || nodeLocations === void 0 ? void 0 : nodeLocations.map((loc) => loc.start);
        this.locations = positions && source ? positions.map((pos) => (0, _location.getLocation)(source, pos)) : nodeLocations === null || nodeLocations === void 0 ? void 0 : nodeLocations.map(
          (loc) => (0, _location.getLocation)(loc.source, loc.start)
        );
        const originalExtensions = (0, _isObjectLike.isObjectLike)(
          originalError === null || originalError === void 0 ? void 0 : originalError.extensions
        ) ? originalError === null || originalError === void 0 ? void 0 : originalError.extensions : void 0;
        this.extensions = (_ref = extensions !== null && extensions !== void 0 ? extensions : originalExtensions) !== null && _ref !== void 0 ? _ref : /* @__PURE__ */ Object.create(null);
        Object.defineProperties(this, {
          message: {
            writable: true,
            enumerable: true
          },
          name: {
            enumerable: false
          },
          nodes: {
            enumerable: false
          },
          source: {
            enumerable: false
          },
          positions: {
            enumerable: false
          },
          originalError: {
            enumerable: false
          }
        });
        if (originalError !== null && originalError !== void 0 && originalError.stack) {
          Object.defineProperty(this, "stack", {
            value: originalError.stack,
            writable: true,
            configurable: true
          });
        } else if (Error.captureStackTrace) {
          Error.captureStackTrace(this, _GraphQLError);
        } else {
          Object.defineProperty(this, "stack", {
            value: Error().stack,
            writable: true,
            configurable: true
          });
        }
      }
      get [Symbol.toStringTag]() {
        return "GraphQLError";
      }
      toString() {
        let output = this.message;
        if (this.nodes) {
          for (const node of this.nodes) {
            if (node.loc) {
              output += "\n\n" + (0, _printLocation.printLocation)(node.loc);
            }
          }
        } else if (this.source && this.locations) {
          for (const location of this.locations) {
            output += "\n\n" + (0, _printLocation.printSourceLocation)(this.source, location);
          }
        }
        return output;
      }
      toJSON() {
        const formattedError = {
          message: this.message
        };
        if (this.locations != null) {
          formattedError.locations = this.locations;
        }
        if (this.path != null) {
          formattedError.path = this.path;
        }
        if (this.extensions != null && Object.keys(this.extensions).length > 0) {
          formattedError.extensions = this.extensions;
        }
        return formattedError;
      }
    };
    exports.GraphQLError = GraphQLError;
    function undefinedIfEmpty(array) {
      return array === void 0 || array.length === 0 ? void 0 : array;
    }
    __name(undefinedIfEmpty, "undefinedIfEmpty");
    function printError(error) {
      return error.toString();
    }
    __name(printError, "printError");
    function formatError(error) {
      return error.toJSON();
    }
    __name(formatError, "formatError");
  }
});

// ../../node_modules/graphql/error/syntaxError.js
var require_syntaxError = __commonJS({
  "../../node_modules/graphql/error/syntaxError.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.syntaxError = syntaxError;
    var _GraphQLError = require_GraphQLError();
    function syntaxError(source, position, description) {
      return new _GraphQLError.GraphQLError(`Syntax Error: ${description}`, {
        source,
        positions: [position]
      });
    }
    __name(syntaxError, "syntaxError");
  }
});

// ../../node_modules/graphql/language/ast.js
var require_ast = __commonJS({
  "../../node_modules/graphql/language/ast.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.Token = exports.QueryDocumentKeys = exports.OperationTypeNode = exports.Location = void 0;
    exports.isNode = isNode;
    var Location = class {
      static {
        __name(this, "Location");
      }
      /**
       * The character offset at which this Node begins.
       */
      /**
       * The character offset at which this Node ends.
       */
      /**
       * The Token at which this Node begins.
       */
      /**
       * The Token at which this Node ends.
       */
      /**
       * The Source document the AST represents.
       */
      constructor(startToken, endToken, source) {
        this.start = startToken.start;
        this.end = endToken.end;
        this.startToken = startToken;
        this.endToken = endToken;
        this.source = source;
      }
      get [Symbol.toStringTag]() {
        return "Location";
      }
      toJSON() {
        return {
          start: this.start,
          end: this.end
        };
      }
    };
    exports.Location = Location;
    var Token = class {
      static {
        __name(this, "Token");
      }
      /**
       * The kind of Token.
       */
      /**
       * The character offset at which this Node begins.
       */
      /**
       * The character offset at which this Node ends.
       */
      /**
       * The 1-indexed line number on which this Token appears.
       */
      /**
       * The 1-indexed column number at which this Token begins.
       */
      /**
       * For non-punctuation tokens, represents the interpreted value of the token.
       *
       * Note: is undefined for punctuation tokens, but typed as string for
       * convenience in the parser.
       */
      /**
       * Tokens exist as nodes in a double-linked-list amongst all tokens
       * including ignored tokens. <SOF> is always the first node and <EOF>
       * the last.
       */
      constructor(kind, start, end, line, column, value) {
        this.kind = kind;
        this.start = start;
        this.end = end;
        this.line = line;
        this.column = column;
        this.value = value;
        this.prev = null;
        this.next = null;
      }
      get [Symbol.toStringTag]() {
        return "Token";
      }
      toJSON() {
        return {
          kind: this.kind,
          value: this.value,
          line: this.line,
          column: this.column
        };
      }
    };
    exports.Token = Token;
    var QueryDocumentKeys = {
      Name: [],
      Document: ["definitions"],
      OperationDefinition: [
        "name",
        "variableDefinitions",
        "directives",
        "selectionSet"
      ],
      VariableDefinition: ["variable", "type", "defaultValue", "directives"],
      Variable: ["name"],
      SelectionSet: ["selections"],
      Field: ["alias", "name", "arguments", "directives", "selectionSet"],
      Argument: ["name", "value"],
      FragmentSpread: ["name", "directives"],
      InlineFragment: ["typeCondition", "directives", "selectionSet"],
      FragmentDefinition: [
        "name",
        // Note: fragment variable definitions are deprecated and will removed in v17.0.0
        "variableDefinitions",
        "typeCondition",
        "directives",
        "selectionSet"
      ],
      IntValue: [],
      FloatValue: [],
      StringValue: [],
      BooleanValue: [],
      NullValue: [],
      EnumValue: [],
      ListValue: ["values"],
      ObjectValue: ["fields"],
      ObjectField: ["name", "value"],
      Directive: ["name", "arguments"],
      NamedType: ["name"],
      ListType: ["type"],
      NonNullType: ["type"],
      SchemaDefinition: ["description", "directives", "operationTypes"],
      OperationTypeDefinition: ["type"],
      ScalarTypeDefinition: ["description", "name", "directives"],
      ObjectTypeDefinition: [
        "description",
        "name",
        "interfaces",
        "directives",
        "fields"
      ],
      FieldDefinition: ["description", "name", "arguments", "type", "directives"],
      InputValueDefinition: [
        "description",
        "name",
        "type",
        "defaultValue",
        "directives"
      ],
      InterfaceTypeDefinition: [
        "description",
        "name",
        "interfaces",
        "directives",
        "fields"
      ],
      UnionTypeDefinition: ["description", "name", "directives", "types"],
      EnumTypeDefinition: ["description", "name", "directives", "values"],
      EnumValueDefinition: ["description", "name", "directives"],
      InputObjectTypeDefinition: ["description", "name", "directives", "fields"],
      DirectiveDefinition: ["description", "name", "arguments", "locations"],
      SchemaExtension: ["directives", "operationTypes"],
      ScalarTypeExtension: ["name", "directives"],
      ObjectTypeExtension: ["name", "interfaces", "directives", "fields"],
      InterfaceTypeExtension: ["name", "interfaces", "directives", "fields"],
      UnionTypeExtension: ["name", "directives", "types"],
      EnumTypeExtension: ["name", "directives", "values"],
      InputObjectTypeExtension: ["name", "directives", "fields"]
    };
    exports.QueryDocumentKeys = QueryDocumentKeys;
    var kindValues = new Set(Object.keys(QueryDocumentKeys));
    function isNode(maybeNode) {
      const maybeKind = maybeNode === null || maybeNode === void 0 ? void 0 : maybeNode.kind;
      return typeof maybeKind === "string" && kindValues.has(maybeKind);
    }
    __name(isNode, "isNode");
    var OperationTypeNode;
    exports.OperationTypeNode = OperationTypeNode;
    (function(OperationTypeNode2) {
      OperationTypeNode2["QUERY"] = "query";
      OperationTypeNode2["MUTATION"] = "mutation";
      OperationTypeNode2["SUBSCRIPTION"] = "subscription";
    })(OperationTypeNode || (exports.OperationTypeNode = OperationTypeNode = {}));
  }
});

// ../../node_modules/graphql/language/directiveLocation.js
var require_directiveLocation = __commonJS({
  "../../node_modules/graphql/language/directiveLocation.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.DirectiveLocation = void 0;
    var DirectiveLocation;
    exports.DirectiveLocation = DirectiveLocation;
    (function(DirectiveLocation2) {
      DirectiveLocation2["QUERY"] = "QUERY";
      DirectiveLocation2["MUTATION"] = "MUTATION";
      DirectiveLocation2["SUBSCRIPTION"] = "SUBSCRIPTION";
      DirectiveLocation2["FIELD"] = "FIELD";
      DirectiveLocation2["FRAGMENT_DEFINITION"] = "FRAGMENT_DEFINITION";
      DirectiveLocation2["FRAGMENT_SPREAD"] = "FRAGMENT_SPREAD";
      DirectiveLocation2["INLINE_FRAGMENT"] = "INLINE_FRAGMENT";
      DirectiveLocation2["VARIABLE_DEFINITION"] = "VARIABLE_DEFINITION";
      DirectiveLocation2["SCHEMA"] = "SCHEMA";
      DirectiveLocation2["SCALAR"] = "SCALAR";
      DirectiveLocation2["OBJECT"] = "OBJECT";
      DirectiveLocation2["FIELD_DEFINITION"] = "FIELD_DEFINITION";
      DirectiveLocation2["ARGUMENT_DEFINITION"] = "ARGUMENT_DEFINITION";
      DirectiveLocation2["INTERFACE"] = "INTERFACE";
      DirectiveLocation2["UNION"] = "UNION";
      DirectiveLocation2["ENUM"] = "ENUM";
      DirectiveLocation2["ENUM_VALUE"] = "ENUM_VALUE";
      DirectiveLocation2["INPUT_OBJECT"] = "INPUT_OBJECT";
      DirectiveLocation2["INPUT_FIELD_DEFINITION"] = "INPUT_FIELD_DEFINITION";
    })(DirectiveLocation || (exports.DirectiveLocation = DirectiveLocation = {}));
  }
});

// ../../node_modules/graphql/language/kinds.js
var require_kinds = __commonJS({
  "../../node_modules/graphql/language/kinds.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.Kind = void 0;
    var Kind;
    exports.Kind = Kind;
    (function(Kind2) {
      Kind2["NAME"] = "Name";
      Kind2["DOCUMENT"] = "Document";
      Kind2["OPERATION_DEFINITION"] = "OperationDefinition";
      Kind2["VARIABLE_DEFINITION"] = "VariableDefinition";
      Kind2["SELECTION_SET"] = "SelectionSet";
      Kind2["FIELD"] = "Field";
      Kind2["ARGUMENT"] = "Argument";
      Kind2["FRAGMENT_SPREAD"] = "FragmentSpread";
      Kind2["INLINE_FRAGMENT"] = "InlineFragment";
      Kind2["FRAGMENT_DEFINITION"] = "FragmentDefinition";
      Kind2["VARIABLE"] = "Variable";
      Kind2["INT"] = "IntValue";
      Kind2["FLOAT"] = "FloatValue";
      Kind2["STRING"] = "StringValue";
      Kind2["BOOLEAN"] = "BooleanValue";
      Kind2["NULL"] = "NullValue";
      Kind2["ENUM"] = "EnumValue";
      Kind2["LIST"] = "ListValue";
      Kind2["OBJECT"] = "ObjectValue";
      Kind2["OBJECT_FIELD"] = "ObjectField";
      Kind2["DIRECTIVE"] = "Directive";
      Kind2["NAMED_TYPE"] = "NamedType";
      Kind2["LIST_TYPE"] = "ListType";
      Kind2["NON_NULL_TYPE"] = "NonNullType";
      Kind2["SCHEMA_DEFINITION"] = "SchemaDefinition";
      Kind2["OPERATION_TYPE_DEFINITION"] = "OperationTypeDefinition";
      Kind2["SCALAR_TYPE_DEFINITION"] = "ScalarTypeDefinition";
      Kind2["OBJECT_TYPE_DEFINITION"] = "ObjectTypeDefinition";
      Kind2["FIELD_DEFINITION"] = "FieldDefinition";
      Kind2["INPUT_VALUE_DEFINITION"] = "InputValueDefinition";
      Kind2["INTERFACE_TYPE_DEFINITION"] = "InterfaceTypeDefinition";
      Kind2["UNION_TYPE_DEFINITION"] = "UnionTypeDefinition";
      Kind2["ENUM_TYPE_DEFINITION"] = "EnumTypeDefinition";
      Kind2["ENUM_VALUE_DEFINITION"] = "EnumValueDefinition";
      Kind2["INPUT_OBJECT_TYPE_DEFINITION"] = "InputObjectTypeDefinition";
      Kind2["DIRECTIVE_DEFINITION"] = "DirectiveDefinition";
      Kind2["SCHEMA_EXTENSION"] = "SchemaExtension";
      Kind2["SCALAR_TYPE_EXTENSION"] = "ScalarTypeExtension";
      Kind2["OBJECT_TYPE_EXTENSION"] = "ObjectTypeExtension";
      Kind2["INTERFACE_TYPE_EXTENSION"] = "InterfaceTypeExtension";
      Kind2["UNION_TYPE_EXTENSION"] = "UnionTypeExtension";
      Kind2["ENUM_TYPE_EXTENSION"] = "EnumTypeExtension";
      Kind2["INPUT_OBJECT_TYPE_EXTENSION"] = "InputObjectTypeExtension";
    })(Kind || (exports.Kind = Kind = {}));
  }
});

// ../../node_modules/graphql/language/characterClasses.js
var require_characterClasses = __commonJS({
  "../../node_modules/graphql/language/characterClasses.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.isDigit = isDigit;
    exports.isLetter = isLetter;
    exports.isNameContinue = isNameContinue;
    exports.isNameStart = isNameStart;
    exports.isWhiteSpace = isWhiteSpace;
    function isWhiteSpace(code) {
      return code === 9 || code === 32;
    }
    __name(isWhiteSpace, "isWhiteSpace");
    function isDigit(code) {
      return code >= 48 && code <= 57;
    }
    __name(isDigit, "isDigit");
    function isLetter(code) {
      return code >= 97 && code <= 122 || // A-Z
      code >= 65 && code <= 90;
    }
    __name(isLetter, "isLetter");
    function isNameStart(code) {
      return isLetter(code) || code === 95;
    }
    __name(isNameStart, "isNameStart");
    function isNameContinue(code) {
      return isLetter(code) || isDigit(code) || code === 95;
    }
    __name(isNameContinue, "isNameContinue");
  }
});

// ../../node_modules/graphql/language/blockString.js
var require_blockString = __commonJS({
  "../../node_modules/graphql/language/blockString.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.dedentBlockStringLines = dedentBlockStringLines;
    exports.isPrintableAsBlockString = isPrintableAsBlockString;
    exports.printBlockString = printBlockString;
    var _characterClasses = require_characterClasses();
    function dedentBlockStringLines(lines) {
      var _firstNonEmptyLine2;
      let commonIndent = Number.MAX_SAFE_INTEGER;
      let firstNonEmptyLine = null;
      let lastNonEmptyLine = -1;
      for (let i = 0; i < lines.length; ++i) {
        var _firstNonEmptyLine;
        const line = lines[i];
        const indent = leadingWhitespace(line);
        if (indent === line.length) {
          continue;
        }
        firstNonEmptyLine = (_firstNonEmptyLine = firstNonEmptyLine) !== null && _firstNonEmptyLine !== void 0 ? _firstNonEmptyLine : i;
        lastNonEmptyLine = i;
        if (i !== 0 && indent < commonIndent) {
          commonIndent = indent;
        }
      }
      return lines.map((line, i) => i === 0 ? line : line.slice(commonIndent)).slice(
        (_firstNonEmptyLine2 = firstNonEmptyLine) !== null && _firstNonEmptyLine2 !== void 0 ? _firstNonEmptyLine2 : 0,
        lastNonEmptyLine + 1
      );
    }
    __name(dedentBlockStringLines, "dedentBlockStringLines");
    function leadingWhitespace(str) {
      let i = 0;
      while (i < str.length && (0, _characterClasses.isWhiteSpace)(str.charCodeAt(i))) {
        ++i;
      }
      return i;
    }
    __name(leadingWhitespace, "leadingWhitespace");
    function isPrintableAsBlockString(value) {
      if (value === "") {
        return true;
      }
      let isEmptyLine = true;
      let hasIndent = false;
      let hasCommonIndent = true;
      let seenNonEmptyLine = false;
      for (let i = 0; i < value.length; ++i) {
        switch (value.codePointAt(i)) {
          case 0:
          case 1:
          case 2:
          case 3:
          case 4:
          case 5:
          case 6:
          case 7:
          case 8:
          case 11:
          case 12:
          case 14:
          case 15:
            return false;
          // Has non-printable characters
          case 13:
            return false;
          // Has \r or \r\n which will be replaced as \n
          case 10:
            if (isEmptyLine && !seenNonEmptyLine) {
              return false;
            }
            seenNonEmptyLine = true;
            isEmptyLine = true;
            hasIndent = false;
            break;
          case 9:
          //   \t
          case 32:
            hasIndent || (hasIndent = isEmptyLine);
            break;
          default:
            hasCommonIndent && (hasCommonIndent = hasIndent);
            isEmptyLine = false;
        }
      }
      if (isEmptyLine) {
        return false;
      }
      if (hasCommonIndent && seenNonEmptyLine) {
        return false;
      }
      return true;
    }
    __name(isPrintableAsBlockString, "isPrintableAsBlockString");
    function printBlockString(value, options) {
      const escapedValue = value.replace(/"""/g, '\\"""');
      const lines = escapedValue.split(/\r\n|[\n\r]/g);
      const isSingleLine = lines.length === 1;
      const forceLeadingNewLine = lines.length > 1 && lines.slice(1).every(
        (line) => line.length === 0 || (0, _characterClasses.isWhiteSpace)(line.charCodeAt(0))
      );
      const hasTrailingTripleQuotes = escapedValue.endsWith('\\"""');
      const hasTrailingQuote = value.endsWith('"') && !hasTrailingTripleQuotes;
      const hasTrailingSlash = value.endsWith("\\");
      const forceTrailingNewline = hasTrailingQuote || hasTrailingSlash;
      const printAsMultipleLines = !(options !== null && options !== void 0 && options.minimize) && // add leading and trailing new lines only if it improves readability
      (!isSingleLine || value.length > 70 || forceTrailingNewline || forceLeadingNewLine || hasTrailingTripleQuotes);
      let result = "";
      const skipLeadingNewLine = isSingleLine && (0, _characterClasses.isWhiteSpace)(value.charCodeAt(0));
      if (printAsMultipleLines && !skipLeadingNewLine || forceLeadingNewLine) {
        result += "\n";
      }
      result += escapedValue;
      if (printAsMultipleLines || forceTrailingNewline) {
        result += "\n";
      }
      return '"""' + result + '"""';
    }
    __name(printBlockString, "printBlockString");
  }
});

// ../../node_modules/graphql/language/tokenKind.js
var require_tokenKind = __commonJS({
  "../../node_modules/graphql/language/tokenKind.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.TokenKind = void 0;
    var TokenKind;
    exports.TokenKind = TokenKind;
    (function(TokenKind2) {
      TokenKind2["SOF"] = "<SOF>";
      TokenKind2["EOF"] = "<EOF>";
      TokenKind2["BANG"] = "!";
      TokenKind2["DOLLAR"] = "$";
      TokenKind2["AMP"] = "&";
      TokenKind2["PAREN_L"] = "(";
      TokenKind2["PAREN_R"] = ")";
      TokenKind2["SPREAD"] = "...";
      TokenKind2["COLON"] = ":";
      TokenKind2["EQUALS"] = "=";
      TokenKind2["AT"] = "@";
      TokenKind2["BRACKET_L"] = "[";
      TokenKind2["BRACKET_R"] = "]";
      TokenKind2["BRACE_L"] = "{";
      TokenKind2["PIPE"] = "|";
      TokenKind2["BRACE_R"] = "}";
      TokenKind2["NAME"] = "Name";
      TokenKind2["INT"] = "Int";
      TokenKind2["FLOAT"] = "Float";
      TokenKind2["STRING"] = "String";
      TokenKind2["BLOCK_STRING"] = "BlockString";
      TokenKind2["COMMENT"] = "Comment";
    })(TokenKind || (exports.TokenKind = TokenKind = {}));
  }
});

// ../../node_modules/graphql/language/lexer.js
var require_lexer = __commonJS({
  "../../node_modules/graphql/language/lexer.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.Lexer = void 0;
    exports.isPunctuatorTokenKind = isPunctuatorTokenKind;
    var _syntaxError = require_syntaxError();
    var _ast = require_ast();
    var _blockString = require_blockString();
    var _characterClasses = require_characterClasses();
    var _tokenKind = require_tokenKind();
    var Lexer = class {
      static {
        __name(this, "Lexer");
      }
      /**
       * The previously focused non-ignored token.
       */
      /**
       * The currently focused non-ignored token.
       */
      /**
       * The (1-indexed) line containing the current token.
       */
      /**
       * The character offset at which the current line begins.
       */
      constructor(source) {
        const startOfFileToken = new _ast.Token(
          _tokenKind.TokenKind.SOF,
          0,
          0,
          0,
          0
        );
        this.source = source;
        this.lastToken = startOfFileToken;
        this.token = startOfFileToken;
        this.line = 1;
        this.lineStart = 0;
      }
      get [Symbol.toStringTag]() {
        return "Lexer";
      }
      /**
       * Advances the token stream to the next non-ignored token.
       */
      advance() {
        this.lastToken = this.token;
        const token = this.token = this.lookahead();
        return token;
      }
      /**
       * Looks ahead and returns the next non-ignored token, but does not change
       * the state of Lexer.
       */
      lookahead() {
        let token = this.token;
        if (token.kind !== _tokenKind.TokenKind.EOF) {
          do {
            if (token.next) {
              token = token.next;
            } else {
              const nextToken = readNextToken(this, token.end);
              token.next = nextToken;
              nextToken.prev = token;
              token = nextToken;
            }
          } while (token.kind === _tokenKind.TokenKind.COMMENT);
        }
        return token;
      }
    };
    exports.Lexer = Lexer;
    function isPunctuatorTokenKind(kind) {
      return kind === _tokenKind.TokenKind.BANG || kind === _tokenKind.TokenKind.DOLLAR || kind === _tokenKind.TokenKind.AMP || kind === _tokenKind.TokenKind.PAREN_L || kind === _tokenKind.TokenKind.PAREN_R || kind === _tokenKind.TokenKind.SPREAD || kind === _tokenKind.TokenKind.COLON || kind === _tokenKind.TokenKind.EQUALS || kind === _tokenKind.TokenKind.AT || kind === _tokenKind.TokenKind.BRACKET_L || kind === _tokenKind.TokenKind.BRACKET_R || kind === _tokenKind.TokenKind.BRACE_L || kind === _tokenKind.TokenKind.PIPE || kind === _tokenKind.TokenKind.BRACE_R;
    }
    __name(isPunctuatorTokenKind, "isPunctuatorTokenKind");
    function isUnicodeScalarValue(code) {
      return code >= 0 && code <= 55295 || code >= 57344 && code <= 1114111;
    }
    __name(isUnicodeScalarValue, "isUnicodeScalarValue");
    function isSupplementaryCodePoint(body, location) {
      return isLeadingSurrogate(body.charCodeAt(location)) && isTrailingSurrogate(body.charCodeAt(location + 1));
    }
    __name(isSupplementaryCodePoint, "isSupplementaryCodePoint");
    function isLeadingSurrogate(code) {
      return code >= 55296 && code <= 56319;
    }
    __name(isLeadingSurrogate, "isLeadingSurrogate");
    function isTrailingSurrogate(code) {
      return code >= 56320 && code <= 57343;
    }
    __name(isTrailingSurrogate, "isTrailingSurrogate");
    function printCodePointAt(lexer, location) {
      const code = lexer.source.body.codePointAt(location);
      if (code === void 0) {
        return _tokenKind.TokenKind.EOF;
      } else if (code >= 32 && code <= 126) {
        const char = String.fromCodePoint(code);
        return char === '"' ? `'"'` : `"${char}"`;
      }
      return "U+" + code.toString(16).toUpperCase().padStart(4, "0");
    }
    __name(printCodePointAt, "printCodePointAt");
    function createToken(lexer, kind, start, end, value) {
      const line = lexer.line;
      const col = 1 + start - lexer.lineStart;
      return new _ast.Token(kind, start, end, line, col, value);
    }
    __name(createToken, "createToken");
    function readNextToken(lexer, start) {
      const body = lexer.source.body;
      const bodyLength = body.length;
      let position = start;
      while (position < bodyLength) {
        const code = body.charCodeAt(position);
        switch (code) {
          // Ignored ::
          //   - UnicodeBOM
          //   - WhiteSpace
          //   - LineTerminator
          //   - Comment
          //   - Comma
          //
          // UnicodeBOM :: "Byte Order Mark (U+FEFF)"
          //
          // WhiteSpace ::
          //   - "Horizontal Tab (U+0009)"
          //   - "Space (U+0020)"
          //
          // Comma :: ,
          case 65279:
          // <BOM>
          case 9:
          // \t
          case 32:
          // <space>
          case 44:
            ++position;
            continue;
          // LineTerminator ::
          //   - "New Line (U+000A)"
          //   - "Carriage Return (U+000D)" [lookahead != "New Line (U+000A)"]
          //   - "Carriage Return (U+000D)" "New Line (U+000A)"
          case 10:
            ++position;
            ++lexer.line;
            lexer.lineStart = position;
            continue;
          case 13:
            if (body.charCodeAt(position + 1) === 10) {
              position += 2;
            } else {
              ++position;
            }
            ++lexer.line;
            lexer.lineStart = position;
            continue;
          // Comment
          case 35:
            return readComment(lexer, position);
          // Token ::
          //   - Punctuator
          //   - Name
          //   - IntValue
          //   - FloatValue
          //   - StringValue
          //
          // Punctuator :: one of ! $ & ( ) ... : = @ [ ] { | }
          case 33:
            return createToken(
              lexer,
              _tokenKind.TokenKind.BANG,
              position,
              position + 1
            );
          case 36:
            return createToken(
              lexer,
              _tokenKind.TokenKind.DOLLAR,
              position,
              position + 1
            );
          case 38:
            return createToken(
              lexer,
              _tokenKind.TokenKind.AMP,
              position,
              position + 1
            );
          case 40:
            return createToken(
              lexer,
              _tokenKind.TokenKind.PAREN_L,
              position,
              position + 1
            );
          case 41:
            return createToken(
              lexer,
              _tokenKind.TokenKind.PAREN_R,
              position,
              position + 1
            );
          case 46:
            if (body.charCodeAt(position + 1) === 46 && body.charCodeAt(position + 2) === 46) {
              return createToken(
                lexer,
                _tokenKind.TokenKind.SPREAD,
                position,
                position + 3
              );
            }
            break;
          case 58:
            return createToken(
              lexer,
              _tokenKind.TokenKind.COLON,
              position,
              position + 1
            );
          case 61:
            return createToken(
              lexer,
              _tokenKind.TokenKind.EQUALS,
              position,
              position + 1
            );
          case 64:
            return createToken(
              lexer,
              _tokenKind.TokenKind.AT,
              position,
              position + 1
            );
          case 91:
            return createToken(
              lexer,
              _tokenKind.TokenKind.BRACKET_L,
              position,
              position + 1
            );
          case 93:
            return createToken(
              lexer,
              _tokenKind.TokenKind.BRACKET_R,
              position,
              position + 1
            );
          case 123:
            return createToken(
              lexer,
              _tokenKind.TokenKind.BRACE_L,
              position,
              position + 1
            );
          case 124:
            return createToken(
              lexer,
              _tokenKind.TokenKind.PIPE,
              position,
              position + 1
            );
          case 125:
            return createToken(
              lexer,
              _tokenKind.TokenKind.BRACE_R,
              position,
              position + 1
            );
          // StringValue
          case 34:
            if (body.charCodeAt(position + 1) === 34 && body.charCodeAt(position + 2) === 34) {
              return readBlockString(lexer, position);
            }
            return readString(lexer, position);
        }
        if ((0, _characterClasses.isDigit)(code) || code === 45) {
          return readNumber(lexer, position, code);
        }
        if ((0, _characterClasses.isNameStart)(code)) {
          return readName(lexer, position);
        }
        throw (0, _syntaxError.syntaxError)(
          lexer.source,
          position,
          code === 39 ? `Unexpected single quote character ('), did you mean to use a double quote (")?` : isUnicodeScalarValue(code) || isSupplementaryCodePoint(body, position) ? `Unexpected character: ${printCodePointAt(lexer, position)}.` : `Invalid character: ${printCodePointAt(lexer, position)}.`
        );
      }
      return createToken(lexer, _tokenKind.TokenKind.EOF, bodyLength, bodyLength);
    }
    __name(readNextToken, "readNextToken");
    function readComment(lexer, start) {
      const body = lexer.source.body;
      const bodyLength = body.length;
      let position = start + 1;
      while (position < bodyLength) {
        const code = body.charCodeAt(position);
        if (code === 10 || code === 13) {
          break;
        }
        if (isUnicodeScalarValue(code)) {
          ++position;
        } else if (isSupplementaryCodePoint(body, position)) {
          position += 2;
        } else {
          break;
        }
      }
      return createToken(
        lexer,
        _tokenKind.TokenKind.COMMENT,
        start,
        position,
        body.slice(start + 1, position)
      );
    }
    __name(readComment, "readComment");
    function readNumber(lexer, start, firstCode) {
      const body = lexer.source.body;
      let position = start;
      let code = firstCode;
      let isFloat = false;
      if (code === 45) {
        code = body.charCodeAt(++position);
      }
      if (code === 48) {
        code = body.charCodeAt(++position);
        if ((0, _characterClasses.isDigit)(code)) {
          throw (0, _syntaxError.syntaxError)(
            lexer.source,
            position,
            `Invalid number, unexpected digit after 0: ${printCodePointAt(
              lexer,
              position
            )}.`
          );
        }
      } else {
        position = readDigits(lexer, position, code);
        code = body.charCodeAt(position);
      }
      if (code === 46) {
        isFloat = true;
        code = body.charCodeAt(++position);
        position = readDigits(lexer, position, code);
        code = body.charCodeAt(position);
      }
      if (code === 69 || code === 101) {
        isFloat = true;
        code = body.charCodeAt(++position);
        if (code === 43 || code === 45) {
          code = body.charCodeAt(++position);
        }
        position = readDigits(lexer, position, code);
        code = body.charCodeAt(position);
      }
      if (code === 46 || (0, _characterClasses.isNameStart)(code)) {
        throw (0, _syntaxError.syntaxError)(
          lexer.source,
          position,
          `Invalid number, expected digit but got: ${printCodePointAt(
            lexer,
            position
          )}.`
        );
      }
      return createToken(
        lexer,
        isFloat ? _tokenKind.TokenKind.FLOAT : _tokenKind.TokenKind.INT,
        start,
        position,
        body.slice(start, position)
      );
    }
    __name(readNumber, "readNumber");
    function readDigits(lexer, start, firstCode) {
      if (!(0, _characterClasses.isDigit)(firstCode)) {
        throw (0, _syntaxError.syntaxError)(
          lexer.source,
          start,
          `Invalid number, expected digit but got: ${printCodePointAt(
            lexer,
            start
          )}.`
        );
      }
      const body = lexer.source.body;
      let position = start + 1;
      while ((0, _characterClasses.isDigit)(body.charCodeAt(position))) {
        ++position;
      }
      return position;
    }
    __name(readDigits, "readDigits");
    function readString(lexer, start) {
      const body = lexer.source.body;
      const bodyLength = body.length;
      let position = start + 1;
      let chunkStart = position;
      let value = "";
      while (position < bodyLength) {
        const code = body.charCodeAt(position);
        if (code === 34) {
          value += body.slice(chunkStart, position);
          return createToken(
            lexer,
            _tokenKind.TokenKind.STRING,
            start,
            position + 1,
            value
          );
        }
        if (code === 92) {
          value += body.slice(chunkStart, position);
          const escape = body.charCodeAt(position + 1) === 117 ? body.charCodeAt(position + 2) === 123 ? readEscapedUnicodeVariableWidth(lexer, position) : readEscapedUnicodeFixedWidth(lexer, position) : readEscapedCharacter(lexer, position);
          value += escape.value;
          position += escape.size;
          chunkStart = position;
          continue;
        }
        if (code === 10 || code === 13) {
          break;
        }
        if (isUnicodeScalarValue(code)) {
          ++position;
        } else if (isSupplementaryCodePoint(body, position)) {
          position += 2;
        } else {
          throw (0, _syntaxError.syntaxError)(
            lexer.source,
            position,
            `Invalid character within String: ${printCodePointAt(
              lexer,
              position
            )}.`
          );
        }
      }
      throw (0, _syntaxError.syntaxError)(
        lexer.source,
        position,
        "Unterminated string."
      );
    }
    __name(readString, "readString");
    function readEscapedUnicodeVariableWidth(lexer, position) {
      const body = lexer.source.body;
      let point = 0;
      let size = 3;
      while (size < 12) {
        const code = body.charCodeAt(position + size++);
        if (code === 125) {
          if (size < 5 || !isUnicodeScalarValue(point)) {
            break;
          }
          return {
            value: String.fromCodePoint(point),
            size
          };
        }
        point = point << 4 | readHexDigit(code);
        if (point < 0) {
          break;
        }
      }
      throw (0, _syntaxError.syntaxError)(
        lexer.source,
        position,
        `Invalid Unicode escape sequence: "${body.slice(
          position,
          position + size
        )}".`
      );
    }
    __name(readEscapedUnicodeVariableWidth, "readEscapedUnicodeVariableWidth");
    function readEscapedUnicodeFixedWidth(lexer, position) {
      const body = lexer.source.body;
      const code = read16BitHexCode(body, position + 2);
      if (isUnicodeScalarValue(code)) {
        return {
          value: String.fromCodePoint(code),
          size: 6
        };
      }
      if (isLeadingSurrogate(code)) {
        if (body.charCodeAt(position + 6) === 92 && body.charCodeAt(position + 7) === 117) {
          const trailingCode = read16BitHexCode(body, position + 8);
          if (isTrailingSurrogate(trailingCode)) {
            return {
              value: String.fromCodePoint(code, trailingCode),
              size: 12
            };
          }
        }
      }
      throw (0, _syntaxError.syntaxError)(
        lexer.source,
        position,
        `Invalid Unicode escape sequence: "${body.slice(position, position + 6)}".`
      );
    }
    __name(readEscapedUnicodeFixedWidth, "readEscapedUnicodeFixedWidth");
    function read16BitHexCode(body, position) {
      return readHexDigit(body.charCodeAt(position)) << 12 | readHexDigit(body.charCodeAt(position + 1)) << 8 | readHexDigit(body.charCodeAt(position + 2)) << 4 | readHexDigit(body.charCodeAt(position + 3));
    }
    __name(read16BitHexCode, "read16BitHexCode");
    function readHexDigit(code) {
      return code >= 48 && code <= 57 ? code - 48 : code >= 65 && code <= 70 ? code - 55 : code >= 97 && code <= 102 ? code - 87 : -1;
    }
    __name(readHexDigit, "readHexDigit");
    function readEscapedCharacter(lexer, position) {
      const body = lexer.source.body;
      const code = body.charCodeAt(position + 1);
      switch (code) {
        case 34:
          return {
            value: '"',
            size: 2
          };
        case 92:
          return {
            value: "\\",
            size: 2
          };
        case 47:
          return {
            value: "/",
            size: 2
          };
        case 98:
          return {
            value: "\b",
            size: 2
          };
        case 102:
          return {
            value: "\f",
            size: 2
          };
        case 110:
          return {
            value: "\n",
            size: 2
          };
        case 114:
          return {
            value: "\r",
            size: 2
          };
        case 116:
          return {
            value: "	",
            size: 2
          };
      }
      throw (0, _syntaxError.syntaxError)(
        lexer.source,
        position,
        `Invalid character escape sequence: "${body.slice(
          position,
          position + 2
        )}".`
      );
    }
    __name(readEscapedCharacter, "readEscapedCharacter");
    function readBlockString(lexer, start) {
      const body = lexer.source.body;
      const bodyLength = body.length;
      let lineStart = lexer.lineStart;
      let position = start + 3;
      let chunkStart = position;
      let currentLine = "";
      const blockLines = [];
      while (position < bodyLength) {
        const code = body.charCodeAt(position);
        if (code === 34 && body.charCodeAt(position + 1) === 34 && body.charCodeAt(position + 2) === 34) {
          currentLine += body.slice(chunkStart, position);
          blockLines.push(currentLine);
          const token = createToken(
            lexer,
            _tokenKind.TokenKind.BLOCK_STRING,
            start,
            position + 3,
            // Return a string of the lines joined with U+000A.
            (0, _blockString.dedentBlockStringLines)(blockLines).join("\n")
          );
          lexer.line += blockLines.length - 1;
          lexer.lineStart = lineStart;
          return token;
        }
        if (code === 92 && body.charCodeAt(position + 1) === 34 && body.charCodeAt(position + 2) === 34 && body.charCodeAt(position + 3) === 34) {
          currentLine += body.slice(chunkStart, position);
          chunkStart = position + 1;
          position += 4;
          continue;
        }
        if (code === 10 || code === 13) {
          currentLine += body.slice(chunkStart, position);
          blockLines.push(currentLine);
          if (code === 13 && body.charCodeAt(position + 1) === 10) {
            position += 2;
          } else {
            ++position;
          }
          currentLine = "";
          chunkStart = position;
          lineStart = position;
          continue;
        }
        if (isUnicodeScalarValue(code)) {
          ++position;
        } else if (isSupplementaryCodePoint(body, position)) {
          position += 2;
        } else {
          throw (0, _syntaxError.syntaxError)(
            lexer.source,
            position,
            `Invalid character within String: ${printCodePointAt(
              lexer,
              position
            )}.`
          );
        }
      }
      throw (0, _syntaxError.syntaxError)(
        lexer.source,
        position,
        "Unterminated string."
      );
    }
    __name(readBlockString, "readBlockString");
    function readName(lexer, start) {
      const body = lexer.source.body;
      const bodyLength = body.length;
      let position = start + 1;
      while (position < bodyLength) {
        const code = body.charCodeAt(position);
        if ((0, _characterClasses.isNameContinue)(code)) {
          ++position;
        } else {
          break;
        }
      }
      return createToken(
        lexer,
        _tokenKind.TokenKind.NAME,
        start,
        position,
        body.slice(start, position)
      );
    }
    __name(readName, "readName");
  }
});

// ../../node_modules/graphql/jsutils/inspect.js
var require_inspect = __commonJS({
  "../../node_modules/graphql/jsutils/inspect.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.inspect = inspect;
    var MAX_ARRAY_LENGTH = 10;
    var MAX_RECURSIVE_DEPTH = 2;
    function inspect(value) {
      return formatValue(value, []);
    }
    __name(inspect, "inspect");
    function formatValue(value, seenValues) {
      switch (typeof value) {
        case "string":
          return JSON.stringify(value);
        case "function":
          return value.name ? `[function ${value.name}]` : "[function]";
        case "object":
          return formatObjectValue(value, seenValues);
        default:
          return String(value);
      }
    }
    __name(formatValue, "formatValue");
    function formatObjectValue(value, previouslySeenValues) {
      if (value === null) {
        return "null";
      }
      if (previouslySeenValues.includes(value)) {
        return "[Circular]";
      }
      const seenValues = [...previouslySeenValues, value];
      if (isJSONable(value)) {
        const jsonValue = value.toJSON();
        if (jsonValue !== value) {
          return typeof jsonValue === "string" ? jsonValue : formatValue(jsonValue, seenValues);
        }
      } else if (Array.isArray(value)) {
        return formatArray(value, seenValues);
      }
      return formatObject(value, seenValues);
    }
    __name(formatObjectValue, "formatObjectValue");
    function isJSONable(value) {
      return typeof value.toJSON === "function";
    }
    __name(isJSONable, "isJSONable");
    function formatObject(object, seenValues) {
      const entries = Object.entries(object);
      if (entries.length === 0) {
        return "{}";
      }
      if (seenValues.length > MAX_RECURSIVE_DEPTH) {
        return "[" + getObjectTag(object) + "]";
      }
      const properties = entries.map(
        ([key, value]) => key + ": " + formatValue(value, seenValues)
      );
      return "{ " + properties.join(", ") + " }";
    }
    __name(formatObject, "formatObject");
    function formatArray(array, seenValues) {
      if (array.length === 0) {
        return "[]";
      }
      if (seenValues.length > MAX_RECURSIVE_DEPTH) {
        return "[Array]";
      }
      const len = Math.min(MAX_ARRAY_LENGTH, array.length);
      const remaining = array.length - len;
      const items = [];
      for (let i = 0; i < len; ++i) {
        items.push(formatValue(array[i], seenValues));
      }
      if (remaining === 1) {
        items.push("... 1 more item");
      } else if (remaining > 1) {
        items.push(`... ${remaining} more items`);
      }
      return "[" + items.join(", ") + "]";
    }
    __name(formatArray, "formatArray");
    function getObjectTag(object) {
      const tag = Object.prototype.toString.call(object).replace(/^\[object /, "").replace(/]$/, "");
      if (tag === "Object" && typeof object.constructor === "function") {
        const name = object.constructor.name;
        if (typeof name === "string" && name !== "") {
          return name;
        }
      }
      return tag;
    }
    __name(getObjectTag, "getObjectTag");
  }
});

// ../../node_modules/graphql/jsutils/instanceOf.js
var require_instanceOf = __commonJS({
  "../../node_modules/graphql/jsutils/instanceOf.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.instanceOf = void 0;
    var _inspect = require_inspect();
    var isProduction = globalThis.process && // eslint-disable-next-line no-undef
    process.env.NODE_ENV === "production";
    var instanceOf = (
      /* c8 ignore next 6 */
      // FIXME: https://github.com/graphql/graphql-js/issues/2317
      isProduction ? /* @__PURE__ */ __name(function instanceOf2(value, constructor) {
        return value instanceof constructor;
      }, "instanceOf") : /* @__PURE__ */ __name(function instanceOf2(value, constructor) {
        if (value instanceof constructor) {
          return true;
        }
        if (typeof value === "object" && value !== null) {
          var _value$constructor;
          const className = constructor.prototype[Symbol.toStringTag];
          const valueClassName = (
            // We still need to support constructor's name to detect conflicts with older versions of this library.
            Symbol.toStringTag in value ? value[Symbol.toStringTag] : (_value$constructor = value.constructor) === null || _value$constructor === void 0 ? void 0 : _value$constructor.name
          );
          if (className === valueClassName) {
            const stringifiedValue = (0, _inspect.inspect)(value);
            throw new Error(`Cannot use ${className} "${stringifiedValue}" from another module or realm.

Ensure that there is only one instance of "graphql" in the node_modules
directory. If different versions of "graphql" are the dependencies of other
relied on modules, use "resolutions" to ensure only one version is installed.

https://yarnpkg.com/en/docs/selective-version-resolutions

Duplicate "graphql" modules cannot be used at the same time since different
versions may have different capabilities and behavior. The data from one
version used in the function from another could produce confusing and
spurious results.`);
          }
        }
        return false;
      }, "instanceOf")
    );
    exports.instanceOf = instanceOf;
  }
});

// ../../node_modules/graphql/language/source.js
var require_source = __commonJS({
  "../../node_modules/graphql/language/source.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.Source = void 0;
    exports.isSource = isSource;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _instanceOf = require_instanceOf();
    var Source = class {
      static {
        __name(this, "Source");
      }
      constructor(body, name = "GraphQL request", locationOffset = {
        line: 1,
        column: 1
      }) {
        typeof body === "string" || (0, _devAssert.devAssert)(
          false,
          `Body must be a string. Received: ${(0, _inspect.inspect)(body)}.`
        );
        this.body = body;
        this.name = name;
        this.locationOffset = locationOffset;
        this.locationOffset.line > 0 || (0, _devAssert.devAssert)(
          false,
          "line in locationOffset is 1-indexed and must be positive."
        );
        this.locationOffset.column > 0 || (0, _devAssert.devAssert)(
          false,
          "column in locationOffset is 1-indexed and must be positive."
        );
      }
      get [Symbol.toStringTag]() {
        return "Source";
      }
    };
    exports.Source = Source;
    function isSource(source) {
      return (0, _instanceOf.instanceOf)(source, Source);
    }
    __name(isSource, "isSource");
  }
});

// ../../node_modules/graphql/language/parser.js
var require_parser = __commonJS({
  "../../node_modules/graphql/language/parser.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.Parser = void 0;
    exports.parse = parse;
    exports.parseConstValue = parseConstValue;
    exports.parseType = parseType;
    exports.parseValue = parseValue;
    var _syntaxError = require_syntaxError();
    var _ast = require_ast();
    var _directiveLocation = require_directiveLocation();
    var _kinds = require_kinds();
    var _lexer = require_lexer();
    var _source = require_source();
    var _tokenKind = require_tokenKind();
    function parse(source, options) {
      const parser = new Parser(source, options);
      const document = parser.parseDocument();
      Object.defineProperty(document, "tokenCount", {
        enumerable: false,
        value: parser.tokenCount
      });
      return document;
    }
    __name(parse, "parse");
    function parseValue(source, options) {
      const parser = new Parser(source, options);
      parser.expectToken(_tokenKind.TokenKind.SOF);
      const value = parser.parseValueLiteral(false);
      parser.expectToken(_tokenKind.TokenKind.EOF);
      return value;
    }
    __name(parseValue, "parseValue");
    function parseConstValue(source, options) {
      const parser = new Parser(source, options);
      parser.expectToken(_tokenKind.TokenKind.SOF);
      const value = parser.parseConstValueLiteral();
      parser.expectToken(_tokenKind.TokenKind.EOF);
      return value;
    }
    __name(parseConstValue, "parseConstValue");
    function parseType(source, options) {
      const parser = new Parser(source, options);
      parser.expectToken(_tokenKind.TokenKind.SOF);
      const type = parser.parseTypeReference();
      parser.expectToken(_tokenKind.TokenKind.EOF);
      return type;
    }
    __name(parseType, "parseType");
    var Parser = class {
      static {
        __name(this, "Parser");
      }
      constructor(source, options = {}) {
        const sourceObj = (0, _source.isSource)(source) ? source : new _source.Source(source);
        this._lexer = new _lexer.Lexer(sourceObj);
        this._options = options;
        this._tokenCounter = 0;
      }
      get tokenCount() {
        return this._tokenCounter;
      }
      /**
       * Converts a name lex token into a name parse node.
       */
      parseName() {
        const token = this.expectToken(_tokenKind.TokenKind.NAME);
        return this.node(token, {
          kind: _kinds.Kind.NAME,
          value: token.value
        });
      }
      // Implements the parsing rules in the Document section.
      /**
       * Document : Definition+
       */
      parseDocument() {
        return this.node(this._lexer.token, {
          kind: _kinds.Kind.DOCUMENT,
          definitions: this.many(
            _tokenKind.TokenKind.SOF,
            this.parseDefinition,
            _tokenKind.TokenKind.EOF
          )
        });
      }
      /**
       * Definition :
       *   - ExecutableDefinition
       *   - TypeSystemDefinition
       *   - TypeSystemExtension
       *
       * ExecutableDefinition :
       *   - OperationDefinition
       *   - FragmentDefinition
       *
       * TypeSystemDefinition :
       *   - SchemaDefinition
       *   - TypeDefinition
       *   - DirectiveDefinition
       *
       * TypeDefinition :
       *   - ScalarTypeDefinition
       *   - ObjectTypeDefinition
       *   - InterfaceTypeDefinition
       *   - UnionTypeDefinition
       *   - EnumTypeDefinition
       *   - InputObjectTypeDefinition
       */
      parseDefinition() {
        if (this.peek(_tokenKind.TokenKind.BRACE_L)) {
          return this.parseOperationDefinition();
        }
        const hasDescription = this.peekDescription();
        const keywordToken = hasDescription ? this._lexer.lookahead() : this._lexer.token;
        if (keywordToken.kind === _tokenKind.TokenKind.NAME) {
          switch (keywordToken.value) {
            case "schema":
              return this.parseSchemaDefinition();
            case "scalar":
              return this.parseScalarTypeDefinition();
            case "type":
              return this.parseObjectTypeDefinition();
            case "interface":
              return this.parseInterfaceTypeDefinition();
            case "union":
              return this.parseUnionTypeDefinition();
            case "enum":
              return this.parseEnumTypeDefinition();
            case "input":
              return this.parseInputObjectTypeDefinition();
            case "directive":
              return this.parseDirectiveDefinition();
          }
          if (hasDescription) {
            throw (0, _syntaxError.syntaxError)(
              this._lexer.source,
              this._lexer.token.start,
              "Unexpected description, descriptions are supported only on type definitions."
            );
          }
          switch (keywordToken.value) {
            case "query":
            case "mutation":
            case "subscription":
              return this.parseOperationDefinition();
            case "fragment":
              return this.parseFragmentDefinition();
            case "extend":
              return this.parseTypeSystemExtension();
          }
        }
        throw this.unexpected(keywordToken);
      }
      // Implements the parsing rules in the Operations section.
      /**
       * OperationDefinition :
       *  - SelectionSet
       *  - OperationType Name? VariableDefinitions? Directives? SelectionSet
       */
      parseOperationDefinition() {
        const start = this._lexer.token;
        if (this.peek(_tokenKind.TokenKind.BRACE_L)) {
          return this.node(start, {
            kind: _kinds.Kind.OPERATION_DEFINITION,
            operation: _ast.OperationTypeNode.QUERY,
            name: void 0,
            variableDefinitions: [],
            directives: [],
            selectionSet: this.parseSelectionSet()
          });
        }
        const operation = this.parseOperationType();
        let name;
        if (this.peek(_tokenKind.TokenKind.NAME)) {
          name = this.parseName();
        }
        return this.node(start, {
          kind: _kinds.Kind.OPERATION_DEFINITION,
          operation,
          name,
          variableDefinitions: this.parseVariableDefinitions(),
          directives: this.parseDirectives(false),
          selectionSet: this.parseSelectionSet()
        });
      }
      /**
       * OperationType : one of query mutation subscription
       */
      parseOperationType() {
        const operationToken = this.expectToken(_tokenKind.TokenKind.NAME);
        switch (operationToken.value) {
          case "query":
            return _ast.OperationTypeNode.QUERY;
          case "mutation":
            return _ast.OperationTypeNode.MUTATION;
          case "subscription":
            return _ast.OperationTypeNode.SUBSCRIPTION;
        }
        throw this.unexpected(operationToken);
      }
      /**
       * VariableDefinitions : ( VariableDefinition+ )
       */
      parseVariableDefinitions() {
        return this.optionalMany(
          _tokenKind.TokenKind.PAREN_L,
          this.parseVariableDefinition,
          _tokenKind.TokenKind.PAREN_R
        );
      }
      /**
       * VariableDefinition : Variable : Type DefaultValue? Directives[Const]?
       */
      parseVariableDefinition() {
        return this.node(this._lexer.token, {
          kind: _kinds.Kind.VARIABLE_DEFINITION,
          variable: this.parseVariable(),
          type: (this.expectToken(_tokenKind.TokenKind.COLON), this.parseTypeReference()),
          defaultValue: this.expectOptionalToken(_tokenKind.TokenKind.EQUALS) ? this.parseConstValueLiteral() : void 0,
          directives: this.parseConstDirectives()
        });
      }
      /**
       * Variable : $ Name
       */
      parseVariable() {
        const start = this._lexer.token;
        this.expectToken(_tokenKind.TokenKind.DOLLAR);
        return this.node(start, {
          kind: _kinds.Kind.VARIABLE,
          name: this.parseName()
        });
      }
      /**
       * ```
       * SelectionSet : { Selection+ }
       * ```
       */
      parseSelectionSet() {
        return this.node(this._lexer.token, {
          kind: _kinds.Kind.SELECTION_SET,
          selections: this.many(
            _tokenKind.TokenKind.BRACE_L,
            this.parseSelection,
            _tokenKind.TokenKind.BRACE_R
          )
        });
      }
      /**
       * Selection :
       *   - Field
       *   - FragmentSpread
       *   - InlineFragment
       */
      parseSelection() {
        return this.peek(_tokenKind.TokenKind.SPREAD) ? this.parseFragment() : this.parseField();
      }
      /**
       * Field : Alias? Name Arguments? Directives? SelectionSet?
       *
       * Alias : Name :
       */
      parseField() {
        const start = this._lexer.token;
        const nameOrAlias = this.parseName();
        let alias;
        let name;
        if (this.expectOptionalToken(_tokenKind.TokenKind.COLON)) {
          alias = nameOrAlias;
          name = this.parseName();
        } else {
          name = nameOrAlias;
        }
        return this.node(start, {
          kind: _kinds.Kind.FIELD,
          alias,
          name,
          arguments: this.parseArguments(false),
          directives: this.parseDirectives(false),
          selectionSet: this.peek(_tokenKind.TokenKind.BRACE_L) ? this.parseSelectionSet() : void 0
        });
      }
      /**
       * Arguments[Const] : ( Argument[?Const]+ )
       */
      parseArguments(isConst) {
        const item = isConst ? this.parseConstArgument : this.parseArgument;
        return this.optionalMany(
          _tokenKind.TokenKind.PAREN_L,
          item,
          _tokenKind.TokenKind.PAREN_R
        );
      }
      /**
       * Argument[Const] : Name : Value[?Const]
       */
      parseArgument(isConst = false) {
        const start = this._lexer.token;
        const name = this.parseName();
        this.expectToken(_tokenKind.TokenKind.COLON);
        return this.node(start, {
          kind: _kinds.Kind.ARGUMENT,
          name,
          value: this.parseValueLiteral(isConst)
        });
      }
      parseConstArgument() {
        return this.parseArgument(true);
      }
      // Implements the parsing rules in the Fragments section.
      /**
       * Corresponds to both FragmentSpread and InlineFragment in the spec.
       *
       * FragmentSpread : ... FragmentName Directives?
       *
       * InlineFragment : ... TypeCondition? Directives? SelectionSet
       */
      parseFragment() {
        const start = this._lexer.token;
        this.expectToken(_tokenKind.TokenKind.SPREAD);
        const hasTypeCondition = this.expectOptionalKeyword("on");
        if (!hasTypeCondition && this.peek(_tokenKind.TokenKind.NAME)) {
          return this.node(start, {
            kind: _kinds.Kind.FRAGMENT_SPREAD,
            name: this.parseFragmentName(),
            directives: this.parseDirectives(false)
          });
        }
        return this.node(start, {
          kind: _kinds.Kind.INLINE_FRAGMENT,
          typeCondition: hasTypeCondition ? this.parseNamedType() : void 0,
          directives: this.parseDirectives(false),
          selectionSet: this.parseSelectionSet()
        });
      }
      /**
       * FragmentDefinition :
       *   - fragment FragmentName on TypeCondition Directives? SelectionSet
       *
       * TypeCondition : NamedType
       */
      parseFragmentDefinition() {
        const start = this._lexer.token;
        this.expectKeyword("fragment");
        if (this._options.allowLegacyFragmentVariables === true) {
          return this.node(start, {
            kind: _kinds.Kind.FRAGMENT_DEFINITION,
            name: this.parseFragmentName(),
            variableDefinitions: this.parseVariableDefinitions(),
            typeCondition: (this.expectKeyword("on"), this.parseNamedType()),
            directives: this.parseDirectives(false),
            selectionSet: this.parseSelectionSet()
          });
        }
        return this.node(start, {
          kind: _kinds.Kind.FRAGMENT_DEFINITION,
          name: this.parseFragmentName(),
          typeCondition: (this.expectKeyword("on"), this.parseNamedType()),
          directives: this.parseDirectives(false),
          selectionSet: this.parseSelectionSet()
        });
      }
      /**
       * FragmentName : Name but not `on`
       */
      parseFragmentName() {
        if (this._lexer.token.value === "on") {
          throw this.unexpected();
        }
        return this.parseName();
      }
      // Implements the parsing rules in the Values section.
      /**
       * Value[Const] :
       *   - [~Const] Variable
       *   - IntValue
       *   - FloatValue
       *   - StringValue
       *   - BooleanValue
       *   - NullValue
       *   - EnumValue
       *   - ListValue[?Const]
       *   - ObjectValue[?Const]
       *
       * BooleanValue : one of `true` `false`
       *
       * NullValue : `null`
       *
       * EnumValue : Name but not `true`, `false` or `null`
       */
      parseValueLiteral(isConst) {
        const token = this._lexer.token;
        switch (token.kind) {
          case _tokenKind.TokenKind.BRACKET_L:
            return this.parseList(isConst);
          case _tokenKind.TokenKind.BRACE_L:
            return this.parseObject(isConst);
          case _tokenKind.TokenKind.INT:
            this.advanceLexer();
            return this.node(token, {
              kind: _kinds.Kind.INT,
              value: token.value
            });
          case _tokenKind.TokenKind.FLOAT:
            this.advanceLexer();
            return this.node(token, {
              kind: _kinds.Kind.FLOAT,
              value: token.value
            });
          case _tokenKind.TokenKind.STRING:
          case _tokenKind.TokenKind.BLOCK_STRING:
            return this.parseStringLiteral();
          case _tokenKind.TokenKind.NAME:
            this.advanceLexer();
            switch (token.value) {
              case "true":
                return this.node(token, {
                  kind: _kinds.Kind.BOOLEAN,
                  value: true
                });
              case "false":
                return this.node(token, {
                  kind: _kinds.Kind.BOOLEAN,
                  value: false
                });
              case "null":
                return this.node(token, {
                  kind: _kinds.Kind.NULL
                });
              default:
                return this.node(token, {
                  kind: _kinds.Kind.ENUM,
                  value: token.value
                });
            }
          case _tokenKind.TokenKind.DOLLAR:
            if (isConst) {
              this.expectToken(_tokenKind.TokenKind.DOLLAR);
              if (this._lexer.token.kind === _tokenKind.TokenKind.NAME) {
                const varName = this._lexer.token.value;
                throw (0, _syntaxError.syntaxError)(
                  this._lexer.source,
                  token.start,
                  `Unexpected variable "$${varName}" in constant value.`
                );
              } else {
                throw this.unexpected(token);
              }
            }
            return this.parseVariable();
          default:
            throw this.unexpected();
        }
      }
      parseConstValueLiteral() {
        return this.parseValueLiteral(true);
      }
      parseStringLiteral() {
        const token = this._lexer.token;
        this.advanceLexer();
        return this.node(token, {
          kind: _kinds.Kind.STRING,
          value: token.value,
          block: token.kind === _tokenKind.TokenKind.BLOCK_STRING
        });
      }
      /**
       * ListValue[Const] :
       *   - [ ]
       *   - [ Value[?Const]+ ]
       */
      parseList(isConst) {
        const item = /* @__PURE__ */ __name(() => this.parseValueLiteral(isConst), "item");
        return this.node(this._lexer.token, {
          kind: _kinds.Kind.LIST,
          values: this.any(
            _tokenKind.TokenKind.BRACKET_L,
            item,
            _tokenKind.TokenKind.BRACKET_R
          )
        });
      }
      /**
       * ```
       * ObjectValue[Const] :
       *   - { }
       *   - { ObjectField[?Const]+ }
       * ```
       */
      parseObject(isConst) {
        const item = /* @__PURE__ */ __name(() => this.parseObjectField(isConst), "item");
        return this.node(this._lexer.token, {
          kind: _kinds.Kind.OBJECT,
          fields: this.any(
            _tokenKind.TokenKind.BRACE_L,
            item,
            _tokenKind.TokenKind.BRACE_R
          )
        });
      }
      /**
       * ObjectField[Const] : Name : Value[?Const]
       */
      parseObjectField(isConst) {
        const start = this._lexer.token;
        const name = this.parseName();
        this.expectToken(_tokenKind.TokenKind.COLON);
        return this.node(start, {
          kind: _kinds.Kind.OBJECT_FIELD,
          name,
          value: this.parseValueLiteral(isConst)
        });
      }
      // Implements the parsing rules in the Directives section.
      /**
       * Directives[Const] : Directive[?Const]+
       */
      parseDirectives(isConst) {
        const directives = [];
        while (this.peek(_tokenKind.TokenKind.AT)) {
          directives.push(this.parseDirective(isConst));
        }
        return directives;
      }
      parseConstDirectives() {
        return this.parseDirectives(true);
      }
      /**
       * ```
       * Directive[Const] : @ Name Arguments[?Const]?
       * ```
       */
      parseDirective(isConst) {
        const start = this._lexer.token;
        this.expectToken(_tokenKind.TokenKind.AT);
        return this.node(start, {
          kind: _kinds.Kind.DIRECTIVE,
          name: this.parseName(),
          arguments: this.parseArguments(isConst)
        });
      }
      // Implements the parsing rules in the Types section.
      /**
       * Type :
       *   - NamedType
       *   - ListType
       *   - NonNullType
       */
      parseTypeReference() {
        const start = this._lexer.token;
        let type;
        if (this.expectOptionalToken(_tokenKind.TokenKind.BRACKET_L)) {
          const innerType = this.parseTypeReference();
          this.expectToken(_tokenKind.TokenKind.BRACKET_R);
          type = this.node(start, {
            kind: _kinds.Kind.LIST_TYPE,
            type: innerType
          });
        } else {
          type = this.parseNamedType();
        }
        if (this.expectOptionalToken(_tokenKind.TokenKind.BANG)) {
          return this.node(start, {
            kind: _kinds.Kind.NON_NULL_TYPE,
            type
          });
        }
        return type;
      }
      /**
       * NamedType : Name
       */
      parseNamedType() {
        return this.node(this._lexer.token, {
          kind: _kinds.Kind.NAMED_TYPE,
          name: this.parseName()
        });
      }
      // Implements the parsing rules in the Type Definition section.
      peekDescription() {
        return this.peek(_tokenKind.TokenKind.STRING) || this.peek(_tokenKind.TokenKind.BLOCK_STRING);
      }
      /**
       * Description : StringValue
       */
      parseDescription() {
        if (this.peekDescription()) {
          return this.parseStringLiteral();
        }
      }
      /**
       * ```
       * SchemaDefinition : Description? schema Directives[Const]? { OperationTypeDefinition+ }
       * ```
       */
      parseSchemaDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("schema");
        const directives = this.parseConstDirectives();
        const operationTypes = this.many(
          _tokenKind.TokenKind.BRACE_L,
          this.parseOperationTypeDefinition,
          _tokenKind.TokenKind.BRACE_R
        );
        return this.node(start, {
          kind: _kinds.Kind.SCHEMA_DEFINITION,
          description,
          directives,
          operationTypes
        });
      }
      /**
       * OperationTypeDefinition : OperationType : NamedType
       */
      parseOperationTypeDefinition() {
        const start = this._lexer.token;
        const operation = this.parseOperationType();
        this.expectToken(_tokenKind.TokenKind.COLON);
        const type = this.parseNamedType();
        return this.node(start, {
          kind: _kinds.Kind.OPERATION_TYPE_DEFINITION,
          operation,
          type
        });
      }
      /**
       * ScalarTypeDefinition : Description? scalar Name Directives[Const]?
       */
      parseScalarTypeDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("scalar");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        return this.node(start, {
          kind: _kinds.Kind.SCALAR_TYPE_DEFINITION,
          description,
          name,
          directives
        });
      }
      /**
       * ObjectTypeDefinition :
       *   Description?
       *   type Name ImplementsInterfaces? Directives[Const]? FieldsDefinition?
       */
      parseObjectTypeDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("type");
        const name = this.parseName();
        const interfaces = this.parseImplementsInterfaces();
        const directives = this.parseConstDirectives();
        const fields = this.parseFieldsDefinition();
        return this.node(start, {
          kind: _kinds.Kind.OBJECT_TYPE_DEFINITION,
          description,
          name,
          interfaces,
          directives,
          fields
        });
      }
      /**
       * ImplementsInterfaces :
       *   - implements `&`? NamedType
       *   - ImplementsInterfaces & NamedType
       */
      parseImplementsInterfaces() {
        return this.expectOptionalKeyword("implements") ? this.delimitedMany(_tokenKind.TokenKind.AMP, this.parseNamedType) : [];
      }
      /**
       * ```
       * FieldsDefinition : { FieldDefinition+ }
       * ```
       */
      parseFieldsDefinition() {
        return this.optionalMany(
          _tokenKind.TokenKind.BRACE_L,
          this.parseFieldDefinition,
          _tokenKind.TokenKind.BRACE_R
        );
      }
      /**
       * FieldDefinition :
       *   - Description? Name ArgumentsDefinition? : Type Directives[Const]?
       */
      parseFieldDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        const name = this.parseName();
        const args = this.parseArgumentDefs();
        this.expectToken(_tokenKind.TokenKind.COLON);
        const type = this.parseTypeReference();
        const directives = this.parseConstDirectives();
        return this.node(start, {
          kind: _kinds.Kind.FIELD_DEFINITION,
          description,
          name,
          arguments: args,
          type,
          directives
        });
      }
      /**
       * ArgumentsDefinition : ( InputValueDefinition+ )
       */
      parseArgumentDefs() {
        return this.optionalMany(
          _tokenKind.TokenKind.PAREN_L,
          this.parseInputValueDef,
          _tokenKind.TokenKind.PAREN_R
        );
      }
      /**
       * InputValueDefinition :
       *   - Description? Name : Type DefaultValue? Directives[Const]?
       */
      parseInputValueDef() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        const name = this.parseName();
        this.expectToken(_tokenKind.TokenKind.COLON);
        const type = this.parseTypeReference();
        let defaultValue;
        if (this.expectOptionalToken(_tokenKind.TokenKind.EQUALS)) {
          defaultValue = this.parseConstValueLiteral();
        }
        const directives = this.parseConstDirectives();
        return this.node(start, {
          kind: _kinds.Kind.INPUT_VALUE_DEFINITION,
          description,
          name,
          type,
          defaultValue,
          directives
        });
      }
      /**
       * InterfaceTypeDefinition :
       *   - Description? interface Name Directives[Const]? FieldsDefinition?
       */
      parseInterfaceTypeDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("interface");
        const name = this.parseName();
        const interfaces = this.parseImplementsInterfaces();
        const directives = this.parseConstDirectives();
        const fields = this.parseFieldsDefinition();
        return this.node(start, {
          kind: _kinds.Kind.INTERFACE_TYPE_DEFINITION,
          description,
          name,
          interfaces,
          directives,
          fields
        });
      }
      /**
       * UnionTypeDefinition :
       *   - Description? union Name Directives[Const]? UnionMemberTypes?
       */
      parseUnionTypeDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("union");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        const types = this.parseUnionMemberTypes();
        return this.node(start, {
          kind: _kinds.Kind.UNION_TYPE_DEFINITION,
          description,
          name,
          directives,
          types
        });
      }
      /**
       * UnionMemberTypes :
       *   - = `|`? NamedType
       *   - UnionMemberTypes | NamedType
       */
      parseUnionMemberTypes() {
        return this.expectOptionalToken(_tokenKind.TokenKind.EQUALS) ? this.delimitedMany(_tokenKind.TokenKind.PIPE, this.parseNamedType) : [];
      }
      /**
       * EnumTypeDefinition :
       *   - Description? enum Name Directives[Const]? EnumValuesDefinition?
       */
      parseEnumTypeDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("enum");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        const values = this.parseEnumValuesDefinition();
        return this.node(start, {
          kind: _kinds.Kind.ENUM_TYPE_DEFINITION,
          description,
          name,
          directives,
          values
        });
      }
      /**
       * ```
       * EnumValuesDefinition : { EnumValueDefinition+ }
       * ```
       */
      parseEnumValuesDefinition() {
        return this.optionalMany(
          _tokenKind.TokenKind.BRACE_L,
          this.parseEnumValueDefinition,
          _tokenKind.TokenKind.BRACE_R
        );
      }
      /**
       * EnumValueDefinition : Description? EnumValue Directives[Const]?
       */
      parseEnumValueDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        const name = this.parseEnumValueName();
        const directives = this.parseConstDirectives();
        return this.node(start, {
          kind: _kinds.Kind.ENUM_VALUE_DEFINITION,
          description,
          name,
          directives
        });
      }
      /**
       * EnumValue : Name but not `true`, `false` or `null`
       */
      parseEnumValueName() {
        if (this._lexer.token.value === "true" || this._lexer.token.value === "false" || this._lexer.token.value === "null") {
          throw (0, _syntaxError.syntaxError)(
            this._lexer.source,
            this._lexer.token.start,
            `${getTokenDesc(
              this._lexer.token
            )} is reserved and cannot be used for an enum value.`
          );
        }
        return this.parseName();
      }
      /**
       * InputObjectTypeDefinition :
       *   - Description? input Name Directives[Const]? InputFieldsDefinition?
       */
      parseInputObjectTypeDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("input");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        const fields = this.parseInputFieldsDefinition();
        return this.node(start, {
          kind: _kinds.Kind.INPUT_OBJECT_TYPE_DEFINITION,
          description,
          name,
          directives,
          fields
        });
      }
      /**
       * ```
       * InputFieldsDefinition : { InputValueDefinition+ }
       * ```
       */
      parseInputFieldsDefinition() {
        return this.optionalMany(
          _tokenKind.TokenKind.BRACE_L,
          this.parseInputValueDef,
          _tokenKind.TokenKind.BRACE_R
        );
      }
      /**
       * TypeSystemExtension :
       *   - SchemaExtension
       *   - TypeExtension
       *
       * TypeExtension :
       *   - ScalarTypeExtension
       *   - ObjectTypeExtension
       *   - InterfaceTypeExtension
       *   - UnionTypeExtension
       *   - EnumTypeExtension
       *   - InputObjectTypeDefinition
       */
      parseTypeSystemExtension() {
        const keywordToken = this._lexer.lookahead();
        if (keywordToken.kind === _tokenKind.TokenKind.NAME) {
          switch (keywordToken.value) {
            case "schema":
              return this.parseSchemaExtension();
            case "scalar":
              return this.parseScalarTypeExtension();
            case "type":
              return this.parseObjectTypeExtension();
            case "interface":
              return this.parseInterfaceTypeExtension();
            case "union":
              return this.parseUnionTypeExtension();
            case "enum":
              return this.parseEnumTypeExtension();
            case "input":
              return this.parseInputObjectTypeExtension();
          }
        }
        throw this.unexpected(keywordToken);
      }
      /**
       * ```
       * SchemaExtension :
       *  - extend schema Directives[Const]? { OperationTypeDefinition+ }
       *  - extend schema Directives[Const]
       * ```
       */
      parseSchemaExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("schema");
        const directives = this.parseConstDirectives();
        const operationTypes = this.optionalMany(
          _tokenKind.TokenKind.BRACE_L,
          this.parseOperationTypeDefinition,
          _tokenKind.TokenKind.BRACE_R
        );
        if (directives.length === 0 && operationTypes.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.SCHEMA_EXTENSION,
          directives,
          operationTypes
        });
      }
      /**
       * ScalarTypeExtension :
       *   - extend scalar Name Directives[Const]
       */
      parseScalarTypeExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("scalar");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        if (directives.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.SCALAR_TYPE_EXTENSION,
          name,
          directives
        });
      }
      /**
       * ObjectTypeExtension :
       *  - extend type Name ImplementsInterfaces? Directives[Const]? FieldsDefinition
       *  - extend type Name ImplementsInterfaces? Directives[Const]
       *  - extend type Name ImplementsInterfaces
       */
      parseObjectTypeExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("type");
        const name = this.parseName();
        const interfaces = this.parseImplementsInterfaces();
        const directives = this.parseConstDirectives();
        const fields = this.parseFieldsDefinition();
        if (interfaces.length === 0 && directives.length === 0 && fields.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.OBJECT_TYPE_EXTENSION,
          name,
          interfaces,
          directives,
          fields
        });
      }
      /**
       * InterfaceTypeExtension :
       *  - extend interface Name ImplementsInterfaces? Directives[Const]? FieldsDefinition
       *  - extend interface Name ImplementsInterfaces? Directives[Const]
       *  - extend interface Name ImplementsInterfaces
       */
      parseInterfaceTypeExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("interface");
        const name = this.parseName();
        const interfaces = this.parseImplementsInterfaces();
        const directives = this.parseConstDirectives();
        const fields = this.parseFieldsDefinition();
        if (interfaces.length === 0 && directives.length === 0 && fields.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.INTERFACE_TYPE_EXTENSION,
          name,
          interfaces,
          directives,
          fields
        });
      }
      /**
       * UnionTypeExtension :
       *   - extend union Name Directives[Const]? UnionMemberTypes
       *   - extend union Name Directives[Const]
       */
      parseUnionTypeExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("union");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        const types = this.parseUnionMemberTypes();
        if (directives.length === 0 && types.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.UNION_TYPE_EXTENSION,
          name,
          directives,
          types
        });
      }
      /**
       * EnumTypeExtension :
       *   - extend enum Name Directives[Const]? EnumValuesDefinition
       *   - extend enum Name Directives[Const]
       */
      parseEnumTypeExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("enum");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        const values = this.parseEnumValuesDefinition();
        if (directives.length === 0 && values.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.ENUM_TYPE_EXTENSION,
          name,
          directives,
          values
        });
      }
      /**
       * InputObjectTypeExtension :
       *   - extend input Name Directives[Const]? InputFieldsDefinition
       *   - extend input Name Directives[Const]
       */
      parseInputObjectTypeExtension() {
        const start = this._lexer.token;
        this.expectKeyword("extend");
        this.expectKeyword("input");
        const name = this.parseName();
        const directives = this.parseConstDirectives();
        const fields = this.parseInputFieldsDefinition();
        if (directives.length === 0 && fields.length === 0) {
          throw this.unexpected();
        }
        return this.node(start, {
          kind: _kinds.Kind.INPUT_OBJECT_TYPE_EXTENSION,
          name,
          directives,
          fields
        });
      }
      /**
       * ```
       * DirectiveDefinition :
       *   - Description? directive @ Name ArgumentsDefinition? `repeatable`? on DirectiveLocations
       * ```
       */
      parseDirectiveDefinition() {
        const start = this._lexer.token;
        const description = this.parseDescription();
        this.expectKeyword("directive");
        this.expectToken(_tokenKind.TokenKind.AT);
        const name = this.parseName();
        const args = this.parseArgumentDefs();
        const repeatable = this.expectOptionalKeyword("repeatable");
        this.expectKeyword("on");
        const locations = this.parseDirectiveLocations();
        return this.node(start, {
          kind: _kinds.Kind.DIRECTIVE_DEFINITION,
          description,
          name,
          arguments: args,
          repeatable,
          locations
        });
      }
      /**
       * DirectiveLocations :
       *   - `|`? DirectiveLocation
       *   - DirectiveLocations | DirectiveLocation
       */
      parseDirectiveLocations() {
        return this.delimitedMany(
          _tokenKind.TokenKind.PIPE,
          this.parseDirectiveLocation
        );
      }
      /*
       * DirectiveLocation :
       *   - ExecutableDirectiveLocation
       *   - TypeSystemDirectiveLocation
       *
       * ExecutableDirectiveLocation : one of
       *   `QUERY`
       *   `MUTATION`
       *   `SUBSCRIPTION`
       *   `FIELD`
       *   `FRAGMENT_DEFINITION`
       *   `FRAGMENT_SPREAD`
       *   `INLINE_FRAGMENT`
       *
       * TypeSystemDirectiveLocation : one of
       *   `SCHEMA`
       *   `SCALAR`
       *   `OBJECT`
       *   `FIELD_DEFINITION`
       *   `ARGUMENT_DEFINITION`
       *   `INTERFACE`
       *   `UNION`
       *   `ENUM`
       *   `ENUM_VALUE`
       *   `INPUT_OBJECT`
       *   `INPUT_FIELD_DEFINITION`
       */
      parseDirectiveLocation() {
        const start = this._lexer.token;
        const name = this.parseName();
        if (Object.prototype.hasOwnProperty.call(
          _directiveLocation.DirectiveLocation,
          name.value
        )) {
          return name;
        }
        throw this.unexpected(start);
      }
      // Core parsing utility functions
      /**
       * Returns a node that, if configured to do so, sets a "loc" field as a
       * location object, used to identify the place in the source that created a
       * given parsed object.
       */
      node(startToken, node) {
        if (this._options.noLocation !== true) {
          node.loc = new _ast.Location(
            startToken,
            this._lexer.lastToken,
            this._lexer.source
          );
        }
        return node;
      }
      /**
       * Determines if the next token is of a given kind
       */
      peek(kind) {
        return this._lexer.token.kind === kind;
      }
      /**
       * If the next token is of the given kind, return that token after advancing the lexer.
       * Otherwise, do not change the parser state and throw an error.
       */
      expectToken(kind) {
        const token = this._lexer.token;
        if (token.kind === kind) {
          this.advanceLexer();
          return token;
        }
        throw (0, _syntaxError.syntaxError)(
          this._lexer.source,
          token.start,
          `Expected ${getTokenKindDesc(kind)}, found ${getTokenDesc(token)}.`
        );
      }
      /**
       * If the next token is of the given kind, return "true" after advancing the lexer.
       * Otherwise, do not change the parser state and return "false".
       */
      expectOptionalToken(kind) {
        const token = this._lexer.token;
        if (token.kind === kind) {
          this.advanceLexer();
          return true;
        }
        return false;
      }
      /**
       * If the next token is a given keyword, advance the lexer.
       * Otherwise, do not change the parser state and throw an error.
       */
      expectKeyword(value) {
        const token = this._lexer.token;
        if (token.kind === _tokenKind.TokenKind.NAME && token.value === value) {
          this.advanceLexer();
        } else {
          throw (0, _syntaxError.syntaxError)(
            this._lexer.source,
            token.start,
            `Expected "${value}", found ${getTokenDesc(token)}.`
          );
        }
      }
      /**
       * If the next token is a given keyword, return "true" after advancing the lexer.
       * Otherwise, do not change the parser state and return "false".
       */
      expectOptionalKeyword(value) {
        const token = this._lexer.token;
        if (token.kind === _tokenKind.TokenKind.NAME && token.value === value) {
          this.advanceLexer();
          return true;
        }
        return false;
      }
      /**
       * Helper function for creating an error when an unexpected lexed token is encountered.
       */
      unexpected(atToken) {
        const token = atToken !== null && atToken !== void 0 ? atToken : this._lexer.token;
        return (0, _syntaxError.syntaxError)(
          this._lexer.source,
          token.start,
          `Unexpected ${getTokenDesc(token)}.`
        );
      }
      /**
       * Returns a possibly empty list of parse nodes, determined by the parseFn.
       * This list begins with a lex token of openKind and ends with a lex token of closeKind.
       * Advances the parser to the next lex token after the closing token.
       */
      any(openKind, parseFn, closeKind) {
        this.expectToken(openKind);
        const nodes = [];
        while (!this.expectOptionalToken(closeKind)) {
          nodes.push(parseFn.call(this));
        }
        return nodes;
      }
      /**
       * Returns a list of parse nodes, determined by the parseFn.
       * It can be empty only if open token is missing otherwise it will always return non-empty list
       * that begins with a lex token of openKind and ends with a lex token of closeKind.
       * Advances the parser to the next lex token after the closing token.
       */
      optionalMany(openKind, parseFn, closeKind) {
        if (this.expectOptionalToken(openKind)) {
          const nodes = [];
          do {
            nodes.push(parseFn.call(this));
          } while (!this.expectOptionalToken(closeKind));
          return nodes;
        }
        return [];
      }
      /**
       * Returns a non-empty list of parse nodes, determined by the parseFn.
       * This list begins with a lex token of openKind and ends with a lex token of closeKind.
       * Advances the parser to the next lex token after the closing token.
       */
      many(openKind, parseFn, closeKind) {
        this.expectToken(openKind);
        const nodes = [];
        do {
          nodes.push(parseFn.call(this));
        } while (!this.expectOptionalToken(closeKind));
        return nodes;
      }
      /**
       * Returns a non-empty list of parse nodes, determined by the parseFn.
       * This list may begin with a lex token of delimiterKind followed by items separated by lex tokens of tokenKind.
       * Advances the parser to the next lex token after last item in the list.
       */
      delimitedMany(delimiterKind, parseFn) {
        this.expectOptionalToken(delimiterKind);
        const nodes = [];
        do {
          nodes.push(parseFn.call(this));
        } while (this.expectOptionalToken(delimiterKind));
        return nodes;
      }
      advanceLexer() {
        const { maxTokens } = this._options;
        const token = this._lexer.advance();
        if (token.kind !== _tokenKind.TokenKind.EOF) {
          ++this._tokenCounter;
          if (maxTokens !== void 0 && this._tokenCounter > maxTokens) {
            throw (0, _syntaxError.syntaxError)(
              this._lexer.source,
              token.start,
              `Document contains more that ${maxTokens} tokens. Parsing aborted.`
            );
          }
        }
      }
    };
    exports.Parser = Parser;
    function getTokenDesc(token) {
      const value = token.value;
      return getTokenKindDesc(token.kind) + (value != null ? ` "${value}"` : "");
    }
    __name(getTokenDesc, "getTokenDesc");
    function getTokenKindDesc(kind) {
      return (0, _lexer.isPunctuatorTokenKind)(kind) ? `"${kind}"` : kind;
    }
    __name(getTokenKindDesc, "getTokenKindDesc");
  }
});

// ../../node_modules/graphql/jsutils/didYouMean.js
var require_didYouMean = __commonJS({
  "../../node_modules/graphql/jsutils/didYouMean.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.didYouMean = didYouMean;
    var MAX_SUGGESTIONS = 5;
    function didYouMean(firstArg, secondArg) {
      const [subMessage, suggestionsArg] = secondArg ? [firstArg, secondArg] : [void 0, firstArg];
      let message = " Did you mean ";
      if (subMessage) {
        message += subMessage + " ";
      }
      const suggestions = suggestionsArg.map((x) => `"${x}"`);
      switch (suggestions.length) {
        case 0:
          return "";
        case 1:
          return message + suggestions[0] + "?";
        case 2:
          return message + suggestions[0] + " or " + suggestions[1] + "?";
      }
      const selected = suggestions.slice(0, MAX_SUGGESTIONS);
      const lastItem = selected.pop();
      return message + selected.join(", ") + ", or " + lastItem + "?";
    }
    __name(didYouMean, "didYouMean");
  }
});

// ../../node_modules/graphql/jsutils/identityFunc.js
var require_identityFunc = __commonJS({
  "../../node_modules/graphql/jsutils/identityFunc.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.identityFunc = identityFunc;
    function identityFunc(x) {
      return x;
    }
    __name(identityFunc, "identityFunc");
  }
});

// ../../node_modules/graphql/jsutils/keyMap.js
var require_keyMap = __commonJS({
  "../../node_modules/graphql/jsutils/keyMap.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.keyMap = keyMap;
    function keyMap(list, keyFn) {
      const result = /* @__PURE__ */ Object.create(null);
      for (const item of list) {
        result[keyFn(item)] = item;
      }
      return result;
    }
    __name(keyMap, "keyMap");
  }
});

// ../../node_modules/graphql/jsutils/keyValMap.js
var require_keyValMap = __commonJS({
  "../../node_modules/graphql/jsutils/keyValMap.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.keyValMap = keyValMap;
    function keyValMap(list, keyFn, valFn) {
      const result = /* @__PURE__ */ Object.create(null);
      for (const item of list) {
        result[keyFn(item)] = valFn(item);
      }
      return result;
    }
    __name(keyValMap, "keyValMap");
  }
});

// ../../node_modules/graphql/jsutils/mapValue.js
var require_mapValue = __commonJS({
  "../../node_modules/graphql/jsutils/mapValue.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.mapValue = mapValue;
    function mapValue(map, fn) {
      const result = /* @__PURE__ */ Object.create(null);
      for (const key of Object.keys(map)) {
        result[key] = fn(map[key], key);
      }
      return result;
    }
    __name(mapValue, "mapValue");
  }
});

// ../../node_modules/graphql/jsutils/naturalCompare.js
var require_naturalCompare = __commonJS({
  "../../node_modules/graphql/jsutils/naturalCompare.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.naturalCompare = naturalCompare;
    function naturalCompare(aStr, bStr) {
      let aIndex = 0;
      let bIndex = 0;
      while (aIndex < aStr.length && bIndex < bStr.length) {
        let aChar = aStr.charCodeAt(aIndex);
        let bChar = bStr.charCodeAt(bIndex);
        if (isDigit(aChar) && isDigit(bChar)) {
          let aNum = 0;
          do {
            ++aIndex;
            aNum = aNum * 10 + aChar - DIGIT_0;
            aChar = aStr.charCodeAt(aIndex);
          } while (isDigit(aChar) && aNum > 0);
          let bNum = 0;
          do {
            ++bIndex;
            bNum = bNum * 10 + bChar - DIGIT_0;
            bChar = bStr.charCodeAt(bIndex);
          } while (isDigit(bChar) && bNum > 0);
          if (aNum < bNum) {
            return -1;
          }
          if (aNum > bNum) {
            return 1;
          }
        } else {
          if (aChar < bChar) {
            return -1;
          }
          if (aChar > bChar) {
            return 1;
          }
          ++aIndex;
          ++bIndex;
        }
      }
      return aStr.length - bStr.length;
    }
    __name(naturalCompare, "naturalCompare");
    var DIGIT_0 = 48;
    var DIGIT_9 = 57;
    function isDigit(code) {
      return !isNaN(code) && DIGIT_0 <= code && code <= DIGIT_9;
    }
    __name(isDigit, "isDigit");
  }
});

// ../../node_modules/graphql/jsutils/suggestionList.js
var require_suggestionList = __commonJS({
  "../../node_modules/graphql/jsutils/suggestionList.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.suggestionList = suggestionList;
    var _naturalCompare = require_naturalCompare();
    function suggestionList(input, options) {
      const optionsByDistance = /* @__PURE__ */ Object.create(null);
      const lexicalDistance = new LexicalDistance(input);
      const threshold = Math.floor(input.length * 0.4) + 1;
      for (const option of options) {
        const distance = lexicalDistance.measure(option, threshold);
        if (distance !== void 0) {
          optionsByDistance[option] = distance;
        }
      }
      return Object.keys(optionsByDistance).sort((a, b) => {
        const distanceDiff = optionsByDistance[a] - optionsByDistance[b];
        return distanceDiff !== 0 ? distanceDiff : (0, _naturalCompare.naturalCompare)(a, b);
      });
    }
    __name(suggestionList, "suggestionList");
    var LexicalDistance = class {
      static {
        __name(this, "LexicalDistance");
      }
      constructor(input) {
        this._input = input;
        this._inputLowerCase = input.toLowerCase();
        this._inputArray = stringToArray(this._inputLowerCase);
        this._rows = [
          new Array(input.length + 1).fill(0),
          new Array(input.length + 1).fill(0),
          new Array(input.length + 1).fill(0)
        ];
      }
      measure(option, threshold) {
        if (this._input === option) {
          return 0;
        }
        const optionLowerCase = option.toLowerCase();
        if (this._inputLowerCase === optionLowerCase) {
          return 1;
        }
        let a = stringToArray(optionLowerCase);
        let b = this._inputArray;
        if (a.length < b.length) {
          const tmp = a;
          a = b;
          b = tmp;
        }
        const aLength = a.length;
        const bLength = b.length;
        if (aLength - bLength > threshold) {
          return void 0;
        }
        const rows = this._rows;
        for (let j = 0; j <= bLength; j++) {
          rows[0][j] = j;
        }
        for (let i = 1; i <= aLength; i++) {
          const upRow = rows[(i - 1) % 3];
          const currentRow = rows[i % 3];
          let smallestCell = currentRow[0] = i;
          for (let j = 1; j <= bLength; j++) {
            const cost = a[i - 1] === b[j - 1] ? 0 : 1;
            let currentCell = Math.min(
              upRow[j] + 1,
              // delete
              currentRow[j - 1] + 1,
              // insert
              upRow[j - 1] + cost
              // substitute
            );
            if (i > 1 && j > 1 && a[i - 1] === b[j - 2] && a[i - 2] === b[j - 1]) {
              const doubleDiagonalCell = rows[(i - 2) % 3][j - 2];
              currentCell = Math.min(currentCell, doubleDiagonalCell + 1);
            }
            if (currentCell < smallestCell) {
              smallestCell = currentCell;
            }
            currentRow[j] = currentCell;
          }
          if (smallestCell > threshold) {
            return void 0;
          }
        }
        const distance = rows[aLength % 3][bLength];
        return distance <= threshold ? distance : void 0;
      }
    };
    function stringToArray(str) {
      const strLength = str.length;
      const array = new Array(strLength);
      for (let i = 0; i < strLength; ++i) {
        array[i] = str.charCodeAt(i);
      }
      return array;
    }
    __name(stringToArray, "stringToArray");
  }
});

// ../../node_modules/graphql/jsutils/toObjMap.js
var require_toObjMap = __commonJS({
  "../../node_modules/graphql/jsutils/toObjMap.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.toObjMap = toObjMap;
    function toObjMap(obj) {
      if (obj == null) {
        return /* @__PURE__ */ Object.create(null);
      }
      if (Object.getPrototypeOf(obj) === null) {
        return obj;
      }
      const map = /* @__PURE__ */ Object.create(null);
      for (const [key, value] of Object.entries(obj)) {
        map[key] = value;
      }
      return map;
    }
    __name(toObjMap, "toObjMap");
  }
});

// ../../node_modules/graphql/language/printString.js
var require_printString = __commonJS({
  "../../node_modules/graphql/language/printString.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.printString = printString;
    function printString(str) {
      return `"${str.replace(escapedRegExp, escapedReplacer)}"`;
    }
    __name(printString, "printString");
    var escapedRegExp = /[\x00-\x1f\x22\x5c\x7f-\x9f]/g;
    function escapedReplacer(str) {
      return escapeSequences[str.charCodeAt(0)];
    }
    __name(escapedReplacer, "escapedReplacer");
    var escapeSequences = [
      "\\u0000",
      "\\u0001",
      "\\u0002",
      "\\u0003",
      "\\u0004",
      "\\u0005",
      "\\u0006",
      "\\u0007",
      "\\b",
      "\\t",
      "\\n",
      "\\u000B",
      "\\f",
      "\\r",
      "\\u000E",
      "\\u000F",
      "\\u0010",
      "\\u0011",
      "\\u0012",
      "\\u0013",
      "\\u0014",
      "\\u0015",
      "\\u0016",
      "\\u0017",
      "\\u0018",
      "\\u0019",
      "\\u001A",
      "\\u001B",
      "\\u001C",
      "\\u001D",
      "\\u001E",
      "\\u001F",
      "",
      "",
      '\\"',
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      // 2F
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      // 3F
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      // 4F
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "\\\\",
      "",
      "",
      "",
      // 5F
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      // 6F
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "",
      "\\u007F",
      "\\u0080",
      "\\u0081",
      "\\u0082",
      "\\u0083",
      "\\u0084",
      "\\u0085",
      "\\u0086",
      "\\u0087",
      "\\u0088",
      "\\u0089",
      "\\u008A",
      "\\u008B",
      "\\u008C",
      "\\u008D",
      "\\u008E",
      "\\u008F",
      "\\u0090",
      "\\u0091",
      "\\u0092",
      "\\u0093",
      "\\u0094",
      "\\u0095",
      "\\u0096",
      "\\u0097",
      "\\u0098",
      "\\u0099",
      "\\u009A",
      "\\u009B",
      "\\u009C",
      "\\u009D",
      "\\u009E",
      "\\u009F"
    ];
  }
});

// ../../node_modules/graphql/language/visitor.js
var require_visitor = __commonJS({
  "../../node_modules/graphql/language/visitor.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.BREAK = void 0;
    exports.getEnterLeaveForKind = getEnterLeaveForKind;
    exports.getVisitFn = getVisitFn;
    exports.visit = visit;
    exports.visitInParallel = visitInParallel;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _ast = require_ast();
    var _kinds = require_kinds();
    var BREAK = Object.freeze({});
    exports.BREAK = BREAK;
    function visit(root, visitor, visitorKeys = _ast.QueryDocumentKeys) {
      const enterLeaveMap = /* @__PURE__ */ new Map();
      for (const kind of Object.values(_kinds.Kind)) {
        enterLeaveMap.set(kind, getEnterLeaveForKind(visitor, kind));
      }
      let stack = void 0;
      let inArray = Array.isArray(root);
      let keys = [root];
      let index = -1;
      let edits = [];
      let node = root;
      let key = void 0;
      let parent = void 0;
      const path2 = [];
      const ancestors = [];
      do {
        index++;
        const isLeaving = index === keys.length;
        const isEdited = isLeaving && edits.length !== 0;
        if (isLeaving) {
          key = ancestors.length === 0 ? void 0 : path2[path2.length - 1];
          node = parent;
          parent = ancestors.pop();
          if (isEdited) {
            if (inArray) {
              node = node.slice();
              let editOffset = 0;
              for (const [editKey, editValue] of edits) {
                const arrayKey = editKey - editOffset;
                if (editValue === null) {
                  node.splice(arrayKey, 1);
                  editOffset++;
                } else {
                  node[arrayKey] = editValue;
                }
              }
            } else {
              node = Object.defineProperties(
                {},
                Object.getOwnPropertyDescriptors(node)
              );
              for (const [editKey, editValue] of edits) {
                node[editKey] = editValue;
              }
            }
          }
          index = stack.index;
          keys = stack.keys;
          edits = stack.edits;
          inArray = stack.inArray;
          stack = stack.prev;
        } else if (parent) {
          key = inArray ? index : keys[index];
          node = parent[key];
          if (node === null || node === void 0) {
            continue;
          }
          path2.push(key);
        }
        let result;
        if (!Array.isArray(node)) {
          var _enterLeaveMap$get, _enterLeaveMap$get2;
          (0, _ast.isNode)(node) || (0, _devAssert.devAssert)(
            false,
            `Invalid AST Node: ${(0, _inspect.inspect)(node)}.`
          );
          const visitFn = isLeaving ? (_enterLeaveMap$get = enterLeaveMap.get(node.kind)) === null || _enterLeaveMap$get === void 0 ? void 0 : _enterLeaveMap$get.leave : (_enterLeaveMap$get2 = enterLeaveMap.get(node.kind)) === null || _enterLeaveMap$get2 === void 0 ? void 0 : _enterLeaveMap$get2.enter;
          result = visitFn === null || visitFn === void 0 ? void 0 : visitFn.call(visitor, node, key, parent, path2, ancestors);
          if (result === BREAK) {
            break;
          }
          if (result === false) {
            if (!isLeaving) {
              path2.pop();
              continue;
            }
          } else if (result !== void 0) {
            edits.push([key, result]);
            if (!isLeaving) {
              if ((0, _ast.isNode)(result)) {
                node = result;
              } else {
                path2.pop();
                continue;
              }
            }
          }
        }
        if (result === void 0 && isEdited) {
          edits.push([key, node]);
        }
        if (isLeaving) {
          path2.pop();
        } else {
          var _node$kind;
          stack = {
            inArray,
            index,
            keys,
            edits,
            prev: stack
          };
          inArray = Array.isArray(node);
          keys = inArray ? node : (_node$kind = visitorKeys[node.kind]) !== null && _node$kind !== void 0 ? _node$kind : [];
          index = -1;
          edits = [];
          if (parent) {
            ancestors.push(parent);
          }
          parent = node;
        }
      } while (stack !== void 0);
      if (edits.length !== 0) {
        return edits[edits.length - 1][1];
      }
      return root;
    }
    __name(visit, "visit");
    function visitInParallel(visitors) {
      const skipping = new Array(visitors.length).fill(null);
      const mergedVisitor = /* @__PURE__ */ Object.create(null);
      for (const kind of Object.values(_kinds.Kind)) {
        let hasVisitor = false;
        const enterList = new Array(visitors.length).fill(void 0);
        const leaveList = new Array(visitors.length).fill(void 0);
        for (let i = 0; i < visitors.length; ++i) {
          const { enter, leave } = getEnterLeaveForKind(visitors[i], kind);
          hasVisitor || (hasVisitor = enter != null || leave != null);
          enterList[i] = enter;
          leaveList[i] = leave;
        }
        if (!hasVisitor) {
          continue;
        }
        const mergedEnterLeave = {
          enter(...args) {
            const node = args[0];
            for (let i = 0; i < visitors.length; i++) {
              if (skipping[i] === null) {
                var _enterList$i;
                const result = (_enterList$i = enterList[i]) === null || _enterList$i === void 0 ? void 0 : _enterList$i.apply(visitors[i], args);
                if (result === false) {
                  skipping[i] = node;
                } else if (result === BREAK) {
                  skipping[i] = BREAK;
                } else if (result !== void 0) {
                  return result;
                }
              }
            }
          },
          leave(...args) {
            const node = args[0];
            for (let i = 0; i < visitors.length; i++) {
              if (skipping[i] === null) {
                var _leaveList$i;
                const result = (_leaveList$i = leaveList[i]) === null || _leaveList$i === void 0 ? void 0 : _leaveList$i.apply(visitors[i], args);
                if (result === BREAK) {
                  skipping[i] = BREAK;
                } else if (result !== void 0 && result !== false) {
                  return result;
                }
              } else if (skipping[i] === node) {
                skipping[i] = null;
              }
            }
          }
        };
        mergedVisitor[kind] = mergedEnterLeave;
      }
      return mergedVisitor;
    }
    __name(visitInParallel, "visitInParallel");
    function getEnterLeaveForKind(visitor, kind) {
      const kindVisitor = visitor[kind];
      if (typeof kindVisitor === "object") {
        return kindVisitor;
      } else if (typeof kindVisitor === "function") {
        return {
          enter: kindVisitor,
          leave: void 0
        };
      }
      return {
        enter: visitor.enter,
        leave: visitor.leave
      };
    }
    __name(getEnterLeaveForKind, "getEnterLeaveForKind");
    function getVisitFn(visitor, kind, isLeaving) {
      const { enter, leave } = getEnterLeaveForKind(visitor, kind);
      return isLeaving ? leave : enter;
    }
    __name(getVisitFn, "getVisitFn");
  }
});

// ../../node_modules/graphql/language/printer.js
var require_printer = __commonJS({
  "../../node_modules/graphql/language/printer.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.print = print2;
    var _blockString = require_blockString();
    var _printString = require_printString();
    var _visitor = require_visitor();
    function print2(ast) {
      return (0, _visitor.visit)(ast, printDocASTReducer);
    }
    __name(print2, "print");
    var MAX_LINE_LENGTH = 80;
    var printDocASTReducer = {
      Name: {
        leave: /* @__PURE__ */ __name((node) => node.value, "leave")
      },
      Variable: {
        leave: /* @__PURE__ */ __name((node) => "$" + node.name, "leave")
      },
      // Document
      Document: {
        leave: /* @__PURE__ */ __name((node) => join(node.definitions, "\n\n"), "leave")
      },
      OperationDefinition: {
        leave(node) {
          const varDefs = wrap("(", join(node.variableDefinitions, ", "), ")");
          const prefix = join(
            [
              node.operation,
              join([node.name, varDefs]),
              join(node.directives, " ")
            ],
            " "
          );
          return (prefix === "query" ? "" : prefix + " ") + node.selectionSet;
        }
      },
      VariableDefinition: {
        leave: /* @__PURE__ */ __name(({ variable, type, defaultValue, directives }) => variable + ": " + type + wrap(" = ", defaultValue) + wrap(" ", join(directives, " ")), "leave")
      },
      SelectionSet: {
        leave: /* @__PURE__ */ __name(({ selections }) => block(selections), "leave")
      },
      Field: {
        leave({ alias, name, arguments: args, directives, selectionSet }) {
          const prefix = wrap("", alias, ": ") + name;
          let argsLine = prefix + wrap("(", join(args, ", "), ")");
          if (argsLine.length > MAX_LINE_LENGTH) {
            argsLine = prefix + wrap("(\n", indent(join(args, "\n")), "\n)");
          }
          return join([argsLine, join(directives, " "), selectionSet], " ");
        }
      },
      Argument: {
        leave: /* @__PURE__ */ __name(({ name, value }) => name + ": " + value, "leave")
      },
      // Fragments
      FragmentSpread: {
        leave: /* @__PURE__ */ __name(({ name, directives }) => "..." + name + wrap(" ", join(directives, " ")), "leave")
      },
      InlineFragment: {
        leave: /* @__PURE__ */ __name(({ typeCondition, directives, selectionSet }) => join(
          [
            "...",
            wrap("on ", typeCondition),
            join(directives, " "),
            selectionSet
          ],
          " "
        ), "leave")
      },
      FragmentDefinition: {
        leave: /* @__PURE__ */ __name(({ name, typeCondition, variableDefinitions, directives, selectionSet }) => (
          // or removed in the future.
          `fragment ${name}${wrap("(", join(variableDefinitions, ", "), ")")} on ${typeCondition} ${wrap("", join(directives, " "), " ")}` + selectionSet
        ), "leave")
      },
      // Value
      IntValue: {
        leave: /* @__PURE__ */ __name(({ value }) => value, "leave")
      },
      FloatValue: {
        leave: /* @__PURE__ */ __name(({ value }) => value, "leave")
      },
      StringValue: {
        leave: /* @__PURE__ */ __name(({ value, block: isBlockString }) => isBlockString ? (0, _blockString.printBlockString)(value) : (0, _printString.printString)(value), "leave")
      },
      BooleanValue: {
        leave: /* @__PURE__ */ __name(({ value }) => value ? "true" : "false", "leave")
      },
      NullValue: {
        leave: /* @__PURE__ */ __name(() => "null", "leave")
      },
      EnumValue: {
        leave: /* @__PURE__ */ __name(({ value }) => value, "leave")
      },
      ListValue: {
        leave: /* @__PURE__ */ __name(({ values }) => "[" + join(values, ", ") + "]", "leave")
      },
      ObjectValue: {
        leave: /* @__PURE__ */ __name(({ fields }) => "{" + join(fields, ", ") + "}", "leave")
      },
      ObjectField: {
        leave: /* @__PURE__ */ __name(({ name, value }) => name + ": " + value, "leave")
      },
      // Directive
      Directive: {
        leave: /* @__PURE__ */ __name(({ name, arguments: args }) => "@" + name + wrap("(", join(args, ", "), ")"), "leave")
      },
      // Type
      NamedType: {
        leave: /* @__PURE__ */ __name(({ name }) => name, "leave")
      },
      ListType: {
        leave: /* @__PURE__ */ __name(({ type }) => "[" + type + "]", "leave")
      },
      NonNullType: {
        leave: /* @__PURE__ */ __name(({ type }) => type + "!", "leave")
      },
      // Type System Definitions
      SchemaDefinition: {
        leave: /* @__PURE__ */ __name(({ description, directives, operationTypes }) => wrap("", description, "\n") + join(["schema", join(directives, " "), block(operationTypes)], " "), "leave")
      },
      OperationTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ operation, type }) => operation + ": " + type, "leave")
      },
      ScalarTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, directives }) => wrap("", description, "\n") + join(["scalar", name, join(directives, " ")], " "), "leave")
      },
      ObjectTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, interfaces, directives, fields }) => wrap("", description, "\n") + join(
          [
            "type",
            name,
            wrap("implements ", join(interfaces, " & ")),
            join(directives, " "),
            block(fields)
          ],
          " "
        ), "leave")
      },
      FieldDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, arguments: args, type, directives }) => wrap("", description, "\n") + name + (hasMultilineItems(args) ? wrap("(\n", indent(join(args, "\n")), "\n)") : wrap("(", join(args, ", "), ")")) + ": " + type + wrap(" ", join(directives, " ")), "leave")
      },
      InputValueDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, type, defaultValue, directives }) => wrap("", description, "\n") + join(
          [name + ": " + type, wrap("= ", defaultValue), join(directives, " ")],
          " "
        ), "leave")
      },
      InterfaceTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, interfaces, directives, fields }) => wrap("", description, "\n") + join(
          [
            "interface",
            name,
            wrap("implements ", join(interfaces, " & ")),
            join(directives, " "),
            block(fields)
          ],
          " "
        ), "leave")
      },
      UnionTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, directives, types }) => wrap("", description, "\n") + join(
          ["union", name, join(directives, " "), wrap("= ", join(types, " | "))],
          " "
        ), "leave")
      },
      EnumTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, directives, values }) => wrap("", description, "\n") + join(["enum", name, join(directives, " "), block(values)], " "), "leave")
      },
      EnumValueDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, directives }) => wrap("", description, "\n") + join([name, join(directives, " ")], " "), "leave")
      },
      InputObjectTypeDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, directives, fields }) => wrap("", description, "\n") + join(["input", name, join(directives, " "), block(fields)], " "), "leave")
      },
      DirectiveDefinition: {
        leave: /* @__PURE__ */ __name(({ description, name, arguments: args, repeatable, locations }) => wrap("", description, "\n") + "directive @" + name + (hasMultilineItems(args) ? wrap("(\n", indent(join(args, "\n")), "\n)") : wrap("(", join(args, ", "), ")")) + (repeatable ? " repeatable" : "") + " on " + join(locations, " | "), "leave")
      },
      SchemaExtension: {
        leave: /* @__PURE__ */ __name(({ directives, operationTypes }) => join(
          ["extend schema", join(directives, " "), block(operationTypes)],
          " "
        ), "leave")
      },
      ScalarTypeExtension: {
        leave: /* @__PURE__ */ __name(({ name, directives }) => join(["extend scalar", name, join(directives, " ")], " "), "leave")
      },
      ObjectTypeExtension: {
        leave: /* @__PURE__ */ __name(({ name, interfaces, directives, fields }) => join(
          [
            "extend type",
            name,
            wrap("implements ", join(interfaces, " & ")),
            join(directives, " "),
            block(fields)
          ],
          " "
        ), "leave")
      },
      InterfaceTypeExtension: {
        leave: /* @__PURE__ */ __name(({ name, interfaces, directives, fields }) => join(
          [
            "extend interface",
            name,
            wrap("implements ", join(interfaces, " & ")),
            join(directives, " "),
            block(fields)
          ],
          " "
        ), "leave")
      },
      UnionTypeExtension: {
        leave: /* @__PURE__ */ __name(({ name, directives, types }) => join(
          [
            "extend union",
            name,
            join(directives, " "),
            wrap("= ", join(types, " | "))
          ],
          " "
        ), "leave")
      },
      EnumTypeExtension: {
        leave: /* @__PURE__ */ __name(({ name, directives, values }) => join(["extend enum", name, join(directives, " "), block(values)], " "), "leave")
      },
      InputObjectTypeExtension: {
        leave: /* @__PURE__ */ __name(({ name, directives, fields }) => join(["extend input", name, join(directives, " "), block(fields)], " "), "leave")
      }
    };
    function join(maybeArray, separator = "") {
      var _maybeArray$filter$jo;
      return (_maybeArray$filter$jo = maybeArray === null || maybeArray === void 0 ? void 0 : maybeArray.filter((x) => x).join(separator)) !== null && _maybeArray$filter$jo !== void 0 ? _maybeArray$filter$jo : "";
    }
    __name(join, "join");
    function block(array) {
      return wrap("{\n", indent(join(array, "\n")), "\n}");
    }
    __name(block, "block");
    function wrap(start, maybeString, end = "") {
      return maybeString != null && maybeString !== "" ? start + maybeString + end : "";
    }
    __name(wrap, "wrap");
    function indent(str) {
      return wrap("  ", str.replace(/\n/g, "\n  "));
    }
    __name(indent, "indent");
    function hasMultilineItems(maybeArray) {
      var _maybeArray$some;
      return (_maybeArray$some = maybeArray === null || maybeArray === void 0 ? void 0 : maybeArray.some((str) => str.includes("\n"))) !== null && _maybeArray$some !== void 0 ? _maybeArray$some : false;
    }
    __name(hasMultilineItems, "hasMultilineItems");
  }
});

// ../../node_modules/graphql/utilities/valueFromASTUntyped.js
var require_valueFromASTUntyped = __commonJS({
  "../../node_modules/graphql/utilities/valueFromASTUntyped.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.valueFromASTUntyped = valueFromASTUntyped;
    var _keyValMap = require_keyValMap();
    var _kinds = require_kinds();
    function valueFromASTUntyped(valueNode, variables) {
      switch (valueNode.kind) {
        case _kinds.Kind.NULL:
          return null;
        case _kinds.Kind.INT:
          return parseInt(valueNode.value, 10);
        case _kinds.Kind.FLOAT:
          return parseFloat(valueNode.value);
        case _kinds.Kind.STRING:
        case _kinds.Kind.ENUM:
        case _kinds.Kind.BOOLEAN:
          return valueNode.value;
        case _kinds.Kind.LIST:
          return valueNode.values.map(
            (node) => valueFromASTUntyped(node, variables)
          );
        case _kinds.Kind.OBJECT:
          return (0, _keyValMap.keyValMap)(
            valueNode.fields,
            (field) => field.name.value,
            (field) => valueFromASTUntyped(field.value, variables)
          );
        case _kinds.Kind.VARIABLE:
          return variables === null || variables === void 0 ? void 0 : variables[valueNode.name.value];
      }
    }
    __name(valueFromASTUntyped, "valueFromASTUntyped");
  }
});

// ../../node_modules/graphql/type/assertName.js
var require_assertName = __commonJS({
  "../../node_modules/graphql/type/assertName.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.assertEnumValueName = assertEnumValueName;
    exports.assertName = assertName;
    var _devAssert = require_devAssert();
    var _GraphQLError = require_GraphQLError();
    var _characterClasses = require_characterClasses();
    function assertName(name) {
      name != null || (0, _devAssert.devAssert)(false, "Must provide name.");
      typeof name === "string" || (0, _devAssert.devAssert)(false, "Expected name to be a string.");
      if (name.length === 0) {
        throw new _GraphQLError.GraphQLError(
          "Expected name to be a non-empty string."
        );
      }
      for (let i = 1; i < name.length; ++i) {
        if (!(0, _characterClasses.isNameContinue)(name.charCodeAt(i))) {
          throw new _GraphQLError.GraphQLError(
            `Names must only contain [_a-zA-Z0-9] but "${name}" does not.`
          );
        }
      }
      if (!(0, _characterClasses.isNameStart)(name.charCodeAt(0))) {
        throw new _GraphQLError.GraphQLError(
          `Names must start with [_a-zA-Z] but "${name}" does not.`
        );
      }
      return name;
    }
    __name(assertName, "assertName");
    function assertEnumValueName(name) {
      if (name === "true" || name === "false" || name === "null") {
        throw new _GraphQLError.GraphQLError(
          `Enum values cannot be named: ${name}`
        );
      }
      return assertName(name);
    }
    __name(assertEnumValueName, "assertEnumValueName");
  }
});

// ../../node_modules/graphql/type/definition.js
var require_definition = __commonJS({
  "../../node_modules/graphql/type/definition.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.GraphQLUnionType = exports.GraphQLScalarType = exports.GraphQLObjectType = exports.GraphQLNonNull = exports.GraphQLList = exports.GraphQLInterfaceType = exports.GraphQLInputObjectType = exports.GraphQLEnumType = void 0;
    exports.argsToArgsConfig = argsToArgsConfig;
    exports.assertAbstractType = assertAbstractType;
    exports.assertCompositeType = assertCompositeType;
    exports.assertEnumType = assertEnumType;
    exports.assertInputObjectType = assertInputObjectType;
    exports.assertInputType = assertInputType;
    exports.assertInterfaceType = assertInterfaceType;
    exports.assertLeafType = assertLeafType;
    exports.assertListType = assertListType;
    exports.assertNamedType = assertNamedType;
    exports.assertNonNullType = assertNonNullType;
    exports.assertNullableType = assertNullableType;
    exports.assertObjectType = assertObjectType;
    exports.assertOutputType = assertOutputType;
    exports.assertScalarType = assertScalarType;
    exports.assertType = assertType;
    exports.assertUnionType = assertUnionType;
    exports.assertWrappingType = assertWrappingType;
    exports.defineArguments = defineArguments;
    exports.getNamedType = getNamedType;
    exports.getNullableType = getNullableType;
    exports.isAbstractType = isAbstractType;
    exports.isCompositeType = isCompositeType;
    exports.isEnumType = isEnumType;
    exports.isInputObjectType = isInputObjectType;
    exports.isInputType = isInputType;
    exports.isInterfaceType = isInterfaceType;
    exports.isLeafType = isLeafType;
    exports.isListType = isListType;
    exports.isNamedType = isNamedType;
    exports.isNonNullType = isNonNullType;
    exports.isNullableType = isNullableType;
    exports.isObjectType = isObjectType;
    exports.isOutputType = isOutputType;
    exports.isRequiredArgument = isRequiredArgument;
    exports.isRequiredInputField = isRequiredInputField;
    exports.isScalarType = isScalarType;
    exports.isType = isType;
    exports.isUnionType = isUnionType;
    exports.isWrappingType = isWrappingType;
    exports.resolveObjMapThunk = resolveObjMapThunk;
    exports.resolveReadonlyArrayThunk = resolveReadonlyArrayThunk;
    var _devAssert = require_devAssert();
    var _didYouMean = require_didYouMean();
    var _identityFunc = require_identityFunc();
    var _inspect = require_inspect();
    var _instanceOf = require_instanceOf();
    var _isObjectLike = require_isObjectLike();
    var _keyMap = require_keyMap();
    var _keyValMap = require_keyValMap();
    var _mapValue = require_mapValue();
    var _suggestionList = require_suggestionList();
    var _toObjMap = require_toObjMap();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _valueFromASTUntyped = require_valueFromASTUntyped();
    var _assertName = require_assertName();
    function isType(type) {
      return isScalarType(type) || isObjectType(type) || isInterfaceType(type) || isUnionType(type) || isEnumType(type) || isInputObjectType(type) || isListType(type) || isNonNullType(type);
    }
    __name(isType, "isType");
    function assertType(type) {
      if (!isType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL type.`
        );
      }
      return type;
    }
    __name(assertType, "assertType");
    function isScalarType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLScalarType);
    }
    __name(isScalarType, "isScalarType");
    function assertScalarType(type) {
      if (!isScalarType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL Scalar type.`
        );
      }
      return type;
    }
    __name(assertScalarType, "assertScalarType");
    function isObjectType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLObjectType);
    }
    __name(isObjectType, "isObjectType");
    function assertObjectType(type) {
      if (!isObjectType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL Object type.`
        );
      }
      return type;
    }
    __name(assertObjectType, "assertObjectType");
    function isInterfaceType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLInterfaceType);
    }
    __name(isInterfaceType, "isInterfaceType");
    function assertInterfaceType(type) {
      if (!isInterfaceType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL Interface type.`
        );
      }
      return type;
    }
    __name(assertInterfaceType, "assertInterfaceType");
    function isUnionType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLUnionType);
    }
    __name(isUnionType, "isUnionType");
    function assertUnionType(type) {
      if (!isUnionType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL Union type.`
        );
      }
      return type;
    }
    __name(assertUnionType, "assertUnionType");
    function isEnumType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLEnumType);
    }
    __name(isEnumType, "isEnumType");
    function assertEnumType(type) {
      if (!isEnumType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL Enum type.`
        );
      }
      return type;
    }
    __name(assertEnumType, "assertEnumType");
    function isInputObjectType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLInputObjectType);
    }
    __name(isInputObjectType, "isInputObjectType");
    function assertInputObjectType(type) {
      if (!isInputObjectType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(
            type
          )} to be a GraphQL Input Object type.`
        );
      }
      return type;
    }
    __name(assertInputObjectType, "assertInputObjectType");
    function isListType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLList);
    }
    __name(isListType, "isListType");
    function assertListType(type) {
      if (!isListType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL List type.`
        );
      }
      return type;
    }
    __name(assertListType, "assertListType");
    function isNonNullType(type) {
      return (0, _instanceOf.instanceOf)(type, GraphQLNonNull);
    }
    __name(isNonNullType, "isNonNullType");
    function assertNonNullType(type) {
      if (!isNonNullType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL Non-Null type.`
        );
      }
      return type;
    }
    __name(assertNonNullType, "assertNonNullType");
    function isInputType(type) {
      return isScalarType(type) || isEnumType(type) || isInputObjectType(type) || isWrappingType(type) && isInputType(type.ofType);
    }
    __name(isInputType, "isInputType");
    function assertInputType(type) {
      if (!isInputType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL input type.`
        );
      }
      return type;
    }
    __name(assertInputType, "assertInputType");
    function isOutputType(type) {
      return isScalarType(type) || isObjectType(type) || isInterfaceType(type) || isUnionType(type) || isEnumType(type) || isWrappingType(type) && isOutputType(type.ofType);
    }
    __name(isOutputType, "isOutputType");
    function assertOutputType(type) {
      if (!isOutputType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL output type.`
        );
      }
      return type;
    }
    __name(assertOutputType, "assertOutputType");
    function isLeafType(type) {
      return isScalarType(type) || isEnumType(type);
    }
    __name(isLeafType, "isLeafType");
    function assertLeafType(type) {
      if (!isLeafType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL leaf type.`
        );
      }
      return type;
    }
    __name(assertLeafType, "assertLeafType");
    function isCompositeType(type) {
      return isObjectType(type) || isInterfaceType(type) || isUnionType(type);
    }
    __name(isCompositeType, "isCompositeType");
    function assertCompositeType(type) {
      if (!isCompositeType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL composite type.`
        );
      }
      return type;
    }
    __name(assertCompositeType, "assertCompositeType");
    function isAbstractType(type) {
      return isInterfaceType(type) || isUnionType(type);
    }
    __name(isAbstractType, "isAbstractType");
    function assertAbstractType(type) {
      if (!isAbstractType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL abstract type.`
        );
      }
      return type;
    }
    __name(assertAbstractType, "assertAbstractType");
    var GraphQLList = class {
      static {
        __name(this, "GraphQLList");
      }
      constructor(ofType) {
        isType(ofType) || (0, _devAssert.devAssert)(
          false,
          `Expected ${(0, _inspect.inspect)(ofType)} to be a GraphQL type.`
        );
        this.ofType = ofType;
      }
      get [Symbol.toStringTag]() {
        return "GraphQLList";
      }
      toString() {
        return "[" + String(this.ofType) + "]";
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLList = GraphQLList;
    var GraphQLNonNull = class {
      static {
        __name(this, "GraphQLNonNull");
      }
      constructor(ofType) {
        isNullableType(ofType) || (0, _devAssert.devAssert)(
          false,
          `Expected ${(0, _inspect.inspect)(
            ofType
          )} to be a GraphQL nullable type.`
        );
        this.ofType = ofType;
      }
      get [Symbol.toStringTag]() {
        return "GraphQLNonNull";
      }
      toString() {
        return String(this.ofType) + "!";
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLNonNull = GraphQLNonNull;
    function isWrappingType(type) {
      return isListType(type) || isNonNullType(type);
    }
    __name(isWrappingType, "isWrappingType");
    function assertWrappingType(type) {
      if (!isWrappingType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL wrapping type.`
        );
      }
      return type;
    }
    __name(assertWrappingType, "assertWrappingType");
    function isNullableType(type) {
      return isType(type) && !isNonNullType(type);
    }
    __name(isNullableType, "isNullableType");
    function assertNullableType(type) {
      if (!isNullableType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL nullable type.`
        );
      }
      return type;
    }
    __name(assertNullableType, "assertNullableType");
    function getNullableType(type) {
      if (type) {
        return isNonNullType(type) ? type.ofType : type;
      }
    }
    __name(getNullableType, "getNullableType");
    function isNamedType(type) {
      return isScalarType(type) || isObjectType(type) || isInterfaceType(type) || isUnionType(type) || isEnumType(type) || isInputObjectType(type);
    }
    __name(isNamedType, "isNamedType");
    function assertNamedType(type) {
      if (!isNamedType(type)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(type)} to be a GraphQL named type.`
        );
      }
      return type;
    }
    __name(assertNamedType, "assertNamedType");
    function getNamedType(type) {
      if (type) {
        let unwrappedType = type;
        while (isWrappingType(unwrappedType)) {
          unwrappedType = unwrappedType.ofType;
        }
        return unwrappedType;
      }
    }
    __name(getNamedType, "getNamedType");
    function resolveReadonlyArrayThunk(thunk) {
      return typeof thunk === "function" ? thunk() : thunk;
    }
    __name(resolveReadonlyArrayThunk, "resolveReadonlyArrayThunk");
    function resolveObjMapThunk(thunk) {
      return typeof thunk === "function" ? thunk() : thunk;
    }
    __name(resolveObjMapThunk, "resolveObjMapThunk");
    var GraphQLScalarType = class {
      static {
        __name(this, "GraphQLScalarType");
      }
      constructor(config) {
        var _config$parseValue, _config$serialize, _config$parseLiteral, _config$extensionASTN;
        const parseValue = (_config$parseValue = config.parseValue) !== null && _config$parseValue !== void 0 ? _config$parseValue : _identityFunc.identityFunc;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.specifiedByURL = config.specifiedByURL;
        this.serialize = (_config$serialize = config.serialize) !== null && _config$serialize !== void 0 ? _config$serialize : _identityFunc.identityFunc;
        this.parseValue = parseValue;
        this.parseLiteral = (_config$parseLiteral = config.parseLiteral) !== null && _config$parseLiteral !== void 0 ? _config$parseLiteral : (node, variables) => parseValue(
          (0, _valueFromASTUntyped.valueFromASTUntyped)(node, variables)
        );
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN = config.extensionASTNodes) !== null && _config$extensionASTN !== void 0 ? _config$extensionASTN : [];
        config.specifiedByURL == null || typeof config.specifiedByURL === "string" || (0, _devAssert.devAssert)(
          false,
          `${this.name} must provide "specifiedByURL" as a string, but got: ${(0, _inspect.inspect)(config.specifiedByURL)}.`
        );
        config.serialize == null || typeof config.serialize === "function" || (0, _devAssert.devAssert)(
          false,
          `${this.name} must provide "serialize" function. If this custom Scalar is also used as an input type, ensure "parseValue" and "parseLiteral" functions are also provided.`
        );
        if (config.parseLiteral) {
          typeof config.parseValue === "function" && typeof config.parseLiteral === "function" || (0, _devAssert.devAssert)(
            false,
            `${this.name} must provide both "parseValue" and "parseLiteral" functions.`
          );
        }
      }
      get [Symbol.toStringTag]() {
        return "GraphQLScalarType";
      }
      toConfig() {
        return {
          name: this.name,
          description: this.description,
          specifiedByURL: this.specifiedByURL,
          serialize: this.serialize,
          parseValue: this.parseValue,
          parseLiteral: this.parseLiteral,
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes
        };
      }
      toString() {
        return this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLScalarType = GraphQLScalarType;
    var GraphQLObjectType = class {
      static {
        __name(this, "GraphQLObjectType");
      }
      constructor(config) {
        var _config$extensionASTN2;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.isTypeOf = config.isTypeOf;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN2 = config.extensionASTNodes) !== null && _config$extensionASTN2 !== void 0 ? _config$extensionASTN2 : [];
        this._fields = () => defineFieldMap(config);
        this._interfaces = () => defineInterfaces(config);
        config.isTypeOf == null || typeof config.isTypeOf === "function" || (0, _devAssert.devAssert)(
          false,
          `${this.name} must provide "isTypeOf" as a function, but got: ${(0, _inspect.inspect)(config.isTypeOf)}.`
        );
      }
      get [Symbol.toStringTag]() {
        return "GraphQLObjectType";
      }
      getFields() {
        if (typeof this._fields === "function") {
          this._fields = this._fields();
        }
        return this._fields;
      }
      getInterfaces() {
        if (typeof this._interfaces === "function") {
          this._interfaces = this._interfaces();
        }
        return this._interfaces;
      }
      toConfig() {
        return {
          name: this.name,
          description: this.description,
          interfaces: this.getInterfaces(),
          fields: fieldsToFieldsConfig(this.getFields()),
          isTypeOf: this.isTypeOf,
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes
        };
      }
      toString() {
        return this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLObjectType = GraphQLObjectType;
    function defineInterfaces(config) {
      var _config$interfaces;
      const interfaces = resolveReadonlyArrayThunk(
        (_config$interfaces = config.interfaces) !== null && _config$interfaces !== void 0 ? _config$interfaces : []
      );
      Array.isArray(interfaces) || (0, _devAssert.devAssert)(
        false,
        `${config.name} interfaces must be an Array or a function which returns an Array.`
      );
      return interfaces;
    }
    __name(defineInterfaces, "defineInterfaces");
    function defineFieldMap(config) {
      const fieldMap = resolveObjMapThunk(config.fields);
      isPlainObj(fieldMap) || (0, _devAssert.devAssert)(
        false,
        `${config.name} fields must be an object with field names as keys or a function which returns such an object.`
      );
      return (0, _mapValue.mapValue)(fieldMap, (fieldConfig, fieldName) => {
        var _fieldConfig$args;
        isPlainObj(fieldConfig) || (0, _devAssert.devAssert)(
          false,
          `${config.name}.${fieldName} field config must be an object.`
        );
        fieldConfig.resolve == null || typeof fieldConfig.resolve === "function" || (0, _devAssert.devAssert)(
          false,
          `${config.name}.${fieldName} field resolver must be a function if provided, but got: ${(0, _inspect.inspect)(fieldConfig.resolve)}.`
        );
        const argsConfig = (_fieldConfig$args = fieldConfig.args) !== null && _fieldConfig$args !== void 0 ? _fieldConfig$args : {};
        isPlainObj(argsConfig) || (0, _devAssert.devAssert)(
          false,
          `${config.name}.${fieldName} args must be an object with argument names as keys.`
        );
        return {
          name: (0, _assertName.assertName)(fieldName),
          description: fieldConfig.description,
          type: fieldConfig.type,
          args: defineArguments(argsConfig),
          resolve: fieldConfig.resolve,
          subscribe: fieldConfig.subscribe,
          deprecationReason: fieldConfig.deprecationReason,
          extensions: (0, _toObjMap.toObjMap)(fieldConfig.extensions),
          astNode: fieldConfig.astNode
        };
      });
    }
    __name(defineFieldMap, "defineFieldMap");
    function defineArguments(config) {
      return Object.entries(config).map(([argName, argConfig]) => ({
        name: (0, _assertName.assertName)(argName),
        description: argConfig.description,
        type: argConfig.type,
        defaultValue: argConfig.defaultValue,
        deprecationReason: argConfig.deprecationReason,
        extensions: (0, _toObjMap.toObjMap)(argConfig.extensions),
        astNode: argConfig.astNode
      }));
    }
    __name(defineArguments, "defineArguments");
    function isPlainObj(obj) {
      return (0, _isObjectLike.isObjectLike)(obj) && !Array.isArray(obj);
    }
    __name(isPlainObj, "isPlainObj");
    function fieldsToFieldsConfig(fields) {
      return (0, _mapValue.mapValue)(fields, (field) => ({
        description: field.description,
        type: field.type,
        args: argsToArgsConfig(field.args),
        resolve: field.resolve,
        subscribe: field.subscribe,
        deprecationReason: field.deprecationReason,
        extensions: field.extensions,
        astNode: field.astNode
      }));
    }
    __name(fieldsToFieldsConfig, "fieldsToFieldsConfig");
    function argsToArgsConfig(args) {
      return (0, _keyValMap.keyValMap)(
        args,
        (arg) => arg.name,
        (arg) => ({
          description: arg.description,
          type: arg.type,
          defaultValue: arg.defaultValue,
          deprecationReason: arg.deprecationReason,
          extensions: arg.extensions,
          astNode: arg.astNode
        })
      );
    }
    __name(argsToArgsConfig, "argsToArgsConfig");
    function isRequiredArgument(arg) {
      return isNonNullType(arg.type) && arg.defaultValue === void 0;
    }
    __name(isRequiredArgument, "isRequiredArgument");
    var GraphQLInterfaceType = class {
      static {
        __name(this, "GraphQLInterfaceType");
      }
      constructor(config) {
        var _config$extensionASTN3;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.resolveType = config.resolveType;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN3 = config.extensionASTNodes) !== null && _config$extensionASTN3 !== void 0 ? _config$extensionASTN3 : [];
        this._fields = defineFieldMap.bind(void 0, config);
        this._interfaces = defineInterfaces.bind(void 0, config);
        config.resolveType == null || typeof config.resolveType === "function" || (0, _devAssert.devAssert)(
          false,
          `${this.name} must provide "resolveType" as a function, but got: ${(0, _inspect.inspect)(config.resolveType)}.`
        );
      }
      get [Symbol.toStringTag]() {
        return "GraphQLInterfaceType";
      }
      getFields() {
        if (typeof this._fields === "function") {
          this._fields = this._fields();
        }
        return this._fields;
      }
      getInterfaces() {
        if (typeof this._interfaces === "function") {
          this._interfaces = this._interfaces();
        }
        return this._interfaces;
      }
      toConfig() {
        return {
          name: this.name,
          description: this.description,
          interfaces: this.getInterfaces(),
          fields: fieldsToFieldsConfig(this.getFields()),
          resolveType: this.resolveType,
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes
        };
      }
      toString() {
        return this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLInterfaceType = GraphQLInterfaceType;
    var GraphQLUnionType = class {
      static {
        __name(this, "GraphQLUnionType");
      }
      constructor(config) {
        var _config$extensionASTN4;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.resolveType = config.resolveType;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN4 = config.extensionASTNodes) !== null && _config$extensionASTN4 !== void 0 ? _config$extensionASTN4 : [];
        this._types = defineTypes.bind(void 0, config);
        config.resolveType == null || typeof config.resolveType === "function" || (0, _devAssert.devAssert)(
          false,
          `${this.name} must provide "resolveType" as a function, but got: ${(0, _inspect.inspect)(config.resolveType)}.`
        );
      }
      get [Symbol.toStringTag]() {
        return "GraphQLUnionType";
      }
      getTypes() {
        if (typeof this._types === "function") {
          this._types = this._types();
        }
        return this._types;
      }
      toConfig() {
        return {
          name: this.name,
          description: this.description,
          types: this.getTypes(),
          resolveType: this.resolveType,
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes
        };
      }
      toString() {
        return this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLUnionType = GraphQLUnionType;
    function defineTypes(config) {
      const types = resolveReadonlyArrayThunk(config.types);
      Array.isArray(types) || (0, _devAssert.devAssert)(
        false,
        `Must provide Array of types or a function which returns such an array for Union ${config.name}.`
      );
      return types;
    }
    __name(defineTypes, "defineTypes");
    var GraphQLEnumType = class {
      static {
        __name(this, "GraphQLEnumType");
      }
      /* <T> */
      constructor(config) {
        var _config$extensionASTN5;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN5 = config.extensionASTNodes) !== null && _config$extensionASTN5 !== void 0 ? _config$extensionASTN5 : [];
        this._values = typeof config.values === "function" ? config.values : defineEnumValues(this.name, config.values);
        this._valueLookup = null;
        this._nameLookup = null;
      }
      get [Symbol.toStringTag]() {
        return "GraphQLEnumType";
      }
      getValues() {
        if (typeof this._values === "function") {
          this._values = defineEnumValues(this.name, this._values());
        }
        return this._values;
      }
      getValue(name) {
        if (this._nameLookup === null) {
          this._nameLookup = (0, _keyMap.keyMap)(
            this.getValues(),
            (value) => value.name
          );
        }
        return this._nameLookup[name];
      }
      serialize(outputValue) {
        if (this._valueLookup === null) {
          this._valueLookup = new Map(
            this.getValues().map((enumValue2) => [enumValue2.value, enumValue2])
          );
        }
        const enumValue = this._valueLookup.get(outputValue);
        if (enumValue === void 0) {
          throw new _GraphQLError.GraphQLError(
            `Enum "${this.name}" cannot represent value: ${(0, _inspect.inspect)(
              outputValue
            )}`
          );
        }
        return enumValue.name;
      }
      parseValue(inputValue) {
        if (typeof inputValue !== "string") {
          const valueStr = (0, _inspect.inspect)(inputValue);
          throw new _GraphQLError.GraphQLError(
            `Enum "${this.name}" cannot represent non-string value: ${valueStr}.` + didYouMeanEnumValue(this, valueStr)
          );
        }
        const enumValue = this.getValue(inputValue);
        if (enumValue == null) {
          throw new _GraphQLError.GraphQLError(
            `Value "${inputValue}" does not exist in "${this.name}" enum.` + didYouMeanEnumValue(this, inputValue)
          );
        }
        return enumValue.value;
      }
      parseLiteral(valueNode, _variables) {
        if (valueNode.kind !== _kinds.Kind.ENUM) {
          const valueStr = (0, _printer.print)(valueNode);
          throw new _GraphQLError.GraphQLError(
            `Enum "${this.name}" cannot represent non-enum value: ${valueStr}.` + didYouMeanEnumValue(this, valueStr),
            {
              nodes: valueNode
            }
          );
        }
        const enumValue = this.getValue(valueNode.value);
        if (enumValue == null) {
          const valueStr = (0, _printer.print)(valueNode);
          throw new _GraphQLError.GraphQLError(
            `Value "${valueStr}" does not exist in "${this.name}" enum.` + didYouMeanEnumValue(this, valueStr),
            {
              nodes: valueNode
            }
          );
        }
        return enumValue.value;
      }
      toConfig() {
        const values = (0, _keyValMap.keyValMap)(
          this.getValues(),
          (value) => value.name,
          (value) => ({
            description: value.description,
            value: value.value,
            deprecationReason: value.deprecationReason,
            extensions: value.extensions,
            astNode: value.astNode
          })
        );
        return {
          name: this.name,
          description: this.description,
          values,
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes
        };
      }
      toString() {
        return this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLEnumType = GraphQLEnumType;
    function didYouMeanEnumValue(enumType, unknownValueStr) {
      const allNames = enumType.getValues().map((value) => value.name);
      const suggestedValues = (0, _suggestionList.suggestionList)(
        unknownValueStr,
        allNames
      );
      return (0, _didYouMean.didYouMean)("the enum value", suggestedValues);
    }
    __name(didYouMeanEnumValue, "didYouMeanEnumValue");
    function defineEnumValues(typeName, valueMap) {
      isPlainObj(valueMap) || (0, _devAssert.devAssert)(
        false,
        `${typeName} values must be an object with value names as keys.`
      );
      return Object.entries(valueMap).map(([valueName, valueConfig]) => {
        isPlainObj(valueConfig) || (0, _devAssert.devAssert)(
          false,
          `${typeName}.${valueName} must refer to an object with a "value" key representing an internal value but got: ${(0, _inspect.inspect)(
            valueConfig
          )}.`
        );
        return {
          name: (0, _assertName.assertEnumValueName)(valueName),
          description: valueConfig.description,
          value: valueConfig.value !== void 0 ? valueConfig.value : valueName,
          deprecationReason: valueConfig.deprecationReason,
          extensions: (0, _toObjMap.toObjMap)(valueConfig.extensions),
          astNode: valueConfig.astNode
        };
      });
    }
    __name(defineEnumValues, "defineEnumValues");
    var GraphQLInputObjectType = class {
      static {
        __name(this, "GraphQLInputObjectType");
      }
      constructor(config) {
        var _config$extensionASTN6, _config$isOneOf;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN6 = config.extensionASTNodes) !== null && _config$extensionASTN6 !== void 0 ? _config$extensionASTN6 : [];
        this.isOneOf = (_config$isOneOf = config.isOneOf) !== null && _config$isOneOf !== void 0 ? _config$isOneOf : false;
        this._fields = defineInputFieldMap.bind(void 0, config);
      }
      get [Symbol.toStringTag]() {
        return "GraphQLInputObjectType";
      }
      getFields() {
        if (typeof this._fields === "function") {
          this._fields = this._fields();
        }
        return this._fields;
      }
      toConfig() {
        const fields = (0, _mapValue.mapValue)(this.getFields(), (field) => ({
          description: field.description,
          type: field.type,
          defaultValue: field.defaultValue,
          deprecationReason: field.deprecationReason,
          extensions: field.extensions,
          astNode: field.astNode
        }));
        return {
          name: this.name,
          description: this.description,
          fields,
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes,
          isOneOf: this.isOneOf
        };
      }
      toString() {
        return this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLInputObjectType = GraphQLInputObjectType;
    function defineInputFieldMap(config) {
      const fieldMap = resolveObjMapThunk(config.fields);
      isPlainObj(fieldMap) || (0, _devAssert.devAssert)(
        false,
        `${config.name} fields must be an object with field names as keys or a function which returns such an object.`
      );
      return (0, _mapValue.mapValue)(fieldMap, (fieldConfig, fieldName) => {
        !("resolve" in fieldConfig) || (0, _devAssert.devAssert)(
          false,
          `${config.name}.${fieldName} field has a resolve property, but Input Types cannot define resolvers.`
        );
        return {
          name: (0, _assertName.assertName)(fieldName),
          description: fieldConfig.description,
          type: fieldConfig.type,
          defaultValue: fieldConfig.defaultValue,
          deprecationReason: fieldConfig.deprecationReason,
          extensions: (0, _toObjMap.toObjMap)(fieldConfig.extensions),
          astNode: fieldConfig.astNode
        };
      });
    }
    __name(defineInputFieldMap, "defineInputFieldMap");
    function isRequiredInputField(field) {
      return isNonNullType(field.type) && field.defaultValue === void 0;
    }
    __name(isRequiredInputField, "isRequiredInputField");
  }
});

// ../../node_modules/graphql/utilities/typeComparators.js
var require_typeComparators = __commonJS({
  "../../node_modules/graphql/utilities/typeComparators.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.doTypesOverlap = doTypesOverlap;
    exports.isEqualType = isEqualType;
    exports.isTypeSubTypeOf = isTypeSubTypeOf;
    var _definition = require_definition();
    function isEqualType(typeA, typeB) {
      if (typeA === typeB) {
        return true;
      }
      if ((0, _definition.isNonNullType)(typeA) && (0, _definition.isNonNullType)(typeB)) {
        return isEqualType(typeA.ofType, typeB.ofType);
      }
      if ((0, _definition.isListType)(typeA) && (0, _definition.isListType)(typeB)) {
        return isEqualType(typeA.ofType, typeB.ofType);
      }
      return false;
    }
    __name(isEqualType, "isEqualType");
    function isTypeSubTypeOf(schema, maybeSubType, superType) {
      if (maybeSubType === superType) {
        return true;
      }
      if ((0, _definition.isNonNullType)(superType)) {
        if ((0, _definition.isNonNullType)(maybeSubType)) {
          return isTypeSubTypeOf(schema, maybeSubType.ofType, superType.ofType);
        }
        return false;
      }
      if ((0, _definition.isNonNullType)(maybeSubType)) {
        return isTypeSubTypeOf(schema, maybeSubType.ofType, superType);
      }
      if ((0, _definition.isListType)(superType)) {
        if ((0, _definition.isListType)(maybeSubType)) {
          return isTypeSubTypeOf(schema, maybeSubType.ofType, superType.ofType);
        }
        return false;
      }
      if ((0, _definition.isListType)(maybeSubType)) {
        return false;
      }
      return (0, _definition.isAbstractType)(superType) && ((0, _definition.isInterfaceType)(maybeSubType) || (0, _definition.isObjectType)(maybeSubType)) && schema.isSubType(superType, maybeSubType);
    }
    __name(isTypeSubTypeOf, "isTypeSubTypeOf");
    function doTypesOverlap(schema, typeA, typeB) {
      if (typeA === typeB) {
        return true;
      }
      if ((0, _definition.isAbstractType)(typeA)) {
        if ((0, _definition.isAbstractType)(typeB)) {
          return schema.getPossibleTypes(typeA).some((type) => schema.isSubType(typeB, type));
        }
        return schema.isSubType(typeA, typeB);
      }
      if ((0, _definition.isAbstractType)(typeB)) {
        return schema.isSubType(typeB, typeA);
      }
      return false;
    }
    __name(doTypesOverlap, "doTypesOverlap");
  }
});

// ../../node_modules/graphql/type/scalars.js
var require_scalars = __commonJS({
  "../../node_modules/graphql/type/scalars.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.GraphQLString = exports.GraphQLInt = exports.GraphQLID = exports.GraphQLFloat = exports.GraphQLBoolean = exports.GRAPHQL_MIN_INT = exports.GRAPHQL_MAX_INT = void 0;
    exports.isSpecifiedScalarType = isSpecifiedScalarType;
    exports.specifiedScalarTypes = void 0;
    var _inspect = require_inspect();
    var _isObjectLike = require_isObjectLike();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _definition = require_definition();
    var GRAPHQL_MAX_INT = 2147483647;
    exports.GRAPHQL_MAX_INT = GRAPHQL_MAX_INT;
    var GRAPHQL_MIN_INT = -2147483648;
    exports.GRAPHQL_MIN_INT = GRAPHQL_MIN_INT;
    var GraphQLInt = new _definition.GraphQLScalarType({
      name: "Int",
      description: "The `Int` scalar type represents non-fractional signed whole numeric values. Int can represent values between -(2^31) and 2^31 - 1.",
      serialize(outputValue) {
        const coercedValue = serializeObject(outputValue);
        if (typeof coercedValue === "boolean") {
          return coercedValue ? 1 : 0;
        }
        let num = coercedValue;
        if (typeof coercedValue === "string" && coercedValue !== "") {
          num = Number(coercedValue);
        }
        if (typeof num !== "number" || !Number.isInteger(num)) {
          throw new _GraphQLError.GraphQLError(
            `Int cannot represent non-integer value: ${(0, _inspect.inspect)(
              coercedValue
            )}`
          );
        }
        if (num > GRAPHQL_MAX_INT || num < GRAPHQL_MIN_INT) {
          throw new _GraphQLError.GraphQLError(
            "Int cannot represent non 32-bit signed integer value: " + (0, _inspect.inspect)(coercedValue)
          );
        }
        return num;
      },
      parseValue(inputValue) {
        if (typeof inputValue !== "number" || !Number.isInteger(inputValue)) {
          throw new _GraphQLError.GraphQLError(
            `Int cannot represent non-integer value: ${(0, _inspect.inspect)(
              inputValue
            )}`
          );
        }
        if (inputValue > GRAPHQL_MAX_INT || inputValue < GRAPHQL_MIN_INT) {
          throw new _GraphQLError.GraphQLError(
            `Int cannot represent non 32-bit signed integer value: ${inputValue}`
          );
        }
        return inputValue;
      },
      parseLiteral(valueNode) {
        if (valueNode.kind !== _kinds.Kind.INT) {
          throw new _GraphQLError.GraphQLError(
            `Int cannot represent non-integer value: ${(0, _printer.print)(
              valueNode
            )}`,
            {
              nodes: valueNode
            }
          );
        }
        const num = parseInt(valueNode.value, 10);
        if (num > GRAPHQL_MAX_INT || num < GRAPHQL_MIN_INT) {
          throw new _GraphQLError.GraphQLError(
            `Int cannot represent non 32-bit signed integer value: ${valueNode.value}`,
            {
              nodes: valueNode
            }
          );
        }
        return num;
      }
    });
    exports.GraphQLInt = GraphQLInt;
    var GraphQLFloat = new _definition.GraphQLScalarType({
      name: "Float",
      description: "The `Float` scalar type represents signed double-precision fractional values as specified by [IEEE 754](https://en.wikipedia.org/wiki/IEEE_floating_point).",
      serialize(outputValue) {
        const coercedValue = serializeObject(outputValue);
        if (typeof coercedValue === "boolean") {
          return coercedValue ? 1 : 0;
        }
        let num = coercedValue;
        if (typeof coercedValue === "string" && coercedValue !== "") {
          num = Number(coercedValue);
        }
        if (typeof num !== "number" || !Number.isFinite(num)) {
          throw new _GraphQLError.GraphQLError(
            `Float cannot represent non numeric value: ${(0, _inspect.inspect)(
              coercedValue
            )}`
          );
        }
        return num;
      },
      parseValue(inputValue) {
        if (typeof inputValue !== "number" || !Number.isFinite(inputValue)) {
          throw new _GraphQLError.GraphQLError(
            `Float cannot represent non numeric value: ${(0, _inspect.inspect)(
              inputValue
            )}`
          );
        }
        return inputValue;
      },
      parseLiteral(valueNode) {
        if (valueNode.kind !== _kinds.Kind.FLOAT && valueNode.kind !== _kinds.Kind.INT) {
          throw new _GraphQLError.GraphQLError(
            `Float cannot represent non numeric value: ${(0, _printer.print)(
              valueNode
            )}`,
            valueNode
          );
        }
        return parseFloat(valueNode.value);
      }
    });
    exports.GraphQLFloat = GraphQLFloat;
    var GraphQLString = new _definition.GraphQLScalarType({
      name: "String",
      description: "The `String` scalar type represents textual data, represented as UTF-8 character sequences. The String type is most often used by GraphQL to represent free-form human-readable text.",
      serialize(outputValue) {
        const coercedValue = serializeObject(outputValue);
        if (typeof coercedValue === "string") {
          return coercedValue;
        }
        if (typeof coercedValue === "boolean") {
          return coercedValue ? "true" : "false";
        }
        if (typeof coercedValue === "number" && Number.isFinite(coercedValue)) {
          return coercedValue.toString();
        }
        throw new _GraphQLError.GraphQLError(
          `String cannot represent value: ${(0, _inspect.inspect)(outputValue)}`
        );
      },
      parseValue(inputValue) {
        if (typeof inputValue !== "string") {
          throw new _GraphQLError.GraphQLError(
            `String cannot represent a non string value: ${(0, _inspect.inspect)(
              inputValue
            )}`
          );
        }
        return inputValue;
      },
      parseLiteral(valueNode) {
        if (valueNode.kind !== _kinds.Kind.STRING) {
          throw new _GraphQLError.GraphQLError(
            `String cannot represent a non string value: ${(0, _printer.print)(
              valueNode
            )}`,
            {
              nodes: valueNode
            }
          );
        }
        return valueNode.value;
      }
    });
    exports.GraphQLString = GraphQLString;
    var GraphQLBoolean = new _definition.GraphQLScalarType({
      name: "Boolean",
      description: "The `Boolean` scalar type represents `true` or `false`.",
      serialize(outputValue) {
        const coercedValue = serializeObject(outputValue);
        if (typeof coercedValue === "boolean") {
          return coercedValue;
        }
        if (Number.isFinite(coercedValue)) {
          return coercedValue !== 0;
        }
        throw new _GraphQLError.GraphQLError(
          `Boolean cannot represent a non boolean value: ${(0, _inspect.inspect)(
            coercedValue
          )}`
        );
      },
      parseValue(inputValue) {
        if (typeof inputValue !== "boolean") {
          throw new _GraphQLError.GraphQLError(
            `Boolean cannot represent a non boolean value: ${(0, _inspect.inspect)(
              inputValue
            )}`
          );
        }
        return inputValue;
      },
      parseLiteral(valueNode) {
        if (valueNode.kind !== _kinds.Kind.BOOLEAN) {
          throw new _GraphQLError.GraphQLError(
            `Boolean cannot represent a non boolean value: ${(0, _printer.print)(
              valueNode
            )}`,
            {
              nodes: valueNode
            }
          );
        }
        return valueNode.value;
      }
    });
    exports.GraphQLBoolean = GraphQLBoolean;
    var GraphQLID = new _definition.GraphQLScalarType({
      name: "ID",
      description: 'The `ID` scalar type represents a unique identifier, often used to refetch an object or as key for a cache. The ID type appears in a JSON response as a String; however, it is not intended to be human-readable. When expected as an input type, any string (such as `"4"`) or integer (such as `4`) input value will be accepted as an ID.',
      serialize(outputValue) {
        const coercedValue = serializeObject(outputValue);
        if (typeof coercedValue === "string") {
          return coercedValue;
        }
        if (Number.isInteger(coercedValue)) {
          return String(coercedValue);
        }
        throw new _GraphQLError.GraphQLError(
          `ID cannot represent value: ${(0, _inspect.inspect)(outputValue)}`
        );
      },
      parseValue(inputValue) {
        if (typeof inputValue === "string") {
          return inputValue;
        }
        if (typeof inputValue === "number" && Number.isInteger(inputValue)) {
          return inputValue.toString();
        }
        throw new _GraphQLError.GraphQLError(
          `ID cannot represent value: ${(0, _inspect.inspect)(inputValue)}`
        );
      },
      parseLiteral(valueNode) {
        if (valueNode.kind !== _kinds.Kind.STRING && valueNode.kind !== _kinds.Kind.INT) {
          throw new _GraphQLError.GraphQLError(
            "ID cannot represent a non-string and non-integer value: " + (0, _printer.print)(valueNode),
            {
              nodes: valueNode
            }
          );
        }
        return valueNode.value;
      }
    });
    exports.GraphQLID = GraphQLID;
    var specifiedScalarTypes = Object.freeze([
      GraphQLString,
      GraphQLInt,
      GraphQLFloat,
      GraphQLBoolean,
      GraphQLID
    ]);
    exports.specifiedScalarTypes = specifiedScalarTypes;
    function isSpecifiedScalarType(type) {
      return specifiedScalarTypes.some(({ name }) => type.name === name);
    }
    __name(isSpecifiedScalarType, "isSpecifiedScalarType");
    function serializeObject(outputValue) {
      if ((0, _isObjectLike.isObjectLike)(outputValue)) {
        if (typeof outputValue.valueOf === "function") {
          const valueOfResult = outputValue.valueOf();
          if (!(0, _isObjectLike.isObjectLike)(valueOfResult)) {
            return valueOfResult;
          }
        }
        if (typeof outputValue.toJSON === "function") {
          return outputValue.toJSON();
        }
      }
      return outputValue;
    }
    __name(serializeObject, "serializeObject");
  }
});

// ../../node_modules/graphql/type/directives.js
var require_directives = __commonJS({
  "../../node_modules/graphql/type/directives.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.GraphQLSpecifiedByDirective = exports.GraphQLSkipDirective = exports.GraphQLOneOfDirective = exports.GraphQLIncludeDirective = exports.GraphQLDirective = exports.GraphQLDeprecatedDirective = exports.DEFAULT_DEPRECATION_REASON = void 0;
    exports.assertDirective = assertDirective;
    exports.isDirective = isDirective;
    exports.isSpecifiedDirective = isSpecifiedDirective;
    exports.specifiedDirectives = void 0;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _instanceOf = require_instanceOf();
    var _isObjectLike = require_isObjectLike();
    var _toObjMap = require_toObjMap();
    var _directiveLocation = require_directiveLocation();
    var _assertName = require_assertName();
    var _definition = require_definition();
    var _scalars = require_scalars();
    function isDirective(directive) {
      return (0, _instanceOf.instanceOf)(directive, GraphQLDirective);
    }
    __name(isDirective, "isDirective");
    function assertDirective(directive) {
      if (!isDirective(directive)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(directive)} to be a GraphQL directive.`
        );
      }
      return directive;
    }
    __name(assertDirective, "assertDirective");
    var GraphQLDirective = class {
      static {
        __name(this, "GraphQLDirective");
      }
      constructor(config) {
        var _config$isRepeatable, _config$args;
        this.name = (0, _assertName.assertName)(config.name);
        this.description = config.description;
        this.locations = config.locations;
        this.isRepeatable = (_config$isRepeatable = config.isRepeatable) !== null && _config$isRepeatable !== void 0 ? _config$isRepeatable : false;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        Array.isArray(config.locations) || (0, _devAssert.devAssert)(
          false,
          `@${config.name} locations must be an Array.`
        );
        const args = (_config$args = config.args) !== null && _config$args !== void 0 ? _config$args : {};
        (0, _isObjectLike.isObjectLike)(args) && !Array.isArray(args) || (0, _devAssert.devAssert)(
          false,
          `@${config.name} args must be an object with argument names as keys.`
        );
        this.args = (0, _definition.defineArguments)(args);
      }
      get [Symbol.toStringTag]() {
        return "GraphQLDirective";
      }
      toConfig() {
        return {
          name: this.name,
          description: this.description,
          locations: this.locations,
          args: (0, _definition.argsToArgsConfig)(this.args),
          isRepeatable: this.isRepeatable,
          extensions: this.extensions,
          astNode: this.astNode
        };
      }
      toString() {
        return "@" + this.name;
      }
      toJSON() {
        return this.toString();
      }
    };
    exports.GraphQLDirective = GraphQLDirective;
    var GraphQLIncludeDirective = new GraphQLDirective({
      name: "include",
      description: "Directs the executor to include this field or fragment only when the `if` argument is true.",
      locations: [
        _directiveLocation.DirectiveLocation.FIELD,
        _directiveLocation.DirectiveLocation.FRAGMENT_SPREAD,
        _directiveLocation.DirectiveLocation.INLINE_FRAGMENT
      ],
      args: {
        if: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLBoolean),
          description: "Included when true."
        }
      }
    });
    exports.GraphQLIncludeDirective = GraphQLIncludeDirective;
    var GraphQLSkipDirective = new GraphQLDirective({
      name: "skip",
      description: "Directs the executor to skip this field or fragment when the `if` argument is true.",
      locations: [
        _directiveLocation.DirectiveLocation.FIELD,
        _directiveLocation.DirectiveLocation.FRAGMENT_SPREAD,
        _directiveLocation.DirectiveLocation.INLINE_FRAGMENT
      ],
      args: {
        if: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLBoolean),
          description: "Skipped when true."
        }
      }
    });
    exports.GraphQLSkipDirective = GraphQLSkipDirective;
    var DEFAULT_DEPRECATION_REASON = "No longer supported";
    exports.DEFAULT_DEPRECATION_REASON = DEFAULT_DEPRECATION_REASON;
    var GraphQLDeprecatedDirective = new GraphQLDirective({
      name: "deprecated",
      description: "Marks an element of a GraphQL schema as no longer supported.",
      locations: [
        _directiveLocation.DirectiveLocation.FIELD_DEFINITION,
        _directiveLocation.DirectiveLocation.ARGUMENT_DEFINITION,
        _directiveLocation.DirectiveLocation.INPUT_FIELD_DEFINITION,
        _directiveLocation.DirectiveLocation.ENUM_VALUE
      ],
      args: {
        reason: {
          type: _scalars.GraphQLString,
          description: "Explains why this element was deprecated, usually also including a suggestion for how to access supported similar data. Formatted using the Markdown syntax, as specified by [CommonMark](https://commonmark.org/).",
          defaultValue: DEFAULT_DEPRECATION_REASON
        }
      }
    });
    exports.GraphQLDeprecatedDirective = GraphQLDeprecatedDirective;
    var GraphQLSpecifiedByDirective = new GraphQLDirective({
      name: "specifiedBy",
      description: "Exposes a URL that specifies the behavior of this scalar.",
      locations: [_directiveLocation.DirectiveLocation.SCALAR],
      args: {
        url: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
          description: "The URL that specifies the behavior of this scalar."
        }
      }
    });
    exports.GraphQLSpecifiedByDirective = GraphQLSpecifiedByDirective;
    var GraphQLOneOfDirective = new GraphQLDirective({
      name: "oneOf",
      description: "Indicates exactly one field must be supplied and this field must not be `null`.",
      locations: [_directiveLocation.DirectiveLocation.INPUT_OBJECT],
      args: {}
    });
    exports.GraphQLOneOfDirective = GraphQLOneOfDirective;
    var specifiedDirectives = Object.freeze([
      GraphQLIncludeDirective,
      GraphQLSkipDirective,
      GraphQLDeprecatedDirective,
      GraphQLSpecifiedByDirective,
      GraphQLOneOfDirective
    ]);
    exports.specifiedDirectives = specifiedDirectives;
    function isSpecifiedDirective(directive) {
      return specifiedDirectives.some(({ name }) => name === directive.name);
    }
    __name(isSpecifiedDirective, "isSpecifiedDirective");
  }
});

// ../../node_modules/graphql/jsutils/isIterableObject.js
var require_isIterableObject = __commonJS({
  "../../node_modules/graphql/jsutils/isIterableObject.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.isIterableObject = isIterableObject;
    function isIterableObject(maybeIterable) {
      return typeof maybeIterable === "object" && typeof (maybeIterable === null || maybeIterable === void 0 ? void 0 : maybeIterable[Symbol.iterator]) === "function";
    }
    __name(isIterableObject, "isIterableObject");
  }
});

// ../../node_modules/graphql/utilities/astFromValue.js
var require_astFromValue = __commonJS({
  "../../node_modules/graphql/utilities/astFromValue.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.astFromValue = astFromValue;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _isIterableObject = require_isIterableObject();
    var _isObjectLike = require_isObjectLike();
    var _kinds = require_kinds();
    var _definition = require_definition();
    var _scalars = require_scalars();
    function astFromValue(value, type) {
      if ((0, _definition.isNonNullType)(type)) {
        const astValue = astFromValue(value, type.ofType);
        if ((astValue === null || astValue === void 0 ? void 0 : astValue.kind) === _kinds.Kind.NULL) {
          return null;
        }
        return astValue;
      }
      if (value === null) {
        return {
          kind: _kinds.Kind.NULL
        };
      }
      if (value === void 0) {
        return null;
      }
      if ((0, _definition.isListType)(type)) {
        const itemType = type.ofType;
        if ((0, _isIterableObject.isIterableObject)(value)) {
          const valuesNodes = [];
          for (const item of value) {
            const itemNode = astFromValue(item, itemType);
            if (itemNode != null) {
              valuesNodes.push(itemNode);
            }
          }
          return {
            kind: _kinds.Kind.LIST,
            values: valuesNodes
          };
        }
        return astFromValue(value, itemType);
      }
      if ((0, _definition.isInputObjectType)(type)) {
        if (!(0, _isObjectLike.isObjectLike)(value)) {
          return null;
        }
        const fieldNodes = [];
        for (const field of Object.values(type.getFields())) {
          const fieldValue = astFromValue(value[field.name], field.type);
          if (fieldValue) {
            fieldNodes.push({
              kind: _kinds.Kind.OBJECT_FIELD,
              name: {
                kind: _kinds.Kind.NAME,
                value: field.name
              },
              value: fieldValue
            });
          }
        }
        return {
          kind: _kinds.Kind.OBJECT,
          fields: fieldNodes
        };
      }
      if ((0, _definition.isLeafType)(type)) {
        const serialized = type.serialize(value);
        if (serialized == null) {
          return null;
        }
        if (typeof serialized === "boolean") {
          return {
            kind: _kinds.Kind.BOOLEAN,
            value: serialized
          };
        }
        if (typeof serialized === "number" && Number.isFinite(serialized)) {
          const stringNum = String(serialized);
          return integerStringRegExp.test(stringNum) ? {
            kind: _kinds.Kind.INT,
            value: stringNum
          } : {
            kind: _kinds.Kind.FLOAT,
            value: stringNum
          };
        }
        if (typeof serialized === "string") {
          if ((0, _definition.isEnumType)(type)) {
            return {
              kind: _kinds.Kind.ENUM,
              value: serialized
            };
          }
          if (type === _scalars.GraphQLID && integerStringRegExp.test(serialized)) {
            return {
              kind: _kinds.Kind.INT,
              value: serialized
            };
          }
          return {
            kind: _kinds.Kind.STRING,
            value: serialized
          };
        }
        throw new TypeError(
          `Cannot convert value to AST: ${(0, _inspect.inspect)(serialized)}.`
        );
      }
      (0, _invariant.invariant)(
        false,
        "Unexpected input type: " + (0, _inspect.inspect)(type)
      );
    }
    __name(astFromValue, "astFromValue");
    var integerStringRegExp = /^-?(?:0|[1-9][0-9]*)$/;
  }
});

// ../../node_modules/graphql/type/introspection.js
var require_introspection = __commonJS({
  "../../node_modules/graphql/type/introspection.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.introspectionTypes = exports.__TypeKind = exports.__Type = exports.__Schema = exports.__InputValue = exports.__Field = exports.__EnumValue = exports.__DirectiveLocation = exports.__Directive = exports.TypeNameMetaFieldDef = exports.TypeMetaFieldDef = exports.TypeKind = exports.SchemaMetaFieldDef = void 0;
    exports.isIntrospectionType = isIntrospectionType;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _directiveLocation = require_directiveLocation();
    var _printer = require_printer();
    var _astFromValue = require_astFromValue();
    var _definition = require_definition();
    var _scalars = require_scalars();
    var __Schema = new _definition.GraphQLObjectType({
      name: "__Schema",
      description: "A GraphQL Schema defines the capabilities of a GraphQL server. It exposes all available types and directives on the server, as well as the entry points for query, mutation, and subscription operations.",
      fields: /* @__PURE__ */ __name(() => ({
        description: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((schema) => schema.description, "resolve")
        },
        types: {
          description: "A list of all types supported by this server.",
          type: new _definition.GraphQLNonNull(
            new _definition.GraphQLList(new _definition.GraphQLNonNull(__Type))
          ),
          resolve(schema) {
            return Object.values(schema.getTypeMap());
          }
        },
        queryType: {
          description: "The type that query operations will be rooted at.",
          type: new _definition.GraphQLNonNull(__Type),
          resolve: /* @__PURE__ */ __name((schema) => schema.getQueryType(), "resolve")
        },
        mutationType: {
          description: "If this server supports mutation, the type that mutation operations will be rooted at.",
          type: __Type,
          resolve: /* @__PURE__ */ __name((schema) => schema.getMutationType(), "resolve")
        },
        subscriptionType: {
          description: "If this server support subscription, the type that subscription operations will be rooted at.",
          type: __Type,
          resolve: /* @__PURE__ */ __name((schema) => schema.getSubscriptionType(), "resolve")
        },
        directives: {
          description: "A list of all directives supported by this server.",
          type: new _definition.GraphQLNonNull(
            new _definition.GraphQLList(
              new _definition.GraphQLNonNull(__Directive)
            )
          ),
          resolve: /* @__PURE__ */ __name((schema) => schema.getDirectives(), "resolve")
        }
      }), "fields")
    });
    exports.__Schema = __Schema;
    var __Directive = new _definition.GraphQLObjectType({
      name: "__Directive",
      description: "A Directive provides a way to describe alternate runtime execution and type validation behavior in a GraphQL document.\n\nIn some cases, you need to provide options to alter GraphQL's execution behavior in ways field arguments will not suffice, such as conditionally including or skipping a field. Directives provide this by describing additional information to the executor.",
      fields: /* @__PURE__ */ __name(() => ({
        name: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
          resolve: /* @__PURE__ */ __name((directive) => directive.name, "resolve")
        },
        description: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((directive) => directive.description, "resolve")
        },
        isRepeatable: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLBoolean),
          resolve: /* @__PURE__ */ __name((directive) => directive.isRepeatable, "resolve")
        },
        locations: {
          type: new _definition.GraphQLNonNull(
            new _definition.GraphQLList(
              new _definition.GraphQLNonNull(__DirectiveLocation)
            )
          ),
          resolve: /* @__PURE__ */ __name((directive) => directive.locations, "resolve")
        },
        args: {
          type: new _definition.GraphQLNonNull(
            new _definition.GraphQLList(
              new _definition.GraphQLNonNull(__InputValue)
            )
          ),
          args: {
            includeDeprecated: {
              type: _scalars.GraphQLBoolean,
              defaultValue: false
            }
          },
          resolve(field, { includeDeprecated }) {
            return includeDeprecated ? field.args : field.args.filter((arg) => arg.deprecationReason == null);
          }
        }
      }), "fields")
    });
    exports.__Directive = __Directive;
    var __DirectiveLocation = new _definition.GraphQLEnumType({
      name: "__DirectiveLocation",
      description: "A Directive can be adjacent to many parts of the GraphQL language, a __DirectiveLocation describes one such possible adjacencies.",
      values: {
        QUERY: {
          value: _directiveLocation.DirectiveLocation.QUERY,
          description: "Location adjacent to a query operation."
        },
        MUTATION: {
          value: _directiveLocation.DirectiveLocation.MUTATION,
          description: "Location adjacent to a mutation operation."
        },
        SUBSCRIPTION: {
          value: _directiveLocation.DirectiveLocation.SUBSCRIPTION,
          description: "Location adjacent to a subscription operation."
        },
        FIELD: {
          value: _directiveLocation.DirectiveLocation.FIELD,
          description: "Location adjacent to a field."
        },
        FRAGMENT_DEFINITION: {
          value: _directiveLocation.DirectiveLocation.FRAGMENT_DEFINITION,
          description: "Location adjacent to a fragment definition."
        },
        FRAGMENT_SPREAD: {
          value: _directiveLocation.DirectiveLocation.FRAGMENT_SPREAD,
          description: "Location adjacent to a fragment spread."
        },
        INLINE_FRAGMENT: {
          value: _directiveLocation.DirectiveLocation.INLINE_FRAGMENT,
          description: "Location adjacent to an inline fragment."
        },
        VARIABLE_DEFINITION: {
          value: _directiveLocation.DirectiveLocation.VARIABLE_DEFINITION,
          description: "Location adjacent to a variable definition."
        },
        SCHEMA: {
          value: _directiveLocation.DirectiveLocation.SCHEMA,
          description: "Location adjacent to a schema definition."
        },
        SCALAR: {
          value: _directiveLocation.DirectiveLocation.SCALAR,
          description: "Location adjacent to a scalar definition."
        },
        OBJECT: {
          value: _directiveLocation.DirectiveLocation.OBJECT,
          description: "Location adjacent to an object type definition."
        },
        FIELD_DEFINITION: {
          value: _directiveLocation.DirectiveLocation.FIELD_DEFINITION,
          description: "Location adjacent to a field definition."
        },
        ARGUMENT_DEFINITION: {
          value: _directiveLocation.DirectiveLocation.ARGUMENT_DEFINITION,
          description: "Location adjacent to an argument definition."
        },
        INTERFACE: {
          value: _directiveLocation.DirectiveLocation.INTERFACE,
          description: "Location adjacent to an interface definition."
        },
        UNION: {
          value: _directiveLocation.DirectiveLocation.UNION,
          description: "Location adjacent to a union definition."
        },
        ENUM: {
          value: _directiveLocation.DirectiveLocation.ENUM,
          description: "Location adjacent to an enum definition."
        },
        ENUM_VALUE: {
          value: _directiveLocation.DirectiveLocation.ENUM_VALUE,
          description: "Location adjacent to an enum value definition."
        },
        INPUT_OBJECT: {
          value: _directiveLocation.DirectiveLocation.INPUT_OBJECT,
          description: "Location adjacent to an input object type definition."
        },
        INPUT_FIELD_DEFINITION: {
          value: _directiveLocation.DirectiveLocation.INPUT_FIELD_DEFINITION,
          description: "Location adjacent to an input object field definition."
        }
      }
    });
    exports.__DirectiveLocation = __DirectiveLocation;
    var __Type = new _definition.GraphQLObjectType({
      name: "__Type",
      description: "The fundamental unit of any GraphQL Schema is the type. There are many kinds of types in GraphQL as represented by the `__TypeKind` enum.\n\nDepending on the kind of a type, certain fields describe information about that type. Scalar types provide no information beyond a name, description and optional `specifiedByURL`, while Enum types provide their values. Object and Interface types provide the fields they describe. Abstract types, Union and Interface, provide the Object types possible at runtime. List and NonNull types compose other types.",
      fields: /* @__PURE__ */ __name(() => ({
        kind: {
          type: new _definition.GraphQLNonNull(__TypeKind),
          resolve(type) {
            if ((0, _definition.isScalarType)(type)) {
              return TypeKind.SCALAR;
            }
            if ((0, _definition.isObjectType)(type)) {
              return TypeKind.OBJECT;
            }
            if ((0, _definition.isInterfaceType)(type)) {
              return TypeKind.INTERFACE;
            }
            if ((0, _definition.isUnionType)(type)) {
              return TypeKind.UNION;
            }
            if ((0, _definition.isEnumType)(type)) {
              return TypeKind.ENUM;
            }
            if ((0, _definition.isInputObjectType)(type)) {
              return TypeKind.INPUT_OBJECT;
            }
            if ((0, _definition.isListType)(type)) {
              return TypeKind.LIST;
            }
            if ((0, _definition.isNonNullType)(type)) {
              return TypeKind.NON_NULL;
            }
            (0, _invariant.invariant)(
              false,
              `Unexpected type: "${(0, _inspect.inspect)(type)}".`
            );
          }
        },
        name: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((type) => "name" in type ? type.name : void 0, "resolve")
        },
        description: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((type) => (
            /* c8 ignore next */
            "description" in type ? type.description : void 0
          ), "resolve")
        },
        specifiedByURL: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((obj) => "specifiedByURL" in obj ? obj.specifiedByURL : void 0, "resolve")
        },
        fields: {
          type: new _definition.GraphQLList(
            new _definition.GraphQLNonNull(__Field)
          ),
          args: {
            includeDeprecated: {
              type: _scalars.GraphQLBoolean,
              defaultValue: false
            }
          },
          resolve(type, { includeDeprecated }) {
            if ((0, _definition.isObjectType)(type) || (0, _definition.isInterfaceType)(type)) {
              const fields = Object.values(type.getFields());
              return includeDeprecated ? fields : fields.filter((field) => field.deprecationReason == null);
            }
          }
        },
        interfaces: {
          type: new _definition.GraphQLList(new _definition.GraphQLNonNull(__Type)),
          resolve(type) {
            if ((0, _definition.isObjectType)(type) || (0, _definition.isInterfaceType)(type)) {
              return type.getInterfaces();
            }
          }
        },
        possibleTypes: {
          type: new _definition.GraphQLList(new _definition.GraphQLNonNull(__Type)),
          resolve(type, _args, _context, { schema }) {
            if ((0, _definition.isAbstractType)(type)) {
              return schema.getPossibleTypes(type);
            }
          }
        },
        enumValues: {
          type: new _definition.GraphQLList(
            new _definition.GraphQLNonNull(__EnumValue)
          ),
          args: {
            includeDeprecated: {
              type: _scalars.GraphQLBoolean,
              defaultValue: false
            }
          },
          resolve(type, { includeDeprecated }) {
            if ((0, _definition.isEnumType)(type)) {
              const values = type.getValues();
              return includeDeprecated ? values : values.filter((field) => field.deprecationReason == null);
            }
          }
        },
        inputFields: {
          type: new _definition.GraphQLList(
            new _definition.GraphQLNonNull(__InputValue)
          ),
          args: {
            includeDeprecated: {
              type: _scalars.GraphQLBoolean,
              defaultValue: false
            }
          },
          resolve(type, { includeDeprecated }) {
            if ((0, _definition.isInputObjectType)(type)) {
              const values = Object.values(type.getFields());
              return includeDeprecated ? values : values.filter((field) => field.deprecationReason == null);
            }
          }
        },
        ofType: {
          type: __Type,
          resolve: /* @__PURE__ */ __name((type) => "ofType" in type ? type.ofType : void 0, "resolve")
        },
        isOneOf: {
          type: _scalars.GraphQLBoolean,
          resolve: /* @__PURE__ */ __name((type) => {
            if ((0, _definition.isInputObjectType)(type)) {
              return type.isOneOf;
            }
          }, "resolve")
        }
      }), "fields")
    });
    exports.__Type = __Type;
    var __Field = new _definition.GraphQLObjectType({
      name: "__Field",
      description: "Object and Interface types are described by a list of Fields, each of which has a name, potentially a list of arguments, and a return type.",
      fields: /* @__PURE__ */ __name(() => ({
        name: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
          resolve: /* @__PURE__ */ __name((field) => field.name, "resolve")
        },
        description: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((field) => field.description, "resolve")
        },
        args: {
          type: new _definition.GraphQLNonNull(
            new _definition.GraphQLList(
              new _definition.GraphQLNonNull(__InputValue)
            )
          ),
          args: {
            includeDeprecated: {
              type: _scalars.GraphQLBoolean,
              defaultValue: false
            }
          },
          resolve(field, { includeDeprecated }) {
            return includeDeprecated ? field.args : field.args.filter((arg) => arg.deprecationReason == null);
          }
        },
        type: {
          type: new _definition.GraphQLNonNull(__Type),
          resolve: /* @__PURE__ */ __name((field) => field.type, "resolve")
        },
        isDeprecated: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLBoolean),
          resolve: /* @__PURE__ */ __name((field) => field.deprecationReason != null, "resolve")
        },
        deprecationReason: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((field) => field.deprecationReason, "resolve")
        }
      }), "fields")
    });
    exports.__Field = __Field;
    var __InputValue = new _definition.GraphQLObjectType({
      name: "__InputValue",
      description: "Arguments provided to Fields or Directives and the input fields of an InputObject are represented as Input Values which describe their type and optionally a default value.",
      fields: /* @__PURE__ */ __name(() => ({
        name: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
          resolve: /* @__PURE__ */ __name((inputValue) => inputValue.name, "resolve")
        },
        description: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((inputValue) => inputValue.description, "resolve")
        },
        type: {
          type: new _definition.GraphQLNonNull(__Type),
          resolve: /* @__PURE__ */ __name((inputValue) => inputValue.type, "resolve")
        },
        defaultValue: {
          type: _scalars.GraphQLString,
          description: "A GraphQL-formatted string representing the default value for this input value.",
          resolve(inputValue) {
            const { type, defaultValue } = inputValue;
            const valueAST = (0, _astFromValue.astFromValue)(defaultValue, type);
            return valueAST ? (0, _printer.print)(valueAST) : null;
          }
        },
        isDeprecated: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLBoolean),
          resolve: /* @__PURE__ */ __name((field) => field.deprecationReason != null, "resolve")
        },
        deprecationReason: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((obj) => obj.deprecationReason, "resolve")
        }
      }), "fields")
    });
    exports.__InputValue = __InputValue;
    var __EnumValue = new _definition.GraphQLObjectType({
      name: "__EnumValue",
      description: "One possible value for a given Enum. Enum values are unique values, not a placeholder for a string or numeric value. However an Enum value is returned in a JSON response as a string.",
      fields: /* @__PURE__ */ __name(() => ({
        name: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
          resolve: /* @__PURE__ */ __name((enumValue) => enumValue.name, "resolve")
        },
        description: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((enumValue) => enumValue.description, "resolve")
        },
        isDeprecated: {
          type: new _definition.GraphQLNonNull(_scalars.GraphQLBoolean),
          resolve: /* @__PURE__ */ __name((enumValue) => enumValue.deprecationReason != null, "resolve")
        },
        deprecationReason: {
          type: _scalars.GraphQLString,
          resolve: /* @__PURE__ */ __name((enumValue) => enumValue.deprecationReason, "resolve")
        }
      }), "fields")
    });
    exports.__EnumValue = __EnumValue;
    var TypeKind;
    exports.TypeKind = TypeKind;
    (function(TypeKind2) {
      TypeKind2["SCALAR"] = "SCALAR";
      TypeKind2["OBJECT"] = "OBJECT";
      TypeKind2["INTERFACE"] = "INTERFACE";
      TypeKind2["UNION"] = "UNION";
      TypeKind2["ENUM"] = "ENUM";
      TypeKind2["INPUT_OBJECT"] = "INPUT_OBJECT";
      TypeKind2["LIST"] = "LIST";
      TypeKind2["NON_NULL"] = "NON_NULL";
    })(TypeKind || (exports.TypeKind = TypeKind = {}));
    var __TypeKind = new _definition.GraphQLEnumType({
      name: "__TypeKind",
      description: "An enum describing what kind of type a given `__Type` is.",
      values: {
        SCALAR: {
          value: TypeKind.SCALAR,
          description: "Indicates this type is a scalar."
        },
        OBJECT: {
          value: TypeKind.OBJECT,
          description: "Indicates this type is an object. `fields` and `interfaces` are valid fields."
        },
        INTERFACE: {
          value: TypeKind.INTERFACE,
          description: "Indicates this type is an interface. `fields`, `interfaces`, and `possibleTypes` are valid fields."
        },
        UNION: {
          value: TypeKind.UNION,
          description: "Indicates this type is a union. `possibleTypes` is a valid field."
        },
        ENUM: {
          value: TypeKind.ENUM,
          description: "Indicates this type is an enum. `enumValues` is a valid field."
        },
        INPUT_OBJECT: {
          value: TypeKind.INPUT_OBJECT,
          description: "Indicates this type is an input object. `inputFields` is a valid field."
        },
        LIST: {
          value: TypeKind.LIST,
          description: "Indicates this type is a list. `ofType` is a valid field."
        },
        NON_NULL: {
          value: TypeKind.NON_NULL,
          description: "Indicates this type is a non-null. `ofType` is a valid field."
        }
      }
    });
    exports.__TypeKind = __TypeKind;
    var SchemaMetaFieldDef = {
      name: "__schema",
      type: new _definition.GraphQLNonNull(__Schema),
      description: "Access the current type schema of this server.",
      args: [],
      resolve: /* @__PURE__ */ __name((_source, _args, _context, { schema }) => schema, "resolve"),
      deprecationReason: void 0,
      extensions: /* @__PURE__ */ Object.create(null),
      astNode: void 0
    };
    exports.SchemaMetaFieldDef = SchemaMetaFieldDef;
    var TypeMetaFieldDef = {
      name: "__type",
      type: __Type,
      description: "Request the type information of a single type.",
      args: [
        {
          name: "name",
          description: void 0,
          type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
          defaultValue: void 0,
          deprecationReason: void 0,
          extensions: /* @__PURE__ */ Object.create(null),
          astNode: void 0
        }
      ],
      resolve: /* @__PURE__ */ __name((_source, { name }, _context, { schema }) => schema.getType(name), "resolve"),
      deprecationReason: void 0,
      extensions: /* @__PURE__ */ Object.create(null),
      astNode: void 0
    };
    exports.TypeMetaFieldDef = TypeMetaFieldDef;
    var TypeNameMetaFieldDef = {
      name: "__typename",
      type: new _definition.GraphQLNonNull(_scalars.GraphQLString),
      description: "The name of the current Object type at runtime.",
      args: [],
      resolve: /* @__PURE__ */ __name((_source, _args, _context, { parentType }) => parentType.name, "resolve"),
      deprecationReason: void 0,
      extensions: /* @__PURE__ */ Object.create(null),
      astNode: void 0
    };
    exports.TypeNameMetaFieldDef = TypeNameMetaFieldDef;
    var introspectionTypes = Object.freeze([
      __Schema,
      __Directive,
      __DirectiveLocation,
      __Type,
      __Field,
      __InputValue,
      __EnumValue,
      __TypeKind
    ]);
    exports.introspectionTypes = introspectionTypes;
    function isIntrospectionType(type) {
      return introspectionTypes.some(({ name }) => type.name === name);
    }
    __name(isIntrospectionType, "isIntrospectionType");
  }
});

// ../../node_modules/graphql/type/schema.js
var require_schema = __commonJS({
  "../../node_modules/graphql/type/schema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.GraphQLSchema = void 0;
    exports.assertSchema = assertSchema;
    exports.isSchema = isSchema;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _instanceOf = require_instanceOf();
    var _isObjectLike = require_isObjectLike();
    var _toObjMap = require_toObjMap();
    var _ast = require_ast();
    var _definition = require_definition();
    var _directives = require_directives();
    var _introspection = require_introspection();
    function isSchema(schema) {
      return (0, _instanceOf.instanceOf)(schema, GraphQLSchema);
    }
    __name(isSchema, "isSchema");
    function assertSchema(schema) {
      if (!isSchema(schema)) {
        throw new Error(
          `Expected ${(0, _inspect.inspect)(schema)} to be a GraphQL schema.`
        );
      }
      return schema;
    }
    __name(assertSchema, "assertSchema");
    var GraphQLSchema = class {
      static {
        __name(this, "GraphQLSchema");
      }
      // Used as a cache for validateSchema().
      constructor(config) {
        var _config$extensionASTN, _config$directives;
        this.__validationErrors = config.assumeValid === true ? [] : void 0;
        (0, _isObjectLike.isObjectLike)(config) || (0, _devAssert.devAssert)(false, "Must provide configuration object.");
        !config.types || Array.isArray(config.types) || (0, _devAssert.devAssert)(
          false,
          `"types" must be Array if provided but got: ${(0, _inspect.inspect)(
            config.types
          )}.`
        );
        !config.directives || Array.isArray(config.directives) || (0, _devAssert.devAssert)(
          false,
          `"directives" must be Array if provided but got: ${(0, _inspect.inspect)(config.directives)}.`
        );
        this.description = config.description;
        this.extensions = (0, _toObjMap.toObjMap)(config.extensions);
        this.astNode = config.astNode;
        this.extensionASTNodes = (_config$extensionASTN = config.extensionASTNodes) !== null && _config$extensionASTN !== void 0 ? _config$extensionASTN : [];
        this._queryType = config.query;
        this._mutationType = config.mutation;
        this._subscriptionType = config.subscription;
        this._directives = (_config$directives = config.directives) !== null && _config$directives !== void 0 ? _config$directives : _directives.specifiedDirectives;
        const allReferencedTypes = new Set(config.types);
        if (config.types != null) {
          for (const type of config.types) {
            allReferencedTypes.delete(type);
            collectReferencedTypes(type, allReferencedTypes);
          }
        }
        if (this._queryType != null) {
          collectReferencedTypes(this._queryType, allReferencedTypes);
        }
        if (this._mutationType != null) {
          collectReferencedTypes(this._mutationType, allReferencedTypes);
        }
        if (this._subscriptionType != null) {
          collectReferencedTypes(this._subscriptionType, allReferencedTypes);
        }
        for (const directive of this._directives) {
          if ((0, _directives.isDirective)(directive)) {
            for (const arg of directive.args) {
              collectReferencedTypes(arg.type, allReferencedTypes);
            }
          }
        }
        collectReferencedTypes(_introspection.__Schema, allReferencedTypes);
        this._typeMap = /* @__PURE__ */ Object.create(null);
        this._subTypeMap = /* @__PURE__ */ Object.create(null);
        this._implementationsMap = /* @__PURE__ */ Object.create(null);
        for (const namedType of allReferencedTypes) {
          if (namedType == null) {
            continue;
          }
          const typeName = namedType.name;
          typeName || (0, _devAssert.devAssert)(
            false,
            "One of the provided types for building the Schema is missing a name."
          );
          if (this._typeMap[typeName] !== void 0) {
            throw new Error(
              `Schema must contain uniquely named types but contains multiple types named "${typeName}".`
            );
          }
          this._typeMap[typeName] = namedType;
          if ((0, _definition.isInterfaceType)(namedType)) {
            for (const iface of namedType.getInterfaces()) {
              if ((0, _definition.isInterfaceType)(iface)) {
                let implementations = this._implementationsMap[iface.name];
                if (implementations === void 0) {
                  implementations = this._implementationsMap[iface.name] = {
                    objects: [],
                    interfaces: []
                  };
                }
                implementations.interfaces.push(namedType);
              }
            }
          } else if ((0, _definition.isObjectType)(namedType)) {
            for (const iface of namedType.getInterfaces()) {
              if ((0, _definition.isInterfaceType)(iface)) {
                let implementations = this._implementationsMap[iface.name];
                if (implementations === void 0) {
                  implementations = this._implementationsMap[iface.name] = {
                    objects: [],
                    interfaces: []
                  };
                }
                implementations.objects.push(namedType);
              }
            }
          }
        }
      }
      get [Symbol.toStringTag]() {
        return "GraphQLSchema";
      }
      getQueryType() {
        return this._queryType;
      }
      getMutationType() {
        return this._mutationType;
      }
      getSubscriptionType() {
        return this._subscriptionType;
      }
      getRootType(operation) {
        switch (operation) {
          case _ast.OperationTypeNode.QUERY:
            return this.getQueryType();
          case _ast.OperationTypeNode.MUTATION:
            return this.getMutationType();
          case _ast.OperationTypeNode.SUBSCRIPTION:
            return this.getSubscriptionType();
        }
      }
      getTypeMap() {
        return this._typeMap;
      }
      getType(name) {
        return this.getTypeMap()[name];
      }
      getPossibleTypes(abstractType) {
        return (0, _definition.isUnionType)(abstractType) ? abstractType.getTypes() : this.getImplementations(abstractType).objects;
      }
      getImplementations(interfaceType) {
        const implementations = this._implementationsMap[interfaceType.name];
        return implementations !== null && implementations !== void 0 ? implementations : {
          objects: [],
          interfaces: []
        };
      }
      isSubType(abstractType, maybeSubType) {
        let map = this._subTypeMap[abstractType.name];
        if (map === void 0) {
          map = /* @__PURE__ */ Object.create(null);
          if ((0, _definition.isUnionType)(abstractType)) {
            for (const type of abstractType.getTypes()) {
              map[type.name] = true;
            }
          } else {
            const implementations = this.getImplementations(abstractType);
            for (const type of implementations.objects) {
              map[type.name] = true;
            }
            for (const type of implementations.interfaces) {
              map[type.name] = true;
            }
          }
          this._subTypeMap[abstractType.name] = map;
        }
        return map[maybeSubType.name] !== void 0;
      }
      getDirectives() {
        return this._directives;
      }
      getDirective(name) {
        return this.getDirectives().find((directive) => directive.name === name);
      }
      toConfig() {
        return {
          description: this.description,
          query: this.getQueryType(),
          mutation: this.getMutationType(),
          subscription: this.getSubscriptionType(),
          types: Object.values(this.getTypeMap()),
          directives: this.getDirectives(),
          extensions: this.extensions,
          astNode: this.astNode,
          extensionASTNodes: this.extensionASTNodes,
          assumeValid: this.__validationErrors !== void 0
        };
      }
    };
    exports.GraphQLSchema = GraphQLSchema;
    function collectReferencedTypes(type, typeSet) {
      const namedType = (0, _definition.getNamedType)(type);
      if (!typeSet.has(namedType)) {
        typeSet.add(namedType);
        if ((0, _definition.isUnionType)(namedType)) {
          for (const memberType of namedType.getTypes()) {
            collectReferencedTypes(memberType, typeSet);
          }
        } else if ((0, _definition.isObjectType)(namedType) || (0, _definition.isInterfaceType)(namedType)) {
          for (const interfaceType of namedType.getInterfaces()) {
            collectReferencedTypes(interfaceType, typeSet);
          }
          for (const field of Object.values(namedType.getFields())) {
            collectReferencedTypes(field.type, typeSet);
            for (const arg of field.args) {
              collectReferencedTypes(arg.type, typeSet);
            }
          }
        } else if ((0, _definition.isInputObjectType)(namedType)) {
          for (const field of Object.values(namedType.getFields())) {
            collectReferencedTypes(field.type, typeSet);
          }
        }
      }
      return typeSet;
    }
    __name(collectReferencedTypes, "collectReferencedTypes");
  }
});

// ../../node_modules/graphql/type/validate.js
var require_validate = __commonJS({
  "../../node_modules/graphql/type/validate.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.assertValidSchema = assertValidSchema;
    exports.validateSchema = validateSchema;
    var _inspect = require_inspect();
    var _GraphQLError = require_GraphQLError();
    var _ast = require_ast();
    var _typeComparators = require_typeComparators();
    var _definition = require_definition();
    var _directives = require_directives();
    var _introspection = require_introspection();
    var _schema = require_schema();
    function validateSchema(schema) {
      (0, _schema.assertSchema)(schema);
      if (schema.__validationErrors) {
        return schema.__validationErrors;
      }
      const context = new SchemaValidationContext(schema);
      validateRootTypes(context);
      validateDirectives(context);
      validateTypes(context);
      const errors = context.getErrors();
      schema.__validationErrors = errors;
      return errors;
    }
    __name(validateSchema, "validateSchema");
    function assertValidSchema(schema) {
      const errors = validateSchema(schema);
      if (errors.length !== 0) {
        throw new Error(errors.map((error) => error.message).join("\n\n"));
      }
    }
    __name(assertValidSchema, "assertValidSchema");
    var SchemaValidationContext = class {
      static {
        __name(this, "SchemaValidationContext");
      }
      constructor(schema) {
        this._errors = [];
        this.schema = schema;
      }
      reportError(message, nodes) {
        const _nodes = Array.isArray(nodes) ? nodes.filter(Boolean) : nodes;
        this._errors.push(
          new _GraphQLError.GraphQLError(message, {
            nodes: _nodes
          })
        );
      }
      getErrors() {
        return this._errors;
      }
    };
    function validateRootTypes(context) {
      const schema = context.schema;
      const queryType = schema.getQueryType();
      if (!queryType) {
        context.reportError("Query root type must be provided.", schema.astNode);
      } else if (!(0, _definition.isObjectType)(queryType)) {
        var _getOperationTypeNode;
        context.reportError(
          `Query root type must be Object type, it cannot be ${(0, _inspect.inspect)(queryType)}.`,
          (_getOperationTypeNode = getOperationTypeNode(
            schema,
            _ast.OperationTypeNode.QUERY
          )) !== null && _getOperationTypeNode !== void 0 ? _getOperationTypeNode : queryType.astNode
        );
      }
      const mutationType = schema.getMutationType();
      if (mutationType && !(0, _definition.isObjectType)(mutationType)) {
        var _getOperationTypeNode2;
        context.reportError(
          `Mutation root type must be Object type if provided, it cannot be ${(0, _inspect.inspect)(mutationType)}.`,
          (_getOperationTypeNode2 = getOperationTypeNode(
            schema,
            _ast.OperationTypeNode.MUTATION
          )) !== null && _getOperationTypeNode2 !== void 0 ? _getOperationTypeNode2 : mutationType.astNode
        );
      }
      const subscriptionType = schema.getSubscriptionType();
      if (subscriptionType && !(0, _definition.isObjectType)(subscriptionType)) {
        var _getOperationTypeNode3;
        context.reportError(
          `Subscription root type must be Object type if provided, it cannot be ${(0, _inspect.inspect)(subscriptionType)}.`,
          (_getOperationTypeNode3 = getOperationTypeNode(
            schema,
            _ast.OperationTypeNode.SUBSCRIPTION
          )) !== null && _getOperationTypeNode3 !== void 0 ? _getOperationTypeNode3 : subscriptionType.astNode
        );
      }
    }
    __name(validateRootTypes, "validateRootTypes");
    function getOperationTypeNode(schema, operation) {
      var _flatMap$find;
      return (_flatMap$find = [schema.astNode, ...schema.extensionASTNodes].flatMap(
        // FIXME: https://github.com/graphql/graphql-js/issues/2203
        (schemaNode) => {
          var _schemaNode$operation;
          return (
            /* c8 ignore next */
            (_schemaNode$operation = schemaNode === null || schemaNode === void 0 ? void 0 : schemaNode.operationTypes) !== null && _schemaNode$operation !== void 0 ? _schemaNode$operation : []
          );
        }
      ).find((operationNode) => operationNode.operation === operation)) === null || _flatMap$find === void 0 ? void 0 : _flatMap$find.type;
    }
    __name(getOperationTypeNode, "getOperationTypeNode");
    function validateDirectives(context) {
      for (const directive of context.schema.getDirectives()) {
        if (!(0, _directives.isDirective)(directive)) {
          context.reportError(
            `Expected directive but got: ${(0, _inspect.inspect)(directive)}.`,
            directive === null || directive === void 0 ? void 0 : directive.astNode
          );
          continue;
        }
        validateName(context, directive);
        if (directive.locations.length === 0) {
          context.reportError(
            `Directive @${directive.name} must include 1 or more locations.`,
            directive.astNode
          );
        }
        for (const arg of directive.args) {
          validateName(context, arg);
          if (!(0, _definition.isInputType)(arg.type)) {
            context.reportError(
              `The type of @${directive.name}(${arg.name}:) must be Input Type but got: ${(0, _inspect.inspect)(arg.type)}.`,
              arg.astNode
            );
          }
          if ((0, _definition.isRequiredArgument)(arg) && arg.deprecationReason != null) {
            var _arg$astNode;
            context.reportError(
              `Required argument @${directive.name}(${arg.name}:) cannot be deprecated.`,
              [
                getDeprecatedDirectiveNode(arg.astNode),
                (_arg$astNode = arg.astNode) === null || _arg$astNode === void 0 ? void 0 : _arg$astNode.type
              ]
            );
          }
        }
      }
    }
    __name(validateDirectives, "validateDirectives");
    function validateName(context, node) {
      if (node.name.startsWith("__")) {
        context.reportError(
          `Name "${node.name}" must not begin with "__", which is reserved by GraphQL introspection.`,
          node.astNode
        );
      }
    }
    __name(validateName, "validateName");
    function validateTypes(context) {
      const validateInputObjectCircularRefs = createInputObjectCircularRefsValidator(context);
      const typeMap = context.schema.getTypeMap();
      for (const type of Object.values(typeMap)) {
        if (!(0, _definition.isNamedType)(type)) {
          context.reportError(
            `Expected GraphQL named type but got: ${(0, _inspect.inspect)(type)}.`,
            type.astNode
          );
          continue;
        }
        if (!(0, _introspection.isIntrospectionType)(type)) {
          validateName(context, type);
        }
        if ((0, _definition.isObjectType)(type)) {
          validateFields(context, type);
          validateInterfaces(context, type);
        } else if ((0, _definition.isInterfaceType)(type)) {
          validateFields(context, type);
          validateInterfaces(context, type);
        } else if ((0, _definition.isUnionType)(type)) {
          validateUnionMembers(context, type);
        } else if ((0, _definition.isEnumType)(type)) {
          validateEnumValues(context, type);
        } else if ((0, _definition.isInputObjectType)(type)) {
          validateInputFields(context, type);
          validateInputObjectCircularRefs(type);
        }
      }
    }
    __name(validateTypes, "validateTypes");
    function validateFields(context, type) {
      const fields = Object.values(type.getFields());
      if (fields.length === 0) {
        context.reportError(`Type ${type.name} must define one or more fields.`, [
          type.astNode,
          ...type.extensionASTNodes
        ]);
      }
      for (const field of fields) {
        validateName(context, field);
        if (!(0, _definition.isOutputType)(field.type)) {
          var _field$astNode;
          context.reportError(
            `The type of ${type.name}.${field.name} must be Output Type but got: ${(0, _inspect.inspect)(field.type)}.`,
            (_field$astNode = field.astNode) === null || _field$astNode === void 0 ? void 0 : _field$astNode.type
          );
        }
        for (const arg of field.args) {
          const argName = arg.name;
          validateName(context, arg);
          if (!(0, _definition.isInputType)(arg.type)) {
            var _arg$astNode2;
            context.reportError(
              `The type of ${type.name}.${field.name}(${argName}:) must be Input Type but got: ${(0, _inspect.inspect)(arg.type)}.`,
              (_arg$astNode2 = arg.astNode) === null || _arg$astNode2 === void 0 ? void 0 : _arg$astNode2.type
            );
          }
          if ((0, _definition.isRequiredArgument)(arg) && arg.deprecationReason != null) {
            var _arg$astNode3;
            context.reportError(
              `Required argument ${type.name}.${field.name}(${argName}:) cannot be deprecated.`,
              [
                getDeprecatedDirectiveNode(arg.astNode),
                (_arg$astNode3 = arg.astNode) === null || _arg$astNode3 === void 0 ? void 0 : _arg$astNode3.type
              ]
            );
          }
        }
      }
    }
    __name(validateFields, "validateFields");
    function validateInterfaces(context, type) {
      const ifaceTypeNames = /* @__PURE__ */ Object.create(null);
      for (const iface of type.getInterfaces()) {
        if (!(0, _definition.isInterfaceType)(iface)) {
          context.reportError(
            `Type ${(0, _inspect.inspect)(
              type
            )} must only implement Interface types, it cannot implement ${(0, _inspect.inspect)(iface)}.`,
            getAllImplementsInterfaceNodes(type, iface)
          );
          continue;
        }
        if (type === iface) {
          context.reportError(
            `Type ${type.name} cannot implement itself because it would create a circular reference.`,
            getAllImplementsInterfaceNodes(type, iface)
          );
          continue;
        }
        if (ifaceTypeNames[iface.name]) {
          context.reportError(
            `Type ${type.name} can only implement ${iface.name} once.`,
            getAllImplementsInterfaceNodes(type, iface)
          );
          continue;
        }
        ifaceTypeNames[iface.name] = true;
        validateTypeImplementsAncestors(context, type, iface);
        validateTypeImplementsInterface(context, type, iface);
      }
    }
    __name(validateInterfaces, "validateInterfaces");
    function validateTypeImplementsInterface(context, type, iface) {
      const typeFieldMap = type.getFields();
      for (const ifaceField of Object.values(iface.getFields())) {
        const fieldName = ifaceField.name;
        const typeField = typeFieldMap[fieldName];
        if (!typeField) {
          context.reportError(
            `Interface field ${iface.name}.${fieldName} expected but ${type.name} does not provide it.`,
            [ifaceField.astNode, type.astNode, ...type.extensionASTNodes]
          );
          continue;
        }
        if (!(0, _typeComparators.isTypeSubTypeOf)(
          context.schema,
          typeField.type,
          ifaceField.type
        )) {
          var _ifaceField$astNode, _typeField$astNode;
          context.reportError(
            `Interface field ${iface.name}.${fieldName} expects type ${(0, _inspect.inspect)(ifaceField.type)} but ${type.name}.${fieldName} is type ${(0, _inspect.inspect)(typeField.type)}.`,
            [
              (_ifaceField$astNode = ifaceField.astNode) === null || _ifaceField$astNode === void 0 ? void 0 : _ifaceField$astNode.type,
              (_typeField$astNode = typeField.astNode) === null || _typeField$astNode === void 0 ? void 0 : _typeField$astNode.type
            ]
          );
        }
        for (const ifaceArg of ifaceField.args) {
          const argName = ifaceArg.name;
          const typeArg = typeField.args.find((arg) => arg.name === argName);
          if (!typeArg) {
            context.reportError(
              `Interface field argument ${iface.name}.${fieldName}(${argName}:) expected but ${type.name}.${fieldName} does not provide it.`,
              [ifaceArg.astNode, typeField.astNode]
            );
            continue;
          }
          if (!(0, _typeComparators.isEqualType)(ifaceArg.type, typeArg.type)) {
            var _ifaceArg$astNode, _typeArg$astNode;
            context.reportError(
              `Interface field argument ${iface.name}.${fieldName}(${argName}:) expects type ${(0, _inspect.inspect)(ifaceArg.type)} but ${type.name}.${fieldName}(${argName}:) is type ${(0, _inspect.inspect)(typeArg.type)}.`,
              [
                (_ifaceArg$astNode = ifaceArg.astNode) === null || _ifaceArg$astNode === void 0 ? void 0 : _ifaceArg$astNode.type,
                (_typeArg$astNode = typeArg.astNode) === null || _typeArg$astNode === void 0 ? void 0 : _typeArg$astNode.type
              ]
            );
          }
        }
        for (const typeArg of typeField.args) {
          const argName = typeArg.name;
          const ifaceArg = ifaceField.args.find((arg) => arg.name === argName);
          if (!ifaceArg && (0, _definition.isRequiredArgument)(typeArg)) {
            context.reportError(
              `Object field ${type.name}.${fieldName} includes required argument ${argName} that is missing from the Interface field ${iface.name}.${fieldName}.`,
              [typeArg.astNode, ifaceField.astNode]
            );
          }
        }
      }
    }
    __name(validateTypeImplementsInterface, "validateTypeImplementsInterface");
    function validateTypeImplementsAncestors(context, type, iface) {
      const ifaceInterfaces = type.getInterfaces();
      for (const transitive of iface.getInterfaces()) {
        if (!ifaceInterfaces.includes(transitive)) {
          context.reportError(
            transitive === type ? `Type ${type.name} cannot implement ${iface.name} because it would create a circular reference.` : `Type ${type.name} must implement ${transitive.name} because it is implemented by ${iface.name}.`,
            [
              ...getAllImplementsInterfaceNodes(iface, transitive),
              ...getAllImplementsInterfaceNodes(type, iface)
            ]
          );
        }
      }
    }
    __name(validateTypeImplementsAncestors, "validateTypeImplementsAncestors");
    function validateUnionMembers(context, union) {
      const memberTypes = union.getTypes();
      if (memberTypes.length === 0) {
        context.reportError(
          `Union type ${union.name} must define one or more member types.`,
          [union.astNode, ...union.extensionASTNodes]
        );
      }
      const includedTypeNames = /* @__PURE__ */ Object.create(null);
      for (const memberType of memberTypes) {
        if (includedTypeNames[memberType.name]) {
          context.reportError(
            `Union type ${union.name} can only include type ${memberType.name} once.`,
            getUnionMemberTypeNodes(union, memberType.name)
          );
          continue;
        }
        includedTypeNames[memberType.name] = true;
        if (!(0, _definition.isObjectType)(memberType)) {
          context.reportError(
            `Union type ${union.name} can only include Object types, it cannot include ${(0, _inspect.inspect)(memberType)}.`,
            getUnionMemberTypeNodes(union, String(memberType))
          );
        }
      }
    }
    __name(validateUnionMembers, "validateUnionMembers");
    function validateEnumValues(context, enumType) {
      const enumValues = enumType.getValues();
      if (enumValues.length === 0) {
        context.reportError(
          `Enum type ${enumType.name} must define one or more values.`,
          [enumType.astNode, ...enumType.extensionASTNodes]
        );
      }
      for (const enumValue of enumValues) {
        validateName(context, enumValue);
      }
    }
    __name(validateEnumValues, "validateEnumValues");
    function validateInputFields(context, inputObj) {
      const fields = Object.values(inputObj.getFields());
      if (fields.length === 0) {
        context.reportError(
          `Input Object type ${inputObj.name} must define one or more fields.`,
          [inputObj.astNode, ...inputObj.extensionASTNodes]
        );
      }
      for (const field of fields) {
        validateName(context, field);
        if (!(0, _definition.isInputType)(field.type)) {
          var _field$astNode2;
          context.reportError(
            `The type of ${inputObj.name}.${field.name} must be Input Type but got: ${(0, _inspect.inspect)(field.type)}.`,
            (_field$astNode2 = field.astNode) === null || _field$astNode2 === void 0 ? void 0 : _field$astNode2.type
          );
        }
        if ((0, _definition.isRequiredInputField)(field) && field.deprecationReason != null) {
          var _field$astNode3;
          context.reportError(
            `Required input field ${inputObj.name}.${field.name} cannot be deprecated.`,
            [
              getDeprecatedDirectiveNode(field.astNode),
              (_field$astNode3 = field.astNode) === null || _field$astNode3 === void 0 ? void 0 : _field$astNode3.type
            ]
          );
        }
        if (inputObj.isOneOf) {
          validateOneOfInputObjectField(inputObj, field, context);
        }
      }
    }
    __name(validateInputFields, "validateInputFields");
    function validateOneOfInputObjectField(type, field, context) {
      if ((0, _definition.isNonNullType)(field.type)) {
        var _field$astNode4;
        context.reportError(
          `OneOf input field ${type.name}.${field.name} must be nullable.`,
          (_field$astNode4 = field.astNode) === null || _field$astNode4 === void 0 ? void 0 : _field$astNode4.type
        );
      }
      if (field.defaultValue !== void 0) {
        context.reportError(
          `OneOf input field ${type.name}.${field.name} cannot have a default value.`,
          field.astNode
        );
      }
    }
    __name(validateOneOfInputObjectField, "validateOneOfInputObjectField");
    function createInputObjectCircularRefsValidator(context) {
      const visitedTypes = /* @__PURE__ */ Object.create(null);
      const fieldPath = [];
      const fieldPathIndexByTypeName = /* @__PURE__ */ Object.create(null);
      return detectCycleRecursive;
      function detectCycleRecursive(inputObj) {
        if (visitedTypes[inputObj.name]) {
          return;
        }
        visitedTypes[inputObj.name] = true;
        fieldPathIndexByTypeName[inputObj.name] = fieldPath.length;
        const fields = Object.values(inputObj.getFields());
        for (const field of fields) {
          if ((0, _definition.isNonNullType)(field.type) && (0, _definition.isInputObjectType)(field.type.ofType)) {
            const fieldType = field.type.ofType;
            const cycleIndex = fieldPathIndexByTypeName[fieldType.name];
            fieldPath.push(field);
            if (cycleIndex === void 0) {
              detectCycleRecursive(fieldType);
            } else {
              const cyclePath = fieldPath.slice(cycleIndex);
              const pathStr = cyclePath.map((fieldObj) => fieldObj.name).join(".");
              context.reportError(
                `Cannot reference Input Object "${fieldType.name}" within itself through a series of non-null fields: "${pathStr}".`,
                cyclePath.map((fieldObj) => fieldObj.astNode)
              );
            }
            fieldPath.pop();
          }
        }
        fieldPathIndexByTypeName[inputObj.name] = void 0;
      }
      __name(detectCycleRecursive, "detectCycleRecursive");
    }
    __name(createInputObjectCircularRefsValidator, "createInputObjectCircularRefsValidator");
    function getAllImplementsInterfaceNodes(type, iface) {
      const { astNode, extensionASTNodes } = type;
      const nodes = astNode != null ? [astNode, ...extensionASTNodes] : extensionASTNodes;
      return nodes.flatMap((typeNode) => {
        var _typeNode$interfaces;
        return (
          /* c8 ignore next */
          (_typeNode$interfaces = typeNode.interfaces) !== null && _typeNode$interfaces !== void 0 ? _typeNode$interfaces : []
        );
      }).filter((ifaceNode) => ifaceNode.name.value === iface.name);
    }
    __name(getAllImplementsInterfaceNodes, "getAllImplementsInterfaceNodes");
    function getUnionMemberTypeNodes(union, typeName) {
      const { astNode, extensionASTNodes } = union;
      const nodes = astNode != null ? [astNode, ...extensionASTNodes] : extensionASTNodes;
      return nodes.flatMap((unionNode) => {
        var _unionNode$types;
        return (
          /* c8 ignore next */
          (_unionNode$types = unionNode.types) !== null && _unionNode$types !== void 0 ? _unionNode$types : []
        );
      }).filter((typeNode) => typeNode.name.value === typeName);
    }
    __name(getUnionMemberTypeNodes, "getUnionMemberTypeNodes");
    function getDeprecatedDirectiveNode(definitionNode) {
      var _definitionNode$direc;
      return definitionNode === null || definitionNode === void 0 ? void 0 : (_definitionNode$direc = definitionNode.directives) === null || _definitionNode$direc === void 0 ? void 0 : _definitionNode$direc.find(
        (node) => node.name.value === _directives.GraphQLDeprecatedDirective.name
      );
    }
    __name(getDeprecatedDirectiveNode, "getDeprecatedDirectiveNode");
  }
});

// ../../node_modules/graphql/utilities/typeFromAST.js
var require_typeFromAST = __commonJS({
  "../../node_modules/graphql/utilities/typeFromAST.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.typeFromAST = typeFromAST;
    var _kinds = require_kinds();
    var _definition = require_definition();
    function typeFromAST(schema, typeNode) {
      switch (typeNode.kind) {
        case _kinds.Kind.LIST_TYPE: {
          const innerType = typeFromAST(schema, typeNode.type);
          return innerType && new _definition.GraphQLList(innerType);
        }
        case _kinds.Kind.NON_NULL_TYPE: {
          const innerType = typeFromAST(schema, typeNode.type);
          return innerType && new _definition.GraphQLNonNull(innerType);
        }
        case _kinds.Kind.NAMED_TYPE:
          return schema.getType(typeNode.name.value);
      }
    }
    __name(typeFromAST, "typeFromAST");
  }
});

// ../../node_modules/graphql/utilities/TypeInfo.js
var require_TypeInfo = __commonJS({
  "../../node_modules/graphql/utilities/TypeInfo.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.TypeInfo = void 0;
    exports.visitWithTypeInfo = visitWithTypeInfo;
    var _ast = require_ast();
    var _kinds = require_kinds();
    var _visitor = require_visitor();
    var _definition = require_definition();
    var _introspection = require_introspection();
    var _typeFromAST = require_typeFromAST();
    var TypeInfo = class {
      static {
        __name(this, "TypeInfo");
      }
      constructor(schema, initialType, getFieldDefFn) {
        this._schema = schema;
        this._typeStack = [];
        this._parentTypeStack = [];
        this._inputTypeStack = [];
        this._fieldDefStack = [];
        this._defaultValueStack = [];
        this._directive = null;
        this._argument = null;
        this._enumValue = null;
        this._getFieldDef = getFieldDefFn !== null && getFieldDefFn !== void 0 ? getFieldDefFn : getFieldDef;
        if (initialType) {
          if ((0, _definition.isInputType)(initialType)) {
            this._inputTypeStack.push(initialType);
          }
          if ((0, _definition.isCompositeType)(initialType)) {
            this._parentTypeStack.push(initialType);
          }
          if ((0, _definition.isOutputType)(initialType)) {
            this._typeStack.push(initialType);
          }
        }
      }
      get [Symbol.toStringTag]() {
        return "TypeInfo";
      }
      getType() {
        if (this._typeStack.length > 0) {
          return this._typeStack[this._typeStack.length - 1];
        }
      }
      getParentType() {
        if (this._parentTypeStack.length > 0) {
          return this._parentTypeStack[this._parentTypeStack.length - 1];
        }
      }
      getInputType() {
        if (this._inputTypeStack.length > 0) {
          return this._inputTypeStack[this._inputTypeStack.length - 1];
        }
      }
      getParentInputType() {
        if (this._inputTypeStack.length > 1) {
          return this._inputTypeStack[this._inputTypeStack.length - 2];
        }
      }
      getFieldDef() {
        if (this._fieldDefStack.length > 0) {
          return this._fieldDefStack[this._fieldDefStack.length - 1];
        }
      }
      getDefaultValue() {
        if (this._defaultValueStack.length > 0) {
          return this._defaultValueStack[this._defaultValueStack.length - 1];
        }
      }
      getDirective() {
        return this._directive;
      }
      getArgument() {
        return this._argument;
      }
      getEnumValue() {
        return this._enumValue;
      }
      enter(node) {
        const schema = this._schema;
        switch (node.kind) {
          case _kinds.Kind.SELECTION_SET: {
            const namedType = (0, _definition.getNamedType)(this.getType());
            this._parentTypeStack.push(
              (0, _definition.isCompositeType)(namedType) ? namedType : void 0
            );
            break;
          }
          case _kinds.Kind.FIELD: {
            const parentType = this.getParentType();
            let fieldDef;
            let fieldType;
            if (parentType) {
              fieldDef = this._getFieldDef(schema, parentType, node);
              if (fieldDef) {
                fieldType = fieldDef.type;
              }
            }
            this._fieldDefStack.push(fieldDef);
            this._typeStack.push(
              (0, _definition.isOutputType)(fieldType) ? fieldType : void 0
            );
            break;
          }
          case _kinds.Kind.DIRECTIVE:
            this._directive = schema.getDirective(node.name.value);
            break;
          case _kinds.Kind.OPERATION_DEFINITION: {
            const rootType = schema.getRootType(node.operation);
            this._typeStack.push(
              (0, _definition.isObjectType)(rootType) ? rootType : void 0
            );
            break;
          }
          case _kinds.Kind.INLINE_FRAGMENT:
          case _kinds.Kind.FRAGMENT_DEFINITION: {
            const typeConditionAST = node.typeCondition;
            const outputType = typeConditionAST ? (0, _typeFromAST.typeFromAST)(schema, typeConditionAST) : (0, _definition.getNamedType)(this.getType());
            this._typeStack.push(
              (0, _definition.isOutputType)(outputType) ? outputType : void 0
            );
            break;
          }
          case _kinds.Kind.VARIABLE_DEFINITION: {
            const inputType = (0, _typeFromAST.typeFromAST)(schema, node.type);
            this._inputTypeStack.push(
              (0, _definition.isInputType)(inputType) ? inputType : void 0
            );
            break;
          }
          case _kinds.Kind.ARGUMENT: {
            var _this$getDirective;
            let argDef;
            let argType;
            const fieldOrDirective = (_this$getDirective = this.getDirective()) !== null && _this$getDirective !== void 0 ? _this$getDirective : this.getFieldDef();
            if (fieldOrDirective) {
              argDef = fieldOrDirective.args.find(
                (arg) => arg.name === node.name.value
              );
              if (argDef) {
                argType = argDef.type;
              }
            }
            this._argument = argDef;
            this._defaultValueStack.push(argDef ? argDef.defaultValue : void 0);
            this._inputTypeStack.push(
              (0, _definition.isInputType)(argType) ? argType : void 0
            );
            break;
          }
          case _kinds.Kind.LIST: {
            const listType = (0, _definition.getNullableType)(this.getInputType());
            const itemType = (0, _definition.isListType)(listType) ? listType.ofType : listType;
            this._defaultValueStack.push(void 0);
            this._inputTypeStack.push(
              (0, _definition.isInputType)(itemType) ? itemType : void 0
            );
            break;
          }
          case _kinds.Kind.OBJECT_FIELD: {
            const objectType = (0, _definition.getNamedType)(this.getInputType());
            let inputFieldType;
            let inputField;
            if ((0, _definition.isInputObjectType)(objectType)) {
              inputField = objectType.getFields()[node.name.value];
              if (inputField) {
                inputFieldType = inputField.type;
              }
            }
            this._defaultValueStack.push(
              inputField ? inputField.defaultValue : void 0
            );
            this._inputTypeStack.push(
              (0, _definition.isInputType)(inputFieldType) ? inputFieldType : void 0
            );
            break;
          }
          case _kinds.Kind.ENUM: {
            const enumType = (0, _definition.getNamedType)(this.getInputType());
            let enumValue;
            if ((0, _definition.isEnumType)(enumType)) {
              enumValue = enumType.getValue(node.value);
            }
            this._enumValue = enumValue;
            break;
          }
          default:
        }
      }
      leave(node) {
        switch (node.kind) {
          case _kinds.Kind.SELECTION_SET:
            this._parentTypeStack.pop();
            break;
          case _kinds.Kind.FIELD:
            this._fieldDefStack.pop();
            this._typeStack.pop();
            break;
          case _kinds.Kind.DIRECTIVE:
            this._directive = null;
            break;
          case _kinds.Kind.OPERATION_DEFINITION:
          case _kinds.Kind.INLINE_FRAGMENT:
          case _kinds.Kind.FRAGMENT_DEFINITION:
            this._typeStack.pop();
            break;
          case _kinds.Kind.VARIABLE_DEFINITION:
            this._inputTypeStack.pop();
            break;
          case _kinds.Kind.ARGUMENT:
            this._argument = null;
            this._defaultValueStack.pop();
            this._inputTypeStack.pop();
            break;
          case _kinds.Kind.LIST:
          case _kinds.Kind.OBJECT_FIELD:
            this._defaultValueStack.pop();
            this._inputTypeStack.pop();
            break;
          case _kinds.Kind.ENUM:
            this._enumValue = null;
            break;
          default:
        }
      }
    };
    exports.TypeInfo = TypeInfo;
    function getFieldDef(schema, parentType, fieldNode) {
      const name = fieldNode.name.value;
      if (name === _introspection.SchemaMetaFieldDef.name && schema.getQueryType() === parentType) {
        return _introspection.SchemaMetaFieldDef;
      }
      if (name === _introspection.TypeMetaFieldDef.name && schema.getQueryType() === parentType) {
        return _introspection.TypeMetaFieldDef;
      }
      if (name === _introspection.TypeNameMetaFieldDef.name && (0, _definition.isCompositeType)(parentType)) {
        return _introspection.TypeNameMetaFieldDef;
      }
      if ((0, _definition.isObjectType)(parentType) || (0, _definition.isInterfaceType)(parentType)) {
        return parentType.getFields()[name];
      }
    }
    __name(getFieldDef, "getFieldDef");
    function visitWithTypeInfo(typeInfo, visitor) {
      return {
        enter(...args) {
          const node = args[0];
          typeInfo.enter(node);
          const fn = (0, _visitor.getEnterLeaveForKind)(visitor, node.kind).enter;
          if (fn) {
            const result = fn.apply(visitor, args);
            if (result !== void 0) {
              typeInfo.leave(node);
              if ((0, _ast.isNode)(result)) {
                typeInfo.enter(result);
              }
            }
            return result;
          }
        },
        leave(...args) {
          const node = args[0];
          const fn = (0, _visitor.getEnterLeaveForKind)(visitor, node.kind).leave;
          let result;
          if (fn) {
            result = fn.apply(visitor, args);
          }
          typeInfo.leave(node);
          return result;
        }
      };
    }
    __name(visitWithTypeInfo, "visitWithTypeInfo");
  }
});

// ../../node_modules/graphql/language/predicates.js
var require_predicates = __commonJS({
  "../../node_modules/graphql/language/predicates.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.isConstValueNode = isConstValueNode;
    exports.isDefinitionNode = isDefinitionNode;
    exports.isExecutableDefinitionNode = isExecutableDefinitionNode;
    exports.isSelectionNode = isSelectionNode;
    exports.isTypeDefinitionNode = isTypeDefinitionNode;
    exports.isTypeExtensionNode = isTypeExtensionNode;
    exports.isTypeNode = isTypeNode;
    exports.isTypeSystemDefinitionNode = isTypeSystemDefinitionNode;
    exports.isTypeSystemExtensionNode = isTypeSystemExtensionNode;
    exports.isValueNode = isValueNode;
    var _kinds = require_kinds();
    function isDefinitionNode(node) {
      return isExecutableDefinitionNode(node) || isTypeSystemDefinitionNode(node) || isTypeSystemExtensionNode(node);
    }
    __name(isDefinitionNode, "isDefinitionNode");
    function isExecutableDefinitionNode(node) {
      return node.kind === _kinds.Kind.OPERATION_DEFINITION || node.kind === _kinds.Kind.FRAGMENT_DEFINITION;
    }
    __name(isExecutableDefinitionNode, "isExecutableDefinitionNode");
    function isSelectionNode(node) {
      return node.kind === _kinds.Kind.FIELD || node.kind === _kinds.Kind.FRAGMENT_SPREAD || node.kind === _kinds.Kind.INLINE_FRAGMENT;
    }
    __name(isSelectionNode, "isSelectionNode");
    function isValueNode(node) {
      return node.kind === _kinds.Kind.VARIABLE || node.kind === _kinds.Kind.INT || node.kind === _kinds.Kind.FLOAT || node.kind === _kinds.Kind.STRING || node.kind === _kinds.Kind.BOOLEAN || node.kind === _kinds.Kind.NULL || node.kind === _kinds.Kind.ENUM || node.kind === _kinds.Kind.LIST || node.kind === _kinds.Kind.OBJECT;
    }
    __name(isValueNode, "isValueNode");
    function isConstValueNode(node) {
      return isValueNode(node) && (node.kind === _kinds.Kind.LIST ? node.values.some(isConstValueNode) : node.kind === _kinds.Kind.OBJECT ? node.fields.some((field) => isConstValueNode(field.value)) : node.kind !== _kinds.Kind.VARIABLE);
    }
    __name(isConstValueNode, "isConstValueNode");
    function isTypeNode(node) {
      return node.kind === _kinds.Kind.NAMED_TYPE || node.kind === _kinds.Kind.LIST_TYPE || node.kind === _kinds.Kind.NON_NULL_TYPE;
    }
    __name(isTypeNode, "isTypeNode");
    function isTypeSystemDefinitionNode(node) {
      return node.kind === _kinds.Kind.SCHEMA_DEFINITION || isTypeDefinitionNode(node) || node.kind === _kinds.Kind.DIRECTIVE_DEFINITION;
    }
    __name(isTypeSystemDefinitionNode, "isTypeSystemDefinitionNode");
    function isTypeDefinitionNode(node) {
      return node.kind === _kinds.Kind.SCALAR_TYPE_DEFINITION || node.kind === _kinds.Kind.OBJECT_TYPE_DEFINITION || node.kind === _kinds.Kind.INTERFACE_TYPE_DEFINITION || node.kind === _kinds.Kind.UNION_TYPE_DEFINITION || node.kind === _kinds.Kind.ENUM_TYPE_DEFINITION || node.kind === _kinds.Kind.INPUT_OBJECT_TYPE_DEFINITION;
    }
    __name(isTypeDefinitionNode, "isTypeDefinitionNode");
    function isTypeSystemExtensionNode(node) {
      return node.kind === _kinds.Kind.SCHEMA_EXTENSION || isTypeExtensionNode(node);
    }
    __name(isTypeSystemExtensionNode, "isTypeSystemExtensionNode");
    function isTypeExtensionNode(node) {
      return node.kind === _kinds.Kind.SCALAR_TYPE_EXTENSION || node.kind === _kinds.Kind.OBJECT_TYPE_EXTENSION || node.kind === _kinds.Kind.INTERFACE_TYPE_EXTENSION || node.kind === _kinds.Kind.UNION_TYPE_EXTENSION || node.kind === _kinds.Kind.ENUM_TYPE_EXTENSION || node.kind === _kinds.Kind.INPUT_OBJECT_TYPE_EXTENSION;
    }
    __name(isTypeExtensionNode, "isTypeExtensionNode");
  }
});

// ../../node_modules/graphql/validation/rules/ExecutableDefinitionsRule.js
var require_ExecutableDefinitionsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/ExecutableDefinitionsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.ExecutableDefinitionsRule = ExecutableDefinitionsRule;
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _predicates = require_predicates();
    function ExecutableDefinitionsRule(context) {
      return {
        Document(node) {
          for (const definition of node.definitions) {
            if (!(0, _predicates.isExecutableDefinitionNode)(definition)) {
              const defName = definition.kind === _kinds.Kind.SCHEMA_DEFINITION || definition.kind === _kinds.Kind.SCHEMA_EXTENSION ? "schema" : '"' + definition.name.value + '"';
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `The ${defName} definition is not executable.`,
                  {
                    nodes: definition
                  }
                )
              );
            }
          }
          return false;
        }
      };
    }
    __name(ExecutableDefinitionsRule, "ExecutableDefinitionsRule");
  }
});

// ../../node_modules/graphql/validation/rules/FieldsOnCorrectTypeRule.js
var require_FieldsOnCorrectTypeRule = __commonJS({
  "../../node_modules/graphql/validation/rules/FieldsOnCorrectTypeRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.FieldsOnCorrectTypeRule = FieldsOnCorrectTypeRule;
    var _didYouMean = require_didYouMean();
    var _naturalCompare = require_naturalCompare();
    var _suggestionList = require_suggestionList();
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    function FieldsOnCorrectTypeRule(context) {
      return {
        Field(node) {
          const type = context.getParentType();
          if (type) {
            const fieldDef = context.getFieldDef();
            if (!fieldDef) {
              const schema = context.getSchema();
              const fieldName = node.name.value;
              let suggestion = (0, _didYouMean.didYouMean)(
                "to use an inline fragment on",
                getSuggestedTypeNames(schema, type, fieldName)
              );
              if (suggestion === "") {
                suggestion = (0, _didYouMean.didYouMean)(
                  getSuggestedFieldNames(type, fieldName)
                );
              }
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Cannot query field "${fieldName}" on type "${type.name}".` + suggestion,
                  {
                    nodes: node
                  }
                )
              );
            }
          }
        }
      };
    }
    __name(FieldsOnCorrectTypeRule, "FieldsOnCorrectTypeRule");
    function getSuggestedTypeNames(schema, type, fieldName) {
      if (!(0, _definition.isAbstractType)(type)) {
        return [];
      }
      const suggestedTypes = /* @__PURE__ */ new Set();
      const usageCount = /* @__PURE__ */ Object.create(null);
      for (const possibleType of schema.getPossibleTypes(type)) {
        if (!possibleType.getFields()[fieldName]) {
          continue;
        }
        suggestedTypes.add(possibleType);
        usageCount[possibleType.name] = 1;
        for (const possibleInterface of possibleType.getInterfaces()) {
          var _usageCount$possibleI;
          if (!possibleInterface.getFields()[fieldName]) {
            continue;
          }
          suggestedTypes.add(possibleInterface);
          usageCount[possibleInterface.name] = ((_usageCount$possibleI = usageCount[possibleInterface.name]) !== null && _usageCount$possibleI !== void 0 ? _usageCount$possibleI : 0) + 1;
        }
      }
      return [...suggestedTypes].sort((typeA, typeB) => {
        const usageCountDiff = usageCount[typeB.name] - usageCount[typeA.name];
        if (usageCountDiff !== 0) {
          return usageCountDiff;
        }
        if ((0, _definition.isInterfaceType)(typeA) && schema.isSubType(typeA, typeB)) {
          return -1;
        }
        if ((0, _definition.isInterfaceType)(typeB) && schema.isSubType(typeB, typeA)) {
          return 1;
        }
        return (0, _naturalCompare.naturalCompare)(typeA.name, typeB.name);
      }).map((x) => x.name);
    }
    __name(getSuggestedTypeNames, "getSuggestedTypeNames");
    function getSuggestedFieldNames(type, fieldName) {
      if ((0, _definition.isObjectType)(type) || (0, _definition.isInterfaceType)(type)) {
        const possibleFieldNames = Object.keys(type.getFields());
        return (0, _suggestionList.suggestionList)(fieldName, possibleFieldNames);
      }
      return [];
    }
    __name(getSuggestedFieldNames, "getSuggestedFieldNames");
  }
});

// ../../node_modules/graphql/validation/rules/FragmentsOnCompositeTypesRule.js
var require_FragmentsOnCompositeTypesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/FragmentsOnCompositeTypesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.FragmentsOnCompositeTypesRule = FragmentsOnCompositeTypesRule;
    var _GraphQLError = require_GraphQLError();
    var _printer = require_printer();
    var _definition = require_definition();
    var _typeFromAST = require_typeFromAST();
    function FragmentsOnCompositeTypesRule(context) {
      return {
        InlineFragment(node) {
          const typeCondition = node.typeCondition;
          if (typeCondition) {
            const type = (0, _typeFromAST.typeFromAST)(
              context.getSchema(),
              typeCondition
            );
            if (type && !(0, _definition.isCompositeType)(type)) {
              const typeStr = (0, _printer.print)(typeCondition);
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Fragment cannot condition on non composite type "${typeStr}".`,
                  {
                    nodes: typeCondition
                  }
                )
              );
            }
          }
        },
        FragmentDefinition(node) {
          const type = (0, _typeFromAST.typeFromAST)(
            context.getSchema(),
            node.typeCondition
          );
          if (type && !(0, _definition.isCompositeType)(type)) {
            const typeStr = (0, _printer.print)(node.typeCondition);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Fragment "${node.name.value}" cannot condition on non composite type "${typeStr}".`,
                {
                  nodes: node.typeCondition
                }
              )
            );
          }
        }
      };
    }
    __name(FragmentsOnCompositeTypesRule, "FragmentsOnCompositeTypesRule");
  }
});

// ../../node_modules/graphql/validation/rules/KnownArgumentNamesRule.js
var require_KnownArgumentNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/KnownArgumentNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.KnownArgumentNamesOnDirectivesRule = KnownArgumentNamesOnDirectivesRule;
    exports.KnownArgumentNamesRule = KnownArgumentNamesRule;
    var _didYouMean = require_didYouMean();
    var _suggestionList = require_suggestionList();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _directives = require_directives();
    function KnownArgumentNamesRule(context) {
      return {
        // eslint-disable-next-line new-cap
        ...KnownArgumentNamesOnDirectivesRule(context),
        Argument(argNode) {
          const argDef = context.getArgument();
          const fieldDef = context.getFieldDef();
          const parentType = context.getParentType();
          if (!argDef && fieldDef && parentType) {
            const argName = argNode.name.value;
            const knownArgsNames = fieldDef.args.map((arg) => arg.name);
            const suggestions = (0, _suggestionList.suggestionList)(
              argName,
              knownArgsNames
            );
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Unknown argument "${argName}" on field "${parentType.name}.${fieldDef.name}".` + (0, _didYouMean.didYouMean)(suggestions),
                {
                  nodes: argNode
                }
              )
            );
          }
        }
      };
    }
    __name(KnownArgumentNamesRule, "KnownArgumentNamesRule");
    function KnownArgumentNamesOnDirectivesRule(context) {
      const directiveArgs = /* @__PURE__ */ Object.create(null);
      const schema = context.getSchema();
      const definedDirectives = schema ? schema.getDirectives() : _directives.specifiedDirectives;
      for (const directive of definedDirectives) {
        directiveArgs[directive.name] = directive.args.map((arg) => arg.name);
      }
      const astDefinitions = context.getDocument().definitions;
      for (const def of astDefinitions) {
        if (def.kind === _kinds.Kind.DIRECTIVE_DEFINITION) {
          var _def$arguments;
          const argsNodes = (_def$arguments = def.arguments) !== null && _def$arguments !== void 0 ? _def$arguments : [];
          directiveArgs[def.name.value] = argsNodes.map((arg) => arg.name.value);
        }
      }
      return {
        Directive(directiveNode) {
          const directiveName = directiveNode.name.value;
          const knownArgs = directiveArgs[directiveName];
          if (directiveNode.arguments && knownArgs) {
            for (const argNode of directiveNode.arguments) {
              const argName = argNode.name.value;
              if (!knownArgs.includes(argName)) {
                const suggestions = (0, _suggestionList.suggestionList)(
                  argName,
                  knownArgs
                );
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    `Unknown argument "${argName}" on directive "@${directiveName}".` + (0, _didYouMean.didYouMean)(suggestions),
                    {
                      nodes: argNode
                    }
                  )
                );
              }
            }
          }
          return false;
        }
      };
    }
    __name(KnownArgumentNamesOnDirectivesRule, "KnownArgumentNamesOnDirectivesRule");
  }
});

// ../../node_modules/graphql/validation/rules/KnownDirectivesRule.js
var require_KnownDirectivesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/KnownDirectivesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.KnownDirectivesRule = KnownDirectivesRule;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _GraphQLError = require_GraphQLError();
    var _ast = require_ast();
    var _directiveLocation = require_directiveLocation();
    var _kinds = require_kinds();
    var _directives = require_directives();
    function KnownDirectivesRule(context) {
      const locationsMap = /* @__PURE__ */ Object.create(null);
      const schema = context.getSchema();
      const definedDirectives = schema ? schema.getDirectives() : _directives.specifiedDirectives;
      for (const directive of definedDirectives) {
        locationsMap[directive.name] = directive.locations;
      }
      const astDefinitions = context.getDocument().definitions;
      for (const def of astDefinitions) {
        if (def.kind === _kinds.Kind.DIRECTIVE_DEFINITION) {
          locationsMap[def.name.value] = def.locations.map((name) => name.value);
        }
      }
      return {
        Directive(node, _key, _parent, _path, ancestors) {
          const name = node.name.value;
          const locations = locationsMap[name];
          if (!locations) {
            context.reportError(
              new _GraphQLError.GraphQLError(`Unknown directive "@${name}".`, {
                nodes: node
              })
            );
            return;
          }
          const candidateLocation = getDirectiveLocationForASTPath(ancestors);
          if (candidateLocation && !locations.includes(candidateLocation)) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Directive "@${name}" may not be used on ${candidateLocation}.`,
                {
                  nodes: node
                }
              )
            );
          }
        }
      };
    }
    __name(KnownDirectivesRule, "KnownDirectivesRule");
    function getDirectiveLocationForASTPath(ancestors) {
      const appliedTo = ancestors[ancestors.length - 1];
      "kind" in appliedTo || (0, _invariant.invariant)(false);
      switch (appliedTo.kind) {
        case _kinds.Kind.OPERATION_DEFINITION:
          return getDirectiveLocationForOperation(appliedTo.operation);
        case _kinds.Kind.FIELD:
          return _directiveLocation.DirectiveLocation.FIELD;
        case _kinds.Kind.FRAGMENT_SPREAD:
          return _directiveLocation.DirectiveLocation.FRAGMENT_SPREAD;
        case _kinds.Kind.INLINE_FRAGMENT:
          return _directiveLocation.DirectiveLocation.INLINE_FRAGMENT;
        case _kinds.Kind.FRAGMENT_DEFINITION:
          return _directiveLocation.DirectiveLocation.FRAGMENT_DEFINITION;
        case _kinds.Kind.VARIABLE_DEFINITION:
          return _directiveLocation.DirectiveLocation.VARIABLE_DEFINITION;
        case _kinds.Kind.SCHEMA_DEFINITION:
        case _kinds.Kind.SCHEMA_EXTENSION:
          return _directiveLocation.DirectiveLocation.SCHEMA;
        case _kinds.Kind.SCALAR_TYPE_DEFINITION:
        case _kinds.Kind.SCALAR_TYPE_EXTENSION:
          return _directiveLocation.DirectiveLocation.SCALAR;
        case _kinds.Kind.OBJECT_TYPE_DEFINITION:
        case _kinds.Kind.OBJECT_TYPE_EXTENSION:
          return _directiveLocation.DirectiveLocation.OBJECT;
        case _kinds.Kind.FIELD_DEFINITION:
          return _directiveLocation.DirectiveLocation.FIELD_DEFINITION;
        case _kinds.Kind.INTERFACE_TYPE_DEFINITION:
        case _kinds.Kind.INTERFACE_TYPE_EXTENSION:
          return _directiveLocation.DirectiveLocation.INTERFACE;
        case _kinds.Kind.UNION_TYPE_DEFINITION:
        case _kinds.Kind.UNION_TYPE_EXTENSION:
          return _directiveLocation.DirectiveLocation.UNION;
        case _kinds.Kind.ENUM_TYPE_DEFINITION:
        case _kinds.Kind.ENUM_TYPE_EXTENSION:
          return _directiveLocation.DirectiveLocation.ENUM;
        case _kinds.Kind.ENUM_VALUE_DEFINITION:
          return _directiveLocation.DirectiveLocation.ENUM_VALUE;
        case _kinds.Kind.INPUT_OBJECT_TYPE_DEFINITION:
        case _kinds.Kind.INPUT_OBJECT_TYPE_EXTENSION:
          return _directiveLocation.DirectiveLocation.INPUT_OBJECT;
        case _kinds.Kind.INPUT_VALUE_DEFINITION: {
          const parentNode = ancestors[ancestors.length - 3];
          "kind" in parentNode || (0, _invariant.invariant)(false);
          return parentNode.kind === _kinds.Kind.INPUT_OBJECT_TYPE_DEFINITION ? _directiveLocation.DirectiveLocation.INPUT_FIELD_DEFINITION : _directiveLocation.DirectiveLocation.ARGUMENT_DEFINITION;
        }
        // Not reachable, all possible types have been considered.
        /* c8 ignore next */
        default:
          (0, _invariant.invariant)(
            false,
            "Unexpected kind: " + (0, _inspect.inspect)(appliedTo.kind)
          );
      }
    }
    __name(getDirectiveLocationForASTPath, "getDirectiveLocationForASTPath");
    function getDirectiveLocationForOperation(operation) {
      switch (operation) {
        case _ast.OperationTypeNode.QUERY:
          return _directiveLocation.DirectiveLocation.QUERY;
        case _ast.OperationTypeNode.MUTATION:
          return _directiveLocation.DirectiveLocation.MUTATION;
        case _ast.OperationTypeNode.SUBSCRIPTION:
          return _directiveLocation.DirectiveLocation.SUBSCRIPTION;
      }
    }
    __name(getDirectiveLocationForOperation, "getDirectiveLocationForOperation");
  }
});

// ../../node_modules/graphql/validation/rules/KnownFragmentNamesRule.js
var require_KnownFragmentNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/KnownFragmentNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.KnownFragmentNamesRule = KnownFragmentNamesRule;
    var _GraphQLError = require_GraphQLError();
    function KnownFragmentNamesRule(context) {
      return {
        FragmentSpread(node) {
          const fragmentName = node.name.value;
          const fragment = context.getFragment(fragmentName);
          if (!fragment) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Unknown fragment "${fragmentName}".`,
                {
                  nodes: node.name
                }
              )
            );
          }
        }
      };
    }
    __name(KnownFragmentNamesRule, "KnownFragmentNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/KnownTypeNamesRule.js
var require_KnownTypeNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/KnownTypeNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.KnownTypeNamesRule = KnownTypeNamesRule;
    var _didYouMean = require_didYouMean();
    var _suggestionList = require_suggestionList();
    var _GraphQLError = require_GraphQLError();
    var _predicates = require_predicates();
    var _introspection = require_introspection();
    var _scalars = require_scalars();
    function KnownTypeNamesRule(context) {
      const schema = context.getSchema();
      const existingTypesMap = schema ? schema.getTypeMap() : /* @__PURE__ */ Object.create(null);
      const definedTypes = /* @__PURE__ */ Object.create(null);
      for (const def of context.getDocument().definitions) {
        if ((0, _predicates.isTypeDefinitionNode)(def)) {
          definedTypes[def.name.value] = true;
        }
      }
      const typeNames = [
        ...Object.keys(existingTypesMap),
        ...Object.keys(definedTypes)
      ];
      return {
        NamedType(node, _1, parent, _2, ancestors) {
          const typeName = node.name.value;
          if (!existingTypesMap[typeName] && !definedTypes[typeName]) {
            var _ancestors$;
            const definitionNode = (_ancestors$ = ancestors[2]) !== null && _ancestors$ !== void 0 ? _ancestors$ : parent;
            const isSDL = definitionNode != null && isSDLNode(definitionNode);
            if (isSDL && standardTypeNames.includes(typeName)) {
              return;
            }
            const suggestedTypes = (0, _suggestionList.suggestionList)(
              typeName,
              isSDL ? standardTypeNames.concat(typeNames) : typeNames
            );
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Unknown type "${typeName}".` + (0, _didYouMean.didYouMean)(suggestedTypes),
                {
                  nodes: node
                }
              )
            );
          }
        }
      };
    }
    __name(KnownTypeNamesRule, "KnownTypeNamesRule");
    var standardTypeNames = [
      ..._scalars.specifiedScalarTypes,
      ..._introspection.introspectionTypes
    ].map((type) => type.name);
    function isSDLNode(value) {
      return "kind" in value && ((0, _predicates.isTypeSystemDefinitionNode)(value) || (0, _predicates.isTypeSystemExtensionNode)(value));
    }
    __name(isSDLNode, "isSDLNode");
  }
});

// ../../node_modules/graphql/validation/rules/LoneAnonymousOperationRule.js
var require_LoneAnonymousOperationRule = __commonJS({
  "../../node_modules/graphql/validation/rules/LoneAnonymousOperationRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.LoneAnonymousOperationRule = LoneAnonymousOperationRule;
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    function LoneAnonymousOperationRule(context) {
      let operationCount = 0;
      return {
        Document(node) {
          operationCount = node.definitions.filter(
            (definition) => definition.kind === _kinds.Kind.OPERATION_DEFINITION
          ).length;
        },
        OperationDefinition(node) {
          if (!node.name && operationCount > 1) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                "This anonymous operation must be the only defined operation.",
                {
                  nodes: node
                }
              )
            );
          }
        }
      };
    }
    __name(LoneAnonymousOperationRule, "LoneAnonymousOperationRule");
  }
});

// ../../node_modules/graphql/validation/rules/LoneSchemaDefinitionRule.js
var require_LoneSchemaDefinitionRule = __commonJS({
  "../../node_modules/graphql/validation/rules/LoneSchemaDefinitionRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.LoneSchemaDefinitionRule = LoneSchemaDefinitionRule;
    var _GraphQLError = require_GraphQLError();
    function LoneSchemaDefinitionRule(context) {
      var _ref, _ref2, _oldSchema$astNode;
      const oldSchema = context.getSchema();
      const alreadyDefined = (_ref = (_ref2 = (_oldSchema$astNode = oldSchema === null || oldSchema === void 0 ? void 0 : oldSchema.astNode) !== null && _oldSchema$astNode !== void 0 ? _oldSchema$astNode : oldSchema === null || oldSchema === void 0 ? void 0 : oldSchema.getQueryType()) !== null && _ref2 !== void 0 ? _ref2 : oldSchema === null || oldSchema === void 0 ? void 0 : oldSchema.getMutationType()) !== null && _ref !== void 0 ? _ref : oldSchema === null || oldSchema === void 0 ? void 0 : oldSchema.getSubscriptionType();
      let schemaDefinitionsCount = 0;
      return {
        SchemaDefinition(node) {
          if (alreadyDefined) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                "Cannot define a new schema within a schema extension.",
                {
                  nodes: node
                }
              )
            );
            return;
          }
          if (schemaDefinitionsCount > 0) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                "Must provide only one schema definition.",
                {
                  nodes: node
                }
              )
            );
          }
          ++schemaDefinitionsCount;
        }
      };
    }
    __name(LoneSchemaDefinitionRule, "LoneSchemaDefinitionRule");
  }
});

// ../../node_modules/graphql/validation/rules/MaxIntrospectionDepthRule.js
var require_MaxIntrospectionDepthRule = __commonJS({
  "../../node_modules/graphql/validation/rules/MaxIntrospectionDepthRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.MaxIntrospectionDepthRule = MaxIntrospectionDepthRule;
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var MAX_LISTS_DEPTH = 3;
    function MaxIntrospectionDepthRule(context) {
      function checkDepth(node, visitedFragments = /* @__PURE__ */ Object.create(null), depth = 0) {
        if (node.kind === _kinds.Kind.FRAGMENT_SPREAD) {
          const fragmentName = node.name.value;
          if (visitedFragments[fragmentName] === true) {
            return false;
          }
          const fragment = context.getFragment(fragmentName);
          if (!fragment) {
            return false;
          }
          try {
            visitedFragments[fragmentName] = true;
            return checkDepth(fragment, visitedFragments, depth);
          } finally {
            visitedFragments[fragmentName] = void 0;
          }
        }
        if (node.kind === _kinds.Kind.FIELD && // check all introspection lists
        (node.name.value === "fields" || node.name.value === "interfaces" || node.name.value === "possibleTypes" || node.name.value === "inputFields")) {
          depth++;
          if (depth >= MAX_LISTS_DEPTH) {
            return true;
          }
        }
        if ("selectionSet" in node && node.selectionSet) {
          for (const child of node.selectionSet.selections) {
            if (checkDepth(child, visitedFragments, depth)) {
              return true;
            }
          }
        }
        return false;
      }
      __name(checkDepth, "checkDepth");
      return {
        Field(node) {
          if (node.name.value === "__schema" || node.name.value === "__type") {
            if (checkDepth(node)) {
              context.reportError(
                new _GraphQLError.GraphQLError(
                  "Maximum introspection depth exceeded",
                  {
                    nodes: [node]
                  }
                )
              );
              return false;
            }
          }
        }
      };
    }
    __name(MaxIntrospectionDepthRule, "MaxIntrospectionDepthRule");
  }
});

// ../../node_modules/graphql/validation/rules/NoFragmentCyclesRule.js
var require_NoFragmentCyclesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/NoFragmentCyclesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.NoFragmentCyclesRule = NoFragmentCyclesRule;
    var _GraphQLError = require_GraphQLError();
    function NoFragmentCyclesRule(context) {
      const visitedFrags = /* @__PURE__ */ Object.create(null);
      const spreadPath = [];
      const spreadPathIndexByName = /* @__PURE__ */ Object.create(null);
      return {
        OperationDefinition: /* @__PURE__ */ __name(() => false, "OperationDefinition"),
        FragmentDefinition(node) {
          detectCycleRecursive(node);
          return false;
        }
      };
      function detectCycleRecursive(fragment) {
        if (visitedFrags[fragment.name.value]) {
          return;
        }
        const fragmentName = fragment.name.value;
        visitedFrags[fragmentName] = true;
        const spreadNodes = context.getFragmentSpreads(fragment.selectionSet);
        if (spreadNodes.length === 0) {
          return;
        }
        spreadPathIndexByName[fragmentName] = spreadPath.length;
        for (const spreadNode of spreadNodes) {
          const spreadName = spreadNode.name.value;
          const cycleIndex = spreadPathIndexByName[spreadName];
          spreadPath.push(spreadNode);
          if (cycleIndex === void 0) {
            const spreadFragment = context.getFragment(spreadName);
            if (spreadFragment) {
              detectCycleRecursive(spreadFragment);
            }
          } else {
            const cyclePath = spreadPath.slice(cycleIndex);
            const viaPath = cyclePath.slice(0, -1).map((s) => '"' + s.name.value + '"').join(", ");
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Cannot spread fragment "${spreadName}" within itself` + (viaPath !== "" ? ` via ${viaPath}.` : "."),
                {
                  nodes: cyclePath
                }
              )
            );
          }
          spreadPath.pop();
        }
        spreadPathIndexByName[fragmentName] = void 0;
      }
      __name(detectCycleRecursive, "detectCycleRecursive");
    }
    __name(NoFragmentCyclesRule, "NoFragmentCyclesRule");
  }
});

// ../../node_modules/graphql/validation/rules/NoUndefinedVariablesRule.js
var require_NoUndefinedVariablesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/NoUndefinedVariablesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.NoUndefinedVariablesRule = NoUndefinedVariablesRule;
    var _GraphQLError = require_GraphQLError();
    function NoUndefinedVariablesRule(context) {
      let variableNameDefined = /* @__PURE__ */ Object.create(null);
      return {
        OperationDefinition: {
          enter() {
            variableNameDefined = /* @__PURE__ */ Object.create(null);
          },
          leave(operation) {
            const usages = context.getRecursiveVariableUsages(operation);
            for (const { node } of usages) {
              const varName = node.name.value;
              if (variableNameDefined[varName] !== true) {
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    operation.name ? `Variable "$${varName}" is not defined by operation "${operation.name.value}".` : `Variable "$${varName}" is not defined.`,
                    {
                      nodes: [node, operation]
                    }
                  )
                );
              }
            }
          }
        },
        VariableDefinition(node) {
          variableNameDefined[node.variable.name.value] = true;
        }
      };
    }
    __name(NoUndefinedVariablesRule, "NoUndefinedVariablesRule");
  }
});

// ../../node_modules/graphql/validation/rules/NoUnusedFragmentsRule.js
var require_NoUnusedFragmentsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/NoUnusedFragmentsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.NoUnusedFragmentsRule = NoUnusedFragmentsRule;
    var _GraphQLError = require_GraphQLError();
    function NoUnusedFragmentsRule(context) {
      const operationDefs = [];
      const fragmentDefs = [];
      return {
        OperationDefinition(node) {
          operationDefs.push(node);
          return false;
        },
        FragmentDefinition(node) {
          fragmentDefs.push(node);
          return false;
        },
        Document: {
          leave() {
            const fragmentNameUsed = /* @__PURE__ */ Object.create(null);
            for (const operation of operationDefs) {
              for (const fragment of context.getRecursivelyReferencedFragments(
                operation
              )) {
                fragmentNameUsed[fragment.name.value] = true;
              }
            }
            for (const fragmentDef of fragmentDefs) {
              const fragName = fragmentDef.name.value;
              if (fragmentNameUsed[fragName] !== true) {
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    `Fragment "${fragName}" is never used.`,
                    {
                      nodes: fragmentDef
                    }
                  )
                );
              }
            }
          }
        }
      };
    }
    __name(NoUnusedFragmentsRule, "NoUnusedFragmentsRule");
  }
});

// ../../node_modules/graphql/validation/rules/NoUnusedVariablesRule.js
var require_NoUnusedVariablesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/NoUnusedVariablesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.NoUnusedVariablesRule = NoUnusedVariablesRule;
    var _GraphQLError = require_GraphQLError();
    function NoUnusedVariablesRule(context) {
      let variableDefs = [];
      return {
        OperationDefinition: {
          enter() {
            variableDefs = [];
          },
          leave(operation) {
            const variableNameUsed = /* @__PURE__ */ Object.create(null);
            const usages = context.getRecursiveVariableUsages(operation);
            for (const { node } of usages) {
              variableNameUsed[node.name.value] = true;
            }
            for (const variableDef of variableDefs) {
              const variableName = variableDef.variable.name.value;
              if (variableNameUsed[variableName] !== true) {
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    operation.name ? `Variable "$${variableName}" is never used in operation "${operation.name.value}".` : `Variable "$${variableName}" is never used.`,
                    {
                      nodes: variableDef
                    }
                  )
                );
              }
            }
          }
        },
        VariableDefinition(def) {
          variableDefs.push(def);
        }
      };
    }
    __name(NoUnusedVariablesRule, "NoUnusedVariablesRule");
  }
});

// ../../node_modules/graphql/utilities/sortValueNode.js
var require_sortValueNode = __commonJS({
  "../../node_modules/graphql/utilities/sortValueNode.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.sortValueNode = sortValueNode;
    var _naturalCompare = require_naturalCompare();
    var _kinds = require_kinds();
    function sortValueNode(valueNode) {
      switch (valueNode.kind) {
        case _kinds.Kind.OBJECT:
          return { ...valueNode, fields: sortFields(valueNode.fields) };
        case _kinds.Kind.LIST:
          return { ...valueNode, values: valueNode.values.map(sortValueNode) };
        case _kinds.Kind.INT:
        case _kinds.Kind.FLOAT:
        case _kinds.Kind.STRING:
        case _kinds.Kind.BOOLEAN:
        case _kinds.Kind.NULL:
        case _kinds.Kind.ENUM:
        case _kinds.Kind.VARIABLE:
          return valueNode;
      }
    }
    __name(sortValueNode, "sortValueNode");
    function sortFields(fields) {
      return fields.map((fieldNode) => ({
        ...fieldNode,
        value: sortValueNode(fieldNode.value)
      })).sort(
        (fieldA, fieldB) => (0, _naturalCompare.naturalCompare)(fieldA.name.value, fieldB.name.value)
      );
    }
    __name(sortFields, "sortFields");
  }
});

// ../../node_modules/graphql/validation/rules/OverlappingFieldsCanBeMergedRule.js
var require_OverlappingFieldsCanBeMergedRule = __commonJS({
  "../../node_modules/graphql/validation/rules/OverlappingFieldsCanBeMergedRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.OverlappingFieldsCanBeMergedRule = OverlappingFieldsCanBeMergedRule;
    var _inspect = require_inspect();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _definition = require_definition();
    var _sortValueNode = require_sortValueNode();
    var _typeFromAST = require_typeFromAST();
    function reasonMessage(reason) {
      if (Array.isArray(reason)) {
        return reason.map(
          ([responseName, subReason]) => `subfields "${responseName}" conflict because ` + reasonMessage(subReason)
        ).join(" and ");
      }
      return reason;
    }
    __name(reasonMessage, "reasonMessage");
    function OverlappingFieldsCanBeMergedRule(context) {
      const comparedFieldsAndFragmentPairs = new OrderedPairSet();
      const comparedFragmentPairs = new PairSet();
      const cachedFieldsAndFragmentNames = /* @__PURE__ */ new Map();
      return {
        SelectionSet(selectionSet) {
          const conflicts = findConflictsWithinSelectionSet(
            context,
            cachedFieldsAndFragmentNames,
            comparedFieldsAndFragmentPairs,
            comparedFragmentPairs,
            context.getParentType(),
            selectionSet
          );
          for (const [[responseName, reason], fields1, fields2] of conflicts) {
            const reasonMsg = reasonMessage(reason);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Fields "${responseName}" conflict because ${reasonMsg}. Use different aliases on the fields to fetch both if this was intentional.`,
                {
                  nodes: fields1.concat(fields2)
                }
              )
            );
          }
        }
      };
    }
    __name(OverlappingFieldsCanBeMergedRule, "OverlappingFieldsCanBeMergedRule");
    function findConflictsWithinSelectionSet(context, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, parentType, selectionSet) {
      const conflicts = [];
      const [fieldMap, fragmentNames] = getFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        parentType,
        selectionSet
      );
      collectConflictsWithin(
        context,
        conflicts,
        cachedFieldsAndFragmentNames,
        comparedFieldsAndFragmentPairs,
        comparedFragmentPairs,
        fieldMap
      );
      if (fragmentNames.length !== 0) {
        for (let i = 0; i < fragmentNames.length; i++) {
          collectConflictsBetweenFieldsAndFragment(
            context,
            conflicts,
            cachedFieldsAndFragmentNames,
            comparedFieldsAndFragmentPairs,
            comparedFragmentPairs,
            false,
            fieldMap,
            fragmentNames[i]
          );
          for (let j = i + 1; j < fragmentNames.length; j++) {
            collectConflictsBetweenFragments(
              context,
              conflicts,
              cachedFieldsAndFragmentNames,
              comparedFieldsAndFragmentPairs,
              comparedFragmentPairs,
              false,
              fragmentNames[i],
              fragmentNames[j]
            );
          }
        }
      }
      return conflicts;
    }
    __name(findConflictsWithinSelectionSet, "findConflictsWithinSelectionSet");
    function collectConflictsBetweenFieldsAndFragment(context, conflicts, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, areMutuallyExclusive, fieldMap, fragmentName) {
      if (comparedFieldsAndFragmentPairs.has(
        fieldMap,
        fragmentName,
        areMutuallyExclusive
      )) {
        return;
      }
      comparedFieldsAndFragmentPairs.add(
        fieldMap,
        fragmentName,
        areMutuallyExclusive
      );
      const fragment = context.getFragment(fragmentName);
      if (!fragment) {
        return;
      }
      const [fieldMap2, referencedFragmentNames] = getReferencedFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        fragment
      );
      if (fieldMap === fieldMap2) {
        return;
      }
      collectConflictsBetween(
        context,
        conflicts,
        cachedFieldsAndFragmentNames,
        comparedFieldsAndFragmentPairs,
        comparedFragmentPairs,
        areMutuallyExclusive,
        fieldMap,
        fieldMap2
      );
      for (const referencedFragmentName of referencedFragmentNames) {
        collectConflictsBetweenFieldsAndFragment(
          context,
          conflicts,
          cachedFieldsAndFragmentNames,
          comparedFieldsAndFragmentPairs,
          comparedFragmentPairs,
          areMutuallyExclusive,
          fieldMap,
          referencedFragmentName
        );
      }
    }
    __name(collectConflictsBetweenFieldsAndFragment, "collectConflictsBetweenFieldsAndFragment");
    function collectConflictsBetweenFragments(context, conflicts, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, areMutuallyExclusive, fragmentName1, fragmentName2) {
      if (fragmentName1 === fragmentName2) {
        return;
      }
      if (comparedFragmentPairs.has(
        fragmentName1,
        fragmentName2,
        areMutuallyExclusive
      )) {
        return;
      }
      comparedFragmentPairs.add(fragmentName1, fragmentName2, areMutuallyExclusive);
      const fragment1 = context.getFragment(fragmentName1);
      const fragment2 = context.getFragment(fragmentName2);
      if (!fragment1 || !fragment2) {
        return;
      }
      const [fieldMap1, referencedFragmentNames1] = getReferencedFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        fragment1
      );
      const [fieldMap2, referencedFragmentNames2] = getReferencedFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        fragment2
      );
      collectConflictsBetween(
        context,
        conflicts,
        cachedFieldsAndFragmentNames,
        comparedFieldsAndFragmentPairs,
        comparedFragmentPairs,
        areMutuallyExclusive,
        fieldMap1,
        fieldMap2
      );
      for (const referencedFragmentName2 of referencedFragmentNames2) {
        collectConflictsBetweenFragments(
          context,
          conflicts,
          cachedFieldsAndFragmentNames,
          comparedFieldsAndFragmentPairs,
          comparedFragmentPairs,
          areMutuallyExclusive,
          fragmentName1,
          referencedFragmentName2
        );
      }
      for (const referencedFragmentName1 of referencedFragmentNames1) {
        collectConflictsBetweenFragments(
          context,
          conflicts,
          cachedFieldsAndFragmentNames,
          comparedFieldsAndFragmentPairs,
          comparedFragmentPairs,
          areMutuallyExclusive,
          referencedFragmentName1,
          fragmentName2
        );
      }
    }
    __name(collectConflictsBetweenFragments, "collectConflictsBetweenFragments");
    function findConflictsBetweenSubSelectionSets(context, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, areMutuallyExclusive, parentType1, selectionSet1, parentType2, selectionSet2) {
      const conflicts = [];
      const [fieldMap1, fragmentNames1] = getFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        parentType1,
        selectionSet1
      );
      const [fieldMap2, fragmentNames2] = getFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        parentType2,
        selectionSet2
      );
      collectConflictsBetween(
        context,
        conflicts,
        cachedFieldsAndFragmentNames,
        comparedFieldsAndFragmentPairs,
        comparedFragmentPairs,
        areMutuallyExclusive,
        fieldMap1,
        fieldMap2
      );
      for (const fragmentName2 of fragmentNames2) {
        collectConflictsBetweenFieldsAndFragment(
          context,
          conflicts,
          cachedFieldsAndFragmentNames,
          comparedFieldsAndFragmentPairs,
          comparedFragmentPairs,
          areMutuallyExclusive,
          fieldMap1,
          fragmentName2
        );
      }
      for (const fragmentName1 of fragmentNames1) {
        collectConflictsBetweenFieldsAndFragment(
          context,
          conflicts,
          cachedFieldsAndFragmentNames,
          comparedFieldsAndFragmentPairs,
          comparedFragmentPairs,
          areMutuallyExclusive,
          fieldMap2,
          fragmentName1
        );
      }
      for (const fragmentName1 of fragmentNames1) {
        for (const fragmentName2 of fragmentNames2) {
          collectConflictsBetweenFragments(
            context,
            conflicts,
            cachedFieldsAndFragmentNames,
            comparedFieldsAndFragmentPairs,
            comparedFragmentPairs,
            areMutuallyExclusive,
            fragmentName1,
            fragmentName2
          );
        }
      }
      return conflicts;
    }
    __name(findConflictsBetweenSubSelectionSets, "findConflictsBetweenSubSelectionSets");
    function collectConflictsWithin(context, conflicts, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, fieldMap) {
      for (const [responseName, fields] of Object.entries(fieldMap)) {
        if (fields.length > 1) {
          for (let i = 0; i < fields.length; i++) {
            for (let j = i + 1; j < fields.length; j++) {
              const conflict = findConflict(
                context,
                cachedFieldsAndFragmentNames,
                comparedFieldsAndFragmentPairs,
                comparedFragmentPairs,
                false,
                // within one collection is never mutually exclusive
                responseName,
                fields[i],
                fields[j]
              );
              if (conflict) {
                conflicts.push(conflict);
              }
            }
          }
        }
      }
    }
    __name(collectConflictsWithin, "collectConflictsWithin");
    function collectConflictsBetween(context, conflicts, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, parentFieldsAreMutuallyExclusive, fieldMap1, fieldMap2) {
      for (const [responseName, fields1] of Object.entries(fieldMap1)) {
        const fields2 = fieldMap2[responseName];
        if (fields2) {
          for (const field1 of fields1) {
            for (const field2 of fields2) {
              const conflict = findConflict(
                context,
                cachedFieldsAndFragmentNames,
                comparedFieldsAndFragmentPairs,
                comparedFragmentPairs,
                parentFieldsAreMutuallyExclusive,
                responseName,
                field1,
                field2
              );
              if (conflict) {
                conflicts.push(conflict);
              }
            }
          }
        }
      }
    }
    __name(collectConflictsBetween, "collectConflictsBetween");
    function findConflict(context, cachedFieldsAndFragmentNames, comparedFieldsAndFragmentPairs, comparedFragmentPairs, parentFieldsAreMutuallyExclusive, responseName, field1, field2) {
      const [parentType1, node1, def1] = field1;
      const [parentType2, node2, def2] = field2;
      const areMutuallyExclusive = parentFieldsAreMutuallyExclusive || parentType1 !== parentType2 && (0, _definition.isObjectType)(parentType1) && (0, _definition.isObjectType)(parentType2);
      if (!areMutuallyExclusive) {
        const name1 = node1.name.value;
        const name2 = node2.name.value;
        if (name1 !== name2) {
          return [
            [responseName, `"${name1}" and "${name2}" are different fields`],
            [node1],
            [node2]
          ];
        }
        if (!sameArguments(node1, node2)) {
          return [
            [responseName, "they have differing arguments"],
            [node1],
            [node2]
          ];
        }
      }
      const type1 = def1 === null || def1 === void 0 ? void 0 : def1.type;
      const type2 = def2 === null || def2 === void 0 ? void 0 : def2.type;
      if (type1 && type2 && doTypesConflict(type1, type2)) {
        return [
          [
            responseName,
            `they return conflicting types "${(0, _inspect.inspect)(
              type1
            )}" and "${(0, _inspect.inspect)(type2)}"`
          ],
          [node1],
          [node2]
        ];
      }
      const selectionSet1 = node1.selectionSet;
      const selectionSet2 = node2.selectionSet;
      if (selectionSet1 && selectionSet2) {
        const conflicts = findConflictsBetweenSubSelectionSets(
          context,
          cachedFieldsAndFragmentNames,
          comparedFieldsAndFragmentPairs,
          comparedFragmentPairs,
          areMutuallyExclusive,
          (0, _definition.getNamedType)(type1),
          selectionSet1,
          (0, _definition.getNamedType)(type2),
          selectionSet2
        );
        return subfieldConflicts(conflicts, responseName, node1, node2);
      }
    }
    __name(findConflict, "findConflict");
    function sameArguments(node1, node2) {
      const args1 = node1.arguments;
      const args2 = node2.arguments;
      if (args1 === void 0 || args1.length === 0) {
        return args2 === void 0 || args2.length === 0;
      }
      if (args2 === void 0 || args2.length === 0) {
        return false;
      }
      if (args1.length !== args2.length) {
        return false;
      }
      const values2 = new Map(args2.map(({ name, value }) => [name.value, value]));
      return args1.every((arg1) => {
        const value1 = arg1.value;
        const value2 = values2.get(arg1.name.value);
        if (value2 === void 0) {
          return false;
        }
        return stringifyValue(value1) === stringifyValue(value2);
      });
    }
    __name(sameArguments, "sameArguments");
    function stringifyValue(value) {
      return (0, _printer.print)((0, _sortValueNode.sortValueNode)(value));
    }
    __name(stringifyValue, "stringifyValue");
    function doTypesConflict(type1, type2) {
      if ((0, _definition.isListType)(type1)) {
        return (0, _definition.isListType)(type2) ? doTypesConflict(type1.ofType, type2.ofType) : true;
      }
      if ((0, _definition.isListType)(type2)) {
        return true;
      }
      if ((0, _definition.isNonNullType)(type1)) {
        return (0, _definition.isNonNullType)(type2) ? doTypesConflict(type1.ofType, type2.ofType) : true;
      }
      if ((0, _definition.isNonNullType)(type2)) {
        return true;
      }
      if ((0, _definition.isLeafType)(type1) || (0, _definition.isLeafType)(type2)) {
        return type1 !== type2;
      }
      return false;
    }
    __name(doTypesConflict, "doTypesConflict");
    function getFieldsAndFragmentNames(context, cachedFieldsAndFragmentNames, parentType, selectionSet) {
      const cached = cachedFieldsAndFragmentNames.get(selectionSet);
      if (cached) {
        return cached;
      }
      const nodeAndDefs = /* @__PURE__ */ Object.create(null);
      const fragmentNames = /* @__PURE__ */ Object.create(null);
      _collectFieldsAndFragmentNames(
        context,
        parentType,
        selectionSet,
        nodeAndDefs,
        fragmentNames
      );
      const result = [nodeAndDefs, Object.keys(fragmentNames)];
      cachedFieldsAndFragmentNames.set(selectionSet, result);
      return result;
    }
    __name(getFieldsAndFragmentNames, "getFieldsAndFragmentNames");
    function getReferencedFieldsAndFragmentNames(context, cachedFieldsAndFragmentNames, fragment) {
      const cached = cachedFieldsAndFragmentNames.get(fragment.selectionSet);
      if (cached) {
        return cached;
      }
      const fragmentType = (0, _typeFromAST.typeFromAST)(
        context.getSchema(),
        fragment.typeCondition
      );
      return getFieldsAndFragmentNames(
        context,
        cachedFieldsAndFragmentNames,
        fragmentType,
        fragment.selectionSet
      );
    }
    __name(getReferencedFieldsAndFragmentNames, "getReferencedFieldsAndFragmentNames");
    function _collectFieldsAndFragmentNames(context, parentType, selectionSet, nodeAndDefs, fragmentNames) {
      for (const selection of selectionSet.selections) {
        switch (selection.kind) {
          case _kinds.Kind.FIELD: {
            const fieldName = selection.name.value;
            let fieldDef;
            if ((0, _definition.isObjectType)(parentType) || (0, _definition.isInterfaceType)(parentType)) {
              fieldDef = parentType.getFields()[fieldName];
            }
            const responseName = selection.alias ? selection.alias.value : fieldName;
            if (!nodeAndDefs[responseName]) {
              nodeAndDefs[responseName] = [];
            }
            nodeAndDefs[responseName].push([parentType, selection, fieldDef]);
            break;
          }
          case _kinds.Kind.FRAGMENT_SPREAD:
            fragmentNames[selection.name.value] = true;
            break;
          case _kinds.Kind.INLINE_FRAGMENT: {
            const typeCondition = selection.typeCondition;
            const inlineFragmentType = typeCondition ? (0, _typeFromAST.typeFromAST)(context.getSchema(), typeCondition) : parentType;
            _collectFieldsAndFragmentNames(
              context,
              inlineFragmentType,
              selection.selectionSet,
              nodeAndDefs,
              fragmentNames
            );
            break;
          }
        }
      }
    }
    __name(_collectFieldsAndFragmentNames, "_collectFieldsAndFragmentNames");
    function subfieldConflicts(conflicts, responseName, node1, node2) {
      if (conflicts.length > 0) {
        return [
          [responseName, conflicts.map(([reason]) => reason)],
          [node1, ...conflicts.map(([, fields1]) => fields1).flat()],
          [node2, ...conflicts.map(([, , fields2]) => fields2).flat()]
        ];
      }
    }
    __name(subfieldConflicts, "subfieldConflicts");
    var OrderedPairSet = class {
      static {
        __name(this, "OrderedPairSet");
      }
      constructor() {
        this._data = /* @__PURE__ */ new Map();
      }
      has(a, b, weaklyPresent) {
        var _this$_data$get;
        const result = (_this$_data$get = this._data.get(a)) === null || _this$_data$get === void 0 ? void 0 : _this$_data$get.get(b);
        if (result === void 0) {
          return false;
        }
        return weaklyPresent ? true : weaklyPresent === result;
      }
      add(a, b, weaklyPresent) {
        const map = this._data.get(a);
        if (map === void 0) {
          this._data.set(a, /* @__PURE__ */ new Map([[b, weaklyPresent]]));
        } else {
          map.set(b, weaklyPresent);
        }
      }
    };
    var PairSet = class {
      static {
        __name(this, "PairSet");
      }
      constructor() {
        this._orderedPairSet = new OrderedPairSet();
      }
      has(a, b, weaklyPresent) {
        return a < b ? this._orderedPairSet.has(a, b, weaklyPresent) : this._orderedPairSet.has(b, a, weaklyPresent);
      }
      add(a, b, weaklyPresent) {
        if (a < b) {
          this._orderedPairSet.add(a, b, weaklyPresent);
        } else {
          this._orderedPairSet.add(b, a, weaklyPresent);
        }
      }
    };
  }
});

// ../../node_modules/graphql/validation/rules/PossibleFragmentSpreadsRule.js
var require_PossibleFragmentSpreadsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/PossibleFragmentSpreadsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.PossibleFragmentSpreadsRule = PossibleFragmentSpreadsRule;
    var _inspect = require_inspect();
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    var _typeComparators = require_typeComparators();
    var _typeFromAST = require_typeFromAST();
    function PossibleFragmentSpreadsRule(context) {
      return {
        InlineFragment(node) {
          const fragType = context.getType();
          const parentType = context.getParentType();
          if ((0, _definition.isCompositeType)(fragType) && (0, _definition.isCompositeType)(parentType) && !(0, _typeComparators.doTypesOverlap)(
            context.getSchema(),
            fragType,
            parentType
          )) {
            const parentTypeStr = (0, _inspect.inspect)(parentType);
            const fragTypeStr = (0, _inspect.inspect)(fragType);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Fragment cannot be spread here as objects of type "${parentTypeStr}" can never be of type "${fragTypeStr}".`,
                {
                  nodes: node
                }
              )
            );
          }
        },
        FragmentSpread(node) {
          const fragName = node.name.value;
          const fragType = getFragmentType(context, fragName);
          const parentType = context.getParentType();
          if (fragType && parentType && !(0, _typeComparators.doTypesOverlap)(
            context.getSchema(),
            fragType,
            parentType
          )) {
            const parentTypeStr = (0, _inspect.inspect)(parentType);
            const fragTypeStr = (0, _inspect.inspect)(fragType);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Fragment "${fragName}" cannot be spread here as objects of type "${parentTypeStr}" can never be of type "${fragTypeStr}".`,
                {
                  nodes: node
                }
              )
            );
          }
        }
      };
    }
    __name(PossibleFragmentSpreadsRule, "PossibleFragmentSpreadsRule");
    function getFragmentType(context, name) {
      const frag = context.getFragment(name);
      if (frag) {
        const type = (0, _typeFromAST.typeFromAST)(
          context.getSchema(),
          frag.typeCondition
        );
        if ((0, _definition.isCompositeType)(type)) {
          return type;
        }
      }
    }
    __name(getFragmentType, "getFragmentType");
  }
});

// ../../node_modules/graphql/validation/rules/PossibleTypeExtensionsRule.js
var require_PossibleTypeExtensionsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/PossibleTypeExtensionsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.PossibleTypeExtensionsRule = PossibleTypeExtensionsRule;
    var _didYouMean = require_didYouMean();
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _suggestionList = require_suggestionList();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _predicates = require_predicates();
    var _definition = require_definition();
    function PossibleTypeExtensionsRule(context) {
      const schema = context.getSchema();
      const definedTypes = /* @__PURE__ */ Object.create(null);
      for (const def of context.getDocument().definitions) {
        if ((0, _predicates.isTypeDefinitionNode)(def)) {
          definedTypes[def.name.value] = def;
        }
      }
      return {
        ScalarTypeExtension: checkExtension,
        ObjectTypeExtension: checkExtension,
        InterfaceTypeExtension: checkExtension,
        UnionTypeExtension: checkExtension,
        EnumTypeExtension: checkExtension,
        InputObjectTypeExtension: checkExtension
      };
      function checkExtension(node) {
        const typeName = node.name.value;
        const defNode = definedTypes[typeName];
        const existingType = schema === null || schema === void 0 ? void 0 : schema.getType(typeName);
        let expectedKind;
        if (defNode) {
          expectedKind = defKindToExtKind[defNode.kind];
        } else if (existingType) {
          expectedKind = typeToExtKind(existingType);
        }
        if (expectedKind) {
          if (expectedKind !== node.kind) {
            const kindStr = extensionKindToTypeName(node.kind);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Cannot extend non-${kindStr} type "${typeName}".`,
                {
                  nodes: defNode ? [defNode, node] : node
                }
              )
            );
          }
        } else {
          const allTypeNames = Object.keys({
            ...definedTypes,
            ...schema === null || schema === void 0 ? void 0 : schema.getTypeMap()
          });
          const suggestedTypes = (0, _suggestionList.suggestionList)(
            typeName,
            allTypeNames
          );
          context.reportError(
            new _GraphQLError.GraphQLError(
              `Cannot extend type "${typeName}" because it is not defined.` + (0, _didYouMean.didYouMean)(suggestedTypes),
              {
                nodes: node.name
              }
            )
          );
        }
      }
      __name(checkExtension, "checkExtension");
    }
    __name(PossibleTypeExtensionsRule, "PossibleTypeExtensionsRule");
    var defKindToExtKind = {
      [_kinds.Kind.SCALAR_TYPE_DEFINITION]: _kinds.Kind.SCALAR_TYPE_EXTENSION,
      [_kinds.Kind.OBJECT_TYPE_DEFINITION]: _kinds.Kind.OBJECT_TYPE_EXTENSION,
      [_kinds.Kind.INTERFACE_TYPE_DEFINITION]: _kinds.Kind.INTERFACE_TYPE_EXTENSION,
      [_kinds.Kind.UNION_TYPE_DEFINITION]: _kinds.Kind.UNION_TYPE_EXTENSION,
      [_kinds.Kind.ENUM_TYPE_DEFINITION]: _kinds.Kind.ENUM_TYPE_EXTENSION,
      [_kinds.Kind.INPUT_OBJECT_TYPE_DEFINITION]: _kinds.Kind.INPUT_OBJECT_TYPE_EXTENSION
    };
    function typeToExtKind(type) {
      if ((0, _definition.isScalarType)(type)) {
        return _kinds.Kind.SCALAR_TYPE_EXTENSION;
      }
      if ((0, _definition.isObjectType)(type)) {
        return _kinds.Kind.OBJECT_TYPE_EXTENSION;
      }
      if ((0, _definition.isInterfaceType)(type)) {
        return _kinds.Kind.INTERFACE_TYPE_EXTENSION;
      }
      if ((0, _definition.isUnionType)(type)) {
        return _kinds.Kind.UNION_TYPE_EXTENSION;
      }
      if ((0, _definition.isEnumType)(type)) {
        return _kinds.Kind.ENUM_TYPE_EXTENSION;
      }
      if ((0, _definition.isInputObjectType)(type)) {
        return _kinds.Kind.INPUT_OBJECT_TYPE_EXTENSION;
      }
      (0, _invariant.invariant)(
        false,
        "Unexpected type: " + (0, _inspect.inspect)(type)
      );
    }
    __name(typeToExtKind, "typeToExtKind");
    function extensionKindToTypeName(kind) {
      switch (kind) {
        case _kinds.Kind.SCALAR_TYPE_EXTENSION:
          return "scalar";
        case _kinds.Kind.OBJECT_TYPE_EXTENSION:
          return "object";
        case _kinds.Kind.INTERFACE_TYPE_EXTENSION:
          return "interface";
        case _kinds.Kind.UNION_TYPE_EXTENSION:
          return "union";
        case _kinds.Kind.ENUM_TYPE_EXTENSION:
          return "enum";
        case _kinds.Kind.INPUT_OBJECT_TYPE_EXTENSION:
          return "input object";
        // Not reachable. All possible types have been considered
        /* c8 ignore next */
        default:
          (0, _invariant.invariant)(
            false,
            "Unexpected kind: " + (0, _inspect.inspect)(kind)
          );
      }
    }
    __name(extensionKindToTypeName, "extensionKindToTypeName");
  }
});

// ../../node_modules/graphql/validation/rules/ProvidedRequiredArgumentsRule.js
var require_ProvidedRequiredArgumentsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/ProvidedRequiredArgumentsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.ProvidedRequiredArgumentsOnDirectivesRule = ProvidedRequiredArgumentsOnDirectivesRule;
    exports.ProvidedRequiredArgumentsRule = ProvidedRequiredArgumentsRule;
    var _inspect = require_inspect();
    var _keyMap = require_keyMap();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _definition = require_definition();
    var _directives = require_directives();
    function ProvidedRequiredArgumentsRule(context) {
      return {
        // eslint-disable-next-line new-cap
        ...ProvidedRequiredArgumentsOnDirectivesRule(context),
        Field: {
          // Validate on leave to allow for deeper errors to appear first.
          leave(fieldNode) {
            var _fieldNode$arguments;
            const fieldDef = context.getFieldDef();
            if (!fieldDef) {
              return false;
            }
            const providedArgs = new Set(
              // FIXME: https://github.com/graphql/graphql-js/issues/2203
              /* c8 ignore next */
              (_fieldNode$arguments = fieldNode.arguments) === null || _fieldNode$arguments === void 0 ? void 0 : _fieldNode$arguments.map((arg) => arg.name.value)
            );
            for (const argDef of fieldDef.args) {
              if (!providedArgs.has(argDef.name) && (0, _definition.isRequiredArgument)(argDef)) {
                const argTypeStr = (0, _inspect.inspect)(argDef.type);
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    `Field "${fieldDef.name}" argument "${argDef.name}" of type "${argTypeStr}" is required, but it was not provided.`,
                    {
                      nodes: fieldNode
                    }
                  )
                );
              }
            }
          }
        }
      };
    }
    __name(ProvidedRequiredArgumentsRule, "ProvidedRequiredArgumentsRule");
    function ProvidedRequiredArgumentsOnDirectivesRule(context) {
      var _schema$getDirectives;
      const requiredArgsMap = /* @__PURE__ */ Object.create(null);
      const schema = context.getSchema();
      const definedDirectives = (_schema$getDirectives = schema === null || schema === void 0 ? void 0 : schema.getDirectives()) !== null && _schema$getDirectives !== void 0 ? _schema$getDirectives : _directives.specifiedDirectives;
      for (const directive of definedDirectives) {
        requiredArgsMap[directive.name] = (0, _keyMap.keyMap)(
          directive.args.filter(_definition.isRequiredArgument),
          (arg) => arg.name
        );
      }
      const astDefinitions = context.getDocument().definitions;
      for (const def of astDefinitions) {
        if (def.kind === _kinds.Kind.DIRECTIVE_DEFINITION) {
          var _def$arguments;
          const argNodes = (_def$arguments = def.arguments) !== null && _def$arguments !== void 0 ? _def$arguments : [];
          requiredArgsMap[def.name.value] = (0, _keyMap.keyMap)(
            argNodes.filter(isRequiredArgumentNode),
            (arg) => arg.name.value
          );
        }
      }
      return {
        Directive: {
          // Validate on leave to allow for deeper errors to appear first.
          leave(directiveNode) {
            const directiveName = directiveNode.name.value;
            const requiredArgs = requiredArgsMap[directiveName];
            if (requiredArgs) {
              var _directiveNode$argume;
              const argNodes = (_directiveNode$argume = directiveNode.arguments) !== null && _directiveNode$argume !== void 0 ? _directiveNode$argume : [];
              const argNodeMap = new Set(argNodes.map((arg) => arg.name.value));
              for (const [argName, argDef] of Object.entries(requiredArgs)) {
                if (!argNodeMap.has(argName)) {
                  const argType = (0, _definition.isType)(argDef.type) ? (0, _inspect.inspect)(argDef.type) : (0, _printer.print)(argDef.type);
                  context.reportError(
                    new _GraphQLError.GraphQLError(
                      `Directive "@${directiveName}" argument "${argName}" of type "${argType}" is required, but it was not provided.`,
                      {
                        nodes: directiveNode
                      }
                    )
                  );
                }
              }
            }
          }
        }
      };
    }
    __name(ProvidedRequiredArgumentsOnDirectivesRule, "ProvidedRequiredArgumentsOnDirectivesRule");
    function isRequiredArgumentNode(arg) {
      return arg.type.kind === _kinds.Kind.NON_NULL_TYPE && arg.defaultValue == null;
    }
    __name(isRequiredArgumentNode, "isRequiredArgumentNode");
  }
});

// ../../node_modules/graphql/validation/rules/ScalarLeafsRule.js
var require_ScalarLeafsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/ScalarLeafsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.ScalarLeafsRule = ScalarLeafsRule;
    var _inspect = require_inspect();
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    function ScalarLeafsRule(context) {
      return {
        Field(node) {
          const type = context.getType();
          const selectionSet = node.selectionSet;
          if (type) {
            if ((0, _definition.isLeafType)((0, _definition.getNamedType)(type))) {
              if (selectionSet) {
                const fieldName = node.name.value;
                const typeStr = (0, _inspect.inspect)(type);
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    `Field "${fieldName}" must not have a selection since type "${typeStr}" has no subfields.`,
                    {
                      nodes: selectionSet
                    }
                  )
                );
              }
            } else if (!selectionSet) {
              const fieldName = node.name.value;
              const typeStr = (0, _inspect.inspect)(type);
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Field "${fieldName}" of type "${typeStr}" must have a selection of subfields. Did you mean "${fieldName} { ... }"?`,
                  {
                    nodes: node
                  }
                )
              );
            } else if (selectionSet.selections.length === 0) {
              const fieldName = node.name.value;
              const typeStr = (0, _inspect.inspect)(type);
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Field "${fieldName}" of type "${typeStr}" must have at least one field selected.`,
                  {
                    nodes: node
                  }
                )
              );
            }
          }
        }
      };
    }
    __name(ScalarLeafsRule, "ScalarLeafsRule");
  }
});

// ../../node_modules/graphql/jsutils/printPathArray.js
var require_printPathArray = __commonJS({
  "../../node_modules/graphql/jsutils/printPathArray.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.printPathArray = printPathArray;
    function printPathArray(path2) {
      return path2.map(
        (key) => typeof key === "number" ? "[" + key.toString() + "]" : "." + key
      ).join("");
    }
    __name(printPathArray, "printPathArray");
  }
});

// ../../node_modules/graphql/jsutils/Path.js
var require_Path = __commonJS({
  "../../node_modules/graphql/jsutils/Path.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.addPath = addPath;
    exports.pathToArray = pathToArray;
    function addPath(prev, key, typename) {
      return {
        prev,
        key,
        typename
      };
    }
    __name(addPath, "addPath");
    function pathToArray(path2) {
      const flattened = [];
      let curr = path2;
      while (curr) {
        flattened.push(curr.key);
        curr = curr.prev;
      }
      return flattened.reverse();
    }
    __name(pathToArray, "pathToArray");
  }
});

// ../../node_modules/graphql/utilities/coerceInputValue.js
var require_coerceInputValue = __commonJS({
  "../../node_modules/graphql/utilities/coerceInputValue.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.coerceInputValue = coerceInputValue;
    var _didYouMean = require_didYouMean();
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _isIterableObject = require_isIterableObject();
    var _isObjectLike = require_isObjectLike();
    var _Path = require_Path();
    var _printPathArray = require_printPathArray();
    var _suggestionList = require_suggestionList();
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    function coerceInputValue(inputValue, type, onError = defaultOnError) {
      return coerceInputValueImpl(inputValue, type, onError, void 0);
    }
    __name(coerceInputValue, "coerceInputValue");
    function defaultOnError(path2, invalidValue, error) {
      let errorPrefix = "Invalid value " + (0, _inspect.inspect)(invalidValue);
      if (path2.length > 0) {
        errorPrefix += ` at "value${(0, _printPathArray.printPathArray)(path2)}"`;
      }
      error.message = errorPrefix + ": " + error.message;
      throw error;
    }
    __name(defaultOnError, "defaultOnError");
    function coerceInputValueImpl(inputValue, type, onError, path2) {
      if ((0, _definition.isNonNullType)(type)) {
        if (inputValue != null) {
          return coerceInputValueImpl(inputValue, type.ofType, onError, path2);
        }
        onError(
          (0, _Path.pathToArray)(path2),
          inputValue,
          new _GraphQLError.GraphQLError(
            `Expected non-nullable type "${(0, _inspect.inspect)(
              type
            )}" not to be null.`
          )
        );
        return;
      }
      if (inputValue == null) {
        return null;
      }
      if ((0, _definition.isListType)(type)) {
        const itemType = type.ofType;
        if ((0, _isIterableObject.isIterableObject)(inputValue)) {
          return Array.from(inputValue, (itemValue, index) => {
            const itemPath = (0, _Path.addPath)(path2, index, void 0);
            return coerceInputValueImpl(itemValue, itemType, onError, itemPath);
          });
        }
        return [coerceInputValueImpl(inputValue, itemType, onError, path2)];
      }
      if ((0, _definition.isInputObjectType)(type)) {
        if (!(0, _isObjectLike.isObjectLike)(inputValue)) {
          onError(
            (0, _Path.pathToArray)(path2),
            inputValue,
            new _GraphQLError.GraphQLError(
              `Expected type "${type.name}" to be an object.`
            )
          );
          return;
        }
        const coercedValue = {};
        const fieldDefs = type.getFields();
        for (const field of Object.values(fieldDefs)) {
          const fieldValue = inputValue[field.name];
          if (fieldValue === void 0) {
            if (field.defaultValue !== void 0) {
              coercedValue[field.name] = field.defaultValue;
            } else if ((0, _definition.isNonNullType)(field.type)) {
              const typeStr = (0, _inspect.inspect)(field.type);
              onError(
                (0, _Path.pathToArray)(path2),
                inputValue,
                new _GraphQLError.GraphQLError(
                  `Field "${field.name}" of required type "${typeStr}" was not provided.`
                )
              );
            }
            continue;
          }
          coercedValue[field.name] = coerceInputValueImpl(
            fieldValue,
            field.type,
            onError,
            (0, _Path.addPath)(path2, field.name, type.name)
          );
        }
        for (const fieldName of Object.keys(inputValue)) {
          if (!fieldDefs[fieldName]) {
            const suggestions = (0, _suggestionList.suggestionList)(
              fieldName,
              Object.keys(type.getFields())
            );
            onError(
              (0, _Path.pathToArray)(path2),
              inputValue,
              new _GraphQLError.GraphQLError(
                `Field "${fieldName}" is not defined by type "${type.name}".` + (0, _didYouMean.didYouMean)(suggestions)
              )
            );
          }
        }
        if (type.isOneOf) {
          const keys = Object.keys(coercedValue);
          if (keys.length !== 1) {
            onError(
              (0, _Path.pathToArray)(path2),
              inputValue,
              new _GraphQLError.GraphQLError(
                `Exactly one key must be specified for OneOf type "${type.name}".`
              )
            );
          }
          const key = keys[0];
          const value = coercedValue[key];
          if (value === null) {
            onError(
              (0, _Path.pathToArray)(path2).concat(key),
              value,
              new _GraphQLError.GraphQLError(`Field "${key}" must be non-null.`)
            );
          }
        }
        return coercedValue;
      }
      if ((0, _definition.isLeafType)(type)) {
        let parseResult;
        try {
          parseResult = type.parseValue(inputValue);
        } catch (error) {
          if (error instanceof _GraphQLError.GraphQLError) {
            onError((0, _Path.pathToArray)(path2), inputValue, error);
          } else {
            onError(
              (0, _Path.pathToArray)(path2),
              inputValue,
              new _GraphQLError.GraphQLError(
                `Expected type "${type.name}". ` + error.message,
                {
                  originalError: error
                }
              )
            );
          }
          return;
        }
        if (parseResult === void 0) {
          onError(
            (0, _Path.pathToArray)(path2),
            inputValue,
            new _GraphQLError.GraphQLError(`Expected type "${type.name}".`)
          );
        }
        return parseResult;
      }
      (0, _invariant.invariant)(
        false,
        "Unexpected input type: " + (0, _inspect.inspect)(type)
      );
    }
    __name(coerceInputValueImpl, "coerceInputValueImpl");
  }
});

// ../../node_modules/graphql/utilities/valueFromAST.js
var require_valueFromAST = __commonJS({
  "../../node_modules/graphql/utilities/valueFromAST.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.valueFromAST = valueFromAST;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _keyMap = require_keyMap();
    var _kinds = require_kinds();
    var _definition = require_definition();
    function valueFromAST(valueNode, type, variables) {
      if (!valueNode) {
        return;
      }
      if (valueNode.kind === _kinds.Kind.VARIABLE) {
        const variableName = valueNode.name.value;
        if (variables == null || variables[variableName] === void 0) {
          return;
        }
        const variableValue = variables[variableName];
        if (variableValue === null && (0, _definition.isNonNullType)(type)) {
          return;
        }
        return variableValue;
      }
      if ((0, _definition.isNonNullType)(type)) {
        if (valueNode.kind === _kinds.Kind.NULL) {
          return;
        }
        return valueFromAST(valueNode, type.ofType, variables);
      }
      if (valueNode.kind === _kinds.Kind.NULL) {
        return null;
      }
      if ((0, _definition.isListType)(type)) {
        const itemType = type.ofType;
        if (valueNode.kind === _kinds.Kind.LIST) {
          const coercedValues = [];
          for (const itemNode of valueNode.values) {
            if (isMissingVariable(itemNode, variables)) {
              if ((0, _definition.isNonNullType)(itemType)) {
                return;
              }
              coercedValues.push(null);
            } else {
              const itemValue = valueFromAST(itemNode, itemType, variables);
              if (itemValue === void 0) {
                return;
              }
              coercedValues.push(itemValue);
            }
          }
          return coercedValues;
        }
        const coercedValue = valueFromAST(valueNode, itemType, variables);
        if (coercedValue === void 0) {
          return;
        }
        return [coercedValue];
      }
      if ((0, _definition.isInputObjectType)(type)) {
        if (valueNode.kind !== _kinds.Kind.OBJECT) {
          return;
        }
        const coercedObj = /* @__PURE__ */ Object.create(null);
        const fieldNodes = (0, _keyMap.keyMap)(
          valueNode.fields,
          (field) => field.name.value
        );
        for (const field of Object.values(type.getFields())) {
          const fieldNode = fieldNodes[field.name];
          if (!fieldNode || isMissingVariable(fieldNode.value, variables)) {
            if (field.defaultValue !== void 0) {
              coercedObj[field.name] = field.defaultValue;
            } else if ((0, _definition.isNonNullType)(field.type)) {
              return;
            }
            continue;
          }
          const fieldValue = valueFromAST(fieldNode.value, field.type, variables);
          if (fieldValue === void 0) {
            return;
          }
          coercedObj[field.name] = fieldValue;
        }
        if (type.isOneOf) {
          const keys = Object.keys(coercedObj);
          if (keys.length !== 1) {
            return;
          }
          if (coercedObj[keys[0]] === null) {
            return;
          }
        }
        return coercedObj;
      }
      if ((0, _definition.isLeafType)(type)) {
        let result;
        try {
          result = type.parseLiteral(valueNode, variables);
        } catch (_error) {
          return;
        }
        if (result === void 0) {
          return;
        }
        return result;
      }
      (0, _invariant.invariant)(
        false,
        "Unexpected input type: " + (0, _inspect.inspect)(type)
      );
    }
    __name(valueFromAST, "valueFromAST");
    function isMissingVariable(valueNode, variables) {
      return valueNode.kind === _kinds.Kind.VARIABLE && (variables == null || variables[valueNode.name.value] === void 0);
    }
    __name(isMissingVariable, "isMissingVariable");
  }
});

// ../../node_modules/graphql/execution/values.js
var require_values = __commonJS({
  "../../node_modules/graphql/execution/values.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.getArgumentValues = getArgumentValues;
    exports.getDirectiveValues = getDirectiveValues;
    exports.getVariableValues = getVariableValues;
    var _inspect = require_inspect();
    var _keyMap = require_keyMap();
    var _printPathArray = require_printPathArray();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _definition = require_definition();
    var _coerceInputValue = require_coerceInputValue();
    var _typeFromAST = require_typeFromAST();
    var _valueFromAST = require_valueFromAST();
    function getVariableValues(schema, varDefNodes, inputs, options) {
      const errors = [];
      const maxErrors = options === null || options === void 0 ? void 0 : options.maxErrors;
      try {
        const coerced = coerceVariableValues(
          schema,
          varDefNodes,
          inputs,
          (error) => {
            if (maxErrors != null && errors.length >= maxErrors) {
              throw new _GraphQLError.GraphQLError(
                "Too many errors processing variables, error limit reached. Execution aborted."
              );
            }
            errors.push(error);
          }
        );
        if (errors.length === 0) {
          return {
            coerced
          };
        }
      } catch (error) {
        errors.push(error);
      }
      return {
        errors
      };
    }
    __name(getVariableValues, "getVariableValues");
    function coerceVariableValues(schema, varDefNodes, inputs, onError) {
      const coercedValues = {};
      for (const varDefNode of varDefNodes) {
        const varName = varDefNode.variable.name.value;
        const varType = (0, _typeFromAST.typeFromAST)(schema, varDefNode.type);
        if (!(0, _definition.isInputType)(varType)) {
          const varTypeStr = (0, _printer.print)(varDefNode.type);
          onError(
            new _GraphQLError.GraphQLError(
              `Variable "$${varName}" expected value of type "${varTypeStr}" which cannot be used as an input type.`,
              {
                nodes: varDefNode.type
              }
            )
          );
          continue;
        }
        if (!hasOwnProperty(inputs, varName)) {
          if (varDefNode.defaultValue) {
            coercedValues[varName] = (0, _valueFromAST.valueFromAST)(
              varDefNode.defaultValue,
              varType
            );
          } else if ((0, _definition.isNonNullType)(varType)) {
            const varTypeStr = (0, _inspect.inspect)(varType);
            onError(
              new _GraphQLError.GraphQLError(
                `Variable "$${varName}" of required type "${varTypeStr}" was not provided.`,
                {
                  nodes: varDefNode
                }
              )
            );
          }
          continue;
        }
        const value = inputs[varName];
        if (value === null && (0, _definition.isNonNullType)(varType)) {
          const varTypeStr = (0, _inspect.inspect)(varType);
          onError(
            new _GraphQLError.GraphQLError(
              `Variable "$${varName}" of non-null type "${varTypeStr}" must not be null.`,
              {
                nodes: varDefNode
              }
            )
          );
          continue;
        }
        coercedValues[varName] = (0, _coerceInputValue.coerceInputValue)(
          value,
          varType,
          (path2, invalidValue, error) => {
            let prefix = `Variable "$${varName}" got invalid value ` + (0, _inspect.inspect)(invalidValue);
            if (path2.length > 0) {
              prefix += ` at "${varName}${(0, _printPathArray.printPathArray)(
                path2
              )}"`;
            }
            onError(
              new _GraphQLError.GraphQLError(prefix + "; " + error.message, {
                nodes: varDefNode,
                originalError: error
              })
            );
          }
        );
      }
      return coercedValues;
    }
    __name(coerceVariableValues, "coerceVariableValues");
    function getArgumentValues(def, node, variableValues) {
      var _node$arguments;
      const coercedValues = {};
      const argumentNodes = (_node$arguments = node.arguments) !== null && _node$arguments !== void 0 ? _node$arguments : [];
      const argNodeMap = (0, _keyMap.keyMap)(
        argumentNodes,
        (arg) => arg.name.value
      );
      for (const argDef of def.args) {
        const name = argDef.name;
        const argType = argDef.type;
        const argumentNode = argNodeMap[name];
        if (!argumentNode) {
          if (argDef.defaultValue !== void 0) {
            coercedValues[name] = argDef.defaultValue;
          } else if ((0, _definition.isNonNullType)(argType)) {
            throw new _GraphQLError.GraphQLError(
              `Argument "${name}" of required type "${(0, _inspect.inspect)(
                argType
              )}" was not provided.`,
              {
                nodes: node
              }
            );
          }
          continue;
        }
        const valueNode = argumentNode.value;
        let isNull = valueNode.kind === _kinds.Kind.NULL;
        if (valueNode.kind === _kinds.Kind.VARIABLE) {
          const variableName = valueNode.name.value;
          if (variableValues == null || !hasOwnProperty(variableValues, variableName)) {
            if (argDef.defaultValue !== void 0) {
              coercedValues[name] = argDef.defaultValue;
            } else if ((0, _definition.isNonNullType)(argType)) {
              throw new _GraphQLError.GraphQLError(
                `Argument "${name}" of required type "${(0, _inspect.inspect)(
                  argType
                )}" was provided the variable "$${variableName}" which was not provided a runtime value.`,
                {
                  nodes: valueNode
                }
              );
            }
            continue;
          }
          isNull = variableValues[variableName] == null;
        }
        if (isNull && (0, _definition.isNonNullType)(argType)) {
          throw new _GraphQLError.GraphQLError(
            `Argument "${name}" of non-null type "${(0, _inspect.inspect)(
              argType
            )}" must not be null.`,
            {
              nodes: valueNode
            }
          );
        }
        const coercedValue = (0, _valueFromAST.valueFromAST)(
          valueNode,
          argType,
          variableValues
        );
        if (coercedValue === void 0) {
          throw new _GraphQLError.GraphQLError(
            `Argument "${name}" has invalid value ${(0, _printer.print)(
              valueNode
            )}.`,
            {
              nodes: valueNode
            }
          );
        }
        coercedValues[name] = coercedValue;
      }
      return coercedValues;
    }
    __name(getArgumentValues, "getArgumentValues");
    function getDirectiveValues(directiveDef, node, variableValues) {
      var _node$directives;
      const directiveNode = (_node$directives = node.directives) === null || _node$directives === void 0 ? void 0 : _node$directives.find(
        (directive) => directive.name.value === directiveDef.name
      );
      if (directiveNode) {
        return getArgumentValues(directiveDef, directiveNode, variableValues);
      }
    }
    __name(getDirectiveValues, "getDirectiveValues");
    function hasOwnProperty(obj, prop) {
      return Object.prototype.hasOwnProperty.call(obj, prop);
    }
    __name(hasOwnProperty, "hasOwnProperty");
  }
});

// ../../node_modules/graphql/execution/collectFields.js
var require_collectFields = __commonJS({
  "../../node_modules/graphql/execution/collectFields.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.collectFields = collectFields;
    exports.collectSubfields = collectSubfields;
    var _kinds = require_kinds();
    var _definition = require_definition();
    var _directives = require_directives();
    var _typeFromAST = require_typeFromAST();
    var _values = require_values();
    function collectFields(schema, fragments, variableValues, runtimeType, selectionSet) {
      const fields = /* @__PURE__ */ new Map();
      collectFieldsImpl(
        schema,
        fragments,
        variableValues,
        runtimeType,
        selectionSet,
        fields,
        /* @__PURE__ */ new Set()
      );
      return fields;
    }
    __name(collectFields, "collectFields");
    function collectSubfields(schema, fragments, variableValues, returnType, fieldNodes) {
      const subFieldNodes = /* @__PURE__ */ new Map();
      const visitedFragmentNames = /* @__PURE__ */ new Set();
      for (const node of fieldNodes) {
        if (node.selectionSet) {
          collectFieldsImpl(
            schema,
            fragments,
            variableValues,
            returnType,
            node.selectionSet,
            subFieldNodes,
            visitedFragmentNames
          );
        }
      }
      return subFieldNodes;
    }
    __name(collectSubfields, "collectSubfields");
    function collectFieldsImpl(schema, fragments, variableValues, runtimeType, selectionSet, fields, visitedFragmentNames) {
      for (const selection of selectionSet.selections) {
        switch (selection.kind) {
          case _kinds.Kind.FIELD: {
            if (!shouldIncludeNode(variableValues, selection)) {
              continue;
            }
            const name = getFieldEntryKey(selection);
            const fieldList = fields.get(name);
            if (fieldList !== void 0) {
              fieldList.push(selection);
            } else {
              fields.set(name, [selection]);
            }
            break;
          }
          case _kinds.Kind.INLINE_FRAGMENT: {
            if (!shouldIncludeNode(variableValues, selection) || !doesFragmentConditionMatch(schema, selection, runtimeType)) {
              continue;
            }
            collectFieldsImpl(
              schema,
              fragments,
              variableValues,
              runtimeType,
              selection.selectionSet,
              fields,
              visitedFragmentNames
            );
            break;
          }
          case _kinds.Kind.FRAGMENT_SPREAD: {
            const fragName = selection.name.value;
            if (visitedFragmentNames.has(fragName) || !shouldIncludeNode(variableValues, selection)) {
              continue;
            }
            visitedFragmentNames.add(fragName);
            const fragment = fragments[fragName];
            if (!fragment || !doesFragmentConditionMatch(schema, fragment, runtimeType)) {
              continue;
            }
            collectFieldsImpl(
              schema,
              fragments,
              variableValues,
              runtimeType,
              fragment.selectionSet,
              fields,
              visitedFragmentNames
            );
            break;
          }
        }
      }
    }
    __name(collectFieldsImpl, "collectFieldsImpl");
    function shouldIncludeNode(variableValues, node) {
      const skip = (0, _values.getDirectiveValues)(
        _directives.GraphQLSkipDirective,
        node,
        variableValues
      );
      if ((skip === null || skip === void 0 ? void 0 : skip.if) === true) {
        return false;
      }
      const include = (0, _values.getDirectiveValues)(
        _directives.GraphQLIncludeDirective,
        node,
        variableValues
      );
      if ((include === null || include === void 0 ? void 0 : include.if) === false) {
        return false;
      }
      return true;
    }
    __name(shouldIncludeNode, "shouldIncludeNode");
    function doesFragmentConditionMatch(schema, fragment, type) {
      const typeConditionNode = fragment.typeCondition;
      if (!typeConditionNode) {
        return true;
      }
      const conditionalType = (0, _typeFromAST.typeFromAST)(
        schema,
        typeConditionNode
      );
      if (conditionalType === type) {
        return true;
      }
      if ((0, _definition.isAbstractType)(conditionalType)) {
        return schema.isSubType(conditionalType, type);
      }
      return false;
    }
    __name(doesFragmentConditionMatch, "doesFragmentConditionMatch");
    function getFieldEntryKey(node) {
      return node.alias ? node.alias.value : node.name.value;
    }
    __name(getFieldEntryKey, "getFieldEntryKey");
  }
});

// ../../node_modules/graphql/validation/rules/SingleFieldSubscriptionsRule.js
var require_SingleFieldSubscriptionsRule = __commonJS({
  "../../node_modules/graphql/validation/rules/SingleFieldSubscriptionsRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.SingleFieldSubscriptionsRule = SingleFieldSubscriptionsRule;
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _collectFields = require_collectFields();
    function SingleFieldSubscriptionsRule(context) {
      return {
        OperationDefinition(node) {
          if (node.operation === "subscription") {
            const schema = context.getSchema();
            const subscriptionType = schema.getSubscriptionType();
            if (subscriptionType) {
              const operationName = node.name ? node.name.value : null;
              const variableValues = /* @__PURE__ */ Object.create(null);
              const document = context.getDocument();
              const fragments = /* @__PURE__ */ Object.create(null);
              for (const definition of document.definitions) {
                if (definition.kind === _kinds.Kind.FRAGMENT_DEFINITION) {
                  fragments[definition.name.value] = definition;
                }
              }
              const fields = (0, _collectFields.collectFields)(
                schema,
                fragments,
                variableValues,
                subscriptionType,
                node.selectionSet
              );
              if (fields.size > 1) {
                const fieldSelectionLists = [...fields.values()];
                const extraFieldSelectionLists = fieldSelectionLists.slice(1);
                const extraFieldSelections = extraFieldSelectionLists.flat();
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    operationName != null ? `Subscription "${operationName}" must select only one top level field.` : "Anonymous Subscription must select only one top level field.",
                    {
                      nodes: extraFieldSelections
                    }
                  )
                );
              }
              for (const fieldNodes of fields.values()) {
                const field = fieldNodes[0];
                const fieldName = field.name.value;
                if (fieldName.startsWith("__")) {
                  context.reportError(
                    new _GraphQLError.GraphQLError(
                      operationName != null ? `Subscription "${operationName}" must not select an introspection top level field.` : "Anonymous Subscription must not select an introspection top level field.",
                      {
                        nodes: fieldNodes
                      }
                    )
                  );
                }
              }
            }
          }
        }
      };
    }
    __name(SingleFieldSubscriptionsRule, "SingleFieldSubscriptionsRule");
  }
});

// ../../node_modules/graphql/jsutils/groupBy.js
var require_groupBy = __commonJS({
  "../../node_modules/graphql/jsutils/groupBy.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.groupBy = groupBy;
    function groupBy(list, keyFn) {
      const result = /* @__PURE__ */ new Map();
      for (const item of list) {
        const key = keyFn(item);
        const group = result.get(key);
        if (group === void 0) {
          result.set(key, [item]);
        } else {
          group.push(item);
        }
      }
      return result;
    }
    __name(groupBy, "groupBy");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueArgumentDefinitionNamesRule.js
var require_UniqueArgumentDefinitionNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueArgumentDefinitionNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueArgumentDefinitionNamesRule = UniqueArgumentDefinitionNamesRule;
    var _groupBy = require_groupBy();
    var _GraphQLError = require_GraphQLError();
    function UniqueArgumentDefinitionNamesRule(context) {
      return {
        DirectiveDefinition(directiveNode) {
          var _directiveNode$argume;
          const argumentNodes = (_directiveNode$argume = directiveNode.arguments) !== null && _directiveNode$argume !== void 0 ? _directiveNode$argume : [];
          return checkArgUniqueness(`@${directiveNode.name.value}`, argumentNodes);
        },
        InterfaceTypeDefinition: checkArgUniquenessPerField,
        InterfaceTypeExtension: checkArgUniquenessPerField,
        ObjectTypeDefinition: checkArgUniquenessPerField,
        ObjectTypeExtension: checkArgUniquenessPerField
      };
      function checkArgUniquenessPerField(typeNode) {
        var _typeNode$fields;
        const typeName = typeNode.name.value;
        const fieldNodes = (_typeNode$fields = typeNode.fields) !== null && _typeNode$fields !== void 0 ? _typeNode$fields : [];
        for (const fieldDef of fieldNodes) {
          var _fieldDef$arguments;
          const fieldName = fieldDef.name.value;
          const argumentNodes = (_fieldDef$arguments = fieldDef.arguments) !== null && _fieldDef$arguments !== void 0 ? _fieldDef$arguments : [];
          checkArgUniqueness(`${typeName}.${fieldName}`, argumentNodes);
        }
        return false;
      }
      __name(checkArgUniquenessPerField, "checkArgUniquenessPerField");
      function checkArgUniqueness(parentName, argumentNodes) {
        const seenArgs = (0, _groupBy.groupBy)(
          argumentNodes,
          (arg) => arg.name.value
        );
        for (const [argName, argNodes] of seenArgs) {
          if (argNodes.length > 1) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Argument "${parentName}(${argName}:)" can only be defined once.`,
                {
                  nodes: argNodes.map((node) => node.name)
                }
              )
            );
          }
        }
        return false;
      }
      __name(checkArgUniqueness, "checkArgUniqueness");
    }
    __name(UniqueArgumentDefinitionNamesRule, "UniqueArgumentDefinitionNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueArgumentNamesRule.js
var require_UniqueArgumentNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueArgumentNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueArgumentNamesRule = UniqueArgumentNamesRule;
    var _groupBy = require_groupBy();
    var _GraphQLError = require_GraphQLError();
    function UniqueArgumentNamesRule(context) {
      return {
        Field: checkArgUniqueness,
        Directive: checkArgUniqueness
      };
      function checkArgUniqueness(parentNode) {
        var _parentNode$arguments;
        const argumentNodes = (_parentNode$arguments = parentNode.arguments) !== null && _parentNode$arguments !== void 0 ? _parentNode$arguments : [];
        const seenArgs = (0, _groupBy.groupBy)(
          argumentNodes,
          (arg) => arg.name.value
        );
        for (const [argName, argNodes] of seenArgs) {
          if (argNodes.length > 1) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `There can be only one argument named "${argName}".`,
                {
                  nodes: argNodes.map((node) => node.name)
                }
              )
            );
          }
        }
      }
      __name(checkArgUniqueness, "checkArgUniqueness");
    }
    __name(UniqueArgumentNamesRule, "UniqueArgumentNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueDirectiveNamesRule.js
var require_UniqueDirectiveNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueDirectiveNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueDirectiveNamesRule = UniqueDirectiveNamesRule;
    var _GraphQLError = require_GraphQLError();
    function UniqueDirectiveNamesRule(context) {
      const knownDirectiveNames = /* @__PURE__ */ Object.create(null);
      const schema = context.getSchema();
      return {
        DirectiveDefinition(node) {
          const directiveName = node.name.value;
          if (schema !== null && schema !== void 0 && schema.getDirective(directiveName)) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Directive "@${directiveName}" already exists in the schema. It cannot be redefined.`,
                {
                  nodes: node.name
                }
              )
            );
            return;
          }
          if (knownDirectiveNames[directiveName]) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `There can be only one directive named "@${directiveName}".`,
                {
                  nodes: [knownDirectiveNames[directiveName], node.name]
                }
              )
            );
          } else {
            knownDirectiveNames[directiveName] = node.name;
          }
          return false;
        }
      };
    }
    __name(UniqueDirectiveNamesRule, "UniqueDirectiveNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueDirectivesPerLocationRule.js
var require_UniqueDirectivesPerLocationRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueDirectivesPerLocationRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueDirectivesPerLocationRule = UniqueDirectivesPerLocationRule;
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _predicates = require_predicates();
    var _directives = require_directives();
    function UniqueDirectivesPerLocationRule(context) {
      const uniqueDirectiveMap = /* @__PURE__ */ Object.create(null);
      const schema = context.getSchema();
      const definedDirectives = schema ? schema.getDirectives() : _directives.specifiedDirectives;
      for (const directive of definedDirectives) {
        uniqueDirectiveMap[directive.name] = !directive.isRepeatable;
      }
      const astDefinitions = context.getDocument().definitions;
      for (const def of astDefinitions) {
        if (def.kind === _kinds.Kind.DIRECTIVE_DEFINITION) {
          uniqueDirectiveMap[def.name.value] = !def.repeatable;
        }
      }
      const schemaDirectives = /* @__PURE__ */ Object.create(null);
      const typeDirectivesMap = /* @__PURE__ */ Object.create(null);
      return {
        // Many different AST nodes may contain directives. Rather than listing
        // them all, just listen for entering any node, and check to see if it
        // defines any directives.
        enter(node) {
          if (!("directives" in node) || !node.directives) {
            return;
          }
          let seenDirectives;
          if (node.kind === _kinds.Kind.SCHEMA_DEFINITION || node.kind === _kinds.Kind.SCHEMA_EXTENSION) {
            seenDirectives = schemaDirectives;
          } else if ((0, _predicates.isTypeDefinitionNode)(node) || (0, _predicates.isTypeExtensionNode)(node)) {
            const typeName = node.name.value;
            seenDirectives = typeDirectivesMap[typeName];
            if (seenDirectives === void 0) {
              typeDirectivesMap[typeName] = seenDirectives = /* @__PURE__ */ Object.create(null);
            }
          } else {
            seenDirectives = /* @__PURE__ */ Object.create(null);
          }
          for (const directive of node.directives) {
            const directiveName = directive.name.value;
            if (uniqueDirectiveMap[directiveName]) {
              if (seenDirectives[directiveName]) {
                context.reportError(
                  new _GraphQLError.GraphQLError(
                    `The directive "@${directiveName}" can only be used once at this location.`,
                    {
                      nodes: [seenDirectives[directiveName], directive]
                    }
                  )
                );
              } else {
                seenDirectives[directiveName] = directive;
              }
            }
          }
        }
      };
    }
    __name(UniqueDirectivesPerLocationRule, "UniqueDirectivesPerLocationRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueEnumValueNamesRule.js
var require_UniqueEnumValueNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueEnumValueNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueEnumValueNamesRule = UniqueEnumValueNamesRule;
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    function UniqueEnumValueNamesRule(context) {
      const schema = context.getSchema();
      const existingTypeMap = schema ? schema.getTypeMap() : /* @__PURE__ */ Object.create(null);
      const knownValueNames = /* @__PURE__ */ Object.create(null);
      return {
        EnumTypeDefinition: checkValueUniqueness,
        EnumTypeExtension: checkValueUniqueness
      };
      function checkValueUniqueness(node) {
        var _node$values;
        const typeName = node.name.value;
        if (!knownValueNames[typeName]) {
          knownValueNames[typeName] = /* @__PURE__ */ Object.create(null);
        }
        const valueNodes = (_node$values = node.values) !== null && _node$values !== void 0 ? _node$values : [];
        const valueNames = knownValueNames[typeName];
        for (const valueDef of valueNodes) {
          const valueName = valueDef.name.value;
          const existingType = existingTypeMap[typeName];
          if ((0, _definition.isEnumType)(existingType) && existingType.getValue(valueName)) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Enum value "${typeName}.${valueName}" already exists in the schema. It cannot also be defined in this type extension.`,
                {
                  nodes: valueDef.name
                }
              )
            );
          } else if (valueNames[valueName]) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Enum value "${typeName}.${valueName}" can only be defined once.`,
                {
                  nodes: [valueNames[valueName], valueDef.name]
                }
              )
            );
          } else {
            valueNames[valueName] = valueDef.name;
          }
        }
        return false;
      }
      __name(checkValueUniqueness, "checkValueUniqueness");
    }
    __name(UniqueEnumValueNamesRule, "UniqueEnumValueNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueFieldDefinitionNamesRule.js
var require_UniqueFieldDefinitionNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueFieldDefinitionNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueFieldDefinitionNamesRule = UniqueFieldDefinitionNamesRule;
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    function UniqueFieldDefinitionNamesRule(context) {
      const schema = context.getSchema();
      const existingTypeMap = schema ? schema.getTypeMap() : /* @__PURE__ */ Object.create(null);
      const knownFieldNames = /* @__PURE__ */ Object.create(null);
      return {
        InputObjectTypeDefinition: checkFieldUniqueness,
        InputObjectTypeExtension: checkFieldUniqueness,
        InterfaceTypeDefinition: checkFieldUniqueness,
        InterfaceTypeExtension: checkFieldUniqueness,
        ObjectTypeDefinition: checkFieldUniqueness,
        ObjectTypeExtension: checkFieldUniqueness
      };
      function checkFieldUniqueness(node) {
        var _node$fields;
        const typeName = node.name.value;
        if (!knownFieldNames[typeName]) {
          knownFieldNames[typeName] = /* @__PURE__ */ Object.create(null);
        }
        const fieldNodes = (_node$fields = node.fields) !== null && _node$fields !== void 0 ? _node$fields : [];
        const fieldNames = knownFieldNames[typeName];
        for (const fieldDef of fieldNodes) {
          const fieldName = fieldDef.name.value;
          if (hasField(existingTypeMap[typeName], fieldName)) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Field "${typeName}.${fieldName}" already exists in the schema. It cannot also be defined in this type extension.`,
                {
                  nodes: fieldDef.name
                }
              )
            );
          } else if (fieldNames[fieldName]) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Field "${typeName}.${fieldName}" can only be defined once.`,
                {
                  nodes: [fieldNames[fieldName], fieldDef.name]
                }
              )
            );
          } else {
            fieldNames[fieldName] = fieldDef.name;
          }
        }
        return false;
      }
      __name(checkFieldUniqueness, "checkFieldUniqueness");
    }
    __name(UniqueFieldDefinitionNamesRule, "UniqueFieldDefinitionNamesRule");
    function hasField(type, fieldName) {
      if ((0, _definition.isObjectType)(type) || (0, _definition.isInterfaceType)(type) || (0, _definition.isInputObjectType)(type)) {
        return type.getFields()[fieldName] != null;
      }
      return false;
    }
    __name(hasField, "hasField");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueFragmentNamesRule.js
var require_UniqueFragmentNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueFragmentNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueFragmentNamesRule = UniqueFragmentNamesRule;
    var _GraphQLError = require_GraphQLError();
    function UniqueFragmentNamesRule(context) {
      const knownFragmentNames = /* @__PURE__ */ Object.create(null);
      return {
        OperationDefinition: /* @__PURE__ */ __name(() => false, "OperationDefinition"),
        FragmentDefinition(node) {
          const fragmentName = node.name.value;
          if (knownFragmentNames[fragmentName]) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `There can be only one fragment named "${fragmentName}".`,
                {
                  nodes: [knownFragmentNames[fragmentName], node.name]
                }
              )
            );
          } else {
            knownFragmentNames[fragmentName] = node.name;
          }
          return false;
        }
      };
    }
    __name(UniqueFragmentNamesRule, "UniqueFragmentNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueInputFieldNamesRule.js
var require_UniqueInputFieldNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueInputFieldNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueInputFieldNamesRule = UniqueInputFieldNamesRule;
    var _invariant = require_invariant();
    var _GraphQLError = require_GraphQLError();
    function UniqueInputFieldNamesRule(context) {
      const knownNameStack = [];
      let knownNames = /* @__PURE__ */ Object.create(null);
      return {
        ObjectValue: {
          enter() {
            knownNameStack.push(knownNames);
            knownNames = /* @__PURE__ */ Object.create(null);
          },
          leave() {
            const prevKnownNames = knownNameStack.pop();
            prevKnownNames || (0, _invariant.invariant)(false);
            knownNames = prevKnownNames;
          }
        },
        ObjectField(node) {
          const fieldName = node.name.value;
          if (knownNames[fieldName]) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `There can be only one input field named "${fieldName}".`,
                {
                  nodes: [knownNames[fieldName], node.name]
                }
              )
            );
          } else {
            knownNames[fieldName] = node.name;
          }
        }
      };
    }
    __name(UniqueInputFieldNamesRule, "UniqueInputFieldNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueOperationNamesRule.js
var require_UniqueOperationNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueOperationNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueOperationNamesRule = UniqueOperationNamesRule;
    var _GraphQLError = require_GraphQLError();
    function UniqueOperationNamesRule(context) {
      const knownOperationNames = /* @__PURE__ */ Object.create(null);
      return {
        OperationDefinition(node) {
          const operationName = node.name;
          if (operationName) {
            if (knownOperationNames[operationName.value]) {
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `There can be only one operation named "${operationName.value}".`,
                  {
                    nodes: [
                      knownOperationNames[operationName.value],
                      operationName
                    ]
                  }
                )
              );
            } else {
              knownOperationNames[operationName.value] = operationName;
            }
          }
          return false;
        },
        FragmentDefinition: /* @__PURE__ */ __name(() => false, "FragmentDefinition")
      };
    }
    __name(UniqueOperationNamesRule, "UniqueOperationNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueOperationTypesRule.js
var require_UniqueOperationTypesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueOperationTypesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueOperationTypesRule = UniqueOperationTypesRule;
    var _GraphQLError = require_GraphQLError();
    function UniqueOperationTypesRule(context) {
      const schema = context.getSchema();
      const definedOperationTypes = /* @__PURE__ */ Object.create(null);
      const existingOperationTypes = schema ? {
        query: schema.getQueryType(),
        mutation: schema.getMutationType(),
        subscription: schema.getSubscriptionType()
      } : {};
      return {
        SchemaDefinition: checkOperationTypes,
        SchemaExtension: checkOperationTypes
      };
      function checkOperationTypes(node) {
        var _node$operationTypes;
        const operationTypesNodes = (_node$operationTypes = node.operationTypes) !== null && _node$operationTypes !== void 0 ? _node$operationTypes : [];
        for (const operationType of operationTypesNodes) {
          const operation = operationType.operation;
          const alreadyDefinedOperationType = definedOperationTypes[operation];
          if (existingOperationTypes[operation]) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Type for ${operation} already defined in the schema. It cannot be redefined.`,
                {
                  nodes: operationType
                }
              )
            );
          } else if (alreadyDefinedOperationType) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `There can be only one ${operation} type in schema.`,
                {
                  nodes: [alreadyDefinedOperationType, operationType]
                }
              )
            );
          } else {
            definedOperationTypes[operation] = operationType;
          }
        }
        return false;
      }
      __name(checkOperationTypes, "checkOperationTypes");
    }
    __name(UniqueOperationTypesRule, "UniqueOperationTypesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueTypeNamesRule.js
var require_UniqueTypeNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueTypeNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueTypeNamesRule = UniqueTypeNamesRule;
    var _GraphQLError = require_GraphQLError();
    function UniqueTypeNamesRule(context) {
      const knownTypeNames = /* @__PURE__ */ Object.create(null);
      const schema = context.getSchema();
      return {
        ScalarTypeDefinition: checkTypeName,
        ObjectTypeDefinition: checkTypeName,
        InterfaceTypeDefinition: checkTypeName,
        UnionTypeDefinition: checkTypeName,
        EnumTypeDefinition: checkTypeName,
        InputObjectTypeDefinition: checkTypeName
      };
      function checkTypeName(node) {
        const typeName = node.name.value;
        if (schema !== null && schema !== void 0 && schema.getType(typeName)) {
          context.reportError(
            new _GraphQLError.GraphQLError(
              `Type "${typeName}" already exists in the schema. It cannot also be defined in this type definition.`,
              {
                nodes: node.name
              }
            )
          );
          return;
        }
        if (knownTypeNames[typeName]) {
          context.reportError(
            new _GraphQLError.GraphQLError(
              `There can be only one type named "${typeName}".`,
              {
                nodes: [knownTypeNames[typeName], node.name]
              }
            )
          );
        } else {
          knownTypeNames[typeName] = node.name;
        }
        return false;
      }
      __name(checkTypeName, "checkTypeName");
    }
    __name(UniqueTypeNamesRule, "UniqueTypeNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/UniqueVariableNamesRule.js
var require_UniqueVariableNamesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/UniqueVariableNamesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.UniqueVariableNamesRule = UniqueVariableNamesRule;
    var _groupBy = require_groupBy();
    var _GraphQLError = require_GraphQLError();
    function UniqueVariableNamesRule(context) {
      return {
        OperationDefinition(operationNode) {
          var _operationNode$variab;
          const variableDefinitions = (_operationNode$variab = operationNode.variableDefinitions) !== null && _operationNode$variab !== void 0 ? _operationNode$variab : [];
          const seenVariableDefinitions = (0, _groupBy.groupBy)(
            variableDefinitions,
            (node) => node.variable.name.value
          );
          for (const [variableName, variableNodes] of seenVariableDefinitions) {
            if (variableNodes.length > 1) {
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `There can be only one variable named "$${variableName}".`,
                  {
                    nodes: variableNodes.map((node) => node.variable.name)
                  }
                )
              );
            }
          }
        }
      };
    }
    __name(UniqueVariableNamesRule, "UniqueVariableNamesRule");
  }
});

// ../../node_modules/graphql/validation/rules/ValuesOfCorrectTypeRule.js
var require_ValuesOfCorrectTypeRule = __commonJS({
  "../../node_modules/graphql/validation/rules/ValuesOfCorrectTypeRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.ValuesOfCorrectTypeRule = ValuesOfCorrectTypeRule;
    var _didYouMean = require_didYouMean();
    var _inspect = require_inspect();
    var _keyMap = require_keyMap();
    var _suggestionList = require_suggestionList();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _definition = require_definition();
    function ValuesOfCorrectTypeRule(context) {
      let variableDefinitions = {};
      return {
        OperationDefinition: {
          enter() {
            variableDefinitions = {};
          }
        },
        VariableDefinition(definition) {
          variableDefinitions[definition.variable.name.value] = definition;
        },
        ListValue(node) {
          const type = (0, _definition.getNullableType)(
            context.getParentInputType()
          );
          if (!(0, _definition.isListType)(type)) {
            isValidValueNode(context, node);
            return false;
          }
        },
        ObjectValue(node) {
          const type = (0, _definition.getNamedType)(context.getInputType());
          if (!(0, _definition.isInputObjectType)(type)) {
            isValidValueNode(context, node);
            return false;
          }
          const fieldNodeMap = (0, _keyMap.keyMap)(
            node.fields,
            (field) => field.name.value
          );
          for (const fieldDef of Object.values(type.getFields())) {
            const fieldNode = fieldNodeMap[fieldDef.name];
            if (!fieldNode && (0, _definition.isRequiredInputField)(fieldDef)) {
              const typeStr = (0, _inspect.inspect)(fieldDef.type);
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Field "${type.name}.${fieldDef.name}" of required type "${typeStr}" was not provided.`,
                  {
                    nodes: node
                  }
                )
              );
            }
          }
          if (type.isOneOf) {
            validateOneOfInputObject(
              context,
              node,
              type,
              fieldNodeMap,
              variableDefinitions
            );
          }
        },
        ObjectField(node) {
          const parentType = (0, _definition.getNamedType)(
            context.getParentInputType()
          );
          const fieldType = context.getInputType();
          if (!fieldType && (0, _definition.isInputObjectType)(parentType)) {
            const suggestions = (0, _suggestionList.suggestionList)(
              node.name.value,
              Object.keys(parentType.getFields())
            );
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Field "${node.name.value}" is not defined by type "${parentType.name}".` + (0, _didYouMean.didYouMean)(suggestions),
                {
                  nodes: node
                }
              )
            );
          }
        },
        NullValue(node) {
          const type = context.getInputType();
          if ((0, _definition.isNonNullType)(type)) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Expected value of type "${(0, _inspect.inspect)(
                  type
                )}", found ${(0, _printer.print)(node)}.`,
                {
                  nodes: node
                }
              )
            );
          }
        },
        EnumValue: /* @__PURE__ */ __name((node) => isValidValueNode(context, node), "EnumValue"),
        IntValue: /* @__PURE__ */ __name((node) => isValidValueNode(context, node), "IntValue"),
        FloatValue: /* @__PURE__ */ __name((node) => isValidValueNode(context, node), "FloatValue"),
        StringValue: /* @__PURE__ */ __name((node) => isValidValueNode(context, node), "StringValue"),
        BooleanValue: /* @__PURE__ */ __name((node) => isValidValueNode(context, node), "BooleanValue")
      };
    }
    __name(ValuesOfCorrectTypeRule, "ValuesOfCorrectTypeRule");
    function isValidValueNode(context, node) {
      const locationType = context.getInputType();
      if (!locationType) {
        return;
      }
      const type = (0, _definition.getNamedType)(locationType);
      if (!(0, _definition.isLeafType)(type)) {
        const typeStr = (0, _inspect.inspect)(locationType);
        context.reportError(
          new _GraphQLError.GraphQLError(
            `Expected value of type "${typeStr}", found ${(0, _printer.print)(
              node
            )}.`,
            {
              nodes: node
            }
          )
        );
        return;
      }
      try {
        const parseResult = type.parseLiteral(
          node,
          void 0
          /* variables */
        );
        if (parseResult === void 0) {
          const typeStr = (0, _inspect.inspect)(locationType);
          context.reportError(
            new _GraphQLError.GraphQLError(
              `Expected value of type "${typeStr}", found ${(0, _printer.print)(
                node
              )}.`,
              {
                nodes: node
              }
            )
          );
        }
      } catch (error) {
        const typeStr = (0, _inspect.inspect)(locationType);
        if (error instanceof _GraphQLError.GraphQLError) {
          context.reportError(error);
        } else {
          context.reportError(
            new _GraphQLError.GraphQLError(
              `Expected value of type "${typeStr}", found ${(0, _printer.print)(
                node
              )}; ` + error.message,
              {
                nodes: node,
                originalError: error
              }
            )
          );
        }
      }
    }
    __name(isValidValueNode, "isValidValueNode");
    function validateOneOfInputObject(context, node, type, fieldNodeMap, variableDefinitions) {
      var _fieldNodeMap$keys$;
      const keys = Object.keys(fieldNodeMap);
      const isNotExactlyOneField = keys.length !== 1;
      if (isNotExactlyOneField) {
        context.reportError(
          new _GraphQLError.GraphQLError(
            `OneOf Input Object "${type.name}" must specify exactly one key.`,
            {
              nodes: [node]
            }
          )
        );
        return;
      }
      const value = (_fieldNodeMap$keys$ = fieldNodeMap[keys[0]]) === null || _fieldNodeMap$keys$ === void 0 ? void 0 : _fieldNodeMap$keys$.value;
      const isNullLiteral = !value || value.kind === _kinds.Kind.NULL;
      const isVariable = (value === null || value === void 0 ? void 0 : value.kind) === _kinds.Kind.VARIABLE;
      if (isNullLiteral) {
        context.reportError(
          new _GraphQLError.GraphQLError(
            `Field "${type.name}.${keys[0]}" must be non-null.`,
            {
              nodes: [node]
            }
          )
        );
        return;
      }
      if (isVariable) {
        const variableName = value.name.value;
        const definition = variableDefinitions[variableName];
        const isNullableVariable = definition.type.kind !== _kinds.Kind.NON_NULL_TYPE;
        if (isNullableVariable) {
          context.reportError(
            new _GraphQLError.GraphQLError(
              `Variable "${variableName}" must be non-nullable to be used for OneOf Input Object "${type.name}".`,
              {
                nodes: [node]
              }
            )
          );
        }
      }
    }
    __name(validateOneOfInputObject, "validateOneOfInputObject");
  }
});

// ../../node_modules/graphql/validation/rules/VariablesAreInputTypesRule.js
var require_VariablesAreInputTypesRule = __commonJS({
  "../../node_modules/graphql/validation/rules/VariablesAreInputTypesRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.VariablesAreInputTypesRule = VariablesAreInputTypesRule;
    var _GraphQLError = require_GraphQLError();
    var _printer = require_printer();
    var _definition = require_definition();
    var _typeFromAST = require_typeFromAST();
    function VariablesAreInputTypesRule(context) {
      return {
        VariableDefinition(node) {
          const type = (0, _typeFromAST.typeFromAST)(
            context.getSchema(),
            node.type
          );
          if (type !== void 0 && !(0, _definition.isInputType)(type)) {
            const variableName = node.variable.name.value;
            const typeName = (0, _printer.print)(node.type);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `Variable "$${variableName}" cannot be non-input type "${typeName}".`,
                {
                  nodes: node.type
                }
              )
            );
          }
        }
      };
    }
    __name(VariablesAreInputTypesRule, "VariablesAreInputTypesRule");
  }
});

// ../../node_modules/graphql/validation/rules/VariablesInAllowedPositionRule.js
var require_VariablesInAllowedPositionRule = __commonJS({
  "../../node_modules/graphql/validation/rules/VariablesInAllowedPositionRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.VariablesInAllowedPositionRule = VariablesInAllowedPositionRule;
    var _inspect = require_inspect();
    var _GraphQLError = require_GraphQLError();
    var _kinds = require_kinds();
    var _definition = require_definition();
    var _typeComparators = require_typeComparators();
    var _typeFromAST = require_typeFromAST();
    function VariablesInAllowedPositionRule(context) {
      let varDefMap = /* @__PURE__ */ Object.create(null);
      return {
        OperationDefinition: {
          enter() {
            varDefMap = /* @__PURE__ */ Object.create(null);
          },
          leave(operation) {
            const usages = context.getRecursiveVariableUsages(operation);
            for (const { node, type, defaultValue } of usages) {
              const varName = node.name.value;
              const varDef = varDefMap[varName];
              if (varDef && type) {
                const schema = context.getSchema();
                const varType = (0, _typeFromAST.typeFromAST)(schema, varDef.type);
                if (varType && !allowedVariableUsage(
                  schema,
                  varType,
                  varDef.defaultValue,
                  type,
                  defaultValue
                )) {
                  const varTypeStr = (0, _inspect.inspect)(varType);
                  const typeStr = (0, _inspect.inspect)(type);
                  context.reportError(
                    new _GraphQLError.GraphQLError(
                      `Variable "$${varName}" of type "${varTypeStr}" used in position expecting type "${typeStr}".`,
                      {
                        nodes: [varDef, node]
                      }
                    )
                  );
                }
              }
            }
          }
        },
        VariableDefinition(node) {
          varDefMap[node.variable.name.value] = node;
        }
      };
    }
    __name(VariablesInAllowedPositionRule, "VariablesInAllowedPositionRule");
    function allowedVariableUsage(schema, varType, varDefaultValue, locationType, locationDefaultValue) {
      if ((0, _definition.isNonNullType)(locationType) && !(0, _definition.isNonNullType)(varType)) {
        const hasNonNullVariableDefaultValue = varDefaultValue != null && varDefaultValue.kind !== _kinds.Kind.NULL;
        const hasLocationDefaultValue = locationDefaultValue !== void 0;
        if (!hasNonNullVariableDefaultValue && !hasLocationDefaultValue) {
          return false;
        }
        const nullableLocationType = locationType.ofType;
        return (0, _typeComparators.isTypeSubTypeOf)(
          schema,
          varType,
          nullableLocationType
        );
      }
      return (0, _typeComparators.isTypeSubTypeOf)(schema, varType, locationType);
    }
    __name(allowedVariableUsage, "allowedVariableUsage");
  }
});

// ../../node_modules/graphql/validation/specifiedRules.js
var require_specifiedRules = __commonJS({
  "../../node_modules/graphql/validation/specifiedRules.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.specifiedSDLRules = exports.specifiedRules = exports.recommendedRules = void 0;
    var _ExecutableDefinitionsRule = require_ExecutableDefinitionsRule();
    var _FieldsOnCorrectTypeRule = require_FieldsOnCorrectTypeRule();
    var _FragmentsOnCompositeTypesRule = require_FragmentsOnCompositeTypesRule();
    var _KnownArgumentNamesRule = require_KnownArgumentNamesRule();
    var _KnownDirectivesRule = require_KnownDirectivesRule();
    var _KnownFragmentNamesRule = require_KnownFragmentNamesRule();
    var _KnownTypeNamesRule = require_KnownTypeNamesRule();
    var _LoneAnonymousOperationRule = require_LoneAnonymousOperationRule();
    var _LoneSchemaDefinitionRule = require_LoneSchemaDefinitionRule();
    var _MaxIntrospectionDepthRule = require_MaxIntrospectionDepthRule();
    var _NoFragmentCyclesRule = require_NoFragmentCyclesRule();
    var _NoUndefinedVariablesRule = require_NoUndefinedVariablesRule();
    var _NoUnusedFragmentsRule = require_NoUnusedFragmentsRule();
    var _NoUnusedVariablesRule = require_NoUnusedVariablesRule();
    var _OverlappingFieldsCanBeMergedRule = require_OverlappingFieldsCanBeMergedRule();
    var _PossibleFragmentSpreadsRule = require_PossibleFragmentSpreadsRule();
    var _PossibleTypeExtensionsRule = require_PossibleTypeExtensionsRule();
    var _ProvidedRequiredArgumentsRule = require_ProvidedRequiredArgumentsRule();
    var _ScalarLeafsRule = require_ScalarLeafsRule();
    var _SingleFieldSubscriptionsRule = require_SingleFieldSubscriptionsRule();
    var _UniqueArgumentDefinitionNamesRule = require_UniqueArgumentDefinitionNamesRule();
    var _UniqueArgumentNamesRule = require_UniqueArgumentNamesRule();
    var _UniqueDirectiveNamesRule = require_UniqueDirectiveNamesRule();
    var _UniqueDirectivesPerLocationRule = require_UniqueDirectivesPerLocationRule();
    var _UniqueEnumValueNamesRule = require_UniqueEnumValueNamesRule();
    var _UniqueFieldDefinitionNamesRule = require_UniqueFieldDefinitionNamesRule();
    var _UniqueFragmentNamesRule = require_UniqueFragmentNamesRule();
    var _UniqueInputFieldNamesRule = require_UniqueInputFieldNamesRule();
    var _UniqueOperationNamesRule = require_UniqueOperationNamesRule();
    var _UniqueOperationTypesRule = require_UniqueOperationTypesRule();
    var _UniqueTypeNamesRule = require_UniqueTypeNamesRule();
    var _UniqueVariableNamesRule = require_UniqueVariableNamesRule();
    var _ValuesOfCorrectTypeRule = require_ValuesOfCorrectTypeRule();
    var _VariablesAreInputTypesRule = require_VariablesAreInputTypesRule();
    var _VariablesInAllowedPositionRule = require_VariablesInAllowedPositionRule();
    var recommendedRules = Object.freeze([
      _MaxIntrospectionDepthRule.MaxIntrospectionDepthRule
    ]);
    exports.recommendedRules = recommendedRules;
    var specifiedRules = Object.freeze([
      _ExecutableDefinitionsRule.ExecutableDefinitionsRule,
      _UniqueOperationNamesRule.UniqueOperationNamesRule,
      _LoneAnonymousOperationRule.LoneAnonymousOperationRule,
      _SingleFieldSubscriptionsRule.SingleFieldSubscriptionsRule,
      _KnownTypeNamesRule.KnownTypeNamesRule,
      _FragmentsOnCompositeTypesRule.FragmentsOnCompositeTypesRule,
      _VariablesAreInputTypesRule.VariablesAreInputTypesRule,
      _ScalarLeafsRule.ScalarLeafsRule,
      _FieldsOnCorrectTypeRule.FieldsOnCorrectTypeRule,
      _UniqueFragmentNamesRule.UniqueFragmentNamesRule,
      _KnownFragmentNamesRule.KnownFragmentNamesRule,
      _NoUnusedFragmentsRule.NoUnusedFragmentsRule,
      _PossibleFragmentSpreadsRule.PossibleFragmentSpreadsRule,
      _NoFragmentCyclesRule.NoFragmentCyclesRule,
      _UniqueVariableNamesRule.UniqueVariableNamesRule,
      _NoUndefinedVariablesRule.NoUndefinedVariablesRule,
      _NoUnusedVariablesRule.NoUnusedVariablesRule,
      _KnownDirectivesRule.KnownDirectivesRule,
      _UniqueDirectivesPerLocationRule.UniqueDirectivesPerLocationRule,
      _KnownArgumentNamesRule.KnownArgumentNamesRule,
      _UniqueArgumentNamesRule.UniqueArgumentNamesRule,
      _ValuesOfCorrectTypeRule.ValuesOfCorrectTypeRule,
      _ProvidedRequiredArgumentsRule.ProvidedRequiredArgumentsRule,
      _VariablesInAllowedPositionRule.VariablesInAllowedPositionRule,
      _OverlappingFieldsCanBeMergedRule.OverlappingFieldsCanBeMergedRule,
      _UniqueInputFieldNamesRule.UniqueInputFieldNamesRule,
      ...recommendedRules
    ]);
    exports.specifiedRules = specifiedRules;
    var specifiedSDLRules = Object.freeze([
      _LoneSchemaDefinitionRule.LoneSchemaDefinitionRule,
      _UniqueOperationTypesRule.UniqueOperationTypesRule,
      _UniqueTypeNamesRule.UniqueTypeNamesRule,
      _UniqueEnumValueNamesRule.UniqueEnumValueNamesRule,
      _UniqueFieldDefinitionNamesRule.UniqueFieldDefinitionNamesRule,
      _UniqueArgumentDefinitionNamesRule.UniqueArgumentDefinitionNamesRule,
      _UniqueDirectiveNamesRule.UniqueDirectiveNamesRule,
      _KnownTypeNamesRule.KnownTypeNamesRule,
      _KnownDirectivesRule.KnownDirectivesRule,
      _UniqueDirectivesPerLocationRule.UniqueDirectivesPerLocationRule,
      _PossibleTypeExtensionsRule.PossibleTypeExtensionsRule,
      _KnownArgumentNamesRule.KnownArgumentNamesOnDirectivesRule,
      _UniqueArgumentNamesRule.UniqueArgumentNamesRule,
      _UniqueInputFieldNamesRule.UniqueInputFieldNamesRule,
      _ProvidedRequiredArgumentsRule.ProvidedRequiredArgumentsOnDirectivesRule
    ]);
    exports.specifiedSDLRules = specifiedSDLRules;
  }
});

// ../../node_modules/graphql/validation/ValidationContext.js
var require_ValidationContext = __commonJS({
  "../../node_modules/graphql/validation/ValidationContext.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.ValidationContext = exports.SDLValidationContext = exports.ASTValidationContext = void 0;
    var _kinds = require_kinds();
    var _visitor = require_visitor();
    var _TypeInfo = require_TypeInfo();
    var ASTValidationContext = class {
      static {
        __name(this, "ASTValidationContext");
      }
      constructor(ast, onError) {
        this._ast = ast;
        this._fragments = void 0;
        this._fragmentSpreads = /* @__PURE__ */ new Map();
        this._recursivelyReferencedFragments = /* @__PURE__ */ new Map();
        this._onError = onError;
      }
      get [Symbol.toStringTag]() {
        return "ASTValidationContext";
      }
      reportError(error) {
        this._onError(error);
      }
      getDocument() {
        return this._ast;
      }
      getFragment(name) {
        let fragments;
        if (this._fragments) {
          fragments = this._fragments;
        } else {
          fragments = /* @__PURE__ */ Object.create(null);
          for (const defNode of this.getDocument().definitions) {
            if (defNode.kind === _kinds.Kind.FRAGMENT_DEFINITION) {
              fragments[defNode.name.value] = defNode;
            }
          }
          this._fragments = fragments;
        }
        return fragments[name];
      }
      getFragmentSpreads(node) {
        let spreads = this._fragmentSpreads.get(node);
        if (!spreads) {
          spreads = [];
          const setsToVisit = [node];
          let set;
          while (set = setsToVisit.pop()) {
            for (const selection of set.selections) {
              if (selection.kind === _kinds.Kind.FRAGMENT_SPREAD) {
                spreads.push(selection);
              } else if (selection.selectionSet) {
                setsToVisit.push(selection.selectionSet);
              }
            }
          }
          this._fragmentSpreads.set(node, spreads);
        }
        return spreads;
      }
      getRecursivelyReferencedFragments(operation) {
        let fragments = this._recursivelyReferencedFragments.get(operation);
        if (!fragments) {
          fragments = [];
          const collectedNames = /* @__PURE__ */ Object.create(null);
          const nodesToVisit = [operation.selectionSet];
          let node;
          while (node = nodesToVisit.pop()) {
            for (const spread of this.getFragmentSpreads(node)) {
              const fragName = spread.name.value;
              if (collectedNames[fragName] !== true) {
                collectedNames[fragName] = true;
                const fragment = this.getFragment(fragName);
                if (fragment) {
                  fragments.push(fragment);
                  nodesToVisit.push(fragment.selectionSet);
                }
              }
            }
          }
          this._recursivelyReferencedFragments.set(operation, fragments);
        }
        return fragments;
      }
    };
    exports.ASTValidationContext = ASTValidationContext;
    var SDLValidationContext = class extends ASTValidationContext {
      static {
        __name(this, "SDLValidationContext");
      }
      constructor(ast, schema, onError) {
        super(ast, onError);
        this._schema = schema;
      }
      get [Symbol.toStringTag]() {
        return "SDLValidationContext";
      }
      getSchema() {
        return this._schema;
      }
    };
    exports.SDLValidationContext = SDLValidationContext;
    var ValidationContext = class extends ASTValidationContext {
      static {
        __name(this, "ValidationContext");
      }
      constructor(schema, ast, typeInfo, onError) {
        super(ast, onError);
        this._schema = schema;
        this._typeInfo = typeInfo;
        this._variableUsages = /* @__PURE__ */ new Map();
        this._recursiveVariableUsages = /* @__PURE__ */ new Map();
      }
      get [Symbol.toStringTag]() {
        return "ValidationContext";
      }
      getSchema() {
        return this._schema;
      }
      getVariableUsages(node) {
        let usages = this._variableUsages.get(node);
        if (!usages) {
          const newUsages = [];
          const typeInfo = new _TypeInfo.TypeInfo(this._schema);
          (0, _visitor.visit)(
            node,
            (0, _TypeInfo.visitWithTypeInfo)(typeInfo, {
              VariableDefinition: /* @__PURE__ */ __name(() => false, "VariableDefinition"),
              Variable(variable) {
                newUsages.push({
                  node: variable,
                  type: typeInfo.getInputType(),
                  defaultValue: typeInfo.getDefaultValue()
                });
              }
            })
          );
          usages = newUsages;
          this._variableUsages.set(node, usages);
        }
        return usages;
      }
      getRecursiveVariableUsages(operation) {
        let usages = this._recursiveVariableUsages.get(operation);
        if (!usages) {
          usages = this.getVariableUsages(operation);
          for (const frag of this.getRecursivelyReferencedFragments(operation)) {
            usages = usages.concat(this.getVariableUsages(frag));
          }
          this._recursiveVariableUsages.set(operation, usages);
        }
        return usages;
      }
      getType() {
        return this._typeInfo.getType();
      }
      getParentType() {
        return this._typeInfo.getParentType();
      }
      getInputType() {
        return this._typeInfo.getInputType();
      }
      getParentInputType() {
        return this._typeInfo.getParentInputType();
      }
      getFieldDef() {
        return this._typeInfo.getFieldDef();
      }
      getDirective() {
        return this._typeInfo.getDirective();
      }
      getArgument() {
        return this._typeInfo.getArgument();
      }
      getEnumValue() {
        return this._typeInfo.getEnumValue();
      }
    };
    exports.ValidationContext = ValidationContext;
  }
});

// ../../node_modules/graphql/validation/validate.js
var require_validate2 = __commonJS({
  "../../node_modules/graphql/validation/validate.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.assertValidSDL = assertValidSDL;
    exports.assertValidSDLExtension = assertValidSDLExtension;
    exports.validate = validate;
    exports.validateSDL = validateSDL;
    var _devAssert = require_devAssert();
    var _GraphQLError = require_GraphQLError();
    var _visitor = require_visitor();
    var _validate = require_validate();
    var _TypeInfo = require_TypeInfo();
    var _specifiedRules = require_specifiedRules();
    var _ValidationContext = require_ValidationContext();
    function validate(schema, documentAST, rules = _specifiedRules.specifiedRules, options, typeInfo = new _TypeInfo.TypeInfo(schema)) {
      var _options$maxErrors;
      const maxErrors = (_options$maxErrors = options === null || options === void 0 ? void 0 : options.maxErrors) !== null && _options$maxErrors !== void 0 ? _options$maxErrors : 100;
      documentAST || (0, _devAssert.devAssert)(false, "Must provide document.");
      (0, _validate.assertValidSchema)(schema);
      const abortObj = Object.freeze({});
      const errors = [];
      const context = new _ValidationContext.ValidationContext(
        schema,
        documentAST,
        typeInfo,
        (error) => {
          if (errors.length >= maxErrors) {
            errors.push(
              new _GraphQLError.GraphQLError(
                "Too many validation errors, error limit reached. Validation aborted."
              )
            );
            throw abortObj;
          }
          errors.push(error);
        }
      );
      const visitor = (0, _visitor.visitInParallel)(
        rules.map((rule) => rule(context))
      );
      try {
        (0, _visitor.visit)(
          documentAST,
          (0, _TypeInfo.visitWithTypeInfo)(typeInfo, visitor)
        );
      } catch (e) {
        if (e !== abortObj) {
          throw e;
        }
      }
      return errors;
    }
    __name(validate, "validate");
    function validateSDL(documentAST, schemaToExtend, rules = _specifiedRules.specifiedSDLRules) {
      const errors = [];
      const context = new _ValidationContext.SDLValidationContext(
        documentAST,
        schemaToExtend,
        (error) => {
          errors.push(error);
        }
      );
      const visitors = rules.map((rule) => rule(context));
      (0, _visitor.visit)(documentAST, (0, _visitor.visitInParallel)(visitors));
      return errors;
    }
    __name(validateSDL, "validateSDL");
    function assertValidSDL(documentAST) {
      const errors = validateSDL(documentAST);
      if (errors.length !== 0) {
        throw new Error(errors.map((error) => error.message).join("\n\n"));
      }
    }
    __name(assertValidSDL, "assertValidSDL");
    function assertValidSDLExtension(documentAST, schema) {
      const errors = validateSDL(documentAST, schema);
      if (errors.length !== 0) {
        throw new Error(errors.map((error) => error.message).join("\n\n"));
      }
    }
    __name(assertValidSDLExtension, "assertValidSDLExtension");
  }
});

// ../../node_modules/graphql/jsutils/memoize3.js
var require_memoize3 = __commonJS({
  "../../node_modules/graphql/jsutils/memoize3.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.memoize3 = memoize3;
    function memoize3(fn) {
      let cache0;
      return /* @__PURE__ */ __name(function memoized(a1, a2, a3) {
        if (cache0 === void 0) {
          cache0 = /* @__PURE__ */ new WeakMap();
        }
        let cache1 = cache0.get(a1);
        if (cache1 === void 0) {
          cache1 = /* @__PURE__ */ new WeakMap();
          cache0.set(a1, cache1);
        }
        let cache2 = cache1.get(a2);
        if (cache2 === void 0) {
          cache2 = /* @__PURE__ */ new WeakMap();
          cache1.set(a2, cache2);
        }
        let fnResult = cache2.get(a3);
        if (fnResult === void 0) {
          fnResult = fn(a1, a2, a3);
          cache2.set(a3, fnResult);
        }
        return fnResult;
      }, "memoized");
    }
    __name(memoize3, "memoize3");
  }
});

// ../../node_modules/graphql/jsutils/promiseForObject.js
var require_promiseForObject = __commonJS({
  "../../node_modules/graphql/jsutils/promiseForObject.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.promiseForObject = promiseForObject;
    function promiseForObject(object) {
      return Promise.all(Object.values(object)).then((resolvedValues) => {
        const resolvedObject = /* @__PURE__ */ Object.create(null);
        for (const [i, key] of Object.keys(object).entries()) {
          resolvedObject[key] = resolvedValues[i];
        }
        return resolvedObject;
      });
    }
    __name(promiseForObject, "promiseForObject");
  }
});

// ../../node_modules/graphql/jsutils/promiseReduce.js
var require_promiseReduce = __commonJS({
  "../../node_modules/graphql/jsutils/promiseReduce.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.promiseReduce = promiseReduce;
    var _isPromise = require_isPromise();
    function promiseReduce(values, callbackFn, initialValue) {
      let accumulator = initialValue;
      for (const value of values) {
        accumulator = (0, _isPromise.isPromise)(accumulator) ? accumulator.then((resolved) => callbackFn(resolved, value)) : callbackFn(accumulator, value);
      }
      return accumulator;
    }
    __name(promiseReduce, "promiseReduce");
  }
});

// ../../node_modules/graphql/jsutils/toError.js
var require_toError = __commonJS({
  "../../node_modules/graphql/jsutils/toError.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.toError = toError;
    var _inspect = require_inspect();
    function toError(thrownValue) {
      return thrownValue instanceof Error ? thrownValue : new NonErrorThrown(thrownValue);
    }
    __name(toError, "toError");
    var NonErrorThrown = class extends Error {
      static {
        __name(this, "NonErrorThrown");
      }
      constructor(thrownValue) {
        super("Unexpected error value: " + (0, _inspect.inspect)(thrownValue));
        this.name = "NonErrorThrown";
        this.thrownValue = thrownValue;
      }
    };
  }
});

// ../../node_modules/graphql/error/locatedError.js
var require_locatedError = __commonJS({
  "../../node_modules/graphql/error/locatedError.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.locatedError = locatedError;
    var _toError = require_toError();
    var _GraphQLError = require_GraphQLError();
    function locatedError(rawOriginalError, nodes, path2) {
      var _nodes;
      const originalError = (0, _toError.toError)(rawOriginalError);
      if (isLocatedGraphQLError(originalError)) {
        return originalError;
      }
      return new _GraphQLError.GraphQLError(originalError.message, {
        nodes: (_nodes = originalError.nodes) !== null && _nodes !== void 0 ? _nodes : nodes,
        source: originalError.source,
        positions: originalError.positions,
        path: path2,
        originalError
      });
    }
    __name(locatedError, "locatedError");
    function isLocatedGraphQLError(error) {
      return Array.isArray(error.path);
    }
    __name(isLocatedGraphQLError, "isLocatedGraphQLError");
  }
});

// ../../node_modules/graphql/execution/execute.js
var require_execute = __commonJS({
  "../../node_modules/graphql/execution/execute.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.assertValidExecutionArguments = assertValidExecutionArguments;
    exports.buildExecutionContext = buildExecutionContext;
    exports.buildResolveInfo = buildResolveInfo;
    exports.defaultTypeResolver = exports.defaultFieldResolver = void 0;
    exports.execute = execute;
    exports.executeSync = executeSync;
    exports.getFieldDef = getFieldDef;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _isIterableObject = require_isIterableObject();
    var _isObjectLike = require_isObjectLike();
    var _isPromise = require_isPromise();
    var _memoize = require_memoize3();
    var _Path = require_Path();
    var _promiseForObject = require_promiseForObject();
    var _promiseReduce = require_promiseReduce();
    var _GraphQLError = require_GraphQLError();
    var _locatedError = require_locatedError();
    var _ast = require_ast();
    var _kinds = require_kinds();
    var _definition = require_definition();
    var _introspection = require_introspection();
    var _validate = require_validate();
    var _collectFields = require_collectFields();
    var _values = require_values();
    var collectSubfields = (0, _memoize.memoize3)(
      (exeContext, returnType, fieldNodes) => (0, _collectFields.collectSubfields)(
        exeContext.schema,
        exeContext.fragments,
        exeContext.variableValues,
        returnType,
        fieldNodes
      )
    );
    function execute(args) {
      arguments.length < 2 || (0, _devAssert.devAssert)(
        false,
        "graphql@16 dropped long-deprecated support for positional arguments, please pass an object instead."
      );
      const { schema, document, variableValues, rootValue } = args;
      assertValidExecutionArguments(schema, document, variableValues);
      const exeContext = buildExecutionContext(args);
      if (!("schema" in exeContext)) {
        return {
          errors: exeContext
        };
      }
      try {
        const { operation } = exeContext;
        const result = executeOperation(exeContext, operation, rootValue);
        if ((0, _isPromise.isPromise)(result)) {
          return result.then(
            (data) => buildResponse(data, exeContext.errors),
            (error) => {
              exeContext.errors.push(error);
              return buildResponse(null, exeContext.errors);
            }
          );
        }
        return buildResponse(result, exeContext.errors);
      } catch (error) {
        exeContext.errors.push(error);
        return buildResponse(null, exeContext.errors);
      }
    }
    __name(execute, "execute");
    function executeSync(args) {
      const result = execute(args);
      if ((0, _isPromise.isPromise)(result)) {
        throw new Error("GraphQL execution failed to complete synchronously.");
      }
      return result;
    }
    __name(executeSync, "executeSync");
    function buildResponse(data, errors) {
      return errors.length === 0 ? {
        data
      } : {
        errors,
        data
      };
    }
    __name(buildResponse, "buildResponse");
    function assertValidExecutionArguments(schema, document, rawVariableValues) {
      document || (0, _devAssert.devAssert)(false, "Must provide document.");
      (0, _validate.assertValidSchema)(schema);
      rawVariableValues == null || (0, _isObjectLike.isObjectLike)(rawVariableValues) || (0, _devAssert.devAssert)(
        false,
        "Variables must be provided as an Object where each property is a variable value. Perhaps look to see if an unparsed JSON string was provided."
      );
    }
    __name(assertValidExecutionArguments, "assertValidExecutionArguments");
    function buildExecutionContext(args) {
      var _definition$name, _operation$variableDe;
      const {
        schema,
        document,
        rootValue,
        contextValue,
        variableValues: rawVariableValues,
        operationName,
        fieldResolver,
        typeResolver,
        subscribeFieldResolver
      } = args;
      let operation;
      const fragments = /* @__PURE__ */ Object.create(null);
      for (const definition of document.definitions) {
        switch (definition.kind) {
          case _kinds.Kind.OPERATION_DEFINITION:
            if (operationName == null) {
              if (operation !== void 0) {
                return [
                  new _GraphQLError.GraphQLError(
                    "Must provide operation name if query contains multiple operations."
                  )
                ];
              }
              operation = definition;
            } else if (((_definition$name = definition.name) === null || _definition$name === void 0 ? void 0 : _definition$name.value) === operationName) {
              operation = definition;
            }
            break;
          case _kinds.Kind.FRAGMENT_DEFINITION:
            fragments[definition.name.value] = definition;
            break;
          default:
        }
      }
      if (!operation) {
        if (operationName != null) {
          return [
            new _GraphQLError.GraphQLError(
              `Unknown operation named "${operationName}".`
            )
          ];
        }
        return [new _GraphQLError.GraphQLError("Must provide an operation.")];
      }
      const variableDefinitions = (_operation$variableDe = operation.variableDefinitions) !== null && _operation$variableDe !== void 0 ? _operation$variableDe : [];
      const coercedVariableValues = (0, _values.getVariableValues)(
        schema,
        variableDefinitions,
        rawVariableValues !== null && rawVariableValues !== void 0 ? rawVariableValues : {},
        {
          maxErrors: 50
        }
      );
      if (coercedVariableValues.errors) {
        return coercedVariableValues.errors;
      }
      return {
        schema,
        fragments,
        rootValue,
        contextValue,
        operation,
        variableValues: coercedVariableValues.coerced,
        fieldResolver: fieldResolver !== null && fieldResolver !== void 0 ? fieldResolver : defaultFieldResolver,
        typeResolver: typeResolver !== null && typeResolver !== void 0 ? typeResolver : defaultTypeResolver,
        subscribeFieldResolver: subscribeFieldResolver !== null && subscribeFieldResolver !== void 0 ? subscribeFieldResolver : defaultFieldResolver,
        errors: []
      };
    }
    __name(buildExecutionContext, "buildExecutionContext");
    function executeOperation(exeContext, operation, rootValue) {
      const rootType = exeContext.schema.getRootType(operation.operation);
      if (rootType == null) {
        throw new _GraphQLError.GraphQLError(
          `Schema is not configured to execute ${operation.operation} operation.`,
          {
            nodes: operation
          }
        );
      }
      const rootFields = (0, _collectFields.collectFields)(
        exeContext.schema,
        exeContext.fragments,
        exeContext.variableValues,
        rootType,
        operation.selectionSet
      );
      const path2 = void 0;
      switch (operation.operation) {
        case _ast.OperationTypeNode.QUERY:
          return executeFields(exeContext, rootType, rootValue, path2, rootFields);
        case _ast.OperationTypeNode.MUTATION:
          return executeFieldsSerially(
            exeContext,
            rootType,
            rootValue,
            path2,
            rootFields
          );
        case _ast.OperationTypeNode.SUBSCRIPTION:
          return executeFields(exeContext, rootType, rootValue, path2, rootFields);
      }
    }
    __name(executeOperation, "executeOperation");
    function executeFieldsSerially(exeContext, parentType, sourceValue, path2, fields) {
      return (0, _promiseReduce.promiseReduce)(
        fields.entries(),
        (results, [responseName, fieldNodes]) => {
          const fieldPath = (0, _Path.addPath)(path2, responseName, parentType.name);
          const result = executeField(
            exeContext,
            parentType,
            sourceValue,
            fieldNodes,
            fieldPath
          );
          if (result === void 0) {
            return results;
          }
          if ((0, _isPromise.isPromise)(result)) {
            return result.then((resolvedResult) => {
              results[responseName] = resolvedResult;
              return results;
            });
          }
          results[responseName] = result;
          return results;
        },
        /* @__PURE__ */ Object.create(null)
      );
    }
    __name(executeFieldsSerially, "executeFieldsSerially");
    function executeFields(exeContext, parentType, sourceValue, path2, fields) {
      const results = /* @__PURE__ */ Object.create(null);
      let containsPromise = false;
      try {
        for (const [responseName, fieldNodes] of fields.entries()) {
          const fieldPath = (0, _Path.addPath)(path2, responseName, parentType.name);
          const result = executeField(
            exeContext,
            parentType,
            sourceValue,
            fieldNodes,
            fieldPath
          );
          if (result !== void 0) {
            results[responseName] = result;
            if ((0, _isPromise.isPromise)(result)) {
              containsPromise = true;
            }
          }
        }
      } catch (error) {
        if (containsPromise) {
          return (0, _promiseForObject.promiseForObject)(results).finally(() => {
            throw error;
          });
        }
        throw error;
      }
      if (!containsPromise) {
        return results;
      }
      return (0, _promiseForObject.promiseForObject)(results);
    }
    __name(executeFields, "executeFields");
    function executeField(exeContext, parentType, source, fieldNodes, path2) {
      var _fieldDef$resolve;
      const fieldDef = getFieldDef(exeContext.schema, parentType, fieldNodes[0]);
      if (!fieldDef) {
        return;
      }
      const returnType = fieldDef.type;
      const resolveFn = (_fieldDef$resolve = fieldDef.resolve) !== null && _fieldDef$resolve !== void 0 ? _fieldDef$resolve : exeContext.fieldResolver;
      const info = buildResolveInfo(
        exeContext,
        fieldDef,
        fieldNodes,
        parentType,
        path2
      );
      try {
        const args = (0, _values.getArgumentValues)(
          fieldDef,
          fieldNodes[0],
          exeContext.variableValues
        );
        const contextValue = exeContext.contextValue;
        const result = resolveFn(source, args, contextValue, info);
        let completed;
        if ((0, _isPromise.isPromise)(result)) {
          completed = result.then(
            (resolved) => completeValue(exeContext, returnType, fieldNodes, info, path2, resolved)
          );
        } else {
          completed = completeValue(
            exeContext,
            returnType,
            fieldNodes,
            info,
            path2,
            result
          );
        }
        if ((0, _isPromise.isPromise)(completed)) {
          return completed.then(void 0, (rawError) => {
            const error = (0, _locatedError.locatedError)(
              rawError,
              fieldNodes,
              (0, _Path.pathToArray)(path2)
            );
            return handleFieldError(error, returnType, exeContext);
          });
        }
        return completed;
      } catch (rawError) {
        const error = (0, _locatedError.locatedError)(
          rawError,
          fieldNodes,
          (0, _Path.pathToArray)(path2)
        );
        return handleFieldError(error, returnType, exeContext);
      }
    }
    __name(executeField, "executeField");
    function buildResolveInfo(exeContext, fieldDef, fieldNodes, parentType, path2) {
      return {
        fieldName: fieldDef.name,
        fieldNodes,
        returnType: fieldDef.type,
        parentType,
        path: path2,
        schema: exeContext.schema,
        fragments: exeContext.fragments,
        rootValue: exeContext.rootValue,
        operation: exeContext.operation,
        variableValues: exeContext.variableValues
      };
    }
    __name(buildResolveInfo, "buildResolveInfo");
    function handleFieldError(error, returnType, exeContext) {
      if ((0, _definition.isNonNullType)(returnType)) {
        throw error;
      }
      exeContext.errors.push(error);
      return null;
    }
    __name(handleFieldError, "handleFieldError");
    function completeValue(exeContext, returnType, fieldNodes, info, path2, result) {
      if (result instanceof Error) {
        throw result;
      }
      if ((0, _definition.isNonNullType)(returnType)) {
        const completed = completeValue(
          exeContext,
          returnType.ofType,
          fieldNodes,
          info,
          path2,
          result
        );
        if (completed === null) {
          throw new Error(
            `Cannot return null for non-nullable field ${info.parentType.name}.${info.fieldName}.`
          );
        }
        return completed;
      }
      if (result == null) {
        return null;
      }
      if ((0, _definition.isListType)(returnType)) {
        return completeListValue(
          exeContext,
          returnType,
          fieldNodes,
          info,
          path2,
          result
        );
      }
      if ((0, _definition.isLeafType)(returnType)) {
        return completeLeafValue(returnType, result);
      }
      if ((0, _definition.isAbstractType)(returnType)) {
        return completeAbstractValue(
          exeContext,
          returnType,
          fieldNodes,
          info,
          path2,
          result
        );
      }
      if ((0, _definition.isObjectType)(returnType)) {
        return completeObjectValue(
          exeContext,
          returnType,
          fieldNodes,
          info,
          path2,
          result
        );
      }
      (0, _invariant.invariant)(
        false,
        "Cannot complete value of unexpected output type: " + (0, _inspect.inspect)(returnType)
      );
    }
    __name(completeValue, "completeValue");
    function completeListValue(exeContext, returnType, fieldNodes, info, path2, result) {
      if (!(0, _isIterableObject.isIterableObject)(result)) {
        throw new _GraphQLError.GraphQLError(
          `Expected Iterable, but did not find one for field "${info.parentType.name}.${info.fieldName}".`
        );
      }
      const itemType = returnType.ofType;
      let containsPromise = false;
      const completedResults = Array.from(result, (item, index) => {
        const itemPath = (0, _Path.addPath)(path2, index, void 0);
        try {
          let completedItem;
          if ((0, _isPromise.isPromise)(item)) {
            completedItem = item.then(
              (resolved) => completeValue(
                exeContext,
                itemType,
                fieldNodes,
                info,
                itemPath,
                resolved
              )
            );
          } else {
            completedItem = completeValue(
              exeContext,
              itemType,
              fieldNodes,
              info,
              itemPath,
              item
            );
          }
          if ((0, _isPromise.isPromise)(completedItem)) {
            containsPromise = true;
            return completedItem.then(void 0, (rawError) => {
              const error = (0, _locatedError.locatedError)(
                rawError,
                fieldNodes,
                (0, _Path.pathToArray)(itemPath)
              );
              return handleFieldError(error, itemType, exeContext);
            });
          }
          return completedItem;
        } catch (rawError) {
          const error = (0, _locatedError.locatedError)(
            rawError,
            fieldNodes,
            (0, _Path.pathToArray)(itemPath)
          );
          return handleFieldError(error, itemType, exeContext);
        }
      });
      return containsPromise ? Promise.all(completedResults) : completedResults;
    }
    __name(completeListValue, "completeListValue");
    function completeLeafValue(returnType, result) {
      const serializedResult = returnType.serialize(result);
      if (serializedResult == null) {
        throw new Error(
          `Expected \`${(0, _inspect.inspect)(returnType)}.serialize(${(0, _inspect.inspect)(result)})\` to return non-nullable value, returned: ${(0, _inspect.inspect)(
            serializedResult
          )}`
        );
      }
      return serializedResult;
    }
    __name(completeLeafValue, "completeLeafValue");
    function completeAbstractValue(exeContext, returnType, fieldNodes, info, path2, result) {
      var _returnType$resolveTy;
      const resolveTypeFn = (_returnType$resolveTy = returnType.resolveType) !== null && _returnType$resolveTy !== void 0 ? _returnType$resolveTy : exeContext.typeResolver;
      const contextValue = exeContext.contextValue;
      const runtimeType = resolveTypeFn(result, contextValue, info, returnType);
      if ((0, _isPromise.isPromise)(runtimeType)) {
        return runtimeType.then(
          (resolvedRuntimeType) => completeObjectValue(
            exeContext,
            ensureValidRuntimeType(
              resolvedRuntimeType,
              exeContext,
              returnType,
              fieldNodes,
              info,
              result
            ),
            fieldNodes,
            info,
            path2,
            result
          )
        );
      }
      return completeObjectValue(
        exeContext,
        ensureValidRuntimeType(
          runtimeType,
          exeContext,
          returnType,
          fieldNodes,
          info,
          result
        ),
        fieldNodes,
        info,
        path2,
        result
      );
    }
    __name(completeAbstractValue, "completeAbstractValue");
    function ensureValidRuntimeType(runtimeTypeName, exeContext, returnType, fieldNodes, info, result) {
      if (runtimeTypeName == null) {
        throw new _GraphQLError.GraphQLError(
          `Abstract type "${returnType.name}" must resolve to an Object type at runtime for field "${info.parentType.name}.${info.fieldName}". Either the "${returnType.name}" type should provide a "resolveType" function or each possible type should provide an "isTypeOf" function.`,
          fieldNodes
        );
      }
      if ((0, _definition.isObjectType)(runtimeTypeName)) {
        throw new _GraphQLError.GraphQLError(
          "Support for returning GraphQLObjectType from resolveType was removed in graphql-js@16.0.0 please return type name instead."
        );
      }
      if (typeof runtimeTypeName !== "string") {
        throw new _GraphQLError.GraphQLError(
          `Abstract type "${returnType.name}" must resolve to an Object type at runtime for field "${info.parentType.name}.${info.fieldName}" with value ${(0, _inspect.inspect)(result)}, received "${(0, _inspect.inspect)(runtimeTypeName)}".`
        );
      }
      const runtimeType = exeContext.schema.getType(runtimeTypeName);
      if (runtimeType == null) {
        throw new _GraphQLError.GraphQLError(
          `Abstract type "${returnType.name}" was resolved to a type "${runtimeTypeName}" that does not exist inside the schema.`,
          {
            nodes: fieldNodes
          }
        );
      }
      if (!(0, _definition.isObjectType)(runtimeType)) {
        throw new _GraphQLError.GraphQLError(
          `Abstract type "${returnType.name}" was resolved to a non-object type "${runtimeTypeName}".`,
          {
            nodes: fieldNodes
          }
        );
      }
      if (!exeContext.schema.isSubType(returnType, runtimeType)) {
        throw new _GraphQLError.GraphQLError(
          `Runtime Object type "${runtimeType.name}" is not a possible type for "${returnType.name}".`,
          {
            nodes: fieldNodes
          }
        );
      }
      return runtimeType;
    }
    __name(ensureValidRuntimeType, "ensureValidRuntimeType");
    function completeObjectValue(exeContext, returnType, fieldNodes, info, path2, result) {
      const subFieldNodes = collectSubfields(exeContext, returnType, fieldNodes);
      if (returnType.isTypeOf) {
        const isTypeOf = returnType.isTypeOf(result, exeContext.contextValue, info);
        if ((0, _isPromise.isPromise)(isTypeOf)) {
          return isTypeOf.then((resolvedIsTypeOf) => {
            if (!resolvedIsTypeOf) {
              throw invalidReturnTypeError(returnType, result, fieldNodes);
            }
            return executeFields(
              exeContext,
              returnType,
              result,
              path2,
              subFieldNodes
            );
          });
        }
        if (!isTypeOf) {
          throw invalidReturnTypeError(returnType, result, fieldNodes);
        }
      }
      return executeFields(exeContext, returnType, result, path2, subFieldNodes);
    }
    __name(completeObjectValue, "completeObjectValue");
    function invalidReturnTypeError(returnType, result, fieldNodes) {
      return new _GraphQLError.GraphQLError(
        `Expected value of type "${returnType.name}" but got: ${(0, _inspect.inspect)(result)}.`,
        {
          nodes: fieldNodes
        }
      );
    }
    __name(invalidReturnTypeError, "invalidReturnTypeError");
    var defaultTypeResolver = /* @__PURE__ */ __name(function(value, contextValue, info, abstractType) {
      if ((0, _isObjectLike.isObjectLike)(value) && typeof value.__typename === "string") {
        return value.__typename;
      }
      const possibleTypes = info.schema.getPossibleTypes(abstractType);
      const promisedIsTypeOfResults = [];
      for (let i = 0; i < possibleTypes.length; i++) {
        const type = possibleTypes[i];
        if (type.isTypeOf) {
          const isTypeOfResult = type.isTypeOf(value, contextValue, info);
          if ((0, _isPromise.isPromise)(isTypeOfResult)) {
            promisedIsTypeOfResults[i] = isTypeOfResult;
          } else if (isTypeOfResult) {
            return type.name;
          }
        }
      }
      if (promisedIsTypeOfResults.length) {
        return Promise.all(promisedIsTypeOfResults).then((isTypeOfResults) => {
          for (let i = 0; i < isTypeOfResults.length; i++) {
            if (isTypeOfResults[i]) {
              return possibleTypes[i].name;
            }
          }
        });
      }
    }, "defaultTypeResolver");
    exports.defaultTypeResolver = defaultTypeResolver;
    var defaultFieldResolver = /* @__PURE__ */ __name(function(source, args, contextValue, info) {
      if ((0, _isObjectLike.isObjectLike)(source) || typeof source === "function") {
        const property = source[info.fieldName];
        if (typeof property === "function") {
          return source[info.fieldName](args, contextValue, info);
        }
        return property;
      }
    }, "defaultFieldResolver");
    exports.defaultFieldResolver = defaultFieldResolver;
    function getFieldDef(schema, parentType, fieldNode) {
      const fieldName = fieldNode.name.value;
      if (fieldName === _introspection.SchemaMetaFieldDef.name && schema.getQueryType() === parentType) {
        return _introspection.SchemaMetaFieldDef;
      } else if (fieldName === _introspection.TypeMetaFieldDef.name && schema.getQueryType() === parentType) {
        return _introspection.TypeMetaFieldDef;
      } else if (fieldName === _introspection.TypeNameMetaFieldDef.name) {
        return _introspection.TypeNameMetaFieldDef;
      }
      return parentType.getFields()[fieldName];
    }
    __name(getFieldDef, "getFieldDef");
  }
});

// ../../node_modules/graphql/graphql.js
var require_graphql = __commonJS({
  "../../node_modules/graphql/graphql.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.graphql = graphql;
    exports.graphqlSync = graphqlSync;
    var _devAssert = require_devAssert();
    var _isPromise = require_isPromise();
    var _parser = require_parser();
    var _validate = require_validate();
    var _validate2 = require_validate2();
    var _execute = require_execute();
    function graphql(args) {
      return new Promise((resolve) => resolve(graphqlImpl(args)));
    }
    __name(graphql, "graphql");
    function graphqlSync(args) {
      const result = graphqlImpl(args);
      if ((0, _isPromise.isPromise)(result)) {
        throw new Error("GraphQL execution failed to complete synchronously.");
      }
      return result;
    }
    __name(graphqlSync, "graphqlSync");
    function graphqlImpl(args) {
      arguments.length < 2 || (0, _devAssert.devAssert)(
        false,
        "graphql@16 dropped long-deprecated support for positional arguments, please pass an object instead."
      );
      const {
        schema,
        source,
        rootValue,
        contextValue,
        variableValues,
        operationName,
        fieldResolver,
        typeResolver
      } = args;
      const schemaValidationErrors = (0, _validate.validateSchema)(schema);
      if (schemaValidationErrors.length > 0) {
        return {
          errors: schemaValidationErrors
        };
      }
      let document;
      try {
        document = (0, _parser.parse)(source);
      } catch (syntaxError) {
        return {
          errors: [syntaxError]
        };
      }
      const validationErrors = (0, _validate2.validate)(schema, document);
      if (validationErrors.length > 0) {
        return {
          errors: validationErrors
        };
      }
      return (0, _execute.execute)({
        schema,
        document,
        rootValue,
        contextValue,
        variableValues,
        operationName,
        fieldResolver,
        typeResolver
      });
    }
    __name(graphqlImpl, "graphqlImpl");
  }
});

// ../../node_modules/graphql/type/index.js
var require_type = __commonJS({
  "../../node_modules/graphql/type/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "DEFAULT_DEPRECATION_REASON", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.DEFAULT_DEPRECATION_REASON;
      }, "get")
    });
    Object.defineProperty(exports, "GRAPHQL_MAX_INT", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GRAPHQL_MAX_INT;
      }, "get")
    });
    Object.defineProperty(exports, "GRAPHQL_MIN_INT", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GRAPHQL_MIN_INT;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLBoolean", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GraphQLBoolean;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLDeprecatedDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.GraphQLDeprecatedDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.GraphQLDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLEnumType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLEnumType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLFloat", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GraphQLFloat;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLID", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GraphQLID;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLIncludeDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.GraphQLIncludeDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLInputObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLInputObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLInt", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GraphQLInt;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLInterfaceType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLInterfaceType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLList", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLList;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLNonNull", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLNonNull;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLOneOfDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.GraphQLOneOfDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _schema.GraphQLSchema;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLSkipDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.GraphQLSkipDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLSpecifiedByDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.GraphQLSpecifiedByDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLString", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.GraphQLString;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLUnionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.GraphQLUnionType;
      }, "get")
    });
    Object.defineProperty(exports, "SchemaMetaFieldDef", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.SchemaMetaFieldDef;
      }, "get")
    });
    Object.defineProperty(exports, "TypeKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.TypeKind;
      }, "get")
    });
    Object.defineProperty(exports, "TypeMetaFieldDef", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.TypeMetaFieldDef;
      }, "get")
    });
    Object.defineProperty(exports, "TypeNameMetaFieldDef", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.TypeNameMetaFieldDef;
      }, "get")
    });
    Object.defineProperty(exports, "__Directive", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__Directive;
      }, "get")
    });
    Object.defineProperty(exports, "__DirectiveLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__DirectiveLocation;
      }, "get")
    });
    Object.defineProperty(exports, "__EnumValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__EnumValue;
      }, "get")
    });
    Object.defineProperty(exports, "__Field", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__Field;
      }, "get")
    });
    Object.defineProperty(exports, "__InputValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__InputValue;
      }, "get")
    });
    Object.defineProperty(exports, "__Schema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__Schema;
      }, "get")
    });
    Object.defineProperty(exports, "__Type", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__Type;
      }, "get")
    });
    Object.defineProperty(exports, "__TypeKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.__TypeKind;
      }, "get")
    });
    Object.defineProperty(exports, "assertAbstractType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertAbstractType;
      }, "get")
    });
    Object.defineProperty(exports, "assertCompositeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertCompositeType;
      }, "get")
    });
    Object.defineProperty(exports, "assertDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.assertDirective;
      }, "get")
    });
    Object.defineProperty(exports, "assertEnumType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertEnumType;
      }, "get")
    });
    Object.defineProperty(exports, "assertEnumValueName", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _assertName.assertEnumValueName;
      }, "get")
    });
    Object.defineProperty(exports, "assertInputObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertInputObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "assertInputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertInputType;
      }, "get")
    });
    Object.defineProperty(exports, "assertInterfaceType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertInterfaceType;
      }, "get")
    });
    Object.defineProperty(exports, "assertLeafType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertLeafType;
      }, "get")
    });
    Object.defineProperty(exports, "assertListType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertListType;
      }, "get")
    });
    Object.defineProperty(exports, "assertName", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _assertName.assertName;
      }, "get")
    });
    Object.defineProperty(exports, "assertNamedType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertNamedType;
      }, "get")
    });
    Object.defineProperty(exports, "assertNonNullType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertNonNullType;
      }, "get")
    });
    Object.defineProperty(exports, "assertNullableType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertNullableType;
      }, "get")
    });
    Object.defineProperty(exports, "assertObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "assertOutputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertOutputType;
      }, "get")
    });
    Object.defineProperty(exports, "assertScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "assertSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _schema.assertSchema;
      }, "get")
    });
    Object.defineProperty(exports, "assertType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertType;
      }, "get")
    });
    Object.defineProperty(exports, "assertUnionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertUnionType;
      }, "get")
    });
    Object.defineProperty(exports, "assertValidSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _validate.assertValidSchema;
      }, "get")
    });
    Object.defineProperty(exports, "assertWrappingType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.assertWrappingType;
      }, "get")
    });
    Object.defineProperty(exports, "getNamedType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.getNamedType;
      }, "get")
    });
    Object.defineProperty(exports, "getNullableType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.getNullableType;
      }, "get")
    });
    Object.defineProperty(exports, "introspectionTypes", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.introspectionTypes;
      }, "get")
    });
    Object.defineProperty(exports, "isAbstractType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isAbstractType;
      }, "get")
    });
    Object.defineProperty(exports, "isCompositeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isCompositeType;
      }, "get")
    });
    Object.defineProperty(exports, "isDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.isDirective;
      }, "get")
    });
    Object.defineProperty(exports, "isEnumType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isEnumType;
      }, "get")
    });
    Object.defineProperty(exports, "isInputObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isInputObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "isInputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isInputType;
      }, "get")
    });
    Object.defineProperty(exports, "isInterfaceType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isInterfaceType;
      }, "get")
    });
    Object.defineProperty(exports, "isIntrospectionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspection.isIntrospectionType;
      }, "get")
    });
    Object.defineProperty(exports, "isLeafType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isLeafType;
      }, "get")
    });
    Object.defineProperty(exports, "isListType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isListType;
      }, "get")
    });
    Object.defineProperty(exports, "isNamedType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isNamedType;
      }, "get")
    });
    Object.defineProperty(exports, "isNonNullType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isNonNullType;
      }, "get")
    });
    Object.defineProperty(exports, "isNullableType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isNullableType;
      }, "get")
    });
    Object.defineProperty(exports, "isObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "isOutputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isOutputType;
      }, "get")
    });
    Object.defineProperty(exports, "isRequiredArgument", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isRequiredArgument;
      }, "get")
    });
    Object.defineProperty(exports, "isRequiredInputField", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isRequiredInputField;
      }, "get")
    });
    Object.defineProperty(exports, "isScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "isSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _schema.isSchema;
      }, "get")
    });
    Object.defineProperty(exports, "isSpecifiedDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.isSpecifiedDirective;
      }, "get")
    });
    Object.defineProperty(exports, "isSpecifiedScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.isSpecifiedScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "isType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isType;
      }, "get")
    });
    Object.defineProperty(exports, "isUnionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isUnionType;
      }, "get")
    });
    Object.defineProperty(exports, "isWrappingType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.isWrappingType;
      }, "get")
    });
    Object.defineProperty(exports, "resolveObjMapThunk", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.resolveObjMapThunk;
      }, "get")
    });
    Object.defineProperty(exports, "resolveReadonlyArrayThunk", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _definition.resolveReadonlyArrayThunk;
      }, "get")
    });
    Object.defineProperty(exports, "specifiedDirectives", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directives.specifiedDirectives;
      }, "get")
    });
    Object.defineProperty(exports, "specifiedScalarTypes", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _scalars.specifiedScalarTypes;
      }, "get")
    });
    Object.defineProperty(exports, "validateSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _validate.validateSchema;
      }, "get")
    });
    var _schema = require_schema();
    var _definition = require_definition();
    var _directives = require_directives();
    var _scalars = require_scalars();
    var _introspection = require_introspection();
    var _validate = require_validate();
    var _assertName = require_assertName();
  }
});

// ../../node_modules/graphql/language/index.js
var require_language = __commonJS({
  "../../node_modules/graphql/language/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "BREAK", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _visitor.BREAK;
      }, "get")
    });
    Object.defineProperty(exports, "DirectiveLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _directiveLocation.DirectiveLocation;
      }, "get")
    });
    Object.defineProperty(exports, "Kind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _kinds.Kind;
      }, "get")
    });
    Object.defineProperty(exports, "Lexer", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _lexer.Lexer;
      }, "get")
    });
    Object.defineProperty(exports, "Location", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ast.Location;
      }, "get")
    });
    Object.defineProperty(exports, "OperationTypeNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ast.OperationTypeNode;
      }, "get")
    });
    Object.defineProperty(exports, "Source", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _source.Source;
      }, "get")
    });
    Object.defineProperty(exports, "Token", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ast.Token;
      }, "get")
    });
    Object.defineProperty(exports, "TokenKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _tokenKind.TokenKind;
      }, "get")
    });
    Object.defineProperty(exports, "getEnterLeaveForKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _visitor.getEnterLeaveForKind;
      }, "get")
    });
    Object.defineProperty(exports, "getLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _location.getLocation;
      }, "get")
    });
    Object.defineProperty(exports, "getVisitFn", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _visitor.getVisitFn;
      }, "get")
    });
    Object.defineProperty(exports, "isConstValueNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isConstValueNode;
      }, "get")
    });
    Object.defineProperty(exports, "isDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isExecutableDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isExecutableDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isSelectionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isSelectionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isTypeDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeExtensionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isTypeExtensionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isTypeNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeSystemDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isTypeSystemDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeSystemExtensionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isTypeSystemExtensionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isValueNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _predicates.isValueNode;
      }, "get")
    });
    Object.defineProperty(exports, "parse", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _parser.parse;
      }, "get")
    });
    Object.defineProperty(exports, "parseConstValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _parser.parseConstValue;
      }, "get")
    });
    Object.defineProperty(exports, "parseType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _parser.parseType;
      }, "get")
    });
    Object.defineProperty(exports, "parseValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _parser.parseValue;
      }, "get")
    });
    Object.defineProperty(exports, "print", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _printer.print;
      }, "get")
    });
    Object.defineProperty(exports, "printLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _printLocation.printLocation;
      }, "get")
    });
    Object.defineProperty(exports, "printSourceLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _printLocation.printSourceLocation;
      }, "get")
    });
    Object.defineProperty(exports, "visit", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _visitor.visit;
      }, "get")
    });
    Object.defineProperty(exports, "visitInParallel", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _visitor.visitInParallel;
      }, "get")
    });
    var _source = require_source();
    var _location = require_location();
    var _printLocation = require_printLocation();
    var _kinds = require_kinds();
    var _tokenKind = require_tokenKind();
    var _lexer = require_lexer();
    var _parser = require_parser();
    var _printer = require_printer();
    var _visitor = require_visitor();
    var _ast = require_ast();
    var _predicates = require_predicates();
    var _directiveLocation = require_directiveLocation();
  }
});

// ../../node_modules/graphql/jsutils/isAsyncIterable.js
var require_isAsyncIterable = __commonJS({
  "../../node_modules/graphql/jsutils/isAsyncIterable.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.isAsyncIterable = isAsyncIterable;
    function isAsyncIterable(maybeAsyncIterable) {
      return typeof (maybeAsyncIterable === null || maybeAsyncIterable === void 0 ? void 0 : maybeAsyncIterable[Symbol.asyncIterator]) === "function";
    }
    __name(isAsyncIterable, "isAsyncIterable");
  }
});

// ../../node_modules/graphql/execution/mapAsyncIterator.js
var require_mapAsyncIterator = __commonJS({
  "../../node_modules/graphql/execution/mapAsyncIterator.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.mapAsyncIterator = mapAsyncIterator;
    function mapAsyncIterator(iterable, callback) {
      const iterator = iterable[Symbol.asyncIterator]();
      async function mapResult(result) {
        if (result.done) {
          return result;
        }
        try {
          return {
            value: await callback(result.value),
            done: false
          };
        } catch (error) {
          if (typeof iterator.return === "function") {
            try {
              await iterator.return();
            } catch (_e) {
            }
          }
          throw error;
        }
      }
      __name(mapResult, "mapResult");
      return {
        async next() {
          return mapResult(await iterator.next());
        },
        async return() {
          return typeof iterator.return === "function" ? mapResult(await iterator.return()) : {
            value: void 0,
            done: true
          };
        },
        async throw(error) {
          if (typeof iterator.throw === "function") {
            return mapResult(await iterator.throw(error));
          }
          throw error;
        },
        [Symbol.asyncIterator]() {
          return this;
        }
      };
    }
    __name(mapAsyncIterator, "mapAsyncIterator");
  }
});

// ../../node_modules/graphql/execution/subscribe.js
var require_subscribe = __commonJS({
  "../../node_modules/graphql/execution/subscribe.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.createSourceEventStream = createSourceEventStream;
    exports.subscribe = subscribe;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _isAsyncIterable = require_isAsyncIterable();
    var _Path = require_Path();
    var _GraphQLError = require_GraphQLError();
    var _locatedError = require_locatedError();
    var _collectFields = require_collectFields();
    var _execute = require_execute();
    var _mapAsyncIterator = require_mapAsyncIterator();
    var _values = require_values();
    async function subscribe(args) {
      arguments.length < 2 || (0, _devAssert.devAssert)(
        false,
        "graphql@16 dropped long-deprecated support for positional arguments, please pass an object instead."
      );
      const resultOrStream = await createSourceEventStream(args);
      if (!(0, _isAsyncIterable.isAsyncIterable)(resultOrStream)) {
        return resultOrStream;
      }
      const mapSourceToResponse = /* @__PURE__ */ __name((payload) => (0, _execute.execute)({ ...args, rootValue: payload }), "mapSourceToResponse");
      return (0, _mapAsyncIterator.mapAsyncIterator)(
        resultOrStream,
        mapSourceToResponse
      );
    }
    __name(subscribe, "subscribe");
    function toNormalizedArgs(args) {
      const firstArg = args[0];
      if (firstArg && "document" in firstArg) {
        return firstArg;
      }
      return {
        schema: firstArg,
        // FIXME: when underlying TS bug fixed, see https://github.com/microsoft/TypeScript/issues/31613
        document: args[1],
        rootValue: args[2],
        contextValue: args[3],
        variableValues: args[4],
        operationName: args[5],
        subscribeFieldResolver: args[6]
      };
    }
    __name(toNormalizedArgs, "toNormalizedArgs");
    async function createSourceEventStream(...rawArgs) {
      const args = toNormalizedArgs(rawArgs);
      const { schema, document, variableValues } = args;
      (0, _execute.assertValidExecutionArguments)(schema, document, variableValues);
      const exeContext = (0, _execute.buildExecutionContext)(args);
      if (!("schema" in exeContext)) {
        return {
          errors: exeContext
        };
      }
      try {
        const eventStream = await executeSubscription(exeContext);
        if (!(0, _isAsyncIterable.isAsyncIterable)(eventStream)) {
          throw new Error(
            `Subscription field must return Async Iterable. Received: ${(0, _inspect.inspect)(eventStream)}.`
          );
        }
        return eventStream;
      } catch (error) {
        if (error instanceof _GraphQLError.GraphQLError) {
          return {
            errors: [error]
          };
        }
        throw error;
      }
    }
    __name(createSourceEventStream, "createSourceEventStream");
    async function executeSubscription(exeContext) {
      const { schema, fragments, operation, variableValues, rootValue } = exeContext;
      const rootType = schema.getSubscriptionType();
      if (rootType == null) {
        throw new _GraphQLError.GraphQLError(
          "Schema is not configured to execute subscription operation.",
          {
            nodes: operation
          }
        );
      }
      const rootFields = (0, _collectFields.collectFields)(
        schema,
        fragments,
        variableValues,
        rootType,
        operation.selectionSet
      );
      const [responseName, fieldNodes] = [...rootFields.entries()][0];
      const fieldDef = (0, _execute.getFieldDef)(schema, rootType, fieldNodes[0]);
      if (!fieldDef) {
        const fieldName = fieldNodes[0].name.value;
        throw new _GraphQLError.GraphQLError(
          `The subscription field "${fieldName}" is not defined.`,
          {
            nodes: fieldNodes
          }
        );
      }
      const path2 = (0, _Path.addPath)(void 0, responseName, rootType.name);
      const info = (0, _execute.buildResolveInfo)(
        exeContext,
        fieldDef,
        fieldNodes,
        rootType,
        path2
      );
      try {
        var _fieldDef$subscribe;
        const args = (0, _values.getArgumentValues)(
          fieldDef,
          fieldNodes[0],
          variableValues
        );
        const contextValue = exeContext.contextValue;
        const resolveFn = (_fieldDef$subscribe = fieldDef.subscribe) !== null && _fieldDef$subscribe !== void 0 ? _fieldDef$subscribe : exeContext.subscribeFieldResolver;
        const eventStream = await resolveFn(rootValue, args, contextValue, info);
        if (eventStream instanceof Error) {
          throw eventStream;
        }
        return eventStream;
      } catch (error) {
        throw (0, _locatedError.locatedError)(
          error,
          fieldNodes,
          (0, _Path.pathToArray)(path2)
        );
      }
    }
    __name(executeSubscription, "executeSubscription");
  }
});

// ../../node_modules/graphql/execution/index.js
var require_execution = __commonJS({
  "../../node_modules/graphql/execution/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "createSourceEventStream", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _subscribe.createSourceEventStream;
      }, "get")
    });
    Object.defineProperty(exports, "defaultFieldResolver", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _execute.defaultFieldResolver;
      }, "get")
    });
    Object.defineProperty(exports, "defaultTypeResolver", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _execute.defaultTypeResolver;
      }, "get")
    });
    Object.defineProperty(exports, "execute", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _execute.execute;
      }, "get")
    });
    Object.defineProperty(exports, "executeSync", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _execute.executeSync;
      }, "get")
    });
    Object.defineProperty(exports, "getArgumentValues", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _values.getArgumentValues;
      }, "get")
    });
    Object.defineProperty(exports, "getDirectiveValues", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _values.getDirectiveValues;
      }, "get")
    });
    Object.defineProperty(exports, "getVariableValues", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _values.getVariableValues;
      }, "get")
    });
    Object.defineProperty(exports, "responsePathAsArray", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _Path.pathToArray;
      }, "get")
    });
    Object.defineProperty(exports, "subscribe", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _subscribe.subscribe;
      }, "get")
    });
    var _Path = require_Path();
    var _execute = require_execute();
    var _subscribe = require_subscribe();
    var _values = require_values();
  }
});

// ../../node_modules/graphql/validation/rules/custom/NoDeprecatedCustomRule.js
var require_NoDeprecatedCustomRule = __commonJS({
  "../../node_modules/graphql/validation/rules/custom/NoDeprecatedCustomRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.NoDeprecatedCustomRule = NoDeprecatedCustomRule;
    var _invariant = require_invariant();
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    function NoDeprecatedCustomRule(context) {
      return {
        Field(node) {
          const fieldDef = context.getFieldDef();
          const deprecationReason = fieldDef === null || fieldDef === void 0 ? void 0 : fieldDef.deprecationReason;
          if (fieldDef && deprecationReason != null) {
            const parentType = context.getParentType();
            parentType != null || (0, _invariant.invariant)(false);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `The field ${parentType.name}.${fieldDef.name} is deprecated. ${deprecationReason}`,
                {
                  nodes: node
                }
              )
            );
          }
        },
        Argument(node) {
          const argDef = context.getArgument();
          const deprecationReason = argDef === null || argDef === void 0 ? void 0 : argDef.deprecationReason;
          if (argDef && deprecationReason != null) {
            const directiveDef = context.getDirective();
            if (directiveDef != null) {
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Directive "@${directiveDef.name}" argument "${argDef.name}" is deprecated. ${deprecationReason}`,
                  {
                    nodes: node
                  }
                )
              );
            } else {
              const parentType = context.getParentType();
              const fieldDef = context.getFieldDef();
              parentType != null && fieldDef != null || (0, _invariant.invariant)(false);
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `Field "${parentType.name}.${fieldDef.name}" argument "${argDef.name}" is deprecated. ${deprecationReason}`,
                  {
                    nodes: node
                  }
                )
              );
            }
          }
        },
        ObjectField(node) {
          const inputObjectDef = (0, _definition.getNamedType)(
            context.getParentInputType()
          );
          if ((0, _definition.isInputObjectType)(inputObjectDef)) {
            const inputFieldDef = inputObjectDef.getFields()[node.name.value];
            const deprecationReason = inputFieldDef === null || inputFieldDef === void 0 ? void 0 : inputFieldDef.deprecationReason;
            if (deprecationReason != null) {
              context.reportError(
                new _GraphQLError.GraphQLError(
                  `The input field ${inputObjectDef.name}.${inputFieldDef.name} is deprecated. ${deprecationReason}`,
                  {
                    nodes: node
                  }
                )
              );
            }
          }
        },
        EnumValue(node) {
          const enumValueDef = context.getEnumValue();
          const deprecationReason = enumValueDef === null || enumValueDef === void 0 ? void 0 : enumValueDef.deprecationReason;
          if (enumValueDef && deprecationReason != null) {
            const enumTypeDef = (0, _definition.getNamedType)(
              context.getInputType()
            );
            enumTypeDef != null || (0, _invariant.invariant)(false);
            context.reportError(
              new _GraphQLError.GraphQLError(
                `The enum value "${enumTypeDef.name}.${enumValueDef.name}" is deprecated. ${deprecationReason}`,
                {
                  nodes: node
                }
              )
            );
          }
        }
      };
    }
    __name(NoDeprecatedCustomRule, "NoDeprecatedCustomRule");
  }
});

// ../../node_modules/graphql/validation/rules/custom/NoSchemaIntrospectionCustomRule.js
var require_NoSchemaIntrospectionCustomRule = __commonJS({
  "../../node_modules/graphql/validation/rules/custom/NoSchemaIntrospectionCustomRule.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.NoSchemaIntrospectionCustomRule = NoSchemaIntrospectionCustomRule;
    var _GraphQLError = require_GraphQLError();
    var _definition = require_definition();
    var _introspection = require_introspection();
    function NoSchemaIntrospectionCustomRule(context) {
      return {
        Field(node) {
          const type = (0, _definition.getNamedType)(context.getType());
          if (type && (0, _introspection.isIntrospectionType)(type)) {
            context.reportError(
              new _GraphQLError.GraphQLError(
                `GraphQL introspection has been disabled, but the requested query contained the field "${node.name.value}".`,
                {
                  nodes: node
                }
              )
            );
          }
        }
      };
    }
    __name(NoSchemaIntrospectionCustomRule, "NoSchemaIntrospectionCustomRule");
  }
});

// ../../node_modules/graphql/validation/index.js
var require_validation = __commonJS({
  "../../node_modules/graphql/validation/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "ExecutableDefinitionsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ExecutableDefinitionsRule.ExecutableDefinitionsRule;
      }, "get")
    });
    Object.defineProperty(exports, "FieldsOnCorrectTypeRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _FieldsOnCorrectTypeRule.FieldsOnCorrectTypeRule;
      }, "get")
    });
    Object.defineProperty(exports, "FragmentsOnCompositeTypesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _FragmentsOnCompositeTypesRule.FragmentsOnCompositeTypesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownArgumentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _KnownArgumentNamesRule.KnownArgumentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownDirectivesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _KnownDirectivesRule.KnownDirectivesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownFragmentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _KnownFragmentNamesRule.KnownFragmentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownTypeNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _KnownTypeNamesRule.KnownTypeNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "LoneAnonymousOperationRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _LoneAnonymousOperationRule.LoneAnonymousOperationRule;
      }, "get")
    });
    Object.defineProperty(exports, "LoneSchemaDefinitionRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _LoneSchemaDefinitionRule.LoneSchemaDefinitionRule;
      }, "get")
    });
    Object.defineProperty(exports, "MaxIntrospectionDepthRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _MaxIntrospectionDepthRule.MaxIntrospectionDepthRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoDeprecatedCustomRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _NoDeprecatedCustomRule.NoDeprecatedCustomRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoFragmentCyclesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _NoFragmentCyclesRule.NoFragmentCyclesRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoSchemaIntrospectionCustomRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _NoSchemaIntrospectionCustomRule.NoSchemaIntrospectionCustomRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoUndefinedVariablesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _NoUndefinedVariablesRule.NoUndefinedVariablesRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoUnusedFragmentsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _NoUnusedFragmentsRule.NoUnusedFragmentsRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoUnusedVariablesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _NoUnusedVariablesRule.NoUnusedVariablesRule;
      }, "get")
    });
    Object.defineProperty(exports, "OverlappingFieldsCanBeMergedRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _OverlappingFieldsCanBeMergedRule.OverlappingFieldsCanBeMergedRule;
      }, "get")
    });
    Object.defineProperty(exports, "PossibleFragmentSpreadsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _PossibleFragmentSpreadsRule.PossibleFragmentSpreadsRule;
      }, "get")
    });
    Object.defineProperty(exports, "PossibleTypeExtensionsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _PossibleTypeExtensionsRule.PossibleTypeExtensionsRule;
      }, "get")
    });
    Object.defineProperty(exports, "ProvidedRequiredArgumentsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ProvidedRequiredArgumentsRule.ProvidedRequiredArgumentsRule;
      }, "get")
    });
    Object.defineProperty(exports, "ScalarLeafsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ScalarLeafsRule.ScalarLeafsRule;
      }, "get")
    });
    Object.defineProperty(exports, "SingleFieldSubscriptionsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _SingleFieldSubscriptionsRule.SingleFieldSubscriptionsRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueArgumentDefinitionNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueArgumentDefinitionNamesRule.UniqueArgumentDefinitionNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueArgumentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueArgumentNamesRule.UniqueArgumentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueDirectiveNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueDirectiveNamesRule.UniqueDirectiveNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueDirectivesPerLocationRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueDirectivesPerLocationRule.UniqueDirectivesPerLocationRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueEnumValueNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueEnumValueNamesRule.UniqueEnumValueNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueFieldDefinitionNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueFieldDefinitionNamesRule.UniqueFieldDefinitionNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueFragmentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueFragmentNamesRule.UniqueFragmentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueInputFieldNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueInputFieldNamesRule.UniqueInputFieldNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueOperationNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueOperationNamesRule.UniqueOperationNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueOperationTypesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueOperationTypesRule.UniqueOperationTypesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueTypeNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueTypeNamesRule.UniqueTypeNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueVariableNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _UniqueVariableNamesRule.UniqueVariableNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "ValidationContext", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ValidationContext.ValidationContext;
      }, "get")
    });
    Object.defineProperty(exports, "ValuesOfCorrectTypeRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _ValuesOfCorrectTypeRule.ValuesOfCorrectTypeRule;
      }, "get")
    });
    Object.defineProperty(exports, "VariablesAreInputTypesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _VariablesAreInputTypesRule.VariablesAreInputTypesRule;
      }, "get")
    });
    Object.defineProperty(exports, "VariablesInAllowedPositionRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _VariablesInAllowedPositionRule.VariablesInAllowedPositionRule;
      }, "get")
    });
    Object.defineProperty(exports, "recommendedRules", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _specifiedRules.recommendedRules;
      }, "get")
    });
    Object.defineProperty(exports, "specifiedRules", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _specifiedRules.specifiedRules;
      }, "get")
    });
    Object.defineProperty(exports, "validate", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _validate.validate;
      }, "get")
    });
    var _validate = require_validate2();
    var _ValidationContext = require_ValidationContext();
    var _specifiedRules = require_specifiedRules();
    var _ExecutableDefinitionsRule = require_ExecutableDefinitionsRule();
    var _FieldsOnCorrectTypeRule = require_FieldsOnCorrectTypeRule();
    var _FragmentsOnCompositeTypesRule = require_FragmentsOnCompositeTypesRule();
    var _KnownArgumentNamesRule = require_KnownArgumentNamesRule();
    var _KnownDirectivesRule = require_KnownDirectivesRule();
    var _KnownFragmentNamesRule = require_KnownFragmentNamesRule();
    var _KnownTypeNamesRule = require_KnownTypeNamesRule();
    var _LoneAnonymousOperationRule = require_LoneAnonymousOperationRule();
    var _NoFragmentCyclesRule = require_NoFragmentCyclesRule();
    var _NoUndefinedVariablesRule = require_NoUndefinedVariablesRule();
    var _NoUnusedFragmentsRule = require_NoUnusedFragmentsRule();
    var _NoUnusedVariablesRule = require_NoUnusedVariablesRule();
    var _OverlappingFieldsCanBeMergedRule = require_OverlappingFieldsCanBeMergedRule();
    var _PossibleFragmentSpreadsRule = require_PossibleFragmentSpreadsRule();
    var _ProvidedRequiredArgumentsRule = require_ProvidedRequiredArgumentsRule();
    var _ScalarLeafsRule = require_ScalarLeafsRule();
    var _SingleFieldSubscriptionsRule = require_SingleFieldSubscriptionsRule();
    var _UniqueArgumentNamesRule = require_UniqueArgumentNamesRule();
    var _UniqueDirectivesPerLocationRule = require_UniqueDirectivesPerLocationRule();
    var _UniqueFragmentNamesRule = require_UniqueFragmentNamesRule();
    var _UniqueInputFieldNamesRule = require_UniqueInputFieldNamesRule();
    var _UniqueOperationNamesRule = require_UniqueOperationNamesRule();
    var _UniqueVariableNamesRule = require_UniqueVariableNamesRule();
    var _ValuesOfCorrectTypeRule = require_ValuesOfCorrectTypeRule();
    var _VariablesAreInputTypesRule = require_VariablesAreInputTypesRule();
    var _VariablesInAllowedPositionRule = require_VariablesInAllowedPositionRule();
    var _MaxIntrospectionDepthRule = require_MaxIntrospectionDepthRule();
    var _LoneSchemaDefinitionRule = require_LoneSchemaDefinitionRule();
    var _UniqueOperationTypesRule = require_UniqueOperationTypesRule();
    var _UniqueTypeNamesRule = require_UniqueTypeNamesRule();
    var _UniqueEnumValueNamesRule = require_UniqueEnumValueNamesRule();
    var _UniqueFieldDefinitionNamesRule = require_UniqueFieldDefinitionNamesRule();
    var _UniqueArgumentDefinitionNamesRule = require_UniqueArgumentDefinitionNamesRule();
    var _UniqueDirectiveNamesRule = require_UniqueDirectiveNamesRule();
    var _PossibleTypeExtensionsRule = require_PossibleTypeExtensionsRule();
    var _NoDeprecatedCustomRule = require_NoDeprecatedCustomRule();
    var _NoSchemaIntrospectionCustomRule = require_NoSchemaIntrospectionCustomRule();
  }
});

// ../../node_modules/graphql/error/index.js
var require_error = __commonJS({
  "../../node_modules/graphql/error/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "GraphQLError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _GraphQLError.GraphQLError;
      }, "get")
    });
    Object.defineProperty(exports, "formatError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _GraphQLError.formatError;
      }, "get")
    });
    Object.defineProperty(exports, "locatedError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _locatedError.locatedError;
      }, "get")
    });
    Object.defineProperty(exports, "printError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _GraphQLError.printError;
      }, "get")
    });
    Object.defineProperty(exports, "syntaxError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _syntaxError.syntaxError;
      }, "get")
    });
    var _GraphQLError = require_GraphQLError();
    var _syntaxError = require_syntaxError();
    var _locatedError = require_locatedError();
  }
});

// ../../node_modules/graphql/utilities/getIntrospectionQuery.js
var require_getIntrospectionQuery = __commonJS({
  "../../node_modules/graphql/utilities/getIntrospectionQuery.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.getIntrospectionQuery = getIntrospectionQuery;
    function getIntrospectionQuery(options) {
      const optionsWithDefault = {
        descriptions: true,
        specifiedByUrl: false,
        directiveIsRepeatable: false,
        schemaDescription: false,
        inputValueDeprecation: false,
        oneOf: false,
        ...options
      };
      const descriptions = optionsWithDefault.descriptions ? "description" : "";
      const specifiedByUrl = optionsWithDefault.specifiedByUrl ? "specifiedByURL" : "";
      const directiveIsRepeatable = optionsWithDefault.directiveIsRepeatable ? "isRepeatable" : "";
      const schemaDescription = optionsWithDefault.schemaDescription ? descriptions : "";
      function inputDeprecation(str) {
        return optionsWithDefault.inputValueDeprecation ? str : "";
      }
      __name(inputDeprecation, "inputDeprecation");
      const oneOf = optionsWithDefault.oneOf ? "isOneOf" : "";
      return `
    query IntrospectionQuery {
      __schema {
        ${schemaDescription}
        queryType { name kind }
        mutationType { name kind }
        subscriptionType { name kind }
        types {
          ...FullType
        }
        directives {
          name
          ${descriptions}
          ${directiveIsRepeatable}
          locations
          args${inputDeprecation("(includeDeprecated: true)")} {
            ...InputValue
          }
        }
      }
    }

    fragment FullType on __Type {
      kind
      name
      ${descriptions}
      ${specifiedByUrl}
      ${oneOf}
      fields(includeDeprecated: true) {
        name
        ${descriptions}
        args${inputDeprecation("(includeDeprecated: true)")} {
          ...InputValue
        }
        type {
          ...TypeRef
        }
        isDeprecated
        deprecationReason
      }
      inputFields${inputDeprecation("(includeDeprecated: true)")} {
        ...InputValue
      }
      interfaces {
        ...TypeRef
      }
      enumValues(includeDeprecated: true) {
        name
        ${descriptions}
        isDeprecated
        deprecationReason
      }
      possibleTypes {
        ...TypeRef
      }
    }

    fragment InputValue on __InputValue {
      name
      ${descriptions}
      type { ...TypeRef }
      defaultValue
      ${inputDeprecation("isDeprecated")}
      ${inputDeprecation("deprecationReason")}
    }

    fragment TypeRef on __Type {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                  ofType {
                    kind
                    name
                    ofType {
                      kind
                      name
                      ofType {
                        kind
                        name
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  `;
    }
    __name(getIntrospectionQuery, "getIntrospectionQuery");
  }
});

// ../../node_modules/graphql/utilities/getOperationAST.js
var require_getOperationAST = __commonJS({
  "../../node_modules/graphql/utilities/getOperationAST.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.getOperationAST = getOperationAST;
    var _kinds = require_kinds();
    function getOperationAST(documentAST, operationName) {
      let operation = null;
      for (const definition of documentAST.definitions) {
        if (definition.kind === _kinds.Kind.OPERATION_DEFINITION) {
          var _definition$name;
          if (operationName == null) {
            if (operation) {
              return null;
            }
            operation = definition;
          } else if (((_definition$name = definition.name) === null || _definition$name === void 0 ? void 0 : _definition$name.value) === operationName) {
            return definition;
          }
        }
      }
      return operation;
    }
    __name(getOperationAST, "getOperationAST");
  }
});

// ../../node_modules/graphql/utilities/getOperationRootType.js
var require_getOperationRootType = __commonJS({
  "../../node_modules/graphql/utilities/getOperationRootType.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.getOperationRootType = getOperationRootType;
    var _GraphQLError = require_GraphQLError();
    function getOperationRootType(schema, operation) {
      if (operation.operation === "query") {
        const queryType = schema.getQueryType();
        if (!queryType) {
          throw new _GraphQLError.GraphQLError(
            "Schema does not define the required query root type.",
            {
              nodes: operation
            }
          );
        }
        return queryType;
      }
      if (operation.operation === "mutation") {
        const mutationType = schema.getMutationType();
        if (!mutationType) {
          throw new _GraphQLError.GraphQLError(
            "Schema is not configured for mutations.",
            {
              nodes: operation
            }
          );
        }
        return mutationType;
      }
      if (operation.operation === "subscription") {
        const subscriptionType = schema.getSubscriptionType();
        if (!subscriptionType) {
          throw new _GraphQLError.GraphQLError(
            "Schema is not configured for subscriptions.",
            {
              nodes: operation
            }
          );
        }
        return subscriptionType;
      }
      throw new _GraphQLError.GraphQLError(
        "Can only have query, mutation and subscription operations.",
        {
          nodes: operation
        }
      );
    }
    __name(getOperationRootType, "getOperationRootType");
  }
});

// ../../node_modules/graphql/utilities/introspectionFromSchema.js
var require_introspectionFromSchema = __commonJS({
  "../../node_modules/graphql/utilities/introspectionFromSchema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.introspectionFromSchema = introspectionFromSchema;
    var _invariant = require_invariant();
    var _parser = require_parser();
    var _execute = require_execute();
    var _getIntrospectionQuery = require_getIntrospectionQuery();
    function introspectionFromSchema(schema, options) {
      const optionsWithDefaults = {
        specifiedByUrl: true,
        directiveIsRepeatable: true,
        schemaDescription: true,
        inputValueDeprecation: true,
        oneOf: true,
        ...options
      };
      const document = (0, _parser.parse)(
        (0, _getIntrospectionQuery.getIntrospectionQuery)(optionsWithDefaults)
      );
      const result = (0, _execute.executeSync)({
        schema,
        document
      });
      !result.errors && result.data || (0, _invariant.invariant)(false);
      return result.data;
    }
    __name(introspectionFromSchema, "introspectionFromSchema");
  }
});

// ../../node_modules/graphql/utilities/buildClientSchema.js
var require_buildClientSchema = __commonJS({
  "../../node_modules/graphql/utilities/buildClientSchema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.buildClientSchema = buildClientSchema;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _isObjectLike = require_isObjectLike();
    var _keyValMap = require_keyValMap();
    var _parser = require_parser();
    var _definition = require_definition();
    var _directives = require_directives();
    var _introspection = require_introspection();
    var _scalars = require_scalars();
    var _schema = require_schema();
    var _valueFromAST = require_valueFromAST();
    function buildClientSchema(introspection, options) {
      (0, _isObjectLike.isObjectLike)(introspection) && (0, _isObjectLike.isObjectLike)(introspection.__schema) || (0, _devAssert.devAssert)(
        false,
        `Invalid or incomplete introspection result. Ensure that you are passing "data" property of introspection response and no "errors" was returned alongside: ${(0, _inspect.inspect)(introspection)}.`
      );
      const schemaIntrospection = introspection.__schema;
      const typeMap = (0, _keyValMap.keyValMap)(
        schemaIntrospection.types,
        (typeIntrospection) => typeIntrospection.name,
        (typeIntrospection) => buildType(typeIntrospection)
      );
      for (const stdType of [
        ..._scalars.specifiedScalarTypes,
        ..._introspection.introspectionTypes
      ]) {
        if (typeMap[stdType.name]) {
          typeMap[stdType.name] = stdType;
        }
      }
      const queryType = schemaIntrospection.queryType ? getObjectType(schemaIntrospection.queryType) : null;
      const mutationType = schemaIntrospection.mutationType ? getObjectType(schemaIntrospection.mutationType) : null;
      const subscriptionType = schemaIntrospection.subscriptionType ? getObjectType(schemaIntrospection.subscriptionType) : null;
      const directives = schemaIntrospection.directives ? schemaIntrospection.directives.map(buildDirective) : [];
      return new _schema.GraphQLSchema({
        description: schemaIntrospection.description,
        query: queryType,
        mutation: mutationType,
        subscription: subscriptionType,
        types: Object.values(typeMap),
        directives,
        assumeValid: options === null || options === void 0 ? void 0 : options.assumeValid
      });
      function getType(typeRef) {
        if (typeRef.kind === _introspection.TypeKind.LIST) {
          const itemRef = typeRef.ofType;
          if (!itemRef) {
            throw new Error("Decorated type deeper than introspection query.");
          }
          return new _definition.GraphQLList(getType(itemRef));
        }
        if (typeRef.kind === _introspection.TypeKind.NON_NULL) {
          const nullableRef = typeRef.ofType;
          if (!nullableRef) {
            throw new Error("Decorated type deeper than introspection query.");
          }
          const nullableType = getType(nullableRef);
          return new _definition.GraphQLNonNull(
            (0, _definition.assertNullableType)(nullableType)
          );
        }
        return getNamedType(typeRef);
      }
      __name(getType, "getType");
      function getNamedType(typeRef) {
        const typeName = typeRef.name;
        if (!typeName) {
          throw new Error(
            `Unknown type reference: ${(0, _inspect.inspect)(typeRef)}.`
          );
        }
        const type = typeMap[typeName];
        if (!type) {
          throw new Error(
            `Invalid or incomplete schema, unknown type: ${typeName}. Ensure that a full introspection query is used in order to build a client schema.`
          );
        }
        return type;
      }
      __name(getNamedType, "getNamedType");
      function getObjectType(typeRef) {
        return (0, _definition.assertObjectType)(getNamedType(typeRef));
      }
      __name(getObjectType, "getObjectType");
      function getInterfaceType(typeRef) {
        return (0, _definition.assertInterfaceType)(getNamedType(typeRef));
      }
      __name(getInterfaceType, "getInterfaceType");
      function buildType(type) {
        if (type != null && type.name != null && type.kind != null) {
          switch (type.kind) {
            case _introspection.TypeKind.SCALAR:
              return buildScalarDef(type);
            case _introspection.TypeKind.OBJECT:
              return buildObjectDef(type);
            case _introspection.TypeKind.INTERFACE:
              return buildInterfaceDef(type);
            case _introspection.TypeKind.UNION:
              return buildUnionDef(type);
            case _introspection.TypeKind.ENUM:
              return buildEnumDef(type);
            case _introspection.TypeKind.INPUT_OBJECT:
              return buildInputObjectDef(type);
          }
        }
        const typeStr = (0, _inspect.inspect)(type);
        throw new Error(
          `Invalid or incomplete introspection result. Ensure that a full introspection query is used in order to build a client schema: ${typeStr}.`
        );
      }
      __name(buildType, "buildType");
      function buildScalarDef(scalarIntrospection) {
        return new _definition.GraphQLScalarType({
          name: scalarIntrospection.name,
          description: scalarIntrospection.description,
          specifiedByURL: scalarIntrospection.specifiedByURL
        });
      }
      __name(buildScalarDef, "buildScalarDef");
      function buildImplementationsList(implementingIntrospection) {
        if (implementingIntrospection.interfaces === null && implementingIntrospection.kind === _introspection.TypeKind.INTERFACE) {
          return [];
        }
        if (!implementingIntrospection.interfaces) {
          const implementingIntrospectionStr = (0, _inspect.inspect)(
            implementingIntrospection
          );
          throw new Error(
            `Introspection result missing interfaces: ${implementingIntrospectionStr}.`
          );
        }
        return implementingIntrospection.interfaces.map(getInterfaceType);
      }
      __name(buildImplementationsList, "buildImplementationsList");
      function buildObjectDef(objectIntrospection) {
        return new _definition.GraphQLObjectType({
          name: objectIntrospection.name,
          description: objectIntrospection.description,
          interfaces: /* @__PURE__ */ __name(() => buildImplementationsList(objectIntrospection), "interfaces"),
          fields: /* @__PURE__ */ __name(() => buildFieldDefMap(objectIntrospection), "fields")
        });
      }
      __name(buildObjectDef, "buildObjectDef");
      function buildInterfaceDef(interfaceIntrospection) {
        return new _definition.GraphQLInterfaceType({
          name: interfaceIntrospection.name,
          description: interfaceIntrospection.description,
          interfaces: /* @__PURE__ */ __name(() => buildImplementationsList(interfaceIntrospection), "interfaces"),
          fields: /* @__PURE__ */ __name(() => buildFieldDefMap(interfaceIntrospection), "fields")
        });
      }
      __name(buildInterfaceDef, "buildInterfaceDef");
      function buildUnionDef(unionIntrospection) {
        if (!unionIntrospection.possibleTypes) {
          const unionIntrospectionStr = (0, _inspect.inspect)(unionIntrospection);
          throw new Error(
            `Introspection result missing possibleTypes: ${unionIntrospectionStr}.`
          );
        }
        return new _definition.GraphQLUnionType({
          name: unionIntrospection.name,
          description: unionIntrospection.description,
          types: /* @__PURE__ */ __name(() => unionIntrospection.possibleTypes.map(getObjectType), "types")
        });
      }
      __name(buildUnionDef, "buildUnionDef");
      function buildEnumDef(enumIntrospection) {
        if (!enumIntrospection.enumValues) {
          const enumIntrospectionStr = (0, _inspect.inspect)(enumIntrospection);
          throw new Error(
            `Introspection result missing enumValues: ${enumIntrospectionStr}.`
          );
        }
        return new _definition.GraphQLEnumType({
          name: enumIntrospection.name,
          description: enumIntrospection.description,
          values: (0, _keyValMap.keyValMap)(
            enumIntrospection.enumValues,
            (valueIntrospection) => valueIntrospection.name,
            (valueIntrospection) => ({
              description: valueIntrospection.description,
              deprecationReason: valueIntrospection.deprecationReason
            })
          )
        });
      }
      __name(buildEnumDef, "buildEnumDef");
      function buildInputObjectDef(inputObjectIntrospection) {
        if (!inputObjectIntrospection.inputFields) {
          const inputObjectIntrospectionStr = (0, _inspect.inspect)(
            inputObjectIntrospection
          );
          throw new Error(
            `Introspection result missing inputFields: ${inputObjectIntrospectionStr}.`
          );
        }
        return new _definition.GraphQLInputObjectType({
          name: inputObjectIntrospection.name,
          description: inputObjectIntrospection.description,
          fields: /* @__PURE__ */ __name(() => buildInputValueDefMap(inputObjectIntrospection.inputFields), "fields"),
          isOneOf: inputObjectIntrospection.isOneOf
        });
      }
      __name(buildInputObjectDef, "buildInputObjectDef");
      function buildFieldDefMap(typeIntrospection) {
        if (!typeIntrospection.fields) {
          throw new Error(
            `Introspection result missing fields: ${(0, _inspect.inspect)(
              typeIntrospection
            )}.`
          );
        }
        return (0, _keyValMap.keyValMap)(
          typeIntrospection.fields,
          (fieldIntrospection) => fieldIntrospection.name,
          buildField
        );
      }
      __name(buildFieldDefMap, "buildFieldDefMap");
      function buildField(fieldIntrospection) {
        const type = getType(fieldIntrospection.type);
        if (!(0, _definition.isOutputType)(type)) {
          const typeStr = (0, _inspect.inspect)(type);
          throw new Error(
            `Introspection must provide output type for fields, but received: ${typeStr}.`
          );
        }
        if (!fieldIntrospection.args) {
          const fieldIntrospectionStr = (0, _inspect.inspect)(fieldIntrospection);
          throw new Error(
            `Introspection result missing field args: ${fieldIntrospectionStr}.`
          );
        }
        return {
          description: fieldIntrospection.description,
          deprecationReason: fieldIntrospection.deprecationReason,
          type,
          args: buildInputValueDefMap(fieldIntrospection.args)
        };
      }
      __name(buildField, "buildField");
      function buildInputValueDefMap(inputValueIntrospections) {
        return (0, _keyValMap.keyValMap)(
          inputValueIntrospections,
          (inputValue) => inputValue.name,
          buildInputValue
        );
      }
      __name(buildInputValueDefMap, "buildInputValueDefMap");
      function buildInputValue(inputValueIntrospection) {
        const type = getType(inputValueIntrospection.type);
        if (!(0, _definition.isInputType)(type)) {
          const typeStr = (0, _inspect.inspect)(type);
          throw new Error(
            `Introspection must provide input type for arguments, but received: ${typeStr}.`
          );
        }
        const defaultValue = inputValueIntrospection.defaultValue != null ? (0, _valueFromAST.valueFromAST)(
          (0, _parser.parseValue)(inputValueIntrospection.defaultValue),
          type
        ) : void 0;
        return {
          description: inputValueIntrospection.description,
          type,
          defaultValue,
          deprecationReason: inputValueIntrospection.deprecationReason
        };
      }
      __name(buildInputValue, "buildInputValue");
      function buildDirective(directiveIntrospection) {
        if (!directiveIntrospection.args) {
          const directiveIntrospectionStr = (0, _inspect.inspect)(
            directiveIntrospection
          );
          throw new Error(
            `Introspection result missing directive args: ${directiveIntrospectionStr}.`
          );
        }
        if (!directiveIntrospection.locations) {
          const directiveIntrospectionStr = (0, _inspect.inspect)(
            directiveIntrospection
          );
          throw new Error(
            `Introspection result missing directive locations: ${directiveIntrospectionStr}.`
          );
        }
        return new _directives.GraphQLDirective({
          name: directiveIntrospection.name,
          description: directiveIntrospection.description,
          isRepeatable: directiveIntrospection.isRepeatable,
          locations: directiveIntrospection.locations.slice(),
          args: buildInputValueDefMap(directiveIntrospection.args)
        });
      }
      __name(buildDirective, "buildDirective");
    }
    __name(buildClientSchema, "buildClientSchema");
  }
});

// ../../node_modules/graphql/utilities/extendSchema.js
var require_extendSchema = __commonJS({
  "../../node_modules/graphql/utilities/extendSchema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.extendSchema = extendSchema;
    exports.extendSchemaImpl = extendSchemaImpl;
    var _devAssert = require_devAssert();
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _keyMap = require_keyMap();
    var _mapValue = require_mapValue();
    var _kinds = require_kinds();
    var _predicates = require_predicates();
    var _definition = require_definition();
    var _directives = require_directives();
    var _introspection = require_introspection();
    var _scalars = require_scalars();
    var _schema = require_schema();
    var _validate = require_validate2();
    var _values = require_values();
    var _valueFromAST = require_valueFromAST();
    function extendSchema(schema, documentAST, options) {
      (0, _schema.assertSchema)(schema);
      documentAST != null && documentAST.kind === _kinds.Kind.DOCUMENT || (0, _devAssert.devAssert)(false, "Must provide valid Document AST.");
      if ((options === null || options === void 0 ? void 0 : options.assumeValid) !== true && (options === null || options === void 0 ? void 0 : options.assumeValidSDL) !== true) {
        (0, _validate.assertValidSDLExtension)(documentAST, schema);
      }
      const schemaConfig = schema.toConfig();
      const extendedConfig = extendSchemaImpl(schemaConfig, documentAST, options);
      return schemaConfig === extendedConfig ? schema : new _schema.GraphQLSchema(extendedConfig);
    }
    __name(extendSchema, "extendSchema");
    function extendSchemaImpl(schemaConfig, documentAST, options) {
      var _schemaDef, _schemaDef$descriptio, _schemaDef2, _options$assumeValid;
      const typeDefs = [];
      const typeExtensionsMap = /* @__PURE__ */ Object.create(null);
      const directiveDefs = [];
      let schemaDef;
      const schemaExtensions = [];
      for (const def of documentAST.definitions) {
        if (def.kind === _kinds.Kind.SCHEMA_DEFINITION) {
          schemaDef = def;
        } else if (def.kind === _kinds.Kind.SCHEMA_EXTENSION) {
          schemaExtensions.push(def);
        } else if ((0, _predicates.isTypeDefinitionNode)(def)) {
          typeDefs.push(def);
        } else if ((0, _predicates.isTypeExtensionNode)(def)) {
          const extendedTypeName = def.name.value;
          const existingTypeExtensions = typeExtensionsMap[extendedTypeName];
          typeExtensionsMap[extendedTypeName] = existingTypeExtensions ? existingTypeExtensions.concat([def]) : [def];
        } else if (def.kind === _kinds.Kind.DIRECTIVE_DEFINITION) {
          directiveDefs.push(def);
        }
      }
      if (Object.keys(typeExtensionsMap).length === 0 && typeDefs.length === 0 && directiveDefs.length === 0 && schemaExtensions.length === 0 && schemaDef == null) {
        return schemaConfig;
      }
      const typeMap = /* @__PURE__ */ Object.create(null);
      for (const existingType of schemaConfig.types) {
        typeMap[existingType.name] = extendNamedType(existingType);
      }
      for (const typeNode of typeDefs) {
        var _stdTypeMap$name;
        const name = typeNode.name.value;
        typeMap[name] = (_stdTypeMap$name = stdTypeMap[name]) !== null && _stdTypeMap$name !== void 0 ? _stdTypeMap$name : buildType(typeNode);
      }
      const operationTypes = {
        // Get the extended root operation types.
        query: schemaConfig.query && replaceNamedType(schemaConfig.query),
        mutation: schemaConfig.mutation && replaceNamedType(schemaConfig.mutation),
        subscription: schemaConfig.subscription && replaceNamedType(schemaConfig.subscription),
        // Then, incorporate schema definition and all schema extensions.
        ...schemaDef && getOperationTypes([schemaDef]),
        ...getOperationTypes(schemaExtensions)
      };
      return {
        description: (_schemaDef = schemaDef) === null || _schemaDef === void 0 ? void 0 : (_schemaDef$descriptio = _schemaDef.description) === null || _schemaDef$descriptio === void 0 ? void 0 : _schemaDef$descriptio.value,
        ...operationTypes,
        types: Object.values(typeMap),
        directives: [
          ...schemaConfig.directives.map(replaceDirective),
          ...directiveDefs.map(buildDirective)
        ],
        extensions: /* @__PURE__ */ Object.create(null),
        astNode: (_schemaDef2 = schemaDef) !== null && _schemaDef2 !== void 0 ? _schemaDef2 : schemaConfig.astNode,
        extensionASTNodes: schemaConfig.extensionASTNodes.concat(schemaExtensions),
        assumeValid: (_options$assumeValid = options === null || options === void 0 ? void 0 : options.assumeValid) !== null && _options$assumeValid !== void 0 ? _options$assumeValid : false
      };
      function replaceType(type) {
        if ((0, _definition.isListType)(type)) {
          return new _definition.GraphQLList(replaceType(type.ofType));
        }
        if ((0, _definition.isNonNullType)(type)) {
          return new _definition.GraphQLNonNull(replaceType(type.ofType));
        }
        return replaceNamedType(type);
      }
      __name(replaceType, "replaceType");
      function replaceNamedType(type) {
        return typeMap[type.name];
      }
      __name(replaceNamedType, "replaceNamedType");
      function replaceDirective(directive) {
        const config = directive.toConfig();
        return new _directives.GraphQLDirective({
          ...config,
          args: (0, _mapValue.mapValue)(config.args, extendArg)
        });
      }
      __name(replaceDirective, "replaceDirective");
      function extendNamedType(type) {
        if ((0, _introspection.isIntrospectionType)(type) || (0, _scalars.isSpecifiedScalarType)(type)) {
          return type;
        }
        if ((0, _definition.isScalarType)(type)) {
          return extendScalarType(type);
        }
        if ((0, _definition.isObjectType)(type)) {
          return extendObjectType(type);
        }
        if ((0, _definition.isInterfaceType)(type)) {
          return extendInterfaceType(type);
        }
        if ((0, _definition.isUnionType)(type)) {
          return extendUnionType(type);
        }
        if ((0, _definition.isEnumType)(type)) {
          return extendEnumType(type);
        }
        if ((0, _definition.isInputObjectType)(type)) {
          return extendInputObjectType(type);
        }
        (0, _invariant.invariant)(
          false,
          "Unexpected type: " + (0, _inspect.inspect)(type)
        );
      }
      __name(extendNamedType, "extendNamedType");
      function extendInputObjectType(type) {
        var _typeExtensionsMap$co;
        const config = type.toConfig();
        const extensions = (_typeExtensionsMap$co = typeExtensionsMap[config.name]) !== null && _typeExtensionsMap$co !== void 0 ? _typeExtensionsMap$co : [];
        return new _definition.GraphQLInputObjectType({
          ...config,
          fields: /* @__PURE__ */ __name(() => ({
            ...(0, _mapValue.mapValue)(config.fields, (field) => ({
              ...field,
              type: replaceType(field.type)
            })),
            ...buildInputFieldMap(extensions)
          }), "fields"),
          extensionASTNodes: config.extensionASTNodes.concat(extensions)
        });
      }
      __name(extendInputObjectType, "extendInputObjectType");
      function extendEnumType(type) {
        var _typeExtensionsMap$ty;
        const config = type.toConfig();
        const extensions = (_typeExtensionsMap$ty = typeExtensionsMap[type.name]) !== null && _typeExtensionsMap$ty !== void 0 ? _typeExtensionsMap$ty : [];
        return new _definition.GraphQLEnumType({
          ...config,
          values: { ...config.values, ...buildEnumValueMap(extensions) },
          extensionASTNodes: config.extensionASTNodes.concat(extensions)
        });
      }
      __name(extendEnumType, "extendEnumType");
      function extendScalarType(type) {
        var _typeExtensionsMap$co2;
        const config = type.toConfig();
        const extensions = (_typeExtensionsMap$co2 = typeExtensionsMap[config.name]) !== null && _typeExtensionsMap$co2 !== void 0 ? _typeExtensionsMap$co2 : [];
        let specifiedByURL = config.specifiedByURL;
        for (const extensionNode of extensions) {
          var _getSpecifiedByURL;
          specifiedByURL = (_getSpecifiedByURL = getSpecifiedByURL(extensionNode)) !== null && _getSpecifiedByURL !== void 0 ? _getSpecifiedByURL : specifiedByURL;
        }
        return new _definition.GraphQLScalarType({
          ...config,
          specifiedByURL,
          extensionASTNodes: config.extensionASTNodes.concat(extensions)
        });
      }
      __name(extendScalarType, "extendScalarType");
      function extendObjectType(type) {
        var _typeExtensionsMap$co3;
        const config = type.toConfig();
        const extensions = (_typeExtensionsMap$co3 = typeExtensionsMap[config.name]) !== null && _typeExtensionsMap$co3 !== void 0 ? _typeExtensionsMap$co3 : [];
        return new _definition.GraphQLObjectType({
          ...config,
          interfaces: /* @__PURE__ */ __name(() => [
            ...type.getInterfaces().map(replaceNamedType),
            ...buildInterfaces(extensions)
          ], "interfaces"),
          fields: /* @__PURE__ */ __name(() => ({
            ...(0, _mapValue.mapValue)(config.fields, extendField),
            ...buildFieldMap(extensions)
          }), "fields"),
          extensionASTNodes: config.extensionASTNodes.concat(extensions)
        });
      }
      __name(extendObjectType, "extendObjectType");
      function extendInterfaceType(type) {
        var _typeExtensionsMap$co4;
        const config = type.toConfig();
        const extensions = (_typeExtensionsMap$co4 = typeExtensionsMap[config.name]) !== null && _typeExtensionsMap$co4 !== void 0 ? _typeExtensionsMap$co4 : [];
        return new _definition.GraphQLInterfaceType({
          ...config,
          interfaces: /* @__PURE__ */ __name(() => [
            ...type.getInterfaces().map(replaceNamedType),
            ...buildInterfaces(extensions)
          ], "interfaces"),
          fields: /* @__PURE__ */ __name(() => ({
            ...(0, _mapValue.mapValue)(config.fields, extendField),
            ...buildFieldMap(extensions)
          }), "fields"),
          extensionASTNodes: config.extensionASTNodes.concat(extensions)
        });
      }
      __name(extendInterfaceType, "extendInterfaceType");
      function extendUnionType(type) {
        var _typeExtensionsMap$co5;
        const config = type.toConfig();
        const extensions = (_typeExtensionsMap$co5 = typeExtensionsMap[config.name]) !== null && _typeExtensionsMap$co5 !== void 0 ? _typeExtensionsMap$co5 : [];
        return new _definition.GraphQLUnionType({
          ...config,
          types: /* @__PURE__ */ __name(() => [
            ...type.getTypes().map(replaceNamedType),
            ...buildUnionTypes(extensions)
          ], "types"),
          extensionASTNodes: config.extensionASTNodes.concat(extensions)
        });
      }
      __name(extendUnionType, "extendUnionType");
      function extendField(field) {
        return {
          ...field,
          type: replaceType(field.type),
          args: field.args && (0, _mapValue.mapValue)(field.args, extendArg)
        };
      }
      __name(extendField, "extendField");
      function extendArg(arg) {
        return { ...arg, type: replaceType(arg.type) };
      }
      __name(extendArg, "extendArg");
      function getOperationTypes(nodes) {
        const opTypes = {};
        for (const node of nodes) {
          var _node$operationTypes;
          const operationTypesNodes = (
            /* c8 ignore next */
            (_node$operationTypes = node.operationTypes) !== null && _node$operationTypes !== void 0 ? _node$operationTypes : []
          );
          for (const operationType of operationTypesNodes) {
            opTypes[operationType.operation] = getNamedType(operationType.type);
          }
        }
        return opTypes;
      }
      __name(getOperationTypes, "getOperationTypes");
      function getNamedType(node) {
        var _stdTypeMap$name2;
        const name = node.name.value;
        const type = (_stdTypeMap$name2 = stdTypeMap[name]) !== null && _stdTypeMap$name2 !== void 0 ? _stdTypeMap$name2 : typeMap[name];
        if (type === void 0) {
          throw new Error(`Unknown type: "${name}".`);
        }
        return type;
      }
      __name(getNamedType, "getNamedType");
      function getWrappedType(node) {
        if (node.kind === _kinds.Kind.LIST_TYPE) {
          return new _definition.GraphQLList(getWrappedType(node.type));
        }
        if (node.kind === _kinds.Kind.NON_NULL_TYPE) {
          return new _definition.GraphQLNonNull(getWrappedType(node.type));
        }
        return getNamedType(node);
      }
      __name(getWrappedType, "getWrappedType");
      function buildDirective(node) {
        var _node$description;
        return new _directives.GraphQLDirective({
          name: node.name.value,
          description: (_node$description = node.description) === null || _node$description === void 0 ? void 0 : _node$description.value,
          // @ts-expect-error
          locations: node.locations.map(({ value }) => value),
          isRepeatable: node.repeatable,
          args: buildArgumentMap(node.arguments),
          astNode: node
        });
      }
      __name(buildDirective, "buildDirective");
      function buildFieldMap(nodes) {
        const fieldConfigMap = /* @__PURE__ */ Object.create(null);
        for (const node of nodes) {
          var _node$fields;
          const nodeFields = (
            /* c8 ignore next */
            (_node$fields = node.fields) !== null && _node$fields !== void 0 ? _node$fields : []
          );
          for (const field of nodeFields) {
            var _field$description;
            fieldConfigMap[field.name.value] = {
              // Note: While this could make assertions to get the correctly typed
              // value, that would throw immediately while type system validation
              // with validateSchema() will produce more actionable results.
              type: getWrappedType(field.type),
              description: (_field$description = field.description) === null || _field$description === void 0 ? void 0 : _field$description.value,
              args: buildArgumentMap(field.arguments),
              deprecationReason: getDeprecationReason(field),
              astNode: field
            };
          }
        }
        return fieldConfigMap;
      }
      __name(buildFieldMap, "buildFieldMap");
      function buildArgumentMap(args) {
        const argsNodes = (
          /* c8 ignore next */
          args !== null && args !== void 0 ? args : []
        );
        const argConfigMap = /* @__PURE__ */ Object.create(null);
        for (const arg of argsNodes) {
          var _arg$description;
          const type = getWrappedType(arg.type);
          argConfigMap[arg.name.value] = {
            type,
            description: (_arg$description = arg.description) === null || _arg$description === void 0 ? void 0 : _arg$description.value,
            defaultValue: (0, _valueFromAST.valueFromAST)(arg.defaultValue, type),
            deprecationReason: getDeprecationReason(arg),
            astNode: arg
          };
        }
        return argConfigMap;
      }
      __name(buildArgumentMap, "buildArgumentMap");
      function buildInputFieldMap(nodes) {
        const inputFieldMap = /* @__PURE__ */ Object.create(null);
        for (const node of nodes) {
          var _node$fields2;
          const fieldsNodes = (
            /* c8 ignore next */
            (_node$fields2 = node.fields) !== null && _node$fields2 !== void 0 ? _node$fields2 : []
          );
          for (const field of fieldsNodes) {
            var _field$description2;
            const type = getWrappedType(field.type);
            inputFieldMap[field.name.value] = {
              type,
              description: (_field$description2 = field.description) === null || _field$description2 === void 0 ? void 0 : _field$description2.value,
              defaultValue: (0, _valueFromAST.valueFromAST)(
                field.defaultValue,
                type
              ),
              deprecationReason: getDeprecationReason(field),
              astNode: field
            };
          }
        }
        return inputFieldMap;
      }
      __name(buildInputFieldMap, "buildInputFieldMap");
      function buildEnumValueMap(nodes) {
        const enumValueMap = /* @__PURE__ */ Object.create(null);
        for (const node of nodes) {
          var _node$values;
          const valuesNodes = (
            /* c8 ignore next */
            (_node$values = node.values) !== null && _node$values !== void 0 ? _node$values : []
          );
          for (const value of valuesNodes) {
            var _value$description;
            enumValueMap[value.name.value] = {
              description: (_value$description = value.description) === null || _value$description === void 0 ? void 0 : _value$description.value,
              deprecationReason: getDeprecationReason(value),
              astNode: value
            };
          }
        }
        return enumValueMap;
      }
      __name(buildEnumValueMap, "buildEnumValueMap");
      function buildInterfaces(nodes) {
        return nodes.flatMap(
          // FIXME: https://github.com/graphql/graphql-js/issues/2203
          (node) => {
            var _node$interfaces$map, _node$interfaces;
            return (
              /* c8 ignore next */
              (_node$interfaces$map = (_node$interfaces = node.interfaces) === null || _node$interfaces === void 0 ? void 0 : _node$interfaces.map(getNamedType)) !== null && _node$interfaces$map !== void 0 ? _node$interfaces$map : []
            );
          }
        );
      }
      __name(buildInterfaces, "buildInterfaces");
      function buildUnionTypes(nodes) {
        return nodes.flatMap(
          // FIXME: https://github.com/graphql/graphql-js/issues/2203
          (node) => {
            var _node$types$map, _node$types;
            return (
              /* c8 ignore next */
              (_node$types$map = (_node$types = node.types) === null || _node$types === void 0 ? void 0 : _node$types.map(getNamedType)) !== null && _node$types$map !== void 0 ? _node$types$map : []
            );
          }
        );
      }
      __name(buildUnionTypes, "buildUnionTypes");
      function buildType(astNode) {
        var _typeExtensionsMap$na;
        const name = astNode.name.value;
        const extensionASTNodes = (_typeExtensionsMap$na = typeExtensionsMap[name]) !== null && _typeExtensionsMap$na !== void 0 ? _typeExtensionsMap$na : [];
        switch (astNode.kind) {
          case _kinds.Kind.OBJECT_TYPE_DEFINITION: {
            var _astNode$description;
            const allNodes = [astNode, ...extensionASTNodes];
            return new _definition.GraphQLObjectType({
              name,
              description: (_astNode$description = astNode.description) === null || _astNode$description === void 0 ? void 0 : _astNode$description.value,
              interfaces: /* @__PURE__ */ __name(() => buildInterfaces(allNodes), "interfaces"),
              fields: /* @__PURE__ */ __name(() => buildFieldMap(allNodes), "fields"),
              astNode,
              extensionASTNodes
            });
          }
          case _kinds.Kind.INTERFACE_TYPE_DEFINITION: {
            var _astNode$description2;
            const allNodes = [astNode, ...extensionASTNodes];
            return new _definition.GraphQLInterfaceType({
              name,
              description: (_astNode$description2 = astNode.description) === null || _astNode$description2 === void 0 ? void 0 : _astNode$description2.value,
              interfaces: /* @__PURE__ */ __name(() => buildInterfaces(allNodes), "interfaces"),
              fields: /* @__PURE__ */ __name(() => buildFieldMap(allNodes), "fields"),
              astNode,
              extensionASTNodes
            });
          }
          case _kinds.Kind.ENUM_TYPE_DEFINITION: {
            var _astNode$description3;
            const allNodes = [astNode, ...extensionASTNodes];
            return new _definition.GraphQLEnumType({
              name,
              description: (_astNode$description3 = astNode.description) === null || _astNode$description3 === void 0 ? void 0 : _astNode$description3.value,
              values: buildEnumValueMap(allNodes),
              astNode,
              extensionASTNodes
            });
          }
          case _kinds.Kind.UNION_TYPE_DEFINITION: {
            var _astNode$description4;
            const allNodes = [astNode, ...extensionASTNodes];
            return new _definition.GraphQLUnionType({
              name,
              description: (_astNode$description4 = astNode.description) === null || _astNode$description4 === void 0 ? void 0 : _astNode$description4.value,
              types: /* @__PURE__ */ __name(() => buildUnionTypes(allNodes), "types"),
              astNode,
              extensionASTNodes
            });
          }
          case _kinds.Kind.SCALAR_TYPE_DEFINITION: {
            var _astNode$description5;
            return new _definition.GraphQLScalarType({
              name,
              description: (_astNode$description5 = astNode.description) === null || _astNode$description5 === void 0 ? void 0 : _astNode$description5.value,
              specifiedByURL: getSpecifiedByURL(astNode),
              astNode,
              extensionASTNodes
            });
          }
          case _kinds.Kind.INPUT_OBJECT_TYPE_DEFINITION: {
            var _astNode$description6;
            const allNodes = [astNode, ...extensionASTNodes];
            return new _definition.GraphQLInputObjectType({
              name,
              description: (_astNode$description6 = astNode.description) === null || _astNode$description6 === void 0 ? void 0 : _astNode$description6.value,
              fields: /* @__PURE__ */ __name(() => buildInputFieldMap(allNodes), "fields"),
              astNode,
              extensionASTNodes,
              isOneOf: isOneOf(astNode)
            });
          }
        }
      }
      __name(buildType, "buildType");
    }
    __name(extendSchemaImpl, "extendSchemaImpl");
    var stdTypeMap = (0, _keyMap.keyMap)(
      [..._scalars.specifiedScalarTypes, ..._introspection.introspectionTypes],
      (type) => type.name
    );
    function getDeprecationReason(node) {
      const deprecated = (0, _values.getDirectiveValues)(
        _directives.GraphQLDeprecatedDirective,
        node
      );
      return deprecated === null || deprecated === void 0 ? void 0 : deprecated.reason;
    }
    __name(getDeprecationReason, "getDeprecationReason");
    function getSpecifiedByURL(node) {
      const specifiedBy = (0, _values.getDirectiveValues)(
        _directives.GraphQLSpecifiedByDirective,
        node
      );
      return specifiedBy === null || specifiedBy === void 0 ? void 0 : specifiedBy.url;
    }
    __name(getSpecifiedByURL, "getSpecifiedByURL");
    function isOneOf(node) {
      return Boolean(
        (0, _values.getDirectiveValues)(_directives.GraphQLOneOfDirective, node)
      );
    }
    __name(isOneOf, "isOneOf");
  }
});

// ../../node_modules/graphql/utilities/buildASTSchema.js
var require_buildASTSchema = __commonJS({
  "../../node_modules/graphql/utilities/buildASTSchema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.buildASTSchema = buildASTSchema;
    exports.buildSchema = buildSchema;
    var _devAssert = require_devAssert();
    var _kinds = require_kinds();
    var _parser = require_parser();
    var _directives = require_directives();
    var _schema = require_schema();
    var _validate = require_validate2();
    var _extendSchema = require_extendSchema();
    function buildASTSchema(documentAST, options) {
      documentAST != null && documentAST.kind === _kinds.Kind.DOCUMENT || (0, _devAssert.devAssert)(false, "Must provide valid Document AST.");
      if ((options === null || options === void 0 ? void 0 : options.assumeValid) !== true && (options === null || options === void 0 ? void 0 : options.assumeValidSDL) !== true) {
        (0, _validate.assertValidSDL)(documentAST);
      }
      const emptySchemaConfig = {
        description: void 0,
        types: [],
        directives: [],
        extensions: /* @__PURE__ */ Object.create(null),
        extensionASTNodes: [],
        assumeValid: false
      };
      const config = (0, _extendSchema.extendSchemaImpl)(
        emptySchemaConfig,
        documentAST,
        options
      );
      if (config.astNode == null) {
        for (const type of config.types) {
          switch (type.name) {
            // Note: While this could make early assertions to get the correctly
            // typed values below, that would throw immediately while type system
            // validation with validateSchema() will produce more actionable results.
            case "Query":
              config.query = type;
              break;
            case "Mutation":
              config.mutation = type;
              break;
            case "Subscription":
              config.subscription = type;
              break;
          }
        }
      }
      const directives = [
        ...config.directives,
        // If specified directives were not explicitly declared, add them.
        ..._directives.specifiedDirectives.filter(
          (stdDirective) => config.directives.every(
            (directive) => directive.name !== stdDirective.name
          )
        )
      ];
      return new _schema.GraphQLSchema({ ...config, directives });
    }
    __name(buildASTSchema, "buildASTSchema");
    function buildSchema(source, options) {
      const document = (0, _parser.parse)(source, {
        noLocation: options === null || options === void 0 ? void 0 : options.noLocation,
        allowLegacyFragmentVariables: options === null || options === void 0 ? void 0 : options.allowLegacyFragmentVariables
      });
      return buildASTSchema(document, {
        assumeValidSDL: options === null || options === void 0 ? void 0 : options.assumeValidSDL,
        assumeValid: options === null || options === void 0 ? void 0 : options.assumeValid
      });
    }
    __name(buildSchema, "buildSchema");
  }
});

// ../../node_modules/graphql/utilities/lexicographicSortSchema.js
var require_lexicographicSortSchema = __commonJS({
  "../../node_modules/graphql/utilities/lexicographicSortSchema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.lexicographicSortSchema = lexicographicSortSchema;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _keyValMap = require_keyValMap();
    var _naturalCompare = require_naturalCompare();
    var _definition = require_definition();
    var _directives = require_directives();
    var _introspection = require_introspection();
    var _schema = require_schema();
    function lexicographicSortSchema(schema) {
      const schemaConfig = schema.toConfig();
      const typeMap = (0, _keyValMap.keyValMap)(
        sortByName(schemaConfig.types),
        (type) => type.name,
        sortNamedType
      );
      return new _schema.GraphQLSchema({
        ...schemaConfig,
        types: Object.values(typeMap),
        directives: sortByName(schemaConfig.directives).map(sortDirective),
        query: replaceMaybeType(schemaConfig.query),
        mutation: replaceMaybeType(schemaConfig.mutation),
        subscription: replaceMaybeType(schemaConfig.subscription)
      });
      function replaceType(type) {
        if ((0, _definition.isListType)(type)) {
          return new _definition.GraphQLList(replaceType(type.ofType));
        } else if ((0, _definition.isNonNullType)(type)) {
          return new _definition.GraphQLNonNull(replaceType(type.ofType));
        }
        return replaceNamedType(type);
      }
      __name(replaceType, "replaceType");
      function replaceNamedType(type) {
        return typeMap[type.name];
      }
      __name(replaceNamedType, "replaceNamedType");
      function replaceMaybeType(maybeType) {
        return maybeType && replaceNamedType(maybeType);
      }
      __name(replaceMaybeType, "replaceMaybeType");
      function sortDirective(directive) {
        const config = directive.toConfig();
        return new _directives.GraphQLDirective({
          ...config,
          locations: sortBy(config.locations, (x) => x),
          args: sortArgs(config.args)
        });
      }
      __name(sortDirective, "sortDirective");
      function sortArgs(args) {
        return sortObjMap(args, (arg) => ({ ...arg, type: replaceType(arg.type) }));
      }
      __name(sortArgs, "sortArgs");
      function sortFields(fieldsMap) {
        return sortObjMap(fieldsMap, (field) => ({
          ...field,
          type: replaceType(field.type),
          args: field.args && sortArgs(field.args)
        }));
      }
      __name(sortFields, "sortFields");
      function sortInputFields(fieldsMap) {
        return sortObjMap(fieldsMap, (field) => ({
          ...field,
          type: replaceType(field.type)
        }));
      }
      __name(sortInputFields, "sortInputFields");
      function sortTypes(array) {
        return sortByName(array).map(replaceNamedType);
      }
      __name(sortTypes, "sortTypes");
      function sortNamedType(type) {
        if ((0, _definition.isScalarType)(type) || (0, _introspection.isIntrospectionType)(type)) {
          return type;
        }
        if ((0, _definition.isObjectType)(type)) {
          const config = type.toConfig();
          return new _definition.GraphQLObjectType({
            ...config,
            interfaces: /* @__PURE__ */ __name(() => sortTypes(config.interfaces), "interfaces"),
            fields: /* @__PURE__ */ __name(() => sortFields(config.fields), "fields")
          });
        }
        if ((0, _definition.isInterfaceType)(type)) {
          const config = type.toConfig();
          return new _definition.GraphQLInterfaceType({
            ...config,
            interfaces: /* @__PURE__ */ __name(() => sortTypes(config.interfaces), "interfaces"),
            fields: /* @__PURE__ */ __name(() => sortFields(config.fields), "fields")
          });
        }
        if ((0, _definition.isUnionType)(type)) {
          const config = type.toConfig();
          return new _definition.GraphQLUnionType({
            ...config,
            types: /* @__PURE__ */ __name(() => sortTypes(config.types), "types")
          });
        }
        if ((0, _definition.isEnumType)(type)) {
          const config = type.toConfig();
          return new _definition.GraphQLEnumType({
            ...config,
            values: sortObjMap(config.values, (value) => value)
          });
        }
        if ((0, _definition.isInputObjectType)(type)) {
          const config = type.toConfig();
          return new _definition.GraphQLInputObjectType({
            ...config,
            fields: /* @__PURE__ */ __name(() => sortInputFields(config.fields), "fields")
          });
        }
        (0, _invariant.invariant)(
          false,
          "Unexpected type: " + (0, _inspect.inspect)(type)
        );
      }
      __name(sortNamedType, "sortNamedType");
    }
    __name(lexicographicSortSchema, "lexicographicSortSchema");
    function sortObjMap(map, sortValueFn) {
      const sortedMap = /* @__PURE__ */ Object.create(null);
      for (const key of Object.keys(map).sort(_naturalCompare.naturalCompare)) {
        sortedMap[key] = sortValueFn(map[key]);
      }
      return sortedMap;
    }
    __name(sortObjMap, "sortObjMap");
    function sortByName(array) {
      return sortBy(array, (obj) => obj.name);
    }
    __name(sortByName, "sortByName");
    function sortBy(array, mapToKey) {
      return array.slice().sort((obj1, obj2) => {
        const key1 = mapToKey(obj1);
        const key2 = mapToKey(obj2);
        return (0, _naturalCompare.naturalCompare)(key1, key2);
      });
    }
    __name(sortBy, "sortBy");
  }
});

// ../../node_modules/graphql/utilities/printSchema.js
var require_printSchema = __commonJS({
  "../../node_modules/graphql/utilities/printSchema.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.printIntrospectionSchema = printIntrospectionSchema;
    exports.printSchema = printSchema;
    exports.printType = printType;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _blockString = require_blockString();
    var _kinds = require_kinds();
    var _printer = require_printer();
    var _definition = require_definition();
    var _directives = require_directives();
    var _introspection = require_introspection();
    var _scalars = require_scalars();
    var _astFromValue = require_astFromValue();
    function printSchema(schema) {
      return printFilteredSchema(
        schema,
        (n) => !(0, _directives.isSpecifiedDirective)(n),
        isDefinedType
      );
    }
    __name(printSchema, "printSchema");
    function printIntrospectionSchema(schema) {
      return printFilteredSchema(
        schema,
        _directives.isSpecifiedDirective,
        _introspection.isIntrospectionType
      );
    }
    __name(printIntrospectionSchema, "printIntrospectionSchema");
    function isDefinedType(type) {
      return !(0, _scalars.isSpecifiedScalarType)(type) && !(0, _introspection.isIntrospectionType)(type);
    }
    __name(isDefinedType, "isDefinedType");
    function printFilteredSchema(schema, directiveFilter, typeFilter) {
      const directives = schema.getDirectives().filter(directiveFilter);
      const types = Object.values(schema.getTypeMap()).filter(typeFilter);
      return [
        printSchemaDefinition(schema),
        ...directives.map((directive) => printDirective(directive)),
        ...types.map((type) => printType(type))
      ].filter(Boolean).join("\n\n");
    }
    __name(printFilteredSchema, "printFilteredSchema");
    function printSchemaDefinition(schema) {
      if (schema.description == null && isSchemaOfCommonNames(schema)) {
        return;
      }
      const operationTypes = [];
      const queryType = schema.getQueryType();
      if (queryType) {
        operationTypes.push(`  query: ${queryType.name}`);
      }
      const mutationType = schema.getMutationType();
      if (mutationType) {
        operationTypes.push(`  mutation: ${mutationType.name}`);
      }
      const subscriptionType = schema.getSubscriptionType();
      if (subscriptionType) {
        operationTypes.push(`  subscription: ${subscriptionType.name}`);
      }
      return printDescription(schema) + `schema {
${operationTypes.join("\n")}
}`;
    }
    __name(printSchemaDefinition, "printSchemaDefinition");
    function isSchemaOfCommonNames(schema) {
      const queryType = schema.getQueryType();
      if (queryType && queryType.name !== "Query") {
        return false;
      }
      const mutationType = schema.getMutationType();
      if (mutationType && mutationType.name !== "Mutation") {
        return false;
      }
      const subscriptionType = schema.getSubscriptionType();
      if (subscriptionType && subscriptionType.name !== "Subscription") {
        return false;
      }
      return true;
    }
    __name(isSchemaOfCommonNames, "isSchemaOfCommonNames");
    function printType(type) {
      if ((0, _definition.isScalarType)(type)) {
        return printScalar(type);
      }
      if ((0, _definition.isObjectType)(type)) {
        return printObject(type);
      }
      if ((0, _definition.isInterfaceType)(type)) {
        return printInterface(type);
      }
      if ((0, _definition.isUnionType)(type)) {
        return printUnion(type);
      }
      if ((0, _definition.isEnumType)(type)) {
        return printEnum(type);
      }
      if ((0, _definition.isInputObjectType)(type)) {
        return printInputObject(type);
      }
      (0, _invariant.invariant)(
        false,
        "Unexpected type: " + (0, _inspect.inspect)(type)
      );
    }
    __name(printType, "printType");
    function printScalar(type) {
      return printDescription(type) + `scalar ${type.name}` + printSpecifiedByURL(type);
    }
    __name(printScalar, "printScalar");
    function printImplementedInterfaces(type) {
      const interfaces = type.getInterfaces();
      return interfaces.length ? " implements " + interfaces.map((i) => i.name).join(" & ") : "";
    }
    __name(printImplementedInterfaces, "printImplementedInterfaces");
    function printObject(type) {
      return printDescription(type) + `type ${type.name}` + printImplementedInterfaces(type) + printFields(type);
    }
    __name(printObject, "printObject");
    function printInterface(type) {
      return printDescription(type) + `interface ${type.name}` + printImplementedInterfaces(type) + printFields(type);
    }
    __name(printInterface, "printInterface");
    function printUnion(type) {
      const types = type.getTypes();
      const possibleTypes = types.length ? " = " + types.join(" | ") : "";
      return printDescription(type) + "union " + type.name + possibleTypes;
    }
    __name(printUnion, "printUnion");
    function printEnum(type) {
      const values = type.getValues().map(
        (value, i) => printDescription(value, "  ", !i) + "  " + value.name + printDeprecated(value.deprecationReason)
      );
      return printDescription(type) + `enum ${type.name}` + printBlock(values);
    }
    __name(printEnum, "printEnum");
    function printInputObject(type) {
      const fields = Object.values(type.getFields()).map(
        (f, i) => printDescription(f, "  ", !i) + "  " + printInputValue(f)
      );
      return printDescription(type) + `input ${type.name}` + (type.isOneOf ? " @oneOf" : "") + printBlock(fields);
    }
    __name(printInputObject, "printInputObject");
    function printFields(type) {
      const fields = Object.values(type.getFields()).map(
        (f, i) => printDescription(f, "  ", !i) + "  " + f.name + printArgs(f.args, "  ") + ": " + String(f.type) + printDeprecated(f.deprecationReason)
      );
      return printBlock(fields);
    }
    __name(printFields, "printFields");
    function printBlock(items) {
      return items.length !== 0 ? " {\n" + items.join("\n") + "\n}" : "";
    }
    __name(printBlock, "printBlock");
    function printArgs(args, indentation = "") {
      if (args.length === 0) {
        return "";
      }
      if (args.every((arg) => !arg.description)) {
        return "(" + args.map(printInputValue).join(", ") + ")";
      }
      return "(\n" + args.map(
        (arg, i) => printDescription(arg, "  " + indentation, !i) + "  " + indentation + printInputValue(arg)
      ).join("\n") + "\n" + indentation + ")";
    }
    __name(printArgs, "printArgs");
    function printInputValue(arg) {
      const defaultAST = (0, _astFromValue.astFromValue)(
        arg.defaultValue,
        arg.type
      );
      let argDecl = arg.name + ": " + String(arg.type);
      if (defaultAST) {
        argDecl += ` = ${(0, _printer.print)(defaultAST)}`;
      }
      return argDecl + printDeprecated(arg.deprecationReason);
    }
    __name(printInputValue, "printInputValue");
    function printDirective(directive) {
      return printDescription(directive) + "directive @" + directive.name + printArgs(directive.args) + (directive.isRepeatable ? " repeatable" : "") + " on " + directive.locations.join(" | ");
    }
    __name(printDirective, "printDirective");
    function printDeprecated(reason) {
      if (reason == null) {
        return "";
      }
      if (reason !== _directives.DEFAULT_DEPRECATION_REASON) {
        const astValue = (0, _printer.print)({
          kind: _kinds.Kind.STRING,
          value: reason
        });
        return ` @deprecated(reason: ${astValue})`;
      }
      return " @deprecated";
    }
    __name(printDeprecated, "printDeprecated");
    function printSpecifiedByURL(scalar) {
      if (scalar.specifiedByURL == null) {
        return "";
      }
      const astValue = (0, _printer.print)({
        kind: _kinds.Kind.STRING,
        value: scalar.specifiedByURL
      });
      return ` @specifiedBy(url: ${astValue})`;
    }
    __name(printSpecifiedByURL, "printSpecifiedByURL");
    function printDescription(def, indentation = "", firstInBlock = true) {
      const { description } = def;
      if (description == null) {
        return "";
      }
      const blockString = (0, _printer.print)({
        kind: _kinds.Kind.STRING,
        value: description,
        block: (0, _blockString.isPrintableAsBlockString)(description)
      });
      const prefix = indentation && !firstInBlock ? "\n" + indentation : indentation;
      return prefix + blockString.replace(/\n/g, "\n" + indentation) + "\n";
    }
    __name(printDescription, "printDescription");
  }
});

// ../../node_modules/graphql/utilities/concatAST.js
var require_concatAST = __commonJS({
  "../../node_modules/graphql/utilities/concatAST.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.concatAST = concatAST;
    var _kinds = require_kinds();
    function concatAST(documents) {
      const definitions = [];
      for (const doc of documents) {
        definitions.push(...doc.definitions);
      }
      return {
        kind: _kinds.Kind.DOCUMENT,
        definitions
      };
    }
    __name(concatAST, "concatAST");
  }
});

// ../../node_modules/graphql/utilities/separateOperations.js
var require_separateOperations = __commonJS({
  "../../node_modules/graphql/utilities/separateOperations.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.separateOperations = separateOperations;
    var _kinds = require_kinds();
    var _visitor = require_visitor();
    function separateOperations(documentAST) {
      const operations = [];
      const depGraph = /* @__PURE__ */ Object.create(null);
      for (const definitionNode of documentAST.definitions) {
        switch (definitionNode.kind) {
          case _kinds.Kind.OPERATION_DEFINITION:
            operations.push(definitionNode);
            break;
          case _kinds.Kind.FRAGMENT_DEFINITION:
            depGraph[definitionNode.name.value] = collectDependencies(
              definitionNode.selectionSet
            );
            break;
          default:
        }
      }
      const separatedDocumentASTs = /* @__PURE__ */ Object.create(null);
      for (const operation of operations) {
        const dependencies = /* @__PURE__ */ new Set();
        for (const fragmentName of collectDependencies(operation.selectionSet)) {
          collectTransitiveDependencies(dependencies, depGraph, fragmentName);
        }
        const operationName = operation.name ? operation.name.value : "";
        separatedDocumentASTs[operationName] = {
          kind: _kinds.Kind.DOCUMENT,
          definitions: documentAST.definitions.filter(
            (node) => node === operation || node.kind === _kinds.Kind.FRAGMENT_DEFINITION && dependencies.has(node.name.value)
          )
        };
      }
      return separatedDocumentASTs;
    }
    __name(separateOperations, "separateOperations");
    function collectTransitiveDependencies(collected, depGraph, fromName) {
      if (!collected.has(fromName)) {
        collected.add(fromName);
        const immediateDeps = depGraph[fromName];
        if (immediateDeps !== void 0) {
          for (const toName of immediateDeps) {
            collectTransitiveDependencies(collected, depGraph, toName);
          }
        }
      }
    }
    __name(collectTransitiveDependencies, "collectTransitiveDependencies");
    function collectDependencies(selectionSet) {
      const dependencies = [];
      (0, _visitor.visit)(selectionSet, {
        FragmentSpread(node) {
          dependencies.push(node.name.value);
        }
      });
      return dependencies;
    }
    __name(collectDependencies, "collectDependencies");
  }
});

// ../../node_modules/graphql/utilities/stripIgnoredCharacters.js
var require_stripIgnoredCharacters = __commonJS({
  "../../node_modules/graphql/utilities/stripIgnoredCharacters.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.stripIgnoredCharacters = stripIgnoredCharacters;
    var _blockString = require_blockString();
    var _lexer = require_lexer();
    var _source = require_source();
    var _tokenKind = require_tokenKind();
    function stripIgnoredCharacters(source) {
      const sourceObj = (0, _source.isSource)(source) ? source : new _source.Source(source);
      const body = sourceObj.body;
      const lexer = new _lexer.Lexer(sourceObj);
      let strippedBody = "";
      let wasLastAddedTokenNonPunctuator = false;
      while (lexer.advance().kind !== _tokenKind.TokenKind.EOF) {
        const currentToken = lexer.token;
        const tokenKind = currentToken.kind;
        const isNonPunctuator = !(0, _lexer.isPunctuatorTokenKind)(
          currentToken.kind
        );
        if (wasLastAddedTokenNonPunctuator) {
          if (isNonPunctuator || currentToken.kind === _tokenKind.TokenKind.SPREAD) {
            strippedBody += " ";
          }
        }
        const tokenBody = body.slice(currentToken.start, currentToken.end);
        if (tokenKind === _tokenKind.TokenKind.BLOCK_STRING) {
          strippedBody += (0, _blockString.printBlockString)(currentToken.value, {
            minimize: true
          });
        } else {
          strippedBody += tokenBody;
        }
        wasLastAddedTokenNonPunctuator = isNonPunctuator;
      }
      return strippedBody;
    }
    __name(stripIgnoredCharacters, "stripIgnoredCharacters");
  }
});

// ../../node_modules/graphql/utilities/assertValidName.js
var require_assertValidName = __commonJS({
  "../../node_modules/graphql/utilities/assertValidName.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.assertValidName = assertValidName;
    exports.isValidNameError = isValidNameError;
    var _devAssert = require_devAssert();
    var _GraphQLError = require_GraphQLError();
    var _assertName = require_assertName();
    function assertValidName(name) {
      const error = isValidNameError(name);
      if (error) {
        throw error;
      }
      return name;
    }
    __name(assertValidName, "assertValidName");
    function isValidNameError(name) {
      typeof name === "string" || (0, _devAssert.devAssert)(false, "Expected name to be a string.");
      if (name.startsWith("__")) {
        return new _GraphQLError.GraphQLError(
          `Name "${name}" must not begin with "__", which is reserved by GraphQL introspection.`
        );
      }
      try {
        (0, _assertName.assertName)(name);
      } catch (error) {
        return error;
      }
    }
    __name(isValidNameError, "isValidNameError");
  }
});

// ../../node_modules/graphql/utilities/findBreakingChanges.js
var require_findBreakingChanges = __commonJS({
  "../../node_modules/graphql/utilities/findBreakingChanges.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    exports.DangerousChangeType = exports.BreakingChangeType = void 0;
    exports.findBreakingChanges = findBreakingChanges;
    exports.findDangerousChanges = findDangerousChanges;
    var _inspect = require_inspect();
    var _invariant = require_invariant();
    var _keyMap = require_keyMap();
    var _printer = require_printer();
    var _definition = require_definition();
    var _scalars = require_scalars();
    var _astFromValue = require_astFromValue();
    var _sortValueNode = require_sortValueNode();
    var BreakingChangeType;
    exports.BreakingChangeType = BreakingChangeType;
    (function(BreakingChangeType2) {
      BreakingChangeType2["TYPE_REMOVED"] = "TYPE_REMOVED";
      BreakingChangeType2["TYPE_CHANGED_KIND"] = "TYPE_CHANGED_KIND";
      BreakingChangeType2["TYPE_REMOVED_FROM_UNION"] = "TYPE_REMOVED_FROM_UNION";
      BreakingChangeType2["VALUE_REMOVED_FROM_ENUM"] = "VALUE_REMOVED_FROM_ENUM";
      BreakingChangeType2["REQUIRED_INPUT_FIELD_ADDED"] = "REQUIRED_INPUT_FIELD_ADDED";
      BreakingChangeType2["IMPLEMENTED_INTERFACE_REMOVED"] = "IMPLEMENTED_INTERFACE_REMOVED";
      BreakingChangeType2["FIELD_REMOVED"] = "FIELD_REMOVED";
      BreakingChangeType2["FIELD_CHANGED_KIND"] = "FIELD_CHANGED_KIND";
      BreakingChangeType2["REQUIRED_ARG_ADDED"] = "REQUIRED_ARG_ADDED";
      BreakingChangeType2["ARG_REMOVED"] = "ARG_REMOVED";
      BreakingChangeType2["ARG_CHANGED_KIND"] = "ARG_CHANGED_KIND";
      BreakingChangeType2["DIRECTIVE_REMOVED"] = "DIRECTIVE_REMOVED";
      BreakingChangeType2["DIRECTIVE_ARG_REMOVED"] = "DIRECTIVE_ARG_REMOVED";
      BreakingChangeType2["REQUIRED_DIRECTIVE_ARG_ADDED"] = "REQUIRED_DIRECTIVE_ARG_ADDED";
      BreakingChangeType2["DIRECTIVE_REPEATABLE_REMOVED"] = "DIRECTIVE_REPEATABLE_REMOVED";
      BreakingChangeType2["DIRECTIVE_LOCATION_REMOVED"] = "DIRECTIVE_LOCATION_REMOVED";
    })(
      BreakingChangeType || (exports.BreakingChangeType = BreakingChangeType = {})
    );
    var DangerousChangeType;
    exports.DangerousChangeType = DangerousChangeType;
    (function(DangerousChangeType2) {
      DangerousChangeType2["VALUE_ADDED_TO_ENUM"] = "VALUE_ADDED_TO_ENUM";
      DangerousChangeType2["TYPE_ADDED_TO_UNION"] = "TYPE_ADDED_TO_UNION";
      DangerousChangeType2["OPTIONAL_INPUT_FIELD_ADDED"] = "OPTIONAL_INPUT_FIELD_ADDED";
      DangerousChangeType2["OPTIONAL_ARG_ADDED"] = "OPTIONAL_ARG_ADDED";
      DangerousChangeType2["IMPLEMENTED_INTERFACE_ADDED"] = "IMPLEMENTED_INTERFACE_ADDED";
      DangerousChangeType2["ARG_DEFAULT_VALUE_CHANGE"] = "ARG_DEFAULT_VALUE_CHANGE";
    })(
      DangerousChangeType || (exports.DangerousChangeType = DangerousChangeType = {})
    );
    function findBreakingChanges(oldSchema, newSchema) {
      return findSchemaChanges(oldSchema, newSchema).filter(
        (change) => change.type in BreakingChangeType
      );
    }
    __name(findBreakingChanges, "findBreakingChanges");
    function findDangerousChanges(oldSchema, newSchema) {
      return findSchemaChanges(oldSchema, newSchema).filter(
        (change) => change.type in DangerousChangeType
      );
    }
    __name(findDangerousChanges, "findDangerousChanges");
    function findSchemaChanges(oldSchema, newSchema) {
      return [
        ...findTypeChanges(oldSchema, newSchema),
        ...findDirectiveChanges(oldSchema, newSchema)
      ];
    }
    __name(findSchemaChanges, "findSchemaChanges");
    function findDirectiveChanges(oldSchema, newSchema) {
      const schemaChanges = [];
      const directivesDiff = diff(
        oldSchema.getDirectives(),
        newSchema.getDirectives()
      );
      for (const oldDirective of directivesDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.DIRECTIVE_REMOVED,
          description: `${oldDirective.name} was removed.`
        });
      }
      for (const [oldDirective, newDirective] of directivesDiff.persisted) {
        const argsDiff = diff(oldDirective.args, newDirective.args);
        for (const newArg of argsDiff.added) {
          if ((0, _definition.isRequiredArgument)(newArg)) {
            schemaChanges.push({
              type: BreakingChangeType.REQUIRED_DIRECTIVE_ARG_ADDED,
              description: `A required arg ${newArg.name} on directive ${oldDirective.name} was added.`
            });
          }
        }
        for (const oldArg of argsDiff.removed) {
          schemaChanges.push({
            type: BreakingChangeType.DIRECTIVE_ARG_REMOVED,
            description: `${oldArg.name} was removed from ${oldDirective.name}.`
          });
        }
        if (oldDirective.isRepeatable && !newDirective.isRepeatable) {
          schemaChanges.push({
            type: BreakingChangeType.DIRECTIVE_REPEATABLE_REMOVED,
            description: `Repeatable flag was removed from ${oldDirective.name}.`
          });
        }
        for (const location of oldDirective.locations) {
          if (!newDirective.locations.includes(location)) {
            schemaChanges.push({
              type: BreakingChangeType.DIRECTIVE_LOCATION_REMOVED,
              description: `${location} was removed from ${oldDirective.name}.`
            });
          }
        }
      }
      return schemaChanges;
    }
    __name(findDirectiveChanges, "findDirectiveChanges");
    function findTypeChanges(oldSchema, newSchema) {
      const schemaChanges = [];
      const typesDiff = diff(
        Object.values(oldSchema.getTypeMap()),
        Object.values(newSchema.getTypeMap())
      );
      for (const oldType of typesDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.TYPE_REMOVED,
          description: (0, _scalars.isSpecifiedScalarType)(oldType) ? `Standard scalar ${oldType.name} was removed because it is not referenced anymore.` : `${oldType.name} was removed.`
        });
      }
      for (const [oldType, newType] of typesDiff.persisted) {
        if ((0, _definition.isEnumType)(oldType) && (0, _definition.isEnumType)(newType)) {
          schemaChanges.push(...findEnumTypeChanges(oldType, newType));
        } else if ((0, _definition.isUnionType)(oldType) && (0, _definition.isUnionType)(newType)) {
          schemaChanges.push(...findUnionTypeChanges(oldType, newType));
        } else if ((0, _definition.isInputObjectType)(oldType) && (0, _definition.isInputObjectType)(newType)) {
          schemaChanges.push(...findInputObjectTypeChanges(oldType, newType));
        } else if ((0, _definition.isObjectType)(oldType) && (0, _definition.isObjectType)(newType)) {
          schemaChanges.push(
            ...findFieldChanges(oldType, newType),
            ...findImplementedInterfacesChanges(oldType, newType)
          );
        } else if ((0, _definition.isInterfaceType)(oldType) && (0, _definition.isInterfaceType)(newType)) {
          schemaChanges.push(
            ...findFieldChanges(oldType, newType),
            ...findImplementedInterfacesChanges(oldType, newType)
          );
        } else if (oldType.constructor !== newType.constructor) {
          schemaChanges.push({
            type: BreakingChangeType.TYPE_CHANGED_KIND,
            description: `${oldType.name} changed from ${typeKindName(oldType)} to ${typeKindName(newType)}.`
          });
        }
      }
      return schemaChanges;
    }
    __name(findTypeChanges, "findTypeChanges");
    function findInputObjectTypeChanges(oldType, newType) {
      const schemaChanges = [];
      const fieldsDiff = diff(
        Object.values(oldType.getFields()),
        Object.values(newType.getFields())
      );
      for (const newField of fieldsDiff.added) {
        if ((0, _definition.isRequiredInputField)(newField)) {
          schemaChanges.push({
            type: BreakingChangeType.REQUIRED_INPUT_FIELD_ADDED,
            description: `A required field ${newField.name} on input type ${oldType.name} was added.`
          });
        } else {
          schemaChanges.push({
            type: DangerousChangeType.OPTIONAL_INPUT_FIELD_ADDED,
            description: `An optional field ${newField.name} on input type ${oldType.name} was added.`
          });
        }
      }
      for (const oldField of fieldsDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.FIELD_REMOVED,
          description: `${oldType.name}.${oldField.name} was removed.`
        });
      }
      for (const [oldField, newField] of fieldsDiff.persisted) {
        const isSafe = isChangeSafeForInputObjectFieldOrFieldArg(
          oldField.type,
          newField.type
        );
        if (!isSafe) {
          schemaChanges.push({
            type: BreakingChangeType.FIELD_CHANGED_KIND,
            description: `${oldType.name}.${oldField.name} changed type from ${String(oldField.type)} to ${String(newField.type)}.`
          });
        }
      }
      return schemaChanges;
    }
    __name(findInputObjectTypeChanges, "findInputObjectTypeChanges");
    function findUnionTypeChanges(oldType, newType) {
      const schemaChanges = [];
      const possibleTypesDiff = diff(oldType.getTypes(), newType.getTypes());
      for (const newPossibleType of possibleTypesDiff.added) {
        schemaChanges.push({
          type: DangerousChangeType.TYPE_ADDED_TO_UNION,
          description: `${newPossibleType.name} was added to union type ${oldType.name}.`
        });
      }
      for (const oldPossibleType of possibleTypesDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.TYPE_REMOVED_FROM_UNION,
          description: `${oldPossibleType.name} was removed from union type ${oldType.name}.`
        });
      }
      return schemaChanges;
    }
    __name(findUnionTypeChanges, "findUnionTypeChanges");
    function findEnumTypeChanges(oldType, newType) {
      const schemaChanges = [];
      const valuesDiff = diff(oldType.getValues(), newType.getValues());
      for (const newValue of valuesDiff.added) {
        schemaChanges.push({
          type: DangerousChangeType.VALUE_ADDED_TO_ENUM,
          description: `${newValue.name} was added to enum type ${oldType.name}.`
        });
      }
      for (const oldValue of valuesDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.VALUE_REMOVED_FROM_ENUM,
          description: `${oldValue.name} was removed from enum type ${oldType.name}.`
        });
      }
      return schemaChanges;
    }
    __name(findEnumTypeChanges, "findEnumTypeChanges");
    function findImplementedInterfacesChanges(oldType, newType) {
      const schemaChanges = [];
      const interfacesDiff = diff(oldType.getInterfaces(), newType.getInterfaces());
      for (const newInterface of interfacesDiff.added) {
        schemaChanges.push({
          type: DangerousChangeType.IMPLEMENTED_INTERFACE_ADDED,
          description: `${newInterface.name} added to interfaces implemented by ${oldType.name}.`
        });
      }
      for (const oldInterface of interfacesDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.IMPLEMENTED_INTERFACE_REMOVED,
          description: `${oldType.name} no longer implements interface ${oldInterface.name}.`
        });
      }
      return schemaChanges;
    }
    __name(findImplementedInterfacesChanges, "findImplementedInterfacesChanges");
    function findFieldChanges(oldType, newType) {
      const schemaChanges = [];
      const fieldsDiff = diff(
        Object.values(oldType.getFields()),
        Object.values(newType.getFields())
      );
      for (const oldField of fieldsDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.FIELD_REMOVED,
          description: `${oldType.name}.${oldField.name} was removed.`
        });
      }
      for (const [oldField, newField] of fieldsDiff.persisted) {
        schemaChanges.push(...findArgChanges(oldType, oldField, newField));
        const isSafe = isChangeSafeForObjectOrInterfaceField(
          oldField.type,
          newField.type
        );
        if (!isSafe) {
          schemaChanges.push({
            type: BreakingChangeType.FIELD_CHANGED_KIND,
            description: `${oldType.name}.${oldField.name} changed type from ${String(oldField.type)} to ${String(newField.type)}.`
          });
        }
      }
      return schemaChanges;
    }
    __name(findFieldChanges, "findFieldChanges");
    function findArgChanges(oldType, oldField, newField) {
      const schemaChanges = [];
      const argsDiff = diff(oldField.args, newField.args);
      for (const oldArg of argsDiff.removed) {
        schemaChanges.push({
          type: BreakingChangeType.ARG_REMOVED,
          description: `${oldType.name}.${oldField.name} arg ${oldArg.name} was removed.`
        });
      }
      for (const [oldArg, newArg] of argsDiff.persisted) {
        const isSafe = isChangeSafeForInputObjectFieldOrFieldArg(
          oldArg.type,
          newArg.type
        );
        if (!isSafe) {
          schemaChanges.push({
            type: BreakingChangeType.ARG_CHANGED_KIND,
            description: `${oldType.name}.${oldField.name} arg ${oldArg.name} has changed type from ${String(oldArg.type)} to ${String(newArg.type)}.`
          });
        } else if (oldArg.defaultValue !== void 0) {
          if (newArg.defaultValue === void 0) {
            schemaChanges.push({
              type: DangerousChangeType.ARG_DEFAULT_VALUE_CHANGE,
              description: `${oldType.name}.${oldField.name} arg ${oldArg.name} defaultValue was removed.`
            });
          } else {
            const oldValueStr = stringifyValue(oldArg.defaultValue, oldArg.type);
            const newValueStr = stringifyValue(newArg.defaultValue, newArg.type);
            if (oldValueStr !== newValueStr) {
              schemaChanges.push({
                type: DangerousChangeType.ARG_DEFAULT_VALUE_CHANGE,
                description: `${oldType.name}.${oldField.name} arg ${oldArg.name} has changed defaultValue from ${oldValueStr} to ${newValueStr}.`
              });
            }
          }
        }
      }
      for (const newArg of argsDiff.added) {
        if ((0, _definition.isRequiredArgument)(newArg)) {
          schemaChanges.push({
            type: BreakingChangeType.REQUIRED_ARG_ADDED,
            description: `A required arg ${newArg.name} on ${oldType.name}.${oldField.name} was added.`
          });
        } else {
          schemaChanges.push({
            type: DangerousChangeType.OPTIONAL_ARG_ADDED,
            description: `An optional arg ${newArg.name} on ${oldType.name}.${oldField.name} was added.`
          });
        }
      }
      return schemaChanges;
    }
    __name(findArgChanges, "findArgChanges");
    function isChangeSafeForObjectOrInterfaceField(oldType, newType) {
      if ((0, _definition.isListType)(oldType)) {
        return (
          // if they're both lists, make sure the underlying types are compatible
          (0, _definition.isListType)(newType) && isChangeSafeForObjectOrInterfaceField(
            oldType.ofType,
            newType.ofType
          ) || // moving from nullable to non-null of the same underlying type is safe
          (0, _definition.isNonNullType)(newType) && isChangeSafeForObjectOrInterfaceField(oldType, newType.ofType)
        );
      }
      if ((0, _definition.isNonNullType)(oldType)) {
        return (0, _definition.isNonNullType)(newType) && isChangeSafeForObjectOrInterfaceField(oldType.ofType, newType.ofType);
      }
      return (
        // if they're both named types, see if their names are equivalent
        (0, _definition.isNamedType)(newType) && oldType.name === newType.name || // moving from nullable to non-null of the same underlying type is safe
        (0, _definition.isNonNullType)(newType) && isChangeSafeForObjectOrInterfaceField(oldType, newType.ofType)
      );
    }
    __name(isChangeSafeForObjectOrInterfaceField, "isChangeSafeForObjectOrInterfaceField");
    function isChangeSafeForInputObjectFieldOrFieldArg(oldType, newType) {
      if ((0, _definition.isListType)(oldType)) {
        return (0, _definition.isListType)(newType) && isChangeSafeForInputObjectFieldOrFieldArg(oldType.ofType, newType.ofType);
      }
      if ((0, _definition.isNonNullType)(oldType)) {
        return (
          // if they're both non-null, make sure the underlying types are
          // compatible
          (0, _definition.isNonNullType)(newType) && isChangeSafeForInputObjectFieldOrFieldArg(
            oldType.ofType,
            newType.ofType
          ) || // moving from non-null to nullable of the same underlying type is safe
          !(0, _definition.isNonNullType)(newType) && isChangeSafeForInputObjectFieldOrFieldArg(oldType.ofType, newType)
        );
      }
      return (0, _definition.isNamedType)(newType) && oldType.name === newType.name;
    }
    __name(isChangeSafeForInputObjectFieldOrFieldArg, "isChangeSafeForInputObjectFieldOrFieldArg");
    function typeKindName(type) {
      if ((0, _definition.isScalarType)(type)) {
        return "a Scalar type";
      }
      if ((0, _definition.isObjectType)(type)) {
        return "an Object type";
      }
      if ((0, _definition.isInterfaceType)(type)) {
        return "an Interface type";
      }
      if ((0, _definition.isUnionType)(type)) {
        return "a Union type";
      }
      if ((0, _definition.isEnumType)(type)) {
        return "an Enum type";
      }
      if ((0, _definition.isInputObjectType)(type)) {
        return "an Input type";
      }
      (0, _invariant.invariant)(
        false,
        "Unexpected type: " + (0, _inspect.inspect)(type)
      );
    }
    __name(typeKindName, "typeKindName");
    function stringifyValue(value, type) {
      const ast = (0, _astFromValue.astFromValue)(value, type);
      ast != null || (0, _invariant.invariant)(false);
      return (0, _printer.print)((0, _sortValueNode.sortValueNode)(ast));
    }
    __name(stringifyValue, "stringifyValue");
    function diff(oldArray, newArray) {
      const added = [];
      const removed = [];
      const persisted = [];
      const oldMap = (0, _keyMap.keyMap)(oldArray, ({ name }) => name);
      const newMap = (0, _keyMap.keyMap)(newArray, ({ name }) => name);
      for (const oldItem of oldArray) {
        const newItem = newMap[oldItem.name];
        if (newItem === void 0) {
          removed.push(oldItem);
        } else {
          persisted.push([oldItem, newItem]);
        }
      }
      for (const newItem of newArray) {
        if (oldMap[newItem.name] === void 0) {
          added.push(newItem);
        }
      }
      return {
        added,
        persisted,
        removed
      };
    }
    __name(diff, "diff");
  }
});

// ../../node_modules/graphql/utilities/index.js
var require_utilities = __commonJS({
  "../../node_modules/graphql/utilities/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "BreakingChangeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _findBreakingChanges.BreakingChangeType;
      }, "get")
    });
    Object.defineProperty(exports, "DangerousChangeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _findBreakingChanges.DangerousChangeType;
      }, "get")
    });
    Object.defineProperty(exports, "TypeInfo", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _TypeInfo.TypeInfo;
      }, "get")
    });
    Object.defineProperty(exports, "assertValidName", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _assertValidName.assertValidName;
      }, "get")
    });
    Object.defineProperty(exports, "astFromValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _astFromValue.astFromValue;
      }, "get")
    });
    Object.defineProperty(exports, "buildASTSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _buildASTSchema.buildASTSchema;
      }, "get")
    });
    Object.defineProperty(exports, "buildClientSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _buildClientSchema.buildClientSchema;
      }, "get")
    });
    Object.defineProperty(exports, "buildSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _buildASTSchema.buildSchema;
      }, "get")
    });
    Object.defineProperty(exports, "coerceInputValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _coerceInputValue.coerceInputValue;
      }, "get")
    });
    Object.defineProperty(exports, "concatAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _concatAST.concatAST;
      }, "get")
    });
    Object.defineProperty(exports, "doTypesOverlap", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _typeComparators.doTypesOverlap;
      }, "get")
    });
    Object.defineProperty(exports, "extendSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _extendSchema.extendSchema;
      }, "get")
    });
    Object.defineProperty(exports, "findBreakingChanges", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _findBreakingChanges.findBreakingChanges;
      }, "get")
    });
    Object.defineProperty(exports, "findDangerousChanges", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _findBreakingChanges.findDangerousChanges;
      }, "get")
    });
    Object.defineProperty(exports, "getIntrospectionQuery", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _getIntrospectionQuery.getIntrospectionQuery;
      }, "get")
    });
    Object.defineProperty(exports, "getOperationAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _getOperationAST.getOperationAST;
      }, "get")
    });
    Object.defineProperty(exports, "getOperationRootType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _getOperationRootType.getOperationRootType;
      }, "get")
    });
    Object.defineProperty(exports, "introspectionFromSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _introspectionFromSchema.introspectionFromSchema;
      }, "get")
    });
    Object.defineProperty(exports, "isEqualType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _typeComparators.isEqualType;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeSubTypeOf", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _typeComparators.isTypeSubTypeOf;
      }, "get")
    });
    Object.defineProperty(exports, "isValidNameError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _assertValidName.isValidNameError;
      }, "get")
    });
    Object.defineProperty(exports, "lexicographicSortSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _lexicographicSortSchema.lexicographicSortSchema;
      }, "get")
    });
    Object.defineProperty(exports, "printIntrospectionSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _printSchema.printIntrospectionSchema;
      }, "get")
    });
    Object.defineProperty(exports, "printSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _printSchema.printSchema;
      }, "get")
    });
    Object.defineProperty(exports, "printType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _printSchema.printType;
      }, "get")
    });
    Object.defineProperty(exports, "separateOperations", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _separateOperations.separateOperations;
      }, "get")
    });
    Object.defineProperty(exports, "stripIgnoredCharacters", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _stripIgnoredCharacters.stripIgnoredCharacters;
      }, "get")
    });
    Object.defineProperty(exports, "typeFromAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _typeFromAST.typeFromAST;
      }, "get")
    });
    Object.defineProperty(exports, "valueFromAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _valueFromAST.valueFromAST;
      }, "get")
    });
    Object.defineProperty(exports, "valueFromASTUntyped", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _valueFromASTUntyped.valueFromASTUntyped;
      }, "get")
    });
    Object.defineProperty(exports, "visitWithTypeInfo", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _TypeInfo.visitWithTypeInfo;
      }, "get")
    });
    var _getIntrospectionQuery = require_getIntrospectionQuery();
    var _getOperationAST = require_getOperationAST();
    var _getOperationRootType = require_getOperationRootType();
    var _introspectionFromSchema = require_introspectionFromSchema();
    var _buildClientSchema = require_buildClientSchema();
    var _buildASTSchema = require_buildASTSchema();
    var _extendSchema = require_extendSchema();
    var _lexicographicSortSchema = require_lexicographicSortSchema();
    var _printSchema = require_printSchema();
    var _typeFromAST = require_typeFromAST();
    var _valueFromAST = require_valueFromAST();
    var _valueFromASTUntyped = require_valueFromASTUntyped();
    var _astFromValue = require_astFromValue();
    var _TypeInfo = require_TypeInfo();
    var _coerceInputValue = require_coerceInputValue();
    var _concatAST = require_concatAST();
    var _separateOperations = require_separateOperations();
    var _stripIgnoredCharacters = require_stripIgnoredCharacters();
    var _typeComparators = require_typeComparators();
    var _assertValidName = require_assertValidName();
    var _findBreakingChanges = require_findBreakingChanges();
  }
});

// ../../node_modules/graphql/index.js
var require_graphql2 = __commonJS({
  "../../node_modules/graphql/index.js"(exports) {
    "use strict";
    Object.defineProperty(exports, "__esModule", {
      value: true
    });
    Object.defineProperty(exports, "BREAK", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.BREAK;
      }, "get")
    });
    Object.defineProperty(exports, "BreakingChangeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.BreakingChangeType;
      }, "get")
    });
    Object.defineProperty(exports, "DEFAULT_DEPRECATION_REASON", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.DEFAULT_DEPRECATION_REASON;
      }, "get")
    });
    Object.defineProperty(exports, "DangerousChangeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.DangerousChangeType;
      }, "get")
    });
    Object.defineProperty(exports, "DirectiveLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.DirectiveLocation;
      }, "get")
    });
    Object.defineProperty(exports, "ExecutableDefinitionsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.ExecutableDefinitionsRule;
      }, "get")
    });
    Object.defineProperty(exports, "FieldsOnCorrectTypeRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.FieldsOnCorrectTypeRule;
      }, "get")
    });
    Object.defineProperty(exports, "FragmentsOnCompositeTypesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.FragmentsOnCompositeTypesRule;
      }, "get")
    });
    Object.defineProperty(exports, "GRAPHQL_MAX_INT", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GRAPHQL_MAX_INT;
      }, "get")
    });
    Object.defineProperty(exports, "GRAPHQL_MIN_INT", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GRAPHQL_MIN_INT;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLBoolean", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLBoolean;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLDeprecatedDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLDeprecatedDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLEnumType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLEnumType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index5.GraphQLError;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLFloat", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLFloat;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLID", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLID;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLIncludeDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLIncludeDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLInputObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLInputObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLInt", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLInt;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLInterfaceType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLInterfaceType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLList", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLList;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLNonNull", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLNonNull;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLOneOfDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLOneOfDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLSchema;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLSkipDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLSkipDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLSpecifiedByDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLSpecifiedByDirective;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLString", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLString;
      }, "get")
    });
    Object.defineProperty(exports, "GraphQLUnionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.GraphQLUnionType;
      }, "get")
    });
    Object.defineProperty(exports, "Kind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.Kind;
      }, "get")
    });
    Object.defineProperty(exports, "KnownArgumentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.KnownArgumentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownDirectivesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.KnownDirectivesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownFragmentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.KnownFragmentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "KnownTypeNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.KnownTypeNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "Lexer", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.Lexer;
      }, "get")
    });
    Object.defineProperty(exports, "Location", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.Location;
      }, "get")
    });
    Object.defineProperty(exports, "LoneAnonymousOperationRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.LoneAnonymousOperationRule;
      }, "get")
    });
    Object.defineProperty(exports, "LoneSchemaDefinitionRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.LoneSchemaDefinitionRule;
      }, "get")
    });
    Object.defineProperty(exports, "MaxIntrospectionDepthRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.MaxIntrospectionDepthRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoDeprecatedCustomRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.NoDeprecatedCustomRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoFragmentCyclesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.NoFragmentCyclesRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoSchemaIntrospectionCustomRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.NoSchemaIntrospectionCustomRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoUndefinedVariablesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.NoUndefinedVariablesRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoUnusedFragmentsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.NoUnusedFragmentsRule;
      }, "get")
    });
    Object.defineProperty(exports, "NoUnusedVariablesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.NoUnusedVariablesRule;
      }, "get")
    });
    Object.defineProperty(exports, "OperationTypeNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.OperationTypeNode;
      }, "get")
    });
    Object.defineProperty(exports, "OverlappingFieldsCanBeMergedRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.OverlappingFieldsCanBeMergedRule;
      }, "get")
    });
    Object.defineProperty(exports, "PossibleFragmentSpreadsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.PossibleFragmentSpreadsRule;
      }, "get")
    });
    Object.defineProperty(exports, "PossibleTypeExtensionsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.PossibleTypeExtensionsRule;
      }, "get")
    });
    Object.defineProperty(exports, "ProvidedRequiredArgumentsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.ProvidedRequiredArgumentsRule;
      }, "get")
    });
    Object.defineProperty(exports, "ScalarLeafsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.ScalarLeafsRule;
      }, "get")
    });
    Object.defineProperty(exports, "SchemaMetaFieldDef", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.SchemaMetaFieldDef;
      }, "get")
    });
    Object.defineProperty(exports, "SingleFieldSubscriptionsRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.SingleFieldSubscriptionsRule;
      }, "get")
    });
    Object.defineProperty(exports, "Source", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.Source;
      }, "get")
    });
    Object.defineProperty(exports, "Token", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.Token;
      }, "get")
    });
    Object.defineProperty(exports, "TokenKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.TokenKind;
      }, "get")
    });
    Object.defineProperty(exports, "TypeInfo", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.TypeInfo;
      }, "get")
    });
    Object.defineProperty(exports, "TypeKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.TypeKind;
      }, "get")
    });
    Object.defineProperty(exports, "TypeMetaFieldDef", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.TypeMetaFieldDef;
      }, "get")
    });
    Object.defineProperty(exports, "TypeNameMetaFieldDef", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.TypeNameMetaFieldDef;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueArgumentDefinitionNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueArgumentDefinitionNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueArgumentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueArgumentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueDirectiveNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueDirectiveNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueDirectivesPerLocationRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueDirectivesPerLocationRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueEnumValueNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueEnumValueNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueFieldDefinitionNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueFieldDefinitionNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueFragmentNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueFragmentNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueInputFieldNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueInputFieldNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueOperationNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueOperationNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueOperationTypesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueOperationTypesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueTypeNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueTypeNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "UniqueVariableNamesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.UniqueVariableNamesRule;
      }, "get")
    });
    Object.defineProperty(exports, "ValidationContext", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.ValidationContext;
      }, "get")
    });
    Object.defineProperty(exports, "ValuesOfCorrectTypeRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.ValuesOfCorrectTypeRule;
      }, "get")
    });
    Object.defineProperty(exports, "VariablesAreInputTypesRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.VariablesAreInputTypesRule;
      }, "get")
    });
    Object.defineProperty(exports, "VariablesInAllowedPositionRule", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.VariablesInAllowedPositionRule;
      }, "get")
    });
    Object.defineProperty(exports, "__Directive", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__Directive;
      }, "get")
    });
    Object.defineProperty(exports, "__DirectiveLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__DirectiveLocation;
      }, "get")
    });
    Object.defineProperty(exports, "__EnumValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__EnumValue;
      }, "get")
    });
    Object.defineProperty(exports, "__Field", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__Field;
      }, "get")
    });
    Object.defineProperty(exports, "__InputValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__InputValue;
      }, "get")
    });
    Object.defineProperty(exports, "__Schema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__Schema;
      }, "get")
    });
    Object.defineProperty(exports, "__Type", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__Type;
      }, "get")
    });
    Object.defineProperty(exports, "__TypeKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.__TypeKind;
      }, "get")
    });
    Object.defineProperty(exports, "assertAbstractType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertAbstractType;
      }, "get")
    });
    Object.defineProperty(exports, "assertCompositeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertCompositeType;
      }, "get")
    });
    Object.defineProperty(exports, "assertDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertDirective;
      }, "get")
    });
    Object.defineProperty(exports, "assertEnumType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertEnumType;
      }, "get")
    });
    Object.defineProperty(exports, "assertEnumValueName", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertEnumValueName;
      }, "get")
    });
    Object.defineProperty(exports, "assertInputObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertInputObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "assertInputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertInputType;
      }, "get")
    });
    Object.defineProperty(exports, "assertInterfaceType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertInterfaceType;
      }, "get")
    });
    Object.defineProperty(exports, "assertLeafType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertLeafType;
      }, "get")
    });
    Object.defineProperty(exports, "assertListType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertListType;
      }, "get")
    });
    Object.defineProperty(exports, "assertName", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertName;
      }, "get")
    });
    Object.defineProperty(exports, "assertNamedType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertNamedType;
      }, "get")
    });
    Object.defineProperty(exports, "assertNonNullType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertNonNullType;
      }, "get")
    });
    Object.defineProperty(exports, "assertNullableType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertNullableType;
      }, "get")
    });
    Object.defineProperty(exports, "assertObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "assertOutputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertOutputType;
      }, "get")
    });
    Object.defineProperty(exports, "assertScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "assertSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertSchema;
      }, "get")
    });
    Object.defineProperty(exports, "assertType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertType;
      }, "get")
    });
    Object.defineProperty(exports, "assertUnionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertUnionType;
      }, "get")
    });
    Object.defineProperty(exports, "assertValidName", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.assertValidName;
      }, "get")
    });
    Object.defineProperty(exports, "assertValidSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertValidSchema;
      }, "get")
    });
    Object.defineProperty(exports, "assertWrappingType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.assertWrappingType;
      }, "get")
    });
    Object.defineProperty(exports, "astFromValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.astFromValue;
      }, "get")
    });
    Object.defineProperty(exports, "buildASTSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.buildASTSchema;
      }, "get")
    });
    Object.defineProperty(exports, "buildClientSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.buildClientSchema;
      }, "get")
    });
    Object.defineProperty(exports, "buildSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.buildSchema;
      }, "get")
    });
    Object.defineProperty(exports, "coerceInputValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.coerceInputValue;
      }, "get")
    });
    Object.defineProperty(exports, "concatAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.concatAST;
      }, "get")
    });
    Object.defineProperty(exports, "createSourceEventStream", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.createSourceEventStream;
      }, "get")
    });
    Object.defineProperty(exports, "defaultFieldResolver", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.defaultFieldResolver;
      }, "get")
    });
    Object.defineProperty(exports, "defaultTypeResolver", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.defaultTypeResolver;
      }, "get")
    });
    Object.defineProperty(exports, "doTypesOverlap", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.doTypesOverlap;
      }, "get")
    });
    Object.defineProperty(exports, "execute", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.execute;
      }, "get")
    });
    Object.defineProperty(exports, "executeSync", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.executeSync;
      }, "get")
    });
    Object.defineProperty(exports, "extendSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.extendSchema;
      }, "get")
    });
    Object.defineProperty(exports, "findBreakingChanges", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.findBreakingChanges;
      }, "get")
    });
    Object.defineProperty(exports, "findDangerousChanges", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.findDangerousChanges;
      }, "get")
    });
    Object.defineProperty(exports, "formatError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index5.formatError;
      }, "get")
    });
    Object.defineProperty(exports, "getArgumentValues", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.getArgumentValues;
      }, "get")
    });
    Object.defineProperty(exports, "getDirectiveValues", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.getDirectiveValues;
      }, "get")
    });
    Object.defineProperty(exports, "getEnterLeaveForKind", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.getEnterLeaveForKind;
      }, "get")
    });
    Object.defineProperty(exports, "getIntrospectionQuery", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.getIntrospectionQuery;
      }, "get")
    });
    Object.defineProperty(exports, "getLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.getLocation;
      }, "get")
    });
    Object.defineProperty(exports, "getNamedType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.getNamedType;
      }, "get")
    });
    Object.defineProperty(exports, "getNullableType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.getNullableType;
      }, "get")
    });
    Object.defineProperty(exports, "getOperationAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.getOperationAST;
      }, "get")
    });
    Object.defineProperty(exports, "getOperationRootType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.getOperationRootType;
      }, "get")
    });
    Object.defineProperty(exports, "getVariableValues", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.getVariableValues;
      }, "get")
    });
    Object.defineProperty(exports, "getVisitFn", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.getVisitFn;
      }, "get")
    });
    Object.defineProperty(exports, "graphql", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _graphql.graphql;
      }, "get")
    });
    Object.defineProperty(exports, "graphqlSync", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _graphql.graphqlSync;
      }, "get")
    });
    Object.defineProperty(exports, "introspectionFromSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.introspectionFromSchema;
      }, "get")
    });
    Object.defineProperty(exports, "introspectionTypes", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.introspectionTypes;
      }, "get")
    });
    Object.defineProperty(exports, "isAbstractType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isAbstractType;
      }, "get")
    });
    Object.defineProperty(exports, "isCompositeType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isCompositeType;
      }, "get")
    });
    Object.defineProperty(exports, "isConstValueNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isConstValueNode;
      }, "get")
    });
    Object.defineProperty(exports, "isDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isDirective;
      }, "get")
    });
    Object.defineProperty(exports, "isEnumType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isEnumType;
      }, "get")
    });
    Object.defineProperty(exports, "isEqualType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.isEqualType;
      }, "get")
    });
    Object.defineProperty(exports, "isExecutableDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isExecutableDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isInputObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isInputObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "isInputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isInputType;
      }, "get")
    });
    Object.defineProperty(exports, "isInterfaceType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isInterfaceType;
      }, "get")
    });
    Object.defineProperty(exports, "isIntrospectionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isIntrospectionType;
      }, "get")
    });
    Object.defineProperty(exports, "isLeafType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isLeafType;
      }, "get")
    });
    Object.defineProperty(exports, "isListType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isListType;
      }, "get")
    });
    Object.defineProperty(exports, "isNamedType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isNamedType;
      }, "get")
    });
    Object.defineProperty(exports, "isNonNullType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isNonNullType;
      }, "get")
    });
    Object.defineProperty(exports, "isNullableType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isNullableType;
      }, "get")
    });
    Object.defineProperty(exports, "isObjectType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isObjectType;
      }, "get")
    });
    Object.defineProperty(exports, "isOutputType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isOutputType;
      }, "get")
    });
    Object.defineProperty(exports, "isRequiredArgument", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isRequiredArgument;
      }, "get")
    });
    Object.defineProperty(exports, "isRequiredInputField", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isRequiredInputField;
      }, "get")
    });
    Object.defineProperty(exports, "isScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "isSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isSchema;
      }, "get")
    });
    Object.defineProperty(exports, "isSelectionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isSelectionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isSpecifiedDirective", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isSpecifiedDirective;
      }, "get")
    });
    Object.defineProperty(exports, "isSpecifiedScalarType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isSpecifiedScalarType;
      }, "get")
    });
    Object.defineProperty(exports, "isType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isType;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isTypeDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeExtensionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isTypeExtensionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isTypeNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeSubTypeOf", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.isTypeSubTypeOf;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeSystemDefinitionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isTypeSystemDefinitionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isTypeSystemExtensionNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isTypeSystemExtensionNode;
      }, "get")
    });
    Object.defineProperty(exports, "isUnionType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isUnionType;
      }, "get")
    });
    Object.defineProperty(exports, "isValidNameError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.isValidNameError;
      }, "get")
    });
    Object.defineProperty(exports, "isValueNode", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.isValueNode;
      }, "get")
    });
    Object.defineProperty(exports, "isWrappingType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.isWrappingType;
      }, "get")
    });
    Object.defineProperty(exports, "lexicographicSortSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.lexicographicSortSchema;
      }, "get")
    });
    Object.defineProperty(exports, "locatedError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index5.locatedError;
      }, "get")
    });
    Object.defineProperty(exports, "parse", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.parse;
      }, "get")
    });
    Object.defineProperty(exports, "parseConstValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.parseConstValue;
      }, "get")
    });
    Object.defineProperty(exports, "parseType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.parseType;
      }, "get")
    });
    Object.defineProperty(exports, "parseValue", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.parseValue;
      }, "get")
    });
    Object.defineProperty(exports, "print", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.print;
      }, "get")
    });
    Object.defineProperty(exports, "printError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index5.printError;
      }, "get")
    });
    Object.defineProperty(exports, "printIntrospectionSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.printIntrospectionSchema;
      }, "get")
    });
    Object.defineProperty(exports, "printLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.printLocation;
      }, "get")
    });
    Object.defineProperty(exports, "printSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.printSchema;
      }, "get")
    });
    Object.defineProperty(exports, "printSourceLocation", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.printSourceLocation;
      }, "get")
    });
    Object.defineProperty(exports, "printType", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.printType;
      }, "get")
    });
    Object.defineProperty(exports, "recommendedRules", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.recommendedRules;
      }, "get")
    });
    Object.defineProperty(exports, "resolveObjMapThunk", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.resolveObjMapThunk;
      }, "get")
    });
    Object.defineProperty(exports, "resolveReadonlyArrayThunk", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.resolveReadonlyArrayThunk;
      }, "get")
    });
    Object.defineProperty(exports, "responsePathAsArray", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.responsePathAsArray;
      }, "get")
    });
    Object.defineProperty(exports, "separateOperations", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.separateOperations;
      }, "get")
    });
    Object.defineProperty(exports, "specifiedDirectives", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.specifiedDirectives;
      }, "get")
    });
    Object.defineProperty(exports, "specifiedRules", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.specifiedRules;
      }, "get")
    });
    Object.defineProperty(exports, "specifiedScalarTypes", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.specifiedScalarTypes;
      }, "get")
    });
    Object.defineProperty(exports, "stripIgnoredCharacters", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.stripIgnoredCharacters;
      }, "get")
    });
    Object.defineProperty(exports, "subscribe", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index3.subscribe;
      }, "get")
    });
    Object.defineProperty(exports, "syntaxError", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index5.syntaxError;
      }, "get")
    });
    Object.defineProperty(exports, "typeFromAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.typeFromAST;
      }, "get")
    });
    Object.defineProperty(exports, "validate", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index4.validate;
      }, "get")
    });
    Object.defineProperty(exports, "validateSchema", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index.validateSchema;
      }, "get")
    });
    Object.defineProperty(exports, "valueFromAST", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.valueFromAST;
      }, "get")
    });
    Object.defineProperty(exports, "valueFromASTUntyped", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.valueFromASTUntyped;
      }, "get")
    });
    Object.defineProperty(exports, "version", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _version.version;
      }, "get")
    });
    Object.defineProperty(exports, "versionInfo", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _version.versionInfo;
      }, "get")
    });
    Object.defineProperty(exports, "visit", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.visit;
      }, "get")
    });
    Object.defineProperty(exports, "visitInParallel", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index2.visitInParallel;
      }, "get")
    });
    Object.defineProperty(exports, "visitWithTypeInfo", {
      enumerable: true,
      get: /* @__PURE__ */ __name(function() {
        return _index6.visitWithTypeInfo;
      }, "get")
    });
    var _version = require_version();
    var _graphql = require_graphql();
    var _index = require_type();
    var _index2 = require_language();
    var _index3 = require_execution();
    var _index4 = require_validation();
    var _index5 = require_error();
    var _index6 = require_utilities();
  }
});

// src/cli.ts
var import_graphql = __toESM(require_graphql2(), 1);
import crypto from "node:crypto";
import { spawnSync, spawn } from "node:child_process";
import fs from "node:fs";
import net from "node:net";
import os from "node:os";
import path from "node:path";

// src/generated/graphql.ts
var CreateApiKeyDocument = { "kind": "Document", "definitions": [{ "kind": "OperationDefinition", "operation": "mutation", "name": { "kind": "Name", "value": "CreateApiKey" }, "variableDefinitions": [{ "kind": "VariableDefinition", "variable": { "kind": "Variable", "name": { "kind": "Name", "value": "description" } }, "type": { "kind": "NonNullType", "type": { "kind": "NamedType", "name": { "kind": "Name", "value": "String" } } } }], "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "admin" }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "createApiKey" }, "arguments": [{ "kind": "Argument", "name": { "kind": "Name", "value": "input" }, "value": { "kind": "ObjectValue", "fields": [{ "kind": "ObjectField", "name": { "kind": "Name", "value": "description" }, "value": { "kind": "Variable", "name": { "kind": "Name", "value": "description" } } }] } }], "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "__typename" } }, { "kind": "InlineFragment", "typeCondition": { "kind": "NamedType", "name": { "kind": "Name", "value": "CreateApiKeyPayload" } }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "value" } }, { "kind": "Field", "name": { "kind": "Name", "value": "apiKey" }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "id" } }] } }] } }, { "kind": "InlineFragment", "typeCondition": { "kind": "NamedType", "name": { "kind": "Name", "value": "ErrorPayload" } }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "message" } }] } }] } }] } }] } }] };
var DisableApiKeyDocument = { "kind": "Document", "definitions": [{ "kind": "OperationDefinition", "operation": "mutation", "name": { "kind": "Name", "value": "DisableApiKey" }, "variableDefinitions": [{ "kind": "VariableDefinition", "variable": { "kind": "Variable", "name": { "kind": "Name", "value": "id" } }, "type": { "kind": "NonNullType", "type": { "kind": "NamedType", "name": { "kind": "Name", "value": "Int64" } } } }], "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "admin" }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "disableApiKey" }, "arguments": [{ "kind": "Argument", "name": { "kind": "Name", "value": "input" }, "value": { "kind": "ObjectValue", "fields": [{ "kind": "ObjectField", "name": { "kind": "Name", "value": "id" }, "value": { "kind": "Variable", "name": { "kind": "Name", "value": "id" } } }] } }], "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "__typename" } }, { "kind": "InlineFragment", "typeCondition": { "kind": "NamedType", "name": { "kind": "Name", "value": "DisableApiKeyPayload" } }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "apiKey" }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "id" } }] } }] } }, { "kind": "InlineFragment", "typeCondition": { "kind": "NamedType", "name": { "kind": "Name", "value": "ErrorPayload" } }, "selectionSet": { "kind": "SelectionSet", "selections": [{ "kind": "Field", "name": { "kind": "Name", "value": "message" } }] } }] } }] } }] } }] };

// src/cli.ts
var CONTAINER_NAME = "trip2g-memory";
function containerName(flags) {
  if (!flags || !flags.name) return CONTAINER_NAME;
  const safe = flags.name.replace(/[^a-zA-Z0-9_.-]/g, "-");
  return `${CONTAINER_NAME}-${safe}`;
}
__name(containerName, "containerName");
var DEFAULT_PORT = 24081;
var DEFAULT_IMAGE = "ghcr.io/trip2g/trip2g:latest";
var DEFAULT_EMAIL = "memory@local";
var READY_TIMEOUT_MS = 6e4;
var READY_POLL_MS = 500;
var EMBED_SERVER_IMAGE = "trip2g-embedding-server";
var EMBED_READY_TIMEOUT_MS = 9e5;
var EMBED_READY_POLL_MS = 2e3;
function networkName(name) {
  return `${name}-net`;
}
__name(networkName, "networkName");
function embeddingContainerName(name) {
  return `${name}-embedding`;
}
__name(embeddingContainerName, "embeddingContainerName");
var DEFAULT_HUB_URL = "https://trip2g.com/_system/mcp";
var DEFAULT_STALE_DAYS = 60;
function base64url(buf) {
  const b = Buffer.isBuffer(buf) ? buf : Buffer.from(buf);
  return b.toString("base64url");
}
__name(base64url, "base64url");
function signHatJwt(secret, email) {
  const headerB64 = base64url(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = {
    e: email,
    ae: true,
    exp: Math.floor(Date.now() / 1e3) + 300
  };
  const payloadB64 = base64url(JSON.stringify(payload));
  const signingInput = `${headerB64}.${payloadB64}`;
  const sig = crypto.createHmac("sha256", secret).update(signingInput).digest();
  return `${signingInput}.${base64url(sig)}`;
}
__name(signHatJwt, "signHatJwt");
function buildHatLoginUrl(publicUrl, jwt) {
  return `${publicUrl}/?hat=${encodeURIComponent(jwt)}`;
}
__name(buildHatLoginUrl, "buildHatLoginUrl");
function hubSlug(hubUrl) {
  const host = new URL(hubUrl).hostname;
  return host.replace(/[^a-zA-Z0-9._-]/g, "-");
}
__name(hubSlug, "hubSlug");
function buildHubNote(hubUrl, id) {
  const host = new URL(hubUrl).hostname;
  const kbId = id ?? host;
  return `---
free: true
mcp_federation_kb_url: ${hubUrl}
mcp_federation_kb_id: ${kbId}
---

# Your trip2g memory

**trip2g** turns a folder of Markdown notes into living memory: you write notes, an agent reads and searches them as persistent long-term memory, and the same notes can be published as a website with subscriptions and a Telegram channel. It is a self-hosted Go + SQLite app.

## What you can do here
- **Remember** \u2014 write Markdown notes into this vault (\`memcli daily "\u2026"\`, \`memcli log <topic> "\u2026"\`, or just create \`.md\` files).
- **Recall** \u2014 grep/read the vault, or query this instance's \`/_system/mcp\` tools (\`search\`, \`expand\`).
- **Search the wider trip2g knowledge base** \u2014 this memory federates to **${host}**. Use the **\`federated_search\`** MCP tool to search ${host} and pull matching docs back, without leaving your own memory.

## Docs
- Long-term memory for agents \u2014 https://trip2g.com/en/user/agent_memory
- Local MCP server & tools \u2014 https://trip2g.com/en/user/mcp
- Federation \u2014 https://trip2g.com/en/user/federation
- How trip2g works \u2014 https://trip2g.com/en/user/protocol
- Self-hosting \u2014 https://trip2g.com/en/user/selfhosted

To disable federation: delete this file or run \`memcli up\` with \`--no-hub\`.
`.trimStart();
}
__name(buildHubNote, "buildHubNote");
function buildIndexNote() {
  return `---
type: index
---

# Index

This is the entry point for your trip2g memory base \u2014 a folder of Markdown notes
an agent reads and searches as persistent long-term memory.

## What this base covers
- Captured thoughts (daily notes, per-topic logs).
- Concepts, sources, decisions, and queries you accumulate over time.
- A living synthesis of open questions and answers.

## Map
- [[AGENTS]] \u2014 the retrieval workflow and the commit-gate rule.
- [[SCHEMA]] \u2014 page types and the OKF \`type:\` requirement.
- [[log]] \u2014 running synthesis and open questions.
`.trimStart();
}
__name(buildIndexNote, "buildIndexNote");
function buildLogNote() {
  return `---
type: log
---

# Log

A running synthesis of what this memory base knows, and the open questions still
to resolve. Append as understanding evolves; promote settled answers into their
own typed notes (concept / source / decision).

## Synthesis
_(running summary \u2014 keep this current)_

## Open questions
- _(none yet \u2014 add questions here as they surface)_
`.trimStart();
}
__name(buildLogNote, "buildLogNote");
function buildAgentsNote() {
  return `---
mcp_method: instructions
type: instructions
---

# Agent instructions

## Retrieval workflow
1. **search** \u2014 query the base for notes relevant to the task.
2. **expand** \u2014 pull the full body of the most promising matches.
3. **note_html** \u2014 render a note when you need its formatted content.

Prefer recalling existing notes over re-deriving knowledge. Link related notes
with \`[[wikilinks]]\` so the graph stays navigable.

## Commit-gate rule
Before treating a write as done, run \`memcli lint\` (or the \`memory_lint\` MCP
tool) and resolve every \`Status: Unresolved\` marker. Do not leave contradictions
or unresolved markers behind \u2014 an unresolved marker means the write is not yet
committed.
`.trimStart();
}
__name(buildAgentsNote, "buildAgentsNote");
function buildSchemaNote() {
  return `---
mcp_method: schema
type: schema
---

# Schema

This base follows the Open Knowledge Format (OKF): **every note carries a
\`type:\` field** in its frontmatter so it can be classified and validated.

## Page types
- **concept** \u2014 a durable idea, definition, or model.
- **source** \u2014 an external reference (article, paper, repo, conversation).
- **decision** \u2014 a choice made, with its rationale and alternatives.
- **query** \u2014 an open question being investigated.
- **daily** \u2014 a dated working-space note of captured thoughts.

## Rule
Every note MUST declare a \`type:\` field. Notes without one fail OKF validation
(see \`memcli lint\` / \`memory_lint\`).
`.trimStart();
}
__name(buildSchemaNote, "buildSchemaNote");
function buildHomeNote() {
  return `---
type: index
---

# Memory

An Open Knowledge Format (OKF) / LLM-wiki base served over MCP as an agent's
persistent memory \u2014 a folder of Markdown notes an agent reads, searches, and
writes back as it works.

- [[index|What this base covers]]
- [[log|Activity log]]
- [[AGENTS|Agent instructions]]
`.trimStart();
}
__name(buildHomeNote, "buildHomeNote");
function buildHeaderNote() {
  return `---
type: header
---

- [[_index|Home]]
- [[index|Index]]
- [[log|Log]]
`.trimStart();
}
__name(buildHeaderNote, "buildHeaderNote");
var KANBAN_SENTINEL = "<!-- memcli:kanban-template -->";
var KANBAN_RELEASE_URL = "https://github.com/trip2g/kanban_template/releases/latest/download/kanban.html";
function buildKanbanBoard() {
  return `---
kanban-plugin: basic
layout: kanban
free: true
---

## Backlog

- [ ] Read the trip2g docs
- [ ] Plan your first workflow

## Doing

- [ ] Try memcli daily "first thought"

## Done

- [ ] Install the kanban template
`.trimStart();
}
__name(buildKanbanBoard, "buildKanbanBoard");
async function installKanban(vault, kanbanBundle, noSeed, dryRun, fetchFn = fetch) {
  const layoutsDir = path.join(vault, "_layouts");
  const layoutFile = path.join(layoutsDir, "kanban.html");
  const boardFile = path.join(vault, "kanban.md");
  const existingLayout = fs.existsSync(layoutFile) ? fs.readFileSync(layoutFile, "utf8") : null;
  const isOurs = existingLayout !== null && existingLayout.startsWith(KANBAN_SENTINEL);
  const shouldWriteLayout = existingLayout === null || isOurs;
  if (shouldWriteLayout) {
    if (dryRun) {
      console.log(`[dry-run] would fetch ${KANBAN_RELEASE_URL} \u2192 ${layoutFile}`);
      if (kanbanBundle) {
        console.log(`[dry-run] would rewrite bundle <script src> to ${kanbanBundle}`);
      }
    } else if (existingLayout === null) {
      const res = await fetchFn(KANBAN_RELEASE_URL);
      if (!res.ok) {
        throw new Error(`Failed to fetch kanban template: ${res.status} ${res.statusText}`);
      }
      let html = KANBAN_SENTINEL + "\n" + await res.text();
      if (kanbanBundle) {
        html = html.replace(/<script src="[^"]*kanban\.js"[^>]*>/g, `<script src="${kanbanBundle}">`);
      }
      fs.mkdirSync(layoutsDir, { recursive: true });
      fs.writeFileSync(layoutFile, html, "utf8");
      console.log(`Installed kanban template \u2192 ${layoutFile}`);
    } else {
      if (kanbanBundle) {
        const updated = existingLayout.replace(
          /<script src="[^"]*kanban\.js"[^>]*>/g,
          `<script src="${kanbanBundle}">`
        );
        fs.writeFileSync(layoutFile, updated, "utf8");
        console.log(`Updated kanban bundle src \u2192 ${kanbanBundle} in ${layoutFile}`);
      } else {
        console.log(`kanban template already installed at ${layoutFile}, skipping.`);
      }
    }
  } else {
    console.log(`${layoutFile} exists (not ours \u2014 skipping to avoid clobber).`);
  }
  if (!noSeed) {
    if (dryRun) {
      console.log(`[dry-run] would write kanban board ${boardFile}`);
    } else if (fs.existsSync(boardFile)) {
      console.log(`kanban board already present at ${boardFile}, skipping.`);
    } else {
      fs.writeFileSync(boardFile, buildKanbanBoard(), "utf8");
      console.log(`Wrote sample kanban board \u2192 ${boardFile}`);
    }
  }
}
__name(installKanban, "installKanban");
function parseArgs(argv) {
  const SUBCOMMANDS = /* @__PURE__ */ new Set(["up", "down", "status", "logs", "key", "daily", "log", "mcp", "hub", "lint", "open"]);
  const flags = {
    dryRun: false,
    help: false,
    folder: "./memory-vault",
    port: DEFAULT_PORT,
    host: "127.0.0.1",
    email: DEFAULT_EMAIL,
    image: DEFAULT_IMAGE,
    publicUrl: null,
    noHub: false,
    noSeed: false,
    hubUrl: DEFAULT_HUB_URL,
    context: 15,
    staleDays: DEFAULT_STALE_DAYS,
    id: null,
    name: null,
    kanban: false,
    kanbanBundle: null,
    embedded: false,
    reranker: false
  };
  let cmd = "up";
  let i = 0;
  const positional = [];
  if (argv.length > 0 && !argv[0].startsWith("-")) {
    if (SUBCOMMANDS.has(argv[0])) {
      cmd = argv[0];
      i = 1;
    }
  }
  while (i < argv.length) {
    const arg = argv[i];
    if (arg === "--dry-run") {
      flags.dryRun = true;
    } else if (arg === "--help" || arg === "-h") {
      flags.help = true;
    } else if (arg === "--folder") {
      flags.folder = argv[++i];
    } else if (arg === "--port") {
      flags.port = parseInt(argv[++i], 10);
    } else if (arg === "--host") {
      const h = argv[++i];
      if (!h.startsWith("127.")) {
        throw new Error("--host must be a loopback address (127.x.x.x)");
      }
      flags.host = h;
    } else if (arg === "--email") {
      flags.email = argv[++i];
    } else if (arg === "--image") {
      flags.image = argv[++i];
    } else if (arg === "--public-url") {
      flags.publicUrl = argv[++i];
    } else if (arg === "--no-hub") {
      flags.noHub = true;
    } else if (arg === "--no-seed") {
      flags.noSeed = true;
    } else if (arg === "--hub-url") {
      flags.hubUrl = argv[++i];
    } else if (arg === "--context") {
      flags.context = parseInt(argv[++i], 10);
    } else if (arg === "--stale-days") {
      flags.staleDays = parseInt(argv[++i], 10);
    } else if (arg === "--id") {
      flags.id = argv[++i];
    } else if (arg === "--name") {
      flags.name = argv[++i];
    } else if (arg === "--kanban") {
      flags.kanban = true;
    } else if (arg === "--kanban-bundle") {
      flags.kanbanBundle = argv[++i];
    } else if (arg === "--embedded") {
      flags.embedded = true;
    } else if (arg === "--reranker") {
      flags.reranker = true;
    } else if (!arg.startsWith("-")) {
      positional.push(arg);
    }
    i++;
  }
  return { cmd, flags, positional };
}
__name(parseArgs, "parseArgs");
function shouldRunMcp(argv, isTty) {
  const KNOWN_CLI_CMDS = /* @__PURE__ */ new Set(["up", "down", "status", "logs", "key", "daily", "log", "hub", "lint", "open"]);
  const first = argv[0];
  if (first !== void 0 && KNOWN_CLI_CMDS.has(first)) return false;
  if (argv.includes("--help") || argv.includes("-h")) return false;
  if (first === "mcp") return true;
  if (!isTty) return true;
  return false;
}
__name(shouldRunMcp, "shouldRunMcp");
function buildToolList() {
  return [
    {
      name: "memory_up",
      description: "Start the local trip2g memory: boots a server (local filesystem storage, no S3 or dev mode), mints an admin key via HAT, starts a two-way sync watcher, and drops a hub note federating to trip2g.com. Idempotent \u2014 safe to call if already running. Call once before remembering/recalling. Arg: folder (vault path, default ./memory-vault).",
      inputSchema: {
        type: "object",
        properties: {
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" },
          port: { type: "number", description: "Public port (default: 24081)" },
          email: { type: "string", description: "Owner email (default: memory@local)" },
          image: { type: "string", description: "Docker image ref (default: ghcr.io/trip2g/trip2g:latest)" },
          noHub: { type: "boolean", description: "Skip writing the federation hub note" },
          noSeed: { type: "boolean", description: "Skip seeding the OKF starter notes (index.md/log.md/AGENTS.md/SCHEMA.md)" },
          hubUrl: { type: "string", description: "Override hub MCP endpoint URL" },
          publicUrl: { type: "string", description: "Override PUBLIC_URL for the server" },
          kanban: { type: "boolean", description: "Install the trip2g kanban template into <vault>/_layouts/kanban.html and seed a sample kanban.md board" },
          kanbanBundle: { type: "string", description: "Override the kanban.js bundle <script src> URL (local-dev override, e.g. http://localhost:8770/kanban.js)" },
          embedded: { type: "boolean", description: "Start the local retriever sidecar (bge-m3, ~2GB, CPU-heavy) and enable vector search" },
          reranker: { type: "boolean", description: "Also start a reranker sidecar (bge-reranker-v2-m3, ~2GB); implies embedded" }
        },
        required: []
      }
    },
    {
      name: "memory_down",
      description: "Stop the memory server container and the sync watcher for a vault. Arg: folder.",
      inputSchema: {
        type: "object",
        properties: {
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" }
        },
        required: []
      }
    },
    {
      name: "memory_status",
      description: "Report whether the memory server and sync watcher are running for a vault. Arg: folder.",
      inputSchema: {
        type: "object",
        properties: {
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" }
        },
        required: []
      }
    },
    {
      name: "memory_logs",
      description: "Show recent server logs (docker logs snapshot) for diagnostics. Arg: folder.",
      inputSchema: {
        type: "object",
        properties: {
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" }
        },
        required: []
      }
    },
    {
      name: "memory_key",
      description: "Rotate the admin API key (mints a new one, disables the previous). Use if the key leaked or you need a fresh one. Arg: folder.",
      inputSchema: {
        type: "object",
        properties: {
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" },
          port: { type: "number", description: "Public port (default: 24081)" },
          email: { type: "string", description: "Owner email (default: memory@local)" },
          publicUrl: { type: "string", description: "Override PUBLIC_URL" }
        },
        required: []
      }
    },
    {
      name: "memory_daily",
      description: "Append a thought to TODAY'S daily note \u2014 the day's general working space for capturing thoughts as they come. The first entry of the day is plain; later same-day entries are timestamped HH:MM. Use this to 'remember' something now. Args: text (the thought), folder, context (lines of the note to echo back).",
      inputSchema: {
        type: "object",
        properties: {
          text: { type: "string", description: "The thought to append (use \\n for newlines)" },
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" },
          context: { type: "number", description: "Lines of note context to return after write (default: 15)" }
        },
        required: ["text"]
      }
    },
    {
      name: "memory_log",
      description: "Append a thought to a specific note's running log, under a '### [[today]]' day header \u2014 an append-only journal of how ONE idea/topic evolves over days. First entry under a new day is plain; later same-day entries get HH:MM. Use this (not memory_daily) when you're tracking the evolution of a specific named topic over time. Args: file (note name without .md), text, folder, context.",
      inputSchema: {
        type: "object",
        properties: {
          file: { type: "string", description: 'Note name without .md extension (e.g. "work", "project-ideas")' },
          text: { type: "string", description: "The thought to append (use \\n for newlines)" },
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" },
          context: { type: "number", description: "Lines of note context to return after write (default: 15)" }
        },
        required: ["file", "text"]
      }
    },
    {
      name: "memory_bind_hub",
      description: "Bind an additional remote trip2g federation hub to this memory so federated_search also queries it. Writes a hub-<host>.md note (with free: true + mcp_federation_kb_url) into the vault. Multiple hubs coexist alongside the default hub.md (trip2g.com). The running --watch daemon will push the note so federation takes effect; if no daemon, run `up` first. Args: url (the hub's /_system/mcp endpoint), folder, id (optional short id, defaults to hostname).",
      inputSchema: {
        type: "object",
        properties: {
          url: { type: "string", description: "The hub MCP endpoint URL (e.g. https://demo.lahab.cc/_system/mcp)" },
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" },
          id: { type: "string", description: "Optional short id for mcp_federation_kb_id (default: hostname)" }
        },
        required: ["url"]
      }
    },
    {
      name: "memory_lint",
      description: "Zero-LLM commit-gate + OKF validation + stale report for a vault. Walks every .md note and reports: unresolved markers / contradictions (errors), notes missing a `type:` field (warnings), a missing index.md/log.md at the vault root (errors), and notes whose timestamp/date frontmatter is older than --stale-days (warnings). Run before treating a write as done. Args: folder, staleDays (default 60).",
      inputSchema: {
        type: "object",
        properties: {
          folder: { type: "string", description: "Vault directory path (default: ./memory-vault)" },
          staleDays: { type: "number", description: "Flag notes older than this many days as stale (default: 60)" }
        },
        required: []
      }
    }
  ];
}
__name(buildToolList, "buildToolList");
function buildServerEnv(opts) {
  const { port, iport, email, secret, encryptionKey } = opts;
  const env = {
    LISTEN_ADDR: `0.0.0.0:${port}`,
    INTERNAL_LISTEN_ADDR: `:${iport}`,
    DB_FILE: "/data/local.sqlite3",
    OWNER_EMAIL: email,
    PUBLIC_URL: `http://localhost:${port}`,
    JWT_SECRET: secret,
    DATA_ENCRYPTION_KEY: encryptionKey,
    STORAGE_BACKEND: "local",
    STORAGE_LOCAL_DIR: "/data/storage"
  };
  if (opts.features) env.FEATURES = opts.features;
  const forbidden = ["DEV", "RESEND_API_KEY", "SMTP_PASSWORD", "GIT_API_REPO_PATH"];
  for (const k of forbidden) {
    if (k in env) {
      throw new Error(`buildServerEnv: forbidden key ${k} must not be in server env`);
    }
  }
  return env;
}
__name(buildServerEnv, "buildServerEnv");
function buildDockerRunArgs(opts) {
  const { port, iport, stateDir, image } = opts;
  const host = opts.host ?? "127.0.0.1";
  const cName = opts.containerName ?? CONTAINER_NAME;
  const env = buildServerEnv(opts);
  const args = [
    "-d",
    "--name",
    cName
  ];
  if (opts.network) {
    args.push("--network", opts.network);
  }
  args.push(
    // Bind main port to the requested loopback address
    "-p",
    `${host}:${port}:${port}`,
    // Internal port always stays on 127.0.0.1 (backend-only)
    "-p",
    `127.0.0.1:${iport}:${iport}`,
    "-v",
    `${stateDir}/data:/data`
  );
  for (const [k, v] of Object.entries(env)) {
    args.push("-e", `${k}=${v}`);
  }
  args.push(image);
  return args;
}
__name(buildDockerRunArgs, "buildDockerRunArgs");
function buildFeaturesJson(serverContainer, reranker) {
  const vectorSearch = {
    enabled: true,
    model: "bge-m3",
    base_url: `http://${serverContainer}:8000/v1`
  };
  if (reranker) {
    vectorSearch.reranker = {
      enabled: true,
      base_url: `http://${serverContainer}:8000/rerank`,
      model: "BAAI/bge-reranker-v2-m3"
    };
  }
  return JSON.stringify({ vector_search: vectorSearch });
}
__name(buildFeaturesJson, "buildFeaturesJson");
function buildEmbedServerRunArgs(opts) {
  const args = [
    "-d",
    "--name",
    opts.name,
    "--network",
    opts.network,
    "-p",
    `127.0.0.1:${opts.hostPort}:8000`,
    "-v",
    `${opts.modelsDir}:/data`
  ];
  if (opts.loadReranker) args.push("-e", "LOAD_RERANKER=1");
  args.push(EMBED_SERVER_IMAGE);
  return args;
}
__name(buildEmbedServerRunArgs, "buildEmbedServerRunArgs");
function buildDataJson(vault, publicUrl, apiKey) {
  return {
    syncDirs: [
      {
        path: vault,
        apiUrl: publicUrl,
        apiKey,
        twoWaySync: true
      }
    ]
  };
}
__name(buildDataJson, "buildDataJson");
async function gqlRequest(endpoint, bearerToken, document, variables) {
  const res = await fetch(endpoint, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "authorization": `Bearer ${bearerToken}`
    },
    body: JSON.stringify({ query: (0, import_graphql.print)(document), variables })
  });
  if (!res.ok) {
    throw new Error(`GraphQL request failed: ${res.status} ${await res.text()}`);
  }
  const body = await res.json();
  if (body.errors && body.errors.length > 0) {
    throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  }
  if (body.data === void 0) {
    throw new Error(`GraphQL response missing data: ${JSON.stringify(body)}`);
  }
  return body.data;
}
__name(gqlRequest, "gqlRequest");
function readEnvFile(envFile) {
  if (!fs.existsSync(envFile)) return {};
  const lines = fs.readFileSync(envFile, "utf8").split("\n");
  const out = {};
  for (const line of lines) {
    const eq = line.indexOf("=");
    if (eq < 0) continue;
    out[line.slice(0, eq).trim()] = line.slice(eq + 1).trim();
  }
  return out;
}
__name(readEnvFile, "readEnvFile");
function writeEnvFile(envFile, data) {
  const content = Object.entries(data).map(([k, v]) => `${k}=${v}`).join("\n") + "\n";
  fs.writeFileSync(envFile, content, { mode: 384 });
  fs.chmodSync(envFile, 384);
}
__name(writeEnvFile, "writeEnvFile");
async function waitReady(url, timeoutMs, pollMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url, { signal: AbortSignal.timeout(2e3) });
      if (res.status === 200) return;
    } catch {
    }
    await new Promise((r) => setTimeout(r, pollMs));
  }
  throw new Error(`Timed out waiting for ${url} to return 200 after ${timeoutMs}ms`);
}
__name(waitReady, "waitReady");
function isPortBusy(port, host = "127.0.0.1") {
  return new Promise((resolve) => {
    const srv = net.createServer();
    srv.once("error", (err) => {
      if (err.code === "EACCES") {
        resolve(false);
      } else {
        resolve(true);
      }
    });
    srv.once("listening", () => srv.close(() => resolve(false)));
    srv.listen(port, host);
  });
}
__name(isPortBusy, "isPortBusy");
async function hatAuth(publicUrl, secret, email) {
  const jwt = signHatJwt(secret, email);
  const hatRes = await fetch(`${publicUrl}/_system/hat`, {
    method: "POST",
    redirect: "manual",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: `token=${encodeURIComponent(jwt)}`
  });
  const setCookie = hatRes.headers.get("set-cookie") || "";
  const match = setCookie.match(/trip2g_token=([^;]+)/);
  if (!match) {
    throw new Error(
      `HAT response did not set trip2g_token cookie. status=${hatRes.status} set-cookie=${setCookie}`
    );
  }
  return match[1];
}
__name(hatAuth, "hatAuth");
async function mintApiKey(publicUrl, secret, email) {
  const token = await hatAuth(publicUrl, secret, email);
  const gqlUrl = `${publicUrl}/_system/graphql`;
  const data = await gqlRequest(gqlUrl, token, CreateApiKeyDocument, {
    description: "agent-memory"
  });
  const result = data.admin.createApiKey;
  if (result.__typename === "ErrorPayload") {
    throw new Error(`createApiKey error: ${result.message}`);
  }
  if (result.__typename !== "CreateApiKeyPayload") {
    throw new Error(`Unknown createApiKey result type: ${result.__typename}`);
  }
  return {
    apiKey: result.value,
    apiKeyId: result.apiKey?.id ?? null
  };
}
__name(mintApiKey, "mintApiKey");
function writeDataJson(vault, publicUrl, apiKey) {
  const pluginDir = path.join(vault, ".obsidian", "plugins", "trip2g");
  fs.mkdirSync(pluginDir, { recursive: true });
  const dataFile = path.join(pluginDir, "data.json");
  fs.writeFileSync(dataFile, JSON.stringify(buildDataJson(vault, publicUrl, apiKey), null, 2));
  return dataFile;
}
__name(writeDataJson, "writeDataJson");
function isPidAlive(pid, cmdlinePattern) {
  try {
    process.kill(pid, 0);
    try {
      const cmdline = fs.readFileSync(`/proc/${pid}/cmdline`, "utf8").replace(/\0/g, " ");
      return cmdlinePattern.test(cmdline);
    } catch {
      return true;
    }
  } catch {
    return false;
  }
}
__name(isPidAlive, "isPidAlive");
function isContainerRunning(name) {
  try {
    const out = spawnSync("docker", ["ps", "-q", "--filter", `name=^${name}$`], { encoding: "utf8" });
    return (out.stdout || "").trim().length > 0;
  } catch {
    return false;
  }
}
__name(isContainerRunning, "isContainerRunning");
function containerExists(name) {
  try {
    const out = spawnSync("docker", ["ps", "-aq", "--filter", `name=^${name}$`], { encoding: "utf8" });
    return (out.stdout || "").trim().length > 0;
  } catch {
    return false;
  }
}
__name(containerExists, "containerExists");
function ensureNetwork(netName, dryRun) {
  if (dryRun) {
    console.log(`[dry-run] docker network create ${netName} (if absent)`);
    return;
  }
  const inspect = spawnSync("docker", ["network", "inspect", netName], { encoding: "utf8" });
  if (inspect.status === 0) return;
  spawnSync("docker", ["network", "create", netName], { encoding: "utf8" });
  console.log(`Created docker network ${netName}.`);
}
__name(ensureNetwork, "ensureNetwork");
function ensureEmbedServerImage(dryRun) {
  const buildDir = path.join(repoRoot(), "retriever");
  if (dryRun) {
    console.log(`[dry-run] docker build -t ${EMBED_SERVER_IMAGE} ${buildDir} (if image absent)`);
    return;
  }
  const inspect = spawnSync("docker", ["image", "inspect", EMBED_SERVER_IMAGE], { encoding: "utf8" });
  if (inspect.status === 0) return;
  console.log(`Building ${EMBED_SERVER_IMAGE} image from ${buildDir} (first time \u2014 a few minutes)...`);
  const build = spawnSync("docker", ["build", "-t", EMBED_SERVER_IMAGE, buildDir], { encoding: "utf8" });
  if (build.status !== 0) {
    throw new Error(`docker build ${EMBED_SERVER_IMAGE} failed: ${(build.stderr || build.stdout || "").trim().slice(-2e3)}`);
  }
  console.log(`Built ${EMBED_SERVER_IMAGE}.`);
}
__name(ensureEmbedServerImage, "ensureEmbedServerImage");
function ensureEmbedServerSidecar(opts, dryRun) {
  const args = buildEmbedServerRunArgs(opts);
  if (dryRun) {
    console.log(`[dry-run] docker run ${args.join(" ")}`);
    return;
  }
  if (isContainerRunning(opts.name)) {
    const env = spawnSync("docker", ["inspect", opts.name, "--format", "{{json .Config.Env}}"], { encoding: "utf8" });
    const hasReranker = (env.stdout || "").includes("LOAD_RERANKER=1");
    if (hasReranker === opts.loadReranker) {
      console.log(`Sidecar ${opts.name} already running.`);
      return;
    }
    console.log(`Sidecar ${opts.name} running without matching reranker mode \u2014 recreating.`);
  }
  if (containerExists(opts.name)) {
    spawnSync("docker", ["rm", "-f", opts.name], { encoding: "utf8" });
  }
  console.log(`Starting sidecar ${opts.name} (bge-m3${opts.loadReranker ? " + bge-reranker-v2-m3" : ""})...`);
  const run = spawnSync("docker", ["run", ...args], { encoding: "utf8" });
  if (run.status !== 0) {
    throw new Error(`docker run ${opts.name} failed: ${(run.stderr || run.stdout || "").trim()}`);
  }
}
__name(ensureEmbedServerSidecar, "ensureEmbedServerSidecar");
function repoRoot() {
  const scriptDir = path.dirname(new URL(import.meta.url).pathname);
  return path.resolve(scriptDir, "..", "..", "..");
}
__name(repoRoot, "repoRoot");
function _tzLabel() {
  const tz = process.env.TZ?.trim();
  if (tz) return tz;
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  } catch {
    return "UTC";
  }
}
__name(_tzLabel, "_tzLabel");
function _stampBlock(text, now) {
  const normalized = text.replace(/\\n/g, "\n");
  const lines = normalized.split("\n");
  const hhmm = now.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", hour12: false });
  lines[0] = `${hhmm} ${lines[0]}`;
  return lines.join("\n");
}
__name(_stampBlock, "_stampBlock");
function _plainBlock(text) {
  return text.replace(/\\n/g, "\n");
}
__name(_plainBlock, "_plainBlock");
function atomicWrite(filePath, content) {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  const tmp = path.join(dir, `.memcli-${process.pid}-${Date.now()}.tmp`);
  try {
    fs.writeFileSync(tmp, content, "utf8");
    fs.renameSync(tmp, filePath);
  } catch (err) {
    try {
      fs.unlinkSync(tmp);
    } catch {
    }
    throw err;
  }
}
__name(atomicWrite, "atomicWrite");
function tailLines(filePath, n) {
  if (n <= 0) return "";
  if (!fs.existsSync(filePath)) return "";
  const lines = fs.readFileSync(filePath, "utf8").split("\n");
  return lines.slice(-n).join("\n");
}
__name(tailLines, "tailLines");
function buildDailyEntry(day, tz, stampedEntry) {
  const frontmatter = `---
timezone: ${tz}
type: daily
---
`;
  const header = `- [[_index|Home]]
- [[daily/_index|Daily]]

# ${day}
`;
  return `${frontmatter}${header}
${stampedEntry}
`;
}
__name(buildDailyEntry, "buildDailyEntry");
function ensureDailyIndex(vault, day) {
  const idx = path.join(vault, "daily", "_index.md");
  if (!fs.existsSync(idx)) {
    atomicWrite(idx, "- [[_index|Home]]\n- [[daily/_index|Daily]]\n\n# Daily\n");
  }
  const text = fs.readFileSync(idx, "utf8");
  if (text.includes(`[[${day}]]`)) return;
  const lines = text.split("\n");
  const out = [];
  let inserted = false;
  for (const line of lines) {
    out.push(line);
    if (!inserted && line.trim() === "# Daily") {
      out.push(`- [[${day}]]`);
      inserted = true;
    }
  }
  if (!inserted) out.push(`- [[${day}]]`);
  atomicWrite(idx, out.join("\n") + "\n");
}
__name(ensureDailyIndex, "ensureDailyIndex");
function appendDaily(vault, text, now) {
  const localDay = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, "0"),
    String(now.getDate()).padStart(2, "0")
  ].join("-");
  const note = path.join(vault, "daily", `${localDay}.md`);
  if (!fs.existsSync(note)) {
    const entry = _plainBlock(text);
    atomicWrite(note, buildDailyEntry(localDay, _tzLabel(), entry));
  } else {
    const entry = _stampBlock(text, now);
    const existing = fs.readFileSync(note, "utf8").replace(/\n+$/, "");
    atomicWrite(note, `${existing}

${entry}
`);
  }
  ensureDailyIndex(vault, localDay);
  return note;
}
__name(appendDaily, "appendDaily");
function appendLog(vault, file, text, now) {
  const localDay = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, "0"),
    String(now.getDate()).padStart(2, "0")
  ].join("-");
  const fileMd = file.endsWith(".md") ? file : `${file}.md`;
  const note = path.join(vault, fileMd);
  const header = `### [[${localDay}]]`;
  const existing = fs.existsSync(note) ? fs.readFileSync(note, "utf8") : "";
  if (existing.split("\n").includes(header)) {
    const entry = _stampBlock(text, now);
    const base = existing.replace(/\n+$/, "");
    atomicWrite(note, `${base}

${entry}
`);
  } else {
    const entry = _plainBlock(text);
    const base = existing.replace(/\n+$/, "");
    const prefix = base ? `${base}

` : "---\ntype: note\n---\n\n";
    atomicWrite(note, `${prefix}${header}

${entry}
`);
  }
  return note;
}
__name(appendLog, "appendLog");
function captureLines(fn) {
  const lines = [];
  const origLog = console.log;
  const origErr = console.error;
  const origWarn = console.warn;
  console.log = (...args) => lines.push(args.map(String).join(" "));
  console.error = (...args) => lines.push(args.map(String).join(" "));
  console.warn = (...args) => lines.push(args.map(String).join(" "));
  try {
    fn();
  } finally {
    console.log = origLog;
    console.error = origErr;
    console.warn = origWarn;
  }
  return lines.join("\n");
}
__name(captureLines, "captureLines");
async function captureAsync(fn) {
  const lines = [];
  const origLog = console.log;
  const origErr = console.error;
  const origWarn = console.warn;
  console.log = (...args) => lines.push(args.map(String).join(" "));
  console.error = (...args) => lines.push(args.map(String).join(" "));
  console.warn = (...args) => lines.push(args.map(String).join(" "));
  try {
    await fn();
  } finally {
    console.log = origLog;
    console.error = origErr;
    console.warn = origWarn;
  }
  return lines.join("\n");
}
__name(captureAsync, "captureAsync");
async function runUp(flags) {
  try {
    const text = await captureAsync(() => cmdUp(flags, flags.dryRun));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runUp, "runUp");
function runDown(flags) {
  try {
    const text = captureLines(() => cmdDown(flags.dryRun, flags.folder, containerName(flags)));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runDown, "runDown");
function runStatus(flags) {
  try {
    const text = captureLines(() => cmdStatus(containerName(flags)));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runStatus, "runStatus");
function runLogs(flags) {
  try {
    const result = spawnSync("docker", ["logs", containerName(flags)], {
      encoding: "utf8"
    });
    if (result.error) throw result.error;
    const text = [result.stdout, result.stderr].filter(Boolean).join("\n");
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runLogs, "runLogs");
async function runKey(flags) {
  try {
    const text = await captureAsync(() => cmdKey(flags, flags.dryRun));
    return { text, isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runKey, "runKey");
function brokenLinkWarning(vault, note) {
  const root = path.resolve(vault);
  const files = walkMarkdown(root);
  const resolveSet = buildResolutionSet(root, files);
  let body;
  try {
    body = splitFrontmatter(fs.readFileSync(note, "utf8")).body;
  } catch {
    return "";
  }
  const broken = findBrokenLinks(body, resolveSet);
  if (broken.length === 0) return "";
  const rel = path.relative(root, note);
  const links = broken.map((t) => `[[${t}]]`).join(", ");
  return `
\u26A0 broken links in ${rel}: ${links} \u2014 create the target note(s) or fix the link.`;
}
__name(brokenLinkWarning, "brokenLinkWarning");
function runDaily(vault, text, context) {
  if (!text) {
    return { text: "Error: `daily` requires a text argument", isError: true };
  }
  try {
    const note = appendDaily(path.resolve(vault), text, /* @__PURE__ */ new Date());
    return { text: tailLines(note, context) + brokenLinkWarning(vault, note), isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runDaily, "runDaily");
function runLog(vault, file, text, context) {
  if (!file || !text) {
    return { text: "Error: `log` requires <file> and <text> arguments", isError: true };
  }
  try {
    const note = appendLog(path.resolve(vault), file, text, /* @__PURE__ */ new Date());
    return { text: tailLines(note, context) + brokenLinkWarning(vault, note), isError: false };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runLog, "runLog");
function runHub(hubUrl, folder, id, dryRun) {
  if (!hubUrl) {
    return { text: "Error: `hub` requires a <url> argument", isError: true };
  }
  let parsed;
  try {
    parsed = new URL(hubUrl);
  } catch {
    return { text: `Error: invalid URL: ${hubUrl}`, isError: true };
  }
  if (!parsed.protocol.startsWith("http")) {
    return { text: `Error: URL must start with http:// or https://, got: ${hubUrl}`, isError: true };
  }
  const slug = hubSlug(hubUrl);
  const kbId = id ?? parsed.hostname;
  const vault = path.resolve(folder);
  const notePath = path.join(vault, `hub-${slug}.md`);
  const noteContent = buildHubNote(hubUrl, kbId);
  if (dryRun) {
    const lines = [
      `[dry-run] would write hub note: ${notePath}`,
      `[dry-run] content:`,
      noteContent,
      `[dry-run] note: the running --watch daemon will push the note so federation takes effect (or run \`up\` first if no daemon).`
    ];
    return { text: lines.join("\n"), isError: false };
  }
  try {
    atomicWrite(notePath, noteContent);
    return {
      text: [
        `bound hub ${kbId} \u2192 ${hubUrl} (note: ${notePath})`,
        `Note: the running --watch daemon will push the note so federation takes effect (or run \`up\` first if no daemon).`
      ].join("\n"),
      isError: false
    };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runHub, "runHub");
function walkMarkdown(dir) {
  const out = [];
  let entries;
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === ".trip2g-memory" || entry.name === ".obsidian") continue;
      out.push(...walkMarkdown(full));
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      out.push(full);
    }
  }
  return out;
}
__name(walkMarkdown, "walkMarkdown");
function splitFrontmatter(content) {
  if (!content.startsWith("---\n") && !content.startsWith("---\r\n")) {
    return { frontmatter: "", body: content };
  }
  const rest = content.slice(content.indexOf("\n") + 1);
  const end = rest.search(/^---\s*$/m);
  if (end < 0) return { frontmatter: "", body: content };
  const frontmatter = rest.slice(0, end);
  const afterFence = rest.slice(end).replace(/^---\s*$/m, "");
  return { frontmatter, body: afterFence };
}
__name(splitFrontmatter, "splitFrontmatter");
function stripCode(body) {
  return body.replace(/```[\s\S]*?```/g, "").replace(/`[^`]*`/g, "");
}
__name(stripCode, "stripCode");
function extractWikilinks(body) {
  const stripped = stripCode(body);
  const out = [];
  const re = /(!?)\[\[([^\]]+)\]\]/g;
  let m;
  while ((m = re.exec(stripped)) !== null) {
    if (m[1] === "!") continue;
    let target = m[2].split("|")[0].split("#")[0].trim();
    if (!target) continue;
    if (/\.[a-z0-9]{1,4}$/i.test(target)) continue;
    out.push(target);
  }
  return out;
}
__name(extractWikilinks, "extractWikilinks");
function buildResolutionSet(root, files) {
  const set = /* @__PURE__ */ new Set();
  for (const file of files) {
    const rel = path.relative(root, file).replace(/\.md$/i, "");
    const base = path.basename(file).replace(/\.md$/i, "");
    set.add(rel.toLowerCase());
    set.add(base.toLowerCase());
  }
  return set;
}
__name(buildResolutionSet, "buildResolutionSet");
function findBrokenLinks(body, resolveSet) {
  const broken = [];
  for (const target of extractWikilinks(body)) {
    if (!resolveSet.has(target.toLowerCase())) broken.push(target);
  }
  return broken;
}
__name(findBrokenLinks, "findBrokenLinks");
function lintVault(vault, staleDays) {
  const violations = [];
  const root = path.resolve(vault);
  const files = walkMarkdown(root);
  const now = /* @__PURE__ */ new Date();
  const staleMs = staleDays * 24 * 60 * 60 * 1e3;
  const resolveSet = buildResolutionSet(root, files);
  for (const file of files) {
    const rel = path.relative(root, file);
    let content;
    try {
      content = fs.readFileSync(file, "utf8");
    } catch {
      continue;
    }
    const { frontmatter, body } = splitFrontmatter(content);
    const bodyForGate = stripCode(body);
    if (/Status:\s*Unresolved/i.test(bodyForGate) || bodyForGate.includes("CONTRADICTION")) {
      violations.push({
        level: "error",
        kind: "commit-gate",
        file: rel,
        detail: "unresolved marker or CONTRADICTION found in note body"
      });
    }
    const isSystemNote = path.basename(file).startsWith("_");
    if (!isSystemNote && !/^\s*type:\s*\S/m.test(frontmatter)) {
      violations.push({
        level: "warn",
        kind: "okf-type",
        file: rel,
        detail: "note has no `type:` field in frontmatter (OKF)"
      });
    }
    const dateMatch = frontmatter.match(/^\s*(?:timestamp|date):\s*(.+?)\s*$/m);
    if (dateMatch) {
      const parsed = new Date(dateMatch[1]);
      if (!isNaN(parsed.getTime())) {
        const age = now.getTime() - parsed.getTime();
        if (age > staleMs) {
          violations.push({
            level: "warn",
            kind: "stale",
            file: rel,
            detail: `note dated ${dateMatch[1]} is older than ${staleDays} days`
          });
        }
      }
    }
    for (const target of findBrokenLinks(body, resolveSet)) {
      violations.push({
        level: "warn",
        kind: "broken-link",
        file: rel,
        detail: `[[${target}]]`
      });
    }
  }
  for (const reserved of ["index.md", "log.md"]) {
    if (!fs.existsSync(path.join(root, reserved))) {
      violations.push({
        level: "error",
        kind: "okf-reserved",
        file: reserved,
        detail: `reserved OKF note ${reserved} missing at vault root`
      });
    }
  }
  return { violations };
}
__name(lintVault, "lintVault");
function formatLintReport(result) {
  const { violations } = result;
  const errors = violations.filter((v) => v.level === "error");
  const warns = violations.filter((v) => v.level === "warn");
  const lines = [];
  if (violations.length === 0) {
    lines.push("lint: clean \u2014 0 errors, 0 warnings");
    return lines.join("\n");
  }
  for (const v of violations) {
    const tag = v.level === "error" ? "ERROR" : "warn ";
    lines.push(`${tag} [${v.kind}] ${v.file}: ${v.detail}`);
  }
  lines.push("");
  lines.push(`lint: ${errors.length} error(s), ${warns.length} warning(s)`);
  return lines.join("\n");
}
__name(formatLintReport, "formatLintReport");
function runLint(folder, staleDays) {
  try {
    const result = lintVault(path.resolve(folder), staleDays);
    const isError = result.violations.some((v) => v.level === "error");
    return { text: formatLintReport(result), isError };
  } catch (err) {
    return { text: `Error: ${err.message}`, isError: true };
  }
}
__name(runLint, "runLint");
function runOpen(flags) {
  const vault = path.resolve(flags.folder);
  const stateDir = path.join(vault, ".trip2g-memory");
  const envFile = path.join(stateDir, "env");
  const existingEnv = readEnvFile(envFile);
  if (!existingEnv.JWT_SECRET) {
    return { text: "Error: no JWT_SECRET found \u2014 run `up` first", isError: true };
  }
  const secret = existingEnv.JWT_SECRET;
  const email = flags.email;
  const port = flags.port;
  const publicUrl = flags.publicUrl || `http://localhost:${port}`;
  const jwt = signHatJwt(secret, email);
  const loginUrl = buildHatLoginUrl(publicUrl, jwt);
  if (flags.dryRun) {
    return {
      text: [
        `[dry-run] would open this ?hat= login URL in the browser (signs in as ${email}, lands on /):`,
        loginUrl
      ].join("\n"),
      isError: false
    };
  }
  const opener = process.platform === "darwin" ? { cmd: "open", args: [loginUrl] } : process.platform === "win32" ? { cmd: "cmd", args: ["/c", "start", "", loginUrl] } : { cmd: "xdg-open", args: [loginUrl] };
  try {
    const child = spawn(opener.cmd, opener.args, { detached: true, stdio: "ignore" });
    child.unref();
  } catch {
    return {
      text: `Could not launch a browser automatically. Open this URL manually to sign in as ${email}:
` + loginUrl,
      isError: false
    };
  }
  return {
    text: [
      `Opening the web UI in your browser, signed in as ${email} \u2192 ${publicUrl}`,
      `If the browser did not open, open this URL manually: ${loginUrl}`
    ].join("\n"),
    isError: false
  };
}
__name(runOpen, "runOpen");
function cmdDaily(vault, text, context) {
  const result = runDaily(vault, text, context);
  if (result.isError) {
    console.error(result.text);
    process.exit(1);
  }
  console.log(result.text);
}
__name(cmdDaily, "cmdDaily");
function cmdLog(vault, file, text, context) {
  const result = runLog(vault, file, text, context);
  if (result.isError) {
    console.error(result.text);
    process.exit(1);
  }
  console.log(result.text);
}
__name(cmdLog, "cmdLog");
function cmdHub(hubUrl, folder, id, dryRun) {
  const result = runHub(hubUrl, folder, id, dryRun);
  if (result.isError) {
    console.error(result.text);
    process.exit(1);
  }
  console.log(result.text);
}
__name(cmdHub, "cmdHub");
function cmdLint(folder, staleDays) {
  const result = runLint(folder, staleDays);
  if (result.isError) {
    console.error(result.text);
    process.exit(1);
  }
  console.log(result.text);
}
__name(cmdLint, "cmdLint");
function cmdOpen(flags) {
  const result = runOpen(flags);
  if (result.isError) {
    console.error(result.text);
    process.exit(1);
  }
  console.log(result.text);
}
__name(cmdOpen, "cmdOpen");
function printHelp() {
  console.log(`
memcli \u2014 minimal self-hosted agent memory backed by trip2g

USAGE
  memcli [SUBCOMMAND] [FLAGS]

SUBCOMMANDS
  up (default)  Boot trip2g server + sync watcher, mint API key
  down          Stop the server and watcher
  status        Show container and watcher status
  logs          Stream container logs
  key           Re-mint the API key (rewrites data.json)
  daily "<text>"         Append a HH:MM entry to today's daily note
  log <file> "<text>"    Append a HH:MM entry under today's ### [[date]] section
  hub <url>              Bind an additional remote federation hub to the vault
  lint          Commit-gate + OKF validation + stale report over the vault
  open          Open the web UI in your browser via a ?hat= login URL (signs in via hot auth token)
  mcp           Run as MCP stdio server (also auto-detected when stdin is piped)

FLAGS (up)
  --folder <path>      Vault directory (default: ./memory-vault)
  --port <n>           Public port (default: ${DEFAULT_PORT})
  --email <addr>       Owner email (default: ${DEFAULT_EMAIL})
  --image <ref>        Docker image (default: ${DEFAULT_IMAGE})
  --public-url <url>   Override PUBLIC_URL (default: http://localhost:<port>)
  --no-hub             Skip writing the federation hub note (hub.md)
  --no-seed            Skip seeding OKF starter notes (index/log/AGENTS/SCHEMA)
  --hub-url <url>      Override hub MCP endpoint (default: ${DEFAULT_HUB_URL})
  --name <id>          Run an isolated instance named trip2g-memory-<id> (needs its own --port)
  --kanban             Install the trip2g kanban template into <vault>/_layouts/kanban.html
                       and seed a sample kanban.md board (skipped with --no-seed)
  --kanban-bundle <url>  Override the kanban.js bundle <script src> (local-dev, e.g. http://localhost:8770/kanban.js)
  --embedded           Start the local embedding server sidecar (bge-m3) and enable
                       vector search. Off by default: the model is ~2GB and CPU-heavy;
                       first boot downloads it and can take minutes. Model cache dir:
                       MODELS_DIR env (default ~/models).
  --reranker           Also load the reranker (bge-reranker-v2-m3, ~2GB) on the same
                       sidecar for second-stage reranking. Implies --embedded.

FLAGS (lint)
  --folder <path>      Vault directory (default: ./memory-vault)
  --stale-days <n>     Flag notes older than this many days (default: ${DEFAULT_STALE_DAYS})

FLAGS (hub)
  --folder <path>      Vault directory (default: ./memory-vault)
  --id <kb-id>         Short id for mcp_federation_kb_id (default: hostname)
  --dry-run            Print what would be written, write nothing

FLAGS (open)
  Reuses --folder/--port/--email/--public-url (same defaults as \`up\`).
  --dry-run            Print the ?hat= login URL, do not launch a browser.

FLAGS (daily / log)
  --folder <path>      Vault directory (default: ./memory-vault)
  --context <n>        Lines of note context to print after write (default: 15)

GLOBAL FLAGS
  --dry-run    Print commands without executing
  --help       Show this help

NOTES
  State is stored in <vault>/.trip2g-memory/ (env file, data dir, PID file).
  The Docker image MUST include local-storage backend support (feat/filestorage).
  daily/log do not call sync \u2014 the running trip2g-sync --watch daemon picks up changes.
  Pipe stdin or use the mcp subcommand to run as an MCP stdio JSON-RPC server.
`.trim());
}
__name(printHelp, "printHelp");
async function cmdUp(flags, dryRun) {
  const { folder, port, email, image } = flags;
  const name = containerName(flags);
  if (!folder) {
    console.error("Error: --folder <vault> is required for `up`");
    process.exit(1);
  }
  const vault = path.resolve(folder);
  const stateDir = path.join(vault, ".trip2g-memory");
  const envFile = path.join(stateDir, "env");
  const pidFile = path.join(stateDir, "watch.pid");
  const iport = port + 1;
  const publicUrl = flags.publicUrl || `http://localhost:${port}`;
  const embedded = flags.embedded || flags.reranker;
  if (flags.reranker && !flags.embedded) {
    console.log("--reranker implies --embedded: enabling the embedding sidecar too.");
  }
  const netName = networkName(name);
  const embName = embeddingContainerName(name);
  const embPort = port + 2;
  const containerRunning = isContainerRunning(name);
  const sidecarsUp = !embedded || isContainerRunning(embName);
  let watcherAlive = false;
  if (fs.existsSync(pidFile)) {
    const pid = parseInt(fs.readFileSync(pidFile, "utf8").trim(), 10);
    if (!isNaN(pid)) {
      watcherAlive = isPidAlive(pid, /trip2g-sync/);
    }
  }
  if (containerRunning && watcherAlive && sidecarsUp && !dryRun) {
    console.log(`${name} is already up. Web: ${publicUrl}`);
    return;
  }
  if (!dryRun) {
    fs.mkdirSync(stateDir, { recursive: true, mode: 448 });
    fs.chmodSync(stateDir, 448);
  }
  let secret;
  let encryptionKey;
  const existingEnv = dryRun ? {} : readEnvFile(envFile);
  const needsWrite = !existingEnv.JWT_SECRET || !existingEnv.DATA_ENCRYPTION_KEY;
  if (existingEnv.JWT_SECRET) {
    secret = existingEnv.JWT_SECRET;
    console.log("Reusing existing JWT_SECRET from state dir.");
  } else {
    secret = crypto.randomBytes(32).toString("hex");
  }
  if (existingEnv.DATA_ENCRYPTION_KEY) {
    encryptionKey = existingEnv.DATA_ENCRYPTION_KEY;
    console.log("Reusing existing DATA_ENCRYPTION_KEY from state dir.");
  } else {
    encryptionKey = crypto.randomBytes(16).toString("hex");
  }
  if (!dryRun && needsWrite) {
    writeEnvFile(envFile, { JWT_SECRET: secret, DATA_ENCRYPTION_KEY: encryptionKey });
    console.log("Generated secrets and wrote to state dir.");
  } else if (dryRun && needsWrite) {
    console.log("[dry-run] Would generate JWT_SECRET and DATA_ENCRYPTION_KEY and write to", envFile);
  }
  if (!dryRun && !containerRunning && await isPortBusy(port, flags.host)) {
    console.error(`Error: port ${port} is busy \u2014 pass --port for this instance (or stop what holds it).`);
    process.exit(1);
  }
  if (embedded) {
    ensureNetwork(netName, dryRun);
    ensureEmbedServerImage(dryRun);
    ensureEmbedServerSidecar({
      name: embName,
      network: netName,
      hostPort: embPort,
      modelsDir: process.env.MODELS_DIR || path.join(os.homedir(), "models"),
      loadReranker: flags.reranker
    }, dryRun);
    if (dryRun) {
      console.log(`[dry-run] Would poll http://localhost:${embPort}/health until 200 (timeout ${EMBED_READY_TIMEOUT_MS}ms)`);
    } else {
      console.log(`Waiting for embedding server at http://localhost:${embPort}/health (first run downloads ~2GB models \u2014 this can take minutes)...`);
      await waitReady(`http://localhost:${embPort}/health`, EMBED_READY_TIMEOUT_MS, EMBED_READY_POLL_MS);
      console.log("Embedding server is ready.");
    }
  }
  const dockerArgs = buildDockerRunArgs({
    port,
    iport,
    host: flags.host,
    email,
    secret,
    encryptionKey,
    stateDir,
    image,
    containerName: name,
    ...embedded ? {
      network: netName,
      features: buildFeaturesJson(embName, flags.reranker)
    } : {}
  });
  if (dryRun) {
    console.log(`[dry-run] docker run ${dockerArgs.join(" ")}`);
  } else if (!containerRunning) {
    console.log(`Starting container ${name} (image: ${image})...`);
    console.log("NOTE: image must include feat/filestorage local-storage backend.");
    spawnSync("docker", ["run", ...dockerArgs], { encoding: "utf8" });
  } else {
    console.log(`Container ${name} already running, skipping docker run.`);
    if (embedded) {
      console.log("NOTE: FEATURES/network apply only on container create \u2014 run `down` first to rewire an existing container.");
    }
  }
  const readyzUrl = `http://localhost:${iport}/readyz`;
  if (dryRun) {
    console.log(`[dry-run] Would poll ${readyzUrl} until 200 (timeout ${READY_TIMEOUT_MS}ms)`);
  } else {
    console.log(`Waiting for server to be ready at ${readyzUrl} ...`);
    await waitReady(readyzUrl, READY_TIMEOUT_MS, READY_POLL_MS);
    console.log("Server is ready.");
  }
  let apiKey;
  if (dryRun) {
    console.log(
      `[dry-run] Would POST ${publicUrl}/_system/hat with HS256 JWT (email=${email}, ae=true, exp=now+300)`
    );
    console.log(`[dry-run] Would POST ${publicUrl}/_system/graphql mutation createApiKey`);
    apiKey = "<api-key-would-be-minted>";
  } else {
    console.log("Minting admin API key via HAT...");
    const result = await mintApiKey(publicUrl, secret, email);
    apiKey = result.apiKey;
    console.log("API key minted.");
    const stateJson = path.join(stateDir, "state.json");
    let state = {};
    try {
      state = JSON.parse(fs.readFileSync(stateJson, "utf8"));
    } catch {
    }
    state.apiKey = apiKey;
    if (result.apiKeyId != null) state.apiKeyId = result.apiKeyId;
    fs.writeFileSync(stateJson, JSON.stringify(state, null, 2), { mode: 384 });
  }
  if (dryRun) {
    const pluginDir = path.join(vault, ".obsidian", "plugins", "trip2g", "data.json");
    console.log(`[dry-run] Would write ${pluginDir}`);
    console.log(
      "[dry-run] Content:",
      JSON.stringify(buildDataJson(vault, publicUrl, "<api-key>"), null, 2)
    );
  } else {
    const dataFile = writeDataJson(vault, publicUrl, apiKey);
    console.log(`Wrote plugin config to ${dataFile}`);
  }
  const hubNotePath = path.join(vault, "hub.md");
  if (!flags.noHub) {
    if (dryRun) {
      console.log(
        `[dry-run] would write hub note ${hubNotePath} (federates \u2192 ${flags.hubUrl})`
      );
    } else if (fs.existsSync(hubNotePath)) {
      console.log(`Hub note already present at ${hubNotePath}, skipping.`);
    } else {
      fs.writeFileSync(hubNotePath, buildHubNote(flags.hubUrl), "utf8");
      console.log(`Wrote hub note to ${hubNotePath} (federates \u2192 ${flags.hubUrl})`);
    }
  }
  if (!flags.noSeed) {
    const seeds = [
      { file: "index.md", content: buildIndexNote() },
      { file: "log.md", content: buildLogNote() },
      { file: "AGENTS.md", content: buildAgentsNote() },
      { file: "SCHEMA.md", content: buildSchemaNote() },
      { file: "_index.md", content: buildHomeNote() },
      { file: "_header.md", content: buildHeaderNote() }
    ];
    for (const { file, content } of seeds) {
      const seedPath = path.join(vault, file);
      if (dryRun) {
        console.log(`[dry-run] would write ${seedPath}`);
      } else if (fs.existsSync(seedPath)) {
        console.log(`Seed note already present at ${seedPath}, skipping.`);
      } else {
        fs.writeFileSync(seedPath, content, "utf8");
        console.log(`Wrote seed note ${seedPath}`);
      }
    }
  }
  if (flags.kanban) {
    await installKanban(vault, flags.kanbanBundle, flags.noSeed, dryRun);
  }
  const repo = repoRoot();
  const syncMjs = path.join(repo, "obsidian-sync", "dist", "trip2g-sync.mjs");
  const watcherArgs = [
    syncMjs,
    "--watch",
    "--folder",
    vault,
    "--api-url",
    `${publicUrl}/_system/graphql`,
    "--api-key",
    apiKey
  ];
  if (dryRun) {
    console.log(`[dry-run] spawn node ${watcherArgs.join(" ")} (detached, PID \u2192 ${pidFile})`);
  } else if (!watcherAlive) {
    console.log("Starting sync watcher...");
    const child = spawn(process.execPath, watcherArgs, {
      detached: true,
      stdio: "ignore"
    });
    child.unref();
    fs.writeFileSync(pidFile, String(child.pid));
    console.log(`Watcher started (PID ${child.pid}), config written to ${pidFile}`);
  } else {
    console.log("Watcher already running.");
  }
  console.log("");
  console.log(`memory live \u2014 web: ${publicUrl}  read/write .md in ${vault}`);
}
__name(cmdUp, "cmdUp");
function cmdDown(dryRun, folder, name) {
  if (dryRun) {
    console.log(`[dry-run] docker stop ${name}`);
    console.log(`[dry-run] docker rm ${name}`);
    console.log(`[dry-run] docker rm -f ${embeddingContainerName(name)} (if present)`);
    console.log(`[dry-run] docker network rm ${networkName(name)} (if present)`);
    if (folder) {
      const pidFile = path.join(path.resolve(folder), ".trip2g-memory", "watch.pid");
      console.log(`[dry-run] Would kill watcher PID from ${pidFile} and remove pid file`);
    }
  } else {
    try {
      spawnSync("docker", ["stop", name], { encoding: "utf8" });
    } catch {
    }
    try {
      spawnSync("docker", ["rm", name], { encoding: "utf8" });
    } catch {
    }
    console.log(`Container ${name} stopped and removed.`);
    for (const sidecar of [embeddingContainerName(name)]) {
      if (containerExists(sidecar)) {
        spawnSync("docker", ["rm", "-f", sidecar], { encoding: "utf8" });
        console.log(`Sidecar ${sidecar} removed.`);
      }
    }
    try {
      spawnSync("docker", ["network", "rm", networkName(name)], { encoding: "utf8" });
    } catch {
    }
    if (folder) {
      const pidFile = path.join(path.resolve(folder), ".trip2g-memory", "watch.pid");
      if (fs.existsSync(pidFile)) {
        const raw = fs.readFileSync(pidFile, "utf8").trim();
        const pid = parseInt(raw, 10);
        if (!isNaN(pid) && isPidAlive(pid, /trip2g-sync/)) {
          try {
            process.kill(pid, "SIGTERM");
            console.log(`Watcher process (PID ${pid}) terminated.`);
          } catch {
          }
        }
        try {
          fs.unlinkSync(pidFile);
        } catch {
        }
      }
    }
  }
}
__name(cmdDown, "cmdDown");
function cmdStatus(name) {
  console.log("=== Container ===");
  const out = spawnSync(
    "docker",
    [
      "ps",
      "-a",
      "--filter",
      `name=^${name}$`,
      "--filter",
      `name=^${embeddingContainerName(name)}$`,
      "--format",
      "table {{.Names}}	{{.Status}}	{{.Ports}}"
    ],
    { encoding: "utf8" }
  );
  console.log(out.stdout || "(no output)");
}
__name(cmdStatus, "cmdStatus");
function cmdLogs(name) {
  const result = spawnSync("docker", ["logs", name], {
    encoding: "utf8",
    stdio: "inherit"
  });
  if (result.error) throw result.error;
}
__name(cmdLogs, "cmdLogs");
async function cmdKey(flags, dryRun) {
  const { folder, port, email } = flags;
  if (!folder) {
    console.error("Error: --folder <vault> is required for `key`");
    process.exit(1);
  }
  const vault = path.resolve(folder);
  const stateDir = path.join(vault, ".trip2g-memory");
  const envFile = path.join(stateDir, "env");
  const stateJson = path.join(stateDir, "state.json");
  const publicUrl = flags.publicUrl || `http://localhost:${port}`;
  const existingEnv = readEnvFile(envFile);
  if (!existingEnv.JWT_SECRET) {
    console.error("No existing JWT_SECRET found. Run `up` first.");
    process.exit(1);
  }
  const secret = existingEnv.JWT_SECRET;
  let prevState = {};
  try {
    prevState = JSON.parse(fs.readFileSync(stateJson, "utf8"));
  } catch {
  }
  const priorApiKeyId = typeof prevState.apiKeyId === "number" ? prevState.apiKeyId : null;
  if (dryRun) {
    console.log(`[dry-run] Would mint new API key via HAT for ${email}`);
    if (priorApiKeyId != null) {
      console.log(`[dry-run] Would disable prior API key id=${priorApiKeyId}`);
    }
    console.log(
      `[dry-run] Would rewrite ${path.join(vault, ".obsidian", "plugins", "trip2g", "data.json")}`
    );
    return;
  }
  console.log("Minting new API key...");
  const { apiKey, apiKeyId } = await mintApiKey(publicUrl, secret, email);
  prevState.apiKey = apiKey;
  if (apiKeyId != null) prevState.apiKeyId = apiKeyId;
  fs.writeFileSync(stateJson, JSON.stringify(prevState, null, 2), { mode: 384 });
  writeDataJson(vault, publicUrl, apiKey);
  console.log("New API key written to data.json.");
  if (priorApiKeyId != null) {
    try {
      const gqlUrl = `${publicUrl}/_system/graphql`;
      const token = await hatAuth(publicUrl, secret, email);
      const data = await gqlRequest(gqlUrl, token, DisableApiKeyDocument, {
        id: priorApiKeyId
      });
      const disResult = data.admin.disableApiKey;
      if (disResult.__typename === "ErrorPayload") {
        console.warn(
          `Warning: could not disable prior API key id=${priorApiKeyId}: ${disResult.message}`
        );
      } else {
        console.log(`Prior API key id=${priorApiKeyId} disabled.`);
      }
    } catch (err) {
      console.warn(
        `Warning: failed to disable prior API key id=${priorApiKeyId}: ${err.message}`
      );
    }
  }
}
__name(cmdKey, "cmdKey");
function mcpSend(resp) {
  process.stdout.write(JSON.stringify(resp) + "\n");
}
__name(mcpSend, "mcpSend");
function mcpError(id, code, message) {
  mcpSend({ jsonrpc: "2.0", id, error: { code, message } });
}
__name(mcpError, "mcpError");
function defaultFlags() {
  return {
    dryRun: false,
    help: false,
    folder: "./memory-vault",
    port: DEFAULT_PORT,
    email: DEFAULT_EMAIL,
    image: DEFAULT_IMAGE,
    publicUrl: null,
    noHub: false,
    noSeed: false,
    hubUrl: DEFAULT_HUB_URL,
    context: 15,
    staleDays: DEFAULT_STALE_DAYS,
    id: null,
    name: null,
    kanban: false,
    kanbanBundle: null,
    embedded: false,
    reranker: false
  };
}
__name(defaultFlags, "defaultFlags");
async function dispatchMcpTool(name, args) {
  function flagsFrom(a) {
    const f = defaultFlags();
    if (typeof a.folder === "string") f.folder = a.folder;
    if (typeof a.port === "number") f.port = a.port;
    if (typeof a.email === "string") f.email = a.email;
    if (typeof a.image === "string") f.image = a.image;
    if (typeof a.publicUrl === "string") f.publicUrl = a.publicUrl;
    if (typeof a.noHub === "boolean") f.noHub = a.noHub;
    if (typeof a.noSeed === "boolean") f.noSeed = a.noSeed;
    if (typeof a.hubUrl === "string") f.hubUrl = a.hubUrl;
    if (typeof a.name === "string") f.name = a.name;
    if (typeof a.kanban === "boolean") f.kanban = a.kanban;
    if (typeof a.kanbanBundle === "string") f.kanbanBundle = a.kanbanBundle;
    if (typeof a.embedded === "boolean") f.embedded = a.embedded;
    if (typeof a.reranker === "boolean") f.reranker = a.reranker;
    return f;
  }
  __name(flagsFrom, "flagsFrom");
  let result;
  switch (name) {
    case "memory_up":
      result = await runUp(flagsFrom(args));
      break;
    case "memory_down":
      result = runDown(flagsFrom(args));
      break;
    case "memory_status":
      result = runStatus(flagsFrom(args));
      break;
    case "memory_logs":
      result = runLogs(flagsFrom(args));
      break;
    case "memory_key":
      result = await runKey(flagsFrom(args));
      break;
    case "memory_daily": {
      const folder = typeof args.folder === "string" ? args.folder : "./memory-vault";
      const context = typeof args.context === "number" ? args.context : 15;
      const text = typeof args.text === "string" ? args.text : "";
      result = runDaily(folder, text, context);
      break;
    }
    case "memory_log": {
      const folder = typeof args.folder === "string" ? args.folder : "./memory-vault";
      const context = typeof args.context === "number" ? args.context : 15;
      const file = typeof args.file === "string" ? args.file : "";
      const text = typeof args.text === "string" ? args.text : "";
      result = runLog(folder, file, text, context);
      break;
    }
    case "memory_bind_hub": {
      const folder = typeof args.folder === "string" ? args.folder : "./memory-vault";
      const hubUrl = typeof args.url === "string" ? args.url : "";
      const hubId = typeof args.id === "string" ? args.id : null;
      result = runHub(hubUrl, folder, hubId, false);
      break;
    }
    case "memory_lint": {
      const folder = typeof args.folder === "string" ? args.folder : "./memory-vault";
      const staleDays = typeof args.staleDays === "number" ? args.staleDays : DEFAULT_STALE_DAYS;
      result = runLint(folder, staleDays);
      break;
    }
    default:
      return {
        content: [{ type: "text", text: `Unknown tool: ${name}` }],
        isError: true
      };
  }
  return {
    content: [{ type: "text", text: result.text }],
    ...result.isError ? { isError: true } : {}
  };
}
__name(dispatchMcpTool, "dispatchMcpTool");
async function startMcpServer() {
  const { createInterface } = await import("node:readline");
  const rl = createInterface({ input: process.stdin, crlfDelay: Infinity });
  for await (const line of rl) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    let msg;
    try {
      msg = JSON.parse(trimmed);
    } catch {
      mcpError(null, -32700, "Parse error");
      continue;
    }
    const id = msg.id ?? null;
    if (!("id" in msg)) continue;
    const method = msg.method;
    if (method === "initialize") {
      mcpSend({
        jsonrpc: "2.0",
        id,
        result: {
          protocolVersion: "2024-11-05",
          capabilities: { tools: {} },
          serverInfo: { name: "memcli", version: "0.1.0" }
        }
      });
    } else if (method === "tools/list") {
      mcpSend({ jsonrpc: "2.0", id, result: { tools: buildToolList() } });
    } else if (method === "tools/call") {
      const params = msg.params ?? {};
      const toolName = typeof params.name === "string" ? params.name : "";
      const toolArgs = params.arguments ?? {};
      try {
        const toolResult = await dispatchMcpTool(toolName, toolArgs);
        mcpSend({ jsonrpc: "2.0", id, result: toolResult });
      } catch (err) {
        mcpSend({
          jsonrpc: "2.0",
          id,
          result: {
            content: [{ type: "text", text: `Error: ${err.message}` }],
            isError: true
          }
        });
      }
    } else {
      mcpError(id, -32601, "Method not found");
    }
  }
}
__name(startMcpServer, "startMcpServer");
var _mainUrl = import.meta.url;
var _argv1Url = process.argv[1] ? (() => {
  try {
    return new URL(
      process.argv[1].startsWith("/") ? "file://" + process.argv[1] : process.argv[1]
    ).href;
  } catch {
    return "";
  }
})() : "";
if (_mainUrl === _argv1Url) {
  const argv = process.argv.slice(2);
  const isTty = process.stdin.isTTY === true;
  if (shouldRunMcp(argv, isTty)) {
    startMcpServer().catch((err) => {
      process.stderr.write(`MCP server error: ${err.message}
`);
      process.exit(1);
    });
  } else {
    const { cmd, flags, positional } = parseArgs(argv);
    if (flags.help) {
      printHelp();
      process.exit(0);
    }
    (async () => {
      try {
        switch (cmd) {
          case "up":
            await cmdUp(flags, flags.dryRun);
            break;
          case "down":
            cmdDown(flags.dryRun, flags.folder, containerName(flags));
            break;
          case "status":
            cmdStatus(containerName(flags));
            break;
          case "logs":
            cmdLogs(containerName(flags));
            break;
          case "key":
            await cmdKey(flags, flags.dryRun);
            break;
          case "daily":
            cmdDaily(flags.folder, positional[0] ?? "", flags.context);
            break;
          case "log":
            cmdLog(flags.folder, positional[0] ?? "", positional[1] ?? "", flags.context);
            break;
          case "hub":
            cmdHub(positional[0] ?? "", flags.folder, flags.id, flags.dryRun);
            break;
          case "lint":
            cmdLint(flags.folder, flags.staleDays);
            break;
          case "open":
            cmdOpen(flags);
            break;
          default:
            console.error(`Unknown subcommand: ${cmd}`);
            printHelp();
            process.exit(1);
        }
      } catch (err) {
        console.error("Error:", err.message);
        process.exit(1);
      }
    })();
  }
}
export {
  EMBED_SERVER_IMAGE,
  KANBAN_RELEASE_URL,
  KANBAN_SENTINEL,
  _plainBlock,
  _stampBlock,
  appendDaily,
  appendLog,
  atomicWrite,
  buildAgentsNote,
  buildDailyEntry,
  buildDataJson,
  buildDockerRunArgs,
  buildEmbedServerRunArgs,
  buildFeaturesJson,
  buildHatLoginUrl,
  buildHeaderNote,
  buildHomeNote,
  buildHubNote,
  buildIndexNote,
  buildKanbanBoard,
  buildLogNote,
  buildSchemaNote,
  buildServerEnv,
  buildToolList,
  containerName,
  embeddingContainerName,
  ensureDailyIndex,
  extractWikilinks,
  findBrokenLinks,
  formatLintReport,
  hubSlug,
  installKanban,
  isPortBusy,
  lintVault,
  networkName,
  parseArgs,
  runDaily,
  runDown,
  runHub,
  runKey,
  runLint,
  runLog,
  runLogs,
  runOpen,
  runStatus,
  runUp,
  shouldRunMcp,
  signHatJwt
};
