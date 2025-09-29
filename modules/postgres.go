package modules

import (
	"fmt"
	"os"
	"task-manager/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresManager struct {
	db *gorm.DB
}

var PostgresClient *PostgresManager

func InitPostgres() error {
	// Read PostgreSQL password from environment or use the one from your .env
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	if pgPassword == "" {
		pgPassword = "EKQH9jQX7gAfV7pLwVmsbLbF3XfY6n4S" // Your actual password from .env
	}

	dsn := fmt.Sprintf("host=localhost user=airflow password=%s dbname=airflow port=5433 sslmode=disable TimeZone=Asia/Tehran", pgPassword)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Auto migrate schemas
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.Task{}, &models.UserGroup{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	PostgresClient = &PostgresManager{db: db}
	fmt.Println("✅ PostgreSQL connected and migrated successfully")
	return nil
}

// User operations
func (p *PostgresManager) SaveUser(user *models.User) error {
	return p.db.Save(user).Error
}

func (p *PostgresManager) GetUser(userID int) (*models.User, error) {
	var user models.User
	err := p.db.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *PostgresManager) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := p.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *PostgresManager) GetAllUsers() ([]*models.User, error) {
	var users []*models.User
	err := p.db.Find(&users).Error
	return users, err
}

func (p *PostgresManager) DeleteUser(userID int) error {
	// Delete related tasks first
	p.db.Where("user_id = ?", userID).Delete(&models.Task{})

	// Delete user-group relationships
	p.db.Where("user_id = ?", userID).Delete(&models.UserGroup{})

	// Delete user
	return p.db.Delete(&models.User{}, userID).Error
}

func (p *PostgresManager) GetMaxUserID() (int, error) {
	var maxID int
	err := p.db.Model(&models.User{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

// Group operations
func (p *PostgresManager) SaveGroup(group *models.Group) error {
	return p.db.Save(group).Error
}

func (p *PostgresManager) GetGroup(groupID int) (*models.Group, error) {
	var group models.Group
	err := p.db.First(&group, groupID).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (p *PostgresManager) GetAllGroups() ([]*models.Group, error) {
	var groups []*models.Group
	err := p.db.Find(&groups).Error
	return groups, err
}

func (p *PostgresManager) GetUserGroups(userID int) ([]*models.Group, error) {
	var groups []*models.Group
	err := p.db.Table("groups").
		Joins("JOIN user_groups ON groups.id = user_groups.group_id").
		Where("user_groups.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
}

func (p *PostgresManager) GetGroupUsers(groupID int) ([]*models.User, error) {
	var users []*models.User
	err := p.db.Table("users").
		Joins("JOIN user_groups ON users.id = user_groups.user_id").
		Where("user_groups.group_id = ?", groupID).
		Find(&users).Error
	return users, err
}

func (p *PostgresManager) DeleteGroup(groupID int) error {
	// Delete related tasks first
	p.db.Where("group_id = ?", groupID).Delete(&models.Task{})

	// Delete user-group relationships
	p.db.Where("group_id = ?", groupID).Delete(&models.UserGroup{})

	// Delete group
	return p.db.Delete(&models.Group{}, groupID).Error
}

func (p *PostgresManager) GetMaxGroupID() (int, error) {
	var maxID int
	err := p.db.Model(&models.Group{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

// Task operations
func (p *PostgresManager) SaveTask(task *models.Task) error {
	return p.db.Save(task).Error
}

func (p *PostgresManager) GetTask(taskID int) (*models.Task, error) {
	var task models.Task
	err := p.db.First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (p *PostgresManager) GetUserTasks(userID int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := p.db.Where("user_id = ?", userID).Find(&tasks).Error
	return tasks, err
}

func (p *PostgresManager) GetGroupTasks(groupID int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := p.db.Where("group_id = ?", groupID).Find(&tasks).Error
	return tasks, err
}

func (p *PostgresManager) DeleteTask(taskID int) error {
	return p.db.Delete(&models.Task{}, taskID).Error
}

func (p *PostgresManager) GetMaxTaskID() (int, error) {
	var maxID int
	err := p.db.Model(&models.Task{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

// UserGroup relationship operations
func (p *PostgresManager) AddUserToGroup(userID, groupID int) error {
	userGroup := &models.UserGroup{
		UserID:  userID,
		GroupID: groupID,
	}
	return p.db.Create(userGroup).Error
}

func (p *PostgresManager) RemoveUserFromGroup(userID, groupID int) error {
	return p.db.Where("user_id = ? AND group_id = ?", userID, groupID).Delete(&models.UserGroup{}).Error
}

// Sync operations - sync from Redis to PostgreSQL
func (p *PostgresManager) SyncUsers(users []*models.User) error {
	tx := p.db.Begin()

	for _, user := range users {
		// Use ON CONFLICT for PostgreSQL to handle duplicates
		var existingUser models.User
		err := tx.Where("id = ?", user.ID).First(&existingUser).Error

		if err == gorm.ErrRecordNotFound {
			// User doesn't exist, create new one
			if err := tx.Create(user).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if err != nil {
			// Other database error
			tx.Rollback()
			return err
		} else {
			// User exists, update it
			if err := tx.Model(&existingUser).Updates(user).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		// Delete existing user-group relationships for this user
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserGroup{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Create new user-group relationships
		for _, groupID := range user.GroupIDs {
			userGroup := &models.UserGroup{
				UserID:  user.ID,
				GroupID: groupID,
			}
			if err := tx.Create(userGroup).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func (p *PostgresManager) SyncGroups(groups []*models.Group) error {
	tx := p.db.Begin()

	for _, group := range groups {
		var existingGroup models.Group
		err := tx.Where("id = ?", group.ID).First(&existingGroup).Error

		if err == gorm.ErrRecordNotFound {
			// Group doesn't exist, create new one
			if err := tx.Create(group).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if err != nil {
			// Other database error
			tx.Rollback()
			return err
		} else {
			// Group exists, update it
			if err := tx.Model(&existingGroup).Updates(group).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func (p *PostgresManager) SyncTasks(tasks []*models.Task) error {
	tx := p.db.Begin()

	for _, task := range tasks {
		if err := tx.Save(task).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// Cleanup operations
func (p *PostgresManager) CleanupDeletedData() error {
	// This method can be used to clean up soft-deleted records
	// For now, we're using hard deletes, so this is placeholder
	return nil
}

// Health check
func (p *PostgresManager) Ping() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Backup operations
func (p *PostgresManager) BackupData() error {
	// This is a placeholder for backup operations
	// In production, you might want to implement actual backup logic
	return nil
}

// Statistics
func (p *PostgresManager) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var userCount, groupCount, taskCount int64

	p.db.Model(&models.User{}).Count(&userCount)
	p.db.Model(&models.Group{}).Count(&groupCount)
	p.db.Model(&models.Task{}).Count(&taskCount)

	stats["users"] = userCount
	stats["groups"] = groupCount
	stats["tasks"] = taskCount

	return stats, nil
}
