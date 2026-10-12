package feed_test

import (
	"encoding/xml"
	"time"

	"ssch.cc/podtrawl/internal/feed"
)

// The EST pubDate is expected at a zero offset, as Rfc2822Datetime does not yet
// resolve the zone abbreviations RFC 2822 defines; see rss20Sample's CST.
var itunesSample feed.Rss = feed.Rss{
	XMLName: xml.Name{Local: "rss", Space: "_"},
	Version: "2.0",
	Channel: feed.Channel{
		Title:       "Hiking Treks",
		Link:        "https://www.apple.com/itunes/podcasts/",
		Description: " Love to get outdoors and discover nature's treasures? Hiking Treks is the show for you. We review hikes and excursions, review outdoor gear and interview a variety of naturalists and adventurers. Look for new episodes each week. ",
		Language:    new("en-us"),
		Copyright:   new("\u00a9 2020 John Appleseed"),
		Itunes: feed.ItunesChannel{
			Author:   new("The Sunset Explorers"),
			Type:     new("serial"),
			Explicit: feed.ExplicitNo,
			Categories: []feed.ItunesCategory{
				{Text: "Sports", Subcategories: []feed.ItunesCategory{{Text: "Wilderness"}}},
			},
		},
		Items: []feed.Item{
			{
				Description: new("\n\t\t\t\tThe Sunset Explorers share tips, techniques and recommendations for great hikes and adventures around the United States. Listen on <a href=\"https://www.apple.com/itunes/podcasts/\">Apple Podcasts</a>.\n\t\t\t"),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/AllAboutEverythingEpisode4.mp3", Length: 498537, Type: "audio/mpeg"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "D03EEC9B-B1B4-475B-92C8-54F853FA2A22"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2019, time.January, 8, 1, 15, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					Title:       new("Hiking Treks Trailer"),
					EpisodeType: new("trailer"),
					Duration:    feed.ItunesDuration{Duration: 1079 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S02 EP04 Mt. Hood, Oregon"),
				Description: new(" Tips for trekking around the tallest mountain in Oregon "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/mthood.m4a", Length: 8727310, Type: "audio/x-m4a"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "22BCFEBF-44FB-4A19-8229-7AC678629F57"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2019, time.May, 7, 12, 0, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     4,
					Season:      2,
					Duration:    feed.ItunesDuration{Duration: 1024 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S02 EP03 Bouldering Around Boulder"),
				Link:        new("href=\"http://example.com/podcasts/everything/"),
				Description: new(" We explore fun walks to climbing areas about the beautiful Colorado city of Boulder. "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/boulder.mp4", Length: 5650889, Type: "video/mp4"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "BE486CAA-B3D5-4FB0-8298-EFEBE71C5982"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2019, time.April, 30, 13, 0, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     3,
					Season:      2,
					Duration:    feed.ItunesDuration{Duration: 3627 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S02 EP02 Caribou Mountain, Maine"),
				Description: new(" Put your fitness to the test with this invigorating hill climb. "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/caribou.m4v", Length: 5650889, Type: "audio/x-m4v"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "142FAFE9-B1DF-4F6D-BAA8-79BDBAF653A9"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2019, time.May, 23, 2, 0, 0, 0, time.FixedZone("", -7*3600))},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     2,
					Season:      2,
					Duration:    feed.ItunesDuration{Duration: 2434 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S02 EP01 Stawamus Chief"),
				Description: new(" We tackle Stawamus Chief outside of Vancouver, BC and you should too! "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/AllAboutEverythingEpisode4.mp3", Length: 498537, Type: "audio/mpeg"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "5F1DBAEB-3327-49FB-ACB3-DB0158A1D0A3"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2019, time.February, 16, 7, 0, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     1,
					Season:      2,
					Duration:    feed.ItunesDuration{Duration: 13*time.Minute + 24*time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S01 EP04 Kuliouou Ridge Trail"),
				Description: new(" Oahu, Hawaii, has some picturesque hikes and this is one of the best! "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/AllAboutEverythingEpisode4.mp3", Length: 498537, Type: "audio/mpeg"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "B5FCEB80-317C-4CD0-A84B-807065B43FB9"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2018, time.November, 27, 1, 15, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     4,
					Season:      1,
					Duration:    feed.ItunesDuration{Duration: 929 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S01 EP03 Blood Mountain Loop"),
				Description: new(" Hiking the Appalachian Trail and Freeman Trail in Georgia "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/AllAboutEverythingEpisode4.mp3", Length: 498537, Type: "audio/mpeg"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "F0C5D763-ED85-4449-9C09-81FEBDF6F126"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2018, time.October, 23, 1, 15, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     3,
					Season:      1,
					Duration:    feed.ItunesDuration{Duration: 1440 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S01 EP02 Garden of the Gods Wilderness"),
				Description: new(" Wilderness Area Garden of the Gods in Illinois is a delightful spot for an extended hike. "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/AllAboutEverythingEpisode4.mp3", Length: 498537, Type: "audio/mpeg"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "821DD0B2-571D-4DFD-8E11-556E8C1EFE6A"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2018, time.September, 18, 1, 15, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     2,
					Season:      1,
					Duration:    feed.ItunesDuration{Duration: 839 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
			{
				Title:       new("S01 EP01 Upper Priest Lake Trail to Continental Creek Trail"),
				Description: new(" We check out this powerfully scenic hike following the river in the Idaho Panhandle National Forests. "),
				Enclosures:  []feed.Enclosure{{Url: "http://example.com/podcasts/everything/AllAboutEverythingEpisode4.mp3", Length: 498537, Type: "audio/mpeg"}},
				Guid:        &feed.Guid{IsPermalink: true, Value: "EABDA7EE-1AC6-4B60-9E11-6B3F30B72F87"},
				PubDate:     &feed.Rfc2822Datetime{Time: time.Date(2018, time.August, 14, 1, 15, 0, 0, time.UTC)},
				Itunes: feed.ItunesItem{
					EpisodeType: new("full"),
					Episode:     1,
					Season:      1,
					Duration:    feed.ItunesDuration{Duration: 1399 * time.Second},
					Explicit:    feed.ExplicitNo,
				},
			},
		},
	},
}
