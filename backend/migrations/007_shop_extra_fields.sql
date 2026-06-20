ALTER TABLE shops
  ADD COLUMN eu_representative VARCHAR(255) NOT NULL DEFAULT '' AFTER external_code,
  ADD COLUMN shop_url VARCHAR(1000) NOT NULL DEFAULT '' AFTER eu_representative;
