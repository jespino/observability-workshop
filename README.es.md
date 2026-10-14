# Instrumentando tu aplicación con OpenTelemetry

¡Bienvenido a un taller práctico donde instrumentarás desde cero una aplicación de varios servicios con OpenTelemetry, y enviarás métricas, logs y trazas al stack de VictoriaMetrics! Aproximadamente 25% charla, 75% teclado.

**Tres tracks en paralelo: Node.js/Express, Python/Flask y Go.** Todo el mundo construye el mismo sistema, contra el mismo backend y con los mismos ejercicios — solo cambia la superficie del SDK. Es deliberado: los tres tracks miran sus datos en las mismas UIs, con los mismos lenguajes de consulta, lo que deja claro que OpenTelemetry es un protocolo y un modelo de datos, no una librería. Donde la telemetría de los tracks sí difiere, los guiones lo dicen — y eso también es una lección.

## Idiomas / Languages

- English: [README.md](./README.md) · módulos como `*.md`
- Español: [README.es.md](./README.es.md) · módulos como `*.es.md`

## Prerrequisitos

- Soltura con APIs HTTP y con JavaScript, Python o Go
- **Docker** con el plugin de Compose, instalado y funcionando (`docker ps`, `docker compose version`), además de `git`, `curl` y una shell Unix
- **No necesitas Node.js ni Python instalados.** La aplicación corre en una imagen
  `node:22-slim` o `python:3.12-slim` de serie de Docker Hub, con tu directorio de código
  montado dentro — nada de este repositorio se construye como imagen
- **No hace falta saber nada de OpenTelemetry, MetricsQL ni LogsQL**
- Al menos 4GB de RAM libre y unos 4GB de espacio en disco

Todo el stack — tu API, Postgres, notify-service, el frontend, el OpenTelemetry Collector
y los tres backends de Victoria — es un único `docker-compose.yml` que se arranca con un
solo comando. El código de tu API está montado dentro de su contenedor y se recarga solo,
así que guardar un fichero es todo lo que hace falta para desplegar un cambio.

## Descripción General del Taller

Este taller consta de 9 módulos que te llevan desde un servicio que no emite absolutamente nada hasta uno que puedes depurar solo con telemetría. Instrumentarás un sistema de varios servicios, ejecutarás un OpenTelemetry Collector como único punto de entrada y consultarás los resultados en VictoriaMetrics, VictoriaLogs, VictoriaTraces y Grafana:

### [Módulo 1: Observabilidad y OpenTelemetry](./modules/01-observability-and-opentelemetry.es.md)

Qué significa observabilidad, las tres señales en las que se apoya — métricas, logs y trazas — y qué es OpenTelemetry.

### [Módulo 2: Preparación y Comprobación del Entorno](./modules/02-setup-and-sanity-check.es.md)

Arranca todo el stack, tu API incluida, con un solo comando de Docker Compose, y comprueba con el script doctor que todo está funcionando.

### [Módulo 3: La Aplicación que Vas a Instrumentar](./modules/03-the-application.es.md)

Recorre la todo-api que vas a editar durante todo el día — sus endpoints, sus dependencias y su código.

### [Módulo 4: El Camino Fácil: Auto-instrumentación Sin Código](./modules/04-auto-instrumentation.es.md)

Haz que fluyan las tres señales sin tocar el código de la aplicación.

### [Módulo 5: El OpenTelemetry Collector](./modules/05-the-collector.es.md)

Qué es el OpenTelemetry Collector, qué puede hacer y qué hace el Collector de este taller con tus métricas, logs y trazas.

### [Módulo 6: Métricas Manuales](./modules/06-manual-metrics.es.md)

Elige el instrumento correcto, emite métricas de negocio y enfréntate a la cardinalidad antes de que te muerda.

### [Módulo 7: Logs Manuales](./modules/07-manual-logs.es.md)

Emite logs estructurados que arrastren el contexto de traza, y consúltalos con LogsQL.

### [Módulo 8: Trazas Manuales](./modules/08-manual-traces.es.md)

Crea spans propios, registra errores como toca y propaga el contexto entre servicios y lenguajes.

### [Módulo 9: Realidades de Producción y Cierre](./modules/09-production-realities.es.md)

Coste, sampling, PII y estrategia de despliegue: qué cambia cuando esto sale de tu portátil.

## Cómo Empezar

1. Clona este repositorio y haz `cd observability-workshop/workshop` — todo lo que ejecutas vive ahí
2. Comienza con el [Módulo 1](./modules/01-observability-and-opentelemetry.es.md), una introducción a la observabilidad y a OpenTelemetry. No hay nada que ejecutar antes: el [Módulo 2](./modules/02-setup-and-sanity-check.es.md) *es* la preparación del entorno
3. Haz los módulos en orden — cada uno se apoya en el anterior
4. Elige el track en el que quieras escribir: JavaScript, Python o Go. Los tres construyen el mismo sistema contra el mismo backend.
5. En la [web](https://jespino.github.io/observability-workshop/es/), usa el **selector de track** de la barra de navegación para cambiar todos los ejemplos de código a tu lenguaje. Tu elección se recuerda mientras te mueves entre módulos.

## Estructura del Repositorio

```
.
├── README.md                 # Versión en inglés
├── workshop.md               # Documento de diseño para facilitadores: esquema, tiempos, equipo
├── workshop/                 # ◀ EL DIRECTORIO EN EL QUE TRABAJAS
│   ├── docker-compose.yml    #   todo el stack, arrancado en el Módulo 2
│   ├── collector/            #   la config del Collector, explicada en el Módulo 5
│   ├── postgres/init.sql     #   esquema y datos iniciales
│   ├── grafana/provisioning/ #   datasources, montados en el contenedor de Grafana
│   ├── frontend/             #   SPA compartida
│   ├── track-js/todo-api/    #   el servicio que editas (Node.js / Express)
│   ├── track-py/todo-api/    #   el servicio que editas (Python / Flask)
│   ├── track-go/todo-api/    #   el servicio que editas (Go / net/http)
│   ├── services/             #   notify-service pre-instrumentado, en Python
│   ├── solutions/            #   soluciones de referencia, una por ejercicio
│   ├── doctor.sh             #   ¿está todo en verde?
│   ├── loadgen.sh            #   generar tráfico
│   └── reset.sh              #   borrar la telemetría, conservar los todos
├── modules/                  # Guiones para participantes (markdown), un fichero por módulo
│   ├── 01-observability-and-opentelemetry.es.md
│   ├── 02-setup-and-sanity-check.es.md
│   └── ...
├── website-generator/        # Programa Go que genera la web a partir del markdown
│   ├── main.go
│   ├── templates.go
│   ├── serve.go
│   └── README.md
├── website/                  # Web generada (HTML), servida desde GitHub Pages
│   ├── index.html
│   └── es/
└── Makefile                  # Automatización
```

Consulta [`workshop/README.md`](workshop/README.md) para el mapa de puertos, una chuleta
con todos los comandos que los guiones te hacen escribir, las reglas de red de los
contenedores y las notas para facilitadores.

## Generador de la Web

Este repositorio incluye un programa en Go que genera una web estática y multilingüe a partir de los ficheros markdown de los módulos.

### Generar la Web

```bash
# Con make (recomendado)
make website

# O directamente
cd website-generator
go run . -modules ../modules -output ../website
```

### Servir en Local

```bash
# Arranca un servidor local con recarga automática
make serve

# Después abre http://localhost:8000 en tu navegador
```

El generador:

- Convierte markdown a HTML usando [goldmark](https://github.com/yuin/goldmark)
- Conserva todo el formato, los emojis y los bloques de código
- Genera la navegación entre módulos
- Crea una página índice con el resumen de módulos
- Construye las webs en inglés y español con un selector de idioma
- Incluye estilos CSS adaptables

Consulta [website-generator/README.md](website-generator/README.md) para más detalles.

## Consejos para el Éxito

- Pega las consultas exactas del guion — lo de "explora los datos" viene después
- Cuando algo pinte mal, lee el log del Collector antes que nada: `docker compose logs -f otel-collector`
- Dentro de un contenedor, `localhost` es el contenedor: los contenedores se encuentran por nombre de servicio (`otel-collector`, `postgres`). Casi todos los "la telemetría no llega y no dice nada" son esto
- Puedes mirar las soluciones en el repositorio

## Recursos

- [Documentación de OpenTelemetry](https://opentelemetry.io/docs/)
- [Convenciones Semánticas de OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [MetricsQL de VictoriaMetrics](https://docs.victoriametrics.com/metricsql/)
- [LogsQL de VictoriaLogs](https://docs.victoriametrics.com/victorialogs/logsql/)
- [VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/)

## Completando el Taller

Al completar todos los módulos, habrás:

- **Instrumentado un servicio** con métricas, logs y trazas, partiendo de cero
- **Ejecutado un OpenTelemetry Collector** como punto de entrada único, repartiendo hacia tres backends
- **Elegido los instrumentos correctos** y razonado sobre cardinalidad antes de que doliera
- **Emitido logs estructurados** que arrastran el contexto de traza, y consultado con LogsQL
- **Escrito spans propios** y propagado el contexto entre servicios y lenguajes

**¡Felicidades!** Habrás ganado la confianza para instrumentar tus propios servicios de forma deliberada y no por accidente. Este conocimiento te permite:

- Añadir a tus servicios telemetría que responda preguntas reales
- Diseñar dashboards y alertas alrededor de preguntas, no de lo que resultó fácil de emitir
- Razonar sobre el coste y la cardinalidad de todo lo que recoges

## Contribuir

¿Encontraste un problema, tienes una idea de mejora o quieres añadir más módulos? ¡Por favor [abre un issue](https://github.com/jespino/observability-workshop/issues) o envía un pull request!

## Licencia

Este proyecto está bajo la Licencia MIT - consulta el fichero [LICENSE](LICENSE) para más detalles.

---

**¡Feliz instrumentación, y bienvenido al mundo de la telemetría que de verdad sirve para algo!**
