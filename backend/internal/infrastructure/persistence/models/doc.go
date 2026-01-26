// Package models contains GORM-specific persistence models that map to database tables.
// These models are separate from domain entities to keep the domain layer pure and free
// from ORM concerns.
//
// Key Principles:
// 1. Domain entities should be free of GORM tags and infrastructure concerns
// 2. Persistence models contain all GORM annotations and table mappings
// 3. Mappers convert between domain entities and persistence models
// 4. Repositories use persistence models for database operations
//
// Structure:
// - base.go: Base persistence models (BaseModel, TenantModel, etc.)
// - identity/: Identity context models (User, Tenant, Role)
// - catalog/: Catalog context models (Product, Category)
// - partner/: Partner context models (Customer, Supplier, Warehouse)
// - inventory/: Inventory context models (InventoryItem, StockBatch)
// - trade/: Trade context models (SalesOrder, PurchaseOrder)
// - finance/: Finance context models (AR, AP, Vouchers)
// - outbox.go: Outbox pattern model for event delivery
package models
