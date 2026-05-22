package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Bajusz15/go-backend-home-assignment/internal/middleware"
	"github.com/Bajusz15/go-backend-home-assignment/internal/service"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authService *service.AuthService
	validate    *validator.Validate
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validate:    validator.New(),
	}
}

// Register godoc
// @Summary      Register a new customer
// @Description  Creates a new customer account and returns an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      service.RegisterInput  true  "Registration details"
// @Success      201   {object}  service.AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, formatValidationError(err))
		return
	}

	resp, err := h.authService.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login godoc
// @Summary      Authenticate a user
// @Description  Authenticates a user and returns an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      service.LoginInput  true  "Login credentials"
// @Success      200   {object}  service.AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, formatValidationError(err))
		return
	}

	resp, err := h.authService.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// WhoAmI godoc
// @Summary      Get current user
// @Description  Returns the currently authenticated user
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.User
// @Failure      401  {object}  ErrorResponse
// @Router       /auth/who-am-i [get]
func (h *AuthHandler) WhoAmI(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	user, err := h.authService.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func formatValidationError(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			switch fe.Tag() {
			case "required":
				return fe.Field() + " is required"
			case "email":
				return fe.Field() + " must be a valid email address"
			case "min":
				return fe.Field() + " must be at least " + fe.Param() + " characters"
			case "max":
				return fe.Field() + " must be at most " + fe.Param() + " characters"
			case "uuid":
				return fe.Field() + " must be a valid UUID"
			}
		}
	}
	return "validation failed"
}
