# Plan: Keyboard Shortcuts

## Objetivo

Hacer la app 100% usable sin mouse. Los shortcuts deben poder combinarse
en flujos naturales de navegación.

---

## Shortcuts

| Elemento                    | Shortcut | Acción                                           | Notas                                                                                                                                 |
|-----------------------------|----------|--------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|
| Searchbar                   | Ctrl+F   | Toggle: abrir/cerrar la barra de búsqueda        | Mantener el texto al cerrar; cambiar ícono si hay filtro activo y barra está cerrada                                                  |
| Input dentro de searchbar   | Ctrl+D   | Limpiar todo el texto escrito (limpieza rápida)  | Ctrl+X y Ctrl+C están reservados por Fyne (Cut/Copy); Ctrl+D no tiene conflicto                                                       |
| Settings button             | Ctrl+/   | Abrir el menú de settings                        | Solo abrir; para cerrar el usuario usa Escape                                                                                         |
| Menú de ayuda               | Ctrl+H   | Abrir menú de ayuda con shortcuts disponibles    | Se usa Ctrl+H (Help) en lugar de `?` para poder interceptarlo globalmente sin importar si el focus está en un campo de texto o no. Imagen de referencia: `/docs/plan/shortcuts/shortcuts-menu` |
| Lista de entries            | ↑ / ↓    | Navegar entre entries                            | —                                                                                                                                     |
| Lista de entries            | Space    | Confirmar selección y copiar la entry al clipboard | Equivalente al OK/confirmar sobre la entry resaltada                                                                                |

---

## Flujo de ejemplo

1. `Ctrl+F` → abre y enfoca el searchbar
2. Escribe "texto de..."
3. `Ctrl+D` → limpia el input
4. Escribe "Libro de..."
5. `↓` → mueve el foco a la lista
6. `↓ / ↑` → navega entre resultados
7. `Space` → confirma la selección y copia la entry al clipboard

---

## Orden de implementación sugerido

1. **↑ / ↓ + Enter** — el mayor impacto de UX, es el flujo principal
2. **Ctrl+F mejorado** — mantener texto + cambio de ícono cuando hay filtro activo
3. **Ctrl+/** — abrir settings
4. **Ctrl+H** — menú de ayuda (requiere construir un widget nuevo)
5. **Ctrl+D** — limpiar input del searchbar

---

## IMPORTANTE: Personalización de shortcuts

Todos los shortcuts, presentes y futuros, deberán ser **100% personalizables**.
El usuario podrá usar los valores por defecto o cambiarlos a lo que prefiera.
Esa opción debe estar siempre disponible.

Próximamente se desarrollará un plan para un submenú de personalización de shortcuts dentro de Settings.
