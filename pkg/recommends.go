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
	
	// Using subquery with GORM
	subQuery := models.DB.Table("recent_searches rs").
		Select("ut.token, rs.created_at").
		Joins("JOIN user_tokens ut ON ut.user = rs.posted_by").
		Where("rs.search_tsv @@ to_tsquery('english', ?) AND ut.is_subscribed = true", searchQuery).
		Order("rs.created_at DESC").
		Limit(50)

	err := models.DB.Table("(?) AS ordered_results", subQuery).
		Select("DISTINCT token").
		Limit(10).
		Pluck("token", &tokens).Error

	if err != nil {
		fmt.Println("Tokens error:", err)
		return nil, err
	}
	fmt.Println("Found tokens:", tokens)
	return tokens, nil
}