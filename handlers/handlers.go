package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"wallet-ledger/models"
	"wallet-ledger/service"

	"github.com/go-chi/chi/v5"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

type Handler struct {
	service *service.WalletService
	repo    interface{ Ping() error }
}

func New(service *service.WalletService, repo interface{ Ping() error }) *Handler {
	return &Handler{
		service: service,
		repo:    repo,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// PurchaseRequest represents a purchase request
type PurchaseRequest struct {
	PackageCode    string `json:"package_code"`
	IdempotencyKey string `json:"idempotency_key"`
}

// WagerRequest represents a wager request
type WagerRequest struct {
	StakeGC        int64  `json:"stake_gc,omitempty"`
	PayoutGC       int64  `json:"payout_gc,omitempty"`
	StakeSC        int64  `json:"stake_sc,omitempty"`
	PayoutSC       int64  `json:"payout_sc,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

// RedeemRequest represents a redeem request
type RedeemRequest struct {
	AmountSC       int64  `json:"amount_sc"`
	IdempotencyKey string `json:"idempotency_key"`
}

// GetUser handles GET /users/:id
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.service.GetUserWithBalances(userID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ListTransactions handles GET /users/:id/transactions
func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Parse query parameters
	cursor := r.URL.Query().Get("cursor")
	var cursorPtr *string
	if cursor != "" {
		cursorPtr = &cursor
	}

	limit := DefaultPageLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid limit: must be a number")
			return
		}
		if parsedLimit <= 0 {
			respondError(w, http.StatusBadRequest, "invalid limit: must be positive")
			return
		}
		if parsedLimit > MaxPageLimit {
			respondError(w, http.StatusBadRequest, "invalid limit: maximum is 100")
			return
		}
		limit = parsedLimit
	}

	var txType *models.TransactionType
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		t := models.TransactionType(typeStr)
		// Validate transaction type
		if t != models.TransactionTypePurchase &&
			t != models.TransactionTypeWagerGC &&
			t != models.TransactionTypeWinGC &&
			t != models.TransactionTypeWagerSC &&
			t != models.TransactionTypeWinSC &&
			t != models.TransactionTypeRedeemSC {
			respondError(w, http.StatusBadRequest, "invalid transaction type")
			return
		}
		txType = &t
	}

	var currency *models.Currency
	if currencyStr := r.URL.Query().Get("currency"); currencyStr != "" {
		c := models.Currency(currencyStr)
		// Validate currency
		if c != models.CurrencyGC && c != models.CurrencySC {
			respondError(w, http.StatusBadRequest, "invalid currency: must be GC or SC")
			return
		}
		currency = &c
	}

	transactions, err := h.service.ListTransactions(userID, cursorPtr, limit, txType, currency)
	if err != nil {
		log.Printf("Error listing transactions: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}

	respondJSON(w, http.StatusOK, transactions)
}

// Purchase handles POST /users/:id/purchase
func (h *Handler) Purchase(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PackageCode == "" {
		respondError(w, http.StatusBadRequest, "package_code is required")
		return
	}

	if req.IdempotencyKey == "" {
		respondError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}

	transactions, err := h.service.Purchase(userID, req.PackageCode, req.IdempotencyKey)
	if err != nil {
		log.Printf("Error processing purchase: %v", err)

		// Check if it's a business logic error
		if errors.Is(err, service.ErrInvalidPackage) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondError(w, http.StatusInternalServerError, "failed to process purchase")
		return
	}

	respondJSON(w, http.StatusOK, transactions)
}

// Wager handles POST /users/:id/wager
func (h *Handler) Wager(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req WagerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.IdempotencyKey == "" {
		respondError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}

	transactions, err := h.service.Wager(userID, req.StakeGC, req.PayoutGC, req.StakeSC, req.PayoutSC, req.IdempotencyKey)
	if err != nil {
		log.Printf("Error processing wager: %v", err)

		// Check if it's a business logic error (insufficient funds, invalid input)
		if errors.Is(err, service.ErrInsufficientFunds) || errors.Is(err, service.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondError(w, http.StatusInternalServerError, "failed to process wager")
		return
	}

	respondJSON(w, http.StatusOK, transactions)
}

// Redeem handles POST /users/:id/redeem
func (h *Handler) Redeem(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req RedeemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AmountSC <= 0 {
		respondError(w, http.StatusBadRequest, "amount_sc must be positive")
		return
	}

	if req.IdempotencyKey == "" {
		respondError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}

	transaction, err := h.service.Redeem(userID, req.AmountSC, req.IdempotencyKey)
	if err != nil {
		log.Printf("Error processing redemption: %v", err)

		// Check if it's a business logic error (insufficient funds, invalid input)
		if errors.Is(err, service.ErrInsufficientFunds) || errors.Is(err, service.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondError(w, http.StatusInternalServerError, "failed to process redemption")
		return
	}

	respondJSON(w, http.StatusOK, transaction)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := h.repo.Ping(); err != nil {
		respondError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// ListPackages handles GET /packages
func (h *Handler) ListPackages(w http.ResponseWriter, r *http.Request) {
	packages := make([]models.Package, 0, len(models.Packages))
	for _, pkg := range models.Packages {
		packages = append(packages, pkg)
	}
	respondJSON(w, http.StatusOK, packages)
}

// SetupRoutes configures all routes
func (h *Handler) SetupRoutes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)

	// Health check
	r.Get("/health", h.HealthCheck)

	// List available packages
	r.Get("/packages", h.ListPackages)

	// User routes
	r.Route("/users/{id}", func(r chi.Router) {
		r.Get("/", h.GetUser)
		r.Get("/transactions", h.ListTransactions)
		r.Post("/purchase", h.Purchase)
		r.Post("/wager", h.Wager)
		r.Post("/redeem", h.Redeem)
	})

	return r
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
