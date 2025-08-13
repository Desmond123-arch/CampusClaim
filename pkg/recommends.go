package pkg

import (
	"github.com/Desmond123-arch/CampusClaim/models"
)

/*
* get all 10 recent users that have searched this keyword
 */
func GetRecentSearched(ItemTitle string) ([]string, error) {
	var tokens []string

	err := models.DB.
		Table("recent_searches rs").
		Select("DISTINCT ut.token").
		Joins("JOIN user_tokens ut ON ut.user_id = rs.posted_by").
		Where("rs.search_tsv @@ to_tsquery('english', ?) AND ut.is_subscribed = true", ItemTitle).
		Pluck("ut.token", &tokens).Error

	return tokens, err
}
