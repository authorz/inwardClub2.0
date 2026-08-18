Component({
  lifetimes: {
    ready() {
      this._videoContext = wx.createVideoContext('appBackgroundVideo', this);
      this.play();
    },
    detached() {
      this._videoContext = null;
    },
  },

  pageLifetimes: {
    show() {
      this.play();
    },
    hide() {
      if (this._videoContext) this._videoContext.pause();
    },
  },

  methods: {
    play() {
      if (this._videoContext) this._videoContext.play();
    },

    onError(e) {
      this._videoError = e && e.detail ? e.detail : null;
    },
  },
});
