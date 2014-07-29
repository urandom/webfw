package webfw

import (
	"time"

	"github.com/urandom/webfw/context"
)

type SitemapItem struct {
	Loc        string
	LastMod    time.Time
	ChangeFreq SitemapFrequency
	Priority   float64
}

// The sitemap controller provides an additional Sitemap method that returns
// a slice of SitemapItems which consist of the url of the controller, relative
// to the dispatcher, the last modification time, the change frequency, and the
// priority.
type SitemapController interface {
	Sitemap(context.Context) []SitemapItem
}

type SitemapFrequency string

const (
	SitemapFrequencyAlways  SitemapFrequency = "always"
	SitemapFrequencyHourly  SitemapFrequency = "hourly"
	SitemapFrequencyDaily   SitemapFrequency = "daily"
	SitemapFrequencyWeekly  SitemapFrequency = "weekly"
	SitemapFrequencyMonthly SitemapFrequency = "monthly"
	SitemapFrequencyYearly  SitemapFrequency = "yearly"
	SitemapFrequencyNever   SitemapFrequency = "never"

	SitemapNoFrequency SitemapFrequency = ""
	SitemapNoPriority  float64          = -1
)

var (
	SitemapNoLastMod = time.Unix(0, 0)
)
