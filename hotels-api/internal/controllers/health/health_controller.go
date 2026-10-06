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
	service  string
	checks   map[string]CheckFunc
	optional map[string]CheckFunc
}

func NewController(service string, checks map[string]CheckFunc, optional ...map[string]CheckFunc) Controller {
	c := Controller{
		service: service,
		checks:  checks,
	}
	if len(optional) > 0 {
		c.optional = optional[0]
	}
	return c
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
// el mapa de estado. Sólo las dependencias obligatorias afectan el código HTTP.
func (controller Controller) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), checkTimeout)
	defer cancel()

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		results  = make(map[string]string, len(controller.checks))
		healthy  = true
		degraded = false
	)
	all := make(map[string]CheckFunc, len(controller.checks)+len(controller.optional))
	for name, check := range controller.optional {
		all[name] = check
	}
	for name, check := range controller.checks {
		all[name] = check
	}
	for name, check := range all {
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
			_, required := controller.checks[name]
			healthy = healthy && (!required || status == "ok")
			degraded = degraded || status != "ok"
			mu.Unlock()
		}(name, check)
	}
	wg.Wait()

	statusCode := http.StatusOK
	statusText := "ok"
	if degraded {
		statusText = "degraded"
	}
	if !healthy {
		statusCode = http.StatusServiceUnavailable
	}
	c.JSON(statusCode, gin.H{
		"status":  statusText,
		"service": controller.service,
		"checks":  results,
	})
}
