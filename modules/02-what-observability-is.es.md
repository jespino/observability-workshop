# Módulo 2: Qué es Realmente la Observabilidad

**Duración:** ~30 min · **Charla y demo — portátiles casi cerrados**

Es el único módulo sin código que escribir. Existe para que todo lo que viene después caiga sobre un vocabulario compartido en vez de sobre una niebla de palabras de moda. 🧠

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Explicar el modelo de datos de OpenTelemetry — resource, scope, signal, atributos, contexto — sin decir "los tres pilares"
- Distinguir la API, el SDK, una librería de instrumentación, un exporter y el Collector, y ver por qué esa distinción resuelve la mayor parte de la confusión posterior
- Entender por qué hay un Collector en medio de nuestra arquitectura
- Saber qué partes del ecosistema OpenTelemetry son estables y cuáles todavía se mueven

## Prerrequisitos

- Módulo 1 completado — deberías poder describir la app

## 2.1 — Empezamos con un fallo, no con una definición

Tu facilitador va a enseñaros dos grabaciones de la misma investigación:

1. **Con el stack terminado.** Un usuario se queja de que la app va lenta. Noventa segundos después, alguien está señalando la línea de código exacta.
2. **Con `docker logs | grep`.** La misma investigación. No termina.

No te preocupes todavía de *cómo* funciona la primera. Solo fíjate en adónde vamos — vas a construir cada pieza tú mismo.

## 2.2 — Por qué "los tres pilares" es un marco engañoso

Seguramente has visto la observabilidad descrita como tres pilares: métricas, logs, trazas. La imagen es mala, porque los pilares están separados, y la separación es justo lo que no quieres.

Lo que de verdad hace que la observabilidad funcione es que una línea de log, un punto de métrica y un span llevan:

- **el mismo resource** — coinciden en quién los emitió, y
- **el mismo contexto de traza** — se pueden coser en una única historia sobre una única petición.

> **La correlación es el producto. Las señales son la materia prima.** 🔗

Todo lo que hay en los Módulos 5 a 8 está al servicio de esa frase. Cuando lleguemos al Módulo 8 harás clic desde una gráfica hasta una línea de log, hasta una traza y hasta una línea de código, y eso solo funciona por decisiones tomadas en los módulos anteriores.

## 2.3 — El modelo de datos de OpenTelemetry

Cinco conceptos. Eso es todo.

**Resource** — *quién* emite. Se define una vez al arrancar y se adjunta a todo lo que produce el proceso.

```
service.name          = todo-api
service.version       = 1.0.0
deployment.environment = workshop
```

`service.name` es el atributo más importante de todo el sistema. Si lo pones mal, nada correlaciona.

**Instrumentation scope** — *qué librería* produjo este trozo de telemetría. Te permite distinguir "esto lo dijo la instrumentación de HTTP" de "esto lo dijo la aplicación".

**El payload de la señal** — un punto de métrica, un registro de log o un span. Es la única parte que cambia según la señal.

**Atributos** — pares clave/valor sobre el payload. La dimensión por la que troceas. `http.route`, `todo.list`, `error.type`.

> ⚠️ Los atributos son también donde se esconden los costes. Cada combinación distinta de valores de atributo en una métrica es una serie temporal nueva. Recuerda esta frase; la vas a pagar en el Módulo 5.4.

**Contexto y propagación** — el mecanismo que arrastra la traza y el span actuales junto con la petición, incluso entre procesos, mediante la cabecera W3C `traceparent`:

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
             ^  ^                                ^                ^
             |  trace id                         span id padre    flags
             versión
```

Si hoy no te llevas nada más, llévate esto: cuando alguien dice "la traza se rompe en esta frontera", casi siempre quiere decir *que esa cabecera no llegó al otro lado*.

## 2.4 — API vs SDK vs instrumentación vs exporter vs Collector

Esta es la distinción que resuelve la mitad de las preguntas que la gente hace después. Tu facilitador la va a dibujar; aquí está en palabras:

| Pieza | Qué es | Quién la publica |
|---|---|---|
| **API** | La interfaz que llama tu código: `counter.add()`, `tracer.startSpan()`. Por sí sola no hace nada — es un no-op salvo que haya un SDK registrado. | OpenTelemetry |
| **SDK** | La implementación. Batching, sampling, views, detección de resource. Se configura una vez al arrancar el proceso. | OpenTelemetry |
| **Librería de instrumentación** | Código que llama a la API *por ti*, enganchándose a un framework — Express, Flask, `pg`, `psycopg`. De esto está hecha la "auto-instrumentación". | OpenTelemetry / comunidad |
| **Exporter** | Convierte los datos del SDK a un formato de red y los envía. Usaremos OTLP sobre HTTP. | OpenTelemetry |
| **Collector** | Un proceso aparte que recibe, procesa y reexporta telemetría. Opcional, pero mira abajo. | OpenTelemetry |

La consecuencia práctica: **una librería que solo usa la API no añade dependencia con ningún backend.** Ese es el objetivo de diseño, y por eso instrumentar una librería que mantienes es un compromiso mucho menor de lo que la gente supone.

## 2.5 — Por qué hay un Collector en nuestra arquitectura

A estas alturas la pregunta razonable es: nuestros tres backends hablan OTLP, ¿por qué no exportar directamente a ellos?

Porque no lo hablan *en la misma dirección*:

| Señal | Endpoint |
|---|---|
| Métricas | `http://victoriametrics:8428/opentelemetry/v1/metrics` |
| Logs | `http://victorialogs:9428/insert/opentelemetry/v1/logs` |
| Trazas | `http://victoriatraces:10428/insert/opentelemetry/v1/traces` |

Tres hosts, tres rutas. El Collector le da a tu aplicación **un solo** `OTEL_EXPORTER_OTLP_ENDPOINT` y hace el reparto él.

Esa es la razón inmediata. Las razones duraderas aparecen después:

- Es donde pones el **sampling** sin redesplegar la app (Módulo 7.5)
- Es donde **redactas** PII de forma central en lugar de en doce servicios (Módulo 10)
- Es donde generas **métricas a partir de spans** gratis (Módulo 8.1)
- Es donde se **almacena y reintenta** cuando un backend está caído, para que tu app no tenga que hacerlo

Hay un ejercicio de apéndice opcional más adelante que te hace exportar directamente a los backends y luego hacer balance de lo que perdiste. Recomendado si eres de los que sospechan que el Collector es puro ceremonial.

## 2.6 — El stack que usamos, y una propiedad genuinamente agradable

- **VictoriaMetrics** guarda métricas y habla **MetricsQL**, un superconjunto de PromQL. Si sabes PromQL, ya lo sabes.
- **VictoriaLogs** guarda logs y habla **LogsQL**.
- **VictoriaTraces** guarda trazas y habla... **LogsQL** también.

Merece la pena pararse en ese último punto. **Aprender un lenguaje de consulta te da dos de las tres señales.** En el Módulo 7 vas a consultar spans con la misma sintaxis que aprendiste para logs en el Módulo 6, y va a parecer trampa. Comparado con aprender PromQL *y* LogQL *y* TraceQL, es un ahorro real.

VictoriaTraces además implementa la **API de consulta de Jaeger**, lo que significa que el datasource Jaeger integrado en Grafana funciona contra ella sin ningún plugin, y te da la vista de cascada de siempre gratis.

## 2.7 — Madurez de las señales, con honestidad

No todo aquí está igual de terminado, y es mejor que te lo digamos nosotros que descubrirlo a las 2 de la mañana:

- **Trazas y métricas** son estables en los dos SDKs. Úsalas con confianza.
- **La señal de logs** todavía se está moviendo. Los paquetes del SDK están marcados como experimentales tanto en JavaScript como en Python. Los vamos a usar igualmente, porque son la vía más limpia para enseñar la historia de la correlación — pero fija las versiones, y en el Módulo 6.4 te enseñaremos la alternativa sólida como una roca (JSON por stdout más un receiver `filelog` en el Collector) que quizá prefieras en producción.
- **Algunas convenciones semánticas** siguen marcadas como experimentales y pueden cambiar entre versiones. Es un coste real; lo volveremos a nombrar en el Módulo 7.3.
- **VictoriaMetrics no soporta exemplars.** Eso elimina un flujo de correlación concreto que sí te daría un stack Prometheus+Tempo. Te enseñaremos el rodeo en el Módulo 8.2 y seremos explícitos sobre el compromiso.

## Debate (5 min, en parejas)

> Piensa en el último incidente de producción en el que estuviste. **¿Qué señal lo habría acortado, y la tenías?**

Dos o tres parejas comparten. Guarda tu respuesta — en el Módulo 9 sabrás si acertaste.

## Lo Que Hemos Conseguido

- Un vocabulario compartido: resource, scope, signal, atributos, contexto
- Un mapa claro de API vs SDK vs instrumentación vs exporter vs Collector
- Una razón concreta para cada caja del diagrama de arquitectura
- Una imagen honesta de qué partes de este ecosistema siguen moviéndose

## Siguientes Pasos

Basta de hablar. Continúa con el [Módulo 3: El Camino Fácil — Auto-instrumentación Sin Código](./03-auto-instrumentation.es.md), donde la app empieza a emitir las tres señales sin que le cambies ni una línea.
