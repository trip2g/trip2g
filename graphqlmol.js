const {
  print,
  getNamedType,
  isScalarType,
  isListType,
  isNonNullType,
} = require('graphql')
const { Kind } = require('graphql')

/* ——— Скаляры: TS‑типы и $mol_data‑парсеры ——— */
const TS_SCALAR = {
  String:  'string',
  ID:      'string',
  Int:     'number',
  Int64:   'number',
  Float:   'number',
  Boolean: 'boolean',
}

const PARSER_SCALAR = {
  String:  '$mol_data_string',
  ID:      '$mol_data_string',
  Int:     '$mol_data_integer',
  Int64:   '$mol_data_integer',
  Float:   '$mol_data_number',
  Boolean: '$mol_data_boolean',
}

/* ——— Утилита snake_case ——— */
const snake = s =>
  s.replace(/([a-z0-9])([A-Z])/g, '$1_$2')
   .replace(/[-\s]+/g, '_')
   .toLowerCase()

/* ——— Убираем обёртки NonNull ——— */
function unwrapNonNull(type) {
  let t = type
  while (isNonNullType(t)) {
    t = t.ofType
  }
  return t
}

/* ——— Генератор $mol_data‑парсера с отступами по глубине ——— */
function genParser(type, sel, depth = 0) {
  const nullable = !isNonNullType(type)
  const core     = unwrapNonNull(type)

  // 1) Список
  if (isListType(core)) {
    const inner = genParser(core.ofType, sel, depth + 1)
    const arr   = `$mol_data_array(${inner})`
    return nullable ? `$mol_data_optional(${arr})` : arr
  }

  // 2) Скалар
  if (isScalarType(core)) {
    const base = PARSER_SCALAR[core.name] || '$mol_data_unknown'
    return nullable ? `$mol_data_optional(${base})` : base
  }

  // 3) Объект
  const indent  = '\t'.repeat(depth + 2)
  const outdent = '\t'.repeat(depth + 1)

  const fields = sel.selections
    .filter(s => s.kind === Kind.FIELD)
    .map(s => {
      const fieldName = s.name.value
      // используем исходный type, чтобы взять NonNull или нет
      const fieldDef  = getNamedType(type).getFields()[fieldName]
      const inner     = genParser(fieldDef.type, s.selectionSet || { selections: [] }, depth + 1)
      const wrapped   = isNonNullType(fieldDef.type)
        ? inner
        : `$mol_data_optional(${inner})`
      return `${indent}${fieldName}: ${wrapped}`
    })
    .join(',\n')

  const rec = `$mol_data_record({\n${fields}\n${outdent}})`
  return nullable ? `$mol_data_optional(${rec})` : rec
}

/* ——— Генератор TS‑типа для переменных ——— */
function genVarTs(type) {
  const nullable = !isNonNullType(type)
  const core     = unwrapNonNull(type)

  let t
  if (isListType(core)) {
    t = `${genVarTs(core.ofType)}[]`
  } else {
    t = TS_SCALAR[core.name] || 'any'
  }
  return nullable ? `${t} | undefined` : t
}

/* ——— Плагин ——— */
module.exports = {
  plugin: (schema, docs) => {
    const ops = docs
      .flatMap(d => d.document.definitions)
      .filter(
        d =>
          d.kind === Kind.OPERATION_DEFINITION &&
          (d.operation === 'query' || d.operation === 'mutation'),
      )

    const out = ['namespace $.$$ {', '']

    for (const op of ops) {
      const name  = op.name.value
      const snk   = snake(name)
      const Q     = `$trip2g_graphql_${snk}_query`
      const R     = `$trip2g_graphql_${snk}_response`
      const V     = `$trip2g_graphql_${snk}_variables`
      const F     = `$trip2g_graphql_${snk}`

      // 1) gql-строка
      out.push(`\texport const ${Q} = \`${print(op).replace(/`/g, '\\`')}\``, '')

      // 2) variables
      const vars = op.variableDefinitions ?? []
      if (vars.length) {
        const body = vars
          .map(v => `\t\t${v.variable.name.value}: ${genVarTs(v.type)}`)
          .join(',\n')
        out.push(`\texport type ${V} = {\n${body}\n\t}`, '')
      }

      // 3) response parser
      const root = op.operation === 'query'
        ? schema.getQueryType()
        : schema.getMutationType()
      out.push(`\texport const ${R} = ${genParser(root, op.selectionSet)}`, '')

      // 4) SDK‑функция
      if (vars.length) {
        out.push(
          `\texport const ${F} = (variables: ${V}) =>`,
          `\t\t${R}($trip2g_graphql_request(${Q}, variables))`,
          '',
        )
      } else {
        out.push(
          `\texport const ${F} = () =>`,
          `\t\t${R}($trip2g_graphql_request(${Q}))`,
          '',
        )
      }
    }

    out.push('}')
    return out.join('\n')
  },
}
