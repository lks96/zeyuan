package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"temu-tools/backend/internal/models"
)

type Store struct {
	db *sql.DB
}

type CreateUserParams struct {
	Username     string
	PasswordHash string
	DisplayName  string
	Role         models.UserRole
	Status       string
}

type UpdateUserParams struct {
	DisplayName  string
	Role         models.UserRole
	Status       string
	PasswordHash string
}

type CreateShopParams struct {
	ShopName         string
	Platform         string
	ExternalCode     string
	EuRepresentative string
	ShopURL          string
	Status           string
	CreatedBy        int64
}

type UpdateShopParams struct {
	ShopName         string
	Platform         string
	ExternalCode     string
	EuRepresentative string
	ShopURL          string
	Status           string
}

type UpsertShopParams struct {
	ShopName         string
	Platform         string
	ExternalCode     string
	EuRepresentative string
	ShopURL          string
	Status           string
	CreatedBy        int64
}

type UpsertToolModuleParams struct {
	ID          string
	Name        string
	Description string
	Status      string
	SortOrder   int
}

type SaveDeliveryExtractParams struct {
	SourceFile  string
	BatchDate   string
	SourceTotal int
	Rows        []models.DeliveryExtractRow
	CreatedBy   int64
}

type DeliveryExtractRowsOptions struct {
	Query     string
	BatchDate string
	RowIDs    []int64
	Page      int
	PageSize  int
	AllRows   bool
}

type SaveProductCollectionParams struct {
	SourceTotal int
	Products    []models.ProductCollectionProduct
	Shop        models.Shop
	CreatedBy   int64
}

type ProductCollectionProductsOptions struct {
	Query     string
	ShopID    int64
	Status    int
	HasStatus bool
	Page      int
	PageSize  int
	AllRows   bool
}

type UpdateProductCollectionProductParams struct {
	ProductConfig string
	CostPriceCent int
}

type ProductCollectionMaintenanceItem struct {
	ProductSkcID  string
	ProductConfig string
	CostPriceCent int
}

type BatchUpdateProductCollectionMaintenanceParams struct {
	Items []ProductCollectionMaintenanceItem
}

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) GetUser(ctx context.Context, id int64) (models.User, error) {
	return s.getUser(ctx, id, true)
}

func (s *Store) GetUserAnyStatus(ctx context.Context, id int64) (models.User, error) {
	return s.getUser(ctx, id, false)
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (models.User, string, error) {
	const query = `
SELECT id, username, password_hash, display_name, role, status, created_at
FROM users
WHERE username = ? AND status = 'active'
LIMIT 1`

	var user models.User
	var passwordHash string
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&passwordHash,
		&user.DisplayName,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, "", ErrNotFound
	}
	return user, passwordHash, err
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	const query = `
SELECT id, username, display_name, role, status, created_at
FROM users
ORDER BY id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.DisplayName,
			&user.Role,
			&user.Status,
			&user.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func (s *Store) ListPermissionsForUser(ctx context.Context, user models.User) ([]string, error) {
	const query = `
SELECT p.code
FROM permissions p
INNER JOIN role_permissions rp ON rp.permission_code = p.code
WHERE rp.role = ?
ORDER BY p.category ASC, p.code ASC`

	rows, err := s.db.QueryContext(ctx, query, user.Role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, rows.Err()
}

func (s *Store) UserHasPermission(ctx context.Context, user models.User, permissionCode string) (bool, error) {
	const query = `
SELECT 1
FROM role_permissions
WHERE role = ? AND permission_code = ?
LIMIT 1`

	var exists int
	err := s.db.QueryRowContext(ctx, query, user.Role, permissionCode).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	const query = `
SELECT code, name, category, description, created_at
FROM permissions
ORDER BY category ASC, code ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]models.Permission, 0)
	for rows.Next() {
		var permission models.Permission
		if err := rows.Scan(
			&permission.Code,
			&permission.Name,
			&permission.Category,
			&permission.Description,
			&permission.CreatedAt,
		); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, rows.Err()
}

func (s *Store) ListRolePermissions(ctx context.Context, role models.UserRole) ([]string, error) {
	const query = `
SELECT permission_code
FROM role_permissions
WHERE role = ?
ORDER BY permission_code ASC`

	rows, err := s.db.QueryContext(ctx, query, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, rows.Err()
}

func (s *Store) ReplaceRolePermissions(ctx context.Context, role models.UserRole, permissionCodes []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role = ?`, role); err != nil {
		return err
	}

	const query = `
INSERT INTO role_permissions (role, permission_code)
VALUES (?, ?)`
	for _, permissionCode := range permissionCodes {
		if _, err := tx.ExecContext(ctx, query, role, permissionCode); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) CreateUser(ctx context.Context, params CreateUserParams) (models.User, error) {
	const query = `
INSERT INTO users (username, password_hash, display_name, role, status)
VALUES (?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(
		ctx,
		query,
		params.Username,
		params.PasswordHash,
		params.DisplayName,
		params.Role,
		params.Status,
	)
	if err != nil {
		return models.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}
	return s.GetUserAnyStatus(ctx, id)
}

func (s *Store) UpdateUser(ctx context.Context, id int64, params UpdateUserParams) (models.User, error) {
	var err error
	if params.PasswordHash == "" {
		const query = `
UPDATE users
SET display_name = ?, role = ?, status = ?
WHERE id = ?`
		_, err = s.db.ExecContext(ctx, query, params.DisplayName, params.Role, params.Status, id)
	} else {
		const query = `
UPDATE users
SET display_name = ?, role = ?, status = ?, password_hash = ?
WHERE id = ?`
		_, err = s.db.ExecContext(ctx, query, params.DisplayName, params.Role, params.Status, params.PasswordHash, id)
	}
	if err != nil {
		return models.User{}, err
	}

	return s.GetUserAnyStatus(ctx, id)
}

func (s *Store) DisableUser(ctx context.Context, id int64) error {
	const query = `UPDATE users SET status = 'disabled' WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListVisibleShops(ctx context.Context, user models.User) ([]models.Shop, error) {
	if user.IsAdmin() {
		return s.listAllShops(ctx)
	}
	return s.listUserShops(ctx, user.ID)
}

func (s *Store) GetShop(ctx context.Context, id int64) (models.Shop, error) {
	const query = `
SELECT id, shop_name, platform, COALESCE(external_code, ''), eu_representative, shop_url, status, created_at
FROM shops
WHERE id = ?
LIMIT 1`

	var shop models.Shop
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&shop.ID,
		&shop.ShopName,
		&shop.Platform,
		&shop.ExternalCode,
		&shop.EuRepresentative,
		&shop.ShopURL,
		&shop.Status,
		&shop.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Shop{}, ErrNotFound
	}
	return shop, err
}

func (s *Store) CreateShop(ctx context.Context, params CreateShopParams) (models.Shop, error) {
	const query = `
INSERT INTO shops (shop_name, platform, external_code, eu_representative, shop_url, status, created_by)
VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?)`

	result, err := s.db.ExecContext(
		ctx,
		query,
		params.ShopName,
		params.Platform,
		params.ExternalCode,
		params.EuRepresentative,
		params.ShopURL,
		params.Status,
		params.CreatedBy,
	)
	if err != nil {
		return models.Shop{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Shop{}, err
	}
	return s.GetShop(ctx, id)
}

func (s *Store) UpsertShopByExternalCode(ctx context.Context, params UpsertShopParams) (models.Shop, error) {
	const query = `
INSERT INTO shops (shop_name, platform, external_code, eu_representative, shop_url, status, created_by)
VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  shop_name = VALUES(shop_name),
  eu_representative = VALUES(eu_representative),
  shop_url = VALUES(shop_url),
  status = VALUES(status)`

	if _, err := s.db.ExecContext(
		ctx,
		query,
		params.ShopName,
		params.Platform,
		params.ExternalCode,
		params.EuRepresentative,
		params.ShopURL,
		params.Status,
		params.CreatedBy,
	); err != nil {
		return models.Shop{}, err
	}

	return s.GetShopByPlatformExternalCode(ctx, params.Platform, params.ExternalCode)
}

func (s *Store) GetShopByPlatformExternalCode(ctx context.Context, platform string, externalCode string) (models.Shop, error) {
	const query = `
SELECT id, shop_name, platform, COALESCE(external_code, ''), eu_representative, shop_url, status, created_at
FROM shops
WHERE platform = ? AND external_code = ?
LIMIT 1`

	var shop models.Shop
	err := s.db.QueryRowContext(ctx, query, platform, externalCode).Scan(
		&shop.ID,
		&shop.ShopName,
		&shop.Platform,
		&shop.ExternalCode,
		&shop.EuRepresentative,
		&shop.ShopURL,
		&shop.Status,
		&shop.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Shop{}, ErrNotFound
	}
	return shop, err
}

func (s *Store) UpdateShop(ctx context.Context, id int64, params UpdateShopParams) (models.Shop, error) {
	const query = `
UPDATE shops
SET shop_name = ?, platform = ?, external_code = NULLIF(?, ''), eu_representative = ?, shop_url = ?, status = ?
WHERE id = ?`

	_, err := s.db.ExecContext(
		ctx,
		query,
		params.ShopName,
		params.Platform,
		params.ExternalCode,
		params.EuRepresentative,
		params.ShopURL,
		params.Status,
		id,
	)
	if err != nil {
		return models.Shop{}, err
	}

	return s.GetShop(ctx, id)
}

func (s *Store) CloseShop(ctx context.Context, id int64) error {
	const query = `UPDATE shops SET status = 'closed' WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	if _, err := s.GetShop(ctx, id); errors.Is(err, ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Store) AssignShop(ctx context.Context, userID int64, shopID int64, shopRole string) error {
	const query = `
INSERT INTO user_shops (user_id, shop_id, shop_role)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE shop_role = VALUES(shop_role)`

	_, err := s.db.ExecContext(ctx, query, userID, shopID, shopRole)
	return err
}

func (s *Store) RemoveShopAssignment(ctx context.Context, userID int64, shopID int64) error {
	const query = `DELETE FROM user_shops WHERE user_id = ? AND shop_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID, shopID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListUserShops(ctx context.Context, userID int64) ([]models.UserShop, error) {
	const query = `
SELECT us.user_id, us.shop_id, s.shop_name, us.shop_role, us.created_at
FROM user_shops us
INNER JOIN shops s ON s.id = us.shop_id
WHERE us.user_id = ?
ORDER BY s.id ASC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments := make([]models.UserShop, 0)
	for rows.Next() {
		var assignment models.UserShop
		if err := rows.Scan(
			&assignment.UserID,
			&assignment.ShopID,
			&assignment.ShopName,
			&assignment.ShopRole,
			&assignment.CreatedAt,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, rows.Err()
}

func (s *Store) ListToolModules(ctx context.Context) ([]models.ToolModule, error) {
	const query = `
SELECT id, name, description, status, sort_order, created_at, updated_at
FROM tool_modules
ORDER BY sort_order ASC, id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	modules := make([]models.ToolModule, 0)
	for rows.Next() {
		var module models.ToolModule
		if err := rows.Scan(
			&module.ID,
			&module.Name,
			&module.Description,
			&module.Status,
			&module.SortOrder,
			&module.CreatedAt,
			&module.UpdatedAt,
		); err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, rows.Err()
}

func (s *Store) ListToolPackages(ctx context.Context) ([]models.ToolPackage, error) {
	const query = `
SELECT id, version, name, description, category, icon, status, package_type, entry_type, entry_path,
       panel_key, removable, recommended, sort_order, permissions_json, manifest_json, installed_at, updated_at
FROM tool_packages
ORDER BY sort_order ASC, id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		if strings.Contains(err.Error(), "tool_packages") {
			return s.listLegacyToolPackages(ctx)
		}
		return nil, err
	}
	defer rows.Close()

	packages := make([]models.ToolPackage, 0)
	for rows.Next() {
		var pkg models.ToolPackage
		var permissionsJSON string
		if err := rows.Scan(
			&pkg.ID,
			&pkg.Version,
			&pkg.Name,
			&pkg.Description,
			&pkg.Category,
			&pkg.Icon,
			&pkg.Status,
			&pkg.PackageType,
			&pkg.EntryType,
			&pkg.EntryPath,
			&pkg.PanelKey,
			&pkg.Removable,
			&pkg.Recommended,
			&pkg.SortOrder,
			&permissionsJSON,
			&pkg.ManifestJSON,
			&pkg.InstalledAt,
			&pkg.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(permissionsJSON), &pkg.Permissions); err != nil {
			pkg.Permissions = []string{}
		}
		packages = append(packages, pkg)
	}

	return packages, rows.Err()
}

func (s *Store) GetToolPackage(ctx context.Context, id string) (models.ToolPackage, error) {
	packages, err := s.ListToolPackages(ctx)
	if err != nil {
		return models.ToolPackage{}, err
	}

	for _, pkg := range packages {
		if pkg.ID == id {
			return pkg, nil
		}
	}

	return models.ToolPackage{}, ErrNotFound
}

func (s *Store) listLegacyToolPackages(ctx context.Context) ([]models.ToolPackage, error) {
	modules, err := s.ListToolModules(ctx)
	if err != nil {
		return nil, err
	}

	packages := make([]models.ToolPackage, 0, len(modules))
	for _, module := range modules {
		pkg := models.ToolPackage{
			ID:           module.ID,
			Version:      "1.0.0",
			Name:         module.Name,
			Description:  module.Description,
			Category:     "店铺运营工具",
			Icon:         "blocks",
			Status:       module.Status,
			PackageType:  "builtin",
			EntryType:    "native",
			PanelKey:     module.ID,
			SortOrder:    module.SortOrder,
			Permissions:  []string{"tools:view", "tools:manage"},
			InstalledAt:  module.CreatedAt,
			UpdatedAt:    module.UpdatedAt,
			ManifestJSON: "{}",
		}
		switch module.ID {
		case "product-research":
			pkg.Description = "导入店铺商品 JSON，维护 SKC、SKU、价格、成本和产品配置。"
			pkg.Category = "店铺运营工具"
			pkg.Icon = "search"
			pkg.Recommended = true
		case "delivery-json-extract":
			pkg.Description = "解析发货单 JSON，支持查询、分页和 Excel 导出。"
			pkg.Category = "数据工具"
			pkg.Icon = "file-json"
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (s *Store) UpsertToolModule(ctx context.Context, params UpsertToolModuleParams) (models.ToolModule, error) {
	const query = `
INSERT INTO tool_modules (id, name, description, status, sort_order)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  status = VALUES(status),
  sort_order = VALUES(sort_order)`

	if _, err := s.db.ExecContext(ctx, query, params.ID, params.Name, params.Description, params.Status, params.SortOrder); err != nil {
		return models.ToolModule{}, err
	}

	return s.GetToolModule(ctx, params.ID)
}

func (s *Store) GetToolModule(ctx context.Context, id string) (models.ToolModule, error) {
	const query = `
SELECT id, name, description, status, sort_order, created_at, updated_at
FROM tool_modules
WHERE id = ?
LIMIT 1`

	var module models.ToolModule
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&module.ID,
		&module.Name,
		&module.Description,
		&module.Status,
		&module.SortOrder,
		&module.CreatedAt,
		&module.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ToolModule{}, ErrNotFound
	}
	return module, err
}

func (s *Store) DeleteToolModule(ctx context.Context, id string) error {
	const query = `DELETE FROM tool_modules WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SaveDeliveryExtract(ctx context.Context, params SaveDeliveryExtractParams) (models.DeliveryExtractBatch, error) {
	targetPayload := struct {
		Data []models.DeliveryExtractRow `json:"data"`
		Date string                      `json:"date"`
	}{
		Data: params.Rows,
		Date: params.BatchDate,
	}
	targetJSON, err := json.Marshal(targetPayload)
	if err != nil {
		return models.DeliveryExtractBatch{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.DeliveryExtractBatch{}, err
	}
	defer tx.Rollback()

	const insertBatch = `
INSERT INTO delivery_extract_batches (source_file, batch_date, source_total, extracted_total, target_json, created_by)
VALUES (?, ?, ?, ?, ?, ?)`
	result, err := tx.ExecContext(
		ctx,
		insertBatch,
		params.SourceFile,
		params.BatchDate,
		params.SourceTotal,
		len(params.Rows),
		string(targetJSON),
		params.CreatedBy,
	)
	if err != nil {
		return models.DeliveryExtractBatch{}, err
	}

	batchID, err := result.LastInsertId()
	if err != nil {
		return models.DeliveryExtractBatch{}, err
	}

	const insertRow = `
INSERT INTO delivery_extract_rows (
  batch_id,
  supplier_id,
  shop_id,
  product_name,
  product_skc_picture,
  delivery_order_sn,
  express_batch_sn,
  skc,
  skc_num,
  sku,
  sku_num,
  receiver_name,
  row_json
)
VALUES (?, ?, (SELECT id FROM shops WHERE platform = 'temu' AND external_code = ? LIMIT 1), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for index := range params.Rows {
		row := &params.Rows[index]
		row.BatchID = batchID
		rowJSON, err := json.Marshal(row)
		if err != nil {
			return models.DeliveryExtractBatch{}, err
		}
		result, err := tx.ExecContext(
			ctx,
			insertRow,
			batchID,
			row.SupplierID,
			row.SupplierID,
			row.ProductName,
			row.ProductSkcPicture,
			row.DeliveryOrderSn,
			row.ExpressBatchSn,
			row.SKC,
			row.SkcNum,
			row.SKU,
			row.SkuNum,
			row.ReceiverName,
			string(rowJSON),
		)
		if err != nil {
			return models.DeliveryExtractBatch{}, err
		}
		rowID, err := result.LastInsertId()
		if err != nil {
			return models.DeliveryExtractBatch{}, err
		}
		row.ID = rowID
	}

	if err := tx.Commit(); err != nil {
		return models.DeliveryExtractBatch{}, err
	}

	return s.GetDeliveryExtractBatch(ctx, batchID, DeliveryExtractRowsOptions{})
}

func (s *Store) ListDeliveryExtractBatches(ctx context.Context, limit int) ([]models.DeliveryExtractBatch, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	const query = `
SELECT id, source_file, batch_date, source_total, extracted_total, created_at
FROM delivery_extract_batches
ORDER BY id DESC
LIMIT ?`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := make([]models.DeliveryExtractBatch, 0)
	for rows.Next() {
		var batch models.DeliveryExtractBatch
		if err := rows.Scan(
			&batch.ID,
			&batch.SourceFile,
			&batch.BatchDate,
			&batch.SourceTotal,
			&batch.ExtractedTotal,
			&batch.CreatedAt,
		); err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}

	return batches, rows.Err()
}

func (s *Store) GetLatestDeliveryExtractBatch(ctx context.Context, options DeliveryExtractRowsOptions) (models.DeliveryExtractBatch, error) {
	options = normalizeDeliveryExtractRowsOptions(options)

	query := `SELECT id FROM delivery_extract_batches`
	args := make([]any, 0, 1)
	if options.BatchDate != "" {
		query += ` WHERE batch_date = ?`
		args = append(args, options.BatchDate)
	}
	query += ` ORDER BY id DESC LIMIT 1`

	var id int64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&id); errors.Is(err, sql.ErrNoRows) {
		return models.DeliveryExtractBatch{}, ErrNotFound
	} else if err != nil {
		return models.DeliveryExtractBatch{}, err
	}
	return s.GetDeliveryExtractBatch(ctx, id, options)
}

func (s *Store) GetDeliveryExtractBatch(ctx context.Context, id int64, options DeliveryExtractRowsOptions) (models.DeliveryExtractBatch, error) {
	options = normalizeDeliveryExtractRowsOptions(options)

	const batchQuery = `
SELECT id, source_file, batch_date, source_total, extracted_total, created_at
FROM delivery_extract_batches
WHERE id = ?
LIMIT 1`
	var batch models.DeliveryExtractBatch
	err := s.db.QueryRowContext(ctx, batchQuery, id).Scan(
		&batch.ID,
		&batch.SourceFile,
		&batch.BatchDate,
		&batch.SourceTotal,
		&batch.ExtractedTotal,
		&batch.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.DeliveryExtractBatch{}, ErrNotFound
	}
	if err != nil {
		return models.DeliveryExtractBatch{}, err
	}

	countWhereClause := "WHERE batch_id = ?"
	rowWhereClause := "WHERE r.batch_id = ?"
	args := []any{id}
	if options.Query != "" {
		countWhereClause += " AND (product_name LIKE ? OR delivery_order_sn LIKE ? OR express_batch_sn LIKE ? OR skc LIKE ? OR sku LIKE ?)"
		rowWhereClause += " AND (r.product_name LIKE ? OR r.delivery_order_sn LIKE ? OR r.express_batch_sn LIKE ? OR r.skc LIKE ? OR r.sku LIKE ?)"
		keyword := "%" + options.Query + "%"
		args = append(args, keyword, keyword, keyword, keyword, keyword)
	}
	if len(options.RowIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(options.RowIDs)), ",")
		countWhereClause += " AND id IN (" + placeholders + ")"
		rowWhereClause += " AND r.id IN (" + placeholders + ")"
		for _, rowID := range options.RowIDs {
			args = append(args, rowID)
		}
	}

	countQuery := "SELECT COUNT(*) FROM delivery_extract_rows " + countWhereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&batch.RowsTotal); err != nil {
		return models.DeliveryExtractBatch{}, err
	}
	if options.AllRows {
		options.PageSize = batch.RowsTotal
	}

	rowQuery := `
SELECT
  r.id,
  r.batch_id,
  r.supplier_id,
  r.shop_id,
  COALESCE(s.shop_name, ''),
  COALESCE(s.eu_representative, ''),
  r.product_name,
  r.product_skc_picture,
  r.delivery_order_sn,
  r.express_batch_sn,
  r.skc,
  r.skc_num,
  r.sku,
  r.sku_num,
  r.receiver_name,
  COALESCE(pc.number_of_pieces_new, 0),
  COALESCE(pc.product_config, ''),
  r.created_at
FROM delivery_extract_rows r
LEFT JOIN shops s ON s.id = r.shop_id OR (r.shop_id IS NULL AND s.platform = 'temu' AND s.external_code = r.supplier_id)
LEFT JOIN product_collection_products pc ON pc.product_skc_id = r.skc AND pc.supplier_id = r.supplier_id
` + rowWhereClause + `
ORDER BY r.id ASC`
	rowArgs := args
	if !options.AllRows {
		offset := (options.Page - 1) * options.PageSize
		rowQuery += `
LIMIT ? OFFSET ?`
		rowArgs = append(rowArgs, options.PageSize, offset)
	}

	rows, err := s.db.QueryContext(ctx, rowQuery, rowArgs...)
	if err != nil {
		return models.DeliveryExtractBatch{}, err
	}
	defer rows.Close()

	batch.Page = options.Page
	batch.PageSize = options.PageSize
	batch.Query = options.Query
	batch.Rows = make([]models.DeliveryExtractRow, 0)
	for rows.Next() {
		var row models.DeliveryExtractRow
		var shopID sql.NullInt64
		if err := rows.Scan(
			&row.ID,
			&row.BatchID,
			&row.SupplierID,
			&shopID,
			&row.ShopName,
			&row.EuRepresentative,
			&row.ProductName,
			&row.ProductSkcPicture,
			&row.DeliveryOrderSn,
			&row.ExpressBatchSn,
			&row.SKC,
			&row.SkcNum,
			&row.SKU,
			&row.SkuNum,
			&row.ReceiverName,
			&row.ProductPieces,
			&row.ProductConfig,
			&row.CreatedAt,
		); err != nil {
			return models.DeliveryExtractBatch{}, err
		}
		if shopID.Valid {
			row.ShopID = shopID.Int64
		}
		batch.Rows = append(batch.Rows, row)
	}

	return batch, rows.Err()
}

func (s *Store) SaveProductCollection(ctx context.Context, params SaveProductCollectionParams, options ProductCollectionProductsOptions) (models.ProductCollectionImportResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ProductCollectionImportResult{}, err
	}
	defer tx.Rollback()

	const query = `
INSERT INTO product_collection_products (
  product_skc_id,
  product_sku_id,
  main_image_url,
  product_name,
  number_of_pieces_new,
  supplier_price_cent,
  skc_top_status,
  product_created_at,
  supplier_id,
  source_json,
  created_by
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  product_sku_id = VALUES(product_sku_id),
  main_image_url = VALUES(main_image_url),
  product_name = VALUES(product_name),
  number_of_pieces_new = VALUES(number_of_pieces_new),
  supplier_price_cent = VALUES(supplier_price_cent),
  skc_top_status = VALUES(skc_top_status),
  product_created_at = VALUES(product_created_at),
  supplier_id = VALUES(supplier_id),
  source_json = VALUES(source_json),
  created_by = VALUES(created_by)`

	imported := 0
	for _, product := range params.Products {
		if product.ProductSkcID == "" {
			continue
		}
		var productCreatedAt any
		if product.ProductCreatedAt != nil {
			productCreatedAt = *product.ProductCreatedAt
		}
		if _, err := tx.ExecContext(
			ctx,
			query,
			product.ProductSkcID,
			product.ProductSkuID,
			product.MainImageURL,
			product.ProductName,
			product.NumberOfPiecesNew,
			product.SupplierPriceCent,
			product.SkcTopStatus,
			productCreatedAt,
			params.Shop.ExternalCode,
			product.RawJSON,
			params.CreatedBy,
		); err != nil {
			return models.ProductCollectionImportResult{}, err
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		return models.ProductCollectionImportResult{}, err
	}

	products, err := s.ListProductCollectionProducts(ctx, options)
	if err != nil {
		return models.ProductCollectionImportResult{}, err
	}

	return models.ProductCollectionImportResult{
		SourceTotal: params.SourceTotal,
		Imported:    imported,
		Shop:        params.Shop,
		Products:    products,
	}, nil
}

func (s *Store) ListProductCollectionProducts(ctx context.Context, options ProductCollectionProductsOptions) (models.ProductCollectionList, error) {
	options = normalizeProductCollectionProductsOptions(options)

	whereClause := "WHERE 1 = 1"
	args := make([]any, 0)
	if options.Query != "" {
		whereClause += ` AND (
  p.product_skc_id LIKE ?
  OR p.product_sku_id LIKE ?
  OR p.product_name LIKE ?
  OR p.product_config LIKE ?
  OR p.supplier_id LIKE ?
  OR COALESCE(s.shop_name, '') LIKE ?
)`
		keyword := "%" + options.Query + "%"
		args = append(args, keyword, keyword, keyword, keyword, keyword, keyword)
	}
	if options.ShopID > 0 {
		whereClause += " AND s.id = ?"
		args = append(args, options.ShopID)
	}
	if options.HasStatus {
		whereClause += " AND p.skc_top_status = ?"
		args = append(args, options.Status)
	}

	list := models.ProductCollectionList{
		Page:     options.Page,
		PageSize: options.PageSize,
		Query:    options.Query,
		Rows:     make([]models.ProductCollectionProduct, 0),
	}

	countQuery := `
SELECT COUNT(*)
FROM product_collection_products p
LEFT JOIN shops s ON s.platform = 'temu' AND s.external_code = p.supplier_id
` + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&list.RowsTotal); err != nil {
		return models.ProductCollectionList{}, err
	}
	if options.AllRows {
		list.PageSize = list.RowsTotal
	}

	rowQuery := `
SELECT
  p.id,
  p.product_skc_id,
  p.product_sku_id,
  p.main_image_url,
  p.product_name,
  p.number_of_pieces_new,
  p.product_config,
  p.supplier_price_cent,
  p.cost_price_cent,
  p.skc_top_status,
  p.product_created_at,
  p.supplier_id,
  s.id,
  COALESCE(s.shop_name, ''),
  p.created_at,
  p.updated_at
FROM product_collection_products p
LEFT JOIN shops s ON s.platform = 'temu' AND s.external_code = p.supplier_id
` + whereClause + `
ORDER BY p.product_created_at IS NULL ASC, p.product_created_at DESC, p.id DESC`
	rowArgs := args
	if !options.AllRows {
		rowQuery += `
LIMIT ? OFFSET ?`
		rowArgs = append(rowArgs, options.PageSize, (options.Page-1)*options.PageSize)
	}
	rows, err := s.db.QueryContext(ctx, rowQuery, rowArgs...)
	if err != nil {
		return models.ProductCollectionList{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.ProductCollectionProduct
		var productCreatedAt sql.NullTime
		var shopID sql.NullInt64
		if err := rows.Scan(
			&product.ID,
			&product.ProductSkcID,
			&product.ProductSkuID,
			&product.MainImageURL,
			&product.ProductName,
			&product.NumberOfPiecesNew,
			&product.ProductConfig,
			&product.SupplierPriceCent,
			&product.CostPriceCent,
			&product.SkcTopStatus,
			&productCreatedAt,
			&product.SupplierID,
			&shopID,
			&product.ShopName,
			&product.ImportedAt,
			&product.UpdatedAt,
		); err != nil {
			return models.ProductCollectionList{}, err
		}
		if productCreatedAt.Valid {
			product.ProductCreatedAt = &productCreatedAt.Time
		}
		if shopID.Valid {
			product.ShopID = shopID.Int64
		}
		list.Rows = append(list.Rows, product)
	}

	return list, rows.Err()
}

func (s *Store) UpdateProductCollectionProduct(ctx context.Context, id int64, params UpdateProductCollectionProductParams) (models.ProductCollectionProduct, error) {
	const query = `
UPDATE product_collection_products
SET product_config = ?, cost_price_cent = ?
WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, params.ProductConfig, params.CostPriceCent, id)
	if err != nil {
		return models.ProductCollectionProduct{}, err
	}
	return s.GetProductCollectionProduct(ctx, id)
}

func (s *Store) BatchUpdateProductCollectionMaintenance(ctx context.Context, params BatchUpdateProductCollectionMaintenanceParams, options ProductCollectionProductsOptions) (models.ProductCollectionBatchUpdateResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ProductCollectionBatchUpdateResult{}, err
	}
	defer tx.Rollback()

	const query = `
UPDATE product_collection_products
SET product_config = ?, cost_price_cent = ?
WHERE product_skc_id = ?`
	notFoundSkcs := make([]string, 0)
	updated := 0
	for _, item := range params.Items {
		result, err := tx.ExecContext(ctx, query, item.ProductConfig, item.CostPriceCent, item.ProductSkcID)
		if err != nil {
			return models.ProductCollectionBatchUpdateResult{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return models.ProductCollectionBatchUpdateResult{}, err
		}
		if affected == 0 {
			var exists int
			if err := tx.QueryRowContext(ctx, `SELECT 1 FROM product_collection_products WHERE product_skc_id = ? LIMIT 1`, item.ProductSkcID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
				notFoundSkcs = append(notFoundSkcs, item.ProductSkcID)
				continue
			} else if err != nil {
				return models.ProductCollectionBatchUpdateResult{}, err
			}
			updated++
			continue
		}
		updated++
	}

	if err := tx.Commit(); err != nil {
		return models.ProductCollectionBatchUpdateResult{}, err
	}

	products, err := s.ListProductCollectionProducts(ctx, options)
	if err != nil {
		return models.ProductCollectionBatchUpdateResult{}, err
	}

	return models.ProductCollectionBatchUpdateResult{
		Total:        len(params.Items),
		Updated:      updated,
		NotFoundSkcs: notFoundSkcs,
		Products:     products,
	}, nil
}

func (s *Store) GetProductCollectionProduct(ctx context.Context, id int64) (models.ProductCollectionProduct, error) {
	const query = `
SELECT
  p.id,
  p.product_skc_id,
  p.product_sku_id,
  p.main_image_url,
  p.product_name,
  p.number_of_pieces_new,
  p.product_config,
  p.supplier_price_cent,
  p.cost_price_cent,
  p.skc_top_status,
  p.product_created_at,
  p.supplier_id,
  s.id,
  COALESCE(s.shop_name, ''),
  p.created_at,
  p.updated_at
FROM product_collection_products p
LEFT JOIN shops s ON s.platform = 'temu' AND s.external_code = p.supplier_id
WHERE p.id = ?
LIMIT 1`
	var product models.ProductCollectionProduct
	var productCreatedAt sql.NullTime
	var shopID sql.NullInt64
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.ProductSkcID,
		&product.ProductSkuID,
		&product.MainImageURL,
		&product.ProductName,
		&product.NumberOfPiecesNew,
		&product.ProductConfig,
		&product.SupplierPriceCent,
		&product.CostPriceCent,
		&product.SkcTopStatus,
		&productCreatedAt,
		&product.SupplierID,
		&shopID,
		&product.ShopName,
		&product.ImportedAt,
		&product.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ProductCollectionProduct{}, ErrNotFound
	}
	if err != nil {
		return models.ProductCollectionProduct{}, err
	}
	if productCreatedAt.Valid {
		product.ProductCreatedAt = &productCreatedAt.Time
	}
	if shopID.Valid {
		product.ShopID = shopID.Int64
	}
	return product, nil
}

func normalizeDeliveryExtractRowsOptions(options DeliveryExtractRowsOptions) DeliveryExtractRowsOptions {
	options.Query = strings.TrimSpace(options.Query)
	options.BatchDate = strings.TrimSpace(options.BatchDate)
	if options.AllRows {
		options.Page = 1
		return options
	}
	if options.Page <= 0 {
		options.Page = 1
	}
	if options.PageSize <= 0 {
		options.PageSize = defaultPageSize
	}
	if options.PageSize > maxPageSize {
		options.PageSize = maxPageSize
	}
	return options
}

func normalizeProductCollectionProductsOptions(options ProductCollectionProductsOptions) ProductCollectionProductsOptions {
	options.Query = strings.TrimSpace(options.Query)
	if options.AllRows {
		options.Page = 1
		return options
	}
	if options.Page <= 0 {
		options.Page = 1
	}
	if options.PageSize <= 0 {
		options.PageSize = defaultPageSize
	}
	if options.PageSize > maxPageSize {
		options.PageSize = maxPageSize
	}
	return options
}

func (s *Store) ListSystemSettings(ctx context.Context) ([]models.SystemSetting, error) {
	const query = `
SELECT setting_key, setting_value, description, updated_at
FROM system_settings
ORDER BY setting_key ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]models.SystemSetting, 0)
	for rows.Next() {
		var setting models.SystemSetting
		if err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.Description,
			&setting.UpdatedAt,
		); err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}

	return settings, rows.Err()
}

func (s *Store) UpdateSystemSettings(ctx context.Context, values map[string]string, updatedBy int64) ([]models.SystemSetting, error) {
	const query = `
INSERT INTO system_settings (setting_key, setting_value, updated_by)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
  setting_value = VALUES(setting_value),
  updated_by = VALUES(updated_by)`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for key, value := range values {
		if _, err := tx.ExecContext(ctx, query, key, value, updatedBy); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.ListSystemSettings(ctx)
}

func (s *Store) TenantSummary(ctx context.Context, user models.User) (models.TenantSummary, error) {
	totalUsers, err := s.count(ctx, "users")
	if err != nil {
		return models.TenantSummary{}, err
	}

	totalShops, err := s.count(ctx, "shops")
	if err != nil {
		return models.TenantSummary{}, err
	}

	visibleShops, err := s.countVisibleShops(ctx, user)
	if err != nil {
		return models.TenantSummary{}, err
	}

	return models.TenantSummary{
		CurrentUser:     user,
		TotalUsers:      totalUsers,
		TotalShops:      totalShops,
		VisibleShops:    visibleShops,
		AdminCanViewAll: user.IsAdmin(),
	}, nil
}

func (s *Store) getUser(ctx context.Context, id int64, activeOnly bool) (models.User, error) {
	query := `
SELECT id, username, display_name, role, status, created_at
FROM users
WHERE id = ?`
	if activeOnly {
		query += ` AND status = 'active'`
	}
	query += ` LIMIT 1`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) listAllShops(ctx context.Context) ([]models.Shop, error) {
	const query = `
SELECT id, shop_name, platform, COALESCE(external_code, ''), eu_representative, shop_url, status, created_at
FROM shops
ORDER BY id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shops := make([]models.Shop, 0)
	for rows.Next() {
		var shop models.Shop
		if err := rows.Scan(
			&shop.ID,
			&shop.ShopName,
			&shop.Platform,
			&shop.ExternalCode,
			&shop.EuRepresentative,
			&shop.ShopURL,
			&shop.Status,
			&shop.CreatedAt,
		); err != nil {
			return nil, err
		}
		shops = append(shops, shop)
	}

	return shops, rows.Err()
}

func (s *Store) listUserShops(ctx context.Context, userID int64) ([]models.Shop, error) {
	const query = `
SELECT s.id, s.shop_name, s.platform, COALESCE(s.external_code, ''), s.eu_representative, s.shop_url, s.status, us.shop_role, s.created_at
FROM shops s
INNER JOIN user_shops us ON us.shop_id = s.id
WHERE us.user_id = ?
ORDER BY s.id ASC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shops := make([]models.Shop, 0)
	for rows.Next() {
		var shop models.Shop
		if err := rows.Scan(
			&shop.ID,
			&shop.ShopName,
			&shop.Platform,
			&shop.ExternalCode,
			&shop.EuRepresentative,
			&shop.ShopURL,
			&shop.Status,
			&shop.ShopRole,
			&shop.CreatedAt,
		); err != nil {
			return nil, err
		}
		shops = append(shops, shop)
	}

	return shops, rows.Err()
}

func (s *Store) count(ctx context.Context, table string) (int, error) {
	query := "SELECT COUNT(*) FROM " + table

	var total int
	if err := s.db.QueryRowContext(ctx, query).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Store) countVisibleShops(ctx context.Context, user models.User) (int, error) {
	if user.IsAdmin() {
		return s.count(ctx, "shops")
	}

	const query = `SELECT COUNT(*) FROM user_shops WHERE user_id = ?`
	var total int
	if err := s.db.QueryRowContext(ctx, query, user.ID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
