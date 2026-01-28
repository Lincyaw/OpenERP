package handler

import (
	"net/http"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
)

// StrategyRegistry defines the interface for listing strategies
type StrategyRegistry interface {
	ListCostStrategies() []string
	ListPricingStrategies() []string
	ListAllocationStrategies() []string
	ListBatchStrategies() []string
	ListValidationStrategies() []string
	GetDefault(strategyType strategy.StrategyType) string
	GetCostStrategy(name string) (strategy.CostCalculationStrategy, error)
	GetPricingStrategy(name string) (strategy.PricingStrategy, error)
	GetAllocationStrategy(name string) (strategy.PaymentAllocationStrategy, error)
	GetBatchStrategy(name string) (strategy.BatchManagementStrategy, error)
	GetValidationStrategy(name string) (strategy.ProductValidationStrategy, error)
}

// StrategyHandler handles strategy-related API endpoints
type StrategyHandler struct {
	BaseHandler
	registry StrategyRegistry
}

// NewStrategyHandler creates a new StrategyHandler
func NewStrategyHandler(registry StrategyRegistry) *StrategyHandler {
	return &StrategyHandler{
		registry: registry,
	}
}

// StrategyInfo represents information about a single strategy
type StrategyInfo struct {
	Name        string `json:"name" example:"fifo"`
	Type        string `json:"type" example:"batch"`
	Description string `json:"description" example:"First In First Out batch selection"`
	IsDefault   bool   `json:"is_default" example:"true"`
}

// StrategiesResponse represents the list of available strategies
type StrategiesResponse struct {
	Cost       []StrategyInfo `json:"cost"`
	Pricing    []StrategyInfo `json:"pricing"`
	Allocation []StrategyInfo `json:"allocation"`
	Batch      []StrategyInfo `json:"batch"`
	Validation []StrategyInfo `json:"validation"`
}

// ListStrategies godoc
// @ID           listSystemStrategies
// @Summary      List available strategies
// @Description  Returns all registered strategies grouped by type
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[StrategiesResponse]
// @Failure      500 {object} dto.ErrorResponse
// @Router       /system/strategies [get]
func (h *StrategyHandler) ListStrategies(c *gin.Context) {
	response := StrategiesResponse{
		Cost:       h.buildCostStrategies(),
		Pricing:    h.buildPricingStrategies(),
		Allocation: h.buildAllocationStrategies(),
		Batch:      h.buildBatchStrategies(),
		Validation: h.buildValidationStrategies(),
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(response))
}

// buildCostStrategies builds the list of cost strategies
func (h *StrategyHandler) buildCostStrategies() []StrategyInfo {
	names := h.registry.ListCostStrategies()
	defaultName := h.registry.GetDefault(strategy.StrategyTypeCost)
	result := make([]StrategyInfo, 0, len(names))

	for _, name := range names {
		info := StrategyInfo{
			Name:      name,
			Type:      string(strategy.StrategyTypeCost),
			IsDefault: name == defaultName,
		}
		if s, err := h.registry.GetCostStrategy(name); err == nil {
			info.Description = s.Description()
		}
		result = append(result, info)
	}
	return result
}

// buildPricingStrategies builds the list of pricing strategies
func (h *StrategyHandler) buildPricingStrategies() []StrategyInfo {
	names := h.registry.ListPricingStrategies()
	defaultName := h.registry.GetDefault(strategy.StrategyTypePricing)
	result := make([]StrategyInfo, 0, len(names))

	for _, name := range names {
		info := StrategyInfo{
			Name:      name,
			Type:      string(strategy.StrategyTypePricing),
			IsDefault: name == defaultName,
		}
		if s, err := h.registry.GetPricingStrategy(name); err == nil {
			info.Description = s.Description()
		}
		result = append(result, info)
	}
	return result
}

// buildAllocationStrategies builds the list of allocation strategies
func (h *StrategyHandler) buildAllocationStrategies() []StrategyInfo {
	names := h.registry.ListAllocationStrategies()
	defaultName := h.registry.GetDefault(strategy.StrategyTypeAllocation)
	result := make([]StrategyInfo, 0, len(names))

	for _, name := range names {
		info := StrategyInfo{
			Name:      name,
			Type:      string(strategy.StrategyTypeAllocation),
			IsDefault: name == defaultName,
		}
		if s, err := h.registry.GetAllocationStrategy(name); err == nil {
			info.Description = s.Description()
		}
		result = append(result, info)
	}
	return result
}

// buildBatchStrategies builds the list of batch strategies
func (h *StrategyHandler) buildBatchStrategies() []StrategyInfo {
	names := h.registry.ListBatchStrategies()
	defaultName := h.registry.GetDefault(strategy.StrategyTypeBatch)
	result := make([]StrategyInfo, 0, len(names))

	for _, name := range names {
		info := StrategyInfo{
			Name:      name,
			Type:      string(strategy.StrategyTypeBatch),
			IsDefault: name == defaultName,
		}
		if s, err := h.registry.GetBatchStrategy(name); err == nil {
			info.Description = s.Description()
		}
		result = append(result, info)
	}
	return result
}

// buildValidationStrategies builds the list of validation strategies
func (h *StrategyHandler) buildValidationStrategies() []StrategyInfo {
	names := h.registry.ListValidationStrategies()
	defaultName := h.registry.GetDefault(strategy.StrategyTypeValidation)
	result := make([]StrategyInfo, 0, len(names))

	for _, name := range names {
		info := StrategyInfo{
			Name:      name,
			Type:      string(strategy.StrategyTypeValidation),
			IsDefault: name == defaultName,
		}
		if s, err := h.registry.GetValidationStrategy(name); err == nil {
			info.Description = s.Description()
		}
		result = append(result, info)
	}
	return result
}

// GetBatchStrategies godoc
// @ID           getSystemBatchStrategies
// @Summary      List batch management strategies
// @Description  Returns all available batch management strategies (FIFO, FEFO, etc.)
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[[]StrategyInfo]
// @Failure      500 {object} dto.ErrorResponse
// @Router       /system/strategies/batch [get]
func (h *StrategyHandler) GetBatchStrategies(c *gin.Context) {
	strategies := h.buildBatchStrategies()
	c.JSON(http.StatusOK, dto.NewSuccessResponse(strategies))
}

// GetCostStrategies godoc
// @ID           getSystemCostStrategies
// @Summary      List cost calculation strategies
// @Description  Returns all available cost calculation strategies (Moving Average, FIFO, etc.)
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[[]StrategyInfo]
// @Failure      500 {object} dto.ErrorResponse
// @Router       /system/strategies/cost [get]
func (h *StrategyHandler) GetCostStrategies(c *gin.Context) {
	strategies := h.buildCostStrategies()
	c.JSON(http.StatusOK, dto.NewSuccessResponse(strategies))
}

// GetPricingStrategies godoc
// @ID           getSystemPricingStrategies
// @Summary      List pricing strategies
// @Description  Returns all available pricing strategies (Standard, Tiered, etc.)
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[[]StrategyInfo]
// @Failure      500 {object} dto.ErrorResponse
// @Router       /system/strategies/pricing [get]
func (h *StrategyHandler) GetPricingStrategies(c *gin.Context) {
	strategies := h.buildPricingStrategies()
	c.JSON(http.StatusOK, dto.NewSuccessResponse(strategies))
}

// GetAllocationStrategies godoc
// @ID           getSystemAllocationStrategies
// @Summary      List payment allocation strategies
// @Description  Returns all available payment allocation strategies (FIFO, etc.)
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[[]StrategyInfo]
// @Failure      500 {object} dto.ErrorResponse
// @Router       /system/strategies/allocation [get]
func (h *StrategyHandler) GetAllocationStrategies(c *gin.Context) {
	strategies := h.buildAllocationStrategies()
	c.JSON(http.StatusOK, dto.NewSuccessResponse(strategies))
}
