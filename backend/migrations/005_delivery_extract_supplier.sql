ALTER TABLE delivery_extract_rows
  ADD COLUMN supplier_id VARCHAR(64) NOT NULL DEFAULT '' AFTER batch_id,
  ADD KEY idx_delivery_extract_rows_supplier_id (supplier_id);
