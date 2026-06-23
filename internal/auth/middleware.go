package auth

import (
	"context"
	"net/http"
)

type sessionProvider interface {
	UserFromRequest(r *http.Request) (User, Session, error)
}

func RequireLogin(svc sessionProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, session, err := svc.UserFromRequest(r)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			ctx = context.WithValue(ctx, contextKey("session"), session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
