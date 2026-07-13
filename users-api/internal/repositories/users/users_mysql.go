package users

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
	// AutoMigrate corre el AutoMigrate de GORM al conectar. En compose va en
	// false: ahí el schema lo maneja el one-shot de golang-migrate (migrations/).
	AutoMigrate bool
}

type MySQL struct {
	db *gorm.DB
}

var (
	// ErrUserNotFound representa que el usuario no existe en el repositorio principal.
	// Se usa para mapear a HTTP 404 en controllers mediante `errors.Is`.
	ErrUserNotFound = errors.New("user not found")
)

var (
	migrate = []interface{}{
		usersDAO.User{},
	}
)

func NewMySQL(config MySQLConfig) MySQL {
	// Build DSN (Data Source Name)
	// timeout/readTimeout/writeTimeout: que el driver corte solo si MySQL no responde (R1)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=5s&writeTimeout=5s",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	// Open connection to MySQL using GORM
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to MySQL: %s", err.Error())
	}

	// Pool de conexiones + ping fail-fast (DB3)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get underlying sql.DB: %s", err.Error())
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("mysql unreachable: %v", err)
	}

	// Automigrate structs to Gorm (gateado por AUTO_MIGRATE)
	if config.AutoMigrate {
		for _, target := range migrate {
			if err := db.AutoMigrate(target); err != nil {
				log.Fatalf("error automigrating structs: %s", err.Error())
			}
		}
	}

	return MySQL{
		db: db,
	}
}

// Ping verifica la conectividad con MySQL (lo usa el /readyz, O3).
func (repository MySQL) Ping(ctx context.Context) error {
	sqlDB, err := repository.db.DB()
	if err != nil {
		return fmt.Errorf("error getting underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

func (repository MySQL) GetAll(ctx context.Context, limit, offset int) ([]usersDAO.User, error) {
	var usersList []usersDAO.User
	// ORDER BY estable (RV8): LIMIT/OFFSET sin orden definido puede repetir o
	// saltear filas entre páginas.
	if err := repository.db.WithContext(ctx).Order("id ASC").Limit(limit).Offset(offset).Find(&usersList).Error; err != nil {
		return nil, fmt.Errorf("error fetching all users: %w", err)
	}
	return usersList, nil
}

func (repository MySQL) CountAll(ctx context.Context) (int64, error) {
	var total int64
	if err := repository.db.WithContext(ctx).Model(&usersDAO.User{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("error counting users: %w", err)
	}
	return total, nil
}

func (repository MySQL) GetByID(ctx context.Context, id int64) (usersDAO.User, error) {
	var user usersDAO.User
	if err := repository.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, ErrUserNotFound
		}
		return user, fmt.Errorf("error fetching user by id: %w", err)
	}
	return user, nil
}

func (repository MySQL) GetByUsername(ctx context.Context, username string) (usersDAO.User, error) {
	var user usersDAO.User
	if err := repository.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, ErrUserNotFound
		}
		return user, fmt.Errorf("error fetching user by username: %w", err)
	}
	return user, nil
}

func (repository MySQL) Create(ctx context.Context, user usersDAO.User) (int64, error) {
	if err := repository.db.WithContext(ctx).Create(&user).Error; err != nil {
		return 0, fmt.Errorf("error creating user: %w", err)
	}
	return user.ID, nil
}

func (repository MySQL) Update(ctx context.Context, user usersDAO.User) error {
	if err := repository.db.WithContext(ctx).Save(&user).Error; err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}
	return nil
}

func (repository MySQL) Delete(ctx context.Context, id int64) error {
	if err := repository.db.WithContext(ctx).Delete(&usersDAO.User{}, id).Error; err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}
	return nil
}
