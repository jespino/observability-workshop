# Módulo 0: Preparación y Comprobación del Entorno

**Duración:** ~45 min · **Todo aquí es igual para los dos tracks**

El objetivo de este módulo es aburrido a propósito: que todos los portátiles de la sala estén en verde antes de empezar a enseñar nada. 🟢

**No hay script de preparación.** Vas a descargar cuatro binarios y arrancar ocho procesos
a mano, uno a uno, comprobando cada uno antes de arrancar el siguiente. Eso tarda más que
ejecutar un script de setup, y ahí está el sentido: el resto del día, cuando algo deje de
funcionar, sabrás exactamente qué proceso era, qué comando lo arrancó y dónde fue su log.
Un script de preparación te habría quitado todo eso en los primeros diez minutos.

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Haber descargado y arrancado todo el stack tú mismo — cuatro procesos de telemetría y cuatro contenedores
- Saber en qué puerto escucha cada pieza, y qué fichero de log leer cuando una se caiga
- Saber en qué tres UIs vas a vivir el resto del día, y tenerlas en marcadores
- Haber elegido tu track
- Haber visto, con tus propios ojos, que la aplicación ahora mismo no emite **nada**

## Prerrequisitos

- **Docker** instalado y funcionando (`docker ps`)
- `curl`, `tar` y una shell Unix. En Windows, usa WSL2.
- Unos 4GB de RAM libre y 2GB de disco
- **No necesitas Node.js ni Python instalados.** La aplicación corre en un contenedor
  `node:22-slim` o `python:3.12-slim` de serie, con tu código fuente montado dentro.
  Nada de este repositorio se construye como imagen.

> 📦 **Preparación previa, si puedes.** Las descargas del Ejercicio 0.1 son unos 450 MB (la
> mayoría, el Collector), y el Ejercicio 0.2 se baja cuatro imágenes. Hacer las dos cosas
> antes de venir convierte un módulo de 45 minutos en uno de 15, y el wifi del sitio no va
> a ser tu amigo:
>
> ```bash
> docker pull postgres:16-alpine
> docker pull node:22-slim          # o python:3.12-slim, para el track de Python
> docker pull python:3.12-slim      # ...este lo necesitas igual, para notify-service
> docker pull nginx:alpine
> docker pull grafana/grafana:12.3.1
> ```

## Ejercicio 0.1 — Levanta el stack de telemetría, proceso a proceso [~20 min]

**Deberías poder responder esto al terminar:** *¿qué procesos están corriendo en tu portátil, qué puertos ocupan, y dónde escribe su log cada uno?*

### Hazlo

1. Clona el repo y entra en el directorio del taller:

   ```bash
   git clone https://github.com/jespino/observability-workshop
   cd observability-workshop/workshop
   ```

   > 📁 Todo lo que ejecutas hoy vive en `workshop/`. Los guiones que estás leyendo viven un
   > nivel más arriba, en `modules/`. **Quédate en `workshop/` el resto del día** — todos
   > los comandos de todos los módulos dan por hecho que estás ahí, y varios usan `$PWD`.

2. Crea los tres directorios en los que va a escribir el stack:

   ```bash
   mkdir -p bin data logs
   ```

   `bin/` para los binarios que vas a descargar, `data/` para el almacenamiento de cada
   backend, `logs/` para su salida. Los tres están en el `.gitignore`.

3. Averigua qué tienes que descargar para tu máquina:

   ```bash
   OS=$(uname -s | tr '[:upper:]' '[:lower:]')
   ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
   echo "$OS/$ARCH"
   ```

   Deberías ver `linux/amd64`, `darwin/arm64` o algo parecido. No cierres esta terminal —
   los cuatro comandos siguientes usan esas dos variables.

4. Descarga los cuatro binarios. Un comando cada uno, y cada uno deja un solo ejecutable en
   `bin/`:

   ```bash
   # VictoriaMetrics — almacenamiento de métricas
   curl -sSL "https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v1.151.0/victoria-metrics-${OS}-${ARCH}-v1.151.0.tar.gz" \
     | tar -xzf - -C bin && mv bin/victoria-metrics-prod bin/victoria-metrics
   ```

   ```bash
   # VictoriaLogs — almacenamiento de logs
   curl -sSL "https://github.com/VictoriaMetrics/VictoriaLogs/releases/download/v1.52.0/victoria-logs-${OS}-${ARCH}-v1.52.0.tar.gz" \
     | tar -xzf - -C bin && mv bin/victoria-logs-prod bin/victoria-logs
   ```

   ```bash
   # VictoriaTraces — almacenamiento de trazas
   curl -sSL "https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.11.0/victoria-traces-${OS}-${ARCH}-v0.11.0.tar.gz" \
     | tar -xzf - -C bin && mv bin/victoria-traces-prod bin/victoria-traces
   ```

   ```bash
   # El OpenTelemetry Collector (distribución contrib — el grande, ~390MB)
   curl -sSL "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.159.0/otelcol-contrib_0.159.0_${OS}_${ARCH}.tar.gz" \
     | tar -xzf - -C bin otelcol-contrib
   ```

   Comprueba que tienes cuatro:

   ```bash
   ls -l bin/
   ./bin/victoria-metrics --version
   ./bin/otelcol-contrib --version
   ```

5. Arranca **VictoriaMetrics**. Lee los flags antes de ejecutarlo — tres de ellos vuelven a
   aparecer más tarde:

   ```bash
   nohup ./bin/victoria-metrics \
     -storageDataPath=./data/victoria-metrics \
     -httpListenAddr=:8428 \
     -retentionPeriod=1 \
     -selfScrapeInterval=10s \
     -search.latencyOffset=1s \
     -opentelemetry.usePrometheusNaming \
     > logs/victoria-metrics.log 2>&1 &
   ```

   | Flag | Por qué |
   |---|---|
   | `-retentionPeriod=1` | un mes, más que suficiente para un día |
   | `-selfScrapeInterval=10s` | VM hace scrape de sus propias métricas. Los módulos 4.2 y 5.4 las consultan. |
   | `-search.latencyOffset=1s` | por defecto VM esconde los últimos 30 segundos para que un intervalo escrito a medias no dibuje una gráfica engañosa. Correcto en producción, desesperante en un aula. |
   | `-opentelemetry.usePrometheusNaming` | guarda `todos.created` como `todos_created_total` y `service.name` como la label `service_name`. Sin él, tus consultas necesitan comillas. El módulo 5.2 se aprovecha de esto. |

   Ahora demuestra que responde:

   ```bash
   curl -s localhost:8428/health
   ```

   **No sigas hasta que eso imprima `OK`.** Si no lo hace, lee `logs/victoria-metrics.log`.

6. Arranca **VictoriaLogs**, y compruébalo igual:

   ```bash
   nohup ./bin/victoria-logs \
     -storageDataPath=./data/victoria-logs \
     -httpListenAddr=:9428 \
     -retentionPeriod=7d \
     > logs/victoria-logs.log 2>&1 &

   curl -s localhost:9428/health
   ```

7. Arranca **VictoriaTraces**:

   ```bash
   nohup ./bin/victoria-traces \
     -storageDataPath=./data/victoria-traces \
     -httpListenAddr=:10428 \
     -retentionPeriod=3d \
     > logs/victoria-traces.log 2>&1 &

   curl -s localhost:10428/health
   ```

   Esos tres los arrancas una vez y te olvidas. `jobs -l` los lista; `kill %1` para uno.

8. Arranca el **OpenTelemetry Collector** — y dale una **terminal para él solo**, en primer
   plano:

   ```bash
   # en una segunda terminal
   cd observability-workshop/workshop
   ./bin/otelcol-contrib --config=collector/config.yaml
   ```

   Tiene su propia terminal porque es el único proceso que vas a reiniciar una y otra vez
   hoy: cada edición de `collector/config.yaml` en los módulos 4 y 8 necesita un reinicio,
   y el reinicio es **Ctrl-C, y luego ↑ Enter**. Ver su log pasar cuando llega telemetría
   ya justifica la terminal por sí solo.

   En una tercera terminal, comprueba sus dos endpoints:

   ```bash
   curl -s localhost:13133/          # extensión health_check
   curl -s localhost:8888/metrics | head -5   # las métricas propias del Collector
   ```

9. Ese es el stack. Cuatro procesos, nativos, en tu portátil:

   | Proceso | Puerto | Log | Qué es |
   |---|---|---|---|
   | `victoria-metrics` | 8428 | `logs/victoria-metrics.log` | almacenamiento de métricas |
   | `victoria-logs` | 9428 | `logs/victoria-logs.log` | almacenamiento de logs |
   | `victoria-traces` | 10428 | `logs/victoria-traces.log` | almacenamiento de trazas |
   | `otelcol-contrib` | 4317 / 4318, 8888, 13133 | su propia terminal | el OpenTelemetry Collector |

   > ⚠️ `nohup … &` mantiene vivos los tres procesos de almacenamiento aunque cierres su
   > terminal, pero siguen siendo procesos normales sin ningún supervisor. Si uno
   > desaparece, nada lo va a reiniciar — lo harás tú. Es un modelo bastante fiel de
   > producción, menos el busca.

### Comprueba en la UI de Victoria

Abre ahora las tres UIs y **guárdalas en marcadores** — las vas a usar constantemente:

| Qué | URL |
|---|---|
| VictoriaMetrics (métricas) | `http://localhost:8428/vmui` |
| VictoriaLogs (logs) | `http://localhost:9428/select/vmui` |
| VictoriaTraces (trazas) | `http://localhost:10428/select/vmui` |

En la **vmui de VictoriaMetrics**, pega esto en la caja de consulta:

```
{__name__=~"todo.*"}
```

Deberías ver: **nada. Ni una serie.** Eso no es un entorno roto — es el sentido de todo el
taller. Quédate un segundo mirándolo. 🫥

Ahora pega esta otra:

```
{__name__=~"otelcol.*"}
```

Esa **sí** devuelve series, y eso es algo muy útil de saber en un día en el que nada más lo
hace. El Collector hace scrape de sus propias métricas en `:8888` con un receiver
`prometheus/self` y las manda por su propio pipeline de métricas. Así que el camino
Collector → VictoriaMetrics ya funciona, y lo acabas de demostrar de punta a punta sin que
la aplicación exista todavía.

### Comprueba en Grafana

Todavía no. Grafana es un contenedor más, y lo arrancamos en el Módulo 3.2, cuando por fin
haya algo que mirar. Una pieza móvil menos por hoy.

### Para pensar

El Collector está entre la app y los tres procesos de almacenamiento. Antes de que
expliquemos por qué, adivina: ¿qué se rompería si la app hablase directamente con los tres?

### Si te atascas

- **Falla una descarga.** Comprueba que llegas a github.com. Si estás detrás de un proxy,
  exporta `HTTPS_PROXY` antes del `curl`. Si `mv` se queja de que el origen no existe, ha
  cambiado la estructura del tarball — mírala con `tar -tzf` y ajusta el nombre.
- **`curl localhost:8428/health` se queda colgado o rechaza la conexión.** Lee
  `logs/victoria-metrics.log`. La causa habitual es otro proceso ya en ese puerto:
  `ss -ltnp | grep 8428`.
- **El Collector termina inmediatamente.** Imprime la línea problemática de
  `collector/config.yaml` antes de morir, en la terminal donde lo arrancaste. Es YAML, así
  que será la indentación.
- **Puertos ya ocupados.** Algo en tu máquina tiene cogido el 5432, el 8428 o el 3000.
  Párralo, o cambia el puerto en el comando de arriba *y* en todos los sitios donde los
  guiones lo mencionan — lo cual es una buena razón para simplemente parar lo otro.

## Ejercicio 0.2 — Arranca la aplicación y elige tu track [~25 min]

**Deberías poder responder esto al terminar:** *¿qué hace la app, en qué puerto está la API de tu track, y cómo llega un proceso dentro de un contenedor a otro que está fuera?*

### Hazlo

1. Elige tu track. Los dos construyen lo mismo contra el mismo backend:

   | Track | Puerto | Directorio que vas a editar | Corre en |
   |---|---|---|---|
   | Node.js / Express | 8081 | `track-js/todo-api/` | `node:22-slim` |
   | Python / Flask | 8082 | `track-py/todo-api/` | `python:3.12-slim` |

   Elige el lenguaje en el que estés *cómodo*. Hoy la fricción tiene que ser
   OpenTelemetry, no la sintaxis. (Elige el otro solo si te sientes valiente y quieres el
   reto extra.)

   **No** necesitas ninguno de los dos lenguajes instalado. Tu código se monta dentro de
   una imagen de serie de Docker Hub, así que el contenedor aporta el runtime y tu editor
   edita ficheros de verdad en tu portátil.

   > 🔌 **Lee esto una vez y el resto del módulo tiene sentido.** Todos los contenedores de
   > abajo llevan dos flags que importan:
   >
   > - `-p 8081:8081` publica el puerto del contenedor en tu portátil, para que tu
   >   navegador y `curl` lleguen a él en `localhost`.
   > - `--add-host=host.docker.internal:host-gateway` le da al contenedor un nombre para
   >   *tu portátil*, para que pueda llegar a las cosas que **no** están en un contenedor:
   >   Postgres, y el endpoint OTLP del Collector en el 4318.
   >
   > Dentro de un contenedor, `localhost` es el contenedor. Este es el error más común de
   > todo el día, y se presenta exactamente como "la telemetría no llega, sin decir nada".
   > Es también la razón de que `collector/config.yaml` escuche en `0.0.0.0` y no en
   > `127.0.0.1`: un receiver atado al loopback rechaza las conexiones de los contenedores.

2. Arranca **Postgres** — la única dependencia de la aplicación:

   ```bash
   docker run -d --name workshop-postgres \
     -p 5432:5432 \
     -e POSTGRES_USER=todo -e POSTGRES_PASSWORD=todo -e POSTGRES_DB=todo \
     -v "$PWD/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro" \
     postgres:16-alpine
   ```

   El `init.sql` que montas crea el esquema y siembra unos cientos de todos. Espérate a que
   esté listo:

   ```bash
   docker exec workshop-postgres pg_isready -U todo -d todo
   ```

3. Arranca **`notify-service`** — el servicio de más abajo al que llama tu API. Ya viene
   instrumentado, porque no es el ejercicio; existe para que una traza de tu API tenga
   adónde ir.

   Está escrito en **el otro lenguaje de tu track**, a propósito, para que la propagación
   de contexto entre lenguajes del Módulo 7.4 funcione en cualquiera de los dos.

   <!-- track:js -->

   ```bash
   docker run -d --name workshop-notify \
     -p 8083:8083 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/services/notify-service-py:/app" -w /app \
     --env-file services/notify-service-py/otel.env \
     python:3.12-slim \
     sh -c 'pip install --user -q --no-warn-script-location -r requirements.txt \
            && opentelemetry-bootstrap -a install > /dev/null \
            && exec opentelemetry-instrument gunicorn -w 2 -b 0.0.0.0:8083 app:app'
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker run -d --name workshop-notify \
     -p 8084:8084 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/services/notify-service-js:/app" -w /app \
     --env-file services/notify-service-js/otel.env \
     node:22-slim \
     sh -c 'npm install --silent --no-fund --no-audit \
            && exec node --require @opentelemetry/auto-instrumentations-node/register server.js'
   ```

   Ese `--require` es lo único con forma de OpenTelemetry en todo el comando, y es lo que
   enciende el SDK. Escribirás el equivalente para tu propia API en el Módulo 3.1.

   <!-- /track -->

   Instala sus propias dependencias al arrancar por primera vez, así que dale un minuto:

   <!-- track:js -->

   ```bash
   curl -s localhost:8083/healthz     # {"ok":true} cuando esté arriba
   docker logs -f workshop-notify     # Ctrl-C cuando veas que está escuchando
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   curl -s localhost:8084/healthz     # {"ok":true} cuando esté arriba
   docker logs -f workshop-notify     # Ctrl-C cuando veas que está escuchando
   ```

   <!-- /track -->

   > 📄 **Abre `services/notify-service-*/otel.env` y léelo.** Ese fichero es toda la
   > instrumentación de este servicio: diez variables de entorno y ni una línea de código.
   > En el Módulo 3.1 escribirás un fichero exactamente de esa forma para tu propia API.
   > Fíjate en el endpoint — `http://host.docker.internal:4318`, porque el Collector no
   > está en un contenedor.

4. Instala las dependencias de **tu** aplicación. La misma idea, salvo que los paquetes
   caen en tu directorio de código en vez de dentro de una imagen, así que sobreviven al
   contenedor:

   <!-- track:js -->

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     node:22-slim npm install
   ```

   Ahora tienes `track-js/todo-api/node_modules/`, tuyo, en tu portátil.
   `-u "$(id -u):$(id -g)"` es lo que hace que sea tuyo y no de root; `-e HOME=/tmp` le da
   a npm un sitio donde escribir su caché.

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker run --rm \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     -e PYTHONUSERBASE=/app/.pylibs \
     python:3.12-slim \
     pip install --user --no-warn-script-location -r requirements.txt
   ```

   Ahora tienes `track-py/todo-api/.pylibs/`, tuyo, en tu portátil — `PYTHONUSERBASE` es lo
   que redirigió el `pip install --user` ahí en vez de dentro de la imagen.
   `-u "$(id -u):$(id -g)"` hace que los ficheros sean tuyos y no de root; `-e HOME=/tmp`
   le da a pip un sitio donde escribir.

   <!-- /track -->

5. En una **cuarta terminal**, arranca tu API. **En primer plano, y déjala ahí** — a partir
   del Módulo 3 la vas a reiniciar constantemente, y verla arrancar es la mitad del
   feedback:

   <!-- track:js -->

   ```bash
   cd observability-workshop/workshop

   docker run --rm -it --name workshop-todo-api \
     -p 8081:8081 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-js/todo-api:/app" -w /app \
     -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
     -e NOTIFY_URL=http://host.docker.internal:8083 \
     -e PORT=8081 \
     node:22-slim node src/server.js
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   cd observability-workshop/workshop

   docker run --rm -it --name workshop-todo-api \
     -p 8082:8082 \
     --add-host=host.docker.internal:host-gateway \
     -u "$(id -u):$(id -g)" -e HOME=/tmp \
     -v "$PWD/track-py/todo-api:/app" -w /app \
     -e PYTHONUSERBASE=/app/.pylibs \
     -e PATH=/app/.pylibs/bin:/usr/local/bin:/usr/bin:/bin \
     -e DATABASE_URL=postgres://todo:todo@host.docker.internal:5432/todo \
     -e NOTIFY_URL=http://host.docker.internal:8084 \
     python:3.12-slim gunicorn -w 2 -b 0.0.0.0:8082 app:app
   ```

   `PYTHONUSERBASE` y `PATH` están ahí por lo mismo que en el paso 4: los paquetes viven en
   `.pylibs/` dentro del montaje, así que hay que decirle a Python de dónde importar y
   dónde encontrar el comando `gunicorn`.

   <!-- /track -->

   > ⌨️ **Este es el comando que más veces vas a reescribir hoy.** No hace falta que lo
   > reescribas: Ctrl-C para el contenedor, y **↑ Enter** para arrancarlo otra vez. Vuelve a
   > leer tus ficheros fuente y (desde el Módulo 3.1) tu `.env` cada vez. También está
   > escrito en [`workshop/README.md`](https://github.com/jespino/observability-workshop/blob/main/workshop/README.md)
   > si pierdes el historial de tu shell.

6. En una **quinta terminal**, sirve el frontend — es una SPA estática, así que un nginx de
   serie sirve:

   ```bash
   docker run -d --name workshop-frontend \
     -p 8080:80 \
     -v "$PWD/frontend:/usr/share/nginx/html:ro" \
     nginx:alpine
   ```

7. Ahora comprueba todo de una vez:

   ```bash
   ./doctor.sh
   ```

   Todas las filas tienen que estar en ✅ antes de continuar. `doctor.sh` no arranca nada —
   solo mira, y te dice a qué paso de este módulo tienes que volver. Es uno de los cuatro
   únicos scripts que quedan en el repositorio, y ninguno de ellos prepara nada.

8. Abre el frontend en `http://localhost:8080` y crea unos cuantos todos a mano. Que al
   menos uno de ellos:
   - tenga la etiqueta `urgent`
   - esté en la lista `work`
   - tenga el título vacío (sí, a propósito — mira qué pasa)

   > 💡 **Usa el selector de track en la barra de navegación, arriba del todo.** Cada
   > módulo está escrito una sola vez para los dos tracks; el selector cambia los ejemplos
   > de código a tu lenguaje y recuerda tu elección mientras avanzas por el taller.

9. Confirma que tu API responde directamente:

   <!-- track:js -->

   ```bash
   curl -s localhost:8081/healthz
   curl -s "localhost:8081/todos?list=work" | head -c 400
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   curl -s localhost:8082/healthz
   curl -s "localhost:8082/todos?list=work" | head -c 400
   ```

   <!-- /track -->

10. Arranca el generador de carga para que siempre haya tráfico que mirar:

    <!-- track:js -->

    ```bash
    ./loadgen.sh 2m
    ```

    <!-- /track -->

    <!-- track:python -->

    ```bash
    TRACK=py ./loadgen.sh 2m
    ```

    <!-- /track -->

Eso son ocho cosas funcionando. Aquí están todas:

| | Corre como | Puerto |
|---|---|---|
| `victoria-metrics` | binario nativo, en background | 8428 |
| `victoria-logs` | binario nativo, en background | 9428 |
| `victoria-traces` | binario nativo, en background | 10428 |
| `otelcol-contrib` | binario nativo, su propia terminal | 4317 / 4318 |
| `workshop-postgres` | contenedor | 5432 |
| `workshop-notify` | contenedor | 8083 (py) / 8084 (js) |
| `workshop-todo-api` | contenedor, tu terminal | 8081 (js) / 8082 (py) |
| `workshop-frontend` | contenedor | 8080 |

### Comprueba en la UI de Victoria

vmui de VictoriaLogs (`http://localhost:9428/select/vmui`), pega:

```
service.name:todo-api
```

Deberías ver: **ningún resultado de `todo-api`.** Su salida se va a la terminal donde lo
arrancaste, y a ningún otro sitio. Ahora mismo la única forma de depurar esta app es
entornar los ojos mirando esa terminal.

Y ahora pega esto:

```
service.name:notify-service
```

Esa **sí** devuelve líneas de log — el servicio de más abajo, el que ya venía
instrumentado, está mandando todo a VictoriaLogs, a través del Collector, desde dentro de
un contenedor. Así que el pipeline funciona. Lo único que falta es tu servicio.

### Comprueba en Grafana

Todavía no.

### Para pensar

- Acabas de crear un todo con el título vacío. ¿Qué pasó en la interfaz? Con lo que tienes
  ahora mismo, ¿podrías decir *por qué*?
- La telemetría de `notify-service` llega y la tuya no, y la diferencia son diez variables
  de entorno en un fichero. ¿Qué te dice eso sobre dónde tiene que vivir la configuración
  de la instrumentación?

### Si te atascas

- **`docker run` falla con "address already in use".** Algo ya tiene ese puerto — a menudo
  un contenedor de una ejecución anterior. `docker rm -f workshop-todo-api` y vuelve a
  intentarlo.
- **La API no llega a la base de datos** (`ECONNREFUSED` / `could not connect to server`).
  Casi seguro que tienes `localhost` donde necesitas `host.docker.internal` en
  `DATABASE_URL`. Dentro del contenedor, `localhost` es el contenedor.
- **(Python) `gunicorn: executable file not found`.** Falta el flag `-e PATH=…`, o está mal
  escrito. `gunicorn` se instaló en `/app/.pylibs/bin`, que no está en el PATH de la imagen
  de serie.
- **(JS) `Cannot find module 'express'`.** El paso 4 no se ejecutó, o se ejecutó contra el
  directorio del otro track. Comprueba `ls track-js/todo-api/node_modules | head`.
- **`node_modules` o `.pylibs` es de root** y no lo puedes borrar. Faltaba el flag
  `-u "$(id -u):$(id -g)"`. Bórralo con `sudo rm -rf` y repite el paso 4.
- **El frontend carga pero no crea todos.** Revisa el selector de track en la cabecera del
  frontend — puede estar apuntando al track que no arrancaste.
- **`loadgen.sh` termina al instante.** Necesita que la API esté sana primero. Vuelve a
  ejecutar `./doctor.sh`.

Este módulo no tiene checkpoint — todavía no has editado nada. A partir de aquí, cada
ejercicio termina con uno:

```bash
./checkpoint.sh NM   # copia la solución de referencia sobre tu directorio de trabajo
```

**Dilo en voz alta ahora:** si llevas más de diez minutos atascado, ejecuta el checkpoint y
sigue. Nadie está juzgando. 🤝

## Lo Que Hemos Conseguido

- Has descargado y arrancado tú mismo cuatro procesos de telemetría, y sabes qué hace cada flag
- Hay cuatro contenedores funcionando, y sabes cuál de ellos puede ver qué
- `./doctor.sh` está todo en verde
- Las tres UIs de Victoria en marcadores
- Tienes un track, y la página te está mostrando código en tu lenguaje
- Has demostrado, empíricamente, que la aplicación es invisible ahora mismo — mientras que
  el servicio de al lado no lo es

## Siguientes Pasos

Antes de instrumentar la aplicación, deberías saber qué *hace*. Continúa con el [Módulo 1: La Aplicación que Vas a Instrumentar](./01-the-application.es.md).
