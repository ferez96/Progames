package web

import (
	"net/http"

	"progames/internal/auth"
	"progames/internal/obs"
)

func (fe *Frontend) home(w http.ResponseWriter, r *http.Request) {
	if _, _, err := fe.authSvc.UserFromRequest(r); err == nil {
		http.Redirect(w, r, "/practice", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (fe *Frontend) signupForm(w http.ResponseWriter, r *http.Request) {
	fe.render(w, r, "Sign Up", "signup", nil)
}

func (fe *Frontend) signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fe.renderStatus(w, r, http.StatusUnprocessableEntity, "Sign Up", "signup", map[string]any{"Error": err.Error()})
		return
	}
	if _, err := fe.authSvc.SignUp(r.FormValue("name"), r.FormValue("email"), r.FormValue("password")); err != nil {
		fe.renderStatus(w, r, http.StatusUnprocessableEntity, "Sign Up", "signup", map[string]any{"Error": err.Error()})
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (fe *Frontend) loginForm(w http.ResponseWriter, r *http.Request) {
	fe.render(w, r, "Login", "login", nil)
}

func (fe *Frontend) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fe.renderStatus(w, r, http.StatusUnprocessableEntity, "Login", "login", map[string]any{"Error": err.Error()})
		return
	}
	_, sessionID, _, err := fe.authSvc.SignIn(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		obs.LoginsFailure.Add(1)
		fe.renderStatus(w, r, http.StatusUnprocessableEntity, "Login", "login", map[string]any{"Error": "invalid email or password"})
		return
	}
	obs.LoginsSuccess.Add(1)
	fe.authSvc.SetSessionCookie(w, r, sessionID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/practice")
		return
	}
	http.Redirect(w, r, "/practice", http.StatusSeeOther)
}

func (fe *Frontend) logout(w http.ResponseWriter, r *http.Request) {
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		_ = fe.authSvc.SignOut(cookie.Value)
	}
	auth.ClearSessionCookie(w)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
