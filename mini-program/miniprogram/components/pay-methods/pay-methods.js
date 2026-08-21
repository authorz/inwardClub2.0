// Payment method selector. Activity tickets additionally support coupon exchange.
const { PAY_METHOD, PAY_METHOD_LABEL } = require('../../constants/index');

Component({
  properties: {
    // which methods to offer; defaults to the regular paid methods
    methods: { type: Array, value: [PAY_METHOD.WECHAT, PAY_METHOD.COIN] },
    value: { type: String, value: PAY_METHOD.WECHAT },
    layout: { type: String, value: 'list' }, // list | grid
    theme: { type: String, value: 'light' }, // light | dark
    coinDisabled: { type: Boolean, value: false },
  },
  data: { items: [] },
  observers: {
    methods(list) {
      const items = (list || []).map((code) => ({
        code,
        label: PAY_METHOD_LABEL[code] || code,
      }));
      this.setData({ items });
    },
  },
  methods: {
    onPick(e) {
      const code = e.currentTarget.dataset.code;
      if (code === 'coin' && this.data.coinDisabled) return;
      if (code === this.data.value) return;
      this.triggerEvent('change', { value: code });
    },
  },
});
