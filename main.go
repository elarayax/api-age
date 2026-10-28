package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "aoe2-api/docs"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "modernc.org/sqlite"
)

// Modelos de datos
type Civilization struct {
	ID         int    `json:"id" example:"1"`
	Name       string `json:"name" example:"Británicos"`
	Class      string `json:"class" example:"Arqueros"`
	UniqueUnit string `json:"unique_unit" example:"Arquero de tiro largo"`
	Specialty  string `json:"specialty" example:"Alcance extra en sus arqueros"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Civilización no encontrada"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operación exitosa"`
}

var db *sql.DB

// @title          AoE2 Civilizations API
// @version        1.0
// @description    API REST para gestionar civilizaciones de Age of Empires II.
// @host           localhost:8080
// @BasePath
func main() {
	var err error
	db, err = sql.Open("sqlite", "./aoe2.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	// Endpoint para Swagger UI
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Endpoints de la API
	http.HandleFunc("/api/v1/civilizations", handleCivilizations)
	http.HandleFunc("/api/v1/civilizations/", handleCivilizationByID)

	fmt.Println("Servidor iniciado en http://localhost:8080")
	fmt.Println("Documentación Swagger en http://localhost:8080/swagger/index.html")

	// ENVOLVER TODO EL ROUTER CON EL MIDDLEWARE DE CORS:
	log.Fatal(http.ListenAndServe(":8080", enableCORS(http.DefaultServeMux)))
}

func initDB() {
	query := `
	CREATE TABLE IF NOT EXISTS civilizations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		class TEXT NOT NULL,
		unique_unit TEXT NOT NULL,
		specialty TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Error al inicializar la base de datos: %v", err)
	}
}

func handleCivilizations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getCivilizations(w, r)
	case http.MethodPost:
		createCivilization(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func handleCivilizationByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/civilizations/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error": "ID inválido"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getCivilizationByID(w, r, id)
	case http.MethodPut:
		updateCivilizationPUT(w, r, id)
	case http.MethodPatch:
		updateCivilizationPATCH(w, r, id)
	case http.MethodDelete:
		deleteCivilization(w, r, id)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// getCivilizations godoc
// @Summary      Obtener todas las civilizaciones
// @Description  Retorna la lista completa de civilizaciones guardadas
// @Tags         Civilizaciones
// @Produce      json
// @Success      200  {array}   Civilization
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/civilizations [get]
func getCivilizations(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, class, unique_unit, specialty FROM civilizations")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	civs := []Civilization{}
	for rows.Next() {
		var c Civilization
		if err := rows.Scan(&c.ID, &c.Name, &c.Class, &c.UniqueUnit, &c.Specialty); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		civs = append(civs, c)
	}

	json.NewEncoder(w).Encode(civs)
}

// getCivilizationByID godoc
// @Summary      Obtener una civilización por ID
// @Description  Busca y retorna una civilización específica
// @Tags         Civilizaciones
// @Produce      json
// @Param        id   path      int  true  "ID de la Civilización"
// @Success      200  {object}  Civilization
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/civilizations/{id} [get]
func getCivilizationByID(w http.ResponseWriter, r *http.Request, id int) {
	var c Civilization
	query := "SELECT id, name, class, unique_unit, specialty FROM civilizations WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&c.ID, &c.Name, &c.Class, &c.UniqueUnit, &c.Specialty)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Civilización no encontrada"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(c)
}

// createCivilization godoc
// @Summary      Crear una civilización
// @Description  Agrega una nueva civilización a la base de datos
// @Tags         Civilizaciones
// @Accept       json
// @Produce      json
// @Param        civilization  body      Civilization  true  "Datos de la civilización"
// @Success      201           {object}  Civilization
// @Failure      400           {object}  ErrorResponse
// @Router       /api/v1/civilizations [post]
func createCivilization(w http.ResponseWriter, r *http.Request) {
	var c Civilization
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, `{"error": "JSON inválido"}`, http.StatusBadRequest)
		return
	}

	query := "INSERT INTO civilizations (name, class, unique_unit, specialty) VALUES (?, ?, ?, ?)"
	res, err := db.Exec(query, c.Name, c.Class, c.UniqueUnit, c.Specialty)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	c.ID = int(id)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

// updateCivilizationPUT godoc
// @Summary      Reemplazar completamente una civilización (PUT)
// @Description  Actualiza todos los campos de una civilización por su ID
// @Tags         Civilizaciones
// @Accept       json
// @Produce      json
// @Param        id            path      int           true  "ID de la Civilización"
// @Param        civilization  body      Civilization  true  "Nuevos datos completos"
// @Success      200           {object}  Civilization
// @Failure      400           {object}  ErrorResponse
// @Failure      404           {object}  ErrorResponse
// @Router       /api/v1/civilizations/{id} [put]
func updateCivilizationPUT(w http.ResponseWriter, r *http.Request, id int) {
	var c Civilization
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, `{"error": "JSON inválido"}`, http.StatusBadRequest)
		return
	}

	query := "UPDATE civilizations SET name = ?, class = ?, unique_unit = ?, specialty = ? WHERE id = ?"
	res, err := db.Exec(query, c.Name, c.Class, c.UniqueUnit, c.Specialty, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error": "Civilización no encontrada"}`, http.StatusNotFound)
		return
	}

	c.ID = id
	json.NewEncoder(w).Encode(c)
}

// updateCivilizationPATCH godoc
// @Summary      Actualizar parcialmente una civilización (PATCH)
// @Description  Actualiza solo los campos enviados en el cuerpo JSON
// @Tags         Civilizaciones
// @Accept       json
// @Produce      json
// @Param        id      path      int     true  "ID de la Civilización"
// @Param        fields  body      object  true  "Campos a actualizar (ej: {\"specialty\": \"Nueva especialidad\"})"
// @Success      200     {object}  Civilization
// @Failure      400     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Router       /api/v1/civilizations/{id} [patch]
func updateCivilizationPATCH(w http.ResponseWriter, r *http.Request, id int) {
	var fields map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		http.Error(w, `{"error": "JSON inválido"}`, http.StatusBadRequest)
		return
	}

	if len(fields) == 0 {
		http.Error(w, `{"error": "Sin campos para actualizar"}`, http.StatusBadRequest)
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	allowedFields := map[string]bool{"name": true, "class": true, "unique_unit": true, "specialty": true}

	for field, val := range fields {
		if allowedFields[field] {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
			args = append(args, val)
		}
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE civilizations SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	res, err := db.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error": "Civilización no encontrada"}`, http.StatusNotFound)
		return
	}

	getCivilizationByID(w, r, id)
}

// deleteCivilization godoc
// @Summary      Eliminar una civilización
// @Description  Borra una civilización de la base de datos por su ID
// @Tags         Civilizaciones
// @Produce      json
// @Param        id   path      int  true  "ID de la Civilización"
// @Success      200  {object}  MessageResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/civilizations/{id} [delete]
func deleteCivilization(w http.ResponseWriter, r *http.Request, id int) {
	res, err := db.Exec("DELETE FROM civilizations WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error": "Civilización no encontrada"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Civilización eliminada exitosamente"}`))
}

// enableCORS envuelve un Handler de http y le añade las cabeceras CORS
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Permitir cualquier origen (para desarrollo)
		// En producción reemplaza "*" por tu dominio, ej: "https://midominio.com"
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Métodos HTTP permitidos
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		// Cabeceras permitidas en las solicitudes
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Manejar las peticiones preflight (OPTIONS)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pasar la petición al siguiente handler
		next.ServeHTTP(w, r)
	})
}
