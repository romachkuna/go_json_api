package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"go-api/repository"
	"log"
	"net/http"
)

type apiFunction func(w http.ResponseWriter, r *http.Request) error

type APIServer struct {
	listenAddress string
	database      repository.PostgresDB
}

type JWTResponse struct {
	Token string `json:"api_jwt_token"`
}

type ApiError struct {
	Error string `json:"error"`
}

func NewAPIServer(address string) *APIServer {
	db, err := repository.NewPostgresDB()
	if err != nil {
		log.Fatal(err)
	}
	return &APIServer{listenAddress: address, database: *db}
}

func (s *APIServer) StartServer() {
	router := mux.NewRouter()
	router.HandleFunc("/signup", makeHandleFunction(s.handlePOSTSignUp))
	router.HandleFunc("/profile/{id}", withJWTAuth(makeHandleFunction(s.handleGETUser)))
	router.HandleFunc("/signin", makeHandleFunction(s.handlePOSTSignIn))
	router.HandleFunc("/createAccount/{id}", withJWTAuth(makeHandleFunction(s.handlePOSTAccount)))
	fmt.Println("Server Running...")

	log.Fatal(http.ListenAndServe(s.listenAddress, router))
}
func writeJson(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
func withJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		tokenString := r.Header.Get("api_jwt_token")
		userID, err := validateJWTToken(tokenString)
		if err != nil {
			http.Error(w, "Permission Denied", http.StatusBadRequest)
			return
		}
		if id == userID {
			handlerFunc(w, r)
			return
		} else {
			http.Error(w, "Permission Denied", http.StatusBadRequest)
		}
	}
}