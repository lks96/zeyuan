DELETE FROM shops
WHERE shop_name IN ('Temu Main Shop', 'Temu Test Shop')
   OR shop_name LIKE 'API Test Shop%';

INSERT INTO shops (shop_name, platform, external_code, eu_representative, shop_url, status, created_by)
VALUES ('Kunsong Grocery', 'temu', '634418227150594', '', '', 'active', 1)
ON DUPLICATE KEY UPDATE
  shop_name = VALUES(shop_name),
  eu_representative = VALUES(eu_representative),
  shop_url = VALUES(shop_url),
  status = VALUES(status);

INSERT IGNORE INTO user_shops (user_id, shop_id, shop_role)
SELECT 1, id, 'owner'
FROM shops
WHERE platform = 'temu' AND external_code = '634418227150594'
LIMIT 1;

UPDATE delivery_extract_rows r
INNER JOIN shops s ON s.platform = 'temu' AND s.external_code = r.supplier_id
SET r.shop_id = s.id
WHERE r.supplier_id = '634418227150594';
