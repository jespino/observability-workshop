# Módulo 2: Las Tres Señales — Métricas, Logs y Trazas

**Duración:** ~20 min · **Charla — sin código que escribir**

OpenTelemetry describe lo que está haciendo un sistema en marcha con tres tipos de datos,
llamados **señales**: métricas, logs y trazas. Este módulo explica cada una con un ejemplo
sacado de la aplicación que vas a instrumentar, para que el resto del taller se apoye en un
vocabulario compartido. 🧠

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Saber qué es una métrica, un registro de log y una traza
- Leer un ejemplo de cada una y decir qué significa cada parte
- Saber dónde se guarda cada señal en este taller

## Prerrequisitos

- Módulo 1 completado — deberías poder describir la app

## 2.1 — Métricas

Una **métrica** es un número medido a lo largo del tiempo. En vez de registrar cada evento,
la aplicación mantiene valores agregados — una cuenta, una suma, una distribución — y los
exporta periódicamente, por defecto cada 60 segundos (cada 10 en este taller). Cada
combinación de un nombre de métrica y un conjunto de atributos es una **serie temporal**
distinta.

OpenTelemetry tiene varios tipos de métrica. Los tres que más vas a usar son:

| Tipo | Qué registra | Ejemplo |
|---|---|---|
| **Counter** | Un total que solo sube | todos creados |
| **Histogram** | La distribución de los valores medidos | duración de las peticiones |
| **Gauge** | Un valor en un momento dado | todos abiertos ahora mismo |

### Ejemplo

Este es un counter que crearás en el Módulo 5, tal como lo guarda VictoriaMetrics:

```
todos_created_total{service_name="todo-api", list="work"}   1287
todos_created_total{service_name="todo-api", list="home"}    342
```

- `todos_created_total` es el nombre de la métrica. En el código se llama `todos.created`;
  el almacenamiento añade `_total` porque es un counter, y cambia los puntos por guiones
  bajos.
- `service_name="todo-api"` dice qué servicio la emitió. Sale de la configuración del
  servicio, no del código que incrementa el counter.
- `list="work"` es un **atributo** que se añade al incrementar el counter. Cada valor de
  `list` produce su propia serie temporal, y por eso hay dos líneas.
- `1287` es el valor: el número de todos creados en la lista `work` desde que arrancó el
  servicio. Por sí solo dice poco; lo que consultas es lo rápido que crece, por ejemplo
  "todos creados por segundo, por lista".

## 2.2 — Logs

Un **registro de log** describe un evento que ocurrió en un momento concreto: una petición
falló, se creó un todo, se reintentó una conexión. A diferencia de una métrica, cada evento
produce su propio registro. En OpenTelemetry un registro de log tiene un cuerpo (el
mensaje), una severidad, una marca de tiempo y atributos, y si se emitió mientras se
atendía una petición, los identificadores de la traza de esa petición.

### Ejemplo

Un registro de log que producirás en el Módulo 6, cuando se crea un todo:

```
timestamp:      2026-10-13T09:15:02.123Z
severity_text:  INFO
body:           todo created
attributes:
  list:         work
  todo_id:      5a082413-27bb-495a-8010-7bd006c20252
resource:
  service.name: todo-api
trace_id:       4bf92f3577b34da6a3ce929d0e0e4736
span_id:        00f067aa0ba902b7
```

- `timestamp` es cuándo ocurrió el evento, y `severity_text` cuánta importancia tiene
  (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- `body` es el mensaje. Es el mismo texto para cada todo creado, así que puedes contar y
  agrupar estos registros.
- `attributes` llevan los detalles que cambian de un evento a otro: qué lista, qué todo.
- `resource` describe quién emitió el registro: el mismo `service.name` que lleva la
  métrica.
- `trace_id` y `span_id` identifican la petición que se estaba atendiendo cuando se
  escribió el registro. Son lo que te permitirá saltar de esta línea de log a la traza
  completa de esa petición en el Módulo 8.

## 2.3 — Trazas

Una **traza** sigue una petición a través del sistema. Está formada por **spans**: cada
span es una operación con un momento de inicio, una duración y atributos — atender una
petición HTTP, ejecutar una consulta a la base de datos, llamar a otro servicio. Los spans
tienen padres, así que juntos forman un árbol que muestra qué pasó, en qué orden y cuánto
tardó cada paso. Cuando una petición pasa de un servicio a otro, el trace id viaja con ella
en una cabecera HTTP (`traceparent`), así que los spans de los dos servicios acaban en la
misma traza.

### Ejemplo

La traza de un `POST /todos`, simplificada y dibujada como un árbol (los nombres exactos de
los spans cambian un poco entre las librerías de JavaScript y de Python):

```
trace 4bf92f3577b34da6a3ce929d0e0e4736

POST /todos                       todo-api   ██████████████████████  24 ms
├─ pg.connect                     todo-api   ██                       2 ms
└─ INSERT INTO todos              todo-api       ███████              8 ms
```

Y el span de arriba en detalle:

```
name:            POST /todos
trace_id:        4bf92f3577b34da6a3ce929d0e0e4736
span_id:         00f067aa0ba902b7
parent_span_id:  (none — this is the root span)
start:           2026-10-13T09:15:02.101Z
duration:        24 ms
status:          OK
attributes:
  http.method:      POST
  http.route:       /todos
  http.status_code: 201
resource:
  service.name:     todo-api
```

- Todos los spans del árbol comparten el mismo `trace_id`: eso es lo que los convierte en
  una sola traza.
- El span raíz, `POST /todos`, no tiene padre. Cubre la petición entera, desde que llegó
  hasta que se envió la respuesta.
- `pg.connect` e `INSERT INTO todos` son sus hijos: tienen `parent_span_id` igual al
  `span_id` de la raíz, y ocurrieron dentro de ella. Las barras muestran cuándo empezó cada
  uno y cuánto duró.
- `attributes` describen la operación: qué método y qué ruta HTTP, qué código de estado se
  devolvió. Un span de base de datos llevaría la consulta en su lugar.
- `status` dice si la operación fue bien. Una petición fallida mostraría `ERROR`.
- El `span_id` del span raíz es el mismo que lleva el registro de log de la sección 2.2.
  Esa línea de log se escribió mientras este span estaba activo.

## 2.4 — Dónde va cada señal en este taller

Tu aplicación envía las tres señales al OpenTelemetry Collector, que reenvía cada una a su
propio almacenamiento:

| Señal | Se guarda en | Se consulta con |
|---|---|---|
| Métricas | VictoriaMetrics | MetricsQL (compatible con PromQL) |
| Logs | VictoriaLogs | LogsQL |
| Trazas | VictoriaTraces | LogsQL |

Los logs y las trazas comparten lenguaje de consulta: lo que aprendas para consultar logs
en el Módulo 6 te sirve también para las trazas en el Módulo 7.

## Lo Que Hemos Conseguido

- Sabes qué es una métrica, un registro de log y una traza
- Sabes leer un ejemplo de cada una y decir qué significa cada campo
- Sabes cómo se conectan las tres — el mismo `service.name`, y el mismo `trace_id` en logs
  y spans — y dónde se guarda cada una

## Siguientes Pasos

Continúa con el [Módulo 3: El Camino Fácil — Auto-instrumentación Sin Código](./03-auto-instrumentation.es.md), donde la app empieza a emitir las tres señales sin que cambies ni una línea de ella.
