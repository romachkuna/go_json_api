package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"time"
)

func makeHandleFunction(f apiFunction) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			apiError := writeJson(w, http.StatusBadRequest, ApiError{Error: fmt.Sprint(err)})
			_ = fmt.Sprint(err)
			if apiError != nil {
				return
			}
		}
	}
}

func verifyRequestMethod(method string, w http.ResponseWriter, r *http.Request) bool {
	if r.Method != method {
		err := writeJson(w, http.StatusBadRequest, ApiError{Error: "Wrong request method"})
		if err != nil {
			return false
		}
		return false
	}
	return true
}
func (s *APIServer) handleGETUser(w http.ResponseWriter, r *http.Request) error {
	if verifyRequestMethod("GET", w, r) {
		id := mux.Vars(r)["id"]
		user, err := s.database.getUserById(id)
		if err != nil {
			return err
		}
		apiError := writeJson(w, http.StatusOK, user)
		if apiError != nil {
			return apiError
		}
		return nil
	}
	return nil
}
func (s *APIServer) handlePOSTSignUp(w http.ResponseWriter, r *http.Request) error {
	if verifyRequestMethod("POST", w, r) {
		user := User{}
		body := r.Body
		defer func(body io.ReadCloser) {
			err := body.Close()
			if err != nil {
				return
			}
		}(body)

		err := json.NewDecoder(body).Decode(&user)
		if err != nil {
			wrtJsonError := writeJson(w, http.StatusBadRequest, ApiError{Error: err.Error()})
			if wrtJsonError != nil {
				return wrtJsonError
			}
			return nil
		}
		if user.Name == "" || user.Surname == "" || user.ID == "" || user.Password == "" {
			wrtJsonError := writeJson(w, http.StatusBadRequest, ApiError{Error: "Name, Surname, ID, and Password fields are required."})
			if wrtJsonError != nil {
				return wrtJsonError
			}
			return nil
		}

		user.RegistrationDate = time.Now()

		err = s.database.insertUser(&user)
		if err != nil {
			return err
		}
		err = writeJson(w, http.StatusCreated, &user)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (s *APIServer) handlePOSTSignIn(w http.ResponseWriter, r *http.Request) error {
	if verifyRequestMethod("POST", w, r) {
		body := r.Body
		defer func(body io.ReadCloser) {
			err := body.Close()
			if err != nil {
				return
			}
		}(body)

		var values map[string]string

		err := json.NewDecoder(body).Decode(&values)
		if err != nil {
			wrtJsonError := writeJson(w, http.StatusBadRequest, ApiError{Error: err.Error()})
			if wrtJsonError != nil {
				return wrtJsonError
			}
			return nil
		}
		id, okid := values["id"]
		password, okpassword := values["password"]

		if okid && okpassword {
			login := s.database.loginUser(id, password)
			if login {
				jwtToken, jwtError := createNewJWTToken(id)
				if jwtError != nil {
					return jwtError
				}
				wrtJsonError := writeJson(w, http.StatusFound, JWTResponse{Token: jwtToken})
				if wrtJsonError != nil {
					return wrtJsonError
				}
				return nil
			}
		} else {
			jsonErr := writeJson(w, http.StatusBadRequest, ApiError{Error: "Authorization Denied"})
			if jsonErr != nil {
				return jsonErr
			}
			return nil
		}
		return nil
	}
	return nil

}

func (s *APIServer) handlePOSTAccount(w http.ResponseWriter, r *http.Request) error {
	return nil
}
