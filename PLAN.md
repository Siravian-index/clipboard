# Clipboard Manager — Plan de desarrollo

## Objetivo

Construir un clipboard manager para Linux (X11) en Go que:
- Corra como daemon en background
- Mantenga un historial de las últimas N entradas copiadas
- Permita seleccionar una entrada anterior via UI y devolverla al clipboard
- Sea activado por un hotkey global

---

## Stack

| Componente | MVP | Futuro |
|---|---|---|
| UI | Fyne | X11 nativo |
| Watcher | Polling cada 500ms | X11 Events (xfixes) |
| Historial | En memoria | SQLite / archivo |
| Hotkey | GNOME Custom Shortcut | — |
| Clipboard | `golang.design/x/clipboard` | — |
| Autostart | systemd user service | — |

---

## Arquitectura

Todos los componentes se exponen a través de interfaces para poder intercambiar implementaciones sin modificar el resto del sistema.

```
clipboard/
├── main.go
├── watcher/
│   ├── watcher.go       — interface Watcher
│   └── polling.go       — implementación polling
├── history/
│   ├── history.go       — interface History
│   └── memory.go        — implementación en memoria
├── ui/
│   ├── ui.go            — interface UI
│   └── fyne.go          — implementación fyne
└── hotkey/
    ├── hotkey.go        — interface HotkeyListener
    └── xgb.go           — implementación X11
```

---

## Interfaces

```go
// Monitorea cambios en el clipboard del sistema
type Watcher interface {
    Start(onChange func(content string)) error
    Stop() error
}

// Almacena y recupera el historial de entradas
type History interface {
    Add(entry string)
    List() []string
    Clear()
}

// Muestra el historial al usuario y devuelve la selección
type UI interface {
    Show(items []string) (selected string, err error)
}

// Escucha hotkeys globales del sistema
type HotkeyListener interface {
    Register(keys string, callback func()) error
    Listen() error
    Stop() error
}
```

---

## Flujo principal

```
[Daemon arranca]
      │
      ├─ Watcher.Start() ──► detecta cambio en clipboard
      │                            │
      │                            ▼
      │                      History.Add(content)
      │
      └─ HotkeyListener.Listen() ──► usuario presiona hotkey
                                          │
                                          ▼
                                    UI.Show(History.List())
                                          │
                                          ▼
                                   usuario selecciona item
                                          │
                                          ▼
                                  escribe selección al clipboard
```

---

## Fases de desarrollo

### Fase 1 — MVP
- [x] `go mod init`
- [x] Interfaces de todos los componentes
- [x] `history/memory.go` — historial en memoria (máx. 50 entradas)
- [x] `watcher/polling.go` — polling cada 500ms con `golang.design/x/clipboard`
- [x] `ui/fyne.go` — ventana popup con lista seleccionable
- [x] `main.go` — daemon + show subcommands via Unix socket
- [x] GNOME custom shortcut para abrir el picker

### Fase 2 — Mejoras
- [x] Live update: daemon hace streaming de nuevas entradas al cliente vía socket, UI de Fyne se refresca en tiempo real sin cerrar la ventana
- [x] Persistencia del historial en disco (SQLite) — las entradas sobreviven reinicios del daemon
- [x] Mantener la ventana abierta después de seleccionar un item — permite pegar múltiples entradas sin reabrir el picker
- [x] Instancia única del picker (`show`): al correr `clipboard-manager show`, verificar si ya hay una instancia activa via un socket dedicado (`~/.clipboard-manager-show.sock`). Si existe, enviarle `{"type":"focus"}` y salir. Si no, levantar la UI, abrir el socket y escuchar mensajes de foco — al recibirlos llamar `w.RequestFocus()`.
- [x] Soporte de imágenes en el historial:
  - Detectar cuando el clipboard contiene una imagen (ej. screenshots de GNOME)
  - Guardar imágenes en disco (`~/.local/share/clipboard-manager/images/`) referenciadas desde SQLite
  - Mostrar thumbnails en la UI del picker
  - Al seleccionar una imagen, devolverla al clipboard como imagen (no como ruta)
  - Considerar límite de tamaño y limpieza de imágenes huérfanas al hacer clear
- [x] Settings integrado en la ventana principal: reemplazar la segunda ventana de Settings por un panel lateral o vista inline dentro de la misma ventana, usando `container.NewAppTabs` o un layout con panel izquierdo (lista) / derecho (settings) para no romper el foco ni abrir ventanas adicionales.
- [x] Keyboard shortcuts: navegación con flechas, Enter para seleccionar, Escape para cerrar, Space para pegar sin cerrar, Ctrl+S para guardar settings, Ctrl+/ para abrir ayuda
- [ ] Menú de personalización de shortcuts: permitir al usuario reasignar los shortcuts desde la UI de Settings, persistiendo la configuración en `config.json`
- [ ] Temas (colores): permitir al usuario cambiar el tema de la UI desde Settings. Fyne soporta `theme.DarkTheme()` y `theme.LightTheme()` nativamente, y permite temas personalizados implementando la interfaz `fyne.Theme`. Persistir la selección en `config.json` y aplicarla con `a.Settings().SetTheme()` al arrancar.
- [ ] Icono de aplicación: diseñar un PNG (256×256), embeber con `fyne bundle` y asignarlo con `a.SetIcon()` para reemplazar el engranaje por defecto en alt-tab, barra de título y taskbar.
- [ ] Detección de contraseñas y ofuscación: identificar entradas del historial que posiblemente sean contraseñas (heurística: sin espacios, longitud mínima, mezcla de caracteres, etc.) y mostrarlas ofuscadas en la UI (`••••••••`) con opción de revelar/ocultar por item.
- [ ] Detección de URLs: identificar entradas que sean links (validando con `url.Parse`) y mostrar un icono adicional en la fila para abrirlos directamente en el browser con `xdg-open`, manteniendo la acción de copiar como principal.
- [ ] Watcher event-driven con X11 xfixes
- [ ] Soporte Wayland via DBus
- [x] Menú de ajustes en la UI con opciones configurables:
  - Mantener ventana abierta tras selección (on/off) — aplica inmediato, solo afecta al cliente
  - Límite máximo de entradas en el historial — aplica enviando SIGHUP al daemon
  - Botón para limpiar el historial completo con confirmación
- [x] Persistir ajustes en `~/.config/clipboard-manager/config.json` via paquete `config/`
- [x] Daemon recarga configuración al recibir SIGHUP (estándar Unix)

### Fase 3 — Distribución
- [x] systemd user service para autostart al login (`~/.config/systemd/user/clipboard-manager.service`)
  - Durante desarrollo: `ExecStart=%h/Code/golang/clipboard/clipboard-manager daemon` — apunta al binario del repo, se actualiza con cada `go build`
  - Para distribución: `make install` copia el binario a `/usr/local/bin` y actualiza el service
- [ ] Makefile con cross-compilation targets
- [ ] Script de instalación (`make install`) que copia el binario, instala el service y lo habilita
- [ ] UI nativa X11 para eliminar dependencia de Fyne

---

## Dependencias iniciales

```
golang.design/x/clipboard   — leer/escribir clipboard
fyne.io/fyne/v2             — UI
github.com/jezek/xgb        — X11 bindings para hotkeys
```

---

## Notas

- El historial no guarda duplicados consecutivos (si copias lo mismo dos veces, solo se almacena una entrada)
- El historial tiene un máximo configurable (default: 50 entradas)
- Al seleccionar un item del historial, ese item pasa a ser el primero de la lista
