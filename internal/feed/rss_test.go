package feed

import (
	"strings"
	"testing"
)

const collidingFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
     xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"
     xmlns:atom="http://www.w3.org/2005/Atom"
     xmlns:media="http://search.yahoo.com/mrss/"
     xmlns:podcast="https://podcastindex.org/namespace/1.0"
     xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
 <channel>
  <itunes:title>iTunes Channel Title</itunes:title>
  <title>Channel Title</title>
  <link>https://example.com/</link>
  <atom:link href="https://example.com/feed.xml" rel="self" type="application/rss+xml"/>
  <description>Channel Description</description>
  <media:copyright>Media Copyright</media:copyright>
  <copyright>Channel Copyright</copyright>
  <category>Channel Category</category>
  <itunes:category text="Technology"><itunes:category text="Gadgets"/></itunes:category>
  <media:category>Media Category</media:category>
  <item>
   <title>179: Episode Title</title>
   <itunes:title>Episode Title</itunes:title>
   <author>editor@example.com</author>
   <itunes:author>Display Name</itunes:author>
   <link>https://example.com/179</link>
   <comments>https://example.com/179#comments</comments>
   <slash:comments>42</slash:comments>
   <guid isPermaLink="false">tag:example.com,2026:179</guid>
   <podcast:guid>917393e3-1b1e-5cef-ace4-edaa54e1f810</podcast:guid>
   <category>Item Category</category>
   <media:category>Media Item Category</media:category>
  </item>
 </channel>
</rss>`

func TestLoadIgnoresOtherNamespaces(t *testing.T) {
	r, err := Load(strings.NewReader(collidingFeed))
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	c := r.Channel

	t.Run("channel", func(t *testing.T) {
		checkString(t, "Title", c.Title, "Channel Title")
		checkString(t, "Link", c.Link, "https://example.com/")
		checkString(t, "Description", c.Description, "Channel Description")
		checkStringPtr(t, "Copyright", c.Copyright, "Channel Copyright")
		if len(c.Categories) != 1 {
			t.Errorf("Categories gave %d entries (%+v), want 1", len(c.Categories), c.Categories)
		} else {
			checkString(t, "Categories[0].Value", c.Categories[0].Value, "Channel Category")
		}
	})

	t.Run("item", func(t *testing.T) {
		if len(c.Items) != 1 {
			t.Fatalf("Items gave %d entries, want 1", len(c.Items))
		}
		i := c.Items[0]
		checkStringPtr(t, "Title", i.Title, "179: Episode Title")
		checkStringPtr(t, "Author", i.Author, "editor@example.com")
		checkStringPtr(t, "Link", i.Link, "https://example.com/179")
		checkStringPtr(t, "Comments", i.Comments, "https://example.com/179#comments")
		if i.Guid == nil {
			t.Errorf("Guid gave nil, want tag:example.com,2026:179")
		} else {
			checkString(t, "Guid.Value", i.Guid.Value, "tag:example.com,2026:179")
			if i.Guid.IsPermalink {
				t.Errorf("Guid.IsPermalink gave true, want false")
			}
		}
		if len(i.Categories) != 1 {
			t.Errorf("Categories gave %d entries (%+v), want 1", len(i.Categories), i.Categories)
		} else {
			checkString(t, "Categories[0].Value", i.Categories[0].Value, "Item Category")
		}
	})
}

func TestLoadDoesNotSubstituteOtherNamespaces(t *testing.T) {
	const f = `<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
	 <channel>
	  <title>Channel Title</title>
	  <link>https://example.com/</link>
	  <description>Channel Description</description>
	  <item>
	   <itunes:title>iTunes Only</itunes:title>
	   <itunes:author>iTunes Only</itunes:author>
	  </item>
	 </channel>
	</rss>`
	r, err := Load(strings.NewReader(f))
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if len(r.Channel.Items) != 1 {
		t.Fatalf("Items gave %d entries, want 1", len(r.Channel.Items))
	}
	i := r.Channel.Items[0]
	if i.Title != nil {
		t.Errorf("Title gave %q, want nil", *i.Title)
	}
	if i.Author != nil {
		t.Errorf("Author gave %q, want nil", *i.Author)
	}
}

func TestLoadSkipsNestedOtherNamespaces(t *testing.T) {
	const f = `<rss version="2.0" xmlns:podcast="https://podcastindex.org/namespace/1.0">
	 <channel>
	  <podcast:person role="host" href="https://example.com/h">Host</podcast:person>
	  <title>Channel Title</title>
	  <podcast:funding url="https://example.com/f"/>
	  <link>https://example.com/</link>
	  <podcast:value type="lightning"><podcast:valueRecipient name="a" split="100"/></podcast:value>
	  <description>Channel Description</description>
	 </channel>
	</rss>`
	r, err := Load(strings.NewReader(f))
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	checkString(t, "Title", r.Channel.Title, "Channel Title")
	checkString(t, "Link", r.Channel.Link, "https://example.com/")
	checkString(t, "Description", r.Channel.Description, "Channel Description")
}

func TestRssTypesAreUnnamespaced(t *testing.T) {
	t.Run("Rss", func(t *testing.T) { checkNamespace[Rss](t, nsUnnamespaced) })
	t.Run("Channel", func(t *testing.T) { checkNamespace[Channel](t, nsUnnamespaced) })
	t.Run("Item", func(t *testing.T) { checkNamespace[Item](t, nsUnnamespaced) })
}

func checkString(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s gave %q, want %q", field, got, want)
	}
}

func checkStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s gave nil, want %q", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s gave %q, want %q", field, *got, want)
	}
}
