# Módulo 8: Correlación — Convertir las Tres Señales en Una

**Duración:** ~45 min · **Casi igual para los dos tracks**

Este es el módulo del premio. Todo lo que has construido hasta ahora era materia prima; aquí lo conectas todo para poder moverte de una gráfica a una línea de log, a una traza y a una línea de código sin escribir ni una consulta. 🔗

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Generar métricas RED a partir de spans automáticamente, sin código de aplicación
- Explicar por qué dos medidas de "la misma" latencia discrepan legítimamente
- Construir los cuatro saltos de navegación entre señales en Grafana
- Saber exactamente qué renuncia este stack frente a un Prometheus+Tempo, y por qué el cambio puede compensar igualmente

## Prerrequisitos

- Módulo 7 completado
- El dashboard del Módulo 5.5, con el panel de logs añadido en el Módulo 6.4

## Ejercicio 8.1 — Métricas a partir de spans con el connector spanmetrics [~20 min] *(recortable)*

**Deberías poder responder esto al terminar:** *ahora tienes dos fuentes de verdad para la latencia de petición. ¿En cuál confías?*

### Hazlo

Un **connector** en el Collector es un exporter en un pipeline y un receiver en otro. El connector `spanmetrics` consume spans y produce métricas RED a partir de ellos. Aquí editas tú mismo la configuración del Collector, usando lo que aprendiste en el Módulo 4.2.

1. Añade el connector a `collector/config.yaml`:

   ```yaml
   connectors:
     spanmetrics:
       histogram:
         explicit:
           buckets: [10ms, 50ms, 100ms, 250ms, 500ms, 1s, 5s]
       dimensions:
         - { name: http.route }
         - { name: todo.list }
   ```

2. Conéctalo a los dos pipelines — esta es la parte que resulta rara la primera vez:

   ```yaml
   service:
     pipelines:
       traces:
         receivers: [otlp]
         processors: [memory_limiter, resource, batch]
         exporters: [otlphttp/victoriatraces, spanmetrics]   # ← también exporter
       metrics:
         receivers: [otlp, spanmetrics]                      # ← y receiver
         processors: [memory_limiter, resource, batch]
         exporters: [otlphttp/victoriametrics]
   ```

3. Reinicia el Collector (`docker compose restart otel-collector`) y lanza el loadgen dos minutos.

Fíjate en la lista de `dimensions`. `http.route` es una convención semántica que te salió gratis; `todo.list` es el atributo que *tú* pusiste en tus spans en el Módulo 7.2. Nombres estándar y nombres de tu dominio, promocionados a etiquetas de métrica codo con codo.

> ⚠️ Las dimensiones son atributos de métrica, así que la regla del Módulo 5.4 aplica con toda su fuerza. `todo.list` está acotado — bien. Añadir `todo.id` aquí recrearía el incidente de cardinalidad, esta vez desde un fichero de configuración en vez de desde el código.

### Comprueba en la UI de Victoria

vmui de VictoriaMetrics:

```
{__name__=~"traces_span_metrics.*"}
```

Deberías ver: `traces_span_metrics_duration_milliseconds_bucket`, `_count`, `_sum` y un counter de llamadas — nada de lo cual existe en el código de tu aplicación.

Y ahora la consulta interesante:

```
histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket{span_name="POST /todos"}[5m])) by (le, todo_list))
```

**Métricas RED para cada span del sistema, gratis, para siempre y sin tocar código de aplicación.** Para un servicio que no es tuyo o que no puedes redesplegar, esta suele ser la vía más rápida para tener métricas de cualquier tipo.

### Comprueba en Grafana

Pon dos paneles uno al lado del otro:

1. Tu histograma escrito a mano del Módulo 5.2 mide *edad al completar*, así que compara mejor contra el histograma de duración HTTP de la auto-instrumentación:

   ```
   histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{http_route="/todos"}[5m])) by (le))
   ```

2. El derivado de spans:

   ```
   histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket{span_name="POST /todos"}[5m])) by (le)) / 1000
   ```

No van a coincidir exactamente. 🤔

### Para pensar

**¿Por qué no coinciden?** Piénsalo antes de seguir leyendo:

- **Sampling.** Si dejaste el tail sampling puesto del 7.5, la métrica derivada de spans solo cuenta las trazas que sobrevivieron. El histograma del SDK las cuenta todas.
- **Fronteras del span.** El span de servidor HTTP y la medida de duración del SDK no empiezan ni terminan exactamente en la misma instrucción.
- **Batching y llegadas tardías.** Los spans que llegan después de la ventana de agregación del Collector caen en otro punto del tiempo.

Entonces, ¿en cuál confías? La respuesta honesta es que miden cosas distintas y deberías saber cuál es cuál. El histograma del SDK es la verdad de referencia para "cuánto tardó mi handler". El derivado de spans es una aproximación cómoda que además cubre servicios que nunca instrumentaste a mano.

Es un punto sutil, real y que casi nunca se enseña, y saber explicarlo hará que parezca que has llevado esto en producción — porque es donde se aprende.

### Si te atascas

- **No aparece ningún `traces_span_metrics_*`.** La causa más habitual es haber puesto el connector bajo `processors:` en vez de en el bloque `connectors:` de primer nivel.
- **La métrica existe pero no tiene etiqueta `todo_list`.** Las dimensiones solo aparecen si el atributo está realmente en el span. Comprueba un span en vmui primero.

## Ejercicio 8.2 — Cableando los saltos en Grafana [~25 min]

**Deberías poder responder esto al terminar:** *partiendo de un pico en un dashboard, ¿cuántos clics hasta la línea de código?*

### Hazlo

Hay cuatro saltos que construir. Cada uno es un pequeño trozo de configuración de Grafana, y juntos son aquello a lo que realmente veníamos.

| # | Desde | Hasta | Mecanismo |
|---|---|---|---|
| 1 | Línea de log | Traza | Datasource VictoriaLogs → derived field sobre `trace_id` |
| 2 | Span | Logs | Datasource Jaeger → "Trace to logs" |
| 3 | Métrica | Logs | Grafana Correlations |
| 4 | Métrica | Traza | Grafana Correlations (con matiz) |

**Salto 1 — De línea de log a traza.**

Grafana → Connections → datasource **VictoriaLogs** → **Derived fields** → Añadir:

- Nombre: `TraceID`
- Campo: `trace_id`
- Internal link: **VictoriaTraces (Jaeger)**
- Consulta: `${__value.raw}`

Ahora cada línea de log en Explore tiene un botón **TraceID**. Púlsalo y aterrizas en la cascada.

**Salto 2 — De span a logs.**

Grafana → Connections → datasource **VictoriaTraces (Jaeger)** → **Trace to logs**:

- Data source: VictoriaLogs
- Tags: `service.name`
- Activa "Filter by trace ID"
- Consulta: `trace_id:${__trace.traceId}`

Ahora cada span de una cascada tiene un botón **Logs for this span**.

**Salto 3 — De métrica a logs.**

Grafana → **Correlations** → Añadir:

- Origen: VictoriaMetrics, destino: VictoriaLogs
- Consulta destino: `service.name:${service_name} AND _time:$__interval`
- Mapea la etiqueta `service_name` del resultado de origen

**Salto 4 — De métrica a traza.** Mira el matiz de abajo y luego constrúyelo como una correlación hacia una *búsqueda filtrada de trazas*:

- Origen: VictoriaMetrics, destino: VictoriaTraces (Jaeger)
- Destino: servicio `${service_name}`, en el mismo rango de tiempo

**Y ahora ejecuta el bucle entero, en vivo.** Lanza el loadgen con perfil de errores:

```bash
./loadgen.sh 3m --errors
```

Ahora, en tu dashboard del Módulo 5: ves el pico de tasa de errores → haces clic hasta los logs → encuentras un registro ERROR → pulsas **TraceID** → aterrizas en la cascada → localizas el span con estado de error → lees sus atributos → abres ese fichero. Cronométrate. Debería costarte menos de un minuto. ⏱️

### Comprueba en Grafana

Los cuatro saltos tienen que navegar de verdad y llegar con datos. Un salto que aterriza en un panel vacío es peor que no tener salto, porque durante un incidente supondrás que los datos no existen.

### Comprueba en la UI de Victoria

Comprueba el join por debajo, fuera de Grafana. Coge un `trace_id` de un registro de log:

```
service.name:todo-api AND severity_text:ERROR | fields _time, trace_id
```

y pégalo en la vmui de VictoriaTraces:

```
trace_id:=<pega aquí el valor>
```

Si eso funciona en vmui pero el salto de Grafana no, el problema es la configuración del datasource, no tus datos.

### 🚧 Sé honesto sobre la ausencia de exemplars

Hay un flujo que este stack **no** te da, y mereces saberlo ahora en vez de descubrirlo durante un incidente.

En un stack Prometheus+Tempo, los histogramas pueden llevar **exemplars** — un trace id de muestra adjunto a la observación de un bucket concreto. Eso habilita el flujo famoso: haces clic en un pico de una gráfica de latencia y aterrizas directamente en *la traza lenta exacta* que hay detrás.

**VictoriaMetrics no soporta exemplars.** Así que ese flujo concreto de un clic no está disponible aquí. El rodeo es el salto 4 de arriba: una Correlation de Grafana que te lleva a una *búsqueda filtrada de trazas* sobre el mismo servicio y ventana de tiempo, de la que eliges tú la traza lenta. Un clic más y algo de criterio.

Plantéalo como un compromiso, porque es lo que es. Este stack te compra un **lenguaje de consulta compartido entre logs y trazas**, muy buena compresión y una operación genuinamente sencilla. Te cuesta el drilldown basado en exemplars, hoy. Que sea un buen trato depende de tu equipo — pero debería ser una decisión, no una sorpresa.

### Para pensar

Acabas de construir cuatro saltos a mano. En una organización de cincuenta servicios, ¿quién es dueño de esta configuración, y cómo la recibe un servicio nuevo sin que alguien tenga que acordarse de hacerlo?

### Si te atascas

- **El derived field no aparece.** Solo aplica a campos que Grafana ve en el registro de log; confirma primero que `trace_id` sale en la tabla de Explore.
- **"Trace to logs" no devuelve nada.** El mapeo de tags suele ser el culpable: Grafana envía `service.name` y tus logs tienen que tener un campo con exactamente ese nombre.
- **No aparece el menú de Correlations.** Puede estar detrás de un feature toggle según tu versión de Grafana — tu facilitador tiene el flag.

## Lo Que Hemos Conseguido

- Métricas RED generadas a partir de spans, sin cambios en la aplicación
- Una explicación clara de por qué dos medidas de latencia discrepan legítimamente
- Cuatro saltos de navegación entre señales, funcionando
- El bucle completo ejecutado en vivo: pico → logs → traza → código
- Un relato honesto de lo que este stack sacrifica

## Siguientes Pasos

Continúa con el [Módulo 9: Proyecto Final — El Reto de Depuración](./09-capstone-debugging-challenge.es.md).
