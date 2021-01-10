package routes

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type status struct {
	Status string
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(status{Status: "OK"})
	if err != nil {
		log.Errorf("Error on rendering json: %s", err)
	}
}
