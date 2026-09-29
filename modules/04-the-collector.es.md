# Módulo 4: El Collector, y Dónde Acaban los Datos

**Duración:** ~45 min · **Checkpoints: 42** · **Igual para los dos tracks**

El Collector es la caja que todo el mundo asiente al ver y nadie lee. Al terminar este módulo lo habrás leído línea a línea, lo habrás editado, lo habrás roto a propósito y habrás encontrado la rotura en su propia telemetría. 📦

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Leer una configuración de OpenTelemetry Collector y decir qué hace cada sección
- Añadir un processor a un pipeline con soltura — lo vas a necesitar otra vez en el Módulo 8
- Entender qué son los stream fields de VictoriaLogs y por qué elegirlos mal sale caro
- Saber si tu propio pipeline de telemetría está sano

## Prerrequisitos

- Módulo 3 completado y con checkpoint — la telemetría está fluyendo

## Ejercicio 4.1 — Lee la configuración [~20 min]

**Deberías poder responder esto al terminar:** *si se pierde un span entre tu app y VictoriaTraces, ¿en qué tres sitios mirarías?*

### Hazlo

Abre `collector/config.yaml` y sigue la explicación. El fichero entero:

```yaml
receivers:
  otlp:
    protocols:
      grpc: { endpoint: 127.0.0.1:4317 }
      http: { endpoint: 127.0.0.1:4318 }

  # The Collector scrapes its own metrics and ships them through its own
  # metrics pipeline. Module 4.2 depends on this: it is how you find out that
  # the pipeline itself has stopped delivering.
  prometheus/self:
    config:
      scrape_configs:
        - job_name: collector
          scrape_interval: 10s
          static_configs:
            - targets: [127.0.0.1:8888]

processors:
  batch: {}
  memory_limiter: { check_interval: 1s, limit_mib: 512 }
  resource:
    attributes:
      - { key: deployment.environment, value: workshop, action: upsert }

  # The JS and Python SDKs disagree on the case of severity_text ("info" vs
  # "INFO"). Normalising it here, once, is cheaper than every query and every
  # dashboard having to cope with both. This is what the Collector is for.
  transform/severity:
    log_statements:
      - context: log
        statements:
          - set(severity_text, ConvertCase(severity_text, "upper")) where severity_text != ""

exporters:
  otlphttp/victoriametrics:
    metrics_endpoint: http://localhost:8428/opentelemetry/v1/metrics
    encoding: proto
    compression: gzip
  otlphttp/victorialogs:
    logs_endpoint: http://localhost:9428/insert/opentelemetry/v1/logs
    headers:
      VL-Stream-Fields: "service.name,deployment.environment"
  otlphttp/victoriatraces:
    traces_endpoint: http://localhost:10428/insert/opentelemetry/v1/traces

extensions:
  health_check:
    endpoint: 127.0.0.1:13133

service:
  extensions: [health_check]
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: 127.0.0.1
                port: 8888
  pipelines:
    traces:  { receivers: [otlp], processors: [memory_limiter, resource, batch], exporters: [otlphttp/victoriatraces] }
    metrics: { receivers: [otlp, prometheus/self], processors: [memory_limiter, resource, batch], exporters: [otlphttp/victoriametrics] }
    logs:    { receivers: [otlp], processors: [memory_limiter, resource, transform/severity, batch], exporters: [otlphttp/victorialogs] }
```

Cuatro secciones, y la cuarta es la que importa:

**`receivers`** — cómo entra la telemetría. Un receiver OTLP, dos protocolos. Tu app habla con `:4318` porque configuraste `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`.

**`processors`** — qué pasa por el camino. Los nuestros hacen tres cosas:
- `memory_limiter` rechaza trabajo en vez de morir por OOM. Ponlo **el primero** en todos los pipelines; un Collector que muere se lleva tu visibilidad justo cuando la necesitas.
- `resource` estampa `deployment.environment` en todo, para que la telemetría de una máquina que se olvidó de ponerlo siga siendo atribuible.
- `transform/severity` pasa `severity_text` a mayúsculas. Está ahí porque el SDK de JS emite `info` y el de Python emite `INFO`, y normalizar una vez en el medio sale más barato que hacer que todas las consultas y todos los dashboards aguanten las dos formas. Aquí es donde el Collector se gana el sueldo.
- `batch` agrupa registros antes de exportar. Siempre el último.

**`exporters`** — a dónde va. Son tres, y fíjate bien en *por qué*: tres hosts distintos, tres rutas distintas, una de ellas necesitando una cabecera especial. **Esta es la razón de que el Collector exista en esta arquitectura.** Tu app conoce un endpoint; el Collector conoce estos tres.

**`service.pipelines`** — el cableado. Nada de lo declarado arriba está activo hasta que aparece en un pipeline. Esto pilla a todo el mundo al menos una vez: añades un processor, reinicias y no cambia nada, porque nunca lo metiste en un pipeline.

Así que, respondiendo a la pregunta del ejercicio: un span que falta puede ser (1) que la app nunca lo exportó, (2) que se descartó o se enrutó mal en el pipeline del Collector, o (3) que el endpoint del exporter lo rechazó. Aprenderás a distinguirlos en 4.2.

### Tres puntos que merecen una pausa

**`VL-Stream-Fields` es específico de VictoriaLogs y sí importa.** Define qué es un *stream*. VictoriaLogs guarda los logs agrupados por stream, así que esta elección condiciona tanto el diseño de almacenamiento como el rendimiento de las consultas. `service.name,deployment.environment` te da un puñado de streams — bien. Meter ahí algo de cardinalidad alta, como `trace_id`, te daría un stream por petición y degradaría VictoriaLogs seriamente. Lo vas a poder sentir en el Módulo 6.3.

**VictoriaMetrics quiere temporalidad cumulative.** Los dos SDKs usan cumulative por defecto, así que estás bien tal cual. Si alguna vez acabas con métricas delta (algunas configuraciones de SDK de ciertos proveedores van así por defecto), el arreglo es el processor `deltatocumulative` en el Collector, no un cambio en la app.

**VictoriaMetrics arranca con `-opentelemetry.usePrometheusNaming`.** Es uno de los flags que escribiste en el Módulo 0.1. Sin ese flag, VM guarda los nombres con puntos de OTel tal cual — `http.server.duration_count` — y tus consultas necesitan comillas. Con él obtienes la forma Prometheus de toda la vida: `http_server_duration_milliseconds_count`, y `service.name` pasa a ser la etiqueta `service_name`. El Módulo 5.2 lo aprovecha.

**Los nombres de endpoint son por señal, no genéricos.** `metrics_endpoint`, `logs_endpoint`, `traces_endpoint`. Si usas la clave genérica `endpoint`, el exporter añade su propia ruta convencional y acabarás con 404 que parecen problemas de conectividad.

### Comprueba en la UI de Victoria

Confirma que el processor `resource` está haciendo su trabajo. vmui de VictoriaLogs:

```
service.name:todo-api AND deployment.environment:workshop
```

Deberías ver: los mismos registros que con `service.name:todo-api` a secas. Ninguno ha perdido el atributo de entorno.

### Comprueba en Grafana

Grafana → Explore → VictoriaMetrics:

```
otelcol_receiver_accepted_spans
```

El Collector informa sobre sí mismo. Fíjate en cómo sube el número.

### Para pensar

`batch` va el último en todos los pipelines y `memory_limiter` el primero. ¿Por qué el orden no es arbitrario?

### Si te atascas

- **Editar la configuración no hace nada.** El Collector lee su configuración una sola vez, al arrancar. Vete a su terminal: **Ctrl-C, y luego ↑ Enter**. Vas a hacer esto una docena de veces hoy, que es justo por lo que en el Módulo 0.1 le dimos una terminal para él solo.
- **El Collector no arranca tras una edición.** Es YAML, y va a ser la indentación. Nombra la línea problemática mientras se muere — y como está en primer plano, no te lo puedes perder.

## Ejercicio 4.2 — Edítalo [~25 min] [checkpoint: 42] *(recortable en la versión de un día)*

**Deberías poder responder esto al terminar:** *¿cómo sabrías, ahora mismo, si tu Collector hubiera dejado de entregar spans?*

### Hazlo

Tres tareas, de menos a más valor.

**Tarea 1 — Descarta un atributo ruidoso.**

Añade un processor `attributes` y mételo en el pipeline de trazas:

```yaml
processors:
  attributes/clean:
    actions:
      - { key: net.peer.ip, action: delete }
```

```yaml
service:
  pipelines:
    traces:
      processors: [memory_limiter, resource, attributes/clean, batch]
```

Reinicia el Collector y confirma que el atributo ya no está en los spans nuevos.

**Tarea 2 — Filtra los health checks.**

```yaml
processors:
  filter/healthz:
    error_mode: ignore
    traces:
      span:
        - 'attributes["url.path"] == "/healthz"'
```

Añade `filter/healthz` al pipeline de trazas, reinicia y confirma que los spans de `/healthz` dejan de llegar.

Y ahora ten la discusión: acabas de hacer invisible tu sonda de disponibilidad. Si el endpoint de salud empieza a fallar, ¿se va a enterar alguien por este sistema? Debate con quien tengas al lado si ese filtro debería llegar a producción.

**Tarea 3 — Rómpelo a propósito. Esta es la importante.** 💥

Apunta un exporter a un puerto en el que no escucha nadie:

```yaml
exporters:
  otlphttp/victoriatraces:
    traces_endpoint: http://victoriatraces:19999/insert/opentelemetry/v1/traces
```

Reinicia el Collector. Lanza el loadgen. Las trazas dejan de llegar a vmui — pero fíjate en que **tu aplicación está tan contenta**. Ni errores, ni lentitud, ni nada en sus logs. Así se ve un pipeline roto en silencio desde fuera, y es exactamente así como equipos reales pierden una semana de datos.

Ahora encuéntralo. El Collector expone sus propias métricas en `:8888`, y ya se están
recogiendo en VictoriaMetrics. La señal más clara es **entra dato y no sale** — pon estas
dos en una misma gráfica en vmui:

```
rate(otelcol_receiver_accepted_spans_total[1m])
```

```
rate(otelcol_exporter_sent_spans_total[1m])
```

La primera sigue subiendo. La segunda cae a cero. Después mira dónde se están acumulando
los spans:

```
otelcol_exporter_queue_size
```

```
otelcol_exporter_queue_capacity
```

Observa cómo crece la cola del exporter de trazas mientras las otras dos se quedan a cero.
Luego arregla el puerto, reinicia y mira cómo se vacía.

> 📌 Puede que esperases `otelcol_exporter_send_failed_spans_total` aquí, que es la métrica a la
> que todo el mundo va primero. Se queda a cero un buen rato: el Collector *reintenta*, así
> que todavía no se ha perdido nada de forma permanente. Solo empieza a contar cuando la
> cola se llena y de verdad se descartan datos. Es una distinción realmente útil:
> **"está fallando" y "está perdiendo datos" son dos alertas distintas con urgencias
> distintas**, y la cola es lo que te compra el tiempo que hay entre las dos.

### Comprueba en la UI de Victoria

vmui de VictoriaMetrics, las dos consultas de arriba. Durante la ventana rota deberías ver: `otelcol_exporter_send_failed_spans_total` subiendo de forma sostenida, y `otelcol_exporter_queue_size` acercándose a su límite. Tras el arreglo, la tasa de fallos se aplana.

### Comprueba en Grafana

Construye un panel pequeño con las dos series y guárdalo. En serio — un panel de "¿está sano mi pipeline de telemetría?" es de los dashboards más valiosos que vas a construir en tu vida, y casi nadie tiene uno.

### Para pensar

> **El pipeline de observabilidad es en sí mismo un sistema de producción, y necesita ser observable.**

La mayoría de los equipos aprende esto por las malas, normalmente durante un incidente, normalmente descubriendo que la razón por la que no ven nada es la cosa que les enseña las cosas. ¿Sobre qué alertarías, y por qué despertarías a una persona?

### Si te atascas

- **El processor `filter` se rechaza al arrancar.** La sintaxis OTTL es quisquillosa: `attributes["url.path"]` con comillas dobles dentro de los corchetes, y toda la condición entre comillas simples.
- **`otelcol_exporter_send_failed_spans_total` no devuelve absolutamente nada.** Es lo esperado — mira la nota de arriba. Usa la comparación entre las tasas del receiver y del exporter.
- **Todas las métricas están planas, incluidas las del propio Collector.** Puede que hayas roto la configuración tan a fondo que el Collector no llegó a arrancar. Mira su terminal — un Collector muerto no exporta nada, incluidas sus propias métricas de fallo. Eso es una lección en sí misma, y es la razón de que merezca la pena saberse `curl -s localhost:13133/` de memoria.

```bash
./checkpoint.sh 42
```

## Lo Que Hemos Conseguido

- Sabes leer una configuración de Collector y decir qué hace cada sección
- Has añadido processors a un pipeline y visto el efecto
- Has vivido un pipeline roto en silencio y lo has diagnosticado desde las métricas del propio Collector
- Sabes por qué `VL-Stream-Fields` es una decisión y no un formalismo

## Siguientes Pasos

La fontanería está entendida. Toca añadir significado, empezando por los números. Continúa con el [Módulo 5: Métricas Manuales](./05-manual-metrics.es.md).
