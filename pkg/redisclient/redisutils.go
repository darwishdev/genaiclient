package redisclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// -----------------------------------------------------------
// Key Helpers
// -----------------------------------------------------------

func generateKey(entityType, id string) string {
	return fmt.Sprintf("%s:%s", entityType, id)
}

// -----------------------------------------------------------
// Base Helpers
// -----------------------------------------------------------

func (r *RedisClient) setJSON(ctx context.Context, key string, data interface{}) error {
	if r.isDisabled {
		return nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, bytes, 0).Err()
}

func (r *RedisClient) getJSONBytes(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}
func getJSON[T any](ctx context.Context, data []byte) (*T, error) {
	var obj T
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func (r *RedisClient) deleteKeys(ctx context.Context, keys ...string) error {
	if r.isDisabled {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) saveToSet(ctx context.Context, setKey, id string) error {
	if r.isDisabled {
		return nil
	}
	return r.client.SAdd(ctx, setKey, id).Err()
}

func (r *RedisClient) removeFromSet(ctx context.Context, setKey, id string) error {
	if r.isDisabled {
		return nil
	}
	return r.client.SRem(ctx, setKey, id).Err()
}

func (r *RedisClient) getSetByKey(ctx context.Context, setKey string) ([]string, error) {
	ids, err := r.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, err
	}
	return ids, nil
}
func listEntitiesGeniric[T any](ctx context.Context, data [][]byte, keyPrefix string) ([]*T, error) {
	results := make([]*T, len(data))
	for index, row := range data {
		obj, err := getJSON[T](ctx, row)
		if err == nil && obj != nil {
			results[index] = obj
		}
	}
	return results, nil
}
func (r *RedisClient) listEntities(ctx context.Context, ids []string, keyPrefix string) ([][]byte, error) {
	results := make([][]byte, len(ids))
	for index, id := range ids {
		bytes, err := r.getJSONBytes(ctx, generateKey(keyPrefix, id))
		if err == nil && bytes != nil {
			results[index] = bytes
		}
	}
	return results, nil
}
