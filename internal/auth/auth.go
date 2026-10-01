package auth

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"microdashboard/internal/store"
)

type Auth struct {
	store *store.Store
}

func New(s *store.Store) *Auth {
	return &Auth{store: s}
}

func (a *Auth) HTTP(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			http.Error(w, "missing X-API-Key header", http.StatusUnauthorized)
			return
		}

		device, err := a.store.GetDeviceByKey(key)
		if err != nil {
			http.Error(w, "invalid API key", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "device", device)
		h(w, r.WithContext(ctx), p)
	}
}

func (a *Auth) Admin(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		token := r.URL.Query().Get("token")
		if token != a.store.GetAdminToken() {
			http.Error(w, "forbidden - invalid admin token", http.StatusForbidden)
			return
		}
		h(w, r, p)
	}
}

func DeviceFromContext(r *http.Request) (*store.DeviceRow, bool) {
	d, ok := r.Context().Value("device").(*store.DeviceRow)
	return d, ok
}