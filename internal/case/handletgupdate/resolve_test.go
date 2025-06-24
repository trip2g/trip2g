package handletgupdate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateMBTI(t *testing.T) {
	testAnswers := map[int]int{
		1: -1, 10: 1, 11: -2, 12: -2, 13: 3, 14: 2, 15: 3, 16: -1, 17: 3, 18: -2,
		19: 3, 2: 3, 20: 2, 21: 3, 22: -2, 23: 3, 24: 3, 25: 3, 26: 2, 27: -2,
		28: -3, 29: 2, 3: -2, 30: 3, 31: -3, 32: -3, 33: -2, 34: 2, 35: -1,
		36: -1, 37: 3, 38: 3, 39: 3, 4: -1, 40: 1, 41: 3, 42: 3, 43: -2, 44: 3,
		45: 3, 46: -2, 47: -2, 48: -2, 49: 2, 5: 3, 50: 1, 51: 3, 52: -3, 53: -3,
		54: 3, 55: 1, 56: -2, 57: 3, 58: -2, 59: 3, 6: 1, 60: 2, 7: -1, 8: -2, 9: 3,
	}

	rawQuestions := []byte(
		`[{"ID":1,"Text":"","Category":"EI"},{"ID":2,"Text":"","Category":"NS"},{"ID":3,"Text":"","Category":"FT"},{"ID":4,"Text":"","Category":"JP"},{"ID":5,"Text":"","Category":"AR"},{"ID":6,"Text":"","Category":"IE"},{"ID":7,"Text":"","Category":"JP"},{"ID":8,"Text":"","Category":"FT"},{"ID":9,"Text":"","Category":"JP"},{"ID":10,"Text":"","Category":"RA"},{"ID":11,"Text":"","Category":"EI"},{"ID":12,"Text":"","Category":"SN"},{"ID":13,"Text":"","Category":"TF"},{"ID":14,"Text":"","Category":"PJ"},{"ID":15,"Text":"","Category":"AR"},{"ID":16,"Text":"","Category":"EI"},{"ID":17,"Text":"","Category":"NS"},{"ID":18,"Text":"","Category":"FT"},{"ID":19,"Text":"","Category":"NS"},{"ID":20,"Text":"","Category":"RA"},{"ID":21,"Text":"","Category":"IE"},{"ID":22,"Text":"","Category":"SN"},{"ID":23,"Text":"","Category":"TF"},{"ID":24,"Text":"","Category":"JP"},{"ID":25,"Text":"","Category":"TF"},{"ID":26,"Text":"","Category":"IE"},{"ID":27,"Text":"","Category":"RA"},{"ID":28,"Text":"","Category":"TF"},{"ID":29,"Text":"","Category":"PJ"},{"ID":30,"Text":"","Category":"NS"},{"ID":31,"Text":"","Category":"EI"},{"ID":32,"Text":"","Category":"SN"},{"ID":33,"Text":"","Category":"FT"},{"ID":34,"Text":"","Category":"PJ"},{"ID":35,"Text":"","Category":"AR"},{"ID":36,"Text":"","Category":"EI"},{"ID":37,"Text":"","Category":"NS"},{"ID":38,"Text":"","Category":"TF"},{"ID":39,"Text":"","Category":"JP"},{"ID":40,"Text":"","Category":"AR"},{"ID":41,"Text":"","Category":"IE"},{"ID":42,"Text":"","Category":"NS"},{"ID":43,"Text":"","Category":"EI"},{"ID":44,"Text":"","Category":"JP"},{"ID":45,"Text":"","Category":"RA"},{"ID":46,"Text":"","Category":"SN"},{"ID":47,"Text":"","Category":"RA"},{"ID":48,"Text":"","Category":"FT"},{"ID":49,"Text":"","Category":"PJ"},{"ID":50,"Text":"","Category":"RA"},{"ID":51,"Text":"","Category":"IE"},{"ID":52,"Text":"","Category":"SN"},{"ID":53,"Text":"","Category":"EI"},{"ID":54,"Text":"","Category":"FT"},{"ID":55,"Text":"","Category":"RA"},{"ID":56,"Text":"","Category":"JP"},{"ID":57,"Text":"","Category":"NS"},{"ID":58,"Text":"","Category":"FT"},{"ID":59,"Text":"","Category":"PJ"},{"ID":60,"Text":"","Category":"AR"}]`,
	)

	var questions []struct {
		ID       int    `json:"ID"`
		Text     string `json:"Text"`
		Category string `json:"Category"`
	}

	err := json.Unmarshal(rawQuestions, &questions)
	require.NoError(t, err)

	// Convert to Question type
	var questionsTyped []Question
	for _, q := range questions {
		questionsTyped = append(questionsTyped, Question{
			ID:       q.ID,
			Text:     q.Text,
			Category: q.Category,
		})
	}

	res := calculateMBTI(questionsTyped, testAnswers)
	require.Equal(t, "INTP-A", res.Name)

	// Verify that we have all 5 categories
	require.Len(t, res.Categories, 5)

	// Verify percentages are between 0 and 1
	for category, percentage := range res.Categories {
		require.GreaterOrEqual(t, percentage, float32(0.0), "Category %s percentage should be >= 0", category)
		require.LessOrEqual(t, percentage, float32(1.0), "Category %s percentage should be <= 1", category)
	}
}
