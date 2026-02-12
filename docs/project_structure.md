# Структура проекта JWKS Operator

## Обзор

Проект организован согласно best practices для Kubernetes операторов на базе Kubebuilder.

## Структура директорий

```
jwks/
├── README.md                 # Основная документация оператора
├── .cursorrules              # Правила разработки для Cursor
├── config.yaml               # Конфигурация оператора
│
├── docs/                     # Документация
│   ├── overview.md           # Обзор оператора
│   ├── architecture.md       # Архитектура системы
│   ├── modules.md            # Описание модулей
│   ├── algorithm.md          # Алгоритм работы
│   ├── configuration.md      # Конфигурация
│   ├── project_structure.md  # Структура проекта (этот файл)
│   ├── instructions.md       # Инструкции для AI при разработке
│   └── deployment.md         # Руководство по развертыванию
│
├── api/                      # API определения (CRD)
│   └── v1alpha1/
│       ├── jwks_types.go
│       └── zz_generated.deepcopy.go
│
├── config/                   # Конфигурационные файлы Kubernetes
│   ├── crd/                  # CRD манифесты
│   ├── manager/              # Манифесты менеджера
│   ├── rbac/                 # RBAC манифесты
│   └── samples/              # Примеры ресурсов
│
├── cmd/                      # Точки входа приложения
│   └── manager/
│       └── main.go           # Главный файл оператора
│
├── pkg/                      # Основной код оператора
│   ├── controller/           # Контроллеры
│   │   └── jwksconfig_controller.go
│   │
│   ├── reconciler/           # Реконсиляция
│   │   ├── reconciler.go
│   │   ├── reconciliation_loop.go
│   │   └── status_updater.go
│   │
│   ├── jwks/                 # Генерация JWKS
│   │   ├── generator.go
│   │   ├── certificate_parser.go
│   │   ├── key_extractor.go
│   │   ├── formatter.go
│   │   └── types.go
│   │
│   ├── configmap/            # Управление ConfigMap
│   │   ├── manager.go
│   │   ├── update_strategy.go
│   │   └── key_rotation.go
│   │
│   ├── nginx/                # Управление nginx ресурсами
│   │   ├── manager.go
│   │   ├── config_generator.go
│   │   ├── deployment.go
│   │   ├── service.go
│   │   └── endpoint_config.go
│   │
│   └── config/               # Конфигурация
│       ├── loader.go
│       ├── defaults.go
│       └── types.go
│
├── test/                     # Тесты
│   ├── unit/                 # Unit тесты
│   ├── integration/          # Интеграционные тесты
│   └── e2e/                  # End-to-end тесты
│
├── hack/                     # Вспомогательные скрипты
│   └── generate.sh           # Скрипты генерации кода
│
├── .github/                  # GitHub Actions
│   └── workflows/
│       └── lint.yml          # Линтинг
│
├── Makefile                  # Make команды
├── go.mod                    # Go модули
├── go.sum                    # Go checksums
└── Dockerfile                # Docker образ
```

## Описание модулей

### Документация (`docs/`)

- **overview.md** - Быстрый старт и обзор оператора
- **architecture.md** - Детальная архитектура системы
- **modules.md** - Описание всех модулей оператора
- **algorithm.md** - Алгоритм работы оператора
- **configuration.md** - Настройка и конфигурация

### API (`api/`)

Определения Custom Resource Definitions (CRD) для JWKS.

### Конфигурация (`config/`)

Kubernetes манифесты для развертывания оператора:
- CRD определения
- Deployment оператора
- RBAC правила
- Примеры использования

### Основной код (`pkg/`)

#### Controller (`pkg/controller/`)

Управление жизненным циклом контроллеров и отслеживание изменений ресурсов.

**Файлы**:
- `jwksconfig_controller.go` - основной контроллер для JWKS ресурсов (< 300 строк)

#### Reconciler (`pkg/reconciler/`)

Координация процесса обновления JWKS и управление состоянием.

**Файлы**:
- `reconciler.go` - основной реконсилятор (< 300 строк)
- `reconciliation_loop.go` - цикл реконсиляции (< 200 строк)
- `status_updater.go` - обновление статуса (< 200 строк)

#### JWKS (`pkg/jwks/`)

Генерация JWKS из сертификатов.

**Файлы**:
- `generator.go` - генератор JWKS (< 300 строк)
- `certificate_parser.go` - парсер сертификатов (< 200 строк)
- `key_extractor.go` - извлечение ключей (< 200 строк)
- `formatter.go` - форматирование JSON (< 200 строк)
- `types.go` - типы данных (< 150 строк)

#### ConfigMap (`pkg/configmap/`)

Управление ConfigMap ресурсами с JWKS данными.

**Файлы**:
- `manager.go` - менеджер ConfigMap (< 300 строк)
- `update_strategy.go` - стратегии обновления (< 200 строк)
- `key_rotation.go` - ротация ключей (< 200 строк)

#### Nginx (`pkg/nginx/`)

Управление nginx ConfigMap, Deployment и Service ресурсами.

**Файлы**:
- `manager.go` - менеджер nginx ресурсов (< 300 строк)
- `config_generator.go` - генератор nginx конфигурации (< 200 строк)
- `deployment.go` - управление nginx Deployment (< 300 строк)
- `service.go` - управление nginx Service (< 200 строк)
- `endpoint_config.go` - конфигурация endpoints (< 150 строк)

#### Config (`pkg/config/`)

Управление конфигурацией оператора.

**Файлы**:
- `loader.go` - загрузка конфигурации (< 200 строк)
- `resolver.go` - разрешение окружений (< 150 строк)
- `defaults.go` - значения по умолчанию (< 100 строк)
- `types.go` - типы конфигурации (< 150 строк)

## Правила разработки

### Размер файлов

- **Максимум**: 500 строк на файл
- **Рекомендация**: При превышении 300 строк - разделить на модули
- **Перед кодогенерацией**: Оценить количество добавляемых строк

### Модульность

- Один файл = одна ответственность
- Анализировать функции на возможность выделения в модуль
- Избегать циклических зависимостей

### Конфигурация

- Не хардкодить значения
- Использовать `config.yaml` для всех специфичных значений
- Конфигурация интегрируется при билде через build tags

### Линтинг

- Применять правила из `.github/workflows/lint.yml`
- Запускать линтинг после кодогенерации
- Исправлять все предупреждения перед коммитом

## Сборка и развертывание

### Локальная разработка

```bash
# Установить зависимости
go mod download

# Запустить тесты
go test ./...

# Собрать оператор
go build -o bin/jwks-operator ./cmd/manager

# Запустить локально
make run
```

### Развертывание

```bash
# Установить CRD
make install

# Развернуть оператор
make deploy

# Проверить статус
kubectl get pods -n jwks-operator-system
```

## Тестирование

### Unit тесты

Каждый модуль имеет соответствующий `*_test.go` файл.

### Интеграционные тесты

Тесты взаимодействия модулей в `test/integration/`.

### E2E тесты

End-to-end тесты в `test/e2e/`.

## Документация

Вся документация находится в `docs/`:

1. Начните с `docs/overview.md` для быстрого старта
2. Изучите `docs/architecture.md` для понимания архитектуры
3. Прочитайте `docs/modules.md` для деталей модулей
4. Смотрите `docs/algorithm.md` для алгоритма работы
5. Настройте через `docs/configuration.md`

## Конфигурация

Конфигурация в `config.yaml`:

- Основные параметры оператора
- Настройки для разных окружений
- Параметры логирования и метрик
- Настройки rate limiting и retry

Подробнее в `docs/configuration.md`.

## Статус реализации

✅ **Завершено:**
- Базовая структура проекта
- CRD определения
- Модуль конфигурации
- Модуль генерации JWKS
- Модуль управления ConfigMap
- Модуль управления nginx
- Модуль реконсиляции
- Контроллер
- Главный файл оператора

⏳ **Требуется доработка:**
- Watchers для Certificate и Secret (опционально)
- Полная реализация key rotation с TTL
- Тесты для всех модулей
- Генерация CRD манифестов через `make manifests`
- Дополнительная обработка ошибок

## Следующие шаги

1. Прочитайте `README.md` для общего понимания
2. Изучите `docs/overview.md` для быстрого старта
3. Ознакомьтесь с `docs/instructions.md` перед разработкой
4. Следуйте `.cursorrules` при написании кода
5. Для развертывания см. `docs/deployment.md`

