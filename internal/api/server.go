package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	router  *mux.Router
	address string
}

func NewServer() *Server {
	s := &Server{}

	router := mux.NewRouter()

	router.HandleFunc("/v1/kv/{key}", s.Get).Methods("GET")
	router.HandleFunc("/v1/kv/{key}", s.Set).Methods("PUT", "POST")
	router.HandleFunc("/v1/kv/{key}", s.Delete).Methods("DELETE")
	router.HandleFunc("/v1/kv", s.GetAll).Methods("GET")

	return s
}

func (s *Server) Run() error {
	return http.ListenAndServe(s.address, s.router)
}

func (s *Server) Get(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) Set(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) Delete(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) GetAll(w http.ResponseWriter, r *http.Request) {

}
