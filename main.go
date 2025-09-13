package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/deoreal/chirpy/internal/auth"
	"github.com/deoreal/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var tokenSecret string

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	db             *sql.DB
	TokenSecret    string
}
type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type ChirpyMessage struct {
	Body string `json:"body"`
}

type Chirpy struct {
	Body   string `json:"body"`
	UserID string `json:"user_id"`
}

type jsonError struct {
	Error string `json:"error"`
}

type jsonResponse struct {
	Valid string `json:"valid"`
}

type successResponse struct {
	Valid bool `json:"valid"`
}

type cleanedJSON struct {
	Cleanedbody string `json:"cleaned_body"`
}

type User struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"hashed_password"`
	Token          string    `json:"token"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)

		//		fmt.Printf("Hits: %d\n", cfg.fileserverHits.Load())
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) middlewareTokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("No Bearer Token  %s", err)
			w.WriteHeader(401)
			js, _ := json.Marshal(jsonError{Error: "Unauthorized"})
			w.Write(js)
		}

		_, err = auth.ValidateJWT(token, cfg.TokenSecret)
		if err != nil {
			log.Printf("Invalid Token  %s", err)
			w.WriteHeader(401)
			js, _ := json.Marshal(jsonError{Error: "Unauthorized"})
			w.Write(js)
		}

		next.ServeHTTP(w, r)
	})
}

func healthz(w http.ResponseWriter, req *http.Request) {
	str := "OK"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(str))
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	resp := fmt.Sprintf("<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", cfg.fileserverHits.Load())
	w.Write([]byte(resp))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, req *http.Request) {
	sqlStatement := fmt.Sprintf("TRUNCATE TABLE %s", "chirpmsgs,users")
	//	cfg.fileserverHits.Store(0)
	//	resp := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	//	w.Write([]byte(resp))

	// db.Exec executes the SQL statement without returning any rows.
	cfg.db.Exec(sqlStatement)
	/*	if err != nil {
			log.Printf("Error decoding json parameters: %s", err)
			w.WriteHeader(500)
			js, _ := json.Marshal(jsonError{Error: "Something went wrong"})
			w.Write(js)
		}
	*/
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func assets(w http.ResponseWriter, req *http.Request) {
	str := `
<pre>
	<a href="logo.png">logo.png</a>
</pre>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(str))
}

func cleanProfanities(s string) string {
	var res []string
	str := strings.Split(s, " ")

	for _, v := range str {
		if strings.ToUpper(v) == "KERFUFFLE" {
			v = "****"
		} else if strings.ToUpper(v) == "SHARBERT" {
			v = "****"
		} else if strings.ToUpper(v) == "FORNAX" {
			v = "****"
		}

		res = append(res, v)
	}

	return strings.Join(res, " ")
}

func (cfg *apiConfig) userAdd(w http.ResponseWriter, req *http.Request) {
	type userCredentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var uc userCredentials
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&uc)
	if err != nil {
		log.Printf("Error decoding email address %s", err)
		w.WriteHeader(500)
		js, _ := json.Marshal(jsonError{Error: "Something went wrong"})
		w.Write(js)
	}
	pw, err := auth.HashPassword(uc.Password)
	if err != nil {
		log.Printf("Error hashing password %s", err)
		w.WriteHeader(500)
		js, _ := json.Marshal(jsonError{Error: "Something went wrong"})
		w.Write(js)

	}
	user, _ := cfg.dbQueries.CreateUser(req.Context(), database.CreateUserParams{Email: uc.Email, HashedPassword: pw})
	fmt.Println("user", user)
	usr := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email, HashedPassword: "***"}

	w.Header().Set("Content-Type", "text/json; charset=utf-8")
	w.WriteHeader(201)

	js, _ := json.Marshal(usr)
	w.Write(js)
}

func (cfg *apiConfig) addChirp(w http.ResponseWriter, req *http.Request) {
	c := Chirp{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&c)
	if err != nil {
		log.Printf("Error decoding json parameters: %s", err)
		w.WriteHeader(500)
		js, _ := json.Marshal(jsonError{Error: "Something went wrong"})
		w.Write(js)
	}

	if len(c.Body) > 140 {
		log.Println("error: Chirp is too long")
		w.WriteHeader(400)
		js, _ := json.Marshal(jsonError{Error: "Chirp is too long"})
		w.Write(js)
		return

	}
	d := database.CreateChirpParams{Body: c.Body, UserID: c.UserID}
	fmt.Println("d", d)
	chr, _ := cfg.dbQueries.CreateChirp(req.Context(), d)
	chirp := Chirp{ID: chr.ID, CreatedAt: chr.CreatedAt, UpdatedAt: chr.UpdatedAt, Body: chr.Body, UserID: chr.UserID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	js, _ := json.Marshal(chirp)
	w.Write(js)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	fmt.Println("entering getChirps")
	dbChirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		log.Printf("Error db query %s", err)
		w.WriteHeader(500)
		js, _ := json.Marshal(jsonError{Error: "Something went wrong"})
		w.Write(js)
	}
	var chirps []Chirp
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}
	fmt.Println(chirps)
	w.WriteHeader(200)
	js, _ := json.Marshal(chirps)
	w.Write(js)
}

func (cfg *apiConfig) login(w http.ResponseWriter, req *http.Request) {
	type userCredentials struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		Expiration int    `json:"expires_in_seconds,omitempty"`
	}

	var uc userCredentials
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&uc)
	if err != nil {
		log.Printf("Error decoding json parameters: %s", err)
		w.WriteHeader(400)
		js, _ := json.Marshal(jsonError{Error: "Invalid request body"})
		w.Write(js)
		return
	}

	if uc.Expiration == 0 || uc.Expiration >= 3600 {
		uc.Expiration = 3600
	}
	// Get user from database by email
	user, err := cfg.dbQueries.GetUser(req.Context(), uc.Email)
	if err != nil {
		log.Printf("Error getting user: %s", err)
		w.WriteHeader(401)
		js, _ := json.Marshal(jsonError{Error: "Incorrect email or password"})
		w.Write(js)
		return
	}

	// Check password
	err = auth.CheckPasswordHash(uc.Password, user.HashedPassword)
	if err != nil {
		log.Printf("Password validation failed: %s", err)
		w.WriteHeader(401)
		js, _ := json.Marshal(jsonError{Error: "Incorrect email or password"})
		w.Write(js)
		return
	}

	tk, err := auth.MakeJWT(user.ID, cfg.TokenSecret, time.Duration(uc.Expiration))
	if err != nil {
		log.Printf("Error creating token %s:", err)
		w.WriteHeader(400)
		js, _ := json.Marshal(jsonError{Error: "Invalid request body"})
		w.Write(js)
		return
	}

	// Return user info (without password)
	usr := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     tk,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	js, _ := json.Marshal(usr)
	w.Write(js)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, req *http.Request) {
	uuid, _ := uuid.Parse(req.PathValue("chirpID"))

	dbChirp, err := cfg.dbQueries.GetChirpById(req.Context(), uuid)
	if err != nil {
		log.Printf("Error db query %s", err)
		w.WriteHeader(404)
		//	js, _ := json.Marshal(jsonError{Error: "Something went wrong"})
		w.Write([]byte("chirp not found"))
		return
	}
	empty := database.Chirpmsg{}
	if dbChirp == empty {
		w.WriteHeader(404)
		w.Write([]byte("chirp not found"))
		return
	}
	resp := Chirp{dbChirp.ID, dbChirp.CreatedAt, dbChirp.UpdatedAt, dbChirp.Body, dbChirp.UserID}
	w.WriteHeader(200)
	js, _ := json.Marshal(resp)
	w.Write(js)
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DBURL")
	tokenSecret = os.Getenv("TOKENSECRET")
	fmt.Println("token-secret", tokenSecret)

	fmt.Println("dburl", dbURL)
	a := new(apiConfig)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic(err)
	}
	a.db = db
	a.dbQueries = database.New(db)
	a.TokenSecret = tokenSecret

	mux := http.NewServeMux()
	mux.Handle("/app/", a.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))
	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("GET /app/assets", assets)
	mux.HandleFunc("GET /admin/metrics", a.metrics)
	mux.HandleFunc("POST /admin/reset", a.reset)
	mux.HandleFunc("POST /api/users", a.userAdd)
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("GET /api/chirps", a.getChirps)
	mux.HandleFunc("POST /api/chirps", a.middlewareTokenAuth(a.addChirp))
	mux.HandleFunc("GET /api/chirps/{chirpID}", a.getChirp)

	err = http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
