CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  password_hash CHAR(64) NOT NULL,
  display_name VARCHAR(128) NOT NULL,
  role ENUM('admin', 'user') NOT NULL DEFAULT 'user',
  status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username),
  KEY idx_users_role (role),
  KEY idx_users_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS shops (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  shop_name VARCHAR(128) NOT NULL,
  platform VARCHAR(32) NOT NULL DEFAULT 'temu',
  external_code VARCHAR(128) NULL,
  status ENUM('active', 'paused', 'closed') NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_shops_platform_external_code (platform, external_code),
  KEY idx_shops_status (status),
  CONSTRAINT fk_shops_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_shops (
  user_id BIGINT UNSIGNED NOT NULL,
  shop_id BIGINT UNSIGNED NOT NULL,
  shop_role ENUM('owner', 'operator', 'viewer') NOT NULL DEFAULT 'operator',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, shop_id),
  KEY idx_user_shops_shop_id (shop_id),
  CONSTRAINT fk_user_shops_user_id FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_user_shops_shop_id FOREIGN KEY (shop_id) REFERENCES shops(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO users (id, username, password_hash, display_name, role, status)
VALUES
  (1, 'admin', 'bf6b5bdb74c79ece9fc0ad0ac9fb0359f9555d4f35a83b2e6ec69ae99e09603d', 'System Admin', 'admin', 'active'),
  (2, 'operator_a', 'bd92c914a9c5a60eb53746d39c1ae7cc576955e45cfbc5281685f1152b0b01b4', 'Operator A', 'user', 'active')
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  display_name = VALUES(display_name),
  role = VALUES(role),
  status = VALUES(status);

INSERT INTO shops (id, shop_name, platform, external_code, status, created_by)
VALUES
  (1, 'Temu Main Shop', 'temu', 'temu-main', 'active', 1),
  (2, 'Temu Test Shop', 'temu', 'temu-test', 'paused', 1)
ON DUPLICATE KEY UPDATE
  shop_name = VALUES(shop_name),
  status = VALUES(status);

INSERT IGNORE INTO user_shops (user_id, shop_id, shop_role)
VALUES
  (1, 1, 'owner'),
  (1, 2, 'owner'),
  (2, 1, 'operator');
