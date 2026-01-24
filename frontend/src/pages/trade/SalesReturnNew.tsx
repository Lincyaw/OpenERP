import { SalesReturnForm } from '@/features/trade/SalesReturnForm'

/**
 * Sales return creation page
 *
 * Renders the SalesReturnForm component in create mode
 * Supports pre-selecting an order via ?order_id query param
 */
export default function SalesReturnNewPage() {
  return <SalesReturnForm />
}
