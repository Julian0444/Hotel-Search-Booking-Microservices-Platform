package users

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	usersDAO "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/dao/users"
	usersDomain "github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform/users-api/internal/domain/users"

	gosqlmysql "github.com/go-sql-driver/mysql"
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

// mysqlDuplicateEntry es el número de error de MySQL para violaciones de
// índice único (C6): la detección del username duplicado es tipada
// (errors.As), no por substring del mensaje del driver.
const mysqlDuplicateEntry = 1062

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

// Close cierra el pool de conexiones a MySQL (graceful shutdown, C12).
func (repository MySQL) Close() error {
	sqlDB, err := repository.db.DB()
	if err != nil {
		return fmt.Errorf("error getting underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
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
			return user, usersDomain.ErrUserNotFound
		}
		return user, fmt.Errorf("error fetching user by id: %w", err)
	}
	return user, nil
}

func (repository MySQL) GetByUsername(ctx context.Context, username string) (usersDAO.User, error) {
	var user usersDAO.User
	if err := repository.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, usersDomain.ErrUserNotFound
		}
		return user, fmt.Errorf("error fetching user by username: %w", err)
	}
	return user, nil
}

func (repository MySQL) Create(ctx context.Context, user usersDAO.User) (int64, error) {
	if err := repository.db.WithContext(ctx).Create(&user).Error; err != nil {
		// Duplicado del índice único de username -> sentinel tipado (C6)
		var mysqlErr *gosqlmysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry {
			return 0, fmt.Errorf("username %q: %w", user.Username, usersDomain.ErrUsernameTaken)
		}
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

// Delete recibe el DAO completo (no solo el id): las cachés necesitan el
// username para invalidar su key user:username:* sin depender de un lookup
// previo (RV28). Acá solo se usa el ID.
func (repository MySQL) Delete(ctx context.Context, user usersDAO.User) error {
	result := repository.db.WithContext(ctx).Delete(&usersDAO.User{}, user.ID)
	if result.Error != nil {
		return fmt.Errorf("error deleting user: %w", result.Error)
	}
	// C9: borrar un id inexistente es 404, consistente con GetByID.
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %d: %w", user.ID, usersDomain.ErrUserNotFound)
	}
	return nil
}
