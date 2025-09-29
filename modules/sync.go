package modules

import (
	"fmt"
	"log"
	"task-manager/models"
	"time"
)

type SyncService struct {
	syncInterval time.Duration
	stopChan     chan bool
	running      bool
}

var Syncer *SyncService

func InitSyncService() {
	Syncer = &SyncService{
		syncInterval: 15 * time.Minute,
		stopChan:     make(chan bool, 1),
		running:      false,
	}
}

func (s *SyncService) Start() {
	if s.running {
		return
	}

	s.running = true
	go s.syncLoop()
	fmt.Println("🔄 Sync service started (15 minute interval)")
}

func (s *SyncService) Stop() {
	if !s.running {
		return
	}

	s.stopChan <- true
	s.running = false
	fmt.Println("⏹️ Sync service stopped")
}

func (s *SyncService) syncLoop() {
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	if err := s.performSync(); err != nil {
		log.Printf("❌ Initial sync failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.performSync(); err != nil {
				log.Printf("❌ Sync failed: %v", err)
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *SyncService) performSync() error {
	log.Println("🔄 Starting sync from Redis to PostgreSQL...")
	startTime := time.Now()

	dirtyTypes, err := RedisClient.GetDirtyTypes()
	if err != nil {
		return fmt.Errorf("failed to get dirty types: %v", err)
	}

	if len(dirtyTypes) == 0 {
		log.Println("✅ No changes detected, sync skipped")
		return nil
	}

	syncStats := make(map[string]int)

	if contains(dirtyTypes, "users") || len(dirtyTypes) == 0 {
		count, err := s.syncUsers()
		if err != nil {
			return fmt.Errorf("failed to sync users: %v", err)
		}
		syncStats["users"] = count
	}

	if contains(dirtyTypes, "groups") || len(dirtyTypes) == 0 {
		count, err := s.syncGroups()
		if err != nil {
			return fmt.Errorf("failed to sync groups: %v", err)
		}
		syncStats["groups"] = count
	}

	if contains(dirtyTypes, "tasks") || len(dirtyTypes) == 0 {
		count, err := s.syncTasks()
		if err != nil {
			return fmt.Errorf("failed to sync tasks: %v", err)
		}
		syncStats["tasks"] = count
	}

	if err := s.syncCounters(); err != nil {
		log.Printf("⚠️ Failed to sync counters: %v", err)
	}

	if err := RedisClient.ClearDirtyTypes(); err != nil {
		log.Printf("⚠️ Failed to clear dirty types: %v", err)
	}

	if err := RedisClient.SetLastSyncTime(); err != nil {
		log.Printf("⚠️ Failed to set last sync time: %v", err)
	}

	duration := time.Since(startTime)
	log.Printf("✅ Sync completed in %v - Users: %d, Groups: %d, Tasks: %d",
		duration, syncStats["users"], syncStats["groups"], syncStats["tasks"])

	return nil
}

func (s *SyncService) syncUsers() (int, error) {
	users, err := RedisClient.GetAllUsers()
	if err != nil {
		return 0, err
	}

	if err := PostgresClient.SyncUsers(users); err != nil {
		return 0, err
	}

	return len(users), nil
}

func (s *SyncService) syncGroups() (int, error) {
	groups, err := RedisClient.GetAllGroups()
	if err != nil {
		return 0, err
	}

	if err := PostgresClient.SyncGroups(groups); err != nil {
		return 0, err
	}

	return len(groups), nil
}

func (s *SyncService) syncTasks() (int, error) {
	users, err := RedisClient.GetAllUsers()
	if err != nil {
		return 0, err
	}

	var allTasks []*models.Task
	for _, user := range users {
		tasks, err := RedisClient.GetUserTasks(user.ID)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, tasks...)
	}

	if err := PostgresClient.SyncTasks(allTasks); err != nil {
		return 0, err
	}

	return len(allTasks), nil
}

func (s *SyncService) syncCounters() error {
	maxUserID, err := PostgresClient.GetMaxUserID()
	if err != nil {
		return err
	}

	maxGroupID, err := PostgresClient.GetMaxGroupID()
	if err != nil {
		return err
	}

	maxTaskID, err := PostgresClient.GetMaxTaskID()
	if err != nil {
		return err
	}

	currentUserID, _ := RedisClient.GetNextUserID()
	if maxUserID >= currentUserID {
		for i := currentUserID; i <= maxUserID; i++ {
			RedisClient.GetNextUserID()
		}
	}

	currentGroupID, _ := RedisClient.GetNextGroupID()
	if maxGroupID >= currentGroupID {
		for i := currentGroupID; i <= maxGroupID; i++ {
			RedisClient.GetNextGroupID()
		}
	}

	currentTaskID, _ := RedisClient.GetNextTaskID()
	if maxTaskID >= currentTaskID {
		for i := currentTaskID; i <= maxTaskID; i++ {
			RedisClient.GetNextTaskID()
		}
	}

	return nil
}

func (s *SyncService) ForceSyncNow() error {
	log.Println("🔄 Force sync requested...")
	return s.performSync()
}

func (s *SyncService) SyncFromPostgresToRedis() error {
	log.Println("🔄 Starting sync from PostgreSQL to Redis...")
	startTime := time.Now()

	users, err := PostgresClient.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users from PostgreSQL: %v", err)
	}

	for _, user := range users {
		if err := RedisClient.SaveUser(user); err != nil {
			log.Printf("⚠️ Failed to save user %d to Redis: %v", user.ID, err)
		}
	}

	groups, err := PostgresClient.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups from PostgreSQL: %v", err)
	}

	for _, group := range groups {
		if err := RedisClient.SaveGroup(group); err != nil {
			log.Printf("⚠️ Failed to save group %d to Redis: %v", group.ID, err)
		}
	}

	for _, user := range users {
		tasks, err := PostgresClient.GetUserTasks(user.ID)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			if err := RedisClient.SaveTask(task); err != nil {
				log.Printf("⚠️ Failed to save task %d to Redis: %v", task.ID, err)
			}
		}
	}

	duration := time.Since(startTime)
	log.Printf("✅ Reverse sync completed in %v - Users: %d, Groups: %d",
		duration, len(users), len(groups))

	return nil
}

func (s *SyncService) IsHealthy() bool {
	lastSync, err := RedisClient.GetLastSyncTime()
	if err != nil {
		return false
	}

	return time.Since(lastSync) < (2 * s.syncInterval)
}

func (s *SyncService) GetSyncStatus() map[string]interface{} {
	status := make(map[string]interface{})
	status["running"] = s.running
	status["interval"] = s.syncInterval.String()

	lastSync, err := RedisClient.GetLastSyncTime()
	if err == nil {
		status["last_sync"] = lastSync
		status["time_since_last_sync"] = time.Since(lastSync).String()
	}

	status["healthy"] = s.IsHealthy()

	dirtyTypes, _ := RedisClient.GetDirtyTypes()
	status["dirty_types"] = dirtyTypes
	status["pending_changes"] = len(dirtyTypes) > 0

	return status
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (s *SyncService) EmergencyBackup() error {
	log.Println("🆘 Performing emergency backup...")

	if err := s.ForceSyncNow(); err != nil {
		return err
	}

	return PostgresClient.BackupData()
}

func (s *SyncService) RestoreFromPostgreSQL() error {
	log.Println("🔧 Restoring data from PostgreSQL...")
	return s.SyncFromPostgresToRedis()
}
