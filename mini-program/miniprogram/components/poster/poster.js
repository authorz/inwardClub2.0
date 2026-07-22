// Poster — activity / ticket artwork. Renders the real asset image when the
// API provides one; otherwise a premium dark placeholder carrying the title
// (so the layout is faithful without shipping the AI mock images).
Component({
  properties: {
    src: { type: String, value: '' },
    title: { type: String, value: '' },
    subtitle: { type: String, value: 'INWARD CLUB' },
    tone: { type: String, value: 'poker' }, // poker | coast | city | member
    radius: { type: String, value: '20rpx' },
  },
});
