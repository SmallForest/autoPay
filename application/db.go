/*
# @Time : 2020/9/23 11:24
# @Author : smallForest
# @SoftWare : GoLand
*/
package application

import (
	"github.com/jinzhu/gorm"
	"log"
)

func Mysql() *gorm.DB {
	db, err := gorm.Open("mysql", MysqlUsername+":"+MysqlPassword+"@tcp("+MysqlHost+":"+MysqlPort+")/"+MysqlDatabase+"?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Println(err)
	}
	return db
}
