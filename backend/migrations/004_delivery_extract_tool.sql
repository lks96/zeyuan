CREATE TABLE IF NOT EXISTS delivery_extract_batches (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_file VARCHAR(255) NOT NULL DEFAULT '',
  batch_date VARCHAR(16) NOT NULL DEFAULT '',
  source_total INT NOT NULL DEFAULT 0,
  extracted_total INT NOT NULL DEFAULT 0,
  target_json LONGTEXT NOT NULL,
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_delivery_extract_batches_created_at (created_at),
  KEY idx_delivery_extract_batches_batch_date (batch_date),
  CONSTRAINT fk_delivery_extract_batches_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS delivery_extract_rows (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  batch_id BIGINT UNSIGNED NOT NULL,
  product_name TEXT NOT NULL,
  product_skc_picture VARCHAR(1000) NOT NULL DEFAULT '',
  delivery_order_sn VARCHAR(64) NOT NULL DEFAULT '',
  skc VARCHAR(64) NOT NULL DEFAULT '',
  skc_num INT NOT NULL DEFAULT 0,
  sku VARCHAR(64) NOT NULL DEFAULT '',
  sku_num INT NOT NULL DEFAULT 0,
  receiver_name VARCHAR(255) NOT NULL DEFAULT '',
  row_json LONGTEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_delivery_extract_rows_batch_id (batch_id),
  KEY idx_delivery_extract_rows_delivery_order_sn (delivery_order_sn),
  KEY idx_delivery_extract_rows_skc (skc),
  CONSTRAINT fk_delivery_extract_rows_batch_id FOREIGN KEY (batch_id) REFERENCES delivery_extract_batches(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO tool_modules (id, name, description, status, sort_order)
VALUES
  ('delivery-json-extract', '发货 JSON 提取', '从 other/source.json 提取字段为 target.json 格式，并保存到数据库。', 'active', 15)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  status = VALUES(status),
  sort_order = VALUES(sort_order);
