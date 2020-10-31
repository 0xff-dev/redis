package auto_complete

import (
	"log"
	"testing"
)

func TestAutoCompleteByCode(t *testing.T) {
	contacts := []string{"aaa", "bbb", "ccc", "ddd", "eee", "fff", "iii", "ooo"}
	user := "coco"
	for _, c := range contacts {
		addUpdateContact(user, c)
	}

	t.Log("fetch prefix: e")
	log.Println(fetchAutoComplete(user, "e"))
}

func TestAutoCompleteByRedisZset(t *testing.T) {
	groupUsers := []string{"asd", "bbb", "abcggg", "abcrer", "abd", "abcd", "aaa", "abcff"}
	groupId := "coco"
	t.Log("join users to group coco: ", groupUsers)
	for _, gu := range groupUsers {
		joinOrLeaveGuild(groupId, gu, true)
	}

	t.Log(autoCompleteByPrefix("abc", groupId))

}
