// 我的入场券 — 待使用/已使用/已过期分段
// Reference: design/mini-program/final/member-subpages/09-my-tickets-v23.png
const api = require('../../services/api');
const { TICKET_STATUS_LABEL } = require('../../constants/index');

// bucket lifecycle statuses into the three visible tabs
function bucketOf(status) {
  if (status === 'used') return 'used';
  if (status === 'expired' || status === 'refunded') return 'expired';
  return 'pending'; // unused | pending_verify
}

Page({
  data: {
    loading: true,
    statusOptions: [
      { label: '待使用', value: 'pending' },
      { label: '已使用', value: 'used' },
      { label: '已过期', value: 'expired' },
    ],
    bucket: 'pending',
    all: [],
    list: [],
  },

  onLoad() {
    api
      .getTickets()
      .then((res) => {
        const all = (res.data || []).map((t) => ({
          id: t.id,
          activityId: t.activityId,
          title: t.title,
          tone: t.tone,
          imageUrl: t.imageUrl || '',
          timeText: t.timeText,
          storeName: t.storeName,
          ticketName: t.ticketName,
          qty: t.qty || 1,
          status: t.status,
          statusLabel: TICKET_STATUS_LABEL[t.status] || t.status,
          code: t.code,
          bucket: bucketOf(t.status),
          usable: bucketOf(t.status) === 'pending',
        }));
        this.setData({ all, loading: false });
        this.applyFilter();
      })
      .catch(() => this.setData({ loading: false }));
  },

  onStatusChange(e) {
    this.setData({ bucket: e.detail.value });
    this.applyFilter();
  },

  applyFilter() {
    this.setData({ list: this.data.all.filter((t) => t.bucket === this.data.bucket) });
  },

  showCode(e) {
    const t = this.data.all.find((x) => x.id === e.currentTarget.dataset.id);
    if (!t) return;
    wx.navigateTo({ url: `/pages/ticket-code/ticket-code?id=${t.id}` });
  },

  goDetail(e) {
    const t = this.data.all.find((x) => x.id === e.currentTarget.dataset.id);
    if (t && t.activityId) wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + t.activityId });
  },
});
