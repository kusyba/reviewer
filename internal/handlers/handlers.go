package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) errorResponse(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	})
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		h.errorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateTeam(&team); err != nil {
		switch err {
		case repository.ErrTeamExists:
			h.errorResponse(w, "TEAM_EXISTS", "team_name already exists", http.StatusBadRequest)
		default:
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"team": team,
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.errorResponse(w, "INVALID_REQUEST", "team_name is required", http.StatusBadRequest)
		return
	}

	team, err := h.service.GetTeam(teamName)
	if err != nil {
		if err == repository.ErrNotFound {
			h.errorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req models.SetActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.service.SetUserActive(req.UserID, req.IsActive)
	if err != nil {
		if err == repository.ErrNotFound {
			h.errorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.service.CreatePR(&req)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			h.errorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		case repository.ErrPRExists:
			h.errorResponse(w, "PR_EXISTS", "PR id already exists", http.StatusConflict)
		default:
			log.Printf("Error creating PR: %v", err)
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req models.MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.service.MergePR(req.PullRequestID)
	if err != nil {
		if err == repository.ErrNotFound {
			h.errorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req models.ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	pr, newUserID, err := h.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			h.errorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		case repository.ErrPRMerged:
			h.errorResponse(w, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
		case repository.ErrNotAssigned:
			h.errorResponse(w, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
		case repository.ErrNoCandidate:
			h.errorResponse(w, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
		default:
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr":          pr,
		"replaced_by": newUserID,
	})
}

func (h *Handler) GetUserPRs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.errorResponse(w, "INVALID_REQUEST", "user_id is required", http.StatusBadRequest)
		return
	}

	response, err := h.service.GetUserPRs(userID)
	if err != nil {
		if err == repository.ErrNotFound {
			h.errorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			h.errorResponse(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (h *Handler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/team/add", h.CreateTeam).Methods("POST")
	router.HandleFunc("/team/get", h.GetTeam).Methods("GET")
	router.HandleFunc("/users/setIsActive", h.SetUserActive).Methods("POST")
	router.HandleFunc("/users/getReview", h.GetUserPRs).Methods("GET")
	router.HandleFunc("/pullRequest/create", h.CreatePR).Methods("POST")
	router.HandleFunc("/pullRequest/merge", h.MergePR).Methods("POST")
	router.HandleFunc("/pullRequest/reassign", h.ReassignReviewer).Methods("POST")
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")

	return router
}
