ALTER TABLE delivery_extract_rows
  ADD KEY idx_delivery_extract_rows_order_skc (delivery_order_sn, skc);
