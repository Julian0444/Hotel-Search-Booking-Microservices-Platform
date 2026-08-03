// Package microservices expone el panel admin de estado de la plataforma
// (C2). Es READ-ONLY y REAL: cada instancia se consulta por su GET /readyz
// (O3) y se reporta estado + latencia medidos — reemplaza al panel anterior,
// que devolvía uptimes hardcodeados, health por hash del hostname y acciones
// de scale/restart/logs que no hacían nada.
package microservices

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// probeTimeout acota cada GET /readyz: una instancia caída responde "down"
// rápido en vez de colgar el panel entero.
const probeTimeout = 2 * time.Second

// ServiceTarget describe un servicio y las URLs base de sus instancias.
type ServiceTarget struct {
	Name         string
	InstanceURLs []string
}

// ParseTargets parsea el spec de MICROSERVICES_TARGETS:
//
//	"users-api=http://users-api-1:8082,http://users-api-2:8082;hotels-api=http://127.0.0.1:8081"
//
// (";" separa servicios, "," separa instancias). Entradas malformadas se
// descartan en silencio: el panel es diagnóstico, no configuración crítica.
func ParseTargets(spec string) []ServiceTarget {
	var targets []ServiceTarget
	for _, entry := range strings.Split(spec, ";") {
		name, urls, found := strings.Cut(strings.TrimSpace(entry), "=")
		if !found || name == "" {
			continue
		}
		var instanceURLs []string
		for _, u := range strings.Split(urls, ",") {
			if u = strings.TrimSpace(u); u != "" {
				instanceURLs = append(instanceURLs, u)
			}
		}
		if len(instanceURLs) > 0 {
			targets = append(targets, ServiceTarget{Name: name, InstanceURLs: instanceURLs})
		}
	}
	return targets
}

// InstanceStatus es el resultado del probe a una instancia.
type InstanceStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Status    string `json:"status"` // "up" | "down"
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// ServiceStatus agrega las instancias de un servicio.
type ServiceStatus struct {
	Name         string           `json:"name"`
	Status       string           `json:"status"` // "up" | "degraded" | "down"
	LoadBalanced bool             `json:"load_balanced"`
	Instances    []InstanceStatus `json:"instances"`
}

type Controller struct {
	targets    []ServiceTarget
	httpClient *http.Client
}

func NewController(targets []ServiceTarget) Controller {
	return Controller{
		targets:    targets,
		httpClient: &http.Client{Timeout: probeTimeout},
	}
}

// GetMicroservicesStatus devuelve el estado real de todos los servicios:
// todas las instancias se prueban en paralelo (el panel responde en
// ~probeTimeout aun con la plataforma entera caída).
// GET /api/v1/admin/microservices
func (controller Controller) GetMicroservicesStatus(ctx *gin.Context) {
	services := make([]ServiceStatus, len(controller.targets))

	var wg sync.WaitGroup
	for i, target := range controller.targets {
		services[i] = ServiceStatus{
			Name:         target.Name,
			LoadBalanced: len(target.InstanceURLs) > 1,
			Instances:    make([]InstanceStatus, len(target.InstanceURLs)),
		}
		for j, url := range target.InstanceURLs {
			wg.Add(1)
			go func(i, j int, url string) {
				defer wg.Done()
				services[i].Instances[j] = controller.probeInstance(ctx.Request.Context(), url)
			}(i, j, url)
		}
	}
	wg.Wait()

	totalInstances := 0
	healthyServices := 0
	for i := range services {
		up := 0
		for _, instance := range services[i].Instances {
			if instance.Status == "up" {
				up++
			}
		}
		totalInstances += len(services[i].Instances)
		switch {
		case up == len(services[i].Instances):
			services[i].Status = "up"
			healthyServices++
		case up > 0:
			services[i].Status = "degraded"
		default:
			services[i].Status = "down"
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"data": gin.H{
		"services": services,
		"summary": gin.H{
			"total_services":   len(services),
			"total_instances":  totalInstances,
			"healthy_services": healthyServices,
		},
	}})
}

// probeInstance hace GET {url}/readyz y mide la latencia. Cualquier cosa que
// no sea un 200 dentro del timeout es "down" (readyz ya devuelve 503 si una
// dependencia del servicio falla, O3).
func (controller Controller) probeInstance(parent context.Context, url string) InstanceStatus {
	status := InstanceStatus{
		Name:   strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://"),
		URL:    url,
		Status: "down",
	}

	probeCtx, cancel := context.WithTimeout(parent, probeTimeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url+"/readyz", nil)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	resp, err := controller.httpClient.Do(req)
	status.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		status.Error = err.Error()
		return status
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		status.Status = "up"
	} else {
		status.Error = fmt.Sprintf("readyz returned status %d", resp.StatusCode)
	}
	return status
}
