# Fix F3 — Frontend: el interceptor 401 no debe comerse el error de login

> **Alcance:** RV11 (interceptor 401 con hard-reload en login), RV12 (minLength inconsistente)
> **Prerequisitos:** ninguno. Independiente de F1/F2/F4.
> **Esfuerzo:** <1 h (S)
> Snippets validados contra el código del 2026-07-11.

## Contexto

- **RV11**: el interceptor de respuesta (`frontend/src/services/api.js:37-47`) trata **todo** 401 como sesión expirada: borra storage y hace `window.location.href = '/login'`. Pero una password incorrecta en `POST /login` también es 401 → el hard-redirect recarga `/login` antes de que `AuthContext.login` pueda mostrar "invalid credentials": el usuario ve la página parpadear sin feedback. Además, en cualquier otra página el hard-redirect pierde el `state.from` que `Login.jsx` sí sabe usar para volver.
- **RV12**: `Login.jsx` valida password con `minLength: 4` hardcodeado (`:165-168`), mientras `Register.jsx` y el backend usan 8 (`VALIDATION.MIN_PASSWORD_LENGTH` en `frontend/src/constants/index.js:95`, `min=8` en `users_domain.go`). `Login.jsx` ya importa `VALIDATION` (`:30`).

## Pasos

### 1. Excluir el request de login del interceptor (RV11)

En `frontend/src/services/api.js:37-47` (`auth.service.js:26` postea a `/login`):

```js
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Un 401 del propio login es "credenciales inválidas", no sesión expirada:
    // debe llegar al formulario, no forzar un reload que se come el error (RV11)
    const isLoginRequest = error.config?.url?.includes('/login');
    if (error.response?.status === 401 && !isLoginRequest) {
      localStorage.removeItem(STORAGE_KEYS.TOKEN);
      localStorage.removeItem(STORAGE_KEYS.USER);
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

> Nota: el hard-redirect para sesiones expiradas fuera de `/login` se mantiene (mejorarlo con navegación SPA + `state.from` es plan 13/RV31). Este fix solo elimina el caso roto.

### 2. minLength del password de Login (RV12)

En `frontend/src/pages/Login.jsx:165-168`, reemplazar el literal:

```js
                minLength: {
                  value: VALIDATION.MIN_PASSWORD_LENGTH,
                  message: `Password must be at least ${VALIDATION.MIN_PASSWORD_LENGTH} characters`,
                },
```

(Ojo: el `minLength` de `:141` es el del username — no tocarlo.)

## Verificar

```bash
cd frontend
npx eslint src/services/api.js src/pages/Login.jsx   # sin errores nuevos
npm run build
npm run dev
```

Manual (compose levantado):
1. `/login` con password incorrecta → **la página NO recarga** y se ve el mensaje de error.
2. `/login` con credenciales demo correctas → entra normal.
3. Sesión expirada simulada: con sesión iniciada, borrar/corromper el token en localStorage (DevTools) y navegar a "My Reservations" → redirige a `/login` (el comportamiento de expiración se conserva).
4. Password de 5 caracteres en login → el form la rechaza client-side (min 8).

Al terminar: tick en `plans/fixes/README.md` y sección nueva en `plans/HANDOFF.md`.
