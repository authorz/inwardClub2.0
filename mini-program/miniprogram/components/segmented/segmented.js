// Segmented control — one component, three visual variants used everywhere:
//   variant="block" : rounded track, black active fill (wallet ledger 金币/积分/券)
//   variant="pill"  : pill track, black active pill (rankings 周/月/总)
//   variant="line"  : underline tabs (order center 点餐/活动..., filters 全部/待支付)
Component({
  properties: {
    options: { type: Array, value: [] }, // [{label, value}] or ["全部","收入"]
    value: { type: null, value: '' },
    variant: { type: String, value: 'block' },
    theme: { type: String, value: 'light' },
    quiet: { type: Boolean, value: false },
  },
  data: { items: [] },
  observers: {
    options(list) {
      const items = (list || []).map((o) =>
        typeof o === 'string' ? { label: o, value: o } : o
      );
      this.setData({ items });
    },
  },
  methods: {
    onSelect(e) {
      const v = e.currentTarget.dataset.value;
      if (v === this.data.value) return;
      this.triggerEvent('change', { value: v });
    },
  },
});
