const {
  print,
  getNamedType,
  isScalarType,
  isListType,
  isNonNullType,
  Kind,
  GraphQLUnionType,
  GraphQLInputObjectType,
  typeFromAST,
} = require('graphql')

// Парсеры скаляров
const PARSER_SCALAR = {
  String:  '$mol_data_string',
  ID:      '$mol_data_string',
  Int:     '$mol_data_integer',
  Int64:   '$mol_data_integer',
  Float:   '$mol_data_number',
  Time:    '$mol_data_pipe( $mol_data_string , $mol_time_moment )',
  Boolean: '$mol_data_boolean',
}

function snakeCase(s) {
  return s
    .replace(/([a-z0-9])([A-Z])/g,'$1_$2')
    .replace(/[-\s]+/g,'_')
    .toLowerCase()
}

function unwrapNonNull(type) {
  while (isNonNullType(type)) type = type.ofType
  return type
}

/**
 * Генератор парсера для response (ObjectType & Union),
 * __typename обрабатывается через $mol_data_const
 * Опционал для nullable добавляем только если depth>0
 */
function genResponseParser(type, sel, depth = 0) {
  const nullable = !isNonNullType(type)
  const core     = unwrapNonNull(type)

  // UNION
  if (core instanceof GraphQLUnionType) {
    const variants = sel.selections
      .filter(s => s.kind === Kind.INLINE_FRAGMENT && s.typeCondition)
      .map(s => {
        const variantName = s.typeCondition.name.value
        const variantType = core.getTypes().find(t => t.name === variantName)
        return variantType
          ? genResponseParser(variantType, s.selectionSet, depth)
          : '$mol_data_unknown'
      })
      .join(', ')
    const node = `$mol_data_variant(${variants})`
    return (nullable && depth > 0) ? `$mol_data_optional(${node})` : node
  }

  // LIST
  if (isListType(core)) {
    const inner = genResponseParser(core.ofType, sel, depth + 1)
    const arr   = `$mol_data_array(${inner})`
    return (nullable && depth > 0) ? `$mol_data_optional(${arr})` : arr
  }

  // SCALAR
  if (isScalarType(core)) {
    const base = PARSER_SCALAR[core.name] || '$mol_data_unknown'
    return (nullable && depth > 0) ? `$mol_data_optional(${base})` : base
  }

  // OBJECT
  const indent  = '\t'.repeat(depth + 2)
  const outdent = '\t'.repeat(depth + 1)
  const ObjType = getNamedType(core)

  const fields = sel.selections
    .flatMap(s => {
      if (s.kind === Kind.FIELD) return [s]
      if (s.kind === Kind.INLINE_FRAGMENT && s.selectionSet)
        return s.selectionSet.selections.filter(x => x.kind === Kind.FIELD)
      return []
    })
    .map(f => {
      const name = f.name.value
      if (name === '__typename') {
        // используем $mol_data_const с названием типа
        return `${indent}__typename: $mol_data_const('${ObjType.name}')`
      }
      const fld = ObjType.getFields()[name]
      if (!fld) {
        return `${indent}${name}: $mol_data_unknown`
      }
      const inner = genResponseParser(fld.type, f.selectionSet || { selections: [] }, depth + 1)
      const wrap  = isNonNullType(fld.type)
        ? inner
        : `$mol_data_optional(${inner})`
      return `${indent}${name}: ${wrap}`
    })
    .join(',\n')

  const rec = `$mol_data_record({\n${fields}\n${outdent}})`
  return (nullable && depth > 0) ? `$mol_data_optional(${rec})` : rec
}

/** Генератор парсера для variables (InputObjectType) */
function genVariablesParser(type, depth = 0) {
  const nullable = !isNonNullType(type)
  const core     = unwrapNonNull(type)

  // LIST
  if (isListType(core)) {
    const inner = genVariablesParser(core.ofType, depth + 1)
    const arr   = `$mol_data_array(${inner})`
    return nullable ? `$mol_data_optional(${arr})` : arr
  }
  // SCALAR
  if (isScalarType(core)) {
    const base = PARSER_SCALAR[core.name] || '$mol_data_unknown'
    return nullable ? `$mol_data_optional(${base})` : base
  }
  // INPUT OBJECT
  if (core instanceof GraphQLInputObjectType) {
    const indent  = '\t'.repeat(depth + 2)
    const outdent = '\t'.repeat(depth + 1)
    const fields  = Object.values(core.getFields())
      .map(fld => {
        const inner = genVariablesParser(fld.type, depth + 1)
        const wrap  = isNonNullType(fld.type)
          ? inner
          : `$mol_data_optional(${inner})`
        return `${indent}${fld.name}: ${wrap}`
      })
      .join(',\n')
    const rec = `$mol_data_record({\n${fields}\n${outdent}})`
    return nullable ? `$mol_data_optional(${rec})` : rec
  }
  // fallback
  return nullable
    ? `$mol_data_optional($mol_data_unknown)`
    : '$mol_data_unknown'
}

module.exports = {
  plugin(schema, documents) {
    const ops = documents
      .flatMap(d => d.document.definitions)
      .filter(d =>
        d.kind === Kind.OPERATION_DEFINITION &&
        (d.operation === 'query' || d.operation === 'mutation')
      )

    const out = ['namespace $.$$ {', '']

    for (const op of ops) {
      const name = op.name.value
      const snk  = snakeCase(name)

      // 1) QUERY
      out.push(
        `\texport const $trip2g_graphql_${snk}_query = \`${print(op).replace(/`/g,'\\`')}\``,
        ''
      )

      // 2) VARIABLES DTO
      const vDefs = op.variableDefinitions || []
      if (vDefs.length) {
        const fields = vDefs
          .map(v => {
            const inputType = typeFromAST(schema, v.type)
            return `\t\t${v.variable.name.value}: ${genVariablesParser(inputType)}`
          })
          .join(',\n')
        out.push(
          `\texport const $trip2g_graphql_${snk}_variables = $mol_data_record({\n${fields}\n\t})`,
          ''
        )
      }

      // 3) RESPONSE DTO
      const rootType = op.operation === 'query'
        ? schema.getQueryType()
        : schema.getMutationType()
      out.push(
        `\texport const $trip2g_graphql_${snk}_response = ${genResponseParser(
          rootType,
          op.selectionSet
        )}`,
        ''
      )

      // 4) FUNCTION
      const hasVars = vDefs.length > 0
      const args    = hasVars
        ? `(variables: typeof $trip2g_graphql_${snk}_variables.Value)`
        : '()'
      const pass    = hasVars ? ', variables' : ''
      out.push(
        `\texport function $trip2g_graphql_${snk}${args} {`,
        `\t\treturn $trip2g_graphql_${snk}_response(` +
          `$trip2g_graphql_request($trip2g_graphql_${snk}_query${pass})`,
        `\t)`,
        `\t}`,
        ''
      )
    }

    out.push('}')
    return out.join('\n')
  },
}
