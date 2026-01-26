// Package integration contains the Integration bounded context.
// This context manages external system integrations including e-commerce platforms.
//
// Key concepts:
//   - EcommercePlatform: Port interface for connecting to e-commerce platforms (Taobao, JD, PDD, Douyin)
//   - ProductMapping: Entity mapping local products to platform products
//   - PlatformOrder: Value object representing orders from external platforms
//   - OrderSync: Interface for synchronizing orders between platforms and ERP
//
// Design Pattern: Ports & Adapters
//   - Ports (interfaces) are defined here in the domain layer
//   - Adapters (implementations) are in the infrastructure layer
package integration
