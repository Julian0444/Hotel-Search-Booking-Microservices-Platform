package microservices

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Estructura para representar el estado de un microservicio
type ServiceStatus struct {
	Name      string    `json:"name"`
	Instances []Instance `json:"instances"`
	Status    string    `json:"status"`
	LoadBalanced bool   `json:"load_balanced"`
}

type Instance struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Port     string `json:"port"`
	UpTime   string `json:"uptime"`
	Health   string `json:"health"`
}

type ScaleRequest struct {
	ServiceName string `json:"service_name" binding:"required"`
	Replicas    int    `json:"replicas" binding:"required,min=1,max=10"`
}

type Controller struct {
	// En una implementación real, aquí tendríamos un cliente de Docker
	// dockerClient *docker.Client
}

func NewController() Controller {
	return Controller{}
}

// GetMicroservicesStatus devuelve el estado de todos los microservicios
func (controller Controller) GetMicroservicesStatus(ctx *gin.Context) {
	// Simular la consulta del estado de los microservicios
	services := []ServiceStatus{
		{
			Name: "users-api",
			Instances: []Instance{
				{
					ID:       "users-api-1",
					Name:     "users-api-1",
					Status:   "running",
					Port:     "8080",
					UpTime:   "2h 15m",
					Health:   controller.checkServiceHealth("users-api-1:8080"),
				},
				{
					ID:       "users-api-2", 
					Name:     "users-api-2",
					Status:   "running",
					Port:     "8081",
					UpTime:   "2h 10m",
					Health:   controller.checkServiceHealth("users-api-2:8080"),
				},
				{
					ID:       "users-api-3",
					Name:     "users-api-3", 
					Status:   "running",
					Port:     "8082",
					UpTime:   "2h 5m",
					Health:   controller.checkServiceHealth("users-api-3:8080"),
				},
			},
			Status:       "healthy",
			LoadBalanced: true,
		},
		{
			Name: "hotels-api",
			Instances: []Instance{
				{
					ID:       "hotels-api-container",
					Name:     "hotels-api-container",
					Status:   "running",
					Port:     "8083",
					UpTime:   "2h 20m",
					Health:   controller.checkServiceHealth("hotels-api:8081"),
				},
			},
			Status:       "healthy",
			LoadBalanced: false,
		},
		{
			Name: "search-api",
			Instances: []Instance{
				{
					ID:       "search-api-container",
					Name:     "search-api-container",
					Status:   "running", 
					Port:     "8084",
					UpTime:   "2h 18m",
					Health:   controller.checkServiceHealth("search-api:8082"),
				},
			},
			Status:       "healthy",
			LoadBalanced: false,
		},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"services": services,
		"summary": gin.H{
			"total_services": len(services),
			"total_instances": controller.countTotalInstances(services),
			"healthy_services": controller.countHealthyServices(services),
			"load_balanced_services": controller.countLoadBalancedServices(services),
		},
	})
}

// ScaleService permite escalar un servicio (crear o eliminar instancias)
func (controller Controller) ScaleService(ctx *gin.Context) {
	var request ScaleRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	// Validar que el servicio existe y es escalable
	if !controller.isServiceScalable(request.ServiceName) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("service %s is not scalable or does not exist", request.ServiceName),
		})
		return
	}

	// En una implementación real, aquí interactuaríamos con Docker Compose o Kubernetes
	// Por ahora, simularemos la respuesta
	message := controller.simulateScaling(request.ServiceName, request.Replicas)

	ctx.JSON(http.StatusOK, gin.H{
		"message": message,
		"service": request.ServiceName,
		"new_replicas": request.Replicas,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetServiceLogs obtiene los logs de un servicio específico
func (controller Controller) GetServiceLogs(ctx *gin.Context) {
	serviceName := ctx.Param("service_name")
	instanceID := ctx.Query("instance_id")
	
	if serviceName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "service_name is required",
		})
		return
	}

	// Simular logs del servicio
	logs := controller.generateMockLogs(serviceName, instanceID)

	ctx.JSON(http.StatusOK, gin.H{
		"service": serviceName,
		"instance": instanceID,
		"logs": logs,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RestartService reinicia un servicio específico
func (controller Controller) RestartService(ctx *gin.Context) {
	serviceName := ctx.Param("service_name")
	instanceID := ctx.Query("instance_id")

	if serviceName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "service_name is required",
		})
		return
	}

	// Simular reinicio del servicio
	message := fmt.Sprintf("Service %s", serviceName)
	if instanceID != "" {
		message += fmt.Sprintf(" (instance: %s)", instanceID)
	}
	message += " restart initiated successfully"

	ctx.JSON(http.StatusOK, gin.H{
		"message": message,
		"service": serviceName,
		"instance": instanceID,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Funciones auxiliares

func (controller Controller) checkServiceHealth(address string) string {
	// En una implementación real, haríamos un HTTP GET al health endpoint
	// Por ahora, simulamos alternando entre healthy y warning
	hash := 0
	for _, char := range address {
		hash += int(char)
	}
	
	if hash%3 == 0 {
		return "warning"
	}
	return "healthy"
}

func (controller Controller) countTotalInstances(services []ServiceStatus) int {
	total := 0
	for _, service := range services {
		total += len(service.Instances)
	}
	return total
}

func (controller Controller) countHealthyServices(services []ServiceStatus) int {
	healthy := 0
	for _, service := range services {
		if service.Status == "healthy" {
			healthy++
		}
	}
	return healthy
}

func (controller Controller) countLoadBalancedServices(services []ServiceStatus) int {
	loadBalanced := 0
	for _, service := range services {
		if service.LoadBalanced {
			loadBalanced++
		}
	}
	return loadBalanced
}

func (controller Controller) isServiceScalable(serviceName string) bool {
	// Solo users-api es escalable por ahora (tiene balanceador de carga)
	scalableServices := []string{"users-api"}
	for _, service := range scalableServices {
		if service == serviceName {
			return true
		}
	}
	return false
}

func (controller Controller) simulateScaling(serviceName string, replicas int) string {
	return fmt.Sprintf("Scaling %s to %d replicas. This would normally interact with Docker Compose or Kubernetes to create/remove instances.", serviceName, replicas)
}

func (controller Controller) generateMockLogs(serviceName, instanceID string) []string {
	logs := []string{
		"[INFO] " + time.Now().Add(-10*time.Minute).Format("2006-01-02 15:04:05") + " Service started successfully",
		"[INFO] " + time.Now().Add(-8*time.Minute).Format("2006-01-02 15:04:05") + " Database connection established",
		"[DEBUG] " + time.Now().Add(-5*time.Minute).Format("2006-01-02 15:04:05") + " Processing request from client",
		"[INFO] " + time.Now().Add(-2*time.Minute).Format("2006-01-02 15:04:05") + " Health check passed",
		"[DEBUG] " + time.Now().Add(-1*time.Minute).Format("2006-01-02 15:04:05") + " Request completed successfully",
	}

	if instanceID != "" {
		for i, log := range logs {
			logs[i] = fmt.Sprintf("[%s] %s", instanceID, log)
		}
	}

	return logs
} 