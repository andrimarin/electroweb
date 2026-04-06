package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

// Estructura de la Receta basada en tus requerimientos
type Ciclo struct {
	TrabajoSeg int `json:"trabajo_seg"`
	PausaSeg   int `json:"pausa_seg"`
}

type Receta struct {
	ID           string  `json:"receta_id"`
	NumCiclos    int     `json:"num_ciclos"`
	Ciclos       []Ciclo `json:"ciclos"`
	FrecuenciaHz int     `json:"frecuencia_hz"`
	AnchoPulsoMs float64 `json:"ancho_pulso_ms"`
	Intensidad   int     `json:"intensidad_ma"` // 0-255
}

// Estado en línea (Memoria)
var (
	currentReceta  Receta
	currentCommand string     = "NONE" // NONE, START, STOP,
	mu             sync.Mutex          // Para evitar colisiones en cambios en línea
)

const fileName = "receta.json"

// Cargar receta desde el archivo al iniciar
func loadReceta() {
	file, err := os.ReadFile(fileName)
	if err != nil {
		// Si no existe, creamos una por defecto
		currentReceta = Receta{ID: "terapia_01", NumCiclos: 1, FrecuenciaHz: 50, AnchoPulsoMs: 0.5, Intensidad: 0}
		saveReceta()
		return
	}
	json.Unmarshal(file, &currentReceta)
}

func saveReceta() {
	data, _ := json.MarshalIndent(currentReceta, "", "  ")
	os.WriteFile(fileName, data, 0644)
}

func handleSetReceta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	json.NewDecoder(r.Body).Decode(&currentReceta)
	saveReceta()
	fmt.Fprintf(w, "Receta actualizada y guardada")
}

func handleSetCommand(w http.ResponseWriter, r *http.Request) {
	// Permite al especialista enviar START o STOP
	cmd := r.URL.Query().Get("cmd")
	mu.Lock()
	currentCommand = cmd
	mu.Unlock()
	fmt.Fprintf(w, "Comando %s enviado al dispositivo", cmd)
}

func handleGetReceta(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentReceta)
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	// El dispositivo envía su estatus y recibe comandos
	mu.Lock()
	defer mu.Unlock()

	// Responder con el comando pendiente y la intensidad actual
	response := map[string]interface{}{
		"command":   currentCommand,
		"intensity": currentReceta.Intensidad,
	}

	// Una vez enviado el comando, lo reseteamos a NONE para no repetirlo
	if currentCommand != "NONE" {
		log.Printf("Dispositivo recibió comando: %s", currentCommand)
		currentCommand = "NONE"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleLog(w http.ResponseWriter, r *http.Request) {
	// Recibe el resultado de la operación
	var logData map[string]interface{}
	json.NewDecoder(r.Body).Decode(&logData)
	fmt.Printf("LOG RECIBIDO: %v\n", logData)
	w.WriteHeader(http.StatusOK)
}

func main() {
	loadReceta()

	// Crear un nuevo multiplexor de rutas (ServeMux)
	mux := http.NewServeMux()

	// Registrar los endpoints de la API en el mux
	mux.HandleFunc("/api/receta", handleGetReceta)
	mux.HandleFunc("/api/heartbeat", handleHeartbeat)
	mux.HandleFunc("/api/log", handleLog)
	mux.HandleFunc("/admin/set-receta", handleSetReceta)
	mux.HandleFunc("/admin/command", handleSetCommand)

	// Servir archivos estáticos para cualquier otra ruta
	fs := http.FileServer(http.Dir("."))
	mux.Handle("/", fs)

	fmt.Println("🚀 Servidor TENS iniciado en http://localhost:8080")
	// Usar el mux personalizado en lugar del DefaultServeMux
	log.Fatal(http.ListenAndServe(":8080", mux))
}
