package redisclient

import (
	"context"
	"encoding/json"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/redis/go-redis/v9"
)

const (
	entityAgent       = "agent"
	allAgentsSetKey   = "agents:set"
	entityUser        = "user"
	entityChat        = "chat"
	allChatsSetKey    = "chats:set"
	entityChatHistory = "chat:history"
)

// RedisClientInterface defines the contract for our Data Access Layer (DAL) using Redis.
// It handles persistence for all entities in the system.
type RedisClientInterface interface {
	// Agent Management
	CreateAgent(ctx context.Context, agentConfig genaiconfig.AgentConfig) error
	GetAgent(ctx context.Context, agentID string) (*genaiconfig.AgentConfig, error)
	ListAgents(ctx context.Context) ([]*genaiconfig.AgentConfig, error)
	RemoveAgent(ctx context.Context, agentID string) error

	// User Management
	FindUserByID(ctx context.Context, userID string) (*genaiconfig.User, error)
	RemoveUser(ctx context.Context, userID string) error

	CreateChat(ctx context.Context, chatConfig genaiconfig.ChatConfig) error
	GetChat(ctx context.Context, chatID string) (*genaiconfig.ChatConfig, error)
	ListChats(ctx context.Context) ([]*genaiconfig.ChatConfig, error)
	RemoveChat(ctx context.Context, chatID string) error
	// Chat History Management
	SaveChatMessage(ctx context.Context, chatID string, message genaiconfig.ChatMessage) error
	GetChatHistory(ctx context.Context, chatID string) ([]genaiconfig.ChatMessage, error)
}

// RedisClient is the concrete implementation of the RedisClientInterface.
type RedisClient struct {
	client     *redis.Client
	isDisabled bool
}

// NewRedisClient is the constructor for the RedisClient.
func NewRedisClient(client *redis.Client, isDisabled bool) RedisClientInterface {
	return &RedisClient{
		client:     client,
		isDisabled: isDisabled,
	}
}

func (r *RedisClient) CreateAgent(ctx context.Context, agent genaiconfig.AgentConfig) error {
	if err := r.setJSON(ctx, generateKey(entityAgent, agent.ID), agent); err != nil {
		return err
	}
	return r.saveToSet(ctx, allAgentsSetKey, agent.ID)
}

func (r *RedisClient) GetAgent(ctx context.Context, agentID string) (*genaiconfig.AgentConfig, error) {
	key := generateKey(entityAgent, agentID)
	bytes, err := r.getJSONBytes(ctx, key)
	if err != nil {
		return nil, err
	}
	return getJSON[genaiconfig.AgentConfig](ctx, bytes)
}

func (r *RedisClient) ListAgents(ctx context.Context) ([]*genaiconfig.AgentConfig, error) {
	ids, err := r.getSetByKey(ctx, allAgentsSetKey)
	if err != nil {
		return nil, err
	}
	listBytes, err := r.listEntities(ctx, ids, allAgentsSetKey)
	if err != nil {
		return nil, err
	}
	return listEntitiesGeniric[genaiconfig.AgentConfig](ctx, listBytes, entityAgent)
}

func (r *RedisClient) RemoveAgent(ctx context.Context, agentID string) error {
	if err := r.deleteKeys(ctx, generateKey(entityAgent, agentID)); err != nil {
		return err
	}
	return r.removeFromSet(ctx, allAgentsSetKey, agentID)
}

// -----------------------------------------------------------
// User Management
// -----------------------------------------------------------

func (r *RedisClient) UpdateUserContext(ctx context.Context, userID string, userContext string) (*genaiconfig.User, error) {
	user := &genaiconfig.User{ID: userID, Context: userContext}
	if err := r.setJSON(ctx, generateKey(entityUser, userID), user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *RedisClient) FindUserByID(ctx context.Context, userID string) (*genaiconfig.User, error) {
	key := generateKey(entityUser, userID)
	bytes, err := r.getJSONBytes(ctx, key)
	if err != nil {
		return nil, err
	}
	return getJSON[genaiconfig.User](ctx, bytes)
}

func (r *RedisClient) RemoveUser(ctx context.Context, userID string) error {
	return r.deleteKeys(ctx, generateKey(entityUser, userID))
}

// -----------------------------------------------------------
// Chat Management
// -----------------------------------------------------------

func (r *RedisClient) CreateChat(ctx context.Context, chat genaiconfig.ChatConfig) error {
	if err := r.setJSON(ctx, generateKey(entityChat, chat.ID), chat); err != nil {
		return err
	}
	return r.saveToSet(ctx, allChatsSetKey, chat.ID)
}

func (r *RedisClient) GetChat(ctx context.Context, chatID string) (*genaiconfig.ChatConfig, error) {
	key := generateKey(entityChat, chatID)
	bytes, err := r.getJSONBytes(ctx, key)
	if err != nil {
		return nil, err
	}
	return getJSON[genaiconfig.ChatConfig](ctx, bytes)
}

func (r *RedisClient) ListChats(ctx context.Context) ([]*genaiconfig.ChatConfig, error) {
	ids, err := r.getSetByKey(ctx, allAgentsSetKey)
	if err != nil {
		return nil, err
	}
	listBytes, err := r.listEntities(ctx, ids, entityChat)
	if err != nil {
		return nil, err
	}
	return listEntitiesGeniric[genaiconfig.ChatConfig](ctx, listBytes, entityAgent)
}

func (r *RedisClient) RemoveChat(ctx context.Context, chatID string) error {
	keys := []string{
		generateKey(entityChat, chatID),
		generateKey(entityChatHistory, chatID),
	}
	if err := r.deleteKeys(ctx, keys...); err != nil {
		return err
	}
	return r.removeFromSet(ctx, allChatsSetKey, chatID)
}

// -----------------------------------------------------------
// Chat History
// -----------------------------------------------------------

func (r *RedisClient) SaveChatMessage(ctx context.Context, chatID string, msg genaiconfig.ChatMessage) error {
	if r.isDisabled {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return r.client.RPush(ctx, generateKey(entityChatHistory, chatID), data).Err()
}

func (r *RedisClient) GetChatHistory(ctx context.Context, chatID string) ([]genaiconfig.ChatMessage, error) {
	if r.isDisabled {
		return nil, nil
	}

	key := generateKey(entityChatHistory, chatID)
	values, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err == redis.Nil {
		return []genaiconfig.ChatMessage{}, nil
	}
	if err != nil {
		return nil, err
	}

	history := make([]genaiconfig.ChatMessage, 0, len(values))
	for _, val := range values {
		var msg genaiconfig.ChatMessage
		if err := json.Unmarshal([]byte(val), &msg); err == nil {
			history = append(history, msg)
		}
	}
	return history, nil
}
