package news_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

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

	// then I expect a 200 with every post, newest first
	resp.RequireStatus(t, 200)
	var posts []newsPostJSON
	resp.DecodeJSON(t, &posts)

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
	if diff := cmp.Diff(want, posts); diff != "" {
		t.Errorf("unexpected news posts (-want +got):\n%s", diff)
	}
}

func TestListNews_ReturnsEmptyArray_WhenNoPostsExist(t *testing.T) {
	// given a guild that has not posted any news
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for the news list
	resp := srv.Get(t, "/api/news")

	// then I expect a 200 with an empty JSON array rather than null
	resp.RequireStatus(t, 200)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("expected an empty JSON array, got %q", got)
	}
}
