# Módulo 6: Métricas Manuales

La auto-instrumentación te dio tasas de peticiones HTTP. A nadie le han dado nunca un aviso por una tasa de peticiones HTTP. Este módulo va de emitir los números que el negocio pregunta de verdad — y del único error que te va a costar dinero real si lo cometes a escala. 📊

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Emitir counters, up-down counters, histogramas y observable gauges desde tus handlers
- Explicarle la cardinalidad a un compañero, después de haberla visto explotar delante de ti
- Leer tus propias métricas de negocio en un dashboard de Grafana

## Ejercicio 6.1 — Tus primeras métricas de negocio

**Deberías poder responder esto al terminar:** *¿cuántos todos se han creado en los últimos cinco minutos en la lista `work`?*

### Hazlo

1. Crea un meter y tres instrumentos, y luego registra desde los handlers.

<!-- track:js -->

En `src/routes/todos.js`, añade `overdueCount` a las funciones que ya importa de `../repo.js` (lo necesitarás en el paso 3), y pon esto cerca del principio del fichero, después de los imports:

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
  // Segundos, minutos, horas, un día: los valores por defecto están pensados para latencia HTTP.
  advice: { explicitBucketBoundaries: [1, 5, 15, 60, 300, 3600, 86400] },
});
```

En el handler de `POST /todos`, después de `invalidate(todo.list)`:

```js
todosCreated.add(1, {
  list: todo.list,
  source: req.get('x-client') ?? 'unknown',
});
todosOpen.add(1, { list: todo.list });
```

En el handler de `PATCH /todos/:id/complete`, después de `invalidate(todo.list)`:

```js
const ageSeconds = (Date.now() - new Date(todo.created_at).getTime()) / 1000;
completionAge.record(ageSeconds, { list: todo.list, priority: String(todo.priority) });
todosOpen.add(-1, { list: todo.list });
```

<!-- /track -->

<!-- track:python -->

En `todo_api/routes.py`, añade `overdue_count` a las funciones que ya importa de `todo_api.repo` (lo necesitarás en el paso 3), y pon esto cerca del principio del fichero, después de los imports:

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
    # Segundos, minutos, horas, un día: los valores por defecto están pensados para latencia HTTP.
    explicit_bucket_boundaries_advisory=[1, 5, 15, 60, 300, 3600, 86400],
)
```

En el handler de `POST /todos`, después de `invalidate(todo.list)`:

```python
todos_created.add(1, {
    "list": todo.list,
    "source": request.headers.get("x-client", "unknown"),
})
todos_open.add(1, {"list": todo.list})
```

En el handler de `PATCH /todos/<todo_id>/complete`, después de `invalidate(todo.list)`:

```python
from datetime import datetime, timezone   # al principio del fichero

age_seconds = (datetime.now(timezone.utc) - todo.created_at).total_seconds()
completion_age.record(age_seconds, {"list": todo.list, "priority": str(todo.priority)})
todos_open.add(-1, {"list": todo.list})
```

<!-- /track -->

<!-- track:go -->

En `routes.go`, añade estos paquetes a sus imports — `go mod tidy` se ejecuta en cada
compilación, así que se descargan solos:

```go
import (
	// ...los imports que ya hay, más:
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)
```

y pon esto después de los imports:

```go
var meter = otel.Meter("todo-api")

var todosCreated, _ = meter.Int64Counter("todos.created",
	metric.WithDescription("Number of todos created"),
	metric.WithUnit("{todo}"),
)

var todosOpen, _ = meter.Int64UpDownCounter("todos.open",
	metric.WithDescription("Todos currently not completed"),
	metric.WithUnit("{todo}"),
)

var completionAge, _ = meter.Float64Histogram("todo.completion.age",
	metric.WithDescription("Wall-clock time between a todo being created and completed"),
	metric.WithUnit("s"),
	// Segundos, minutos, horas, un día: los valores por defecto están pensados para latencia HTTP.
	metric.WithExplicitBucketBoundaries(1, 5, 15, 60, 300, 3600, 86400),
)
```

En `postTodo`, después de `invalidate(todo.List)`:

```go
source := r.Header.Get("x-client")
if source == "" {
	source = "unknown"
}
todosCreated.Add(ctx, 1, metric.WithAttributes(
	attribute.String("list", todo.List),
	attribute.String("source", source),
))
todosOpen.Add(ctx, 1, metric.WithAttributes(attribute.String("list", todo.List)))
```

En `patchComplete`, después de `invalidate(todo.List)`:

```go
ctx := r.Context()
completionAge.Record(ctx, time.Since(todo.CreatedAt).Seconds(), metric.WithAttributes(
	attribute.String("list", todo.List),
	attribute.String("priority", strconv.Itoa(todo.Priority)),
))
todosOpen.Add(ctx, -1, metric.WithAttributes(attribute.String("list", todo.List)))
```

<!-- /track -->

> ℹ️ `todos.open` es un **up-down counter**: suma los cambios que le comunica tu código.
> Empieza en cero cada vez que arranca el proceso — los todos que ya estaban en la base de
> datos no cuentan, y se reinicia cada vez que guardas un fichero. El gauge del paso 3 es
> la otra forma de obtener un número así: le pide el valor real a la base de datos cada
> vez.

2. Fíjate en lo que `todo.completion.age` **no** es: no es latencia HTTP. Eso ya te lo dio la auto-instrumentación. Esto es una duración *de negocio* — cuánto tiempo estuvo ahí un todo antes de que alguien lo hiciera. Ninguna librería podría haberla producido jamás, porque ninguna librería sabe qué es un todo. Ese contraste es el objetivo del ejercicio.

3. Ahora añade un **observable gauge** para los todos vencidos, respaldado por una consulta periódica. Los instrumentos observables los consulta el SDK en el momento de exportar, en lugar de empujarlos tu código. Ponlo justo después de los tres instrumentos:

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
        yield metrics.Observation(row["count"], {"list": row["list"]})

meter.create_observable_gauge(
    "todos.overdue",
    callbacks=[observe_overdue],
    description="Todos past their due date and not completed",
    unit="{todo}",
)
```

<!-- /track -->

<!-- track:go -->

Un instrumento observable se registra una sola vez, así que va en una función `init`:

```go
func init() {
	meter.Int64ObservableGauge("todos.overdue",
		metric.WithDescription("Todos past their due date and not completed"),
		metric.WithUnit("{todo}"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			rows, err := overdueCount(ctx)          // repo.go
			if err != nil {
				return err
			}
			for _, row := range rows {
				o.Observe(row.Count, metric.WithAttributes(attribute.String("list", row.List)))
			}
			return nil
		}),
	)
}
```

<!-- /track -->

4. Guarda el fichero y genera carga. La API se reinicia sola en cuanto guardas — míralo en
   la terminal de su log. Después, en otra terminal:

   ```bash
   ./loadgen.sh 3m
   ```

   El loadgen crea todos y completa algunos de los que ha creado, así que todos los
   instrumentos reciben datos.

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
max(todos_overdue) by (list)
```

`max`, no `sum`: cada proceso de tu API informa del mismo valor, leído de la misma base
de datos. Sumarlos contaría cada todo vencido más de una vez.

> 🧩 **El mapeo de nombres, y pilla a todo el mundo.** Escribiste `todos.created`. Consultaste `todos_created_total`. Los nombres con puntos de OpenTelemetry se convierten en guiones bajos del lado de VictoriaMetrics, y los counters ganan un sufijo `_total`. Igualmente `todo.completion.age` con `unit: s` se convierte en `todo_completion_age_seconds`, y un histograma trae consigo `_bucket`, `_sum` y `_count`. Si una métrica "no está", esto es lo primero que hay que mirar.

### Comprueba en Grafana

Abre Grafana en `http://localhost:3000`.

Abre **Dashboards → Workshop → Manual Instrumentation**. Los cuatro paneles de la fila **Business metrics** — todos creados, todos abiertos, edad de completado p95 y todos vencidos — se van llenando mientras corre el loadgen. Cambia la variable `list` entre valores y confirma que los paneles la siguen.

## Ejercicio 6.2 — El laboratorio de cardinalidad

**Deberías poder responder esto al terminar:** *¿qué regla te dice si un valor es seguro como atributo de métrica?*

### Hazlo

Vamos a hacer lo incorrecto a propósito, verlo doler, y luego arreglarlo.

1. Añade `todo_id` como atributo del counter de creación — una línea más en la llamada a `add()` que ya tienes:

<!-- track:js -->

```js
todosCreated.add(1, {
  list: todo.list,
  todo_id: todo.id,        // ⚠️ mal a propósito
  source: req.get('x-client') ?? 'unknown',
});
```

<!-- /track -->

<!-- track:python -->

```python
todos_created.add(1, {
    "list": todo.list,
    "todo_id": str(todo.id),   # ⚠️ mal a propósito
    "source": request.headers.get("x-client", "unknown"),
})
```

<!-- /track -->

<!-- track:go -->

```go
todosCreated.Add(ctx, 1, metric.WithAttributes(
	attribute.String("list", todo.List),
	attribute.String("todo_id", todo.ID),     // ⚠️ mal a propósito
	attribute.String("source", source),
))
```

<!-- /track -->

2. Guarda el fichero y lanza el generador de carga a fondo:

   ```bash
   ./loadgen.sh 2m --rps 50
   ```

3. Y ahora ve a mirar el destrozo.

### Comprueba en la UI de Victoria

**vmui → pestaña Explore Cardinality** (`http://localhost:8428/vmui`, navegación superior). Mira "Metrics with the highest number of series". Observa cómo `todos_created_total` va subiendo en la lista mientras corre el loadgen.

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

### Comprueba en Grafana

Abre Grafana en `http://localhost:3000`.

Pon `vm_cache_entries{type="storage/tsid"}` en una gráfica junto a `rate(todos_created_total[1m])` — Explore → VictoriaMetrics, una consulta cada una. Las formas son muy distintas, y esa diferencia *es* la lección: una tasa de peticiones modesta produjo un número de series sin límite.

### Limpia

1. **Arréglalo.** Quita `todo_id` de la llamada a `add()` y guarda el fichero.

2. Después limpia. El destrozo es real y está en tu VictoriaMetrics local:

   ```bash
   ./reset.sh
   ```

   No hay API de borrado por consulta para logs, así que limpiar un backend significa
   pararlo, borrar su almacenamiento y arrancarlo otra vez — que es todo lo que hace
   `reset.sh`: borra los tres contenedores de almacenamiento y sus volúmenes de Docker, los
   arranca otra vez vacíos, y reinicia el Collector para que sus colas empiecen limpias. El
   volumen `workshop_postgres`, con tus todos, no se toca.

> ⭐ **La regla:** el valor de un atributo tiene que venir de un conjunto acotado que
> podrías escribir de antemano. `list` la pasa — hay seis. `priority` la pasa — hay tres.
> `todo_id`, `user_id`, `request_id`, `session_id` y las rutas completas de URL fallan
> todas. Cuando `todo_id` es de verdad lo que necesitas saber — y a veces lo es — va en un
> **span** y en un **registro de log**, no en una métrica. El mismo valor, otra señal, el
> veredicto completamente opuesto.
