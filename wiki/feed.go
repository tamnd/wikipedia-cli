package wiki

import (
	"context"
	"fmt"
	"time"
)

// FeaturedFeed is the daily feed for a date.
type FeaturedFeed struct {
	TFA       *FeedArticle  `json:"tfa,omitempty"`
	MostRead  []FeedArticle `json:"mostread,omitempty"`
	Image     *FeedImage    `json:"image,omitempty"`
	News      []FeedNews    `json:"news,omitempty"`
	OnThisDay []OnThisDay   `json:"onthisday,omitempty"`
}

// FeedArticle is an article reference inside a feed.
type FeedArticle struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Extract     string `json:"extract,omitempty"`
	URL         string `json:"url"`
	Views       int    `json:"views,omitempty"`
	Rank        int    `json:"rank,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
}

// FeedImage is the picture of the day.
type FeedImage struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// FeedNews is an in-the-news story.
type FeedNews struct {
	Story string        `json:"story"`
	Links []FeedArticle `json:"links,omitempty"`
}

// OnThisDay is a historical event.
type OnThisDay struct {
	Year  int      `json:"year"`
	Text  string   `json:"text"`
	Pages []string `json:"pages,omitempty"`
}

// Featured fetches the daily featured feed for a date (in the wiki's language).
func (c *Client) Featured(ctx context.Context, date time.Time) (*FeaturedFeed, error) {
	path := fmt.Sprintf("feed/featured/%04d/%02d/%02d", date.Year(), int(date.Month()), date.Day())
	var resp struct {
		TFA struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Extract     string `json:"extract"`
			ContentURLs struct {
				Desktop struct {
					Page string `json:"page"`
				} `json:"desktop"`
			} `json:"content_urls"`
		} `json:"tfa"`
		MostRead struct {
			Articles []struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Views       int    `json:"views"`
				Rank        int    `json:"rank"`
				ContentURLs struct {
					Desktop struct {
						Page string `json:"page"`
					} `json:"desktop"`
				} `json:"content_urls"`
			} `json:"articles"`
		} `json:"mostread"`
		Image struct {
			Title       string `json:"title"`
			Description struct {
				Text string `json:"text"`
			} `json:"description"`
			Image struct {
				Source string `json:"source"`
			} `json:"image"`
		} `json:"image"`
		News []struct {
			Story string `json:"story"`
			Links []struct {
				Title       string `json:"title"`
				ContentURLs struct {
					Desktop struct {
						Page string `json:"page"`
					} `json:"desktop"`
				} `json:"content_urls"`
			} `json:"links"`
		} `json:"news"`
		OnThisDay []struct {
			Year int    `json:"year"`
			Text string `json:"text"`
		} `json:"onthisday"`
	}
	if err := c.HTTP.GetJSON(ctx, c.Site.RestV1(path), ttlFeed, &resp); err != nil {
		return nil, err
	}
	feed := &FeaturedFeed{}
	if resp.TFA.Title != "" {
		feed.TFA = &FeedArticle{
			Title: resp.TFA.Title, Description: resp.TFA.Description,
			Extract: resp.TFA.Extract, URL: resp.TFA.ContentURLs.Desktop.Page,
		}
	}
	for _, a := range resp.MostRead.Articles {
		feed.MostRead = append(feed.MostRead, FeedArticle{
			Title: a.Title, Description: a.Description, Views: a.Views,
			Rank: a.Rank, URL: a.ContentURLs.Desktop.Page,
		})
	}
	if resp.Image.Title != "" {
		feed.Image = &FeedImage{
			Title: resp.Image.Title, Description: stripHTML(resp.Image.Description.Text),
			URL: resp.Image.Image.Source,
		}
	}
	for _, n := range resp.News {
		fn := FeedNews{Story: stripHTML(n.Story)}
		for _, l := range n.Links {
			fn.Links = append(fn.Links, FeedArticle{Title: l.Title, URL: l.ContentURLs.Desktop.Page})
		}
		feed.News = append(feed.News, fn)
	}
	for _, o := range resp.OnThisDay {
		feed.OnThisDay = append(feed.OnThisDay, OnThisDay{Year: o.Year, Text: stripHTML(o.Text)})
	}
	return feed, nil
}

// OnThisDayEvents fetches historical events of a given type for a month/day.
// eventType is one of all/selected/births/deaths/holidays/events.
func (c *Client) OnThisDayEvents(ctx context.Context, eventType string, month, day int) ([]OnThisDay, error) {
	if eventType == "" {
		eventType = "all"
	}
	path := fmt.Sprintf("feed/onthisday/%s/%02d/%02d", eventType, month, day)
	var resp map[string][]struct {
		Year  int    `json:"year"`
		Text  string `json:"text"`
		Pages []struct {
			Title string `json:"title"`
		} `json:"pages"`
	}
	if err := c.HTTP.GetJSON(ctx, c.Site.RestV1(path), ttlFeed, &resp); err != nil {
		return nil, err
	}
	var out []OnThisDay
	order := []string{"selected", "events", "births", "deaths", "holidays"}
	emit := func(key string) {
		for _, e := range resp[key] {
			otd := OnThisDay{Year: e.Year, Text: stripHTML(e.Text)}
			for _, p := range e.Pages {
				otd.Pages = append(otd.Pages, p.Title)
			}
			out = append(out, otd)
		}
	}
	if eventType == "all" {
		for _, k := range order {
			emit(k)
		}
	} else {
		emit(eventType)
	}
	return out, nil
}
