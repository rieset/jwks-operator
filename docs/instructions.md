# Инструкции для AI при разработке JWKS Operator

## ⚠️ КРИТИЧЕСКОЕ ТРЕБОВАНИЕ

**НЕ СОЗДАВАТЬ ФАЙЛЫ ДО ЯВНОГО ЗАПРОСА ПОЛЬЗОВАТЕЛЯ**

- ❌ **ЗАПРЕЩЕНО** создавать файлы без явного указания пользователя
- ✅ **РАЗРЕШЕНО** создавать файлы только когда пользователь явно просит это сделать
- ✅ **РАЗРЕШЕНО** редактировать существующие файлы по запросу пользователя
- ✅ **РАЗРЕШЕНО** предлагать создание файлов, но не создавать их без подтверждения

**Это требование имеет наивысший приоритет и должно соблюдаться всегда.**

## Общие принципы работы

При разработке кода для JWKS Operator следуй этим инструкциям:

### 1. Анализ перед кодогенерацией

**⚠️ ВАЖНО: Создавать файлы ТОЛЬКО по явному запросу пользователя!**

**Перед созданием любого файла (только если пользователь явно попросил):**

1. ✅ Убедиться, что пользователь явно попросил создать файл
2. ✅ Проверить документацию в `docs/` для понимания архитектуры
3. ✅ Оценить размер файла - не превышать 500 строк
4. ✅ Если файл будет больше 300 строк - разделить на модули
5. ✅ Проанализировать функции на возможность выделения в отдельный модуль
6. ✅ Проверить зависимости и избежать циклических импортов

### 2. Структура и модульность

**При создании нового модуля:**

```
pkg/
├── module_name/
│   ├── module.go          # Основной интерфейс и типы (< 300 строк)
│   ├── implementation.go  # Реализация (< 300 строк)
│   ├── helpers.go         # Вспомогательные функции (< 200 строк)
│   └── module_test.go     # Тесты
```

**Правила разделения:**

- Если файл > 300 строк → разделить на логические части
- Если функция > 50 строк → рассмотреть выделение в отдельную функцию
- Если пакет > 5 файлов → рассмотреть разделение на подпакеты

### 3. Конфигурация и хардкодинг

**Запрещено:**

❌ Хардкодить значения (строки, числа, URL)
❌ Использовать магические числа без констант
❌ Прописывать namespace напрямую в коде

**Разрешено:**

✅ Использовать значения из `config.yaml`
✅ Использовать константы из `pkg/constants/`
✅ Использовать переменные окружения через `pkg/config/`

**Пример:**

```go
// ❌ Плохо
namespace := "example-app-b2b"
interval := 3600

// ✅ Хорошо
namespace := cfg.DefaultNamespace
interval := cfg.ReconcileInterval
```

### 4. Линтинг и качество кода

**⚠️ КРИТИЧЕСКОЕ ТРЕБОВАНИЕ: ВСЕГДА запускать линтинг после создания или изменения файлов!**

**После создания или изменения ЛЮБОГО файла (ОБЯЗАТЕЛЬНО):**

1. ✅ **ОБЯЗАТЕЛЬНО** запустить `gofmt -w <путь_к_файлу>` для форматирования
2. ✅ **ОБЯЗАТЕЛЬНО** запустить `goimports -w <путь_к_файлу>` для проверки импортов
3. ✅ **ОБЯЗАТЕЛЬНО** запустить `make lint` или `golangci-lint run ./<путь_к_файлу>`
4. ✅ **ОБЯЗАТЕЛЬНО** исправить все предупреждения и ошибки линтинга
5. ✅ **ОБЯЗАТЕЛЬНО** убедиться, что `gofmt -l .` не показывает измененных файлов

**Правила линтинга:**

- Следовать `.golangci.yml` конфигурации
- Все экспортируемые функции должны иметь комментарии
- Обрабатывать все ошибки
- Использовать `context.Context` для долгих операций
- **НЕ коммитить код, который не проходит линтинг**

**Пример последовательности действий:**

```bash
# 1. Создать/изменить файл
# 2. Отформатировать
gofmt -w pkg/nginx/service.go

# 3. Проверить импорты
goimports -w pkg/nginx/service.go

# 4. Запустить линтинг
golangci-lint run ./pkg/nginx/service.go

# 5. Проверить форматирование всех файлов
gofmt -l .  # Должно быть пусто

# 6. Исправить все ошибки и повторить шаги 2-5
```

### 5. Best Practices Go

**Обработка ошибок:**

```go
// ❌ Плохо
result, _ := someFunction()

// ✅ Хорошо
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to execute: %w", err)
}
```

**Контекст:**

```go
// ✅ Всегда передавать context
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Использовать ctx для таймаутов и отмены
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
}
```

**Логирование:**

```go
// ✅ Использовать структурированное логирование
logger.Info("reconciling JWKS config",
    "namespace", req.Namespace,
    "name", req.Name,
    "configMap", jwksConfig.Spec.ConfigMapName,
)
```

### 6. Тестирование

**При создании функций:**

1. ✅ Создавать unit-тесты для всех публичных функций
2. ✅ Использовать табличные тесты для множественных сценариев
3. ✅ Мокировать внешние зависимости
4. ✅ Тестировать граничные случаи и ошибки

**Пример:**

```go
func TestGenerateJWKS(t *testing.T) {
    tests := []struct {
        name    string
        cert    []byte
        wantErr bool
    }{
        {
            name:    "valid certificate",
            cert:    validCertPEM,
            wantErr: false,
        },
        {
            name:    "invalid certificate",
            cert:    []byte("invalid"),
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // тест
        })
    }
}
```

### 7. Работа с Kubernetes API

**Controller-runtime паттерны:**

```go
// ✅ Правильная структура контроллера
type JWKSConfigReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Log    logr.Logger
    Config *config.Config
}

// ✅ Правильная реконсиляция
func (r *JWKSConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Получить ресурс
    // 2. Проверить существование
    // 3. Выполнить логику
    // 4. Обновить статус
    // 5. Вернуть результат
}
```

**Обработка ресурсов:**

```go
// ✅ Использовать Finalizers для очистки
if !controllerutil.ContainsFinalizer(obj, finalizerName) {
    controllerutil.AddFinalizer(obj, finalizerName)
    return ctrl.Result{}, r.Update(ctx, obj)
}

// ✅ Проверять OwnerReferences
if !metav1.IsControlledBy(obj, owner) {
    return fmt.Errorf("resource is not owned by %s", owner.Name)
}
```

### 8. Конфигурация из config.yaml

**Загрузка конфигурации:**

```go
// ✅ Загружать при старте
cfg, err := config.Load("config.yaml")
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// ✅ Использовать значения из конфигурации
reconcileInterval := cfg.ReconcileInterval
defaultNamespace := cfg.DefaultNamespace
```

**Структура config.yaml:**

```yaml
defaultNamespace: "example-app-b2b"
reconcileInterval: "5m"
jwksUpdateInterval: "6h"
maxOldKeys: 2
environments:
  b2b:
    namespace: "example-app-b2b"
  b2c:
    namespace: "example-app-b2c"
```

### 9. Документация

**При создании нового модуля:**

1. ✅ Обновить `docs/architecture.md` если изменяется архитектура
2. ✅ Добавить примеры использования в godoc
3. ✅ Обновить `README.md` если добавляется новая функциональность

**Формат комментариев:**

```go
// GenerateJWKS generates a JWKS (JSON Web Key Set) from a certificate.
// It extracts the public key from the certificate and formats it according
// to RFC 7517.
//
// Parameters:
//   - cert: PEM-encoded certificate
//
// Returns:
//   - jwks: JWKS JSON structure
//   - error: if certificate parsing fails
func GenerateJWKS(cert []byte) (*JWKS, error) {
    // ...
}
```

### 10. Проверочный список перед коммитом

Перед завершением работы убедись:

- [ ] Все файлы < 500 строк
- [ ] Файлы > 300 строк разделены на модули
- [ ] Нет хардкода значений
- [ ] Используется config.yaml для конфигурации
- [ ] **ОБЯЗАТЕЛЬНО: Линтинг проходит без ошибок** (`make lint` или `golangci-lint run`)
- [ ] **ОБЯЗАТЕЛЬНО: Форматирование проверено** (`gofmt -l .` должно быть пусто)
- [ ] **ОБЯЗАТЕЛЬНО: Импорты проверены** (`goimports -w` применен)
- [ ] **ОБЯЗАТЕЛЬНО: Сообщение коммита на английском языке**
- [ ] Все тесты проходят
- [ ] Документация обновлена
- [ ] Ошибки обрабатываются правильно
- [ ] Используется структурированное логирование
- [ ] Код следует best practices Go

### 11. Git коммиты

**⚠️ КРИТИЧЕСКОЕ ТРЕБОВАНИЕ: Все коммиты должны быть на английском языке!**

**Правила написания коммитов:**

- ✅ **ОБЯЗАТЕЛЬНО**: Сообщение коммита на английском языке
- ✅ Использовать формат: `type: description`
- ✅ Типы коммитов: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`
- ✅ Краткое описание (до 50 символов в первой строке)
- ✅ Один коммит = одна логическая единица

**Примеры правильных коммитов:**

```
feat: add GitHub Actions workflow for CI/CD
fix: correct CRD name in jwks.yaml
docs: update deployment instructions
refactor: split large file into modules
test: add unit tests for JWKS generator
chore: remove GitLab CI configuration
ci: add linting workflow
```

**Примеры неправильных коммитов:**

```
❌ Добавил GitHub Actions workflow
❌ Исправлена ошибка в CRD
❌ Обновлена документация
```

## Примеры правильной структуры

### Малый модуль (< 300 строк)

```go
// pkg/jwks/generator.go
package jwks

// GenerateJWKS generates JWKS from certificate
func GenerateJWKS(cert []byte) (*JWKS, error) {
    // реализация
}
```

### Большой модуль (> 300 строк) - разделить

```go
// pkg/jwks/generator.go - основная логика
package jwks

func GenerateJWKS(cert []byte) (*JWKS, error) {
    // основная логика
}

// pkg/jwks/key_extractor.go - извлечение ключей
package jwks

func ExtractPublicKey(cert []byte) (*rsa.PublicKey, error) {
    // извлечение ключа
}

// pkg/jwks/formatter.go - форматирование
package jwks

func FormatJWKS(key *rsa.PublicKey) (*JWKS, error) {
    // форматирование
}
```

## Вопросы и уточнения

Если не уверен в подходе:

1. Проверь документацию в `docs/`
2. Посмотри на существующие модули как примеры
3. Следуй паттернам controller-runtime
4. Придерживайся принципов SOLID

