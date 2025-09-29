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
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	if pgPassword == "" {
		pgPassword = "EKQH9jQX7gAfV7pLwVmsbLbF3XfY6n4S"
	}

	dsn := fmt.Sprintf("host=localhost user=airflow password=%s dbname=airflow port=5433 sslmode=disable TimeZone=Asia/Tehran", pgPassword)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.Task{}, &models.UserGroup{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	PostgresClient = &PostgresManager{db: db}
	fmt.Println("✅ PostgreSQL connected and migrated successfully")
	return nil
}

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
	p.db.Where("user_id = ?", userID).Delete(&models.Task{})
	p.db.Where("user_id = ?", userID).Delete(&models.UserGroup{})
	return p.db.Delete(&models.User{}, userID).Error
}

func (p *PostgresManager) GetMaxUserID() (int, error) {
	var maxID int
	err := p.db.Model(&models.User{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

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
	p.db.Where("group_id = ?", groupID).Delete(&models.Task{})
	p.db.Where("group_id = ?", groupID).Delete(&models.UserGroup{})
	return p.db.Delete(&models.Group{}, groupID).Error
}

func (p *PostgresManager) GetMaxGroupID() (int, error) {
	var maxID int
	err := p.db.Model(&models.Group{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

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

func (p *PostgresManager) SyncUsers(users []*models.User) error {
	tx := p.db.Begin()

	for _, user := range users {
		var existingUser models.User

		errByID := tx.Where("id = ?", user.ID).First(&existingUser).Error

		if errByID == gorm.ErrRecordNotFound {
			var userByEmail models.User
			errByEmail := tx.Where("email = ?", user.Email).First(&userByEmail).Error

			if errByEmail == gorm.ErrRecordNotFound {
				if createErr := tx.Create(user).Error; createErr != nil {
					tx.Rollback()
					return createErr
				}
			} else if errByEmail != nil {
				tx.Rollback()
				return errByEmail
			} else {
				if delErr := tx.Delete(&userByEmail).Error; delErr != nil {
					tx.Rollback()
					return delErr
				}
				tx.Where("user_id = ?", userByEmail.ID).Delete(&models.UserGroup{})

				if createErr := tx.Create(user).Error; createErr != nil {
					tx.Rollback()
					return createErr
				}
			}
		} else if errByID != nil {
			tx.Rollback()
			return errByID
		} else {
			existingUser.FullName = user.FullName
			existingUser.Role = user.Role
			existingUser.GroupIDs = user.GroupIDs
			existingUser.Number = user.Number
			existingUser.Email = user.Email
			existingUser.Password = user.Password
			existingUser.WorkTimes = user.WorkTimes
			existingUser.UpdatedAt = user.UpdatedAt

			if saveErr := tx.Save(&existingUser).Error; saveErr != nil {
				tx.Rollback()
				return saveErr
			}
		}

		if delErr := tx.Where("user_id = ?", user.ID).Delete(&models.UserGroup{}).Error; delErr != nil {
			tx.Rollback()
			return delErr
		}

		for _, groupID := range user.GroupIDs {
			userGroup := &models.UserGroup{
				UserID:  user.ID,
				GroupID: groupID,
			}
			tx.Create(userGroup)
		}
	}

	return tx.Commit().Error
}

func (p *PostgresManager) SyncGroups(groups []*models.Group) error {
	tx := p.db.Begin()

	for _, group := range groups {
		var existingGroup models.Group

		errByID := tx.Where("id = ?", group.ID).First(&existingGroup).Error

		if errByID == gorm.ErrRecordNotFound {
			var groupByName models.Group
			errByName := tx.Where("name = ?", group.Name).First(&groupByName).Error

			if errByName == gorm.ErrRecordNotFound {
				if createErr := tx.Create(group).Error; createErr != nil {
					tx.Rollback()
					return createErr
				}
			} else if errByName != nil {
				tx.Rollback()
				return errByName
			} else {
				if delErr := tx.Delete(&groupByName).Error; delErr != nil {
					tx.Rollback()
					return delErr
				}
				tx.Where("group_id = ?", groupByName.ID).Delete(&models.UserGroup{})
				tx.Where("group_id = ?", groupByName.ID).Delete(&models.Task{})

				if createErr := tx.Create(group).Error; createErr != nil {
					tx.Rollback()
					return createErr
				}
			}
		} else if errByID != nil {
			tx.Rollback()
			return errByID
		} else {
			existingGroup.Name = group.Name
			existingGroup.AdminID = group.AdminID
			existingGroup.UpdatedAt = group.UpdatedAt

			if saveErr := tx.Save(&existingGroup).Error; saveErr != nil {
				tx.Rollback()
				return saveErr
			}
		}
	}

	return tx.Commit().Error
}

func (p *PostgresManager) SyncTasks(tasks []*models.Task) error {
	tx := p.db.Begin()

	for _, task := range tasks {
		if saveErr := tx.Save(task).Error; saveErr != nil {
			tx.Rollback()
			return saveErr
		}
	}

	return tx.Commit().Error
}

func (p *PostgresManager) CleanupDeletedData() error {
	return nil
}

func (p *PostgresManager) Ping() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (p *PostgresManager) BackupData() error {
	return nil
}

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
