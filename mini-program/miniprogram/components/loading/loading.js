// Shared transparent loading state for both page and section loads.
Component({
  properties: {
    show: { type: Boolean, value: true },
    text: { type: String, value: '' },
    theme: { type: String, value: 'dark' },
    fullscreen: { type: Boolean, value: true },
  },
});
