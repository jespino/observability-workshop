# Módulo 3: El Camino Fácil — Auto-instrumentación Sin Código

**Duración:** ~60 min · **Checkpoints: 31**

En este módulo la aplicación empieza a emitir métricas, logs y trazas. No vas a cambiar ni una línea de código de aplicación para conseguirlo. Después miraremos con lupa qué te da eso — y qué no. ⚡

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Tener las tres señales llegando desde `todo-api` a los backends de Victoria, configurado enteramente con variables de entorno
- Leer una traza distribuida e identificar los atributos de convención semántica de un span
- Moverte con soltura por las tres UIs de Victoria y por el modo Explore de Grafana
- Enunciar con precisión qué no puede contarte la auto-instrumentación, y por qué

## Prerrequisitos

- Módulos 0–2 completados
- Stack funcionando, `./doctor.sh` en verde

## Ejercicio 3.1 — Enciéndelo [~20 min] [checkpoint: 31]

**Deberías poder responder esto al terminar:** *¿qué fichero has tenido que editar para tener telemetría, y cuál no?*

### Hazlo

<!-- track:js -->

1. Añade los dos paquetes. La misma forma que la instalación del Módulo 0.2 — un
   contenedor de un solo uso escribiendo en tu directorio de código montado:

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     node:22-slim npm install @opentelemetry/api @opentelemetry/auto-instrumentations-node
   ```

   `package.json` y `node_modules/` de tu portátil han cambiado los dos. No se ha metido
   nada en ninguna imagen, y no se ha reconstruido ninguna imagen.

2. Crea `track-js/todo-api/.env`. **Es el único fichero que creas, y la aplicación no lo
   lee nunca** — se lo pasas al contenedor con `--env-file`, exactamente igual que una
   plataforma de despliegue le entregaría el entorno a un proceso:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://host.docker.internal:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   NODE_OPTIONS=--require @opentelemetry/auto-instrumentations-node/register
   ```

   > 🚨 **Dos cosas de ese fichero te van a morder si las cambias.**
   >
   > **No pongas comillas alrededor del valor de `NODE_OPTIONS`.** El `--env-file` de Docker
   > se toma literalmente el resto de la línea, comillas incluidas, así que
   > `NODE_OPTIONS="--require …"` llega con las comillas pegadas y Node ignora todo el
   > valor. Tu shell no se comporta así, y es la razón de que el fichero no tenga comillas
   > en ninguna parte.
   >
   > **El endpoint es `host.docker.internal`, no `localhost`.** Tu API está en un
   > contenedor; el Collector no. Dentro del contenedor, `localhost` es el contenedor, y el
   > SDK se pasará todo el día fallando al exportar a la nada.

   Esa última línea es la que de verdad enciende el SDK. Es la forma canónica:

   ```
   node --require @opentelemetry/auto-instrumentations-node/register src/server.js
   ```

   > ⚠️ El `--require` tiene que cargarse **antes** que el código de tu aplicación, o la
   > instrumentación no tendrá nada a lo que engancharse. Además solo parchea llamadas
   > `require()` de CommonJS, y por eso esta aplicación es CommonJS y no ESM. Con ESM
   > tendrías spans de HTTP y perderías en silencio la instrumentación de Express, de base
   > de datos y de logging — un modo de fallo especialmente feo, porque parece que funciona.

<!-- /track -->

<!-- track:python -->

1. Añade los paquetes a `track-py/todo-api/requirements.txt`:

   ```
   opentelemetry-distro
   opentelemetry-exporter-otlp
   ```

   Y luego instálalos, igual que en el Módulo 0.2 — más un comando extra:

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     -e PYTHONUSERBASE=/app/.pylibs \
     -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
     python:3.12-slim \
     sh -c 'pip install --user --no-warn-script-location -r requirements.txt \
            && opentelemetry-bootstrap -a install'
   ```

   > ℹ️ `opentelemetry-distro` trae `opentelemetry-bootstrap`, y ese segundo comando es lo
   > que marca la diferencia entre "solo trazas" y las tres señales. Escanea tus
   > dependencias instaladas e instala una librería de instrumentación por cada framework
   > que reconoce — Flask, psycopg, requests, logging. Sáltatelo y te preguntarás dónde
   > están tus spans de base de datos.

2. Crea `track-py/todo-api/.env`. **Es el único fichero que creas, y la aplicación no lo lee
   nunca** — se lo pasas al contenedor con `--env-file`, exactamente igual que una
   plataforma de despliegue le entregaría el entorno a un proceso:

   ```bash
   OTEL_SERVICE_NAME=todo-api
   OTEL_RESOURCE_ATTRIBUTES=service.version=1.0.0,deployment.environment=workshop
   OTEL_EXPORTER_OTLP_ENDPOINT=http://host.docker.internal:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_EXPORTER=otlp
   OTEL_METRICS_EXPORTER=otlp
   OTEL_LOGS_EXPORTER=otlp
   OTEL_METRIC_EXPORT_INTERVAL=10000
   OTEL_METRIC_EXPORT_TIMEOUT=5000
   OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true
   OTEL_PYTHON_LOG_CORRELATION=true
   ```

   > 🚨 **El endpoint es `host.docker.internal`, no `localhost`.** Tu API está en un
   > contenedor; el Collector no. Dentro del contenedor, `localhost` es el contenedor, y el
   > SDK se pasará todo el día fallando al exportar a la nada. Fíjate además en que no hay
   > comillas en ninguna parte del fichero: el `--env-file` de Docker se toma literalmente
   > el resto de cada línea, comillas incluidas.

   A diferencia del track de JavaScript, ninguna variable de entorno puede encender el SDK
   de Python. Lo que cambias es el **comando**, en el paso siguiente, y eso te da la forma
   canónica:

   ```
   opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

<!-- /track -->

3. Reinicia tu API. Ctrl-C en su terminal — y esta vez el comando cambia, así que no le
   des solo a ↑. Dos añadidos al comando del Módulo 0.2:

   <!-- track:js -->

   ```bash
   docker run --rm -it --name workshop-todo-api \
     -p 8081:8081 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     --env-file track-js/todo-api/.env \
     -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
     -e NOTIFY_URL=http://host.docker.internal:8083 \
     -e PORT=8081 \
     node:22-slim node src/server.js
   ```

   Lo único nuevo es `--env-file track-js/todo-api/.env`. La imagen es la misma, el comando
   es el mismo, tu código es el mismo. **Este** es el comando al que le das ↑ de aquí en
   adelante.

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker run --rm -it --name workshop-todo-api \
     -p 8082:8082 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     --env-file track-py/todo-api/.env \
     -e PYTHONUSERBASE=/app/.pylibs \
     -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
     -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
     -e NOTIFY_URL=http://host.docker.internal:8084 \
     python:3.12-slim opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   Dos cosas nuevas: `--env-file track-py/todo-api/.env`, y `opentelemetry-instrument`
   delante de `gunicorn`. La imagen es la misma y tu código no ha cambiado. **Este** es el
   comando al que le das ↑ de aquí en adelante.

   > ⚠️ **No** añadas `--preload` a gunicorn. Con `--preload`, el SDK se inicializa antes
   > del fork y los workers no emiten absolutamente nada. Si tienes que precargar,
   > inicializa el SDK en un hook `post_fork`.

   <!-- /track -->

4. Genera tráfico en otra terminal:

   ```bash
   ./loadgen.sh 2m
   ```

Dos ajustes de ese bloque merecen explicación en vez de copiar y pegar:

- **`OTEL_METRIC_EXPORT_INTERVAL: 10000`.** El valor por defecto es 60 segundos. En un taller, sesenta segundos de nada se leen como "está roto". Diez segundos mantienen la sala cuerda. En producción lo dejarías por defecto — cada exportación se paga.
- **`OTEL_METRIC_EXPORT_TIMEOUT: 5000`.** Tiene que ser menor que el intervalo, y su valor por defecto es 30 segundos. Baja el intervalo sin bajar esto y el SDK de JavaScript se niega a inicializarse: una línea en stderr, y después ninguna telemetría de ningún tipo. Merece la pena ver una vez un error de configuración que lo desactiva todo.
- **`-search.latencyOffset=1s` en VictoriaMetrics** — uno de los flags que escribiste tú en el Módulo 0.1. Por defecto VM oculta los últimos 30 segundos de datos, para que un intervalo escrito a medias nunca produzca una gráfica engañosa. Es el comportamiento correcto y es el equivocado para un aula. Fíjate en que *corrección* y *frescura* están aquí en tensión real.

### Comprueba en la UI de Victoria

**Trazas** — `http://localhost:10428/select/vmui`, pega:

```
_time:15m "resource_attr:service.name":=todo-api
```

Deberías ver: spans llegando en unos 10 segundos. Abre uno de `POST /todos` y despliégalo.

**Métricas** — `http://localhost:8428/vmui`, pega:

```
{__name__=~"http.*"}
```

Deberías ver: un puñado de series HTTP de servidor que no existían en el Módulo 0.

**Logs** — `http://localhost:9428/select/vmui`, pega:

```
service.name:todo-api
```

Deberías ver: la salida de `console.log` / `logging.info` que ya tenía tu app, ahora almacenada y buscable — pero llegando como texto sin estructura en el cuerpo del mensaje.

### Comprueba en Grafana

Grafana → **Explore** → datasource **VictoriaMetrics** → ejecuta:

```
rate(http_server_duration_milliseconds_count[1m])
```

Deberías ver una línea que responde cuando lanzas el loadgen.

### Para pensar

Has cambiado un comando de arranque y unas variables de entorno. No has tocado la aplicación. ¿Para qué tipo de problema convierte eso a la auto-instrumentación en la herramienta correcta — y qué implica para desplegarla en cincuenta servicios el lunes?

### Si te atascas

- **No llega ningún dato después de 60 segundos.** Lee la terminal del Collector — está ahí mismo, en primer plano. Una conexión rechazada significa que la variable de endpoint o de protocolo está mal. Ojo: `http/protobuf` va al puerto **4318**, gRPC al **4317** — confundirlos es el error más común de todos.
- **No llega nada, y la terminal del Collector está en silencio.** Entonces nunca llegó nada hasta él. Por orden: ¿el endpoint es `host.docker.internal` y no `localhost`? ¿`docker exec workshop-todo-api env | grep OTEL` muestra tus variables? ¿`grep -n '\"' track-*/todo-api/.env` encuentra alguna comilla suelta?
- **(JS) Una línea de error al arrancar, después logs normales y ninguna telemetría.** Lee esa línea. `exportIntervalMillis must be greater than or equal to exportTimeoutMillis` significa que falta `OTEL_METRIC_EXPORT_TIMEOUT=5000` en tu `.env`.
- **(Python) Aparecen trazas pero nada más.** No se ejecutó `opentelemetry-bootstrap -a install`, o falta `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true` en tu `.env`. Para empezar de cero limpio: `rm -rf track-py/todo-api/.pylibs`, y repite el paso 1.

```bash
./checkpoint.sh 31
```

## Ejercicio 3.2 — Explora lo que ha llegado [~25 min]

**Deberías poder responder esto al terminar:** *¿qué ha aprendido el SDK sobre tu aplicación sin que nadie se lo cuente?*

### Hazlo

1. Asegúrate de que hay tráfico: `./loadgen.sh 2m`.
2. Recorre cada una de las cuatro UIs de abajo. No las mires por encima — vas a vivir en ellas el resto del día.

### Comprueba en la UI de Victoria

**VictoriaTraces** (`http://localhost:10428/select/vmui`) — busca una traza de creación:

```
_time:15m "resource_attr:service.name":=todo-api name:="POST /todos"
```

Abre una y mira el árbol de spans que has conseguido gratis:

```
POST /todos                          ← span de servidor HTTP, nombrado por la ruta
├── middleware - jsonParser          ← las capas del propio framework web
├── middleware - query
├── request handler - /todos
├── pg-pool.connect                  ← instrumentación de postgres
├── pg.query:INSERT todo
└── POST                             ← HTTP saliente a notify-service
    └── POST /notify                 ← y el span del *otro servicio*, misma traza
```

Nadie ha escrito nada de eso. Fíjate sobre todo en las dos últimas líneas: la traza ya
cruza una frontera de servicio **y** una de lenguaje, porque la instrumentación del
cliente HTTP inyectó una cabecera `traceparent` a la salida y la del servidor HTTP del
otro servicio la leyó a la entrada. En el Módulo 7.4 romperás y repararás exactamente eso.

> 🔑 **Nombres de campo en VictoriaTraces.** Los atributos de resource se guardan con el
> prefijo `resource_attr:` y los de span con `span_attr:`, así que el nombre del campo
> lleva dos puntos y hay que entrecomillarlo:
> `"resource_attr:service.name":=todo-api`, `"span_attr:http.route":=/todos`.
> `:=` es coincidencia exacta. Los campos sin prefijo son `name`, `kind`, `duration`,
> `status_code`, `trace_id`, `span_id` y `parent_span_id`.

Haz clic en un span suelto y lee sus atributos. Busca estos tres, que son **convenciones semánticas** — nombres estándar que usa cualquier servicio instrumentado con OTel del mundo:

- `span_attr:http.method`
- `span_attr:http.target`
- `span_attr:db.system`

> ⚠️ **Compara notas con el otro track ahora — son los cinco minutos más útiles del módulo.**
>
> Ejecuta `count by (service_name, __name__) ({__name__=~"http.*"})` en vmui. Verás que:
>
> - El `todo-api` del **track de Python** emite `http_server_duration_milliseconds_{count,sum,bucket}`.
> - El `todo-api` del **track de JavaScript** **no emite ninguna métrica HTTP.** No es que
>   se llamen distinto: no hay ninguna.
>
> No es un fallo del taller. `@opentelemetry/instrumentation-http` no registra métricas de
> duración de servidor, y `opentelemetry-instrumentation-flask` sí. Los dos tracks
> obtienen *trazas* idénticas y *logs* idénticos; donde divergen es en la señal de métricas.
>
> Piensa en lo que eso implica. "Hemos auto-instrumentado todo, así que tenemos métricas"
> es una frase que puede ser falsa en un lenguaje y verdadera en otro, dentro de la misma
> flota, sin que nadie se entere hasta que un dashboard aparece vacío en mitad de un
> incidente. Lo arreglarás para el track de JS en el Módulo 8.1, generando las métricas a
> partir de los spans — que es la respuesta general a toda esta familia de problemas.

**VictoriaMetrics** (`http://localhost:8428/vmui`) — mira qué ha aparecido:

```
{__name__=~"http.*"}
```

y luego

```
rate(http_server_duration_milliseconds_count[1m])
```

**VictoriaLogs** (`http://localhost:9428/select/vmui`):

```
service.name:todo-api
```

Fíjate en la forma de lo que ha llegado. El mensaje es un bloque de texto en `_msg`. No
hay campo `list`, no hay campo `todo_id`, no hay nada por lo que filtrar o agregar.
Arreglar eso es el Módulo 6.

> 🔑 **Nombres de campo en VictoriaLogs**, para contrastar con el recuadro de trazas de
> arriba: el mensaje es `_msg`, el nivel es `severity_text`, y los atributos de resource
> son campos normales — `service.name:todo-api`, sin prefijo y sin comillas. El mismo
> lenguaje de consulta, dos esquemas distintos. Mejor darse cuenta ahora que en mitad de
> un incidente.

### Comprueba en Grafana

Grafana es la novena cosa que arrancas hoy, y la última. La dejamos para ahora porque
hasta ahora no había nada que mirar:

```bash
docker run -d --name workshop-grafana \
  -p 3000:3000 \
  --add-host=host.docker.internal:host-gateway \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -e GF_INSTALL_PLUGINS=victoriametrics-metrics-datasource,victoriametrics-logs-datasource \
  -e GF_FEATURE_TOGGLES_ENABLE=correlations,traceToMetrics,traceqlEditor \
  -v "$PWD/grafana/provisioning:/etc/grafana/provisioning:ro" \
  grafana/grafana:12.3.1
```

El primer arranque tarda un minuto — instala los dos plugins de VictoriaMetrics. Míralo con
`docker logs -f workshop-grafana`, y después abre <http://localhost:3000> y entra con
`admin` / `admin`.

**Connections → Data sources.** Hay cuatro pre-provisionados desde `grafana/provisioning`:

| Nombre | Tipo | Apunta a |
|---|---|---|
| VictoriaMetrics | `victoriametrics-metrics-datasource` | `host.docker.internal:8428` |
| VictoriaLogs | `victoriametrics-logs-datasource` | `host.docker.internal:9428` |
| VictoriaTraces (Jaeger) | `jaeger` (core, sin plugin) | `host.docker.internal:10428/select/jaeger` |
| VictoriaTraces (Tempo) | `tempo` (core, sin plugin) | `host.docker.internal:10428/select/tempo` |

Entra en cada uno y pulsa **Save & test**. Los cuatro tienen que ponerse verdes.

Fíjate en el host de esa última columna, y fíjate en que *no* es lo que escribes tú en tu
propio navegador. Grafana está en un contenedor y los tres backends no, así que desde el
punto de vista de Grafana viven en `host.docker.internal` — la misma razón por la que lo
dice el `.env` de tu API. Tú llegas a esos mismísimos procesos en `localhost`, porque tú no
estás en un contenedor.

Después:

1. **Explore → VictoriaMetrics** — la misma consulta de antes, mejor gráfica.
2. **Explore → VictoriaTraces (Jaeger)** — busca por servicio `todo-api`. Obtienes la vista de cascada de siempre. Funciona porque VictoriaTraces implementa la API de consulta de Jaeger; no hay ningún plugin de por medio.
3. **Explore → VictoriaLogs** — la misma consulta de antes, con un constructor visual de consultas junto a la caja de LogsQL en crudo.
4. **Demo (mira, no construyas):** el datasource **VictoriaTraces (Tempo)** apunta a `/select/tempo` y desbloquea la app **Traces Drilldown** de Grafana — paneles RED generados a partir de tus spans sin ningún dashboard que construir. Es experimental; trátalo como una vista previa, no como algo de lo que depender.

### Para pensar

Vuelve a mirar el árbol de la traza. ¿Cuáles de esos spans vienen de una librería que mantiene el proyecto OpenTelemetry, y cuáles de tu lógica de negocio? Cuéntalos.

### Si te atascas

- **Las trazas solo muestran el span HTTP, sin hijos.** Las instrumentaciones de base de datos y del framework se cargan con el SDK. Comprueba qué recibió de verdad el contenedor: `docker exec workshop-todo-api env | grep -E 'NODE_OPTIONS|OTEL'`. En el track de Python, comprueba que se ejecutó `opentelemetry-bootstrap -a install`.
- **El datasource Jaeger de Grafana no devuelve nada pero vmui sí tiene datos.** Revisa la URL del datasource — tiene que ser `http://host.docker.internal:10428/select/jaeger`, con `/select/jaeger` al final.
- **Grafana no llega a ningún datasource.** Lo arrancaste sin `--add-host=host.docker.internal:host-gateway`. `docker rm -f workshop-grafana` y vuelve a ejecutar el comando.

## Ejercicio 3.3 — Encuentra el techo [~15 min]

**Deberías poder responder esto al terminar:** *¿qué es estructuralmente incapaz de contarte la auto-instrumentación?*

### Hazlo

Aquí tienes tres preguntas que un product owner o alguien de guardia harían de verdad. Intenta responder cada una con la telemetría que tienes ahora. **Se espera que falles en las tres.** Dedica cinco minutos a intentarlo igualmente — el fracaso es la lección.

1. **"¿Cuántos todos se crearon hoy, desglosados por lista?"**
   Prueba: vmui de VictoriaMetrics, busca cualquier cosa que se parezca a un evento de negocio.

2. **"¿Qué usuario se topó con ese error?"**
   Prueba: VictoriaLogs, `service.name:todo-api AND _time:1h`, e intenta filtrar por usuario.

3. **"¿Por qué fue lenta *esta* petición en concreto?"**
   Prueba: encuentra una traza lenta en VictoriaTraces. Verás un span `pg` lento. Ahora di *qué decisión de la aplicación* produjo esa consulta.

### Comprueba en la UI de Victoria

Para la pregunta 1, esto es lo más cerca que puedes llegar — y solo en el track de Python,
según la nota de arriba:

```
sum(rate(http_server_duration_milliseconds_count{http_method="POST", http_target="/todos"}[5m]))
```

Eso es una tasa de peticiones HTTP. Cuenta creaciones fallidas, reintentos y health checks
de tu generador de carga. *No* es el número de todos creados, y ninguna astucia en la
consulta lo va a convertir en eso. En el track de JavaScript ni siquiera puedes llegar
hasta aquí.

### Comprueba en Grafana

Intenta construir un panel titulado "Todos creados por lista". Llega tan lejos como puedas. Fíjate exactamente en dónde te paras.

### Para pensar

Vuelve a las tres preguntas que clasificaste en el Módulo 1.2 — cuál era de logging, cuál de tracing, cuál de métricas. ¿Acertaste?

### Si te atascas

Se supone que estás atascado. Ese es el ejercicio. 🙂

## La conclusión

> **La auto-instrumentación te da la fontanería, no el significado.**

Sabe de HTTP, de Express o Flask y de Postgres, porque son estándar, populares y alguien escribió una librería de instrumentación para ellos. No sabe nada de todos, listas, urgencias ni usuarios — y absolutamente nada de esa caché en proceso, porque ahí no intervino ninguna librería. Nadie ha visto nunca tu dominio.

Todo lo que viene ahora consiste en añadir el significado:

- El Módulo 5 añade **métricas de negocio** — los números que pide tu product owner
- El Módulo 6 añade **logs estructurados** — los campos por los que filtras un incidente
- El Módulo 7 añade **spans de dominio** — la "qué decisión de la aplicación" que necesitaba la pregunta 3

## Lo Que Hemos Conseguido

- Las tres señales fluyendo, sin cambiar código de aplicación
- Una traza que atraviesa dos servicios y dos lenguajes, obtenida gratis
- Familiaridad con las cuatro UIs
- Una comprensión precisa y práctica de dónde se acaba el camino fácil

## Siguientes Pasos

Antes de añadir significado, desmitifiquemos la caja del medio. Continúa con el [Módulo 4: El Collector, y Dónde Acaban los Datos](./04-the-collector.es.md).
