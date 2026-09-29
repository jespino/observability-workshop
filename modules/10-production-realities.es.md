# Módulo 10: Realidades de Producción y Cierre

**Duración:** ~30 min · **Charla, sin ejercicios**

Todo lo que has construido hoy corría en un portátil, con un servicio y un generador de carga. Esto es lo que cambia cuando no es así. 🚀

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Saber a dónde se va el dinero en cada señal, y cómo razonar sobre ello a grandes rasgos
- Saber qué hay que redactar y dónde hacerlo
- Haber visto alertas sobre MetricsQL y sobre LogsQL
- Tener un plan concreto y ordenado para desplegar esto en un equipo real

## Prerrequisitos

- Módulo 9 completado, incluida la puesta en común

## 10.1 — Coste

Cada señal tiene su propio motor de coste, y fallan de forma distinta.

| Señal | Qué cuesta dinero | Qué explota |
|---|---|---|
| **Métricas** | Cardinalidad — series temporales activas | Un atributo sin acotar. Lo viste pasar en el Módulo 5.4. El crecimiento es *permanente* hasta que expira la retención |
| **Logs** | Volumen — bytes ingeridos y almacenados | Un log de debug olvidado en un bucle caliente. El crecimiento es proporcional al tráfico, así que al menos es predecible |
| **Trazas** | Tasa de sampling × spans por traza | Un N+1 en un camino caliente. Una sola petición puede producir mil spans |

Intuiciones útiles:

- El coste de métricas lo mueven las *combinaciones distintas de etiquetas*, no la tasa de peticiones. Diez millones de peticiones sobre diez series es barato. Diez mil peticiones sobre diez millones de series no.
- El coste de logs escala con el tráfico, lo que significa que puedes preverlo. El coste de cardinalidad de métricas escala con el *descuido*, lo que significa que no.
- Una traza vale como mil líneas de log durante un incidente, y nada en absoluto el resto del tiempo. Esa asimetría es la razón de ser del tail sampling.

## 10.2 — Retención

Distinta por señal, y por buenas razones:

| Señal | Retención típica | Por qué |
|---|---|---|
| Métricas | 6–13 meses | Baratas por unidad y necesitas comparar año contra año |
| Logs | 7–30 días | Caros por unidad; casi todo el valor está en las primeras horas |
| Trazas | 3–7 días | Igual que los logs, y más |

Los flags los escribiste tú en el Módulo 0.1:

```
victoria-metrics  -retentionPeriod=1   -search.latencyOffset=1s ...
victoria-logs     -retentionPeriod=7d
victoria-traces   -retentionPeriod=3d
```

La pregunta interesante en producción no es "cuánto tiempo" — es "qué hago con el *agregado* cuando expira el detalle". Las recording rules que consolidan una métrica de alta resolución en otra de baja resolución son la respuesta estándar.

## 10.3 — PII y cumplimiento

La telemetría filtra datos personales en tres sitios, y el cuerpo de los logs es solo el más evidente:

1. **Cuerpos de log** — alguien loguea el objeto de petición entero. Una vez.
2. **Atributos de span** — una URL completa con un email en el query string, `db.query.text` con valores literales dentro.
3. **Atributos de métrica** — más raro, pero una etiqueta `user_email` es a la vez un incidente de privacidad y uno de cardinalidad.

Este es uno de los argumentos más fuertes a favor de la arquitectura con Collector. Puedes imponerlo **de forma central**, una vez, en lugar de revisarlo en cada servicio:

```yaml
processors:
  redaction:
    allow_all_keys: true
    blocked_values:
      - '[0-9]{16}'                                  # números de tarjeta
      - '\b[\w.%+-]+@[\w.-]+\.[A-Za-z]{2,}\b'        # emails
    summary: debug

  transform:
    log_statements:
      - context: log
        statements:
          - delete_key(attributes, "authorization")
```

Un equipo de servicio que se olvide queda cubierto por la plataforma. Esa es una postura de seguridad genuinamente distinta de "lo pusimos en el estándar de código".

Fíjate en el hábito que empezamos en el Módulo 6.2: loguear `title_length` en vez de `title`. La redacción en el Collector es la red de seguridad; no recogerlo desde el principio es el arreglo de verdad.

## 10.4 — Alertas

**vmalert** evalúa reglas y escribe los resultados de vuelta en VictoriaMetrics. El lado de MetricsQL es lo que esperarías:

```yaml
groups:
  - name: todo-api
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_server_duration_milliseconds_count{
            service_name="todo-api", http_response_status_code=~"5.."
          }[5m]))
          /
          sum(rate(http_server_duration_milliseconds_count{service_name="todo-api"}[5m]))
          > 0.05
        for: 5m
        labels: { severity: page }
        annotations:
          summary: "tasa de error de todo-api por encima del 5% durante 5 minutos"
```

Y aquí está la propiedad agradable de este stack: **vmalert también ejecuta reglas LogsQL**, contra VictoriaLogs y contra VictoriaTraces. Así que puedes alertar sobre un patrón de log o una condición de span y que el resultado aterrice en VictoriaMetrics como una serie temporal normal y corriente:

```yaml
- alert: CreateFailuresSpiking
  expr: 'service.name:todo-api AND severity_text:ERROR AND _msg:"create todo failed"'
  # evaluada por vmalert contra VictoriaLogs
  for: 10m
```

Eso significa un solo sistema de alertas, un solo camino de notificación y reglas sobre las tres señales. Tu facilitador hará una demo de una regla disparándose.

## 10.5 — Estrategia de despliegue para un equipo real

El orden importa más de lo que la gente espera. Este está probado en batalla:

1. **Auto-instrumenta todo.** Todos los servicios, solo configuración, sin cambios de código. Te va a asombrar lo lejos que llega esto, y probarlo no cuesta casi nada.
2. **Despliega un Collector** y enruta todo a través de él. Hazlo pronto, antes de tener veinte servicios con endpoints de backend a fuego.
3. **Añade spans de dominio a tus dos endpoints más dolorosos.** No a todos. Dos. Aquellos con los que ya temes que te llamen.
4. **Añade métricas de negocio** para los tres o cuatro números de los que tu equipo habla de verdad en la daily.
5. **Cablea la correlación** — los saltos del Módulo 8. Aquí es donde empieza a parecer un sistema y no tres herramientas.
6. **Y entonces, y solo entonces**, piensa en taxonomía, estándares de nombrado y gobierno.

> ❌ **No empieces diseñando una taxonomía de métricas.** Es el modo de fallo más común: seis semanas de reuniones, un documento precioso, cero telemetría en producción y el entusiasmo desaparecido. Envía el paso 1 esta semana en su lugar.

Dos consejos más que no cuestan nada:

- **Elige los valores de `service.name` con cuidado y no los cambies nunca.** Son la clave de join de todo.
- **Pon la configuración de tu SDK en un paquete interno compartido** desde el primer día. Cuando necesites añadir una View, cambiar un sampler o arreglar un atributo de resource en toda la flota, agradecerás que sea una subida de dependencia y no cuarenta pull requests.

## 10.6 — Qué leer después

- [Registro de convenciones semánticas de OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/) — hojea las secciones de las tecnologías que usas de verdad
- [Lista de processors del Collector](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor) — encontrarás algo útil que no sabías que existía
- [MetricsQL](https://docs.victoriametrics.com/metricsql/) — las funciones que PromQL no tiene
- [LogsQL](https://docs.victoriametrics.com/victorialogs/logsql/) — los operadores de pipe, sobre todo `stats` y `top`
- [Documentación de OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)

## Cierre

Mira atrás a lo que has hecho hoy:

- Empezaste con una aplicación que no emitía **nada**
- Conseguiste las tres señales fluyendo sin cambiarle una línea
- Entendiste y editaste el pipeline del medio
- Añadiste métricas de negocio, y rompiste tu base de datos de métricas a propósito para aprender por qué importa la cardinalidad
- Convertiste los logs en registros consultables que llevan contexto de traza
- Escribiste spans que describen tu dominio, y reparaste una traza a través de una frontera de servicio y de lenguaje
- Conectaste las tres señales en un sistema navegable
- Diagnosticaste dos fallos desconocidos usando solo telemetría

Y ahora mira la pared, las respuestas de la puesta en común del Módulo 9. Esa lista — *lo que añadiría el lunes* — es el resultado real de este taller.

**Por favor, rellena el formulario de feedback ahora, mientras sigues en la sala.** 📝 No luego por email. El luego nunca llega.

Gracias por dedicarle el día a esto. Ve a instrumentar algo. 👋

## Lo Que Hemos Conseguido

- Un modelo funcional de dónde vienen los costes de telemetría
- Una estrategia central para PII, y el hábito que la hace innecesaria
- Alertas sobre las tres señales desde un solo sistema
- Un plan de despliegue ordenado que puedes empezar el lunes

## Siguientes Pasos

Este es el último módulo. Vuelve al [índice del taller](../README.es.md) — o mejor, abre tu propio servicio y haz el paso 1.
