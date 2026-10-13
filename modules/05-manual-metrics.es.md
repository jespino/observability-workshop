# Módulo 5: Métricas Manuales

**Duración:** ~90 min

La auto-instrumentación te dio tasas de peticiones HTTP. A nadie le han dado nunca un aviso por una tasa de peticiones HTTP. Este módulo va de emitir los números que el negocio pregunta de verdad — y del único error que te va a costar dinero real si lo cometes a escala. 📊

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Elegir el instrumento correcto para una pregunta, de entre los cuatro que hay
- Emitir counters, up-down counters, histogramas y observable gauges desde tus handlers
- Remodelar métricas sin tocar código de aplicación, usando Views
- Explicarle la cardinalidad a un compañero, después de haberla visto explotar delante de ti
- Construir un dashboard RED en Grafana con tus propias métricas

## Prerrequisitos

- Módulo 4 completado; telemetría fluyendo y el Collector sano

## 5.1 — Conceptos [~15 min, charla]

Hay cuatro instrumentos. Elegir el correcto lo determina casi por completo una sola pregunta:

> **¿Quiero una tasa, un valor actual o una distribución?**

| Instrumento | Úsalo cuando | Ejemplo |
|---|---|---|
| **Counter** | Un total monótono. Lo consultarás con `rate()` | `todos.created` |
| **UpDownCounter** | Un valor que sube y baja | `todos.open` |
| **Histogram** | Una distribución — te importa el p95, no la media | `todo.completion.age` |
| **Observable (async) gauge** | Estado muestreado que lees bajo demanda | `todos.overdue` |

Los dos modos de fallo son elegir un gauge cuando querías un counter (pierdes todo lo que pasa entre muestras), y elegir un counter cuando querías un histograma (puedes calcular una media, y las medias mienten).

**Los atributos son dimensiones.** `todos.created` con `{list="work"}` y `todos.created` con `{list="home"}` son dos series temporales. Cada combinación distinta es una más. Retén esa idea hasta 5.4.

**Temporalidad de agregación**, brevemente: cumulative significa "total desde que arrancó el proceso", delta significa "desde la última exportación". VictoriaMetrics quiere cumulative, los dos SDKs usan cumulative por defecto, así que estás bien — pero ahora conoces la palabra.

## Ejercicio 5.2 — Tus primeras métricas de negocio [~30 min]

**Deberías poder responder esto al terminar:** *¿cuántos todos se han creado en los últimos cinco minutos en la lista `work`?* — la pregunta que no pudiste responder en el Módulo 3.3.

### Hazlo

1. Crea un meter y tres instrumentos, y luego registra desde los handlers.

<!-- track:js -->

En `src/routes/todos.js`:

```js
const { metrics } = require('@opentelemetry/api');

const meter = metrics.getMeter('todo-api');

const todosCreated = meter.createCounter('todos.created', {
  description: 'Number of todos created',
  unit: '{todo}',
});

const todosOpen = meter.createUpDownCounter('todos.open', {
  description: 'Todos currently not completed',
  unit: '{todo}',
});

const completionAge = meter.createHistogram('todo.completion.age', {
  description: 'Wall-clock time between a todo being created and completed',
  unit: 's',
});
```

En el handler de creación:

```js
todosCreated.add(1, {
  list: todo.list,
  source: req.get('x-client') ?? 'unknown',
});
todosOpen.add(1, { list: todo.list });
```

En el handler de completado:

```js
const ageSeconds = (Date.now() - new Date(todo.created_at).getTime()) / 1000;
completionAge.record(ageSeconds, { list: todo.list, priority: String(todo.priority) });
todosOpen.add(-1, { list: todo.list });
```

<!-- /track -->

<!-- track:python -->

En `todo_api/routes.py`:

```python
from opentelemetry import metrics

meter = metrics.get_meter("todo-api")

todos_created = meter.create_counter(
    "todos.created",
    description="Number of todos created",
    unit="{todo}",
)

todos_open = meter.create_up_down_counter(
    "todos.open",
    description="Todos currently not completed",
    unit="{todo}",
)

completion_age = meter.create_histogram(
    "todo.completion.age",
    description="Wall-clock time between a todo being created and completed",
    unit="s",
)
```

En el handler de creación:

```python
todos_created.add(1, {
    "list": todo.list,
    "source": request.headers.get("x-client", "unknown"),
})
todos_open.add(1, {"list": todo.list})
```

En el handler de completado:

```python
age_seconds = (datetime.now(timezone.utc) - todo.created_at).total_seconds()
completion_age.record(age_seconds, {"list": todo.list, "priority": str(todo.priority)})
todos_open.add(-1, {"list": todo.list})
```

<!-- /track -->

2. Fíjate en lo que `todo.completion.age` **no** es: no es latencia HTTP. Eso ya te lo dio la auto-instrumentación. Esto es una duración *de negocio* — cuánto tiempo estuvo ahí un todo antes de que alguien lo hiciera. Ninguna librería podría haberla producido jamás, porque ninguna librería sabe qué es un todo. Ese contraste es el objetivo del ejercicio.

3. Ahora añade un **observable gauge** para los todos vencidos, respaldado por una consulta periódica. Los instrumentos observables los consulta el SDK en el momento de exportar, en lugar de empujarlos tu código:

<!-- track:js -->

```js
meter.createObservableGauge('todos.overdue', {
  description: 'Todos past their due date and not completed',
  unit: '{todo}',
}).addCallback(async (result) => {
  for (const row of await overdueCount()) {     // repo.js
    result.observe(row.count, { list: row.list });
  }
});
```

<!-- /track -->

<!-- track:python -->

```python
def observe_overdue(options):
    for row in overdue_count():                  # repo.py
        yield metrics.Observation(row.count, {"list": row.list})

meter.create_observable_gauge(
    "todos.overdue",
    callbacks=[observe_overdue],
    description="Todos past their due date and not completed",
    unit="{todo}",
)
```

<!-- /track -->

4. Guarda el fichero y genera carga. La API se reinicia sola en cuanto guardas — míralo en
   la terminal de su log. Después, en otra terminal:

   ```bash
   ./loadgen.sh 3m
   ```

### Comprueba en la UI de Victoria

vmui de VictoriaMetrics (`http://localhost:8428/vmui`):

```
sum(rate(todos_created_total[5m])) by (list)
```

Deberías ver: una línea por lista, respondiendo al generador de carga.

```
todos_open
```

```
histogram_quantile(0.95, sum(rate(todo_completion_age_seconds_bucket[5m])) by (le, list))
```

```
todos_overdue
```

> 🧩 **El mapeo de nombres, y pilla a todo el mundo.** Escribiste `todos.created`. Consultaste `todos_created_total`. Los nombres con puntos de OpenTelemetry se convierten en guiones bajos del lado de VictoriaMetrics, y los counters ganan un sufijo `_total`. Igualmente `todo.completion.age` con `unit: s` se convierte en `todo_completion_age_seconds`, y un histograma trae consigo `_bucket`, `_sum` y `_count`. Si una métrica "no está", esto es lo primero que hay que mirar.

### Comprueba en Grafana

Explore → VictoriaMetrics → ejecuta la consulta `sum(rate(...)) by (list)`. Cambia la visualización a serie temporal y confirma que las listas aparecen en la leyenda.

### Para pensar

Añadiste `source` como atributo de `todos.created`, rellenado desde una cabecera del cliente. ¿Quién controla el valor de esa cabecera, y qué es lo peor que podría meter ahí?

### Si te atascas

- **La métrica no aparece.** Espera 10 segundos (el intervalo de exportación) y luego revisa el mapeo de nombres de arriba. Después mira los logs del Collector.
- **(JS) `metrics.getMeter` devuelve un no-op.** La API es un no-op salvo que haya un SDK registrado — asegúrate de que el flag `--require` del Módulo 3 sigue en el comando de arranque.
- **(Python) El callback del observable gauge nunca se dispara.** Tiene que registrarse en tiempo de import, no dentro de un handler de petición.

## Ejercicio 5.3 — Views: moldear métricas sin cambiar código [~20 min]

**Deberías poder responder esto al terminar:** *¿cómo arreglas un histograma mal bucketizado dentro de una librería que no es tuya?*

### Hazlo

Tu histograma `todo.completion.age` está usando los límites de bucket por defecto del SDK, pensados para latencias HTTP en segundos — más o menos de 0ms a 10s. Tus todos viven de minutos a días. Todos ellos caen en el último bucket, lo que significa que tu p95 no significa nada.

Las Views son la costura entre "lo que decidió quien escribió la librería" y "lo que decide mi equipo de plataforma". Configura tres:

<!-- track:js -->

En la configuración del SDK (`src/telemetry.js` — créalo si no existe, y cárgalo antes que la app):

```js
const { View, ExplicitBucketHistogramAggregation } = require('@opentelemetry/sdk-metrics');

const views = [
  // 1. Buckets que se ajustan a la realidad: segundos, minutos, horas, un día
  new View({
    instrumentName: 'todo.completion.age',
    aggregation: new ExplicitBucketHistogramAggregation([1, 5, 15, 60, 300, 3600, 86400]),
  }),

  // 2. Descarta un atributo que has decidido que es demasiado caro
  new View({
    instrumentName: 'todos.created',
    attributeKeys: ['list'],          // 'source' se descarta
  }),

  // 3. Renombra un instrumento sin tocar el sitio donde se llama
  new View({
    instrumentName: 'todos.open',
    name: 'todos.active',
  }),
];
```

<!-- /track -->

<!-- track:python -->

En la configuración del SDK (`todo_api/telemetry.py` — créalo si no existe, e impórtalo desde `app.py`):

```python
from opentelemetry.sdk.metrics.view import (
    View, ExplicitBucketHistogramAggregation,
)

views = [
    # 1. Buckets que se ajustan a la realidad: segundos, minutos, horas, un día
    View(
        instrument_name="todo.completion.age",
        aggregation=ExplicitBucketHistogramAggregation(
            [1, 5, 15, 60, 300, 3600, 86400]
        ),
    ),

    # 2. Descarta un atributo que has decidido que es demasiado caro
    View(
        instrument_name="todos.created",
        attribute_keys={"list"},          # "source" se descarta
    ),

    # 3. Renombra un instrumento sin tocar el sitio donde se llama
    View(
        instrument_name="todos.open",
        name="todos.active",
    ),
]
```

<!-- /track -->

Reconstruye, lanza el loadgen y confirma que las tres han surtido efecto.

### Comprueba en la UI de Victoria

Buckets:

```
todo_completion_age_seconds_bucket
```

Deberías ver: valores de la etiqueta `le` de `1`, `5`, `15`, `60`, `300`, `3600`, `86400`, `+Inf` — y, sobre todo, cuentas repartidas por varios de ellos en lugar de amontonadas en `+Inf`.

Atributo descartado:

```
todos_created_total
```

Deberías ver: ninguna serie con etiqueta `source`.

Renombrado:

```
todos_active
```

Deberías ver: datos. Y `todos_open` debería dejar de crecer.

### Comprueba en Grafana

Vuelve a ejecutar tu consulta de p95. Ahora debería devolver un número que cambia cuando cambia el perfil de carga, en vez de clavarse en el bucket más alto.

### Para pensar

Muy poca gente sabe que las Views existen. Dado que permiten a un equipo de plataforma arreglar de forma central las métricas de una librería de terceros, ¿por qué crees que es así — y dónde pondrías la configuración de Views en una organización real?

### Si te atascas

- **Los buckets antiguos siguen ahí.** Las series viejas no desaparecen; estás mirando datos anteriores al reinicio. Reduce el rango de tiempo a los últimos dos minutos.
- **El renombrado produjo los dos nombres.** Lo mismo — series históricas. Es lo esperado.

## Ejercicio 5.4 — El laboratorio de cardinalidad [~25 min] *(recortable — pero es el mejor ejercicio del módulo)*

**Deberías poder responder esto al terminar:** *¿qué regla te dice si un valor es seguro como atributo de métrica?*

### Hazlo

Vamos a hacer lo incorrecto a propósito, verlo doler, y arreglarlo luego de dos formas distintas.

1. Añade `todo_id` como atributo del counter de creación:

<!-- track:js -->

```js
todosCreated.add(1, {
  list: todo.list,
  todo_id: todo.id,        // ⚠️ mal a propósito
});
```

<!-- /track -->

<!-- track:python -->

```python
todos_created.add(1, {
    "list": todo.list,
    "todo_id": todo.id,      # ⚠️ mal a propósito
})
```

<!-- /track -->

2. Reconstruye y lanza el generador de carga a fondo:

   ```bash
   ./loadgen.sh 2m --rps 50
   ```

3. Y ahora ve a mirar el destrozo.

### Comprueba en la UI de Victoria

**vmui → pestaña Explore Cardinality** (`http://localhost:8428/vmui`, navegación superior). Mira "Metrics with the highest number of series". Observa cómo `todos_created_total` sube al primer puesto en menos de un minuto.

Luego los números en crudo:

```bash
curl -s 'http://localhost:8428/api/v1/status/tsdb' | head -c 2000
```

Y el coste de VictoriaMetrics, en sus propias métricas:

```
vm_cache_entries{type="storage/tsid"}
```

```
process_resident_memory_bytes{job="victoria-metrics"}
```

Deberías ver: número de series y memoria subiendo de forma sostenida, y siguiendo hacia arriba mientras el loadgen esté funcionando. Cada todo que creas es ya, permanentemente, una serie temporal.

4. **Arréglalo dos veces y compara.**

   - **Arreglo A — en el sitio de la llamada.** Quita `todo_id` de la llamada a `add()`. Correcto, permanente, requiere desplegar el servicio.
   - **Arreglo B — con una View.** Deja la llamada como está y añade `attributeKeys: ['list']` / `attribute_keys={"list"}` para `todos.created`. También correcto, también requiere un despliegue — pero de *tu configuración de SDK*, que en muchas organizaciones es un módulo compartido propiedad del equipo de plataforma y no del equipo del servicio.

5. Después limpia. El destrozo es real y está en tu VictoriaMetrics local:

   ```bash
   ./reset.sh
   ```

   No hay API de borrado por consulta para logs, así que limpiar un backend significa
   pararlo, borrar su almacenamiento y arrancarlo otra vez — que es todo lo que hace
   `reset.sh`: borra los tres contenedores de almacenamiento y sus volúmenes de Docker, los
   arranca otra vez vacíos, y reinicia el Collector para que sus colas empiecen limpias. El
   volumen `workshop_postgres`, con tus todos, no se toca.

### Comprueba en Grafana

Antes de resetear, pon `vm_cache_entries{type="storage/tsid"}` en una gráfica junto a `rate(todos_created_total[1m])`. Las formas son muy distintas, y esa diferencia *es* la lección: una tasa de peticiones modesta produjo un número de series sin límite.

### Para pensar

Dos preguntas, y la segunda es la buena.

1. **¿Cuál es la regla?** Algo así como: *el valor de un atributo tiene que venir de un conjunto acotado que podrías escribir de antemano.* `list` la pasa — hay seis. `priority` la pasa — hay tres. `todo_id`, `user_id`, `request_id`, `session_id` y las rutas completas de URL fallan todas.

2. **Cuando `todo_id` es de verdad lo que necesitas saber** — y a veces lo es — ¿dónde va? En una métrica no. Va en un **span**, y en un **registro de log**. El mismo valor, otra señal, el veredicto completamente opuesto.

### Si te atascas

- **La cardinalidad no parece crecer.** Puede que el loadgen esté reutilizando un conjunto fijo de todos. Usa `--rps 50` y asegúrate de que está creando, no solo leyendo.
- **VictoriaMetrics se pone lenta o no responde.** Enhorabuena, has reproducido un incidente de producción real. Ejecuta `./reset.sh`.

## Ejercicio 5.5 — Grafana [~15 min]

**Deberías poder responder esto al terminar:** *¿qué muestra un dashboard RED, y qué se le escapa?*

### Hazlo

Construye un dashboard con tu facilitador. Usa el datasource **VictoriaMetrics**, no el de Prometheus — las funciones extra de MetricsQL y la traza de consultas solo están disponibles por ahí.

Tres paneles de negocio, que tienen los dos tracks:

| Panel | Consulta |
|---|---|
| Todos creados | `sum(rate(todos_created_total{list=~"$list"}[5m])) by (list)` |
| Todos abiertos | `sum(todos_active{list=~"$list"}) by (list)` |
| p95 de edad al completar | `histogram_quantile(0.95, sum(rate(todo_completion_age_seconds_bucket{list=~"$list"}[5m])) by (le))` |

Y luego los paneles RED, que es donde pasa factura la divergencia del Módulo 3.2.

<!-- track:python -->

La auto-instrumentación de tu track te dio métricas HTTP de servidor, así que los tres funcionan:

| Panel | Consulta |
|---|---|
| **R**ate | `sum(rate(http_server_duration_milliseconds_count{service_name="todo-api"}[5m]))` |
| **E**rrors | `sum(rate(http_server_duration_milliseconds_count{service_name="todo-api",http_status_code=~"5.."}[5m]))` |
| **D**uration | `histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))` |

<!-- /track -->

<!-- track:js -->

Tu track **no tiene métricas HTTP de servidor** — `@opentelemetry/instrumentation-http` no
las registra. Añade los tres paneles igualmente, con las consultas de abajo, y déjalos vacíos:

| Panel | Consulta |
|---|---|
| **R**ate | `sum(rate(traces_span_metrics_calls_total{service_name="todo-api"}[5m]))` |
| **E**rrors | `sum(rate(traces_span_metrics_calls_total{service_name="todo-api",status_code="STATUS_CODE_ERROR"}[5m]))` |
| **D**uration | `histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket{service_name="todo-api"}[5m])) by (le))` |

Leen métricas RED derivadas de los spans, que todavía no produce nada, así que se quedan
vacíos.

<!-- /track -->

Añade una **variable de plantilla** llamada `list`, de tipo Query, contra el datasource VictoriaMetrics:

```
label_values(todos_created_total, list)
```

Ponla en multivalor e incluye la opción "All". Guarda el dashboard.

### Comprueba en Grafana

Cambia la variable `list` entre valores y confirma que los tres paneles de abajo la siguen y los tres de arriba no. Esa diferencia — métricas de infraestructura frente a métricas de negocio — merece que se señale.

### Para pensar

Los paneles RED son idénticos para cualquier servicio HTTP del mundo. Los tres de abajo existen solo para esta app. ¿Cuál de los dos conjuntos construirías primero para un servicio nuevo, y por cuál te despertarían de madrugada?

### Si te atascas

- **La variable de plantilla está vacía.** `label_values()` necesita que la métrica exista en el rango de tiempo seleccionado. Amplíalo a 6 horas.
- **Los paneles están vacíos pero vmui funciona.** Comprueba que elegiste el datasource VictoriaMetrics y no Prometheus.

## Lo Que Hemos Conseguido

- Cuatro instrumentos, y una regla para elegir entre ellos
- Métricas de negocio que responden preguntas que las de HTTP no pueden
- Views: remodelar telemetría sin tocar el sitio de la llamada
- Un incidente de cardinalidad, reproducido a propósito y diagnosticado en menos de cinco minutos
- Un dashboard RED con tus propios paneles de negocio encima

## Siguientes Pasos

Continúa con el [Módulo 6: Logs Manuales](./06-manual-logs.es.md).
