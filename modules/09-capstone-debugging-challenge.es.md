# Módulo 9: Proyecto Final — El Reto de Depuración

**Duración:** ~45 min (30 min investigando, 15 min de puesta en común) · **Equipos mixtos de los dos tracks**

Todo lo que has construido es ahora una herramienta. Aquí está la prueba. 🕵️

## Objetivos de Aprendizaje

Al terminar este módulo, serás capaz de:

- Haber diagnosticado un fallo desconocido usando solo telemetría
- Saber por experiencia qué señal destapa cada tipo de problema
- Tener una lista concreta de lo que ojalá hubieras instrumentado

## Prerrequisitos

- Módulos 0–8 completados
- Tu dashboard, tu panel de logs y tus cuatro saltos de Grafana funcionando

## El montaje

Tu facilitador va a reiniciar `todo-api` como una "nueva release" y a subir la carga. Tarda unos treinta segundos — una variable de entorno y un perfil de carga más agresivo.

El único briefing que recibes es:

> **"Los usuarios se quejan de que la app va lenta. Adelante."**

## Las reglas

1. **No se lee código de la aplicación hasta que puedas señalar el problema en la telemetría.** En eso consiste el ejercicio. Saltarte esta regla no nos engaña a nosotros, te engaña a ti.
2. **Equipos de 2–3, mezclando tracks.** Alguien de JS y alguien de Python mirando el mismo dashboard es la pareja ideal.
3. **Escribe mientras trabajas.** Mantened un documento compartido con cuatro columnas, y rellenadlo sobre la marcha:

   | Síntoma | Señal que lo mostró | Hipótesis | Confirmación |
   |---|---|---|---|
   | | | | |

4. **Hay más de un problema.** No te pares en el primero que encuentres.
5. A los diez minutos, si no tenéis nada, vuestro facilitador dará una pista. Cogedla — no hay premio por sufrir.

## Por dónde empezar

Si estás mirando una pantalla en blanco, aquí tienes una secuencia que funciona. Es la misma que usarías en un sistema real.

1. **Confirma que el síntoma es real, en métricas.** ¿Ha subido de verdad la latencia? ¿En todos los endpoints o en algunos? ¿Desde cuándo?

   ```
   histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket[5m])) by (le, http_route))
   ```

2. **Estrecha a una ruta, y luego a un atributo.** ¿Qué ruta se degradó? ¿Correlaciona con una `list`, una `priority`, un cliente?

3. **Busca errores, por separado de la latencia.** A menudo son dos problemas distintos y el ruidoso tapa al silencioso.

   ```
   service.name:todo-api AND severity_text:ERROR AND _time:30m | stats by (body) count()
   ```

4. **Vete a las trazas de las peticiones lentas.** No la media — una lenta.

   ```
   _time:30m "resource_attr:service.name":=todo-api duration:>1s
   ```

   Y pregúntale a la cascada: ¿hay un span lento, o hay *muchos* spans? Son enfermedades completamente distintas.

5. **Mira tus métricas de negocio.** Las que construiste en el Módulo 5 y que nadie más tendría. ¿Alguna se está comportando de forma rara de un modo que las métricas HTTP no muestran?

6. **Revisa el propio pipeline.** Si algo parece imposible, asegúrate de que no estás mirando un problema del Collector (Módulo 4.2).

## Qué estás buscando

Hay dos fallos independientes activos. Están elegidos a propósito para que los encuentren señales distintas, así que un equipo que solo mire un dashboard encontrará uno y se perderá el otro.

Sin destripar nada:

- Uno de ellos es **invisible en métricas agregadas** porque afecta a una minoría de peticiones. Las medias lo tapan. Solo lo verás si miras una traza concreta, o si troceas por el atributo correcto.
- El otro es **completamente invisible en trazas**, y no porque sea rápido: porque no toca la red en absoluto. Ahí no intervino ninguna librería, así que ninguna librería lo instrumentó. Solo una métrica que construiste tú en el Módulo 5 delata que existe.

El segundo es la lección más afilada. Una traza te enseña lo que cruzó una frontera. El
trabajo que se queda dentro de tu proceso es invisible salvo que *tú* lo hagas visible.
**Toda señal tiene un punto ciego, y no es el mismo punto ciego.**

## Puesta en común [~15 min]

Es la conversación más importante del taller. No dejes que se la coma el reloj.

Dad la vuelta a la sala:

1. **¿Qué señal te lo destapó?** Distintos equipos responderán distinto para el mismo fallo, y eso ya es interesante de por sí.
2. **¿Qué te habría gustado instrumentar en el Módulo 5 y no hiciste?** Sé concreto. Es la pregunta más valiosa del día.
3. **¿Qué construiste que resultó ser inútil?** Igual de valiosa, y se pregunta mucho menos.
4. **¿Qué añadirías a tu propio servicio el lunes?** Una cosa. Concreta. Escríbela.

Escribid la respuesta colectiva en la pared y dejadla ahí el resto de la sesión.

## Para pensar

Has encontrado los problemas con un sistema que construiste en un día, sobre una aplicación que no habías visto hasta esta mañana. El sistema en el que trabajas de verdad es más grande y lo conoces mejor.

Entonces: ¿cuál es el bloqueo real para hacer esto en el trabajo? ¿Es de herramientas, o es otra cosa?

## Si te atascas

- **Todo parece normal.** Amplía el rango de tiempo hasta antes del despliegue y compara. Una métrica que aislada parece bien, a menudo se ve claramente mal al lado de la de ayer.
- **Habéis encontrado un fallo y la sala sigue.** Volved al paso 5. El segundo fallo no está en la misma señal que el primero.
- **Quieres leer el código.** Todavía no. Diez minutos más. 😉

## Lo Que Hemos Conseguido

- Un diagnóstico real, del síntoma a la causa raíz, usando solo telemetría
- Experiencia directa del punto ciego de cada señal
- Una lista concreta y personal de qué instrumentar a continuación

## Siguientes Pasos

Continúa con el [Módulo 10: Realidades de Producción y Cierre](./10-production-realities.es.md).
