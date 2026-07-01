package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"temu-tools/backend/internal/auth"
	"temu-tools/backend/internal/config"
	"temu-tools/backend/internal/database"
	"temu-tools/backend/internal/models"
	"temu-tools/backend/internal/store"
)

type apiResponse struct {
	Data any `json:"data"`
}

type apiError struct {
	Error string `json:"error"`
}

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

type healthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Database  string    `json:"database"`
	Timestamp time.Time `json:"timestamp"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	User        models.User `json:"user"`
	Token       string      `json:"token"`
	Permissions []string    `json:"permissions"`
}

type createUserRequest struct {
	Username    string          `json:"username"`
	Password    string          `json:"password"`
	DisplayName string          `json:"displayName"`
	Role        models.UserRole `json:"role"`
	Status      string          `json:"status"`
}

type updateUserRequest struct {
	Password    string          `json:"password"`
	DisplayName string          `json:"displayName"`
	Role        models.UserRole `json:"role"`
	Status      string          `json:"status"`
}

type createShopRequest struct {
	ShopName         string `json:"shopName"`
	Platform         string `json:"platform"`
	ExternalCode     string `json:"externalCode"`
	EuRepresentative string `json:"euRepresentative"`
	Status           string `json:"status"`
}

type updateShopRequest struct {
	ShopName         string `json:"shopName"`
	Platform         string `json:"platform"`
	ExternalCode     string `json:"externalCode"`
	EuRepresentative string `json:"euRepresentative"`
	Status           string `json:"status"`
}

type assignShopRequest struct {
	ShopID   int64  `json:"shopId"`
	ShopRole string `json:"shopRole"`
}

type upsertToolModuleRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sortOrder"`
}

type updateSettingsRequest struct {
	Values map[string]string `json:"values"`
}

type rolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

type deliveryExtractSource struct {
	List   []deliveryExtractSourceItem `json:"list"`
	Result deliveryExtractSourceResult `json:"result"`
	Total  int                         `json:"total"`
}

type deliveryExtractSourceResult struct {
	List  []deliveryExtractSourceItem `json:"list"`
	Total int                         `json:"total"`
}

type deliveryExtractImportRequest struct {
	SourceName string `json:"sourceName"`
	Content    string `json:"content"`
}

type productCollectionImportRequest struct {
	SourceName string `json:"sourceName"`
	Content    string `json:"content"`
	ShopID     int64  `json:"shopId"`
}

type extensionSellerShopSyncRequest struct {
	SourceName string `json:"sourceName"`
	Content    string `json:"content"`
}

type extensionSellerShopSyncResponse struct {
	Shops []models.Shop `json:"shops"`
}

type sellerShopCandidate struct {
	ExternalCode string
	ShopName     string
}

type updateProductCollectionProductRequest struct {
	ProductConfig string `json:"productConfig"`
	CostPriceCent int    `json:"costPrice"`
}

type batchUpdateProductCollectionMaintenanceRequest struct {
	Content string `json:"content"`
}

type salesOverallImportRequest struct {
	SourceName string `json:"sourceName"`
	Content    string `json:"content"`
}

type deliveryExtractSourceItem struct {
	DeliveryOrderSn       string                       `json:"deliveryOrderSn"`
	ExpressBatchSn        string                       `json:"expressBatchSn"`
	PlatExpressStatusTip  string                       `json:"platExpressStatusTip"`
	CanCancelExpress      *bool                        `json:"canCancelExpress"`
	ExpectPickUpGoodsTime int64                        `json:"expectPickUpGoodsTime"`
	SupplierID            int64                        `json:"supplierId"`
	ProductSkcID          int64                        `json:"productSkcId"`
	DeliverSkcNum         int                          `json:"deliverSkcNum"`
	SkcPurchaseNum        int                          `json:"skcPurchaseNum"`
	SubPurchaseOrderBasic deliveryExtractProductBasic  `json:"subPurchaseOrderBasicVO"`
	ReceiveAddressInfo    deliveryExtractReceiveInfo   `json:"receiveAddressInfo"`
	PackageList           []deliveryExtractPackage     `json:"packageList"`
	PackageDetailList     []deliveryExtractPackageItem `json:"packageDetailList"`
	DeliveryOrderList     []deliveryExtractSourceItem  `json:"deliveryOrderList"`
}

type deliveryExtractProductBasic struct {
	ProductName       string `json:"productName"`
	ProductSkcPicture string `json:"productSkcPicture"`
	ProductSkcID      int64  `json:"productSkcId"`
	SupplierID        int64  `json:"supplierId"`
	PurchaseQuantity  int    `json:"purchaseQuantity"`
}

type deliveryExtractReceiveInfo struct {
	ReceiverName string `json:"receiverName"`
}

type deliveryExtractPackage struct {
	SkcNum int `json:"skcNum"`
}

type deliveryExtractPackageItem struct {
	ProductSkuID int64 `json:"productSkuId"`
	SkuNum       int   `json:"skuNum"`
}

type salesOverallSource struct {
	Items []salesOverallProduct
	Total int
}

type salesOverallSourceEnvelope struct {
	List         []json.RawMessage        `json:"list"`
	Data         []json.RawMessage        `json:"data"`
	SubOrderList []json.RawMessage        `json:"subOrderList"`
	Result       salesOverallSourceResult `json:"result"`
	Total        int                      `json:"total"`
}

type salesOverallSourceResult struct {
	SubOrderList []json.RawMessage `json:"subOrderList"`
	List         []json.RawMessage `json:"list"`
	Data         []json.RawMessage `json:"data"`
	Total        int               `json:"total"`
}

type salesOverallProduct struct {
	ProductName           string                 `json:"productName"`
	Category              string                 `json:"category"`
	ProductSkcID          int64                  `json:"productSkcId"`
	ProductSkcPicture     string                 `json:"productSkcPicture"`
	SupplierID            int64                  `json:"supplierId"`
	SupplierName          string                 `json:"supplierName"`
	SkuQuantityDetailList []salesOverallSku      `json:"skuQuantityDetailList"`
	SkuQuantityTotalInfo  salesOverallSkuSummary `json:"skuQuantityTotalInfo"`
	Raw                   json.RawMessage        `json:"-"`
}

type salesOverallSku struct {
	ProductSkuID                   int64                        `json:"productSkuId"`
	ClassName                      string                       `json:"className"`
	CurrencyType                   string                       `json:"currencyType"`
	SupplierPrice                  int                          `json:"supplierPrice"`
	IsVerifyPrice                  bool                         `json:"isVerifyPrice"`
	PriceReviewStatus              int                          `json:"priceReviewStatus"`
	LackQuantity                   *int                         `json:"lackQuantity"`
	InCardNumber                   int                          `json:"inCardNumber"`
	NoMessageSubscribeArrivalCount int                          `json:"nomsgSubsCntCntSth"`
	InCartNumber7d                 int                          `json:"inCartNumber7d"`
	IsSubscribeArrivalRemind       bool                         `json:"isSubscribeArrivalRemind"`
	TodaySaleVolume                int                          `json:"todaySaleVolume"`
	LastSevenDaysSaleVolume        int                          `json:"lastSevenDaysSaleVolume"`
	LastThirtyDaysSaleVolume       int                          `json:"lastThirtyDaysSaleVolume"`
	TotalSaleVolume                int                          `json:"totalSaleVolume"`
	InventoryNumInfo               salesOverallInventoryNumInfo `json:"inventoryNumInfo"`
	WarehouseAvailableSaleDays     *float64                     `json:"warehouseAvailableSaleDays"`
	AvailableSaleDays              *float64                     `json:"availableSaleDays"`
	AvailableSaleDaysFromInventory *float64                     `json:"availableSaleDaysFromInventory"`
	PurchaseConfig                 string                       `json:"purchaseConfig"`
	SellerWarehouseStock           int                          `json:"sellerWhStock"`
	TargetProduceDays              *float64                     `json:"targetProduceDays"`
	TargetProduceNum               *int                         `json:"targetProduceNum"`
	AdviceProduceNum               *int                         `json:"adviceProduceNum"`
	AdviceQuantity                 *int                         `json:"adviceQuantity"`
	PredictSaleAdviceQuantity      *int                         `json:"predictSaleAdviceQuantity"`
	ShowStockGuide                 bool                         `json:"showStockGuide"`
	Raw                            json.RawMessage              `json:"-"`
}

type salesOverallSkuSummary struct {
	LackQuantity             int `json:"lackQuantity"`
	AdviceQuantity           int `json:"adviceQuantity"`
	TodaySaleVolume          int `json:"todaySaleVolume"`
	LastSevenDaysSaleVolume  int `json:"lastSevenDaysSaleVolume"`
	LastThirtyDaysSaleVolume int `json:"lastThirtyDaysSaleVolume"`
	TotalSaleVolume          int `json:"totalSaleVolume"`
}

type salesOverallInventoryNumInfo struct {
	WarehouseInventoryNum            int `json:"warehouseInventoryNum"`
	ExpectedOccupiedInventoryNum     int `json:"expectedOccupiedInventoryNum"`
	UnavailableWarehouseInventoryNum int `json:"unavailableWarehouseInventoryNum"`
	WaitDeliveryInventoryNum         int `json:"waitDeliveryInventoryNum"`
	WaitReceiveNum                   int `json:"waitReceiveNum"`
	WaitApproveInventoryNum          int `json:"waitApproveInventoryNum"`
}

type productCollectionSource struct {
	Items []productCollectionSourceEntry
	Total int
}

type productCollectionSourceEntry struct {
	Item productCollectionSourceItem
	Raw  json.RawMessage
}

type productCollectionSourceEnvelope struct {
	List   []json.RawMessage             `json:"list"`
	Data   []json.RawMessage             `json:"data"`
	Result productCollectionSourceResult `json:"result"`
	Total  int                           `json:"total"`
}

type productCollectionSourceResult struct {
	PageItems []json.RawMessage `json:"pageItems"`
	List      []json.RawMessage `json:"list"`
	Data      []json.RawMessage `json:"data"`
	Total     int               `json:"total"`
}

type productCollectionSourceItem struct {
	ProductSkcID        int64                         `json:"productSkcId"`
	ProductSkuID        int64                         `json:"productSkuId"`
	MainImageURL        string                        `json:"mainImageUrl"`
	ProductName         string                        `json:"productName"`
	NumberOfPiecesNew   int                           `json:"numberOfPiecesNew"`
	SupplierPrice       int                           `json:"supplierPrice"`
	SkcTopStatus        int                           `json:"skcTopStatus"`
	CreatedAt           int64                         `json:"createdAt"`
	SupplierID          int64                         `json:"supplierId"`
	ProductSkuSummaries []productCollectionSkuSummary `json:"productSkuSummaries"`
}

type productCollectionSkuSummary struct {
	ProductSkuID        int64                         `json:"productSkuId"`
	SupplierPrice       int                           `json:"supplierPrice"`
	ProductSkuMultiPack productCollectionSkuMultiPack `json:"productSkuMultiPack"`
}

type productCollectionSkuMultiPack struct {
	NumberOfPiecesNew int `json:"numberOfPiecesNew"`
}

type authedHandler func(http.ResponseWriter, *http.Request, models.User)

type appServer struct {
	cfg   config.Config
	store *store.Store
}

func main() {
	cfg := config.Load()

	db, err := openDatabaseWithRetry(cfg, 3, 15*time.Second, 2*time.Second)
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}
	defer db.Close()

	app := appServer{
		cfg:   cfg,
		store: store.New(db),
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("temu-tools api listening on http://localhost:%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func openDatabaseWithRetry(cfg config.Config, attempts int, timeout time.Duration, delay time.Duration) (*sql.DB, error) {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		db, err := database.Open(ctx, cfg)
		cancel()
		if err == nil {
			return db, nil
		}

		lastErr = err
		if attempt < attempts {
			log.Printf("mysql connect attempt %d/%d failed: %v; retrying in %s", attempt, attempts, err, delay)
			time.Sleep(delay)
		}
	}

	return nil, lastErr
}

func (app appServer) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", app.handleHealth)
	mux.HandleFunc("POST /api/auth/login", app.handleLogin)

	mux.HandleFunc("GET /api/tool-packages", app.requirePermission("tools:view", app.handleToolPackages))
	mux.HandleFunc("GET /api/tool-packages/{id}/export", app.requirePermission("tools:manage", app.handleExportToolPackage))
	mux.HandleFunc("GET /api/extension/archive", app.requirePermission("tools:view", app.handleExportExtensionArchive))
	mux.HandleFunc("GET /api/image-cache", app.handleCachedImage)
	mux.HandleFunc("GET /api/modules", app.requirePermission("tools:view", app.handleModules))
	mux.HandleFunc("POST /api/modules", app.requirePermission("tools:manage", app.handleUpsertModule))
	mux.HandleFunc("PUT /api/modules/{id}", app.requirePermission("tools:manage", app.handleUpdateModule))
	mux.HandleFunc("DELETE /api/modules/{id}", app.requirePermission("tools:manage", app.handleDeleteModule))
	mux.HandleFunc("GET /api/dashboard/sales-overall", app.requirePermission("dashboard:view", app.handleSalesOverallDashboard))
	mux.HandleFunc("POST /api/dashboard/sales-overall/import-json", app.requirePermission("tools:manage", app.handleImportSalesOverallJSON))
	mux.HandleFunc("GET /api/tools/delivery-extractions", app.requirePermission("tools:view", app.handleDeliveryExtractBatches))
	mux.HandleFunc("GET /api/tools/delivery-extractions/latest/export", app.requirePermission("tools:view", app.handleExportLatestDeliveryExtractBatch))
	mux.HandleFunc("GET /api/tools/delivery-extractions/latest", app.requirePermission("tools:view", app.handleLatestDeliveryExtractBatch))
	mux.HandleFunc("POST /api/tools/delivery-extractions/import-source", app.requirePermission("tools:manage", app.handleImportDeliveryExtractSource))
	mux.HandleFunc("POST /api/tools/delivery-extractions/import-json", app.requirePermission("tools:manage", app.handleImportDeliveryExtractJSON))
	mux.HandleFunc("GET /api/tools/product-collection/products", app.requirePermission("tools:view", app.handleProductCollectionProducts))
	mux.HandleFunc("GET /api/tools/product-collection/products/export", app.requirePermission("tools:view", app.handleExportProductCollectionProducts))
	mux.HandleFunc("POST /api/tools/product-collection/import-json", app.requirePermission("tools:manage", app.handleImportProductCollectionJSON))
	mux.HandleFunc("POST /api/tools/product-collection/products/batch-maintenance", app.requirePermission("tools:manage", app.handleBatchUpdateProductCollectionMaintenance))
	mux.HandleFunc("PUT /api/tools/product-collection/products/{id}", app.requirePermission("tools:manage", app.handleUpdateProductCollectionProduct))
	mux.HandleFunc("POST /api/extension/seller-shops/sync", app.authenticated(app.handleSyncExtensionSellerShops))
	mux.HandleFunc("GET /api/me", app.authenticated(app.handleMe))
	mux.HandleFunc("GET /api/tenant/summary", app.requirePermission("dashboard:view", app.handleTenantSummary))
	mux.HandleFunc("GET /api/settings", app.requirePermission("settings:view", app.handleSettings))
	mux.HandleFunc("PUT /api/settings", app.requirePermission("settings:update", app.handleUpdateSettings))
	mux.HandleFunc("GET /api/permissions", app.requirePermission("permissions:view", app.handlePermissions))
	mux.HandleFunc("GET /api/roles/{role}/permissions", app.requirePermission("permissions:view", app.handleRolePermissions))
	mux.HandleFunc("PUT /api/roles/{role}/permissions", app.requirePermission("permissions:manage", app.handleUpdateRolePermissions))

	mux.HandleFunc("GET /api/shops", app.requirePermission("shops:view", app.handleShops))
	mux.HandleFunc("POST /api/shops", app.requirePermission("shops:create", app.handleCreateShop))
	mux.HandleFunc("PUT /api/shops/{id}", app.requirePermission("shops:update", app.handleUpdateShop))
	mux.HandleFunc("DELETE /api/shops/{id}", app.requirePermission("shops:delete", app.handleCloseShop))

	mux.HandleFunc("GET /api/users", app.requirePermission("users:view", app.handleUsers))
	mux.HandleFunc("POST /api/users", app.requirePermission("users:create", app.handleCreateUser))
	mux.HandleFunc("PUT /api/users/{id}", app.requirePermission("users:update", app.handleUpdateUser))
	mux.HandleFunc("DELETE /api/users/{id}", app.requirePermission("users:disable", app.handleDisableUser))
	mux.HandleFunc("GET /api/users/{id}/shops", app.requirePermission("users:assign_shops", app.handleUserShops))
	mux.HandleFunc("POST /api/users/{id}/shops", app.requirePermission("users:assign_shops", app.handleAssignShop))
	mux.HandleFunc("DELETE /api/users/{id}/shops/{shopID}", app.requirePermission("users:assign_shops", app.handleRemoveShopAssignment))

	return corsMiddleware(mux)
}

func (app appServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	if request.Username == "" || request.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, expectedHash, err := app.store.GetUserByUsername(r.Context(), request.Username)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	actualHash := auth.HashPassword(request.Username, request.Password)
	if subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.SignToken(app.cfg.AppSecret, user.ID, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to sign token")
		return
	}

	permissions, err := app.store.ListPermissionsForUser(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load permissions")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Data: loginResponse{
			User:        user,
			Token:       token,
			Permissions: permissions,
		},
	})
}

func (app appServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	databaseStatus := "ok"
	if err := app.store.Ping(r.Context()); err != nil {
		databaseStatus = "error"
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Data: healthResponse{
			Status:    "ok",
			Service:   "temu-tools-api",
			Database:  databaseStatus,
			Timestamp: time.Now().UTC(),
		},
	})
}

func (app appServer) handleModules(w http.ResponseWriter, r *http.Request, _ models.User) {
	modules, err := app.store.ListToolModules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load modules")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: modules})
}

func (app appServer) handleToolPackages(w http.ResponseWriter, r *http.Request, _ models.User) {
	packages, err := app.store.ListToolPackages(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tool packages")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: packages})
}

func (app appServer) handleSalesOverallDashboard(w http.ResponseWriter, r *http.Request, user models.User) {
	dashboard, err := app.store.SalesDashboard(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load sales dashboard")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: dashboard})
}

func (app appServer) handleImportSalesOverallJSON(w http.ResponseWriter, r *http.Request, user models.User) {
	var request salesOverallImportRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.SourceName = strings.TrimSpace(request.SourceName)
	if request.SourceName == "" {
		request.SourceName = "listOverall.json"
	}
	if strings.TrimSpace(request.Content) == "" {
		writeError(w, http.StatusBadRequest, "json content is required")
		return
	}

	var source salesOverallSource
	if err := decodeSalesOverallSource([]byte(request.Content), &source); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rows := extractSalesOverallRows(source.Items)
	if len(rows) == 0 {
		writeError(w, http.StatusBadRequest, "no sales rows found")
		return
	}

	supplierID := rows[0].SupplierID
	supplierName := rows[0].SupplierName
	result, err := app.store.SaveSalesOverall(r.Context(), store.SaveSalesOverallParams{
		SourceName:   request.SourceName,
		SourceTotal:  source.Total,
		SupplierID:   supplierID,
		SupplierName: supplierName,
		Rows:         rows,
		CreatedBy:    user.ID,
	}, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save sales dashboard data")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: result})
}

func (app appServer) handleUpsertModule(w http.ResponseWriter, r *http.Request, _ models.User) {
	var request upsertToolModuleRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.normalize()
	if err := validateToolModuleInput(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	module, err := app.store.UpsertToolModule(r.Context(), store.UpsertToolModuleParams{
		ID:          request.ID,
		Name:        request.Name,
		Description: request.Description,
		Status:      request.Status,
		SortOrder:   request.SortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save module")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: module})
}

func (app appServer) handleUpdateModule(w http.ResponseWriter, r *http.Request, _ models.User) {
	var request upsertToolModuleRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.ID = strings.TrimSpace(r.PathValue("id"))
	request.normalize()
	if err := validateToolModuleInput(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	module, err := app.store.UpsertToolModule(r.Context(), store.UpsertToolModuleParams{
		ID:          request.ID,
		Name:        request.Name,
		Description: request.Description,
		Status:      request.Status,
		SortOrder:   request.SortOrder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update module")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: module})
}

func (app appServer) handleDeleteModule(w http.ResponseWriter, r *http.Request, _ models.User) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	if err := app.store.DeleteToolModule(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "module not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete module")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"deleted": true}})
}

func (app appServer) handleDeliveryExtractBatches(w http.ResponseWriter, r *http.Request, _ models.User) {
	batches, err := app.store.ListDeliveryExtractBatches(r.Context(), 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load delivery extract batches")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: batches})
}

func (app appServer) handleLatestDeliveryExtractBatch(w http.ResponseWriter, r *http.Request, _ models.User) {
	options := deliveryExtractRowsOptionsFromRequest(r)
	batch, err := app.store.GetLatestDeliveryExtractBatch(r.Context(), options)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, apiResponse{Data: nil})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load latest delivery extract batch")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: batch})
}

func (app appServer) handleExportLatestDeliveryExtractBatch(w http.ResponseWriter, r *http.Request, _ models.User) {
	options := deliveryExtractRowsOptionsFromRequest(r)
	options.AllRows = true

	batch, err := app.store.GetLatestDeliveryExtractBatch(r.Context(), options)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "delivery extract batch not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load delivery extract batch")
		return
	}

	workbook, err := buildDeliveryExtractWorkbook(r.Context(), batch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build export workbook")
		return
	}

	filename := deliveryExtractExportFilename(batch, time.Now())
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", deliveryExportContentDisposition(filename))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(workbook); err != nil {
		log.Printf("failed to write export workbook: %v", err)
	}
}

func deliveryExtractExportFilename(batch models.DeliveryExtractBatch, now time.Time) string {
	name := "发货提取"
	if len(batch.Rows) > 0 {
		firstRow := batch.Rows[0]
		name = strings.TrimSpace(firstRow.EuRepresentative)
		if name == "" {
			name = strings.TrimSpace(firstRow.ShopName)
		}
		if name == "" {
			name = strings.TrimSpace(firstRow.SupplierID)
		}
	}

	name = safeDeliveryExportFilenamePart(name)
	if name == "" {
		name = "发货提取"
	}

	exportDate := now
	if now.Hour() >= 16 {
		exportDate = exportDate.AddDate(0, 0, 1)
	}

	return fmt.Sprintf("%s%d-%d.xlsx", name, int(exportDate.Month()), exportDate.Day())
}

func safeDeliveryExportFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	for _, char := range value {
		if char < 32 {
			continue
		}
		switch char {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			builder.WriteRune('-')
		default:
			builder.WriteRune(char)
		}
	}

	return strings.TrimSpace(builder.String())
}

func deliveryExportContentDisposition(filename string) string {
	encodedFilename := url.PathEscape(filename)
	return fmt.Sprintf(`attachment; filename="delivery-extract.xlsx"; filename*=UTF-8''%s`, encodedFilename)
}

func (app appServer) handleImportDeliveryExtractSource(w http.ResponseWriter, r *http.Request, user models.User) {
	sourcePath, err := resolveOtherSourcePath()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read other/source.json")
		return
	}

	app.importDeliveryExtractContent(w, r, user, content, filepath.ToSlash(sourcePath))
}

func (app appServer) handleImportDeliveryExtractJSON(w http.ResponseWriter, r *http.Request, user models.User) {
	var request deliveryExtractImportRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.SourceName = strings.TrimSpace(request.SourceName)
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		writeError(w, http.StatusBadRequest, "json content is required")
		return
	}
	if request.SourceName == "" {
		request.SourceName = "manual-json"
	}

	app.importDeliveryExtractContent(w, r, user, []byte(request.Content), request.SourceName)
}

func (app appServer) importDeliveryExtractContent(w http.ResponseWriter, r *http.Request, user models.User, content []byte, sourceName string) {
	var source deliveryExtractSource
	if err := decodeDeliveryExtractSource(content, &source); err != nil {
		writeError(w, http.StatusBadRequest, "json content is not valid delivery data")
		return
	}
	if len(source.List) == 0 {
		writeError(w, http.StatusBadRequest, "json content does not contain list data")
		return
	}

	rows, batchDate := extractDeliveryRows(source.List)
	if len(rows) == 0 {
		writeError(w, http.StatusBadRequest, "no delivery rows can be extracted")
		return
	}

	sourceTotal := source.Total
	if sourceTotal == 0 {
		sourceTotal = len(source.List)
	}

	batch, err := app.store.SaveDeliveryExtract(r.Context(), store.SaveDeliveryExtractParams{
		SourceFile:  sourceName,
		BatchDate:   batchDate,
		SourceTotal: sourceTotal,
		Rows:        rows,
		CreatedBy:   user.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save extracted delivery rows")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: batch})
}

func (app appServer) handleProductCollectionProducts(w http.ResponseWriter, r *http.Request, _ models.User) {
	products, err := app.store.ListProductCollectionProducts(r.Context(), productCollectionProductsOptionsFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load product collection products")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: products})
}

func (app appServer) handleCachedImage(w http.ResponseWriter, r *http.Request) {
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "image url is required")
		return
	}

	imageURL, err := normalizeImageURL(rawURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image url")
		return
	}

	content, mediaType, _, err := downloadCachedExportImage(r.Context(), imageURL, 15*time.Second)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to load image")
		return
	}

	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		log.Printf("failed to write cached image: %v", err)
	}
}

func (app appServer) handleExportProductCollectionProducts(w http.ResponseWriter, r *http.Request, _ models.User) {
	options := productCollectionProductsOptionsFromRequest(r)
	options.AllRows = true

	products, err := app.store.ListProductCollectionProducts(r.Context(), options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load product collection products")
		return
	}

	workbook, err := buildProductCollectionWorkbook(r.Context(), products.Rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build product collection workbook")
		return
	}

	filename := fmt.Sprintf("商品采集导出-%s.xlsx", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", deliveryExportContentDisposition(filename))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(workbook); err != nil {
		log.Printf("failed to write product collection workbook: %v", err)
	}
}

func (app appServer) handleImportProductCollectionJSON(w http.ResponseWriter, r *http.Request, user models.User) {
	var request productCollectionImportRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.SourceName = strings.TrimSpace(request.SourceName)
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		writeError(w, http.StatusBadRequest, "json content is required")
		return
	}
	if request.SourceName == "" {
		request.SourceName = "manual-product-json"
	}

	app.importProductCollectionContent(w, r, user, []byte(request.Content), request.SourceName, request.ShopID)
}

func (app appServer) importProductCollectionContent(w http.ResponseWriter, r *http.Request, user models.User, content []byte, sourceName string, shopID int64) {
	if shopID <= 0 {
		writeError(w, http.StatusBadRequest, "shopId is required")
		return
	}

	shop, err := app.store.GetShop(r.Context(), shopID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "selected shop does not exist")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load selected shop")
		return
	}
	if strings.TrimSpace(shop.ExternalCode) == "" {
		writeError(w, http.StatusBadRequest, "selected shop externalCode is required")
		return
	}

	var source productCollectionSource
	if err := decodeProductCollectionSource(content, &source); err != nil {
		writeError(w, http.StatusBadRequest, "json content is not valid product collection data")
		return
	}
	if len(source.Items) == 0 {
		writeError(w, http.StatusBadRequest, "json content does not contain product data")
		return
	}

	products := extractProductCollectionProducts(source.Items, shop)
	if len(products) == 0 {
		writeError(w, http.StatusBadRequest, "no product rows can be extracted")
		return
	}
	sourceTotal := source.Total
	if sourceTotal == 0 {
		sourceTotal = len(source.Items)
	}

	result, err := app.store.SaveProductCollection(r.Context(), store.SaveProductCollectionParams{
		SourceTotal: sourceTotal,
		Products:    products,
		Shop:        shop,
		CreatedBy:   user.ID,
	}, productCollectionProductsOptionsFromRequest(r))
	if err != nil {
		log.Printf("failed to import product collection from %s: %v", sourceName, err)
		writeError(w, http.StatusInternalServerError, "failed to save product collection rows")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: result})
}

func (app appServer) handleUpdateProductCollectionProduct(w http.ResponseWriter, r *http.Request, _ models.User) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var request updateProductCollectionProductRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ProductConfig = strings.TrimSpace(request.ProductConfig)
	if request.CostPriceCent < 0 {
		writeError(w, http.StatusBadRequest, "costPrice cannot be negative")
		return
	}

	product, err := app.store.UpdateProductCollectionProduct(r.Context(), id, store.UpdateProductCollectionProductParams{
		ProductConfig: request.ProductConfig,
		CostPriceCent: request.CostPriceCent,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update product")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: product})
}

func (app appServer) handleSyncExtensionSellerShops(w http.ResponseWriter, r *http.Request, user models.User) {
	var request extensionSellerShopSyncRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		writeError(w, http.StatusBadRequest, "seller userInfo content is required")
		return
	}

	var payload any
	if err := json.Unmarshal([]byte(request.Content), &payload); err != nil {
		writeError(w, http.StatusBadRequest, "seller userInfo content is not valid json")
		return
	}

	candidates := extractSellerShopCandidates(payload)
	if len(candidates) == 0 {
		writeError(w, http.StatusBadRequest, "seller userInfo does not contain shop data")
		return
	}

	shops := make([]models.Shop, 0, len(candidates))
	for _, candidate := range candidates {
		shop, err := app.store.UpsertShopByExternalCode(r.Context(), store.UpsertShopParams{
			ShopName:         candidate.ShopName,
			Platform:         "temu",
			ExternalCode:     candidate.ExternalCode,
			EuRepresentative: "",
			Status:           "active",
			CreatedBy:        user.ID,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save seller shop")
			return
		}
		if err := app.store.AssignShop(r.Context(), user.ID, shop.ID, "operator"); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to assign seller shop")
			return
		}
		shop.ShopRole = "operator"
		shops = append(shops, shop)
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: extensionSellerShopSyncResponse{Shops: shops}})
}

func (app appServer) handleBatchUpdateProductCollectionMaintenance(w http.ResponseWriter, r *http.Request, _ models.User) {
	var request batchUpdateProductCollectionMaintenanceRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	items, err := parseProductMaintenanceContent(request.Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "batch content does not contain product maintenance rows")
		return
	}

	result, err := app.store.BatchUpdateProductCollectionMaintenance(r.Context(), store.BatchUpdateProductCollectionMaintenanceParams{
		Items: items,
	}, productCollectionProductsOptionsFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to batch update product maintenance")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

func (app appServer) handleMe(w http.ResponseWriter, r *http.Request, user models.User) {
	permissions, err := app.store.ListPermissionsForUser(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load permissions")
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: loginResponse{
		User:        user,
		Token:       "",
		Permissions: permissions,
	}})
}

func (app appServer) handleTenantSummary(w http.ResponseWriter, r *http.Request, user models.User) {
	summary, err := app.store.TenantSummary(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tenant summary")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: summary})
}

func (app appServer) handleSettings(w http.ResponseWriter, r *http.Request, _ models.User) {
	settings, err := app.store.ListSystemSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: settings})
}

func (app appServer) handleUpdateSettings(w http.ResponseWriter, r *http.Request, user models.User) {
	var request updateSettingsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.Values) == 0 {
		writeError(w, http.StatusBadRequest, "settings values are required")
		return
	}

	settings, err := app.store.UpdateSystemSettings(r.Context(), request.Values, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: settings})
}

func (app appServer) handlePermissions(w http.ResponseWriter, r *http.Request, _ models.User) {
	permissions, err := app.store.ListPermissions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load permissions")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: permissions})
}

func (app appServer) handleRolePermissions(w http.ResponseWriter, r *http.Request, _ models.User) {
	role, ok := roleFromPath(w, r)
	if !ok {
		return
	}

	permissions, err := app.store.ListRolePermissions(r.Context(), role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load role permissions")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: permissions})
}

func (app appServer) handleUpdateRolePermissions(w http.ResponseWriter, r *http.Request, _ models.User) {
	role, ok := roleFromPath(w, r)
	if !ok {
		return
	}

	var request rolePermissionsRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	if err := app.store.ReplaceRolePermissions(r.Context(), role, request.Permissions); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update role permissions")
		return
	}

	permissions, err := app.store.ListRolePermissions(r.Context(), role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load role permissions")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: permissions})
}

func (app appServer) handleShops(w http.ResponseWriter, r *http.Request, user models.User) {
	shops, err := app.store.ListVisibleShops(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load shops")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: shops})
}

func (app appServer) handleCreateShop(w http.ResponseWriter, r *http.Request, user models.User) {
	var request createShopRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.normalize()
	if err := validateShopInput(request.ShopName, request.Platform, request.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	shop, err := app.store.CreateShop(r.Context(), store.CreateShopParams{
		ShopName:         request.ShopName,
		Platform:         request.Platform,
		ExternalCode:     request.ExternalCode,
		EuRepresentative: request.EuRepresentative,
		Status:           request.Status,
		CreatedBy:        user.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create shop")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: shop})
}

func (app appServer) handleUpdateShop(w http.ResponseWriter, r *http.Request, _ models.User) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var request updateShopRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.normalize()
	if err := validateShopInput(request.ShopName, request.Platform, request.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	shop, err := app.store.UpdateShop(r.Context(), id, store.UpdateShopParams{
		ShopName:         request.ShopName,
		Platform:         request.Platform,
		ExternalCode:     request.ExternalCode,
		EuRepresentative: request.EuRepresentative,
		Status:           request.Status,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "shop not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update shop")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: shop})
}

func (app appServer) handleCloseShop(w http.ResponseWriter, r *http.Request, _ models.User) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	if err := app.store.CloseShop(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "shop not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to close shop")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"closed": true}})
}

func (app appServer) handleUsers(w http.ResponseWriter, r *http.Request, _ models.User) {
	users, err := app.store.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: users})
}

func (app appServer) handleCreateUser(w http.ResponseWriter, r *http.Request, _ models.User) {
	var request createUserRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.normalize()
	if err := validateCreateUserInput(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := app.store.CreateUser(r.Context(), store.CreateUserParams{
		Username:     request.Username,
		PasswordHash: auth.HashPassword(request.Username, request.Password),
		DisplayName:  request.DisplayName,
		Role:         request.Role,
		Status:       request.Status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: user})
}

func (app appServer) handleUpdateUser(w http.ResponseWriter, r *http.Request, _ models.User) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var request updateUserRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.normalize()
	if err := validateUpdateUserInput(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	passwordHash := ""
	if request.Password != "" {
		current, err := app.store.GetUserAnyStatus(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load user")
			return
		}
		passwordHash = auth.HashPassword(current.Username, request.Password)
	}

	user, err := app.store.UpdateUser(r.Context(), id, store.UpdateUserParams{
		PasswordHash: passwordHash,
		DisplayName:  request.DisplayName,
		Role:         request.Role,
		Status:       request.Status,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: user})
}

func (app appServer) handleDisableUser(w http.ResponseWriter, r *http.Request, actor models.User) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if id == actor.ID {
		writeError(w, http.StatusBadRequest, "cannot disable current user")
		return
	}

	if err := app.store.DisableUser(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to disable user")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"disabled": true}})
}

func (app appServer) handleUserShops(w http.ResponseWriter, r *http.Request, _ models.User) {
	userID, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	assignments, err := app.store.ListUserShops(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user shops")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: assignments})
}

func (app appServer) handleAssignShop(w http.ResponseWriter, r *http.Request, _ models.User) {
	userID, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var request assignShopRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	request.ShopRole = strings.TrimSpace(request.ShopRole)
	if request.ShopID <= 0 || !validShopRole(request.ShopRole) {
		writeError(w, http.StatusBadRequest, "valid shopId and shopRole are required")
		return
	}

	if err := app.store.AssignShop(r.Context(), userID, request.ShopID, request.ShopRole); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign shop")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"assigned": true}})
}

func (app appServer) handleRemoveShopAssignment(w http.ResponseWriter, r *http.Request, _ models.User) {
	userID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	shopID, ok := pathID(w, r, "shopID")
	if !ok {
		return
	}

	if err := app.store.RemoveShopAssignment(r.Context(), userID, shopID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove assignment")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"removed": true}})
}

func (app appServer) authenticated(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := app.currentUser(w, r)
		if !ok {
			return
		}
		next(w, r, user)
	}
}

func (app appServer) adminOnly(next authedHandler) http.HandlerFunc {
	return app.authenticated(func(w http.ResponseWriter, r *http.Request, user models.User) {
		if !user.IsAdmin() {
			writeError(w, http.StatusForbidden, "admin role required")
			return
		}
		next(w, r, user)
	})
}

func (app appServer) requirePermission(permissionCode string, next authedHandler) http.HandlerFunc {
	return app.authenticated(func(w http.ResponseWriter, r *http.Request, user models.User) {
		allowed, err := app.store.UserHasPermission(r.Context(), user, permissionCode)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check permission")
			return
		}
		if !allowed {
			writeError(w, http.StatusForbidden, "permission required: "+permissionCode)
			return
		}
		next(w, r, user)
	})
}

func (app appServer) currentUser(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	userID, err := app.userIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user identity")
		return models.User{}, false
	}

	user, err := app.store.GetUser(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "user not found")
		return models.User{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return models.User{}, false
	}

	return user, true
}

func (app appServer) userIDFromRequest(r *http.Request) (int64, error) {
	authorization := r.Header.Get("Authorization")
	if strings.HasPrefix(authorization, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
		claims, err := auth.ParseToken(app.cfg.AppSecret, token)
		if err != nil {
			return 0, err
		}
		return claims.UserID, nil
	}

	userIDText := r.Header.Get("X-User-ID")
	if userIDText == "" && app.cfg.DevAuthEnabled {
		userIDText = app.cfg.DevUserID
	}
	if userIDText == "" || !app.cfg.DevAuthEnabled {
		return 0, errors.New("missing token")
	}

	userID, err := strconv.ParseInt(userIDText, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errors.New("invalid user id")
	}
	return userID, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid path id")
		return 0, false
	}
	return id, true
}

func roleFromPath(w http.ResponseWriter, r *http.Request) (models.UserRole, bool) {
	role := models.UserRole(strings.TrimSpace(r.PathValue("role")))
	if role != models.RoleAdmin && role != models.RoleUser {
		writeError(w, http.StatusBadRequest, "invalid role")
		return "", false
	}
	return role, true
}

func deliveryExtractRowsOptionsFromRequest(r *http.Request) store.DeliveryExtractRowsOptions {
	query := r.URL.Query()
	return store.DeliveryExtractRowsOptions{
		Query:     strings.TrimSpace(query.Get("q")),
		BatchDate: normalizeDeliveryBatchDate(query.Get("batchDate")),
		RowIDs:    deliveryExtractRowIDsFromQuery(query),
		Page:      positiveQueryInt(query.Get("page"), 1, 0),
		PageSize:  positiveQueryInt(query.Get("pageSize"), defaultPageSize, maxPageSize),
	}
}

func deliveryExtractRowIDsFromQuery(query url.Values) []int64 {
	rawValues := append([]string{}, query["rowIds"]...)
	rawValues = append(rawValues, query["rowIds[]"]...)

	rowIDs := make([]int64, 0, len(rawValues))
	seen := make(map[int64]bool)
	for _, rawValue := range rawValues {
		parts := strings.FieldsFunc(rawValue, func(char rune) bool {
			return char == ',' || char == ' '
		})
		for _, part := range parts {
			rowID, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || rowID <= 0 || seen[rowID] {
				continue
			}
			seen[rowID] = true
			rowIDs = append(rowIDs, rowID)
		}
	}

	return rowIDs
}

func productCollectionProductsOptionsFromRequest(r *http.Request) store.ProductCollectionProductsOptions {
	query := r.URL.Query()
	options := store.ProductCollectionProductsOptions{
		Query:    strings.TrimSpace(query.Get("q")),
		ShopID:   int64(positiveQueryInt(query.Get("shopId"), 0, 0)),
		Page:     positiveQueryInt(query.Get("page"), 1, 0),
		PageSize: positiveQueryInt(query.Get("pageSize"), defaultPageSize, maxPageSize),
	}
	if status, ok := productStatusFromQuery(query.Get("status")); ok {
		options.Status = status
		options.HasStatus = true
	}
	return options
}

func productStatusFromQuery(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	status, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	switch status {
	case 0, 100, 200, 300:
		return status, true
	default:
		return 0, false
	}
}

func parseProductMaintenanceContent(content string) ([]store.ProductCollectionMaintenanceItem, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	records := strings.Split(content, ";")
	items := make([]store.ProductCollectionMaintenanceItem, 0, len(records))

	for index, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		item, err := parseProductMaintenanceRecord(record)
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", index+1, err)
		}
		items = append(items, item)
	}

	return items, nil
}

func parseProductMaintenanceRecord(record string) (store.ProductCollectionMaintenanceItem, error) {
	firstComma := strings.Index(record, ",")
	if firstComma < 0 {
		return store.ProductCollectionMaintenanceItem{}, errors.New("missing comma after skc")
	}

	skc := strings.TrimSpace(record[:firstComma])
	if skc == "" {
		return store.ProductCollectionMaintenanceItem{}, errors.New("skc is required")
	}

	tail := strings.TrimLeft(record[firstComma+1:], " \t\n")
	costText := ""
	configText := ""
	secondComma := strings.Index(tail, ",")
	firstNewline := strings.Index(tail, "\n")
	if secondComma >= 0 && (firstNewline < 0 || secondComma < firstNewline) {
		costText = tail[:secondComma]
		configText = tail[secondComma+1:]
	} else if firstNewline >= 0 {
		costText = tail[:firstNewline]
		configText = tail[firstNewline+1:]
	} else {
		costText = tail
	}

	costText = strings.TrimSpace(costText)
	if costText == "" {
		return store.ProductCollectionMaintenanceItem{}, errors.New("cost is required")
	}
	cost, err := strconv.ParseFloat(costText, 64)
	if err != nil || cost < 0 {
		return store.ProductCollectionMaintenanceItem{}, errors.New("cost must be a non-negative number")
	}

	return store.ProductCollectionMaintenanceItem{
		ProductSkcID:  skc,
		CostPriceCent: int(math.Round(cost * 100)),
		ProductConfig: strings.TrimSpace(configText),
	}, nil
}

func extractSellerShopCandidates(payload any) []sellerShopCandidate {
	candidatesByExternalCode := make(map[string]sellerShopCandidate)
	collectSellerShopCandidates(payload, candidatesByExternalCode)

	candidates := make([]sellerShopCandidate, 0, len(candidatesByExternalCode))
	for _, candidate := range candidatesByExternalCode {
		if candidate.ExternalCode == "" {
			continue
		}
		if candidate.ShopName == "" {
			candidate.ShopName = "Temu Shop " + candidate.ExternalCode
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func collectSellerShopCandidates(value any, candidates map[string]sellerShopCandidate) {
	switch typed := value.(type) {
	case map[string]any:
		candidate := sellerShopCandidate{
			ExternalCode: firstJSONTextByKeys(typed, "mallId", "mallID", "mallid", "mall_id", "mallIdStr", "supplierId", "sellerId", "merchantId"),
			ShopName:     firstJSONTextByKeys(typed, "mallName", "mall_name", "mallDisplayName", "shopName", "shop_name", "storeName", "store_name", "sellerName", "merchantName", "companyName"),
		}
		if candidate.ExternalCode != "" {
			if existing, ok := candidates[candidate.ExternalCode]; ok {
				candidate.ShopName = firstNonEmptyString(candidate.ShopName, existing.ShopName)
			}
			candidates[candidate.ExternalCode] = candidate
		}
		for _, child := range typed {
			collectSellerShopCandidates(child, candidates)
		}
	case []any:
		for _, child := range typed {
			collectSellerShopCandidates(child, candidates)
		}
	}
}

func firstJSONTextByKeys(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if text := jsonScalarToString(value); text != "" {
				return text
			}
		}
	}
	return ""
}

func jsonScalarToString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == math.Trunc(typed) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func positiveQueryInt(raw string, fallback int, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		value = fallback
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{Error: message})
}

func (request *createUserRequest) normalize() {
	request.Username = strings.TrimSpace(request.Username)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	if request.Role == "" {
		request.Role = models.RoleUser
	}
	if request.Status == "" {
		request.Status = "active"
	}
}

func (request *updateUserRequest) normalize() {
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	if request.Role == "" {
		request.Role = models.RoleUser
	}
	if request.Status == "" {
		request.Status = "active"
	}
}

func (request *createShopRequest) normalize() {
	request.ShopName = strings.TrimSpace(request.ShopName)
	request.Platform = strings.TrimSpace(request.Platform)
	request.ExternalCode = strings.TrimSpace(request.ExternalCode)
	request.EuRepresentative = strings.TrimSpace(request.EuRepresentative)
	if request.Platform == "" {
		request.Platform = "temu"
	}
	if request.Status == "" {
		request.Status = "active"
	}
}

func (request *updateShopRequest) normalize() {
	request.ShopName = strings.TrimSpace(request.ShopName)
	request.Platform = strings.TrimSpace(request.Platform)
	request.ExternalCode = strings.TrimSpace(request.ExternalCode)
	request.EuRepresentative = strings.TrimSpace(request.EuRepresentative)
	if request.Platform == "" {
		request.Platform = "temu"
	}
	if request.Status == "" {
		request.Status = "active"
	}
}

func (request *upsertToolModuleRequest) normalize() {
	request.ID = strings.TrimSpace(request.ID)
	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)
	if request.Status == "" {
		request.Status = "planning"
	}
	if request.SortOrder == 0 {
		request.SortOrder = 100
	}
}

func validateCreateUserInput(request createUserRequest) error {
	if request.Username == "" || request.Password == "" || request.DisplayName == "" {
		return errors.New("username, password and displayName are required")
	}
	if len(request.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return validateUserRoleAndStatus(request.Role, request.Status)
}

func validateUpdateUserInput(request updateUserRequest) error {
	if request.DisplayName == "" {
		return errors.New("displayName is required")
	}
	if request.Password != "" && len(request.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return validateUserRoleAndStatus(request.Role, request.Status)
}

func validateUserRoleAndStatus(role models.UserRole, status string) error {
	if role != models.RoleAdmin && role != models.RoleUser {
		return errors.New("role must be admin or user")
	}
	if status != "active" && status != "disabled" {
		return errors.New("status must be active or disabled")
	}
	return nil
}

func validateShopInput(shopName string, platform string, status string) error {
	if shopName == "" || platform == "" {
		return errors.New("shopName and platform are required")
	}
	if status != "active" && status != "paused" && status != "closed" {
		return errors.New("status must be active, paused or closed")
	}
	return nil
}

func validateToolModuleInput(request upsertToolModuleRequest) error {
	if request.ID == "" || request.Name == "" {
		return errors.New("id and name are required")
	}
	if request.Status != "planning" && request.Status != "active" && request.Status != "paused" {
		return errors.New("status must be planning, active or paused")
	}
	return nil
}

func validShopRole(role string) bool {
	return role == "owner" || role == "operator" || role == "viewer"
}

func resolveOtherSourcePath() (string, error) {
	candidates := []string{
		filepath.Join("..", "other", "source.json"),
		filepath.Join("other", "source.json"),
		filepath.Join("..", "..", "other", "source.json"),
	}
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absolute); err == nil {
			return absolute, nil
		}
	}
	return "", fmt.Errorf("cannot find other/source.json")
}

func decodeDeliveryExtractSource(content []byte, source *deliveryExtractSource) error {
	if err := json.Unmarshal(content, source); err == nil && len(source.List) > 0 {
		return nil
	}
	if len(source.Result.List) > 0 {
		source.List = source.Result.List
		source.Total = source.Result.Total
		return nil
	}

	var list []deliveryExtractSourceItem
	if err := json.Unmarshal(content, &list); err != nil {
		return err
	}
	source.List = list
	source.Total = len(list)
	return nil
}

func decodeProductCollectionSource(content []byte, source *productCollectionSource) error {
	var rawList []json.RawMessage
	if err := json.Unmarshal(content, &rawList); err == nil && len(rawList) > 0 {
		return decodeProductCollectionRawList(rawList, len(rawList), source)
	}

	var envelope productCollectionSourceEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return err
	}
	rawList = envelope.List
	if len(rawList) == 0 {
		rawList = envelope.Data
	}
	total := envelope.Total
	if len(rawList) == 0 {
		rawList = envelope.Result.PageItems
		total = envelope.Result.Total
	}
	if len(rawList) == 0 {
		rawList = envelope.Result.List
	}
	if len(rawList) == 0 {
		rawList = envelope.Result.Data
	}
	return decodeProductCollectionRawList(rawList, total, source)
}

func decodeProductCollectionRawList(rawList []json.RawMessage, total int, source *productCollectionSource) error {
	source.Items = make([]productCollectionSourceEntry, 0, len(rawList))
	source.Total = total
	for _, raw := range rawList {
		var item productCollectionSourceItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		source.Items = append(source.Items, productCollectionSourceEntry{Item: item, Raw: raw})
	}
	if source.Total == 0 {
		source.Total = len(source.Items)
	}
	return nil
}

func decodeSalesOverallSource(content []byte, source *salesOverallSource) error {
	var rawList []json.RawMessage
	if err := json.Unmarshal(content, &rawList); err == nil && len(rawList) > 0 {
		return decodeSalesOverallRawList(rawList, len(rawList), source)
	}

	var envelope salesOverallSourceEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return err
	}

	rawList = envelope.SubOrderList
	total := envelope.Total
	if len(rawList) == 0 {
		rawList = envelope.List
	}
	if len(rawList) == 0 {
		rawList = envelope.Data
	}
	if len(rawList) == 0 {
		rawList = envelope.Result.SubOrderList
		total = envelope.Result.Total
	}
	if len(rawList) == 0 {
		rawList = envelope.Result.List
	}
	if len(rawList) == 0 {
		rawList = envelope.Result.Data
	}
	if len(rawList) == 0 {
		return errors.New("no subOrderList found")
	}
	return decodeSalesOverallRawList(rawList, total, source)
}

func decodeSalesOverallRawList(rawList []json.RawMessage, total int, source *salesOverallSource) error {
	source.Items = make([]salesOverallProduct, 0, len(rawList))
	source.Total = total
	for _, raw := range rawList {
		var item salesOverallProduct
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		item.Raw = raw
		source.Items = append(source.Items, item)
	}
	if source.Total == 0 {
		source.Total = len(source.Items)
	}
	return nil
}

func extractDeliveryRows(items []deliveryExtractSourceItem) ([]models.DeliveryExtractRow, string) {
	items = flattenDeliveryItems(items)
	rows := make([]models.DeliveryExtractRow, 0, len(items))
	batchDate := ""

	for _, item := range items {
		if !isImportableDeliveryItem(item) {
			continue
		}
		if batchDate == "" {
			batchDate = deliveryBatchDate(item)
		}
		supplierID := item.SupplierID
		if supplierID == 0 {
			supplierID = item.SubPurchaseOrderBasic.SupplierID
		}
		skc := item.ProductSkcID
		if skc == 0 {
			skc = item.SubPurchaseOrderBasic.ProductSkcID
		}
		skcNum := firstPositiveInt(item.DeliverSkcNum, item.SkcPurchaseNum, item.SubPurchaseOrderBasic.PurchaseQuantity)
		if len(item.PackageList) > 0 {
			skcNum = firstPositiveInt(item.PackageList[0].SkcNum, skcNum)
		}

		if len(item.PackageDetailList) == 0 {
			rows = append(rows, models.DeliveryExtractRow{
				SupplierID:            int64ToString(supplierID),
				ProductName:           item.SubPurchaseOrderBasic.ProductName,
				ProductSkcPicture:     item.SubPurchaseOrderBasic.ProductSkcPicture,
				DeliveryOrderSn:       item.DeliveryOrderSn,
				ExpressBatchSn:        item.ExpressBatchSn,
				ExpectPickUpGoodsTime: item.ExpectPickUpGoodsTime,
				SKC:                   int64ToString(skc),
				SkcNum:                skcNum,
				ReceiverName:          item.ReceiveAddressInfo.ReceiverName,
			})
			continue
		}

		for _, detail := range item.PackageDetailList {
			rows = append(rows, models.DeliveryExtractRow{
				SupplierID:            int64ToString(supplierID),
				ProductName:           item.SubPurchaseOrderBasic.ProductName,
				ProductSkcPicture:     item.SubPurchaseOrderBasic.ProductSkcPicture,
				DeliveryOrderSn:       item.DeliveryOrderSn,
				ExpressBatchSn:        item.ExpressBatchSn,
				ExpectPickUpGoodsTime: item.ExpectPickUpGoodsTime,
				SKC:                   int64ToString(skc),
				SkcNum:                skcNum,
				SKU:                   int64ToString(detail.ProductSkuID),
				SkuNum:                firstPositiveInt(detail.SkuNum, skcNum),
				ReceiverName:          item.ReceiveAddressInfo.ReceiverName,
			})
		}
	}

	return rows, batchDate
}

func isImportableDeliveryItem(item deliveryExtractSourceItem) bool {
	if strings.TrimSpace(item.PlatExpressStatusTip) != "待快递揽收" {
		return false
	}
	return item.CanCancelExpress == nil || *item.CanCancelExpress
}

func flattenDeliveryItems(items []deliveryExtractSourceItem) []deliveryExtractSourceItem {
	flattened := make([]deliveryExtractSourceItem, 0, len(items))
	for _, item := range items {
		if len(item.DeliveryOrderList) == 0 {
			flattened = append(flattened, item)
			continue
		}

		for _, child := range item.DeliveryOrderList {
			child.ExpressBatchSn = firstNonEmptyString(child.ExpressBatchSn, item.ExpressBatchSn)
			child.DeliveryOrderSn = firstNonEmptyString(child.DeliveryOrderSn, item.DeliveryOrderSn)
			child.PlatExpressStatusTip = firstNonEmptyString(child.PlatExpressStatusTip, item.PlatExpressStatusTip)
			if child.CanCancelExpress == nil {
				child.CanCancelExpress = item.CanCancelExpress
			}
			child.ExpectPickUpGoodsTime = firstPositiveInt64(child.ExpectPickUpGoodsTime, item.ExpectPickUpGoodsTime)
			child.SupplierID = firstPositiveInt64(child.SupplierID, item.SupplierID)
			if strings.TrimSpace(child.ReceiveAddressInfo.ReceiverName) == "" {
				child.ReceiveAddressInfo = item.ReceiveAddressInfo
			}
			flattened = append(flattened, flattenDeliveryItems([]deliveryExtractSourceItem{child})...)
		}
	}
	return flattened
}

func extractSalesOverallRows(items []salesOverallProduct) []models.SalesOverallRow {
	rows := make([]models.SalesOverallRow, 0, len(items))
	for _, item := range items {
		if item.ProductSkcID == 0 {
			continue
		}
		for _, sku := range item.SkuQuantityDetailList {
			if sku.ProductSkuID == 0 {
				continue
			}
			rowRaw, _ := json.Marshal(struct {
				Product salesOverallProduct `json:"product"`
				SKU     salesOverallSku     `json:"sku"`
			}{Product: item, SKU: sku})
			lackQuantity := item.SkuQuantityTotalInfo.LackQuantity
			if sku.LackQuantity != nil {
				lackQuantity = *sku.LackQuantity
			}
			adviceQuantity := firstPositiveInt(item.SkuQuantityTotalInfo.AdviceQuantity, optionalIntValue(sku.AdviceQuantity), optionalIntValue(sku.PredictSaleAdviceQuantity))
			subscribeCount := sku.NoMessageSubscribeArrivalCount
			if subscribeCount == 0 && sku.IsSubscribeArrivalRemind {
				subscribeCount = 1
			}

			rows = append(rows, models.SalesOverallRow{
				SupplierID:                       int64ToString(item.SupplierID),
				SupplierName:                     strings.TrimSpace(item.SupplierName),
				ProductSkcID:                     int64ToString(item.ProductSkcID),
				ProductSkuID:                     int64ToString(sku.ProductSkuID),
				ProductName:                      strings.TrimSpace(item.ProductName),
				ProductImage:                     strings.TrimSpace(item.ProductSkcPicture),
				Category:                         strings.TrimSpace(item.Category),
				SkuClassName:                     strings.TrimSpace(sku.ClassName),
				SupplierPriceCent:                sku.SupplierPrice,
				PriceReviewStatus:                sku.PriceReviewStatus,
				IsVerifyPrice:                    sku.IsVerifyPrice,
				LackQuantity:                     lackQuantity,
				InCartNumber7d:                   sku.InCartNumber7d,
				InCartNumberTotal:                sku.InCardNumber,
				SubscribeArrivalRemindCount:      subscribeCount,
				TodaySaleVolume:                  sku.TodaySaleVolume,
				LastSevenDaysSaleVolume:          sku.LastSevenDaysSaleVolume,
				LastThirtyDaysSaleVolume:         sku.LastThirtyDaysSaleVolume,
				TotalSaleVolume:                  sku.TotalSaleVolume,
				WarehouseInventoryNum:            sku.InventoryNumInfo.WarehouseInventoryNum,
				ExpectedOccupiedInventoryNum:     sku.InventoryNumInfo.ExpectedOccupiedInventoryNum,
				UnavailableWarehouseInventoryNum: sku.InventoryNumInfo.UnavailableWarehouseInventoryNum,
				WaitDeliveryInventoryNum:         sku.InventoryNumInfo.WaitDeliveryInventoryNum,
				WaitReceiveNum:                   sku.InventoryNumInfo.WaitReceiveNum,
				WaitApproveInventoryNum:          sku.InventoryNumInfo.WaitApproveInventoryNum,
				SellerWarehouseStock:             sku.SellerWarehouseStock,
				AdviceQuantity:                   adviceQuantity,
				AvailableSaleDays:                firstFloatPointer(sku.AvailableSaleDays, sku.AvailableSaleDaysFromInventory),
				WarehouseAvailableSaleDays:       sku.WarehouseAvailableSaleDays,
				PurchaseConfig:                   strings.TrimSpace(sku.PurchaseConfig),
				TargetProduceDays:                sku.TargetProduceDays,
				TargetProduceNum:                 sku.TargetProduceNum,
				AdviceProduceNum:                 sku.AdviceProduceNum,
				ShowStockGuide:                   sku.ShowStockGuide,
				RawJSON:                          string(rowRaw),
			})
		}
	}
	return rows
}

func extractProductCollectionProducts(entries []productCollectionSourceEntry, shop models.Shop) []models.ProductCollectionProduct {
	products := make([]models.ProductCollectionProduct, 0, len(entries))
	for _, entry := range entries {
		item := entry.Item
		if item.ProductSkcID == 0 {
			continue
		}

		productSkuID := item.ProductSkuID
		supplierPrice := item.SupplierPrice
		numberOfPiecesNew := item.NumberOfPiecesNew
		if len(item.ProductSkuSummaries) > 0 {
			sku := item.ProductSkuSummaries[0]
			productSkuID = firstPositiveInt64(sku.ProductSkuID, productSkuID)
			supplierPrice = firstPositiveInt(sku.SupplierPrice, supplierPrice)
			numberOfPiecesNew = firstPositiveInt(sku.ProductSkuMultiPack.NumberOfPiecesNew, numberOfPiecesNew)
		}

		products = append(products, models.ProductCollectionProduct{
			ProductSkcID:      int64ToString(item.ProductSkcID),
			ProductSkuID:      int64ToString(productSkuID),
			MainImageURL:      strings.TrimSpace(item.MainImageURL),
			ProductName:       strings.TrimSpace(item.ProductName),
			NumberOfPiecesNew: numberOfPiecesNew,
			SupplierPriceCent: supplierPrice,
			SkcTopStatus:      item.SkcTopStatus,
			ProductCreatedAt:  timestampToTime(item.CreatedAt),
			SupplierID:        shop.ExternalCode,
			ShopID:            shop.ID,
			ShopName:          shop.ShopName,
			RawJSON:           string(entry.Raw),
		})
	}
	return products
}

func normalizeDeliveryBatchDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.Format("060102")
	}
	if parsed, err := time.Parse("20060102", value); err == nil {
		return parsed.Format("060102")
	}
	if len(value) == 6 {
		return value
	}
	return ""
}

func deliveryDateFromOrderSn(orderSn string) string {
	orderSn = strings.TrimSpace(orderSn)
	if len(orderSn) >= 8 && strings.HasPrefix(orderSn, "FH") {
		return orderSn[2:8]
	}
	return ""
}

func deliveryBatchDate(item deliveryExtractSourceItem) string {
	if date := deliveryDateFromTimestamp(item.ExpectPickUpGoodsTime); date != "" {
		return date
	}
	return deliveryDateFromOrderSn(item.DeliveryOrderSn)
}

func deliveryDateFromTimestamp(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}

	var parsed time.Time
	if timestamp > 9999999999 {
		parsed = time.UnixMilli(timestamp)
	} else {
		parsed = time.Unix(timestamp, 0)
	}

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.Local
	}
	return parsed.In(location).Format("060102")
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func optionalIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func firstFloatPointer(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func timestampToTime(timestamp int64) *time.Time {
	if timestamp <= 0 {
		return nil
	}
	if timestamp > 9999999999 {
		parsed := time.UnixMilli(timestamp)
		return &parsed
	}
	parsed := time.Unix(timestamp, 0)
	return &parsed
}

func int64ToString(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}
