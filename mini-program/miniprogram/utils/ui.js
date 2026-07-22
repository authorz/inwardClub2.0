/**
 * Centralized UI feedback: loading, toast, confirm dialogs.
 * Pages never call wx.showToast / wx.showModal directly — they go through here
 * so copy and behavior stay consistent (GLOBAL_DESIGN_RULES §7).
 */

let loadingCount = 0;

function showLoading(title) {
  loadingCount += 1;
  wx.showLoading({ title: title || '加载中', mask: true });
}

function hideLoading() {
  loadingCount = Math.max(0, loadingCount - 1);
  if (loadingCount === 0) wx.hideLoading();
}

function toast(title, icon) {
  wx.showToast({ title: title || '', icon: icon || 'none', duration: 1800 });
}

function success(title) {
  wx.showToast({ title: title || '操作成功', icon: 'success', duration: 1500 });
}

function error(message) {
  wx.showToast({ title: message || '操作失败', icon: 'none', duration: 2000 });
}

/** promise-based confirm */
function confirm(options) {
  const opts = typeof options === 'string' ? { content: options } : options || {};
  return new Promise((resolve) => {
    wx.showModal({
      title: opts.title || '提示',
      content: opts.content || '',
      confirmText: opts.confirmText || '确定',
      cancelText: opts.cancelText || '取消',
      confirmColor: '#111111',
      showCancel: opts.showCancel !== false,
      success: (res) => resolve(!!res.confirm),
      fail: () => resolve(false),
    });
  });
}

function copy(text, tip) {
  wx.setClipboardData({
    data: String(text || ''),
    success: () => toast(tip || '已复制'),
  });
}

module.exports = { showLoading, hideLoading, toast, success, error, confirm, copy };
