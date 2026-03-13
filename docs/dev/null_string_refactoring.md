# Рефакторинг: sql.Null* → указатели

## Цель

Убрать sql.NullString, sql.NullInt64, sql.NullTime из бизнес-логики. Nullable поля становятся указателями (`*string`, `*int64`, `*time.Time`).

## Изменения в sqlc.yaml

```yaml
emit_pointers_for_null_types: true
```

Добавлено в ОБА блока (read и write queries).

## Результат

- Удаляется слой конверсии между БД и бизнес-логикой
- gqlgen работает с указателями напрямую без кастомных резолверов
- Код становится идиоматичнее — `*T` это стандартный Go-способ представления optional значений
- Меньше бойлерплейта и импортов `database/sql` вне репозиториев

## Паттерны замены

### Проверка на null
```go
// Было
if obj.UserID.Valid {
    doSomething(obj.UserID.Int64)
}

// Стало
if obj.UserID != nil {
    doSomething(*obj.UserID)
}
```

### Создание nullable значения
```go
// Было
params := db.SomeParams{
    UserID: sql.NullInt64{Int64: userID, Valid: true},
    Email:  sql.NullString{String: email, Valid: true},
}

// Стало
params := db.SomeParams{
    UserID: &userID,
    Email:  &email,
}

// Или через хелпер (для литералов)
params := db.SomeParams{
    UserID: ptr.To(int64(123)),
    Email:  ptr.To("test@example.com"),
}
```

### Хелпер ptr.To
```go
// internal/ptr/ptr.go
package ptr

func To[T any](v T) *T {
    return &v
}
```

### resolveOnePtr для GraphQL резолверов
```go
// internal/graph/helpers.go
func resolveOnePtr[T any, K any](
    ctx context.Context,
    id *K,
    fetch func(context.Context, K) (T, error),
) (*T, error) {
    if id == nil {
        return nil, nil
    }
    return resolveOne(ctx, *id, fetch)
}

// Использование - было
func (r *resolver) User(ctx context.Context, obj *db.Purchase) (*db.User, error) {
    if obj.UserID == nil {
        return nil, nil
    }
    return resolveOne[db.User](ctx, *obj.UserID, r.env(ctx).UserByID)
}

// Использование - стало
func (r *resolver) User(ctx context.Context, obj *db.Purchase) (*db.User, error) {
    return resolveOnePtr[db.User](ctx, obj.UserID, r.env(ctx).UserByID)
}
```

## Что нужно исправить

### Файлы с ошибками компиляции (после `go build ./...`)

```
internal/case/admin/completetelegramaccountauth/resolve.go
internal/case/admin/creategittoken/resolve.go
internal/case/admin/createhtmlinjection/resolve.go
internal/case/admin/createoffer/resolve.go
internal/case/admin/createrelease/resolve.go
internal/case/admin/deleteboostycredentials/resolve.go
internal/case/admin/deletepatreoncredentials/resolve.go
internal/case/admin/disableapikey/resolve.go
internal/case/admin/disablegittoken/resolve.go
internal/case/admin/revokeusersubgraphaccess/resolve.go
internal/case/admin/signouttelegramaccount/resolve.go
internal/case/admin/updatehtmlinjection/resolve.go
internal/case/admin/updatenotegraphpositions/resolve.go
internal/case/admin/updateoffer/resolve.go
internal/case/admin/updatesubgraph/resolve.go
internal/case/admin/updatetelegramaccount/resolve.go
internal/case/admin/updatetgbot/resolve.go
internal/case/admin/updateuser/resolve.go
internal/case/admin/updateusersubgraphaccess/resolve.go
internal/case/backjob/sendtelegramaccountmessage/resolve.go
internal/case/backjob/sendtelegrammessage/resolve.go
internal/case/createemailwaitlistrequest/resolve.go
internal/case/cronjob/refreshtelegramaccounts/resolve.go
internal/case/hidenotes/resolve.go
internal/case/processnotionwebook/resolve.go
internal/case/processnowpaymentsipn/resolve.go
internal/case/refreshboostydata/resolve.go
internal/case/refreshboostytoken/resolve.go
internal/case/refreshpatreondata/resolve.go
internal/case/sendtelegramaccountpublishpost/resolve.go
internal/case/sendtelegrampublishpost/resolve.go
internal/cronjobs/jobs.go
internal/graph/schema.resolvers.go (несколько мест)
```

### Тестовые файлы (после `go test ./...`)

Все тесты, которые создают структуры с sql.Null* полями.

## Уже исправлено

- `internal/patreonjobs/jobs.go`
- `internal/boostyjobs/jobs.go`
- `internal/case/getboostyuser/resolve.go`
- `internal/case/signinbypurchasetoken/resolve.go`
- `internal/case/cronjob/removeexpiredtgchatmembers/resolve.go`
- `internal/case/processpatreonwebhook/resolve.go`
- `internal/case/createpaymentlink/resolve.go`
- `internal/case/signinbyemail/resolve.go`
- `internal/case/getpatreonuser/resolve.go`
- `internal/case/handletgupdate/resolve.go` и `access.go`
- `internal/graph/schema.resolvers.go` (частично)
- `internal/graph/helpers.go` (добавлен resolveOnePtr)

## Функции db.ToNullable*

Эти функции (`db.ToNullableInt64`, `db.ToNullableTime`, etc.) больше не нужны в большинстве случаев — просто передавайте указатель напрямую.

```go
// Было
params.CreatedAtGte = db.ToNullableTime(filter.CreatedAt.Gte)

// Стало
params.CreatedAtGte = filter.CreatedAt.Gte  // уже *time.Time
```

## Подозрительные места для проверки

### Общие паттерны

1. Места где nullable поле используется в логе — nil pointer dereference
2. Сравнения типа `obj.Field.Int64 == someValue` — теперь нужно `*obj.Field == someValue` с проверкой на nil
3. Функции nullableString, nullableBool в updatetelegramaccount/updatetgbot — нужно переписать

### Конкретные файлы

#### `internal/case/cronjob/removeexpiredtgchatmembers/resolve.go:134`
Логирование `user.TgUserID.Int64` когда `TgUserID` может быть nil. Исправлено на pointer, но нужно проверить что логика не сломалась — особенно в случае когда user есть, но TgUserID == nil.

#### `internal/patreonjobs/jobs.go:72` и `internal/boostyjobs/jobs.go:60`
Логирование `cred.SyncedAt.Time` — после исправления на pointer передаётся `lastSync` переменная. Проверить что логгер корректно обрабатывает zero time.

#### `internal/graph/schema.resolvers.go:1827`
```go
if !data.X.Valid || !data.Y.Valid {
```
Это проверка координат для note graph positions. Нужно заменить на:
```go
if data.X == nil || data.Y == nil {
```
И ниже `data.X.Float64` на `*data.X`.

#### `internal/graph/schema.resolvers.go:2131`
```go
if !obj.BannedBy.Valid {
```
Проверка забаненного пользователя. Заменить на `obj.BannedBy == nil`.

#### `internal/case/admin/updatetelegramaccount/resolve.go`
Функции `nullableString`, `nullableBoolToInt64` возвращают `sql.NullString` и `sql.NullInt64`. Нужно переписать:
```go
// Было
func nullableString(s *string) sql.NullString {
    if s == nil {
        return sql.NullString{}
    }
    return sql.NullString{String: *s, Valid: true}
}

// Стало — функция больше не нужна, просто используй s напрямую
params.DisplayName = input.DisplayName  // уже *string
```

#### `internal/case/admin/updatetgbot/resolve.go`
Аналогично — функции `nullableString`, `nullableBool` нужно убрать и передавать указатели напрямую.

#### `internal/case/refreshpatreondata/resolve.go:217`
```go
currentTierID := sql.NullInt64{}
// ...
currentTierID = sql.NullInt64{Int64: tier.ID, Valid: true}
```
Нужно заменить на:
```go
var currentTierID *int64
// ...
currentTierID = &tier.ID
```

#### `internal/case/refreshboostydata/resolve.go:101`
Аналогичная ситуация с `sql.NullInt64` — заменить на `*int64`.

#### `internal/cronjobs/jobs.go:208, 261-262`
Создание `sql.NullString` для error message и report data. Заменить на указатели:
```go
// Было
ErrorMessage: sql.NullString{String: errMsg, Valid: true}

// Стало
ErrorMessage: &errMsg
```

### Тесты

Все тестовые файлы используют старый синтаксис. Основные:
- `internal/case/admin/createhtmlinjection/resolve_test.go`
- `internal/case/admin/createoffer/resolve_test.go`
- `internal/case/admin/createrelease/resolve_test.go`
- `internal/case/admin/createuser/resolve_test.go`
- `internal/case/admin/deleteboostycredentials/resolve_test.go`
- `internal/case/admin/deletepatreoncredentials/resolve_test.go`
- `internal/case/admin/disableapikey/resolve_test.go`
- `internal/case/admin/resettelegrampublishnote/resolve_test.go`
- `internal/case/admin/restoreboostycredentials/resolve_test.go`

В тестах паттерн замены тот же:
```go
// Было
want := db.SomeStruct{
    UserID: sql.NullInt64{Int64: 123, Valid: true},
}

// Стало
want := db.SomeStruct{
    UserID: ptr.To(int64(123)),
}
```
