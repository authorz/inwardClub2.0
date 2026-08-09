// 会员权益 — 当前等级 / 成长值 / 本级权益 / 等级说明
// Reference: design/mini-program/final/member-subpages/12-member-benefits-v23.png
const api = require('../../services/api');

// 本级权益图标：本页稳定的 CSS 线性图标（bf__icon--*），不依赖背景大图
const BENEFIT_ICON = { calendar: 'bf__icon--calendar', coupon: 'bf__icon--coupon', gift: 'bf__icon--gift' };

// 等级 code -> banner 背景图，后端 code 不认识时回退 v1-normal
// 临时使用七牛静态 URL，后续由接口返回
const LEVEL_BANNER = {
  normal: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v1-normal.jpg',
  bronze: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v2-bronze.jpg',
  silver: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v3-silver.jpg',
  gold: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v4-gold.jpg',
  platinum: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v5-platinum.jpg',
  diamond: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v6-diamond.jpg',
  star: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v7-star.jpg',
  master: 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v8-master.jpg',
};
// 临时使用七牛静态 URL，后续由接口返回
const BANNER_FALLBACK = 'https://assets.inwardclub.com/mini-program/benefits/member-banners/v1-normal.jpg';

// 等级 code -> 等级图标，后端 code 不认识时回退 vip1-normal
const LEVEL_ICON = {
  normal: 'https://assets.inwardclub.com/public/level-icons/vip1-normal.png',
  bronze: 'https://assets.inwardclub.com/public/level-icons/vip2-bronze.png',
  silver: 'https://assets.inwardclub.com/public/level-icons/vip3-silver.png',
  gold: 'https://assets.inwardclub.com/public/level-icons/vip4-gold.png',
  platinum: 'https://assets.inwardclub.com/public/level-icons/vip5-platinum.png',
  diamond: 'https://assets.inwardclub.com/public/level-icons/vip6-diamond.png',
  star: 'https://assets.inwardclub.com/public/level-icons/vip7-star.png',
  master: 'https://assets.inwardclub.com/public/level-icons/vip8-master.png',
};
const LEVEL_ICON_FALLBACK = 'https://assets.inwardclub.com/public/level-icons/vip1-normal.png';

Page({
  data: { loading: true, tier: null },

  onLoad() {
    api
      .getBenefitsOverview()
      .then((res) => {
        const d = res.data || {};
        const levels = d.levels || [];
        const currentIdx = Math.max(0, levels.findIndex((l) => l.code === d.current));
        const growthValue = d.growthValue || 0;
        const growthMax = d.growthMax || 1000;
        const total = String(levels.length).padStart(2, '0');
        this.setData({
          loading: false,
          tier: {
            // swiper 初始定位到当前等级，用户左右滑动查看全部等级卡
            current: currentIdx,
            currentName: (levels[currentIdx] || {}).name || 'VIP1',
            growthValue,
            growthMax,
            growthPct: Math.min(100, Math.round((growthValue / growthMax) * 100)),
            // 已达成(unlocked) / 当前(current) / 待解锁(locked)，用于卡面文案
            cards: levels.map((l, i) => ({
              code: l.code,
              name: l.name,
              no: String(i + 1).padStart(2, '0'),
              total,
              banner: LEVEL_BANNER[l.code] || BANNER_FALLBACK,
              state: i === currentIdx ? 'current' : i < currentIdx ? 'unlocked' : 'locked',
            })),
            benefits: (d.benefits || []).map((b) => ({ icon: BENEFIT_ICON[b.icon] || 'bf__icon--gift', title: b.title, desc: b.desc })),
            levels: levels.map((l, i) => ({
              code: l.code,
              name: l.name,
              desc: l.desc,
              ver: 'VIP' + (i + 1),
              icon: LEVEL_ICON[l.code] || LEVEL_ICON_FALLBACK,
              current: l.code === d.current,
              state: i === currentIdx ? 'current' : i < currentIdx ? 'unlocked' : 'locked',
            })),
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },
});
