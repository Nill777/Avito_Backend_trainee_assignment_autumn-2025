# Сервис назначения ревьюеров для Pull Request’ов
Выполнено в рамках тестового задания Avito (Backend, Autumn 2025)

## Быстрый старт

1. **Запуск сервиса:**
   Команда соберет приложение, поднимет PostgreSQL, применит миграции и запустит HTTP-сервер
   ```bash
   make docker-up
   ```

2. **Остановка сервиса:**
   Команда останавливает контейнеры и очищает данные БД
   ```bash
   make docker-down
   ```

## Доступ к API

После запуска доступны следующие эндпоинты:

- Swagger UI (визуальный интерфейс): [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- OpenAPI Spec (исходный файл): [http://localhost:8080/openapi.yaml](http://localhost:8080/openapi.yaml)

## Дополнительные задания

### 1. Линтер

Интегрирован статический анализатор **golangci-lint**

Конфигурация с набором правил находится в файле `.golangci.yml`

**Запуск анализатора:**
```bash
make lint
```

## Конфигурация

Настройки задаются через переменные окружения в `docker-compose.yml`:

| Описание | Значение по умолчанию |
| :--- | :--- |
| Порт HTTP сервера | `8080` |
| Подключение к PostgreSQL | `postgres://app:app@db:5432/app?sslmode=disable` |
