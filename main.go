package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/AbdullahOmar20/Chirpy/internal/database"
	"github.com/AbdullahOmar20/Chirpy/internal/auth"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func main(){
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil{
		log.Println("error while connecting to db", err)
	}
	
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

	
	apiConfig := apiConfig{
		dbQueries: database.New(db),
		platform: os.Getenv("PLATFORM"),
	}
	
	fileServer := http.FileServer(http.Dir("."))
	
	mux.Handle("/app/", http.StripPrefix("/app" ,apiConfig.midllewareMetricInc(fileServer)))
	mux.HandleFunc("GET /api/metrics", apiConfig.fileServerHitsHandler)
	mux.HandleFunc("POST /api/chirps", apiConfig.CreateChirpHander)
	mux.HandleFunc("GET /api/chirps", apiConfig.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiConfig.GetChirpByIdHandler)
	mux.HandleFunc("POST /api/users", apiConfig.createUserHandler)
	mux.HandleFunc("POST /api/login", apiConfig.LoginHandler)
	mux.HandleFunc("GET /admin/metrics", apiConfig.fileServerHitsAdminHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.deleteUsersAdminHandler)
	
	server.ListenAndServe()
}

type apiConfig struct{
	fileServerHits atomic.Int32
	dbQueries *database.Queries
	platform string
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
	ID        uuid.UUID	`json:"id"`
	CreatedAt time.Time	`json:"created_at"`
	UpdatedAt time.Time	`json:"updated_at"`
	Body      string	`json:"body"`
	UserID    uuid.UUID	`json:"user_id"`
}

func (cfg *apiConfig)CreateChirpHander(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()
	
	type responseBody struct{
		CleandBody string `json:"cleaned_body"`
	}
	type requestBody struct{
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(req.Body)

	chirpRequestBody := requestBody{}
	if err := decoder.Decode(&chirpRequestBody); err != nil{
		responseWithError(w, 500, "Something went wrong")
		return
	}

	if err := ValidateChirp(chirpRequestBody.Body); err != nil{
		responseWithError(w, 400, err.Error())
		return
	}

	chirp, err := cfg.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body: chirpRequestBody.Body,
		UserID: chirpRequestBody.UserId,
	})
	if err != nil{
		responseWithError(w, 400, "chirp exists")
		return
	}

	responseWithJson(w, 201, Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	})
}

func (cfg *apiConfig) GetChirpsHandler(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()

	chirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil{
		responseWithError(w, 500, "error fetching chirps")
	}

	chirpsResult := []Chirp{}
	for _, item := range chirps{
		chirpsResult = append(chirpsResult, Chirp{
			ID:  item.ID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			Body: item.Body,
			UserID: item.UserID,
		})
	}

	responseWithJson(w, 200, chirpsResult)
}

func (cfg *apiConfig) GetChirpByIdHandler(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()

	chirpIDString := req.PathValue("chirpID")

	if len(chirpIDString) == 0{
		responseWithError(w, 400, "Invalid Id")
		return
	}

	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil{
		responseWithError(w, 400, "Invalid Id")
		return
	}

	chirp, err := cfg.dbQueries.GetChirpById(req.Context(), chirpID)
	if err != nil{
		responseWithError(w, 404, "Chirp not found")
		return
	}

	result := Chirp{
		ID:  chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	}
	responseWithJson(w, 200, result)
}

func ValidateChirp(chirp string) error{
	if len(chirp) > 140{
		return errors.New("Chirp is too long")
	}

	return nil
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

type User struct{
	Id 			uuid.UUID 	`json:"id"`
	CreatedAt 	time.Time 	`json:"created_at"`
	UpdatedAt 	time.Time 	`json:"updated_at"`
	Email 		string		`json:"email"`
}
type userRequest struct{
	Email string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)

	userReq := userRequest{}
	if err := decoder.Decode(&userReq); err != nil{
		responseWithError(w, 500, "Something went wrong")
		return
	}

	hasedPassword, err := auth.HashPassword(userReq.Password)
	if err != nil{
		responseWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.dbQueries.CreateUser(req.Context(), database.CreateUserParams{
		Email: userReq.Email,
		HashedPassword: hasedPassword,
	})
	if err != nil{
		responseWithError(w, 400, "user already exists")
		return
	}

	userResult := User{
		Id: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	responseWithJson(w, 201, userResult)

}

func (cfg *apiConfig) deleteUsersAdminHandler(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()

	if cfg.platform != "dev"{
		responseWithError(w, 403, "Forbidden access")
		return
	}
	
	err := cfg.dbQueries.DeleteUsers(req.Context())
	if err != nil{
		responseWithError(w, 500, "error deleting users")
		return
	}

	responseWithJson(w, 200, "")
}

func (cfg *apiConfig) LoginHandler(w http.ResponseWriter, req *http.Request){
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)

	userReq := userRequest{}
	if err := decoder.Decode(&userReq); err != nil{
		responseWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.dbQueries.GetUserByEmail(req.Context(), userReq.Email)
	if err != nil{
		responseWithError(w, 401, "Incorrect email or password")
		return
	}

	passwordMatch, err := auth.CheckPasswordHash(userReq.Password, user.HashedPassword)
	if err != nil{
		responseWithError(w, 401, "Incorrect email or password")
		return
	}

	if passwordMatch == false{
		responseWithError(w, 401, "Incorrect email or password")
		return
	}

	userResult := User{
		Id: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	responseWithJson(w, 200, userResult)
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