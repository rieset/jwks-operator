# Модули JWKS Operator

## Обзор модулей

Оператор разделен на логические модули, каждый из которых отвечает за определенную функциональность. Все модули следуют принципу единственной ответственности (SRP).

## 1. Controller Module (`pkg/controller/`)

### Назначение

Управление жизненным циклом контроллеров и отслеживание изменений Kubernetes ресурсов.

### Компоненты

#### `jwksconfig_controller.go` (< 300 строк)

Основной контроллер для JWKSConfig CRD ресурсов.

**Ответственность**:
- Регистрация контроллера с manager
- Настройка watches на связанные ресурсы
- Обработка событий от Kubernetes API

**Основные функции**:
```go
func (r *JWKSConfigReconciler) SetupWithManager(mgr ctrl.Manager) error
func (r *JWKSConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```

Оператор использует стандартный механизм controller-runtime для отслеживания изменений ресурсов через watches, настроенные в `SetupWithManager`.

### Зависимости

- `pkg/reconciler/` - для выполнения реконсиляции
- `pkg/config/` - для получения конфигурации

## 2. Reconciler Module (`pkg/reconciler/`)

### Назначение

Координация процесса обновления JWKS и управление состоянием ресурсов.

### Компоненты

#### `reconciler.go` (< 300 строк)

Основной реконсилятор для JWKSConfig ресурсов.

**Ответственность**:
- Получение текущего состояния ресурсов
- Определение необходимых действий
- Координация обновления JWKS
- Обработка ошибок и retry логики

**Основные функции**:
```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
func (r *Reconciler) reconcileJWKSConfig(ctx context.Context, jwksConfig *jwksv1alpha1.JWKSConfig) error
```

#### `reconciliation_loop.go` (< 200 строк)

Цикл реконсиляции с обработкой различных сценариев.

**Ответственность**:
- Определение типа операции (create/update/delete)
- Выполнение последовательности шагов реконсиляции
- Управление retry логикой

**Основные функции**:
```go
func (l *ReconciliationLoop) Execute(ctx context.Context, jwksConfig *jwksv1alpha1.JWKSConfig) error
func (l *ReconciliationLoop) shouldReconcile(jwksConfig *jwksv1alpha1.JWKSConfig) bool
```

#### `status_updater.go` (< 200 строк)

Обновление статуса JWKSConfig ресурсов.

**Ответственность**:
- Обновление условий (conditions) ресурса
- Запись времени последнего обновления
- Запись ошибок и предупреждений

**Основные функции**:
```go
func (u *StatusUpdater) UpdateStatus(ctx context.Context, jwksConfig *jwksv1alpha1.JWKSConfig, status *jwksv1alpha1.JWKSConfigStatus) error
func (u *StatusUpdater) SetCondition(jwksConfig *jwksv1alpha1.JWKSConfig, conditionType string, status metav1.ConditionStatus, reason, message string)
```

### Зависимости

- `pkg/jwks/` - для генерации JWKS
- `pkg/configmap/` - для обновления ConfigMap
- `pkg/config/` - для получения конфигурации

## 3. JWKS Generation Module (`pkg/jwks/`)

### Назначение

Генерация JWKS (JSON Web Key Set) из сертификатов.

### Компоненты

#### `generator.go` (< 300 строк)

Основной генератор JWKS из сертификатов.

**Ответственность**:
- Координация процесса генерации JWKS
- Интеграция компонентов парсинга и форматирования
- Поддержка множественных ключей

**Основные функции**:
```go
func (g *Generator) GenerateFromCertificate(cert []byte) (*JWKS, error)
func (g *Generator) GenerateFromSecret(secret *corev1.Secret) (*JWKS, error)
func (g *Generator) MergeJWKS(oldJWKS, newJWKS *JWKS) (*JWKS, error)
```

#### `certificate_parser.go` (< 200 строк)

Парсинг PEM сертификатов.

**Ответственность**:
- Декодирование PEM формата
- Парсинг X.509 сертификатов
- Валидация сертификатов

**Основные функции**:
```go
func ParseCertificate(pemData []byte) (*x509.Certificate, error)
func ParseCertificateFromSecret(secret *corev1.Secret) (*x509.Certificate, error)
func ValidateCertificate(cert *x509.Certificate) error
```

#### `key_extractor.go` (< 200 строк)

Извлечение публичных ключей из сертификатов.

**Ответственность**:
- Извлечение публичного ключа
- Определение типа ключа (RSA, ECDSA, etc.)
- Генерация kid (Key ID) из сертификата

**Основные функции**:
```go
func ExtractPublicKey(cert *x509.Certificate) (interface{}, error)
func ExtractRSAKey(cert *x509.Certificate) (*rsa.PublicKey, error)
func GenerateKeyID(cert *x509.Certificate) (string, error)
```

#### `formatter.go` (< 200 строк)

Форматирование ключей в JWKS JSON формат.

**Ответственность**:
- Конвертация ключей в JWK формат
- Генерация правильного JSON
- Поддержка различных алгоритмов

**Основные функции**:
```go
func FormatJWK(key interface{}, kid string) (*JWK, error)
func FormatRSAKey(key *rsa.PublicKey, kid string) (*JWK, error)
func ToJSON(jwks *JWKS) ([]byte, error)
```

#### `types.go` (< 150 строк)

Определение типов данных для JWKS.

**Типы**:
```go
type JWKS struct {
    Keys []JWK `json:"keys"`
}

type JWK struct {
    Kty string `json:"kty"`
    Use string `json:"use"`
    Kid string `json:"kid"`
    // ... другие поля
}
```

### Зависимости

- Стандартные библиотеки: `crypto/x509`, `crypto/rsa`
- Внешние зависимости: минимальные

## 4. ConfigMap Management Module (`pkg/configmap/`)

### Назначение

Управление ConfigMap ресурсами с JWKS данными.

### Компоненты

#### `manager.go` (< 300 строк)

Менеджер для работы с ConfigMap ресурсами.

**Ответственность**:
- Создание/обновление ConfigMap
- Получение текущего состояния ConfigMap
- Валидация данных перед обновлением

**Основные функции**:
```go
func (m *Manager) UpdateJWKS(ctx context.Context, namespace, configMapName string, jwks *jwks.JWKS) error
func (m *Manager) GetJWKS(ctx context.Context, namespace, configMapName string) (*jwks.JWKS, error)
func (m *Manager) CreateConfigMap(ctx context.Context, namespace, configMapName string, jwks *jwks.JWKS) error
```

#### `update_strategy.go` (< 200 строк)

Стратегии обновления ConfigMap.

**Ответственность**:
- Определение стратегии обновления (rolling, immediate)
- Применение стратегии graceful rotation
- Управление версионированием

**Основные функции**:
```go
func (s *UpdateStrategy) Apply(ctx context.Context, manager *Manager, configMapName string, newJWKS *jwks.JWKS) error
func (s *UpdateStrategy) ShouldUpdate(oldJWKS, newJWKS *jwks.JWKS) bool
```

#### `key_rotation.go` (< 200 строк)

Управление ротацией ключей в JWKS.

**Ответственность**:
- Добавление новых ключей
- Удаление устаревших ключей
- Управление TTL старых ключей

**Основные функции**:
```go
func (r *KeyRotationManager) AddNewKey(jwks *jwks.JWKS, newKey *jwks.JWK) error
func (r *KeyRotationManager) RemoveExpiredKeys(jwks *jwks.JWKS, ttl time.Duration) error
func (r *KeyRotationManager) ShouldKeepOldKeys(jwksConfig *jwksv1alpha1.JWKSConfig) bool
```

### Зависимости

- `pkg/jwks/` - для работы с JWKS структурами
- Kubernetes client для работы с ConfigMap

## 5. Nginx Config Management Module (`pkg/nginx/`)

### Назначение

Управление nginx конфигурацией для JWKS сервера, который раздает публичные ключи через HTTP endpoint.

### Компоненты

#### `manager.go` (< 300 строк)

Менеджер для работы с nginx ConfigMap, Deployment и Service ресурсами.

**Ответственность**:
- Создание/обновление nginx ConfigMap
- Управление nginx Deployment
- Управление nginx Service
- Интеграция с JWKS ConfigMap

**Основные функции**:
```go
func (m *Manager) UpdateConfig(ctx context.Context, namespace, configMapName string, jwksConfigMapName string, endpoint string) error
func (m *Manager) EnsureDeployment(ctx context.Context, namespace, jwksName, nginxConfigMapName, jwksConfigMapName, endpoint string, nginxResources *NginxResources) error
func (m *Manager) EnsureService(ctx context.Context, namespace, jwksName string) error
```

#### `config_generator.go` (< 200 строк)

Генератор nginx конфигурации для JWKS сервера.

**Ответственность**:
- Генерация nginx конфигурации
- Настройка location для JWKS endpoint
- Конфигурация маршрутизации (JWKS доступен по всем путям через `location /`)

**Основные функции**:
```go
func (g *ConfigGenerator) GenerateConfig(jwksConfigMapName string, endpoint string) (string, error)
func (g *ConfigGenerator) GenerateAllPathsLocationBlock(jwksPath string) string
func (g *ConfigGenerator) GenerateServerBlockWithLocations(port int, rootLocationBlock, jwksLocationBlock string) string
```

#### `deployment.go` (< 300 строк)

Управление nginx Deployment ресурсами.

**Ответственность**:
- Создание/обновление nginx Deployment
- Настройка volumes для nginx конфигурации и JWKS данных
- Управление ресурсами nginx контейнера

**Основные функции**:
```go
func (m *DeploymentManager) EnsureDeployment(ctx context.Context, namespace, jwksConfigName, nginxConfigMapName, jwksConfigMapName, endpoint string, nginxResources *NginxResources) error
func (m *DeploymentManager) DeleteDeployment(ctx context.Context, namespace, jwksConfigName string) error
```

#### `service.go` (< 200 строк)

Управление nginx Service ресурсами.

**Ответственность**:
- Создание/обновление ClusterIP Service для nginx Deployment
- Настройка портов и селекторов

**Основные функции**:
```go
func (m *ServiceManager) EnsureService(ctx context.Context, namespace, jwksConfigName string) error
func (m *ServiceManager) DeleteService(ctx context.Context, namespace, jwksConfigName string) error
```

#### `endpoint_config.go` (< 150 строк)

Конфигурация endpoints для раздачи JWKS.

**Ответственность**:
- Валидация endpoint путей
- Предоставление значений по умолчанию
- Нормализация endpoint путей

**Основные функции**:
```go
func DefaultEndpoint() string
func ValidateEndpoint(endpoint string) error
func NormalizeEndpoint(endpoint string) string
```

### Зависимости

- Kubernetes client для работы с ConfigMap
- Знание структуры JWKS ConfigMap для интеграции

## 5. Configuration Module (`pkg/config/`)

### Назначение

Управление конфигурацией оператора.

### Компоненты

#### `loader.go` (< 200 строк)

Загрузка конфигурации из файла и переменных окружения.

**Ответственность**:
- Загрузка config.yaml
- Разрешение переменных окружения
- Валидация конфигурации

**Основные функции**:
```go
func Load(configPath string) (*Config, error)
func LoadFromEnv() (*Config, error)
func Validate(config *Config) error
```

#### `resolver.go` (< 150 строк)

Разрешение значений конфигурации для разных окружений.

**Ответственность**:
- Определение текущего окружения
- Разрешение значений из секций окружений
- Предоставление значений по умолчанию

**Основные функции**:
```go
func (r *Resolver) ResolveEnvironment() string
func (r *Resolver) GetNamespace(env string) string
func (r *Resolver) GetConfigForEnvironment(env string) (*EnvironmentConfig, error)
```

#### `defaults.go` (< 100 строк)

Значения по умолчанию для конфигурации.

**Ответственность**:
- Определение дефолтных значений
- Предоставление fallback значений

**Основные функции**:
```go
func DefaultConfig() *Config
func DefaultReconcileInterval() time.Duration
func DefaultJWKSUpdateInterval() time.Duration
```

#### `types.go` (< 150 строк)

Определение типов конфигурации.

**Типы**:
```go
type Config struct {
    DefaultNamespace     string
    ReconcileInterval    time.Duration
    JWKSUpdateInterval   time.Duration
    MaxOldKeys           int
    Environments         map[string]EnvironmentConfig
}

type EnvironmentConfig struct {
    Namespace string
    // ... другие поля
}
```

### Зависимости

- Стандартные библиотеки: `os`, `time`
- Внешние зависимости: `gopkg.in/yaml.v3`

## Взаимодействие модулей

### Схема зависимостей

```
controller/
  │
  ├─> reconciler/
  │     │
  │     ├─> jwks/
  │     │     └─> (standalone)
  │     │
  │     ├─> configmap/
  │     │     └─> jwks/
  │     │
  │     ├─> nginx/
  │     │     └─> (standalone, использует имя JWKS ConfigMap)
  │     │
  │     └─> config/
  │           └─> (standalone)
  │
  └─> config/
```

### Правила взаимодействия

1. **Controller** вызывает только **Reconciler**
2. **Reconciler** координирует работу **JWKS**, **ConfigMap** и **Nginx**
3. **ConfigMap** использует типы из **JWKS**
4. **Nginx** использует имя JWKS ConfigMap для интеграции
5. Все модули могут использовать **Config**
6. Избегать циклических зависимостей

## Тестирование модулей

### Unit тесты

Каждый модуль имеет соответствующий `*_test.go` файл:
- `controller_test.go`
- `reconciler_test.go`
- `generator_test.go`
- `manager_test.go` (ConfigMap)
- `manager_test.go` (Nginx)
- `loader_test.go`

### Интеграционные тесты

Тесты взаимодействия модулей в `test/integration/`:
- `reconciler_jwks_integration_test.go`
- `configmap_jwks_integration_test.go`
- `nginx_jwks_integration_test.go`

### Моки

Интерфейсы позволяют легко создавать моки для тестирования:
- `mock_jwks_generator.go`
- `mock_configmap_manager.go`
- `mock_nginx_manager.go`

