# Конфигурация JWKS Operator

## Обзор

Конфигурация оператора загружается из файла `config.yaml` при старте. Значения могут быть переопределены через переменные окружения с префиксом `JWKS_OPERATOR_`.

## Структура конфигурации

### Основные параметры

#### `defaultNamespace`
- **Тип**: `string`
- **По умолчанию**: `"example-namespace"`
- **Описание**: Namespace по умолчанию для работы оператора

#### `reconcileInterval`
- **Тип**: `string` (Go duration)
- **По умолчанию**: `"5m"`
- **Описание**: Интервал между реконсиляциями для каждого JWKSConfig ресурса

#### `jwksUpdateInterval`
- **Тип**: `string` (Go duration)
- **По умолчанию**: `"6h"`
- **Описание**: Как часто проверять необходимость обновления JWKS

#### `jwksVerificationInterval`
- **Тип**: `string` (Go duration)
- **По умолчанию**: `"1m"`
- **Описание**: Интервал проверки валидности JWKS через nginx. Оператор периодически проверяет, что JWKS, отдаваемый nginx, может верифицировать JWT токены, подписанные приватным ключом сертификата

#### `maxOldKeys`
- **Тип**: `int`
- **По умолчанию**: `2`
- **Описание**: Максимальное количество старых ключей в JWKS для graceful rotation

#### `defaultOldKeysTTL`
- **Тип**: `string` (Go duration)
- **По умолчанию**: `"720h"` (30 дней)
- **Описание**: Время хранения старых ключей после ротации

#### `defaultUpdateStrategy`
- **Тип**: `string`
- **Возможные значения**: `"rolling"`, `"immediate"`
- **По умолчанию**: `"rolling"`
- **Описание**: Стратегия обновления JWKS

#### `defaultKeepOldKeys`
- **Тип**: `bool`
- **По умолчанию**: `true`
- **Описание**: Сохранять старые ключи для graceful rotation

### Конфигурация окружений

Каждое окружение может иметь свои настройки:

```yaml
environments:
  b2b:
    namespace: "example-namespace"
    reconcileInterval: "5m"
    jwksUpdateInterval: "6h"
    maxOldKeys: 2
    oldKeysTTL: "720h"
    updateStrategy: "rolling"
    keepOldKeys: true
```

### Логирование

```yaml
logging:
  level: "info"              # debug, info, warn, error
  format: "json"             # json, text
  verboseReconcile: false    # детальное логирование реконсиляции
```

### Метрики

```yaml
metrics:
  port: 8080                 # Порт для метрик Prometheus
  path: "/metrics"           # Путь для метрик
  detailed: true             # Детальные метрики
```

### Health Checks

```yaml
health:
  port: 8081                 # Порт для health check
  livenessPath: "/healthz"   # Путь для liveness probe
  readinessPath: "/readyz"   # Путь для readiness probe
```

### Rate Limiting

```yaml
rateLimit:
  maxConfigMapUpdatesPerMinute: 1    # Макс. обновлений ConfigMap/мин
  maxReconcilesPerMinute: 12         # Макс. реконсиляций/мин
  minReconcileInterval: "5s"         # Мин. интервал между реконсиляциями
```

### Retry

```yaml
retry:
  maxAttempts: 5             # Максимальное количество попыток
  initialDelay: "5s"         # Начальная задержка
  maxDelay: "5m"             # Максимальная задержка
  backoffMultiplier: 2.0     # Множитель для экспоненциальной задержки
```

### Кэширование

```yaml
cache:
  enableJWKSCache: true      # Кэширование JWKS
  jwksCacheTTL: "1h"         # TTL для кэша JWKS
  enableCertCache: true       # Кэширование сертификатов
  certCacheTTL: "30m"        # TTL для кэша сертификатов
```

### Верификация JWKS

Настройки для периодической проверки валидности JWKS через nginx:

```yaml
verification:
  timeout: "10s"             # Таймаут для HTTP запросов при верификации
  retryCount: 3              # Количество попыток при неудачной верификации
  retryDelay: "2s"           # Задержка между попытками
  contextTimeout: "30s"      # Таймаут контекста для верификации
```

**Как работает верификация:**
- Оператор периодически (по умолчанию каждую минуту) проверяет, что JWKS, отдаваемый nginx, может верифицировать JWT токены
- Создается тестовый JWT токен, подписанный приватным ключом сертификата
- Токен проверяется с использованием публичного ключа из JWKS, полученного от nginx
- Если верификация не удается, оператор логирует предупреждение, но не прерывает работу (некритичная ошибка)

### Nginx конфигурация

Настройки для nginx Deployment, создаваемого оператором:

```yaml
nginx:
  image: "nginx:1.25-alpine"  # Образ nginx контейнера
  port: 80                     # Порт nginx
  replicas: 1                  # Количество реплик
  cacheMaxAge: 3600            # Cache-Control max-age в секундах (1 час)
  resources:
    requests:
      cpu: "50m"               # CPU request
      memory: "64Mi"           # Memory request
    limits:
      cpu: "200m"              # CPU limit
      memory: "128Mi"          # Memory limit
```

## Переменные окружения

Все параметры конфигурации могут быть переопределены через переменные окружения:

### Формат

`JWKS_OPERATOR_<SECTION>_<KEY>`

Где `<SECTION>` и `<KEY>` преобразуются в UPPER_SNAKE_CASE.

### Примеры

```bash
# Переопределение defaultNamespace
export JWKS_OPERATOR_DEFAULT_NAMESPACE="example-namespace"

# Переопределение reconcileInterval
export JWKS_OPERATOR_RECONCILE_INTERVAL="10m"

# Переопределение уровня логирования
export JWKS_OPERATOR_LOGGING_LEVEL="debug"

# Переопределение порта метрик
export JWKS_OPERATOR_METRICS_PORT="9090"

# Переопределение интервала верификации JWKS
export JWKS_OPERATOR_JWKS_VERIFICATION_INTERVAL="10m"

# Переопределение таймаута верификации
export JWKS_OPERATOR_VERIFICATION__TIMEOUT="15s"

# Переопределение образа nginx
export JWKS_OPERATOR_NGINX__IMAGE="nginx:1.26-alpine"
```

### Вложенные параметры

Для вложенных параметров используйте двойное подчеркивание:

```bash
# Переопределение maxConfigMapUpdatesPerMinute
export JWKS_OPERATOR_RATE_LIMIT__MAX_CONFIG_MAP_UPDATES_PER_MINUTE="2"
```

## Загрузка конфигурации

### Порядок приоритета

1. **Переменные окружения** (наивысший приоритет)
2. **Конфигурация окружения** из `config.yaml`
3. **Значения по умолчанию** из `config.yaml`

### Определение окружения

Окружение определяется через переменную окружения:

```bash
export JWKS_OPERATOR_ENVIRONMENT="b2b"
```

Если не указано, используется `defaultNamespace` для определения окружения.

## Валидация конфигурации

При загрузке конфигурации выполняются следующие проверки:

1. **Проверка формата duration** для временных интервалов
2. **Проверка диапазонов** для числовых значений
3. **Проверка допустимых значений** для enum типов
4. **Проверка обязательных полей**

### Пример ошибки валидации

```
Error: invalid configuration: reconcileInterval must be a valid duration
```

## Использование в коде

### Загрузка конфигурации

```go
import "github.com/jwks-operator/jwks-operator/pkg/config"

// Загрузка из файла
cfg, err := config.Load("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Использование конфигурации
reconcileInterval := cfg.ReconcileInterval
defaultNamespace := cfg.DefaultNamespace
```

### Доступ к конфигурации окружения

```go
// Получение конфигурации для конкретного окружения
envCfg, err := cfg.GetEnvironmentConfig("b2b")
if err != nil {
    log.Fatal(err)
}

namespace := envCfg.Namespace
reconcileInterval := envCfg.ReconcileInterval
```

## Примеры конфигурации

### Минимальная конфигурация

```yaml
defaultNamespace: "example-namespace"
reconcileInterval: "5m"
```

### Полная конфигурация для production

```yaml
defaultNamespace: "example-prod"
reconcileInterval: "10m"
jwksUpdateInterval: "12h"
jwksVerificationInterval: "1m"  # Проверка валидности JWKS каждую минуту
maxOldKeys: 3
defaultOldKeysTTL: "1440h"
defaultUpdateStrategy: "rolling"
defaultKeepOldKeys: true

environments:
  production:
    namespace: "example-prod"
    reconcileInterval: "10m"
    jwksUpdateInterval: "12h"
    jwksVerificationInterval: "1m"
    maxOldKeys: 3
    oldKeysTTL: "1440h"
    updateStrategy: "rolling"
    keepOldKeys: true

logging:
  level: "info"
  format: "json"
  verboseReconcile: false

metrics:
  port: 8080
  path: "/metrics"
  detailed: true

rateLimit:
  maxConfigMapUpdatesPerMinute: 1
  maxReconcilesPerMinute: 12
  minReconcileInterval: "5s"

retry:
  maxAttempts: 5
  initialDelay: "5s"
  maxDelay: "5m"
  backoffMultiplier: 2.0

cache:
  enableJWKSCache: true
  jwksCacheTTL: "1h"
  enableCertCache: true
  certCacheTTL: "30m"

verification:
  timeout: "10s"
  retryCount: 3
  retryDelay: "2s"
  contextTimeout: "30s"

nginx:
  image: "nginx:1.25-alpine"
  port: 80
  replicas: 1
  cacheMaxAge: 3600
  resources:
    requests:
      cpu: "50m"
      memory: "64Mi"
    limits:
      cpu: "200m"
      memory: "128Mi"
```

## Миграция конфигурации

При обновлении оператора проверяйте изменения в структуре конфигурации:

1. **Проверьте changelog** на breaking changes
2. **Обновите config.yaml** согласно новой структуре
3. **Проверьте валидацию** после обновления

## Troubleshooting

### Проблема: Конфигурация не загружается

**Решение**: Проверьте формат YAML и наличие файла `config.yaml`

### Проблема: Переменные окружения не применяются

**Решение**: Убедитесь что используется правильный формат `JWKS_OPERATOR_<SECTION>_<KEY>`

### Проблема: Неверные значения duration

**Решение**: Используйте формат Go duration: `"5m"`, `"1h"`, `"30s"`

