# Módulo 8: Trazas Manuales

La auto-instrumentación te dio spans de HTTP, de tu framework web y de Postgres. Esos describen tu *infraestructura*. Este módulo va de spans que describen tu *dominio* — y del momento en que una traza cruza una frontera de servicio y sigue entera. 🧵

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Crear spans personalizados alrededor de operaciones de negocio, con atributos y eventos útiles
- Registrar errores en los spans para que se vean como errores, y no solo como texto
- Nombrar spans de forma que no revientes tu backend
- Usar convenciones semánticas, y saber qué te aportan
- Propagar contexto entre servicios a mano, después de haberlo visto romperse

## 8.1 — Conceptos

Un **span** es una operación con un inicio, un fin y un sitio en un árbol:

| Parte | Notas |
|---|---|
| **nombre** | Cardinalidad baja. `GET /todos/:id`, nunca `GET /todos/8f3a…` |
| **kind** | `SERVER`, `CLIENT`, `INTERNAL`, `PRODUCER`, `CONSUMER` |
| **atributos** | Aquí la cardinalidad alta *está bien y se fomenta* |
| **eventos** | Puntos con marca de tiempo dentro del span |
| **status** | `UNSET`, `OK` o `ERROR` |
| **padre** | Lo que lo convierte en un árbol |

> ⭐ **La regla de nombrado:** cardinalidad baja en el nombre, cardinalidad alta en los atributos. Es exactamente la inversa de la regla de métricas del Módulo 6.2, y por la misma razón de fondo — el nombre es una clave de índice, los atributos son la carga útil.

**La propagación de contexto** es cómo sobrevive el árbol de spans a un salto de red. El ID de la traza y el del span activo viajan en la cabecera W3C `traceparent`; el servicio receptor lo lee y hace su span hijo. Cuando alguien dice "la traza se rompe en esta frontera", la cabecera no llegó. Casi siempre es toda la historia.

## Ejercicio 8.2 — Spans personalizados alrededor de operaciones de negocio

**Deberías poder responder esto al terminar:** *¿podría un compañero depurar una creación fallida a las 3 de la mañana solo con esta traza, sin abrir el código?*

### Hazlo

1. Envuelve `enrich` en un span. Es el primer candidato ideal: lógica de negocio de verdad, sin instrumentación de librería propia, ahora mismo un punto totalmente ciego.

<!-- track:js -->

En `src/routes/todos.js`, donde `enrich` ya está importado:

```js
const { trace, SpanStatusCode } = require('@opentelemetry/api');

const tracer = trace.getTracer('todo-api');

async function enrichTraced(todo) {
  return tracer.startActiveSpan('todo.enrich', async (span) => {
    try {
      span.setAttribute('todo.list', todo.list);
      span.setAttribute('todo.tag_count', todo.tags.length);
      span.setAttribute('todo.priority', todo.priority);

      const result = await enrich(todo);

      span.addEvent('enrichment.complete', { source: result.source });
      span.setStatus({ code: SpanStatusCode.OK });
      return result;
    } catch (err) {
      span.recordException(err);
      span.setStatus({ code: SpanStatusCode.ERROR, message: err.message });
      throw err;
    } finally {
      span.end();
    }
  });
}
```

> ⚠️ `span.end()` en un bloque `finally`, siempre. Un span que nunca termina nunca se exporta, y la traza pierde un subárbol en silencio.

Después haz que el handler de `POST /todos` lo llame en lugar de `enrich`:

```js
const enriched = await enrichTraced(todo);
```

<!-- /track -->

<!-- track:python -->

En `todo_api/routes.py`, donde `enrich` ya está importado:

```python
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

tracer = trace.get_tracer("todo-api")


def enrich_traced(todo):
    with tracer.start_as_current_span(
        "todo.enrich", record_exception=False, set_status_on_exception=False,
    ) as span:
        try:
            span.set_attribute("todo.list", todo.list)
            span.set_attribute("todo.tag_count", len(todo.tags))
            span.set_attribute("todo.priority", todo.priority)

            result = enrich(todo)

            span.add_event("enrichment.complete", {"source": result["source"]})
            span.set_status(Status(StatusCode.OK))
            return result
        except Exception as err:
            span.record_exception(err)
            span.set_status(Status(StatusCode.ERROR, str(err)))
            raise
```

> ℹ️ El context manager cierra el span por ti, también si hay excepción. Por defecto además
> registra la excepción y marca el span como error; aquí esas dos cosas están desactivadas
> para que el código lo haga explícitamente, como en los otros tracks, en vez de dos veces.

Después haz que el handler de `POST /todos` lo llame en lugar de `enrich`:

```python
enriched = enrich_traced(todo)
```

<!-- /track -->

<!-- track:go -->

Crea un fichero nuevo, `tracing.go`, junto a `routes.go` — es el mismo paquete, así que
`enrich` ya está a mano:

```go
package main

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("todo-api")

func enrichTraced(ctx context.Context, todo Todo) (Enriched, error) {
	_, span := tracer.Start(ctx, "todo.enrich")
	defer span.End()

	span.SetAttributes(
		attribute.String("todo.list", todo.List),
		attribute.Int("todo.tag_count", len(todo.Tags)),
		attribute.Int("todo.priority", todo.Priority),
	)

	result, err := enrich(todo)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return result, err
	}

	span.AddEvent("enrichment.complete", trace.WithAttributes(attribute.String("source", result.Source)))
	span.SetStatus(codes.Ok, "")
	return result, nil
}
```

> ⚠️ `defer span.End()` justo después de `tracer.Start`, siempre. Un span que nunca termina nunca se exporta, y la traza pierde un subárbol en silencio.

Después haz que `postTodo` lo llame en lugar de `enrich`, pasándole el contexto de la petición:

```go
enriched, err := enrichTraced(ctx, todo)
```

<!-- /track -->

2. Haz lo mismo con **dos operaciones más**. Sugeridas, por orden de rentabilidad:
   - `notifyUrgent` / `notify_urgent` — la llamada saliente, para verla aparte del span HTTP crudo
   - `readSummary` / `read_summary` — el camino de lectura de caché, con un atributo para si acertó

   Para la de caché, envuelve la llamada a `readSummary` / `read_summary` en el handler de `GET /todos` y registra el resultado. Ya devuelve si acertó y la clave que buscó:

<!-- track:js -->

```js
const cached = tracer.startActiveSpan('cache.read', (span) => {
  const result = readSummary(list);
  span.setAttribute('cache.hit', result.hit);
  span.setAttribute('cache.key', result.key);
  span.end();
  return result;
});
```

<!-- /track -->

<!-- track:python -->

```python
with tracer.start_as_current_span("cache.read") as span:
    hit, key, value = read_summary(list_name)
    span.set_attribute("cache.hit", hit)
    span.set_attribute("cache.key", key)
```

<!-- /track -->

<!-- track:go -->

```go
_, span := tracer.Start(r.Context(), "cache.read")
hit, key, value := readSummary(list)
span.SetAttributes(attribute.Bool("cache.hit", hit), attribute.String("cache.key", key))
span.End()
```

<!-- /track -->

3. Guarda el fichero, genera carga y luego haz la **prueba de las 3 de la mañana**: abre una traza de `POST /todos` y pregúntate si podrías depurarla sin el código delante. Itera sobre los atributos hasta que la respuesta sea sí. Esa iteración *es* el ejercicio — el primer conjunto de atributos que eliges nunca es el correcto.

### Comprueba en la UI de Victoria

vmui de VictoriaTraces (`http://localhost:10428/select/vmui`):

```
_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich"
```

Deberías ver: tu span, con `todo.list`, `todo.tag_count` y `todo.priority`, anidado bajo el span de servidor `POST /todos`.

Busca un fallo:

```
_time:15m "resource_attr:service.name":=todo-api status_code:=2
```

`status_code` vale `0` para UNSET, `1` para OK y `2` para ERROR. Si todavía no hay fallos, `./loadgen.sh 1m --errors` añade algunos.

Deberías ver: spans marcados como error con una excepción registrada — no solo una línea de log en otro sitio que casualmente menciona un fallo.

### Comprueba en Grafana

Abre Grafana en `http://localhost:3000`.

Explore → VictoriaTraces (Jaeger) → busca servicio `todo-api`, operación `POST /todos`. En la cascada, tu span `todo.enrich` debería quedar después de los `INSERT` y antes del `UPDATE` que guarda el slug.

## Ejercicio 8.3 — Convenciones semánticas

**Deberías poder responder esto al terminar:** *¿qué ganas por usar un nombre de atributo estándar en vez del tuyo?*

### Hazlo

1. Abre el [registro de convenciones semánticas](https://opentelemetry.io/docs/specs/semconv/) y mira las secciones de HTTP, de base de datos y de errores.

2. Mira los atributos que añadiste en el 8.2: `todo.list`, `todo.tag_count`, `todo.priority`,
   `cache.hit`, `cache.key`. Describen tu dominio, y no hay ningún estándar que los cubra,
   así que un espacio de nombres propio — `todo.` — es lo correcto. Pero hay una cosa que tu
   span `todo.enrich` registra que sí tiene nombre estándar: **qué tipo de error** ocurrió.
   El estándar es `error.type`.

3. Añádelo donde `todo.enrich` registra el fallo:

<!-- track:js -->

```js
} catch (err) {
  span.recordException(err);
  span.setAttribute('error.type', err.name);       // p. ej. TypeError
  span.setStatus({ code: SpanStatusCode.ERROR, message: err.message });
  throw err;
}
```

<!-- /track -->

<!-- track:python -->

```python
except Exception as err:
    span.record_exception(err)
    span.set_attribute("error.type", type(err).__name__)   # p. ej. ValueError
    span.set_status(Status(StatusCode.ERROR, str(err)))
    raise
```

<!-- /track -->

<!-- track:go -->

```go
// añade "fmt" a los imports de tracing.go
if err != nil {
	span.RecordError(err)
	span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))   // p. ej. *errors.errorString
	span.SetStatus(codes.Error, err.Error())
	return result, err
}
```

<!-- /track -->

   Guarda el fichero y genera algunos fallos: `./loadgen.sh 1m --errors`.

4. Ahora mira qué le puso la *auto*-instrumentación al span de servidor `POST /todos`:

   ```
   _time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
   ```

<!-- track:js -->

   Encontrarás `span_attr:http.request.method`, `span_attr:http.response.status_code` y
   `span_attr:url.path` — los nombres que usa el registro que acabas de leer. Las librerías
   de instrumentación de JavaScript ya se han pasado a ellos.

<!-- /track -->

<!-- track:python -->

   Encontrarás `span_attr:http.method`, `span_attr:http.status_code` y
   `span_attr:http.target` — los nombres **antiguos**. El registro que acabas de leer dice
   `http.request.method`, `http.response.status_code` y `url.path`. La convención se movió;
   las librerías de instrumentación de Python todavía no se han puesto al día.

<!-- /track -->

<!-- track:go -->

   Encontrarás `span_attr:http.request.method`, `span_attr:http.response.status_code` y
   `span_attr:url.path` — los nombres que usa el registro que acabas de leer. La
   instrumentación que compila `otelc` ya los usa. Incluso pone `error.type` por su cuenta
   en una petición fallida: el código de estado, `500`.

<!-- /track -->

   La misma petición recibe nombres de atributo distintos según qué SDK, y qué versión de
   él, la haya producido. Eso no es una nota al pie: todos los dashboards, alertas y
   consultas guardadas escritos contra los nombres viejos se rompen el día que las librerías
   se pongan al día. Por eso, en producción, fijas las versiones de tu instrumentación y las
   actualizas a propósito.

### Comprueba en la UI de Victoria

Abre VictoriaTraces en `http://localhost:10428/select/vmui`.

```
_time:15m "resource_attr:service.name":=todo-api name:="todo.enrich" "span_attr:error.type":*
```

Deberías ver: los spans `todo.enrich` fallidos, cada uno con su `error.type`.

```
_time:15m "span_attr:http.route":="/todos" status_code:=2
```

Deberías ver: las peticiones fallidas de esa ruta, seleccionadas por `http.route` — un
nombre estándar en el que los tres tracks coinciden. La misma consulta funciona contra
cualquier servicio instrumentado con OTel.

### Comprueba en Grafana

Abre Grafana en `http://localhost:3000`.

Copia el `trace_id` de uno de los spans `todo.enrich` fallidos. En Explore → VictoriaTraces (Jaeger), pégalo en la búsqueda por trace ID y despliega `todo.enrich`: `error.type` está junto a la excepción registrada.

## Ejercicio 8.4 — Propagación entre servicios

**Deberías poder responder esto al terminar:** *¿qué viaja exactamente entre dos servicios para que dos trazas sean una?*

### Hazlo

**Paso 1 — Confirma que ya funciona.**

Crea un todo urgente:

<!-- track:js -->

```bash
curl -s -X POST localhost:8081/todos -H 'content-type: application/json' -d '{"title":"Call the bank","list":"work","tags":["urgent"]}'
```

<!-- /track -->

<!-- track:python -->

```bash
curl -s -X POST localhost:8082/todos -H 'content-type: application/json' -d '{"title":"Call the bank","list":"work","tags":["urgent"]}'
```

<!-- /track -->

<!-- track:go -->

```bash
curl -s -X POST localhost:8085/todos -H 'content-type: application/json' -d '{"title":"Call the bank","list":"work","tags":["urgent"]}'
```

<!-- /track -->

Busca su lado en `notify-service`, en VictoriaTraces:

```
_time:5m "resource_attr:service.name":=notify-service name:="POST /notify"
```

Copia su `trace_id` y trae la traza entera:

```
trace_id:=<pega aquí el valor>
```

Deberías ver spans de **los dos** servicios en una misma traza. Nadie escribió código de
propagación — la instrumentación del cliente HTTP inyectó `traceparent` a la salida, y la
instrumentación del servidor HTTP de `notify-service` lo extrajo a la entrada.

**Paso 2 — Rómpelo, a propósito.**

<!-- track:js -->

Imagina que la librería de cliente HTTP que usa tu API no tuviera instrumentación — algo
bastante habitual en librerías menos populares. Apaga su instrumentación: añade una línea
al `.env` de tu API y recrea el contenedor.

`notify.js` llama a `notify-service` con `fetch`, que cubre la instrumentación `undici`:

```bash
# track-js/todo-api/.env
OTEL_NODE_DISABLED_INSTRUMENTATIONS=undici
```

```bash
docker compose up -d --force-recreate todo-api-js
```

<!-- /track -->

<!-- track:python -->

Imagina que la librería de cliente HTTP que usa tu API no tuviera instrumentación — algo
bastante habitual en librerías menos populares. Apaga su instrumentación: añade una línea
al `.env` de tu API y recrea el contenedor.

`notify.py` llama a `notify-service` con `requests`, que funciona sobre `urllib3`; las dos
están instrumentadas:

```bash
# track-py/todo-api/.env
OTEL_PYTHON_DISABLED_INSTRUMENTATIONS=requests,urllib3
```

```bash
docker compose up -d --force-recreate todo-api-py
```

<!-- /track -->

<!-- track:go -->

En Go, la traza viaja dentro del `context.Context` que pasas de función en función, y la
instrumentación del cliente HTTP la lee del contexto de la petición para inyectar
`traceparent`. Pierde el contexto y pierdes la traza — la forma más habitual de que pase en
Go. En `notify.go`, haz que `post` construya su petición con `context.Background()` en vez
de con el `ctx` que recibe:

```go
req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, notifyURL+path, bytes.NewReader(payload))
```

Guarda el fichero.

<!-- /track -->

Crea otro todo urgente y repite el paso 1. Ahora tienes **dos trazas huérfanas** en vez de
una: la de tu servicio, que termina en el trabajo de base de datos, y otra que empieza sin
ella y contiene el span de `notify-service`.

> 🧠 Párate un momento aquí. **Este estado "antes" es lo que parecen la mayoría de los sistemas distribuidos reales** — todos los servicios instrumentados, ninguna traza que cruce más de uno. Parece que funciona hasta justo el momento en que lo necesitas.

**Paso 3 — Arréglalo a mano.**

<!-- track:js -->

Inyecta tú el contexto en las cabeceras de la petición, con la API de propagación.

En `src/notify.js`:

```js
const { context, propagation } = require('@opentelemetry/api');

async function post(path, body) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  try {
    const headers = { 'content-type': 'application/json' };
    propagation.inject(context.active(), headers);   // añade traceparent
    const res = await fetch(`${NOTIFY_URL}${path}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
      signal: controller.signal,
    });
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}
```

Guarda el fichero, crea un todo urgente y confirma que la traza vuelve a estar entera. Hay
una diferencia: no hay span `POST` de cliente entre los spans de tu API y el de
`notify-service`, porque ya nada instrumenta el cliente. La cabecera `traceparent` por sí
sola basta para unir los dos servicios.

<!-- /track -->

<!-- track:python -->

Inyecta tú el contexto en las cabeceras de la petición, con la API de propagación.

En `todo_api/notify.py`:

```python
from opentelemetry.propagate import inject


def _post(path: str, body: dict) -> dict:
    headers = {"content-type": "application/json"}
    inject(headers)                                  # añade traceparent
    response = requests.post(
        f"{NOTIFY_URL}{path}",
        json=body,
        headers=headers,
        timeout=TIMEOUT_S,
    )
    return response.json()
```

Guarda el fichero, crea un todo urgente y confirma que la traza vuelve a estar entera. Hay
una diferencia: no hay span `POST` de cliente entre los spans de tu API y el de
`notify-service`, porque ya nada instrumenta el cliente. La cabecera `traceparent` por sí
sola basta para unir los dos servicios.

<!-- /track -->

<!-- track:go -->

Devuélvele su contexto a la petición — `ctx` en vez de `context.Background()`:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodPost, notifyURL+path, bytes.NewReader(payload))
```

Guarda el fichero, crea un todo urgente y confirma que la traza vuelve a estar entera. No
has inyectado nada tú: eso lo hace la instrumentación, siempre que le llegue el contexto.
En Go, propagar la traza *entre* servicios empieza por pasar `ctx` *dentro* del tuyo.

<!-- /track -->

**Paso 4 — Déjalo como estaba.**

<!-- track:js -->

Quita la línea que añadiste al `.env`, recrea el contenedor y deshaz el cambio del
`inject` en el cliente de notify. Con la instrumentación otra vez activa, inyectar también
a mano envía la cabecera dos veces — y en el track de JS eso vuelve a romper la traza.

<!-- /track -->

<!-- track:python -->

Quita la línea que añadiste al `.env`, recrea el contenedor y deshaz el cambio del
`inject` en el cliente de notify. Con la instrumentación otra vez activa, inyectar también
a mano envía la cabecera dos veces, y eso no lo quieres.

<!-- /track -->

<!-- track:go -->

No hay nada que deshacer: el paso 3 dejó `notify.go` como estaba.

<!-- /track -->

### Comprueba en la UI de Victoria

Abre VictoriaTraces en `http://localhost:10428/select/vmui`.

Coge la traza de la llamada más reciente a `notify-service`:

```
_time:5m "resource_attr:service.name":=notify-service name:="POST /notify" | fields _time, trace_id
```

```
trace_id:=<pega aquí el valor>
```

Deberías ver: mientras está roto, ningún `POST /todos` en esa traza — empieza sin tu
petición. Después del arreglo, las mismas dos consultas muestran `POST /todos` y todo lo
que cuelga de él, el span de `notify-service` incluido, en una sola traza.

### Comprueba en Grafana

Abre Grafana en `http://localhost:3000`.

Explore → VictoriaTraces (Jaeger), busca el servicio `todo-api`, operación `POST /todos`. Antes del arreglo, la cascada de una creación urgente termina en tu servicio. Después, la cascada continúa dentro de `notify-service`, y puedes ver exactamente cuánto de la petición se lleva.
