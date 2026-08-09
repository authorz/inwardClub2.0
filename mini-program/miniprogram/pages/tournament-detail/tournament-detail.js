const api = require('../../services/api');
const fmt = require('../../utils/format');

Page({
  data: { loading: true, loadError: '', event: null, navStatusBar: 20, navContentHeight: 44, navRightGap: 96 },
  onLoad(options) {
    this.measureNav();
    const id = Number(options && options.id);
    if (!Number.isInteger(id) || id <= 0) return this.setData({ loading: false, loadError: '赛事活动不存在' });
    api.getTournamentEvent(id).then((res) => {
      const event = res.data || {};
      const startText = fmt.dateTime(event.startAt);
      const endText = fmt.dateTime(event.endAt);
      this.setData({ loading: false, event: {
        id: event.id, title: event.title || '今日赛事', summary: event.summary || '', imageUrl: event.imageUrl || '',
        storeName: event.storeName || '', timeText: startText && endText ? `${startText} 至 ${endText}` : startText || endText,
        content: event.content || '',
      } });
    }).catch(() => this.setData({ loading: false, loadError: '赛事活动加载失败，请稍后重试' }));
  },
  measureNav() {
    try {
      const win = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const cap = wx.getMenuButtonBoundingClientRect();
      const statusBar = win.statusBarHeight || 20;
      const gap = Math.max(cap.top - statusBar, 4);
      this.setData({ navStatusBar: statusBar, navContentHeight: cap.height + gap * 2, navRightGap: Math.max(win.windowWidth - cap.left + 8, 96) });
    } catch { /* keep defaults */ }
  },
  goBack() {
    if (getCurrentPages().length > 1) wx.navigateBack();
    else wx.switchTab({ url: '/pages/reservation/reservation' });
  },
});
