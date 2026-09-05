// Custom tab bar — the reusable bottom navigation (首页/预约/点餐/我的).
// Uses the design-frozen tab icons from assets/tab (converted from
// design/mini-program/tab-icons SVGs). Tab pages set `selected` in onShow.
const memberAccess = require('../utils/member-access');

Component({
  data: {
    selected: 0,
    hidden: false,
    list: [
      { pagePath: '/pages/index/index', text: '首页', icon: '/assets/tab/home.png', active: '/assets/tab/home-active.png' },
      { pagePath: '/pages/reservation/reservation', text: '预约', icon: '/assets/tab/reservation.png', active: '/assets/tab/reservation-active.png' },
      { pagePath: '/pages/order/order', text: '点餐', icon: '/assets/tab/order.png', active: '/assets/tab/order-active.png' },
      { pagePath: '/pages/home/home', text: '我的', icon: '/assets/tab/me.png', active: '/assets/tab/me-active.png' },
    ],
  },
  methods: {
    onTap(e) {
      const index = Number(e.currentTarget.dataset.index);
      const path = this.data.list[index].pagePath;
      if (index > 0 && index !== 2) {
        memberAccess.requireCompleteProfile(() => wx.switchTab({ url: path }));
        return;
      }
      wx.switchTab({ url: path });
    },
  },
});
