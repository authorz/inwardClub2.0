// Ticket / coupon row for lists and details.
Component({
  properties: {
    title: { type: String, value: '' },
    code: { type: String, value: '' },
    status: { type: String, value: '' },
    desc: { type: String, value: '' },
    timeText: { type: String, value: '' },
    amountText: { type: String, value: '' },
  },
  methods: {
    onTap() { this.triggerEvent('tap'); },
  },
});
