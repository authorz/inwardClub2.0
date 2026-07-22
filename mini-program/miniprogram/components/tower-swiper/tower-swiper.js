// Tower swiper — a card-stack / photo-album carousel for native WeChat.
// Ported from the idea of zenochan/tower-swiper: cards are absolutely stacked
// and each one is transformed by its signed offset from the active card, so the
// active poster sits up front with its neighbours fanned out (tilted, dropped,
// scaled and dimmed) around a bottom pivot. Finger drag drives a continuous
// position so the deck follows the touch, then snaps to the nearest card.
//
// Reusable: pass a `list` of items ({ id, imageUrl, title, tone }); the poster
// artwork is rendered through the shared ic-poster component. Emits:
//   change  { current }        — active card changed
//   tap     { item, index }    — the front card was tapped
Component({
  properties: {
    // Cards to display, front-to-back ordered by array index.
    list: { type: Array, value: [] },
  },

  data: {
    current: 0, // active card index
    pos: 0, // continuous position (fractional while dragging)
    cards: [], // list decorated with per-card transform styles
    dragging: false, // true mid-swipe → CSS transition is suspended
    entered: false, // false = collapsed deck, true = fanned open
  },

  lifetimes: {
    created() {
      // px travelled by the finger that equals one card step; refined once we
      // can read the real viewport width.
      this.unit = 150;
    },
    attached() {
      try {
        const info = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
        this.unit = Math.max((info.windowWidth || 375) * 0.42, 120);
      } catch {
        /* keep default */
      }
    },
    detached() {
      if (this._enterTimer) clearTimeout(this._enterTimer);
    },
  },

  observers: {
    // New data → replay the entrance: render collapsed first, then fan open on
    // the next frame so the transition animates from the centred stack.
    list(list) {
      if (this._enterTimer) clearTimeout(this._enterTimer);
      if (!list || !list.length) {
        this.setData({ cards: [], current: 0, pos: 0, entered: false });
        return;
      }
      this.setData({ current: 0, entered: false });
      this.render(0);
      this._enterTimer = setTimeout(() => {
        this.setData({ entered: true });
        this.render(this.data.current);
      }, 160);
    },
  },

  methods: {
    // Nearest signed distance from card `i` to `pos` on a ring of `n` cards, so
    // even at index 0 the last card wraps onto the left and the second onto the
    // right — a true circular photo-album stack. Range: (-n/2, n/2].
    circularOffset(i, pos, n) {
      if (n <= 1) return i - pos;
      let o = ((((i - pos) % n) + n) % n); // wrap into [0, n)
      if (o > n / 2) o -= n; // fold onto the nearer side
      return o;
    },

    // Compute the transform for every card from its offset to `pos` and commit.
    render(pos) {
      const list = this.data.list || [];
      const n = list.length;
      const entered = this.data.entered;
      const RANGE = 2; // how many neighbours splay out on each side
      const cards = list.map((item, i) => {
        const o = this.circularOffset(i, pos, n); // signed ring distance
        const abs = Math.abs(o);
        const e = Math.max(-RANGE, Math.min(RANGE, o)); // clamp far cards
        const ae = Math.abs(e);
        const sign = e < 0 ? -1 : 1;
        // Split the offset so spread stays continuous through centre (no jump
        // as a card crosses the active slot mid-drag) yet eases off past the
        // first neighbour: slope changes at |e|=1.
        const a1 = Math.min(ae, 1); // 0→1 over the first step
        const a2 = Math.max(ae - 1, 0); // 0→1 over the second step
        let tx;
        let ty;
        let rot;
        let scale;
        let opacity;
        if (!entered) {
          // Collapsed deck: everyone stacked dead-centre, tucked behind.
          tx = 0;
          ty = 0;
          rot = 0;
          scale = 1 - Math.min(abs, RANGE) * 0.04;
          opacity = abs < 0.5 ? 1 : Math.max(0.3, 0.7 - (abs - 1) * 0.15);
        } else {
          // Fanned open around the bottom pivot — compact album stack: the
          // front card presses onto its neighbours, leaving their side and
          // bottom edges peeking out rather than splaying them across the view.
          tx = sign * (a1 * 160 + a2 * 45); // rpx — neighbours peek from the sides
          ty = a1 * 55 + a2 * 25; // rpx — dropped just enough to stack
          rot = sign * (a1 * 10 + a2 * 5); // deg — a gentle outward tilt
          scale = 1 - a1 * 0.06 - a2 * 0.04;
          opacity = abs > RANGE + 0.6 ? 0 : 1 - a1 * 0.42 - a2 * 0.25;
        }
        const z = 100 - Math.round(abs * 10); // front-most on top
        const style =
          `transform: translateX(${tx}rpx) translateY(${ty}rpx) rotate(${rot}deg) scale(${scale});` +
          `opacity:${opacity};z-index:${z};`;
        return Object.assign({}, item, { _style: style });
      });
      this.setData({ pos, cards });
    },

    onStart(e) {
      if ((this.data.list || []).length <= 1) return;
      this._startX = e.touches[0].clientX;
      this._moved = false;
      this.setData({ dragging: true });
    },

    onMove(e) {
      if (!this.data.dragging) return;
      const dx = e.touches[0].clientX - this._startX;
      if (Math.abs(dx) > 6) this._moved = true;
      // Drive a continuous, unbounded position; render() wraps it through
      // circularOffset, so dragging past either end fans the wrapped neighbour
      // (0 ↔ n-1) into view instead of hitting a rubber-band wall.
      this.render(this.data.current - dx / this.unit);
    },

    onEnd(e) {
      if (!this.data.dragging) return;
      const n = this.data.list.length;
      const dx = e.changedTouches[0].clientX - this._startX;
      const prev = this.data.current;
      let next = prev;
      if (dx <= -this.unit * 0.22) next += 1;
      else if (dx >= this.unit * 0.22) next -= 1;
      // Wrap onto the ring so 0→n-1 and n-1→0 are both reachable. render() is
      // periodic in pos (period n), so committing the wrapped index yields the
      // same transforms as the raw step — the snap stays continuous.
      const current = ((next % n) + n) % n;
      this.setData({ dragging: false, current });
      this.render(current);
      if (current !== prev) this.triggerEvent('change', { current });
    },

    onTap(e) {
      if (this._moved) return; // it was a swipe, not a tap
      const index = e.currentTarget.dataset.index;
      if (index === this.data.current) {
        this.triggerEvent('tap', { item: this.data.list[index], index });
      } else {
        // Tapping a fanned neighbour brings it to the front.
        this.setData({ current: index });
        this.render(index);
        this.triggerEvent('change', { current: index });
      }
    },
  },
});
