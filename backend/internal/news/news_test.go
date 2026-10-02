package news_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type newsPostJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Excerpt     string  `json:"excerpt"`
	Category    string  `json:"category"`
	ImageURL    *string `json:"imageUrl"`
	AuthorName  string  `json:"authorName"`
	PublishedAt string  `json:"publishedAt"`
	Pinned      bool    `json:"pinned"`
	Body        string  `json:"body"`
}

type linkJSON struct {
	Href string `json:"href"`
}

type newsPageJSON struct {
	Total    int                 `json:"total"`
	Links    map[string]linkJSON `json:"_links"`
	Embedded struct {
		News []newsPostJSON `json:"news"`
	} `json:"_embedded"`
}

func TestListNews_ReturnsPostsNewestFirst(t *testing.T) {
	// given three news posts published on different days
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, image_url, author_name, published_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Welcome our newest officers', 'Two promotions from the raid team.', 'guild-news', NULL, 'Officer', '2026-09-11T18:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Fuzion defeated Queen Ansurek on Mythic', '8/8 down after weeks on the enrage.', 'raid-progress', 'https://cdn.example/ansurek.jpg', 'Officer', '2026-09-15T18:00:00Z'),
			('33333333-3333-3333-3333-333333333333', 'Now recruiting: Restoration Druid', 'One raid spot open.', 'recruitment', NULL, 'Officer', '2026-09-13T18:00:00Z')`)

	// when I ask for the news list
	resp := srv.Get(t, "/api/news")

	// then I expect a 200 with every post embedded, newest first
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	imageURL := "https://cdn.example/ansurek.jpg"
	want := []newsPostJSON{
		{
			ID:          "22222222-2222-2222-2222-222222222222",
			Title:       "Fuzion defeated Queen Ansurek on Mythic",
			Excerpt:     "8/8 down after weeks on the enrage.",
			Category:    "raid-progress",
			ImageURL:    &imageURL,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-15T18:00:00Z",
		},
		{
			ID:          "33333333-3333-3333-3333-333333333333",
			Title:       "Now recruiting: Restoration Druid",
			Excerpt:     "One raid spot open.",
			Category:    "recruitment",
			ImageURL:    nil,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-13T18:00:00Z",
		},
		{
			ID:          "11111111-1111-1111-1111-111111111111",
			Title:       "Welcome our newest officers",
			Excerpt:     "Two promotions from the raid team.",
			Category:    "guild-news",
			ImageURL:    nil,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-11T18:00:00Z",
		},
	}
	if diff := cmp.Diff(want, page.Embedded.News); diff != "" {
		t.Errorf("unexpected news posts (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsEmptyPostList_WhenNoPostsExist(t *testing.T) {
	// given a guild that has not posted any news
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for the news list
	resp := srv.Get(t, "/api/news")

	// then I expect a 200 with a total of zero and an empty JSON array rather than null
	resp.RequireStatus(t, 200)
	if body := string(resp.Body); !strings.Contains(body, `"total":0`) || !strings.Contains(body, `"news":[]`) {
		t.Errorf("expected total 0 and an empty news array, got %q", body)
	}
}

func TestListNews_ReturnsOnlyTheNewestPosts_WhenLimitIsSet(t *testing.T) {
	// given three news posts published on different days
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, image_url, author_name, published_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Welcome our newest officers', 'Two promotions from the raid team.', 'guild-news', NULL, 'Officer', '2026-09-11T18:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Fuzion defeated Queen Ansurek on Mythic', '8/8 down after weeks on the enrage.', 'raid-progress', NULL, 'Officer', '2026-09-15T18:00:00Z'),
			('33333333-3333-3333-3333-333333333333', 'Now recruiting: Restoration Druid', 'One raid spot open.', 'recruitment', NULL, 'Officer', '2026-09-13T18:00:00Z')`)

	// when I ask for the news list with a limit of two
	resp := srv.Get(t, "/api/news?limit=2")

	// then I expect a 200 with only the two newest posts, newest first
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := []newsPostJSON{
		{
			ID:          "22222222-2222-2222-2222-222222222222",
			Title:       "Fuzion defeated Queen Ansurek on Mythic",
			Excerpt:     "8/8 down after weeks on the enrage.",
			Category:    "raid-progress",
			ImageURL:    nil,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-15T18:00:00Z",
		},
		{
			ID:          "33333333-3333-3333-3333-333333333333",
			Title:       "Now recruiting: Restoration Druid",
			Excerpt:     "One raid spot open.",
			Category:    "recruitment",
			ImageURL:    nil,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-13T18:00:00Z",
		},
	}
	if diff := cmp.Diff(want, page.Embedded.News); diff != "" {
		t.Errorf("unexpected news posts (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsEveryPost_WhenLimitExceedsThePostCount(t *testing.T) {
	// given a single news post
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, image_url, author_name, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Welcome our newest officers', 'Two promotions from the raid team.', 'guild-news', NULL, 'Officer', '2026-09-11T18:00:00Z')`)

	// when I ask for the news list with a limit of ten
	resp := srv.Get(t, "/api/news?limit=10")

	// then I expect a 200 with that one post
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := []newsPostJSON{
		{
			ID:          "11111111-1111-1111-1111-111111111111",
			Title:       "Welcome our newest officers",
			Excerpt:     "Two promotions from the raid team.",
			Category:    "guild-news",
			ImageURL:    nil,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-11T18:00:00Z",
		},
	}
	if diff := cmp.Diff(want, page.Embedded.News); diff != "" {
		t.Errorf("unexpected news posts (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsTheTenNewestPosts_WhenNoLimitIsSet(t *testing.T) {
	// given twelve news posts, one published per day
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for the news list without a limit
	resp := srv.Get(t, "/api/news")

	// then I expect a 200 with only the ten newest posts, newest first
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := []string{"Post 12", "Post 11", "Post 10", "Post 09", "Post 08", "Post 07", "Post 06", "Post 05", "Post 04", "Post 03"}
	if diff := cmp.Diff(want, titlesOf(page.Embedded.News)); diff != "" {
		t.Errorf("unexpected post titles (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsTheSecondPage_WhenOffsetAndLimitAreSet(t *testing.T) {
	// given twelve news posts, one published per day
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for the second page of five
	resp := srv.Get(t, "/api/news?limit=5&offset=5")

	// then I expect a 200 with the sixth to tenth newest posts, newest first
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := []string{"Post 07", "Post 06", "Post 05", "Post 04", "Post 03"}
	if diff := cmp.Diff(want, titlesOf(page.Embedded.News)); diff != "" {
		t.Errorf("unexpected post titles (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsEmptyPostList_WhenOffsetIsPastTheLastPost(t *testing.T) {
	// given three news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 3)

	// when I ask for a page that starts after the last post
	resp := srv.Get(t, "/api/news?offset=3")

	// then I expect a 200 that still reports the total, with an empty JSON array rather than null
	resp.RequireStatus(t, 200)
	if body := string(resp.Body); !strings.Contains(body, `"total":3`) || !strings.Contains(body, `"news":[]`) {
		t.Errorf("expected total 3 and an empty news array, got %q", body)
	}
}

func TestListNews_ReturnsOnlyThatCategory_WhenCategoryIsSet(t *testing.T) {
	// given news posts in three different categories
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, image_url, author_name, published_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Welcome our newest officers', 'Two promotions from the raid team.', 'guild-news', NULL, 'Officer', '2026-09-11T18:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Fuzion defeated Queen Ansurek on Mythic', '8/8 down after weeks on the enrage.', 'raid-progress', NULL, 'Officer', '2026-09-15T18:00:00Z'),
			('33333333-3333-3333-3333-333333333333', 'Now recruiting: Restoration Druid', 'One raid spot open.', 'recruitment', NULL, 'Officer', '2026-09-13T18:00:00Z')`)

	// when I ask for the recruitment posts
	resp := srv.Get(t, "/api/news?category=recruitment")

	// then I expect a 200 with only the recruitment post
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := []newsPostJSON{
		{
			ID:          "33333333-3333-3333-3333-333333333333",
			Title:       "Now recruiting: Restoration Druid",
			Excerpt:     "One raid spot open.",
			Category:    "recruitment",
			ImageURL:    nil,
			AuthorName:  "Officer",
			PublishedAt: "2026-09-13T18:00:00Z",
		},
	}
	if diff := cmp.Diff(want, page.Embedded.News); diff != "" {
		t.Errorf("unexpected news posts (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsTheUnpagedTotal_WhenLimitIsSet(t *testing.T) {
	// given twelve news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for a page of five
	resp := srv.Get(t, "/api/news?limit=5")

	// then I expect the total to count all twelve posts
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)
	if page.Total != 12 {
		t.Errorf("total = %d, want 12", page.Total)
	}
}

func TestListNews_ReturnsTheFilteredTotal_WhenCategoryIsSet(t *testing.T) {
	// given twelve news posts of which four are raid-progress
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for a page of two raid-progress posts
	resp := srv.Get(t, "/api/news?category=raid-progress&limit=2")

	// then I expect the total to count only the four raid-progress posts
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)
	if page.Total != 4 {
		t.Errorf("total = %d, want 4", page.Total)
	}
}

func TestListNews_ReturnsSelfFirstPrevNextAndLastLinks_WhenOnAMiddlePage(t *testing.T) {
	// given twelve news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for the second page of five
	resp := srv.Get(t, "/api/news?limit=5&offset=5")

	// then I expect links to every page around it
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?limit=5&offset=5"},
		"first": {Href: "/api/news?limit=5"},
		"prev":  {Href: "/api/news?limit=5"},
		"next":  {Href: "/api/news?limit=5&offset=10"},
		"last":  {Href: "/api/news?limit=5&offset=10"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_OmitsThePrevLink_WhenOnTheFirstPage(t *testing.T) {
	// given twelve news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for the first page of five
	resp := srv.Get(t, "/api/news?limit=5")

	// then I expect no prev link, and no offset in the links that point at the first page
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?limit=5"},
		"first": {Href: "/api/news?limit=5"},
		"next":  {Href: "/api/news?limit=5&offset=5"},
		"last":  {Href: "/api/news?limit=5&offset=10"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_OmitsTheNextLink_WhenOnTheLastPage(t *testing.T) {
	// given twelve news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for the last page of five
	resp := srv.Get(t, "/api/news?limit=5&offset=10")

	// then I expect no next link
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?limit=5&offset=10"},
		"first": {Href: "/api/news?limit=5"},
		"prev":  {Href: "/api/news?limit=5&offset=5"},
		"last":  {Href: "/api/news?limit=5&offset=10"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_PointsThePrevLinkAtTheLastPage_WhenOffsetIsPastTheLastPost(t *testing.T) {
	// given twelve news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for a page far beyond the last post
	resp := srv.Get(t, "/api/news?limit=5&offset=100")

	// then I expect prev to lead back to the last page that has posts, and no next link
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?limit=5&offset=100"},
		"first": {Href: "/api/news?limit=5"},
		"prev":  {Href: "/api/news?limit=5&offset=10"},
		"last":  {Href: "/api/news?limit=5&offset=10"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsOnlySelfFirstAndLastLinksAtOffsetZero_WhenNoPostsExist(t *testing.T) {
	// given a guild that has not posted any news
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for the news list
	resp := srv.Get(t, "/api/news")

	// then I expect no prev or next link
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news"},
		"first": {Href: "/api/news"},
		"last":  {Href: "/api/news"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_OmitsTheLimitFromEveryLink_WhenLimitWasNotSent(t *testing.T) {
	// given twenty-five news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 25)

	// when I ask for the second page without sending a limit
	resp := srv.Get(t, "/api/news?offset=10")

	// then I expect no link to spell out the default limit of ten
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?offset=10"},
		"first": {Href: "/api/news"},
		"prev":  {Href: "/api/news"},
		"next":  {Href: "/api/news?offset=20"},
		"last":  {Href: "/api/news?offset=20"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_KeepsTheLimitInEveryLink_WhenLimitEqualsTheDefault(t *testing.T) {
	// given twenty-five news posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 25)

	// when I ask for the second page and send the default limit explicitly
	resp := srv.Get(t, "/api/news?limit=10&offset=10")

	// then I expect every link to keep the limit I sent
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?limit=10&offset=10"},
		"first": {Href: "/api/news?limit=10"},
		"prev":  {Href: "/api/news?limit=10"},
		"next":  {Href: "/api/news?limit=10&offset=20"},
		"last":  {Href: "/api/news?limit=10&offset=20"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_KeepsTheCategoryInEveryLink_WhenCategoryIsSet(t *testing.T) {
	// given twelve news posts of which eight are guild-news
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedNumberedPosts(db, 12)

	// when I ask for the second page of three guild-news posts
	resp := srv.Get(t, "/api/news?category=guild-news&limit=3&offset=3")

	// then I expect every link to keep the category filter and page over the eight matches
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)

	want := map[string]linkJSON{
		"self":  {Href: "/api/news?limit=3&offset=3&category=guild-news"},
		"first": {Href: "/api/news?limit=3&category=guild-news"},
		"prev":  {Href: "/api/news?limit=3&category=guild-news"},
		"next":  {Href: "/api/news?limit=3&offset=6&category=guild-news"},
		"last":  {Href: "/api/news?limit=3&offset=6&category=guild-news"},
	}
	if diff := cmp.Diff(want, page.Links); diff != "" {
		t.Errorf("unexpected links (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsBadRequest_WhenLimitIsOutsideOneToFifty(t *testing.T) {
	for name, limit := range map[string]string{
		"limit is not a number":  "abc",
		"limit is zero":          "0",
		"limit is negative":      "-1",
		"limit is above the max": "51",
		"limit is empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			// given a running server
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)

			// when I ask for the news list with an invalid limit
			resp := srv.Get(t, "/api/news?limit="+limit)

			// then I expect a 400 explaining the limit rule
			resp.RequireStatus(t, 400)
			var body struct {
				Error string `json:"error"`
			}
			resp.DecodeJSON(t, &body)
			if want := "limit: must be an integer between 1 and 50"; body.Error != want {
				t.Errorf("error = %q, want %q", body.Error, want)
			}
		})
	}
}

func TestListNews_ReturnsBadRequest_WhenOffsetIsNotANonNegativeInteger(t *testing.T) {
	for name, offset := range map[string]string{
		"offset is not a number": "abc",
		"offset is negative":     "-1",
		"offset is empty":        "",
	} {
		t.Run(name, func(t *testing.T) {
			// given a running server
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)

			// when I ask for the news list with an invalid offset
			resp := srv.Get(t, "/api/news?offset="+offset)

			// then I expect a 400 explaining the offset rule
			resp.RequireStatus(t, 400)
			var body struct {
				Error string `json:"error"`
			}
			resp.DecodeJSON(t, &body)
			if want := "offset: must be a non-negative integer"; body.Error != want {
				t.Errorf("error = %q, want %q", body.Error, want)
			}
		})
	}
}

func TestListNews_ReturnsBadRequest_WhenCategoryIsUnknown(t *testing.T) {
	for name, category := range map[string]string{
		"category is not a known one": "memes",
		"category is empty":           "",
	} {
		t.Run(name, func(t *testing.T) {
			// given a running server
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)

			// when I ask for the news list with an unknown category
			resp := srv.Get(t, "/api/news?category="+category)

			// then I expect a 400 listing the valid categories
			resp.RequireStatus(t, 400)
			var body struct {
				Error string `json:"error"`
			}
			resp.DecodeJSON(t, &body)
			if want := "category: must be one of raid-progress, recruitment, guild-news, patch-notes"; body.Error != want {
				t.Errorf("error = %q, want %q", body.Error, want)
			}
		})
	}
}

// seedNumberedPosts inserts count posts titled "Post 01", "Post 02", ...
// published one day apart, so a higher number is always newer. The first four
// are raid-progress and the rest guild-news.
func seedNumberedPosts(db *sqlx.DB, count int) {
	db.MustExec(`
		INSERT INTO news_posts (title, excerpt, category, author_name, published_at)
		SELECT
			'Post ' || lpad(n::text, 2, '0'),
			'Excerpt',
			CASE WHEN n <= 4 THEN 'raid-progress' ELSE 'guild-news' END,
			'Officer',
			'2026-09-01T18:00:00Z'::timestamptz + n * interval '1 day'
		FROM generate_series(1, $1) AS n`, count)
}

func titlesOf(posts []newsPostJSON) []string {
	titles := make([]string, len(posts))
	for i, post := range posts {
		titles[i] = post.Title
	}
	return titles
}

type officerPostJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Excerpt     string  `json:"excerpt"`
	Category    string  `json:"category"`
	Body        string  `json:"body"`
	Pinned      bool    `json:"pinned"`
	AuthorName  string  `json:"authorName"`
	PublishedAt *string `json:"publishedAt"`
}

func TestListNews_ExcludesDrafts(t *testing.T) {
	// given one published post and one draft
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Published post', 'Out now.', 'guild-news', 'Officer', '2026-09-11T18:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Draft post', 'Not yet.', 'guild-news', 'Officer', NULL)`)

	// when I ask for the news list
	resp := srv.Get(t, "/api/news")

	// then I expect only the published post and a total of one
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)
	if page.Total != 1 || len(page.Embedded.News) != 1 || page.Embedded.News[0].ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("expected only the published post, got total %d and %+v", page.Total, page.Embedded.News)
	}
}

func TestListNews_ReturnsPinnedPostsFirst(t *testing.T) {
	// given an old pinned post and a newer unpinned post
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, pinned, published_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Old pinned post', 'Read me first.', 'guild-news', 'Officer', true, '2026-01-01T18:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'New post', 'Fresh.', 'guild-news', 'Officer', false, '2026-09-15T18:00:00Z')`)

	// when I ask for the news list
	resp := srv.Get(t, "/api/news")

	// then I expect the pinned post before the newer one
	resp.RequireStatus(t, 200)
	var page newsPageJSON
	resp.DecodeJSON(t, &page)
	gotIDs := []string{}
	for _, p := range page.Embedded.News {
		gotIDs = append(gotIDs, p.ID)
	}
	wantIDs := []string{"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Errorf("unexpected order (-want +got):\n%s", diff)
	}
	if !page.Embedded.News[0].Pinned {
		t.Errorf("expected the first post to be reported as pinned")
	}
}

func TestGetNewsPost_ReturnsPublishedPostWithBody(t *testing.T) {
	// given a published post with a Markdown body
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Patch day', 'Notes.', 'patch-notes', 'Officer', '# Heading', '2026-09-11T18:00:00Z')`)

	// when I ask for that post
	resp := srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111")

	// then I expect a 200 with the Markdown returned as-is
	resp.RequireStatus(t, 200)
	var got officerPostJSON
	resp.DecodeJSON(t, &got)
	publishedAt := "2026-09-11T18:00:00Z"
	want := officerPostJSON{
		ID: "11111111-1111-1111-1111-111111111111", Title: "Patch day", Excerpt: "Notes.", Category: "patch-notes",
		Body: "# Heading", AuthorName: "Officer", PublishedAt: &publishedAt,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected post (-want +got):\n%s", diff)
	}
}

func TestGetNewsPost_ReturnsNotFound_WhenPostIsDraft(t *testing.T) {
	// given a draft post
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Draft post', 'Not yet.', 'guild-news', 'Officer', NULL)`)

	// when I ask for that post
	resp := srv.Get(t, "/api/news/22222222-2222-2222-2222-222222222222")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestGetNewsPost_ReturnsNotFound_WhenPostDoesNotExist(t *testing.T) {
	// given no posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for an unknown post
	resp := srv.Get(t, "/api/news/99999999-9999-9999-9999-999999999999")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestCreateNewsPost_CreatesDraft_WhenOfficerSubmits(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	officer := srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())

	// when I create a post
	resp := srv.Post(t, "/api/news", map[string]any{
		"title": "Patch 11.0 notes", "excerpt": "What changed.", "category": "patch-notes", "body": "## Changes", "pinned": true,
	})

	// then I expect a 201 with a draft authored by me
	resp.RequireStatus(t, 201)
	var got officerPostJSON
	resp.DecodeJSON(t, &got)
	if got.ID == "" {
		t.Fatalf("expected an id, got %q", resp.Body)
	}
	want := officerPostJSON{
		ID: got.ID, Title: "Patch 11.0 notes", Excerpt: "What changed.", Category: "patch-notes",
		Body: "## Changes", Pinned: true, AuthorName: "Thrall", PublishedAt: nil,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected post (-want +got):\n%s", diff)
	}
	var authorID string
	if err := db.Get(&authorID, `SELECT author_user_id FROM news_posts WHERE id = $1`, got.ID); err != nil {
		t.Fatalf("read author: %v", err)
	}
	if authorID != officer.ID.String() {
		t.Errorf("expected author_user_id %s, got %s", officer.ID, authorID)
	}
}

func TestCreateNewsPost_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I create a post
	resp := srv.Post(t, "/api/news", map[string]any{"title": "x", "excerpt": "y", "category": "guild-news"})

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestCreateNewsPost_ReturnsBadRequest_WhenTitleIsEmpty(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())

	// when I create a post without a title
	resp := srv.Post(t, "/api/news", map[string]any{"title": "", "excerpt": "y", "category": "guild-news"})

	// then I expect a 400 naming the title
	resp.RequireStatus(t, 400)
	if !strings.Contains(string(resp.Body), "title") {
		t.Errorf("expected error to mention title, got %q", resp.Body)
	}
}

func TestCreateNewsPost_ReturnsBadRequest_WhenCategoryIsUnknown(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())

	// when I create a post with an unknown category
	resp := srv.Post(t, "/api/news", map[string]any{"title": "x", "excerpt": "y", "category": "memes"})

	// then I expect a 400 naming the category
	resp.RequireStatus(t, 400)
	if !strings.Contains(string(resp.Body), "category") {
		t.Errorf("expected error to mention category, got %q", resp.Body)
	}
}

func TestUpdateNewsPost_ChangesOnlyProvidedFields(t *testing.T) {
	// given a draft and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Old title', 'Old excerpt.', 'guild-news', 'Officer', 'Old body', NULL)`)

	// when I patch only the body
	resp := srv.Patch(t, "/api/news/22222222-2222-2222-2222-222222222222", map[string]any{"body": "New body"})

	// then I expect a 200 where only the body changed
	resp.RequireStatus(t, 200)
	var got officerPostJSON
	resp.DecodeJSON(t, &got)
	want := officerPostJSON{
		ID: "22222222-2222-2222-2222-222222222222", Title: "Old title", Excerpt: "Old excerpt.", Category: "guild-news",
		Body: "New body", AuthorName: "Officer",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected post (-want +got):\n%s", diff)
	}
}

func TestUpdateNewsPost_ReturnsNotFound_WhenPostDoesNotExist(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())

	// when I patch an unknown post
	resp := srv.Patch(t, "/api/news/99999999-9999-9999-9999-999999999999", map[string]any{"body": "x"})

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestPublishNewsPost_SetsPublishedAt_OnFirstPublishOnly(t *testing.T) {
	// given a draft and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Draft post', 'Soon.', 'guild-news', 'Officer', NULL)`)

	// and the post has been published once
	first := srv.Post(t, "/api/news/22222222-2222-2222-2222-222222222222/publish", nil)
	first.RequireStatus(t, 200)
	var firstPost officerPostJSON
	first.DecodeJSON(t, &firstPost)
	if firstPost.PublishedAt == nil || *firstPost.PublishedAt != "2026-03-14T20:00:00Z" {
		t.Fatalf("expected first publish to use the clock, got %v", firstPost.PublishedAt)
	}

	// when I publish it again
	resp := srv.Post(t, "/api/news/22222222-2222-2222-2222-222222222222/publish", nil)

	// then I expect a 200 and the original publish date to be kept
	resp.RequireStatus(t, 200)
	var got officerPostJSON
	resp.DecodeJSON(t, &got)
	if got.PublishedAt == nil || *got.PublishedAt != "2026-03-14T20:00:00Z" {
		t.Errorf("expected publishedAt to stay 2026-03-14T20:00:00Z, got %v", got.PublishedAt)
	}
}

func TestPublishNewsPost_KeepsExistingPublishedAt_WhenPostIsAlreadyPublished(t *testing.T) {
	// given a post published long before the server clock's "now"
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Live post', 'Out.', 'guild-news', 'Officer', '2026-01-01T00:00:00Z')`)

	// when I publish it again
	resp := srv.Post(t, "/api/news/11111111-1111-1111-1111-111111111111/publish", nil)

	// then I expect the original date back
	resp.RequireStatus(t, 200)
	var got officerPostJSON
	resp.DecodeJSON(t, &got)
	if got.PublishedAt == nil || *got.PublishedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("expected publishedAt 2026-01-01T00:00:00Z, got %v", got.PublishedAt)
	}
}

func TestPublishNewsPost_PostsToDiscordWebhook(t *testing.T) {
	// given a draft and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Server first', 'We did it.', 'raid-progress', 'Officer', NULL)`)

	// when I publish it
	resp := srv.Post(t, "/api/news/22222222-2222-2222-2222-222222222222/publish", nil)

	// then I expect one announcement embed linking to the post
	resp.RequireStatus(t, 200)
	bodies := srv.Announcements.Bodies()
	if len(bodies) != 1 {
		t.Fatalf("expected exactly 1 announcement, got %d", len(bodies))
	}
	var msg struct {
		Embeds []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
		} `json:"embeds"`
	}
	if err := json.Unmarshal(bodies[0], &msg); err != nil {
		t.Fatalf("decode webhook body %q: %v", bodies[0], err)
	}
	if len(msg.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(msg.Embeds))
	}
	if got := msg.Embeds[0]; got.Title != "Server first" || got.Description != "We did it." ||
		got.URL != testutil.SiteBaseURL+"/news/22222222-2222-2222-2222-222222222222" {
		t.Errorf("unexpected embed %+v", got)
	}
	if n := len(srv.Webhook.Bodies()); n != 0 {
		t.Errorf("expected no recruiting posts, got %d", n)
	}
}

func TestPublishNewsPost_DoesNotPostAgain_WhenPostIsAlreadyPublished(t *testing.T) {
	// given an already published post and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Live post', 'Out.', 'guild-news', 'Officer', '2026-09-11T18:00:00Z')`)

	// when I publish it again
	resp := srv.Post(t, "/api/news/11111111-1111-1111-1111-111111111111/publish", nil)

	// then I expect a 200 and no announcement
	resp.RequireStatus(t, 200)
	if n := len(srv.Announcements.Bodies()); n != 0 {
		t.Errorf("expected no announcements, got %d", n)
	}
}

func TestPublishNewsPost_StillReturnsOK_WhenDiscordWebhookFails(t *testing.T) {
	// given an announcements webhook that answers 500 and a draft
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	srv.Announcements.Status = 500
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Draft post', 'Soon.', 'guild-news', 'Officer', NULL)`)

	// when I publish it
	resp := srv.Post(t, "/api/news/22222222-2222-2222-2222-222222222222/publish", nil)

	// then I expect a 200 with the post published
	resp.RequireStatus(t, 200)
	var got officerPostJSON
	resp.DecodeJSON(t, &got)
	if got.PublishedAt == nil {
		t.Errorf("expected the post to be published")
	}
}

func TestPublishNewsPost_DoesNotCallDiscord_WhenWebhookURLIsEmpty(t *testing.T) {
	// given a deploy without an announcements webhook and a draft
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithoutAnnouncementsWebhook())
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Draft post', 'Soon.', 'guild-news', 'Officer', NULL)`)

	// when I publish it
	resp := srv.Post(t, "/api/news/22222222-2222-2222-2222-222222222222/publish", nil)

	// then I expect a 200 and no announcement
	resp.RequireStatus(t, 200)
	if n := len(srv.Announcements.Bodies()); n != 0 {
		t.Errorf("expected no announcements, got %d", n)
	}
}

func TestPublishNewsPost_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a draft and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Draft post', 'Soon.', 'guild-news', 'Officer', NULL)`)

	// when I publish it
	resp := srv.Post(t, "/api/news/22222222-2222-2222-2222-222222222222/publish", nil)

	// then I expect a 403 and the post to stay a draft
	resp.RequireStatus(t, 403)
	var isDraft bool
	if err := db.Get(&isDraft, `SELECT published_at IS NULL FROM news_posts WHERE id = '22222222-2222-2222-2222-222222222222'`); err != nil {
		t.Fatalf("read post: %v", err)
	}
	if !isDraft {
		t.Errorf("expected the post to remain a draft")
	}
}

func TestListDrafts_ReturnsOnlyUnpublishedPosts_WhenOfficerAsks(t *testing.T) {
	// given a published post, a draft and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Published post', 'Out.', 'guild-news', 'Officer', '2026-09-11T18:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Draft post', 'Soon.', 'guild-news', 'Officer', NULL)`)

	// when I ask for the drafts
	resp := srv.Get(t, "/api/news/drafts")

	// then I expect a plain array holding only the draft
	resp.RequireStatus(t, 200)
	var got []officerPostJSON
	resp.DecodeJSON(t, &got)
	if len(got) != 1 || got[0].ID != "22222222-2222-2222-2222-222222222222" || got[0].PublishedAt != nil {
		t.Errorf("expected only the draft, got %+v", got)
	}
}

func TestListDrafts_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")

	// when I ask for the drafts
	resp := srv.Get(t, "/api/news/drafts")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestDeleteNewsPost_RemovesPost_WhenOfficerDeletes(t *testing.T) {
	// given a post and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Live post', 'Out.', 'guild-news', 'Officer', '2026-09-11T18:00:00Z')`)

	// when I delete it
	resp := srv.Delete(t, "/api/news/11111111-1111-1111-1111-111111111111")

	// then I expect a 204 and the post to be gone
	resp.RequireStatus(t, 204)
	srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111").RequireStatus(t, 404)
}

func TestDeleteNewsPost_ReturnsNotFound_WhenPostDoesNotExist(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Thrall", testutil.AsOfficer())

	// when I delete an unknown post
	resp := srv.Delete(t, "/api/news/99999999-9999-9999-9999-999999999999")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}
