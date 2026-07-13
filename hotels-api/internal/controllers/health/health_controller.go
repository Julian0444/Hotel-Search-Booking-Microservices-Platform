package health

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CheckFunc pinguea una dependencia del servicio; nil = sana.
type CheckFunc func(ctx context.Context) error

// checkTimeout es menor que el timeout del healthcheck del compose (5s):
// mejor responder 503 nosotros que dejar que docker mate el probe.
const checkTimeout = 3 * time.Second

type Controller struct {
	service string
	checks  map[string]CheckFunc
}

func NewController(service string, checks map[string]CheckFunc) Controller {
	return Controller{
		service: service,
		checks:  checks,
	}
}

// Livez responde 200 sin tocar dependencias: solo indica que el proceso está
// vivo (O3). /health queda como alias de este handler por compatibilidad.
func (controller Controller) Livez(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": controller.service,
	})
}

// Readyz pinguea las dependencias del servicio en paralelo y devuelve 503 con
// el mapa de estado si alguna está caída (O3).
func (controller Controller) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), checkTimeout)
	defer cancel()

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make(map[string]string, len(controller.checks))
		healthy = true
	)
	for name, check := range controller.checks {
		wg.Add(1)
		go func(name string, check CheckFunc) {
			defer wg.Done()
			status := "ok"
			if err := check(ctx); err != nil {
				status = "down"
				slog.Warn("readiness check failed", "check", name, "error", err)
			}
			mu.Lock()
			results[name] = status
			healthy = healthy && status == "ok"
			mu.Unlock()
		}(name, check)
	}
	wg.Wait()

	statusCode := http.StatusOK
	statusText := "ok"
	if !healthy {
		statusCode = http.StatusServiceUnavailable
		statusText = "degraded"
	}
	c.JSON(statusCode, gin.H{
		"status":  statusText,
		"service": controller.service,
		"checks":  results,
	})
}
