package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var k8sClient client.Client

func main() {
	mode := os.Getenv("BFF_MODE") // "mock", "k8s", or unset (defaults to mock)
	if mode == "" {
		mode = "mock"
	}
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Kubernetes client if in k8s mode
	if mode == "k8s" {
		if err := initK8sClient(); err != nil {
			log.Fatalf("failed to init k8s client: %v", err)
		}
		log.Println("Kubernetes client initialized (in-cluster config)")
	} else {
		log.Println("Running in mock mode")
	}

	r := gin.Default()

	// CORS for local development
	r.Use(corsMiddleware())

	// JWT auth middleware
	r.Use(authMiddleware())

	// Health check (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "mode": mode})
	})

	// Tenant endpoints
	r.GET("/api/v1/tenants", GetTenantsHandler(mode))
	r.POST("/api/v1/tenants", CreateTenantHandler(mode))
	r.GET("/api/v1/tenants/:name", GetTenantDetailHandler(mode))
	r.GET("/api/v1/tenants/:name/metrics", GetTenantMetricsHandler(mode))
	r.GET("/api/v1/tenants/:name/kubeconfig", GetTenantKubeconfigHandler(mode))
	r.PATCH("/api/v1/tenants/:name", UpdateTenantHandler(mode))
	r.DELETE("/api/v1/tenants/:name", DeleteTenantHandler(mode))

	port := os.Getenv("BFF_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting BFF on :%s (mode=%s)", port, mode)
	err := r.Run(":" + port)
	if err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func initK8sClient() error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}
	k8sClient = cl
	return nil
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow health check without auth
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		// Extract JWT from Authorization header
		auth := c.GetHeader("Authorization")
		if auth == "" {
			// For demo: allow requests without auth if no JWT secret is set
			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				log.Println("Warning: JWT_SECRET not set, allowing all requests")
				c.Next()
				return
			}
			c.AbortWithStatusJSON(401, gin.H{"error": "missing authorization header"})
			return
		}

		// Validate JWT
		if !validateJWT(auth) {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}
		c.Next()
	}
}

func validateJWT(authHeader string) bool {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return false
	}
	tokenString := authHeader[7:]
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return true // Allow if no secret configured
	}
	// In production: validate JWT signature
	_ = tokenString
	return true
}
