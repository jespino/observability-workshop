# Módulo 0: Preparación y Comprobación del Entorno

**Duración:** ~10 min · **Todo aquí es igual para los dos tracks**

El objetivo de este módulo es aburrido a propósito: arrancarlo todo con un solo comando, y
comprobar que todo está funcionando antes de empezar a enseñar nada. 🟢

## Prerrequisitos

- **Docker** con el plugin de Compose: `docker compose version` tiene que responder
- `git` y una shell Unix. En Windows, usa WSL2.
- Unos 4GB de RAM libre y 3GB de disco
- **No necesitas Node.js ni Python instalados** — todo corre en contenedores

> 📦 **Preparación previa, si puedes.** El stack son unos 2 GB de imágenes, y el wifi del
> sitio no va a ser tu amigo. Clona el repo (el paso 1 de abajo) y descárgatelas antes de
> venir:

```bash
cd observability-workshop/workshop
docker compose --profile js --profile py --profile grafana pull
```

## Preparación

1. Clona el repo y entra en el directorio del taller. **Quédate en `workshop/` el resto del
   día** — todos los comandos de todos los módulos dan por hecho que estás ahí.

   ```bash
   git clone https://github.com/jespino/observability-workshop
   cd observability-workshop/workshop
   ```

2. Elige tu track — el lenguaje en el que te sientas *cómodo*, para que la fricción de hoy
   sea OpenTelemetry y no la sintaxis:

   | Track | Puerto de tu API | Directorio que vas a editar |
   |---|---|---|
   | Node.js / Express | 8081 | `track-js/todo-api/` |
   | Python / Flask | 8082 | `track-py/todo-api/` |

   > 💡 **Usa el selector de track en la barra de navegación, arriba del todo.** Cambia
   > todos los ejemplos de código a tu lenguaje y recuerda tu elección.

3. Arranca el stack de tu track:

   <!-- track:js -->

   ```bash
   docker compose --profile js up -d
   ```

   <!-- /track -->

   <!-- track:python -->

   ```bash
   docker compose --profile py up -d
   ```

   <!-- /track -->

   Esto arranca un contenedor por pieza: **tu API**, su base de datos Postgres,
   `notify-service` (el servicio de más abajo al que llama tu API), el frontend, el
   OpenTelemetry Collector, y VictoriaMetrics, VictoriaLogs y VictoriaTraces para guardar
   métricas, logs y trazas. Sus datos viven en **volúmenes de Docker**, fuera del
   repositorio: sobreviven a los reinicios y a `docker compose down`, y
   `docker compose down -v` los borra.

   ![Todo lo que arranca docker compose: tu navegador llega al frontend en localhost; el frontend llama a todo-api, que usa Postgres y llama a notify-service; los dos envían OTLP al OpenTelemetry Collector, que reparte métricas, logs y trazas entre VictoriaMetrics, VictoriaLogs y VictoriaTraces, cada uno con su volumen; Grafana, a partir del Módulo 3, lee de los tres.](images/docker-compose-stack.png)

   El código de tu API está montado dentro de su contenedor, así que **guardar un fichero
   la reinicia** — ese es todo el ciclo de despliegue de hoy.

4. Comprueba que todo está funcionando:

   ```bash
   ./doctor.sh
   ```

   Espera a que el stack arranque — la primera vez tarda un minuto mientras tu API instala
   sus dependencias — y después comprueba cada pieza. **Todas las filas tienen que estar en
   ✅.** Al final te muestra las UIs que vas a usar todo el día: guárdalas en marcadores.

### Si te atascas

- **`docker compose` no es un comando.** Tienes el antiguo `docker-compose` independiente,
  o no tienes Compose. Instala el plugin de Docker Compose (viene con Docker Desktop).
- **`port is already allocated`.** Algo en tu máquina tiene cogido uno de los puertos (a
  menudo un contenedor viejo de una ejecución anterior). Búscalo con `docker ps`, páralo, y
  vuelve a lanzar el `up`.
- **Una fila de `doctor.sh` sale en rojo.** Las pistas de debajo de la tabla te dicen dónde
  mirar; normalmente es `docker compose logs SERVICIO` del servicio que ha fallado.

## Siguientes Pasos

Antes de instrumentar la aplicación, deberías saber qué *hace*. Continúa con el [Módulo 1: La Aplicación que Vas a Instrumentar](./01-the-application.es.md).
