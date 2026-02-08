# GraphQL: gqlgen

## Файлы

```
internal/graph/
├── schema.graphqls        # Схема (типы, queries, mutations)
├── schema.resolvers.go    # Реализация резолверов (follow-schema layout)
├── generated.go           # Сгенерированный код (~60K строк)
├── model/models_gen.go    # Сгенерированные модели
├── resolver.go            # Resolver struct
├── handler.go             # HTTP handler
├── helpers.go             # Хелперы
└── config_builders.go     # Config builders
```

## Команды

```bash
make gqlgen          # Перегенерировать Go-код из схемы
make graphqlgen      # gqlgen + TypeScript клиент
```

## Конфигурация (gqlgen.yml)

### Оптимизации генерации

| Флаг | Значение | Зачем |
|------|----------|-------|
| `skip_mod_tidy: true` | Пропускает `go mod tidy` | Быстрее генерация |
| `omit_complexity: true` | Не генерирует ComplexityRoot | −6K строк в generated.go |

### Layout

- **exec**: `single-file` → один `generated.go`
- **resolver**: `follow-schema` → один `{name}.resolvers.go` на файл схемы

Если добавить новый файл схемы (например `admin.graphqls`), gqlgen автоматически создаст `admin.resolvers.go`.

### Autobind

gqlgen автоматически привязывает Go-типы из пакетов:
- `trip2g/internal/model`
- `trip2g/internal/db`

Если тип в схеме совпадает с Go-типом по имени — отдельная модель не генерируется.

## Разделение схемы на файлы

Конфиг уже поддерживает glob: `internal/graph/*.graphqls`. Можно добавлять файлы по доменам.

**Когда это делать:** для удобства навигации, когда схема вырастет за 3-4K строк.

**Не ускорит компиляцию:** Go компилирует на уровне пакета, а не файла. Пакет `graph` останется тем же размером.

## Complexity

`omit_complexity: true` — complexity estimation отключена. Если понадобится для отдельных полей, использовать whitelist через кастомный код, а не генерацию.
