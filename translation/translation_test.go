package translation

import "testing"

func TestRedisTranslation(t *testing.T) {
	addInfo()
	checkInfo()
	listItems("item1", "1", 10)
	checkInfo()
	buyItem("2", "item1", "1", 10)
	checkInfo()
}
