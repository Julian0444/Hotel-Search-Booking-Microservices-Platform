# Kubernetes (plan 09 — CN1)

Manifiestos YAML planos que hacen verdadero el titular de "orquestación / load
balancing / escalado horizontal": Deployments con probes reales (`/readyz`,
`/livez` del plan 05), Services que balancean en lugar del upstream estático de
nginx, HPA, y rolling deploys sin drops gracias al SIGTERM del plan 08.

Escalar pasa de "duplicar YAML del compose + editar nginx.conf + reiniciar" a:

```bash
kubectl scale deploy users-api --replicas=5 -n hotel-platform
```

## Layout

| Archivo | Qué trae |
|---|---|
| `00-namespace.yaml` | Namespace `hotel-platform` (todo vive ahí) |
| `01-configmap.yaml` | Config no-secreta compartida (hosts = nombres de Service) |
| `02-secrets.yaml` | Secret con valores DEMO (`stringData`) — cambiar fuera de local |
| `10-mysql.yaml` … `15-solr.yaml` | Datastores dev: StatefulSets con PVC chico (memcached es Deployment: no tiene estado) + Job de migraciones de users-api |
| `20-users-api.yaml` | **Template canónico**: Deployment ×3 + Service + HPA (2–6, CPU 70%) |
| `21-hotels-api.yaml`, `22-search-api.yaml` | Deltas del template (hotels: `containerPort` 8081) |
| `30-ingress.yaml` | Opcional: espejo del ruteo de `nginx.conf` (necesita ingress-nginx) |

## Quickstart con kind

```bash
# 0) Cluster local
kind create cluster --name hotel-platform

# 1) Imágenes de los 3 servicios con tag inmutable por git-SHA (CN4).
#    hotels/search buildean con contexto RAÍZ (el replace de go.mod necesita
#    platform-contracts/); users-api con su propio contexto.
SHA=$(git rev-parse --short HEAD)
docker build -t ghcr.io/julian0444/users-api:sha-$SHA  users-api/
docker build -t ghcr.io/julian0444/hotels-api:sha-$SHA -f hotels-api/Dockerfile .
docker build -t ghcr.io/julian0444/search-api:sha-$SHA -f search-api/Dockerfile .

# 2) Cargarlas en el cluster (kind no ve el daemon del host). Precargar también
#    los datastores ahorra pulls lentos dentro del nodo.
kind load docker-image ghcr.io/julian0444/users-api:sha-$SHA \
  ghcr.io/julian0444/hotels-api:sha-$SHA ghcr.io/julian0444/search-api:sha-$SHA \
  --name hotel-platform
kind load docker-image mysql:8 mongo:6 rabbitmq:3-management solr:9 memcached:1.6-alpine \
  --name hotel-platform
# Gotcha (Docker Desktop con containerd image store): si `kind load` falla con
# "ctr: content digest ... not found" en imágenes PULLEADAS (las built locales
# cargan bien), usar el fallback por archive de una sola plataforma:
#   docker save --platform linux/arm64 mysql:8 -o /tmp/i.tar && \
#   kind load image-archive /tmp/i.tar --name hotel-platform
# — o simplemente omitir la precarga y dejar que el nodo pullee del registry.

# 3) Si el SHA actual no coincide con el de los manifests, actualizar el tag:
grep -rn "image: ghcr.io" k8s/   # y editar, o: kubectl set image ... tras aplicar

# 4) Aplicar todo y esperar Ready (Solr tarda ~90s; los APIs crash-loopean
#    hasta que sus dependencias pasan el probe — es el self-healing normal)
kubectl apply -f k8s/
kubectl get pods -n hotel-platform -w

# 5) Smoke test por port-forward
kubectl port-forward svc/users-api 8082:8082 -n hotel-platform &
curl -s localhost:8082/readyz && kill %1
```

## La demo que importa

```bash
# Escalado horizontal sin tocar nginx ni duplicar YAML (CN1)
kubectl scale deploy users-api --replicas=5 -n hotel-platform
kubectl get endpoints users-api -n hotel-platform    # 5 endpoints

# Rolling restart sin drops (SIGTERM + preStop; plan 08)
kubectl -n hotel-platform run curl-loop --image=curlimages/curl --restart=Never -- \
  sh -c 'while true; do curl -so /dev/null -w "%{http_code}\n" --max-time 2 \
    http://users-api:8082/livez; sleep 0.2; done'
kubectl rollout restart deploy users-api -n hotel-platform
kubectl rollout status  deploy users-api -n hotel-platform
kubectl logs curl-loop -n hotel-platform | sort | uniq -c   # solo 200s
kubectl delete pod curl-loop -n hotel-platform

# Self-healing
kubectl delete pod -l app=users-api -n hotel-platform --wait=false
kubectl get pods -n hotel-platform -w   # los repone solo
```

## HPA

El HPA necesita metrics-server, que kind no trae:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl patch deploy metrics-server -n kube-system --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl get hpa -n hotel-platform   # deja de mostrar <unknown> en ~1 min
```

Sin metrics-server el HPA simplemente no actúa (los pods corren igual).
Gotcha conocido: re-aplicar `20-users-api.yaml` resetea `replicas` a 3 aunque
el HPA hubiera escalado — en un setup serio, el campo `replicas` se omite del
manifest una vez que el HPA es dueño del conteo.

## Ingress (opcional)

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
# (kind necesita extraPortMappings en el cluster para exponerlo en localhost:80;
#  ver https://kind.sigs.k8s.io/docs/user/ingress/)
```

`30-ingress.yaml` espeja solo el **ruteo** del gateway — incluidas las dos
sub-rutas `/api/v1/users/{id}/…/reservations` que van a **hotels-api**
(`use-regex`). Rate-limiting, security headers, CORS y el TLS del plan 10
siguen viviendo en `nginx.conf` para la demo compose.

## Notas honestas (dev vs prod)

- **Datastores**: StatefulSets con PVC chico para que el cluster demo sea
  autocontenido. En prod serían servicios managed (RDS/Atlas/CloudAMQP/
  SolrCloud u OpenSearch); nada del código cambia — solo la config de hosts.
- **Mongo sigue standalone** a propósito: el no-overbooking (plan 04) usa
  claims atómicos por hotel-noche, no transacciones multi-documento.
- **Secretos**: `02-secrets.yaml` trae valores demo commiteados igual que
  `.env.example`. En prod el Secret lo materializa un secrets manager
  (External Secrets / sealed-secrets / SOPS), nunca el repo.
- **ConfigMaps copiados**: las migraciones SQL, el seed de Mongo y el config de
  Solr son copias de sus fuentes canónicas (`users-api/migrations/`,
  `hotels-api/seed/`, `search-api/internal/solr-config/`) — si cambian, hay que
  regenerarlos. El template de un Job es inmutable: ante cambios,
  `kubectl delete job users-migrate` antes de re-aplicar.
- **Imágenes**: tags inmutables `sha-<git-sha>` (CN4). El job `docker` del CI
  las buildea multi-arch, las gatea con Trivy y las pushea a GHCR en pushes a
  `main`; para desarrollo local, `kind load docker-image` como arriba.

## Limpieza

```bash
kind delete cluster --name hotel-platform
```
