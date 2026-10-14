# Módulo 7: Logs Manuales

Ahora mismo tus logs son texto. El texto es lo que grepeas a las 3 de la mañana deseando tener algo mejor. Este módulo los convierte en registros que puedes filtrar, agregar y — lo importante — unir a una traza. 🪵

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Explicar qué contiene un registro de log de OpenTelemetry, y qué va en el cuerpo frente a los atributos
- Emitir logs estructurados que arrastren `trace_id` y `span_id` automáticamente
- Registrar un camino de error de forma que otra persona pueda actuar sobre él
- Consultar logs con LogsQL, incluidas agregaciones
- Saber cuándo usar el SDK y cuándo usar stdout más un receiver del Collector

## Ejercicio 7.1 — Logging estructurado con contexto de traza

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
auto-instrumentations que añadiste en el Módulo 4, y por eso esas líneas de log llevan
llegando a VictoriaLogs desde entonces. Lo que falta es *estructura*.

Hoy los handlers lo llaman así — un mensaje con los datos fundidos dentro:

```js
log.info('todo created ' + todo.id);
```

Cámbialos para pasar primero un objeto de atributos y después un mensaje **constante**:

```js
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

No hay nada que instalar: con `OTEL_PYTHON_LOG_AUTO_INSTRUMENTATION=true` del Módulo 4, la
instrumentación de logging enganchó su handler al logger root, y por eso esas líneas de
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
log.info("todo created", extra={"list": todo.list, "todo_id": str(todo.id)})
```

<!-- /track -->

<!-- track:go -->

La app registra logs con `slog`, en `log.go`:

```go
var log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

Ese handler solo escribe líneas JSON en la salida del contenedor, y por eso nada de tu API
ha llegado todavía a VictoriaLogs: `otelc` añade el trace ID a cada registro, pero no envía
los logs a ninguna parte. Manda también los mismos registros a OpenTelemetry, con el bridge
`otelslog` — un handler para la terminal y otro para OpenTelemetry:

```go
import (
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var log = slog.New(slog.NewMultiHandler(
	slog.NewJSONHandler(os.Stdout, nil),
	otelslog.NewHandler("todo-api"),
))
```

El bridge le pasa cada registro al logger provider que `otelc` preparó en el Módulo 4. Lo
que falta ahora es *estructura*.

Hoy los handlers lo llaman así — un mensaje con los datos fundidos dentro:

```go
log.Info("todo created " + todo.ID)
```

Cámbialos para pasar el contexto de la petición, un mensaje **constante** y los datos como
pares clave-valor:

```go
log.InfoContext(ctx, "todo created", "list", todo.List, "todo_id", todo.ID)
```

La variante `Context` no es opcional: el bridge saca el trace ID del span que hay en `ctx`.
Sin ella, el registro llega sin trace ID.

<!-- /track -->

2. **Convierte tres sentencias de log existentes** a forma estructurada. Buenos candidatos:
   - el camino de creación (`todo created`)
   - el camino de completado (`todo completed`)
   - la consulta a la caché en el handler de `GET /todos` — registra la clave que buscó y si
     acertó. Las dos cosas vienen en lo que devuelve `readSummary` / `read_summary` (en Python,
     renombra su `_key` a `key` para poder usarla; en Go, el `_` de `hit, _, value`)

3. **Registra bien el camino de error.** Envuelve todo el cuerpo del handler de `POST /todos`, para que cuando algo falle dentro registre algo accionable:

<!-- track:js -->

```js
todosRouter.post('/todos', async (req, res) => {
  try {
    const todo = await createTodo(req.body);
    // ... el resto del handler, sin cambios, dentro del try
  } catch (err) {
    log.error({
      err,                                  // pino serializa tipo, mensaje y stack
      list: req.body?.list,
      title_length: req.body?.title?.length ?? 0,
    }, 'create todo failed');
    res.status(500).json({ error: 'internal' });
  }
});
```

<!-- /track -->

<!-- track:python -->

```python
@bp.post("/todos")
def post_todo():
    payload = request.get_json(silent=True) or {}
    try:
        todo = create_todo(payload)
        # ... el resto del handler, sin cambios, dentro del try
    except Exception:
        log.exception("create todo failed", extra={       # exception() captura el stack
            "list": payload.get("list"),
            "title_length": len(payload.get("title") or ""),
        })
        return jsonify(error="internal"), 500
```

<!-- /track -->

<!-- track:go -->

Dale al handler un único sitio desde el que informar de un fallo, y úsalo en cada camino de error:

```go
func postTodo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	body, _ := io.ReadAll(r.Body)

	var in struct {
		Title string `json:"title"`
		List  string `json:"list"`
	}
	_ = json.Unmarshal(body, &in)
	fail := func(err error) error {
		log.ErrorContext(ctx, "create todo failed",
			"err", err,
			"list", in.List,
			"title_length", len(in.Title),
		)
		return err                                    // sigue respondiendo 500
	}

	todo, err := createTodo(ctx, body)
	if err != nil {
		return fail(err)
	}
	// ... el resto del handler, con cada `return err` convertido en `return fail(err)`
}
```

<!-- /track -->

Fíjate en lo que hay ahí: severidad, la excepción, y **los atributos que otra persona necesitaría para reproducirlo**. `title_length` en lugar de `title`, porque el título podría contener algo que no deberías estar guardando — un hábito que conviene adquirir.

4. Guarda el fichero y crea un todo fallido para generar uno de estos:

   <!-- track:js -->

   Cuando la API se haya reiniciado (mira la terminal de su log), en otra terminal:

   ```bash
   curl -s -X POST localhost:8081/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
   ```

   <!-- /track -->

   <!-- track:python -->

   Cuando la API se haya reiniciado (mira la terminal de su log), en otra terminal:

   ```bash
   curl -s -X POST localhost:8082/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
   ```

   <!-- /track -->

   <!-- track:go -->

   Cuando la API se haya recompilado (míralo en su terminal de logs), en otra terminal:

   ```bash
   curl -s -X POST localhost:8085/todos -H 'content-type: application/json' -d '{"title":"","list":"work"}'
   ./loadgen.sh 2m
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

> 🔎 Las dos causas habituales: en JS, loguear desde fuera de un contexto de span activo (un callback, un timer, un `setImmediate`). En Python, loguear fuera de una petición — al importar un módulo, o desde un hilo en segundo plano — donde no hay ningún span activo del que sacar los IDs. En Go, llamar a `log.Info` en vez de a `log.InfoContext`, con lo que el bridge nunca ve el span.

Y como `list` es ahora un campo de verdad, puedes agregar por él:

```
service.name:todo-api AND _time:5m AND list:* | stats by (list) count()
```

Deberías ver: una fila por lista, con su cuenta.

Por último, copia un `trace_id` de cualquier línea de log y pégalo en la vmui de VictoriaTraces (`http://localhost:10428/select/vmui`):

```
trace_id:=<pega aquí el valor>
```

Deberías aterrizar en la petición exacta que produjo esa línea de log.

### Comprueba en Grafana

Abre Grafana en `http://localhost:3000`.

Abre **Dashboards → Workshop → Manual Instrumentation**. La fila **Structured logs** muestra tus registros. Pon la variable `list` en `work` y luego en `home`, y confirma que el panel cambia: filtra por el campo `list` que acabas de añadir.

> ℹ️ **La otra vía de entrada.** En vez del SDK de logs, una aplicación puede escribir líneas
> JSON por stdout y dejar que el Collector las lea con un receiver `filelog`. Entonces eres
> *tú* quien tiene que meter `trace_id` y `span_id` en ese JSON, y decirle al Collector dónde
> encontrarlos. Es la opción adecuada para servicios de terceros a los que no puedes añadir un
> SDK, para lenguajes cuyo SDK de logs no está listo, y allí donde ya existe un buen logging
> JSON.
