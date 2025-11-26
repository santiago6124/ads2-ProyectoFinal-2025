# Database Seeds

Este directorio contiene los archivos de seed para inicializar datos en la base de datos.

## Usuario Administrador

**Archivo**: `001_admin_user.sql`

### Credenciales por defecto:
- **Email**: `admin@cryptosim.com`
- **Username**: `admin`
- **Password**: `admin1234`
- **Role**: `admin`
- **Initial Balance**: `$100,000.00`

## Cómo ejecutar las seeds

### Opción 1: Usando Docker Compose

```bash
# Desde el directorio raíz del proyecto
docker-compose exec users-db mysql -u cryptosim_user -pcryptosim_password cryptosim_users < users-api/seeds/001_admin_user.sql
```

### Opción 2: Usando el comando Go

```bash
# Desde el directorio users-api
go run cmd/seed/main.go
```

### Opción 3: Manualmente con MySQL CLI

```bash
mysql -h localhost -P 3306 -u cryptosim_user -pcryptosim_password cryptosim_users < users-api/seeds/001_admin_user.sql
```

## Verificar que el usuario fue creado

```sql
SELECT id, username, email, role, is_active, created_at
FROM users
WHERE email = 'admin@cryptosim.com';
```

## Nota de Seguridad

⚠️ **IMPORTANTE**: Este usuario y contraseña son solo para desarrollo. En producción:
1. Cambiar la contraseña inmediatamente
2. Usar credenciales seguras
3. Considerar autenticación de dos factores
4. Limitar accesos desde IPs específicas
