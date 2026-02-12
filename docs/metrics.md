# Метрики JWKS Operator

## Обзор

JWKS Operator предоставляет метрики Prometheus для мониторинга работы оператора. Все метрики доступны через HTTP endpoint `/metrics` на порту 8080 (по умолчанию).

## Доступ к метрикам

### Endpoint

```
http://<operator-pod-ip>:8080/metrics
```

### В Kubernetes

```bash
# Порт-форвард для доступа к метрикам
kubectl port-forward -n system deployment/jwks-operator 8080:8080

# Проверить метрики
curl http://localhost:8080/metrics
```

### ServiceMonitor для Prometheus Operator

Если используется Prometheus Operator, создайте ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: jwks-operator
  namespace: system
spec:
  selector:
    matchLabels:
      app: jwks-operator
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

## Метрики

### 1. jwks_operator_reconcile_total

**Тип**: Counter  
**Описание**: Общее количество попыток реконсиляции  
**Метки**:
- `result` - результат реконсиляции: `success` или `error`

**Пример**:
```
jwks_operator_reconcile_total{result="success"} 150
jwks_operator_reconcile_total{result="error"} 5
```

**Использование**:
- Мониторинг общего количества реконсиляций
- Отслеживание успешных и неудачных реконсиляций
- Расчет rate реконсиляций

**PromQL запросы**:
```promql
# Rate успешных реконсиляций за последние 5 минут
rate(jwks_operator_reconcile_total{result="success"}[5m])

# Rate ошибок реконсиляции
rate(jwks_operator_reconcile_total{result="error"}[5m])

# Процент успешных реконсиляций
sum(rate(jwks_operator_reconcile_total{result="success"}[5m])) / 
sum(rate(jwks_operator_reconcile_total[5m])) * 100
```

### 2. jwks_operator_reconcile_duration_seconds

**Тип**: Histogram  
**Описание**: Длительность реконсиляции в секундах  
**Метки**:
- `result` - результат реконсиляции: `success` или `error`

**Buckets**: Стандартные Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Пример**:
```
jwks_operator_reconcile_duration_seconds_bucket{result="success",le="0.005"} 10
jwks_operator_reconcile_duration_seconds_bucket{result="success",le="0.01"} 25
jwks_operator_reconcile_duration_seconds_bucket{result="success",le="0.025"} 50
jwks_operator_reconcile_duration_seconds_sum{result="success"} 12.5
jwks_operator_reconcile_duration_seconds_count{result="success"} 150
```

**Использование**:
- Мониторинг производительности реконсиляции
- Выявление медленных реконсиляций
- Анализ распределения времени выполнения

**PromQL запросы**:
```promql
# Средняя длительность реконсиляции
rate(jwks_operator_reconcile_duration_seconds_sum[5m]) / 
rate(jwks_operator_reconcile_duration_seconds_count[5m])

# 95-й перцентиль длительности
histogram_quantile(0.95, 
  rate(jwks_operator_reconcile_duration_seconds_bucket[5m])
)

# 99-й перцентиль длительности
histogram_quantile(0.99, 
  rate(jwks_operator_reconcile_duration_seconds_bucket[5m])
)
```

### 3. jwks_operator_configmap_updates_total

**Тип**: Counter  
**Описание**: Общее количество обновлений ConfigMap  
**Метки**:
- `type` - тип ConfigMap: `jwks` или `nginx`
- `result` - результат обновления: `success` или `error`

**Пример**:
```
jwks_operator_configmap_updates_total{type="jwks",result="success"} 200
jwks_operator_configmap_updates_total{type="jwks",result="error"} 2
jwks_operator_configmap_updates_total{type="nginx",result="success"} 50
jwks_operator_configmap_updates_total{type="nginx",result="error"} 1
```

**Использование**:
- Мониторинг обновлений ConfigMap
- Отслеживание ошибок обновления
- Разделение по типам ConfigMap

**PromQL запросы**:
```promql
# Rate обновлений JWKS ConfigMap
rate(jwks_operator_configmap_updates_total{type="jwks"}[5m])

# Rate ошибок обновления ConfigMap
rate(jwks_operator_configmap_updates_total{result="error"}[5m])

# Общее количество обновлений по типам
sum by (type) (jwks_operator_configmap_updates_total)
```

### 4. jwks_operator_jwks_generation_total

**Тип**: Counter  
**Описание**: Общее количество попыток генерации JWKS  
**Метки**:
- `result` - результат генерации: `success` или `error`

**Пример**:
```
jwks_operator_jwks_generation_total{result="success"} 200
jwks_operator_jwks_generation_total{result="error"} 3
```

**Использование**:
- Мониторинг генерации JWKS
- Отслеживание ошибок генерации
- Расчет успешности генерации

**PromQL запросы**:
```promql
# Rate генерации JWKS
rate(jwks_operator_jwks_generation_total[5m])

# Процент успешных генераций
sum(rate(jwks_operator_jwks_generation_total{result="success"}[5m])) / 
sum(rate(jwks_operator_jwks_generation_total[5m])) * 100
```

### 5. jwks_operator_nginx_operations_total

**Тип**: Counter  
**Описание**: Общее количество операций с nginx ресурсами  
**Метки**:
- `operation` - тип операции: `deployment`, `service`, `config`
- `result` - результат операции: `success` или `error`

**Пример**:
```
jwks_operator_nginx_operations_total{operation="deployment",result="success"} 50
jwks_operator_nginx_operations_total{operation="deployment",result="error"} 1
jwks_operator_nginx_operations_total{operation="service",result="success"} 50
jwks_operator_nginx_operations_total{operation="config",result="success"} 100
```

**Использование**:
- Мониторинг операций с nginx
- Отслеживание ошибок по типам операций
- Анализ работы nginx компонента

**PromQL запросы**:
```promql
# Rate операций по типам
sum by (operation) (rate(jwks_operator_nginx_operations_total[5m]))

# Rate ошибок nginx операций
rate(jwks_operator_nginx_operations_total{result="error"}[5m])

# Успешность операций по типам
sum by (operation) (rate(jwks_operator_nginx_operations_total{result="success"}[5m])) / 
sum by (operation) (rate(jwks_operator_nginx_operations_total[5m])) * 100
```

### 6. jwks_operator_jwks_verification_total

**Тип**: Counter  
**Описание**: Общее количество попыток верификации JWKS  
**Метки**:
- `result` - результат верификации: `success` или `error`

**Пример**:
```
jwks_operator_jwks_verification_total{result="success"} 1000
jwks_operator_jwks_verification_total{result="error"} 5
```

**Использование**:
- Мониторинг верификации JWKS
- Отслеживание ошибок верификации
- Проверка работоспособности JWKS endpoint

**PromQL запросы**:
```promql
# Rate верификации JWKS
rate(jwks_operator_jwks_verification_total[5m])

# Процент успешных верификаций
sum(rate(jwks_operator_jwks_verification_total{result="success"}[5m])) / 
sum(rate(jwks_operator_jwks_verification_total[5m])) * 100

# Количество ошибок верификации за последний час
increase(jwks_operator_jwks_verification_total{result="error"}[1h])
```

### 7. jwks_operator_errors_total

**Тип**: Counter  
**Описание**: Общее количество ошибок по типам  
**Метки**:
- `type` - тип ошибки:
  - `jwks_nil` - JWKS ресурс равен nil
  - `secret_not_found` - Secret не найден
  - `jwks_generation_failed` - Ошибка генерации JWKS
  - `configmap_update_failed` - Ошибка обновления ConfigMap
  - `nginx_config_update_failed` - Ошибка обновления nginx конфигурации
  - `nginx_deployment_failed` - Ошибка создания/обновления nginx Deployment
  - `nginx_service_failed` - Ошибка создания/обновления nginx Service

**Пример**:
```
jwks_operator_errors_total{type="secret_not_found"} 5
jwks_operator_errors_total{type="jwks_generation_failed"} 2
jwks_operator_errors_total{type="configmap_update_failed"} 1
```

**Использование**:
- Мониторинг ошибок по типам
- Выявление проблемных областей
- Анализ частоты ошибок

**PromQL запросы**:
```promql
# Rate ошибок по типам
sum by (type) (rate(jwks_operator_errors_total[5m]))

# Топ-5 типов ошибок
topk(5, rate(jwks_operator_errors_total[5m]))

# Общее количество ошибок за последний час
sum(increase(jwks_operator_errors_total[1h]))
```

## Дашборды Grafana

### Пример дашборда

```json
{
  "dashboard": {
    "title": "JWKS Operator",
    "panels": [
      {
        "title": "Reconciliation Rate",
        "targets": [{
          "expr": "rate(jwks_operator_reconcile_total[5m])"
        }]
      },
      {
        "title": "Reconciliation Duration (95th percentile)",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(jwks_operator_reconcile_duration_seconds_bucket[5m]))"
        }]
      },
      {
        "title": "ConfigMap Updates",
        "targets": [{
          "expr": "rate(jwks_operator_configmap_updates_total[5m])"
        }]
      },
      {
        "title": "JWKS Verification Success Rate",
        "targets": [{
          "expr": "sum(rate(jwks_operator_jwks_verification_total{result=\"success\"}[5m])) / sum(rate(jwks_operator_jwks_verification_total[5m])) * 100"
        }]
      },
      {
        "title": "Errors by Type",
        "targets": [{
          "expr": "sum by (type) (rate(jwks_operator_errors_total[5m]))"
        }]
      }
    ]
  }
}
```

## Алерты

### Примеры правил алертинга

```yaml
groups:
- name: jwks_operator
  interval: 30s
  rules:
  - alert: JWKSOperatorHighErrorRate
    expr: rate(jwks_operator_reconcile_total{result="error"}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "JWKS Operator has high error rate"
      description: "Error rate is {{ $value }} errors/sec"

  - alert: JWKSOperatorReconciliationSlow
    expr: histogram_quantile(0.95, rate(jwks_operator_reconcile_duration_seconds_bucket[5m])) > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "JWKS Operator reconciliation is slow"
      description: "95th percentile duration is {{ $value }}s"

  - alert: JWKSOperatorConfigMapUpdateFailed
    expr: rate(jwks_operator_configmap_updates_total{result="error"}[5m]) > 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "JWKS Operator ConfigMap update failed"
      description: "ConfigMap update errors detected"

  - alert: JWKSOperatorJWKSVerificationFailed
    expr: rate(jwks_operator_jwks_verification_total{result="error"}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "JWKS verification is failing"
      description: "JWKS verification error rate is {{ $value }} errors/sec"

  - alert: JWKSOperatorHighErrorCount
    expr: sum(increase(jwks_operator_errors_total[1h])) > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "JWKS Operator has many errors"
      description: "Total errors in last hour: {{ $value }}"
```

## Интеграция с Prometheus

### Scrape конфигурация

```yaml
scrape_configs:
  - job_name: 'jwks-operator'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        action: keep
        regex: jwks-operator
      - source_labels: [__meta_kubernetes_pod_ip]
        action: replace
        target_label: __address__
        replacement: $1:8080
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: kubernetes_pod_name
```

## Мониторинг производительности

### Ключевые метрики для мониторинга

1. **Reconciliation Rate** - частота реконсиляций
2. **Reconciliation Duration** - время выполнения реконсиляции
3. **Error Rate** - частота ошибок
4. **ConfigMap Update Success Rate** - успешность обновлений ConfigMap
5. **JWKS Verification Success Rate** - успешность верификации JWKS

### Рекомендуемые пороги

- **Reconciliation Duration (95th percentile)**: < 5 секунд
- **Error Rate**: < 0.1 ошибок/сек
- **ConfigMap Update Success Rate**: > 99%
- **JWKS Verification Success Rate**: > 99%

## Отладка

### Полезные запросы для отладки

```promql
# Последние ошибки
topk(10, jwks_operator_errors_total)

# Тренд реконсиляций
rate(jwks_operator_reconcile_total[1h])

# Распределение длительности реконсиляции
histogram_quantile(0.50, rate(jwks_operator_reconcile_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(jwks_operator_reconcile_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(jwks_operator_reconcile_duration_seconds_bucket[5m]))

# Детализация ошибок
sum by (type) (increase(jwks_operator_errors_total[1h]))
```

## Связанные документы

- [Архитектура](architecture.md) - общая архитектура оператора
- [Конфигурация](configuration.md) - настройка оператора
- [Развертывание](deployment.md) - развертывание оператора

