/**
 * WeChat JSAPI payment closeout. `api.payWechatJsapi()` returns
 * WeChatJSAPIResponse {paymentOrderId, prepay} (order/dto.go); the prepay params
 * must be handed to wx.requestPayment before the order counts as paid.
 *
 * We invoke wx.requestPayment and only resolve on its success callback; a user
 * cancel / failure rejects so the page stays put and shows a toast.
 */

// settle takes the payWechatJsapi() response and resolves once payment is
// complete. Missing payment capability is an explicit error.
function settle(res) {
  const prepay = res && res.data && res.data.prepay;
  if (!prepay || typeof wx === 'undefined' || typeof wx.requestPayment !== 'function') {
    return Promise.reject(new Error('微信支付参数无效'));
  }
  return new Promise((resolve, reject) => {
    wx.requestPayment({
      timeStamp: prepay.timeStamp,
      nonceStr: prepay.nonceStr,
      package: prepay.package,
      signType: prepay.signType,
      paySign: prepay.paySign,
      success: () => resolve(res),
      fail: (err) => reject(err),
    });
  });
}

module.exports = { settle };
