package asset

import "time"

// Asset is the persisted representation of an uploaded object.
type Asset struct {
	ID               int64
	Bucket           string
	ObjectKey        string
	Etag             string
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
	Width            *int
	Height           *int
	Purpose          string
	Visibility       string
	Status           string
	UploadedByType   string
	UploadedByID     int64
	CreatedAt        time.Time
	UploadedAt       *time.Time
	DeletedAt        *time.Time
}

// purposePolicy defines the accepted MIME types and size ceiling per purpose.
type purposePolicy struct {
	mimes   map[string]string // content-type -> file extension
	maxSize int64
}

const (
	mib = 1 << 20
)

var imageMimes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

var richMimes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"video/mp4":  "mp4",
}

// policies is the purpose→constraints table (Qiniu spec §6).
var policies = map[UploadPurpose]purposePolicy{
	UploadPurposeAvatar:         {mimes: imageMimes, maxSize: 5 * mib},
	UploadPurposeStoreLogo:      {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeStoreContactQR: {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeBanner:         {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeCategory:       {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeProduct:        {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeActivity:       {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeTableLayout:    {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeSeatLayout:     {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeVipIcon:        {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeVipBanner:      {mimes: imageMimes, maxSize: 10 * mib},
	UploadPurposeRichContent:    {mimes: richMimes, maxSize: 200 * mib},
}
