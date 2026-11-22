# Checklist de pruebas en navegador (antes de la demo)

Cuentas: `promaty.admin`, `promaty.obra`, `promaty.rrhh` (@gmail.com). Orden pensado para que un fallo
temprano te deje tiempo de arreglarlo. Marca cada paso.

## 0. Antes de empezar

- [ ] `rrhh-server`: `mvn test` en verde (comando en el resumen de la sesión).
- [ ] Respaldo creado: `git tag backup/pre-rex-http; git branch backup/pre-rex-http-branch`
      (corre esto **antes** de commitear los cambios de la HU-B111).
- [ ] Anotado `git log -1` de cada repo.

## A. Regresión con el stub Java (`REXPLUS_MODE` sin definir o `stub`)

- [ ] Login con las 3 cuentas; cada una ve solo lo que su rol permite.
- [ ] Crear un colaborador completo (catálogos, adjuntos); aparece en el listado; ordenar por 2 columnas; editarlo.
- [ ] Contrato como `obra` → aprobar nivel 1 (jefatura) → aprobar nivel 2 (`rrhh`) → `ACTIVE` con `REX-…`.
- [ ] Anexo sobre ese contrato → validar como RRHH → `ACTIVE` con `REX-ANX-…`.
- [ ] Finiquito → aprobar ambos niveles → `REX-TERM-…`, contrato cerrado, colaborador bloqueado.
- [ ] Anular el finiquito (pide motivo) → contrato y colaborador vuelven a su estado previo.
- [ ] Rechazar una solicitud en cada nivel y corregir la rechazada; adjuntar y descargar un archivo.

## B. Integración real (`REXPLUS_MODE=http`)

Levantar `rex-stub` y reiniciar `rrhh-server` con las mismas claves:

```powershell
$env:API_KEY = "dev-key"; go run ./cmd
```

```powershell
$env:REXPLUS_MODE = "http"; $env:REXPLUS_API_KEY = "dev-key"
```

- [ ] Repetir contrato → anexo → finiquito → anulación con datos nuevos. Las referencias salen
      secuenciales: `REX-000001`, `REX-ANX-000001`, `REX-TERM-000001`, y coinciden con el log de `rex-stub`.
- [ ] Falla simulada: `forceError:true` → aprobar contrato como RRHH → queda `SYNC_ERROR` con mensaje,
      sin pantalla de error. Con `forceError:false` y reintentando → `ACTIVE`.
- [ ] Rex+ caído: detener `rex-stub` → aprobar contrato → `SYNC_ERROR` en ~2 s, nunca un 500.
- [ ] Latencia 3000 ms → tarda ~3 s y termina bien. Latencia 7000 ms → `SYNC_ERROR` por timeout.

Cambiar la simulación sin reiniciar:

```powershell
curl.exe -X PUT http://localhost:8090/admin/simulation -H "X-API-Key: dev-key" -H "Content-Type: application/json" -d '{\"forceError\":true}'
```

## C. Ensayo del rollback (el más importante)

- [ ] Quitar `REXPLUS_MODE` (o ponerlo en `stub`), reiniciar `rrhh-server` → aprobar un contrato → funciona sin `rex-stub`.
- [ ] Decidir y anotar en papel qué modo usarás en la demo. Si algo de B falló y no lo puedes explicar:
      demo en modo `stub` y muestra el adaptador HTTP como código y tests.

## D. Mañana de la demo (3 min)

- [ ] `GET http://localhost:8090/health` → `UP`.
- [ ] Un contrato de prueba de punta a punta.
- [ ] `rex-stub` corriendo y el modo elegido ya configurado.
