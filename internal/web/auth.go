package web

import (
	"net/http"

	"progames/internal/auth"
	"progames/internal/obs"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if _, _, err := s.auth.UserFromRequest(r); err == nil {
		http.Redirect(w, r, "/practice", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) signupForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "Sign Up", "signup", nil)
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, r, "Sign Up", "signup", map[string]any{"Error": err.Error()})
		return
	}
	if _, err := s.auth.SignUp(r.FormValue("name"), r.FormValue("email"), r.FormValue("password")); err != nil {
		s.render(w, r, "Sign Up", "signup", map[string]any{"Error": err.Error()})
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "Login", "login", nil)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, r, "Login", "login", map[string]any{"Error": err.Error()})
		return
	}
	_, sessionID, _, err := s.auth.SignIn(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		obs.LoginsFailure.Add(1)
		s.render(w, r, "Login", "login", map[string]any{"Error": "invalid email or password"})
		return
	}
	obs.LoginsSuccess.Add(1)
	s.auth.SetSessionCookie(w, r, sessionID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/practice")
		return
	}
	http.Redirect(w, r, "/practice", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		_ = s.auth.SignOut(cookie.Value)
	}
	auth.ClearSessionCookie(w)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
