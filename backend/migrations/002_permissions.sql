CREATE TABLE IF NOT EXISTS permissions (
  code VARCHAR(80) NOT NULL,
  name VARCHAR(128) NOT NULL,
  category VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (code),
  KEY idx_permissions_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS role_permissions (
  role ENUM('admin', 'user') NOT NULL,
  permission_code VARCHAR(80) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (role, permission_code),
  KEY idx_role_permissions_code (permission_code),
  CONSTRAINT fk_role_permissions_code FOREIGN KEY (permission_code) REFERENCES permissions(code)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO permissions (code, name, category, description)
VALUES
  ('dashboard:view', 'Dashboard view', 'dashboard', 'View dashboard overview'),
  ('tools:view', 'Tools view', 'tools', 'View tool center'),
  ('tools:manage', 'Tools manage', 'tools', 'Manage tool modules'),
  ('tasks:create', 'Task create', 'tasks', 'Create backend tasks'),
  ('shops:view', 'Shops view', 'shops', 'View visible shops'),
  ('shops:create', 'Shop create', 'shops', 'Create shops'),
  ('shops:update', 'Shop update', 'shops', 'Update shops'),
  ('shops:delete', 'Shop close', 'shops', 'Close shops'),
  ('users:view', 'Users view', 'users', 'View user list'),
  ('users:create', 'User create', 'users', 'Create users'),
  ('users:update', 'User update', 'users', 'Update users'),
  ('users:disable', 'User disable', 'users', 'Disable users'),
  ('users:assign_shops', 'User shop assignment', 'users', 'Assign shops to users'),
  ('settings:view', 'Settings view', 'settings', 'View system settings'),
  ('settings:update', 'Settings update', 'settings', 'Update system settings'),
  ('permissions:view', 'Permissions view', 'permissions', 'View permission matrix'),
  ('permissions:manage', 'Permissions manage', 'permissions', 'Manage role permissions')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  category = VALUES(category),
  description = VALUES(description);

INSERT IGNORE INTO role_permissions (role, permission_code)
SELECT 'admin', code FROM permissions;

INSERT IGNORE INTO role_permissions (role, permission_code)
VALUES
  ('user', 'dashboard:view'),
  ('user', 'tools:view'),
  ('user', 'shops:view'),
  ('user', 'settings:view');
