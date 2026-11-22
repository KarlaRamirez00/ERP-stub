# rex-stub

Simulador externo del ERP de RRHH **Rex+** para Promaty. Servicio HTTP en Go (solo librería estándar)
que `rrhh-server` consume como si fuera un sistema de terceros. Contrato y criterios de aceptación en
[`docs/spec.md`](docs/spec.md).

> El estado vive en memoria: reiniciar el servicio borra las referencias asignadas.

## Ejecutar

Con Go (PowerShell):

```powershell
$env:API_KEY = "dev-key"; go run ./cmd
```

Con Docker:

```powershell
$env:API_KEY = "dev-key"; docker compose up --build
```

## Variables de entorno

| Variable           | Default       | Descripción                                    |
| ------------------ | ------------- | ---------------------------------------------- |
| `PORT`             | `8090`        | Puerto HTTP                                    |
| `API_KEY`          | (obligatoria) | Valor esperado en el header `X-API-Key`        |
| `SIM_LATENCY_MS`   | `0`           | Latencia artificial por request                |
| `SIM_FAILURE_RATE` | `0`           | Probabilidad (0–1) de responder 503            |
| `SIM_FORCE_ERROR`  | `false`       | Si es `true`, todo endpoint de negocio da 503  |

## Endpoints

| Método y ruta                                   | Respuesta                                       |
| ----------------------------------------------- | ----------------------------------------------- |
| `POST /api/v1/contracts`                        | 201 `{"referenceNumber":"REX-000001"}`          |
| `POST /api/v1/terminations`                     | 201 `{"referenceNumber":"REX-TERM-000001"}`     |
| `POST /api/v1/terminations/{ref}/annulment`     | 200 con la misma referencia                     |
| `POST /api/v1/annexes`                          | 201 `{"referenceNumber":"REX-ANX-000001"}`      |
| `GET /health`                                   | 200 `{"status":"UP"}` (sin API key)             |
| `GET` / `PUT /admin/simulation`                 | Ver y cambiar la simulación en caliente         |

Repetir un POST con el mismo `externalId` devuelve la misma referencia con 200.

## Ejemplos

```powershell
curl.exe -X POST http://localhost:8090/api/v1/contracts -H "X-API-Key: dev-key" -H "Content-Type: application/json" -d '{\"externalId\":\"1\",\"staffRut\":\"12.345.678-9\",\"contractType\":\"O\",\"startDate\":\"2026-12-01\",\"baseSalary\":850000,\"projectCode\":\"PRJ-01\"}'
```

Simular que Rex+ está caído, sin reiniciar:

```powershell
curl.exe -X PUT http://localhost:8090/admin/simulation -H "X-API-Key: dev-key" -H "Content-Type: application/json" -d '{\"forceError\":true}'
```

## Tests

```powershell
go vet ./...; go test ./...
```
