# 🛡️ Especificación Técnica y Arquitectónica de la API REST - Viking-App

---

## 1. Visión General y Arquitectura

La API REST de **Viking-App** es el núcleo transaccional e integrador del ecosistema **El Vikingo Store**. Diseñada bajo estándares modernos de arquitectura limpia y microservicios sin estado (*stateless*), esta API proporciona un punto de acceso seguro, escalable y robusto para clientes web (React/Next.js/Vite) y aplicaciones móviles (React Native / Expo).

### ⚙️ Stack Tecnológico Principal
* **Framework Core:** Spring Boot 3.3.3 (Java 17).
* **Persistencia:** JPA / Hibernate ORM con base de datos relacional PostgreSQL.
* **Seguridad:** Spring Security y JSON Web Tokens (JWT) sin estado usando la biblioteca `jjwt-api` y `jose4j`.
* **Documentación y Contrato API:** OpenAPI 3.0.1 (generado dinámicamente mediante `springdoc-openapi`).
* **Manejo Multimedia:** Carga de archivos y servicio estático de imágenes para evidencias de diagnóstico e historial de reparaciones (`multipart/form-data` hasta 10MB).

> [!IMPORTANT]
> **Archivo de Especificación OpenAPI:**  
> Puedes acceder a la especificación oficial en formato YAML desde el siguiente enlace para importarla directamente a tus colecciones de **Postman**, **Insomnia**, **Bruno** o el editor en línea de **Swagger UI**:  
> 👉 [openapi_spec.yaml](file:///home/d3xtro/.gemini/antigravity-ide/brain/1b206a3b-137f-4c45-a680-8193aa99739e/openapi_spec.yaml)

---

## 2. Flujo de Autenticación y Seguridad (El "Por qué" Arquitectónico)

### ¿Por qué JWT Stateless (Sin Estado)?
En una arquitectura moderna de aplicaciones multiplataforma (Web + Móvil), mantener sesiones en memoria o en servidor (*Stateful*) limita drásticamente el escalamiento horizontal e introduce acoplamiento entre el balanceador de carga y las instancias de backend.  
Adoptamos **JWT Stateless** porque permite que la propia firma criptográfica (HMAC-SHA con clave secreta en servidor) valide la identidad y los roles del usuario de manera autónoma en cada petición, reduciendo la latencia de consultas reiteradas a la base de datos en endpoints críticos.

```mermaid
sequenceDiagram
    autonumber
    actor Cliente as Cliente (Web/App)
    participant Auth as Auth Controller (/auth)
    participant Sec as Spring Security Filter
    participant API as API Controllers (/api/*)
    participant DB as PostgreSQL DB

    Note over Cliente, Auth: 1. Flujo de Identificación (Login)
    Cliente->>Auth: POST /auth/login {email, password}
    Auth->>DB: Validar credenciales y roles
    DB-->>Auth: Datos de Usuario OK
    Auth-->>Cliente: 200 OK + Token JWT Firmado

    Note over Cliente, API: 2. Flujo de Peticiones Autorizadas
    Cliente->>Sec: GET /api/work-order/search (Header: Authorization Bearer <token>)
    Sec->>Sec: Validar firma criptográfica y expiración
    alt Token Inválido o Expirado
        Sec-->>Cliente: 401 Unauthorized / 403 Forbidden
    else Token Válido
        Sec->>API: Permitir paso al Controlador
        API->>DB: Ejecutar consulta transaccional
        DB-->>API: Retornar entidades
        API-->>Cliente: 200 OK (JSON Response)
    end
```

### 🔑 Instrucciones de Autenticación para Developers
1. **Solicitud de Token:** Envía una petición `POST` a `/auth/login` con el JSON conteniendo tu `email` y `password`.
2. **Almacenamiento:** El backend responderá con un objeto que contiene el token de acceso. Almacénalo de forma segura en tu aplicación (ej. *SecureStore* en React Native o *HttpOnly/Memory* en Web).
3. **Inyección en Cabeceras:** Para cada petición hacia los endpoints protegidos bajo `/api/**` o `/auth/roles/**`, debes incluir obligatoriamente el siguiente encabezado HTTP:
   ```http
   Authorization: Bearer eyJhbGciOiJIUzUxMiJ9.eyJzdWIiOiJleGFtcGxlQGV4Y...
   ```
4. **Validación de Token:** Puedes consultar la validez o expiración de un token de forma ligera llamando a `GET /auth/validate` antes de iniciar flujos costosos en la UI.

---

## 3. Catálogo Técnico de Módulos y Endpoints

La API está dividida de forma modular en **9 controladores principales**, reflejando las fronteras del dominio del negocio de reparaciones y servicio técnico.

### 🛡️ Módulo 1: Auth Controller (Autenticación y Registro)
Responsable del punto de entrada público, emisión de tokens JWT y validación de sesiones en el cliente.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `POST` | `/auth/login` | **Iniciar Sesión:** Autentica credenciales y emite el token JWT de acceso. | *Público* |
| `POST` | `/auth/signup` | **Registrar Usuario:** Registra un nuevo usuario o cliente en el sistema. | *Público / Staff* |
| `GET` | `/auth/validate` | **Validar Token:** Verifica si el JWT en cabecera sigue activo y firmado. | *Bearer Token* |

---

### 👥 Módulo 2: User Controller (Gestión de Usuarios)
Permite la administración de perfiles de clientes, técnicos (*staff*) y administradores, incluyendo búsquedas dinámicas y mantenimiento de datos.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `GET` | `/api/user/current` | **Usuario Actual:** Obtiene la información detallada del usuario autenticado. | *Authenticated* |
| `GET` | `/api/user/search` | **Buscar Usuarios:** Búsqueda multicriterio por `id`, `dni`, `email`, `roleId` o texto libre (`query`). | *Staff / Admin* |
| `POST` | `/api/user/save` | **Crear Usuario:** Crea un nuevo perfil en la base de datos de forma administrativa. | *Admin / Staff* |
| `PUT` | `/api/user/update/{id}`| **Actualizar Usuario:** Modifica datos personales, teléfonos, dirección o rol de un usuario. | *Admin / Staff* |
| `DELETE`| `/api/user/delete/{id}`| **Eliminar Usuario:** da de baja o elimina un usuario del sistema por ID UUID. | *Admin* |

---

### 👔 Módulo 3: Role Controller (Gestión y Catálogo de Roles)
Módulo encargado de definir las jerarquías y permisos granulares dentro de la plataforma (ej. `ADMIN`, `STAFF`, `CLIENTE`).

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `GET` | `/auth/roles` | **Listar Roles:** Retorna la lista completa de roles disponibles en el sistema. | *Authenticated* |
| `GET` | `/auth/roles/{id}` | **Obtener Rol por ID:** Consulta los detalles y permisos de un rol específico. | *Authenticated* |
| `POST` | `/auth/roles` | **Crear Rol:** Define una nueva jerarquía con su permiso asociado (`permission`). | *Admin* |
| `PUT` | `/auth/roles/{id}` | **Actualizar Rol:** Modifica la descripción o permisos de un rol existente. | *Admin* |
| `DELETE`| `/auth/roles/{id}` | **Eliminar Rol:** Elimina un rol del catálogo si no posee usuarios activos. | *Admin* |

---

### 🚦 Módulo 4: User Role Controller (Inspección Rápida de Permisos)
Endpoints optimizados para la interfaz de usuario para verificar de manera instantánea si el usuario cuenta con elevación de privilegios sin consultar objetos pesados.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `GET` | `/api/user-roles/is-staff` | **Es Staff?:** Retorna un booleano (`true`/`false`) indicando si el usuario es técnico. | *Authenticated* |
| `GET` | `/api/user-roles/user-permission` | **Permiso Admin:** Verifica el nivel de acceso administrativo del token actual. | *Authenticated* |

---

### 📱 Módulo 5: Device Controller (Gestión de Dispositivos y Equipos)
Administra el inventario de equipos tecnológicos (teléfonos, tablets, consolas, PCs) ingresados por los clientes para reparación.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `POST` | `/api/device/save` | **Registrar Dispositivo:** Vincula un nuevo equipo (`brand`, `model`, `serialNumber`) a un cliente (`userId`). | *Staff / Admin* |
| `GET` | `/api/device/search` | **Buscar Dispositivos:** Filtra equipos por `id`, `serialNumber`, `brand`, `userDni` o `query`. | *Staff / Admin* |
| `PUT` | `/api/device/update/{id}`| **Actualizar Dispositivo:** Modifica características técnicas o asignación de propietario. | *Staff / Admin* |
| `DELETE`| `/api/device/delete/{id}`| **Eliminar Dispositivo:** Remueve un equipo del registro del sistema. | *Admin* |

---

### 🔧 Módulo 6: Work Order Controller (Órdenes de Trabajo)
El corazón operativo del taller técnico. Gestiona el ciclo de vida completo de las reparaciones, diagnósticos iniciales y cambios de estado.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `POST` | `/api/work-order/save` | **Crear Orden de Trabajo:** Genera una nueva orden uniendo `clientId`, `deviceId`, descripción del problema y estado inicial. | *Staff / Admin* |
| `GET` | `/api/work-order/search` | **Buscar Órdenes:** Permite rastrear reparaciones por `staffId`, `clientDni`, `deviceSerialNumber` o término general. | *Staff / Admin* |
| `PATCH`| `/api/work-order/update/{orderId}` | **Actualizar Estado:** Cambia el estado de reparación de una orden (`RECEIVED` -> `IN_PROGRESS` -> `DONE`). | *Staff / Admin* |
| `DELETE`| `/api/work-order/delete/{orderId}` | **Eliminar Orden:** Borra o cancela una orden de trabajo por UUID. | *Admin* |

---

### 📸 Módulo 7: Diagnostic Point Controller (Puntos de Diagnóstico y Evidencias)
Permite a los técnicos adjuntar notas técnicas y fotografías de evidencia (ej. estado en que llegó el equipo o pieza dañada) a una orden de trabajo.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `POST` | `/api/diagnostic-points/add` | **Agregar Punto de Diagnóstico:** Envío `multipart/form-data` combinando JSON/Texto (`diagnosticPoint`) y archivo de imagen (`file`). | *Staff / Admin* |
| `GET` | `/api/diagnostic-points/by-work-order/{workOrderId}/client/{clientId}` | **Historial por Orden:** Consulta todos los diagnósticos y fotos adjuntas de una orden específica de un cliente. | *Authenticated* |

---

### 📂 Módulo 8: File Controller (Servidor de Archivos / Uploads)
Módulo encargado de servir de manera estática y pública/protegida los recursos multimedia subidos al servidor (imágenes, reportes PDF, etc.).

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `GET` | `/auth/uploads/{filename}` | **Descargar / Ver Archivo:** Retorna el stream binario (`format: binary`) de la imagen o documento almacenado. | *Público / Auth* |

---

### 🏠 Módulo 9: Home Controller (Health Check y Diagnóstico de Red)
Punto de prueba simple para verificar latencia y disponibilidad de la API REST desde balanceadores de carga o clientes.

| Método | Endpoint | Operación / Descripción | Roles / Permisos |
| :---: | :--- | :--- | :---: |
| `GET` | `/api/` | **Greeting Check:** Responde un mensaje de bienvenida confirmando que el servidor Spring Boot está activo. | *Público* |

---

## 4. Estructuras de Datos y Modelos (DTOs)

Para facilitar la integración en el frontend, se detallan los esquemas JSON de las estructuras de petición más utilizadas:

### 📦 Creación de Orden de Trabajo (`WorkOrderCreateRequest`)
Este objeto define el contrato al recibir un equipo en el taller. Nótese el uso del enumerador estricto para el estado (`repairStatus`):

```json
{
  "clientId": "123e4567-e89b-12d3-a456-426614174000",
  "deviceId": "987fcdeb-51a2-43d1-9876-543210987654",
  "issueDescription": "El equipo no enciende, presenta olor a quemado cerca del puerto de carga.",
  "repairStatus": "RECEIVED"
}
```

> [!TIP]
> **Valores Válidos para `repairStatus` (Enum):**
> * `RECEIVED`: Equipo recibido en mostrador, esperando diagnóstico.
> * `IN_QUEUE`: En cola de espera asignado al área técnica.
> * `IN_PROGRESS`: Técnico trabajando activamente en la reparación.
> * `DONE`: Reparación finalizada con éxito, listo para retiro.
> * `WITHDRAWN`: Equipo retirado y entregado al cliente (Ciclo cerrado).

---

### 👤 Registro de Nuevo Usuario / Cliente (`RegisterDto` / `UserCreateRequestDto`)
Contrato para la creación de cuentas de usuario, validado en servidor con anotaciones de `jakarta.validation`:

```json
{
  "name": "Juan Pérez",
  "dni": 35444555,
  "address": "Av. Valparaíso 1234, Viña del Mar",
  "phoneNumber": "+56912345678",
  "secondaryPhoneNumber": "+56987654321",
  "email": "juan.perez@example.com",
  "password": "PasswordSegura123!",
  "roleId": "3383101c-cb82-48d8-baaa-39cc0f3628bf"
}
```

---

### 📱 Registro de Dispositivo (`DeviceCreateRequestDto`)
Estructura para asociar un hardware en la base de datos de inventario del cliente:

```json
{
  "type": "SMARTPHONE",
  "brand": "Samsung",
  "model": "Galaxy S23 Ultra",
  "serialNumber": "SN-998877665544-X",
  "userId": "123e4567-e89b-12d3-a456-426614174000"
}
```

---

## 5. Buenas Prácticas y Recomendaciones de Consumo (Para el Equipo)

Para garantizar que el frontend (Web/Expo) y otros consumidores interactúen de manera óptima con la API, sigan estas directrices:

1. **Manejo de Expiración del Token:**
   El token JWT tiene un tiempo de expiración configurado por defecto en **1 hora** (`3600000 ms`). Si una llamada interceptada en React o Axios devuelve un código HTTP `401 Unauthorized` o `403 Forbidden`, el cliente debe redirigir inmediatamente a la pantalla de Login o solicitar un nuevo inicio de sesión, evitando reintentos infinitos.

2. **Carga de Evidencias Multimedia (`multipart/form-data`):**
   Al consumir el endpoint `/api/diagnostic-points/add`, no se debe enviar una cabecera de `Content-Type: application/json`. El navegador o framework móvil debe autoconfigurar la cabecera con el *boundary* adecuado para envíos multipart:
   ```javascript
   const formData = new FormData();
   formData.append('diagnosticPoint', 'Se adjunta fotografía del flex dañado.');
   formData.append('file', {
     uri: imageUri,
     type: 'image/jpeg',
     name: 'evidencia_01.jpg'
   });

   await axios.post('http://localhost:8080/api/diagnostic-points/add', formData, {
     headers: {
       'Authorization': `Bearer ${token}`,
       'Content-Type': 'multipart/form-data'
     }
   });
   ```

3. **Optimización en Endpoints de Búsqueda:**
   Los endpoints `/search` (como `/api/work-order/search` o `/api/device/search`) aceptan parámetros opcionales en la query de la URL (ej. `?query=Samsung&clientDni=35444555`). En el frontend, utilicen técnicas de **Debounce** (250-500ms) al tipar en los inputs de búsqueda para evitar saturar el servidor PostgreSQL con peticiones en cada tecla presionado.
