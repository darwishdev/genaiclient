package genaiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"google.golang.org/adk/session"
)

// RedisSessionService implements session.Service backed by Redis
type RedisSessionService struct {
	client *redis.Client
	ttl    time.Duration // optional TTL for session keys
}

func NewRedisSessionService(client *redis.Client, ttl time.Duration) session.Service {
	return &RedisSessionService{client: client, ttl: ttl}
}

// Keys
func sessionKey(appName, userID, sessionID string) string {
	return fmt.Sprintf("sess:%s:%s:%s", appName, userID, sessionID)
}
func eventsKey(appName, userID, sessionID string) string {
	return fmt.Sprintf("%s:events", sessionKey(appName, userID, sessionID))
}
func stateKey(appName, userID, sessionID string) string {
	return fmt.Sprintf("%s:state", sessionKey(appName, userID, sessionID))
}
func userStateKey(app, user string) string { return fmt.Sprintf("sess:%s:%s:state", app, user) }
func appStateKey(app string) string        { return fmt.Sprintf("sess:%s:state", app) }

// Create a new session
func (s *RedisSessionService) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	if req.AppName == "" || req.UserID == "" || req.SessionID == "" {
		return nil, fmt.Errorf("app_name, user_id, and session_id are required")
	}

	now := time.Now().UTC()
	key := sessionKey(req.AppName, req.UserID, req.SessionID)

	// Store session metadata as hash
	if err := s.client.HSet(ctx, key, map[string]interface{}{
		"id":        req.SessionID,
		"appName":   req.AppName,
		"userID":    req.UserID,
		"updatedAt": now.Format(time.RFC3339Nano),
	}).Err(); err != nil {
		return nil, err
	}

	if s.ttl > 0 {
		s.client.Expire(ctx, key, s.ttl)
	}

	sess := &redisSession{
		id:        req.SessionID,
		appName:   req.AppName,
		userID:    req.UserID,
		events:    []*session.Event{},
		state:     make(map[string]any),
		updatedAt: now,
	}

	return &session.CreateResponse{Session: sess}, nil
}

// Get session from Redis
func (s *RedisSessionService) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	key := sessionKey(req.AppName, req.UserID, req.SessionID)
	log.Debug().Str("key", key).Msg("Fetching session key")

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		// create session if not exists
		createReq := &session.CreateRequest{
			AppName:   req.AppName,
			UserID:    req.UserID,
			SessionID: req.SessionID,
		}
		resp, err := s.Create(ctx, createReq)
		if err != nil {
			return nil, err
		}
		return &session.GetResponse{Session: resp.Session}, nil
	}

	// Load metadata
	meta, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	sessionStateMap := make(map[string]any)
	sFields, _ := s.client.HGetAll(ctx, stateKey(req.AppName, req.UserID, req.SessionID)).Result()
	for k, v := range sFields {
		sessionStateMap[k] = v
	}

	// load user-level state
	userStateMap := make(map[string]any)
	uFields, _ := s.client.HGetAll(ctx, userStateKey(req.AppName, req.UserID)).Result()
	for k, v := range uFields {
		userStateMap[k] = v
	}

	// load app-level state
	appStateMap := make(map[string]any)
	aFields, _ := s.client.HGetAll(ctx, appStateKey(req.AppName)).Result()
	for k, v := range aFields {
		appStateMap[k] = v
	}

	// merge states: app <- user <- session
	mergedState := make(map[string]any)
	for k, v := range appStateMap {
		mergedState[k] = v
	}
	for k, v := range userStateMap {
		mergedState[k] = v
	}
	for k, v := range sessionStateMap {
		mergedState[k] = v
	}

	// load last 10 events
	rawEvents, _ := s.client.LRange(ctx, eventsKey(req.AppName, req.UserID, req.SessionID), -10, -1).Result()
	eventsList := make([]*session.Event, 0, len(rawEvents))

	updatedAt, _ := time.Parse(time.RFC3339Nano, meta["updatedAt"])
	for _, e := range rawEvents {
		var ev session.Event
		if err := json.Unmarshal([]byte(e), &ev); err != nil {
			continue
		}
		eventsList = append(eventsList, &ev)
	}
	sess := &redisSession{
		id:        meta["id"],
		appName:   meta["appName"],
		userID:    meta["userID"],
		state:     mergedState,
		events:    eventsList,
		updatedAt: updatedAt,
	}

	return &session.GetResponse{Session: sess}, nil
}

// AppendEvent adds an event and updates state
func (s *RedisSessionService) AppendEvent(ctx context.Context, sess session.Session, event *session.Event) error {
	rsess, ok := sess.(*redisSession)
	if !ok {
		return fmt.Errorf("invalid session type")
	}
	if event.Partial {
		return nil
	}

	rsess.mu.Lock()
	defer rsess.mu.Unlock()

	// merge state changes
	sessionDelta := make(map[string]any)
	userDelta := make(map[string]any)
	appDelta := make(map[string]any)

	for k, v := range event.Actions.StateDelta {
		if strings.HasPrefix(k, session.KeyPrefixTemp) {
			continue
		}
		rsess.state[k] = v
		sessionDelta[k] = v
		userDelta[k] = v // for simplicity, can separate
		appDelta[k] = v
	}

	// persist state to Redis
	if len(sessionDelta) > 0 {
		s.client.HSet(ctx, stateKey(rsess.appName, rsess.userID, rsess.id), sessionDelta)
		s.client.HSet(ctx, userStateKey(rsess.appName, rsess.userID), userDelta)
		s.client.HSet(ctx, appStateKey(rsess.appName), appDelta)
	}

	// append event to Redis
	data, _ := json.Marshal(event)
	s.client.RPush(ctx, eventsKey(rsess.appName, rsess.userID, rsess.id), data)

	rsess.events = append(rsess.events, event)
	rsess.updatedAt = event.Timestamp

	// update metadata
	s.client.HSet(ctx, sessionKey(rsess.appName, rsess.userID, rsess.id),
		"updatedAt", rsess.updatedAt.Format(time.RFC3339Nano))

	// TTL
	if s.ttl > 0 {
		s.client.Expire(ctx, sessionKey(rsess.appName, rsess.userID, rsess.id), s.ttl)
	}

	return nil
}

// List all sessions for a user
func (s *RedisSessionService) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	pattern := fmt.Sprintf("sess:%s:%s:*", req.AppName, req.UserID)
	keys, _ := s.client.Keys(ctx, pattern).Result()

	sessions := make([]session.Session, 0, len(keys))
	for _, k := range keys {
		resp, err := s.Get(ctx, &session.GetRequest{
			AppName:   req.AppName,
			UserID:    req.UserID,
			SessionID: strings.Split(k, ":")[3],
		})
		if err != nil {
			continue
		}
		sessions = append(sessions, resp.Session)
	}

	return &session.ListResponse{Sessions: sessions}, nil
}

// Delete session
func (s *RedisSessionService) Delete(ctx context.Context, req *session.DeleteRequest) error {
	s.client.Del(ctx, sessionKey(req.AppName, req.UserID, req.SessionID))
	s.client.Del(ctx, eventsKey(req.AppName, req.UserID, req.SessionID))
	s.client.Del(ctx, stateKey(req.AppName, req.UserID, req.SessionID))
	return nil
}

// --- Session object
type redisSession struct {
	mu        sync.RWMutex
	id        string
	appName   string
	userID    string
	events    []*session.Event
	state     map[string]any
	updatedAt time.Time
}

func (s *redisSession) ID() string                { return s.id }
func (s *redisSession) AppName() string           { return s.appName }
func (s *redisSession) UserID() string            { return s.userID }
func (s *redisSession) LastUpdateTime() time.Time { return s.updatedAt }
func (s *redisSession) State() session.State      { return &redisState{state: s.state} }
func (s *redisSession) Events() session.Events    { return &redisEvents{events: s.events} }

// --- State
type redisState struct {
	state map[string]any
}

func (s *redisState) Get(key string) (any, error) {
	v, ok := s.state[key]
	if !ok {
		return nil, session.ErrStateKeyNotExist
	}
	return v, nil
}
func (s *redisState) Set(key string, value any) error {
	s.state[key] = value
	return nil
}
func (s *redisState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.state {
			if !yield(k, v) {
				return
			}
		}
	}
}

// --- Events
type redisEvents struct {
	events []*session.Event
}

func (e *redisEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, ev := range e.events {
			if !yield(ev) {
				return
			}
		}
	}
}
func (e *redisEvents) At(i int) *session.Event {
	if i >= 0 && i < len(e.events) {
		return e.events[i]
	}
	return nil
}
func (e *redisEvents) Len() int {
	return len(e.events)
}
