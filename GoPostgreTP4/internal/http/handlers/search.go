package handlers

import "net/http"

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	result, err := h.service.SearchNotes(r.Context(), q)
	if err != nil {
		handleServiceError(w, h.logger, err)
		return
	}

	writeList(w, http.StatusOK, result.Notes, Meta{Total: result.Total})
}
