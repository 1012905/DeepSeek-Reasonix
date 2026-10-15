package serve

import "net/http"

func (s *Server) registerJobRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /jobs/{id}/cancel", s.cancelJob)
}

// cancelJob stops one of this session's background jobs. A job that already
// ended, or belongs to another session, is refused rather than acknowledged:
// the panel is showing a row that no longer answers to it.
func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	ctl, id := s.ctl(), r.PathValue("id")
	if !ctl.CancelJob(id) {
		// A job the kill was already asked for stays on the running list until its
		// process tree has gone, so "listed, and the kill refuses" is that wait.
		for _, j := range ctl.Jobs() {
			if j.ID == id {
				refuse(w, http.StatusConflict, "job.stop_in_progress", "the stop was already requested; the job is still shutting down", nil)
				return
			}
		}
		refuse(w, http.StatusConflict, "job.not_running", "no running background job with this id in this session", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
