/**
 * Print API Service
 *
 * API service for the printing module.
 * Uses axios instance configured with auth and tenant headers.
 */

import { axiosInstance } from '@/services/axios-instance'
import type {
  DocumentType,
  GeneratePDFRequest,
  ListTemplatesRequest,
  PaginatedResponse,
  PaperSize,
  PreviewDocumentRequest,
  PrintJob,
  PrintPreviewResponse,
  PrintTemplate,
} from './types'

const BASE_PATH = '/print'

/**
 * Preview a document as HTML
 */
export async function previewDocument(
  request: PreviewDocumentRequest
): Promise<PrintPreviewResponse> {
  const response = await axiosInstance.post<{ data: PrintPreviewResponse }>(
    `${BASE_PATH}/preview`,
    {
      document_type: request.documentType,
      document_id: request.documentId,
      template_id: request.templateId,
      data: request.data,
    }
  )
  return response.data.data
}

/**
 * Generate a PDF for a document
 */
export async function generatePDF(request: GeneratePDFRequest): Promise<PrintJob> {
  const response = await axiosInstance.post<{ data: PrintJob }>(`${BASE_PATH}/generate`, {
    document_type: request.documentType,
    document_id: request.documentId,
    document_number: request.documentNumber,
    template_id: request.templateId,
    copies: request.copies,
    data: request.data,
  })
  return response.data.data
}

/**
 * Get templates by document type
 */
export async function getTemplatesByDocType(docType: string): Promise<PrintTemplate[]> {
  const response = await axiosInstance.get<{ data: PrintTemplate[] }>(
    `${BASE_PATH}/templates/by-doc-type/${docType}`
  )
  return response.data.data
}

/**
 * List all templates with pagination
 */
export async function listTemplates(
  params?: ListTemplatesRequest
): Promise<PaginatedResponse<PrintTemplate>> {
  const queryParams = new URLSearchParams()
  if (params?.page) queryParams.set('page', params.page.toString())
  if (params?.pageSize) queryParams.set('page_size', params.pageSize.toString())
  if (params?.orderBy) queryParams.set('order_by', params.orderBy)
  if (params?.orderDir) queryParams.set('order_dir', params.orderDir)
  if (params?.search) queryParams.set('search', params.search)
  if (params?.docType) queryParams.set('doc_type', params.docType)
  if (params?.status) queryParams.set('status', params.status)

  const response = await axiosInstance.get<{
    data: PrintTemplate[]
    meta: { total: number; page: number; limit: number }
  }>(`${BASE_PATH}/templates?${queryParams.toString()}`)

  return {
    items: response.data.data,
    total: response.data.meta.total,
    page: response.data.meta.page,
    size: response.data.meta.limit,
  }
}

/**
 * Get a single template by ID
 */
export async function getTemplate(templateId: string): Promise<PrintTemplate> {
  const response = await axiosInstance.get<{ data: PrintTemplate }>(
    `${BASE_PATH}/templates/${templateId}`
  )
  return response.data.data
}

/**
 * Get print job by ID
 */
export async function getJob(jobId: string): Promise<PrintJob> {
  const response = await axiosInstance.get<{ data: PrintJob }>(`${BASE_PATH}/jobs/${jobId}`)
  return response.data.data
}

/**
 * Get jobs for a specific document
 */
export async function getJobsByDocument(docType: string, documentId: string): Promise<PrintJob[]> {
  const response = await axiosInstance.get<{ data: PrintJob[] }>(
    `${BASE_PATH}/jobs/by-document/${docType}/${documentId}`
  )
  return response.data.data
}

/**
 * Get available document types
 */
export async function getDocumentTypes(): Promise<DocumentType[]> {
  const response = await axiosInstance.get<{ data: DocumentType[] }>(`${BASE_PATH}/document-types`)
  return response.data.data
}

/**
 * Get available paper sizes
 */
export async function getPaperSizes(): Promise<PaperSize[]> {
  const response = await axiosInstance.get<{ data: PaperSize[] }>(`${BASE_PATH}/paper-sizes`)
  return response.data.data
}

/**
 * Get PDF download URL for a job
 */
export function getDownloadUrl(jobId: string): string {
  return `${BASE_PATH}/jobs/${jobId}/download`
}
