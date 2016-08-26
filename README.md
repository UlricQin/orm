*最简单的orm小框架，只支持mysql，下面是基本用法范例*

```go
package main

import (
	"log"

	"github.com/ulricqin/orm"

	_ "github.com/go-sql-driver/mysql"
)

// User 表在default库，即：minos_portal库
type User struct {
	ID       int64 `orm:"id"`
	Username string
	Nickname string
}

// UserRepo 这是每次DB操作的入口函数
// user表在default库，不需要使用Use来特别指定
func UserRepo() *orm.Repo {
	return Orm.NewRepo("user")
}

// Judge 在naming库，即：minos_naming库
type Judge struct {
	ID      int64  `orm:"id"`
	Address string `orm:"address"`
}

// JudgeRepo 这是每次DB操作的入口函数
// judge表不在默认的default库，故而需要执行Use
func JudgeRepo() *orm.Repo {
	return Orm.NewRepo("judge").Use("naming")
}

// DBConfig 数据库配置，支持配置多个库
// 至少有个default库
type DBConfig struct {
	Addr map[string]string
	Idle int
	Max  int
}

var configs = DBConfig{
	Addr: map[string]string{
		"default": "root@tcp(127.0.0.1:3306)/minos_portal?charset=utf8&&loc=Asia%2FShanghai",
		"naming":  "root@tcp(127.0.0.1:3306)/minos_naming?charset=utf8&&loc=Asia%2FShanghai",
	},
	Idle: 2,
	Max:  10,
}

// Orm 全局操作入口
var Orm *orm.Orm

func main() {

	// Orm 对象可以放在程序全局，程序启动的时候初始化好
	Orm = orm.New()

	// 配置Orm的DataSource
	for k, v := range configs.Addr {
		if err := Orm.Add(k, v, configs.Idle, configs.Max); err != nil {
			// 程序启动的时候如果发现数据库连接不上，直接报错退出
			log.Fatalln(err)
		}
	}

	// 将各个model注册给Orm，这样才能识别Struct中各个字段的orm tag
	Orm.Register(new(User), new(Judge))

	// 插入一条记录
	lastid, err := UserRepo().Insert(orm.G{
		"username": "UlricQin",
		"nickname": "秦晓辉",
	})
	dangerous(err)

	log.Println("insert user success, lastid:", lastid)

	// 查一条记录出来
	var user User
	has, err := UserRepo().Where("id=?", lastid).Find(&user)
	dangerous(err)

	if !has {
		log.Fatalln("no such user")
	}

	log.Println("Find user:", user)

	// 更新一条记录，如果调用了Quiet，将不打印sql语句
	num, err := UserRepo().Quiet().Where("id=?", lastid).Update(orm.G{
		"username": "Ulric2",
		"nickname": "晓辉",
	})
	dangerous(err)

	log.Println("update affected rows:", num)

	// 再插入一条记录，做个列表查询
	_, err = UserRepo().Insert(orm.G{
		"username": "Ulric1",
		"nickname": "Flame",
	})
	dangerous(err)

	// 计数
	count, err := UserRepo().Where("id>=?", lastid).Count()
	dangerous(err)

	log.Printf("user count of id>=%d is %d", lastid, count)

	// 只查询一列
	usernames, err := UserRepo().Where("id>=?", lastid).OrderBy("username").Limit(1, 1).StrCol("username")
	dangerous(err)

	log.Println("usernames, should only has Ulric2 => ", usernames)

	// 查询列表
	var users []*User
	err = UserRepo().Where("id>=?", lastid).Finds(&users)
	dangerous(err)

	log.Println("Find users:")
	for i := 0; i < len(users); i++ {
		log.Println(users[i])
	}

	// 删除操作
	num, err = UserRepo().Limit(2).Where("id>=?", lastid).Delete()
	dangerous(err)

	log.Println("delete user affected:", num)

	log.Println("------------------")

	// 以上封装的方法都是针对单表的，这个简易orm框架也就只做这些事情
	// 复杂的sql操作可以直接使用内部的*sql.DB，比如

	ret, err := Orm.Use("naming").Exec("insert into judge(address, last_update) values(?, now())", "127.0.0.1:7788")
	dangerous(err)

	lastid, err = ret.LastInsertId()
	dangerous(err)

	log.Println("insert address success, lastid:", lastid)

	row := Orm.Use("naming").QueryRow("select address from judge where id = ?", lastid)
	var address string
	err = row.Scan(&address)
	dangerous(err)
	log.Println("query row address:", address)

	_, err = Orm.Use("naming").Exec("delete from judge where id=?", lastid)
	dangerous(err)
}

func dangerous(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}


```