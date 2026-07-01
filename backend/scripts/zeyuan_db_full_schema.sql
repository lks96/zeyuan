-- 当前完整建库脚本
-- 数据库：zeyuan_db
-- 用法示例：
--   mysql -h <host> -P <port> -u <user> -p < backend/scripts/init_zeyuan_db.sql
--
-- 说明：
-- 1. 新生产库或全新环境优先执行本脚本，建表语句会直接创建当前最终结构。
-- 2. migrations 目录仍然保留历史增量脚本，给已有数据库升级使用。
-- 3. 每次新增迁移后，也需要同步维护本脚本。
-- 4. 脚本会写入 schema_migrations 记录，后续仍可继续使用 Go 迁移器执行新增迁移。
-- 5. 初始店铺只保留当前店铺：Kunsong Grocery / 634418227150594。

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 生产执行前必须修改这里的初始管理员密码。
-- 后端密码规则为 SHA256(username + ':' + password)，脚本会自动生成 password_hash。
SET @admin_username = 'admin';
SET @admin_initial_password = 'CHANGE_ME_BEFORE_RUN';
SET @admin_display_name = '系统管理员';

CREATE DATABASE IF NOT EXISTS `zeyuan_db`
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE `zeyuan_db`;

DROP PROCEDURE IF EXISTS ensure_admin_password_changed;
DELIMITER //
CREATE PROCEDURE ensure_admin_password_changed()
BEGIN
  IF @admin_initial_password = 'CHANGE_ME_BEFORE_RUN' OR CHAR_LENGTH(@admin_initial_password) < 12 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '请先修改 init_zeyuan_db.sql 中的 @admin_initial_password，且长度至少 12 位。';
  END IF;
END//
DELIMITER ;
CALL ensure_admin_password_changed();
DROP PROCEDURE ensure_admin_password_changed;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(32) NOT NULL COMMENT '迁移版本号',
  name VARCHAR(255) NOT NULL COMMENT '迁移文件名',
  checksum CHAR(64) NOT NULL COMMENT '迁移文件校验值',
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '迁移应用时间',
  PRIMARY KEY (version),
  UNIQUE KEY uk_schema_migrations_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据库迁移记录表';

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  username VARCHAR(64) NOT NULL COMMENT '登录账号',
  password_hash CHAR(64) NOT NULL COMMENT '密码哈希',
  display_name VARCHAR(128) NOT NULL COMMENT '显示名称',
  role ENUM('admin', 'user') NOT NULL DEFAULT 'user' COMMENT '系统角色',
  status ENUM('active', 'disabled') NOT NULL DEFAULT 'active' COMMENT '用户状态',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username),
  KEY idx_users_role (role),
  KEY idx_users_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统用户表';

CREATE TABLE IF NOT EXISTS shops (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '店铺ID',
  shop_name VARCHAR(128) NOT NULL COMMENT '店铺名称',
  platform VARCHAR(32) NOT NULL DEFAULT 'temu' COMMENT '所属平台',
  external_code VARCHAR(128) NULL COMMENT '平台店铺编号',
  eu_representative VARCHAR(255) NOT NULL DEFAULT '' COMMENT '欧代信息',
  status ENUM('active', 'paused', 'closed') NOT NULL DEFAULT 'active' COMMENT '店铺状态',
  created_by BIGINT UNSIGNED NULL COMMENT '创建人用户ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_shops_platform_external_code (platform, external_code),
  KEY idx_shops_status (status),
  CONSTRAINT fk_shops_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='店铺信息表';

CREATE TABLE IF NOT EXISTS user_shops (
  user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  shop_id BIGINT UNSIGNED NOT NULL COMMENT '店铺ID',
  shop_role ENUM('owner', 'operator', 'viewer') NOT NULL DEFAULT 'operator' COMMENT '用户在店铺中的角色',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '绑定时间',
  PRIMARY KEY (user_id, shop_id),
  KEY idx_user_shops_shop_id (shop_id),
  CONSTRAINT fk_user_shops_user_id FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_user_shops_shop_id FOREIGN KEY (shop_id) REFERENCES shops(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户店铺绑定表';

CREATE TABLE IF NOT EXISTS permissions (
  code VARCHAR(80) NOT NULL COMMENT '权限编码',
  name VARCHAR(128) NOT NULL COMMENT '权限名称',
  category VARCHAR(64) NOT NULL COMMENT '权限分类',
  description VARCHAR(255) NOT NULL DEFAULT '' COMMENT '权限说明',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (code),
  KEY idx_permissions_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限定义表';

CREATE TABLE IF NOT EXISTS role_permissions (
  role ENUM('admin', 'user') NOT NULL COMMENT '系统角色',
  permission_code VARCHAR(80) NOT NULL COMMENT '权限编码',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授权时间',
  PRIMARY KEY (role, permission_code),
  KEY idx_role_permissions_code (permission_code),
  CONSTRAINT fk_role_permissions_code FOREIGN KEY (permission_code) REFERENCES permissions(code)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色权限绑定表';

CREATE TABLE IF NOT EXISTS tool_modules (
  id VARCHAR(64) NOT NULL COMMENT '工具模块ID',
  name VARCHAR(128) NOT NULL COMMENT '工具名称',
  description VARCHAR(255) NOT NULL DEFAULT '' COMMENT '工具说明',
  status ENUM('planning', 'active', 'paused') NOT NULL DEFAULT 'planning' COMMENT '工具状态',
  sort_order INT NOT NULL DEFAULT 100 COMMENT '排序值',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  KEY idx_tool_modules_status (status),
  KEY idx_tool_modules_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工具中心模块表';

CREATE TABLE IF NOT EXISTS tool_packages (
  id VARCHAR(64) NOT NULL COMMENT '工具包ID',
  version VARCHAR(32) NOT NULL DEFAULT '1.0.0' COMMENT '工具包版本',
  name VARCHAR(128) NOT NULL COMMENT '工具名称',
  description VARCHAR(255) NOT NULL DEFAULT '' COMMENT '工具说明',
  category VARCHAR(64) NOT NULL DEFAULT '店铺运营工具' COMMENT '工具分类',
  icon VARCHAR(64) NOT NULL DEFAULT 'blocks' COMMENT '工具图标',
  status ENUM('planning', 'active', 'paused') NOT NULL DEFAULT 'planning' COMMENT '工具状态',
  package_type ENUM('builtin', 'installed') NOT NULL DEFAULT 'builtin' COMMENT '工具包类型',
  entry_type ENUM('native', 'iframe') NOT NULL DEFAULT 'native' COMMENT '前端入口类型',
  entry_path VARCHAR(500) NOT NULL DEFAULT '' COMMENT '前端入口路径',
  panel_key VARCHAR(64) NOT NULL DEFAULT '' COMMENT '内置面板标识',
  removable TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否允许卸载',
  recommended TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否推荐',
  sort_order INT NOT NULL DEFAULT 100 COMMENT '排序值',
  permissions_json TEXT NOT NULL COMMENT '工具权限声明JSON',
  manifest_json LONGTEXT NOT NULL COMMENT '工具包清单JSON',
  installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '安装时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  KEY idx_tool_packages_status (status),
  KEY idx_tool_packages_category (category),
  KEY idx_tool_packages_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工具包安装表';

CREATE TABLE IF NOT EXISTS system_settings (
  setting_key VARCHAR(80) NOT NULL COMMENT '设置键',
  setting_value VARCHAR(500) NOT NULL DEFAULT '' COMMENT '设置值',
  description VARCHAR(255) NOT NULL DEFAULT '' COMMENT '设置说明',
  updated_by BIGINT UNSIGNED NULL COMMENT '更新人用户ID',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (setting_key),
  CONSTRAINT fk_system_settings_updated_by FOREIGN KEY (updated_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统设置表';

CREATE TABLE IF NOT EXISTS delivery_extract_batches (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '提取批次ID',
  source_file VARCHAR(255) NOT NULL DEFAULT '' COMMENT '来源文件或来源名称',
  batch_date VARCHAR(16) NOT NULL DEFAULT '' COMMENT '批次日期',
  source_total INT NOT NULL DEFAULT 0 COMMENT '源记录数量',
  extracted_total INT NOT NULL DEFAULT 0 COMMENT '提取结果数量',
  target_json LONGTEXT NOT NULL COMMENT '提取后的目标JSON',
  created_by BIGINT UNSIGNED NULL COMMENT '创建人用户ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (id),
  KEY idx_delivery_extract_batches_created_at (created_at),
  KEY idx_delivery_extract_batches_batch_date (batch_date),
  CONSTRAINT fk_delivery_extract_batches_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发货JSON提取批次表';

CREATE TABLE IF NOT EXISTS delivery_extract_rows (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '提取明细ID',
  batch_id BIGINT UNSIGNED NOT NULL COMMENT '提取批次ID',
  supplier_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '供应商或店铺编号',
  product_name TEXT NOT NULL COMMENT '商品名称',
  product_skc_picture VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '产品图片地址',
  delivery_order_sn VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发货单号',
  express_batch_sn VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发货批次号',
  expect_pick_up_goods_time BIGINT NOT NULL DEFAULT 0 COMMENT '上门取货时间戳',
  skc VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'SKC码',
  skc_num INT NOT NULL DEFAULT 0 COMMENT 'SKC数量',
  sku VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'SKU码',
  sku_num INT NOT NULL DEFAULT 0 COMMENT 'SKU数量',
  receiver_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '收货仓库',
  row_json LONGTEXT NOT NULL COMMENT '原始明细JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (id),
  KEY idx_delivery_extract_rows_batch_id (batch_id),
  KEY idx_delivery_extract_rows_supplier_id (supplier_id),
  KEY idx_delivery_extract_rows_delivery_order_sn (delivery_order_sn),
  KEY idx_delivery_extract_rows_order_skc (delivery_order_sn, skc),
  KEY idx_delivery_extract_rows_express_batch_sn (express_batch_sn),
  KEY idx_delivery_extract_rows_expect_pick_up_goods_time (expect_pick_up_goods_time),
  KEY idx_delivery_extract_rows_skc (skc),
  CONSTRAINT fk_delivery_extract_rows_batch_id FOREIGN KEY (batch_id) REFERENCES delivery_extract_batches(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='发货JSON提取明细表';

CREATE TABLE IF NOT EXISTS product_collection_products (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '商品记录ID',
  product_skc_id VARCHAR(64) NOT NULL COMMENT 'SKC码',
  product_sku_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'SKU码',
  main_image_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '商品主图地址',
  product_name VARCHAR(500) NOT NULL DEFAULT '' COMMENT '商品名称',
  number_of_pieces_new INT NOT NULL DEFAULT 0 COMMENT '产品根数',
  product_config VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '产品配置',
  supplier_price_cent INT NOT NULL DEFAULT 0 COMMENT '供货价格（分）',
  cost_price_cent INT NOT NULL DEFAULT 0 COMMENT '成本价格（分）',
  skc_top_status INT NOT NULL DEFAULT 0 COMMENT '商品状态',
  product_created_at DATETIME NULL COMMENT '商品创建时间',
  supplier_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '店铺编码',
  source_json LONGTEXT NOT NULL COMMENT '原始商品JSON',
  created_by BIGINT UNSIGNED NULL COMMENT '导入人用户ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '导入时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_product_collection_products_skc (product_skc_id),
  KEY idx_product_collection_products_sku (product_sku_id),
  KEY idx_product_collection_products_supplier_id (supplier_id),
  KEY idx_product_collection_products_status (skc_top_status),
  KEY idx_product_collection_products_created_at (product_created_at),
  CONSTRAINT fk_product_collection_products_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品采集表';

CREATE TABLE IF NOT EXISTS sales_overall_batches (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '销售总览快照ID',
  source_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '来源名称',
  supplier_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '店铺编码',
  supplier_name VARCHAR(128) NOT NULL DEFAULT '' COMMENT '店铺名称',
  source_total INT NOT NULL DEFAULT 0 COMMENT '源商品数量',
  imported_total INT NOT NULL DEFAULT 0 COMMENT '导入SKU明细数量',
  created_by BIGINT UNSIGNED NULL COMMENT '导入人用户ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '导入时间',
  PRIMARY KEY (id),
  KEY idx_sales_overall_batches_supplier_id (supplier_id),
  KEY idx_sales_overall_batches_created_at (created_at),
  CONSTRAINT fk_sales_overall_batches_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='销售总览快照批次表';

CREATE TABLE IF NOT EXISTS sales_overall_rows (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '销售总览明细ID',
  batch_id BIGINT UNSIGNED NOT NULL COMMENT '销售总览快照ID',
  supplier_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '店铺编码',
  supplier_name VARCHAR(128) NOT NULL DEFAULT '' COMMENT '店铺名称',
  product_skc_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'SKC码',
  product_sku_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'SKU码',
  product_name VARCHAR(500) NOT NULL DEFAULT '' COMMENT '商品名称',
  product_image VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '商品图片',
  category VARCHAR(128) NOT NULL DEFAULT '' COMMENT '商品类目',
  sku_class_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'SKU规格名',
  supplier_price_cent INT NOT NULL DEFAULT 0 COMMENT '申报价格（分）',
  cost_price_cent INT NOT NULL DEFAULT 0 COMMENT '成本价格（分）',
  price_review_status INT NOT NULL DEFAULT 0 COMMENT '价格审核状态',
  is_verify_price TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已校验价格',
  lack_quantity INT NOT NULL DEFAULT 0 COMMENT '缺货数量',
  in_cart_number_7d INT NOT NULL DEFAULT 0 COMMENT '近7日用户加购数量',
  in_cart_number_total INT NOT NULL DEFAULT 0 COMMENT '用户累计加购数量',
  subscribe_arrival_remind_count INT NOT NULL DEFAULT 0 COMMENT '订阅到货提醒数量',
  today_sale_volume INT NOT NULL DEFAULT 0 COMMENT '今日销量',
  last_seven_days_sale_volume INT NOT NULL DEFAULT 0 COMMENT '近7日销量',
  last_thirty_days_sale_volume INT NOT NULL DEFAULT 0 COMMENT '近30日销量',
  total_sale_volume INT NOT NULL DEFAULT 0 COMMENT '累计销量',
  warehouse_inventory_num INT NOT NULL DEFAULT 0 COMMENT '仓内可用库存',
  expected_occupied_inventory_num INT NOT NULL DEFAULT 0 COMMENT '仓内预占用库存',
  unavailable_warehouse_inventory_num INT NOT NULL DEFAULT 0 COMMENT '仓内暂不可用库存',
  wait_delivery_inventory_num INT NOT NULL DEFAULT 0 COMMENT '已发货库存',
  wait_receive_num INT NOT NULL DEFAULT 0 COMMENT '已创建备货单待发货库存',
  wait_approve_inventory_num INT NOT NULL DEFAULT 0 COMMENT '待审核备货库存',
  seller_warehouse_stock INT NOT NULL DEFAULT 0 COMMENT '卖家仓库存',
  advice_quantity INT NOT NULL DEFAULT 0 COMMENT '建议备货量',
  available_sale_days DECIMAL(10,2) NULL COMMENT '库存可售天数',
  warehouse_available_sale_days DECIMAL(10,2) NULL COMMENT '仓内库存可售天数',
  purchase_config VARCHAR(128) NOT NULL DEFAULT '' COMMENT '备货逻辑',
  target_produce_days DECIMAL(10,2) NULL COMMENT '建议目标库存天数',
  target_produce_num INT NULL COMMENT '建议目标库存',
  advice_produce_num INT NULL COMMENT '建议生产或采购量',
  show_stock_guide TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否展示备货引导',
  row_json LONGTEXT NOT NULL COMMENT '原始明细JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (id),
  KEY idx_sales_overall_rows_batch_id (batch_id),
  KEY idx_sales_overall_rows_supplier_id (supplier_id),
  KEY idx_sales_overall_rows_product_skc_id (product_skc_id),
  KEY idx_sales_overall_rows_product_sku_id (product_sku_id),
  KEY idx_sales_overall_rows_sales_volume (last_thirty_days_sale_volume, last_seven_days_sale_volume),
  CONSTRAINT fk_sales_overall_rows_batch_id FOREIGN KEY (batch_id) REFERENCES sales_overall_batches(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='销售总览快照明细表';

INSERT INTO users (id, username, password_hash, display_name, role, status)
VALUES
  (1, @admin_username, SHA2(CONCAT(@admin_username, ':', @admin_initial_password), 256), @admin_display_name, 'admin', 'active')
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  display_name = VALUES(display_name),
  role = VALUES(role),
  status = VALUES(status);

INSERT INTO shops (id, shop_name, platform, external_code, eu_representative, status, created_by)
VALUES
  (1, 'Kunsong Grocery', 'temu', '634418227150594', '', 'active', 1)
ON DUPLICATE KEY UPDATE
  shop_name = VALUES(shop_name),
  eu_representative = IF(VALUES(eu_representative) = '', eu_representative, VALUES(eu_representative)),
  status = VALUES(status),
  created_by = VALUES(created_by);

INSERT INTO user_shops (user_id, shop_id, shop_role)
VALUES
  (1, 1, 'owner')
ON DUPLICATE KEY UPDATE
  shop_role = VALUES(shop_role);

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

INSERT INTO role_permissions (role, permission_code)
VALUES
  ('user', 'dashboard:view'),
  ('user', 'tools:view'),
  ('user', 'shops:view'),
  ('user', 'settings:view')
ON DUPLICATE KEY UPDATE
  permission_code = VALUES(permission_code);

INSERT INTO tool_modules (id, name, description, status, sort_order)
VALUES
  ('product-research', '商品采集', '解析店铺商品 JSON，采集 SKC、SKU、图片、名称、根数、价格、状态和创建时间。', 'active', 10),
  ('delivery-json-extract', '发货 JSON 提取', '解析发货单 JSON，保存明细并支持查询、分页和 Excel 导出。', 'active', 15),
  ('price-monitor', '价格监控', '预留商品价格、库存和竞品变化监控能力。', 'planning', 20),
  ('order-assistant', '订单助手', '预留订单同步、异常提醒和履约跟踪能力。', 'planning', 30),
  ('analytics', '数据看板', '预留销售趋势、利润估算和运营指标分析能力。', 'planning', 40)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  status = VALUES(status),
  sort_order = VALUES(sort_order);

INSERT INTO tool_packages (
  id, version, name, description, category, icon, status, package_type, entry_type, entry_path,
  panel_key, removable, recommended, sort_order, permissions_json, manifest_json
)
VALUES
  (
    'product-research',
    '1.0.0',
    '商品采集',
    '导入店铺商品 JSON，维护 SKC、SKU、价格、成本和产品配置。',
    '店铺运营工具',
    'search',
    'active',
    'builtin',
    'native',
    '',
    'product-research',
    0,
    1,
    10,
    JSON_ARRAY('tools:view', 'tools:manage'),
    JSON_OBJECT(
      'toolId', 'product-research',
      'toolName', '商品采集',
      'version', '1.0.0',
      'toolDesc', '导入店铺商品 JSON，维护 SKC、SKU、价格、成本和产品配置。',
      'toolIcon', 'search',
      'toolCategory', '店铺运营工具',
      'toolStatus', 'active',
      'packageType', 'builtin',
      'entryType', 'native',
      'panelKey', 'product-research',
      'isRecommended', true,
      'sortOrder', 10
    )
  ),
  (
    'delivery-json-extract',
    '1.0.0',
    '发货 JSON 提取',
    '解析发货单 JSON，支持查询、分页和 Excel 导出。',
    '数据工具',
    'file-json',
    'active',
    'builtin',
    'native',
    '',
    'delivery-json-extract',
    0,
    0,
    15,
    JSON_ARRAY('tools:view', 'tools:manage'),
    JSON_OBJECT(
      'toolId', 'delivery-json-extract',
      'toolName', '发货 JSON 提取',
      'version', '1.0.0',
      'toolDesc', '解析发货单 JSON，支持查询、分页和 Excel 导出。',
      'toolIcon', 'file-json',
      'toolCategory', '数据工具',
      'toolStatus', 'active',
      'packageType', 'builtin',
      'entryType', 'native',
      'panelKey', 'delivery-json-extract',
      'isRecommended', false,
      'sortOrder', 15
    )
  )
ON DUPLICATE KEY UPDATE
  version = VALUES(version),
  name = VALUES(name),
  description = VALUES(description),
  category = VALUES(category),
  icon = VALUES(icon),
  status = VALUES(status),
  package_type = VALUES(package_type),
  entry_type = VALUES(entry_type),
  entry_path = VALUES(entry_path),
  panel_key = VALUES(panel_key),
  removable = VALUES(removable),
  recommended = VALUES(recommended),
  sort_order = VALUES(sort_order),
  permissions_json = VALUES(permissions_json),
  manifest_json = VALUES(manifest_json);

INSERT INTO system_settings (setting_key, setting_value, description, updated_by)
VALUES
  ('api_base_url', '', '后端 API 基础地址', 1),
  ('shop_alias', '', '默认店铺别名', 1),
  ('sync_interval', '30 分钟', '默认同步间隔', 1),
  ('webhook_url', '', 'Webhook 地址', 1)
ON DUPLICATE KEY UPDATE
  setting_value = VALUES(setting_value),
  description = VALUES(description),
  updated_by = VALUES(updated_by);

INSERT INTO schema_migrations (version, name, checksum)
VALUES
  ('001', '001_init.sql', '5066594cbc036589e57c711f6cf22404ac108fe3089d9819ed18a4de6ae1798f'),
  ('002', '002_permissions.sql', 'af28a590841b0ffec81b181da6af5020f8eb820ce8435d09a91475a6b0d89608'),
  ('003', '003_feature_interfaces.sql', '49c58dfd991976f803d97398cbf6029d1ac984033b12ce95936de697bb315e5b'),
  ('004', '004_delivery_extract_tool.sql', 'f46f4d7f88fccc1a4c9f6ba19d357077740ebfb61dfa2e83a11beec31672c319'),
  ('005', '005_delivery_extract_supplier.sql', 'f78b8e025d01e19f82cff7dbedc59846b63de34fa4261d6ffc2edd3bdff2a4c3'),
  ('006', '006_delivery_extract_shop_link.sql', 'ad68796bdba5354892d37ef3bc00f12c9272d08e6ee18f06fbd15999bc4536c5'),
  ('007', '007_shop_extra_fields.sql', 'b77780d13d8e32cef8acc8fb7d7cd8584b653a0f5e7ed1a777eea1f9cca63e0d'),
  ('008', '008_seed_kunsong_grocery_shop.sql', '53b981c3dedf016dfb7031943e6cbe5eba42eaae5af4b56651171dda8530c12a'),
  ('009', '009_clear_kunsong_optional_fields.sql', '45036896e3c28c3e049395df80f122920e13a73cba580eb1189ad936bb6a7702'),
  ('010', '010_remove_api_verification_extract.sql', '2560f2a1da4d54514c8c2c43d3cd9f7772ad182fe2881c3a7f1d7b37d32272b6'),
  ('011', '011_add_schema_comments.sql', '6f342343b4ddf34060439592e15c91949f4ec71866f545370f16efdfa4092674'),
  ('012', '012_product_collection.sql', 'd7243c493d715f6e99bff20c62c9d3b84dfce0fec016d32baa6c75fef09d6112'),
  ('013', '013_delivery_extract_express_batch.sql', '0291043eb5b65c079bb2a1f534af29dc116bcb62069698ebfea3977bbace9b9b'),
  ('014', '014_tool_packages.sql', '4e6c8b2dcb8c202beda1a36d21560eb4d0e0bf698fe02cdae073f3a678e312c0'),
  ('015', '015_product_collection_supplier_shop_link.sql', 'ba4dc6ab62f7bcc6a37985329c0c67d8afd34719b2fbea362be95ad23ec370c9'),
  ('016', '016_remove_shop_url.sql', 'daedad226e5ef7f510a1a037b00a4fab3680d2f949934760c47f297cf38b9984'),
  ('017', '017_sales_overall_dashboard.sql', 'f5b601710fa2e26773ec099add3fc34caa23582d8143edd6911b855b50889938'),
  ('018', '018_delivery_extract_supplier_time.sql', 'bc6eb14c01378c23350da47b359fe561bd106d4edc3e36628b3bef80cb1e57ae'),
  ('019', '019_delivery_extract_order_skc_index.sql', '8889c3ee099f4f14769f03ca90b067c1f69a32c20ac0d6fdb1bc6546234a92b7')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  checksum = VALUES(checksum);

SET FOREIGN_KEY_CHECKS = 1;
