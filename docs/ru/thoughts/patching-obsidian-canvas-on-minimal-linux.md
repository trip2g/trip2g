---
title: "Чиним Obsidian Canvas на минимальном Linux"
order: 15
layout: trip2g/content-first
free: true
lang_redirect: "[[en/thoughts/patching-obsidian-canvas-on-minimal-linux]]"
---

Ubuntu 24.04 LTS aarch64, VMware на Mac, без десктопного окружения — dwm под голым X11, `startx` из TTY. Obsidian 1.12.7 (проверено и Flatpak, и AppImage).

Canvas открывается. Ноды рендерятся. Клик работает. Но перетаскивать существующие ноды нельзя, и рёбра между ними не рисуются. Новые ноды, созданные через тулбар, тащатся нормально. Консоль молчит.

**Решение: плагин из двух файлов, который переписывает все не-мышиные pointer-события в `pointerType: "mouse"` до того, как их увидят обработчики Obsidian.** Установить, перезапустить — готово. Ниже — корневая причина и тупики, если интересно понять, почему так.

---

## Что не помогло

- Переустановка AppImage, переход на Flatpak — то же самое.
- Форс X11-бэкенда: `GDK_BACKEND=x11 ELECTRON_OZONE_PLATFORM_HINT=x11` — без эффекта.
- Community-плагин "Canvas Drag Fix" — не помог. Объясним почему ниже.
- udev-правило с `ID_INPUT_TABLET=0` `ID_INPUT_MOUSE=1` — тоже не помогло. Chromium читает XInput2 напрямую; udev-свойства до него не доходят.

## Как нашли причину

Вставьте это в DevTools Obsidian (Ctrl+Shift+I → Console) и потяните ноду:

```js
['pointerdown','pointermove','pointerup'].forEach(name =>
  window.addEventListener(name, e =>
    console.log(name, 'type:', e.pointerType, 'id:', e.pointerId), true));
```

Вывод:

```
pointerdown  type: mouse  id: 1
pointermove  type: pen    id: 4
pointermove  type: pen    id: 4
pointerup    type: mouse  id: 1
```

Та же физическая мышь. `pointerdown` и `pointerup` помечены как `"mouse"`. Каждый `pointermove` между ними — как `"pen"` с другим `pointerId`.

Обработчик перетаскивания в Canvas начинает drag на `pointerdown`, захватывает указатель, а затем ожидает, что следующие `pointermove` совпадут по `pointerType` и `pointerId`. Они не совпадают. Drag молча отклоняется каждый раз.

Причина — классификация устройств в Chromium через XInput2. Виртуальная мышь VMware выставляет стилусные оси (абсолютные координаты, давление), поэтому Chromium маркирует её события движения как перо. `pointerdown` и `pointerup` идут через другой путь и остаются мышью.

**Почему "Canvas Drag Fix" не помогает:** тот плагин перехватывает `pointerdown` с `pointerType === 'pen'` и повторно отправляет их как мышиные события. Наш `pointerdown` уже `"mouse"` — проблема в `pointermove`, которого плагин не касается.

## Решение

Небольшой плагин, который вешает capture-фазный listener на уровне `window` и переписывает любой не-`"mouse"` `pointerType` в `"mouse"` (и нормализует `pointerId` до `1`) раньше любого другого обработчика. Охватывает все десять типов pointer-событий.

### Установка

1. Создайте директорию `<vault>/.obsidian/plugins/pointer-fix/`
2. Сохраните в неё два файла ниже
3. Откройте `<vault>/.obsidian/community-plugins.json` и добавьте `"pointer-fix"` в массив (если файла нет — создайте: `["pointer-fix"]`)
4. Если community-плагины ещё не включены: Настройки → Community plugins → «Turn on community plugins»
5. Перезапустите Obsidian
6. Откройте DevTools → Console: должна появиться строка `[pointer-fix] installed`

Перетаскивание в Canvas работает. Рёбра рисуются.

### `main.js`

```js
'use strict';

const obsidian = require('obsidian');

const EVENTS = [
	'pointerdown', 'pointermove', 'pointerup', 'pointercancel',
	'pointerover', 'pointerout', 'pointerenter', 'pointerleave',
	'gotpointercapture', 'lostpointercapture'
];

function fixPointer(ev) {
	if (ev.pointerType !== 'mouse') {
		try { Object.defineProperty(ev, 'pointerType', { value: 'mouse', configurable: true }); } catch (e) {}
		try { Object.defineProperty(ev, 'pointerId', { value: 1, configurable: true }); } catch (e) {}
	}
}

class PointerFixPlugin extends obsidian.Plugin {
	async onload() {
		for (const name of EVENTS) {
			window.addEventListener(name, fixPointer, { capture: true });
		}
		console.log('[pointer-fix] installed');
	}

	onunload() {
		for (const name of EVENTS) {
			window.removeEventListener(name, fixPointer, { capture: true });
		}
	}
}

module.exports = PointerFixPlugin;
```

### `manifest.json`

```json
{
	"id": "pointer-fix",
	"name": "Pointer Type Fix",
	"version": "0.1.0",
	"minAppVersion": "1.0.0",
	"description": "Rewrites non-mouse pointerType to 'mouse'. Workaround for Chromium misclassifying VMware/virtual mice as pen, which breaks Canvas drag.",
	"author": "alexes",
	"isDesktopOnly": true
}
```

## AppImage и `--no-sandbox`

На некоторых минимальных Linux-сетапах — ядра или дистрибутивы, ограничивающие непривилегированные user namespaces — AppImage Obsidian не запускается. Процесс падает сразу с ошибкой sandbox. Обходной путь:

```bash
~/Obsidian.AppImage --no-sandbox
```

Это отключает песочницу рендер-процесса Chromium. На личной dev-VM это приемлемый компромисс. На общей или публично доступной машине — нет.

## Важно

Если на машине есть настоящий стилус или планшет Wacom, этот плагин скроет специфичные для пера данные — давление, наклон, идентификатор указателя. Отключите его перед рисованием.

---

Когда Chromium-приложение ведёт себя странно на Linux — логируйте каждое pointer-событие на capture-фазе: правда в классификации устройства, а не в слоях выше.
