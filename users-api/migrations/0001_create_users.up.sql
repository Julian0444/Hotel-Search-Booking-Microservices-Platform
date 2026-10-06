-- 0001: tabla users. Copia EXACTA del schema que venía creando GORM AutoMigrate
-- (verificado con SHOW CREATE TABLE contra el contenedor mysql:8), para que
-- volúmenes existentes y volúmenes nuevos queden idénticos.
-- IF NOT EXISTS: en un volumen pre-migraciones la tabla ya existe; golang-migrate
-- registra la versión igual y el estado converge sin dirty flag.
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(100) NOT NULL,
  `password` varchar(255) NOT NULL,
  `tipo` enum('cliente','administrador') DEFAULT 'cliente',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
