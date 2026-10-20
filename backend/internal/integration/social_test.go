package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/resource"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
)

// The test creates and drops an isolated schema; existing application data is untouched.
func TestSocialEndpointsPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "test_social_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
	}()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	db, err := postgres.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, file := range []string{"init/init.sql", "0004_friends_posts_feed.sql"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			t.Fatal(err)
		}
	}
	alice, bob := uuid.New(), uuid.New()
	aliceToken, bobToken := uuid.New(), uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, first_name, second_name, password_hash) VALUES ($1,'A','A','x'),($2,'B','B','x')`, alice, bob)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO auth_tokens (token, user_id, expires_at) VALUES ($1,$2,now()+interval '1 hour'),($3,$4,now()+interval '1 hour')`, aliceToken, alice, bobToken, bob)
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := postgres.NewCluster(ctx, u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cluster.Close()
	users := postgres.NewUserRepo(cluster)
	tokens := postgres.NewTokenRepo(cluster)
	router := resource.NewRouter(service.NewUserService(users), service.NewAuthService(users, tokens), service.NewLoadTestService(postgres.NewLoadTestRepo(cluster)), service.NewSocialService(postgres.NewSocialRepo(cluster)), tokens)
	request := func(method, path, token, body string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: got %d %s, want %d", method, path, w.Code, w.Body.String(), want)
		}
		if want == 400 || want == 404 || want == 500 {
			if w.Header().Get("Content-Type") != "application/json" || w.Body.String() != "null\n" {
				t.Fatalf("%s %s: unexpected error response: content type %q, body %q", method, path, w.Header().Get("Content-Type"), w.Body.String())
			}
		}
		return w.Body.String()
	}
	revision := func(user uuid.UUID, want int) {
		t.Helper()
		var got int
		err := db.QueryRowContext(ctx, `SELECT revision FROM feed_revisions WHERE user_id=$1`, user).Scan(&got)
		if errorsIsNoRows(err) && want == 0 {
			return
		}
		if err != nil || got != want {
			t.Fatalf("revision for %s = %d, %v; want %d", user, got, err, want)
		}
	}
	request("PUT", "/friend/set/"+bob.String(), "", "", 401)
	request("PUT", "/friend/set/not-a-uuid", aliceToken.String(), "", 400)
	request("PUT", "/friend/set/"+alice.String(), aliceToken.String(), "", 400)
	request("PUT", "/friend/set/"+uuid.NewString(), aliceToken.String(), "", 404)
	request("PUT", "/friend/set/"+bob.String(), aliceToken.String(), "", 200)
	request("PUT", "/friend/set/"+bob.String(), aliceToken.String(), "", 200)
	revision(alice, 1)
	revision(bob, 0)
	request("POST", "/post/create", bobToken.String(), `{}`, 400)
	body := request("POST", "/post/create", bobToken.String(), `{"text":"hello"}`, 200)
	var postID string
	if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&postID); err != nil {
		t.Fatal(err)
	}
	revision(alice, 2)
	got := request("GET", "/post/get/"+postID, "", "", 200)
	var post struct {
		ID     string `json:"id"`
		Text   string `json:"text"`
		Author string `json:"author_user_id"`
	}
	if err := json.Unmarshal([]byte(got), &post); err != nil || post.ID != postID || post.Text != "hello" || post.Author != bob.String() {
		t.Fatalf("unexpected post: %s, %v", got, err)
	}
	request("PUT", "/post/update", aliceToken.String(), fmt.Sprintf(`{"id":%q,"text":"wrong"}`, postID), 404)
	request("PUT", "/post/delete/"+postID, aliceToken.String(), "", 404)
	revision(alice, 2)
	// An invalidation failure must roll back the post mutation as well.
	if _, err := db.ExecContext(ctx, `UPDATE feed_revisions SET revision = 9223372036854775807 WHERE user_id = $1`, alice); err != nil {
		t.Fatal(err)
	}
	request("PUT", "/post/update", bobToken.String(), fmt.Sprintf(`{"id":%q,"text":"rolled back"}`, postID), 500)
	var storedText string
	if err := db.QueryRowContext(ctx, `SELECT text FROM posts WHERE id = $1`, postID).Scan(&storedText); err != nil || storedText != "hello" {
		t.Fatalf("post after failed revision update: %q, %v", storedText, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE feed_revisions SET revision = 2 WHERE user_id = $1`, alice); err != nil {
		t.Fatal(err)
	}
	request("PUT", "/post/update", bobToken.String(), fmt.Sprintf(`{"id":%q,"text":"updated"}`, postID), 200)
	revision(alice, 3)
	request("PUT", "/post/delete/"+postID, bobToken.String(), "", 200)
	revision(alice, 4)
	request("GET", "/post/get/"+postID, "", "", 404)
	request("PUT", "/friend/delete/"+bob.String(), aliceToken.String(), "", 200)
	request("PUT", "/friend/delete/"+bob.String(), aliceToken.String(), "", 200)
	revision(alice, 5)
	request("POST", "/post/create", bobToken.String(), `{"text":"after removal"}`, 200)
	revision(alice, 5)
}

func errorsIsNoRows(err error) bool { return err == sql.ErrNoRows }
