package orm

import (
	"database/sql"
	"reflect"
)

// Orm 一个程序创建一个全局的Orm对象即可
type Orm struct {
	dbs      map[string]*sql.DB
	mappings map[string]map[string]string
	ShowSQL  bool
}

// New 创建全局的Orm对象
func New() *Orm {
	return &Orm{
		dbs:      make(map[string]*sql.DB),
		mappings: make(map[string]map[string]string),
		ShowSQL:  true,
	}
}

// Add 增加一个DataSource
func (o *Orm) Add(name, addr string, idle, max int) error {
	db, err := sql.Open("mysql", addr)
	if err != nil {
		return err
	}

	db.SetMaxIdleConns(idle)
	db.SetMaxOpenConns(max)

	o.dbs[name] = db
	return nil
}

// Register 注册Struct，程序启动的时候先进行Register
// e.g. orm.New().Register(new(User), new(Topic))
func (o *Orm) Register(vs ...interface{}) {
	l := len(vs)
	for i := 0; i < l; i++ {
		typ := reflect.TypeOf(vs[i])
		ele := typ.Elem()
		num := ele.NumField()
		fields := make(map[string]string)
		for j := 0; j < num; j++ {
			field := ele.Field(j)
			tag := field.Tag.Get("orm")
			if tag != "" {
				fields[tag] = field.Name
			}
		}
		o.mappings[typ.String()] = fields
	}
}

// NewRepo 创建一个Repo，每做一次SQL操作都要新new一个Repo
func (o *Orm) NewRepo(tbl string) *Repo {
	return &Repo{
		o:       o,
		tbl:     tbl,
		showSQL: o.ShowSQL,
	}
}

// Use 使用哪个数据库
func (o *Orm) Use(name string) *sql.DB {
	db, has := o.dbs[name]
	if !has {
		panic("no such database: " + name)
	}
	return db
}

// Tag2field 通过tag查字段名称
func (o *Orm) Tag2field(typ reflect.Type, key string) string {
	m, has := o.mappings[typ.String()]
	if !has {
		return snakeToUpperCamel(key)
	}

	val, has := m[key]
	if !has {
		return snakeToUpperCamel(key)
	}

	return val
}
