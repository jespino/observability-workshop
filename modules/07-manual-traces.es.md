# Módulo 7: Trazas Manuales

**Duración:** ~90 min

La auto-instrumentación te dio spans de HTTP, de tu framework web y de Postgres. Esos describen tu *infraestructura*. Este módulo va de spans que describen tu *dominio* — y del momento en que una traza cruza una frontera de servicio y una frontera de lenguaje y sigue entera. 🧵

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Crear spans personalizados alrededor de operaciones de negocio, con atributos y eventos útiles
- Registrar errores en los spans para que se vean como errores, y no solo como texto
- Nombrar spans de forma que no revientes tu backend
- Usar convenciones semánticas, y saber qué te aportan
- Propagar contexto entre servicios a mano, después de haberlo visto romperse
- Aplicar sampling de cabecera y de cola, y articular el compromiso entre ambos

## Prerrequisitos

- Módulo 6 completado

## 7.1 — Conceptos [~10 min]

Un **span** es una operación con un inicio, un fin y un sitio en un árbol:

| Parte | Notas |
|---|---|
| **nombre** | Cardinalidad baja. `GET /todos/:id`, nunca `GET /todos/8f3a…` |
| **kind** | `SERVER`, `CLIENT`, `INTERNAL`, `PRODUCER`, `CONSUMER` |
| **atributos** | Aquí la cardinalidad alta *está bien y se fomenta* |
| **eventos** | Puntos con marca de tiempo dentro del span |
| **status** | `UNSET`, `OK` o `ERROR` |
| **padre** | Lo que lo convierte en un árbol |

> ⭐ **La regla de nombrado:** cardinalidad baja en el nombre, cardinalidad alta en los atributos. Es exactamente la inversa de la regla de métricas del Módulo 5.4, y por la misma razón de fondo — el nombre es una clave de índice, los atributos son la carga útil.

**La propagación de contexto** es cómo sobrevive el árbol de spans a un salto de red. El id del span activo viaja en la cabecera W3C `traceparent`; el servicio receptor lo lee y hace su span hijo. Cuando alguien dice "la traza se rompe en esta frontera", la cabecera no llegó. Casi siempre es toda la historia.

## Ejercicio 7.2 — Spans personalizados alrededor de operaciones de negocio [~25 min]

**Deberías poder responder esto al terminar:** *¿podría un compañero depurar una creación fallida a las 3 de la mañana solo con esta traza, sin abrir el código?*

### Hazlo

1. Envuelve `enrich` en un span. Es el primer candidato ideal: lógica de negocio de verdad, sin instrumentación de librería propia, ahora mismo un punto totalmente ciego.

<!-- track:js -->

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

<!-- /track -->

<!-- track:python -->

```python
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

tracer = trace.get_tracer("todo-api")


def enrich_traced(todo):
    with tracer.start_as_current_span("todo.enrich") as span:
        try:
            span.set_attribute("todo.list", todo.list)
            span.set_attribute("todo.tag_count", len(todo.tags))
            span.set_attribute("todo.priority", todo.priority)

            result = enrich(todo)

            span.add_event("enrichment.complete", {"source": result.source})
            span.set_status(Status(StatusCode.OK))
            return result
        except Exception as err:
            span.record_exception(err)
            span.set_status(Status(StatusCode.ERROR, str(err)))
            raise
```

> ℹ️ El context manager cierra el span por ti, también si hay excepción. Por eso `start_as_current_span` es preferible a crear spans a mano.

<!-- /track -->

2. Haz lo mismo con **dos operaciones más**. Sugeridas, por orden de rentabilidad:
   - `notifyUrgent` / `notify_urgent` — la llamada saliente, para verla aparte del span HTTP crudo
   - `readSummary` / `read_summary` — el camino de lectura de caché, con un atributo para si acertó

   Para la de caché, añade un atributo que registre el resultado:

<!-- track:js -->

```js
span.setAttribute('cache.hit', hit);
span.setAttribute('cache.key', key);
```

<!-- /track -->

<!-- track:python -->

```python
span.set_attribute("cache.hit", hit)
span.set_attribute("cache.key", key)
```

<!-- /track -->

3. Reconstruye, genera carga y luego haz la **prueba de las 3 de la mañana**: abre una traza de `POST /todos` y pregúntate si podrías depurarla sin el código delante. Itera sobre los atributos hasta que la respuesta sea sí. Esa iteración *es* el ejercicio — el primer conjunto de atributos que eliges nunca es el correcto.

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

Deberías ver: spans marcados como error con una excepción registrada — no solo una línea de log en otro sitio que casualmente menciona un fallo.

### Comprueba en Grafana

Explore → VictoriaTraces (Jaeger) → busca servicio `todo-api`, operación `POST /todos`. En la cascada, tu span `todo.enrich` debería quedar entre el trabajo de base de datos y la llamada saliente.

### Para pensar

Añadiste `cache.key` como atributo de span. En el Módulo 5.4 un valor sin acotar en una *métrica* fue un desastre. ¿Por qué es aceptable aquí — y hay algún límite a lo que deberías poner en un span?

### Si te atascas

- **El span aparece en la raíz en vez de anidado.** Lo creaste fuera del contexto activo. En JS eso suele significar que perdiste el contexto al cruzar un `await` en un callback no instrumentado; en Python, que usaste `start_span` en vez de `start_as_current_span`.
- **El span no aparece nunca.** Comprueba que se llama a `end()` en todos los caminos.

## Ejercicio 7.3 — Convenciones semánticas [~10 min]

**Deberías poder responder esto al terminar:** *¿qué ganas por usar un nombre de atributo estándar en vez del tuyo?*

### Hazlo

1. Abre el [registro de convenciones semánticas](https://opentelemetry.io/docs/specs/semconv/) y mira las secciones de HTTP y de base de datos.

2. Repasa los atributos que te inventaste en 7.2 y busca **al menos uno** que ya tenga un nombre estándar. Candidatos probables en este código:

   | Probablemente escribiste | El estándar es |
   |---|---|
   | `http.status` | `http.response.status_code` |
   | `db.query` | `db.query.text` |
   | `cache.error` | `error.type` |

3. Arréglalo. Con un atributo basta para entender el punto.

4. Ahora mira qué le puso la *auto*-instrumentación a tus spans:

   ```
   _time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
   ```

   Encontrarás `span_attr:http.method`, `span_attr:http.status_code` y
   `span_attr:http.target` — los nombres **antiguos**. El registro que acabas de leer dice
   `http.request.method`, `http.response.status_code` y `url.path`. La convención se movió;
   las librerías de instrumentación de este taller todavía no.

   Eso no es una nota al pie. Todos los dashboards, alertas y consultas guardadas escritos
   contra los nombres viejos se rompen el día que se pongan al día. Que es justo de lo que
   habla el párrafo siguiente.

### Comprueba en la UI de Victoria

```
_time:15m "span_attr:http.route":="/todos" status_code:=2
```

Deberías ver: las peticiones fallidas de esa ruta, seleccionadas por `http.route` — un nombre de atributo estándar. La misma consulta funciona contra cualquier servicio instrumentado con OTel en cualquier parte.

### Comprueba en Grafana

Explore → VictoriaTraces (Tempo) → abre **Traces Drilldown**. Sus paneles RED se construyen casando nombres de atributo estándar. Cuantas más convenciones sigas, más de esto obtienes sin configurar nada.

### Para pensar

Los nombres estándar hacen que los dashboards de proveedores, el Drilldown de Grafana y el connector spanmetrics (que añadirás en el Módulo 8) funcionen sobre tus datos con cero configuración. Ese es el premio.

El coste también es real: **algunas convenciones siguen marcadas como experimentales y sí cambian entre versiones.** `http.method` pasó a ser `http.request.method`; `db.system` pasó a ser `db.system.name`. Si te atas a una convención que se mueve, tus dashboards se rompen. Nombra ese coste en vez de fingir que no existe — y fija las versiones de tu SDK.

### Si te atascas

- **Tu consulta no devuelve nada tras el renombrado.** Los spans antiguos conservan el atributo antiguo. Reduce el rango de tiempo a después de tu reinicio.

## Ejercicio 7.4 — Propagación entre servicios y entre lenguajes [~25 min]

**Deberías poder responder esto al terminar:** *¿qué viaja exactamente entre dos servicios para que dos trazas sean una?*

### Hazlo

Recuerda que `notify-service` está escrito en el **otro** lenguaje respecto a tu track. De eso va este ejercicio.

**Paso 1 — Confirma que ya funciona.**

Crea un todo urgente y busca la traza:

```
_time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
```

Ábrela. Deberías ver spans de **los dos** servicios en un mismo árbol. Nadie escribió código de propagación — la instrumentación del cliente HTTP inyectó `traceparent` a la salida, y la instrumentación del servidor HTTP del otro servicio lo extrajo a la entrada.

**Paso 2 — Rómpelo, a propósito.**

Sustituye el cliente HTTP instrumentado de `notify.js` / `notify.py` por uno crudo que no lleve cabeceras:

<!-- track:js -->

```js
// Roto a propósito: petición pelada sin propagación
const { request } = require('node:http');

function notifyUrgentBroken(todo) {
  return new Promise((resolve, reject) => {
    const req = request(
      { host: 'notify-service', port: 8083, path: '/notify', method: 'POST',
        headers: { 'content-type': 'application/json' } },   // ← sin traceparent
      resolve,
    );
    req.on('error', reject);
    req.end(JSON.stringify({ id: todo.id, list: todo.list }));
  });
}
```

<!-- /track -->

<!-- track:python -->

```python
# Roto a propósito: petición pelada sin propagación
import http.client, json

def notify_urgent_broken(todo):
    conn = http.client.HTTPConnection("notify-service", 8083)
    conn.request(
        "POST", "/notify",
        body=json.dumps({"id": todo.id, "list": todo.list}),
        headers={"content-type": "application/json"},        # ← sin traceparent
    )
    conn.getresponse().read()
    conn.close()
```

<!-- /track -->

Reconstruye, crea un todo urgente y mira las trazas. Ahora tienes **dos trazas huérfanas** en vez de una: la de tu servicio y la de `notify-service`, sin ninguna relación entre ellas.

> 🧠 Párate un momento aquí. **Este estado "antes" es lo que parecen la mayoría de los sistemas distribuidos reales** — todos los servicios instrumentados, ninguna traza que cruce más de uno. Parece que funciona hasta justo el momento en que lo necesitas.

**Paso 3 — Arréglalo a mano.**

Inyecta el contexto explícitamente con la API de propagación:

<!-- track:js -->

```js
const { context, propagation } = require('@opentelemetry/api');

const headers = { 'content-type': 'application/json' };
propagation.inject(context.active(), headers);      // añade traceparent
// ...pasa `headers` a tu cliente crudo
```

<!-- /track -->

<!-- track:python -->

```python
from opentelemetry.propagate import inject

headers = {"content-type": "application/json"}
inject(headers)                                      # añade traceparent
# ...pasa `headers` a tu cliente crudo
```

<!-- /track -->

Reconstruye y confirma que la traza vuelve a estar entera.

**Paso 4 — Extra, si da tiempo.** Añade instrumentación de navegador al frontend compartido para que la traza empiece en el clic del usuario y no en tu servidor. Es la demo que hace que la gente diga *"ahhh"*. Necesita que la API permita la cabecera `traceparent` en CORS — eso está resuelto en la rama `browser-tracing` si quieres mirarlo.

### Comprueba en la UI de Victoria

Mientras está roto:

```
_time:15m "resource_attr:service.name":=notify-service
```

Abre un span y fíjate en que su `trace_id` no aparece en ninguna traza de `todo-api`.

Tras el arreglo, coge un `trace_id` de un span de `todo-api` y ejecuta:

```
trace_id:=<pega aquí el valor>
```

Deberías ver: spans de **los dos** servicios, en una misma lista, ordenados por tiempo.

### Comprueba en Grafana

Explore → VictoriaTraces (Jaeger). Antes del arreglo, la cascada de `POST /todos` termina en tu servicio. Después, la cascada continúa dentro de `notify-service` — en el otro lenguaje — y puedes ver exactamente qué parte de la petición se lleva.

### Para pensar

La propagación funcionaba sola hasta que sustituiste una librería. ¿Qué forma general tiene ese riesgo en un código real — dónde más se esconde? (Piensa: colas de mensajes, trabajos en segundo plano, pools de hilos, `setTimeout`, cualquier cosa que salga del contexto de ejecución de la petición.)

### Si te atascas

- **`inject()` no añade nada a las cabeceras.** No hay span activo en ese punto del código — estás fuera del contexto de la petición. Loguea `trace.getActiveSpan()` / `trace.get_current_span()` para confirmarlo.
- **Los dos servicios tienen spans pero con trace ids distintos tras el arreglo.** Comprueba que la cabecera llega realmente al cable; `docker logs workshop-notify-py` (o `-js`) imprime las cabeceras que recibió.

## Ejercicio 7.5 — Sampling [~20 min] *(recortable)*

**Deberías poder responder esto al terminar:** *¿por qué "muestrea el 10% de las trazas" es un mal valor por defecto para un error budget?*

### Hazlo

**Head sampling** — la decisión se toma cuando arranca la traza, en el SDK:

```yaml
environment:
  OTEL_TRACES_SAMPLER: parentbased_traceidratio
  OTEL_TRACES_SAMPLER_ARG: "0.1"
```

Reinicia, lanza el loadgen y mira cómo adelgazan las trazas. Barato, sencillo y funciona a cualquier escala — un proceso decide y todos los servicios aguas abajo respetan la decisión a través de `traceparent`.

Y ahora fíjate en el problema: **acabas de tirar el 90% de tus errores junto con el 90% de tus éxitos.** Las trazas que más querías tenían exactamente la misma probabilidad de ser descartadas que las aburridas.

**Tail sampling** — la decisión se toma en el Collector después de que haya llegado la traza entera:

```yaml
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: keep-errors
        type: status_code
        status_code: { status_codes: [ERROR] }
      - name: keep-slow
        type: latency
        latency: { threshold_ms: 500 }
      - name: sample-the-rest
        type: probabilistic
        probabilistic: { sampling_percentage: 10 }
```

Añádelo al pipeline de trazas, apaga el head sampling, reinicia el Collector y confirma que sobreviven todas las trazas con error mientras las correctas adelgazan.

### Comprueba en la UI de Victoria

Con tail sampling activo, lanza el loadgen y compara:

```
_time:15m "resource_attr:service.name":=todo-api status_code:=2 | stats count()
```

```
_time:15m "resource_attr:service.name":=todo-api status_code:=0 | stats count()
```

Deberías ver: los errores conservados enteros, y los éxitos reducidos a más o menos una décima parte.

### Comprueba en Grafana

Compara `rate(http_server_duration_milliseconds_count[1m])` — que no se ve afectado, porque las métricas no se muestrean — con el número de trazas que puedes encontrar. Esa diferencia es lo que hay que interiorizar: **el sampling cambia tus trazas y no tus métricas**, que es una de las buenas razones para tener las dos.

### Para pensar

El tail sampling necesita que la traza *entera* llegue a *una sola* instancia del Collector antes de poder decidir. Eso condiciona cómo escalas tu flota de Collectors: necesitas hashing consistente por trace id delante, o una topología de dos niveles.

Aquí no hay respuesta gratis. ¿Qué harías de verdad con 200 servicios y 50k spans/segundo?

### Si te atascas

- **El tail sampling lo tira todo.** `decision_wait` tiene que ser mayor que tu traza más lenta, o los spans que llegan tarde se juzgan contra una traza incompleta.
- **Head y tail sampling los dos activos.** Se multiplican y no vas a ver casi nada. Apaga primero el head sampling.

## Ejercicio 7.6 — Explora [~15 min, repartido entre los anteriores]

**Deberías poder responder esto al terminar:** *¿a cuál de las tres UIs de trazas acudirías, y cuándo?*

### Hazlo

Mira los mismos datos de tres formas, deliberadamente.

**1. vmui de VictoriaTraces** (`http://localhost:10428/select/vmui`)

Fíjate en lo que estás escribiendo: **LogsQL**. El mismo lenguaje del Módulo 6. Como decía el Módulo 2: los logs y las trazas comparten lenguaje de consulta.

```
_time:15m "resource_attr:service.name":=todo-api duration:>500ms
```

```
_time:15m "span_attr:todo.list":=work status_code:=2
```

```
_time:15m "resource_attr:service.name":=todo-api | stats by (name) count()
```

**2. Grafana → VictoriaTraces (Jaeger)**

La cascada. El mapa de servicios. El filtrado de spans dentro de una traza. Esta es la vista para entender *una* petición a fondo.

**3. Grafana → VictoriaTraces (Tempo) → Traces Drilldown**

Paneles RED generados a partir de tus spans, sin dashboard hecho a mano. Experimental, pero realmente útil para "¿dónde debería estar mirando?".

### Comprueba en la UI de Victoria

Ejecuta las tres consultas de arriba y consigue un resultado en cada una.

### Comprueba en Grafana

Abre la misma traza en la vista de Jaeger y en Drilldown. Fíjate en qué facilita cada una y qué dificulta.

### Para pensar

¿Cuándo acudirías a cada una? Una respuesta razonable:

- **vmui** — "encuentra la aguja por atributo". Sabes qué buscas.
- **Vista Jaeger** — "entiende esta petición". Tienes un trace id.
- **Drilldown** — "¿dónde miro siquiera?". Tienes un síntoma y ninguna hipótesis.

### Si te atascas

- **`duration:>500ms` no devuelve nada.** Los filtros de duración necesitan la unidad. Prueba `duration:>500ms` exactamente, o `duration:>0.5s`.

## Lo Que Hemos Conseguido

- Spans personalizados que describen tu dominio, con atributos elegidos por iteración y no por corazonada
- Errores registrados como status de span y excepciones, no solo como texto en otro sitio
- Al menos un atributo inventado sustituido por su nombre estándar
- Una traza rota a propósito en una frontera de servicio, y reparada a mano
- Head y tail sampling aplicados, con una explicación honesta del compromiso
- Tres UIs de trazas comparadas lado a lado

## Siguientes Pasos

Ya tienes tres señales. Ahora conviértelas en una sola cosa. Continúa con el [Módulo 8: Correlación](./08-correlation.es.md).
