// 出示票码 — 从接口返回的核销码生成可出示码
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');

Page({
  data: { title: '', sub: '', code: '', codeRaw: '', qr: [] },

  onLoad(options) {
    const raw = decodeURIComponent(options.code || '');
    this.setData({
      title: decodeURIComponent(options.title || ''),
      sub: decodeURIComponent(options.sub || ''),
      codeRaw: raw,
      code: fmt.codeGroups(raw),
      qr: codeart.grid(raw),
    });
    wx.setKeepScreenOn && wx.setKeepScreenOn({ keepScreenOn: true });
  },

  copyCode() {
    if (this.data.code) ui.copy(this.data.code, '核销码已复制');
  },
});
