// Generic list cell: label + value (+ optional arrow). Reused by profile,
// order details, benefits, store bar, and every "label — value >" row.
Component({
  options: { multipleSlots: true },
  properties: {
    label: { type: String, value: '' },
    value: { type: String, value: '' },
    arrow: { type: Boolean, value: false },
    bordered: { type: Boolean, value: true },
    valueStrong: { type: Boolean, value: false },
    label2: { type: String, value: '' }, // secondary line under label
  },
  methods: {
    onTap() {
      this.triggerEvent('tap');
    },
  },
});
