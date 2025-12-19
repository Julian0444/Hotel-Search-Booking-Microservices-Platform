package config

import "time"

const (
	MySQLHost     = "mysql"
	MySQLPort     = "3306"
	MySQLDatabase = "users-api"
	MySQLUsername = "root"
	MySQLPassword = "root"

	CacheDuration = 30 * time.Second

	MemcachedHost = "memcached"
	MemcachedPort = "11211"

	JWTKey      = "ThisIsAnExampleJWTKey!"
	JWTDuration = 24 * time.Hour

	// BcryptCost controla el costo de hashing de contraseñas.
	// 10 es un default razonable para desarrollo.
	BcryptCost = 10

	// Port es el puerto HTTP donde expone el server.
	Port = "8082"
)
