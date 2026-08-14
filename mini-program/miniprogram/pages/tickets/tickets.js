// 我的入场券 — 待使用/已使用/已过期分段
// Reference: design/mini-program/final/member-subpages/09-my-tickets-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');
const { TICKET_STATUS_LABEL } = require('../../constants/index');

const TICKET_STACK_STEP_RPX = 116;
const TICKET_SWIPE_THRESHOLD_PX = 48;
const TICKET_SWIPE_EXIT_PX = 520;
const TICKET_SWIPE_DURATION_MS = 280;
const TICKET_ENTRANCE_START_DELAY_MS = 180;
const TICKET_ENTRANCE_DURATION_MS = 720;
const TICKET_ENTRANCE_STAGGER_MS = 120;
const TICKET_ENTRANCE_MAX_STAGGER_MS = 360;

function buildTicketLayers(list, activeIndex) {
  const activeTicket = list[activeIndex];
  if (!activeTicket) return [];
  const entranceStep = list.length > 1
    ? Math.min(TICKET_ENTRANCE_STAGGER_MS, Math.floor(TICKET_ENTRANCE_MAX_STAGGER_MS / (list.length - 1)))
    : 0;
  return list
    .filter((ticket) => ticket.ticketIndex !== activeTicket.ticketIndex)
    .concat(activeTicket)
    .map((ticket, stackIndex) => ({
      ...ticket,
      stackTop: stackIndex * TICKET_STACK_STEP_RPX,
      stackZ: stackIndex + 1,
      entranceDelay: TICKET_ENTRANCE_START_DELAY_MS + stackIndex * entranceStep,
    }));
}

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
    layers: [],
    currentTicketIndex: 0,
    dragTicketIndex: -1,
    dragOffset: 0,
    isTicketDragging: false,
    isTicketEntering: false,
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
        this.applyFilter(true);
      })
      .catch(() => this.setData({ loading: false }));
  },

  onStatusChange(e) {
    this.setData({ bucket: e.detail.value });
    this.applyFilter(false);
  },

  applyFilter(playEntrance) {
    if (this._ticketEntranceTimer) {
      clearTimeout(this._ticketEntranceTimer);
      this._ticketEntranceTimer = null;
    }
    const list = this.data.all.filter((t) => t.bucket === this.data.bucket);
    const indexedTickets = list.map((ticket, ticketIndex) => ({ ...ticket, ticketIndex }));
    const currentTicketIndex = Math.max(list.length - 1, 0);
    const layers = buildTicketLayers(indexedTickets, currentTicketIndex);
    const isTicketEntering = Boolean(playEntrance && layers.length);
    this.setData({
      list: indexedTickets,
      layers,
      currentTicketIndex,
      dragTicketIndex: -1,
      dragOffset: 0,
      isTicketDragging: false,
      isTicketEntering,
    }, () => {
      if (!isTicketEntering) return;
      const lastDelay = layers[layers.length - 1].entranceDelay;
      this._ticketEntranceTimer = setTimeout(() => {
        this._ticketEntranceTimer = null;
        this.setData({ isTicketEntering: false });
      }, TICKET_ENTRANCE_DURATION_MS + lastDelay);
    });
  },

  finishTicketEntrance() {
    if (this._ticketEntranceTimer) {
      clearTimeout(this._ticketEntranceTimer);
      this._ticketEntranceTimer = null;
    }
    if (this.data.isTicketEntering) this.setData({ isTicketEntering: false });
  },

  activateTicket(index) {
    if (!Number.isInteger(index) || index < 0 || index >= this.data.list.length) return;
    this.setData({
      currentTicketIndex: index,
      layers: buildTicketLayers(this.data.list, index),
      dragTicketIndex: -1,
      dragOffset: 0,
      isTicketDragging: false,
    });
  },

  onTicketTouchStart(e) {
    if (this._ticketSettleTimer || this.data.list.length < 2) return;
    this.finishTicketEntrance();
    const touch = e.touches && e.touches[0];
    const index = Number(e.currentTarget.dataset.index);
    if (!touch || !Number.isInteger(index)) return;
    this._ticketGesture = { index, startY: touch.clientY };
    this.setData({ dragTicketIndex: index, dragOffset: 0, isTicketDragging: true });
  },

  onTicketTouchMove(e) {
    const gesture = this._ticketGesture;
    const touch = e.touches && e.touches[0];
    if (!gesture || !touch) return;
    const offset = Math.max(-TICKET_SWIPE_EXIT_PX, Math.min(TICKET_SWIPE_EXIT_PX, touch.clientY - gesture.startY));
    this.setData({ dragOffset: offset });
  },

  onTicketTouchEnd(e) {
    const gesture = this._ticketGesture;
    if (!gesture) return;
    const touch = e.changedTouches && e.changedTouches[0];
    const offset = touch
      ? Math.max(-TICKET_SWIPE_EXIT_PX, Math.min(TICKET_SWIPE_EXIT_PX, touch.clientY - gesture.startY))
      : this.data.dragOffset;
    this._ticketGesture = null;

    if (Math.abs(offset) < 6) {
      this.setData({ dragTicketIndex: -1, dragOffset: 0, isTicketDragging: false });
      return;
    }
    this._suppressTicketTapUntil = Date.now() + TICKET_SWIPE_DURATION_MS + 80;
    if (Math.abs(offset) < TICKET_SWIPE_THRESHOLD_PX) {
      this.setData({ dragTicketIndex: -1, dragOffset: 0, isTicketDragging: false });
      return;
    }

    let nextIndex = gesture.index;
    if (gesture.index === this.data.currentTicketIndex) {
      const direction = offset < 0 ? 1 : -1;
      nextIndex = (gesture.index + direction + this.data.list.length) % this.data.list.length;
    }
    this.setData({
      dragOffset: offset < 0 ? -TICKET_SWIPE_EXIT_PX : TICKET_SWIPE_EXIT_PX,
      isTicketDragging: false,
    });
    this._ticketSettleTimer = setTimeout(() => {
      this._ticketSettleTimer = null;
      this.activateTicket(nextIndex);
    }, TICKET_SWIPE_DURATION_MS);
  },

  onTicketTouchCancel() {
    this._ticketGesture = null;
    this.setData({ dragTicketIndex: -1, dragOffset: 0, isTicketDragging: false });
  },

  onTicketTap(e) {
    if (this._suppressTicketTapUntil && Date.now() < this._suppressTicketTapUntil) return;
    this.finishTicketEntrance();
    const index = Number(e.currentTarget.dataset.index);
    if (!Number.isInteger(index) || index < 0 || index >= this.data.list.length) return;
    if (index === this.data.currentTicketIndex) {
      this.showCode(e);
      return;
    }
    this.activateTicket(index);
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
    if (this._ticketSettleTimer) clearTimeout(this._ticketSettleTimer);
    if (this._ticketEntranceTimer) clearTimeout(this._ticketEntranceTimer);
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
