package store

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"mioru/internal/model"
)

const (
	userKey  = "user:"
	emailKey = "email:"
	notesKey = "notes"
	notesCh  = "notes_updates"
)

type Store struct {
	rdb *redis.Client
	mu  sync.RWMutex
}

func New(addr, pw string) *Store {
	// Support both plain addr:port and redis:// URLs (Upstash)
	opts := &redis.Options{}
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		// Parse Redis URL
		opt, err := redis.ParseURL(addr)
		if err != nil {
			log.Printf("[STORE] Failed to parse Redis URL: %v, falling back to addr:pw", err)
			opts.Addr = addr
			opts.Password = pw
		} else {
			opts = opt
		}
	} else {
		opts.Addr = addr
		opts.Password = pw
	}

	rdb := redis.NewClient(opts)
	return &Store{rdb: rdb}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}

// ── Users ──

func (s *Store) CreateUser(ctx context.Context, u model.User) error {
	exists, err := s.rdb.Exists(ctx, userKey+u.Username).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return errors.New("username already exists")
	}

	emailLower := strings.ToLower(u.Email)
	emailExists, err := s.rdb.Exists(ctx, emailKey+emailLower).Result()
	if err != nil {
		return err
	}
	if emailExists > 0 {
		return errors.New("email already registered")
	}

	b, _ := json.Marshal(u)
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, userKey+u.Username, b, 0)
	pipe.Set(ctx, emailKey+emailLower, u.Username, 0)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) GetUser(ctx context.Context, username string) (*model.User, error) {
	b, err := s.rdb.Get(ctx, userKey+username).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var u model.User
	json.Unmarshal(b, &u)
	return &u, nil
}

func (s *Store) UpdateUser(ctx context.Context, username string, updates map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.rdb.Get(ctx, userKey+username).Bytes()
	if err != nil {
		return err
	}
	var u model.User
	json.Unmarshal(b, &u)
	for k, v := range updates {
		switch k {
		case "display_name":
			u.DisplayName = v
		case "avatar_color":
			u.AvatarColor = v
		case "first_name":
			u.FirstName = v
		case "last_name":
			u.LastName = v
		}
	}
	b, _ = json.Marshal(u)
	return s.rdb.Set(ctx, userKey+username, b, 0).Err()
}

func (s *Store) UpdatePassword(ctx context.Context, username, hashedPW string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.rdb.Get(ctx, userKey+username).Bytes()
	if err != nil {
		return err
	}
	var u model.User
	json.Unmarshal(b, &u)
	u.HashedPW = hashedPW
	b, _ = json.Marshal(u)
	return s.rdb.Set(ctx, userKey+username, b, 0).Err()
}

// ── Notes ──

func (s *Store) GetUserNotes(ctx context.Context, username string) ([]model.Note, error) {
	ids, err := s.rdb.SMembers(ctx, notesKey).Result()
	if err != nil {
		return nil, err
	}
	var notes []model.Note
	for _, id := range ids {
		b, err := s.rdb.Get(ctx, "note:"+id).Bytes()
		if err != nil {
			continue
		}
		var n model.Note
		json.Unmarshal(b, &n)
		if n.Author == username {
			notes = append(notes, n)
		}
	}
	return notes, nil
}

func (s *Store) GetNote(ctx context.Context, id string) (*model.Note, error) {
	b, err := s.rdb.Get(ctx, "note:"+id).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var n model.Note
	json.Unmarshal(b, &n)
	return &n, nil
}

func (s *Store) CreateNote(ctx context.Context, n model.Note) error {
	b, _ := json.Marshal(n)
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, "note:"+n.ID, b, 0)
	pipe.SAdd(ctx, notesKey, n.ID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return s.publish("created", n)
}

func (s *Store) UpdateNote(ctx context.Context, id string, updates map[string]interface{}) (*model.Note, error) {
	b, err := s.rdb.Get(ctx, "note:"+id).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var n model.Note
	json.Unmarshal(b, &n)
	for k, v := range updates {
		switch k {
		case "content":
			if s, ok := v.(string); ok {
				n.Content = s
			}
		case "color":
			if s, ok := v.(string); ok {
				n.Color = s
			}
		case "position_x":
			if f, ok := v.(float64); ok {
				n.PosX = int(f)
			}
		case "position_y":
			if f, ok := v.(float64); ok {
				n.PosY = int(f)
			}
		}
	}
	b, _ = json.Marshal(n)
	if err := s.rdb.Set(ctx, "note:"+id, b, 0).Err(); err != nil {
		return nil, err
	}
	s.publish("updated", n)
	return &n, nil
}

func (s *Store) DeleteNote(ctx context.Context, id string) error {
	b, err := s.rdb.Get(ctx, "note:"+id).Bytes()
	if err == redis.Nil {
		return errors.New("note not found")
	}
	if err != nil {
		return err
	}
	var n model.Note
	json.Unmarshal(b, &n)
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, "note:"+id)
	pipe.SRem(ctx, notesKey, id)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return s.publish("deleted", n)
}

func (s *Store) publish(action string, n model.Note) error {
	msg := model.WSMessage{Action: action, Note: &n}
	b, _ := json.Marshal(msg)
	return s.rdb.Publish(context.Background(), notesCh, b).Err()
}

func (s *Store) SubscribeNotes() *redis.PubSub {
	return s.rdb.Subscribe(context.Background(), notesCh)
}

func (s *Store) GetAllNoteIDs(ctx context.Context) ([]string, error) {
	return s.rdb.SMembers(ctx, notesKey).Result()
}

func (s *Store) GetRawNote(ctx context.Context, id string) ([]byte, error) {
	return s.rdb.Get(ctx, "note:"+id).Bytes()
}

// ── Password Reset Tokens ──

const resetKey = "reset:"

func (s *Store) CreateResetToken(ctx context.Context, email, token string) error {
	emailLower := strings.ToLower(email)
	username, err := s.rdb.Get(ctx, emailKey+emailLower).Result()
	if err == redis.Nil {
		return nil // email not found — silently succeed for security
	}
	if err != nil {
		return err
	}
	// Store token -> username mapping with 1 hour TTL
	return s.rdb.Set(ctx, resetKey+token, username, time.Hour).Err()
}

// Keys returns all Redis keys matching pattern (for migration).
func (s *Store) Keys(ctx context.Context, pattern string) ([]string, error) {
	return s.rdb.Keys(ctx, pattern).Result()
}

// GetRaw returns the raw value for a Redis key (for migration).
func (s *Store) GetRaw(ctx context.Context, key string) ([]byte, error) {
	return s.rdb.Get(ctx, key).Bytes()
}

func (s *Store) ConsumeResetToken(ctx context.Context, token string) (string, error) {
	username, err := s.rdb.Get(ctx, resetKey+token).Result()
	if err == redis.Nil {
		return "", errors.New("invalid or expired token")
	}
	if err != nil {
		return "", err
	}
	// One-time use: delete token
	s.rdb.Del(ctx, resetKey+token)
	return username, nil
}
