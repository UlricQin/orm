package orm

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Repo 封装一次查询
type Repo struct {
	o       *Orm
	tbl     string
	db      *sql.DB
	where   string
	args    []interface{}
	orderBy string
	limit   int
	offset  int
	showSQL bool
	cols    string
	sql     string
}

// Use 使用哪个数据库实例
func (r *Repo) Use(name string) *Repo {
	r.db = r.o.Use(name)
	return r
}

// Where 查询条件
func (r *Repo) Where(where string, args ...interface{}) *Repo {
	r.where = where
	r.args = args
	return r
}

// OrderBy e.g. name => order by name
func (r *Repo) OrderBy(by string) *Repo {
	r.orderBy = by
	return r
}

// Limit 设置limit和offset
func (r *Repo) Limit(limit int, offset ...int) *Repo {
	r.limit = limit
	if len(offset) > 0 {
		r.offset = offset[0]
	}
	return r
}

// Quiet 不传参数则设置showSQL为false
func (r *Repo) Quiet(showSQL ...bool) *Repo {
	if len(showSQL) > 0 {
		r.showSQL = showSQL[0]
	} else {
		r.showSQL = false
	}
	return r
}

// Cols 设置要查询的column
func (r *Repo) Cols(cols string) *Repo {
	r.cols = cols
	return r
}

func (r *Repo) insure() {
	if r.db == nil {
		r.Use("default")
	}
}

func (r *Repo) p(query string, args []interface{}) {
	if r.showSQL {
		log.Println("[orm]", query, "params:", args)
	}
}

func (r *Repo) exec(query string, args ...interface{}) (sql.Result, error) {
	r.insure()
	r.p(query, args)
	return r.db.Exec(query, args...)
}

func (r *Repo) queryRow(query string, args ...interface{}) *sql.Row {
	r.insure()
	r.p(query, args)
	return r.db.QueryRow(query, args...)
}

func (r *Repo) query(query string, args ...interface{}) (*sql.Rows, error) {
	r.insure()
	r.p(query, args)
	return r.db.Query(query, args...)
}

func (r *Repo) buildSQL() {
	if r.cols == "" {
		r.cols = "*"
	}

	buf := new(bytes.Buffer)
	buf.WriteString("SELECT ")
	buf.WriteString(r.cols)
	buf.WriteString(" FROM `")
	buf.WriteString(r.tbl)
	buf.WriteString("`")

	if r.where != "" {
		buf.WriteString(" WHERE ")
		buf.WriteString(r.where)
	}

	if r.orderBy != "" {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(r.orderBy)
	}

	if r.limit > 0 {
		buf.WriteString(" LIMIT ?")
		r.args = append(r.args, r.limit)
	}

	if r.offset > 0 {
		buf.WriteString(" OFFSET ?")
		r.args = append(r.args, r.offset)
	}

	r.sql = buf.String()
}

// Count 统计数目
func (r *Repo) Count() (count int, err error) {
	r.cols = "count(*) as count"
	r.buildSQL()
	err = r.queryRow(r.sql, r.args...).Scan(&count)
	return
}

// Insert 保存一条数据，返回lastid
func (r *Repo) Insert(attrs G) (int64, error) {
	ln := len(attrs)
	keys := make([]string, 0, ln)
	qms := make([]string, 0, ln)
	vals := make([]interface{}, 0, ln)
	for k, v := range attrs {
		keys = append(keys, fmt.Sprintf("`%s`", k))
		qms = append(qms, "?")
		vals = append(vals, v)
	}

	s := fmt.Sprintf(
		"INSERT INTO `%s`(%s) VALUES(%s)",
		r.tbl,
		strings.Join(keys, ","),
		strings.Join(qms, ","),
	)

	ret, err := r.exec(s, vals...)
	if err != nil {
		return 0, err
	}

	return ret.LastInsertId()
}

// Delete 根据where条件做删除，返回被影响的行数
func (r *Repo) Delete() (int64, error) {
	s := fmt.Sprintf("DELETE FROM `%s`", r.tbl)
	if r.where != "" {
		s += " WHERE " + r.where
	}

	if r.limit > 0 {
		s += " LIMIT ?"
		r.args = append(r.args, r.limit)
	}

	ret, err := r.exec(s, r.args...)
	if err != nil {
		return 0, err
	}

	return ret.RowsAffected()
}

// Update 更新记录
func (r *Repo) Update(attrs G) (int64, error) {
	ln := len(attrs)
	keys := make([]string, 0, ln)
	vals := make([]interface{}, 0, ln)
	for k, v := range attrs {
		keys = append(keys, fmt.Sprintf("`%s`=?", k))
		vals = append(vals, v)
	}

	s := fmt.Sprintf("UPDATE `%s` SET %s", r.tbl, strings.Join(keys, ","))
	if r.where != "" {
		s += " WHERE " + r.where
		vals = append(vals, r.args...)
	}

	if r.limit > 0 {
		vals = append(vals, r.limit)
	}

	ret, err := r.exec(s, vals...)
	if err != nil {
		return 0, err
	}

	return ret.RowsAffected()
}

// I64Col 获取一列数据，数据类型是int64
func (r *Repo) I64Col(col string) ([]int64, error) {
	cols := []int64{}
	rs, err := r.col(col)
	if err != nil {
		return cols, err
	}

	defer rs.Close()

	for rs.Next() {
		var item int64
		err = rs.Scan(&item)
		if err != nil {
			return cols, err
		}

		cols = append(cols, item)
	}

	return cols, err
}

// StrCol 获取一列数据，数据类型是string
func (r *Repo) StrCol(col string) ([]string, error) {
	cols := []string{}
	rs, err := r.col(col)
	if err != nil {
		return cols, err
	}

	defer rs.Close()

	for rs.Next() {
		var item string
		err = rs.Scan(&item)
		if err != nil {
			return cols, err
		}

		cols = append(cols, item)
	}

	return cols, err
}

func (r *Repo) col(col string) (*sql.Rows, error) {
	r.cols = col
	r.buildSQL()
	return r.query(r.sql, r.args...)
}

// U64s 将uint64类型的slice拼接成逗号分隔的string
func U64s(ids []uint64) string {
	count := len(ids)
	strs := make([]string, count)
	for i := 0; i < count; i++ {
		strs[i] = fmt.Sprint(ids[i])
	}
	return strings.Join(strs, ",")
}

// I64s 将int64类型的slice拼接成逗号分隔的string
func I64s(ids []int64) string {
	count := len(ids)
	strs := make([]string, count)
	for i := 0; i < count; i++ {
		strs[i] = fmt.Sprint(ids[i])
	}
	return strings.Join(strs, ",")
}

// I64Arr 将逗号分隔的字符串ID转换成[]int64
func I64Arr(ids string) []int64 {
	if ids == "" {
		return []int64{}
	}

	arr := strings.Split(ids, ",")
	count := len(arr)
	ret := make([]int64, 0, count)
	for i := 0; i < count; i++ {
		if arr[i] == "" {
			continue
		}
		id, err := strconv.ParseInt(arr[i], 10, 64)
		if err != nil {
			continue
		}
		ret = append(ret, id)
	}
	return ret
}

// Rows 查询多行记录
func (r *Repo) Rows() (*sql.Rows, error) {
	r.insure()
	r.buildSQL()
	r.p(r.sql, r.args)
	stmt, err := r.db.Prepare(r.sql)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()
	return stmt.Query(r.args...)
}

// Row 查询一条记录
func (r *Repo) Row() *sql.Row {
	r.buildSQL()
	return r.queryRow(r.sql, r.args...)
}

// Find 查找一个struct，传入的第一个参数是struct的指针
func (r *Repo) Find(ptr interface{}) (bool, error) {
	rows, err := r.Rows()
	if err != nil {
		return false, err
	}

	val := reflect.ValueOf(ptr)

	defer rows.Close()

	if rows.Next() {
		err = r.scanRows(val, rows)
		if err != nil {
			return false, err
		}
	} else {
		return false, nil
	}

	return true, nil
}

// Finds 查询一个列表，ptr e.g. var user []*User -> &user
func (r *Repo) Finds(ptr interface{}) error {
	rows, err := r.Rows()
	if err != nil {
		return err
	}

	sliceValue := reflect.Indirect(reflect.ValueOf(ptr))
	structType := sliceValue.Type().Elem().Elem()

	defer rows.Close()

	for rows.Next() {
		rowValue := reflect.New(structType)
		err = r.scanRows(rowValue, rows)
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, rowValue))
	}

	return nil
}

func (r *Repo) scanRows(val reflect.Value, rows *sql.Rows) (err error) {
	cols, _ := rows.Columns()

	containers := make([]interface{}, 0, len(cols))
	for i := 0; i < cap(containers); i++ {
		var v interface{}
		containers = append(containers, &v)
	}

	err = rows.Scan(containers...)
	if err != nil {
		return
	}

	typ := val.Type()

	for i, v := range containers {
		value := reflect.Indirect(reflect.ValueOf(v))
		if !value.Elem().IsValid() {
			continue
		}

		key := cols[i]

		field := val.Elem().FieldByName(r.o.Tag2field(typ, key))
		if field.IsValid() {
			// value -> field
			err = setModelValue(value, field)
			if err != nil {
				return
			}
		}
	}

	return
}

func parseBool(value reflect.Value) bool {
	return value.Bool()
}

func setPtrValue(driverValue, fieldValue reflect.Value) {
	t := fieldValue.Type().Elem()
	v := reflect.New(t)
	fieldValue.Set(v)
	switch t.Kind() {
	case reflect.String:
		v.Elem().SetString(string(driverValue.Interface().([]uint8)))
	case reflect.Int64:
		v.Elem().SetInt(driverValue.Interface().(int64))
	case reflect.Float64:
		v.Elem().SetFloat(driverValue.Interface().(float64))
	case reflect.Bool:
		v.Elem().SetBool(driverValue.Interface().(bool))
	}
}

func setModelValue(driverValue, fieldValue reflect.Value) error {
	switch fieldValue.Type().Kind() {
	case reflect.Bool:
		fieldValue.SetBool(parseBool(driverValue.Elem()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(driverValue.Elem().Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// reading uint from int value causes panic
		switch driverValue.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldValue.SetUint(uint64(driverValue.Elem().Int()))
		default:
			fieldValue.SetUint(driverValue.Elem().Uint())
		}
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(driverValue.Elem().Float())
	case reflect.String:
		fieldValue.SetString(string(driverValue.Elem().Bytes()))
	case reflect.Slice:
		if reflect.TypeOf(driverValue.Interface()).Elem().Kind() == reflect.Uint8 {
			fieldValue.SetBytes(driverValue.Elem().Bytes())
		}
	case reflect.Ptr:
		setPtrValue(driverValue, fieldValue)
	case reflect.Struct:
		switch fieldValue.Interface().(type) {
		case time.Time:
			fieldValue.Set(driverValue.Elem())
		default:
			if scanner, ok := fieldValue.Addr().Interface().(sql.Scanner); ok {
				return scanner.Scan(driverValue.Interface())
			}
		}
	}
	return nil
}
