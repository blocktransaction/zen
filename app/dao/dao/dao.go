package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	// 可选：开启 total 缓存时需要
	"github.com/redis/go-redis/v9"
)

// -------- 条件与 JOIN --------

type Condition struct {
	Field string
	Op    string
	Value any
}

type ConditionGroup struct {
	And []any // 元素为 Condition 或 ConditionGroup
	Or  []any
}

type Join struct {
	Table string
	On    string
	Type  string // "INNER JOIN" / "LEFT JOIN"
}

// -------- DAO 定义 --------

type DAO[T any] struct {
	db       *gorm.DB
	rdb      *redis.Client // 可为 nil（未使用缓存时）
	conds    []any
	selects  []string
	orderBy  []string
	joins    []*Join
	limit    int
	offset   int
	unscoped bool
	err      error
}

// --- 构造 ---

func NewDAO[T any](ctx context.Context, db *gorm.DB) *DAO[T] {
	return &DAO[T]{
		db: db.WithContext(ctx),
	}
}

func NewDAOWithRdb[T any](ctx context.Context, db *gorm.DB, rdb *redis.Client) *DAO[T] {
	return &DAO[T]{
		db:  db.WithContext(ctx),
		rdb: rdb,
	}
}

func (d *DAO[T]) clone() *DAO[T] {
	cp := &DAO[T]{
		db:       d.db,
		rdb:      d.rdb,
		conds:    append([]any{}, d.conds...),
		selects:  append([]string{}, d.selects...),
		orderBy:  append([]string{}, d.orderBy...),
		limit:    d.limit,
		offset:   d.offset,
		unscoped: d.unscoped,
		err:      d.err,
	}
	if len(d.joins) > 0 {
		cp.joins = append([]*Join{}, d.joins...)
	}
	return cp
}

// --- 链式构建 ---
func (d *DAO[T]) Select(fields ...string) *DAO[T] {
	nd := d.clone()
	nd.selects = append(nd.selects, fields...)
	return nd
}

func (d *DAO[T]) OrderBy(order string) *DAO[T] {
	nd := d.clone()
	nd.orderBy = append(nd.orderBy, order)
	return nd
}

func (d *DAO[T]) Paginate(pageIndex, pageSize int) *DAO[T] {
	nd := d.clone()
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize > 0 {
		nd.limit = pageSize
		nd.offset = (pageIndex - 1) * pageSize
	}
	return nd
}

func (d *DAO[T]) WithDeleted() *DAO[T] {
	nd := d.clone()
	nd.unscoped = true
	return nd
}

// where条件
func (d *DAO[T]) Where(field, op string, value any) *DAO[T] {
	nd := d.clone()
	nd.conds = append(nd.conds, Condition{Field: field, Op: normalizeOp(op), Value: value})
	return nd
}

// wheremap条件映射
func (d *DAO[T]) WhereMap(m map[string]any) *DAO[T] {
	nd := d.clone()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f, op := parseKey(k) // 你的 parseKey/normalizeOp 保持不变
		nd.conds = append(nd.conds, Condition{Field: f, Op: op, Value: m[k]})
	}
	return nd
}

// andgroup
func (d *DAO[T]) AndGroup(conds ...any) *DAO[T] {
	nd := d.clone()
	nd.conds = append(nd.conds, ConditionGroup{And: conds})
	return nd
}

// Count 把数量写入传入的指针，保持链式调用
func (d *DAO[T]) Count(count *int64) *DAO[T] {
	if d.err != nil {
		return d
	}
	var c int64
	tx := d.buildQuery(d.db).Count(&c)
	if tx.Error != nil {
		d.err = tx.Error
		return d
	}
	*count = c
	return d
}

// orgroup
func (d *DAO[T]) OrGroup(conds ...any) *DAO[T] {
	nd := d.clone()
	nd.conds = append(nd.conds, ConditionGroup{Or: conds})
	return nd
}

// join
func (d *DAO[T]) Join(joinType, table, on string) *DAO[T] {
	nd := d.clone()
	nd.joins = append(nd.joins, &Join{Table: table, On: on, Type: joinType})
	return nd
}

func (d *DAO[T]) InnerJoin(table, on string) *DAO[T] { return d.Join("INNER JOIN", table, on) }
func (d *DAO[T]) LeftJoin(table, on string) *DAO[T]  { return d.Join("LEFT JOIN", table, on) }

// --- 便捷条件 ---

func (d *DAO[T]) Eq(field string, v any) *DAO[T]  { return d.Where(field, "=", v) }
func (d *DAO[T]) Ne(field string, v any) *DAO[T]  { return d.Where(field, "!=", v) }
func (d *DAO[T]) Gt(field string, v any) *DAO[T]  { return d.Where(field, ">", v) }
func (d *DAO[T]) Gte(field string, v any) *DAO[T] { return d.Where(field, ">=", v) }
func (d *DAO[T]) Lt(field string, v any) *DAO[T]  { return d.Where(field, "<", v) }
func (d *DAO[T]) Lte(field string, v any) *DAO[T] { return d.Where(field, "<=", v) }
func (d *DAO[T]) Like(field string, pat string) *DAO[T] {
	return d.Where(field, "LIKE", pat)
}
func (d *DAO[T]) In(field string, list any) *DAO[T] { return d.Where(field, "IN", list) }

// -------- 执行方法 --------

func (d *DAO[T]) Find(out *[]T) error {
	if d.err != nil {
		return d.err
	}
	tx := d.buildQuery(d.db)
	return tx.Find(out).Error
}

// first
func (d *DAO[T]) First(out *T) error {
	if d.err != nil {
		return d.err
	}
	tx := d.buildQuery(d.db)
	return tx.First(out).Error
}

// count
func (d *DAO[T]) count() (int64, error) {
	if d.err != nil {
		return 0, d.err
	}
	var count int64
	tx := d.buildQuery(d.db).Count(&count)
	return count, tx.Error
}

// CRUD（根据当前条件）
func (d *DAO[T]) Create(entity *T) error {
	return d.db.Create(entity).Error
}

func (d *DAO[T]) CreateBatch(entities []T) error {
	return d.db.Create(&entities).Error
}

func (d *DAO[T]) Update(fields map[string]any) error {
	tx := d.buildQuery(d.db)
	return tx.Updates(fields).Error
}

func (d *DAO[T]) Delete() error {
	tx := d.buildQuery(d.db)
	return tx.Delete(new(T)).Error
}

// 软删恢复（在条件下把 deleted_at 设回 NULL）
func (d *DAO[T]) Restore() error {
	tx := d.buildQuery(d.db.Unscoped())
	return tx.Model(new(T)).Update("deleted_at", nil).Error
}

// 事务（闭包形式）
func (d *DAO[T]) WithTx(fn func(txDAO *DAO[T]) error) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		return fn(&DAO[T]{
			db:       tx,
			rdb:      d.rdb,
			conds:    nil,
			selects:  nil,
			orderBy:  nil,
			joins:    nil,
			limit:    0,
			offset:   0,
			unscoped: false,
		})
	})
}

// -------- 内部：Query 构建 --------

func (d *DAO[T]) buildQuery(tx *gorm.DB) *gorm.DB {
	if d.unscoped {
		tx = tx.Unscoped()
	}
	// 基表
	tx = tx.Model(new(T))

	// JOIN
	for _, j := range d.joins {
		joinClause := fmt.Sprintf("%s %s ON %s", j.Type, j.Table, j.On)
		tx = tx.Joins(joinClause)
	}

	// SELECT
	if len(d.selects) > 0 {
		tx = tx.Select(d.selects)
	}

	// WHERE（递归）
	if len(d.conds) > 0 {
		tx = applyConditions(tx, d.conds)
	}

	// ORDER
	for _, o := range d.orderBy {
		tx = tx.Order(o)
	}

	// LIMIT/OFFSET
	if d.limit > 0 {
		tx = tx.Limit(d.limit).Offset(d.offset)
	}

	return tx
}

func applyConditions(tx *gorm.DB, conds []any) *gorm.DB {
	for _, c := range conds {
		switch v := c.(type) {
		case Condition:
			op := strings.ToUpper(v.Op)
			if op == "IN" {
				tx = tx.Where(fmt.Sprintf("%s IN (?)", v.Field), v.Value)
			} else if v.Value == nil {
				// NULL 处理：= NULL/!= NULL -> IS (NOT) NULL
				switch op {
				case "=":
					tx = tx.Where(fmt.Sprintf("%s IS NULL", v.Field))
				case "!=":
					tx = tx.Where(fmt.Sprintf("%s IS NOT NULL", v.Field))
				default:
					tx = tx.Where(fmt.Sprintf("%s %s NULL", v.Field, op))
				}
			} else {
				tx = tx.Where(fmt.Sprintf("%s %s ?", v.Field, op), v.Value)
			}
		case ConditionGroup:
			tx = tx.Where(func(s *gorm.DB) *gorm.DB {
				// AND 子组
				if len(v.And) > 0 {
					s = applyConditions(s, v.And)
				}
				// OR 子组
				if len(v.Or) > 0 {
					s = s.Or(func(orDB *gorm.DB) *gorm.DB {
						return applyConditions(orDB, v.Or)
					})
				}
				return s
			})
		}
	}
	return tx
}

// -------- 键解析 / 操作符映射 --------

var opMap = map[string]string{
	"eq":   "=",
	"ne":   "!=",
	"gt":   ">",
	"lt":   "<",
	"gte":  ">=",
	"lte":  "<=",
	"like": "LIKE",
	"in":   "IN",
}

func parseKey(k string) (field, sqlOp string) {
	parts := strings.Split(k, "__")
	field = parts[0]
	if len(parts) == 1 {
		return field, "="
	}
	if op, ok := opMap[strings.ToLower(parts[1])]; ok {
		return field, op
	}
	// 允许直接写原生操作符
	return field, normalizeOp(parts[1])
}

func normalizeOp(op string) string {
	if sql, ok := opMap[strings.ToLower(op)]; ok {
		return sql
	}
	return strings.ToUpper(op)
}

// -------- 可选：分页 total 缓存（Redis） --------

type PageResult[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
}

// PaginateWithCache：若 rdb 为 nil 则退化为 DB count
func (d *DAO[T]) PaginateWithCache(ctx context.Context, table string, page, pageSize int, ttl time.Duration) (*PageResult[T], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	// 先算 total
	total, err := d.cachedCount(ctx, table, ttl)
	if err != nil {
		return nil, err
	}
	// 再取数据页
	nd := d.Paginate(page, pageSize)
	var list []T
	if err := nd.Find(&list); err != nil {
		return nil, err
	}
	return &PageResult[T]{Total: total, List: list}, nil
}

// 简单 total 缓存：以条件/排序/选择做指纹。表写入后请外部清理相关 key 或使用表版本号策略（可按需扩展）。
func (d *DAO[T]) cachedCount(ctx context.Context, table string, ttl time.Duration) (int64, error) {
	if d.rdb == nil {
		return d.count()
	}
	sqlStr, args := d.renderWhereFingerprint() // 指纹化
	key := "count:" + table + ":" + sqlStr + ":" + fmt.Sprint(args...)
	// 查缓存
	if s, err := d.rdb.Get(ctx, key).Result(); err == nil {
		var v int64
		fmt.Sscan(s, &v)
		return v, nil
	}
	// 走 DB
	cnt, err := d.count()
	if err != nil {
		return 0, err
	}
	_ = d.rdb.Set(ctx, key, fmt.Sprintf("%d", cnt), ttl).Err()
	return cnt, nil
}

// 仅用于做指纹（避免生成实际 SQL）：平铺 conds 成字符串
// import 需要： "fmt", "reflect", "sort", "strings"

func (d *DAO[T]) renderWhereFingerprint() (string, []any) {
	sql, args := renderConds(d.conds, "AND")
	return sql, args
}

// 递归渲染一组条件，使用指定连接词（"AND"/"OR"）
func renderConds(conds []any, joiner string) (string, []any) {
	parts := make([]string, 0, len(conds))
	args := make([]any, 0, 8)

	for _, c := range conds {
		switch v := c.(type) {
		case Condition:
			s, a := renderCond(v)
			parts = append(parts, s)
			args = append(args, a...)
		case ConditionGroup:
			// 同时存在 And/Or 时，语义为 ( (A1 AND A2 ...) OR (O1 OR O2 ...) )
			var sect []string
			var sectArgs []any
			if len(v.And) > 0 {
				s, a := renderConds(v.And, "AND")
				sect = append(sect, "("+s+")")
				sectArgs = append(sectArgs, a...)
			}
			if len(v.Or) > 0 {
				s, a := renderConds(v.Or, "OR")
				sect = append(sect, "("+s+")")
				sectArgs = append(sectArgs, a...)
			}
			if len(sect) == 0 {
				// 空组：跳过
				continue
			}
			parts = append(parts, "("+strings.Join(sect, " OR ")+")")
			args = append(args, sectArgs...)
		}
	}
	return strings.Join(parts, " "+joiner+" "), args
}

func renderCond(c Condition) (string, []any) {
	op := strings.ToUpper(c.Op)
	switch op {
	case "IN":
		vs := flattenSlice(c.Value)
		// 空 IN => 不匹配：稳定地渲染为 1=0，避免 IN ()
		if len(vs) == 0 {
			return "1=0", nil
		}
		ph := strings.TrimRight(strings.Repeat("?,", len(vs)), ",")
		return fmt.Sprintf("%s IN (%s)", c.Field, ph), vs
	default:
		// NULL 语义与实际查询保持一致
		if c.Value == nil {
			if op == "=" {
				return fmt.Sprintf("%s IS NULL", c.Field), nil
			}
			if op == "!=" {
				return fmt.Sprintf("%s IS NOT NULL", c.Field), nil
			}
			return fmt.Sprintf("%s %s NULL", c.Field, op), nil
		}
		return fmt.Sprintf("%s %s ?", c.Field, op), []any{c.Value}
	}
}

// 扁平化任意切片/数组为 []any
func flattenSlice(v any) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []any{v}
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out
}

// -------- 小工具：健康检查（可选） --------

func PingDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func WaitDB(db *gorm.DB, timeout time.Duration) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		if err := sqlDB.Ping(); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("wait db timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func Transactional(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}
