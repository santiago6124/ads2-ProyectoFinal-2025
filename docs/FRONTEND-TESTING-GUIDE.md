# Frontend Testing Guide - Admin Panel & Balance Features

Este documento describe las pruebas funcionales que deben realizarse manualmente en el frontend para validar las nuevas funcionalidades implementadas.

## Pre-requisitos

1. **Todos los servicios corriendo**: `docker-compose up -d`
2. **Frontend accesible**: http://localhost:3000
3. **Usuario admin disponible**: testadmin / admin1234
4. **Usuario normal disponible**: Cualquier usuario registrado

## Checklist de Pruebas Funcionales

### 1. Visualización de Balance Actual

**Objetivo**: Verificar que el balance actual del usuario se muestre correctamente en el sidebar.

**Pasos**:
1. Iniciar sesión con cualquier usuario
2. Observar el sidebar (desktop) o el menú (móvil)
3. Verificar que se muestre el balance actual del usuario en verde
4. El formato debe ser: `$XX,XXX.XX`

**Resultado Esperado**:
- ✅ El balance mostrado corresponde al `current_balance` del usuario
- ✅ Se actualiza cuando el usuario navega entre páginas
- ✅ El formato numérico es correcto (con comas y 2 decimales)

---

### 2. Acceso al Panel de Administración (Usuario Normal)

**Objetivo**: Verificar que usuarios normales NO puedan acceder al panel admin.

**Pasos**:
1. Iniciar sesión con un usuario normal (NO admin)
2. Observar el menú de navegación en sidebar
3. Intentar acceder manualmente a http://localhost:3000/admin

**Resultado Esperado**:
- ✅ NO aparece el botón "Admin Panel" en el sidebar
- ✅ Al intentar acceder a `/admin`, se redirige al dashboard
- ✅ No se muestran errores en consola del navegador

---

### 3. Acceso al Panel de Administración (Usuario Admin)

**Objetivo**: Verificar que usuarios admin puedan acceder al panel.

**Pasos**:
1. Iniciar sesión con usuario admin (testadmin / admin1234)
2. Observar el sidebar/menú de navegación
3. Hacer clic en "Admin Panel" (botón morado con ícono de escudo)

**Resultado Esperado**:
- ✅ Aparece botón "Admin Panel" con borde morado y ícono de escudo
- ✅ Al hacer clic, navega a `/admin`
- ✅ La página del panel admin carga correctamente
- ✅ El botón queda resaltado en morado cuando está activo

---

### 4. Listado de Usuarios en Panel Admin

**Objetivo**: Verificar que el panel admin muestra correctamente la lista de usuarios.

**Pasos**:
1. En el panel admin, observar la tabla de usuarios
2. Verificar las columnas: ID, Username, Email, Role, Current Balance, Status, Actions

**Resultado Esperado**:
- ✅ Se muestra la tabla con todos los usuarios registrados
- ✅ Los roles se muestran con badges de colores:
  - `admin` → badge morado
  - `normal` → badge azul
- ✅ El balance actual se muestra con formato `$XX,XXX`
- ✅ El status se muestra con badges:
  - `Active` → badge verde
  - `Inactive` → badge rojo
- ✅ Cada fila tiene 2 botones de acción ($ y 🗑️)

---

### 5. Modificación de Balance (Incremento)

**Objetivo**: Verificar que un admin pueda incrementar el balance de un usuario.

**Pasos**:
1. En el panel admin, identificar un usuario de prueba
2. Anotar el balance actual del usuario
3. Hacer clic en el botón del dólar ($) en la fila del usuario
4. En el diálogo que aparece:
   - Ingresar un monto positivo (ej: 5000)
   - Agregar descripción opcional (ej: "Bonus de prueba")
5. Hacer clic en "Update Balance"
6. Esperar la confirmación
7. Verificar el nuevo balance en la tabla

**Resultado Esperado**:
- ✅ El diálogo se abre mostrando el balance actual
- ✅ El campo "Amount" acepta números decimales
- ✅ Al confirmar, aparece mensaje de éxito
- ✅ El diálogo se cierra automáticamente
- ✅ La tabla se actualiza mostrando el nuevo balance
- ✅ Nuevo balance = Balance anterior + Monto ingresado

**Validación Adicional**:
- Verificar en la base de datos que se creó un registro en `balance_transactions`
```sql
SELECT * FROM balance_transactions WHERE user_id = <ID_USUARIO> ORDER BY created_at DESC LIMIT 1;
```

---

### 6. Modificación de Balance (Decremento)

**Objetivo**: Verificar que un admin pueda decrementar el balance de un usuario.

**Pasos**:
1. Seleccionar un usuario con balance > 0
2. Anotar el balance actual
3. Hacer clic en el botón del dólar ($)
4. Ingresar un monto NEGATIVO (ej: -1000)
5. Agregar descripción (ej: "Ajuste de balance")
6. Hacer clic en "Update Balance"
7. Verificar el nuevo balance

**Resultado Esperado**:
- ✅ El sistema acepta montos negativos
- ✅ El balance se reduce correctamente
- ✅ Nuevo balance = Balance anterior + Monto negativo
- ✅ Si el resultado sería negativo, debe mostrar error de "insufficient balance"

**Caso de Error**:
- Intentar restar más del balance disponible
- Debe mostrar error y NO actualizar el balance

---

### 7. Eliminación de Usuario (Soft Delete)

**Objetivo**: Verificar que un admin pueda desactivar usuarios.

**Pasos**:
1. En el panel admin, seleccionar un usuario de prueba (NO el admin actual)
2. Hacer clic en el botón de eliminar (🗑️)
3. Confirmar en el diálogo de confirmación
4. Observar la actualización de la tabla

**Resultado Esperado**:
- ✅ Aparece confirmación antes de eliminar
- ✅ Al confirmar, el usuario se marca como "Inactive"
- ✅ El badge de status cambia a rojo
- ✅ El usuario desaparece o se marca visualmente como inactivo
- ✅ El admin NO puede eliminarse a sí mismo (botón deshabilitado)

---

### 8. Actualización de Balance en Sidebar (Tiempo Real)

**Objetivo**: Verificar que el balance en el sidebar se actualiza después de modificaciones.

**Pasos**:
1. Iniciar sesión con un usuario normal
2. Anotar el balance mostrado en el sidebar
3. En otra pestaña/navegador, iniciar sesión como admin
4. Modificar el balance del usuario normal (+5000)
5. Volver a la pestaña del usuario normal
6. Refrescar la página o navegar entre secciones

**Resultado Esperado**:
- ✅ Después de refrescar, el balance en el sidebar refleja el nuevo valor
- ✅ El formato numérico se mantiene correcto
- ✅ No hay errores en consola

---

### 9. Responsividad del Panel Admin

**Objetivo**: Verificar que el panel admin funciona en dispositivos móviles.

**Pasos**:
1. Abrir DevTools y cambiar a vista móvil (ej: iPhone 12)
2. Iniciar sesión como admin
3. Navegar al panel admin
4. Intentar las operaciones básicas:
   - Ver lista de usuarios (scroll horizontal)
   - Abrir diálogo de balance
   - Modificar un balance

**Resultado Esperado**:
- ✅ La tabla es scrolleable horizontalmente
- ✅ Los diálogos se adaptan al tamaño de pantalla
- ✅ Los botones son táctiles y del tamaño adecuado
- ✅ El texto es legible sin zoom

---

### 10. Validaciones y Manejo de Errores

**Objetivo**: Verificar que el sistema maneja correctamente errores y validaciones.

**Pruebas a Realizar**:

#### A. Balance vacío
1. Abrir diálogo de modificación de balance
2. Dejar el campo "Amount" vacío
3. Intentar guardar

**Esperado**: Botón "Update Balance" está deshabilitado

#### B. Balance con caracteres no numéricos
1. Ingresar texto en el campo "Amount" (ej: "abc")
2. Intentar guardar

**Esperado**: El input type="number" previene caracteres no numéricos

#### C. Token expirado
1. Mantener sesión abierta por tiempo prolongado
2. Intentar modificar un balance

**Esperado**: Mensaje de error o redirección al login

#### D. Conexión con backend interrumpida
1. Detener el servicio users-api: `docker-compose stop users-api`
2. Intentar cargar el panel admin
3. Intentar modificar un balance

**Esperado**: Mensaje de error claro, sin crash del frontend

---

## Pruebas de Integración Backend-Frontend

### 11. Verificar Transacciones en Base de Datos

**Después de modificar balances**, verificar que se registren correctamente:

```sql
-- Conectar a MySQL
docker-compose exec users-mysql mysql -u root -proot users_db

-- Ver últimas transacciones
SELECT
    id,
    order_id,
    user_id,
    amount,
    transaction_type,
    previous_balance,
    new_balance,
    created_at
FROM balance_transactions
ORDER BY created_at DESC
LIMIT 10;

-- Ver balance actual de usuarios
SELECT
    id,
    username,
    email,
    role,
    initial_balance,
    current_balance,
    is_active
FROM users
ORDER BY id;
```

**Resultado Esperado**:
- ✅ Cada modificación de balance genera una transacción
- ✅ Los campos `previous_balance` y `new_balance` son correctos
- ✅ El `transaction_type` es "deposit" o "withdrawal" según el monto
- ✅ El `current_balance` del usuario coincide con la última transacción

---

## Checklist Rápido (Post-Validación)

Una vez completadas todas las pruebas anteriores, verificar:

- [ ] El panel admin es accesible solo para usuarios con role=admin
- [ ] La lista de usuarios se carga correctamente
- [ ] Se pueden incrementar balances con montos positivos
- [ ] Se pueden decrementar balances con montos negativos
- [ ] Se previenen balances negativos
- [ ] Se pueden desactivar usuarios (soft delete)
- [ ] El admin no puede eliminarse a sí mismo
- [ ] El balance en el sidebar muestra `current_balance`
- [ ] Las transacciones se registran en la base de datos
- [ ] El panel es responsivo en móviles
- [ ] Los errores se manejan correctamente

---

## Problemas Conocidos y Soluciones

### Problema: Frontend muestra "unhealthy"
**Solución**:
```bash
docker-compose restart frontend
docker-compose logs frontend
```

### Problema: No aparece el panel admin siendo usuario admin
**Solución**:
1. Verificar en Settings que el rol sea "admin"
2. Refrescar la página (F5)
3. Cerrar sesión y volver a iniciar

### Problema: Error 401 en llamadas API
**Solución**:
1. Verificar que el token JWT sea válido en localStorage
2. Cerrar sesión y volver a iniciar
3. Verificar que users-api esté corriendo: `docker-compose ps users-api`

### Problema: Balance no se actualiza después de modificación
**Solución**:
1. Verificar logs del backend: `docker-compose logs users-api`
2. Verificar que la transacción se creó en la DB
3. Refrescar el panel admin para recargar datos

---

## Contacto y Soporte

Para reportar problemas encontrados durante las pruebas, incluir:
1. Capturas de pantalla
2. Logs de consola del navegador (F12 → Console)
3. Logs del backend: `docker-compose logs users-api | tail -50`
4. Pasos exactos para reproducir el problema
