package pagination

const (
	DefaultPage     int64 = 1
	DefaultPageSize int64 = 50
	MaxPageSize     int64 = 200
)

type Params struct {
	Page     int64
	PageSize int64
}

type Result[T any] struct {
	Items    []T
	Page     int64
	PageSize int64
	Total    int64
}

func Normalize(params Params) Params {
	if params.Page <= 0 {
		params.Page = DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = DefaultPageSize
	}
	if params.PageSize > MaxPageSize {
		params.PageSize = MaxPageSize
	}

	return params
}

func Offset(params Params) int64 {
	params = Normalize(params)
	return (params.Page - 1) * params.PageSize
}

func FromSlice[T any](items []T, params Params) Result[T] {
	params = Normalize(params)
	total := int64(len(items))
	offset := Offset(params)
	if offset >= total {
		return Result[T]{
			Items:    []T{},
			Page:     params.Page,
			PageSize: params.PageSize,
			Total:    total,
		}
	}

	end := offset + params.PageSize
	if end > total {
		end = total
	}

	return Result[T]{
		Items:    items[offset:end],
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}
}

func NewResult[T any](items []T, params Params, total int64) Result[T] {
	params = Normalize(params)
	if items == nil {
		items = []T{}
	}
	if total < 0 {
		total = 0
	}

	return Result[T]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}
}
