package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"task-manager/config"
	"task-manager/models"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisManager struct {
	client *redis.Client
	ctx    context.Context
	config *config.Config
}

var RedisClient *RedisManager

// InitRedis initializes Redis connection with retry logic
func InitRedis(cfg *config.Config) error {
	if cfg == nil {
		cfg = config.AppConfig
	}

	maxRetries := 5
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("🔄 Connecting to Redis (attempt %d/%d)...\n", attempt, maxRetries)

		client := redis.NewClient(&redis.Options{
			Addr:         cfg.GetRedisAddr(),
			Password:     cfg.RedisPassword,
			DB:           cfg.RedisDB,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 3,
		})

		ctx := context.Background()

		// Test connection
		_, err := client.Ping(ctx).Result()
		if err == nil {
			RedisClient = &RedisManager{
				client: client,
				ctx:    ctx,
				config: cfg,
			}
			fmt.Printf("✅ Redis connected successfully at %s\n", cfg.GetRedisAddr())
			return nil
		}

		fmt.Printf("⚠️  Redis connection failed: %v\n", err)

		if attempt < maxRetries {
			fmt.Printf("⏳ Retrying in %v...\n", retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("failed to connect to Redis after %d attempts", maxRetries)
}

// Health check for Redis
func (r *RedisManager) Ping() error {
	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	return r.client.Ping(ctx).Err()
}

// Close Redis connection
func (r *RedisManager) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// GetConnectionInfo returns connection information
func (r *RedisManager) GetConnectionInfo() map[string]interface{} {
	return map[string]interface{}{
		"address":  r.config.GetRedisAddr(),
		"database": r.config.RedisDB,
		"status":   "connected",
	}
}

// User operations
func (r *RedisManager) SaveUser(user *models.User) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%d", user.ID)
	err = r.client.Set(r.ctx, key, userJSON, 0).Err()
	if err != nil {
		return err
	}

	// Add to users index
	r.client.SAdd(r.ctx, "users:all", user.ID)

	// Add to email index
	r.client.Set(r.ctx, fmt.Sprintf("user:email:%s", user.Email), user.ID, 0)

	// Add to group indexes
	for _, groupID := range user.GroupIDs {
		r.client.SAdd(r.ctx, fmt.Sprintf("group:%d:users", groupID), user.ID)
	}

	return nil
}

func (r *RedisManager) GetUser(userID int) (*models.User, error) {
	key := fmt.Sprintf("user:%d", userID)
	userJSON, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	var user models.User
	err = json.Unmarshal([]byte(userJSON), &user)
	return &user, err
}

func (r *RedisManager) GetUserByEmail(email string) (*models.User, error) {
	userIDStr, err := r.client.Get(r.ctx, fmt.Sprintf("user:email:%s", email)).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, err
	}

	return r.GetUser(userID)
}

func (r *RedisManager) GetAllUsers() ([]*models.User, error) {
	userIDs, err := r.client.SMembers(r.ctx, "users:all").Result()
	if err != nil {
		return nil, err
	}

	var users []*models.User
	for _, userIDStr := range userIDs {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			continue
		}

		user, err := r.GetUser(userID)
		if err == nil {
			users = append(users, user)
		}
	}

	return users, nil
}

func (r *RedisManager) DeleteUser(userID int) error {
	// Get user first to remove from indexes
	user, err := r.GetUser(userID)
	if err != nil {
		return err
	}

	// Remove from indexes
	r.client.SRem(r.ctx, "users:all", userID)
	r.client.Del(r.ctx, fmt.Sprintf("user:email:%s", user.Email))

	for _, groupID := range user.GroupIDs {
		r.client.SRem(r.ctx, fmt.Sprintf("group:%d:users", groupID), userID)
	}

	// Delete user data
	key := fmt.Sprintf("user:%d", userID)
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisManager) SearchUsers(query string) ([]*models.User, error) {
	users, err := r.GetAllUsers()
	if err != nil {
		return nil, err
	}

	var results []*models.User
	lowerQuery := strings.ToLower(query)

	for _, user := range users {
		if strings.Contains(strings.ToLower(user.FullName), lowerQuery) ||
			strings.Contains(strings.ToLower(user.Email), lowerQuery) {
			results = append(results, user)
		}
	}

	return results, nil
}

// Group operations
func (r *RedisManager) SaveGroup(group *models.Group) error {
	groupJSON, err := json.Marshal(group)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("group:%d", group.ID)
	err = r.client.Set(r.ctx, key, groupJSON, 0).Err()
	if err != nil {
		return err
	}

	// Add to groups index
	r.client.SAdd(r.ctx, "groups:all", group.ID)

	// Add to admin index
	r.client.SAdd(r.ctx, fmt.Sprintf("user:%d:admin_groups", group.AdminID), group.ID)

	return nil
}

func (r *RedisManager) GetGroup(groupID int) (*models.Group, error) {
	key := fmt.Sprintf("group:%d", groupID)
	groupJSON, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, err
	}

	var group models.Group
	err = json.Unmarshal([]byte(groupJSON), &group)
	return &group, err
}

func (r *RedisManager) GetAllGroups() ([]*models.Group, error) {
	groupIDs, err := r.client.SMembers(r.ctx, "groups:all").Result()
	if err != nil {
		return nil, err
	}

	var groups []*models.Group
	for _, groupIDStr := range groupIDs {
		groupID, err := strconv.Atoi(groupIDStr)
		if err != nil {
			continue
		}

		group, err := r.GetGroup(groupID)
		if err == nil {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func (r *RedisManager) GetUserGroups(userID int) ([]*models.Group, error) {
	user, err := r.GetUser(userID)
	if err != nil {
		return nil, err
	}

	var groups []*models.Group
	for _, groupID := range user.GroupIDs {
		group, err := r.GetGroup(groupID)
		if err == nil {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func (r *RedisManager) GetGroupUsers(groupID int) ([]*models.User, error) {
	userIDs, err := r.client.SMembers(r.ctx, fmt.Sprintf("group:%d:users", groupID)).Result()
	if err != nil {
		return nil, err
	}

	var users []*models.User
	for _, userIDStr := range userIDs {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			continue
		}

		user, err := r.GetUser(userID)
		if err == nil {
			users = append(users, user)
		}
	}

	return users, nil
}

func (r *RedisManager) DeleteGroup(groupID int) error {
	// Remove from indexes
	r.client.SRem(r.ctx, "groups:all", groupID)

	// Get group first to remove from admin index
	group, err := r.GetGroup(groupID)
	if err == nil {
		r.client.SRem(r.ctx, fmt.Sprintf("user:%d:admin_groups", group.AdminID), groupID)
	}

	// Remove users from group index
	r.client.Del(r.ctx, fmt.Sprintf("group:%d:users", groupID))

	// Delete group data
	key := fmt.Sprintf("group:%d", groupID)
	return r.client.Del(r.ctx, key).Err()
}

// Task operations
func (r *RedisManager) SaveTask(task *models.Task) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("task:%d", task.ID)
	err = r.client.Set(r.ctx, key, taskJSON, 0).Err()
	if err != nil {
		return err
	}

	// Add to indexes
	r.client.SAdd(r.ctx, "tasks:all", task.ID)
	r.client.SAdd(r.ctx, fmt.Sprintf("user:%d:tasks", task.UserID), task.ID)
	r.client.SAdd(r.ctx, fmt.Sprintf("group:%d:tasks", task.GroupID), task.ID)

	return nil
}

func (r *RedisManager) GetTask(taskID int) (*models.Task, error) {
	key := fmt.Sprintf("task:%d", taskID)
	taskJSON, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, err
	}

	var task models.Task
	err = json.Unmarshal([]byte(taskJSON), &task)
	return &task, err
}

func (r *RedisManager) GetUserTasks(userID int) ([]*models.Task, error) {
	taskIDs, err := r.client.SMembers(r.ctx, fmt.Sprintf("user:%d:tasks", userID)).Result()
	if err != nil {
		return nil, err
	}

	var tasks []*models.Task
	for _, taskIDStr := range taskIDs {
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			continue
		}

		task, err := r.GetTask(taskID)
		if err == nil {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (r *RedisManager) GetGroupTasks(groupID int) ([]*models.Task, error) {
	taskIDs, err := r.client.SMembers(r.ctx, fmt.Sprintf("group:%d:tasks", groupID)).Result()
	if err != nil {
		return nil, err
	}

	var tasks []*models.Task
	for _, taskIDStr := range taskIDs {
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			continue
		}

		task, err := r.GetTask(taskID)
		if err == nil {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (r *RedisManager) DeleteTask(taskID int) error {
	// Get task first to remove from indexes
	task, err := r.GetTask(taskID)
	if err != nil {
		return err
	}

	// Remove from indexes
	r.client.SRem(r.ctx, "tasks:all", taskID)
	r.client.SRem(r.ctx, fmt.Sprintf("user:%d:tasks", task.UserID), taskID)
	r.client.SRem(r.ctx, fmt.Sprintf("group:%d:tasks", task.GroupID), taskID)

	// Delete task data
	key := fmt.Sprintf("task:%d", taskID)
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisManager) SearchTasks(query string) ([]*models.SearchTask, error) {
	taskIDs, err := r.client.SMembers(r.ctx, "tasks:all").Result()
	if err != nil {
		return nil, err
	}

	var results []*models.SearchTask
	lowerQuery := strings.ToLower(query)

	for _, taskIDStr := range taskIDs {
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			continue
		}

		task, err := r.GetTask(taskID)
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToLower(task.Title), lowerQuery) ||
			strings.Contains(strings.ToLower(task.Information), lowerQuery) {
			results = append(results, &models.SearchTask{
				UserID: task.UserID,
				Task:   *task,
			})
		}
	}

	return results, nil
}

// Counter operations
func (r *RedisManager) GetNextUserID() (int, error) {
	id, err := r.client.Incr(r.ctx, "counter:user_id").Result()
	return int(id), err
}

func (r *RedisManager) GetNextGroupID() (int, error) {
	id, err := r.client.Incr(r.ctx, "counter:group_id").Result()
	return int(id), err
}

func (r *RedisManager) GetNextTaskID() (int, error) {
	id, err := r.client.Incr(r.ctx, "counter:task_id").Result()
	return int(id), err
}

// Utility functions
func (r *RedisManager) SetLastSyncTime() error {
	return r.client.Set(r.ctx, "sync:last_time", time.Now().Unix(), 0).Err()
}

func (r *RedisManager) GetLastSyncTime() (time.Time, error) {
	timestamp, err := r.client.Get(r.ctx, "sync:last_time").Result()
	if err == redis.Nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(ts, 0), nil
}

func (r *RedisManager) MarkDirty(dataType string) error {
	return r.client.SAdd(r.ctx, "dirty:types", dataType).Err()
}

func (r *RedisManager) GetDirtyTypes() ([]string, error) {
	return r.client.SMembers(r.ctx, "dirty:types").Result()
}

func (r *RedisManager) ClearDirtyTypes() error {
	return r.client.Del(r.ctx, "dirty:types").Err()
}
