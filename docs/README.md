---
title: Документация Редполитики
description: Полная документация — правила, API, развёртывание, плагины
language: ru
weight: 0
---

# Документация Редполитики

| Раздел | Описание |
|--------|----------|
| [Обзор](overview.md) | Архитектура, концепция, стек |
| [Быстрый старт](quickstart.md) | Запуск за 5 минут |
| [Правила](guide-rules.md) | Полный формат YAML-правил — detect/fix деревья |
| [API](guide-api.md) | REST + WebSocket, примеры запросов |
| [Развёртывание](guide-deployment.md) | Docker, конфигурация, переменные окружения |
| [Плагины](guide-plugins.md) | Расширения через gRPC |
| [Рецепты](cookbook.md) | Готовые паттерны правил |

Быстрый старт: `docker compose -f deploy/docker-compose.yml up` → http://localhost:8080

Образ не содержит встроенных правил — монтируйте свои YAML и настройте `RULES_DIR` (см. [развёртывание](guide-deployment.md)).
