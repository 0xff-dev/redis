package vote

import "testing"

func TestPostArticle(t *testing.T) {
	if articleId, err := postArticle("coco", "测试文章", "https://stevenshuang.github.io/2020/09/13/1/"); err != nil {
		t.Fatalf("add article error: %s", err)
	} else {
		t.Logf("add successfully, id: %s", articleId)
	}

}

func TestGetArticle(t *testing.T) {
	for _, item := range getArticle(1, "score:") {
		t.Log(item)
	}
}

func TestVoteArticle(t *testing.T) {
	if err := voteArticle("coco", "article:1"); err != nil {
		t.Fatalf("vote for article article:1 error: %s", err)
	}
	t.Log("vote successfully!")
	for _, item := range getArticle(1, "score:") {
		t.Log(item)
	}
}

func TestAddGroup(t *testing.T) {
	for _, id := range []string{"2", "3"} {
		addOrRemoveGroups(id, []string{"one", "two"}, nil)
	}
}

func TestGetGroupArticles(t *testing.T) {
	for _, g := range []string{"one", "two"} {
		t.Log(getGroupArticles(1, g, "score:"))
	}
}
