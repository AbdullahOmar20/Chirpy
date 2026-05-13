package main

import(
	"net/http"
	"sync/atomic"
	"fmt"
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