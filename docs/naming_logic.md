# Логика именования ресурсов JWKS Operator

## Общая схема

```
JWKS Resource (example-app-jwks)
    ↓
    ├─→ Deployment Name: example-app-jwks
    │   └─→ Pod Labels: app=nginx-jwks, jwks-config=example-app-jwks
    │
    └─→ Service Name: example-app-jwks
        └─→ Selectors: app=nginx-jwks, jwks-config=example-app-jwks
```

## 1. Источник именования

**Все имена ресурсов берутся из имени JWKS Custom Resource:**

```go
// pkg/reconciler/phases.go:189
jwks.Name  // Например: "example-app-jwks"
```

## 2. Deployment

### Имя Deployment

```go
// pkg/nginx/deployment.go:43
deploymentName := jwksConfigName  // = jwks.Name
```

**Пример:** Для JWKS ресурса `example-app-jwks` → Deployment будет называться `example-app-jwks`

### Labels Deployment (метаданные Deployment)

```go
// pkg/nginx/deployment_builder.go:121-126
func buildLabels(name string) map[string]string {
    return map[string]string{
        config.LabelApp:        config.LabelAppValue,        // "app": "nginx-jwks"
        config.LabelJWKSConfig: name,                        // "jwks-config": "example-app-jwks"
        config.LabelManagedBy:  config.LabelManagedByValue,  // "managed-by": "jwks-operator"
    }
}
```

**Результат:**
- `app: nginx-jwks`
- `jwks-config: example-app-jwks`
- `managed-by: jwks-operator`

### Labels Pods (PodTemplate labels)

```go
// pkg/nginx/deployment_builder.go:129-135
func buildSelectorLabels(name string) map[string]string {
    return map[string]string{
        config.LabelApp:        config.LabelAppValue,  // "app": "nginx-jwks"
        config.LabelJWKSConfig: name,                 // "jwks-config": "example-app-jwks"
    }
}
```

**Результат для Pods:**
- `app: nginx-jwks`
- `jwks-config: example-app-jwks`

**Важно:** Эти labels устанавливаются в `deployment.Spec.Template.Labels` и применяются ко всем Pods, созданным этим Deployment.

### Имя Pods

Kubernetes автоматически генерирует имена Pods на основе имени Deployment:
- Формат: `<deployment-name>-<replica-set-hash>-<random-suffix>`
- Пример: `example-app-jwks-7b9f5b849b-r88sd`

## 3. Service

### Имя Service

```go
// pkg/nginx/service.go:36
serviceName := jwksConfigName  // = jwks.Name
```

**Пример:** Для JWKS ресурса `example-app-jwks` → Service будет называться `example-app-jwks`

### Labels Service

```go
// pkg/nginx/service.go:143-147
Labels: map[string]string{
    config.LabelApp:        config.LabelAppValue,        // "app": "nginx-jwks"
    config.LabelJWKSConfig: jwksConfigName,             // "jwks-config": "example-app-jwks"
    config.LabelManagedBy:  config.LabelManagedByValue, // "managed-by": "jwks-operator"
}
```

### Selectors Service

```go
// pkg/nginx/service.go:151-154
Selector: map[string]string{
    config.LabelApp:        config.LabelAppValue,  // "app": "nginx-jwks"
    config.LabelJWKSConfig: jwksConfigName,       // "jwks-config": "example-app-jwks"
}
```

**Результат:**
- Service ищет Pods с labels:
  - `app: nginx-jwks`
  - `jwks-config: example-app-jwks`

## 4. Константы

```go
// pkg/config/constants.go:45-57
const (
    LabelApp        = "app"
    LabelJWKSConfig = "jwks-config"
    LabelManagedBy  = "managed-by"
    
    LabelAppValue       = "nginx-jwks"      // Фиксированное значение для всех nginx Pods
    LabelManagedByValue = "jwks-operator"  // Фиксированное значение для всех ресурсов оператора
)
```

## 5. Цепочка вызовов

### Создание Deployment

```
ReconciliationLoop.phase5EnsureNginxDeployment()
    ↓
    jwks.Name  // "example-app-jwks"
    ↓
nginxManager.EnsureDeployment()
    ↓
    jwksConfigName = jwks.Name  // "example-app-jwks"
    ↓
DeploymentManager.EnsureDeployment()
    ↓
    deploymentName := jwksConfigName  // "example-app-jwks"
    ↓
createDeployment(name="example-app-jwks", ...)
    ↓
    buildSelectorLabels("example-app-jwks")
    ↓
    PodTemplate.Labels = {
        "app": "nginx-jwks",
        "jwks-config": "example-app-jwks"
    }
```

### Создание Service

```
ReconciliationLoop.phase6EnsureNginxService()
    ↓
    jwks.Name  // "example-app-jwks"
    ↓
nginxManager.EnsureService()
    ↓
    jwksConfigName = jwks.Name  // "example-app-jwks"
    ↓
ServiceManager.EnsureService()
    ↓
    serviceName := jwksConfigName  // "example-app-jwks"
    ↓
createService(name="example-app-jwks", ...)
    ↓
    Service.Spec.Selector = {
        "app": "nginx-jwks",
        "jwks-config": "example-app-jwks"
    }
```

## 6. Связь между Service и Pods

Service использует **selectors** для поиска Pods:

```yaml
Service:
  spec:
    selector:
      app: nginx-jwks
      jwks-config: example-app-jwks
```

Deployment создает Pods с **labels**:

```yaml
Deployment:
  spec:
    template:
      metadata:
        labels:
          app: nginx-jwks
          jwks-config: example-app-jwks
```

**Совпадение:** Service находит Pods, потому что их labels совпадают с selectors Service.

## 7. Проблема с несовпадением labels

Если Deployment был создан не оператором (например, вручную или через Helm chart), его Pods могут иметь другие labels:

**Неправильно:**
```yaml
Deployment:
  spec:
    template:
      metadata:
        labels:
          app: example-app-jwks        # ❌ Должно быть "nginx-jwks"
          chart: example-app-0.0.1     # ❌ Лишний label
          commit: NP-null-zGu2-NP  # ❌ Лишний label
          # jwks-config отсутствует # ❌ Должен быть "jwks-config: example-app-jwks"
```

**Правильно (создается оператором):**
```yaml
Deployment:
  spec:
    template:
      metadata:
        labels:
          app: nginx-jwks           # ✅ Правильное значение
          jwks-config: example-app-jwks # ✅ Правильное значение
```

**Решение:** Оператор пытается автоматически обновить селекторы Service на основе labels Deployment PodTemplate (см. `pkg/nginx/service.go:58-85`), но лучше удалить неправильный Deployment и позволить оператору создать правильный.

## 8. Обратная совместимость

Оператор поддерживает поиск Deployment с префиксом `nginx-`:

```go
// pkg/nginx/service.go:50-55
deploymentErr := m.client.Get(ctx, deploymentKey, deployment)
if deploymentErr != nil {
    // Try with nginx- prefix (for backward compatibility)
    deploymentKey.Name = "nginx-" + jwksConfigName
    deploymentErr = m.client.Get(ctx, deploymentKey, deployment)
}
```

Это позволяет работать со старыми Deployment, созданными с именами вида `nginx-example-app-jwks`.

## 9. Резюме

| Ресурс | Имя | Источник | Labels/Selectors |
|--------|-----|----------|------------------|
| **JWKS CR** | `example-app-jwks` | Пользователь | - |
| **Deployment** | `example-app-jwks` | `jwks.Name` | `app=nginx-jwks`, `jwks-config=example-app-jwks`, `managed-by=jwks-operator` |
| **Pods** | `example-app-jwks-<hash>-<suffix>` | Kubernetes (автоматически) | `app=nginx-jwks`, `jwks-config=example-app-jwks` |
| **Service** | `example-app-jwks` | `jwks.Name` | Selectors: `app=nginx-jwks`, `jwks-config=example-app-jwks` |

**Ключевой момент:** Все имена ресурсов и значения `jwks-config` label берутся из имени JWKS Custom Resource (`jwks.Name`).

