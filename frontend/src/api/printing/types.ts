/**
 * Print API Types
 *
 * Types for the printing module API endpoints.
 * Based on backend DTOs from backend/internal/application/printing/dto.go
 */

/** Page margins configuration */
export interface Margins {
  top: number
  right: number
  bottom: number
  left: number
}

/** Paper size information */
export interface PaperSize {
  code: string
  width: number
  height: number
}

/** Document type information */
export interface DocumentType {
  code: string
  displayName: string
}

/** Print template response */
export interface PrintTemplate {
  id: string
  tenantId: string
  documentType: string
  name: string
  description: string
  content?: string
  paperSize: string
  orientation: string
  margins: Margins
  isDefault: boolean
  status: string
  createdAt: string
  updatedAt: string
}

/** Preview response */
export interface PrintPreviewResponse {
  html: string
  templateId: string
  paperSize: string
  orientation: string
  margins: Margins
}

/** Print job response */
export interface PrintJob {
  id: string
  tenantId: string
  templateId: string
  documentType: string
  documentId: string
  documentNumber: string
  status: 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED'
  copies: number
  pdfUrl?: string
  errorMessage?: string
  printedAt?: string
  printedBy?: string
  createdAt: string
  updatedAt: string
}

/** Preview document request */
export interface PreviewDocumentRequest {
  documentType: string
  documentId: string
  templateId?: string
  data?: unknown
}

/** Generate PDF request */
export interface GeneratePDFRequest {
  documentType: string
  documentId: string
  documentNumber: string
  templateId?: string
  copies?: number
  data?: unknown
}

/** List templates request */
export interface ListTemplatesRequest {
  page?: number
  pageSize?: number
  orderBy?: string
  orderDir?: 'asc' | 'desc'
  search?: string
  docType?: string
  status?: string
}

/** Paginated response wrapper */
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  size: number
}
