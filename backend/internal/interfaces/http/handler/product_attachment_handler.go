package handler

import (
	"github.com/erp/backend/internal/application/catalog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductAttachmentHandler handles product attachment-related API endpoints
type ProductAttachmentHandler struct {
	BaseHandler
	attachmentService *catalog.AttachmentService
}

// NewProductAttachmentHandler creates a new ProductAttachmentHandler
func NewProductAttachmentHandler(attachmentService *catalog.AttachmentService) *ProductAttachmentHandler {
	return &ProductAttachmentHandler{
		attachmentService: attachmentService,
	}
}

// InitiateUploadRequest represents a request to initiate a file upload
//
//	@Description	Request body for initiating a file upload
type InitiateUploadRequest struct {
	ProductID   string `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type        string `json:"type" binding:"required,oneof=main_image gallery_image document other" example:"gallery_image"`
	FileName    string `json:"file_name" binding:"required,min=1,max=255" example:"product-photo.jpg"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0,max=104857600" example:"1048576"` // max 100MB
	ContentType string `json:"content_type" binding:"required" example:"image/jpeg"`
}

// ConfirmUploadRequest represents a request to confirm a file upload
//
//	@Description	Request body for confirming a file upload
type ConfirmUploadRequest struct {
	// no body fields needed, attachment ID comes from URL path
}

// SetMainImageRequest represents a request to set an attachment as main image
//
//	@Description	Request body for setting an attachment as main image
type SetMainImageRequest struct {
	// no body fields needed, attachment ID comes from URL path
}

// ReorderAttachmentsRequest represents a request to reorder attachments
//
//	@Description	Request body for reordering attachments
type ReorderAttachmentsRequest struct {
	AttachmentIDs []string `json:"attachment_ids" binding:"required,min=1" example:"550e8400-e29b-41d4-a716-446655440001,550e8400-e29b-41d4-a716-446655440002"`
}

// InitiateUpload godoc
//
//	@Summary		Initiate a file upload
//	@Description	Creates a pending attachment record and returns a presigned upload URL
//	@Tags			product-attachments
//	@ID				initiateProductAttachmentUpload
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			request		body		InitiateUploadRequest	true	"Upload initiation request"
//	@Success		201			{object}	APIResponse[catalog.InitiateUploadResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse	"Product not found"
//	@Failure		422			{object}	dto.ErrorResponse	"Attachment limit exceeded or disallowed content type"
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/attachments/upload [post]
func (h *ProductAttachmentHandler) InitiateUpload(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req InitiateUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	// Get user ID from JWT context (optional)
	userID, _ := getUserID(c)
	var uploadedBy *uuid.UUID
	if userID != uuid.Nil {
		uploadedBy = &userID
	}

	// Convert to application DTO
	appReq := catalog.InitiateUploadRequest{
		ProductID:   productID,
		Type:        req.Type,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
	}

	response, err := h.attachmentService.InitiateUpload(c.Request.Context(), tenantID, appReq, uploadedBy)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, response)
}

// ConfirmUpload godoc
//
//	@Summary		Confirm a file upload
//	@Description	Verifies the upload completed and activates the attachment
//	@Tags			product-attachments
//	@ID				confirmProductAttachmentUpload
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Attachment ID"	format(uuid)
//	@Success		200			{object}	APIResponse[catalog.AttachmentResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse	"Attachment not found"
//	@Failure		422			{object}	dto.ErrorResponse	"Upload not found in storage"
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/attachments/{id}/confirm [post]
func (h *ProductAttachmentHandler) ConfirmUpload(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid attachment ID format")
		return
	}

	response, err := h.attachmentService.ConfirmUpload(c.Request.Context(), tenantID, attachmentID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, response)
}

// GetByID godoc
//
//	@Summary		Get attachment by ID
//	@Description	Retrieve an attachment by its ID
//	@Tags			product-attachments
//	@ID				getProductAttachmentById
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Attachment ID"	format(uuid)
//	@Success		200			{object}	APIResponse[catalog.AttachmentResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/attachments/{id} [get]
func (h *ProductAttachmentHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid attachment ID format")
		return
	}

	response, err := h.attachmentService.GetByID(c.Request.Context(), tenantID, attachmentID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, response)
}

// ListByProduct godoc
//
//	@Summary		List attachments by product
//	@Description	Retrieve a paginated list of attachments for a specific product
//	@Tags			product-attachments
//	@ID				listProductAttachments
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Product ID"	format(uuid)
//	@Param			search		query		string	false	"Search term (file name)"
//	@Param			status		query		string	false	"Attachment status"	Enums(pending, active, deleted)
//	@Param			type		query		string	false	"Attachment type"	Enums(main_image, gallery_image, document, other)
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			page_size	query		int		false	"Page size"			default(20)	maximum(100)
//	@Param			order_by	query		string	false	"Order by field"	default(sort_order)
//	@Param			order_dir	query		string	false	"Order direction"	Enums(asc, desc)	default(asc)
//	@Success		200			{object}	APIResponse[[]catalog.AttachmentListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse	"Product not found"
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/products/{id}/attachments [get]
func (h *ProductAttachmentHandler) ListByProduct(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var filter catalog.AttachmentListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	attachments, total, err := h.attachmentService.GetByProduct(c.Request.Context(), tenantID, productID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, attachments, total, filter.Page, filter.PageSize)
}

// GetMainImage godoc
//
//	@Summary		Get product main image
//	@Description	Retrieve the main image for a specific product
//	@Tags			product-attachments
//	@ID				getProductMainImage
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Product ID"	format(uuid)
//	@Success		200			{object}	APIResponse[catalog.AttachmentResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse	"Product or main image not found"
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/products/{id}/attachments/main [get]
func (h *ProductAttachmentHandler) GetMainImage(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	response, err := h.attachmentService.GetMainImage(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, response)
}

// Delete godoc
//
//	@Summary		Delete an attachment
//	@Description	Soft delete an attachment (marks as deleted)
//	@Tags			product-attachments
//	@ID				deleteProductAttachment
//	@Produce		json
//	@Param			X-Tenant-ID	header	string	false	"Tenant ID (optional for dev)"
//	@Param			id			path	string	true	"Attachment ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/attachments/{id} [delete]
func (h *ProductAttachmentHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid attachment ID format")
		return
	}

	err = h.attachmentService.Delete(c.Request.Context(), tenantID, attachmentID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// SetAsMainImage godoc
//
//	@Summary		Set attachment as main image
//	@Description	Promote an image attachment to be the product's main image
//	@Tags			product-attachments
//	@ID				setProductAttachmentAsMainImage
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Attachment ID"	format(uuid)
//	@Success		200			{object}	APIResponse[catalog.AttachmentResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse	"Attachment not found"
//	@Failure		422			{object}	dto.ErrorResponse	"Attachment is not an image"
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/attachments/{id}/main [post]
func (h *ProductAttachmentHandler) SetAsMainImage(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid attachment ID format")
		return
	}

	response, err := h.attachmentService.SetAsMainImage(c.Request.Context(), tenantID, attachmentID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, response)
}

// Reorder godoc
//
//	@Summary		Reorder attachments
//	@Description	Update the sort order of attachments for a product
//	@Tags			product-attachments
//	@ID				reorderProductAttachments
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			id			path		string						true	"Product ID"	format(uuid)
//	@Param			request		body		ReorderAttachmentsRequest	true	"Reorder request"
//	@Success		200			{object}	APIResponse[[]catalog.AttachmentListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse	"Product or attachments not found"
//	@Failure		422			{object}	dto.ErrorResponse	"Invalid attachment belongs to another product"
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/products/{id}/attachments/reorder [post]
func (h *ProductAttachmentHandler) Reorder(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var req ReorderAttachmentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert string IDs to UUIDs
	attachmentIDs := make([]uuid.UUID, len(req.AttachmentIDs))
	for i, idStr := range req.AttachmentIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			h.BadRequest(c, "Invalid attachment ID format")
			return
		}
		attachmentIDs[i] = id
	}

	responses, err := h.attachmentService.ReorderAttachments(c.Request.Context(), tenantID, productID, attachmentIDs)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, responses)
}
