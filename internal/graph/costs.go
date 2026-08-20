package graph

import (
	"encoding/json"
	"sort"

	"trip2g/internal/graph/model"
)

// decodeCosts turns a delivery's stored cost object into the API's list of
// {unit, amount} pairs. GraphQL cannot express an open map, and the set of units
// is open by design — an LLM agent reports tokens, a billing one reports money —
// so the unit rides in the id. A malformed object yields an empty list rather
// than an error: a careless agent must not break the page that shows its run.
func decodeCosts(raw *string) []model.AdminCost {
	if raw == nil || *raw == "" {
		return []model.AdminCost{}
	}
	var costs map[string]float64
	if err := json.Unmarshal([]byte(*raw), &costs); err != nil {
		return []model.AdminCost{}
	}
	return sortedCosts(costs)
}

// sumCosts adds up several cost objects per unit. Chains mix executors, so a
// chain total is not one number but one number per unit that appeared in it.
func sumCosts(raws []*string) []model.AdminCost {
	total := map[string]float64{}
	for _, raw := range raws {
		if raw == nil || *raw == "" {
			continue
		}
		var costs map[string]float64
		if err := json.Unmarshal([]byte(*raw), &costs); err != nil {
			continue
		}
		for unit, amount := range costs {
			total[unit] += amount
		}
	}
	return sortedCosts(total)
}

// sortedCosts gives the list a stable order — map iteration would otherwise
// reshuffle the columns of the same run between two page loads.
func sortedCosts(costs map[string]float64) []model.AdminCost {
	units := make([]string, 0, len(costs))
	for unit := range costs {
		units = append(units, unit)
	}
	sort.Strings(units)

	out := make([]model.AdminCost, 0, len(units))
	for _, unit := range units {
		out = append(out, model.AdminCost{ID: unit, Value: costs[unit]})
	}
	return out
}
