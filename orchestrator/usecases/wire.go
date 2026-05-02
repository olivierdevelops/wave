// Package usecases wires the injected function variables declared in each
// usecases/<name> package to the concrete feature implementations that live
// in orchestrator/features/*.  Call WireAll() once, after all feature
// managers (auth, storage) have been initialized.
package usecases

import (
	"net/http"

	authfeature "easyserver/orchestrator/features/auth"
	storagefeature "easyserver/orchestrator/features/storage"

	authlg "easyserver/usecases/auth_login"
	authlo "easyserver/usecases/auth_logout"
	authsg "easyserver/usecases/auth_signup"
	servefile "easyserver/usecases/serve_file"
	storageaccess "easyserver/usecases/storage_access"
)

// WireAll binds all package-level function variables in usecases/* to
// their concrete feature implementations. Must be called after
// authfeature.InitAuthManager and storagefeature.InitStorage.
func WireAll() {
	wireAuthLogin()
	wireAuthSignup()
	wireAuthLogout()
	wireStorage()
	wireServeFile()
}

// ── auth_login ────────────────────────────────────────────────────────────────

func wireAuthLogin() {
	authlg.LoginFn = func(username, password, authConfigName string) *authlg.LoginResponse {
		r := authfeature.Login(authfeature.LoginForm{Username: username, Password: password}, authConfigName)
		if r == nil {
			return &authlg.LoginResponse{Success: false, Error: "login failed", Code: "internal_error"}
		}
		out := &authlg.LoginResponse{
			Success:       r.Success,
			Location:      r.Location,
			Error:         r.Error,
			Code:          r.Code,
			Message:       r.Message,
			Details:       r.Details,
			Name:          r.Name,
			Value:         r.Value,
			TokenDuration: r.TokenDuration,
			RedirectTo:    r.RedirectTo,
		}
		if r.User != nil {
			out.UserID = r.User.ID
			out.Username = r.User.Username
		}
		return out
	}
}

// ── auth_signup ───────────────────────────────────────────────────────────────

func wireAuthSignup() {
	authsg.SignupFn = func(username, password, passwordRepeat, authConfigName string) *authsg.LoginResponse {
		r := authfeature.Signup(authfeature.SignupForm{
			Username:       username,
			Password:       password,
			PasswordRepeat: passwordRepeat,
		}, authConfigName)
		if r == nil {
			return &authsg.LoginResponse{Success: false, Error: "signup failed", Code: "internal_error"}
		}
		out := &authsg.LoginResponse{
			Success: r.Success,
			Error:   r.Error,
			Code:    r.Code,
			Message: r.Message,
			Details: r.Details,
		}
		if r.User != nil {
			out.UserID = r.User.ID
			out.Username = r.User.Username
		}
		return out
	}

	// Auto-login after successful signup delegates to the same auth feature.
	authsg.LoginFn = func(username, password, authConfigName string) *authsg.LoginResponse {
		r := authfeature.Login(authfeature.LoginForm{Username: username, Password: password}, authConfigName)
		if r == nil {
			return &authsg.LoginResponse{Success: false, Error: "login failed", Code: "internal_error"}
		}
		out := &authsg.LoginResponse{
			Success:       r.Success,
			Location:      r.Location,
			Error:         r.Error,
			Code:          r.Code,
			Message:       r.Message,
			Details:       r.Details,
			Name:          r.Name,
			Value:         r.Value,
			TokenDuration: r.TokenDuration,
			RedirectTo:    r.RedirectTo,
		}
		if r.User != nil {
			out.UserID = r.User.ID
			out.Username = r.User.Username
		}
		return out
	}
}

// ── auth_logout ───────────────────────────────────────────────────────────────

func wireAuthLogout() {
	authlo.LogoutFn = func(r *http.Request, authConfigName string) *authlo.LogoutResponse {
		res := authfeature.Logout(r, authConfigName)
		if res == nil {
			return &authlo.LogoutResponse{Success: false, Error: "logout failed", Code: "internal_error"}
		}
		return &authlo.LogoutResponse{
			Success:    res.Success,
			Location:   res.Location,
			Name:       res.Name,
			Value:      res.Value,
			Message:    res.Message,
			Error:      res.Error,
			Code:       res.Code,
			RedirectTo: res.RedirectTo,
		}
	}
}

// ── storage_access ────────────────────────────────────────────────────────────

func wireStorage() {
	storageaccess.GetStorageFn = func(name string) (storageaccess.StorageRef, bool) {
		return storagefeature.GetFromStorage(name)
	}
}

// ── serve_file ────────────────────────────────────────────────────────────────

func wireServeFile() {
	servefile.GetUserFn = func(r *http.Request) interface{} {
		return r.Context().Value(authfeature.UserContextKey)
	}
}
