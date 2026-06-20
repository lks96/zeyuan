ALTER TABLE delivery_extract_rows
  ADD COLUMN express_batch_sn VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发货批次号' AFTER delivery_order_sn,
  ADD KEY idx_delivery_extract_rows_express_batch_sn (express_batch_sn);
