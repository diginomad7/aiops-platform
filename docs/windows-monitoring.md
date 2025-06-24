# Windows Server Мониторинг

## Обзор

Данный документ описывает настройку мониторинга Windows Server (192.168.0.5) с использованием Prometheus и Grafana в рамках платформы AIOps. Мониторинг включает в себя основные метрики: доступность, загрузка CPU, использование RAM, заполнение дисков и сетевой трафик.

## Архитектура

Мониторинг Windows Server основан на следующей архитектуре:

```
Windows Server (192.168.0.5) -> Windows Exporter (порт 9182) -> Prometheus -> Grafana
```

- **Windows Exporter**: Агент, установленный на Windows Server, который собирает метрики и экспортирует их в формате Prometheus
- **Prometheus**: Собирает метрики с Windows Exporter через HTTP
- **Grafana**: Визуализирует собранные метрики в виде дашбордов

## Установка Windows Exporter

### Предварительные требования

- Windows Server 2016/2019/2022
- Доступ администратора
- Сетевой доступ между Windows Server и Kubernetes кластером

### Шаги установки

1. Скачать установщик Windows Exporter с GitHub:

```powershell
# Вариант 1: Стандартный способ
$url = "https://github.com/prometheus-community/windows_exporter/releases/download/v0.25.0/windows_exporter-0.25.0-amd64.msi"
$output = "$env:TEMP\windows_exporter.msi"
Invoke-WebRequest -Uri $url -OutFile $output

# Вариант 2: Если возникают проблемы с SSL/TLS
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Invoke-WebRequest -Uri $url -OutFile $output

# Вариант 3: Через .NET WebClient (альтернативный способ)
$client = New-Object System.Net.WebClient
$client.DownloadFile($url, $output)
```

2. Если все способы выше не работают, скачайте файл вручную через браузер с URL:
   - https://github.com/prometheus-community/windows_exporter/releases/download/v0.25.0/windows_exporter-0.25.0-amd64.msi

3. Установить Windows Exporter как службу Windows:

```powershell
Start-Process msiexec.exe -ArgumentList "/i $output ENABLED_COLLECTORS=cpu,memory,logical_disk,os,system,net,tcp LISTEN_PORT=9182 /quiet" -Wait
```

4. Проверить статус службы:

```powershell
Get-Service windows_exporter
```

5. Настроить брандмауэр Windows для разрешения входящих подключений к порту 9182:

```powershell
New-NetFirewallRule -DisplayName "Windows Exporter" -Direction Inbound -LocalPort 9182 -Protocol TCP -Action Allow
```

6. Проверить доступность метрик:

```powershell
Invoke-WebRequest -Uri "http://localhost:9182/metrics"
```

## Конфигурация Prometheus

Для сбора метрик с Windows Server в Prometheus были созданы следующие конфигурационные файлы:

### 1. windows-scrape-config.yaml

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-windows-scrape
  namespace: monitoring
data:
  windows-scrape.yaml: |
    - job_name: 'windows-server'
      scrape_interval: 30s
      static_configs:
        - targets: ['192.168.0.5:9182']
          labels:
            instance: 'windows-server-prod'
            os: 'windows'
```

### 2. Обновление prometheus-values-final.yaml

В файл `prometheus-values-final.yaml` добавлена конфигурация для сбора метрик с Windows Server:

```yaml
additionalScrapeConfigs:
  - job_name: 'windows-server'
    scrape_interval: 30s
    static_configs:
      - targets: ['192.168.0.5:9182']
        labels:
          instance: 'windows-server-prod'
          os: 'windows'
```

Также добавлен маршрут для оповещений Windows в AlertManager:

```yaml
routes:
- match:
    context: windows
  receiver: 'windows-team'
receivers:
- name: 'windows-team'
  # Конфигурация получателя оповещений
```

## Grafana Дашборд

Для визуализации метрик Windows Server создан специальный дашборд Grafana, который включает следующие панели:

1. **CPU Usage**: Отображает процент использования CPU
2. **Memory Usage**: Отображает процент использования оперативной памяти
3. **Disk Usage**: Отображает процент заполнения дисков C: и D:
4. **Network Traffic**: Отображает входящий и исходящий сетевой трафик

Дашборд автоматически загружается в Grafana благодаря механизму auto-discovery через ConfigMap с меткой `grafana_dashboard: "1"`.

## Оповещения

Настроены следующие оповещения для Windows Server:

| Оповещение | Описание | Выражение | Длительность | Серьезность |
|------------|----------|-----------|--------------|-------------|
| WindowsServerDown | Windows Server недоступен | `up{job="windows-server"} == 0` | 5m | critical |
| WindowsHighCPUUsage | Высокая загрузка CPU | `100 - (avg by (instance) (rate(windows_cpu_time_total{mode="idle"}[2m])) * 100) > 90` | 10m | warning |
| WindowsHighMemoryUsage | Высокая загрузка памяти | `100 * (1 - ((windows_memory_available_bytes) / (windows_os_physical_memory_free_bytes + windows_os_physical_memory_used_bytes))) > 90` | 10m | warning |
| WindowsLowDiskSpace | Низкий объем свободного места на диске | `100 - (windows_logical_disk_free_bytes / windows_logical_disk_size_bytes * 100) > 90` | 10m | warning |

## Метрики Windows Exporter

### Основные метрики

| Метрика | Описание | PromQL запрос |
|---------|----------|--------------|
| CPU Usage | Загрузка процессора в процентах | `100 - (avg by (instance) (rate(windows_cpu_time_total{mode="idle"}[2m])) * 100)` |
| Memory Usage | Использование памяти в процентах | `100 * (1 - ((windows_memory_available_bytes) / (windows_os_physical_memory_free_bytes + windows_os_physical_memory_used_bytes)))` |
| Disk Usage | Использование дискового пространства | `100 - (windows_logical_disk_free_bytes / windows_logical_disk_size_bytes * 100)` |
| Network Traffic | Сетевой трафик | `rate(windows_net_bytes_received_total[5m])`, `rate(windows_net_bytes_sent_total[5m])` |

### Дополнительные метрики

Windows Exporter предоставляет множество дополнительных метрик, которые можно включить при установке через параметр `ENABLED_COLLECTORS`. Полный список доступных коллекторов можно найти в [официальной документации](https://github.com/prometheus-community/windows_exporter).

## Устранение неполадок

### Windows Exporter не запускается

1. Проверьте логи службы в Event Viewer
2. Убедитесь, что установлены все необходимые зависимости
3. Проверьте права доступа для службы

### Проблемы с SSL/TLS при скачивании

Если вы сталкиваетесь с ошибкой "Не удалось создать защищенный канал SSL/TLS":

1. Обновите настройки TLS в PowerShell:
   ```powershell
   [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
   ```

2. Если это не помогает, используйте альтернативный метод скачивания:
   ```powershell
   $client = New-Object System.Net.WebClient
   $client.DownloadFile($url, $output)
   ```

3. В крайнем случае, скачайте файл вручную через браузер и перенесите его на сервер.

### Prometheus не видит Windows Exporter

1. Проверьте сетевую доступность с Prometheus до Windows Server по порту 9182:
   ```bash
   kubectl exec -it -n monitoring prometheus-prometheus-kube-prometheus-prometheus-0 -- wget -T 5 -O- http://192.168.0.5:9182/metrics
   ```

2. Убедитесь, что брандмауэр Windows разрешает входящие соединения
3. Проверьте конфигурацию scrape_configs в Prometheus

### Дашборд не отображает данные

1. Проверьте, что источник данных Prometheus правильно настроен в Grafana
2. Убедитесь, что метрики Windows доступны в Prometheus:
   ```bash
   kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
   # Открыть http://localhost:9090/graph и выполнить запрос:
   # up{job="windows-server"}
   ```
3. Проверьте запросы PromQL в панелях дашборда

## Дальнейшие улучшения

1. **Мониторинг служб Windows**: Добавить мониторинг состояния критичных служб Windows
2. **Мониторинг IIS/SQL Server**: Настроить специфичные метрики для IIS или SQL Server, если они используются
3. **Автоматическое восстановление**: Интегрировать с системой автоматического восстановления для реагирования на оповещения
4. **Расширенный анализ производительности**: Добавить метрики производительности приложений и более детальные метрики системы