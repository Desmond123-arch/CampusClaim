package pkg

import (
	"fmt"
	"strings"

	"github.com/Desmond123-arch/CampusClaim/models"
)

func GetRecentSearched(ItemTitle string) ([]string, error) {
	var tokens []string

	searchQuery := strings.ReplaceAll(ItemTitle, " ", " & ")
	fmt.Println(ItemTitle)
	err := models.DB.Raw(`
		SELECT DISTINCT token FROM (
			SELECT ut.token, rs.created_at
			FROM recent_searches rs
			JOIN user_tokens ut ON ut.user = rs.posted_by
			WHERE rs.search_tsv @@ to_tsquery('english', ?) AND ut.is_subscribed = true
			ORDER BY rs.created_at DESC
			LIMIT 50
		) AS ordered_results
		LIMIT 10
	`, searchQuery).Scan(&tokens).Error

	if err != nil {
		fmt.Println("Tokens error")
		return nil, err
	}
	fmt.Println(tokens)
	return tokens, nil
}
