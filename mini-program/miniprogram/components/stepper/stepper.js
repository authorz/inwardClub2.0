// Quantity stepper (− value +). Reused by ordering grid and ticket purchase.
Component({
  properties: {
    value: { type: Number, value: 0 },
    min: { type: Number, value: 0 },
    max: { type: Number, value: 99 },
    size: { type: String, value: 'md' }, // md | sm
    theme: { type: String, value: 'light' }, // light | dark
    hideZero: { type: Boolean, value: false }, // hide "−/value" when 0, show only +
  },
  methods: {
    dec() {
      const v = Math.max(this.data.min, this.data.value - 1);
      if (v === this.data.value) return;
      this.triggerEvent('change', { value: v });
    },
    inc() {
      const v = Math.min(this.data.max, this.data.value + 1);
      if (v === this.data.value) return;
      this.triggerEvent('change', { value: v });
    },
  },
});
