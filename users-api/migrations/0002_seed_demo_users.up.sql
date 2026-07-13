-- 0002: cliente demo para el quickstart (P7).
-- El ADMIN NO se siembra acá a propósito: la única fuente del primer admin es el
-- seed env-driven de users-api al arranque (ADMIN_USERNAME/ADMIN_PASSWORD, plan 01).
-- Hash bcrypt $2a$ generado con golang.org/x/crypto/bcrypt (la lib de la app);
-- htpasswd genera $2y$ y la lib lo rechaza. Password: DemoCliente123
INSERT INTO `users` (`username`, `password`, `tipo`)
VALUES ('demo', '$2a$10$lPwUI6m7avM9T.qYWNp9PelLaGO.k1n32LQXb4jlf7AnZPUOzgyHi', 'cliente')
ON DUPLICATE KEY UPDATE `username` = `username`;
