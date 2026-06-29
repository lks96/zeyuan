package models

import "time"

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Role        UserRole  `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

type Shop struct {
	ID               int64     `json:"id"`
	ShopName         string    `json:"shopName"`
	Platform         string    `json:"platform"`
	ExternalCode     string    `json:"externalCode"`
	EuRepresentative string    `json:"euRepresentative"`
	ShopURL          string    `json:"shopUrl,omitempty"`
	Status           string    `json:"status"`
	ShopRole         string    `json:"shopRole,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type UserShop struct {
	UserID    int64     `json:"userId"`
	ShopID    int64     `json:"shopId"`
	ShopName  string    `json:"shopName"`
	ShopRole  string    `json:"shopRole"`
	CreatedAt time.Time `json:"createdAt"`
}

type Permission struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ToolModule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ToolPackage struct {
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Category     string    `json:"category"`
	Icon         string    `json:"icon"`
	Status       string    `json:"status"`
	PackageType  string    `json:"packageType"`
	EntryType    string    `json:"entryType"`
	EntryPath    string    `json:"entryPath"`
	PanelKey     string    `json:"panelKey"`
	Removable    bool      `json:"removable"`
	Recommended  bool      `json:"recommended"`
	SortOrder    int       `json:"sortOrder"`
	Permissions  []string  `json:"permissions"`
	ManifestJSON string    `json:"manifestJson"`
	InstalledAt  time.Time `json:"installedAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type DeliveryExtractBatch struct {
	ID             int64                `json:"id"`
	SourceFile     string               `json:"sourceFile"`
	BatchDate      string               `json:"date"`
	SourceTotal    int                  `json:"sourceTotal"`
	ExtractedTotal int                  `json:"extractedTotal"`
	RowsTotal      int                  `json:"rowsTotal,omitempty"`
	Page           int                  `json:"page,omitempty"`
	PageSize       int                  `json:"pageSize,omitempty"`
	Query          string               `json:"query,omitempty"`
	Rows           []DeliveryExtractRow `json:"data,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
}

type DeliveryExtractRow struct {
	ID                int64     `json:"id"`
	BatchID           int64     `json:"batchId,omitempty"`
	SupplierID        string    `json:"supplierId"`
	ShopID            int64     `json:"shopId,omitempty"`
	ShopName          string    `json:"shopName,omitempty"`
	EuRepresentative  string    `json:"euRepresentative,omitempty"`
	ProductName       string    `json:"productName"`
	ProductSkcPicture string    `json:"productSkcPicture"`
	DeliveryOrderSn   string    `json:"deliveryOrderSn"`
	ExpressBatchSn    string    `json:"expressBatchSn"`
	SKC               string    `json:"SKC"`
	SkcNum            int       `json:"skcNum"`
	SKU               string    `json:"SKU"`
	SkuNum            int       `json:"skuNum"`
	ReceiverName      string    `json:"receiverName"`
	ProductPieces     int       `json:"productPieces,omitempty"`
	ProductConfig     string    `json:"productConfig,omitempty"`
	CreatedAt         time.Time `json:"createdAt,omitempty"`
}

type ProductCollectionProduct struct {
	ID                int64      `json:"id"`
	ProductSkcID      string     `json:"productSkcId"`
	ProductSkuID      string     `json:"productSkuId"`
	MainImageURL      string     `json:"mainImageUrl"`
	ProductName       string     `json:"productName"`
	NumberOfPiecesNew int        `json:"numberOfPiecesNew"`
	ProductConfig     string     `json:"productConfig"`
	SupplierPriceCent int        `json:"supplierPrice"`
	CostPriceCent     int        `json:"costPrice"`
	SkcTopStatus      int        `json:"skcTopStatus"`
	ProductCreatedAt  *time.Time `json:"createdAt,omitempty"`
	SupplierID        string     `json:"supplierId"`
	ShopID            int64      `json:"shopId,omitempty"`
	ShopName          string     `json:"shopName,omitempty"`
	RawJSON           string     `json:"-"`
	ImportedAt        time.Time  `json:"importedAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type ProductCollectionList struct {
	Rows      []ProductCollectionProduct `json:"data"`
	RowsTotal int                        `json:"rowsTotal"`
	Page      int                        `json:"page"`
	PageSize  int                        `json:"pageSize"`
	Query     string                     `json:"query,omitempty"`
}

type ProductCollectionImportResult struct {
	SourceTotal int                   `json:"sourceTotal"`
	Imported    int                   `json:"imported"`
	Shop        Shop                  `json:"shop"`
	Products    ProductCollectionList `json:"products"`
}

type ProductCollectionBatchUpdateResult struct {
	Total        int                   `json:"total"`
	Updated      int                   `json:"updated"`
	NotFoundSkcs []string              `json:"notFoundSkcs"`
	Products     ProductCollectionList `json:"products"`
}

type SalesOverallRow struct {
	ID                               int64     `json:"id,omitempty"`
	BatchID                          int64     `json:"batchId,omitempty"`
	SupplierID                       string    `json:"supplierId"`
	SupplierName                     string    `json:"supplierName"`
	ProductSkcID                     string    `json:"productSkcId"`
	ProductSkuID                     string    `json:"productSkuId"`
	ProductName                      string    `json:"productName"`
	ProductImage                     string    `json:"productImage"`
	Category                         string    `json:"category"`
	SkuClassName                     string    `json:"skuClassName"`
	SupplierPriceCent                int       `json:"supplierPrice"`
	CostPriceCent                    int       `json:"costPrice"`
	PriceReviewStatus                int       `json:"priceReviewStatus"`
	IsVerifyPrice                    bool      `json:"isVerifyPrice"`
	LackQuantity                     int       `json:"lackQuantity"`
	InCartNumber7d                   int       `json:"inCartNumber7d"`
	InCartNumberTotal                int       `json:"inCartNumberTotal"`
	SubscribeArrivalRemindCount      int       `json:"subscribeArrivalRemindCount"`
	TodaySaleVolume                  int       `json:"todaySaleVolume"`
	LastSevenDaysSaleVolume          int       `json:"lastSevenDaysSaleVolume"`
	LastThirtyDaysSaleVolume         int       `json:"lastThirtyDaysSaleVolume"`
	TotalSaleVolume                  int       `json:"totalSaleVolume"`
	WarehouseInventoryNum            int       `json:"warehouseInventoryNum"`
	ExpectedOccupiedInventoryNum     int       `json:"expectedOccupiedInventoryNum"`
	UnavailableWarehouseInventoryNum int       `json:"unavailableWarehouseInventoryNum"`
	WaitDeliveryInventoryNum         int       `json:"waitDeliveryInventoryNum"`
	WaitReceiveNum                   int       `json:"waitReceiveNum"`
	WaitApproveInventoryNum          int       `json:"waitApproveInventoryNum"`
	SellerWarehouseStock             int       `json:"sellerWarehouseStock"`
	AdviceQuantity                   int       `json:"adviceQuantity"`
	AvailableSaleDays                *float64  `json:"availableSaleDays,omitempty"`
	WarehouseAvailableSaleDays       *float64  `json:"warehouseAvailableSaleDays,omitempty"`
	PurchaseConfig                   string    `json:"purchaseConfig"`
	TargetProduceDays                *float64  `json:"targetProduceDays,omitempty"`
	TargetProduceNum                 *int      `json:"targetProduceNum,omitempty"`
	AdviceProduceNum                 *int      `json:"adviceProduceNum,omitempty"`
	ShowStockGuide                   bool      `json:"showStockGuide"`
	RawJSON                          string    `json:"-"`
	CreatedAt                        time.Time `json:"createdAt,omitempty"`
}

type SalesOverallBatch struct {
	ID            int64     `json:"id"`
	SourceName    string    `json:"sourceName"`
	SupplierID    string    `json:"supplierId"`
	SupplierName  string    `json:"supplierName"`
	SourceTotal   int       `json:"sourceTotal"`
	ImportedTotal int       `json:"importedTotal"`
	CreatedAt     time.Time `json:"createdAt"`
}

type SalesDashboardPeriodMetric struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	SalesVolume     int    `json:"salesVolume"`
	SalesAmountCent int64  `json:"salesAmount"`
	GrossProfitCent int64  `json:"grossProfit"`
}

type SalesInventorySummary struct {
	LackQuantity                     int `json:"lackQuantity"`
	AdviceQuantity                   int `json:"adviceQuantity"`
	WarehouseInventoryNum            int `json:"warehouseInventoryNum"`
	ExpectedOccupiedInventoryNum     int `json:"expectedOccupiedInventoryNum"`
	UnavailableWarehouseInventoryNum int `json:"unavailableWarehouseInventoryNum"`
	WaitDeliveryInventoryNum         int `json:"waitDeliveryInventoryNum"`
	WaitReceiveNum                   int `json:"waitReceiveNum"`
	WaitApproveInventoryNum          int `json:"waitApproveInventoryNum"`
	SellerWarehouseStock             int `json:"sellerWarehouseStock"`
}

type SalesTopProduct struct {
	ProductSkcID             string `json:"productSkcId"`
	ProductName              string `json:"productName"`
	ProductImage             string `json:"productImage"`
	SupplierID               string `json:"supplierId"`
	SupplierName             string `json:"supplierName"`
	LastThirtyDaysSaleVolume int    `json:"lastThirtyDaysSaleVolume"`
	LastSevenDaysSaleVolume  int    `json:"lastSevenDaysSaleVolume"`
	TodaySaleVolume          int    `json:"todaySaleVolume"`
	SalesAmountCent          int64  `json:"salesAmount"`
	GrossProfitCent          int64  `json:"grossProfit"`
}

type SalesFieldMapping struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Note  string `json:"note"`
}

type SalesDashboard struct {
	LatestBatch  *SalesOverallBatch           `json:"latestBatch,omitempty"`
	Periods      []SalesDashboardPeriodMetric `json:"periods"`
	Inventory    SalesInventorySummary        `json:"inventory"`
	TopProducts  []SalesTopProduct            `json:"topProducts"`
	FieldMapping []SalesFieldMapping          `json:"fieldMapping"`
}

type SalesOverallImportResult struct {
	Batch     SalesOverallBatch `json:"batch"`
	Dashboard SalesDashboard    `json:"dashboard"`
}

type SystemSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TenantSummary struct {
	CurrentUser     User `json:"currentUser"`
	TotalUsers      int  `json:"totalUsers"`
	TotalShops      int  `json:"totalShops"`
	VisibleShops    int  `json:"visibleShops"`
	AdminCanViewAll bool `json:"adminCanViewAll"`
}
