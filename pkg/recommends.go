package pkg

import (
	"fmt"
	"strings"

	"github.com/Desmond123-arch/CampusClaim/models"
)

func GetRecentSearched(ItemTitle string) ([]string, error) {
	var tokens []string

	searchQuery := strings.ReplaceAll(ItemTitle, " ", " & ")
	fmt.Println("Search Query:", searchQuery)
	
	err := models.DB.Raw(`
		SELECT DISTINCT ut.token
		FROM recent_searches rs
		JOIN user_tokens ut ON ut.user = rs.posted_by
		WHERE rs.search_tsv @@ to_tsquery('english', ?) 
		AND ut.is_subscribed = true
		LIMIT 10
	`, searchQuery).Scan(&tokens).Error

	if err != nil {
		fmt.Println("Tokens error:", err)
		return nil, err
	}
	fmt.Println("Found tokens:", tokens)
	return tokens, nil
}