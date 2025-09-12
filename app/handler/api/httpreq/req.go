package httpreq

type FindReq struct {
	Pagination
}

type Pagination struct {
	PageIndex int `form:"pageIndex"`
	PageSize  int `form:"pageSize"`
}
