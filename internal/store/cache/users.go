package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mubashshir3767/currencyExchange/internal/store"
)

const UserExpTime = time.Minute

type UsersStore struct {
	rdb *redis.Client
}

func (s *UsersStore) Set(ctx context.Context, user *store.User) error {
	if user.ID == 0 {
		return fmt.Errorf("user id not found")
	}
	cacheKey := fmt.Sprintf("user-%v", user.ID)

	json, err := json.Marshal(user)
	if err != nil {
		return err
	}
	log.Println("CACHE SET METHOD USED")

	return s.rdb.Set(ctx, cacheKey, json, UserExpTime).Err()
}

func (s *UsersStore) Delete(ctx context.Context, userId int64) error {
	cacheKey := fmt.Sprintf("user-%v", userId)
	return s.rdb.Del(ctx, cacheKey).Err()
}

func (s *UsersStore) Get(ctx context.Context, userId int64) (*store.User, error) {
	cacheKey := fmt.Sprintf("user-%v", userId)
	log.Println("CACHE GET METHOD USED")

	data, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var user store.User
	if data != "" {
		err := json.Unmarshal([]byte(data), &user)
		if err != nil {
			return nil, err
		}
	}

	return &user, nil
}
