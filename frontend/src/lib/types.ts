// Shared API types for the HK SHAREHOLDER Platform.
// NOTE: Intentionally NO fields for guaranteed/target return, ROI, profit, etc.
// The only "money back" figure is dividend_received_vnd (actually paid dividends).

export type Role = "investor" | "admin" | "saler";
export type KycStatus = "unverified" | "pending" | "approved" | "rejected";

export interface User {
  id: string;
  full_name: string;
  email: string;
  phone: string;
  role: Role;
  kyc_status: KycStatus;
  kyc_message?: string;
  referral_code: string;
}

export interface Tokens {
  access_token: string;
  refresh_token: string;
}

export interface AuthResponse {
  user: User;
  tokens: Tokens;
}

export interface RefreshResponse {
  tokens: Tokens;
}

export interface OfferingTier {
  id: string;
  name: string;
  amount_vnd: number;
  shares: number;
  ownership_pct: number;
  active?: boolean;
  sort_order?: number;
  commission_rate?: number;
}

export interface Offering {
  id: string;
  name: string;
  valuation_vnd: number;
  total_shares: number;
  shares_for_sale: number;
  shares_sold: number;
  status: string;
}

export interface BandBreakdown {
  key: string;
  label: string;
  rate: number;
  accounts: number;
  bonus_total: number;
}

// Preview of the "đồng chia + bonus" (9%+6%) distribution plan — drives the admin pie chart.
export interface TieredPlan {
  period: string;
  revenue: number;
  equal_rate: number;
  bonus_rate: number;
  equal_pool: number;
  bonus_pool: number;
  accounts: number;
  equal_each: number;
  bands: BandBreakdown[];
  residual_mode: string;
  residual: number;
  distributed: number;
  scaled: boolean;
}

// Kết quả xem trước / thực thi "Quét cổ tức" (gom 15% pool của đơn thành công chưa gộp).
export interface SweepResult {
  swept_orders: number;
  revenue_vnd: number;
  pool_vnd: number;
  plan: TieredPlan | null;
}

export interface OfferingResponse {
  offering: Offering;
  tiers: OfferingTier[];
}

// Authenticated invest-page payload.
export interface MyOfferingResponse {
  offering: Offering;
  tiers: OfferingTier[];
}

export interface InvestorProfile {
  user_id?: string;
  date_of_birth: string;
  gender: string;
  nationality: string;
  cccd_number: string;
  cccd_issue_date: string;
  cccd_issue_place: string;
  permanent_address: string;
  contact_address: string;
  occupation: string;
  tax_code: string;
  bank_name: string;
  bank_account_number: string;
  bank_account_name: string;
  updated_at?: string;
}

export interface PoolStatus {
  valuation_vnd: number;
  pool_target_vnd: number;
  raised_vnd: number;
  remaining_vnd: number;
  progress_pct: number;
  shares_sold: number;
  shares_for_sale: number;
  total_revenue_vnd: number;
  total_investor_pool_vnd: number;
  pool_rate: number;
  investor_share_rate: number;
}

export interface RevenueDistribution {
  id: string;
  period: string;
  total_revenue: number;
  pool_rate: number;
  investor_share_rate: number;
  investor_pool: number;
  created_at: string;
}

export interface Wallet {
  earned_vnd: number;
  withdrawn_vnd: number;
  available_vnd: number;
  pending_vnd: number; // hoa hồng chưa duyệt (đã gộp trong available) — chờ admin duyệt
}

// Lịch rút tiền: các ngày trong tháng được phép gửi yêu cầu rút (mặc định 15 & 30).
export interface WithdrawalWindow {
  days: number[];
  today: number;
  open_today: boolean;
  next_date: string; // YYYY-MM-DD
  days_until: number;
}

// GET /me/wallet — số dư + lịch rút.
export interface WalletInfo extends Wallet {
  window: WithdrawalWindow;
}

// GET /admin/withdrawals — danh sách + lịch rút hiện hành.
export interface AdminWithdrawalList {
  items: AdminWithdrawal[];
  window: WithdrawalWindow;
}

export interface Withdrawal {
  id: string;
  user_id: string;
  amount: number;
  status: "pending" | "approved" | "paid" | "rejected";
  note: string;
  requested_at: string;
  processed_at: string | null;
}

export interface AdminWithdrawal extends Withdrawal {
  source: "commission" | "dividend";
  full_name: string;
  email: string;
}

// GET /admin/wallets — số dư ví hoa hồng từng tài khoản (để admin lập lệnh rút dùm).
export interface AdminWalletBalance extends Wallet {
  id: string;
  email: string;
  full_name: string;
}

export interface AdminUser {
  id: string;
  full_name: string;
  phone: string;
  email: string;
  role: "investor" | "admin" | "saler";
  kyc_status: "unverified" | "pending" | "approved" | "rejected";
  referral_code: string;
  created_at: string;
}

export interface KycRecord {
  id: string;
  user_id: string;
  cccd_number: string;
  cccd_image_url?: string;
  cccd_back_url?: string;
  selfie_url?: string;
  status: KycStatus | string;
  created_at: string;
}

export interface Dashboard {
  capital_contributed_vnd: number;
  shares: number;
  ownership_pct: number;
  dividend_received_vnd: number;
}

export type InvestmentStatus = "pending" | "reconciled" | "approved" | "rejected";

export interface Investment {
  id: string;
  code: string;
  amount_vnd: number;
  shares: number;
  status: InvestmentStatus;
  created_at: string;
}

export interface Referral {
  referee_id: string;
  referrer_id: string;
  referral_type: string;
  created_at: string;
}

export interface Commission {
  id: string;
  level: number;
  base_amount: number;
  rate: number;
  amount: number;
  tax_pit: number;
  net_amount: number;
  status: string;
}

export interface ReferralsResponse {
  referrals: Referral[];
  commissions: Commission[];
}

// Investor view (/me/dividends): one row per payout received.
export interface Dividend {
  amount: number;
  shares: number;
  paid_at: string;
  period: string;
  note: string;
}

// Admin view (/admin/dividends): the declared dividend header.
export interface AdminDividend {
  id: string;
  declared_by?: string;
  declared_at: string;
  period: string;
  total_amount: number;
  note: string;
}

// Admin view (/admin/dividends/{id}/payouts): per-shareholder payout.
export interface DividendPayout {
  id: string;
  dividend_id: string;
  user_id: string;
  full_name: string;
  email: string;
  shares: number;
  amount: number;
  paid_at: string | null;
  // Cấu thành cổ tức tiered (đồng chia + bonus theo hạng đầu tư). 0/"" với cổ tức pro-rata thủ công.
  equal_share: number;
  bonus: number;
  band: string;
  band_rate: number;
  invested_vnd: number;
}

export interface ContractResponse {
  contract: { id: string };
  otp_ref: string;
  otp_code: string;
}

export interface PaymentInfo {
  bank: string;
  company_account: string;
  company_account_name: string;
  amount_vnd: number;
  transfer_note: string;
}

export interface SignResponse {
  investment: {
    id: string;
    code: string;
    amount_vnd: number;
    shares: number;
    status: InvestmentStatus;
  };
  payment: PaymentInfo;
}

// ---- Admin ----
export interface AdminDashboard {
  stats: {
    total_capital_reconciled: number;
    shareholder_count: number;
    shares_sold: number;
    shares_for_sale: number;
  };
  customer_commission_gross_vnd: number;
  customer_commission_tax_vnd: number;
  customer_commission_net_vnd: number;
  investor_commission_gross_vnd: number;
}

export interface CapTableRow {
  user_id: string;
  full_name: string;
  email: string;
  shares: number;
  ownership_pct: number;
}

export interface IntegrityCheck {
  healthy: boolean;
  mismatches: unknown[] | null;
}

export interface AuditLog {
  id: string;
  actor_id: string;
  action: string;
  entity: string;
  entity_id: string;
  created_at: string;
}

export interface ApiError {
  error: string;
  code?: string;
}

// ---- Sales: danh mục + sản phẩm ----
export interface ProductCategory {
  id: string;
  name: string;
  slug: string;
  description: string;
  sort_order: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Product {
  id: string;
  category_id: string | null;
  sku: string;
  name: string;
  badge: string;
  price_vnd: number;
  cost_vnd: number;
  image_url: string;
  summary: string;
  description: string;
  spec_warranty: string;
  spec_trace: string;
  spec_delivery: string;
  spec_return: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

// ---- Sales: đơn hàng + hoa hồng + giám sát saler ----
export type SalesOrderStatus = "pending" | "paid" | "cancelled";

export interface SalesOrderRow {
  id: string;
  code: string;
  customer_name: string;
  customer_phone: string;
  seller_id: string;
  affiliate_id: string | null;
  subtotal_vnd: number;
  cost_vnd: number;
  status: SalesOrderStatus;
  note: string;
  created_by: string;
  paid_at: string | null;
  created_at: string;
  seller_name: string;
  affiliate_name: string;
}

export interface SalesOrderItem {
  id: string;
  order_id: string;
  product_id: string;
  name: string;
  qty: number;
  unit_price_vnd: number;
  unit_cost_vnd: number;
  line_total_vnd: number;
}

export interface SalesDistribution {
  order_id: string;
  total_vnd: number;
  seller_vnd: number;
  affiliate_vnd: number;
  equal_share_vnd: number;
  pool_vnd: number;
  cost_vnd: number;
  operations_vnd: number;
  dividend_pool_vnd: number;
}

export interface OrderDetail {
  order: SalesOrderRow;
  items: SalesOrderItem[];
  distribution?: SalesDistribution | null;
  seller_name: string;
  affiliate_name: string;
}

export interface SalesCommission {
  id: string;
  order_id: string;
  beneficiary_id: string;
  kind: "seller" | "affiliate";
  base_amount: number;
  rate: number;
  amount: number;
  tax_pit: number;
  net_amount: number;
  status: string;
  created_at: string;
}

export interface SalerStat {
  seller_id: string;
  full_name: string;
  email: string;
  phone: string;
  paid_orders: number;
  pending_orders: number;
  revenue_vnd: number;
  commission_net_vnd: number;
}

export interface Saler {
  id: string;
  full_name: string;
  email: string;
  phone: string;
}
