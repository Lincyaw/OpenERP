import { PurchaseReturnForm } from '@/features/trade/PurchaseReturnForm'

/**
 * Purchase return creation page
 *
 * Renders the PurchaseReturnForm component in create mode
 * Supports pre-selecting an order via ?order_id query param
 */
export default function PurchaseReturnNewPage() {
  return <PurchaseReturnForm />
}
