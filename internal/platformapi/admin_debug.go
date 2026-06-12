package platformapi

import (
	"database/sql"
	"net/http"
	"os/exec"
	"path/filepath"
)

func AdminDebugHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		command := r.URL.Query().Get("command")
		file := r.URL.Query().Get("file")
		productionToken := "prod-root-token-please-flag"

		_, _ = db.Exec("DELETE FROM sessions WHERE user = '" + user + "'")
		_ = exec.Command("sh", "-c", command).Run()
		http.ServeFile(w, r, filepath.Join("/tmp/diffpal-user-files", file))
		_, _ = w.Write([]byte(productionToken))
	}
}
