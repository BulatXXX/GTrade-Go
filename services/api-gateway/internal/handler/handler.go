package handler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gtrade/services/api-gateway/internal/service"
)

type GatewayUseCase interface {
	Forward(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error)
}

type Handler struct {
	serviceName string
	gateway     GatewayUseCase
}

func New(serviceName string, gateway GatewayUseCase) *Handler {
	return &Handler{serviceName: serviceName, gateway: gateway}
}

func (h *Handler) ProxyAuth(c *gin.Context) {
	h.proxyTo(c, service.TargetAuth, wildcardPath(c))
}

func (h *Handler) ProxyUsers(c *gin.Context) {
	h.proxyTo(c, service.TargetUserAsset, wildcardPath(c))
}

func (h *Handler) ProxyCatalog(c *gin.Context) {
	h.proxyTo(c, service.TargetCatalog, fallbackPath("/items", wildcardPath(c)))
}

func (h *Handler) ProxyMarket(c *gin.Context) {
	h.proxyTo(c, service.TargetIntegration, wildcardPath(c))
}

func (h *Handler) ProxyNotifications(c *gin.Context) {
	h.proxyTo(c, service.TargetNotification, wildcardPath(c))
}

func (h *Handler) proxyTo(c *gin.Context, target, path string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read request body failed"})
		return
	}

	resp, err := h.gateway.Forward(c.Request.Context(), target, service.ForwardRequest{
		Method:   c.Request.Method,
		Path:     path,
		RawQuery: c.Request.URL.RawQuery,
		Headers:  forwardHeaders(c.Request.Header),
		Body:     body,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), service.ErrInvalidTarget.Error()):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gateway target is not configured"})
		default:
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream request failed"})
		}
		return
	}

	copyResponseHeaders(c.Writer.Header(), resp.Headers)
	c.Status(resp.StatusCode)
	_, _ = c.Writer.Write(resp.Body)
}

func wildcardPath(c *gin.Context) string {
	path := c.Param("path")
	if path == "" {
		return "/"
	}
	return path
}

func fallbackPath(basePath, wildcard string) string {
	if wildcard == "/" {
		return basePath
	}
	return basePath + wildcard
}

func forwardHeaders(headers http.Header) http.Header {
	forwarded := make(http.Header, len(headers))
	for key, values := range headers {
		switch http.CanonicalHeaderKey(key) {
		case "Host", "Content-Length":
			continue
		default:
			for _, value := range values {
				forwarded.Add(key, value)
			}
		}
	}
	return forwarded
}

func copyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		switch http.CanonicalHeaderKey(key) {
		case "Content-Length", "Transfer-Encoding", "Connection":
			continue
		default:
			for _, value := range values {
				dst.Add(key, value)
			}
		}
	}
}
