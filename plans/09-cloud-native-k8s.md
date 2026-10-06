# Plan 09 — Cloud-native: Kubernetes, imágenes y supply-chain

> **Alcance:** CN1, CN2, CN3, CN4, I2
> **Base:** playbook 7.8 de `plantofinish.md` + I2 (Dockerfiles multi-stage, afinidad directa)
> **Prerequisitos:** plan 05 (**`/readyz`/`/livez`** — los probes los necesitan), plan 08 (**SIGTERM** — drain limpio en rolling deploys), plan 02 (CI donde enganchar Trivy/push)
> **Esfuerzo:** 1–2 días (L) — el agregado de mayor impacto del repo: hace verdadero el titular de "orquestación/escalado"

## Contexto

El titular del proyecto es "orquestación / load balancing / escalado horizontal" pero no hay orquestador: escalar = duplicar YAML a mano + editar nginx + reiniciar (CN1, `LOAD_BALANCER.md:120-146`, `docker-compose.yml:125-208` con 3 bloques copy-paste). Además: 2 de 3 Dockerfiles single-stage como root (I2, contradice `ProyectoBackend.md`), sin `.dockerignore` (CN2), sin resource limits (CN3), todo `:latest` sin scan ni registry (CN4).

## Correcciones de la Sección 7.0 que aplican

- **CN1**: los probes asumen `/readyz`/`/livez` — por eso el plan 05 es prerequisito (si no corrió, apuntar ambos probes a `/health` como interino). hotels-api **ignora `PORT`** (hardcodea `:8081`; el fix es C5, plan 11) → fijar `containerPort: 8081` sin depender de la env.
- **CN4**: pinear un digest por-arquitectura rompe el build multi-arch → pinear el digest del **manifest-list** (`docker buildx imagetools inspect <imagen>`). Renombrar `hotels-api/dockerfile` → `Dockerfile` (la minúscula rompe un matrix de CI).
- **CN1/HPA**: el HPA necesita `resources.requests` obligatorios.
- DNS interno = nombre del Service (no los hostnames del compose).

## Pasos

### 1. Dockerfiles multi-stage / no-root (I2)

- Renombrar `hotels-api/dockerfile` → `Dockerfile` **y actualizar la referencia en `docker-compose.yml:216`** (`dockerfile: dockerfile` → `dockerfile: Dockerfile`) — si no, `docker compose build` rompe.
- Replicar el patrón de `users-api/Dockerfile` en hotels-api y search-api: builder con `go mod download` (no `tidy`) + `CGO_ENABLED=0 go build -ldflags "-s -w"`, imagen final `alpine` (o `gcr.io/distroless/static`), usuario no-root.
- Nota go.work (plan 02): el build en Docker corre fuera del workspace → el `replace ../platform-contracts` de cada go.mod debe resolverse; copiar `platform-contracts/` al contexto de build (context raíz del repo con `dockerfile: ./hotels-api/Dockerfile`) o vendorear.

### 2. `.dockerignore` por servicio (CN2)

Uno por cada contexto de build: `node_modules`, `dist`, `*.md`, `.git`, colecciones Bruno, seeds, `coverage.out`. El del **frontend obligatorio** excluye `node_modules` (hoy el del host puede sombrear el `npm ci` del contenedor).

### 3. Manifiestos k8s (CN1)

Estructura `k8s/` (YAML plano; Helm opcional después):

- `namespace.yaml`, `configmap.yaml` (hosts de DBs, puertos), `secret.yaml` (**un** `JWT_SECRET` compartido, `MYSQL_ROOT_PASSWORD`, etc. — para demo `stringData`, en prod referirlo a un secrets manager).
- **users-api como template canónico** — Deployment + Service + HPA. *Gotcha de puerto:* users-api escucha en **8082** (config, Dockerfile y compose lo confirman) — **no** usar 8080 o los probes fallan → CrashLoopBackOff:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: { name: users-api }
spec:
  replicas: 3
  selector: { matchLabels: { app: users-api } }
  template:
    metadata: { labels: { app: users-api } }
    spec:
      containers:
        - name: users-api
          image: ghcr.io/julian0444/users-api:<git-sha>   # tag inmutable (paso 6)
          ports: [{ containerPort: 8082 }]
          envFrom: [{ configMapRef: { name: platform-config } }, { secretRef: { name: platform-secrets } }]
          readinessProbe: { httpGet: { path: /readyz, port: 8082 }, periodSeconds: 10 }
          livenessProbe:  { httpGet: { path: /livez,  port: 8082 }, periodSeconds: 20 }
          resources:
            requests: { cpu: 100m, memory: 128Mi }   # obligatorio para el HPA
            limits:   { cpu: 500m, memory: 256Mi }
---
apiVersion: v1
kind: Service
metadata: { name: users-api }
spec: { selector: { app: users-api }, ports: [{ port: 8082 }] }   # ClusterIP: reemplaza las 3 réplicas copy-paste y el upstream estático
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata: { name: users-api }
spec:
  scaleTargetRef: { apiVersion: apps/v1, kind: Deployment, name: users-api }
  minReplicas: 2
  maxReplicas: 6
  metrics: [{ type: Resource, resource: { name: cpu, target: { type: Utilization, averageUtilization: 70 } } }]
```

- hotels-api y search-api como deltas del template (hotels: `containerPort: 8081` **fijo**).
- **El punto de venta:** escalar pasa de "copiar YAML + editar nginx + reiniciar" a `kubectl scale deploy users-api --replicas=5` (o automático vía HPA), porque el Service hace el balanceo que antes hacía el upstream estático.

### 4. Datastores (CN1)

StatefulSets dev para MySQL/Mongo/Solr/RabbitMQ/Memcached con PVCs chicos. Nota honesta para el README: en prod serían servicios managed; Mongo sigue standalone (las transacciones no son necesarias — el plan 04 lo resolvió con contadores atómicos).

### 5. Ingress (opcional) + resource limits en compose (CN3)

- Ingress-nginx espejando el ruteo del gateway (*gotcha:* las 2 sub-rutas `/users/{id}/reservations` van a **hotels-api** → `nginx.ingress.kubernetes.io/use-regex: "true"`).
- En `docker-compose.yml` (sigue siendo el camino demo rápido): `deploy.resources.limits` + `reservations` en los 3 APIs y las DBs (`limits` se enforcea; `reservations.cpus` es best-effort). **Solr ≥ 1G o hace OOM.**

### 6. Tags, scan y push (CN4)

- Tag por **git-SHA + semver** (`ghcr.io/julian0444/hotels-api:sha-abc1234`, `:v1.2.0`); nunca `:latest` en manifests.
- Pin de imágenes base por digest del **manifest-list**: `docker buildx imagetools inspect golang:1.23-alpine` → `FROM golang:1.23-alpine@sha256:...`.
- Job de CI (extiende el del plan 02): `docker buildx build --platform linux/amd64,linux/arm64` + **Trivy** como gate (`trivy image --exit-code 1 --severity HIGH,CRITICAL`) + push a GHCR en pushes a main.
- Los Dockerfiles multi-stage del paso 1 son prerequisito: sobre single-stage, Trivy marca todo el toolchain.

## Verificar

```bash
# Cluster local
kind create cluster --name hotel-platform   # o minikube start
kubectl apply -f k8s/ && kubectl get pods -w   # todo Ready (readiness real vía /readyz)

# El titular ahora es verdad: escalar sin tocar nginx ni YAML duplicado
kubectl scale deploy users-api --replicas=5 && kubectl get endpoints users-api   # 5 endpoints

# Rolling restart sin 502 (gracias al SIGTERM del plan 08)
( while true; do kubectl run tmp --rm -i --image=curlimages/curl --restart=Never -- \
    -so /dev/null -w "%{http_code}\n" http://users-api:8082/livez; done ) &
kubectl rollout restart deploy users-api && kubectl rollout status deploy users-api
kill %1   # solo 200s

# Self-healing
kubectl delete pod -l app=users-api --wait=false && kubectl get pods -w   # repone solo

# Imágenes
trivy image ghcr.io/julian0444/users-api:sha-$(git rev-parse --short HEAD)   # sin HIGH/CRITICAL
docker compose up -d --build    # el compose sigue funcionando con limits
```

## Al terminar

El versionado lo hace el usuario manualmente; la sesión NO ejecuta comandos de git.
