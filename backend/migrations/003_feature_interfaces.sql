CREATE TABLE IF NOT EXISTS tool_modules (
  id VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  status ENUM('planning', 'active', 'paused') NOT NULL DEFAULT 'planning',
  sort_order INT NOT NULL DEFAULT 100,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_tool_modules_status (status),
  KEY idx_tool_modules_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS system_settings (
  setting_key VARCHAR(80) NOT NULL,
  setting_value VARCHAR(500) NOT NULL DEFAULT '',
  description VARCHAR(255) NOT NULL DEFAULT '',
  updated_by BIGINT UNSIGNED NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (setting_key),
  CONSTRAINT fk_system_settings_updated_by FOREIGN KEY (updated_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO tool_modules (id, name, description, status, sort_order)
VALUES
  ('product-research', '商品采集', '预留 Temu 商品链接采集、SKU 信息解析和批量导入能力。', 'planning', 10),
  ('price-monitor', '价格监控', '预留商品价格、库存和竞品变化监控能力。', 'planning', 20),
  ('order-assistant', '订单助手', '预留订单同步、异常提醒和履约跟踪能力。', 'planning', 30),
  ('analytics', '数据看板', '预留销售趋势、利润估算和运营指标分析能力。', 'planning', 40)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  sort_order = VALUES(sort_order);

INSERT INTO system_settings (setting_key, setting_value, description, updated_by)
VALUES
  ('api_base_url', 'http://localhost:8080', 'Backend API base URL', 1),
  ('shop_alias', '', 'Default shop alias', 1),
  ('sync_interval', '30 分钟', 'Default sync interval', 1),
  ('webhook_url', '', 'Webhook URL', 1)
ON DUPLICATE KEY UPDATE
  description = VALUES(description);
