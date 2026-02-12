# Руководство по развертыванию и проверке JWKS Operator

Это руководство описывает процесс развертывания JWKS Operator в Kubernetes кластер и проверки его корректной работы.

## Предварительные требования

- `kubectl` установлен и настроен
- `helm` установлен (для установки через Helm chart)
- Доступ к Kubernetes кластеру с соответствующим контекстом
- Права на создание ресурсов в кластере (CRD, ClusterRole, Deployment и т.д.)
- Доступ к GitHub Container Registry или другому реестру образов (для установки Helm chart)

## Установка оператора

Оператор устанавливается через Helm chart. Подробные инструкции по установке см. в [README.md](../README.md#установка).

### Быстрая установка

```bash
# Установить оператор из GitHub Container Registry (OCI registry)
helm upgrade --install jwks-operator \
  oci://ghcr.io/rieset/helm-charts/jwks-operator \
  --version 0.2.0 \
  --namespace system \
  --create-namespace \
  --set image.tag=latest \
  --set enableCRD=true
```

**Примечание:** Helm chart публикуется в GitHub Container Registry по пути `ghcr.io/rieset/helm-charts/jwks-operator`. Для приватных репозиториев может потребоваться аутентификация через `helm registry login ghcr.io`.

### Шаг 2: Создание JWKS ресурса

После развертывания оператора создайте JWKS ресурс:

```bash
kubectl apply -f config/samples/jwks.yaml -n <namespace>
```

Или создайте свой ресурс:

```yaml
apiVersion: example.com/v1alpha1
kind: JWKS
metadata:
  name: example-app-jwks-config
  namespace: <namespace>
spec:
  certificateSecret: example-app-jwt-cert
  configMapName: example-app-jwks-config
  nginxConfigMapName: example-app-nginx-config
  # Endpoint field is kept for backward compatibility
  # JWKS is available at both "/" and "/jwks.json" paths
  endpoint: "/jwks.json"
  updateStrategy: rolling
  keepOldKeys: true
  oldKeysTTL: "720h"
  # Опционально: настройка интервалов реконсиляции и обновления
  # reconcileInterval: "5m"      # Интервал между реконсиляциями (по умолчанию из config.yaml)
  # jwksUpdateInterval: "6h"      # Интервал проверки обновления JWKS (по умолчанию из config.yaml)
```

### Шаг 3: Проверка работы

```bash
# Проверить статус JWKS ресурса
kubectl get jwks example-app-jwks-config -n <namespace>

# Проверить ConfigMap с JWKS
kubectl get configmap example-app-jwks-config -n <namespace> -o yaml

# Проверить nginx Deployment (если указан nginxConfigMapName)
kubectl get deployment example-app-jwks-config -n <namespace>

# Проверить nginx Service
kubectl get service example-app-jwks-config -n <namespace>

# Проверить доступность JWKS через nginx
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://example-app-jwks-config.<namespace>.svc.cluster.local/jwks.json
```

## Устранение проблем

### Оператор не запускается

1. Проверьте логи:
   ```bash
   kubectl logs -n <namespace> -l app=jwks-operator --context=<context>
   ```

2. Проверьте статус deployment:
   ```bash
   kubectl get deployment jwks-operator -n <namespace> --context=<context>
   ```

3. Проверьте события:
   ```bash
   kubectl get events -n <namespace> --context=<context> --sort-by='.lastTimestamp'
   ```

### JWKS не обновляется

1. Проверьте статус JWKS:
   ```bash
   kubectl get jwks <name> -n <namespace> --context=<context> -o yaml
   ```

2. Проверьте наличие Secret с сертификатом:
   ```bash
   kubectl get secret <cert-secret> -n <namespace> --context=<context>
   ```

3. Проверьте логи оператора на наличие ошибок

### Nginx не отдает JWKS

1. Проверьте наличие nginx Deployment:
   ```bash
   kubectl get deployment -n <namespace> -l app=nginx-jwks --context=<context>
   ```

2. Проверьте наличие nginx pod:
   ```bash
   kubectl get pods -n <namespace> -l app=nginx-jwks --context=<context>
   ```

3. Проверьте конфигурацию nginx:
   ```bash
   kubectl get configmap <nginx-configmap> -n <namespace> --context=<context> -o yaml
   ```

4. Проверьте доступность JWKS через pod (доступен по всем путям, включая `/` и `/jwks.json`):
   ```bash
   # Проверка корневого пути
   kubectl exec -n <namespace> <nginx-pod> --context=<context> -- curl http://localhost/
   
   # Проверка пути /jwks.json
   kubectl exec -n <namespace> <nginx-pod> --context=<context> -- curl http://localhost/jwks.json
   
   # Проверка через Service
   kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- curl http://<service-name>.<namespace>.svc.cluster.local/jwks.json
   ```

### ConfigMap был удален, но не пересоздается

Оператор автоматически отслеживает ConfigMaps (`configMapName` и `nginxConfigMapName`) и пересоздает их при удалении. Если ConfigMap не пересоздается:

1. Проверьте логи оператора на наличие ошибок
2. Убедитесь, что JWKS ресурс существует и имеет правильные настройки
3. Проверьте права оператора на создание ConfigMaps

## Дополнительные команды

### Просмотр всех JWKS ресурсов

```bash
kubectl get jwks -n <namespace> --context=<context>
```

### Просмотр детальной информации о JWKS

```bash
kubectl describe jwks <name> -n <namespace> --context=<context>
```

### Просмотр JWKS из ConfigMap

```bash
kubectl get configmap <configmap-name> -n <namespace> --context=<context> -o jsonpath='{.data.jwks}' | jq .
```

### Удаление оператора

```bash
# Удалить deployment
kubectl delete deployment jwks-operator -n <namespace> --context=<context>

# Удалить ServiceAccount
kubectl delete serviceaccount jwks-operator -n <namespace> --context=<context>

# Удалить ClusterRoleBinding
kubectl delete clusterrolebinding jwks-operator-rolebinding --context=<context>

# Удалить ClusterRole (если больше не используется)
kubectl delete clusterrole manager-role --context=<context>

# Удалить CRD (будет удален и JWKS ресурс)
kubectl delete crd jwks.example.com --context=<context>
```

## Особенности работы оператора

### Автоматическое пересоздание ConfigMaps

Оператор отслеживает ConfigMaps, указанные в `spec.configMapName` и `spec.nginxConfigMapName`. Если эти ConfigMaps удаляются, оператор автоматически пересоздает их при следующей реконсиляции:

- **JWKS ConfigMap** (`configMapName`) - пересоздается с актуальными JWKS данными из сертификата
- **Nginx ConfigMap** (`nginxConfigMapName`) - пересоздается с актуальной конфигурацией nginx

### Автоматическое создание Nginx Deployment

Для каждого JWKS ресурса, у которого указан `nginxConfigMapName`, оператор автоматически создает отдельный Deployment с nginx:

- **Имя Deployment**: `<jwks-name>` (без префикса nginx-)
- **Образ**: `nginx:1.25-alpine`
- **Volumes**: 
  - Nginx конфигурация из `nginxConfigMapName`
  - JWKS данные из `configMapName`
- **Endpoints**: JWKS доступен по всем путям (включая `/` и `/jwks.json`) благодаря использованию `location /` с `try_files`
- **Service**: Создается ClusterIP Service с тем же именем для доступа к nginx pod
  
  Поле `spec.endpoint` сохраняется для обратной совместимости, но не влияет на генерацию конфигурации nginx.

При удалении JWKS ресурса соответствующий nginx Deployment и Service также автоматически удаляются.

## Примечания

- Убедитесь, что у вас есть права на создание ClusterRole и ClusterRoleBinding
- Интервалы реконсиляции и обновления JWKS могут быть настроены в каждом JWKS ресурсе через поля `reconcileInterval` и `jwksUpdateInterval`
- Если интервалы не указаны в CRD, используются значения по умолчанию из `config.yaml`
- Оператор автоматически отслеживает и пересоздает удаленные ConfigMaps
- На каждый JWKS создается отдельный nginx Deployment и Service для обслуживания JWKS endpoint
- Имя Deployment и Service совпадает с именем JWKS ресурса (без префикса nginx-)
- JWKS доступен по всем путям через nginx благодаря использованию `location /` с `try_files`

