/**
 * API service — the single catalog of `/api/v2/mini/*` (member) and
 * `/api/v2/store/*` (staff) endpoints. Pages call these named methods; they
 * never hardcode paths. Paths follow CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md §8.
 */
const http = require('../utils/request');

const m = (p) => '/mini' + p; // member scope
const s = (p) => '/mini/staff' + p; // mini staff, store scope from staff token

function qs(params) {
  if (!params) return '';
  const parts = Object.keys(params)
    .filter((k) => params[k] !== undefined && params[k] !== null && params[k] !== '')
    .map((k) => `${encodeURIComponent(k)}=${encodeURIComponent(params[k])}`);
  return parts.length ? '?' + parts.join('&') : '';
}

// GET /mini/wallet returns an array of {assetType,availableAmount,heldAmount}
// accounts (server/internal/modules/wallet/wallet.go Account), not a flat
// object. Normalize it here so pages keep reading wallet.coins/points/etc.
// There is no couponCount on this resource — callers must derive it from
// getCoupons() separately.
function normalizeWallet(accounts) {
  const byType = {};
  (accounts || []).forEach((a) => { byType[a.assetType] = a; });
  return {
    coins: (byType.coins && byType.coins.availableAmount) || 0,
    points: (byType.points && byType.points.availableAmount) || 0,
    cashBalanceCent: (byType.cash_balance && byType.cash_balance.availableAmount) || 0,
    growthValue: (byType.growth_value && byType.growth_value.availableAmount) || 0,
  };
}

// The backend only ever writes these reason codes to the wallet ledger
// (server/internal/modules/order/write_repository.go,
// payment/settlement_repository.go, wallet/points_repository.go, wallet/wallet.go).
const LEDGER_REASON_LABEL = {
  order_payment: '订单支付',
  food_order_reward: '购买餐品赠送积分',
  food_order_cancel_clawback: '取消订单扣回赠送积分',
  food_order_cancel_rollback: '取消订单失败返还积分',
  refund: '订单退款返还',
  recharge: '金币充值到账',
  first_recharge_reward: '用户首充获得积分',
  high_value_recharge_reward: '满额充值获得积分',
  low_spend_reward: '预约低消达标奖励',
  invitation_first_low_spend: '邀请奖励',
  invitation_commission: '邀请奖励',
  invitation_refund_clawback: '邀请奖励退款冲正',
  sign_in: '签到奖励',
  admin_adjustment: '系统调整',
};

// Ledger entries come back as {id,assetType,direction,amount,balanceAfter,
// reason,sourceType,createdAt} with direction 'credit'|'debit'. Normalize to
// the {title,note,direction,delta,createdAt} shape pages already render.
function normalizeLedgerEntry(e) {
  const isDebit = e.direction === 'debit';
  return {
    id: e.id,
    title: LEDGER_REASON_LABEL[e.reason] || e.reason,
    note: '',
    direction: isDebit ? 'expense' : 'income',
    delta: (isDebit ? -1 : 1) * e.amount,
    createdAt: e.createdAt,
  };
}

// GET /mini/stores(/:id) returns StoreView {status,businessHours,latitude,
// longitude,...} (server/internal/modules/store/handler.go), but pages render
// open/hours/lat/lng. Derive those from the server fields, preferring any
// legacy field of the same meaning when present so both shapes work. Every
// other field (id/name/address/phone/distanceMeters + demo-only intro/
// addressDetail) passes through untouched. Note: the backend has no intro/
// addressDetail — those cards are wx:if-guarded and simply hide.
function normalizeStore(s) {
  if (!s) return s;
  return Object.assign({}, s, {
    open: s.open != null ? s.open : s.status === 'active',
    hours: s.hours || s.businessHours || '',
    lat: s.lat != null ? s.lat : s.latitude,
    lng: s.lng != null ? s.lng : s.longitude,
  });
}

// GET /mini/stores/:id/catalog/items returns ItemView with `stockQuantity`
// (server/internal/modules/catalog/catalog.go), but pages read `stock`.
function normalizeItem(it) {
  if (!it) return it;
  return Object.assign({}, it, {
    stock: it.stock != null ? it.stock : it.stockQuantity,
  });
}

// GET /mini/coupons returns MemberCouponView {entitlementId,description,
// couponType,valueCent,expiresAt,status} with status active|used|void|expired
// (coupon/dto.go, coupon/model.go), but pages read id/desc/type/amountCent/
// validUntil and filter status unused. Map the fields and the status vocabulary.
const COUPON_STATUS_MAP = { active: 'unused', used: 'used', expired: 'expired', void: 'expired' };
function normalizeCoupon(c) {
  if (!c) return c;
  return Object.assign({}, c, {
    id: c.id != null ? c.id : c.entitlementId,
    desc: c.desc != null ? c.desc : c.description,
    type: c.type != null ? c.type : c.couponType,
    amountCent: c.amountCent != null ? c.amountCent : c.valueCent,
    validUntil: c.validUntil != null ? c.validUntil : c.expiresAt,
    status: COUPON_STATUS_MAP[c.status] || c.status,
  });
}

// Keep the page model stable while the API exposes explicit payment, coin and
// point quantities. The fallback only supports servers that predate amountCent.
function normalizeRechargeProduct(p) {
  if (!p) return p;
  const priceCent = p.amountCent != null ? p.amountCent : (p.priceCent != null ? p.priceCent : p.amount);
  const legacyBonus = p.bonusAmount || 0;
  const legacyCoins = Math.floor((p.amount || 0) / 100) + (p.assetType === 'coin' ? legacyBonus : 0);
  return Object.assign({}, p, {
    priceCent,
    coins: p.coinAmount != null ? p.coinAmount : (p.coins != null ? p.coins : legacyCoins),
    points: p.pointsAmount != null ? p.pointsAmount : (p.assetType === 'point' ? legacyBonus : 0),
  });
}

// GET /mini/food-orders/:id returns FoodOrderView {fulfillmentStatus,
// totalAmountCent,payMethod,items:{quantity,unitPriceCent}} (order/dto.go), but
// the detail page reads status/totalCent/payChannel + item qty/priceCent.
function normalizeFoodOrder(o) {
  if (!o) return o;
  return Object.assign({}, o, {
    status: o.status != null ? o.status : o.fulfillmentStatus,
    totalCent: o.totalCent != null ? o.totalCent : o.totalAmountCent,
    payChannel: o.payChannel != null ? o.payChannel : o.payMethod,
    orderNo: o.orderNo || o.businessOrderId,
    payCent: o.paidAmountCent != null ? o.paidAmountCent : o.totalAmountCent,
    refundCent: o.refundAmountCent || 0,
    storeName: o.storeName || '',
    tableText: o.tableName || '',
    items: (o.items || []).map((it) =>
      Object.assign({}, it, {
        qty: it.qty != null ? it.qty : it.quantity,
        priceCent: it.priceCent != null ? it.priceCent : it.unitPriceCent,
      })),
  });
}

// GET /mini/activity-orders/:id returns ActivityOrderView {status,ticketCount,
// totalAmountCent,payMethod,tickets:{ticketNo,status}} (order/dto.go); the
// verify code lives on the ticket (ticketNo), not a verifyCode field. Surface it
// from the first usable (paid/active) ticket and map amount/qty/pay.
function normalizeActivityOrder(o) {
  if (!o) return o;
  const tickets = o.tickets || [];
  const usable = tickets.find((t) => t.status === 'active') || tickets[0];
  return Object.assign({}, o, {
    qty: o.qty != null ? o.qty : o.ticketCount,
    amountCent: o.amountCent != null ? o.amountCent : o.totalAmountCent,
    payChannel: o.payChannel != null ? o.payChannel : o.payMethod,
    verifyCode: o.verifyCode != null ? o.verifyCode : usable && usable.ticketNo,
  });
}

// GET /mini/me returns MemberProfile {id,nickname,phone,inviteCode,status,
// vipTier?} (server/internal/modules/auth/dto.go). vipTier is the member's
// current VIP level: it exposes the SHORT user-facing label (label: "VIP1"..),
// level/threshold and the resolved icon/banner URLs (server-joined from
// QINIU_PUBLIC_DOMAIN, never a raw stored URL); it is omitted when the member is
// unranked. The home / profile cards render the member's VIP level, so this
// single seam maps vipTier's short label + banner onto the flat tier shape the
// pages read (tierShort/tierName/tierBannerUrl). It still falls back to the
// legacy flat tierCode/tierName or nested tier{code,name}, then VIP1
// when tier-less. Member copy shows only the short label, so tierName is set to
// the short label too (never the full admin name like "VIP1 普通会员"). The raw
// vipTier object passes straight through for any page that needs it. An empty
// (logged-out) member object is left as-is so no tier is fabricated for a guest.
const TIER_SHORT = { normal: 'VIP1', bronze: 'VIP2', silver: 'VIP3', gold: 'VIP4', platinum: 'VIP5', diamond: 'VIP6', star: 'VIP7', master: 'VIP8' };
const DEFAULT_TIER_NAME = 'VIP1';
function tierShortOf(code, name) {
  if (code && TIER_SHORT[code]) return TIER_SHORT[code];
  const match = name && name.match(/VIP\s*(\d+)/i);
  return match ? 'VIP' + match[1] : 'VIP1';
}
function normalizeMe(me) {
  if (!me || (!me.id && !me.nickname && !me.tierCode && !me.tierName && !me.tier && !me.vipTier)) return me;
  const vip = me.vipTier;
  const t = me.tier || {};
  const tierCode = me.tierCode || t.code || '';
  // Prefer the backend's short VIP label — the /mini/me contract intentionally
  // returns only the short label (e.g. "VIP1"), not the full admin tier name.
  const tierShort = (vip && vip.label) || tierShortOf(tierCode, me.tierName || t.name) || DEFAULT_TIER_NAME;
  // Member copy shows only the short label, never the full admin tier name.
  const tierName = tierShort;
  // Current-tier VIP banner (server-resolved QINIU_PUBLIC_DOMAIN + path); the
  // home page renders it after login when present.
  const tierBannerUrl = (vip && vip.bannerUrl) || '';
  return Object.assign({}, me, { tierCode, tierName, tierShort, tierBannerUrl });
}

// GET /mini/recharge-orders/:id returns RechargeOrderView {businessOrderNo,
// orderStatus,totalAmountCent} (order/dto.go), but the detail page reads
// orderNo/status/amountCent. (coins/bonusCoins/paidAt are not on this view.)
function normalizeRechargeOrder(o) {
  if (!o) return o;
  return Object.assign({}, o, {
    orderNo: o.orderNo != null ? o.orderNo : o.businessOrderNo,
    status: o.status != null ? o.status : o.orderStatus,
    amountCent: o.amountCent != null ? o.amountCent : o.totalAmountCent,
  });
}

// ---- request-body normalization (page shape -> server contract) ----
// Pages collect UI-friendly payloads (payChannel/qty/note/couponId, items with
// name+qty). The Go create handlers (server/internal/modules/order/dto.go,
// coupon/dto.go) expect payMethod/quantity/remark/entitlementId and items as
// {itemId,quantity}. Translate here — the single seam — so pages stay UI-shaped
// and the write succeeds on the real API. These are pure renames of data the
// page already supplied (no invented fields). The Go binder ignores extra keys.
function foodOrderBody(data) {
  const d = data || {};
  return Object.assign({}, d, {
    payMethod: d.payMethod || d.payChannel,
    remark: d.remark != null ? d.remark : d.note || '',
    items: (d.items || []).map((it) => ({
      itemId: it.itemId != null ? it.itemId : it.id,
      variantId: it.variantId,
      quantity: it.quantity != null ? it.quantity : it.qty,
    })),
  });
}

function activityOrderBody(data) {
  const d = data || {};
  return Object.assign({}, d, {
    quantity: d.quantity != null ? d.quantity : d.qty,
    payMethod: d.payMethod || d.payChannel,
  });
}

function rechargeOrderBody(data) {
  const d = data || {};
  return Object.assign({}, d, {
    payMethod: d.payMethod || d.payChannel,
  });
}

function couponRedemptionBody(data) {
  const d = data || {};
  return Object.assign({}, d, {
    entitlementId: d.entitlementId != null ? d.entitlementId : d.couponId,
    items: (d.items || []).map((item) => ({
      itemId: item.itemId != null ? item.itemId : item.id,
      quantity: item.quantity != null ? item.quantity : item.qty,
    })),
  });
}

// A seat reservation has no member-selected arrival time. The server records
// its creation time and keeps the seat occupied until cancellation or the
// daily 04:00 reset; `partySize` remains one seat by default.
function reservationBody(data) {
  const d = data || {};
  return Object.assign({}, d, {
    partySize: d.partySize != null ? d.partySize : 1,
    remark: d.remark != null ? d.remark : '',
  });
}

// ---- activity display normalization (server -> page) ----
// GET /mini/activities(/:id) returns {id,title,storeName,imageUrl,content,startAt,
// endAt,payChannels,purchaseLimit,status,ticketTypes} (activity/activity.go). The
// list/detail pages additionally read a preformatted `timeText`, a visual `tone`,
// and rich `detailHtml`. None are invented data — timeText is derived from startAt,
// tone is a stable per-activity theme (poker/coast/city/member), and detailHtml
// maps from the server `content`. imageUrl/storeName/purchaseLimit/ticket fields
// pass through untouched.
const ACT_TONES = ['poker', 'coast', 'city', 'member'];
function activityTone(id) {
  const n = Number(String(id == null ? 0 : id).replace(/\D/g, '')) || 0;
  return ACT_TONES[n % ACT_TONES.length];
}
// Server activity status is `published` (the mini List returns only published
// rows), but the pages speak the vocabulary in activity-detail's STATUS_LABEL
// (enrolling/upcoming/ended/soldout) — the list page even filters on
// `status === 'enrolling'`. Map the vocabulary here like normalizeCoupon does.
const ACTIVITY_STATUS_MAP = { published: 'enrolling' };
function activityMoment(input) {
  if (!input) return '';
  const d = new Date(input);
  if (isNaN(d.getTime())) return '';
  const p2 = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p2(d.getMonth() + 1)}-${p2(d.getDate())} ${p2(d.getHours())}:${p2(d.getMinutes())}`;
}
function activityTimeText(a) {
  if (a.timeText) return a.timeText;
  const start = activityMoment(a.startAt);
  const end = activityMoment(a.endAt);
  if (start && end) return `${start} 至 ${end}`;
  return start || end;
}
function normalizeActivityTicketType(ticket) {
  const t = ticket || {};
  const legacyStock = Number(t.stock);
  const unlimitedStock = t.unlimitedStock != null ? Boolean(t.unlimitedStock) : legacyStock < 0;
  const rawRemaining = t.remainingStock != null ? t.remainingStock : t.stock;
  const remainingStock = unlimitedStock ? null : Math.max(0, Number(rawRemaining) || 0);
  return Object.assign({}, t, { unlimitedStock, remainingStock });
}
function activityStatus(a) {
  if (a.status !== 'published') return ACTIVITY_STATUS_MAP[a.status] || a.status;
  const now = Date.now();
  const start = a.startAt ? new Date(a.startAt).getTime() : NaN;
  const end = a.endAt ? new Date(a.endAt).getTime() : NaN;
  if (!isNaN(end) && end < now) return 'ended';
  if (!isNaN(start) && start > now) return 'upcoming';
  const tickets = (a.ticketTypes || []).map(normalizeActivityTicketType);
  if (tickets.length && tickets.every((ticket) => !ticket.unlimitedStock && ticket.remainingStock === 0)) return 'soldout';
  return 'enrolling';
}
function normalizeActivity(a) {
  if (!a) return a;
  return Object.assign({}, a, {
    tone: a.tone || activityTone(a.id),
    timeText: activityTimeText(a),
    storeName: a.storeName || (a.scopeType === 'global' || a.storeId == null ? '全部门店' : ''),
    detailHtml: a.detailHtml || a.content || '',
    status: activityStatus(a),
    ticketTypes: (a.ticketTypes || []).map(normalizeActivityTicketType),
  });
}

// ---- membership benefits overview (server arrays -> benefits page object) ----
// The 会员权益 page (pages/benefits) reads an OBJECT {current(code), currentLevel,
// growthValue, growthMax, benefits:[{icon,title,desc}],
// levels:[{code,level,name,threshold,desc}]}, but the
// server returns a flat tier ARRAY (GET /mini/membership-tiers), the member's
// current tier on /mini/me, and growth on the /mini/wallet growth_value account.
// getBenefitsOverview() composes the object from that real data: tier level int ->
// art code, growth from wallet, growthMax from the next tier threshold, and the
// per-tier benefit text split out of the tier's `benefits` string.
const LEVEL_CODE = ['normal', 'bronze', 'silver', 'gold', 'platinum', 'diamond', 'star', 'master'];
function levelToCode(level) {
  return LEVEL_CODE[Math.max(0, Math.min(LEVEL_CODE.length - 1, (level || 1) - 1))];
}
const BENEFIT_ICONS = ['calendar', 'coupon', 'gift'];
function benefitItems(benefitsStr) {
  return String(benefitsStr || '')
    .split(/[\n;；/、,，]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map((title, i) => ({ icon: BENEFIT_ICONS[i % BENEFIT_ICONS.length], title, desc: '' }));
}

const api = {
  /* ---------- auth ---------- */
  wechatLogin: (data) => http.post(m('/auth/wechat/login'), data, { noAuth: true }),
  preRegister: (data) => http.post(m('/auth/wechat/pre-register'), data, { noAuth: true }),
  // First-time registration: authorized by the register ticket from wechatLogin,
  // NOT a session. This is what creates the member row.
  register: (data) => http.post(m('/auth/wechat/register'), data, { noAuth: true }),
  // Get masked phone number during registration (no auth required).
  getPhoneMask: (data) => http.post(m('/auth/wechat/phone-mask'), data, { noAuth: true }),
  // Upload the chosen avatar during registration (authorized by the register
  // ticket, not a session). Returns { avatarUrl } — the public https URL to
  // submit to register.
  uploadRegisterAvatar: (filePath, registerTicket) =>
    http.uploadFile(m('/auth/wechat/register-avatar'), filePath, { registerTicket }, { noAuth: true, name: 'file' }),
  logout: () => http.post(m('/auth/logout')),

  /* ---------- stores / home ---------- */
  getStores: (params) =>
    http.get(m('/stores') + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeStore), meta: res.meta })),
  getStore: (id) => http.get(m(`/stores/${id}`)).then((res) => ({ data: normalizeStore(res.data), meta: res.meta })),
  getBanners: (storeId) => http.get(m(`/stores/${storeId}/banners`)),
  getMembershipTiers: () => http.get(m('/membership-tiers')),
  // Benefits overview: composes the flat tier array + member current tier
  // (/mini/me) + wallet growth into the object shape the 会员权益 page reads.
  getBenefitsOverview: () =>
    Promise.all([http.get(m('/membership-tiers')), api.getMe(), http.get(m('/wallet'))]).then(
      ([tiersRes, meRes, walletRes]) => {
        const tiers = (tiersRes.data || []).slice().sort((a, b) => (a.level || 0) - (b.level || 0));
        const vip = (meRes.data || {}).vipTier || {};
        const wallet = normalizeWallet(walletRes.data);
        const curLevel = vip.level || (tiers[0] && tiers[0].level) || 1;
        const curTier = tiers.find((t) => t.id === vip.id) || tiers.find((t) => t.level === curLevel) || tiers[0] || {};
        const next = tiers.find((t) => (t.level || 0) > curLevel);
        return {
          data: {
            current: levelToCode(curLevel),
            currentLevel: curLevel,
            growthValue: wallet.growthValue || 0,
            growthMax: (next && next.threshold) || curTier.threshold || 1000,
            benefits: benefitItems(curTier.benefits),
            levels: tiers.map((t) => ({
              code: levelToCode(t.level),
              level: t.level,
              name: t.name,
              threshold: t.threshold || 0,
              desc: t.benefits || '',
            })),
          },
        };
      }
    ),
  getRechargeProducts: () =>
    http.get(m('/recharge-products')).then((res) => ({ data: (res.data || []).map(normalizeRechargeProduct), meta: res.meta })),
  getRankings: (params) => http.get(m('/rankings') + qs(params)),
  getFranchiseInquiryConfig: () => http.get(m('/franchise-inquiries/config'), { noAuth: true }),
  // The endpoint remains public, but a valid login token lets the server link
  // the consultation to the submitting member without trusting a client ID.
  createFranchiseInquiry: (data) => http.post(m('/franchise-inquiries'), data),

  /* ---------- member / wallet ---------- */
  getMe: () => http.get(m('/me')).then((res) => ({ data: normalizeMe(res.data), meta: res.meta })),
  updateMe: (data) => http.patch(m('/me'), data),
  bindPhone: (data) => http.post(m('/me/phone-bindings'), data, { idempotent: true }),
  getWallet: () => http.get(m('/wallet')).then((res) => ({ data: normalizeWallet(res.data), meta: res.meta })),
  getWalletLedger: (params) =>
    http.get(m('/wallet/ledger') + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeLedgerEntry), meta: res.meta })),
  getSignInStatus: () => http.get(m('/sign-ins/status')),
  signIn: () => http.post(m('/sign-ins'), {}, { idempotent: true }),
  getCoupons: (params) =>
    http.get(m('/coupons') + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeCoupon), meta: res.meta })),
  getTickets: (params) => http.get(m('/tickets') + qs(params)),
  getInvitations: () => http.get(m('/invitations')),
  getInvitationRewardConfig: () => http.get(m('/invitation-reward-config')),
  bindInvitation: (data) => {
    const d = data || {};
    return http.post(
      m('/invitations/bind'),
      { inviteCode: (d.inviteCode || d.code || '').trim() },
      { idempotent: true }
    );
  },

  // 存取积分 (fixed to current store; reviewed by staff)
  createPointSaving: (data, key) => {
    const input = data || {};
    const path = input.direction === 'withdraw' ? '/point-withdrawals' : '/point-savings';
    return http.post(
      m(path),
      { amount: Number(input.amount || input.points || 0), storeId: input.storeId },
      { idempotent: true, idempotencyKey: key }
    );
  },
  // 金币充值 (wechat only)
  createRechargeOrder: (data, key) =>
    http.post(m('/recharge-orders'), rechargeOrderBody(data), { idempotent: true, idempotencyKey: key }),
  getRechargeOrders: (params) => http.get(m('/recharge-orders') + qs(params)),
  getRechargeOrder: (id) =>
    http.get(m(`/recharge-orders/${id}`)).then((res) => ({ data: normalizeRechargeOrder(res.data), meta: res.meta })),

  /* ---------- catalog / ordering ---------- */
  getCategories: (storeId) => http.get(m(`/stores/${storeId}/catalog/categories`)),
  getItems: (storeId, params) =>
    http.get(m(`/stores/${storeId}/catalog/items`) + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeItem), meta: res.meta })),
  createFoodOrder: (data, key) =>
    http.post(m('/food-orders'), foodOrderBody(data), { idempotent: true, idempotencyKey: key }),
  getFoodOrders: (params) => http.get(m('/food-orders') + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeFoodOrder), meta: res.meta })),
  getFoodOrder: (id) =>
    http.get(m(`/food-orders/${id}`)).then((res) => ({ data: normalizeFoodOrder(res.data), meta: res.meta })),

  /* ---------- payment (wechat + coin only) ---------- */
  payWechatJsapi: (paymentOrderId, key) =>
    http.post(m(`/payment-orders/${paymentOrderId}/wechat-jsapi`), {}, { idempotent: true, idempotencyKey: key }),
  payByCoin: (paymentOrderId, key) =>
    http.post(m(`/payment-orders/${paymentOrderId}/pay-by-coin`), {}, { idempotent: true, idempotencyKey: key }),

  /* ---------- reservation ---------- */
  getTables: (storeId, params) => http.get(m(`/stores/${storeId}/tables`) + qs(params)),
  getSeats: (storeId, params) => http.get(m(`/stores/${storeId}/seats`) + qs(params)),
  getReservations: (params) => http.get(m('/reservations') + qs(params)),
  getReservation: (id) => http.get(m(`/reservations/${id}`)),
  createReservation: (data, key) =>
    http.post(m('/reservations'), reservationBody(data), { idempotent: true, idempotencyKey: key }),
  cancelReservation: (id, key) =>
    http.post(m(`/reservations/${id}/cancel`), {}, { idempotent: true, idempotencyKey: key }),
  joinWaitlist: (data, key) =>
    http.post(m('/waitlist-entries'), data, { idempotent: true, idempotencyKey: key }),

  /* ---------- activity ---------- */
  getActivities: (params) =>
    http.get(m('/activities') + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeActivity), meta: res.meta })),
  getStoreActivities: (storeId, params) =>
    http.get(m(`/stores/${storeId}/activities`) + qs(params)).then((res) => ({ data: (res.data || []).map(normalizeActivity), meta: res.meta })),
  getActivity: (id) => http.get(m(`/activities/${id}`)).then((res) => ({ data: normalizeActivity(res.data), meta: res.meta })),
  getTournamentEvents: (storeId) => http.get(m(`/stores/${storeId}/tournament-events`)),
  getTournamentEvent: (id) => http.get(m(`/tournament-events/${id}`)),
  createActivityOrder: (data, key) =>
    http.post(m('/activity-orders'), activityOrderBody(data), { idempotent: true, idempotencyKey: key }),
  getActivityOrders: (params) => http.get(m('/activity-orders') + qs(params)),
  getActivityOrder: (id) =>
    http.get(m(`/activity-orders/${id}`)).then((res) => ({ data: normalizeActivityOrder(res.data), meta: res.meta })),

  /* ---------- coupon redemption ---------- */
  getCouponRedeemableItems: (params) =>
    http.get(m('/coupon-redemptions/eligible-items') + qs(params)),
  createCouponRedemption: (data, key) =>
    http.post(m('/coupon-redemptions'), couponRedemptionBody(data), { idempotent: true, idempotencyKey: key }),
  getRedemptionOrders: (params) => http.get(m('/coupon-redemptions') + qs(params)),
  getRedemptionOrder: (id) => http.get(m(`/coupon-redemptions/${id}`)),

  /* ---------- staff (store scope, single bound store) ---------- */
  staff: {
    home: () => http.get(s('/home')),
    getPointSavings: (params) => http.get(s('/point-savings') + qs(params)),
    getPointSaving: (id) => http.get(s(`/point-savings/${id}`)),
    reviewPointSaving: (id, data, key) =>
      http.post(s(`/point-savings/${id}/review`), data, { idempotent: true, idempotencyKey: key }),
    getTodayActivities: () => http.get(s('/activities/today')),
    getTodayOperations: (params) => http.get(s('/operations/today') + qs(params)),
    verifyTicket: (data, key) =>
      http.post(s('/tickets/verify'), data, { idempotent: true, idempotencyKey: key }),
    getVerifications: (params) => http.get(s('/verifications') + qs(params)),
  },
};

module.exports = api;
