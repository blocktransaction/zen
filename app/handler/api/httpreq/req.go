package httpreq

type FindReq struct {
	PageIndex int `form:"pageIndex"`
	PageSize  int `form:"pageSize"`
}
