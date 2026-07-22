// Asset summary strip: 金币 / 积分 / 券 with big numbers.
// Reused by 我的首页 (tappable → recharge / point sheets) and 钱包流水 header.
const { amount } = require('../../utils/format');

Component({
  properties: {
    coins: { type: Number, value: 0 },
    points: { type: Number, value: 0 },
    coupons: { type: Number, value: 0 },
    tappable: { type: Boolean, value: false }, // enable tap events on 金币/积分
  },
  data: { coinsText: '0', pointsText: '0' },
  observers: {
    'coins,points'(c, p) {
      this.setData({ coinsText: amount(c), pointsText: amount(p) });
    },
  },
  methods: {
    tapCoin() {
      if (this.data.tappable) this.triggerEvent('asset', { asset: 'coins' });
    },
    tapPoint() {
      if (this.data.tappable) this.triggerEvent('asset', { asset: 'points' });
    },
    tapCoupon() {
      if (this.data.tappable) this.triggerEvent('asset', { asset: 'coupons' });
    },
  },
});
