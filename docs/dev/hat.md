# Hot Auth Token (HAT) — Быстрая авторизация из внешних систем

**Проблема:** Нужна авторизация пользователя в Trip2G из панели хостинга одним кликом, без OAuth flow.

**Решение:** JWT-токен с коротким TTL, который создаётся на стороне панели хостинга и отправляется POST-запросом в Trip2G.

---

## Архитектура

```
Hosting Panel                          Trip2G
     │                                    │
     │  1. Generate JWT token             │
     │     (email + optional ae flag)     │
     │                                    │
     │  2. Auto-submit POST form          │
     ├────────────────────────────────────>│
     │   POST /_system/hat                │
     │   Body: token=eyJhbGc...           │
     │                                    │
     │                            3. Validate JWT
     │                            4. Get/Create User
     │                            5. Make Admin (if ae=true)
     │                            6. Create Session
     │                            7. Set HttpOnly Cookie
     │                                    │
     │  8. Redirect to /                  │
     │<────────────────────────────────────┤
     │   302 Location: /                  │
     │   Set-Cookie: token=...            │
```

---

## Формат токена

JWT токен с payload:

```json
{
  "e": "admin@example.com",   // email пользователя
  "ae": true,                  // admin enter (optional)
  "exp": 1234567890            // unix timestamp (auto, 5 min TTL)
}
```

**Поля:**
- `e` (email) — **обязательное**, email пользователя
- `ae` (admin enter) — **опциональное**, если `true` → пользователь становится админом
- `exp` — автоматически добавляется при генерации (5 минут с момента создания)

---

## Генерация токена

### Go (из Trip2G)

```go
// В cmd/server/main.go уже есть hotAuthTokenManager

token, err := env.GenerateHotAuthToken(ctx, model.HotAuthToken{
    Email:      "user@example.com",
    AdminEnter: false, // обычный пользователь
})

// Для админа:
token, err := env.GenerateHotAuthToken(ctx, model.HotAuthToken{
    Email:      "admin@example.com",
    AdminEnter: true, // создать/сделать админом
})
```

### PHP (из панели хостинга)

```php
<?php
function generateHotAuthToken($email, $isAdmin, $secret) {
    $header = json_encode(['alg' => 'HS256', 'typ' => 'JWT']);
    $payload = json_encode([
        'e' => $email,
        'ae' => $isAdmin,
        'exp' => time() + 300 // 5 минут
    ]);
    
    $base64UrlHeader = str_replace(['+', '/', '='], ['-', '_', ''], base64_encode($header));
    $base64UrlPayload = str_replace(['+', '/', '='], ['-', '_', ''], base64_encode($payload));
    
    $signature = hash_hmac('sha256', $base64UrlHeader . "." . $base64UrlPayload, $secret, true);
    $base64UrlSignature = str_replace(['+', '/', '='], ['-', '_', ''], base64_encode($signature));
    
    return $base64UrlHeader . "." . $base64UrlPayload . "." . $base64UrlSignature;
}

$secret = getenv('HOT_AUTH_TOKEN_SECRET'); // тот же secret что в Trip2G
$token = generateHotAuthToken('admin@example.com', true, $secret);
?>
```

### Python (из панели хостинга)

```python
import jwt
import time
import os

def generate_hot_auth_token(email: str, is_admin: bool = False) -> str:
    secret = os.getenv('HOT_AUTH_TOKEN_SECRET')
    payload = {
        'e': email,
        'ae': is_admin,
        'exp': int(time.time()) + 300  # 5 минут
    }
    return jwt.encode(payload, secret, algorithm='HS256')

token = generate_hot_auth_token('admin@example.com', is_admin=True)
```

---

## Отправка токена в Trip2G

### HTML форма с auto-submit

```html
<!-- Панель хостинга генерирует эту страницу -->
<!DOCTYPE html>
<html>
<head>
    <title>Signing in...</title>
</head>
<body>
    <p>Авторизация...</p>
    <form id="authForm" action="https://trip2g.example.com/_system/hat" method="POST">
        <input type="hidden" name="token" value="<?= $token ?>">
    </form>
    <script>
        document.getElementById('authForm').submit();
    </script>
</body>
</html>
```

### cURL (для тестирования)

```bash
# Сгенерировать токен в Go (добавь в main.go временно):
# token, _ := hotAuthTokenManager.NewToken(model.HotAuthToken{
#     Email: "admin@example.com", 
#     AdminEnter: true,
# })
# fmt.Println(token)

curl -X POST https://trip2g.example.com/_system/hat \
  -d "token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -v
```

---

## Логика обработки токена

**Endpoint:** `POST /_system/hat`  
**Handler:** `internal/case/signinbyhat/endpoint.go`

### Шаги обработки:

1. **Валидация JWT**
   - Проверка подписи с secret
   - Проверка expiration (5 минут)
   - Извлечение email и ae флага

2. **Получение/создание пользователя**
   ```go
   user, err := env.UserByEmail(ctx, email)
   if IsNoFound(err) {
       // Создать нового пользователя
       user, err = env.InsertUserWithEmail(ctx, InsertUserWithEmailParams{
           Email:      email,
           CreatedVia: "hot_auth_token",
       })
   }
   ```

3. **Админ права (если ae=true)**
   ```go
   if hotAuthToken.AdminEnter {
       admin, err := env.AdminByUserID(ctx, user.ID)
       if IsNoFound(err) {
           // Сделать пользователя админом
           env.InsertAdmin(ctx, InsertAdminParams{UserID: user.ID})
       }
   }
   ```

4. **Создание сессии**
   ```go
   env.SetupUserToken(ctx, user.ID) // JWT cookie
   ```

5. **Редирект на главную**
   ```go
   ctx.Redirect("/", http.StatusFound)
   ```

---

## Сценарии использования

### 1. Обычный пользователь (ae=false)

```go
token := generateToken("user@example.com", false)
// → Пользователь авторизуется, НЕ админ
```

**Что происходит:**
- Если пользователь существует → sign in
- Если не существует → создаётся новый пользователь + sign in

### 2. Новый админ (ae=true, пользователя нет)

```go
token := generateToken("admin@example.com", true)
// → Создаётся пользователь + делается админом
```

**Что происходит:**
1. Создаётся `users` запись
2. Создаётся `admins` запись
3. Sign in как админ

### 3. Апгрейд до админа (ae=true, пользователь есть)

```go
token := generateToken("existing@example.com", true)
// → Существующий пользователь становится админом
```

**Что происходит:**
1. Находится существующий пользователь
2. Проверяется наличие `admins` записи
3. Если нет → создаётся
4. Sign in как админ

---

## Конфигурация

### Environment Variables

```bash
# Secret для подписи JWT (должен совпадать в Trip2G и панели хостинга)
HOT_AUTH_TOKEN_SECRET="your-super-secret-key-here"

# TTL токена (по умолчанию 5 минут)
HOT_AUTH_TOKEN_EXPIRES_IN="5m"
```

### Command Line Flags

```bash
./trip2g \
  --hot-auth-token-secret="your-secret" \
  --hot-auth-token-expires-in=5m
```

### Код (internal/appconfig/config.go)

```go
type Config struct {
    // ...
    HotAuthToken hotauthtoken.Config
    // ...
}

// Default: 5 минут
hotAuthTokenDefaults := hotauthtoken.DefaultConfig()
```

---

## Безопасность

### ✅ Что защищает

1. **JWT подпись** — только панель хостинга с секретом может создать валидный токен
2. **Короткий TTL (5 минут)** — украденный токен быстро протухает
3. **POST body** — токен не светится в URL/логах (в отличие от GET параметра)
4. **HttpOnly cookie** — сессия недоступна из JavaScript

### ⚠️ Требования безопасности

1. **HTTPS обязателен** — иначе токен может быть перехвачен
2. **Секрет должен быть сложным** — минимум 32 символа, случайный
3. **Секрет должен быть одинаковым** — в Trip2G и панели хостинга
4. **Не логировать токены** — не выводить в консоль/файлы

### ❌ Чего НЕТ (намеренно упрощено)

- **Single-use токены** — токен можно использовать много раз в течение 5 минут
- **Rate limiting** — нет ограничений на количество попыток
- **Audit log** — не логируется кто когда логинился через HAT

Эти ограничения приемлемы для использования между доверенными системами (панель хостинга ↔ Trip2G).

---

## Тестирование

### Unit тесты

```bash
go test ./internal/case/signinbyhat/ -v
```

**Покрытие:**
- ✅ Невалидный токен → отклонён
- ✅ Существующий пользователь → sign in
- ✅ Существующий админ → sign in
- ✅ Новый пользователь → создание + sign in
- ✅ Новый админ → создание + админ + sign in
- ✅ Апгрейд до админа → добавление прав
- ✅ Ошибки (create user fails, make admin fails, etc.)

### Интеграционное тестирование

1. **Запустить Trip2G:**
   ```bash
   export HOT_AUTH_TOKEN_SECRET="test-secret-key"
   export HOT_AUTH_TOKEN_EXPIRES_IN="5m"
   ./trip2g
   ```

2. **Сгенерировать токен** (в Go коде или онлайн на jwt.io)

3. **Отправить POST запрос:**
   ```bash
   curl -X POST http://localhost:8081/_system/hat \
     -d "token=YOUR_JWT_TOKEN" \
     -v -c cookies.txt
   ```

4. **Проверить редирект:**
   - Должен быть `302 Found` с `Location: /`
   - Должна быть установлена cookie `token=...`

5. **Проверить доступ:**
   ```bash
   curl http://localhost:8081/admin -b cookies.txt
   # Должна открыться админка (если ae=true)
   ```

---

## Отладка

### Проверка токена

```bash
# Декодировать токен (без проверки подписи)
echo "eyJhbGc..." | base64 -d

# Проверить на jwt.io
# Secret: ваш HOT_AUTH_TOKEN_SECRET
```

### Проверка конфига

```bash
# Запустить Trip2G с выводом конфига
./trip2g 2>&1 | grep -i "hot.*auth"
```

### Логи

При ошибках в `/_system/hat` смотри логи Trip2G:

```bash
tail -f /var/log/trip2g/app.log | grep -i "hot\|hat\|auth"
```

**Типичные ошибки:**
- `invalid signature` → разные secrets в панели и Trip2G
- `token is expired` → TTL истёк, увеличь или генерируй заново
- `failed to parse token` → кривой формат JWT

---

## Пример: Интеграция с панелью хостинга

### Кнопка в панели

```html
<!-- hosting-panel/templates/trip2g_login.php -->
<a href="trip2g_auth.php?user_email=<?= urlencode($user->email) ?>&make_admin=1" 
   class="btn btn-primary">
    Войти в Trip2G как админ
</a>
```

### Генератор токена

```php
<?php
// hosting-panel/trip2g_auth.php

require_once 'config.php';

$email = $_GET['user_email'] ?? '';
$makeAdmin = isset($_GET['make_admin']) && $_GET['make_admin'] == '1';

if (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
    die('Invalid email');
}

$secret = TRIP2G_SECRET; // из конфига
$token = generateHotAuthToken($email, $makeAdmin, $secret);

?>
<!DOCTYPE html>
<html>
<head>
    <title>Вход в Trip2G</title>
</head>
<body>
    <p>Авторизация в Trip2G...</p>
    <form id="authForm" action="<?= TRIP2G_URL ?>/_system/hat" method="POST">
        <input type="hidden" name="token" value="<?= htmlspecialchars($token) ?>">
    </form>
    <script>
        document.getElementById('authForm').submit();
    </script>
</body>
</html>
```

### Конфиг панели

```php
<?php
// hosting-panel/config.php

define('TRIP2G_URL', 'https://trip2g.example.com');
define('TRIP2G_SECRET', getenv('HOT_AUTH_TOKEN_SECRET'));

function generateHotAuthToken($email, $isAdmin, $secret) {
    // ... (см. раздел "Генерация токена" выше)
}
```

---

## FAQ

### Можно ли использовать GET вместо POST?

Нет, специально сделан только POST:
- Токен не попадает в логи веб-сервера
- Токен не попадает в browser history
- Токен не передаётся в Referer header

### Почему 5 минут, а не 30 секунд?

- 5 минут — баланс между безопасностью и UX
- Достаточно для медленного интернета
- Достаточно для часовых поясов (clock skew)
- Можно настроить через `--hot-auth-token-expires-in`

### Можно ли отозвать токен досрочно?

Нет, т.к. нет single-use механизма. Токен валиден до истечения TTL.

### Что если пользователь уже залогинен?

Создастся новая сессия, старая останется валидной.

### Можно ли использовать для обычных пользователей (не админов)?

Да, просто не передавай `ae: true`.

---

## См. также

- `internal/case/signinbyhat/` — реализация
- `internal/hotauthtoken/` — JWT manager
- `internal/model/hot_auth_token.go` — модель токена
- `docs/principles.md` — архитектурные принципы
