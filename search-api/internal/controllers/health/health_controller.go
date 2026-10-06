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
	var extra map[string]CheckFunc
	if len(optional) > 0 {
		extra = optional[0]
	}
	return Controller{
		service:  service,
		checks:   checks,
		optional: extra,
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
		mu              sync.Mutex
		wg              sync.WaitGroup
		results         = make(map[string]string, len(controller.checks))
		healthy         = true
		degraded        = false
		optionalResults = make(map[string]string, len(controller.optional))
	)
	run := func(name string, check CheckFunc, required bool) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := "ok"
			if err := check(ctx); err != nil {
				status = "down"
				slog.Warn("dependency check failed", "check", name, "error", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if required {
				results[name] = status
				healthy = healthy && status == "ok"
			} else {
				optionalResults[name] = status
			}
			degraded = degraded || status != "ok"
		}()
	}
	for name, check := range controller.checks {
		run(name, check, true)
	}
	for name, check := range controller.optional {
		run(name, check, false)
	}
	wg.Wait()

	statusCode := http.StatusOK
	statusText := "ok"
	if degraded {
		statusText = "degraded"
	}
	if !healthy {
		statusCode = http.StatusServiceUnavailable
		statusText = "degraded"
	}
	c.JSON(statusCode, gin.H{
		"status":          statusText,
		"service":         controller.service,
		"checks":          results,
		"optional_checks": optionalResults,
	})
}
