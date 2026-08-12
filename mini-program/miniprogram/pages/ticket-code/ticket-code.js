// 出示二维码 — 根据票券接口数据展示完整入场凭证
const api = require('../../services/api');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');
const { TICKET_STATUS_LABEL } = require('../../constants/index');

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
    missing: false,
    ticket: null,
    qr: [],
  },

  onLoad(options) {
    this.ticketId = String(options.id || '');
    wx.setKeepScreenOn && wx.setKeepScreenOn({ keepScreenOn: true });
    this.loadTicket();
  },

  onUnload() {
    wx.setKeepScreenOn && wx.setKeepScreenOn({ keepScreenOn: false });
  },

  loadTicket() {
    api
      .getTickets()
      .then((res) => {
        const ticket = (res.data || []).find((item) => String(item.id) === this.ticketId);
        if (!ticket) {
          this.setData({ loading: false, missing: true });
          return;
        }
        const raw = String(ticket.code || '');
        const time = splitTicketTime(ticket.timeText);
        this.setData({
          loading: false,
          ticket: {
            id: ticket.id,
            title: ticket.title,
            imageUrl: ticket.imageUrl || '',
            tone: ticket.tone || '',
            ticketName: ticket.ticketName,
            qty: ticket.qty || 1,
            storeName: ticket.storeName || '活动地点待定',
            status: ticket.status,
            statusLabel: TICKET_STATUS_LABEL[ticket.status] || ticket.status,
            usable: ticket.status === 'unused' || ticket.status === 'pending_verify',
            dateText: time.dateText,
            clockText: time.clockText,
            codeRaw: raw,
            code: fmt.codeGroups(raw),
          },
          qr: codeart.grid(raw),
        });
      })
      .catch(() => this.setData({ loading: false, missing: true }));
  },

  copyCode() {
    if (this.data.ticket && this.data.ticket.codeRaw) ui.copy(this.data.ticket.codeRaw, '核销码已复制');
  },
});
