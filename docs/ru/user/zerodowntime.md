---
title: "Деплой без простоя"
free: true
lang_redirect: "[[en/user/zerodowntime]]"
---

Выкатка новой версии trip2g не должна терять ни одного чтения. trip2g даёт оркестратору — или даже голому серверу без балансировщика — ровно те health-сигналы и поведение прогрева, чтобы менять версии, пока читатели продолжают получать свои страницы.

## Две health-ручки (нужны ОБЕ)

trip2g отдаёт эти пробники на **внутреннем порту** — *втором* порту (задаётся `--internal-listen-addr`, например `:8081`), отдельном от публичного веб-порта, на котором живёт ваш сайт. Публичные посетители на него не попадают, и — важно — **каждый health-check в рецептах ниже должен целиться в этот internal-порт, а не в публичный веб-порт.** (Сайт на, скажем, `:8080`; `/readyz` отвечает на `:8081`.) Правильно подключить обе ручки — это вся суть; они отвечают на разные вопросы.

### `/livez` — liveness (обязательно)

Возвращает **200, пока процесс способен ответить**, даже во время прогрева или дренажа. Отвечает на «процесс жив?». Оркестратор по нему **рестартит зависший** инстанс. Он **никогда** не должен отдавать 503 на прогреве — иначе оркестратор решит, что прогревающийся (здоровый!) инстанс сломан, и убьёт его → crash-loop, он так и не прогреется.

### `/readyz` — readiness (обязательно)

Возвращает **200 только когда инстанс полностью готов обслуживать** (прогрелся, взял слот писателя). Возвращает **503 на прогреве и при выключении**. Балансировщик и гейт деплоя по нему решают, **слать ли трафик**. Это и держит старую версию в работе, пока новая по-настоящему не готова.

(`/healthz` тоже отдаётся — оставлен для обратной совместимости.)

## Зачем нужен прогрев

На старте trip2g строит in-memory состояние — отрендеренные заметки, поисковый индекс — до того как сможет отдать хоть одну страницу. На маленьком волте это секунды, на большом дольше. Если деплой пошлёт трафик на новый инстанс раньше — читатели получат ошибки. Решение: новый инстанс прогревается, **пока старый обслуживает**, а `/readyz` гейтит переключение.

## Ручки хэндоффа

- `--shutdown-grace-period` — при выключении trip2g переводит `/readyz` в 503 и продолжает обслуживать ещё столько времени перед реальным стопом. Ставьте **≥ окна детекта нездоровья у балансировщика** (например 5с при health-check раз в 1–2с), чтобы балансировщик успел дренировать старый инстанс до его смерти. Слишком мало → несколько `502 Bad Gateway` на переключении.
- `--simple-backup-on-shutdown=false` — пропустить финальный бэкап при rolling-деплое; рядом уже встаёт сменщик (см. [[backup]]).
- `--vacuum-cron` — по умолчанию выключен; так и оставьте (см. [[litestream]]).

## Рецепты

### Managed: Fly.io

Самый простой путь. `fly launch` создаёт `fly.toml`; сопоставьте его health-check с `/readyz` и `/livez`, и деплой Fly гейтится по ним автоматически. Для настоящего zero-downtime SQLite пара к этому — **LiteFS** (read-реплики): bluegreen-стратегия Fly не умеет делить volume, а LiteFS обходит это, давая каждой машине свою реплику. Читайте статьи Fly: [Litestream VFS](https://fly.io/blog/litestream-vfs/), [Introducing LiteFS](https://fly.io/blog/introducing-litefs/), [LiteFS docs](https://fly.io/docs/litefs/), [Seamless deployments](https://fly.io/docs/blueprints/seamless-deployments/).

### Self-hosted: Nomad + Traefik + Consul

Замерено до чистых **100%** (ноль потерянных чтений сквозь rolling canary-деплой). Пять частей:

1. **Consul service discovery** (`provider = "consul"`, не голый Nomad SD) с Consul-`check` на `/readyz` на internal-порту. Каталог гейтится по health, поэтому прогревающийся canary не получает трафик, пока `/readyz` ≠ 200 — убирает все 503 «роут на прогревающийся инстанс».
2. **`--providers.consulcatalog.refreshInterval=5s`** у Traefik. Дефолт — **15с**, слишком медленно: только что убитый бэкенд остаётся в пуле до 15с, это и есть главный источник 502.
3. **Traefik LB active health-check** на `/readyz` (interval 1с, `loadbalancer.healthcheck.port` = internal-порт) — выкидывает дренажащийся бэкенд за ~1с, независимо от обновления каталога.
4. **retry middleware** (`retry.attempts = 4`) — страховка: идемпотентный GET, попавший на только что убитый бэкенд, ретраится на живой → читатель получает 200.
5. **App graceful drain** — `--shutdown-grace-period` ≥ окна детекта (5–6с), Nomad `kill_timeout` чуть выше, чтобы старый инстанс отдавал 200, пока балансировщик его дренажит.

Два rolling-деплоя замерены **100% (11 000 и 12 500 запросов, ноль ошибок)**, p99 ~4ms. Инвариант: *старый инстанс должен отдавать 200, пока Traefik не перестанет на него слать* — крути либо скорость детекта (низкий `refreshInterval` + health-check 1с), либо длину drain (≥ окна обновления); это эквивалентно, а retry-middleware делает 100% устойчивым к таймингу в любом случае.

Целиком, копипаст. Обратите внимание на **два порта** и health-check, нацеленный на internal:

```hcl
# nomad job — group
network {
  port "http"     { to = 8080 }   # публичный сайт
  port "internal" { to = 8081 }   # /livez, /readyz, /healthz
}

update {
  canary           = 1
  auto_promote     = true
  health_check     = "checks"
  min_healthy_time = "5s"
}

service {
  name     = "trip2g"
  port     = "http"
  provider = "consul"
  tags = [
    "traefik.enable=true",
    "traefik.http.routers.t.rule=Host(`example.com`)",
    "traefik.http.routers.t.middlewares=t-retry",
    "traefik.http.middlewares.t-retry.retry.attempts=4",
    "traefik.http.services.t.loadbalancer.healthcheck.path=/readyz",
    "traefik.http.services.t.loadbalancer.healthcheck.port=8081",   # internal-порт
    "traefik.http.services.t.loadbalancer.healthcheck.interval=1s",
  ]
  check {                 # Consul-гейт — тоже на internal-порту
    type     = "http"
    port     = "internal"
    path     = "/readyz"
    interval = "2s"
    timeout  = "1s"
  }
}

task "trip2g" {
  driver       = "docker"
  kill_timeout = "15s"    # должен превышать grace ниже
  env {
    LISTEN_ADDR           = "0.0.0.0:8080"
    INTERNAL_LISTEN_ADDR  = ":8081"
    SHUTDOWN_GRACE_PERIOD = "6s"
  }
}
```

И запустите Traefik с `--providers.consulcatalog.refreshInterval=5s` (единственный не-дефолтный флаг; остальное — в тегах выше).

### Self-hosted: Kubernetes / k3s

Самый чистый 100%, и то, что у большинства корпоратов уже крутится. `Deployment` с rolling-обновлениями, `readinessProbe` на `/readyz` и `livenessProbe` на `/livez`, обе целятся в **internal-порт**:

```yaml
strategy:
  rollingUpdate: { maxUnavailable: 0, maxSurge: 1 }
# ...
readinessProbe:
  httpGet: { path: /readyz, port: 8081 }
  periodSeconds: 2
livenessProbe:
  httpGet: { path: /livez, port: 8081 }
  periodSeconds: 5
```

Kubernetes не шлёт трафик в Pod, пока не пройдёт readiness-проба, а с `maxUnavailable: 0` не убирает старый Pod, пока новый не готов — выкатка гейтится от и до без доп. настройки. Замерено на k3s: **100%** (7 000 запросов, ноль ошибок) сквозь `v1→v2`. Для масштаба чтения / multi-node SQLite — пара с LiteFS (см. [[litestream]]).

### Self-hosted: Caddy (blue-green)

Поднимите два инстанса trip2g (`trip2g@blue` / `trip2g@green`, на двух машинах или двух container-IP) и пропишите оба как upstreams Caddy с active health-check. Поскольку `/readyz` на **internal-порту**, целимся в него через `health_port`:

```
:80 {
	reverse_proxy blue:8080 green:8080 {
		health_uri  /readyz
		health_port 8081
		health_interval 1s
	}
}
```

Caddy шлёт только на upstreams, чей `/readyz` отдаёт 200. Деплой по одному цвету: поднимите новый, дождитесь его `/readyz`, затем `SIGTERM` старому — он переводит `/readyz` в 503 (но ещё отдаёт 200 своё grace-окно), Caddy выкидывает его за ~1с и шлёт на живой цвет. Reload не нужен (а `caddy reload` graceful, если правите Caddyfile). Замерено **100%** (9 000 запросов, ноль ошибок) сквозь blue→green.

### Голый сервер, один порт, БЕЗ балансировщика

trip2g умеет передать слушающий порт от старого процесса новому вообще без прокси — через **SO_REUSEPORT** (новый процесс биндит общий порт только *после* прогрева, так что ядро шлёт на него лишь когда он готов) или передачу fd сокета. Однопортовый путь дал ~99.8%; хвост — in-flight соединения, обрезанные на выходе старого процесса, что убирается мягким дренажом.

## Замеренные цифры

Всё на Hetzner **cpx32** (4 vCPU / 8 ГБ, x86, AMD), один писатель SQLite, vegeta 80–100 rps сквозь rolling-деплой. Полный метод и поран­овые данные — в dev-логе исследования.

| Путь деплоя | чтений сохранено (200) |
|---|---|
| Наивный (прокси не гейтит по health) | ~77 % |
| Traefik active `/readyz` health-check | ~98 % |
| Consul SD + app drain (частично) | ~99 % |
| **Nomad + Traefik + Consul** (полный рецепт выше) | **100 %** |
| **Caddy** (health-gated upstreams) | **100 %** |
| **Kubernetes / k3s** (readiness-gated rollout) | **100 %** |
| SO_REUSEPORT, один порт, без LB | ~99.8 % |

Закономерность: прокси без health-гейта шлёт и на прогревающийся (503), и на умирающий (502) инстанс. Загейтите по `/readyz` (Consul, собственный health-check прокси или k8s readiness) и дайте приложению реальное окно дренажа — и потери чтений падают **в ноль**: все три оркестрированных пути выше дают чистые 100%. Однопортовый SO_REUSEPORT даёт 99.8% (его хвост — in-flight соединения на выходе процесса, не роутинг).
