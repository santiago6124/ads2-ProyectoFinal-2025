# 📚 Documentación Legacy

Esta carpeta contiene la documentación detallada original de cada microservicio.

## ⚠️ Nota Importante

**Esta documentación está archivada como referencia histórica.** Para información actualizada sobre cómo usar los servicios, consulta:

- **[README Principal](../../README.md)** - Documentación completa del proyecto
- **[QUICKSTART](../../QUICKSTART.md)** - Guía de inicio rápido
- **READMEs individuales** en cada carpeta de microservicio

## 📄 Archivos en esta carpeta

| Archivo | Descripción |
|---------|-------------|
| [cryptosim-readme.md](cryptosim-readme.md) | Documentación general del proyecto |
| [users-api-readme.md](users-api-readme.md) | Documentación detallada de Users API |
| [orders-api-readme.md](orders-api-readme.md) | Documentación detallada de Orders API |
| [search-api-readme.md](search-api-readme.md) | Documentación detallada de Search API |
| [market-data-api-readme.md](market-data-api-readme.md) | Documentación detallada de Market Data API |
| [portfolio-api-readme.md](portfolio-api-readme.md) | Documentación detallada de Portfolio API |
| [wallet-api-readme.md](wallet-api-readme.md) | Documentación detallada de Wallet API |

## 🔄 Cambios Recientes

Estos archivos fueron movidos aquí después de la implementación del **Docker Compose unificado** (2025-10-12).

### ¿Por qué están aquí?

Anteriormente, cada servicio tenía:
- Su propio `docker-compose.yml` individual
- Documentación extensa en archivos `*-readme.md` en la raíz

Ahora:
- ✅ Existe un **docker-compose.yml unificado** en la raíz
- ✅ Cada servicio tiene su propio `README.md` actualizado dentro de su carpeta
- ✅ Esta documentación legacy se preserva como referencia

## 📖 Para qué sirven estos archivos

Útiles si necesitas:
- Entender la arquitectura original detallada
- Consultar ejemplos de configuración más extensos
- Referencia histórica del diseño del sistema
- Información técnica profunda de cada servicio

## 🚀 Para empezar a usar el proyecto

**NO uses esta documentación para setup inicial.** En su lugar:

```bash
# 1. Lee el README principal
cat ../../README.md

# 2. Sigue el QUICKSTART
cat ../../QUICKSTART.md

# 3. Levanta todo con un comando
cd /ads2-ProyectoFinal-2025
make up
```

---

📌 **Nota**: Esta documentación se mantiene por valor histórico pero puede estar desactualizada respecto a la implementación actual.
