ALTER TABLE delivery_extract_rows
  ADD COLUMN shop_id BIGINT UNSIGNED NULL AFTER supplier_id,
  ADD KEY idx_delivery_extract_rows_shop_id (shop_id),
  ADD CONSTRAINT fk_delivery_extract_rows_shop_id FOREIGN KEY (shop_id) REFERENCES shops(id)
    ON DELETE SET NULL ON UPDATE CASCADE;
