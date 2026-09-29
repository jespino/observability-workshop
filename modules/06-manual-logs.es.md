# Módulo 6: Logs Manuales

**Duración:** ~75 min · **Checkpoints: 62**

Ahora mismo tus logs son texto. El texto es lo que grepeas a las 3 de la mañana deseando tener algo mejor. Este módulo los convierte en registros que puedes filtrar, agregar y — lo importante — unir a una traza. 🪵

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Explicar qué contiene un registro de log de OpenTelemetry, y qué va en el cuerpo frente a los atributos
- Emitir logs estructurados que arrastren `trace_id` y `span_id` automáticamente
- Registrar un camino de error de forma que otra persona pueda actuar sobre él
- Consultar logs con LogsQL, incluidas agregaciones
- Saber cuándo usar el SDK y cuándo usar stdout más un receiver del Collector

## Prerrequisitos

- Módulo 5 completado y con checkpoint

## 6.1 — Conceptos [~10 min]

Un **registro de log** de OpenTelemetry no es una cadena. Tiene:

| Campo | Qué va dentro |
|---|---|
| `timestamp` | cuándo |
| `severity` | `DEBUG` / `INFO` / `WARN` / `ERROR`, como texto y como número |
| `body` | el mensaje legible por humanos — *y nada más* |
| `attributes` | los campos estructurados: `list`, `todo_id`, `user`, `error.type` |
| `trace_id` / `span_id` | ⭐ las claves de correlación |
| resource | heredado: `service.name`, `service.version`, `deployment.environment` |

Toda la disciplina del logging estructurado es una decisión, tomada una y otra vez: **qué va en el cuerpo y qué va en los atributos.**

La regla que funciona: el cuerpo debe ser una cadena corta y *constante* que describa el tipo de evento. Todo lo variable va en atributos. `"todo created"` con `{todo_id: "...", list: "work"}` — nunca `"todo abc-123 creado en work"`. Lo primero se puede contar, agrupar y filtrar. Lo segundo solo se puede grepear.

### Dos arquitecturas legítimas

Las dos las usan equipos serios en producción, y merece la pena saber por qué elegirías cada una:

**A. SDK → OTLP → Collector** *(la que vamos a construir)*
La correlación con el contexto de traza es automática y limpia. Coste: el SDK de logs sigue marcado como experimental en los dos lenguajes.

**B. JSON por stdout → receiver `filelog` del Collector**
Sólido como una roca, sirve para cualquier lenguaje y cualquier contenedor de terceros, sin dependencia experimental. Coste: el contexto de traza lo inyectas tú en el JSON, y la configuración de parseo es cosa tuya.

Construiremos la A y demostraremos la B en 6.4.

## Ejercicio 6.2 — Logging estructurado con contexto de traza [~25 min] [checkpoint: 62]

**Deberías poder responder esto al terminar:** *dada una línea de log de error, ¿cómo llegas a la traza de la petición que la produjo?*

### Hazlo

1. Prepara el logger estructurado.

<!-- track:js -->

La app ya usa pino, en `src/log.js`:

```js
const pino = require('pino');

const log = pino({ level: process.env.LOG_LEVEL ?? 'info' });

module.exports = { log };
```

No hay nada que instalar: `@opentelemetry/instrumentation-pino` viene en el bundle de
auto-instrumentations que añadiste en el Módulo 3, y por eso esas líneas de log llevan
llegando a VictoriaLogs desde entonces. Lo que falta es *estructura*.

Hoy los handlers lo llaman así — un mensaje con los datos fundidos dentro:

```js
log.info('todo created ' + todo.id);
```

Cámbialos para pasar primero un objeto de atributos y después un mensaje **constante**:

```js
const { log } = require('../log.js');

log.info({ list: todo.list, todo_id: todo.id }, 'todo created');
```

<!-- /track -->

<!-- track:python -->

La app ya tiene un logger, en `todo_api/log.py`:

```python
import logging
import os

LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()

logging.basicConfig(level=LEVEL)

# Fijamos el nivel en nuestro propio logger en vez de confiar en el del root.
# Con `opentelemetry-instrument`, el logger root se configura antes de que la
# aplicación importe nada, lo que convierte un basicConfig() posterior en un no-op.
log = logging.getLogger("todo-api")
log.setLevel(LEVEL)
```

No hay nada que instalar: `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true` del
Módulo 3 enganchó el `LoggingHandler` del SDK al logger root, y por eso esas líneas de
log llevan llegando a VictoriaLogs desde entonces. Lo que falta es *estructura*.

> 📌 Ese comentario sobre el logger root no es decoración. Si configuras el logging por
> tu cuenta antes de leerlo, tus registros `INFO` desaparecen en silencio. Es la forma
> más común de que un servicio Python acabe sin logs en el backend y sin ningún error
> en ninguna parte.

Hoy los handlers lo llaman así — un mensaje con los datos fundidos dentro:

```python
log.info("todo created %s", todo.id)
```

Cámbialos para meter los datos variables en `extra` y dejar el mensaje **constante**:

```python
log.info("todo created", extra={"list": todo.list, "todo_id": todo.id})
```

<!-- /track -->

2. **Convierte tres sentencias de log existentes** a forma estructurada. Buenos candidatos:
   - el camino de creación (`todo created`)
   - el camino de completado (`todo completed`)
   - el camino de caché en `cache.js` / `cache.py` — registra la clave que buscaste y si acertó

3. **Registra bien el camino de error.** Busca el handler que devuelve un 500 cuando el título está vacío (lo conociste en el Módulo 1.2) y haz que produzca algo accionable:

<!-- track:js -->

```js
try {
  const todo = await createTodo(req.body);
  // ...
} catch (err) {
  log.error({
    err,                                  // pino serializa tipo, mensaje y stack
    list: req.body?.list,
    title_length: req.body?.title?.length ?? 0,
  }, 'create todo failed');
  res.status(500).json({ error: 'internal' });
}
```

<!-- /track -->

<!-- track:python -->

```python
try:
    todo = create_todo(request.get_json())
    # ...
except Exception:
    log.exception("create todo failed", extra={          # exception() captura el stack
        "list": (request.get_json() or {}).get("list"),
        "title_length": len((request.get_json() or {}).get("title") or ""),
    })
    return jsonify(error="internal"), 500
```

<!-- /track -->

Fíjate en lo que hay ahí: severidad, la excepción, y **los atributos que otra persona necesitaría para reproducirlo**. `title_length` en lugar de `title`, porque el título podría contener algo que no deberías estar guardando — un hábito que conviene adquirir ya, y al que volveremos en el Módulo 10.

4. Reconstruye y crea un todo fallido para generar uno de estos:

   <!-- track:js -->

   Reinicia la API — Ctrl-C en su terminal, y luego ↑ Enter — y en otra terminal:

   ```bash
   curl -s -X POST localhost:8081/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
   ```

   <!-- /track -->

   <!-- track:python -->

   Reinicia la API — Ctrl-C en su terminal, y luego ↑ Enter — y en otra terminal:

   ```bash
   curl -s -X POST localhost:8082/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   TRACK=py ./loadgen.sh 2m
   ```

   <!-- /track -->

### Comprueba en la UI de Victoria

vmui de VictoriaLogs (`http://localhost:9428/select/vmui`):

```
service.name:todo-api AND severity_text:ERROR
```

Deberías ver: tu registro `create todo failed`. Ábrelo y confirma que están todos estos:

- `list` y `title_length` como campos reales, no como texto en el mensaje
- un stack trace o el tipo de excepción
- **`trace_id` y `span_id`**

Luego comprueba que la correlación se mantiene también en el camino feliz:

```
service.name:todo-api AND _msg:"todo created" | fields _time, trace_id, list, todo_id
```

Todas las filas tienen que tener `trace_id`. **Si alguna fila lo tiene vacío, ese es el verdadero ejercicio** — ve a averiguar por qué.

> 🔎 Las dos causas habituales: en JS, loguear desde fuera de un contexto de span activo (un callback, un timer, un `setImmediate`). En Python, un logger creado y configurado *antes* de adjuntar el handler, con lo que nunca recibió el de OTel.

Por último, copia un `trace_id` de cualquier línea de log y pégalo en la vmui de VictoriaTraces:

```
trace_id:=<pega aquí el valor>
```

Deberías aterrizar en la petición exacta que produjo esa línea de log. Ese es el premio, y es el mecanismo que hay detrás de todo el Módulo 8.

### Comprueba en Grafana

Explore → **VictoriaLogs** → la misma consulta. Activa el modo **Live** y mira cómo llegan los registros mientras corre el loadgen.

### Para pensar

En el Módulo 5.4, añadir `todo_id` como atributo de métrica fue un desastre que consumió memoria hasta que reseteaste la base de datos. Aquí, `todo_id` como atributo **de log** no solo está bien, es el campo más útil del registro.

El mismo valor. El veredicto opuesto. ¿Por qué? Dilo en voz alta — si sabes articularlo, has entendido lo más útil en la práctica de todo este taller.

### Si te atascas

- **Llegan logs pero sin atributos.** Probablemente sigas interpolando dentro del mensaje. `log.info("created " + id)` produce un cuerpo y nada más.
- **(JS) No hay `trace_id` por ninguna parte.** La instrumentación de pino solo parchea instancias creadas *después* de que cargara el SDK, y solo parchea llamadas `require()` de CommonJS. Comprueba que el flag `--require` sigue en el comando de arranque y que no convertiste `src/log.js` en un módulo ES.
- **(Python) No llega nada a VictoriaLogs.** Comprueba que `OTEL_LOGS_EXPORTER=otlp` sigue configurado, y que adjuntaste el handler al logger raíz y no a uno hijo.

```bash
./checkpoint.sh 62
```

## Ejercicio 6.3 — Streams y LogsQL [~25 min]

**Deberías poder responder esto al terminar:** *¿qué es un stream de VictoriaLogs, y qué pasa si eliges mal sus campos?*

### Hazlo

**Parte 1 — Streams.**

En el Módulo 4 leíste esta línea de la configuración del Collector:

```yaml
headers:
  VL-Stream-Fields: "service.name,deployment.environment"
```

Eso define qué es un *stream*. VictoriaLogs agrupa los registros en streams y guarda juntos los datos de cada stream, y eso es lo que lo hace rápido y compacto.

1. Mira qué produjo tu elección:

   ```
   _stream:{service.name="todo-api"}
   ```

2. Ahora rómpelo a propósito. En la configuración del Collector, cambia la cabecera por:

   ```yaml
   VL-Stream-Fields: "service.name,trace_id"
   ```

   Reinicia el Collector, lanza el loadgen un minuto y observa. Acabas de crear un stream por petición. El rendimiento de consulta se degrada, el almacenamiento crece y el selector de stream se vuelve inútil.

3. Vuelve a dejarlo como estaba y reinicia. 🙏

**Parte 2 — LogsQL, paso a paso.**

Pega cada una de estas en la vmui de VictoriaLogs, en orden. Cada una añade una idea:

```
service.name:todo-api
```

```
service.name:todo-api AND severity_text:ERROR
```

```
service.name:todo-api AND _time:5m | stats by (severity_text) count()
```

```
service.name:todo-api | filter list:"work" | fields _time, _msg, trace_id
```

```
_stream:{service.name="todo-api"} | top 10 by (list)
```

Luego prueba los tres modos de visualización de la UI — **Group**, **Table** y **JSON**. El modo JSON muestra la respuesta cruda de `/select/logsql/query`, que es exactamente lo que obtendrías desde un script o una comprobación de CI:

```bash
curl -s http://localhost:9428/select/logsql/query \
  --data-urlencode 'query=service.name:todo-api AND severity_text:ERROR AND _time:1h' | head -c 1000
```

### Comprueba en la UI de Victoria

La consulta `stats by (level) count()` debería devolver una fila por severidad presente, con sus cuentas. Si falta `ERROR`, genera uno con el curl de título vacío de 6.2.

### Comprueba en Grafana

Explore → VictoriaLogs. Prueba las mismas consultas con el **constructor visual** y luego cambia a la caja de LogsQL en crudo para ver qué generó. El constructor va bien para explorar; la caja en crudo es lo que pegarás en un runbook.

### Para pensar

Acabas de aprender un lenguaje de consulta. En el Módulo 7 vas a consultar **spans** con el mismo. ¿Qué te ahorra eso, comparado con un stack donde métricas, logs y trazas tienen cada uno su sintaxis?

### Si te atascas

- **`_stream:{...}` no devuelve nada.** Los selectores de stream casan con los campos declarados en `VL-Stream-Fields` en el momento de *ingesta*. Los registros escritos antes de tu cambio tienen la forma de stream antigua.
- **`| stats` da error.** Los operadores de pipe van después del filtro, y `_time:5m` es un filtro, no un pipe.

## Ejercicio 6.4 — Grafana, y el camino alternativo [~15 min]

**Deberías poder responder esto al terminar:** *¿cuándo decidirías deliberadamente no usar el SDK de logs?*

### Hazlo

1. Abre el dashboard que construiste en el Módulo 5.5 y añade un panel de **Logs**:
   - Datasource: VictoriaLogs
   - Consulta: `service.name:todo-api AND list:$list`
   - Colócalo debajo de los paneles de negocio

   Ahora tu variable de plantilla `list` filtra métricas *y* logs a la vez. Es el primer sabor de la correlación, y te ha costado una línea.

2. **Mira la demo — no la construyas.** Tu facilitador enseñará la arquitectura alternativa: la app escribe JSON por stdout y el Collector lo recoge con un receiver `filelog`:

   ```yaml
   receivers:
     filelog:
       include: [/var/lib/docker/containers/*/*.log]
       operators:
         - type: json_parser
           timestamp: { parse_from: attributes.time, layout: '%Y-%m-%dT%H:%M:%S.%fZ' }
         - type: trace_parser
           trace_id: { parse_from: attributes.trace_id }
           span_id:  { parse_from: attributes.span_id }
   ```

   Fíjate en el operador `trace_parser`: con esta arquitectura eres tú quien tiene que meter el `trace_id` en tu JSON, y luego decirle al Collector dónde encontrarlo.

3. Debatid cuándo la elegirías:
   - Servicios de terceros a los que no puedes añadir un SDK
   - Lenguajes donde el SDK de logs no está listo
   - Organizaciones que no van a llevar un paquete experimental a producción
   - Cualquier sitio donde ya tienes buen logging JSON y no quieres rehacerlo

### Comprueba en Grafana

El panel de logs del dashboard debería responder a la variable `list`. Ponla en `work`, luego en `home`, y confirma que el panel cambia.

### Para pensar

La arquitectura B desacopla tu aplicación de OpenTelemetry casi por completo — la app solo escribe JSON. ¿Eso es una ventaja o una oportunidad perdida?

### Si te atascas

- **El panel de logs ignora `$list`.** En LogsQL la variable se interpola como valor pelado: `list:$list` funciona, `list:"$list"` puede que no si el valor no tiene espacios. Revísalo en el inspector de consultas del panel.

## Lo Que Hemos Conseguido

- Logs que son registros, no cadenas
- Cada registro llevando contexto de traza, verificado
- Un camino de error sobre el que otra persona puede actuar de verdad
- LogsQL funcionando, incluida agregación y la API de consulta en crudo
- Un panel de logs conectado a la misma variable de plantilla que tus métricas
- Una visión clara de la alternativa stdout+Collector

## Siguientes Pasos

Las métricas te dicen *que*. Los logs te dicen *qué*. Continúa con el [Módulo 7: Trazas Manuales](./07-manual-traces.es.md) para averiguar *dónde* y *por qué*.
