# Обзор JWKS Operator

## Что это?

JWKS Operator - это Kubernetes оператор для автоматического управления JWKS (JSON Web Key Set) конфигурациями при ротации сертификатов JWT, управляемых cert-manager.

## Зачем это нужно?

### Проблема

При ротации сертификатов JWT возникает несколько проблем:

1. **JWKS ConfigMap не обновляется автоматически** - старый публичный ключ остается в ConfigMap
2. **Pods не перезапускаются** - продолжают использовать старые данные из Secret/ConfigMap
3. **Нет механизма graceful rotation** - невозможно поддерживать старые токены во время ротации

### Решение

JWKS Operator автоматически:

- ✅ Отслеживает изменения сертификатов от cert-manager
- ✅ Генерирует JWKS из новых сертификатов
- ✅ Обновляет ConfigMap с новыми ключами
- ✅ Создает/обновляет nginx конфигурацию для JWKS сервера
- ✅ Автоматически создает nginx Deployment и Service для раздачи JWKS
- ✅ Поддерживает graceful rotation с множественными ключами

## Как это работает?

### Простой пример

```
1. cert-manager создает новый сертификат
   └─> Secret обновляется

2. Operator обнаруживает изменение
   └─> Генерирует JWKS из нового сертификата

3. ConfigMap обновляется
   └─> Новый ключ добавляется, старый сохраняется (graceful rotation)

4. Nginx Config создается/обновляется
   └─> Создается nginx конфигурация для JWKS сервера
   └─> Настраивается location для отдачи публичного ключа
   └─> JWKS доступен через HTTP endpoint по всем путям (включая `/` и `/jwks.json`)

5. Nginx Deployment и Service создаются/обновляются
   └─> Создается nginx Deployment с конфигурацией и JWKS данными
   └─> Создается ClusterIP Service для доступа к nginx pod
   └─> JWKS endpoint доступен с актуальными публичными ключами
   └─> Новые токены подписываются новым ключом
   └─> Старые токены верифицируются старым ключом
```

**Примечание**: Оператор автоматически создает nginx Deployment и Service для каждого JWKS ресурса, у которого указан `nginxConfigMapName`. Nginx контейнер монтирует ConfigMap с конфигурацией и JWKS данными, поэтому изменения автоматически применяются при обновлении ConfigMap.

### Graceful Rotation

Оператор поддерживает graceful rotation:

- **Новые токены** подписываются новым ключом
- **Старые токены** верифицируются старым ключом
- **Оба ключа** доступны в JWKS endpoint
- **Старые ключи** удаляются после TTL

## Архитектура

### Компоненты

1. **Controller** - отслеживает изменения ресурсов
2. **Reconciler** - координирует процесс обновления
3. **JWKS Generator** - генерирует JWKS из сертификатов
4. **ConfigMap Manager** - управляет ConfigMap ресурсами с JWKS данными
5. **Nginx Config Manager** - создает/обновляет nginx конфигурацию для JWKS сервера

### Модули

Оператор разделен на модули:

- `pkg/controller/` - контроллеры и watchers
- `pkg/reconciler/` - реконсиляция ресурсов
- `pkg/jwks/` - генерация JWKS
- `pkg/configmap/` - управление ConfigMap с JWKS данными
- `pkg/nginx/` - управление nginx конфигурацией для JWKS сервера
- `pkg/config/` - конфигурация оператора

Подробнее в [modules.md](modules.md).

## Быстрый старт

### 1. Установка

```bash
# Применить CRD
kubectl apply -f config/crd/bases/

# Применить оператор
kubectl apply -f config/manager/manager.yaml
kubectl apply -f config/rbac/rbac.yaml
```

### 2. Создание JWKS

```yaml
apiVersion: example.com/v1alpha1
kind: JWKS
metadata:
  name: example-app-jwks-config
  namespace: example-app-b2b
spec:
  certificateSecret: example-app-jwt-cert
  configMapName: example-app-jwks-config
  updateStrategy: rolling
  keepOldKeys: true
  oldKeysTTL: 720h
```

### 3. Проверка работы

```bash
# Проверить статус оператора
kubectl get pods -n jwks-operator-system

# Проверить JWKS
kubectl get jwks -n example-app-b2b

# Проверить ConfigMap
kubectl get configmap example-app-jwks-config -n example-app-b2b -o yaml
```

## Автоматическое создание Nginx ресурсов

Оператор автоматически создает следующие ресурсы для каждого JWKS, у которого указан `nginxConfigMapName`:

- **Nginx ConfigMap** - конфигурация nginx для раздачи JWKS
- **Nginx Deployment** - Deployment с nginx контейнером (имя совпадает с именем JWKS ресурса)
- **Nginx Service** - ClusterIP Service для доступа к nginx pod (имя совпадает с именем JWKS ресурса)

JWKS доступен по всем путям через nginx благодаря использованию `location /` с `try_files`. Это означает, что JWKS можно получить как по пути `/`, так и по `/jwks.json`, и по любому другому пути.

## Документация

- [Архитектура](architecture.md) - детальное описание архитектуры
- [Модули](modules.md) - описание модулей оператора
- [Алгоритм](algorithm.md) - алгоритм работы оператора
- [Конфигурация](configuration.md) - настройка оператора

## Разработка

### Требования

- Go 1.21+
- Kubernetes 1.27+
- kubebuilder 3.x

### Сборка

```bash
# Установить зависимости
go mod download

# Запустить тесты
go test ./...

# Собрать оператор
go build -o bin/jwks-operator ./cmd/manager
```

### Запуск локально

```bash
# Установить CRD
make install

# Запустить оператор
make run
```

## Конфигурация

Конфигурация находится в `config.yaml`:

```yaml
defaultNamespace: "example-app-b2b"
reconcileInterval: "5m"
jwksUpdateInterval: "6h"
maxOldKeys: 2
defaultOldKeysTTL: "720h"
```

Подробнее в [configuration.md](configuration.md).

## Мониторинг

Оператор предоставляет метрики Prometheus для мониторинга работы:

- `jwks_operator_reconcile_total` - счетчик реконсиляций
- `jwks_operator_reconcile_duration_seconds` - длительность реконсиляции
- `jwks_operator_configmap_updates_total` - обновления ConfigMap
- `jwks_operator_jwks_generation_total` - генерация JWKS
- `jwks_operator_nginx_operations_total` - операции nginx
- `jwks_operator_jwks_verification_total` - верификация JWKS
- `jwks_operator_errors_total` - ошибки по типам

Подробнее в [docs/metrics.md](metrics.md).

## Лицензия

Copyright (c) 2026 JWKS Operator Contributors

