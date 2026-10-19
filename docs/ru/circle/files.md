---
title: "Flat-file CMS без контента внутри"
label: "flat-file CMS"
order: 9
motion_visualization: files
link: "[[ru/thoughts/flat-file-cms]]"
related:
  - "[[ru/thoughts/contentless-cms]]"
  - "[[ru/thoughts/markdown-operating-system]]"
  - "[[ru/user/self-hosted-wiki-sqlite]]"
  - "[[ru/user/docs-image]]"
layout: mesh/circle
lang: ru
free: true
noindex: true
lang_redirect: "[[en/circle/files]]"
---

Контент лежит в markdown-файлах в вашей папке, а не в базе CMS. Сервер рендерит их на лету: один бинарник на Go и один файл SQLite, без отдельного сервера базы данных и без сборки. Поправили файл — страница изменилась; ушли — забрали все файлы с собой. Целую базу можно даже собрать в один контейнер.
