package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dejobratic/tbd/internal/orders/app"
	"github.com/dejobratic/tbd/internal/orders/domain"
	"github.com/dejobratic/tbd/internal/orders/ports"
)

// Handler exposes HTTP endpoints for order operations.
type Handler struct {
	service *app.Service
}

// NewHandler constructs a Handler.
func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

// Register binds the order handlers to the provided ServeMux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/orders", h.handleOrders)
	mux.HandleFunc("/v1/orders/", h.handleOrderByID)
}

func (h *Handler) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createOrder(w, r)
	case http.MethodGet:
		h.listOrders(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/v1/orders/")
	if trimmed == "" {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	if strings.HasSuffix(trimmed, "/cancel") {
		id := strings.TrimSuffix(trimmed, "/cancel")
		id = strings.TrimSuffix(id, "/")
		if id == "" {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.cancelOrder(w, r, id)
		return
	}

	id := strings.TrimSuffix(trimmed, "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.getOrder(w, r, id)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header required")
		return
	}

	if stored, err := h.service.GetIdempotentResponse(ctx, idemKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if stored != nil {
		for key, values := range restoreHeaders(stored.StatusCode) {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(stored.StatusCode)
		_, _ = w.Write(stored.Body)
		return
	}

	var payload app.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	order, err := h.service.CreateOrder(ctx, payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]any{"order": order}
	body, err := json.Marshal(response)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stored := ports.StoredResponse{
		StatusCode: http.StatusAccepted,
		Body:       body,
		OrderID:    order.ID,
	}

	if err := h.service.SaveIdempotentResponse(ctx, idemKey, stored); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(body)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request, id string) {
	order, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	filter := ports.ListFilter{}
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		status := domain.OrderStatus(statusParam)
		filter.Status = &status
	}

	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if page, err := strconv.Atoi(pageParam); err == nil {
			filter.Page = page
		}
	}

	if pageSizeParam := r.URL.Query().Get("page_size"); pageSizeParam != "" {
		if pageSize, err := strconv.Atoi(pageSizeParam); err == nil {
			filter.PageSize = pageSize
		}
	}

	orders, err := h.service.ListOrders(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request, id string) {
	order, err := h.service.CancelOrder(r.Context(), id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

// restoreHeaders is a hook for replayed responses. For now it only sets content-type.
func restoreHeaders(status int) http.Header {
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	if status == http.StatusAccepted {
		header.Set("Retry-After", "0")
	}
	return header
}
