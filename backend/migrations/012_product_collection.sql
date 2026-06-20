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
  shop_id BIGINT UNSIGNED NULL COMMENT '关联店铺ID',
  source_json LONGTEXT NOT NULL COMMENT '原始商品JSON',
  created_by BIGINT UNSIGNED NULL COMMENT '导入人用户ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '导入时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_product_collection_products_skc (product_skc_id),
  KEY idx_product_collection_products_sku (product_sku_id),
  KEY idx_product_collection_products_shop_id (shop_id),
  KEY idx_product_collection_products_supplier_id (supplier_id),
  KEY idx_product_collection_products_status (skc_top_status),
  KEY idx_product_collection_products_created_at (product_created_at),
  CONSTRAINT fk_product_collection_products_shop_id FOREIGN KEY (shop_id) REFERENCES shops(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT fk_product_collection_products_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品采集表';

INSERT INTO tool_modules (id, name, description, status, sort_order)
VALUES
  ('product-research', '商品采集', '解析店铺商品 JSON，采集 SKC、SKU、图片、名称、根数、价格、状态和创建时间。', 'active', 10)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  status = VALUES(status),
  sort_order = VALUES(sort_order);
