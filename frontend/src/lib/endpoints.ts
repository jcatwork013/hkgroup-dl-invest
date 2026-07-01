import { apiFetch } from "./api";
import type {
  AdminDashboard,
  AdminDividend,
  AdminUser,
  Product,
  ProductCategory,
  SalesOrderRow,
  OrderDetail,
  SalerStat,
  Saler,
  SalesCommission,
  AuthResponse,
  AuditLog,
  CapTableRow,
  Commission,
  ContractResponse,
  Dashboard,
  Dividend,
  DividendPayout,
  IntegrityCheck,
  Investment,
  InvestorProfile,
  KycRecord,
  Offering,
  OfferingResponse,
  MyOfferingResponse,
  OfferingTier,
  TieredPlan,
  PaymentInfo,
  PoolStatus,
  ReferralsResponse,
  RevenueDistribution,
  User,
  Wallet,
  WalletInfo,
  Withdrawal,
  AdminWithdrawal,
  AdminWithdrawalList,
  AdminWalletBalance,
  SignResponse,
} from "./types";

// ---- Auth ----
export interface RegisterInput {
  full_name: string;
  phone: string;
  email: string;
  password: string;
  referral_code?: string;
  referral_type?: string;
}

export const authApi = {
  register: (input: RegisterInput) =>
    apiFetch<AuthResponse>("/api/v1/auth/register", {
      method: "POST",
      body: input,
    }),
  login: (email: string, password: string) =>
    apiFetch<AuthResponse>("/api/v1/auth/login", {
      method: "POST",
      body: { email, password },
    }),
  // Fresh current user (KYC status + message) — drives the notification bell.
  me: () => apiFetch<User>("/api/v1/me", { auth: true }),
  // Đổi mật khẩu — dùng chung cho investor & admin (route chỉ cần token hợp lệ).
  changePassword: (current_password: string, new_password: string) =>
    apiFetch<{ status: string }>("/api/v1/me/password", {
      method: "POST",
      body: { current_password, new_password },
      auth: true,
    }),
  // Quên mật khẩu — gửi link đặt lại tới email.
  forgotPassword: (email: string) =>
    apiFetch<{ status: string }>("/api/v1/auth/forgot-password", {
      method: "POST",
      body: { email },
    }),
  // Đặt mật khẩu mới bằng token từ link email.
  resetPassword: (token: string, new_password: string) =>
    apiFetch<{ status: string }>("/api/v1/auth/reset-password", {
      method: "POST",
      body: { token, new_password },
    }),
};

// ---- Public ----
export type SiteSettings = Record<string, string>;

export const publicApi = {
  offering: () => apiFetch<OfferingResponse>("/api/v1/offering"),
  settings: () => apiFetch<SiteSettings>("/api/v1/settings"),
  pool: () => apiFetch<PoolStatus>("/api/v1/pool"),
};

// ---- Investor ----
export const investorApi = {
  // Authenticated offering: early-bird (special) tiers appear only when whitelisted.
  myOffering: () =>
    apiFetch<MyOfferingResponse>("/api/v1/me/offering", { auth: true }),
  dashboard: () =>
    apiFetch<Dashboard>("/api/v1/me/dashboard", { auth: true }),
  investments: () =>
    apiFetch<Investment[]>("/api/v1/me/investments", { auth: true }),
  referrals: () =>
    apiFetch<ReferralsResponse>("/api/v1/me/referrals", { auth: true }),
  dividends: () =>
    apiFetch<Dividend[]>("/api/v1/me/dividends", { auth: true }),
  submitKyc: (body: {
    cccd_number: string;
    cccd_image_url: string;
    cccd_back_url: string;
    selfie_url: string;
  }) =>
    apiFetch<KycRecord>("/api/v1/kyc", { method: "POST", body, auth: true }),
  uploadKyc: (file: File, kind: "cccd" | "cccd_back" | "selfie") => {
    const fd = new FormData();
    fd.append("file", file);
    fd.append("kind", kind);
    return apiFetch<{ id: string; url: string }>("/api/v1/uploads/kyc", {
      method: "POST",
      body: fd,
      auth: true,
    });
  },
  consent: (type: string) =>
    apiFetch<unknown>("/api/v1/consent", {
      method: "POST",
      body: { type },
      auth: true,
    }),
  startContract: (tier_id: string) =>
    apiFetch<ContractResponse>("/api/v1/investments/contract", {
      method: "POST",
      body: { tier_id },
      auth: true,
    }),
  signContract: (
    body: { contract_id: string; otp_ref: string; otp_code: string },
    idempotencyKey: string
  ) =>
    apiFetch<SignResponse>("/api/v1/investments/sign", {
      method: "POST",
      body,
      auth: true,
      headers: { "Idempotency-Key": idempotencyKey },
    }),
  declareTransfer: (investmentId: string) =>
    apiFetch<PaymentInfo>(
      `/api/v1/investments/${investmentId}/declare-transfer`,
      { method: "POST", auth: true }
    ),
  getProfile: () =>
    apiFetch<InvestorProfile>("/api/v1/me/profile", { auth: true }),
  updateProfile: (body: InvestorProfile) =>
    apiFetch<InvestorProfile>("/api/v1/me/profile", {
      method: "PUT",
      body,
      auth: true,
    }),
  wallet: () => apiFetch<WalletInfo>("/api/v1/me/wallet", { auth: true }),
  withdrawals: () => apiFetch<Withdrawal[]>("/api/v1/me/withdrawals", { auth: true }),
  requestWithdrawal: (amount: number, note: string) =>
    apiFetch<Withdrawal>("/api/v1/me/withdrawals", {
      method: "POST",
      body: { amount, note },
      auth: true,
    }),
  // Ví CỔ TỨC — số dư rút riêng, cùng lịch rút với ví hoa hồng.
  dividendWallet: () => apiFetch<WalletInfo>("/api/v1/me/dividend-wallet", { auth: true }),
  dividendWithdrawals: () => apiFetch<Withdrawal[]>("/api/v1/me/dividend-withdrawals", { auth: true }),
  requestDividendWithdrawal: (amount: number, note: string) =>
    apiFetch<Withdrawal>("/api/v1/me/dividend-withdrawals", {
      method: "POST",
      body: { amount, note },
      auth: true,
    }),
};

// ---- Admin ----
export const adminApi = {
  dashboard: () =>
    apiFetch<AdminDashboard>("/api/v1/admin/dashboard", { auth: true }),
  capTable: () =>
    apiFetch<CapTableRow[]>("/api/v1/admin/cap-table", { auth: true }),
  integrityCheck: () =>
    apiFetch<IntegrityCheck>("/api/v1/admin/integrity-check", { auth: true }),
  auditLogs: (limit = 50, offset = 0) =>
    apiFetch<AuditLog[]>(
      `/api/v1/admin/audit-logs?limit=${limit}&offset=${offset}`,
      { auth: true }
    ),
  pendingKyc: () =>
    apiFetch<KycRecord[]>("/api/v1/admin/kyc/pending", { auth: true }),
  users: () =>
    apiFetch<AdminUser[]>("/api/v1/admin/users", { auth: true }),
  createUser: (body: {
    full_name: string;
    phone: string;
    email: string;
    password: string;
    role: string;
  }) =>
    apiFetch<AdminUser>("/api/v1/admin/users", {
      method: "POST",
      body,
      auth: true,
    }),
  manualKyc: (id: string, approve: boolean, reason?: string) =>
    apiFetch<AdminUser>(`/api/v1/admin/users/${id}/kyc`, {
      method: "POST",
      body: { approve, reason },
      auth: true,
    }),
  userProfile: (id: string) =>
    apiFetch<InvestorProfile>(`/api/v1/admin/users/${id}/profile`, { auth: true }),
  userCommissions: (id: string) =>
    apiFetch<{ wallet: Wallet; commissions: Commission[] }>(
      `/api/v1/admin/users/${id}/commissions`,
      { auth: true }
    ),
  userKyc: (id: string) =>
    apiFetch<KycRecord>(`/api/v1/admin/users/${id}/kyc`, { auth: true }),
  deleteUser: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/users/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // ---- Danh mục sản phẩm ----
  categories: () =>
    apiFetch<ProductCategory[]>("/api/v1/admin/categories", { auth: true }),
  createCategory: (body: Partial<ProductCategory>) =>
    apiFetch<ProductCategory>("/api/v1/admin/categories", { method: "POST", body, auth: true }),
  updateCategory: (id: string, body: Partial<ProductCategory>) =>
    apiFetch<ProductCategory>(`/api/v1/admin/categories/${id}`, { method: "PUT", body, auth: true }),
  deleteCategory: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/categories/${id}`, { method: "DELETE", auth: true }),

  // ---- Sản phẩm ----
  products: () =>
    apiFetch<Product[]>("/api/v1/admin/products", { auth: true }),
  createProduct: (body: Partial<Product>) =>
    apiFetch<Product>("/api/v1/admin/products", { method: "POST", body, auth: true }),
  updateProduct: (id: string, body: Partial<Product>) =>
    apiFetch<Product>(`/api/v1/admin/products/${id}`, { method: "PUT", body, auth: true }),
  deleteProduct: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/products/${id}`, { method: "DELETE", auth: true }),
  uploadProductImage: (file: File) => {
    const fd = new FormData();
    fd.append("file", file);
    return apiFetch<{ id: string; url: string }>("/api/v1/admin/products/image", { method: "POST", body: fd, auth: true });
  },

  // ---- Bán hàng (admin) ----
  allOrders: () => apiFetch<SalesOrderRow[]>("/api/v1/admin/orders", { auth: true }),
  salerStats: () => apiFetch<SalerStat[]>("/api/v1/admin/salers", { auth: true }),
  reviewKyc: (id: string, approve: boolean, reason?: string) =>
    apiFetch<unknown>(`/api/v1/admin/kyc/${id}/review`, {
      method: "POST",
      body: { approve, reason },
      auth: true,
    }),
  investments: (status?: string) =>
    apiFetch<Investment[]>(
      `/api/v1/admin/investments${status ? `?status=${status}` : ""}`,
      { auth: true }
    ),
  reconcile: (id: string) =>
    apiFetch<unknown>(`/api/v1/admin/investments/${id}/reconcile`, {
      method: "POST",
      auth: true,
    }),
  approveInvestment: (id: string) =>
    apiFetch<unknown>(`/api/v1/admin/investments/${id}/approve`, {
      method: "POST",
      auth: true,
    }),
  rejectInvestment: (id: string, reason: string) =>
    apiFetch<unknown>(`/api/v1/admin/investments/${id}/reject`, {
      method: "POST",
      body: { reason },
      auth: true,
    }),
  approveCommission: (id: string) =>
    apiFetch<unknown>(`/api/v1/admin/commissions/${id}/approve`, {
      method: "POST",
      auth: true,
    }),
  payCommission: (id: string) =>
    apiFetch<unknown>(`/api/v1/admin/commissions/${id}/pay`, {
      method: "POST",
      auth: true,
    }),
  dividends: () =>
    apiFetch<AdminDividend[]>("/api/v1/admin/dividends", { auth: true }),
  declareDividend: (body: {
    period: string;
    total_amount: number;
    note: string;
  }) =>
    apiFetch<{ dividend: AdminDividend; payouts: DividendPayout[] }>(
      "/api/v1/admin/dividends",
      { method: "POST", body, auth: true }
    ),
  dividendPayouts: (dividendId: string) =>
    apiFetch<DividendPayout[]>(
      `/api/v1/admin/dividends/${dividendId}/payouts`,
      { auth: true }
    ),
  payPayout: (id: string) =>
    apiFetch<unknown>(`/api/v1/admin/dividend-payouts/${id}/pay`, {
      method: "POST",
      auth: true,
    }),
  deleteDividend: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/dividends/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  // Admin đọc settings qua endpoint riêng: secret (resend_api_key) KHÔNG trả về,
  // chỉ kèm cờ "<key>_configured" để biết đã cấu hình hay chưa.
  settings: () => apiFetch<SiteSettings>("/api/v1/admin/settings", { auth: true }),
  updateSettings: (body: SiteSettings) =>
    apiFetch<SiteSettings>("/api/v1/admin/settings", {
      method: "PUT",
      body,
      auth: true,
    }),
  // Admin gửi email link đặt lại mật khẩu cho 1 tài khoản.
  resetUserPassword: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/users/${id}/reset-password`, {
      method: "POST",
      auth: true,
    }),
  distributions: () =>
    apiFetch<RevenueDistribution[]>("/api/v1/admin/distributions", { auth: true }),
  distribute: (period: string, total_revenue: number) =>
    apiFetch<unknown>("/api/v1/admin/distributions", {
      method: "POST",
      body: { period, total_revenue },
      auth: true,
    }),
  deleteDistribution: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/distributions/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  previewTiered: (period: string, total_revenue: number) =>
    apiFetch<TieredPlan>("/api/v1/admin/distributions/tiered/preview", {
      method: "POST",
      body: { period, total_revenue },
      auth: true,
    }),
  distributeTiered: (period: string, total_revenue: number) =>
    apiFetch<unknown>("/api/v1/admin/distributions/tiered", {
      method: "POST",
      body: { period, total_revenue },
      auth: true,
    }),
  withdrawals: () =>
    apiFetch<AdminWithdrawalList>("/api/v1/admin/withdrawals", { auth: true }),
  processWithdrawal: (id: string, status: string) =>
    apiFetch<unknown>(`/api/v1/admin/withdrawals/${id}/process`, {
      method: "POST",
      body: { status },
      auth: true,
    }),
  // Số dư ví hoa hồng từng tài khoản — để admin lập lệnh rút dùm.
  wallets: () => apiFetch<AdminWalletBalance[]>("/api/v1/admin/wallets", { auth: true }),
  createWithdrawalFor: (userId: string, amount: number, note: string) =>
    apiFetch<Withdrawal>(`/api/v1/admin/users/${userId}/withdrawals`, {
      method: "POST",
      body: { amount, note },
      auth: true,
    }),
  // ---- Tier management + early-bird whitelist ----
  tiers: () => apiFetch<OfferingTier[]>("/api/v1/admin/tiers", { auth: true }),
  createTier: (body: Partial<OfferingTier>) =>
    apiFetch<OfferingTier>("/api/v1/admin/tiers", {
      method: "POST",
      body,
      auth: true,
    }),
  updateTier: (id: string, body: Partial<OfferingTier>) =>
    apiFetch<OfferingTier>(`/api/v1/admin/tiers/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  setTierActive: (id: string, active: boolean) =>
    apiFetch<OfferingTier>(`/api/v1/admin/tiers/${id}/active`, {
      method: "POST",
      body: { active },
      auth: true,
    }),
  deleteTier: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/admin/tiers/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  offerings: () => apiFetch<Offering[]>("/api/v1/admin/offerings", { auth: true }),
  openRound: (body: {
    name: string;
    valuation_vnd: number;
    total_shares: number;
    shares_for_sale: number;
  }) =>
    apiFetch<Offering>("/api/v1/admin/offerings", {
      method: "POST",
      body,
      auth: true,
    }),
};

export type { Commission, AdminDashboard };

// salesApi — dùng chung cho admin & saler (RBAC ở backend: requireAnyRole admin|saler).
export const salesApi = {
  products: () => apiFetch<Product[]>("/api/v1/sales/products", { auth: true }),
  salers: () => apiFetch<Saler[]>("/api/v1/sales/salers", { auth: true }),
  myOrders: () => apiFetch<SalesOrderRow[]>("/api/v1/sales/my-orders", { auth: true }),
  myCommissions: () => apiFetch<SalesCommission[]>("/api/v1/sales/my-commissions", { auth: true }),
  orderDetail: (id: string) => apiFetch<OrderDetail>(`/api/v1/sales/orders/${id}`, { auth: true }),
  createOrder: (body: {
    customer_name: string;
    customer_phone: string;
    seller_id?: string;
    affiliate_id?: string;
    note: string;
    items: { product_id: string; qty: number }[];
  }) => apiFetch<SalesOrderRow>("/api/v1/sales/orders", { method: "POST", body, auth: true }),
  payOrder: (id: string) => apiFetch<OrderDetail>(`/api/v1/sales/orders/${id}/pay`, { method: "POST", auth: true }),
  cancelOrder: (id: string) => apiFetch<SalesOrderRow>(`/api/v1/sales/orders/${id}/cancel`, { method: "POST", auth: true }),
};
