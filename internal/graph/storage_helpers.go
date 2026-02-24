package graph

import "trip2g/internal/graph/model"

func convertStorageSize(bytes int64, format *model.StorageSizeFormat) float64 {
	if format == nil {
		return float64(bytes)
	}

	switch *format {
	case model.StorageSizeFormatBytes:
		return float64(bytes)
	case model.StorageSizeFormatKb:
		return float64(bytes) / 1024
	case model.StorageSizeFormatMb:
		return float64(bytes) / (1024 * 1024)
	}

	return float64(bytes)
}
