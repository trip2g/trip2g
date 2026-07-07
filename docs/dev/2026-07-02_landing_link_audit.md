# Landing link audit — trip2g.com (2026-07-02)

## Вывод

Навигация для новичка **скорее нелогичная**. Ссылки почти не битые (2 внутренних 404 на ~140 URL), но главный маршрут сломан по смыслу: hero обещает «Obsidian vault → сайт за минуту», а главная CTA «start →» (7 штук на странице) ведёт на `/simple` — страницу про подключение AI-агента по MCP. Новичок, который пришёл публиковать vault, получает «вставь mcp:add в Claude Code». Топ проблем по убыванию:

1. **CTA «start →» ведёт не туда.** `/simple` — это трюк для агентов, а не старт для человека. Логичный старт (`/en/user/getting_started`) спрятан во второстепенную ссылку.
2. **MCP косяк №1: initialize-нота — про Марка Аврелия.** Эндпоинт `trip2g.com/_system/mcp` на initialize отдаёт агенту системный промпт «You are connected to a self-describing RAG server for Marcus Aurelius… the text of Meditations». А `/simple` обещает «ask it: what is trip2g?». Промпт скопирован с демо-базы Медитаций и не заменён.
3. **MCP косяк №2: `mcp:add …` — несуществующая команда.** «Copy the line above and paste it into Claude Code, Cursor, Codex» — такой команды нет ни в одном клиенте.
4. **4 мёртвых `href="#"` на главной:** «read full →» (philosophy), и три чипа «community ↗», «book consult ↗», «github ↗» в блоке Try Now (текст рядом обещает «book a 30-min consultation»).
5. **Тупик для не-агентных новичков:** `/simple` шлёт их на `/how_to_get_started_with_ai_agents`, где весь контент — «You are not my audience. Sorry.»

Плюс два честных 404: `/team_knowledge_base` (сайдбар доков) и `/ru/user/webhooks` (переключатель языка на странице webhooks).

## Два заявленных косяка — детально

### 1. «start →» → /simple

- **Кнопка:** `start →` (header, hero, pill-блок «mcp:add →», «start now →», subscribe «→ /simple», footer). Все 7 ведут на `https://trip2g.com/simple` (200).
- **Куда попадаешь:** заголовок «Join the network. Your agents will find the rest.», текст: «Yeah. This is it. No animations, no gradients, no mesh graph… you can start right here», дальше одна строка `mcp:add https://trip2g.com/_system/mcp` и «From here on, I'll talk to you through MCP. See you on the other side.»
- **Что не так:** hero главной обещает «Publish your Obsidian vault as a website in under a minute». Человек жмёт «start» и ожидает регистрацию/установку — а получает страницу, где старт возможен только если у него уже есть MCP-клиент. Реальный старт (`/en/user/getting_started` — отличный пошаговый гайд с flowchart) на главной есть, но как второстепенная ссылка «getting started» и «Full guide».
- **Фикс:** «start →» → `/en/user/getting_started` (или на signup simplecloud). `/simple` оставить как отдельную честно подписанную дорожку: «for AI agents →» / «connect your agent →».

### 2. MCP косяк

Два дефекта, оба на маршруте `/simple` → `/_system/mcp`:

**(а) Фейковая команда.** На `/simple` (и на `/simple_v2`):

> `mcp:add https://trip2g.com/_system/mcp`
> «Copy the line above and paste it into Claude Code, Cursor, Codex, or any MCP-aware agent.»

Команды `mcp:add` не существует. Реально:
- Claude Code: `claude mcp add --transport http trip2g https://trip2g.com/_system/mcp`
- Codex: `codex mcp add trip2g …`
- Cursor / Claude Desktop: JSON-конфиг (он, кстати, корректно показан в plain-text ответе GET `https://trip2g.com/_system/mcp` — можно просто на него сослаться).

Вставка строки в чат может сработать как просьба к агенту, но в Cursor вставлять некуда, а новичок без агента вообще не поймёт, что делать.

**(б) Чужая initialize-нота.** POST `initialize` на `https://trip2g.com/_system/mcp` возвращает (verbatim):

> «You are connected to a self-describing RAG server for **Marcus Aurelius**. This server contains the text of Meditations together with commentary… Prefer primary source notes from the Meditations corpus for final evidence.»

При этом `search` по базе возвращает нормальные trip2g-доки (`en/user/protocol.md`, `en/user/okf.md`…). То есть контент правильный, а системная инструкция — от демо-базы Медитаций: агент, которому `/simple` велит спросить «what is trip2g?», получает промпт «ты сервер про Марка Аврелия». **Фикс:** переписать ноту `mcp_method: initialize` на базе trip2g.com — описать trip2g, воронку (getting started, hosting, selfhosted), убрать Meditations.

## Link tree (2 уровня)

Статусы реальные (curl). Вердикт: OK = логично для новичка, ?? = сбивает, XX = сломано.

### Level 1 — главная `https://trip2g.com/` (200)

| Секция | Анкор | URL | Статус | Вердикт |
|---|---|---|---|---|
| header | Docs | /en/user | 200 | OK — «Documentation» hub |
| header | Use cases | /en/user/use_cases | 200 | OK |
| header | Philosophy | /en/thoughts | 200 | OK |
| header | EN / RU | / , /ru | 200 | OK |
| header | ★ star us | github.com/trip2g/trip2g | 200 | OK |
| header | start → | /simple | 200 | ?? — MCP-трюк вместо старта (см. выше) |
| header | ⌘K | # | js | OK — открывает overlay, не битая |
| agent-блок | mcp:add https://trip2g.com/_system/mcp | /_system/mcp | 200 | ?? — человек кликает и видит raw plain-text; для агента норм, для человека сыро |
| hero | start → | /simple | 200 | ?? |
| hero | getting started | /en/user/getting_started | 200 | OK — вот это и есть правильный «start» |
| hero | github ↗ / docs | github, /en/user | 200 | OK |
| body | Full guide | /en/user/getting_started | 200 | OK |
| capabilities | Publish from Obsidian | /en/user/two_way_sync | 200 | OK |
| capabilities | Publish to Telegram | /en/user/telegram | 200 | OK |
| capabilities | Charge for content | /en/user/monetization | 200 | OK |
| capabilities | An AI that answers from your notes | /en/user/mcp | 200 | OK — хорошая дока |
| capabilities | Make it look how you want | /en/user/templates | 200 | OK |
| capabilities | Run it yourself | /en/user/selfhosted | 200 | OK |
| compatibility | claude code ↗ | docs.anthropic.com/claude/docs/claude-code | 301→200 (code.claude.com/docs) | OK, но URL легаси — заменить на code.claude.com/docs |
| compatibility | codex ↗ | github.com/openai/codex | 200 | OK |
| compatibility | other clients | /en/user/mcp | 200 | OK |
| roadmap/philo | read full → | **#** | — | XX — мёртвая, обещает полный текст |
| pill | mcp:add → | /simple | 200 | ?? |
| pill | close tab | # | — | OK как шутка (blue pill) |
| try now | simplecloud.2pub.me → | https://simplecloud.2pub.me | 302→/login 200 | ?? — «try free» приземляет на login-форму без объяснений |
| try now | self-host guide | /en/user/selfhosted | 200 | OK |
| try now | community ↗ | **#** | — | XX — мёртвая |
| try now | book consult ↗ | **#** | — | XX — мёртвая, текст обещает консультацию |
| try now | github ↗ | **#** | — | XX — мёртвая (при живых github-ссылках рядом) |
| try now | start now → | /simple | 200 | ?? |
| subscribe | → /simple | /simple | 200 | ?? |
| pricing | install → / try free → / contact → | selfhosted, simplecloud, mailto | 200 | OK |
| footer | start → | /simple | 200 | ?? |
| footer | /knowledge_graph, /three_zones, /why_not_book, /digital_sovereignty | /en/thoughts/* | 200 | OK |
| footer | /personal-agent | /en/agent | 200 | OK (анкор не совпадает с URL, мелочь) |
| footer | /getting-started, /federation, /telegram, /webhooks | /en/user/* | 200 | OK |
| footer | github / email / @trip2g | ext | 200 | OK |

### Level 2 — со страниц уровня 1

**/simple (200):**

| Анкор | URL | Статус | Вердикт |
|---|---|---|---|
| How to get started with AI agents | /how_to_get_started_with_ai_agents | 200 | XX по смыслу — контент целиком: «You are not my audience. Sorry. Maybe later I will make a funnel for you, but not today.» Тупик ровно для тех, кому эта ссылка адресована |
| simplecloud.2pub.me | https://simplecloud.2pub.me | 302→login | ?? |
| simple_v2 | /simple_v2 | 200 | XX — черновик-огрызок («Tell your agent: mcp:add …»), не должен быть публично залинкован |
| Documentation (nav) | /docs | 200 | OK — алиас /en/user, но URL-разнобой с главной |
| Русский | /ru/simple | 200 | OK |
| thoughts + docs-ссылки в футере | /en/thoughts/*, /en/user/* | 200 | OK |

**Docs-страницы (`/en/user`, getting_started, two_way_sync, telegram, monetization, mcp, templates, selfhosted, federation, webhooks) — общий сайдбар ~70 ссылок:**

- Все 200, **кроме**: `Team knowledge base on a bare VM` → `/team_knowledge_base` → **404** (есть в сайдбаре каждой docs-страницы).
- `/en/user/webhooks` → переключатель «Русский» → `/ru/user/webhooks` → **404** (RU-версии ноты нет).
- `/en/user/getting_started` — сильная страница: flowchart, шаги, предупреждение про тестовый vault. Логичный старт-пункт.
- `/en/user/mcp` — корректные инструкции (frontmatter `mcp_method`, curl-примеры, named entry points). Ссылается на `/en/user/expand`, `/en/user/agent_admin` — обе 200.

**/en/thoughts + 4 эссе, /en/agent, /ru:** все внутренние ссылки 200, контент соответствует анкорам. `/en/agent` («Personal agent — batteries included») — хорошая страница, но с главной в неё ведёт только мелкий футер-линк.

Итого проверено: 134 внутренних URL уровня 2 → 200 везде, кроме двух 404 выше. Внешние: anthropic-линк 301→301→200 (легаси-цепочка), остальные живые.

## Recommended fixes (по убыванию импакта)

Сайт — это trip2g-vault, так что всё чинится правкой нот:

1. **Перенаправить «start →».** В ноте главной (mesh-лендинг) все `start` CTA → `[[en/user/getting_started]]` (или signup simplecloud). `/simple` подписать честно: «for AI agents →». Одна страница — одно обещание.
2. **Переписать initialize-ноту базы trip2g.com** (`mcp_method: initialize`): убрать Marcus Aurelius / Meditations, описать trip2g и типовые вопросы новичка. Это правка одной ноты, а чинит весь заявленный сценарий «ask it: what is trip2g?».
3. **Починить команду на `/simple` и `/simple_v2`:** вместо `mcp:add …` дать per-client блок: `claude mcp add --transport http trip2g https://trip2g.com/_system/mcp`, `codex mcp add trip2g …`, JSON для Cursor/Claude Desktop (готовый текст уже есть в GET `/_system/mcp` — можно переиспользовать).
4. **Убрать/заполнить 4 мёртвых `#` на главной:** «read full →» → на существующее эссе в `/en/thoughts` (или убрать); «community ↗» → t.me/+WuOzft… (ссылка уже есть в футере); «book consult ↗» → mailto или календарь; «github ↗» → репо. Пока ссылки нет — чип не показывать.
5. **`/how_to_get_started_with_ai_agents`:** заменить «You are not my audience» на короткий реальный гайд (поставь Claude Code / Codex, одна команда) или убрать ссылку с `/simple`. Сейчас это пощёчина ровно целевому новичку.
6. **Убрать публичную ссылку на `/simple_v2`** (черновик) с `/simple`; ноту при желании увести из free.
7. **404-мелочи:** создать `/team_knowledge_base` или поправить ссылку сайдбара на `/en/user/team_knowledge_base_mcp`; добавить `docs/ru/user/webhooks.md` или убрать RU-переключатель у ноты (lang frontmatter).
8. **Косметика:** anthropic-линк → `https://code.claude.com/docs` напрямую; «try free» → вести на signup или добавить на login-страницу simplecloud строку «нет аккаунта — регистрация по email»; унифицировать `/docs` vs `/en/user` в навигации.
