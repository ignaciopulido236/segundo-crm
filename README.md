# PrimerCRM — Windows + Google Sheets

CRM de escritorio para Windows 10/11 x64. Go + Wails v2, React + TypeScript + Vite. La única persistencia de clientes es la API de Google Apps Script existente: este proyecto no crea una base de datos ni modifica Apps Script.

## Descargar una compilación

- **Código fuente:** [repositorio segundo-crm](https://github.com/ignaciopulido236/segundo-crm). El botón Code / Download ZIP descarga código, **no una aplicación instalable**.
- **Ejecutable de cada build:** [Actions](https://github.com/ignaciopulido236/segundo-crm/actions) → ejecución verde de “Windows build and test” → Artifacts → `PrimerCRM-windows-amd64`. Descomprime el ZIP y abre `PrimerCRM.exe`. Puede requerir iniciar sesión en GitHub; los artifacts expiran según la retención de GitHub.
- **Versiones publicadas:** [Releases](https://github.com/ignaciopulido236/segundo-crm/releases) → Assets → `PrimerCRM.exe`. Se adjunta automáticamente cuando se publica un tag `v*`, después de superar las pruebas.
- Compilación local: `build\bin\PrimerCRM.exe`. Los ejecutables y `frontend/dist` están excluidos de Git. El binario no está firmado; Windows puede mostrar SmartScreen.

## Requisitos y versiones

- Windows 10/11 de 64 bits y [Microsoft Edge WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/). Es necesario también para ejecutar el binario. `wails doctor` comprueba el entorno.
- Go **1.25.1** para igualar CI (el proyecto requiere 1.25.0 o posterior).
- Node.js **22.14.0** para igualar CI y npm. Vite 7 requiere Node 20.19+ o 22.12+.
- Wails **v2.11.0**, React **19.1.1**, TypeScript **5.9.2**, Vite **7.3.7**, fijados junto con `go.sum` y `frontend/package-lock.json`.
- Git. Para `go test -race`, compilador C de 64 bits (por ejemplo MinGW-w64 con GCC en PATH). La compilación Windows normal usa Go/WebView2 sin requerir GCC.

Compatibilidad verificada antes de fijar versiones: [instalación oficial de Wails v2](https://v2.wails.io/docs/gettingstarted/installation/), [go.mod oficial de v2.11.0](https://github.com/wailsapp/wails/blob/v2.11.0/v2/go.mod) (Go 1.22.0 mínimo), [requisitos de Vite](https://vite.dev/guide/) y los metadatos `engines` de `vite@7.3.7`. No se usa Wails v3 beta.

## Ejecutar en desarrollo — PowerShell

```powershell
git clone https://github.com/ignaciopulido236/segundo-crm.git
cd segundo-crm
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
$env:Path += ";$(go env GOPATH)\bin"
go mod download
Push-Location frontend
npm ci
Pop-Location
wails doctor
wails dev
```

No uses `npm run dev` como sustituto del escritorio: sin Wails no existe el puente Go y no se puede conectar a Google. No hay una API mock en la aplicación de producción.

## Probar y compilar — PowerShell

```powershell
Push-Location frontend
npm ci
npm run check
npm test
npm run build
Pop-Location
go test -race ./...
go vet ./...
wails build -clean -platform windows/amd64 -o PrimerCRM.exe
.\build\bin\PrimerCRM.exe
```

CI ejecuta esos controles en Windows e incluye las pruebas DPAPI reales. Las pruebas Go usan servidores/transportes simulados; las pruebas React inyectan un puente simulado. **Ninguna prueba accede ni modifica una hoja real.** Para publicar una versión desde un commit verificado:

```powershell
git tag v1.0.0
git push origin v1.0.0
```

## Configurar credenciales

La URL inicial está vacía porque no se proporcionó en la especificación. En el primer inicio introduce una URL `https://script.google.com/macros/s/ID/exec` y una API key autorizada. La aplicación hace una operación `list` de lectura y verifica el contrato antes de guardar. En Configuración puedes reemplazar ambas; vuelve a introducir la clave incluso al cambiar solamente la URL. Una configuración fallida conserva la conexión anterior.

Se guarda exclusivamente URL + clave cifrada por **Windows DPAPI CurrentUser** en `%APPDATA%\PrimerCRM\connection.json`, fuera del repositorio. El archivo temporal se escribe, sincroniza y sustituye atómicamente con `MoveFileExW(REPLACE_EXISTING | WRITE_THROUGH)` en la misma carpeta. Si no se puede descifrar o leer se solicita reconfiguración. No hay fallback de texto plano, cifrado de máquina ni persistencia de clientes.

La clave introducida se envía una vez de React a Go y se limpia del formulario tras verificar. No existe getter que devuelva la clave o el ciphertext; el DTO público contiene `url`, `configured` y, si aplica, un diagnóstico sin secretos. Go mantiene la clave descifrada en memoria para las solicitudes. El formulario de configuración necesariamente tiene la clave mientras el usuario la escribe; no se guarda en localStorage, URLs, logs ni assets. DPAPI **no protege frente a procesos que controlen la misma cuenta de Windows**, ni frente a un equipo ya comprometido.

## Comportamiento HTTP y recuperación

Todas las operaciones iniciales son POST JSON sobre HTTPS, con `api_key` en el cuerpo. El tiempo máximo total es 45 segundos, incluidas redirecciones. Sólo se admite como URL inicial el host exacto `script.google.com` y ruta de implementación `/macros/s/ID/exec`, sin credenciales, query ni fragmento. Sólo se siguen redirecciones HTTPS al host exacto `script.googleusercontent.com`, ruta `/macros/echo`, en GET sin cuerpo, máximo cinco saltos. Las redirecciones 302/303 usan el comportamiento de Go: POST → GET; el cuerpo con la clave no se reenvía. 307/308 que intenten reenviar POST se bloquean.

Se comprueban HTTP, JSON, `ok`, `status`, estructura de `data` y metadatos de paginación. Las páginas HTML de Google se distinguen de errores JSON o de transporte. Se muestran errores seguros predefinidos, nunca mensajes arbitrarios del servidor que pudieran repetir la clave. No hay reintentos automáticos de escrituras. Ante pérdida de respuesta, fallo HTTP/HTML o contrato de escritura inválido, el mensaje indica **resultado incierto**: actualiza y consulta antes de repetir; duplicar una creación podría producir dos registros si el servidor ya la ejecutó.

La búsqueda se realiza al pulsar Enter/Buscar, no por pulsación. Se ignoran resultados antiguos mediante generaciones de solicitudes; las lecturas anteriores pueden terminar pero no sustituyen resultados recientes. Los totales y límites provienen de `meta`, no se extrapolan desde la página. Se consulta `get` antes de ver y de editar. Update envía únicamente campos modificados: omitir conserva; `""` vacía un campo opcional. Teléfonos siempre son cadenas. Los formularios se conservan ante errores; botones y bloqueo de envío evitan duplicados pendientes. Un guardado confirmado sigue mostrando éxito aunque falle el refresco posterior.

## Estructura y pruebas

- `internal/crm`: contrato HTTP, validación, almacenamiento y DPAPI con build tags Windows.
- `app.go`: única superficie enlazada a React; no getters de credenciales.
- `frontend/src`: UI, puente TypeScript y pruebas de integración React Testing Library/Vitest.
- `.github/workflows/windows.yml`: pruebas, análisis, frontend, build Wails, artifact y Release por tag.

Cobertura funcional: CRUD, contrato POST, HTTP 200 con error JSON, HTML, HTTP fallido, redirecciones sin reenviar credenciales, validaciones y límites Unicode, update parcial, paginación, configuración pública sin secretos, reemplazo atómico, DPAPI ida/vuelta y corrupción, búsqueda/filtros, descarte y conservación de formularios, doble envío, error de refresco posterior y respuestas antiguas.

La verificación de una conexión real requiere la URL y clave autorizadas. Los tests simulados no prueban la disponibilidad, permisos, rendimiento o datos de una implementación real de Apps Script. La prueba visual nativa, WebView2 y DPAPI debe realizarse en Windows; una compilación cruzada desde Linux no ejecuta esas funciones. La API no proporciona control de versiones/ETag: dos usuarios podrían sobrescribir cambios concurrentes en los mismos campos; consultar antes de editar reduce información obsoleta pero no elimina esa limitación.
