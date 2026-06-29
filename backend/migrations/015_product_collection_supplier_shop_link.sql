ALTER TABLE product_collection_products
  DROP FOREIGN KEY fk_product_collection_products_shop_id;

ALTER TABLE product_collection_products
  DROP INDEX idx_product_collection_products_shop_id;

ALTER TABLE product_collection_products
  DROP COLUMN shop_id;
