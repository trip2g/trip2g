package mdloader

import "fmt"

func Subgraphs(pages map[string]*Page) ([]string, error) {
	subgraphs := make(map[string]struct{})

	for _, page := range pages {
		err := extractSubgraphs(subgraphs, page.RawMeta["subgraph"])
		if err != nil {
			return nil, fmt.Errorf("error extracting subgraph: %w", err)
		}

		err = extractSubgraphs(subgraphs, page.RawMeta["subgraphs"])
		if err != nil {
			return nil, fmt.Errorf("error extracting subgraphs: %w", err)
		}
	}

	res := make([]string, 0, len(subgraphs))

	for k := range subgraphs {
		res = append(res, k)
	}

	return res, nil
}

func extractSubgraphs(target map[string]struct{}, val interface{}) error {
	switch val := val.(type) {
	case string:
		target[val] = struct{}{}
	case []interface{}:
		for _, v := range val {
			if vStr, ok := v.(string); ok {
				target[vStr] = struct{}{}
			} else {
				return fmt.Errorf("invalid subgraph type: %T", v)
			}
		}
	case nil:
		return nil
	default:
		return fmt.Errorf("invalid subgraph type: %T", val)
	}

	return nil
}
