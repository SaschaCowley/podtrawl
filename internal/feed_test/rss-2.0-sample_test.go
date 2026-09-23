package feed_test

import (
	"encoding/xml"
	"time"

	"ssch.cc/podtrawl/internal/feed"
)

var rss20Sample feed.Rss = feed.Rss{
	XMLName: xml.Name{Local: "rss", Space: "_"},
	Version: "2.0",
	Channel: feed.Channel{
		Description: "Current headlines from the Dallas Times-Herald newspaper",
		Link:        "https://dallas.example.com",
		Title:       "Dallas Times-Herald",
		Categories: []feed.Category{
			{Value: "Media"},
			{Domain: new("Newspapers/Regional/United_States"), Value: "Texas"},
		},
		Copyright:      new("Copyright 2006 Dallas Times-Herald"),
		Language:       new("epo"),
		LastBuildDate:  &feed.Rfc2822Datetime{Time: time.Date(2006, time.January, 29, 17, 17, 44, 0, time.UTC)},
		ManagingEditor: new("jlehrer@dallas.example.com (Jim Lehrer)"),
		PubDate:        &feed.Rfc2822Datetime{Time: time.Date(2006, time.January, 29, 5, 0, 0, 0, time.UTC)},
		SkipDays:       []string{"Saturday", "Sunday"},
		SkipHours:      []int{0, 1, 2, 22, 23},
		Ttl:            new(60),
		WebMaster:      new("bob.brock@dallas.example.com (Bob Brock)"),
		Items: []feed.Item{
			{
				Title:       new("Seventh Heaven! Ryan Hurls Another No Hitter"),
				Link:        new("https://dallas.example.com/1991/05/02/nolan.htm"),
				Description: new("Texas Rangers pitcher Nolan Ryan hurled the seventh no-hitter of his legendary career on Arlington Appreciation Night, defeating the Toronto Blue Jays 3-0. The 44-year-old struck out 16 batters before a crowd of 33,439."),
				Guid:        &feed.Guid{IsPermalink: true, Value: "https://dallas.example.com/1991/05/02/nolan.htm"},
			}, {
				Author: new("jbb@dallas.example.com (Joe Bob Briggs)"),
				Categories: []feed.Category{
					{Value: "rec.arts.movies.reviews"},
				},
				Comments:    new("https://dallas.example.com/feedback/1983/06/joebob.htm"),
				Description: new("I'm headed for France. I wasn't gonna go this year, but then last week \"Valley Girl\" came out and I said to myself, Joe Bob, you gotta get out of the country for a while."),
				Enclosures: []feed.Enclosure{
					{
						Length: 24986239,
						Type:   "audio/mpeg",
						Url:    "https://dallas.example.com/joebob_050689.mp3",
					},
				},
				Guid:    &feed.Guid{IsPermalink: true, Value: "https://dallas.example.com/1983/05/06/joebob.htm"},
				Link:    new("https://dallas.example.com/1983/05/06/joebob.htm"),
				PubDate: &feed.Rfc2822Datetime{Time: time.Date(1983, time.May, 6, 9, 0, 0, 0, time.UTC)},
				Source:  &feed.Source{Url: "https://la.example.com/rss.xml", Title: "Los Angeles Herald-Examiner"},
				Title:   new("Joe Bob Goes to the Drive-In"),
			}, {
				Description: new("I'm headed for France. I wasn't gonna go this year, but then last week <a href=\"https://www.imdb.com/title/tt0086525/\">Valley Girl</a> came out and I said to myself, Joe Bob, you gotta get out of the country for a while."),
				Guid:        &feed.Guid{IsPermalink: false, Value: "tag:dallas.example.com,4131:news"},
			}, {
				Description: new("I'm headed for France. I wasn't gonna go this year, but then last week <a href=\"https://www.imdb.com/title/tt0086525/\">Valley Girl</a> came out and I said to myself, Joe Bob, you gotta get out of the country for a while."),
				Guid:        &feed.Guid{IsPermalink: false, Value: "1983-05-06+lifestyle+joebob+2"},
			},
		},
	},
}
