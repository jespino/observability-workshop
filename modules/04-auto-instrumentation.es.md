# Módulo 4: El Camino Fácil — Auto-instrumentación Sin Código

En este módulo la aplicación empieza a emitir métricas, logs y trazas. No vas a cambiar ni una línea de código de aplicación para conseguirlo. ⚡

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Tener telemetría llegando desde `todo-api` a los backends de Victoria sin cambiar una línea del código de la aplicación
- Leer una traza distribuida e identificar los atributos de convención semántica de un span
- Moverte con soltura por las tres UIs de Victoria y por el dashboard Auto-Instrumentation de Grafana

## Ejercicio 4.1 — Enciéndelo

**Deberías poder responder esto al terminar:** *¿qué ficheros has tenido que tocar para tener telemetría, y qué fichero de la aplicación no?*

### Hazlo

<!-- track:js -->

1. Añade los dos paquetes a las `dependencies` de `track-js/todo-api/package.json`:

   ```json
   "dependencies": {
     "@opentelemetry/api": "latest",
     "@opentelemetry/auto-instrumentations-node": "latest",
     "express": "^4.21.2",
     ...
   }
   ```

   El contenedor de tu API ejecuta `npm install` cada vez que arranca, así que no hay nada
   que instalar a mano. No se mete nada en ninguna imagen, y no se reconstruye ninguna
   imagen.

2. Crea `track-js/todo-api/.env`. **Es el único fichero que creas, y la aplicación no lo
   lee nunca** — `docker-compose.yml` ya se lo pasa al contenedor como entorno
   (`env_file:`), exactamente igual que una plataforma de despliegue le entregaría el
   entorno a un proceso:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   NODE_OPTIONS=--require @opentelemetry/auto-instrumentations-node/register
   ```

   Esa última línea es la que de verdad enciende el SDK. Es la forma canónica:

   ```
   node --require @opentelemetry/auto-instrumentations-node/register src/server.js
   ```

   > 🚨 **El endpoint es `otel-collector`, no `localhost`.** Es el nombre de servicio del
   > Collector en `docker-compose.yml`. Dentro del contenedor de tu API, `localhost` es el
   > contenedor, y el SDK se pasaría todo el día fallando al exportar a la nada.

   > ⚠️ El `--require` tiene que cargarse **antes** que el código de tu aplicación, o la
   > instrumentación no tendrá nada a lo que engancharse. Además solo parchea llamadas
   > `require()` de CommonJS, y por eso esta aplicación es CommonJS y no ESM. Con ESM
   > tendrías spans de HTTP y perderías en silencio la instrumentación de Express, de base
   > de datos y de logging — un modo de fallo especialmente feo, porque parece que funciona.

3. Recrea el contenedor de tu API, para que coja el entorno nuevo e instale las
   dependencias nuevas. Esta vez no basta con guardar un fichero — lo que ha cambiado es la
   configuración del contenedor, no tu código:

   ```bash
   docker compose up -d --force-recreate todo-api-js
   ```

   Después abre una segunda terminal y sigue en ella el log de tu API. Déjala abierta el
   resto del día — los guiones la llaman la *terminal del log*:

   ```bash
   docker compose logs -f todo-api-js
   ```

   Espera a la línea `todo-api listening on 8081`. La imagen es la misma, el comando es el
   mismo, tu código es el mismo. Lo único nuevo es el entorno.

   Puede que también veas un `MetadataLookupWarning`. Es el SDK comprobando si corre en un
   proveedor de nube, para describir dónde se ejecuta. Fuera de una nube la comprobación
   falla; es inofensivo.

4. Genera tráfico en otra terminal:

   ```bash
   ./loadgen.sh 2m
   ```

<!-- /track -->

<!-- track:python -->

1. Añade los paquetes a `track-py/todo-api/requirements.txt`:

   ```
   opentelemetry-distro>=0.66b1
   opentelemetry-exporter-otlp
   ```

   El contenedor de tu API ejecuta `pip install` cada vez que arranca, así que no hay nada
   que instalar a mano.

2. Crea `track-py/todo-api/.env`. **Es el único fichero que creas, y la aplicación no lo lee
   nunca** — `docker-compose.yml` ya se lo pasa al contenedor como entorno (`env_file:`),
   exactamente igual que una plataforma de despliegue le entregaría el entorno a un proceso:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   OTEL_PYTHON_LOG_AUTO_INSTRUMENTATION=true
   OTEL_PYTHON_LOG_CORRELATION=true
   ```

   A diferencia del track de JavaScript, ninguna variable de entorno puede encender el SDK
   de Python. Lo que cambias es el **comando**. En `docker-compose.yml`, busca el servicio
   `todo-api-py` y pon `opentelemetry-instrument` delante de `gunicorn`:

   ```yaml
       command: opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   Esa es la forma canónica. No añadas el `--reload` de gunicorn: el contenedor ya reinicia
   todo el comando cuando cambia un fichero, y bajo `opentelemetry-instrument` el recargador
   propio de gunicorn hace que los logs dejen de exportarse sin decir nada.

3. Recrea el contenedor de tu API, para que coja el entorno nuevo e instale las
   dependencias nuevas. Esta vez no basta con guardar un fichero — lo que ha cambiado es la
   configuración del contenedor, no tu código:

   ```bash
   docker compose up -d --force-recreate todo-api-py
   ```

   Después abre una segunda terminal y sigue en ella el log de tu API. Déjala abierta el
   resto del día — los guiones la llaman la *terminal del log*:

   ```bash
   docker compose logs -f todo-api-py
   ```

   Se queda en silencio un minuto mientras pip instala los paquetes nuevos, y luego
   gunicorn escribe `Listening at: http://0.0.0.0:8082`. La imagen es la misma y tu código no
   ha cambiado. Lo nuevo es el entorno y una
   palabra delante del comando.

4. Genera tráfico en otra terminal:

   ```bash
   ./loadgen.sh 2m
   ```

<!-- /track -->

<!-- track:go -->

1. No hay nada que añadir a `go.mod`. En este track la instrumentación no se carga cuando
   arranca el programa — Go no tiene un mecanismo para eso. Se **compila dentro del
   binario**, con una herramienta llamada `otelc` que envuelve a `go build`: encuentra las
   librerías que usa tu código (`net/http`, `database/sql`, …) y les añade OpenTelemetry al
   compilar. Ya está instalada en el contenedor de tu API.

2. Crea `track-go/todo-api/.env`. **Es el único fichero que creas, y la aplicación nunca lo
   lee** — `docker-compose.yml` ya se lo pasa al contenedor como entorno (`env_file:`),
   exactamente igual que una plataforma de despliegue le pasaría el entorno a un proceso:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   ```

   Ahora cambia cómo se **compila** el binario. En `docker-compose.yml`, busca el servicio
   `todo-api-go`. Su comando ejecuta `air`, que recompila y reinicia el servidor cada vez
   que guardas un fichero `.go`. El `--build.cmd` del final de la línea es cómo compila:

   ```yaml
       command: air --tmp_dir /tmp/air --build.exclude_file "otel.instrumentation.go,otelc.runtime.go" --build.entrypoint /tmp/todo-api --build.cmd "go mod tidy && go build -o /tmp/todo-api ."
   ```

   Pon `otelc` delante de `go build`:

   ```yaml
       command: air --tmp_dir /tmp/air --build.exclude_file "otel.instrumentation.go,otelc.runtime.go" --build.entrypoint /tmp/todo-api --build.cmd "go mod tidy && otelc go build -o /tmp/todo-api ."
   ```

   Esa es la forma canónica: `otelc go build` en vez de `go build`, y nada más cambia.

3. Recrea el contenedor de tu API, para que coja el entorno nuevo y el nuevo comando de
   compilación. Esta vez no basta con guardar un fichero de código — ha cambiado la
   configuración del contenedor, no tu código:

   ```bash
   docker compose up -d --force-recreate todo-api-go
   ```

   Después abre una segunda terminal y sigue en ella el log de tu API. Déjala abierta el
   resto del día — los guiones la llaman tu *terminal de logs*:

   ```bash
   docker compose logs -f todo-api-go
   ```

   La primera compilación tarda un minuto mientras `otelc` descarga la instrumentación;
   las siguientes tardan un par de segundos. Después verás `OpenTelemetry initialized` y
   `todo-api listening on 8085`. La imagen es la misma y tu código no ha cambiado. Lo
   nuevo es el entorno y una palabra delante de `go build`.

4. Genera tráfico en otra terminal:

   ```bash
   ./loadgen.sh 2m
   ```

<!-- /track -->

Tres ajustes merecen explicación en vez de copiar y pegar — dos de ese bloque y uno de `docker-compose.yml`:

- **`OTEL_METRIC_EXPORT_INTERVAL=10000`.** El valor por defecto es 60 segundos. Sesenta segundos sin nada después de un cambio se leen como "está roto"; con diez segundos ves el resultado casi al momento. En producción lo dejarías por defecto — cada exportación se paga.
- **`OTEL_METRIC_EXPORT_TIMEOUT=5000`.** No puede ser mayor que el intervalo, y su valor por defecto es 30 segundos. Baja el intervalo sin bajar esto y el SDK tiene que resolver el conflicto por ti: los SDKs de JavaScript antiguos se negaban a arrancar, y los actuales acortan el timeout sin decir nada. Fijar los dos explícitamente evita ambas cosas.
- **`-search.latencyOffset=1s` en VictoriaMetrics** — puesto en `docker-compose.yml`. Por defecto VM oculta los últimos 30 segundos de datos, para que un intervalo escrito a medias nunca produzca una gráfica engañosa. Es el comportamiento correcto y es el equivocado para un aula. Fíjate en que *corrección* y *frescura* están aquí en tensión real.

## Ejercicio 4.2 — Explora lo que ha llegado

**Deberías poder responder esto al terminar:** *¿qué ha aprendido el SDK sobre tu aplicación sin que nadie se lo cuente?*

### Hazlo

1. Asegúrate de que hay tráfico: `./loadgen.sh 2m`.
2. Recorre cada una de las cuatro UIs de abajo. No las mires por encima — vas a vivir en ellas el resto del día.

### Comprueba en la UI de Victoria

**VictoriaMetrics** (`http://localhost:8428/vmui`) — mira qué ha aparecido:

```
{__name__=~"http.*"}
```

y luego

<!-- track:js -->

```
rate(http_server_request_duration_seconds_count{service_name="todo-api"}[1m])
```

<!-- /track -->

<!-- track:python -->

```
rate(http_server_duration_milliseconds_count{service_name="todo-api"}[1m])
```

<!-- /track -->

<!-- track:go -->

```
go_goroutine_count{service_name="todo-api"}
```

`otelc` todavía no registra métricas HTTP: su instrumentación de HTTP produce spans, no
métricas. Lo que sí tienes son las métricas del propio runtime de Go — goroutines,
memoria, recolección de basura — todas empiezan por `go_`. En este track los paneles HTTP
de los dashboards se quedan vacíos, y eso también es información.

<!-- /track -->

**VictoriaLogs** (`http://localhost:9428/select/vmui`):

```
service.name:todo-api
```

Fíjate en la forma de lo que ha llegado. El mensaje es un bloque de texto en `_msg`. No
hay campo `list`, no hay campo `todo_id`, no hay nada por lo que filtrar o agregar.

La app solo escribe logs cuando se **crea** o se **completa** un todo, así que verás líneas como `todo created …`. El loadgen crea todos, así que habrá alguna.

<!-- track:go -->

En el track de Go aquí todavía no llega nada. `otelc` añade el trace ID a cada registro de
`slog`, pero no los envía a ninguna parte: tus logs siguen yendo solo a la salida del
contenedor, que es lo que muestra tu terminal de logs.

<!-- /track -->

> 🔑 **Nombres de campo en VictoriaLogs.** El mensaje es `_msg`, el nivel es
> `severity_text`, y los atributos de resource son campos normales —
> `service.name:todo-api`, sin prefijo y sin comillas.

**VictoriaTraces** (`http://localhost:10428/select/vmui`) — busca la traza de una
creación urgente. Solo los todos con la etiqueta `urgent` llaman a `notify-service`, así
que empieza por su lado:

```
_time:15m "resource_attr:service.name":=notify-service name:="POST /notify"
```

Copia el `trace_id` de cualquier fila y trae la traza entera:

```
trace_id:=<pega aquí el valor>
```

Ordena sus spans por sus padres, y este es el árbol que has conseguido gratis:

<!-- track:js -->

```
POST /todos                          ← span de servidor HTTP, nombrado por la ruta
├── middleware - jsonParser          ← las capas del propio framework web
├── middleware - query
└── request handler - /todos
    ├── pg-pool.connect              ← instrumentación de postgres
    ├── pg.query:INSERT todo
    ├── pg.query:UPDATE todo
    └── POST                         ← HTTP saliente a notify-service
        └── POST /notify             ← y el span del *otro servicio*, misma traza
```

<!-- /track -->

<!-- track:python -->

```
POST /todos                          ← span de servidor HTTP, nombrado por la ruta
├── INSERT                           ← instrumentación de postgres
├── UPDATE
└── POST                             ← HTTP saliente a notify-service
    └── POST /notify                 ← y el span del *otro servicio*, misma traza
```

<!-- /track -->

<!-- track:go -->

```
POST /todos                          ← span de servidor HTTP, nombrado por la ruta
├── INSERT                           ← instrumentación de database/sql
├── UPDATE
└── POST                             ← HTTP saliente a notify-service
    └── POST /notify                 ← y el span del *otro servicio*, misma traza
```

<!-- /track -->

Nadie ha escrito nada de eso. Fíjate sobre todo en las dos últimas líneas: la traza ya
cruza una frontera de servicio, porque la instrumentación del cliente HTTP inyectó una
cabecera `traceparent` a la salida y la del servidor HTTP del otro servicio la leyó a la
entrada. En los tracks de JS y de Go cruza además una frontera de lenguaje:
`notify-service` está escrito en Python. Las creaciones sin etiqueta terminan en el trabajo de base de datos.

> 🔑 **Nombres de campo en VictoriaTraces**, para contrastar con el recuadro de logs de
> arriba. Los atributos de resource se guardan con el prefijo `resource_attr:` y los de
> span con `span_attr:`, así que el nombre del campo lleva dos puntos y hay que
> entrecomillarlo:
> `"resource_attr:service.name":=todo-api`, `"span_attr:http.route":=/todos`.
> `:=` es coincidencia exacta. Los campos sin prefijo son `name`, `kind`, `duration`,
> `status_code`, `trace_id`, `span_id` y `parent_span_id`. El mismo lenguaje de
> consulta, dos esquemas distintos. Mejor darse cuenta ahora que en mitad de un incidente.

Lee los atributos del span raíz, `POST /todos`, y después los de un span de base de
datos. Busca estos tres, que son **convenciones semánticas** — nombres estándar que usa
cualquier servicio instrumentado con OTel del mundo para el método, la ruta y la base de
datos:

<!-- track:js -->

- `span_attr:http.request.method`
- `span_attr:url.path`
- `span_attr:db.system.name`

<!-- /track -->

<!-- track:python -->

- `span_attr:http.method`
- `span_attr:http.target`
- `span_attr:db.system`

<!-- /track -->

<!-- track:go -->

- `span_attr:http.request.method`
- `span_attr:url.path`
- `span_attr:db.system.name`

<!-- /track -->

Los tracks no usan todos los mismos nombres para las mismas tres cosas. Las convenciones
cambiaron hace unas cuantas versiones: las librerías de instrumentación de JavaScript y de
Go ya usan los nombres nuevos, y las de Python siguen con los antiguos. La misma petición, nombres distintos:
conviene saberlo antes de escribir una consulta o un dashboard contra ellos.

### Comprueba en Grafana

Grafana lleva en marcha desde el Módulo 2. Abre <http://localhost:3000> y entra con
`admin` / `admin`.

**Connections → Data sources.** Hay tres pre-provisionados desde `grafana/provisioning`:

| Nombre | Tipo | Apunta a |
|---|---|---|
| VictoriaMetrics | `victoriametrics-metrics-datasource` | `victoriametrics:8428` |
| VictoriaLogs | `victoriametrics-logs-datasource` | `victorialogs:9428` |
| VictoriaTraces (Jaeger) | `jaeger` (core, sin plugin) | `victoriatraces:10428/select/jaeger` |

Entra en cada uno y pulsa **Save & test**. Los tres tienen que ponerse verdes.

Fíjate en el host de esa última columna, y fíjate en que *no* es lo que escribes tú en tu
propio navegador. Grafana está en el mismo proyecto de Compose que los tres backends, así
que llega a ellos por nombre de servicio, igual que el Collector, e igual que tu API llega
a `otel-collector`. Tú llegas a esos mismísimos contenedores en `localhost`, a través de
los puertos que Compose publicó en tu portátil, porque tú no estás en un contenedor.

Después abre **Dashboards → Workshop → Auto-Instrumentation** — todo lo anterior en una sola pantalla: peticiones, errores y duración a partir de las métricas HTTP, llamadas a `notify-service`, los logs, las trazas recientes y la salud del propio Collector. Todo sale solo de la auto-instrumentación.

## La conclusión

> **La auto-instrumentación te da la fontanería, no el significado.**

Sabe de HTTP, de tu framework web y de Postgres, porque son estándar, populares y alguien escribió una librería de instrumentación para ellos. No sabe nada de todos, listas, urgencias ni usuarios — y absolutamente nada de esa caché en proceso, porque ahí no intervino ninguna librería. Nadie ha visto nunca tu dominio.
