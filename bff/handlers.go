package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// TenantSummary is a simplified representation returned by the BFF
type TenantSummary struct {
	Name             string    `json:"name"`
	Tier             string    `json:"tier"`
	Owner            string    `json:"owner"`
	State            string    `json:"state,omitempty"`
	Namespace        string    `json:"namespace,omitempty"`
	CreatedAt        time.Time `json:"createdAt,omitempty"`
	CPU              string    `json:"cpu,omitempty"`
	Memory           string    `json:"memory,omitempty"`
	APIEndpoint      string    `json:"apiEndpoint,omitempty"`
	KubeconfigSecret string    `json:"kubeconfigSecret,omitempty"`
}

// TenantDetail extends TenantSummary with more details
type TenantDetail struct {
	TenantSummary
	NetworkPolicy map[string]interface{} `json:"networkPolicy,omitempty"`
	Events        []string               `json:"events,omitempty"`
}

// GetTenantsHandler returns a handler function for listing tenants
func GetTenantsHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mode == "k8s" {
			getTenantsK8s(c)
		} else {
			getTenantsMock(c)
		}
	}
}

func getTenantsMock(c *gin.Context) {
	examplesDir := filepath.Join("..", "examples", "tenants")
	var tenants []TenantSummary
	_ = filepath.WalkDir(examplesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		docs := strings.Split(string(b), "---")
		for _, doc := range docs {
			doc = strings.TrimSpace(doc)
			if doc == "" {
				continue
			}
			var m map[string]any
			if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
				continue
			}
			meta, _ := m["metadata"].(map[string]any)
			spec, _ := m["spec"].(map[string]any)
			status, _ := m["status"].(map[string]any)
			name := ""
			owner := ""
			tier := ""
			state := ""
			namespace := ""
			cpu := ""
			memory := ""
			if meta != nil {
				if v, ok := meta["name"].(string); ok {
					name = v
				}
			}
			if spec != nil {
				if v, ok := spec["owner"].(string); ok {
					owner = v
				}
				if v, ok := spec["tier"].(string); ok {
					tier = v
				}
				if res, ok := spec["resources"].(map[string]any); ok {
					if v, ok2 := res["cpu"].(string); ok2 {
						cpu = v
					}
					if v, ok2 := res["memory"].(string); ok2 {
						memory = v
					}
				}
			}
			if status != nil {
				if v, ok := status["state"].(string); ok {
					state = v
				}
				if v, ok := status["namespace"].(string); ok {
					namespace = v
				}
			}
			if name != "" {
				tenants = append(tenants, TenantSummary{
					Name:      name,
					Tier:      tier,
					Owner:     owner,
					State:     state,
					Namespace: namespace,
					CPU:       cpu,
					Memory:    memory,
				})
			}
		}
		return nil
	})
	c.JSON(http.StatusOK, tenants)
}

func getTenantsK8s(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.io",
		Version: "v1alpha1",
		Kind:    "TenantList",
	})

	err := k8sClient.List(ctx, list)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var tenants []TenantSummary
	for _, item := range list.Items {
		spec, _, _ := unstructured.NestedMap(item.Object, "spec")
		status, _, _ := unstructured.NestedMap(item.Object, "status")

		t := TenantSummary{
			Name:      item.GetName(),
			CreatedAt: item.GetCreationTimestamp().Time,
		}

		if tier, ok := spec["tier"].(string); ok {
			t.Tier = tier
		}
		if owner, ok := spec["owner"].(string); ok {
			t.Owner = owner
		}
		if resources, ok := spec["resources"].(map[string]interface{}); ok {
			if cpu, ok := resources["cpu"].(string); ok {
				t.CPU = cpu
			}
			if mem, ok := resources["memory"].(string); ok {
				t.Memory = mem
			}
		}
		if state, ok := status["state"].(string); ok {
			t.State = state
		}
		if ns, ok := status["namespace"].(string); ok {
			t.Namespace = ns
		}
		if endpoint, ok := status["apiEndpoint"].(string); ok {
			t.APIEndpoint = endpoint
		}
		if secret, ok := status["adminKubeconfigSecret"].(string); ok {
			t.KubeconfigSecret = secret
		}

		tenants = append(tenants, t)
	}

	c.JSON(http.StatusOK, tenants)
}

// GetTenantDetailHandler returns full details of a single tenant
func GetTenantDetailHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if mode == "k8s" {
			getTenantDetailK8s(c, name)
		} else {
			getTenantDetailMock(c, name)
		}
	}
}

func getTenantDetailMock(c *gin.Context, name string) {
	examplesDir := filepath.Join("..", "examples", "tenants")
	path := filepath.Join(examplesDir, name+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid yaml"})
		return
	}
	spec, _ := m["spec"].(map[string]any)
	status, _ := m["status"].(map[string]any)
	detail := TenantDetail{
		TenantSummary: TenantSummary{
			Name: name,
		},
	}
	if spec != nil {
		if tier, ok := spec["tier"].(string); ok {
			detail.Tier = tier
		}
		if owner, ok := spec["owner"].(string); ok {
			detail.Owner = owner
		}
	}
	if status != nil {
		if state, ok := status["state"].(string); ok {
			detail.State = state
		}
	}
	c.JSON(http.StatusOK, detail)
}

func getTenantDetailK8s(c *gin.Context, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.io",
		Version: "v1alpha1",
		Kind:    "Tenant",
	})

	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, obj)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	status, _, _ := unstructured.NestedMap(obj.Object, "status")

	detail := TenantDetail{
		TenantSummary: TenantSummary{
			Name:      obj.GetName(),
			CreatedAt: obj.GetCreationTimestamp().Time,
		},
	}

	if tier, ok := spec["tier"].(string); ok {
		detail.Tier = tier
	}
	if owner, ok := spec["owner"].(string); ok {
		detail.Owner = owner
	}
	if state, ok := status["state"].(string); ok {
		detail.State = state
	}

	c.JSON(http.StatusOK, detail)
}

// CreateTenantHandler creates a new tenant from JSON
func CreateTenantHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req map[string]any
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}

		name, ok := req["name"].(string)
		if !ok || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant name"})
			return
		}

		if mode == "k8s" {
			createTenantK8s(c, name, req)
		} else {
			createTenantMock(c, name, req)
		}
	}
}

func createTenantMock(c *gin.Context, name string, spec map[string]any) {
	crd := map[string]any{
		"apiVersion": "platform.io/v1alpha1",
		"kind":       "Tenant",
		"metadata":   map[string]any{"name": name},
		"spec":       spec,
	}
	out, err := yaml.Marshal(crd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal"})
		return
	}
	examplesDir := filepath.Join("..", "examples", "tenants")
	_ = os.MkdirAll(examplesDir, 0755)
	path := filepath.Join(examplesDir, name+".yaml")
	if err := os.WriteFile(path, out, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"created": name, "path": path})
}

func createTenantK8s(c *gin.Context, name string, spec map[string]any) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.io",
		Version: "v1alpha1",
		Kind:    "Tenant",
	})
	obj.SetName(name)
	obj.SetNamespace("")

	// Set spec fields
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")

	if err := k8sClient.Create(ctx, obj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create tenant: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"created": name})
}

// UpdateTenantHandler updates an existing tenant
func UpdateTenantHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		var updates map[string]any
		if err := c.BindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if mode == "k8s" {
			updateTenantK8s(c, name, updates)
		} else {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "update not supported in mock mode"})
		}
	}
}

func updateTenantK8s(c *gin.Context, name string, updates map[string]any) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.io",
		Version: "v1alpha1",
		Kind:    "Tenant",
	})

	if err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, obj); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	for k, v := range updates {
		spec[k] = v
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")

	if err := k8sClient.Update(ctx, obj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update tenant: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"updated": name})
}

// DeleteTenantHandler deletes a tenant
func DeleteTenantHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if mode == "k8s" {
			deleteTenantK8s(c, name)
		} else {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "delete not supported in mock mode"})
		}
	}
}

func deleteTenantK8s(c *gin.Context, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.io",
		Version: "v1alpha1",
		Kind:    "Tenant",
	})
	obj.SetName(name)

	if err := k8sClient.Delete(ctx, obj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete tenant: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": name})
}

// GetTenantMetricsHandler retrieves metrics for a tenant
func GetTenantMetricsHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		// Mocked metrics response
		c.JSON(http.StatusOK, gin.H{
			"tenant": name,
			"metrics": gin.H{
				"cpu_usage":                 "250m",
				"memory_usage":              "512Mi",
				"last_provisioning_seconds": 42.5,
				"active":                    true,
			},
		})
	}
}

// GetTenantKubeconfigHandler retrieves kubeconfig for a tenant
func GetTenantKubeconfigHandler(mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if mode == "k8s" {
			getTenantKubeconfigK8s(c, name)
		} else {
			getTenantKubeconfigMock(c, name)
		}
	}
}

func getTenantKubeconfigMock(c *gin.Context, name string) {
	secretPath := filepath.Join("..", "examples", "tenants", name+"-kubeconfig.secret")
	b, err := os.ReadFile(secretPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "kubeconfig not found"})
		return
	}
	c.Data(http.StatusOK, "text/plain", b)
}

func getTenantKubeconfigK8s(c *gin.Context, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.io",
		Version: "v1alpha1",
		Kind:    "Tenant",
	})

	if err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, obj); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	secretName, ok := status["adminKubeconfigSecret"].(string)
	if !ok || secretName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "kubeconfig secret not available"})
		return
	}

	// TODO: Fetch secret from Kubernetes
	c.JSON(http.StatusOK, gin.H{"secret": secretName})
}
