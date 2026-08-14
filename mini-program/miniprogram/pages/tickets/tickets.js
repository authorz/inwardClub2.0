// 我的入场券 — 待使用/已使用/已过期分段
// Reference: design/mini-program/final/member-subpages/09-my-tickets-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');
const { TICKET_STATUS_LABEL } = require('../../constants/index');

const TICKET_STACK_STEP_RPX = 116;

// bucket lifecycle statuses into the three visible tabs
function bucketOf(status) {
  if (status === 'used') return 'used';
  if (status === 'expired' || status === 'refunded') return 'expired';
  return 'pending'; // unused | pending_verify
}

function splitTicketTime(value) {
  const text = String(value || '').trim();
  const match = text.match(/^\d{4}[./-](\d{1,2})[./-](\d{1,2})(?:\s+(.+))?$/);
  if (!match) return { dateText: text || '时间待定', clockText: '' };
  return {
    dateText: `${match[1].padStart(2, '0')}.${match[2].padStart(2, '0')}`,
    clockText: match[3] || '',
  };
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
    stackTravel: 0,
    stackViewportHeight: 430,
    showCodeModal: false,
    codeTicket: null,
    qr: [],
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

  onReady() {
    this.measureTicketViewport();
  },

  onStatusChange(e) {
    this.setData({ bucket: e.detail.value });
    this.applyFilter();
  },

  applyFilter() {
    const list = this.data.all.filter((t) => t.bucket === this.data.bucket);
    const stackTravel = Math.max(list.length - 1, 0) * TICKET_STACK_STEP_RPX;
    this.setData({ list, stackTravel }, () => this.measureTicketViewport());
  },

  measureTicketViewport() {
    if (!wx.createSelectorQuery || !wx.getWindowInfo) return;
    wx.nextTick(() => {
      wx.createSelectorQuery()
        .select('.tk__stack-scroll')
        .boundingClientRect((rect) => {
          if (!rect || !rect.height) return;
          const windowWidth = wx.getWindowInfo().windowWidth;
          if (!windowWidth) return;
          const stackViewportHeight = Math.max(Math.round((rect.height * 750) / windowWidth), 430);
          if (stackViewportHeight !== this.data.stackViewportHeight) {
            this.setData({ stackViewportHeight });
          }
        })
        .exec();
    });
  },

  onResize() {
    this.measureTicketViewport();
  },

  showCode(e) {
    const t = this.data.all.find((x) => x.id === e.currentTarget.dataset.id);
    if (!t) return;
    const raw = String(t.code || '');
    const time = splitTicketTime(t.timeText);
    let qr;
    try {
      qr = codeart.grid(raw);
    } catch {
      ui.error('二维码生成失败，请刷新后重试');
      return;
    }
    this.setData({
      showCodeModal: true,
      codeTicket: {
        id: t.id,
        title: t.title,
        imageUrl: t.imageUrl || '',
        tone: t.tone || '',
        ticketName: t.ticketName,
        qty: t.qty || 1,
        storeName: t.storeName || '活动地点待定',
        status: t.status,
        statusLabel: t.statusLabel,
        usable: t.status === 'unused' || t.status === 'pending_verify',
        dateText: time.dateText,
        clockText: time.clockText,
        codeRaw: raw,
        code: fmt.codeGroups(raw),
      },
      qr,
    });
    wx.setKeepScreenOn && wx.setKeepScreenOn({ keepScreenOn: true });
  },

  closeCode() {
    this.setData({ showCodeModal: false, codeTicket: null, qr: [] });
    wx.setKeepScreenOn && wx.setKeepScreenOn({ keepScreenOn: false });
  },

  copyCode() {
    const ticket = this.data.codeTicket;
    if (ticket && ticket.codeRaw) ui.copy(ticket.codeRaw, '核销码已复制');
  },

  noop() {},

  onUnload() {
    if (this.data.showCodeModal) {
      wx.setKeepScreenOn && wx.setKeepScreenOn({ keepScreenOn: false });
    }
  },

  goDetail(e) {
    const t = this.data.all.find((x) => x.id === e.currentTarget.dataset.id);
    if (!t || !t.activityId) return;
    if (this.data.showCodeModal) this.closeCode();
    wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + t.activityId });
  },
});
