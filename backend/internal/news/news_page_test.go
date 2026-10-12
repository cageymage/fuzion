package news_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestGetNewsPostPage_ReturnsPostTagsWithThumbImage_WhenPostHasUploadedImage(t *testing.T) {
	// given a published post whose body starts with an uploaded image
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Patch day', 'Notes for the new patch.', 'patch-notes', 'Officer',
			'![Boss kill](/api/images/33333333-3333-3333-3333-333333333333) Great night.', '2026-09-11T18:00:00Z')`)

	// when I ask for that post's page
	resp := srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111/page")

	// then I expect a 200 whose og tags describe the post and point at the image thumbnail
	resp.RequireStatus(t, 200)
	if got, want := resp.Header.Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	wantBlock := `
    <title>Patch day | Fuzion</title>
    <meta name="description" content="Notes for the new patch." />
    <link rel="canonical" href="https://fuzion.example/news/11111111-1111-1111-1111-111111111111" />
    <meta property="og:type" content="article" />
    <meta property="og:site_name" content="Fuzion" />
    <meta property="og:title" content="Patch day" />
    <meta property="og:description" content="Notes for the new patch." />
    <meta property="og:url" content="https://fuzion.example/news/11111111-1111-1111-1111-111111111111" />
    <meta property="og:image" content="https://fuzion.example/api/images/33333333-3333-3333-3333-333333333333/thumb" />
    <meta name="twitter:card" content="summary_large_image" />
    `
	if diff := cmp.Diff(testutil.ShellPrefix+wantBlock+testutil.ShellSuffix, string(resp.Body)); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
}

func TestGetNewsPostPage_ReturnsSiteImageAndDescription_WhenPostHasNoImageOrExcerpt(t *testing.T) {
	// given a published post with no image in its body and no excerpt
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Recruiting', '', 'recruitment', 'Officer', 'We need a healer.', '2026-09-11T18:00:00Z')`)

	// when I ask for that post's page
	resp := srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111/page")

	// then I expect the site description and the site share image with its dimensions
	resp.RequireStatus(t, 200)
	wantBlock := `
    <title>Recruiting | Fuzion</title>
    <meta name="description" content="A PvE guild for World of Warcraft: Forever on the US servers. Raiding, dungeons and community." />
    <link rel="canonical" href="https://fuzion.example/news/11111111-1111-1111-1111-111111111111" />
    <meta property="og:type" content="article" />
    <meta property="og:site_name" content="Fuzion" />
    <meta property="og:title" content="Recruiting" />
    <meta property="og:description" content="A PvE guild for World of Warcraft: Forever on the US servers. Raiding, dungeons and community." />
    <meta property="og:url" content="https://fuzion.example/news/11111111-1111-1111-1111-111111111111" />
    <meta property="og:image" content="https://fuzion.example/og-image.jpg" />
    <meta property="og:image:width" content="1200" />
    <meta property="og:image:height" content="630" />
    <meta property="og:image:alt" content="The Fuzion logo, a gold atom with glowing blue orbs, in front of an ancient vault with a glowing doorway." />
    <meta name="twitter:card" content="summary_large_image" />
    `
	if diff := cmp.Diff(testutil.ShellPrefix+wantBlock+testutil.ShellSuffix, string(resp.Body)); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
}

func TestGetNewsPostPage_ReturnsExternalImageUnchanged_WhenBodyImageIsAbsolute(t *testing.T) {
	// given a published post whose first image is hosted elsewhere
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Clip', 'A clip.', 'guild-news', 'Officer',
			'![Clip](https://cdn.example.com/clip.png)', '2026-09-11T18:00:00Z')`)

	// when I ask for that post's page
	resp := srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111/page")

	// then I expect the external URL as the og image
	resp.RequireStatus(t, 200)
	if want := `<meta property="og:image" content="https://cdn.example.com/clip.png" />`; !strings.Contains(string(resp.Body), want) {
		t.Errorf("page does not contain %s:\n%s", want, resp.Body)
	}
}

func TestGetNewsPostPage_EscapesTitle_WhenTitleContainsMarkup(t *testing.T) {
	// given a published post whose title contains HTML
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', '<b>Raid</b>', 'Excerpt.', 'guild-news', 'Officer', 'Body.', '2026-09-11T18:00:00Z')`)

	// when I ask for that post's page
	resp := srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111/page")

	// then I expect the markup to be escaped in both the title element and the og:title tag
	resp.RequireStatus(t, 200)
	body := string(resp.Body)
	if want := `<title>&lt;b&gt;Raid&lt;/b&gt; | Fuzion</title>`; !strings.Contains(body, want) {
		t.Errorf("page does not contain %s:\n%s", want, body)
	}
	if want := `<meta property="og:title" content="&lt;b&gt;Raid&lt;/b&gt;" />`; !strings.Contains(body, want) {
		t.Errorf("page does not contain %s:\n%s", want, body)
	}
	if strings.Contains(body, "<b>Raid</b>") {
		t.Errorf("page contains the unescaped title:\n%s", body)
	}
}

func TestGetNewsPostPage_ReturnsShellWithDefaultTags_WhenPostIsDraft(t *testing.T) {
	// given a draft post
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, published_at)
		VALUES ('22222222-2222-2222-2222-222222222222', 'Secret plans', 'Not yet.', 'guild-news', 'Officer', NULL)`)

	// when I ask for that post's page
	resp := srv.Get(t, "/api/news/22222222-2222-2222-2222-222222222222/page")

	// then I expect a 404 carrying the untouched shell so the app can render its not-found view
	resp.RequireStatus(t, 404)
	if diff := cmp.Diff(testutil.ShellHTML, string(resp.Body)); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
}

func TestGetNewsPostPage_ReturnsShellWithDefaultTags_WhenPostDoesNotExist(t *testing.T) {
	// given no posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for an unknown post's page
	resp := srv.Get(t, "/api/news/99999999-9999-9999-9999-999999999999/page")

	// then I expect a 404 carrying the untouched shell
	resp.RequireStatus(t, 404)
	if diff := cmp.Diff(testutil.ShellHTML, string(resp.Body)); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
}

func TestGetNewsPostPage_ReturnsShellWithDefaultTags_WhenIDIsNotAUUID(t *testing.T) {
	// given no posts
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for a page with a malformed id
	resp := srv.Get(t, "/api/news/not-a-uuid/page")

	// then I expect a 404 carrying the untouched shell
	resp.RequireStatus(t, 404)
	if diff := cmp.Diff(testutil.ShellHTML, string(resp.Body)); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
}

func TestGetNewsPostPage_ReturnsBadGateway_WhenShellIsUnavailable(t *testing.T) {
	// given a published post and a static site that is down
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.Shell.Status = 503
	db.MustExec(`
		INSERT INTO news_posts (id, title, excerpt, category, author_name, body, published_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Patch day', 'Notes.', 'patch-notes', 'Officer', 'Body.', '2026-09-11T18:00:00Z')`)

	// when I ask for that post's page
	resp := srv.Get(t, "/api/news/11111111-1111-1111-1111-111111111111/page")

	// then I expect a 502
	resp.RequireStatus(t, 502)
}
