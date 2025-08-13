package pkg

import (
	"strings"

	"github.com/Desmond123-arch/CampusClaim/models"
)

/*
* get all 10 recent users that have searched this keyword
 */
 func GetRecentSearched(ItemTitle string) ([]string, error) {
    var tokens []string

    // Sanitize ItemTitle for to_tsquery
    tsQuery := strings.ReplaceAll(ItemTitle, " ", " & ")

    err := models.DB.
        Table("recent_searches rs").
        Select("DISTINCT ut.token").
        Joins("JOIN user_tokens ut ON ut.user_id = rs.posted_by").
        Where("rs.search_tsv @@ to_tsquery('english', ?) AND ut.is_subscribed = true", tsQuery).
        Pluck("ut.token", &tokens).Error

    return tokens, err
}
