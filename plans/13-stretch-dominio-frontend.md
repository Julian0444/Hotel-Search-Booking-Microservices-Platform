# Plan 13 — Frontend portfolio-grade: StayLux como producto terminado

> **Objetivo:** convertir el SPA actual en una experiencia confiable, accesible, rápida y visualmente distintiva, lista para ser la cara pública del portfolio. El plan corrige RV21, RV22 y RV31, hace obligatorio FE2 (`React.lazy`) y deja FE1/DM3/DM4/DM6/DM7 como un menú explícito que no bloquea el cierre.
> **Arquitectura frontend:** React Router conserva la URL como fuente de verdad; TanStack Query administra estado remoto, caché e invalidaciones; una capa de servicios normaliza el contrato `/api/v1`; MUI concentra tokens y componentes; Vitest/Testing Library/MSW prueban comportamiento aislado y Playwright prueba los recorridos reales.
> **Stack base:** React 19, Vite 7, MUI 7, React Router 6, React Hook Form, Axios y date-fns. Agregar TanStack Query, Vitest, Testing Library, MSW, Playwright y axe. **No migrar a TypeScript en el alcance obligatorio.**
> **Prerequisitos:** plan 07 completo (envelopes, `/api/v1`, `meta.total`, errores e idempotencia) y plan 11 completo (rename `AvailableRooms`, status real de microservicios y profile de frontend). Ejecutar **antes del plan 12** para que README, OpenAPI, capturas y GIF documenten el producto final.
> **Esfuerzo realista:** 4–6 días enfocados para el núcleo. `User` rico suma ~1 día. TypeScript o cualquiera de los agregados de dominio se estima y ejecuta como otro proyecto.

## Decisión de alcance

Plan 13 es un **menú**, no una obligación de implementar todos los stretch goals. Para un portfolio orientado a backend, el orden de retorno es:

1. **Obligatorio:** estabilidad del contrato, búsqueda/paginación, reserva completa, historial, admin usable, accesibilidad, tests, performance y FE2.
2. **Opcional de alto ROI:** DM7 (`User` con email/display name/rol tipado), solo después de dejar verde el núcleo.
3. **Opcional de bajo ROI:** FE1 (TypeScript). Los tests y contratos estables deben existir antes de migrar.
4. **Diferir a otro proyecto:** DM3 (reviews), DM4 (pagos/saga) y DM6 (room types). Cambian el dominio, el modelo de datos y los contratos; no son trabajo de “pulido del frontend”.

La checkbox de este plan en `plans/README.md` se marca al completar el **núcleo obligatorio** y su bloque `Verificar`. Los ítems opcionales llevan sus propias checkboxes al final y no forman parte de la definición de terminado.

## Evidencia de la auditoría (2026-07-14)

La base no necesita un rewrite: la navegación, las cards, el theme navy/dorado, React Hook Form y el responsive general son rescatables. En pruebas a 1440 px y 390 px no hubo overflow horizontal. Los problemas que hoy impiden llamarlo producto terminado son concretos:

- `npm run lint` falla: `src/context/AuthContext.jsx` exporta provider + hook (`react-refresh/only-export-components`) y `src/pages/Admin/HotelForm.jsx` tiene dependencias incompletas en un effect.
- `npm run build` termina, pero genera un único chunk JS de **734.65 kB minificado / 227.33 kB gzip** y Vite advierte que supera 500 kB.
- `Search.jsx` calcula `totalPages` con el largo de la página actual; la paginación no puede avanzar. Además ordena solo la página descargada y el estado `page/sort` no vive en la URL.
- `AuthContext.jsx` confía en cualquier token/user de `localStorage`, no valida `exp`; el interceptor hace `window.location.href`, pierde `state.from` y rompe la continuidad SPA.
- `HotelDetail.jsx` muestra horas RFC3339 crudas, tolera a la vez snake/camel y el typo viejo, no permite elegir habitaciones/huéspedes y no manda `Idempotency-Key`.
- `MyReservations.jsx` oculta reservas canceladas y reconstruye el estado desde las fechas, aunque el backend ya es dueño del lifecycle.
- `helpers.js` usa UTC para el mínimo de los date pickers: cerca de medianoche puede ofrecer el día incorrecto. El parseo de fechas `YYYY-MM-DD` tampoco está aislado del huso horario.
- `Dashboard.jsx` usa `Promise.all`: la caída de una fuente oculta todo. El admin puede borrarse a sí mismo y las tablas no tienen una adaptación móvil seria.
- `HotelForm.jsx` manda `HH:mm` contra el contrato viejo `time.Time`, usa amenities libres, no valida URLs de imágenes, no protege cambios sin guardar y navega con `setTimeout`.
- `Footer.jsx` contiene links `href="#"`, datos de contacto y claims no verificables. Home afirma soporte 24/7, máxima seguridad, hoteles verificados y miles de viajeros sin que el producto lo respalde.
- Solo hay un `<title>` estático, sigue el favicon de Vite, faltan 404 real, skip link, nombres accesibles en varios icon buttons, anuncios de ruta, `prefers-reduced-motion` y fallbacks robustos de imágenes.
- No existe suite de tests del frontend; CI solo ejecuta build y `npm audit`.

### Hallazgos locales que cubre este plan

Estos IDs son internos de este archivo y no alteran la tabla maestra de `plantofinish.md`:

| ID | Hallazgo | Fase |
|----|----------|:----:|
| F13-01 | Contrato HTTP duplicado y fallbacks legacy en componentes | 1 |
| F13-02 | Sesión zombie y redirect duro en 401 | 2 |
| F13-03 | Paginación/orden/URL de búsqueda incorrectos | 3 |
| F13-04 | Fechas locales, horas y dinero inconsistentes | 1, 5 |
| F13-05 | Booking incompleto y sin idempotencia visible | 5 |
| F13-06 | Historial oculta canceladas y deriva estados | 6 |
| F13-07 | Dashboard fail-hard, self-delete y tablas frágiles | 7 |
| F13-08 | Form de hotel roto y poco seguro | 7 |
| F13-09 | Copy/links/claims que degradan confianza | 4 |
| F13-10 | Accesibilidad, responsive, SEO e imágenes incompletos | 8 |
| F13-11 | Bundle monolítico y carga sin estrategia | 9 |
| F13-12 | Sin tests frontend ni quality gate | 0, 10 |

## Contrato visual y de producto

### Dirección estética: “hospitality editorial argentina”

- Conservar **ink navy + antique gold** como firma, sobre fondo warm parchment; sumar terracotta/sage solo como acentos semánticos, no como arcoíris de dashboard.
- Mantener `Cormorant Garamond` para titulares y `Source Sans 3` para lectura/UI. Asegurar fallback local del sistema para que la app no quede rota si Google Fonts falla.
- Inspiración: revista de viaje boutique, no marketplace genérico. Mucho espacio, grilla editorial, bordes finos, radios moderados y sombras cortas. Evitar gradientes decorativos, glassmorphism y animaciones constantes.
- La imagen memorable será el **booking concierge**: panel sticky en desktop y bottom action en mobile, con fechas, habitaciones, huéspedes, noches, total y disponibilidad antes de confirmar.
- Animación limitada a entrada de resultados, feedback de interacción y transiciones de dialog. Respetar `prefers-reduced-motion` y nunca bloquear interacción por animar.
- El producto visible queda en inglés consistente; los docs pueden seguir bilingües. No mezclar idiomas dentro de una pantalla.

### Regla de honestidad

Solo mostrar capacidades que existen. Quitar “24/7 support”, “verified hotels”, “highest security”, “thousands of travelers”, dirección/teléfono ficticios y links muertos. Reemplazarlos por beneficios demostrables: búsqueda indexada, disponibilidad protegida, reserva idempotente, historial y panel administrativo real.

### Breakpoints de aceptación

Revisar cada recorrido al menos en `320×568`, `390×844`, `768×1024` y `1440×900`. A 320 px no puede haber scroll horizontal, controles cortados ni targets de menos de 44 px. En desktop, el contenido principal debe quedar legible entre 1120 y 1280 px, sin filas de cards huérfanas estiradas artificialmente.

## Estrategia de trabajo

Cada fase sigue el mismo orden: **test rojo → implementación mínima → refactor → verificación enfocada**. No maquillar páginas sobre adapters legacy: primero se estabilizan contrato, sesión y fechas; después se rediseña.

## Fase 0 — Red de seguridad del frontend

### Archivos

- Modificar `frontend/package.json`, `frontend/package-lock.json`, `frontend/vite.config.js`, `frontend/eslint.config.js` y `.github/workflows/ci.yml`.
- Crear `frontend/src/test/setup.js`, `frontend/src/test/render.jsx`, `frontend/src/test/handlers.js`, `frontend/src/test/server.js` y `frontend/src/test/fixtures.js`.
- Crear `frontend/playwright.config.js` y `frontend/e2e/`.

### Pasos

1. Instalar dependencias compatibles con el lockfile actual:

   ```bash
   cd frontend
   npm install @tanstack/react-query
   npm install -D vitest jsdom @vitest/coverage-v8 @testing-library/react @testing-library/user-event @testing-library/jest-dom msw @playwright/test @axe-core/playwright
   npx playwright install chromium
   ```

2. Agregar scripts estables:

   ```json
   {
     "test": "vitest",
     "test:run": "vitest run",
     "test:coverage": "vitest run --coverage",
     "test:e2e": "playwright test",
     "check": "npm run lint && npm run test:coverage && npm run build"
   }
   ```

3. Configurar jsdom, `jest-dom`, limpieza después de cada test y MSW con `onUnhandledRequest: 'error'`. El helper `render` envuelve ThemeProvider, QueryClientProvider, AuthProvider y MemoryRouter; cada test recibe un QueryClient nuevo con retries desactivados.
4. Crear fixtures en el **contrato final del plan 07/11**, sin camelCase ni `avaiable_*`. MSW debe responder envelopes `{data, meta}` y errores `{error:{code,message,trace_id}}`.
5. Escribir primero un smoke test de Home y uno de error normalizado; confirmar que fallan antes de crear los adapters de la fase 1.
6. En CI frontend, ejecutar `npm run lint`, `npm run test:coverage`, `npm run build` y luego `npm audit --audit-level=high`. Mantener Playwright end-to-end en job separado, solo cuando el compose esté listo, para que un test unitario no dependa de Docker.

### Salida de la fase

`npm run test:run` descubre tests; un request no mockeado falla; CI ya impide volver a introducir un frontend sin tests.

## Fase 1 — Contrato HTTP, errores, server state y fechas

### Archivos

- Modificar `frontend/src/services/api.js`, todos los `frontend/src/services/*.service.js`, `frontend/src/main.jsx`, `frontend/src/types/index.js` y `frontend/src/utils/helpers.js`.
- Crear `frontend/src/services/queryClient.js`, `frontend/src/services/apiError.js`, `frontend/src/services/envelope.js`, `frontend/src/utils/dateOnly.js`, `frontend/src/hooks/queries/` y `frontend/src/hooks/mutations/`.
- Crear tests junto a módulos (`*.test.js`/`*.test.jsx`) o bajo `frontend/src/test/`; elegir una convención y mantenerla.

### Tests primero

- `unwrapObject` devuelve `body.data`; `unwrapList` devuelve `{items,total,limit,offset}` y rechaza un envelope inválido.
- `ApiError` conserva `status`, `code`, `message` y `traceId`, pero nunca imprime internals del response.
- Los servicios devuelven un solo shape final. Un grep de tests debe demostrar que no soportan camelCase ni el typo viejo.
- `todayLocal()` y `addDaysDateOnly()` funcionan con `TZ=America/Los_Angeles` cerca del cambio de día; `differenceInNights` no cambia por DST.
- El formatter de hora convierte el `HH:mm` decidido en plan 07 a una etiqueta legible y nunca muestra RFC3339.

### Implementación

1. Centralizar `/api/v1` en `API_CONFIG.BASE_URL`. Los componentes no deben leer `response.data`, status codes ni nombres legacy; solo hooks/services.
2. Normalizar una vez:

   ```js
   export const unwrapList = ({ data, meta }) => ({
     items: data,
     total: meta.total,
     limit: meta.limit,
     offset: meta.offset,
   });
   ```

3. Convertir el envelope de error a `ApiError`. En UI mostrar mensaje estable y, solo para errores inesperados, una línea secundaria `Reference: <trace_id>` copiable.
4. Crear QueryClient con defaults deliberados: retry solo en GET y máximo una vez, `staleTime` corto para búsqueda, sin retry de 4xx, y mutations sin retry automático. No duplicar estado remoto en `useState`.
5. Definir query keys (`['hotels','search',params]`, `['hotel',id]`, `['reservations',userId]`, `['admin','users',params]`, `['admin','health']`) y factories de hooks.
6. Pasar `AbortSignal` de TanStack Query a Axios (`signal`) para cancelar búsquedas anteriores y evitar carreras.
7. Tratar fechas de reserva como strings civiles `YYYY-MM-DD`. Construirlas por componentes locales; no usar `toISOString().split('T')[0]` ni `new Date('YYYY-MM-DD')` en lógica de producto.
8. Respetar las dos unidades que hoy define el dominio: `hotel.price_per_night` sigue en unidades mayores (`float64`) y `reservation.total_price` viaja en centavos (`int64`). Crear helpers separados (`formatMajorAmount`/`formatMinorAmount`) y calcular el preview en centavos con `Math.round(price_per_night * 100)`; no mezclar ni multiplicar floats por las páginas.
9. Actualizar JSDoc: `user_id` es string, Register no acepta `tipo`, Reservation incluye status/rooms/guests/total/currency y Hotel usa solo `available_rooms`.

### Salida de la fase

No hay fallbacks `pricePerNight`, `checkInTime`, `avaiable*` ni acceso directo al envelope fuera de `services/`. Las fechas pasan tests en LA y UTC.

## Fase 2 — Sesión y navegación confiables (RV31)

### Archivos

- Modificar `frontend/src/context/AuthContext.jsx`, `frontend/src/hooks/useAuth.js`, `frontend/src/services/api.js`, `frontend/src/App.jsx`, `frontend/src/pages/Login.jsx` y `frontend/src/pages/Register.jsx`.
- Crear `frontend/src/context/auth-context.js`, `frontend/src/services/authStorage.js`, `frontend/src/services/authEvents.js`, `frontend/src/components/auth/ProtectedRoute.jsx` y `frontend/src/components/common/AppLoader.jsx`.

### Tests primero

- Token ausente, malformado o expirado inicia como anónimo y limpia storage.
- Un 401 de login llega al formulario; un 401 de sesión emite `session-expired`, no hace hard reload.
- Ir anónimo a `/reservations` manda a `/login` con `state.from`; login correcto vuelve a la ruta original.
- Cliente que intenta `/admin` recibe una pantalla 403/redirect explícito; nunca ve un frame del dashboard.
- Loading de bootstrap no es el mismo estado que submitting de login/register.

### Implementación

1. Mover el `createContext` a `context/auth-context.js`; `AuthContext.jsx` exporta únicamente el provider/componente y `useAuth.js` consume el context primitivo. Esto elimina el error actual de Fast Refresh sin esconderlo con una regla de ESLint.
2. Encapsular lectura/escritura de storage. Decodificar localmente los claims necesarios y validar al menos `exp`, `iss` y audience antes de aceptar una sesión; el backend sigue siendo autoridad.
3. El interceptor solo normaliza/rechaza y emite un evento de sesión. AuthProvider limpia estado; Router navega a login preservando ubicación. **Eliminar `window.location.href`.**
4. Separar `isBootstrapping`, `isSubmitting` y errores de formulario. La app muestra un loader accesible durante bootstrap, no un `return null`.
5. Login/Register usan `aria-live`, autocompletado correcto (`username`, `current-password`, `new-password`), focus en el primer error y copy consistente.
6. Logout invalida queries privadas y reemplaza la history entry para que Back no exponga una vista cacheada.

### Salida de la fase

La sesión expirada conduce a Login sin reload, explica qué pasó y vuelve a la intención original tras reautenticar.

## Fase 3 — Búsqueda correcta, compartible y global (RV22)

### Archivos

- Modificar `frontend/src/pages/Search.jsx`, `frontend/src/components/Hotels/SearchBar.jsx`, `frontend/src/components/Hotels/HotelCard.jsx`, `frontend/src/services/hotels.service.js` y `frontend/src/constants/index.js`.
- Crear `frontend/src/hooks/useHotelSearchParams.js`, `frontend/src/components/Hotels/HotelGridSkeleton.jsx` y `frontend/src/components/common/{EmptyState,ErrorState}.jsx`.
- Si se conserva el sort, modificar `search-api/internal/controllers/search/search_controller.go`, `search-api/internal/services/search/search_service.go`, `search-api/internal/repositories/hotels/hotels_solr.go` y sus tests.

### Decisión obligatoria sobre sort

El sort actual es engañoso porque ordena solo los resultados ya descargados. Elegir una de dos opciones y cubrirla con tests:

- **Recomendada:** aceptar `sort=relevance|price_asc|price_desc|rating_desc` en search-api, whitelist en controller/service y mapping a sort seguro de Solr. `meta.total` corresponde a la query global.
- **Recorte válido:** quitar el selector de la UI. Nunca mantener orden local con paginación remota.

### Tests primero

- URL `/search?q=bariloche&page=2&sort=price_asc` hidrata controles y calcula `offset` correctamente.
- Back/Forward y un link compartido restauran query, page y sort.
- `meta.total=45, limit=12` muestra 4 páginas aunque `data.length=12`.
- Nueva búsqueda resetea page a 1; cambiar sort también.
- Una respuesta lenta anterior no pisa la más reciente.
- Un término sin resultados, un 502 y un retry tienen estados visuales distintos.
- Si se implementa sort backend: tests Go rechazan valores arbitrarios y verifican el sort Solr esperado.

### Implementación

1. URL es la única fuente de verdad para `q`, `page`, `sort`; no duplicarla en varios `useState`.
2. Debounce solo al escribir; submit y chips actualizan URL de inmediato. Query key incluye todos los parámetros.
3. `keepPreviousData` conserva el grid durante paginación, con indicador sutil; skeleton completo solo en primera carga.
4. Mostrar “45 stays” desde `meta.total`, no `hotels.length`. Deshabilitar páginas fuera de rango y corregir URL si el total bajó.
5. Cards con aspect ratio estable, `loading="lazy"`, `decoding="async"`, fallback local y precio/rating accesibles. Toda la card no debe ser un link anidado con botones.
6. En desktop usar grilla que cierre bien con 3 columnas para el seed actual; en mobile una columna compacta. Evitar la fila de cuatro más una card huérfana observada en la auditoría.

### Salida de la fase

Paginación, orden y count representan todo el resultado; copiar/pegar la URL reproduce exactamente la pantalla.

## Fase 4 — Home, layout y sistema visual

### Archivos

- Modificar `frontend/src/theme/theme.js`, `frontend/src/index.css`, `frontend/src/pages/Home.jsx`, `frontend/src/components/Layout/{Layout,Navbar,Footer}.jsx`, `frontend/index.html` y `frontend/src/constants/index.js`.
- Crear `frontend/src/components/common/{PageHeader,SectionHeading,ResponsiveImage,StatusChip,RouteMeta}.jsx`.
- Crear activos propios en `frontend/public/brand/` y `frontend/public/images/` (logo/mark, favicon y fallback hotel). SVG simple y code-native; no hace falta generar una marca compleja.

### Tests primero

- Navbar ofrece los links correctos para anónimo/cliente/admin y cierra el drawer al navegar.
- Footer no contiene `href="#"`; todos sus destinos son reales.
- Home renderiza datos del seed/API y sus estados loading/error/empty sin claims inventados.
- `ResponsiveImage` cambia a fallback después de error y conserva alt/aspect ratio.

### Implementación

1. Formalizar tokens: `surface`, `surfaceElevated`, `ink`, `muted`, `gold`, `terracotta`, `sage`, radios, sombras, ancho de contenido y escala de spacing. No usar hex sueltos por página.
2. Home queda en tres actos, no una landing interminable:
   - hero editorial con búsqueda y un destino real del seed;
   - selección dinámica de estadías/destinos;
   - “How it works” de tres pasos verificables (search, reserve, manage).
3. Eliminar estadísticas y promesas no demostrables. Si se quiere enseñar la arquitectura, usar una franja discreta “Built as three Go services + event-driven search” enlazada a README, no fingir valor comercial.
4. Navbar con marca propia, estado activo, skip target y mobile drawer accesible. Footer compacto: navegación, repositorio, arquitectura/licencia si existen al terminar plan 12; sin social icons falsos.
5. Reducir sombras gigantes y cards redundantes. Conservar serif en headings, pero limitar tamaños móviles con `clamp()`.
6. Reemplazar `/vite.svg`, añadir theme color y metadata base. No depender de una imagen externa para que Login/Home sean legibles.

### Salida de la fase

Home comunica en menos de dos scrolls móviles qué se puede hacer; toda promesa visible se puede demostrar en la app.

## Fase 5 — Hotel detail y booking concierge (RV21)

### Archivos

- Modificar `frontend/src/pages/HotelDetail.jsx`, `frontend/src/services/reservations.service.js`, `frontend/src/services/hotels.service.js` y `frontend/src/utils/helpers.js`.
- Crear `frontend/src/components/booking/{BookingPanel,BookingDialog,BookingSummary,BookingSuccess}.jsx` y `frontend/src/components/Hotels/HotelGallery.jsx`.

### Tests primero

- Gallery se adapta a 0, 1, 2 o 4 imágenes sin huecos; una URL rota cae al placeholder.
- Horas `14:00`/`10:00` se muestran localizadas y nunca como RFC3339.
- Check-out mínimo es el día siguiente local; noches y total son correctos alrededor de DST.
- Rooms/guests tienen límites, labels y errores; submit queda deshabilitado con fechas inválidas.
- Cada intento nuevo genera `crypto.randomUUID()` y lo manda como `Idempotency-Key`; un retry de red del mismo intento conserva la key.
- `no_availability`, `request_in_flight`, 401 y 502 producen mensajes/acciones distintas.

### Implementación

1. Gallery editorial: mosaico si hay varias imágenes, una hero completa si solo hay una; botón “View all photos” abre dialog/lightbox con teclado y focus trap.
2. Panel sticky en desktop; CTA sticky inferior abre dialog en mobile. Permitir elegir `check_in`, `check_out`, `num_rooms`, `num_guests` y mostrar noches, precio por noche, total/currency.
3. Usar el endpoint de disponibilidad antes de confirmar como feedback rápido, pero dejar que `POST /reservations` sea la autoridad atómica; explicar un 409 sin culpar al usuario.
4. Idempotencia se maneja por intento de usuario, no globalmente. Deshabilitar doble submit y conservar key ante timeout/retry.
5. Tras 201, mostrar confirmación con reservation ID, resumen y CTA a My Reservations. Invalidar disponibilidad, hotel y reservations queries.
6. Mostrar dirección/contacto solo si existen; no fabricar. Amenities desconocidas usan icono genérico y texto normalizado.

### Salida de la fase

Un usuario puede entender costo, capacidad y fechas antes de confirmar; el éxito tiene evidencia y próximo paso, y el doble click no duplica la reserva.

## Fase 6 — Historial de reservas fiel al dominio

### Archivos

- Modificar `frontend/src/pages/MyReservations.jsx` y `frontend/src/services/reservations.service.js`.
- Crear `frontend/src/components/reservations/{ReservationCard,ReservationFilters,CancelReservationDialog}.jsx`.

### Tests primero

- Confirmed/upcoming, in-progress, completed y cancelled se agrupan sin ocultarse.
- `status` del backend manda; las fechas solo derivan la sección temporal si hace falta.
- Cancelar confirmado abre confirmación, hace mutation, invalida la query y conserva la card como cancelled.
- Una cancelación ya aplicada/204 es idempotente en UI; una no permitida explica por qué.
- Estados empty, partial data y error con retry son accesibles.

### Implementación

1. Tabs o filtros `Upcoming / Past / Cancelled / All` con counts. No borrar canceladas del historial.
2. Card muestra hotel, rango civil, nights, rooms, guests, total/currency, status y reservation ID corto/copiable.
3. Botón Cancel solo cuando el backend/lifecycle lo permite; no inferir permiso exclusivamente por reloj del browser.
4. Mutation usa invalidación de query o updater funcional; eliminar el `setReservations(reservations.filter(...))` capturando estado viejo.
5. En mobile cards apiladas; en desktop no convertirlas en tabla si eso empeora legibilidad.

### Salida de la fase

My Reservations funciona como historial auditable, no como una lista que hace desaparecer operaciones.

## Fase 7 — Admin usable y seguro (RV31)

### Archivos

- Modificar `frontend/src/pages/Admin/Dashboard.jsx`, `frontend/src/pages/Admin/HotelForm.jsx`, `frontend/src/services/admin.service.js`, `frontend/src/services/hotels.service.js` y `frontend/src/utils/validators.js`.
- Crear `frontend/src/components/admin/{AdminHotelList,AdminUserList,ServiceHealthGrid,HotelImageFields}.jsx` y `frontend/src/hooks/useUnsavedChanges.js`.

### Tests primero

- Error de hotels no oculta users/health y viceversa; cada panel tiene retry propio.
- Usuario logueado no puede activar delete sobre sí mismo y ve el motivo.
- Paginación usa `meta.total`, búsqueda resetea page y mobile no requiere scroll horizontal para acciones básicas.
- HotelForm envía `HH:mm`, `price_per_night` numérico y `available_rooms` entero según el contrato final; los centavos pertenecen a `reservation.total_price`, no al payload del hotel.
- URL de imagen no HTTP(S), email inválido, price/rooms fuera de rango y check-out time inválido bloquean submit.
- Cambios sucios piden confirmación al abandonar; submit exitoso navega al recurso sin `setTimeout`.
- Service status muestra valores reales del plan 11 y no ofrece scale/restart/logs ficticios.

### Implementación

1. Reemplazar `Promise.all` por queries independientes. Stats se derivan de data disponible; un error parcial no borra todo el dashboard.
2. Separar tabs `Hotels`, `Users`, `Services`; cada una carga solo al visitarse. Tab Services es read-only, muestra readiness/latencia/última actualización y aclara que es observabilidad, no control plane.
3. Listas con búsqueda y paginación de backend. Desktop usa tabla; mobile cambia a cards o una fila con disclosure, sin esconder acciones críticas fuera de viewport.
4. Deshabilitar self-delete comparando IDs canónicos string. Confirm dialogs nombran el recurso y separan cancel/delete.
5. HotelForm usa secciones cortas, summary de errores y focus al primero. Amenities vienen de un catálogo permitido con opción controlada; URLs muestran preview/fallback y pueden reordenarse.
6. Adaptar estrictamente el contrato de plan 07 para `check_in_time/check_out_time`; adaptar solo `available_rooms` del plan 11. Eliminar todos los fallbacks antiguos.
7. Navegar inmediatamente tras mutation confirmada o mostrar “View hotel / Back to dashboard”; no usar temporizadores.

### Salida de la fase

El panel sigue siendo útil con un servicio caído, no simula operaciones y no permite que el admin se borre por accidente.

## Fase 8 — Accesibilidad, responsive, SEO y estados límite

### Archivos

- Modificar `frontend/src/App.jsx`, Layout/Navbar/Footer, todas las páginas y `frontend/src/index.css`.
- Crear `frontend/src/pages/NotFound.jsx`, `frontend/src/components/a11y/{SkipLink,RouteAnnouncer}.jsx` y el hook/meta de ruta.

### Tests primero

- Axe sin violaciones `serious`/`critical` en Home, Search, HotelDetail, Login, MyReservations y Dashboard.
- Todos los icon buttons tienen nombre accesible; dialogs devuelven focus al trigger; tabs/drawer funcionan por teclado.
- Al cambiar ruta, el foco va a `main`/h1 y se anuncia el título sin robar focus durante typing.
- `prefers-reduced-motion: reduce` elimina scroll suave/transforms no esenciales.
- Tests viewport confirman `documentElement.scrollWidth === innerWidth` a 320 y 390 px.
- Ruta desconocida muestra 404 útil, no redirect silencioso.

### Implementación

1. Un solo `<main id="main-content">`, skip link visible al focus, heading hierarchy por ruta y landmarks correctos.
2. `aria-label` en acciones solo-icono, texto visible para acciones importantes, errores ligados con `aria-describedby` y notificaciones con severidad/live region adecuada.
3. Contraste WCAG AA, focus ring visible y targets ≥44 px. Rating visual tiene equivalente textual.
4. `RouteMeta` actualiza title/description por pantalla; HotelDetail incluye nombre/ubicación. Crear 404 con búsqueda y regreso.
5. Imágenes con width/height o aspect-ratio para evitar layout shift, lazy debajo del fold y prioridad solo para hero relevante.
6. Revisar copy de errores: qué ocurrió, qué puede hacer el usuario y referencia técnica solo si ayuda.

### Salida de la fase

Los seis recorridos principales se completan por teclado y sin violaciones axe serias/críticas; 320 px es un viewport soportado.

## Fase 9 — Performance y code splitting (FE2 obligatorio)

### Archivos

- Modificar `frontend/src/App.jsx`, `frontend/src/main.jsx` y `frontend/vite.config.js` solo si el análisis de chunks lo justifica.
- Crear `frontend/src/components/common/RouteFallback.jsx`.

### Tests/medición primero

1. Guardar baseline de `npm run build` (734.65 kB actual) y Lighthouse en preview production.
2. Escribir test de routing que espere el fallback y luego la página lazy.
3. Registrar requests/chunks iniciales en Home y al entrar a Admin; no aprobar el cambio solo porque existen más archivos.

### Implementación

1. `React.lazy` para Search, HotelDetail, Login, Register, MyReservations, Dashboard, HotelForm y NotFound. Home puede quedar eager; Admin nunca entra al bundle inicial de visitante.
2. Un `Suspense` de ruta con skeleton/layout estable; no spinner full-screen que haga saltar navbar/footer.
3. Prefetch intencional al hover/focus de “View hotel” o al quedar idle, sin prefetch de todo el sitio.
4. Activar los future flags compatibles de React Router que eliminan las advertencias actuales; si requieren upgrade, hacerlo aislado y cubierto por routing tests.
5. Usar `manualChunks` solo después de mirar el analyzer; no crear un mega `vendor` eterno. Remover imports/dependencias sin uso antes de micro-optimizar.
6. Optimizar activos locales a WebP/AVIF cuando corresponda; external hotel URLs mantienen fallback y dimensiones.

### Presupuesto

- Cero warning de Vite por chunks >500 kB.
- Bundle inicial de visitante ≤170 kB gzip (objetivo, ajustar con evidencia si MUI lo hace imposible).
- Dashboard/Admin no se descarga en Home.
- Lighthouse production en desktop/mobile: Performance ≥85, Accessibility ≥95, Best Practices ≥90, SEO ≥90; documentar hardware/red y no perseguir 100 artificial.

### Salida de la fase

FE2 está realmente implementado: la primera visita descarga menos código y las rutas siguen teniendo loading/error UX coherente.

## Fase 10 — E2E y cierre adversarial

### Archivos

- Crear `frontend/e2e/anonymous-search.spec.js`, `frontend/e2e/customer-booking.spec.js`, `frontend/e2e/admin.spec.js`, `frontend/e2e/accessibility.spec.js` y helpers/fixtures necesarios.
- Modificar `.github/workflows/ci.yml`, `Makefile` y `README.md` solo para exponer comandos; el contenido final/capturas sigue en plan 12.

### Recorridos Playwright

1. **Anonymous:** Home → search → ordenar → abrir hotel → intentar reservar → login con retorno. La paginación se cubre con MSW y, si el seed real supera una página, también acá.
2. **Customer:** login demo → elegir fechas/rooms/guests → reservar → ver confirmación → My Reservations → cancelar → verla en Cancelled.
3. **Admin:** login seed env-driven → hotels/users/services cargan independiente → crear/editar hotel → aparece en search → self-delete bloqueado.
4. **Degradación:** search-api o hotels-api indisponible produce estado útil, retry y `trace_id`, no pantalla blanca ni HTML de nginx.
5. Ejecutar Anonymous y Customer en Chromium desktop + mobile; admin al menos desktop. Fallar por errores no permitidos de consola o requests sin manejar.
6. Screenshots visuales solo de superficies estables (Home, Search, HotelDetail, booking, Dashboard) a 390 y 1440. Enmascarar fechas/IDs dinámicos para evitar snapshots ruidosos.

### Revisión adversarial R1–R6

- **R1 — Regresiones:** auth, disponibilidad, idempotencia, cancelación, CRUD admin y C11 siguen funcionando.
- **R2 — Edge cases:** token expirado, query vacía, page fuera de rango, 0/1/4 imágenes, hotel sin amenities, fecha de hoy cerca de medianoche, 409 concurrente, 502 y payload parcial.
- **R3 — Integración:** `/api/v1`, envelopes/meta, string user ID, HH:mm, centavos, `available_rooms`, status real y profile compose coinciden con planes 07/11.
- **R4 — Calidad:** no warnings de React/Router, no timers de navegación, no estado remoto duplicado, no copy inventado, tests nombran comportamiento.
- **R5 — Seguridad/privacidad:** no token en logs/URL, no error interno en UI, admin guards efectivos, endpoint de reservas ajenas protegido, links externos con `rel="noreferrer"`.
- **R6 — Evidencia:** guardar resultados de lint/tests/build/e2e/Lighthouse y capturas finales para que plan 12 los incorpore.

## Menú opcional después del núcleo

### [ ] DM7 — `User` rico (único stretch recomendado)

Hacerlo solo si se quiere una segunda señal backend+frontend:

- `Role` tipado en users-api, `email` único validado, `display_name` y `created_at` con migración versionada.
- Registration pide email/display name; navbar/avatar y admin list muestran display name con username como fallback.
- JWT mantiene claims mínimos; no meter PII innecesaria en el token.
- Agregar Profile read/edit solo si existe endpoint real y autorización owner/admin; no crear una pantalla decorativa.
- Actualizar fixtures, contract tests, OpenAPI y e2e. Checkbox propia; no mezclar este cambio con las fases de estabilización.

### [ ] FE1 — TypeScript (diferir salvo tiempo sobrante)

- Empezar con `allowJs` y migrar de afuera hacia adentro: envelopes/errors/date utils → services/hooks → componentes → páginas.
- Generar tipos desde OpenAPI solo después del plan 12 o mantener tipos manuales únicos, nunca ambos divergentes.
- No aceptar `any` para silenciar el compilador. Agregar `typecheck` a CI cuando todo `src/` esté migrado.
- Estimarlo como proyecto separado; TypeScript no corrige por sí mismo UX, accesibilidad ni contratos malos.

### [ ] DM3 / [ ] DM4 / [ ] DM6 — otro proyecto

- **DM3 Reviews:** nuevo agregado + rating derivado + moderación/autorización + reindexado.
- **DM4 Payments:** provider/stub + estados monetarios + compensaciones + idempotencia/webhooks.
- **DM6 Room types:** inventario por tipo/noche + migración + pricing/capacidad + selector UI.

Si se elige alguno, primero escribir un design doc y un plan nuevo. No anexarlo a este cierre frontend: cada uno cambia arquitectura, datos y alcance de entrevista.

## Definición de terminado del núcleo

- [ ] `npm run lint`, `npm run test:coverage` y `npm run build` terminan en 0.
- [ ] Coverage global frontend ≥60% statements/lines/functions y ≥50% branches; services, date utils, auth y mutations críticas ≥80%. Priorizar casos de riesgo sobre tests cosméticos para inflar el número.
- [ ] Cero `avaiable`, fallbacks camelCase, raw RFC3339 de horas, `href="#"`, `window.location.href`, `toISOString().split('T')[0]` y claims falsos en `frontend/src`.
- [ ] Search usa `meta.total`; URL reproduce q/page/sort y el sort es global o no existe.
- [ ] Reserva manda rooms/guests + `Idempotency-Key`, muestra total y termina en una confirmación verificable.
- [ ] Cancelled permanece visible; status/money vienen del dominio.
- [ ] Dashboard tolera error parcial, no permite self-delete y Services es real/read-only.
- [ ] Axe: 0 serious/critical en seis rutas; teclado y focus verificados.
- [ ] Sin overflow a 320 px; screenshots aprobados a 390/1440.
- [ ] Bundle sin chunk >500 kB, Admin lazy y presupuesto Lighthouse cumplido o desvío documentado con evidencia.
- [ ] E2E Anonymous, Customer y Admin en verde con el stack seed.
- [ ] Plan 12 recibe capturas finales y facts reales, no mocks.

## Verificar

```bash
# 1) Estático + unit/component + coverage + production build
cd frontend
npm ci
npm run lint
TZ=America/Los_Angeles npm run test:coverage
npm run build

# 2) Invariantes concretas: todos deben dar 0 hits
rg -n "avaiable|pricePerNight|checkInTime|checkOutTime|window\.location\.href|href=[\"']#[\"']|toISOString\(\)\.split" src
rg -ni "24/7|verified hotels|highest security|thousands of" src

# 3) Backend adjunct del sort, solo si se eligió conservarlo
cd ../search-api
go test -race ./internal/controllers/search ./internal/services/search ./internal/repositories/hotels

# 4) Stack real + E2E
cd ..
test -f .env || cp .env.example .env   # no pisar secretos locales
docker compose --profile frontend up -d --build
docker compose ps      # esperar todos healthy; Solr puede tardar ~90s
cd frontend
npm run test:e2e

# 5) Performance sobre build de producción (preview en otra terminal)
npm run preview -- --host 127.0.0.1
# correr Lighthouse mobile+desktop y guardar reportes para plan 12
```

Resultado esperado: todos los checks de la definición de terminado están verdes; los tres flujos atraviesan nginx y servicios reales; no hay errores/warnings inesperados en consola; Home/Search/Detail/Booking/Dashboard quedan listos para las capturas del plan 12.

## Al terminar

Marcar Plan 13 en `plans/README.md`, registrar evidencia y decisiones en `plans/HANDOFF.md`, y recién entonces ejecutar el plan 12. El versionado lo hace el usuario manualmente; la sesión **NO ejecuta comandos de git**.
