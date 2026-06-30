ALTER TABLE delivery_extract_rows
  ADD COLUMN expect_pick_up_goods_time BIGINT NOT NULL DEFAULT 0 COMMENT '上门取货时间戳' AFTER express_batch_sn,
  DROP FOREIGN KEY fk_delivery_extract_rows_shop_id,
  DROP INDEX idx_delivery_extract_rows_shop_id,
  DROP COLUMN shop_id,
  ADD KEY idx_delivery_extract_rows_expect_pick_up_goods_time (expect_pick_up_goods_time);
