package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"mioru/internal/model"
	"mioru/internal/store"
)

// MigrateUsersHandler returns an http.HandlerFunc that migrates users from Redis to PostgreSQL.
// POST /api/admin/migrate-users
func MigrateUsersHandler(redisStore *store.Store, pgStore *store.PostgresStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check if PostgreSQL users table is empty
		empty, err := pgStore.IsUsersTableEmpty(ctx)
		if err != nil {
			jsonError(w, "failed to check users table: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !empty {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":      true,
				"message": "users table is not empty, migration skipped",
				"count":   0,
			})
			return
		}

		// Get all Redis keys matching "user:*"
		keys, err := redisStore.Keys(ctx, "user:*")
		if err != nil {
			jsonError(w, "failed to read Redis keys: "+err.Error(), http.StatusInternalServerError)
			return
		}

		migrated := 0
		errors := []string{}

		for _, key := range keys {
			// Get raw JSON from Redis
			data, err := redisStore.GetRaw(ctx, key)
			if err != nil {
				errors = append(errors, "failed to get key "+key+": "+err.Error())
				continue
			}

			var u model.User
			if err := json.Unmarshal(data, &u); err != nil {
				errors = append(errors, "failed to unmarshal "+key+": "+err.Error())
				continue
			}

			// Set default role for migrated users
			if u.Role == "" {
				u.Role = "admin"
			}

			if err := pgStore.CreateUser(ctx, u); err != nil {
				errors = append(errors, "failed to insert "+u.Username+": "+err.Error())
				log.Printf("[MIGRATE] Error inserting user %s: %v", u.Username, err)
				continue
			}

			migrated++
			log.Printf("[MIGRATE] Migrated user: %s", u.Username)
		}

		log.Printf("[MIGRATE] Migration complete: %d users migrated, %d errors", migrated, len(errors))

		resp := map[string]interface{}{
			"ok":       true,
			"migrated": migrated,
			"total":    len(keys),
		}
		if len(errors) > 0 {
			resp["errors"] = errors
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
