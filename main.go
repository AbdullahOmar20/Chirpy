package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"strings"
	"slices"
)

func main(){
	mux := http.ServeMux{}
	server := http.Server{
		Handler: &mux,
		Addr: ":8080",
	}
	
	mux.HandleFunc("GET /api/healthz", func (writer http.ResponseWriter, request *http.Request){
		writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(200)
		// io.WriteString(writer, "OK")
		writer.Write([]byte("OK"))
	})

	
	apiConfig := apiConfig{}
	
	fileServer := http.FileServer(http.Dir("."))
	
	mux.Handle("/app/", http.StripPrefix("/app" ,apiConfig.midllewareMetricInc(fileServer)))
	mux.HandleFunc("GET /api/metrics", apiConfig.fileServerHitsHandler)
	mux.HandleFunc("POST /api/validate_chirp", ValidateChirpHander)
	mux.HandleFunc("GET /admin/metrics", apiConfig.fileServerHitsAdminHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.fileServerHitsResetHandler)
	
	server.ListenAndServe()
}

type apiConfig struct{
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) midllewareMetricInc(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request){
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, req)
	})
}
func (cfg *apiConfig) fileServerHitsHandler(writer http.ResponseWriter, request *http.Request){
	writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileServerHits.Load())))
}
func (cfg *apiConfig) fileServerHitsAdminHandler(writer http.ResponseWriter, request *http.Request){
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load())))
}
func (cfg *apiConfig) fileServerHitsResetHandler(writer http.ResponseWriter, request *http.Request){
	writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	cfg.fileServerHits.Store(0)
}

type Chirp struct {
	Body string `json:"body"`
}

func ValidateChirpHander(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()
	
	type responseBody struct{
		CleandBody string `json:"cleaned_body"`
	}
	decoder := json.NewDecoder(req.Body)

	chirp := Chirp{}
	if err := decoder.Decode(&chirp); err != nil{
		responseWithError(w, 500, "Something went wrong")
		return
	}

	if len(chirp.Body) > 140{
		responseWithError(w, 400, "Chirp is too long")
		return
	}

	responseWithJson(w, 200, responseBody{
		CleandBody: replaceBadWords(chirp.Body),
	})
}

func replaceBadWords(msg string) string{
	badWords := []string {"kerfuffle", "sharbert", "fornax"}

	split := strings.Split(msg, " ")
	for indx, s := range split{
		if slices.Contains(badWords, strings.ToLower(s)){
			split[indx] = "****"
		}
	}

	result := strings.Join(split, " ")
	return result
}

func responseWithJson(w http.ResponseWriter, code int, payload interface{}) error{
	json, err := json.Marshal(payload)
	if err != nil{
		return err
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(json)
	return nil
}
func responseWithError(w http.ResponseWriter, code int, msg string) error{
	return responseWithJson(w, code, map[string]string{"error": msg})
}