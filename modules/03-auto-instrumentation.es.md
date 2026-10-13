# Módulo 3: El Camino Fácil — Auto-instrumentación Sin Código

**Duración:** ~60 min

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

## Ejercicio 3.1 — Enciéndelo [~20 min]

**Deberías poder responder esto al terminar:** *¿qué fichero has tenido que editar para tener telemetría, y cuál no?*

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

<!-- /track -->

<!-- track:python -->

1. Añade los paquetes a `track-py/todo-api/requirements.txt`:

   ```
   opentelemetry-distro
   opentelemetry-exporter-otlp
   ```

   El contenedor de tu API ejecuta `pip install` cada vez que arranca, así que no hay nada
   que instalar a mano.

   > ℹ️ `opentelemetry-distro` trae `opentelemetry-bootstrap`, que el contenedor también
   > ejecuta al arrancar en cuanto existe. Es lo que marca la diferencia entre "solo
   > trazas" y las tres señales: escanea tus dependencias instaladas e instala una librería
   > de instrumentación por cada framework que reconoce — Flask, psycopg, requests, logging.
   > Sin él te preguntarías dónde están tus spans de base de datos.

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
   OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true
   OTEL_PYTHON_LOG_CORRELATION=true
   ```

   > 🚨 **El endpoint es `otel-collector`, no `localhost`.** Es el nombre de servicio del
   > Collector en `docker-compose.yml`. Dentro del contenedor de tu API, `localhost` es el
   > contenedor, y el SDK se pasaría todo el día fallando al exportar a la nada.

   A diferencia del track de JavaScript, ninguna variable de entorno puede encender el SDK
   de Python. Lo que cambias es el **comando**. En `docker-compose.yml`, busca el servicio
   `todo-api-py` y pon `opentelemetry-instrument` delante de `gunicorn`:

   ```yaml
       command: opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   Esa es la forma canónica. No añadas el `--reload` de gunicorn: el contenedor ya reinicia
   todo el comando cuando cambia un fichero, y bajo `opentelemetry-instrument` el recargador
   propio de gunicorn hace que los logs dejen de exportarse sin decir nada.

   > ⚠️ Tampoco añadas `--preload` a gunicorn. Con `--preload`, el SDK se inicializa antes
   > del fork y los workers no emiten absolutamente nada. Si tienes que precargar,
   > inicializa el SDK en un hook `post_fork`.

<!-- /track -->

3. Recrea el contenedor de tu API, para que coja el entorno nuevo e instale las
   dependencias nuevas. Esta vez no basta con guardar un fichero — lo que ha cambiado es la
   configuración del contenedor, no tu código:

   <!-- track:js -->

   ```bash
   docker compose up -d --force-recreate todo-api-js
   ```

   Después abre una segunda terminal y sigue en ella el log de tu API. Déjala abierta el
   resto del día — los guiones la llaman la *terminal del log*:

   ```bash
   docker compose logs -f todo-api-js
   ```

   Espera a ver `OpenTelemetry automatic instrumentation started
   successfully`. La imagen es la misma, el comando es el mismo, tu código es el mismo. Lo
   único nuevo es el entorno.

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker compose up -d --force-recreate todo-api-py
   ```

   Después abre una segunda terminal y sigue en ella el log de tu API. Déjala abierta el
   resto del día — los guiones la llaman la *terminal del log*:

   ```bash
   docker compose logs -f todo-api-py
   ```

   Muestra cómo se instalan los paquetes nuevos y luego cómo arranca
   gunicorn. La imagen es la misma y tu código no ha cambiado. Lo nuevo es el entorno y una
   palabra delante del comando.

   <!-- /track -->

4. Genera tráfico en otra terminal:

   ```bash
   ./loadgen.sh 2m
   ```

Dos ajustes de ese bloque merecen explicación en vez de copiar y pegar:

- **`OTEL_METRIC_EXPORT_INTERVAL: 10000`.** El valor por defecto es 60 segundos. En un taller, sesenta segundos de nada se leen como "está roto". Diez segundos mantienen la sala cuerda. En producción lo dejarías por defecto — cada exportación se paga.
- **`OTEL_METRIC_EXPORT_TIMEOUT: 5000`.** Tiene que ser menor que el intervalo, y su valor por defecto es 30 segundos. Baja el intervalo sin bajar esto y el SDK de JavaScript se niega a inicializarse: una línea en stderr, y después ninguna telemetría de ningún tipo. Merece la pena ver una vez un error de configuración que lo desactiva todo.
- **`-search.latencyOffset=1s` en VictoriaMetrics** — puesto en `docker-compose.yml`. Por defecto VM oculta los últimos 30 segundos de datos, para que un intervalo escrito a medias nunca produzca una gráfica engañosa. Es el comportamiento correcto y es el equivocado para un aula. Fíjate en que *corrección* y *frescura* están aquí en tensión real.

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

- **No llega ningún dato después de 60 segundos.** Lee el log del Collector — `docker compose logs -f otel-collector`. Una conexión rechazada significa que la variable de endpoint o de protocolo está mal. Ojo: `http/protobuf` va al puerto **4318**, gRPC al **4317** — confundirlos es el error más común de todos.
- **No llega nada, y el log del Collector está en silencio.** Entonces nunca llegó nada hasta él. Por orden: ¿el endpoint es `otel-collector` y no `localhost`? ¿Recreaste el contenedor después de crear el `.env`? ¿`docker compose exec todo-api-js env | grep OTEL` (o `todo-api-py`) muestra tus variables?
- **(JS) Una línea de error al arrancar, después logs normales y ninguna telemetría.** Lee esa línea. `exportIntervalMillis must be greater than or equal to exportTimeoutMillis` significa que falta `OTEL_METRIC_EXPORT_TIMEOUT=5000` en tu `.env`.
- **(Python) Aparecen trazas pero nada más.** No se ejecutó `opentelemetry-bootstrap`, o falta `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true` en tu `.env`. Lee el principio de `docker compose logs todo-api-py`: el paso de bootstrap se ejecuta ahí, antes de que arranque gunicorn.

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

Grafana es lo último que arrancas hoy. Lleva en `docker-compose.yml` desde el Módulo 0,
detrás de su propio profile; la dejamos para ahora porque hasta ahora no había nada que
mirar. Basta con nombrar el servicio para arrancarlo:

```bash
docker compose up -d grafana
```

El primer arranque tarda un minuto — instala los dos plugins de VictoriaMetrics. Míralo con
`docker compose logs -f grafana`, y después abre <http://localhost:3000> y entra con
`admin` / `admin`.

**Connections → Data sources.** Hay cuatro pre-provisionados desde `grafana/provisioning`:

| Nombre | Tipo | Apunta a |
|---|---|---|
| VictoriaMetrics | `victoriametrics-metrics-datasource` | `victoriametrics:8428` |
| VictoriaLogs | `victoriametrics-logs-datasource` | `victorialogs:9428` |
| VictoriaTraces (Jaeger) | `jaeger` (core, sin plugin) | `victoriatraces:10428/select/jaeger` |
| VictoriaTraces (Tempo) | `tempo` (core, sin plugin) | `victoriatraces:10428/select/tempo` |

Entra en cada uno y pulsa **Save & test**. Los cuatro tienen que ponerse verdes.

Fíjate en el host de esa última columna, y fíjate en que *no* es lo que escribes tú en tu
propio navegador. Grafana está en el mismo proyecto de Compose que los tres backends, así
que llega a ellos por nombre de servicio, igual que el Collector, e igual que tu API llega
a `otel-collector`. Tú llegas a esos mismísimos contenedores en `localhost`, a través de
los puertos que Compose publicó en tu portátil, porque tú no estás en un contenedor.

Después:

1. **Explore → VictoriaMetrics** — la misma consulta de antes, mejor gráfica.
2. **Explore → VictoriaTraces (Jaeger)** — busca por servicio `todo-api`. Obtienes la vista de cascada de siempre. Funciona porque VictoriaTraces implementa la API de consulta de Jaeger; no hay ningún plugin de por medio.
3. **Explore → VictoriaLogs** — la misma consulta de antes, con un constructor visual de consultas junto a la caja de LogsQL en crudo.
4. **Demo (mira, no construyas):** el datasource **VictoriaTraces (Tempo)** apunta a `/select/tempo` y desbloquea la app **Traces Drilldown** de Grafana — paneles RED generados a partir de tus spans sin ningún dashboard que construir. Es experimental; trátalo como una vista previa, no como algo de lo que depender.

### Para pensar

Vuelve a mirar el árbol de la traza. ¿Cuáles de esos spans vienen de una librería que mantiene el proyecto OpenTelemetry, y cuáles de tu lógica de negocio? Cuéntalos.

### Si te atascas

- **Las trazas solo muestran el span HTTP, sin hijos.** Las instrumentaciones de base de datos y del framework se cargan con el SDK. Comprueba qué recibió de verdad el contenedor: `docker compose exec todo-api-js env | grep -E 'NODE_OPTIONS|OTEL'`. En el track de Python, busca el paso de bootstrap al principio de `docker compose logs todo-api-py`.
- **El datasource Jaeger de Grafana no devuelve nada pero vmui sí tiene datos.** Revisa la URL del datasource — tiene que ser `http://victoriatraces:10428/select/jaeger`, con `/select/jaeger` al final.
- **Grafana no llega a ningún datasource.** Comprueba que los tres backends están arriba (`docker compose ps`) y que arrancaste Grafana a través de Compose, no con un `docker run` aparte — solo los contenedores del proyecto de Compose pueden resolver `victoriametrics`, `victorialogs` y `victoriatraces`.

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

Un todo con el título vacío devuelve un 500. Tres preguntas sobre él: *¿qué función lanzó el error?*, *¿se tocó la base de datos antes de fallar?*, *¿cuántos usuarios se lo han encontrado hoy?* ¿Cuál es un problema de logging, cuál de tracing y cuál de métricas — y cuáles puedes responder ahora mismo, solo con auto-instrumentación?

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
