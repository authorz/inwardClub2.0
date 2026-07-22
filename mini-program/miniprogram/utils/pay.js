/**
 * WeChat JSAPI payment closeout. `api.payWechatJsapi()` returns
 * WeChatJSAPIResponse {paymentOrderId, prepay} (order/dto.go); the prepay params
 * must be handed to wx.requestPayment before the order counts as paid.
 *
 * In mock mode the fixtures still return a prepay object, but there is no real
 * transaction to complete — so we resolve immediately and let the flow finish,
 * exactly as before. Against the real API we invoke wx.requestPayment and only
 * resolve on its success callback; a user cancel / failure rejects so the page
 * stays put and shows a toast.
 */
const ENV = require('../config/env');

// settle takes the payWechatJsapi() response and resolves once payment is
// complete (or is a no-op in mock / when wx.requestPayment is unavailable).
function settle(res) {
  const prepay = res && res.data && res.data.prepay;
  if (ENV.useMock || !prepay || typeof wx === 'undefined' || typeof wx.requestPayment !== 'function') {
    return Promise.resolve(res);
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
