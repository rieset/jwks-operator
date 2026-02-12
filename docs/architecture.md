# Архитектура JWKS Operator

## Обзор

JWKS Operator - это Kubernetes оператор, построенный на базе [Kubebuilder](https://book.kubebuilder.io/) и controller-runtime. Оператор автоматически управляет JWKS (JSON Web Key Set) конфигурациями при ротации сертификатов JWT, управляемых cert-manager.

## Высокоуровневая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                       │
│                                                              │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │ cert-manager │      │ JWKS Operator│                    │
│  │              │      │              │                    │
│  │ Certificate  │──────│ Controller   │                    │
│  │ Resources    │      │              │                    │
│  └──────────────┘      └──────┬───────┘                    │
│                               │                             │
│                               │ Watches                     │
│                               │                             │
│  ┌──────────────┐      ┌─────▼───────┐                    │
│  │   Secrets    │◄─────│ Reconciler   │                    │
│  │ (Certificates)│      │              │                    │
│  └──────────────┘      └─────┬───────┘                    │
│                               │                             │
│                               │ Updates                     │
│                               │                             │
│  ┌──────────────┐      ┌─────▼───────┐                    │
│  │  ConfigMaps  │◄─────│ JWKS Manager │                    │
│  │  (JWKS)      │      │              │                    │
│  └──────────────┘      └──────────────┘                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Компоненты системы

### 1. Controller Layer

**Назначение**: Отслеживание изменений Kubernetes ресурсов

**Компоненты**:
- `JWKSController` - основной контроллер для JWKS CRD
- `CertificateWatcher` - отслеживание изменений Certificate ресурсов
- `SecretWatcher` - отслеживание изменений Secret ресурсов

**Ответственность**:
- Регистрация watches на ресурсы
- Обработка событий создания/обновления/удаления
- Постановка задач в очередь реконсиляции

### 2. Reconciler Layer

**Назначение**: Координация процесса обновления JWKS

**Компоненты**:
- `JWKSReconciler` - основной реконсилятор
- `ReconciliationLoop` - цикл реконсиляции
- `StatusUpdater` - обновление статуса ресурсов

**Ответственность**:
- Получение текущего состояния ресурсов
- Определение необходимых действий
- Координация обновления ConfigMap
- Обновление статуса операций

### 3. JWKS Generation Layer

**Назначение**: Генерация JWKS из сертификатов

**Компоненты**:
- `JWKSGenerator` - генератор JWKS
- `CertificateParser` - парсер сертификатов
- `KeyExtractor` - извлечение публичных ключей
- `JWKSFormatter` - форматирование JWKS JSON

**Ответственность**:
- Парсинг PEM сертификатов
- Извлечение публичных ключей
- Генерация JWKS структуры
- Поддержка множественных ключей

### 4. ConfigMap Management Layer

**Назначение**: Управление ConfigMap ресурсами

**Компоненты**:
- `ConfigMapManager` - менеджер ConfigMap
- `UpdateStrategy` - стратегии обновления
- `KeyRotationManager` - управление ротацией ключей

**Ответственность**:
- Создание/обновление ConfigMap
- Управление множественными ключами
- Graceful rotation
- Валидация обновлений

### 5. Nginx Config Management Layer

**Назначение**: Управление nginx конфигурацией для JWKS сервера

**Компоненты**:
- `NginxConfigManager` - менеджер nginx конфигурации
- `NginxConfigGenerator` - генератор nginx конфигурации
- `EndpointConfig` - конфигурация endpoints
- `DeploymentManager` - управление nginx Deployment
- `ServiceManager` - управление nginx Service

**Ответственность**:
- Создание/обновление nginx ConfigMap
- Настройка endpoints для раздачи JWKS
- Конфигурация маршрутизации (например, `/.well-known/jwks.json`)
- Интеграция с JWKS ConfigMap
- Управление nginx Deployment и Service

### 6. Verification Layer

**Назначение**: Периодическая проверка валидности JWKS через nginx

**Компоненты**:
- `Verifier` - верификатор JWKS
- `JWKSVerifier` - проверка JWKS от nginx

**Ответственность**:
- Периодическая проверка (каждые 5 минут по умолчанию)
- Получение JWKS от nginx Service
- Создание тестового JWT токена с приватным ключом
- Верификация токена с публичным ключом из JWKS
- Обновление статуса верификации в JWKS ресурсе

### 7. Configuration Layer

**Назначение**: Управление конфигурацией оператора

**Компоненты**:
- `ConfigLoader` - загрузка конфигурации
- `EnvironmentResolver` - разрешение окружений
- `DefaultsProvider` - значения по умолчанию

**Ответственность**:
- Загрузка config.yaml
- Разрешение переменных окружения
- Предоставление конфигурации компонентам

## Поток данных

### Стандартный цикл реконсиляции

```
1. Event Triggered
   └─> Certificate/Secret изменен
       └─> Controller получает событие
           └─> Постановка в очередь реконсиляции

2. Reconciliation Started
   └─> Reconciler.Reconcile() вызван
       └─> Получение JWKS ресурса
           └─> Получение связанного Secret
               └─> Проверка необходимости обновления

3. JWKS Generation
   └─> CertificateParser парсит сертификат
       └─> KeyExtractor извлекает публичный ключ
           └─> JWKSGenerator генерирует JWKS
               └─> Форматирование в JSON

4. ConfigMap Update
   └─> ConfigMapManager получает новый JWKS
       └─> Применение стратегии обновления
           └─> Объединение со старыми ключами (если нужно)
               └─> Обновление ConfigMap с JWKS данными

5. Nginx Config Update
   └─> NginxConfigManager генерирует nginx конфигурацию
       └─> Настройка endpoint для раздачи JWKS (например, `/.well-known/jwks.json`)
           └─> Обновление nginx ConfigMap
               └─> Nginx сервер готов к раздаче публичных ключей

6. JWKS Verification (периодически, каждые 5 минут по умолчанию)
   └─> Verifier проверяет валидность JWKS через nginx
       └─> Получение JWKS от nginx Service
           └─> Создание тестового JWT токена с приватным ключом
               └─> Верификация токена с публичным ключом из JWKS
                   └─> Обновление статуса верификации

7. Status Update
   └─> StatusUpdater обновляет статус JWKS
       └─> Запись времени последнего обновления
           └─> Запись времени последней верификации
               └─> Запись ошибок (если есть)
                   └─> Завершение реконсиляции
```

### Graceful Rotation Flow

```
1. Новый сертификат создан cert-manager
   └─> Secret обновлен с новым сертификатом

2. Operator обнаруживает изменение
   └─> Извлекает новый публичный ключ
       └─> Генерирует новый JWKS entry с новым kid

3. Обновление ConfigMap
   └─> Добавляет новый ключ в JWKS
       └─> Сохраняет старые ключи (если keepOldKeys=true)
           └─> Обновляет ConfigMap с JWKS данными

4. Обновление Nginx Config
   └─> Генерирует nginx конфигурацию для JWKS сервера
       └─> Настраивает endpoint для раздачи публичного ключа
           └─> Обновляет nginx ConfigMap

5. Pods перезапускаются (через Reloader)
   └─> Новые токены подписываются новым ключом
       └─> Старые токены верифицируются старым ключом
       └─> JWKS доступен через HTTP endpoint (например, `/.well-known/jwks.json`)

6. После TTL старых ключей
   └─> Удаление старых ключей из JWKS
       └─> Финальное обновление ConfigMap
```

## Модульная структура

### Пакет `pkg/controller/`

```
controller/
├── jwks_controller.go          # Основной контроллер (< 300 строк)
├── certificate_watcher.go       # Watcher для Certificate (< 200 строк)
├── secret_watcher.go            # Watcher для Secret (< 200 строк)
└── controller_test.go           # Тесты контроллера
```

### Пакет `pkg/reconciler/`

```
reconciler/
├── reconciler.go                # Основной реконсилятор (< 300 строк)
├── reconciliation_loop.go        # Цикл реконсиляции (< 200 строк)
├── status_updater.go            # Обновление статуса (< 200 строк)
└── reconciler_test.go           # Тесты реконсилятора
```

### Пакет `pkg/jwks/`

```
jwks/
├── generator.go                 # Генератор JWKS (< 300 строк)
├── certificate_parser.go        # Парсер сертификатов (< 200 строк)
├── key_extractor.go             # Извлечение ключей (< 200 строк)
├── formatter.go                 # Форматирование JSON (< 200 строк)
├── types.go                     # Типы данных (< 150 строк)
└── generator_test.go            # Тесты генератора
```

### Пакет `pkg/configmap/`

```
configmap/
├── manager.go                   # Менеджер ConfigMap (< 300 строк)
├── update_strategy.go           # Стратегии обновления (< 200 строк)
├── key_rotation.go              # Ротация ключей (< 200 строк)
└── manager_test.go              # Тесты менеджера
```

### Пакет `pkg/nginx/`

```
nginx/
├── manager.go                   # Менеджер nginx ConfigMap (< 300 строк)
├── config_generator.go          # Генератор nginx конфигурации (< 200 строк)
├── endpoint_config.go           # Конфигурация endpoints (< 150 строк)
└── manager_test.go              # Тесты менеджера
```

### Пакет `pkg/config/`

```
config/
├── loader.go                    # Загрузка конфигурации (< 200 строк)
├── resolver.go                  # Разрешение окружений (< 150 строк)
├── defaults.go                  # Значения по умолчанию (< 100 строк)
└── types.go                     # Типы конфигурации (< 150 строк)
```

## Взаимодействие компонентов

### Зависимости между модулями

```
controller/
  └─> reconciler/
        └─> jwks/
        └─> configmap/
        └─> config/
```

**Правила**:
- Controller зависит только от Reconciler
- Reconciler зависит от JWKS, ConfigMap, Nginx, Config
- JWKS, ConfigMap и Nginx независимы друг от друга
- Config используется всеми модулями

### Интерфейсы

Для обеспечения тестируемости используются интерфейсы:

```go
// JWKS Generator Interface
type JWKSGenerator interface {
    GenerateFromCertificate(cert []byte) (*JWKS, error)
    GenerateFromSecret(secret *corev1.Secret) (*JWKS, error)
}

// ConfigMap Manager Interface
type ConfigMapManager interface {
    UpdateJWKS(ctx context.Context, configMapName string, jwks *JWKS) error
    GetJWKS(ctx context.Context, configMapName string) (*JWKS, error)
}

// Nginx Config Manager Interface
type NginxConfigManager interface {
    UpdateConfig(ctx context.Context, configMapName string, jwksConfigMapName string, endpoint string) error
    GenerateConfig(jwksConfigMapName string, endpoint string) (string, error)
}
```

## Масштабирование и производительность

### Оптимизации

1. **Кэширование**: Использование кэша Kubernetes client для чтения ресурсов
2. **Rate Limiting**: Ограничение частоты обновлений ConfigMap
3. **Batching**: Группировка обновлений при множественных изменениях
4. **Requeue Strategy**: Правильная стратегия requeue для избежания лишних циклов

### Ограничения

- Один оператор может обрабатывать до 1000 JWKS ресурсов
- Максимальная частота обновлений: 1 раз в минуту на ConfigMap
- Поддержка до 10 старых ключей в JWKS

## Безопасность

### RBAC

Оператор требует минимальные права:

- `get`, `list`, `watch` на Secrets и ConfigMaps
- `create`, `update`, `patch` на ConfigMaps
- `get`, `list`, `watch`, `update` на JWKS CRD

### Валидация

- Валидация сертификатов перед генерацией JWKS
- Проверка формата JWKS перед обновлением ConfigMap
- Валидация входных данных из CRD
- Периодическая верификация JWKS через nginx (каждые 5 минут)

## Мониторинг и наблюдаемость

### Метрики

Оператор предоставляет метрики Prometheus для мониторинга работы. Все метрики доступны через HTTP endpoint `/metrics` на порту 8080.

**Основные метрики**:
- `jwks_operator_reconcile_total` - счетчик реконсиляций (с меткой `result`)
- `jwks_operator_reconcile_duration_seconds` - длительность реконсиляции (histogram)
- `jwks_operator_configmap_updates_total` - обновления ConfigMap (с метками `type`, `result`)
- `jwks_operator_jwks_generation_total` - генерация JWKS (с меткой `result`)
- `jwks_operator_nginx_operations_total` - операции nginx (с метками `operation`, `result`)
- `jwks_operator_jwks_verification_total` - верификация JWKS (с меткой `result`)
- `jwks_operator_errors_total` - ошибки по типам (с меткой `type`)

**Детальная документация**: См. [docs/metrics.md](metrics.md) для полного описания всех метрик, примеров PromQL запросов, дашбордов Grafana и правил алертинга.

### Логирование

Структурированное логирование через zap logger:
- Уровни: DEBUG, INFO, WARN, ERROR
- Контекстная информация в каждом логе
- Трейсинг через correlation IDs

### Health Checks

- `/health` - проверка здоровья оператора
- `/ready` - проверка готовности к работе
- `/metrics` - метрики Prometheus

