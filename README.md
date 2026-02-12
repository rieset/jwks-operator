# JWKS Operator

## Назначение

JWKS Operator - это Kubernetes оператор для автоматического управления JWKS (JSON Web Key Set) конфигурациями при ротации сертификатов JWT.

Оператор решает следующие задачи:

1. **Автоматическое обновление JWKS ConfigMap** при ротации сертификатов
2. **Отслеживание изменений Certificate resources** от cert-manager
3. **Генерация JWKS** из публичных ключей сертификатов
4. **Поддержка graceful rotation** с множественными ключами
5. **Автоматический перезапуск подов** при обновлении ConfigMap

## Быстрый старт

### Требования

- Kubernetes 1.27+
- cert-manager установлен в кластере
- kubectl настроен для доступа к кластеру

### Установка

#### Установка через Helm Chart (рекомендуется)

Helm chart доступен в GitHub Container Registry как OCI artifact по пути `ghcr.io/rieset/helm-charts/jwks-operator`.

Для установки:

**Установка из OCI registry**

```bash
# Установить или обновить оператор из GitHub Container Registry
# Ресурсы будут созданы в namespace, указанном в --namespace
helm upgrade --install jwks-operator \
  oci://ghcr.io/rieset/helm-charts/jwks-operator \
  --version 0.2.0 \
  --namespace system \
  --create-namespace \
  --set image.tag=latest \
  --set enableCRD=true
```

**Аутентификация для приватного репозитория**

Если репозиторий приватный, требуется аутентификация через GitHub Personal Access Token:

```bash
# Создать Personal Access Token с правами read:packages
# Затем выполнить аутентификацию:
echo $GITHUB_TOKEN | helm registry login ghcr.io \
  --username <your-github-username> \
  --password-stdin

# Или использовать переменную окружения напрямую:
helm registry login ghcr.io \
  --username <your-github-username> \
  --password $GITHUB_TOKEN
```

**Примечание:** Helm chart публикуется в GitHub Container Registry по пути `ghcr.io/rieset/helm-charts/jwks-operator`. Для публичных репозиториев аутентификация не требуется.

**Примечания:**
- Chart имя: **jwks-operator**

**Версионирование Helm Chart:**

Версии Helm chart формируются автоматически в зависимости от ветки и типа сборки:

- **Release (теги Git)**: При создании тега (например, `v1.2.3`) версия chart будет равна версии тега без префикса `v` (например, `1.2.3`). Image tag также будет равен тегу (`v1.2.3`).

- **Main ветка**: Используется версия из `Chart.yaml` (например, `0.2.0`). Версия не изменяется при сборке. Image tag будет равен номеру GitHub Actions run.

- **Development ветка**: Версия уменьшается на единицу во втором разряде от версии в `Chart.yaml`. Например, если в `Chart.yaml` версия `0.2.0`, то для development будет `0.1.0`. Image tag будет `latest`.

- **Другие ветки**: Версия и image tag будут равны номеру GitHub Actions run.

**Примеры:**
- Если в `Chart.yaml` версия `0.2.0`:
  - Main: chart версия `0.2.0`, image tag `123` (GitHub Actions run number)
  - Development: chart версия `0.1.0`, image tag `latest`
  - Release `v1.0.0`: chart версия `1.0.0`, image tag `v1.0.0`
- **Namespace:** Ресурсы создаются в namespace, указанном в `--namespace` при установке (например, `--namespace system` или `--namespace example-system`)
- **Image repository:** По умолчанию используется GitHub Container Registry (`ghcr.io/<owner>/<repo>`). Для переопределения используйте `--set image.repository=<your-registry>`

**Пример полной установки с кастомными параметрами:**

```bash
helm upgrade --install jwks-operator \
  oci://ghcr.io/rieset/helm-charts/jwks-operator \
  --version 0.2.0 \
  --namespace system \
  --create-namespace \
  --set image.tag=v1.0.0 \
  --set enableCRD=true \
  --set rbac.create=true \
  --set replicaCount=2 \
  --set resources.limits.memory=1Gi
```

**Решение проблемы с существующими ресурсами:**

Если при установке или обновлении возникает ошибка о том, что ресурсы уже существуют, выполните одно из следующих действий:

1. **Удалить существующие ресурсы перед установкой:**
```bash
kubectl delete clusterrolebinding jwks-operator-rolebinding
kubectl delete clusterrole jwks-operator-role
kubectl delete deployment jwks-operator -n system
kubectl delete serviceaccount jwks-operator -n system
kubectl delete configmap jwks-operator-config -n system
```

2. **Или добавить метки и аннотации Helm к существующим ресурсам (рекомендуется):**
```bash
# Для CRD (если enableCRD=true) - ОБЯЗАТЕЛЬНО перед установкой!
kubectl label crd jwks.example.com app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate crd jwks.example.com meta.helm.sh/release-name=jwks-operator --overwrite
kubectl annotate crd jwks.example.com meta.helm.sh/release-namespace=example-system --overwrite

# Для ClusterRoleBinding (если уже существует)
kubectl label clusterrolebinding jwks-operator-rolebinding app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate clusterrolebinding jwks-operator-rolebinding meta.helm.sh/release-name=jwks-operator --overwrite
kubectl annotate clusterrolebinding jwks-operator-rolebinding meta.helm.sh/release-namespace=example-system --overwrite

# Для ClusterRole (если уже существует)
kubectl label clusterrole jwks-operator-role app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate clusterrole jwks-operator-role meta.helm.sh/release-name=jwks-operator --overwrite
kubectl annotate clusterrole jwks-operator-role meta.helm.sh/release-namespace=example-system --overwrite
```

**Важно:** 
- Ресурсы создаются в namespace, указанном в `--namespace` при установке Helm chart
- Если используете другой namespace (например, `example-system`), замените его во всех командах выше

#### Установка CRD и оператора (вручную)

Если вы предпочитаете установку без Helm:

```bash
# Применить CRD
kubectl apply -f config/crd/bases/

# Применить манифесты оператора
kubectl apply -f config/manager/manager.yaml
kubectl apply -f config/rbac/rbac.yaml
```

#### Настройка для namespace

**Важно:** Перед созданием JWKS ресурса убедитесь, что Secret с сертификатом уже существует в кластере. Secret должен содержать TLS сертификат в формате, который может быть использован для генерации JWKS (обычно это `tls.crt` и `tls.key`).

Создать JWKS ресурс для отслеживания:

```yaml
apiVersion: example.com/v1alpha1
kind: JWKS
metadata:
  name: example-app-jwks-config
  namespace: example-namespace
spec:
  certificateSecret: example-app-jwt-cert  # Имя Secret с сертификатом (должен существовать)
  configMapName: example-app-jwks-config
  nginxConfigMapName: example-app-nginx-config  # Опционально: для создания nginx Deployment
  updateStrategy: rolling
  keepOldKeys: true
  oldKeysTTL: 720h  # 30 дней
  endpoint: /jwks.json  # Опционально: путь для JWKS endpoint (по умолчанию /jwks.json)
```

**Примечания:**
- **Secret должен существовать:** Если Secret, указанный в `certificateSecret`, не найден, оператор будет периодически проверять его появление (каждые 30 секунд) и автоматически начнет работу, когда Secret появится.
- **Если указан `nginxConfigMapName`**, оператор автоматически:
  - Создаст ConfigMap с конфигурацией nginx для раздачи JWKS через HTTP
  - Создаст Deployment `<jwks-name>` (без префикса nginx-) для обслуживания JWKS endpoint
  - Создаст ClusterIP Service `<jwks-name>` для доступа к nginx pod
  - JWKS будет доступен по всем путям (включая `/` и `/jwks.json`) через nginx pod благодаря использованию `location /` с `try_files`

### Использование

После установки оператор автоматически отслеживает изменения в Certificate ресурсах и обновляет соответствующие JWKS ConfigMap. Подробнее о конфигурации см. [Конфигурация](#конфигурация).

## Разработка

### Требования для разработки

- Go 1.21+
- Kubernetes 1.27+
- cert-manager установлен в кластере
- kubectl настроен для доступа к кластеру

### Запуск локально

```bash
# Установить CRD
make install

# Запустить оператор локально (требует доступ к кластеру)
make run
```

### Сборка

#### Локальная сборка

```bash
# Клонировать репозиторий
# Установить зависимости
go mod download

# Запустить тесты
go test ./...

# Собрать бинарный файл
go build -o bin/jwks-operator ./cmd/manager

# Собрать Docker образ
docker build -t jwks-operator:latest .
```

#### Сборка с конфигурацией

Конфигурация из `config.yaml` интегрируется в сборку через build tags:

```bash
# Сборка с конфигурацией по умолчанию
go build -tags=config -o bin/jwks-operator ./cmd/manager

# Сборка для конкретного окружения
go build -tags=config -ldflags="-X main.configEnv=production" -o bin/jwks-operator ./cmd/manager
```

### Запуск тестов

```bash
# Все тесты
make test

# Тесты с покрытием
make test-coverage

# Линтинг
make lint

# Проверка форматирования
make fmt-check

# Все проверки (форматирование + линтинг)
make check
```

### Pre-commit hook

Для автоматической проверки кода перед коммитом установите pre-commit hook:

```bash
git config core.hooksPath .githooks
```

Подробнее в [.githooks/README.md](.githooks/README.md).

### Генерация кода

```bash
# Генерация CRD манифестов
make manifests

# Генерация кода клиентов
make generate
```

## Архитектура

Оператор построен на базе [Kubebuilder](https://book.kubebuilder.io/) и использует controller-runtime для работы с Kubernetes API.

### Основные компоненты:

- **Controller** - отслеживает изменения Certificate и Secret ресурсов
- **JWKS Generator** - генерирует JWKS из сертификатов
- **ConfigMap Manager** - управляет обновлением ConfigMap
- **Reconciler** - координирует процесс обновления

Подробная архитектура описана в [docs/architecture.md](docs/architecture.md).

## Конфигурация

Конфигурация оператора находится в `config.yaml`. Поддерживаются следующие параметры:

- `defaultNamespace` - namespace по умолчанию
- `reconcileInterval` - интервал реконсиляции (по умолчанию 5 минут)
- `jwksUpdateInterval` - интервал обновления JWKS (по умолчанию 6 часов)
- `jwksVerificationInterval` - интервал проверки валидности JWKS через nginx (по умолчанию 5 минут)
- `maxOldKeys` - максимальное количество старых ключей
- `verification` - настройки верификации JWKS (таймауты, retry)
- `nginx` - настройки nginx Deployment (образ, порт, ресурсы, cache-control)

**Верификация JWKS:**
Оператор периодически (каждые 5 минут по умолчанию) проверяет, что JWKS, отдаваемый nginx, может верифицировать JWT токены. Это гарантирует, что JWKS корректно работает и может использоваться для проверки подписи токенов.

Подробнее в [docs/configuration.md](docs/configuration.md).

## Мониторинг

Оператор предоставляет метрики Prometheus для мониторинга работы:

- `jwks_operator_reconcile_total` - общее количество реконсиляций
- `jwks_operator_reconcile_duration_seconds` - длительность реконсиляции
- `jwks_operator_configmap_updates_total` - обновления ConfigMap
- `jwks_operator_jwks_generation_total` - генерация JWKS
- `jwks_operator_nginx_operations_total` - операции nginx
- `jwks_operator_jwks_verification_total` - верификация JWKS
- `jwks_operator_errors_total` - ошибки оператора по типам

Метрики доступны через HTTP endpoint `/metrics` на порту 8080.

Подробнее в [docs/metrics.md](docs/metrics.md).

## Логирование

Оператор использует структурированное логирование через zap logger. Уровни логирования:

- `DEBUG` - детальная отладочная информация
- `INFO` - информационные сообщения
- `WARN` - предупреждения
- `ERROR` - ошибки

## Лицензия

Copyright (c) 2026 JWKS Operator Contributors
