# Spec — rex-stub (simulador externo de Rex+)

> Metodología SDD: este documento es la fuente de verdad. El código se deriva de él; si el código
> necesita algo que el spec no dice, se actualiza el spec primero.

## 1. Propósito

Servicio HTTP independiente que simula el ERP de RRHH **Rex+** para Promaty. Existe para que
`rrhh-server` integre un sistema **realmente externo** (otro proceso, otro lenguaje, otro
despliegue), tal como muestra el diagrama de arquitectura. No es un ERP: no calcula sueldos ni valida
reglas laborales; solo asigna números de referencia y permite simular fallos.

**Fuera de alcance**: base de datos, persistencia entre reinicios, autenticación de usuarios,
cálculo de remuneraciones, cualquier endpoint que Promaty no consuma hoy.

## 2. Contrato que reemplaza

Puerto actual en `rrhh-server` (`RexPlusClient`), hoy con adaptador sin red (`RexPlusStubClient`):

| Método del puerto                | Endpoint del stub                              |
| -------------------------------- | ---------------------------------------------- |
| `syncContract(Contract)`         | `POST /api/v1/contracts`                       |
| `annulContract(Contract, reason)` | `POST /api/v1/contracts/{ref}/annulment`      |
| `syncTermination(Termination)`   | `POST /api/v1/terminations`                    |
| `annulTermination(Termination)`  | `POST /api/v1/terminations/{ref}/annulment`    |
| `syncAnnex(Annex)`               | `POST /api/v1/annexes`                         |

Todos devuelven lo mismo que `RexSyncResult`: éxito con `referenceNumber`, o error con mensaje.

## 3. Stack y estructura

Go 1.23+, **solo librería estándar** (`net/http` con el mux de Go 1.22+ que soporta método y
`{param}` en la ruta). Sin frameworks: el servicio es chico y así se demuestra dominio del lenguaje.
Estructura calcada de `Solicitudes-documentos-Go` (capas `endpoint → service → repository`).

```
rex-stub/
├── cmd/main.go                   # arranque: lee config, arma dependencias, levanta el servidor
├── internal/
│   ├── config/config.go          # variables de entorno
│   ├── rex/
│   │   ├── model.go              # structs de request/response
│   │   ├── endpoint.go           # handlers HTTP
│   │   ├── service.go            # reglas: referencias, idempotencia, validación
│   │   └── repository.go         # almacén en memoria (map + mutex)
│   ├── middleware/
│   │   ├── apikey.go             # valida X-API-Key
│   │   ├── simulation.go         # latencia y fallos simulados
│   │   └── logging.go            # log por request
│   └── simulation/state.go       # estado mutable de la simulación (thread-safe)
├── docs/spec.md
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env.example
└── README.md
```

## 4. Configuración (variables de entorno)

| Variable              | Default        | Descripción                                             |
| --------------------- | -------------- | ------------------------------------------------------- |
| `PORT`                | `8090`         | Puerto HTTP                                             |
| `API_KEY`             | (obligatoria)  | Valor esperado en `X-API-Key`; el servicio no arranca sin ella |
| `SIM_LATENCY_MS`      | `0`            | Latencia artificial por request                         |
| `SIM_FAILURE_RATE`    | `0`            | Probabilidad (0.0–1.0) de responder 503                 |
| `SIM_FORCE_ERROR`     | `false`        | Si es `true`, todo endpoint de negocio responde 503     |

## 5. API

Base: `/api/v1`. JSON en UTF-8. Todos los endpoints de negocio exigen `X-API-Key`.

### 5.1 POST /contracts → 201

Body: `{ "externalId": "123", "staffRut": "12.345.678-9", "contractType": "O", "startDate": "2026-12-01",
"endDate": "2027-03-01", "baseSalary": 850000, "projectCode": "PRJ-01" }`.
Respuesta: `{ "referenceNumber": "REX-000001" }`.

### 5.2 POST /terminations → 201

Body: `{ "externalId": "45", "contractReferenceNumber": "REX-000001", "terminationDate": "2027-01-15",
"legalCause": "ART_159_4", "definitive": false }`.
Respuesta: `{ "referenceNumber": "REX-TERM-000001" }`.
Falla 422 si `contractReferenceNumber` no existe en el stub.

### 5.3 POST /terminations/{ref}/annulment → 200

Body: `{ "reason": "Error de digitación" }`. Respuesta: `{ "referenceNumber": "REX-TERM-000001" }`
(el mismo número, igual que hace hoy el stub Java). 404 si `{ref}` no existe.

### 5.4 POST /annexes → 201

Body: `{ "externalId": "7", "contractReferenceNumber": "REX-000001", "annexType": "SALARY",
"effectiveDate": "2027-02-01", "newBaseSalary": 900000 }`.
Respuesta: `{ "referenceNumber": "REX-ANX-000001" }`. 422 si el contrato no existe.

### 5.5 Validación de los bodies

Se informan **todos** los campos inválidos juntos (CA-8). Campos desconocidos se ignoran, para que
`rrhh-server` pueda enviar más datos sin romper el stub. Mensaje por campo: `campo obligatorio`,
`debe tener formato AAAA-MM-DD`, `debe ser mayor a 0`.

| Endpoint              | Obligatorios                                                         | Opcionales                    |
| --------------------- | -------------------------------------------------------------------- | ----------------------------- |
| `POST /contracts`     | `externalId`, `staffRut`, `contractType`, `startDate`, `baseSalary` (> 0), `projectCode` | `endDate` (>= `startDate`)    |
| `POST /terminations`  | `externalId`, `contractReferenceNumber`, `terminationDate`, `legalCause` | `definitive`                  |
| `POST .../annulment`  | `reason`                                                             | —                             |
| `POST /annexes`       | `externalId`, `contractReferenceNumber`, `annexType`, `effectiveDate` | `newBaseSalary` (>= 0)        |

### 5.6 Endpoints auxiliares (sin API key salvo `/admin`)

- `GET /health` → 200 `{ "status": "UP" }` (sin API key; lo usa Docker).
- `GET /admin/simulation` → estado actual de la simulación.
- `PUT /admin/simulation` → reemplaza el estado completo (los campos omitidos quedan en 0/false), body `{ "latencyMs": 2000, "failureRate": 0.3, "forceError": false }`.
  Cambia la simulación **en caliente**, sin reiniciar — sirve para demostrar en vivo cómo reacciona
  `rrhh-server` (estado `SYNC_ERROR`). Requiere `X-API-Key`.

## 6. Reglas de negocio simuladas

1. **Referencias secuenciales** por tipo, con formato `REX-%06d`, `REX-TERM-%06d`, `REX-ANX-%06d`
   (reemplazan los `System.currentTimeMillis()` del stub Java, que no eran reproducibles).
2. **Idempotencia por `externalId`**: repetir un POST con el mismo `externalId` y tipo devuelve la
   misma referencia con 200 (no 201) y no crea otra. Motivo: `rrhh-server` puede reintentar tras un
   timeout sin duplicar el registro en el ERP.
3. **Integridad referencial mínima**: terminación y anexo exigen un contrato ya sincronizado.
4. **Anulación idempotente**: anular dos veces la misma terminación devuelve 200 en ambas.
5. El estado vive en memoria; reiniciar el servicio lo borra (documentado en el README).

## 7. Errores

Un único formato, igual para todo error:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "texto en español", "fields": { "startDate": "es obligatoria" } } }
```

| Situación                          | HTTP | `code`              |
| ---------------------------------- | ---- | ------------------- |
| JSON inválido / campo faltante     | 400  | `VALIDATION_ERROR`  |
| API key ausente o incorrecta       | 401  | `UNAUTHORIZED`      |
| Referencia inexistente (anulación) | 404  | `NOT_FOUND`         |
| Contrato referenciado no existe    | 422  | `UNPROCESSABLE`     |
| Falla simulada                     | 503  | `REX_UNAVAILABLE`   |

Los mensajes van en español (los puede ver el usuario final vía `rrhh-server`); código, rutas y
campos en inglés.

## 8. Requisitos no funcionales

- Arranque < 1 s; imagen Docker final basada en `scratch` o `distroless` (< 20 MB).
- Concurrencia segura: repositorio y estado de simulación protegidos con `sync.RWMutex`.
- Apagado ordenado (`http.Server.Shutdown` con `SIGTERM`).
- Logs estructurados con `log/slog`, sin imprimir la API key.
- Cobertura de tests: service y repository con tests unitarios; endpoints con `httptest`; los
  middleware con tests propios. Todo criterio de aceptación (sección 10) tiene al menos un test.

## 9. Cambios asociados en `rrhh-server` (se hacen en ese repo, no acá; HU-B111, escrito el 2026-11-18, pendiente de verificar)

Variaciones respecto a lo propuesto originalmente: `annulTermination` pasó a recibir el motivo
(`annulTermination(termination, reason)`) porque el stub lo exige; `projectCode` sale del id del proyecto
y `annexType` del nombre del tipo de anexo, porque esas entidades no tienen `code`.

1. Nuevo adaptador `RexPlusHttpClient implements RexPlusClient` con `RestClient`, timeouts de
   conexión (2 s) y lectura (5 s), header `X-API-Key` desde propiedad.
2. Cualquier fallo de red, timeout o 5xx se traduce a `RexSyncResult.failure(mensaje)` — nunca se
   propaga una excepción, para que el flujo existente deje el registro en `SYNC_ERROR`.
3. Selección por propiedad `rexplus.mode=stub|http`; `RexPlusStubClient` se conserva para tests y
   para correr sin el servicio externo.
4. Propiedades nuevas: `rexplus.http.base-url`, `rexplus.http.api-key`, timeouts.

## 10. Criterios de aceptación

- **CA-1** `POST /contracts` válido con API key → 201 y `referenceNumber` con formato `REX-NNNNNN`.
- **CA-2** Sin API key o con una incorrecta → 401 con el formato de error de la sección 7.
- **CA-3** Mismo `externalId` dos veces → misma referencia, segundo status 200.
- **CA-4** `POST /terminations` con contrato inexistente → 422.
- **CA-5** `POST /terminations/{ref}/annulment` con `{ref}` inexistente → 404; repetida → 200 ambas.
- **CA-6** Con `SIM_FORCE_ERROR=true` los endpoints de negocio responden 503; `/health` sigue en 200.
- **CA-7** `PUT /admin/simulation` cambia el comportamiento sin reiniciar.
- **CA-8** Body con campos faltantes → 400 con `fields` listando cada campo (todos, no solo el primero).
- **CA-9** `docker compose up` deja el servicio sano en `/health` y `go test ./...` pasa.
- **CA-10** Con `rrhh-server` en `rexplus.mode=http`, aprobar un contrato como RRHH deja el contrato
  `ACTIVE` con `rexContractNumber` del stub; con el stub apagado queda en `SYNC_ERROR`.

## 11. Decisiones tomadas (y por qué)

- **Go sobre Flask**: reutiliza la estructura de `Solicitudes-documentos-Go` y suma un segundo
  lenguaje al sistema, que es el argumento de "integración con servicios externos" en la defensa.
- **Solo stdlib**: menos dependencias que mantener o auditar; el mux de Go 1.22+ cubre el ruteo.
- **Sin BD**: un ERP simulado no justifica persistencia; la idempotencia en memoria basta.
- **`/admin/simulation`**: la demo en vivo de un fallo es más convincente que reiniciar con otra
  variable de entorno.

## 12. Preguntas abiertas

- ¿Los payloads de 5.1–5.4 deben incluir más campos del `Contract`/`Annex` real (jornada, sitio, cargo)
  o basta con el subconjunto listado? Propuesta: subconjunto, y ampliar solo si `rrhh-server` lo necesita.
