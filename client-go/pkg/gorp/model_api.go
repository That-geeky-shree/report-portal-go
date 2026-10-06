package gorp

// Response is a generic paginated response wrapper.
type Response struct {
	Page struct {
		Number        int `json:"number,omitempty"`
		Size          int `json:"size,omitempty"`
		TotalElements int `json:"totalElements,omitempty"`
		TotalPages    int `json:"totalPages,omitempty"`
	} `json:"page,omitempty"`
}

// LaunchResource is the GET launch response model.
type LaunchResource struct {
	ID                  int          `json:"id"`
	UUID                string       `json:"uuid"`
	Name                string       `json:"name,omitempty"`
	Number              int          `json:"number"`
	Description         string       `json:"description,omitempty"`
	StartTime           Timestamp    `json:"startTime,omitempty"`
	EndTime             Timestamp    `json:"endTime,omitempty"`
	Status              Status       `json:"status,omitempty"`
	Attributes          []*Attribute `json:"attributes,omitempty"`
	Mode                LaunchMode   `json:"mode,omitempty"`
	ApproximateDuration float32      `json:"approximateDuration,omitempty"`
	HasRetries          bool         `json:"hasRetries,omitempty"`
	Statistics          *Statistics  `json:"statistics,omitempty"`
	Analyzers           []string     `json:"analysing,omitempty"` //nolint:misspell // matches server-side field name
}

// Statistics holds execution and defect statistics for a launch.
type Statistics struct {
	Executions map[string]int            `json:"executions,omitempty"`
	Defects    map[string]map[string]int `json:"defects,omitempty"`
}

// FilterResource is the GET filter response model.
type FilterResource struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Type            TestItemType          `json:"type"`
	Owner           string                `json:"owner"`
	Entities        []*FilterEntity       `json:"entities"`
	SelectionParams *FilterSelectionParam `json:"selection_parameters,omitempty"`
}

// FilterEntity is one condition in a saved filter.
type FilterEntity struct {
	Field     string `json:"filtering_field"`
	Condition string `json:"condition"`
	Value     string `json:"value"`
}

// FilterSelectionParam describes filter paging and ordering.
type FilterSelectionParam struct {
	PageNumber int            `json:"page_number"`
	Orders     []*FilterOrder `json:"orders,omitempty"`
}

// FilterOrder describes sort direction for a filter.
type FilterOrder struct {
	SortingColumn string `json:"sorting_column"`
	Asc           bool   `json:"is_asc"`
}

// FilterPage is the paginated GET filter response.
type FilterPage struct {
	Content []*FilterResource
	Response
}

// LaunchPage is the paginated GET launch response.
type LaunchPage struct {
	Content []*LaunchResource
	Response
}

// MergeLaunchesRQ is the request payload for merging launches.
type MergeLaunchesRQ struct {
	Description             string     `json:"description,omitempty"`
	StartTime               *Timestamp `json:"startTime,omitempty"`
	EndTime                 *Timestamp `json:"endTime,omitempty"`
	ExtendSuitesDescription bool       `json:"extendSuitesDescription,omitempty"`
	Launches                []int      `json:"launches"`
	MergeType               MergeType  `json:"mergeType,omitempty"`
	Mode                    string     `json:"mode,omitempty"`
	Name                    string     `json:"name,omitempty"`
}
