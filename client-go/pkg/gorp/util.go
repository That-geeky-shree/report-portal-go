package gorp

import (
	"fmt"
	"strconv"
	"strings"
)

// ConvertToFilterParams converts a FilterResource into query params for the ReportPortal API.
func ConvertToFilterParams(filter *FilterResource) map[string]string {
	params := map[string]string{}
	for _, f := range filter.Entities {
		params[fmt.Sprintf("filter.%s.%s", f.Condition, f.Field)] = f.Value
	}
	if filter.SelectionParams != nil {
		if filter.SelectionParams.PageNumber != 0 {
			params["page.page"] = strconv.Itoa(filter.SelectionParams.PageNumber)
		}
		// The RP API accepts repeated page.sort values. Encode them as a
		// comma-joined list in a single parameter to avoid map-key collisions
		// that would silently drop all but the last sort order.
		var sortParts []string
		for _, order := range filter.SelectionParams.Orders {
			sortParts = append(sortParts, fmt.Sprintf("%s,%s", order.SortingColumn, directionToStr(order.Asc)))
		}
		if len(sortParts) > 0 {
			params["page.sort"] = strings.Join(sortParts, "&page.sort=")
		}
	}
	return params
}

func directionToStr(asc bool) string {
	if asc {
		return "ASC"
	}
	return "DESC"
}
