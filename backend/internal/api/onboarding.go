package api

import (
	"errors"
	"net/http"

	"github.com/opale-app/opale/internal/store"
)

func (s *Server) onboardingError(w http.ResponseWriter, err error) {
	var validation *store.OnboardingValidationError
	switch {
	case errors.Is(err, store.ErrOnboardingConflict):
		writeError(w, http.StatusConflict, "onboarding_conflict", "La configuration a changé. Recharge le brouillon avant de continuer.")
	case errors.Is(err, store.ErrOnboardingUnavailable):
		writeError(w, http.StatusForbidden, "onboarding_unavailable", "La première configuration est réservée à un profil personnel.")
	case errors.As(err, &validation):
		writeError(w, http.StatusUnprocessableEntity, "invalid_onboarding", validation.Message)
	default:
		s.storeErr(w, err, "financial setup")
	}
}

func (s *Server) handleGetOnboarding(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.GetOnboarding(r.Context(), profileFromContext(r.Context()).ID)
	if err != nil {
		s.onboardingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleSaveOnboarding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExpectedRevision *int64                 `json:"expected_revision"`
		Step             *int                   `json:"step"`
		Draft            *store.OnboardingDraft `json:"draft"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	if err := decodeJSON(r, &req); err != nil || req.ExpectedRevision == nil || req.Step == nil || req.Draft == nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Révision, étape et brouillon valides requis.")
		return
	}
	state, err := s.store.SaveOnboarding(r.Context(), profileFromContext(r.Context()).ID, *req.ExpectedRevision, *req.Step, *req.Draft)
	if err != nil {
		s.onboardingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleFinishOnboarding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExpectedRevision *int64 `json:"expected_revision"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := decodeJSON(r, &req); err != nil || req.ExpectedRevision == nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Révision du brouillon requise.")
		return
	}
	profile := profileFromContext(r.Context()).ID
	var state store.OnboardingState
	var err error
	if r.URL.Path == "/v1/onboarding/skip" {
		state, err = s.store.SkipOnboarding(r.Context(), profile, *req.ExpectedRevision)
	} else {
		state, err = s.store.CompleteOnboarding(r.Context(), profile, *req.ExpectedRevision)
	}
	if err != nil {
		s.onboardingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}
