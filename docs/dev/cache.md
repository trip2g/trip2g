# Кеширование Telegram API ответов

## Проблема

Telegram API возвращает `FLOOD_WAIT (29)` при частых запросах списка диалогов. Нужен кеш на уровне GraphQL резолвера.

## Решение

**jellydator/ttlcache/v3** — in-memory кеш с generics и автоматическим cleanup.

- TTL 30-60 сек (хватает чтобы не ловить FLOOD_WAIT)
- Кеш-ключ: account ID
- Кеш в резолвере GraphQL — перехватываем до вызова Telegram RPC

```go
import "github.com/jellydator/ttlcache/v3"

// Инициализация (один раз, например в Env или resolver struct)
dialogCache := ttlcache.New[string, []*model.Dialog](
    ttlcache.WithTTL[string, []*model.Dialog](30 * time.Second),
)
go dialogCache.Start() // автоматический cleanup expired записей

// В резолвере
func (r *queryResolver) Dialogs(ctx context.Context, accountID string) ([]*model.Dialog, error) {
    item := dialogCache.Get(accountID)
    if item != nil {
        return item.Value(), nil
    }

    dialogs, err := r.env.ListDialogs(ctx, accountID)
    if err != nil {
        return nil, err
    }

    dialogCache.Set(accountID, dialogs, ttlcache.DefaultTTL)
    return dialogs, nil
}
```

## Почему ttlcache

| Либа | Выбор | Причина |
|------|-------|---------|
| **jellydator/ttlcache/v3** | **выбрана** | generics, auto-cleanup, loader func, активно поддерживается |
| patrickmn/go-cache | нет | не обновляется с 2017, нет generics, `any` везде |
| sync.Map + time | нет | велосипед, ручной cleanup |
| dgraph-io/ristretto | нет | overkill для одного кеша, сложный API |
| SQLite таблица | нет | миграция + сериализация ради 30 сек TTL — лишнее |
| dgraph-io/badger | нет | отдельный data dir, тяжёлая зависимость |
