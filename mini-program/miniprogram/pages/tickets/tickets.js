// 我的入场券 — 待使用/已使用/已过期分段
// Reference: design/mini-program/final/member-subpages/09-my-tickets-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');
const { TICKET_STATUS_LABEL } = require('../../constants/index');

const STACK_PREVIEW_TOP_RPX = 414;
const STACK_PREVIEW_STEP_RPX = 76;

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
    decks: [],
    currentTicketIndex: 0,
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

  onStatusChange(e) {
    this.setData({ bucket: e.detail.value });
    this.applyFilter();
  },

  applyFilter() {
    const list = this.data.all.filter((t) => t.bucket === this.data.bucket);
    const decks = list.map((ticket, ticketIndex) => ({
      ...ticket,
      previews: list.slice(ticketIndex + 1, ticketIndex + 3).map((preview, previewIndex) => ({
        ...preview,
        targetIndex: ticketIndex + previewIndex + 1,
        stackTop: STACK_PREVIEW_TOP_RPX + previewIndex * STACK_PREVIEW_STEP_RPX,
        stackZ: 2 - previewIndex,
      })),
    }));
    this.setData({ list, decks, currentTicketIndex: 0 });
  },

  onTicketChange(e) {
    const current = Number(e.detail.current);
    if (!Number.isInteger(current) || current === this.data.currentTicketIndex) return;
    this.setData({ currentTicketIndex: current });
  },

  selectTicket(e) {
    const index = Number(e.currentTarget.dataset.index);
    if (!Number.isInteger(index) || index < 0 || index >= this.data.decks.length) return;
    this.setData({ currentTicketIndex: index });
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
