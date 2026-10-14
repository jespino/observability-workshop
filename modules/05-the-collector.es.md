# Módulo 5: El OpenTelemetry Collector

En el Módulo 4 tu aplicación empezó a enviar las tres señales a una sola dirección,
`otel-collector:4318`, y acabaron en tres backends distintos. Este módulo va de lo que hay
en esa dirección: qué es el OpenTelemetry Collector, qué puede hacer y qué está haciendo ya
con tu telemetría el de este taller. 📦

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Saber qué es el OpenTelemetry Collector y dónde se sitúa entre tu aplicación y tus
  backends
- Conocer sus piezas — receivers, processors, exporters, connectors y pipelines
- Conocer el tipo de problemas que resuelve
- Decir qué hace el Collector de este taller con tus métricas, logs y trazas

## 5.1 — Qué es el Collector

El **OpenTelemetry Collector** es un programa independiente, parte del proyecto
OpenTelemetry, que recibe telemetría, la procesa y la reenvía. No está atado a ningún
proveedor: habla OTLP, el protocolo de OpenTelemetry, y decenas de formatos más, de
entrada y de salida.

Tu aplicación no lo necesita — un SDK puede exportar directamente a un backend. Pero
poner un Collector en medio cambia quién tiene que saber qué:

![todo-api y notify-service, tus servicios, envían su telemetría por OTLP al OTel Collector, que manda las métricas a VictoriaMetrics, los logs a VictoriaLogs y las trazas a VictoriaTraces](images/collector-overview.png)

- **Tu aplicación conoce un solo endpoint.** Lo envía todo por OTLP al Collector y se
  olvida. No sabe qué backends existen, dónde están ni qué espera cada uno.
- **El Collector sabe todo lo demás.** Qué backend recibe cada señal, en qué URL, con qué
  cabeceras, y qué cambiar por el camino. Cambia un backend, añade un segundo o empieza a
  descartar algo ruidoso, y no hay que volver a desplegar ninguna aplicación.

## 5.2 — Las piezas

Un Collector se configura a partir de unos pocos tipos de componente:

| Componente | Qué hace | Ejemplos |
|---|---|---|
| **Receivers** | Meten la telemetría — escuchando o yendo a buscarla | OTLP, scraping de Prometheus, ficheros de log, Kafka |
| **Processors** | Cambian la telemetría a su paso | batch, filtrar, añadir o quitar atributos, sampling, redacción |
| **Exporters** | Sacan la telemetría | OTLP a cualquier backend compatible, Prometheus remote write, Kafka, ficheros |
| **Connectors** | Unen dos pipelines: son exporter en uno y receiver en el otro | convertir spans en métricas, enrutar por atributo |
| **Extensions** | Lo que no tiene que ver con los datos en sí | health checks, autenticación |

Los componentes se conectan en **pipelines**, uno por señal. Un pipeline dice: coge las
métricas (o los logs, o las trazas) de estos receivers, pásalas por estos processors en
este orden y entrégaselas a estos exporters. Un componente declarado pero que no está en
ningún pipeline no hace nada.

## 5.3 — Qué puede hacer

Casi todo lo que hace que merezca la pena tener un Collector cae en unos pocos grupos:

- **Enrutar.** Mandar cada señal al backend que la guarda, o la misma señal a dos
  backends a la vez — útil durante una migración.
- **Enriquecer.** Añadir atributos que la aplicación no conoce: el entorno, el clúster, la
  región.
- **Normalizar.** Hacer que la telemetría de SDKs, lenguajes y versiones distintos tenga la
  misma forma antes de guardarse, para que las consultas y los dashboards solo tengan que
  tratar con una.
- **Reducir.** Descartar lo que no necesitas — spans de health checks, logs de debug, un
  atributo que nadie consulta — antes de pagar por guardarlo. Muestrear trazas, quedándote
  con las interesantes.
- **Proteger.** Quitar datos sensibles, como emails o tokens, en un único sitio en vez de
  en cada aplicación.
- **Derivar.** Producir telemetría nueva a partir de la existente, como la tasa de
  peticiones, los errores y la duración calculados a partir de los spans.
- **Dar fiabilidad.** Agrupar la telemetría para ser eficiente, reintentar cuando un
  backend está caído, encolar mientras se recupera, y rechazar trabajo antes que quedarse
  sin memoria.
- **Traducir.** Aceptar telemetría en un formato y enviarla en otro — hacer scraping de
  endpoints de Prometheus, leer ficheros de log, recibir Jaeger o Zipkin.

Se suele desplegar de dos formas, a menudo las dos a la vez: como **agente**, junto a cada
aplicación, en el mismo host o en el mismo pod, y como **gateway**, un grupo central al que
envían los agentes y las aplicaciones.

## 5.4 — Qué hace el Collector de este taller

El Collector al que llevas enviando desde el Módulo 4 se configura en
`collector/config.yaml`. No necesitas leerlo ni cambiarlo; esto es lo que hace.

![Dentro del Collector de este taller: tu aplicación envía OTLP al receiver otlp, que lo reparte en tres pipelines. Las métricas pasan por memory_limiter, resource y batch; los logs por memory_limiter, resource, transform/severity y batch; las trazas por memory_limiter, resource y batch. Cada pipeline termina en su propio exporter, hacia VictoriaMetrics, VictoriaLogs y VictoriaTraces. La extensión health_check queda aparte, fuera del camino de los datos](images/collector-pipelines.png)

**Receivers**
- **OTLP**, en el puerto 4318 (HTTP) y 4317 (gRPC). Es donde `todo-api` y
  `notify-service` envían sus métricas, logs y trazas.
- **Sus propias métricas.** El Collector informa sobre sí mismo — qué ha recibido, qué ha
  enviado, cómo de llenas están sus colas — y hace scraping de esos números hacia su
  propio pipeline de métricas, así que acaban en VictoriaMetrics como cualquier otra
  métrica. La fila **Telemetry pipeline** del dashboard Auto-Instrumentation de Grafana se
  construye con ellas.

**Processors**, aplicados en este orden:
- **`memory_limiter`** — hace que el Collector rechace datos cuando está cerca de su
  límite de memoria, en vez de caerse y perderlo todo.
- **`resource`** — añade `deployment.environment=workshop` a toda la telemetría, para que
  cada métrica, log y span diga de qué entorno viene, aunque la aplicación se haya olvidado
  de ponerlo.
- **`transform/severity`** — solo para logs. El SDK de JavaScript da los niveles en
  minúsculas (`info`) y el de Python en mayúsculas (`INFO`). Este processor los pasa todos a
  mayúsculas, así que una consulta por `ERROR` encuentra los errores de los dos lenguajes.
- **`batch`** — agrupa la telemetría en peticiones más grandes antes de enviarla, lo que
  sale mucho más barato tanto para el Collector como para los backends.

**Exporters**, uno por backend:
- **VictoriaMetrics** recibe las métricas.
- **VictoriaLogs** recibe los logs, con una cabecera `VL-Stream-Fields` que le dice a
  VictoriaLogs cómo agruparlos en streams: por `service.name` y `deployment.environment`.
- **VictoriaTraces** recibe las trazas.

**Extensions**
- **`health_check`**, en el puerto 13133. Responde si el Collector está en marcha;
  `doctor.sh` lo usa.

Y los pipelines que lo conectan todo:

| Señal | Receivers | Processors | Exporter |
|---|---|---|---|
| Métricas | OTLP, sus propias métricas | `memory_limiter`, `resource`, `batch` | VictoriaMetrics |
| Logs | OTLP | `memory_limiter`, `resource`, `transform/severity`, `batch` | VictoriaLogs |
| Trazas | OTLP | `memory_limiter`, `resource`, `batch` | VictoriaTraces |
