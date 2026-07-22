/**
 * Mock backend — serves realistic fixtures for every `/api/v2/mini/*` and
 * `/api/v2/store/*` endpoint so the whole app runs in WeChat DevTools without
 * a live server. Enabled via config/env.js `useMock`.
 *
 * Fixtures use business-correct fields (integer cents, RFC3339 times, string
 * status codes) and deliberately do NOT copy sample typos / amounts from the
 * AI design mocks. Codes (核销码/兑换码/二维码 payloads) are produced here to
 * mirror "generated from interface data", not hardcoded UI placeholders.
 */
const ENV = require('../config/env');
const auth = require('../utils/auth');

/* ------------------------------------------------------------------ */
/* fixture data                                                        */
/* ------------------------------------------------------------------ */

const now = Date.now();
const iso = (offsetMs) => new Date(now + (offsetMs || 0)).toISOString();

const STORES = [
  { id: 'st_1', name: 'Inward Club 三里屯店', distanceMeters: 1200, open: true, hours: '10:00-02:00', address: '北京市朝阳区三里屯路 19 号', addressDetail: '太古里北区 N4-30', phone: '010-6417 0000', intro: '私享德州扑克与精酿酒吧，安静克制的会员空间。', lat: 39.937, lng: 116.454 },
  { id: 'st_2', name: 'Inward Club 国贸店', distanceMeters: 2800, open: true, hours: '10:00-02:00', address: '北京市朝阳区建国门外大街 1 号', addressDetail: '国贸商城 L3-12', phone: '010-6505 0000', intro: '商务会员店，含独立牌桌与会谈区。', lat: 39.909, lng: 116.457 },
  { id: 'st_3', name: 'Inward Club 望京店', distanceMeters: 5300, open: true, hours: '11:00-01:00', address: '北京市朝阳区阜通东大街 6 号', addressDetail: '方恒购物中心 L2-08', phone: '010-8470 0000', intro: '社区会员店，家庭友好时段与竞技牌局并行。', lat: 39.996, lng: 116.470 },
  { id: 'st_4', name: 'Inward Club 五道口店', distanceMeters: 7600, open: false, hours: '12:00-00:00', address: '北京市海淀区成府路 28 号', addressDetail: '优盛大厦 L1-05', phone: '010-8232 0000', intro: '校区周边会员店，轻食与积分局为主。', lat: 39.992, lng: 116.337 },
  { id: 'st_5', name: 'Inward Club 亦庄店', distanceMeters: 15400, open: true, hours: '10:00-23:00', address: '北京市大兴区荣华中路 8 号', addressDetail: '力宝广场 L1-18', phone: '010-6789 0000', intro: '大空间竞技场馆，含多张标准德州牌桌。', lat: 39.795, lng: 116.506 },
];

const BANNERS = [
  { id: 'bn_1', title: 'Inward Club', subtitle: 'BELONG TO YOUR BEST', imageUrl: '', tone: 'lounge' },
  { id: 'bn_2', title: '黑桃周赛', subtitle: 'WEEKLY TOURNAMENT', imageUrl: '', tone: 'poker' },
];

const ME = {
  id: 'mb_1',
  nickname: 'Authoz',
  avatarUrl: '',
  tierCode: 'normal',
  tierName: 'VIP1',
  // 对齐真实 GET /mini/me 契约：vipTier 提供短标 label 与服务端拼好的 bannerUrl（QINIU_PUBLIC_DOMAIN + path）
  vipTier: {
    id: 1,
    label: 'VIP1',
    level: 1,
    threshold: 0,
    bannerUrl: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v1-normal.jpg',
  },
  memberNo: '6272 0020 035',
  inviteCode: '8A7B3C',
  phone: '13800006272',
  boundInviter: null,
  growthValue: 320,
  growthMax: 1000,
  isStaff: true, // toggle to reveal 工作人员入口 in this demo build
};

// Backend GET /mini/wallet returns an array of accounts (assetType/
// availableAmount/heldAmount), not a flat {coins,points,...} object — see
// server/internal/modules/wallet/wallet.go Account struct. There is no
// couponCount field on the wallet resource; coupon counts must be derived
// from GET /mini/coupons.
const WALLET_ACCOUNTS = [
  { assetType: 'coins', availableAmount: 2680, heldAmount: 0 },
  { assetType: 'points', availableAmount: 5680, heldAmount: 0 },
  { assetType: 'cash_balance', availableAmount: 0, heldAmount: 0 },
  { assetType: 'growth_value', availableAmount: 320, heldAmount: 0 },
];
const EMPTY_WALLET_ACCOUNTS = [
  { assetType: 'coins', availableAmount: 0, heldAmount: 0 },
  { assetType: 'points', availableAmount: 0, heldAmount: 0 },
  { assetType: 'cash_balance', availableAmount: 0, heldAmount: 0 },
  { assetType: 'growth_value', availableAmount: 0, heldAmount: 0 },
];

// Backend ledger entries carry {id,assetType,direction,amount,balanceAfter,
// reason,sourceType,createdAt} (direction is 'credit'|'debit') — see
// server/internal/modules/wallet/wallet.go LedgerEntry. `reason`/`sourceType`
// are limited to what the backend actually writes today: order_payment,
// recharge, sign_in, admin_adjustment. Point-savings (存取积分) write to a
// separate point_savings table and never appear in this ledger, and there is
// no coupons/recharge assetType — so those two tabs have no backing data yet.
const LEDGER = {
  coins: [
    { id: 'l1', assetType: 'coins', direction: 'debit', amount: 56, balanceAfter: 2680, reason: 'order_payment', sourceType: 'payment_order', createdAt: iso(-3600 * 1000) },
    { id: 'l2', assetType: 'coins', direction: 'credit', amount: 200, balanceAfter: 2736, reason: 'admin_adjustment', sourceType: 'admin_adjustment', createdAt: iso(-2 * 86400 * 1000) },
    { id: 'l3', assetType: 'coins', direction: 'credit', amount: 3200, balanceAfter: 2536, reason: 'recharge', sourceType: 'recharge_order', createdAt: iso(-3 * 86400 * 1000) },
  ],
  points: [
    { id: 'p1', assetType: 'points', direction: 'credit', amount: 56, balanceAfter: 5680, reason: 'sign_in', sourceType: 'sign_in', createdAt: iso(-86400 * 1000) },
  ],
  coupons: [],
  recharge: [],
};

const CATEGORIES = [
  { id: 'cat_1', name: '酒水' },
  { id: 'cat_2', name: '软饮' },
  { id: 'cat_3', name: '小吃' },
  { id: 'cat_4', name: '果盘' },
  { id: 'cat_5', name: '套餐' },
];

const ITEMS = [
  { id: 'it_1', categoryId: 'cat_1', name: '经典啤酒', priceCent: 2800, stock: 50, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_2', categoryId: 'cat_2', name: '苏打水', priceCent: 1200, stock: 80, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_3', categoryId: 'cat_3', name: '花生米', priceCent: 1600, stock: 40, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_4', categoryId: 'cat_3', name: '香脆鸡块', priceCent: 2600, stock: 30, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_5', categoryId: 'cat_2', name: '柠檬茶', priceCent: 2200, stock: 60, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_6', categoryId: 'cat_2', name: '美式咖啡', priceCent: 2000, stock: 60, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_7', categoryId: 'cat_3', name: '薯条', priceCent: 1800, stock: 0, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_8', categoryId: 'cat_4', name: '果盘', priceCent: 3600, stock: 20, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_9', categoryId: 'cat_1', name: '精酿黑啤', priceCent: 3200, stock: 25, imageUrl: '', payChannels: ['wechat', 'coin'] },
  { id: 'it_10', categoryId: 'cat_5', name: '双人小聚套餐', priceCent: 8800, stock: 15, imageUrl: '', payChannels: ['wechat', 'coin'] },
];

// 预约用户占位（无头像资源时用昵称首字/缩写代替头像）
const SEAT_USERS = [
  { nickname: '陈一鸣', avatarUrl: '', gender: 'male' },
  { nickname: '林晓', avatarUrl: '', gender: 'female' },
  { nickname: 'Alex', avatarUrl: '', gender: 'male' },
  { nickname: '苏蔓', avatarUrl: '', gender: 'female' },
  { nickname: '王大锤', avatarUrl: '', gender: 'male' },
  { nickname: 'Nina', avatarUrl: '', gender: 'female' },
  { nickname: '赵子龙', avatarUrl: '', gender: 'male' },
  { nickname: '周然', avatarUrl: '', gender: 'female' },
];

function buildSeats(reservedCount) {
  const seats = [];
  for (let i = 1; i <= 9; i += 1) {
    if (i <= reservedCount) {
      seats.push({ no: i, state: 'reserved', user: SEAT_USERS[(i - 1) % SEAT_USERS.length] });
    } else {
      seats.push({ no: i, state: 'free' });
    }
  }
  return seats;
}

const TABLES = [
  { id: 'tb_1', name: '1号桌', type: '德州扑克桌', capacity: 9, reserved: 5, updatedAt: iso(-15 * 60 * 1000), status: 'active', seats: buildSeats(5) },
  { id: 'tb_2', name: '2号桌', type: '德州扑克桌', capacity: 9, reserved: 2, updatedAt: iso(-25 * 60 * 1000), status: 'active', seats: buildSeats(2) },
  { id: 'tb_3', name: '3号桌', type: '德州扑克桌', capacity: 9, reserved: 8, updatedAt: iso(-8 * 60 * 1000), status: 'active', seats: buildSeats(8) },
];

const ACTIVITIES = [
  {
    id: 'ac_1', title: '黑桃周赛', tone: 'poker', imageUrl: '', status: 'enrolling',
    timeText: '本周六 20:00', storeName: 'Inward Club 三里屯店', distanceMeters: 1200,
    saleEndAt: iso(2 * 86400 * 1000), startAt: iso(3 * 86400 * 1000),
    intro: '每周六晚固定开赛，德州扑克竞技赛制，专业裁判执裁，公平透明。冠军荣耀与丰厚奖励等你来赢。',
    detailHtml: '<p>每周六晚固定开赛，<strong>德州扑克</strong>竞技赛制，专业裁判执裁，公平透明。</p><ul><li>标准锦标赛结构</li><li>专业发牌与计时</li><li>冠军荣耀与丰厚奖励</li></ul>',
    purchaseLimit: 4,
    ticketTypes: [
      { id: 'tt_1', name: '早鸟票', priceCent: 12800, stock: 36, saleEndAt: iso(1 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
      { id: 'tt_2', name: '预售票', priceCent: 15800, stock: 89, saleEndAt: iso(4 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
      { id: 'tt_3', name: '双人票', priceCent: 24800, stock: 18, saleEndAt: iso(4 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
    ],
  },
  {
    id: 'ac_2', title: '海岸挑战赛', tone: 'coast', imageUrl: '', status: 'enrolling',
    timeText: '本周日 19:00', storeName: 'Inward Club 国贸店', distanceMeters: 2800,
    saleEndAt: iso(3 * 86400 * 1000), startAt: iso(4 * 86400 * 1000),
    intro: '海景主题德州扑克友谊赛，轻松竞技，适合进阶会员练手。',
    purchaseLimit: 4,
    ticketTypes: [
      { id: 'tt_4', name: '标准票', priceCent: 9800, stock: 50, saleEndAt: iso(3 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
    ],
  },
  {
    id: 'ac_3', title: '城市夜赛', tone: 'city', imageUrl: '', status: 'enrolling',
    timeText: '7-25 20:00', storeName: 'Inward Club 望京店', distanceMeters: 5300,
    saleEndAt: iso(6 * 86400 * 1000), startAt: iso(8 * 86400 * 1000),
    intro: '城市夜景牌局，限量席位，含专属饮品。',
    purchaseLimit: 4,
    ticketTypes: [
      { id: 'tt_5', name: '标准票', priceCent: 11800, stock: 40, saleEndAt: iso(6 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
      { id: 'tt_6', name: 'VIP票', priceCent: 19800, stock: 12, saleEndAt: iso(6 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
    ],
  },
  {
    id: 'ac_4', title: '会员私享局', tone: 'member', imageUrl: '', status: 'enrolling',
    timeText: '7-28 20:00', storeName: 'Inward Club 三里屯店', distanceMeters: 1200,
    saleEndAt: iso(9 * 86400 * 1000), startAt: iso(12 * 86400 * 1000),
    intro: '仅限会员参与的私享牌局，含侍酒服务。',
    purchaseLimit: 2,
    ticketTypes: [
      { id: 'tt_7', name: '双人票', priceCent: 30800, stock: 8, saleEndAt: iso(9 * 86400 * 1000), payChannels: ['wechat', 'coin'] },
    ],
  },
];

const HISTORY_ACTIVITIES = [
  { id: 'ah_1', title: '五月月赛', timeText: '05-31 20:00', storeName: 'Inward Club 三里屯店', status: 'ended' },
  { id: 'ah_2', title: '春季邀请赛', timeText: '04-20 19:30', storeName: 'Inward Club 国贸店', status: 'ended' },
];

const FOOD_ORDERS = [
  { id: 'fo_1', orderNo: 'F2026071501', storeName: 'Inward Club 三里屯店', tableText: '1号桌 · 5号座位', totalCent: 5600, discountCent: 800, payCent: 4800, status: 'completed', payChannel: 'wechat+coin', createdAt: iso(-3600 * 1000), items: [{ name: '经典啤酒', qty: 2, priceCent: 3600, imageUrl: '' }, { name: '苏打水', qty: 1, priceCent: 800, imageUrl: '' }, { name: '花生米', qty: 1, priceCent: 1200, imageUrl: '' }] },
  { id: 'fo_2', orderNo: 'F2026071408', storeName: 'Inward Club 三里屯店', tableText: '1号桌 · 5号座位', totalCent: 8800, discountCent: 0, payCent: 8800, status: 'preparing', payChannel: 'wechat', createdAt: iso(-2 * 3600 * 1000), items: [{ name: '双人小聚套餐', qty: 1, priceCent: 8800, imageUrl: '' }] },
];

const ACTIVITY_ORDERS = [
  { id: 'ao_1', orderNo: 'A2026071506', activityId: 'ac_1', activityTitle: '黑桃周赛', timeText: '本周六 19:30', storeName: 'Inward Club 三里屯店', ticketName: '标准票', qty: 1, payChannel: 'wechat', amountCent: 12800, status: 'pending_verify', createdAt: iso(-3 * 3600 * 1000), verifyCode: '8392 6174' },
  { id: 'ao_2', orderNo: 'A2026070903', activityId: 'ac_3', activityTitle: '城市夜赛', timeText: '07-25 20:00', storeName: 'Inward Club 望京店', ticketName: 'VIP票', qty: 1, payChannel: 'coin', amountCent: 19800, status: 'completed', createdAt: iso(-6 * 86400 * 1000), verifyCode: '2231 8890' },
];

const RECHARGE_ORDERS = [
  { id: 'ro_1', orderNo: 'R2026071502', amountCent: 50000, coins: 5200, bonusCoins: 200, payChannel: 'wechat', status: 'completed', paidAt: iso(-5 * 3600 * 1000), creditedAt: iso(-5 * 3600 * 1000), note: '充值到账后不可直接提现' },
];

const REDEMPTION_ORDERS = [
  { id: 'eo_1', orderNo: 'E2026071503', title: '香槟券兑换', couponName: '消耗券', qty: 1, validUntil: '2026-07-22', code: '5268 9031', storeName: 'Inward Club 三里屯店', createdAt: iso(-90 * 60 * 1000), status: 'pending_verify' },
];

const COUPONS = [
  { id: 'cp_1', name: '酒水券', desc: '可兑换酒水 / 软饮商品', type: 'drink', amountCent: 19900, validUntil: '2026-07-22', status: 'unused', action: 'order' },
  { id: 'cp_2', name: '餐食券', desc: '可兑换套餐 / 果盘商品', type: 'food', amountCent: 12800, validUntil: '2026-08-01', status: 'unused', action: 'order' },
  { id: 'cp_3', name: '小吃券', desc: '可兑换小吃商品', type: 'snack', amountCent: 5800, validUntil: '2026-07-30', status: 'unused', action: 'order' },
  { id: 'cp_4', name: '酒水券', desc: '可兑换酒水 / 软饮商品', type: 'drink', amountCent: 19900, validUntil: '2026-05-01', status: 'expired', action: 'none' },
  { id: 'cp_5', name: '餐食券', desc: '可兑换套餐 / 果盘商品', type: 'food', amountCent: 12800, validUntil: '2026-06-10', usedAt: '2026-06-08', status: 'used', action: 'none' },
];

const TICKETS = [
  { id: 'tk_1', activityId: 'ac_1', title: '黑桃周赛', tone: 'poker', imageUrl: '', timeText: '本周六 19:30', storeName: 'Inward Club 三里屯店', ticketName: '标准票', qty: 1, status: 'pending_verify', code: '8392 6174', orderId: 'ao_1' },
  { id: 'tk_2', activityId: 'ac_4', title: '会员私享局', tone: 'member', imageUrl: '', timeText: '07-28 20:00', storeName: 'Inward Club 三里屯店', ticketName: '双人票', qty: 1, status: 'unused', code: '5561 2200', orderId: 'ao_3' },
  { id: 'tk_3', activityId: 'ac_3', title: '城市夜赛', tone: 'city', imageUrl: '', timeText: '07-25 20:00', storeName: 'Inward Club 望京店', ticketName: 'VIP票', qty: 1, status: 'used', code: '2231 8890', orderId: 'ao_2' },
];

const RECHARGE_PRODUCTS = [
  { id: 'rp_1', priceCent: 5000, coins: 500, bonusCoins: 0 },
  { id: 'rp_2', priceCent: 10000, coins: 1000, bonusCoins: 50 },
  { id: 'rp_3', priceCent: 20000, coins: 2000, bonusCoins: 150 },
  { id: 'rp_4', priceCent: 30000, coins: 3000, bonusCoins: 300 },
  { id: 'rp_5', priceCent: 50000, coins: 5000, bonusCoins: 600 },
  { id: 'rp_6', priceCent: 100000, coins: 10000, bonusCoins: 1500 },
];

const INVITATIONS = {
  inviteCode: '8A7B3C',
  invited: 12,
  effective: 8,
  pending: 2,
  rules: [
    { icon: 'gift', title: '邀请奖励', desc: '按后台规则发放' },
    { icon: 'doc', title: '发放方式', desc: '按后台规则发放' },
    { icon: 'shield', title: '结算条件', desc: '仅在有效消费后结算' },
  ],
  records: [
    { id: 'iv_1', name: 'Luna', status: 'effective', date: '2026-07-12' },
    { id: 'iv_2', name: 'Ken', status: 'pending', date: '2026-07-10' },
    { id: 'iv_3', name: 'Echo', status: 'effective', date: '2026-07-05' },
  ],
};

const RANKINGS = {
  myRank: 12,
  myScore: 5680,
  top: [
    { rank: 1, name: 'Mason', score: 12800, avatarUrl: '' },
    { rank: 2, name: 'Authoz', score: 5680, avatarUrl: '' },
    { rank: 3, name: 'Luna', score: 5300, avatarUrl: '' },
    { rank: 4, name: 'Echo', score: 4980, avatarUrl: '' },
    { rank: 5, name: 'Ray', score: 4720, avatarUrl: '' },
    { rank: 6, name: 'Kevin', score: 4310, avatarUrl: '' },
    { rank: 7, name: 'Zoe', score: 3890, avatarUrl: '' },
    { rank: 8, name: 'Jaden', score: 3520, avatarUrl: '' },
    { rank: 9, name: 'Vivi', score: 3180, avatarUrl: '' },
    { rank: 10, name: 'Chen', score: 2950, avatarUrl: '' },
  ],
  updatedText: '榜单每日 02:00 更新',
};

const TIERS = {
  current: 'normal',
  growthValue: 320,
  growthMax: 1000,
  benefits: [
    { icon: 'calendar', title: '活动优先报名', desc: '按后台规则发放' },
    { icon: 'coupon', title: '专属券包', desc: '按后台规则发放' },
    { icon: 'gift', title: '生日权益', desc: '按后台规则发放' },
  ],
  levels: [
    { code: 'normal', name: 'VIP1', desc: '入会即享，解锁基础权益' },
    { code: 'bronze', name: 'VIP2', desc: '成长值达标升级，权益进阶' },
    { code: 'silver', name: 'VIP3', desc: '成长值达标升级，权益进阶' },
    { code: 'gold', name: 'VIP4', desc: '成长值达标升级，权益进阶' },
    { code: 'platinum', name: 'VIP5', desc: '成长值达标升级，权益进阶' },
    { code: 'diamond', name: 'VIP6', desc: '成长值达标升级，权益进阶' },
    { code: 'star', name: 'VIP7', desc: '成长值达标升级，权益进阶' },
    { code: 'master', name: 'VIP8', desc: '至臻等级，享全部会员权益' },
  ],
};

/* ---- staff fixtures (single bound store, no switch) ---- */
const STAFF_STORE = { id: 'st_1', name: 'Inward Club 三里屯店', bound: true };

const STAFF_HOME = {
  store: STAFF_STORE,
  pendingReview: 6,
  todayVerifications: 18,
  todayActivity: { title: '黑桃周赛', timeText: '19:30' },
};

const POINT_SAVINGS = [
  { id: 'ps_1', memberName: 'Luna', phone: '13800001111', direction: 'deposit', points: 1000, storeName: 'Inward Club 三里屯店', note: '会员现场存入', status: 'pending', createdAt: iso(-30 * 60 * 1000) },
  { id: 'ps_2', memberName: 'Ken', phone: '13900002222', direction: 'withdraw', points: 500, storeName: 'Inward Club 三里屯店', note: '', status: 'pending', createdAt: iso(-55 * 60 * 1000) },
  { id: 'ps_3', memberName: 'Echo', phone: '13700003333', direction: 'deposit', points: 2000, storeName: 'Inward Club 三里屯店', note: '活动奖励存入', status: 'approved', createdAt: iso(-3 * 3600 * 1000) },
];

const TODAY_ACTIVITIES = [
  { id: 'ac_1', title: '黑桃周赛', timeText: '2026-07-14 19:30 - 23:59', storeName: 'Inward Club 三里屯店', pendingVerify: 128, verified: 86 },
];

const VERIFICATIONS = [
  { id: 'vf_1', code: '8812 7098 3312', result: 'success', at: iso(-20 * 60 * 1000), memberName: 'Vivi', activityTitle: '黑桃周赛' },
  { id: 'vf_2', code: '7731 9920 1182', result: 'success', at: iso(-32 * 60 * 1000), memberName: 'Chen', activityTitle: '海岸挑战赛' },
  { id: 'vf_3', code: '9921 3308 6621', result: 'failed', at: iso(-40 * 60 * 1000), memberName: 'Ray', activityTitle: '城市夜赛' },
];

/* ------------------------------------------------------------------ */
/* router                                                              */
/* ------------------------------------------------------------------ */

function paginate(list, opts) {
  const q = (opts && opts.query) || {};
  const page = Number(q.page || 1);
  const pageSize = Number(q.pageSize || 20);
  const start = (page - 1) * pageSize;
  return { data: list.slice(start, start + pageSize), meta: { page, pageSize, total: list.length } };
}

function parse(path) {
  const idx = path.indexOf('?');
  const clean = idx >= 0 ? path.slice(0, idx) : path;
  const query = {};
  if (idx >= 0) {
    path.slice(idx + 1).split('&').forEach((kv) => {
      const [k, v] = kv.split('=');
      if (k) query[decodeURIComponent(k)] = decodeURIComponent(v || '');
    });
  }
  return { clean, query };
}

/**
 * Route table: [httpMethod, pathPattern, handler].
 * Typed as a tuple array so `re` is a RegExp and `fn` is callable.
 * @type {Array<[string, RegExp, (m: RegExpExecArray, o: { data: any, query: any }) => { data: any, meta?: any }]>}
 */
const routes = [
  // ---- auth ----
  // Accepts the profile payload the page collects (userInfo/rawData/signature/
  // encryptedData/iv) and mirrors the real WeChat nickname/avatar into the mock
  // member so the card shows the actual account after login.
  ['POST', /^\/mini\/auth\/wechat\/login$/, (m, o) => {
    const info = (o.data && o.data.userInfo) || {};
    if (info.nickName) ME.nickname = info.nickName;
    if (info.avatarUrl) ME.avatarUrl = info.avatarUrl;
    if (info.gender) ME.gender = info.gender;
    // Mirror the real auth.LoginResponse: { token: TokenPair, profile }
    // (server/internal/modules/auth/dto.go). subjectType/storeId are NOT part
    // of the backend contract — the mini login carries no staff/store hint (see
    // the API gap report) — so they are demo-only extras kept here to keep the
    // 工作人员入口 reachable in mock mode.
    return {
      data: {
        token: { accessToken: 'mock-access', refreshToken: 'mock-refresh', expiresAt: iso(3600 * 1000), tokenType: 'Bearer' },
        profile: { id: ME.id, nickname: ME.nickname, avatarUrl: ME.avatarUrl || '', phone: ME.phone, inviteCode: ME.inviteCode, status: 'active' },
        subjectType: ME.isStaff ? 'staff' : 'member',
        storeId: 'st_1',
      },
    };
  }],
  ['POST', /^\/mini\/auth\/logout$/, () => ({ data: {} })],

  // ---- stores / home ----
  ['GET', /^\/mini\/stores$/, (m, o) => paginate(STORES, o)],
  ['GET', /^\/mini\/stores\/([^/]+)\/banners$/, () => ({ data: BANNERS })],
  ['GET', /^\/mini\/stores\/([^/]+)\/catalog\/categories$/, () => ({ data: CATEGORIES })],
  ['GET', /^\/mini\/stores\/([^/]+)\/catalog\/items$/, () => ({ data: ITEMS })],
  ['GET', /^\/mini\/stores\/([^/]+)\/tables$/, () => ({ data: TABLES })],
  ['GET', /^\/mini\/stores\/([^/]+)\/seats$/, () => ({ data: [] })],
  ['GET', /^\/mini\/stores\/([^/]+)\/activities$/, () => ({ data: ACTIVITIES })],
  ['GET', /^\/mini\/stores\/([^/]+)$/, (m) => ({ data: STORES.find((s) => s.id === m[1]) || STORES[0] })],

  ['GET', /^\/mini\/membership-tiers$/, () => ({ data: TIERS })],
  ['GET', /^\/mini\/recharge-products$/, () => ({ data: RECHARGE_PRODUCTS })],
  ['GET', /^\/mini\/rankings$/, () => ({ data: RANKINGS })],

  // ---- member / wallet ----
  // Member profile / wallet are only meaningful once authenticated. Before
  // login (cleared storage) the mock returns a guest shape so no fake member
  // data (nickname/会员号/等级/资产) ever shows up — it appears only after the
  // user taps 登录 and the wechatLogin route above seeds the session.
  ['GET', /^\/mini\/me$/, () => ({ data: auth.isLoggedIn() ? ME : {} })],
  ['PATCH', /^\/mini\/me$/, (m, o) => ({ data: Object.assign({}, ME, o.data) })],
  ['POST', /^\/mini\/me\/phone-bindings$/, () => ({ data: { phoneMasked: '138****6272' } })],
  ['GET', /^\/mini\/wallet$/, () => ({ data: auth.isLoggedIn() ? WALLET_ACCOUNTS : EMPTY_WALLET_ACCOUNTS })],
  ['GET', /^\/mini\/wallet\/ledger$/, (m, o) => {
    const q = (o && o.query) || {};
    const asset = q.assetType || q.asset || 'coins';
    return paginate(LEDGER[asset] || [], o);
  }],
  ['GET', /^\/mini\/coupons$/, () => ({ data: COUPONS })],
  ['GET', /^\/mini\/invitations$/, () => ({ data: INVITATIONS })],
  ['POST', /^\/mini\/invitations\/bind$/, () => ({ data: { bound: true } })],

  ['POST', /^\/mini\/point-savings$/, (m, o) => ({ data: { id: 'ps_new', status: 'pending', direction: (o.data && o.data.direction) || 'deposit', points: (o.data && o.data.points) || 0 } })],
  ['POST', /^\/mini\/recharge-orders$/, (m, o) => ({ data: { id: 'ro_new', orderNo: 'R' + Date.now(), status: 'pending', paymentOrderId: 'po_recharge', amountCent: (o.data && o.data.amountCent) || 0 } })],
  ['GET', /^\/mini\/recharge-orders$/, (m, o) => paginate(RECHARGE_ORDERS, o)],
  ['GET', /^\/mini\/recharge-orders\/([^/]+)$/, (m) => ({ data: RECHARGE_ORDERS.find((r) => r.id === m[1]) || RECHARGE_ORDERS[0] })],

  // ---- ordering ----
  ['POST', /^\/mini\/food-orders$/, (m, o) => ({ data: { id: 'fo_new', orderNo: 'F' + Date.now(), status: 'pending', paymentOrderId: 'po_food', totalCent: (o.data && o.data.totalCent) || 0 } })],
  ['GET', /^\/mini\/food-orders$/, (m, o) => paginate(FOOD_ORDERS, o)],
  ['GET', /^\/mini\/food-orders\/([^/]+)$/, (m) => ({ data: FOOD_ORDERS.find((f) => f.id === m[1]) || FOOD_ORDERS[0] })],

  // ---- payment ----
  // Shapes match order.WeChatJSAPIResponse / payment.WeChatPrepay exactly.
  // Note: there is no GET /mini/payment-orders(/:id) on the backend — payment
  // order status is only observable via the owning food/recharge/activity
  // order (paymentOrderId/paymentOrderNo/payMethod fields on those views).
  ['POST', /^\/mini\/payment-orders\/([^/]+)\/wechat-jsapi$/, (m) => ({
    data: {
      paymentOrderId: m[1],
      prepay: { prepayId: 'mock_prepay', nonceStr: 'n', package: 'prepay_id=mock_prepay', signType: 'RSA', paySign: 'sig', timeStamp: String(Math.floor(now / 1000)) },
    },
  })],
  // Backend returns no body for pay-by-coin (204 No Content).
  ['POST', /^\/mini\/payment-orders\/([^/]+)\/pay-by-coin$/, () => ({ data: null })],

  // ---- reservation ----
  ['GET', /^\/mini\/reservations$/, (m, o) => paginate([
    { id: 'rv_1', tableName: '1号桌', seatNo: 5, storeName: 'Inward Club 三里屯店', timeText: '今天 20:00', status: 'active', createdAt: iso(-30 * 60 * 1000) },
    { id: 'rv_2', tableName: '2号桌', seatNo: 3, storeName: 'Inward Club 三里屯店', timeText: '07-10 21:00', status: 'completed', createdAt: iso(-4 * 86400 * 1000) },
  ], o)],
  ['GET', /^\/mini\/reservations\/([^/]+)$/, (m) => ({ data: { id: m[1], tableName: '1号桌', seatNo: 5, storeName: 'Inward Club 三里屯店', timeText: '今天 20:00', status: 'active', note: '请按预约时间到店，超时将释放座位', createdAt: iso(-30 * 60 * 1000) } })],
  ['POST', /^\/mini\/reservations$/, (m, o) => ({ data: { id: 'rv_new', status: 'active', tableName: (o.data && o.data.tableName) || '1号桌', seatNo: (o.data && o.data.seatNo) || 1 } })],
  ['POST', /^\/mini\/reservations\/([^/]+)\/cancel$/, () => ({ data: { status: 'cancelled' } })],
  ['POST', /^\/mini\/waitlist-entries$/, () => ({ data: { id: 'wl_new', position: 3, status: 'waiting' } })],

  // ---- activity ----
  ['GET', /^\/mini\/activities$/, (m, o) => {
    if (o.query && o.query.scope === 'history') return { data: HISTORY_ACTIVITIES };
    return { data: ACTIVITIES };
  }],
  ['GET', /^\/mini\/activities\/([^/]+)$/, (m) => ({ data: ACTIVITIES.find((a) => a.id === m[1]) || ACTIVITIES[0] })],
  ['POST', /^\/mini\/activity-orders$/, (m, o) => ({ data: { id: 'ao_new', orderNo: 'A' + Date.now(), status: 'pending', paymentOrderId: 'po_activity', amountCent: (o.data && o.data.amountCent) || 0 } })],
  ['GET', /^\/mini\/activity-orders$/, (m, o) => paginate(ACTIVITY_ORDERS, o)],
  ['GET', /^\/mini\/activity-orders\/([^/]+)$/, (m) => ({ data: ACTIVITY_ORDERS.find((a) => a.id === m[1]) || ACTIVITY_ORDERS[0] })],

  // ---- coupon redemption ----
  ['POST', /^\/mini\/coupon-redemptions$/, (m, o) => {
    const body = o.data || {};
    const items = body.items || [];
    const coupon = COUPONS.find((c) => c.id === body.couponId) || { name: '', validUntil: '' };
    const store = STORES.find((s) => s.id === body.storeId);
    const totalQty = items.reduce((n, it) => n + (it.qty || 0), 0);
    const firstName = items[0] ? items[0].name : '兑换商品';
    const id = 'eo_' + Date.now();
    const order = {
      id,
      orderNo: 'E' + Date.now(),
      title: items.length > 1 ? `${firstName} 等 ${items.length} 件` : firstName,
      couponName: coupon.name || '兑换券',
      items: items.map((it) => ({ name: it.name, qty: it.qty })),
      qty: totalQty,
      validUntil: coupon.validUntil || '',
      code: String(1000 + Math.floor(Math.random() * 9000)) + ' ' + String(1000 + Math.floor(Math.random() * 9000)),
      storeName: store ? store.name : '',
      createdAt: iso(0),
      status: 'pending_verify',
    };
    REDEMPTION_ORDERS.unshift(order);
    return { data: order };
  }],
  ['GET', /^\/mini\/coupon-redemptions\/([^/]+)$/, (m) => ({ data: REDEMPTION_ORDERS.find((r) => r.id === m[1]) || REDEMPTION_ORDERS[0] })],
  ['GET', /^\/mini\/coupon-redemptions$/, (m, o) => paginate(REDEMPTION_ORDERS, o)],

  // ---- tickets (my tickets) - served via coupons-like endpoint reuse ----
  ['GET', /^\/mini\/tickets$/, (m, o) => ({ data: TICKETS })],

  // ---- staff ----
  ['GET', /^\/store\/staff\/home$/, () => ({ data: STAFF_HOME })],
  ['GET', /^\/store\/point-savings$/, (m, o) => paginate(POINT_SAVINGS, o)],
  ['GET', /^\/store\/point-savings\/([^/]+)$/, (m) => ({ data: POINT_SAVINGS.find((p) => p.id === m[1]) || POINT_SAVINGS[0] })],
  ['POST', /^\/store\/point-savings\/([^/]+)\/review$/, (m, o) => ({ data: { id: m[1], status: (o.data && o.data.decision) === 'reject' ? 'rejected' : 'approved' } })],
  ['GET', /^\/store\/activities\/today$/, () => ({ data: TODAY_ACTIVITIES })],
  ['POST', /^\/store\/tickets\/verify$/, (m, o) => {
    const code = (o.data && o.data.code) || '';
    const ok = /\d/.test(code) && code.replace(/\s/g, '').length >= 8;
    return { data: { result: ok ? 'success' : 'failed', code, memberName: ok ? 'Vivi' : '', message: ok ? '核销成功' : '核销码无效或已使用' } };
  }],
  ['GET', /^\/store\/verifications$/, (m, o) => paginate(VERIFICATIONS, o)],
];

function handle(method, path, options) {
  const opts = options || {};
  const { clean, query } = parse(path);
  const ctx = { data: opts.data, query };
  for (const [rm, re, fn] of routes) {
    if (rm !== method) continue;
    const match = re.exec(clean);
    if (match) {
      const result = fn(match, ctx);
      return new Promise((resolve) => {
        setTimeout(() => resolve(result || { data: null }), ENV.mockLatency);
      });
    }
  }
  return Promise.reject({ isApiError: true, code: 'MOCK_NOT_FOUND', message: `未配置的接口: ${method} ${clean}` });
}

module.exports = { handle };
